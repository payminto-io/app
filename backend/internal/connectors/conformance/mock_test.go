package conformance_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/conformance"
	"github.com/payminto/payminto/backend/internal/connectors/mock"
	"github.com/shopspring/decimal"
)

func TestMockConformance(t *testing.T) {
	conformance.Run(t, conformance.Harness{
		New:    func(t *testing.T) connectors.Connector { return mock.New() },
		Method: connectors.MethodCard,
		Money:  connectors.Money{Amount: decimal.NewFromInt(100), Asset: "USD"},
		PaymentMethod: func(o conformance.Outcome) (connectors.PaymentMethod, bool) {
			scenario := map[conformance.Outcome]string{
				conformance.OutcomeSettled:    mock.ScenarioSuccess,
				conformance.OutcomeAuthorized: mock.ScenarioSuccess,
				conformance.OutcomeDeclined:   mock.ScenarioDecline,
				conformance.OutcomePending:    mock.ScenarioAsync,
			}[o]
			return connectors.PaymentMethod{Type: connectors.MethodCard, Token: scenario}, scenario != ""
		},
		Classify: func(raw connectors.RawStatus) conformance.Class {
			switch raw {
			case mock.StatusCaptured, mock.StatusPartiallyCaptured:
				return conformance.ClassSettled
			case mock.StatusAuthorized:
				return conformance.ClassAuthorized
			case mock.StatusPending:
				return conformance.ClassPending
			case mock.StatusActionRequired:
				return conformance.ClassActionRequired
			case mock.StatusVoided:
				return conformance.ClassVoided
			default:
				return conformance.ClassFailed
			}
		},
		Settle: func(t *testing.T, c connectors.Connector, _ string, txID string) {
			if err := c.(*mock.Connector).Settle(txID, mock.StatusCaptured); err != nil {
				t.Fatalf("settle: %v", err)
			}
		},
		Webhook: func(t *testing.T, c connectors.Connector, eventID, txID string) (http.Header, []byte) {
			return c.(*mock.Connector).SignWebhook(mock.Event{EventID: eventID, TransactionID: txID, Status: string(mock.StatusCaptured)})
		},
		StaleWebhook: func(t *testing.T, c connectors.Connector, eventID, txID string) (http.Header, []byte) {
			return c.(*mock.Connector).SignWebhookAt(mock.Event{EventID: eventID, TransactionID: txID, Status: string(mock.StatusCaptured)}, time.Now().Add(-mock.WebhookWindow-time.Minute))
		},
		SettleRefund: func(t *testing.T, c connectors.Connector, refundID string) {
			if err := c.(*mock.Connector).SettleRefund(refundID, mock.RefundDone); err != nil {
				t.Fatalf("settle refund: %v", err)
			}
		},
		PendingRefundReason: mock.ScenarioRefundAsync,
	})
}
