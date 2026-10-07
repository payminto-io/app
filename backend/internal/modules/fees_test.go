package modules

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/ledger"
	"gorm.io/gorm"
)

func feesDeps(env string) Deps {
	return Deps{DB: &gorm.DB{}, Ledger: ledger.New(&gorm.DB{}), Config: &config.Config{Server: config.ServerConfig{Environment: env}}}
}

func TestWireFeesAdminGate(t *testing.T) {
	cases := []struct {
		env, operator string
		enabled       bool
		id            uint
	}{
		{config.EnvironmentDevelopment, "", true, 0},
		{config.EnvironmentTest, "", true, 0},
		{config.EnvironmentProduction, "", false, 0},
		{config.EnvironmentStaging, "", false, 0},
		{config.EnvironmentProduction, "3", true, 3},
	}
	for _, tc := range cases {
		t.Setenv("FEES_OPERATOR_PLATFORM_ID", tc.operator)
		m, err := WireFees(feesDeps(tc.env))
		if err != nil {
			t.Fatalf("%s/%q: %v", tc.env, tc.operator, err)
		}
		if m.AdminEnabled != tc.enabled || m.OperatorPlatformID != tc.id || m.Port == nil {
			t.Errorf("%s/%q: module = %+v", tc.env, tc.operator, m)
		}
	}
}

func TestWireFeesRejectsBadConfig(t *testing.T) {
	t.Setenv("FEES_OPERATOR_PLATFORM_ID", "abc")
	if _, err := WireFees(feesDeps(config.EnvironmentDevelopment)); err == nil {
		t.Error("bad operator id accepted")
	}
	t.Setenv("FEES_OPERATOR_PLATFORM_ID", "")
	t.Setenv("FEES_SURCHARGE_FORBIDDEN_METHODS", "card,cash")
	if _, err := WireFees(feesDeps(config.EnvironmentDevelopment)); err == nil {
		t.Error("unknown surcharge method accepted")
	}
	if _, err := WireFees(Deps{}); err == nil {
		t.Error("missing deps accepted")
	}
}
