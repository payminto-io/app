//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/paymentlifecycle"
	"gorm.io/gorm"
)

func TestStoreOpenPersistsAndReplaysOneAtomicResult(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
	store := New(db)
	record := validRecord()

	created, err := store.Open(context.Background(), record)
	if err != nil {
		t.Fatalf("Open(created) error = %v (cause: %v)", err, errors.Unwrap(err))
	}
	if created.Disposition != paymentlifecycle.StoreCreated {
		t.Fatalf("created disposition = %q", created.Disposition)
	}
	replayed, err := store.Open(context.Background(), record)
	if err != nil {
		t.Fatalf("Open(replayed) error = %v", err)
	}
	if replayed.Disposition != paymentlifecycle.StoreReplayed || replayed.View != created.View {
		t.Fatalf("replayed result = %#v, want identical %#v", replayed, created.View)
	}
	assertEffectCounts(t, db, 1)
}

func TestStoreLookupReceiptDistinguishesMissReplayAndConflict(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
	store := New(db)
	record := validRecord()
	lookup := paymentlifecycle.ReceiptLookup{
		TenantID: record.Command.TenantID, Operation: record.Operation,
		IdempotencyKey: record.Command.IdempotencyKey, RequestHash: record.RequestHash,
	}

	miss, err := store.LookupReceipt(context.Background(), lookup)
	if err != nil || miss.Disposition != paymentlifecycle.ReceiptMiss {
		t.Fatalf("LookupReceipt(miss) = %#v, %v", miss, err)
	}
	created, err := store.Open(context.Background(), record)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	replayed, err := store.LookupReceipt(context.Background(), lookup)
	if err != nil || replayed.Disposition != paymentlifecycle.ReceiptReplayed || replayed.View != created.View {
		t.Fatalf("LookupReceipt(replay) = %#v, %v; want %#v", replayed, err, created.View)
	}
	lookup.RequestHash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := store.LookupReceipt(context.Background(), lookup); !errors.Is(err, paymentlifecycle.ErrIdempotencyConflict) {
		t.Fatalf("LookupReceipt(conflict) error = %v", err)
	}
}

func TestStoreRejectsInvalidReceiptScopeBeforeQuery(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	store := New(db)
	valid := paymentlifecycle.ReceiptLookup{
		TenantID: "tenant-acme", Operation: paymentlifecycle.OperationOpen,
		IdempotencyKey: "checkout-20260827-0001",
		RequestHash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	tests := []struct {
		name  string
		alter func(*paymentlifecycle.ReceiptLookup)
	}{
		{"unsupported operation", func(v *paymentlifecycle.ReceiptLookup) { v.Operation = "payment_lifecycle.apply" }},
		{"malformed hash", func(v *paymentlifecycle.ReceiptLookup) { v.RequestHash = "ABC" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lookup := valid
			tc.alter(&lookup)
			if _, err := store.LookupReceipt(context.Background(), lookup); !errors.Is(err, paymentlifecycle.ErrStorageUnavailable) {
				t.Fatalf("LookupReceipt() error = %v, want storage unavailable", err)
			}
		})
	}
}

func TestStoreOpenSerializesConcurrentIdempotentCommands(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
	store := New(db)
	record := validRecord()

	start := make(chan struct{})
	results := make(chan paymentlifecycle.StoreResult, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := store.Open(context.Background(), record)
			results <- result
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Open() error = %v", err)
		}
	}
	created, replayed := 0, 0
	for result := range results {
		switch result.Disposition {
		case paymentlifecycle.StoreCreated:
			created++
		case paymentlifecycle.StoreReplayed:
			replayed++
		}
	}
	if created != 1 || replayed != 1 {
		t.Fatalf("created=%d replayed=%d, want 1 and 1", created, replayed)
	}
	assertEffectCounts(t, db, 1)
}

