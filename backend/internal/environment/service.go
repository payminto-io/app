package environment

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
)

// ProcessGuard is the one Guard implementation: the process serves exactly one environment.
type ProcessGuard struct {
	env Environment
}

// NewGuard builds the guard for the environment this process serves.
func NewGuard(env Environment) (*ProcessGuard, error) {
	if !env.Valid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalid, env)
	}
	return &ProcessGuard{env: env}, nil
}

func (g *ProcessGuard) Current() Environment { return g.env }

// Context tags ctx with the process environment.
func (g *ProcessGuard) Context(ctx context.Context) context.Context {
	return WithContext(ctx, g.env)
}

// Require checks env, then the context, against the process environment.
func (g *ProcessGuard) Require(ctx context.Context, env Environment) error {
	if env == "" {
		if fromCtx, ok := FromContext(ctx); ok {
			env = fromCtx
		} else {
			env = g.env
		}
	}
	if !env.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalid, env)
	}
	if env != g.env {
		return &MismatchError{Process: g.env, Requested: env}
	}
	if fromCtx, ok := FromContext(ctx); ok && fromCtx != g.env {
		return &MismatchError{Process: g.env, Requested: fromCtx}
	}
	return nil
}

// Resolve returns env when set, else the context's environment, else fallback.
func Resolve(ctx context.Context, env, fallback Environment) Environment {
	if env != "" {
		return env
	}
	if fromCtx, ok := FromContext(ctx); ok {
		return fromCtx
	}
	return fallback
}

// MockProvider is the provider name every slot module ships for keyless development.
const MockProvider = "mock"

// BootFacts is everything the boot policy needs, gathered by the wiring layer so this
// package never reads configuration itself.
type BootFacts struct {
	Environment Environment
	// DatabaseName and DatabaseHost identify the Postgres this process will open.
	DatabaseName string
	DatabaseHost string
	// TestDatabaseName is the name reserved for test money; live must never open it.
	TestDatabaseName string
	// DevKeystore is true when the development keystore or a local vault master key is configured.
	DevKeystore bool
	// VaultDevMode is true when the secrets vault runs without a real passphrase.
	VaultDevMode bool
	// SlotProviders maps each slot module to its configured provider name.
	SlotProviders map[string]string
	// DatabaseSSLMode is the Postgres sslmode the connection will use.
	DatabaseSSLMode string
	// DatabaseInsecureLocalException is the operator's explicit opt-out for a loopback database.
	DatabaseInsecureLocalException bool
	// DeploymentHardened is true when the server config is in a deployment profile (strong secrets, HSTS).
	DeploymentHardened bool
	// NetworkType is the chain network the process will watch: testnet or mainnet.
	NetworkType string
	// JWTSecretWeak is true when JWT_SECRET is a known development default or shorter than 32 bytes.
	JWTSecretWeak bool
}

// BootError carries every refusal at once so an operator fixes them in one pass.
type BootError struct {
	Environment Environment
	Reasons     []string
}

func (e *BootError) Error() string {
	return fmt.Sprintf("environment: refusing to boot as %s: %s", e.Environment, strings.Join(e.Reasons, "; "))
}

func (e *BootError) Is(target error) bool { return target == ErrBoot }

