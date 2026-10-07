package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func clearCRE(t *testing.T) {
	t.Helper()
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "CRE_") {
			t.Setenv(strings.SplitN(e, "=", 2)[0], "")
		}
	}
}

func setChainlinkKeys(t *testing.T) {
	t.Helper()
	t.Setenv("CRE_CHAIN_RPC_URL", "http://127.0.0.1:8545")
	t.Setenv("CRE_CONSUMER_ADDRESS", "0x1111111111111111111111111111111111111111")
	t.Setenv("CRE_FORWARDER_ADDRESS", "0x2222222222222222222222222222222222222222")
	t.Setenv("CRE_WORKFLOW_OWNER", "0x3333333333333333333333333333333333333333")
	t.Setenv("CRE_WORKFLOW_ID_SOLVENCY", strings.Repeat("a", 64))
	t.Setenv("CRE_WORKFLOW_ID_DEPOSIT_FINALITY", strings.Repeat("b", 64))
	t.Setenv("CRE_WORKFLOW_ID_CONVERSION_REFERENCE", strings.Repeat("c", 64))
	t.Setenv("CRE_TRIGGER_SIGNER", "keyring://cre-trigger")
	t.Setenv("CRE_PUBLIC_BASE_URL", "https://pay.example.test")
	t.Setenv("CRE_READ_TOKEN_SOLVENCY", strings.Repeat("s", 64))
}

func liveEnv(t *testing.T) {
	t.Helper()
	t.Setenv("SERVER", "production")
	t.Setenv("POSTGRES_SCHEMA_MODE", "validate")
	t.Setenv("POSTGRES_SSL_MODE", "verify-full")
	t.Setenv("POSTGRES_HOST", "db.internal.example")
	t.Setenv("POSTGRES_PASSWORD", "k9Q2vX7mN4pL1sR8tW3y")
	t.Setenv("JWT_SECRET", "Zq8vN3kL7pR2sW5xT9yB4mC6dF1gH0jK")
}

func TestCRE_DefaultIsOffAndNone(t *testing.T) {
	clearCRE(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CRE.Enabled || cfg.CRE.Provider != CREProviderNone {
		t.Fatalf("default CRE = %+v, want disabled and none", cfg.CRE)
	}
}

func TestCRE_DisabledForcesNone(t *testing.T) {
	clearCRE(t)
	t.Setenv("CRE_ENABLED", "false")
	t.Setenv("CRE_PROVIDER", "chainlink")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CRE.Provider != CREProviderNone {
		t.Fatalf("provider = %q, want none when CRE_ENABLED=false", cfg.CRE.Provider)
	}
}

func TestCRE_UnknownProvider(t *testing.T) {
	clearCRE(t)
	t.Setenv("CRE_ENABLED", "true")
	t.Setenv("CRE_PROVIDER", "oracle")
	if _, err := Load(); err == nil {
		t.Fatal("unknown provider accepted")
	}
}

func TestCRE_MockRefusedInLive(t *testing.T) {
	clearCRE(t)
	liveEnv(t)
	t.Setenv("CRE_ENABLED", "true")
	t.Setenv("CRE_PROVIDER", "mock")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "mock") {
		t.Fatalf("err = %v, want mock refusal", err)
	}
}

func TestCRE_MockAllowedInDevelopment(t *testing.T) {
	clearCRE(t)
	t.Setenv("CRE_ENABLED", "true")
	t.Setenv("CRE_PROVIDER", "mock")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CRE.Provider != CREProviderMock || cfg.CRE.DegradedFrom != "" {
		t.Fatalf("CRE = %+v", cfg.CRE)
	}
}

func TestCRE_ChainlinkMissingKeysRefusedInLive(t *testing.T) {
	clearCRE(t)
	liveEnv(t)
	t.Setenv("CRE_ENABLED", "true")
	t.Setenv("CRE_PROVIDER", "chainlink")
	setChainlinkKeys(t)
	t.Setenv("CRE_WORKFLOW_ID_SOLVENCY", "")
	t.Setenv("CRE_TRIGGER_SIGNER", "")
	_, err := Load()
	if err == nil {
		t.Fatal("chainlink with missing keys booted in production")
	}
	for _, key := range []string{"CRE_WORKFLOW_ID_SOLVENCY", "CRE_TRIGGER_SIGNER"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q does not name %s", err, key)
		}
	}
}

