package chaindeposit

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

// BackendHarness drives one Backend implementation through RunBackendSuite. Confirm books a confirmed deposit
// against a reference the way Payminto's block processors and finalizer would.
type BackendHarness struct {
	New     func(t *testing.T) Backend
	Confirm func(t *testing.T, b Backend, reference string, amount decimal.Decimal)
}

// RunBackendSuite is the contract both the memory double and the Payminto adapter must satisfy, so the tests
// cannot prove a lookup production cannot do.
func RunBackendSuite(t *testing.T, h BackendHarness) {
	t.Helper()
	ctx := context.Background()
	open := func(t *testing.T, b Backend, attempt string) OpenResult {
		t.Helper()
		res, err := b.OpenPayment(ctx, OpenRequest{MerchantMemberID: 7, PlatformID: 3, AmountInUSD: decimal.NewFromInt(100), ChainCode: "ETH", CurrencyCode: "USDC", AttemptID: attempt})
		if err != nil {
			t.Fatalf("OpenPayment: %v", err)
		}
		return res
	}

	t.Run("open picks its own reference and keeps the attempt id", func(t *testing.T) {
		b := h.New(t)
		res := open(t, b, "pa_suite_1")
		if res.Reference == "" || res.Reference == "pa_suite_1" || res.Address == "" {
			t.Fatalf("open = %+v; the backend, not the caller, owns the reference", res)
		}
		byRef, err := b.PaymentStatus(ctx, res.Reference)
		if err != nil || byRef.State != "OPEN" || byRef.Reference != res.Reference || !byRef.AmountInUSD.Equal(decimal.NewFromInt(100)) {
			t.Fatalf("status by reference = %+v, %v", byRef, err)
		}
		byAttempt, err := b.PaymentStatusByAttempt(ctx, "pa_suite_1")
		if err != nil || byAttempt.Reference != res.Reference {
			t.Fatalf("status by attempt = %+v, %v; must find the request the way a crashed switch would", byAttempt, err)
		}
		if byRef.CurrencyCode != "USDC" || byRef.ChainCode != "ETH" || byRef.Address != res.Address {
			t.Fatalf("status must carry chain, currency and address: %+v", byRef)
		}
		if _, err := b.OpenPayment(ctx, OpenRequest{MerchantMemberID: 7, PlatformID: 3, AmountInUSD: decimal.NewFromInt(1), ChainCode: "ETH", CurrencyCode: "USDC"}); err == nil {
			t.Fatal("open without an attempt id must fail")
		}
	})

	t.Run("unknown references", func(t *testing.T) {
		b := h.New(t)
		if _, err := b.PaymentStatus(ctx, "nope"); !errors.Is(err, ErrBackendNotFound) {
			t.Fatalf("status err = %v", err)
		}
		if _, err := b.PaymentStatusByAttempt(ctx, "nope"); !errors.Is(err, ErrBackendNotFound) {
			t.Fatalf("status by attempt err = %v", err)
		}
		if err := b.CancelPayment(ctx, "nope"); !errors.Is(err, ErrBackendNotFound) {
			t.Fatalf("cancel err = %v", err)
		}
	})

	t.Run("deposits move the state like the finalizer", func(t *testing.T) {
		b := h.New(t)
		res := open(t, b, "pa_suite_2")
		h.Confirm(t, b, res.Reference, decimal.NewFromInt(40))
		st, _ := b.PaymentStatus(ctx, res.Reference)
		if st.State != "PARTIALLY_FILLED" || !st.Received.Equal(decimal.NewFromInt(40)) {
			t.Fatalf("after 40 = %+v", st)
		}
		h.Confirm(t, b, res.Reference, decimal.NewFromInt(60))
		st, _ = b.PaymentStatus(ctx, res.Reference)
		if st.State != "FILLED" || !st.Received.Equal(decimal.NewFromInt(100)) {
			t.Fatalf("after 100 = %+v", st)
		}
		if err := b.CancelPayment(ctx, res.Reference); !errors.Is(err, ErrNotCancellable) {
			t.Fatalf("a filled request cannot be cancelled, err = %v", err)
		}
		over := open(t, b, "pa_suite_3")
		h.Confirm(t, b, over.Reference, decimal.NewFromInt(150))
		st, _ = b.PaymentStatus(ctx, over.Reference)
		if st.State != "OVER_FILLED" || !st.Received.Equal(decimal.NewFromInt(150)) {
			t.Fatalf("after 150 = %+v", st)
		}
	})

	t.Run("money after cancel is still reported", func(t *testing.T) {
		b := h.New(t)
		res := open(t, b, "pa_suite_6")
		if err := b.CancelPayment(ctx, res.Reference); err != nil {
			t.Fatal(err)
		}
		h.Confirm(t, b, res.Reference, decimal.NewFromInt(100))
		st, err := b.PaymentStatus(ctx, res.Reference)
		if err != nil || st.State != "CANCELLED" || !st.Received.Equal(decimal.NewFromInt(100)) {
			t.Fatalf("late deposit on a cancelled request must stay visible: %+v, %v", st, err)
		}
	})

	t.Run("cancel open and partially filled, never twice", func(t *testing.T) {
		b := h.New(t)
		res := open(t, b, "pa_suite_4")
		if err := b.CancelPayment(ctx, res.Reference); err != nil {
			t.Fatalf("cancel open: %v", err)
		}
		if st, _ := b.PaymentStatus(ctx, res.Reference); st.State != "CANCELLED" {
			t.Fatalf("state = %s", st.State)
		}
		if err := b.CancelPayment(ctx, res.Reference); !errors.Is(err, ErrNotCancellable) {
			t.Fatalf("cancelling twice err = %v, want ErrNotCancellable", err)
		}
		partial := open(t, b, "pa_suite_5")
		h.Confirm(t, b, partial.Reference, decimal.NewFromInt(10))
		if err := b.CancelPayment(ctx, partial.Reference); err != nil {
			t.Fatalf("cancel partially filled: %v", err)
		}
		st, _ := b.PaymentStatus(ctx, partial.Reference)
		if st.State != "CANCELLED" || !st.Received.Equal(decimal.NewFromInt(10)) {
			t.Fatalf("cancelled underpaid = %+v; received funds must stay visible", st)
		}
	})
}
