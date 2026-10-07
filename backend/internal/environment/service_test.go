package environment

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGuardRequire(t *testing.T) {
	g, err := NewGuard(Live)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := g.Require(ctx, Live); err != nil {
		t.Errorf("explicit live on live: %v", err)
	}
	if err := g.Require(ctx, ""); err != nil {
		t.Errorf("empty env, empty ctx falls back to process: %v", err)
	}
	if err := g.Require(WithContext(ctx, Live), ""); err != nil {
		t.Errorf("ctx live: %v", err)
	}
	if err := g.Require(ctx, Test); !errors.Is(err, ErrMismatch) {
		t.Errorf("explicit test on live = %v, want mismatch", err)
	}
	if err := g.Require(WithContext(ctx, Test), ""); !errors.Is(err, ErrMismatch) {
		t.Errorf("ctx test on live = %v, want mismatch", err)
	}
	if err := g.Require(WithContext(ctx, Test), Live); !errors.Is(err, ErrMismatch) {
		t.Errorf("ctx test with explicit live on live = %v, want mismatch (context lies)", err)
	}
	if err := g.Require(ctx, "prod"); !errors.Is(err, ErrInvalid) {
		t.Errorf("invalid env = %v", err)
	}
	if _, err := NewGuard(""); !errors.Is(err, ErrInvalid) {
		t.Errorf("NewGuard(\"\") = %v", err)
	}
}

func TestResolve(t *testing.T) {
	ctx := context.Background()
	if got := Resolve(ctx, Live, Test); got != Live {
		t.Errorf("explicit wins: %s", got)
	}
	if got := Resolve(WithContext(ctx, Live), "", Test); got != Live {
		t.Errorf("context second: %s", got)
	}
	if got := Resolve(ctx, "", Test); got != Test {
		t.Errorf("fallback last: %s", got)
	}
}

func liveFacts() BootFacts {
	return BootFacts{
		Environment:        Live,
		DatabaseName:       "gateway",
		DatabaseHost:       "db.internal",
		TestDatabaseName:   "gateway_test",
		SlotProviders:      map[string]string{"custody": "bitgo"},
		DatabaseSSLMode:    "verify-full",
		DeploymentHardened: true,
		NetworkType:        "mainnet",
	}
}

func TestCheckBoot_LiveHappyPath(t *testing.T) {
	if err := CheckBoot(liveFacts()); err != nil {
		t.Fatalf("live happy path refused: %v", err)
	}
	local := liveFacts()
	local.DatabaseHost, local.DatabaseSSLMode, local.DatabaseInsecureLocalException = "127.0.0.1", "disable", true
	if err := CheckBoot(local); err != nil {
		t.Fatalf("live with explicit local database exception refused: %v", err)
	}
}

