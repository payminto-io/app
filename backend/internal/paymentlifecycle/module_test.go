package paymentlifecycle

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestOpenCreatesServerQuotedPayment(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	cmd := validOpenPayment(now)
	quote := validQuote(now)
	quotes := NewMemoryQuoteAdapter([]Quote{quote})
	store := NewMemoryStore([]AvailableDepositAddress{{
		TenantID: TenantID("tenant-acme"),
		ChainID:  ChainID("eip155:1"),
		AssetID:  AssetID("eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7"),
		Address:  "0x1111111111111111111111111111111111111111",
	}})
	lifecycle := New(quotes, store, func() time.Time { return now })

	got, err := lifecycle.Open(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if got.TenantID != TenantID("tenant-acme") {
		t.Fatalf("TenantID = %q", got.TenantID)
	}
	if got.InvoiceAmount.MinorUnits != "1250" {
		t.Fatalf("InvoiceAmount.MinorUnits = %q", got.InvoiceAmount.MinorUnits)
	}
	if got.Quote.RequiredAtomicUnits != "10000000" {
		t.Fatalf("Quote.RequiredAtomicUnits = %q", got.Quote.RequiredAtomicUnits)
	}
	if got.DepositAddress.Address != "0x1111111111111111111111111111111111111111" {
		t.Fatalf("DepositAddress.Address = %q", got.DepositAddress.Address)
	}
	if got.State != InvoiceOpen {
		t.Fatalf("State = %q", got.State)
	}
	if got.Revision != 1 {
		t.Fatalf("Revision = %d", got.Revision)
	}
}

func TestOpenReplaysIdenticalTenantScopedCommandWithoutNewEffects(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	store := NewMemoryStore([]AvailableDepositAddress{{
		TenantID: "tenant-acme",
		ChainID:  "eip155:1", AssetID: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7",
		Address: "0x1111111111111111111111111111111111111111",
	}})
	lifecycle := New(NewMemoryQuoteAdapter([]Quote{validQuote(now)}), store, func() time.Time { return now })

	first, err := lifecycle.Open(context.Background(), validOpenPayment(now))
	if err != nil {
		t.Fatalf("first Open() error = %v", err)
	}
	second, err := lifecycle.Open(context.Background(), validOpenPayment(now))
	if err != nil {
		t.Fatalf("second Open() error = %v", err)
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("replay differs:\nfirst:  %#v\nsecond: %#v", first, second)
	}
	assertSingleOpenEffects(t, store.Snapshot(), now)
}

func TestOpenReplayPreservesOriginalImmutableQuoteWhenMarketMoves(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	clockNow := now
	firstQuote := validQuote(now)
	secondQuote := validQuote(now)
	secondQuote.ID = "quote-0002"
	secondQuote.RequiredAtomicUnits = "11000000"
	secondQuote.RateNumerator = "1375"
	secondQuote.RateDenominator = "1000"
	secondQuote.QuotedAt = now.Add(time.Minute)
	secondQuote.ExpiresAt = now.Add(6 * time.Minute)
	quotes := &sequenceQuoteAdapter{quotes: []Quote{firstQuote, secondQuote}}
	store := NewMemoryStore([]AvailableDepositAddress{{
		TenantID: "tenant-acme", ChainID: "eip155:1",
		AssetID: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7",
		Address: "0x1111111111111111111111111111111111111111",
	}})
	lifecycle := New(quotes, store, func() time.Time { return clockNow })

	first, err := lifecycle.Open(context.Background(), validOpenPayment(now))
	if err != nil {
		t.Fatalf("first Open() error = %v", err)
	}
	clockNow = now.Add(time.Minute)
	replay, err := lifecycle.Open(context.Background(), validOpenPayment(now))
	if err != nil {
		t.Fatalf("replay Open() error = %v", err)
	}
	if !reflect.DeepEqual(replay, first) || replay.Quote.ID != "quote-0001" {
		t.Fatalf("replay did not preserve original Quote: %#v", replay.Quote)
	}
}

func TestOpenReplaysReceiptAfterInvoiceAndQuoteExpiryDuringOracleOutage(t *testing.T) {
	openedAt := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	clockNow := openedAt
	quotes := &countingQuoteAdapter{quote: validQuote(openedAt)}
	store := NewMemoryStore([]AvailableDepositAddress{{
		TenantID: "tenant-acme", ChainID: "eip155:1",
		AssetID: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7",
		Address: "0x1111111111111111111111111111111111111111",
	}})
	lifecycle := New(quotes, store, func() time.Time { return clockNow })
	command := validOpenPayment(openedAt)

	created, err := lifecycle.Open(context.Background(), command)
	if err != nil {
		t.Fatalf("create Open() error = %v", err)
	}
	clockNow = openedAt.Add(2 * time.Hour)
	quotes.err = errors.New("oracle offline")
	quotes.quote = Quote{}

	replayed, err := lifecycle.Open(context.Background(), command)
	if err != nil {
		t.Fatalf("expired replay Open() error = %v", err)
	}
	if !reflect.DeepEqual(replayed, created) {
		t.Fatalf("expired replay differs: %#v != %#v", replayed, created)
	}
	if quotes.calls != 1 {
		t.Fatalf("Quote calls = %d, want only the create call", quotes.calls)
	}
}

func TestOpenReplayDoesNotDependOnCurrentClockPosition(t *testing.T) {
	openedAt := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	clockNow := openedAt
	store := NewMemoryStore([]AvailableDepositAddress{{
		TenantID: "tenant-acme", ChainID: "eip155:1",
		AssetID: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7",
		Address: "0x1111111111111111111111111111111111111111",
	}})
	lifecycle := New(NewMemoryQuoteAdapter([]Quote{validQuote(openedAt)}), store, func() time.Time { return clockNow })
	created, err := lifecycle.Open(context.Background(), validOpenPayment(openedAt))
	if err != nil {
		t.Fatalf("create Open() error = %v", err)
	}
	clockNow = openedAt.Add(-time.Hour)

	replayed, err := lifecycle.Open(context.Background(), validOpenPayment(openedAt))
	if err != nil {
		t.Fatalf("clock-skew replay Open() error = %v", err)
	}
	if !reflect.DeepEqual(replayed, created) {
		t.Fatalf("clock-skew replay differs: %#v != %#v", replayed, created)
	}
}

func TestOpenChangedCommandConflictsFromReceiptWithoutQuoteCall(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	quotes := &countingQuoteAdapter{quote: validQuote(now)}
	store := NewMemoryStore([]AvailableDepositAddress{{
		TenantID: "tenant-acme", ChainID: "eip155:1",
		AssetID: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7",
		Address: "0x1111111111111111111111111111111111111111",
	}})
	lifecycle := New(quotes, store, func() time.Time { return now })
	if _, err := lifecycle.Open(context.Background(), validOpenPayment(now)); err != nil {
		t.Fatalf("create Open() error = %v", err)
	}
	changed := validOpenPayment(now)
	changed.MerchantReference = "order-1043"

	_, err := lifecycle.Open(context.Background(), changed)
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want idempotency conflict", err)
	}
	if quotes.calls != 1 {
		t.Fatalf("Quote calls = %d, want only the create call", quotes.calls)
	}
}

