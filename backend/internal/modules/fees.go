package modules

import (
	"fmt"
	"strings"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/fees"
)

// FeesModule is the wired fees module; config keys are documented in internal/fees/README.md.
type FeesModule struct {
	Port fees.Port
	// OperatorPlatformID, when set, is the only platform whose system.admin may manage fee rules.
	OperatorPlatformID uint
	// AdminEnabled is false in staging/production until FEES_OPERATOR_PLATFORM_ID is set.
	AdminEnabled bool
}

func WireFees(deps Deps) (*FeesModule, error) {
	if deps.DB == nil || deps.Ledger == nil || deps.Config == nil {
		return nil, fmt.Errorf("modules: fees needs DB, Ledger and Config")
	}
	cfg := deps.Config.Fees
	policy, err := fees.ParsePolicy(cfg.SurchargeForbiddenMethods, cfg.AssetPrecision)
	if err != nil {
		return nil, err
	}
	env := strings.ToUpper(strings.TrimSpace(deps.Config.Server.Environment))
	live := env == config.EnvironmentStaging || env == config.EnvironmentProduction
	return &FeesModule{
		Port:               fees.NewService(deps.DB, deps.Ledger, policy),
		OperatorPlatformID: cfg.OperatorPlatformID,
		AdminEnabled:       cfg.OperatorPlatformID != 0 || !live,
	}, nil
}