func TestCRE_ChainlinkMissingKeysDegradesToMockElsewhere(t *testing.T) {
	clearCRE(t)
	t.Setenv("CRE_ENABLED", "true")
	t.Setenv("CRE_PROVIDER", "chainlink")
	setChainlinkKeys(t)
	t.Setenv("CRE_CONSUMER_ADDRESS", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CRE.Provider != CREProviderMock || cfg.CRE.DegradedFrom != CREProviderChainlink {
		t.Fatalf("CRE = %+v, want degraded to mock", cfg.CRE)
	}
	if len(cfg.CRE.MissingKeys) != 1 || cfg.CRE.MissingKeys[0] != "CRE_CONSUMER_ADDRESS" {
		t.Fatalf("missing = %v", cfg.CRE.MissingKeys)
	}
}

func TestCRE_ChainlinkCompleteBootsInLive(t *testing.T) {
	clearCRE(t)
	liveEnv(t)
	t.Setenv("CRE_ENABLED", "true")
	t.Setenv("CRE_PROVIDER", "chainlink")
	setChainlinkKeys(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CRE.Provider != CREProviderChainlink || cfg.CRE.DegradedFrom != "" {
		t.Fatalf("CRE = %+v", cfg.CRE)
	}
	if cfg.CRE.SolvencyInterval != time.Hour || cfg.CRE.FinalityBatchInterval != 60*time.Second || !cfg.CRE.PublicVerifyEnabled {
		t.Fatalf("defaults = %+v", cfg.CRE)
	}
}

func TestCRE_TriggerSignerIsAReferenceNeverAKey(t *testing.T) {
	clearCRE(t)
	t.Setenv("CRE_ENABLED", "true")
	t.Setenv("CRE_PROVIDER", "chainlink")
	setChainlinkKeys(t)
	t.Setenv("CRE_TRIGGER_SIGNER", "0x"+strings.Repeat("ab", 32))
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "never a private key") {
		t.Fatalf("raw key accepted as signer reference: %v", err)
	}
}

func TestCRE_ValidationOfShapes(t *testing.T) {
	cases := map[string]string{
		"CRE_CONSUMER_ADDRESS":        "not-an-address",
		"CRE_WORKFLOW_ID_SOLVENCY":    "abc",
		"CRE_SOLVENCY_INTERVAL":       "10s",
		"CRE_FINALITY_BATCH_INTERVAL": "5s",
		"CRE_POLL_INTERVAL":           "1s",
		"CRE_VERIFY_CONFIRMATIONS":    "-1",
	}
	for key, bad := range cases {
		t.Run(key, func(t *testing.T) {
			clearCRE(t)
			t.Setenv("CRE_ENABLED", "true")
			t.Setenv("CRE_PROVIDER", "mock")
			t.Setenv(key, bad)
			if _, err := Load(); err == nil {
				t.Errorf("%s=%q accepted", key, bad)
			}
		})
	}
}

func TestCRE_InvalidShapesIgnoredWhenDisabled(t *testing.T) {
	clearCRE(t)
	t.Setenv("CRE_ENABLED", "false")
	t.Setenv("CRE_PROVIDER", "mock")
	t.Setenv("CRE_CONSUMER_ADDRESS", "not-an-address")
	if _, err := Load(); err != nil {
		t.Fatalf("disabled CRE validated its keys: %v", err)
	}
}

func TestCRE_SlotFollowsTheResolvedProvider(t *testing.T) {
	clearCRE(t)
	t.Setenv("CRE_PROVIDER", "mock")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, present := cfg.Modules.Providers["cre"]; present {
		t.Fatalf("disabled CRE registered a slot provider: %v", cfg.Modules.Providers)
	}
	t.Setenv("CRE_ENABLED", "true")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Modules.Providers["cre"] != CREProviderMock {
		t.Fatalf("slot = %v, want mock", cfg.Modules.Providers)
	}
	// The slot carries the resolved provider: GATEWAY_ENVIRONMENT=live refuses it through the environment gate.
	facts := cfg.BootFacts()
	if facts.SlotProviders["cre"] != CREProviderMock {
		t.Fatalf("boot facts slot = %v", facts.SlotProviders)
	}
}

func TestCRE_ShortTokenRefusedAndNamesDefault(t *testing.T) {
	clearCRE(t)
	t.Setenv("CRE_ENABLED", "true")
	t.Setenv("CRE_PROVIDER", "mock")
	t.Setenv("CRE_READ_TOKEN_SOLVENCY", "short")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "CRE_READ_TOKEN_SOLVENCY") {
		t.Fatalf("short token accepted: %v", err)
	}
	t.Setenv("CRE_READ_TOKEN_SOLVENCY", strings.Repeat("a", 64))
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CRE.WorkflowNameSolvency != "solvency" || cfg.CRE.WorkflowNameDepositFinality != "deposit-finality" || cfg.CRE.WorkflowNameConversionReference != "conversion-reference" {
		t.Fatalf("names = %+v", cfg.CRE)
	}
	if cfg.CRE.VerifyConfirmations != 0 {
		t.Fatalf("development confirmations default = %d", cfg.CRE.VerifyConfirmations)
	}
}

func TestCRE_LiveConfirmationsDefaultAndZeroRefused(t *testing.T) {
	clearCRE(t)
	liveEnv(t)
	t.Setenv("CRE_ENABLED", "true")
	t.Setenv("CRE_PROVIDER", "chainlink")
	setChainlinkKeys(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CRE.VerifyConfirmations != defaultLiveConfirmations {
		t.Fatalf("live default confirmations = %d", cfg.CRE.VerifyConfirmations)
	}
	t.Setenv("CRE_VERIFY_CONFIRMATIONS", "0")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "CRE_VERIFY_CONFIRMATIONS") {
		t.Fatalf("zero confirmations accepted in live: %v", err)
	}
}