func TestOpenConflictsWhenSameTenantKeyRepresentsDifferentCommand(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	tests := []struct {
		name  string
		alter func(*OpenPayment)
	}{
		{"merchant reference", func(c *OpenPayment) { c.MerchantReference = "order-1043" }},
		{"invoice amount", func(c *OpenPayment) { c.InvoiceAmount.MinorUnits = "1251" }},
		{"payment method", func(c *OpenPayment) {
			c.PaymentMethod = PaymentMethod{ChainID: "eip155:10", AssetID: "eip155:10/slip44:60"}
		}},
		{"expiry", func(c *OpenPayment) { c.ExpiresAt = c.ExpiresAt.Add(time.Minute) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore([]AvailableDepositAddress{{
				TenantID: "tenant-acme",
				ChainID:  "eip155:1", AssetID: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7",
				Address: "0x1111111111111111111111111111111111111111",
			}})
			lifecycle := New(requestQuoteAdapter{now: now}, store, func() time.Time { return now })
			if _, err := lifecycle.Open(context.Background(), validOpenPayment(now)); err != nil {
				t.Fatalf("first Open() error = %v", err)
			}
			changed := validOpenPayment(now)
			tt.alter(&changed)

			_, err := lifecycle.Open(context.Background(), changed)
			if !errors.Is(err, ErrIdempotencyConflict) {
				t.Fatalf("error = %v, want idempotency conflict", err)
			}
			if store.Snapshot().Invoices != 1 {
				t.Fatalf("invoices = %d, want 1", store.Snapshot().Invoices)
			}
		})
	}
}

