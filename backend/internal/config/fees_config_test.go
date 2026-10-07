package config

import "testing"

func TestLoad_FeesKeys(t *testing.T) {
	t.Setenv("FEES_SURCHARGE_FORBIDDEN_METHODS", "card,upi")
	t.Setenv("FEES_ASSET_PRECISION", "XRP:6")
	t.Setenv("FEES_OPERATOR_PLATFORM_ID", "7")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Fees.SurchargeForbiddenMethods != "card,upi" || cfg.Fees.AssetPrecision != "XRP:6" || cfg.Fees.OperatorPlatformID != 7 {
		t.Fatalf("fees config = %+v", cfg.Fees)
	}
	for _, bad := range []string{"abc", "0", "-3"} {
		t.Setenv("FEES_OPERATOR_PLATFORM_ID", bad)
		if _, err := Load(); err == nil {
			t.Errorf("FEES_OPERATOR_PLATFORM_ID=%q accepted", bad)
		}
	}
}
