// Package modules wires each capability module from configuration (MODULES.md rule 4), one file per module.
package modules

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/chaindeposit"
	"github.com/payminto/payminto/backend/internal/connectors/mock"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
	"gorm.io/gorm"
)

var ErrMockInDeployment = errors.New("modules: the mock connector cannot run in a deployment environment")

// Deps is what every Wire<Module> may draw on; modules take what they need.
type Deps struct {
	DB          *gorm.DB
	Config      *config.Config
	Ledger      *ledger.Service
	Environment string
	// ChainDeposit is the Payminto deposit flow the chaindeposit connector drives; nil disables that connector.
	ChainDeposit chaindeposit.Backend
}

type PaymentSwitchModule struct {
	Service    *paymentswitch.Service
	Connectors *connectors.Registry
	Enabled    []connectors.Code
}

// WirePaymentSwitch builds the connector registry from SWITCH_CONNECTORS, checks every connector's status map,
// and returns the switch service with the default first-enabled selector (ticket 06 swaps the selector).
func WirePaymentSwitch(deps Deps) (*PaymentSwitchModule, error) {
	if deps.DB == nil || deps.Ledger == nil || deps.Config == nil {
		return nil, fmt.Errorf("modules: paymentswitch needs DB, Ledger and Config")
	}
	deployment := deps.Environment == config.EnvironmentProduction || deps.Environment == config.EnvironmentStaging
	registry := connectors.NewRegistry()
	var enabled []connectors.Code
	for _, raw := range deps.Config.Switch.Connectors {
		code := connectors.Code(strings.ToLower(strings.TrimSpace(raw)))
		if code == "" || slices.Contains(enabled, code) {
			continue
		}
		var c connectors.Connector
		switch code {
		case mock.Code:
			if deployment {
				return nil, fmt.Errorf("%w: %s", ErrMockInDeployment, deps.Environment)
			}
			c = mock.New(mock.WithSecret(deps.Config.Switch.MockWebhookSecret))
		case chaindeposit.Code:
			if deps.ChainDeposit == nil {
				return nil, fmt.Errorf("modules: connector %s configured but no deposit backend wired", code)
			}
			c = chaindeposit.New(deps.ChainDeposit)
		default:
			return nil, fmt.Errorf("%w: %q in SWITCH_CONNECTORS", connectors.ErrUnknownConnector, code)
		}
		if err := paymentswitch.CheckStatusMap(c); err != nil {
			return nil, err
		}
		if err := registry.Register(c); err != nil {
			return nil, err
		}
		enabled = append(enabled, code)
	}
	selector := paymentswitch.FirstEnabledSelector{Merchants: paymentswitch.StaticMerchantConnectors(enabled), Connectors: registry}
	return &PaymentSwitchModule{
		Service:    paymentswitch.New(deps.DB, registry, selector, deps.Ledger),
		Connectors: registry,
		Enabled:    enabled,
	}, nil
}