func TestOpenScopesKeyAndMerchantReferenceByTenant(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	store := NewMemoryStore([]AvailableDepositAddress{
		{TenantID: "tenant-acme", ChainID: "eip155:1", AssetID: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7", Address: "0x1111111111111111111111111111111111111111"},
		{TenantID: "tenant-bravo", ChainID: "eip155:1", AssetID: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7", Address: "0x2222222222222222222222222222222222222222"},
	})
	lifecycle := New(requestQuoteAdapter{now: now}, store, func() time.Time { return now })

	first, err := lifecycle.Open(context.Background(), validOpenPayment(now))
	if err != nil {
		t.Fatalf("Tenant A Open() error = %v", err)
	}
	secondCommand := validOpenPayment(now)
	secondCommand.TenantID = "tenant-bravo"
	second, err := lifecycle.Open(context.Background(), secondCommand)
	if err != nil {
		t.Fatalf("Tenant B Open() error = %v", err)
	}
	if first.InvoiceID == second.InvoiceID || first.DepositAddress.Address == second.DepositAddress.Address {
		t.Fatalf("tenant records were not independently allocated: %#v %#v", first, second)
	}
	if got := store.Snapshot().Invoices; got != 2 {
		t.Fatalf("invoices = %d, want 2", got)
	}
}

func TestOpenConcurrentMemoryReplaysAllocateOneAddressAndOutboxEvent(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	store := NewMemoryStore([]AvailableDepositAddress{{
		TenantID: "tenant-acme", ChainID: "eip155:1",
		AssetID: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7",
		Address: "0x1111111111111111111111111111111111111111",
	}})
	lifecycle := New(requestQuoteAdapter{now: now}, store, func() time.Time { return now })
	type outcome struct {
		view PaymentView
		err  error
	}
	outcomes := make(chan outcome, 24)
	for range 24 {
		go func() {
			view, err := lifecycle.Open(context.Background(), validOpenPayment(now))
			outcomes <- outcome{view: view, err: err}
		}()
	}
	var first PaymentView
	for i := 0; i < 24; i++ {
		got := <-outcomes
		if got.err != nil {
			t.Fatalf("concurrent Open() error = %v", got.err)
		}
		if i == 0 {
			first = got.view
		} else if !reflect.DeepEqual(got.view, first) {
			t.Fatalf("concurrent replay differs: %#v != %#v", got.view, first)
		}
	}
	assertSingleOpenEffects(t, store.Snapshot(), now)
}

func TestOpenRejectsDuplicateMerchantReferenceWithDifferentKey(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	store := NewMemoryStore([]AvailableDepositAddress{
		{TenantID: "tenant-acme", ChainID: "eip155:1", AssetID: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7", Address: "0x1111111111111111111111111111111111111111"},
		{TenantID: "tenant-acme", ChainID: "eip155:1", AssetID: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7", Address: "0x2222222222222222222222222222222222222222"},
	})
	lifecycle := New(requestQuoteAdapter{now: now}, store, func() time.Time { return now })
	if _, err := lifecycle.Open(context.Background(), validOpenPayment(now)); err != nil {
		t.Fatalf("first Open() error = %v", err)
	}
	duplicate := validOpenPayment(now)
	duplicate.IdempotencyKey = "checkout-20260827-0002"

	_, err := lifecycle.Open(context.Background(), duplicate)
	if !errors.Is(err, ErrMerchantReferenceConflict) {
		t.Fatalf("error = %v, want merchant reference conflict", err)
	}
	if got := store.Snapshot(); got.Invoices != 1 || got.AddressAssignments != 1 || got.OutboxEvents != 1 {
		t.Fatalf("effects after conflict = %#v", got)
	}
}

func TestOpenMapsAdapterFailuresAndReturnsNoPartialView(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	adapterFailure := errors.New("adapter offline")
	tests := []struct {
		name   string
		quotes QuoteAdapter
		store  Store
		want   error
	}{
		{"unsupported method remains typed", &countingQuoteAdapter{err: ErrUnsupportedPaymentMethod}, &countingStore{}, ErrUnsupportedPaymentMethod},
		{"quote failure is unavailable", &countingQuoteAdapter{err: adapterFailure}, &countingStore{}, ErrQuoteUnavailable},
		{"store failure is unavailable", &countingQuoteAdapter{quote: validQuote(now)}, &countingStore{result: StoreResult{View: PaymentView{InvoiceID: "must-not-leak"}}, err: adapterFailure}, ErrStorageUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lifecycle := New(tt.quotes, tt.store, func() time.Time { return now })
			view, err := lifecycle.Open(context.Background(), validOpenPayment(now))
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			if view != (PaymentView{}) {
				t.Fatalf("partial view leaked = %#v", view)
			}
		})
	}
}

func TestOpenSuppliesStableCompleteAtomicWriteIntent(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	store := &recordingStore{}
	lifecycle := New(NewMemoryQuoteAdapter([]Quote{validQuote(now)}), store, func() time.Time { return now })

	if _, err := lifecycle.Open(context.Background(), validOpenPayment(now)); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	record := store.record
	if record.RequestHash != "252cde09df7afea91db4616b90dfa80a4e60698d8e484911d8b9d87ecd7d7bfe" {
		t.Fatalf("RequestHash = %q", record.RequestHash)
	}
	if record.Operation != OperationOpen || record.State != InvoiceOpen || record.Revision != 1 {
		t.Fatalf("aggregate intent = %#v", record)
	}
	if record.LifecycleEvent != (LifecycleEventSpec{Type: EventInvoiceOpen, Revision: 1, OccurredAt: now}) {
		t.Fatalf("lifecycle event = %#v", record.LifecycleEvent)
	}
	if record.OutboxEvent != (OutboxEventSpec{Type: EventInvoiceOpen, SchemaVersion: 1, OccurredAt: now}) {
		t.Fatalf("outbox event = %#v", record.OutboxEvent)
	}
}

func TestOpenRejectsInvalidStoreResultWithoutExposingIt(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	valid := viewForRecord(OpenRecord{Command: validOpenPayment(now), Quote: validQuote(now), State: InvoiceOpen, Revision: 1, OpenedAt: now})
	tests := []struct {
		name  string
		alter func(*StoreResult)
	}{
		{"disposition", func(r *StoreResult) { r.Disposition = "unknown" }},
		{"invoice identity", func(r *StoreResult) { r.View.InvoiceID = "" }},
		{"tenant mismatch", func(r *StoreResult) { r.View.TenantID = "tenant-other" }},
		{"quote mismatch", func(r *StoreResult) { r.View.Quote.RequiredAtomicUnits = "1" }},
		{"address missing", func(r *StoreResult) { r.View.DepositAddress.Address = "" }},
		{"state mismatch", func(r *StoreResult) { r.View.State = "filled" }},
		{"revision mismatch", func(r *StoreResult) { r.View.Revision = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StoreResult{Disposition: StoreCreated, View: valid}
			tt.alter(&result)
			lifecycle := New(NewMemoryQuoteAdapter([]Quote{validQuote(now)}), &countingStore{result: result}, func() time.Time { return now })

			view, err := lifecycle.Open(context.Background(), validOpenPayment(now))
			if !errors.Is(err, ErrStorageUnavailable) {
				t.Fatalf("error = %v, want storage unavailable", err)
			}
			if view != (PaymentView{}) {
				t.Fatalf("invalid store view leaked = %#v", view)
			}
		})
	}
}

func TestOpenRejectsCorruptReceiptProjectionWithoutCallingQuote(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	record := OpenRecord{Command: validOpenPayment(now), Quote: validQuote(now), State: InvoiceOpen, Revision: 1, OpenedAt: now}
	tests := []struct {
		name  string
		alter func(*PaymentView)
	}{
		{"asset identity", func(v *PaymentView) { v.Quote.AssetID = "eip155:1/slip44:60" }},
		{"quote timeline", func(v *PaymentView) { v.Quote.QuotedAt = now.Add(time.Second) }},
		{"asset decimals", func(v *PaymentView) { v.Quote.AssetDecimals = 78 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			corrupt := viewForRecord(record)
			tt.alter(&corrupt)
			store := &countingStore{receipt: ReceiptResult{Disposition: ReceiptReplayed, View: corrupt}}
			quotes := &countingQuoteAdapter{err: errors.New("must not be called")}
			lifecycle := New(quotes, store, func() time.Time { return now })

			view, err := lifecycle.Open(context.Background(), validOpenPayment(now))
			if !errors.Is(err, ErrStorageUnavailable) {
				t.Fatalf("error = %v, want storage unavailable", err)
			}
			if view != (PaymentView{}) || quotes.calls != 0 {
				t.Fatalf("corrupt replay leaked or quoted: view=%#v quote_calls=%d", view, quotes.calls)
			}
		})
	}
}

func TestMemoryStoreRejectsUnsupportedReceiptLookupOperationAndHash(t *testing.T) {
	store := NewMemoryStore(nil)
	lookups := []ReceiptLookup{
		{TenantID: "tenant-acme", Operation: "payment_lifecycle.apply", IdempotencyKey: "checkout-20260827-0001", RequestHash: "252cde09df7afea91db4616b90dfa80a4e60698d8e484911d8b9d87ecd7d7bfe"},
		{TenantID: "tenant-acme", Operation: OperationOpen, IdempotencyKey: "checkout-20260827-0001", RequestHash: "not-a-sha256"},
	}
	for _, lookup := range lookups {
		_, err := store.LookupReceipt(context.Background(), lookup)
		if !errors.Is(err, ErrStorageUnavailable) {
			t.Fatalf("LookupReceipt(%#v) error = %v", lookup, err)
		}
	}
}

func TestOpenRejectsInvalidCommandsBeforeQuoting(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	tests := []struct {
		name  string
		alter func(*OpenPayment)
		field string
	}{
		{"tenant is required", func(c *OpenPayment) { c.TenantID = "" }, "tenant_id"},
		{"tenant is canonical", func(c *OpenPayment) { c.TenantID = " tenant-acme" }, "tenant_id"},
		{"key minimum length", func(c *OpenPayment) { c.IdempotencyKey = "too-short" }, "idempotency_key"},
		{"key visible ASCII", func(c *OpenPayment) { c.IdempotencyKey = "checkout-20260827\n" }, "idempotency_key"},
		{"reference required", func(c *OpenPayment) { c.MerchantReference = "" }, "merchant_reference"},
		{"currency uppercase", func(c *OpenPayment) { c.InvoiceAmount.Currency = "usd" }, "invoice_amount.currency"},
		{"minor units positive", func(c *OpenPayment) { c.InvoiceAmount.MinorUnits = "0" }, "invoice_amount.minor_units"},
		{"minor units canonical", func(c *OpenPayment) { c.InvoiceAmount.MinorUnits = "01250" }, "invoice_amount.minor_units"},
		{"chain required", func(c *OpenPayment) { c.PaymentMethod.ChainID = "" }, "payment_method.chain_id"},
		{"chain fits storage", func(c *OpenPayment) {
			chain := ChainID("c" + strings.Repeat("x", 128))
			c.PaymentMethod.ChainID = chain
			c.PaymentMethod.AssetID = AssetID(string(chain) + "/asset")
		}, "payment_method.chain_id"},
		{"asset required", func(c *OpenPayment) { c.PaymentMethod.AssetID = "" }, "payment_method.asset_id"},
		{"asset is chain scoped", func(c *OpenPayment) { c.PaymentMethod.AssetID = "eip155:10/slip44:60" }, "payment_method.asset_id"},
		{"asset suffix required", func(c *OpenPayment) { c.PaymentMethod.AssetID = "eip155:1/" }, "payment_method.asset_id"},
		{"expiry future", func(c *OpenPayment) { c.ExpiresAt = now }, "expires_at"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := validOpenPayment(now)
			tt.alter(&cmd)
			quotes := &countingQuoteAdapter{quote: validQuote(now)}
			lifecycle := New(quotes, NewMemoryStore(nil), func() time.Time { return now })

			_, err := lifecycle.Open(context.Background(), cmd)
			if !errors.Is(err, ErrInvalidCommand) {
				t.Fatalf("error = %v, want invalid command", err)
			}
			var domainErr *Error
			if !errors.As(err, &domainErr) || domainErr.Field != tt.field {
				t.Fatalf("error field = %q, want %q", domainErr.Field, tt.field)
			}
			if quotes.calls != 0 {
				t.Fatalf("quote calls = %d, want 0", quotes.calls)
			}
		})
	}
}

func TestOpenRejectsExpiredOrMismatchedQuoteBeforePersistence(t *testing.T) {
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	tests := []struct {
		name  string
		alter func(*Quote)
		want  error
	}{
		{"expired", func(q *Quote) { q.ExpiresAt = now }, ErrQuoteExpired},
		{"quote identity", func(q *Quote) { q.ID = "" }, ErrQuoteMismatched},
		{"quote identity fits storage", func(q *Quote) { q.ID = QuoteID("q" + strings.Repeat("x", 128)) }, ErrQuoteMismatched},
		{"invoice currency", func(q *Quote) { q.InvoiceCurrency = "EUR" }, ErrQuoteMismatched},
		{"invoice amount", func(q *Quote) { q.InvoiceMinorUnits = "1251" }, ErrQuoteMismatched},
		{"chain identity", func(q *Quote) { q.ChainID = "eip155:10" }, ErrQuoteMismatched},
		{"asset identity", func(q *Quote) { q.AssetID = "eip155:1/slip44:60" }, ErrQuoteMismatched},
		{"atomic amount positive", func(q *Quote) { q.RequiredAtomicUnits = "0" }, ErrQuoteMismatched},
		{"atomic amount canonical", func(q *Quote) { q.RequiredAtomicUnits = "010000000" }, ErrQuoteMismatched},
		{"asset decimals bounded", func(q *Quote) { q.AssetDecimals = 78 }, ErrQuoteMismatched},
		{"rate numerator positive", func(q *Quote) { q.RateNumerator = "0" }, ErrQuoteMismatched},
		{"rate denominator positive", func(q *Quote) { q.RateDenominator = "0" }, ErrQuoteMismatched},
		{"source required", func(q *Quote) { q.Source = "" }, ErrQuoteMismatched},
		{"quoted time required", func(q *Quote) { q.QuotedAt = time.Time{} }, ErrQuoteMismatched},
		{"quoted time not future", func(q *Quote) { q.QuotedAt = now.Add(time.Second) }, ErrQuoteMismatched},
		{"rounding required", func(q *Quote) { q.Rounding = "" }, ErrQuoteMismatched},
		{"rounding known", func(q *Quote) { q.Rounding = "nearest-ish" }, ErrQuoteMismatched},
		{"quote within invoice expiry", func(q *Quote) { q.ExpiresAt = now.Add(time.Hour) }, ErrQuoteMismatched},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quote := validQuote(now)
			tt.alter(&quote)
			store := &countingStore{}
			lifecycle := New(&countingQuoteAdapter{quote: quote}, store, func() time.Time { return now })

			_, err := lifecycle.Open(context.Background(), validOpenPayment(now))
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
			if store.calls != 0 {
				t.Fatalf("store calls = %d, want 0", store.calls)
			}
		})
	}
}

