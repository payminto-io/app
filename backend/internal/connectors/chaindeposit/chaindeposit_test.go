package chaindeposit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/chaindeposit"
	"github.com/payminto/payminto/backend/internal/connectors/conformance"
	"github.com/shopspring/decimal"
)

func usdc() connectors.PaymentMethod {
	return connectors.PaymentMethod{Type: connectors.MethodChain, Token: "usdc", Details: map[string]string{chaindeposit.DetailChain: "ETH", chaindeposit.DetailAsset: "USDC"}}
}

func TestChainDepositConformance(t *testing.T) {
	var backend *chaindeposit.MemoryBackend
	conformance.Run(t, conformance.Harness{
		New: func(t *testing.T) connectors.Connector {
			backend = chaindeposit.NewMemoryBackend()
			return chaindeposit.New(backend)
		},
		Method: connectors.MethodChain,
		Money:  connectors.Money{Amount: decimal.NewFromInt(50), Asset: chaindeposit.PricingAsset},
		PaymentMethod: func(o conformance.Outcome) (connectors.PaymentMethod, bool) {
			return usdc(), o == conformance.OutcomePending
		},
		Classify: func(raw connectors.RawStatus) conformance.Class {
			switch raw {
			case chaindeposit.StatusFilled, chaindeposit.StatusOverFilled:
				return conformance.ClassSettled
			case chaindeposit.StatusOpen, chaindeposit.StatusPartiallyFilled:
				return conformance.ClassPending
			case chaindeposit.StatusCancelled:
				return conformance.ClassVoided
			default:
				return conformance.ClassFailed
			}
		},
		Settle: func(t *testing.T, _ connectors.Connector, _ string, ref string) {
			if err := backend.Deposit(ref, decimal.NewFromInt(50)); err != nil {
				t.Fatalf("deposit: %v", err)
			}
		},
	})
}

func TestChainDeposit_UnderAndOverPaymentAreExplicitStatuses(t *testing.T) {
	backend := chaindeposit.NewMemoryBackend()
	c := chaindeposit.New(backend)
	ctx := context.Background()
	resp, err := c.Authorize(ctx, connectors.AuthorizeRequest{AttemptID: "pa_1", MerchantID: "7", PlatformID: "3", Money: connectors.Money{Amount: decimal.NewFromInt(100), Asset: "USD"}, CaptureMethod: connectors.CaptureAutomatic, PaymentMethod: usdc()})
	if err != nil || resp.RawStatus != chaindeposit.StatusOpen || resp.NextAction == nil || resp.NextAction.Address == "" || resp.NextAction.Type != "pay_to_address" {
		t.Fatalf("authorize = %+v, %v", resp, err)
	}
	if err := backend.Deposit(resp.ConnectorTransactionID, decimal.NewFromInt(40)); err != nil {
		t.Fatal(err)
	}
	sync, err := c.Sync(ctx, connectors.SyncRequest{AttemptID: "pa_1", ConnectorTransactionID: resp.ConnectorTransactionID})
	if err != nil || sync.RawStatus != chaindeposit.StatusPartiallyFilled || sync.AmountReceived == nil || !sync.AmountReceived.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("under-payment sync = %+v, %v", sync, err)
	}
	if _, err := c.Void(ctx, connectors.VoidRequest{ConnectorTransactionID: resp.ConnectorTransactionID}); !errors.Is(err, connectors.ErrInvalidRequest) {
		t.Fatalf("void after funds arrived err = %v", err)
	}
	if err := backend.Deposit(resp.ConnectorTransactionID, decimal.NewFromInt(70)); err != nil {
		t.Fatal(err)
	}
	sync, err = c.Sync(ctx, connectors.SyncRequest{ConnectorTransactionID: resp.ConnectorTransactionID})
	if err != nil || sync.RawStatus != chaindeposit.StatusOverFilled || !sync.AmountReceived.Equal(decimal.NewFromInt(110)) {
		t.Fatalf("over-payment sync = %+v, %v", sync, err)
	}
}

func TestChainDeposit_RejectsWhatPaymintoCannotDo(t *testing.T) {
	c := chaindeposit.New(chaindeposit.NewMemoryBackend())
	ctx := context.Background()
	base := connectors.AuthorizeRequest{AttemptID: "pa_1", MerchantID: "7", PlatformID: "3", Money: connectors.Money{Amount: decimal.NewFromInt(100), Asset: "USD"}, CaptureMethod: connectors.CaptureAutomatic, PaymentMethod: usdc()}

	eur := base
	eur.Money.Asset = "EUR"
	if _, err := c.Authorize(ctx, eur); !errors.Is(err, connectors.ErrInvalidRequest) {
		t.Fatalf("non-USD err = %v", err)
	}
	manual := base
	manual.CaptureMethod = connectors.CaptureManual
	if _, err := c.Authorize(ctx, manual); !errors.Is(err, connectors.ErrUnsupported) {
		t.Fatalf("manual capture err = %v", err)
	}
	noChain := base
	noChain.PaymentMethod = connectors.PaymentMethod{Type: connectors.MethodChain, Token: "usdc"}
	if _, err := c.Authorize(ctx, noChain); !errors.Is(err, connectors.ErrInvalidRequest) {
		t.Fatalf("missing details err = %v", err)
	}
	badMerchant := base
	badMerchant.MerchantID = "merchant-1"
	if _, err := c.Authorize(ctx, badMerchant); !errors.Is(err, connectors.ErrInvalidRequest) {
		t.Fatalf("non-numeric merchant err = %v", err)
	}
	if _, err := c.Sync(ctx, connectors.SyncRequest{ConnectorTransactionID: "missing"}); !errors.Is(err, connectors.ErrNotFound) {
		t.Fatalf("sync missing err = %v", err)
	}
}
