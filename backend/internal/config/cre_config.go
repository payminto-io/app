package config

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// CRE providers; docs/cre/SPEC.md section 2 defines the rules applied in validateCRE.
const (
	CREProviderNone      = "none"
	CREProviderMock      = "mock"
	CREProviderChainlink = "chainlink"
)

// CREConfig holds the CRE_* keys; internal/cre/README.md "Configuration" documents them.
// It never holds key material: TriggerSigner is a reference the signer service resolves.
type CREConfig struct {
	Enabled  bool
	Provider string

	Chain            string
	ChainRPCURL      string
	ConsumerAddress  string
	ForwarderAddress string
	WorkflowOwner    string
	GatewayURL       string

	WorkflowIDSolvency            string
	WorkflowIDDepositFinality     string
	WorkflowIDConversionReference string
	// Per-kind owner overrides; empty falls back to WorkflowOwner.
	WorkflowOwnerSolvency            string
	WorkflowOwnerDepositFinality     string
	WorkflowOwnerConversionReference string
	// Workflow names as in workflow.yaml; the Keystone name in every report is derived from them.
	WorkflowNameSolvency            string
	WorkflowNameDepositFinality     string
	WorkflowNameConversionReference string

	TriggerSigner string

	SolvencyInterval      time.Duration
	FinalityBatchInterval time.Duration
	PollInterval          time.Duration
	VerifyConfirmations   uint64

	// StartBlock is the consumer contract's deployment block; a fresh poll cursor starts there. Required for chainlink.
	StartBlock uint64
	// ForwarderSimulated marks CRE_FORWARDER_ADDRESS as a mock or simulation forwarder; refused in live.
	ForwarderSimulated  bool
	PublicVerifyEnabled bool
	// PublicBaseURL is this deployment's public API base; gatewayId is its keccak256 (SPEC section 5).
	PublicBaseURL string

	// Per-workflow bearer credentials for the pull and push routes; empty refuses that workflow's routes.
	ReadTokenSolvency            string
	ReadTokenDepositFinality     string
	ReadTokenConversionReference string

	// AssetDecimals adds "CODE:decimals,..." for assets the built-in table does not know.
	AssetDecimals string

	// DegradedFrom and MissingKeys record a chainlink configuration that fell back to mock outside live.
	DegradedFrom string
	MissingKeys  []string
}

var (
	evmAddressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	workflowIDPattern = regexp.MustCompile(`^(0x)?[0-9a-fA-F]{64}$`)
	rawKeyPattern     = regexp.MustCompile(`^(0x)?[0-9a-fA-F]{64}$`)
)