func TestStoreOpenSerializesConcurrentDifferentKeysForOneMerchantReference(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
	seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x2222222222222222222222222222222222222222")
	store := New(db)
	first := validRecord()
	second := validRecord()
	second.Command.IdempotencyKey = "checkout-20260827-0002"
	second.RequestHash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	second.Quote.ID = "quote-0002"

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, record := range []paymentlifecycle.OpenRecord{first, second} {
		wg.Add(1)
		go func(record paymentlifecycle.OpenRecord) {
			defer wg.Done()
			<-start
			_, err := store.Open(context.Background(), record)
			errs <- err
		}(record)
	}
	close(start)
	wg.Wait()
	close(errs)
	created, conflicts := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			created++
		case errors.Is(err, paymentlifecycle.ErrMerchantReferenceConflict):
			conflicts++
		default:
			t.Fatalf("Open() unexpected error = %v (cause: %v)", err, errors.Unwrap(err))
		}
	}
	if created != 1 || conflicts != 1 {
		t.Fatalf("created=%d conflicts=%d, want 1 and 1", created, conflicts)
	}
	assertEffectCounts(t, db, 1)
}

func TestStoreOpenRejectsChangedRequestForSameIdempotencyKey(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
	store := New(db)
	record := validRecord()
	if _, err := store.Open(context.Background(), record); err != nil {
		t.Fatalf("first Open() error = %v", err)
	}
	record.RequestHash = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := store.Open(context.Background(), record); !errors.Is(err, paymentlifecycle.ErrIdempotencyConflict) {
		t.Fatalf("changed Open() error = %v, want idempotency conflict", err)
	}
	assertEffectCounts(t, db, 1)
}

func TestStoreOpenScopesKeyAndReferenceByTenant(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
	seedAddress(t, db, "tenant-beta", "eip155:1", usdtAsset, "0x2222222222222222222222222222222222222222")
	store := New(db)
	record := validRecord()
	if _, err := store.Open(context.Background(), record); err != nil {
		t.Fatalf("tenant A Open() error = %v", err)
	}
	record.Command.TenantID = "tenant-beta"
	if _, err := store.Open(context.Background(), record); err != nil {
		t.Fatalf("tenant B Open() error = %v", err)
	}
	assertEffectCounts(t, db, 2)
}

func TestStoreOpenWithNoAddressLeavesNoPartialInvoice(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	store := New(db)

	if _, err := store.Open(context.Background(), validRecord()); !errors.Is(err, paymentlifecycle.ErrDepositAddressUnavailable) {
		t.Fatalf("Open() error = %v, want address unavailable", err)
	}
	assertEffectCounts(t, db, 0)
}

