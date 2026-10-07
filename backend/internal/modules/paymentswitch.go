// Package modules wires each capability module from configuration (MODULES.md rule 4), one file per module.
package modules

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/chaindeposit"
	"github.com/payminto/payminto/backend/internal/connectors/mock"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
)

// ConnectorsSlot is the slot name of environment.KnownSlots the connectors module answers to.
const ConnectorsSlot = "connectors"

// DefaultConnectors is the enabled list: SWITCH_CONNECTORS when set; otherwise the environment contract's
// default, the mock and chaindeposit in test, CONNECTORS_PROVIDER or nothing at all in live. A live process with
// nothing configured boots with no connector rather than with the mock.
func DefaultConnectors(env environment.Environment, cfg *config.Config) []string {
	if cfg.Switch.Connectors != nil {
		return cfg.Switch.Connectors
	}
	resolved := environment.ResolveProvider(env, cfg.Modules.Providers[ConnectorsSlot])
	switch {
	case resolved == "":
		return nil
	case resolved == environment.MockProvider:
		return []string{environment.MockProvider, "chaindeposit"}
	default:
		return []string{resolved}
	}
}

type PaymentSwitchModule struct {
	Service    *paymentswitch.Service
	Connectors *connectors.Registry
	Enabled    []connectors.Code
	// Reconciler syncs unknown outcomes; main registers it with the worker manager.
	Reconciler *paymentswitch.Reconciler
}

// DomainEmitter is the slice of service.EventEmitterService the switch needs.
type DomainEmitter interface {
	EmitDomain(eventType string, payload any) error
}

// EmitterEvents adapts the existing ee_events emitter to the switch's Events port.
type EmitterEvents struct {
	Emitter DomainEmitter
}

func (e EmitterEvents) Emit(_ context.Context, ev paymentswitch.Event) error {
	payload := map[string]any{"merchant_id": ev.MerchantID, "payment_id": ev.IntentID, "attempt_id": ev.AttemptID}
	if ev.RefundID != "" {
		payload["refund_id"] = ev.RefundID
	}
	for k, v := range ev.Payload {
		payload[k] = v
	}
	return e.Emitter.EmitDomain(ev.Type, payload)
}

// WirePaymentSwitch builds the connector registry from SWITCH_CONNECTORS, checks every connector's status map,
// and returns the switch service with the default first-enabled selector (ticket 06 swaps the selector).
func WirePaymentSwitch(deps Deps) (*PaymentSwitchModule, error) {
	if deps.DB == nil || deps.Ledger == nil || deps.Config == nil {
		return nil, fmt.Errorf("modules: paymentswitch needs DB, Ledger and Config")
	}
	deployment := deps.Config.Server.Environment == config.EnvironmentProduction || deps.Config.Server.Environment == config.EnvironmentStaging
	var guard environment.Guard
	if deps.Environment != nil {
		guard = deps.Environment.Guard
	} else {
		if deployment {
			return nil, fmt.Errorf("modules: paymentswitch needs the environment module in %s", deps.Config.Server.Environment)
		}
		guard, _ = environment.NewGuard(environment.Test)
	}
	registry := connectors.NewRegistry()
	var enabled []connectors.Code
	codes := DefaultConnectors(guard.Current(), deps.Config)
	if len(codes) == 0 {
		// The environment contract: a slot that resolves to nothing is refused in live (test may run without connectors).
		if err := guard.(*environment.ProcessGuard).RequireProvider(ConnectorsSlot, ""); err != nil {
			return nil, err
		}
	}
	for _, raw := range codes {
		code := connectors.Code(strings.ToLower(strings.TrimSpace(raw)))
		if code == "" || slices.Contains(enabled, code) {
			continue
		}
		// Connectors are a slot: live refuses the mock through the environment guard.
		if err := guard.(*environment.ProcessGuard).RequireProvider(ConnectorsSlot, string(code)); err != nil {
			return nil, err
		}
		var c connectors.Connector
		switch code {
		case mock.Code:
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
	// The ledger handle must be bound to the same environment, or every live journal fails after the connector charged (F1).
	if deps.Ledger.Environment() != guard.Current() {
		return nil, fmt.Errorf("%w: the switch's ledger is bound to %s but the process is %s", environment.ErrMismatch, deps.Ledger.Environment(), guard.Current())
	}
	selector := paymentswitch.FirstEnabledSelector{Merchants: paymentswitch.StaticMerchantConnectors(enabled), Connectors: registry}
	var opts []paymentswitch.Option
	if deps.Events != nil {
		opts = append(opts, paymentswitch.WithEvents(deps.Events))
	}
	if deps.Fees != nil {
		records := deps.Records
		if records == nil {
			records = PaymintoPaymentRecords{DB: deps.DB}
		}
		opts = append(opts, paymentswitch.WithFees(deps.Fees, records))
	} else if deployment {
		return nil, fmt.Errorf("modules: paymentswitch needs the fees module in %s", deps.Config.Server.Environment)
	}
	opts = append(opts, paymentswitch.WithGuard(guard), paymentswitch.WithLease(deps.Config.Switch.ClaimLease), paymentswitch.WithLateReceiptRetention(deps.Config.Switch.LateReceiptRetention), paymentswitch.WithIntentTTL(deps.Config.Switch.IntentTTL))
	svc := paymentswitch.New(deps.DB, registry, selector, deps.Ledger, opts...)
	return &PaymentSwitchModule{
		Service:    svc,
		Connectors: registry,
		Enabled:    enabled,
		Reconciler: paymentswitch.NewReconciler(svc),
	}, nil
}