func loadCRE() (CREConfig, error) {
	enabled, err := envBoolStrict("CRE_ENABLED", false)
	if err != nil {
		return CREConfig{}, err
	}
	publicVerify, err := envBoolStrict("CRE_PUBLIC_VERIFY_ENABLED", true)
	if err != nil {
		return CREConfig{}, err
	}
	forwarderSimulated, err := envBoolStrict("CRE_FORWARDER_SIMULATED", false)
	if err != nil {
		return CREConfig{}, err
	}
	startBlock, _, err := envUint64Opt("CRE_START_BLOCK")
	if err != nil {
		return CREConfig{}, err
	}
	solvency, err := envDuration("CRE_SOLVENCY_INTERVAL", time.Hour)
	if err != nil {
		return CREConfig{}, err
	}
	finality, err := envDuration("CRE_FINALITY_BATCH_INTERVAL", 60*time.Second)
	if err != nil {
		return CREConfig{}, err
	}
	poll, err := envDuration("CRE_POLL_INTERVAL", 30*time.Second)
	if err != nil {
		return CREConfig{}, err
	}
	confirmations, confirmationsSet, err := envUint64Opt("CRE_VERIFY_CONFIRMATIONS")
	if err != nil {
		return CREConfig{}, err
	}
	if !confirmationsSet && isDeploymentEnvironment(normalizedServerEnv()) {
		confirmations = defaultLiveConfirmations
	}
	return CREConfig{
		Enabled:                          enabled,
		Provider:                         strings.ToLower(strings.TrimSpace(envStr("CRE_PROVIDER", CREProviderNone))),
		Chain:                            strings.TrimSpace(envStr("CRE_CHAIN", "ethereum-testnet-sepolia-base-1")),
		ChainRPCURL:                      strings.TrimSpace(envStr("CRE_CHAIN_RPC_URL", "")),
		ConsumerAddress:                  strings.TrimSpace(envStr("CRE_CONSUMER_ADDRESS", "")),
		ForwarderAddress:                 strings.TrimSpace(envStr("CRE_FORWARDER_ADDRESS", "")),
		WorkflowOwner:                    strings.TrimSpace(envStr("CRE_WORKFLOW_OWNER", "")),
		GatewayURL:                       strings.TrimRight(strings.TrimSpace(envStr("CRE_GATEWAY_URL", "https://01.gateway.zone-a.cre.chain.link")), "/"),
		WorkflowIDSolvency:               strings.TrimSpace(envStr("CRE_WORKFLOW_ID_SOLVENCY", "")),
		WorkflowIDDepositFinality:        strings.TrimSpace(envStr("CRE_WORKFLOW_ID_DEPOSIT_FINALITY", "")),
		WorkflowIDConversionReference:    strings.TrimSpace(envStr("CRE_WORKFLOW_ID_CONVERSION_REFERENCE", "")),
		WorkflowOwnerSolvency:            strings.TrimSpace(envStr("CRE_WORKFLOW_OWNER_SOLVENCY", "")),
		WorkflowOwnerDepositFinality:     strings.TrimSpace(envStr("CRE_WORKFLOW_OWNER_DEPOSIT_FINALITY", "")),
		WorkflowOwnerConversionReference: strings.TrimSpace(envStr("CRE_WORKFLOW_OWNER_CONVERSION_REFERENCE", "")),
		WorkflowNameSolvency:             strings.TrimSpace(envStr("CRE_WORKFLOW_NAME_SOLVENCY", "solvency")),
		WorkflowNameDepositFinality:      strings.TrimSpace(envStr("CRE_WORKFLOW_NAME_DEPOSIT_FINALITY", "deposit-finality")),
		WorkflowNameConversionReference:  strings.TrimSpace(envStr("CRE_WORKFLOW_NAME_CONVERSION_REFERENCE", "conversion-reference")),
		TriggerSigner:                    strings.TrimSpace(envStr("CRE_TRIGGER_SIGNER", "")),
		SolvencyInterval:                 solvency,
		FinalityBatchInterval:            finality,
		PollInterval:                     poll,
		VerifyConfirmations:              confirmations,
		PublicVerifyEnabled:              publicVerify,
		ForwarderSimulated:               forwarderSimulated,
		StartBlock:                       startBlock,
		PublicBaseURL:                    strings.TrimRight(strings.TrimSpace(envStr("CRE_PUBLIC_BASE_URL", "")), "/"),
		ReadTokenSolvency:                strings.TrimSpace(envStr("CRE_READ_TOKEN_SOLVENCY", "")),
		ReadTokenDepositFinality:         strings.TrimSpace(envStr("CRE_READ_TOKEN_DEPOSIT_FINALITY", "")),
		ReadTokenConversionReference:     strings.TrimSpace(envStr("CRE_READ_TOKEN_CONVERSION_REFERENCE", "")),
		AssetDecimals:                    strings.TrimSpace(envStr("CRE_ASSET_DECIMALS", "")),
	}, nil
}

