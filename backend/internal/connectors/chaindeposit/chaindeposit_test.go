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

func classify(raw connectors.RawStatus) conformance.Class {
	switch raw {
	case chaindeposit.StatusFilled, chaindeposit.StatusOverFilled:
		return conformance.ClassSettled
	case chaindeposit.StatusOpen, chaindeposit.StatusPartiallyFilled:
		return conformance.ClassActionRequired
	case chaindeposit.StatusCancelled, chaindeposit.StatusCancelledUnderpaid:
		return conformance.ClassVoided
	default:
		return conformance.ClassFailed
	}
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
		Classify: classify,
		Settle: func(t *testing.T, _ connectors.Connector, _ string, ref string) {
			if err := backend.Deposit(ref, decimal.NewFromInt(50)); err != nil {
				t.Fatalf("deposit: %v", err)
			}
		},
	})
}

func TestMemoryBackendSuite(t *testing.T) {
	chaindeposit.RunBackendSuite(t, chaindeposit.BackendHarness{
		New: func(t *testing.T) chaindeposit.Backend { return chaindeposit.NewMemoryBackend() },
		Confirm: func(t *testing.T, b chaindeposit.Backend, ref string, amount decimal.Decimal) {
			if err := b.(*chaindeposit.MemoryBackend).Deposit(ref, amount); err != nil {
				t.Fatal(err)
			}
		},
	})
}

func TestChainDeposit_UnderAndOverPaymentAreExplicitStatuses(t *testing.T) {
	backend := chaindeposit.NewMemoryBackend()
	c := chaindeposit.New(backend)
	ctx := context.Background()
	resp, err := c.Authorize(ctx, connectors.AuthorizeRequest{AttemptID: "pa_1", MerchantID: "7", PlatformID: "3", Money: connectors.Money{Amount: decimal.NewFromInt(100), Asset: "USD"}, CaptureMethod: connectors.CaptureAutomatic, PaymentMethod: usdc()})
	if err != nil || resp.RawStatus != chaindeposit.StatusOpen || resp.NextAction == nil || resp.NextAction.Address == "" || resp.NextAction.Type != "pay_to_address" || resp.NextAction.ExpiresAt == "" {
		t.Fatalf("authorize = %+v, %v", resp, err)
	}
	// Sync by our attempt id alone, as after a crash before the response was stored.
	sync, err := c.Sync(ctx, connectors.SyncRequest{AttemptID: "pa_1"})
	if err != nil || sync.ConnectorTransactionID != resp.ConnectorTransactionID || sync.RawStatus != chaindeposit.StatusOpen || sync.NextAction == nil || sync.NextAction.Address != resp.NextAction.Address {
		t.Fatalf("sync by attempt = %+v, %v; an open deposit keeps its address", sync, err)
	}
	if err := backend.Deposit(resp.ConnectorTransactionID, decimal.NewFromInt(40)); err != nil {
		t.Fatal(err)
	}
	sync, err = c.Sync(ctx, connectors.SyncRequest{AttemptID: "pa_1", ConnectorTransactionID: resp.ConnectorTransactionID})
	if err != nil || sync.RawStatus != chaindeposit.StatusPartiallyFilled || sync.AmountReceived == nil || !sync.AmountReceived.Equal(decimal.NewFromInt(40)) || sync.ReceivedAsset != "USDC.ETH" || sync.NextAction == nil {
		t.Fatalf("under-payment sync = %+v, %v", sync, err)
	}
	v, err := c.Void(ctx, connectors.VoidRequest{ConnectorTransactionID: resp.ConnectorTransactionID})
	if err != nil || v.RawStatus != chaindeposit.StatusCancelledUnderpaid || v.AmountReceived == nil || !v.AmountReceived.Equal(decimal.NewFromInt(40)) || v.ReceivedAsset != "USDC.ETH" {
		t.Fatalf("void of a partially filled request = %+v, %v", v, err)
	}
	sync, err = c.Sync(ctx, connectors.SyncRequest{ConnectorTransactionID: resp.ConnectorTransactionID})
	if err != nil || sync.RawStatus != chaindeposit.StatusCancelledUnderpaid || !sync.AmountReceived.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("sync after underpaid cancel = %+v, %v", sync, err)
	}

	over, _ := c.Authorize(ctx, connectors.AuthorizeRequest{AttemptID: "pa_2", MerchantID: "7", PlatformID: "3", Money: connectors.Money{Amount: decimal.NewFromInt(100), Asset: "USD"}, CaptureMethod: connectors.CaptureAutomatic, PaymentMethod: usdc()})
	if err := backend.Deposit(over.ConnectorTransactionID, decimal.NewFromInt(110)); err != nil {
		t.Fatal(err)
	}
	sync, err = c.Sync(ctx, connectors.SyncRequest{ConnectorTransactionID: over.ConnectorTransactionID})
	if err != nil || sync.RawStatus != chaindeposit.StatusOverFilled || !sync.AmountReceived.Equal(decimal.NewFromInt(110)) || sync.NextAction != nil {
		t.Fatalf("over-payment sync = %+v, %v", sync, err)
	}
	if _, err := c.Void(ctx, connectors.VoidRequest{ConnectorTransactionID: over.ConnectorTransactionID}); !errors.Is(err, connectors.ErrInvalidRequest) {
		t.Fatalf("void after fill err = %v", err)
	}
}

