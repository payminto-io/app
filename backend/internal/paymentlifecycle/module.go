package paymentlifecycle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

type Clock func() time.Time

type Module struct {
	quotes QuoteAdapter
	store  Store
	now    Clock
}

var _ PaymentLifecycle = (*Module)(nil)

func New(quotes QuoteAdapter, store Store, now Clock) *Module {
	return &Module{quotes: quotes, store: store, now: now}
}

func (m *Module) Open(ctx context.Context, cmd OpenPayment) (PaymentView, error) {
	if m == nil || m.now == nil {
		return PaymentView{}, failure(CodeStorageUnavailable, "module.clock", nil)
	}
	now := m.now().UTC()
	if err := validateCommandStructure(cmd); err != nil {
		return PaymentView{}, err
	}
	if m.store == nil {
		return PaymentView{}, failure(CodeStorageUnavailable, "store", nil)
	}
	requestHash := canonicalRequestHash(cmd)
	receipt, err := m.store.LookupReceipt(ctx, ReceiptLookup{
		TenantID: cmd.TenantID, Operation: OperationOpen,
		IdempotencyKey: cmd.IdempotencyKey, RequestHash: requestHash,
	})
	if err != nil {
		if errors.Is(err, ErrIdempotencyConflict) {
			return PaymentView{}, err
		}
		return PaymentView{}, failure(CodeStorageUnavailable, "receipt", err)
	}
	switch receipt.Disposition {
	case ReceiptReplayed:
		result := StoreResult{Disposition: StoreReplayed, View: receipt.View}
		if err := validateStoreResult(result, cmd, Quote{}, now); err != nil {
			return PaymentView{}, err
		}
		return receipt.View, nil
	case ReceiptMiss:
		// Continue into the atomic create path. Store.Open repeats the receipt
		// check under its transaction lock to close the lookup/create race.
	default:
		return PaymentView{}, failure(CodeStorageUnavailable, "receipt.disposition", nil)
	}
	if err := validateCreateCommand(cmd, now); err != nil {
		return PaymentView{}, err
	}
	if m.quotes == nil {
		return PaymentView{}, failure(CodeQuoteUnavailable, "quote", nil)
	}
	quote, err := m.quotes.Quote(ctx, QuoteRequest{
		TenantID: cmd.TenantID, InvoiceAmount: cmd.InvoiceAmount,
		PaymentMethod: cmd.PaymentMethod, InvoiceExpiry: cmd.ExpiresAt,
		RequestedAt: now,
	})
	if err != nil {
		if errors.Is(err, ErrUnsupportedPaymentMethod) || errors.Is(err, ErrQuoteUnavailable) {
			return PaymentView{}, err
		}
		return PaymentView{}, failure(CodeQuoteUnavailable, "quote", err)
	}
	if err := validateQuote(quote, cmd, now); err != nil {
		return PaymentView{}, err
	}
	result, err := m.store.Open(ctx, OpenRecord{
		Operation: OperationOpen, RequestHash: requestHash,
		Command: cmd, Quote: quote, State: InvoiceOpen, Revision: 1,
		LifecycleEvent: LifecycleEventSpec{Type: EventInvoiceOpen, Revision: 1, OccurredAt: now},
		OutboxEvent:    OutboxEventSpec{Type: EventInvoiceOpen, SchemaVersion: 1, OccurredAt: now},
		OpenedAt:       now,
	})
	if err != nil {
		if errors.Is(err, ErrIdempotencyConflict) || errors.Is(err, ErrMerchantReferenceConflict) || errors.Is(err, ErrDepositAddressUnavailable) {
			return PaymentView{}, err
		}
		return PaymentView{}, failure(CodeStorageUnavailable, "store", err)
	}
	if err := validateStoreResult(result, cmd, quote, now); err != nil {
		return PaymentView{}, err
	}
	return result.View, nil
}

func canonicalRequestHash(cmd OpenPayment) string {
	h := sha256.New()
	write := func(value string) {
		_, _ = h.Write([]byte(strconv.Itoa(len(value))))
		_, _ = h.Write([]byte{':'})
		_, _ = h.Write([]byte(value))
		_, _ = h.Write([]byte{'|'})
	}
	write("paymentlifecycle.open.v1")
	write(string(cmd.TenantID))
	write(string(cmd.IdempotencyKey))
	write(string(cmd.MerchantReference))
	write(cmd.InvoiceAmount.Currency)
	write(cmd.InvoiceAmount.MinorUnits)
	write(string(cmd.PaymentMethod.ChainID))
	write(string(cmd.PaymentMethod.AssetID))
	write(cmd.ExpiresAt.UTC().Format(time.RFC3339Nano))
	return strings.ToLower(hex.EncodeToString(h.Sum(nil)))
}
