// Package environment is the core module that keeps live and test money apart.
// One process serves one environment; every money-moving module asks the Guard
// before it writes. Design: .scratch/payments-v1/issues/13-environments.md.
package environment

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Environment is the money mode a process, key or row belongs to.
type Environment string

const (
	Test Environment = "test"
	Live Environment = "live"
)

// All lists the valid environments in a stable order.
func All() []Environment { return []Environment{Test, Live} }

// Parse accepts "test" or "live" in any case; anything else is an error.
func Parse(value string) (Environment, error) {
	switch Environment(strings.ToLower(strings.TrimSpace(value))) {
	case Test:
		return Test, nil
	case Live:
		return Live, nil
	default:
		return "", fmt.Errorf("%w: %q (want test or live)", ErrInvalid, value)
	}
}

func (e Environment) Valid() bool { return e == Test || e == Live }

func (e Environment) String() string { return string(e) }

// KeyKind distinguishes the two API key families a merchant can hold.
type KeyKind string

const (
	SecretKey      KeyKind = "sk"
	PublishableKey KeyKind = "pk"
)

// KeyPrefix is the visible prefix an API key of this kind carries, e.g. "sk_live_".
func (e Environment) KeyPrefix(kind KeyKind) string {
	return string(kind) + "_" + string(e) + "_"
}

// KeyEnvironment reads the environment out of a raw key's prefix. Keys issued before
// prefixes existed ("pm_...") report ok=false and are treated as test by their row.
func KeyEnvironment(rawKey string) (env Environment, kind KeyKind, ok bool) {
	for _, k := range []KeyKind{SecretKey, PublishableKey} {
		for _, e := range All() {
			if strings.HasPrefix(rawKey, e.KeyPrefix(k)) {
				return e, k, true
			}
		}
	}
	return "", "", false
}

var (
	ErrInvalid  = errors.New("environment: invalid environment")
	ErrMismatch = errors.New("environment: mismatch")
	ErrBoot     = errors.New("environment: refusing to boot")
)

// MismatchError is the typed error every guard returns; errors.Is(err, ErrMismatch) holds.
type MismatchError struct {
	Process   Environment
	Requested Environment
}

func (e *MismatchError) Error() string {
	return fmt.Sprintf("environment: this process is %s but the request is %s", e.Process, e.Requested)
}

func (e *MismatchError) Is(target error) bool { return target == ErrMismatch }

// Guard is what money-moving modules call before touching a row.
type Guard interface {
	// Current is the environment this process serves.
	Current() Environment
	// Require returns a *MismatchError unless env (or, when env is empty, the
	// context's environment) equals the process environment.
	Require(ctx context.Context, env Environment) error
}

type contextKey struct{}

// WithContext tags ctx with env so downstream modules can resolve it.
func WithContext(ctx context.Context, env Environment) context.Context {
	return context.WithValue(ctx, contextKey{}, env)
}

// FromContext returns the environment WithContext stored, if any.
func FromContext(ctx context.Context) (Environment, bool) {
	if ctx == nil {
		return "", false
	}
	env, ok := ctx.Value(contextKey{}).(Environment)
	return env, ok && env.Valid()
}
