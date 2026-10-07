// Package modules wires each capability module from configuration (MODULES.md rule 4), one file per module.
package modules

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/chaindeposit"
	"github.com/payminto/payminto/backend/internal/connectors/mock"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
)

var ErrMockInDeployment = errors.New("modules: the mock connector cannot run in a deployment environment")

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
				return nil, fmt.Errorf("%w: %s", ErrMockInDeployment, deps.Config.Server.Environment)
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
	opts = append(opts, paymentswitch.WithLease(deps.Config.Switch.ClaimLease), paymentswitch.WithLateReceiptRetention(deps.Config.Switch.LateReceiptRetention))
	svc := paymentswitch.New(deps.DB, registry, selector, deps.Ledger, opts...)
	return &PaymentSwitchModule{
		Service:    svc,
		Connectors: registry,
		Enabled:    enabled,
		Reconciler: paymentswitch.NewReconciler(svc),
	}, nil
}
