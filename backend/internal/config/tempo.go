package config

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/payminto/payminto/backend/internal/environment"
)

// ValidateTempo keeps the Moderato-only payment rail out of live money processes.
// It is called both by config loading and wiring, including programmatically built configs.
func (c *Config) ValidateTempo() error {
	c.Tempo.MerchantAddress = strings.TrimSpace(c.Tempo.MerchantAddress)
	if c.Tempo.MerchantAddress == "" {
		return nil
	}
	if environment.Environment(c.Gateway.Environment) != environment.Test || !strings.EqualFold(strings.TrimSpace(c.Blockchain.NetworkType), "testnet") {
		return fmt.Errorf("TEMPO supports Moderato testnet only; require GATEWAY_ENVIRONMENT=test and BLOCKCHAIN_NETWORK_TYPE=testnet")
	}
	if !common.IsHexAddress(c.Tempo.MerchantAddress) || common.HexToAddress(c.Tempo.MerchantAddress) == (common.Address{}) {
		return fmt.Errorf("TEMPO_MERCHANT_ADDRESS must be a non-zero EVM address")
	}
	if c.Tempo.MemberID < 0 || c.Tempo.StartBlock < 0 || c.Tempo.PollIntervalSeconds < 1 {
		return fmt.Errorf("TEMPO_MEMBER_ID and TEMPO_START_BLOCK must be non-negative; TEMPO_POLL_INTERVAL_SECONDS must be positive")
	}
	return nil
}