type countingQuoteAdapter struct {
	quote Quote
	err   error
	calls int
}

type sequenceQuoteAdapter struct {
	quotes []Quote
	next   int
}

func (a *sequenceQuoteAdapter) Quote(context.Context, QuoteRequest) (Quote, error) {
	quote := a.quotes[a.next]
	a.next++
	return quote, nil
}

func (a *countingQuoteAdapter) Quote(context.Context, QuoteRequest) (Quote, error) {
	a.calls++
	return a.quote, a.err
}

type countingStore struct {
	receipt   ReceiptResult
	lookupErr error
	lookups   int
	result    StoreResult
	err       error
	calls     int
}

func (s *countingStore) LookupReceipt(context.Context, ReceiptLookup) (ReceiptResult, error) {
	s.lookups++
	if s.receipt.Disposition == "" && s.lookupErr == nil {
		return ReceiptResult{Disposition: ReceiptMiss}, nil
	}
	return s.receipt, s.lookupErr
}

func (s *countingStore) Open(context.Context, OpenRecord) (StoreResult, error) {
	s.calls++
	return s.result, s.err
}

type requestQuoteAdapter struct{ now time.Time }

func (a requestQuoteAdapter) Quote(_ context.Context, req QuoteRequest) (Quote, error) {
	quote := validQuote(a.now)
	quote.InvoiceCurrency = req.InvoiceAmount.Currency
	quote.InvoiceMinorUnits = req.InvoiceAmount.MinorUnits
	quote.ChainID = req.PaymentMethod.ChainID
	quote.AssetID = req.PaymentMethod.AssetID
	quote.ExpiresAt = req.InvoiceExpiry.Add(-time.Minute)
	return quote, nil
}