func TestStoreOpenCannotAllocateAnotherTenantsAddress(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	seedAddress(t, db, "tenant-beta", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
	store := New(db)
	if _, err := store.Open(context.Background(), validRecord()); !errors.Is(err, paymentlifecycle.ErrDepositAddressUnavailable) {
		t.Fatalf("Open() error = %v, want address unavailable", err)
	}
	assertEffectCounts(t, db, 0)
}

func TestDepositAddressIdentityCannotBeDuplicatedAcrossTenants(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
	if err := db.Exec(`INSERT INTO payment_lifecycle_deposit_addresses
        (tenant_id, chain_id, asset_id, address) VALUES (?, ?, ?, ?)`,
		"tenant-beta", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111").Error; err == nil {
		t.Fatal("cross-tenant duplicate chain/asset/address was accepted")
	}
}

func TestStoreOpenEnforcesAssetDecimalDomain(t *testing.T) {
	for _, tc := range []struct {
		decimals uint8
		wantOK   bool
	}{{0, true}, {77, true}, {78, false}} {
		t.Run(fmt.Sprintf("decimals_%d", tc.decimals), func(t *testing.T) {
			db, cleanup := openMigratedDB(t)
			defer cleanup()
			seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
			record := validRecord()
			record.Quote.AssetDecimals = tc.decimals
			_, err := New(db).Open(context.Background(), record)
			if tc.wantOK && err != nil {
				t.Fatalf("Open() error = %v (cause: %v)", err, errors.Unwrap(err))
			}
			if !tc.wantOK && !errors.Is(err, paymentlifecycle.ErrStorageUnavailable) {
				t.Fatalf("Open() error = %v, want storage unavailable", err)
			}
		})
	}
}

func TestStoreOpenRejectsQuoteBeyondInvoiceExpiry(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
	record := validRecord()
	record.Quote.ExpiresAt = record.Command.ExpiresAt.Add(time.Second)
	if _, err := New(db).Open(context.Background(), record); !errors.Is(err, paymentlifecycle.ErrStorageUnavailable) {
		t.Fatalf("Open() error = %v, want storage unavailable", err)
	}
	assertEffectCounts(t, db, 0)
}

func TestStoreOpenEnforcesQuoteTimelineAgainstInvoiceOpen(t *testing.T) {
	tests := []struct {
		name  string
		alter func(*paymentlifecycle.OpenRecord)
	}{
		{
			name: "quoted after invoice opened",
			alter: func(record *paymentlifecycle.OpenRecord) {
				record.Quote.QuotedAt = record.OpenedAt.Add(time.Second)
			},
		},
		{
			name: "quote expires when invoice opens",
			alter: func(record *paymentlifecycle.OpenRecord) {
				record.Quote.QuotedAt = record.OpenedAt.Add(-time.Minute)
				record.Quote.ExpiresAt = record.OpenedAt
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, cleanup := openMigratedDB(t)
			defer cleanup()
			seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
			record := validRecord()
			tc.alter(&record)
			if _, err := New(db).Open(context.Background(), record); !errors.Is(err, paymentlifecycle.ErrStorageUnavailable) {
				t.Fatalf("Open() error = %v, want storage unavailable", err)
			}
			assertEffectCounts(t, db, 0)
		})
	}
}

func TestStoreOpenRollsBackWhenLateOutboxInsertFails(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
	if err := db.Exec(`
CREATE FUNCTION test_reject_outbox_insert() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'injected outbox failure'; END; $$;
CREATE TRIGGER test_reject_outbox_insert BEFORE INSERT ON payment_lifecycle_outbox_events
FOR EACH ROW EXECUTE FUNCTION test_reject_outbox_insert();`).Error; err != nil {
		t.Fatalf("install failure trigger: %v", err)
	}
	if _, err := New(db).Open(context.Background(), validRecord()); !errors.Is(err, paymentlifecycle.ErrStorageUnavailable) {
		t.Fatalf("Open() error = %v, want storage unavailable", err)
	}
	assertEffectCounts(t, db, 0)
}

func TestStoreOpenPersistsImmutableQuoteAndDomainFacts(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
	store := New(db)
	created, err := store.Open(context.Background(), validRecord())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	mutations := []struct {
		table  string
		column string
		value  any
	}{
		{"payment_lifecycle_quotes", "quote_id", "quote-0001"},
		{"payment_lifecycle_payment_methods", "invoice_id", string(created.View.InvoiceID)},
		{"payment_lifecycle_idempotency_receipts", "invoice_id", string(created.View.InvoiceID)},
		{"payment_lifecycle_address_assignments", "invoice_id", string(created.View.InvoiceID)},
		{"payment_lifecycle_history", "invoice_id", string(created.View.InvoiceID)},
		{"payment_lifecycle_outbox_events", "aggregate_id", string(created.View.InvoiceID)},
	}
	for _, mutation := range mutations {
		if err := db.Exec("DELETE FROM "+mutation.table+" WHERE "+mutation.column+" = ?", mutation.value).Error; err == nil {
			t.Errorf("DELETE from immutable %s succeeded", mutation.table)
		}
		if err := db.Exec("UPDATE "+mutation.table+" SET "+mutation.column+" = "+mutation.column+" WHERE "+mutation.column+" = ?", mutation.value).Error; err == nil {
			t.Errorf("UPDATE of immutable %s succeeded", mutation.table)
		}
	}
}

func TestStoreOpenPersistsStableV1ReceiptAndEventPayloads(t *testing.T) {
	db, cleanup := openMigratedDB(t)
	defer cleanup()
	seedAddress(t, db, "tenant-acme", "eip155:1", usdtAsset, "0x1111111111111111111111111111111111111111")
	if _, err := New(db).Open(context.Background(), validRecord()); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	var receiptSchema, receiptAtomic, eventSchema, eventAtomic string
	if err := db.Raw(`SELECT response->>'schema_version', response#>>'{payment,quote,required_atomic_units}'
        FROM payment_lifecycle_idempotency_receipts`).Row().Scan(&receiptSchema, &receiptAtomic); err != nil {
		t.Fatalf("read receipt payload: %v", err)
	}
	if err := db.Raw(`SELECT payload->>'schema_version', payload#>>'{payment,quote,required_atomic_units}'
        FROM payment_lifecycle_outbox_events`).Row().Scan(&eventSchema, &eventAtomic); err != nil {
		t.Fatalf("read event payload: %v", err)
	}
	if receiptSchema != "1" || eventSchema != "1" || receiptAtomic != "10000000" || eventAtomic != "10000000" {
		t.Fatalf("receipt/event payloads = schema %q/%q atomic %q/%q", receiptSchema, eventSchema, receiptAtomic, eventAtomic)
	}
}

const usdtAsset = "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7"

func validRecord() paymentlifecycle.OpenRecord {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	return paymentlifecycle.OpenRecord{
		Operation:   paymentlifecycle.OperationOpen,
		RequestHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Command: paymentlifecycle.OpenPayment{
			TenantID: "tenant-acme", IdempotencyKey: "checkout-20260827-0001", MerchantReference: "order-1042",
			InvoiceAmount: paymentlifecycle.FiatAmount{Currency: "USD", MinorUnits: "1250"},
			PaymentMethod: paymentlifecycle.PaymentMethod{ChainID: "eip155:1", AssetID: usdtAsset},
			ExpiresAt:     now.Add(30 * time.Minute),
		},
		Quote: paymentlifecycle.Quote{
			ID: "quote-0001", InvoiceCurrency: "USD", InvoiceMinorUnits: "1250",
			ChainID: "eip155:1", AssetID: usdtAsset, RequiredAtomicUnits: "10000000", AssetDecimals: 6,
			RateNumerator: "125", RateDenominator: "100", Source: "merchant-price-oracle",
			QuotedAt: now, ExpiresAt: now.Add(5 * time.Minute), Rounding: "ceil",
		},
		State:          paymentlifecycle.InvoiceOpen,
		Revision:       1,
		LifecycleEvent: paymentlifecycle.LifecycleEventSpec{Type: paymentlifecycle.EventInvoiceOpen, Revision: 1, OccurredAt: now},
		OutboxEvent:    paymentlifecycle.OutboxEventSpec{Type: paymentlifecycle.EventInvoiceOpen, SchemaVersion: 1, OccurredAt: now},
		OpenedAt:       now,
	}
}

func openMigratedDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	db, cleanup := database.NewTestDB(t)
	if _, err := database.ApplyMigrations(context.Background(), db); err != nil {
		cleanup()
		t.Fatalf("apply migrations: %v", err)
	}
	return db, cleanup
}

func seedAddress(t *testing.T, db *gorm.DB, tenant, chain, asset, address string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO payment_lifecycle_deposit_addresses
        (tenant_id, chain_id, asset_id, address) VALUES (?, ?, ?, ?)`, tenant, chain, asset, address).Error; err != nil {
		t.Fatalf("seed address: %v", err)
	}
}

func assertEffectCounts(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()
	for _, table := range []string{
		"payment_lifecycle_invoices", "payment_lifecycle_quotes", "payment_lifecycle_payment_methods",
		"payment_lifecycle_idempotency_receipts", "payment_lifecycle_address_assignments",
		"payment_lifecycle_history", "payment_lifecycle_outbox_events",
	} {
		var got int64
		if err := db.Table(table).Count(&got).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Errorf("%s count = %d, want %d", table, got, want)
		}
	}
}
