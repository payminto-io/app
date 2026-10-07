package modules

import (
	"fmt"
	"os"
	"strconv"
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
	policy, err := fees.ParsePolicy(os.Getenv("FEES_SURCHARGE_FORBIDDEN_METHODS"))
	if err != nil {
		return nil, err
	}
	var operator uint
	if raw := strings.TrimSpace(os.Getenv("FEES_OPERATOR_PLATFORM_ID")); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || id == 0 {
			return nil, fmt.Errorf("modules: FEES_OPERATOR_PLATFORM_ID must be a positive platform id, got %q", raw)
		}
		operator = uint(id)
	}
	env := strings.ToUpper(strings.TrimSpace(deps.Config.Server.Environment))
	live := env == config.EnvironmentStaging || env == config.EnvironmentProduction
	return &FeesModule{
		Port:               fees.NewService(deps.DB, deps.Ledger, policy),
		OperatorPlatformID: operator,
		AdminEnabled:       operator != 0 || !live,
	}, nil
}
