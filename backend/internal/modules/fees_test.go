package modules

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/ledger"
	"gorm.io/gorm"
)

func feesDeps(env string, fc config.FeesConfig) Deps {
	return Deps{DB: &gorm.DB{}, Ledger: ledger.New(&gorm.DB{}), Config: &config.Config{Server: config.ServerConfig{Environment: env}, Fees: fc}}
}

func TestWireFeesAdminGate(t *testing.T) {
	cases := []struct {
		env      string
		operator uint
		enabled  bool
	}{
		{config.EnvironmentDevelopment, 0, true},
		{config.EnvironmentTest, 0, true},
		{config.EnvironmentProduction, 0, false},
		{config.EnvironmentStaging, 0, false},
		{config.EnvironmentProduction, 3, true},
	}
	for _, tc := range cases {
		m, err := WireFees(feesDeps(tc.env, config.FeesConfig{OperatorPlatformID: tc.operator}))
		if err != nil {
			t.Fatalf("%s/%d: %v", tc.env, tc.operator, err)
		}
		if m.AdminEnabled != tc.enabled || m.OperatorPlatformID != tc.operator || m.Port == nil {
			t.Errorf("%s/%d: module = %+v", tc.env, tc.operator, m)
		}
	}
}

func TestWireFeesRejectsBadConfig(t *testing.T) {
	for _, fc := range []config.FeesConfig{{SurchargeForbiddenMethods: "card,cash"}, {AssetPrecision: "XRP:x"}, {AssetPrecision: "USD:3"}} {
		if _, err := WireFees(feesDeps(config.EnvironmentDevelopment, fc)); err == nil {
			t.Errorf("%+v accepted", fc)
		}
	}
	if _, err := WireFees(Deps{}); err == nil {
		t.Error("missing deps accepted")
	}
}