// chainlinkRequired lists the keys the chainlink provider cannot run without (SPEC section 2).
func (c CREConfig) chainlinkMissing() []string {
	required := []struct{ key, value string }{
		{"CRE_CHAIN_RPC_URL", c.ChainRPCURL},
		{"CRE_CONSUMER_ADDRESS", c.ConsumerAddress},
		{"CRE_FORWARDER_ADDRESS", c.ForwarderAddress},
		{"CRE_WORKFLOW_OWNER", c.WorkflowOwner},
		{"CRE_WORKFLOW_ID_SOLVENCY", c.WorkflowIDSolvency},
		{"CRE_WORKFLOW_ID_DEPOSIT_FINALITY", c.WorkflowIDDepositFinality},
		{"CRE_WORKFLOW_ID_CONVERSION_REFERENCE", c.WorkflowIDConversionReference},
		{"CRE_TRIGGER_SIGNER", c.TriggerSigner},
		{"CRE_PUBLIC_BASE_URL", c.PublicBaseURL},
		{"CRE_START_BLOCK", startBlockString(c.StartBlock)},
	}
	var missing []string
	for _, r := range required {
		if r.value == "" {
			missing = append(missing, r.key)
		}
	}
	return missing
}

// validateCRE applies SPEC section 2: off forces none, mock never runs live, chainlink needs every key in live.
func (c *CREConfig) validate(environment string) error {
	live := isDeploymentEnvironment(environment)
	if !c.Enabled {
		c.Provider = CREProviderNone
		return nil
	}
	if !slices.Contains([]string{CREProviderNone, CREProviderMock, CREProviderChainlink}, c.Provider) {
		return fmt.Errorf("CRE_PROVIDER must be none, mock or chainlink; got %q", c.Provider)
	}
	if c.Provider == CREProviderNone {
		return nil
	}
	if rawKeyPattern.MatchString(c.TriggerSigner) {
		return fmt.Errorf("CRE_TRIGGER_SIGNER must be a key reference such as keyring://cre-trigger, never a private key")
	}
	if c.SolvencyInterval < time.Minute {
		return fmt.Errorf("CRE_SOLVENCY_INTERVAL must be at least 1m; got %s", c.SolvencyInterval)
	}
	if c.FinalityBatchInterval < 30*time.Second {
		return fmt.Errorf("CRE_FINALITY_BATCH_INTERVAL must be at least 30s; got %s", c.FinalityBatchInterval)
	}
	if c.PollInterval < 5*time.Second {
		return fmt.Errorf("CRE_POLL_INTERVAL must be at least 5s; got %s", c.PollInterval)
	}
	for key, value := range map[string]string{"CRE_READ_TOKEN_SOLVENCY": c.ReadTokenSolvency, "CRE_READ_TOKEN_DEPOSIT_FINALITY": c.ReadTokenDepositFinality, "CRE_READ_TOKEN_CONVERSION_REFERENCE": c.ReadTokenConversionReference} {
		if value != "" && len(value) < minReadTokenLength {
			return fmt.Errorf("%s must be at least %d characters (openssl rand -hex 32)", key, minReadTokenLength)
		}
	}
	for key, value := range map[string]string{"CRE_CONSUMER_ADDRESS": c.ConsumerAddress, "CRE_FORWARDER_ADDRESS": c.ForwarderAddress, "CRE_WORKFLOW_OWNER": c.WorkflowOwner,
		"CRE_WORKFLOW_OWNER_SOLVENCY": c.WorkflowOwnerSolvency, "CRE_WORKFLOW_OWNER_DEPOSIT_FINALITY": c.WorkflowOwnerDepositFinality, "CRE_WORKFLOW_OWNER_CONVERSION_REFERENCE": c.WorkflowOwnerConversionReference} {
		if value != "" && !evmAddressPattern.MatchString(value) {
			return fmt.Errorf("%s must be a 0x-prefixed 20-byte hex address; got %q", key, value)
		}
	}
	for key, value := range map[string]string{"CRE_WORKFLOW_ID_SOLVENCY": c.WorkflowIDSolvency, "CRE_WORKFLOW_ID_DEPOSIT_FINALITY": c.WorkflowIDDepositFinality, "CRE_WORKFLOW_ID_CONVERSION_REFERENCE": c.WorkflowIDConversionReference} {
		if value != "" && !workflowIDPattern.MatchString(value) {
			return fmt.Errorf("%s must be a 32-byte hex workflow id; got %q", key, value)
		}
	}
	if live && c.ForwarderSimulated {
		return fmt.Errorf("%s refuses CRE_FORWARDER_SIMULATED=true; a simulation forwarder never carries production attestations", strings.ToLower(environment))
	}
	if live {
		for key, value := range map[string]string{"CRE_WORKFLOW_ID_SOLVENCY": c.WorkflowIDSolvency, "CRE_WORKFLOW_ID_DEPOSIT_FINALITY": c.WorkflowIDDepositFinality, "CRE_WORKFLOW_ID_CONVERSION_REFERENCE": c.WorkflowIDConversionReference} {
			if strings.EqualFold(strings.TrimPrefix(value, "0x"), strings.Repeat("11", 32)) {
				return fmt.Errorf("%s refuses %s: that is the CRE simulator's fixed workflow id", strings.ToLower(environment), key)
			}
		}
		for key, value := range map[string]string{"CRE_WORKFLOW_OWNER": c.WorkflowOwner, "CRE_WORKFLOW_OWNER_SOLVENCY": c.WorkflowOwnerSolvency, "CRE_WORKFLOW_OWNER_DEPOSIT_FINALITY": c.WorkflowOwnerDepositFinality, "CRE_WORKFLOW_OWNER_CONVERSION_REFERENCE": c.WorkflowOwnerConversionReference} {
			if strings.EqualFold(strings.TrimPrefix(value, "0x"), strings.Repeat("aa", 20)) {
				return fmt.Errorf("%s refuses %s: that is the CRE simulator's fixed owner", strings.ToLower(environment), key)
			}
		}
	}
	if c.Provider == CREProviderChainlink && live && c.VerifyConfirmations == 0 {
		return fmt.Errorf("%s CRE_VERIFY_CONFIRMATIONS must be above zero so a failing finalized tag never means zero confirmations", strings.ToLower(environment))
	}
	if c.Provider == CREProviderMock {
		if live {
			return fmt.Errorf("%s refuses CRE_PROVIDER=mock; mock attestations are not independently signed", strings.ToLower(environment))
		}
		return nil
	}
	missing := c.chainlinkMissing()
	if len(missing) == 0 {
		return nil
	}
	slices.Sort(missing)
	if live {
		return fmt.Errorf("%s CRE_PROVIDER=chainlink is missing %s", strings.ToLower(environment), strings.Join(missing, ", "))
	}
	c.DegradedFrom = CREProviderChainlink
	c.MissingKeys = missing
	c.Provider = CREProviderMock
	return nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(envStrRaw(key))
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration such as 60s or 1h; got %q", key, raw)
	}
	return d, nil
}

func startBlockString(n uint64) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprint(n)
}

// minReadTokenLength keeps a workflow credential at 32 bytes of hex or better; it is only ever compared by hash.
const minReadTokenLength = 32

// defaultLiveConfirmations applies when CRE_VERIFY_CONFIRMATIONS is unset in staging and production.
const defaultLiveConfirmations = 12

func normalizedServerEnv() string {
	env, err := normalizeEnvironment(envStr("SERVER", EnvironmentDevelopment))
	if err != nil {
		return EnvironmentDevelopment
	}
	return env
}

func envUint64Opt(key string) (uint64, bool, error) {
	raw := strings.TrimSpace(envStrRaw(key))
	if raw == "" {
		return 0, false, nil
	}
	var n uint64
	if _, err := fmt.Sscanf(raw, "%d", &n); err != nil || fmt.Sprint(n) != raw {
		return 0, false, fmt.Errorf("%s must be a non-negative integer; got %q", key, raw)
	}
	return n, true, nil
}