type recordingStore struct{ record OpenRecord }

func (s *recordingStore) LookupReceipt(context.Context, ReceiptLookup) (ReceiptResult, error) {
	return ReceiptResult{Disposition: ReceiptMiss}, nil
}

func (s *recordingStore) Open(_ context.Context, record OpenRecord) (StoreResult, error) {
	s.record = record
	return StoreResult{Disposition: StoreCreated, View: viewForRecord(record)}, nil
}

func viewForRecord(record OpenRecord) PaymentView {
	return PaymentView{
		InvoiceID: "invoice-000001", TenantID: record.Command.TenantID,
		MerchantReference: record.Command.MerchantReference,
		InvoiceAmount:     record.Command.InvoiceAmount, PaymentMethod: record.Command.PaymentMethod,
		Quote:          record.Quote,
		DepositAddress: DepositAddress{AssignmentID: "assignment-000001", Address: "0x1111111111111111111111111111111111111111"},
		State:          record.State, Revision: record.Revision, OpenedAt: record.OpenedAt, ExpiresAt: record.Command.ExpiresAt,
	}
}

func assertSingleOpenEffects(t *testing.T, got MemoryStoreSnapshot, occurredAt time.Time) {
	t.Helper()
	if got.Invoices != 1 || got.AddressAssignments != 1 || got.LifecycleEvents != 1 || got.OutboxEvents != 1 || got.Receipts != 1 {
		t.Fatalf("effect counts = %#v", got)
	}
	wantLifecycle := []LifecycleEventSpec{{Type: EventInvoiceOpen, Revision: 1, OccurredAt: occurredAt}}
	wantOutbox := []OutboxEventSpec{{Type: EventInvoiceOpen, SchemaVersion: 1, OccurredAt: occurredAt}}
	if !reflect.DeepEqual(got.LifecycleEventSpecs, wantLifecycle) || !reflect.DeepEqual(got.OutboxEventSpecs, wantOutbox) {
		t.Fatalf("persisted event specs = %#v / %#v", got.LifecycleEventSpecs, got.OutboxEventSpecs)
	}
}

func validOpenPayment(now time.Time) OpenPayment {
	return OpenPayment{
		TenantID:          TenantID("tenant-acme"),
		IdempotencyKey:    IdempotencyKey("checkout-20260827-0001"),
		MerchantReference: MerchantReference("order-1042"),
		InvoiceAmount:     FiatAmount{Currency: "USD", MinorUnits: "1250"},
		PaymentMethod: PaymentMethod{
			ChainID: ChainID("eip155:1"),
			AssetID: AssetID("eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7"),
		},
		ExpiresAt: now.Add(30 * time.Minute),
	}
}

func validQuote(now time.Time) Quote {
	return Quote{
		ID:                  QuoteID("quote-0001"),
		InvoiceCurrency:     "USD",
		InvoiceMinorUnits:   "1250",
		ChainID:             ChainID("eip155:1"),
		AssetID:             AssetID("eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7"),
		RequiredAtomicUnits: "10000000",
		AssetDecimals:       6,
		RateNumerator:       "125",
		RateDenominator:     "100",
		Source:              "merchant-price-oracle",
		QuotedAt:            now,
		ExpiresAt:           now.Add(5 * time.Minute),
		Rounding:            "ceil",
	}
}
