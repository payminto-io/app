package paymentlifecycle

import (
	"context"
	"fmt"
	"sync"
)

type MemoryQuoteAdapter struct {
	quotes []Quote
}

var _ QuoteAdapter = (*MemoryQuoteAdapter)(nil)

func NewMemoryQuoteAdapter(quotes []Quote) *MemoryQuoteAdapter {
	return &MemoryQuoteAdapter{quotes: append([]Quote(nil), quotes...)}
}

func (a *MemoryQuoteAdapter) Quote(ctx context.Context, req QuoteRequest) (Quote, error) {
	if err := ctx.Err(); err != nil {
		return Quote{}, err
	}
	for _, quote := range a.quotes {
		if quote.InvoiceCurrency == req.InvoiceAmount.Currency &&
			quote.InvoiceMinorUnits == req.InvoiceAmount.MinorUnits &&
			quote.ChainID == req.PaymentMethod.ChainID &&
			quote.AssetID == req.PaymentMethod.AssetID {
			return quote, nil
		}
	}
	return Quote{}, ErrQuoteUnavailable
}

type memoryReceipt struct {
	hash string
	view PaymentView
}

type MemoryStore struct {
	mu         sync.Mutex
	addresses  []AvailableDepositAddress
	receipts   map[string]memoryReceipt
	references map[string]struct{}
	events     []LifecycleEventSpec
	outbox     []OutboxEventSpec
	next       uint64
}

func (s *MemoryStore) LookupReceipt(ctx context.Context, lookup ReceiptLookup) (ReceiptResult, error) {
	if err := ctx.Err(); err != nil {
		return ReceiptResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if lookup.Operation != OperationOpen || !requestHashPattern.MatchString(lookup.RequestHash) {
		return ReceiptResult{}, failure(CodeStorageUnavailable, "receipt.lookup", nil)
	}
	receiptKey := memoryReceiptKey(lookup.TenantID, lookup.Operation, lookup.IdempotencyKey)
	receipt, ok := s.receipts[receiptKey]
	if !ok {
		return ReceiptResult{Disposition: ReceiptMiss}, nil
	}
	if receipt.hash != lookup.RequestHash {
		return ReceiptResult{}, ErrIdempotencyConflict
	}
	return ReceiptResult{Disposition: ReceiptReplayed, View: receipt.view}, nil
}

var _ Store = (*MemoryStore)(nil)

func NewMemoryStore(addresses []AvailableDepositAddress) *MemoryStore {
	return &MemoryStore{
		addresses:  append([]AvailableDepositAddress(nil), addresses...),
		receipts:   make(map[string]memoryReceipt),
		references: make(map[string]struct{}),
	}
}

func (s *MemoryStore) Open(ctx context.Context, record OpenRecord) (StoreResult, error) {
	if err := ctx.Err(); err != nil {
		return StoreResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return StoreResult{}, err
	}
	if record.Operation != OperationOpen || !requestHashPattern.MatchString(record.RequestHash) {
		return StoreResult{}, failure(CodeStorageUnavailable, "store.record", nil)
	}

	receiptKey := memoryReceiptKey(record.Command.TenantID, record.Operation, record.Command.IdempotencyKey)
	if receipt, ok := s.receipts[receiptKey]; ok {
		if receipt.hash != record.RequestHash {
			return StoreResult{}, ErrIdempotencyConflict
		}
		return StoreResult{Disposition: StoreReplayed, View: receipt.view}, nil
	}
	referenceKey := string(record.Command.TenantID) + "\x00" + string(record.Command.MerchantReference)
	if _, exists := s.references[referenceKey]; exists {
		return StoreResult{}, ErrMerchantReferenceConflict
	}
	addressIndex := -1
	for i, candidate := range s.addresses {
		if candidate.TenantID == record.Command.TenantID && candidate.ChainID == record.Command.PaymentMethod.ChainID && candidate.AssetID == record.Command.PaymentMethod.AssetID && visibleASCII(candidate.Address, 1, 512) {
			addressIndex = i
			break
		}
	}
	if addressIndex < 0 {
		return StoreResult{}, ErrDepositAddressUnavailable
	}
	address := s.addresses[addressIndex]
	s.addresses = append(s.addresses[:addressIndex], s.addresses[addressIndex+1:]...)
	s.next++
	view := PaymentView{
		InvoiceID:         InvoiceID(fmt.Sprintf("invoice-%06d", s.next)),
		TenantID:          record.Command.TenantID,
		MerchantReference: record.Command.MerchantReference,
		InvoiceAmount:     record.Command.InvoiceAmount,
		PaymentMethod:     record.Command.PaymentMethod,
		Quote:             record.Quote,
		DepositAddress: DepositAddress{
			AssignmentID: AddressAssignmentID(fmt.Sprintf("assignment-%06d", s.next)),
			Address:      address.Address,
		},
		State:     record.State,
		Revision:  record.Revision,
		OpenedAt:  record.OpenedAt,
		ExpiresAt: record.Command.ExpiresAt,
	}
	s.receipts[receiptKey] = memoryReceipt{hash: record.RequestHash, view: view}
	s.references[referenceKey] = struct{}{}
	s.events = append(s.events, record.LifecycleEvent)
	s.outbox = append(s.outbox, record.OutboxEvent)
	return StoreResult{Disposition: StoreCreated, View: view}, nil
}

func memoryReceiptKey(tenantID TenantID, operation string, key IdempotencyKey) string {
	return string(tenantID) + "\x00" + operation + "\x00" + string(key)
}

func (s *MemoryStore) Snapshot() MemoryStoreSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := len(s.receipts)
	return MemoryStoreSnapshot{
		Invoices: count, AddressAssignments: count, LifecycleEvents: len(s.events),
		OutboxEvents: len(s.outbox), Receipts: count,
		LifecycleEventSpecs: append([]LifecycleEventSpec(nil), s.events...),
		OutboxEventSpecs:    append([]OutboxEventSpec(nil), s.outbox...),
	}
}