func TestCheckBoot_LiveRefusals(t *testing.T) {
	cases := map[string]struct {
		mutate func(*BootFacts)
		want   string
	}{
		"dev keystore":           {func(f *BootFacts) { f.DevKeystore = true }, "development keystore"},
		"vault dev mode":         {func(f *BootFacts) { f.VaultDevMode = true }, "development mode"},
		"mock provider":          {func(f *BootFacts) { f.SlotProviders["connectors"] = "Mock" }, "slot connectors resolved to the mock provider in live"},
		"test database by name":  {func(f *BootFacts) { f.DatabaseName = "gateway_test" }, "is the test database"},
		"test suffix":            {func(f *BootFacts) { f.DatabaseName, f.TestDatabaseName = "other_test", "x" }, "ends in _test"},
		"not hardened":           {func(f *BootFacts) { f.DeploymentHardened = false }, "SERVER must be staging or production"},
		"weak ssl":               {func(f *BootFacts) { f.DatabaseSSLMode = "require" }, "verify-full"},
		"ssl disabled no opt-in": {func(f *BootFacts) { f.DatabaseHost, f.DatabaseSSLMode = "localhost", "disable" }, "verify-full"},
		"testnet":                {func(f *BootFacts) { f.NetworkType = "testnet" }, "must be mainnet"},
		"empty database":         {func(f *BootFacts) { f.DatabaseName = "" }, "database name is empty"},
		"weak jwt secret":        {func(f *BootFacts) { f.JWTSecretWeak = true }, "JWT_SECRET is a development default"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			facts := liveFacts()
			tc.mutate(&facts)
			err := CheckBoot(facts)
			if !IsBootRefusal(err) {
				t.Fatalf("CheckBoot = %v, want boot refusal", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("CheckBoot = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestCheckBoot_ReportsEveryReasonAtOnce(t *testing.T) {
	facts := liveFacts()
	facts.DevKeystore = true
	facts.SlotProviders["custody"] = "mock"
	err := CheckBoot(facts)
	var boot *BootError
	if !errors.As(err, &boot) || len(boot.Reasons) != 2 {
		t.Fatalf("CheckBoot = %v, want two reasons", err)
	}
}

func TestCheckBoot_Test(t *testing.T) {
	ok := []BootFacts{
		{Environment: Test, DatabaseName: "gateway_test", DatabaseHost: "db.internal", NetworkType: "testnet"},
		{Environment: Test, DatabaseName: "payminto", DatabaseHost: "localhost", NetworkType: "testnet"},
		{Environment: Test, DatabaseName: "payminto", DatabaseHost: "127.0.0.1", NetworkType: "testnet"},
		{Environment: Test, DatabaseName: "payminto", DatabaseHost: "::1", DevKeystore: true, SlotProviders: map[string]string{"custody": "mock"}, NetworkType: "testnet"},
	}
	for _, facts := range ok {
		if err := CheckBoot(facts); err != nil {
			t.Errorf("test facts %+v refused: %v", facts, err)
		}
	}
	remote := BootFacts{Environment: Test, DatabaseName: "gateway", DatabaseHost: "db.internal", NetworkType: "testnet"}
	if err := CheckBoot(remote); !IsBootRefusal(err) || !strings.Contains(err.Error(), "neither named *_test nor local") {
		t.Errorf("remote non-test database = %v, want refusal", err)
	}
	mainnet := BootFacts{Environment: Test, DatabaseName: "gateway_test", NetworkType: "mainnet"}
	if err := CheckBoot(mainnet); !IsBootRefusal(err) {
		t.Errorf("test on mainnet = %v, want refusal", err)
	}
	if err := CheckBoot(BootFacts{Environment: "prod"}); !errors.Is(err, ErrInvalid) {
		t.Errorf("invalid environment = %v", err)
	}
}

func TestProviders_ResolveAndRequire(t *testing.T) {
	if got := ResolveProvider(Test, ""); got != MockProvider {
		t.Errorf("test default = %q, want mock", got)
	}
	if got := ResolveProvider(Live, ""); got != "" {
		t.Errorf("live default = %q, want nothing", got)
	}
	if got := ResolveProvider(Live, " BitGo "); got != "bitgo" {
		t.Errorf("normalised = %q", got)
	}
	guard, _ := NewGuard(Live)
	for _, slot := range KnownSlots {
		if err := guard.RequireProvider(slot, ResolveProvider(Live, "")); !errors.Is(err, ErrProvider) {
			t.Errorf("live %s with no config = %v, want ErrProvider", slot, err)
		}
		if err := guard.RequireProvider(slot, ResolveProvider(Live, "mock")); !errors.Is(err, ErrProvider) {
			t.Errorf("live %s mock = %v, want ErrProvider", slot, err)
		}
		if err := guard.RequireProvider(slot, ResolveProvider(Live, "bitgo")); err != nil {
			t.Errorf("live %s real provider refused: %v", slot, err)
		}
	}
	testGuard, _ := NewGuard(Test)
	if err := testGuard.RequireProvider("custody", ResolveProvider(Test, "")); err != nil {
		t.Errorf("test mock refused: %v", err)
	}
}