// CheckBoot is the boot gate. It returns nil when the process may serve facts.Environment.
func CheckBoot(facts BootFacts) error {
	if !facts.Environment.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalid, facts.Environment)
	}
	var reasons []string
	refuse := func(format string, args ...any) { reasons = append(reasons, fmt.Sprintf(format, args...)) }

	// The configured name is checked before connecting; VerifyDatabase repeats the policy on
	// what Postgres reports plus the stamp, which is the authority.
	if err := CheckDatabase(facts.Environment, facts.DatabaseName, facts.TestDatabaseName, facts.DatabaseHost, ""); err != nil {
		var boot *BootError
		if errors.As(err, &boot) {
			reasons = append(reasons, boot.Reasons...)
		} else {
			refuse("%v", err)
		}
	}
	switch facts.Environment {
	case Live:
		if facts.DevKeystore {
			refuse("development keystore or local vault master key is configured (unset AES_KEY and DEV_KEYSTORE)")
		}
		if facts.VaultDevMode {
			refuse("secrets vault is in development mode")
		}
		for _, slot := range sortedSlots(facts.SlotProviders) {
			if strings.EqualFold(strings.TrimSpace(facts.SlotProviders[slot]), MockProvider) {
				refuse("slot %s is configured with the mock provider", slot)
			}
		}
		if !facts.DeploymentHardened {
			refuse("SERVER must be staging or production so secure cookie, HSTS and secret-strength rules apply")
		}
		if strings.ToLower(facts.DatabaseSSLMode) != "verify-full" {
			local := isLoopbackHost(facts.DatabaseHost) && facts.DatabaseInsecureLocalException && strings.ToLower(facts.DatabaseSSLMode) == "disable"
			if !local {
				refuse("POSTGRES_SSL_MODE must be verify-full (got %q)", facts.DatabaseSSLMode)
			}
		}
		if !strings.EqualFold(facts.NetworkType, "mainnet") {
			refuse("BLOCKCHAIN_NETWORK_TYPE must be mainnet for live money (got %q)", facts.NetworkType)
		}
		if facts.JWTSecretWeak {
			refuse("JWT_SECRET is a development default or shorter than 32 bytes; sessions would be forgeable")
		}
	case Test:
		if strings.EqualFold(facts.NetworkType, "mainnet") {
			refuse("BLOCKCHAIN_NETWORK_TYPE must not be mainnet for test money")
		}
	}
	if len(reasons) == 0 {
		return nil
	}
	return &BootError{Environment: facts.Environment, Reasons: reasons}
}

// NormalizeDatabaseName is how every database name is compared: trimmed and case-folded.
func NormalizeDatabaseName(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

// CheckDatabase is the database policy for env. name is the database (configured, or as Postgres
// reports it), stamp is the environment the database is stamped with ("" when new or unstamped).
// A test process may use a non-*_test name only on a loopback host and only when the stamp says
// test or the database is new; a stamp never agrees with the other environment.
func CheckDatabase(env Environment, name, testName, host string, stamp Environment) error {
	if !env.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalid, env)
	}
	var reasons []string
	refuse := func(format string, args ...any) { reasons = append(reasons, fmt.Sprintf(format, args...)) }
	name, testName = NormalizeDatabaseName(name), NormalizeDatabaseName(testName)
	if name == "" {
		refuse("database name is empty")
	}
	if stamp != "" && !stamp.Valid() {
		refuse("database carries an unknown environment stamp %q", stamp)
	}
	if stamp.Valid() && stamp != env {
		refuse("database %q is stamped %s; a %s process must not open it", name, stamp, env)
	}
	switch env {
	case Live:
		if name != "" && testName != "" && name == testName {
			refuse("database %q is the test database", name)
		}
		if strings.HasSuffix(name, "_test") {
			refuse("database %q ends in _test", name)
		}
	case Test:
		if name != "" && !strings.HasSuffix(name, "_test") {
			switch {
			case !isLoopbackHost(host):
				refuse("database %q on %q is neither named *_test nor local; test money needs its own database", name, host)
			case stamp == "":
				// New or unstamped on loopback: allowed, and the process stamps it test.
			case stamp != Test:
				refuse("database %q on a loopback host is stamped %s", name, stamp)
			}
		}
	}
	if len(reasons) == 0 {
		return nil
	}
	return &BootError{Environment: env, Reasons: reasons}
}

func sortedSlots(providers map[string]string) []string {
	slots := make([]string, 0, len(providers))
	for slot := range providers {
		slots = append(slots, slot)
	}
	sort.Strings(slots)
	return slots
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// IsBootRefusal reports whether err came from CheckBoot.
func IsBootRefusal(err error) bool { return errors.Is(err, ErrBoot) }
