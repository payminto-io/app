package config

import (
	"strings"
	"testing"
)

func TestTempoConfigurationRejectsLiveAndMainnet(t *testing.T) {
	for _, settings := range []struct{ env, network string }{{"live", "mainnet"}, {"test", "mainnet"}} {
		t.Run(settings.env+"/"+settings.network, func(t *testing.T) {
			t.Setenv("TEMPO_MERCHANT_ADDRESS", "0x82c70b91731914e7b678a232b366336Af1436455")
			t.Setenv("GATEWAY_ENVIRONMENT", settings.env)
			t.Setenv("BLOCKCHAIN_NETWORK_TYPE", settings.network)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), "TEMPO") {
				t.Fatalf("Tempo live/mainnet config accepted: %v", err)
			}
		})
	}
}

func TestTempoConfigurationRejectsInvalidReceiver(t *testing.T) {
	for _, address := range []string{"invalid", "0x0000000000000000000000000000000000000000"} {
		t.Run(address, func(t *testing.T) {
			t.Setenv("TEMPO_MERCHANT_ADDRESS", address)
			t.Setenv("GATEWAY_ENVIRONMENT", "test")
			t.Setenv("BLOCKCHAIN_NETWORK_TYPE", "testnet")
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), "TEMPO") {
				t.Fatalf("invalid receiver accepted: %v", err)
			}
		})
	}
}