// A fill that wins the race against a cancel is reported as filled; a cancelled request that later receives its
// full amount is cancelled (with the money reported), never underpaid.
func TestChainDeposit_VoidReReadsAndPaidAfterCancelIsNotUnderpaid(t *testing.T) {
	backend := chaindeposit.NewMemoryBackend()
	c := chaindeposit.New(backend)
	ctx := context.Background()
	resp, _ := c.Authorize(ctx, connectors.AuthorizeRequest{AttemptID: "pa_1", MerchantID: "7", PlatformID: "3", Money: connectors.Money{Amount: decimal.NewFromInt(100), Asset: "USD"}, CaptureMethod: connectors.CaptureAutomatic, PaymentMethod: usdc()})
	v, err := c.Void(ctx, connectors.VoidRequest{ConnectorTransactionID: resp.ConnectorTransactionID})
	if err != nil || v.RawStatus != chaindeposit.StatusCancelled || v.AmountReceived != nil {
		t.Fatalf("void = %+v, %v", v, err)
	}
	if err := backend.Deposit(resp.ConnectorTransactionID, decimal.NewFromInt(100)); err != nil {
		t.Fatal(err)
	}
	sync, err := c.Sync(ctx, connectors.SyncRequest{ConnectorTransactionID: resp.ConnectorTransactionID})
	if err != nil || sync.RawStatus != chaindeposit.StatusCancelled || sync.AmountReceived == nil || !sync.AmountReceived.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("paid after cancel = %+v, %v; must be cancelled with the money, not underpaid", sync, err)
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
	eth := base
	eth.PaymentMethod = connectors.PaymentMethod{Type: connectors.MethodChain, Token: "eth", Details: map[string]string{chaindeposit.DetailChain: "ETH", chaindeposit.DetailAsset: "ETH"}}
	if _, err := c.Authorize(ctx, eth); !errors.Is(err, connectors.ErrUnsupported) {
		t.Fatalf("a volatile asset priced in USD must be refused until conversion exists, err = %v", err)
	}
	badMerchant := base
	badMerchant.MerchantID = "merchant-1"
	if _, err := c.Authorize(ctx, badMerchant); !errors.Is(err, connectors.ErrInvalidRequest) {
		t.Fatalf("non-numeric merchant err = %v", err)
	}
	noAttempt := base
	noAttempt.AttemptID = ""
	if _, err := c.Authorize(ctx, noAttempt); !errors.Is(err, connectors.ErrInvalidRequest) {
		t.Fatalf("missing attempt id err = %v", err)
	}
	if _, err := c.Sync(ctx, connectors.SyncRequest{ConnectorTransactionID: "missing"}); !errors.Is(err, connectors.ErrNotFound) {
		t.Fatalf("sync missing err = %v", err)
	}
	if _, err := c.SyncRefund(ctx, connectors.SyncRefundRequest{RefundID: "x"}); !errors.Is(err, connectors.ErrUnsupported) {
		t.Fatalf("sync refund err = %v", err)
	}
}
