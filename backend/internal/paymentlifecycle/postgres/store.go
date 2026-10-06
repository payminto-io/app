package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/payminto/payminto/backend/internal/paymentlifecycle"
	"gorm.io/gorm"
)

// Store is the PostgreSQL adapter at paymentlifecycle.Store. Receipt lookup is
// the read-only replay fast path; Open repeats that check under its transaction
// lock and hides address allocation plus the complete atomic persistence graph.
type Store struct {
	db *sql.DB
}

var _ paymentlifecycle.Store = (*Store)(nil)

func New(db *gorm.DB) *Store {
	if db == nil {
		return &Store{}
	}
	sqlDB, _ := db.DB()
	return &Store{db: sqlDB}
}

// LookupReceipt is the receipt-first replay path. It deliberately does not
// touch Quote or address tables, so a valid replay remains available after the
// original Invoice/Quote expires or while a Quote Adapter is unavailable.
func (s *Store) LookupReceipt(ctx context.Context, lookup paymentlifecycle.ReceiptLookup) (paymentlifecycle.ReceiptResult, error) {
	if s == nil || s.db == nil {
		return paymentlifecycle.ReceiptResult{}, storageFailure(errors.New("database is nil"))
	}
	if err := validateReceiptIdentity(lookup.Operation, lookup.RequestHash); err != nil {
		return paymentlifecycle.ReceiptResult{}, storageFailure(err)
	}
	var storedHash string
	var storedResponse []byte
	err := s.db.QueryRowContext(ctx, `
SELECT request_hash, response
FROM payment_lifecycle_idempotency_receipts
WHERE tenant_id = $1 AND operation = $2 AND idempotency_key = $3`,
		string(lookup.TenantID), lookup.Operation, string(lookup.IdempotencyKey),
	).Scan(&storedHash, &storedResponse)
	if errors.Is(err, sql.ErrNoRows) {
		return paymentlifecycle.ReceiptResult{Disposition: paymentlifecycle.ReceiptMiss}, nil
	}
	if err != nil {
		return paymentlifecycle.ReceiptResult{}, storageFailure(fmt.Errorf("read idempotency receipt: %w", err))
	}
	if storedHash != lookup.RequestHash {
		return paymentlifecycle.ReceiptResult{}, paymentlifecycle.ErrIdempotencyConflict
	}
	view, err := decodePaymentViewV1(storedResponse)
	if err != nil {
		return paymentlifecycle.ReceiptResult{}, storageFailure(fmt.Errorf("decode replay receipt: %w", err))
	}
	return paymentlifecycle.ReceiptResult{Disposition: paymentlifecycle.ReceiptReplayed, View: view}, nil
}

func (s *Store) Open(ctx context.Context, record paymentlifecycle.OpenRecord) (result paymentlifecycle.StoreResult, err error) {
	if s == nil || s.db == nil {
		return paymentlifecycle.StoreResult{}, storageFailure(errors.New("database is nil"))
	}
	if validationErr := validateReceiptIdentity(record.Operation, record.RequestHash); validationErr != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(validationErr)
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("begin transaction: %w", err))
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	tenant := string(record.Command.TenantID)
	operation := record.Operation
	idempotencyKey := string(record.Command.IdempotencyKey)
	lockKey := fmt.Sprintf("%d:%s%d:%s%d:%s", len(tenant), tenant, len(operation), operation, len(idempotencyKey), idempotencyKey)
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("lock idempotency key: %w", err))
	}

	var storedHash string
	var storedResponse []byte
	err = tx.QueryRowContext(ctx, `
SELECT request_hash, response
FROM payment_lifecycle_idempotency_receipts
WHERE tenant_id = $1 AND operation = $2 AND idempotency_key = $3`,
		string(record.Command.TenantID), record.Operation, string(record.Command.IdempotencyKey),
	).Scan(&storedHash, &storedResponse)
	switch {
	case err == nil:
		if storedHash != record.RequestHash {
			return paymentlifecycle.StoreResult{}, paymentlifecycle.ErrIdempotencyConflict
		}
		view, decodeErr := decodePaymentViewV1(storedResponse)
		if decodeErr != nil {
			return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("decode replay receipt: %w", decodeErr))
		}
		if err = tx.Commit(); err != nil {
			return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("commit replay: %w", err))
		}
		return paymentlifecycle.StoreResult{Disposition: paymentlifecycle.StoreReplayed, View: view}, nil
	case !errors.Is(err, sql.ErrNoRows):
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("read idempotency receipt: %w", err))
	}

	var referenceExists bool
	if err = tx.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1 FROM payment_lifecycle_invoices
    WHERE tenant_id = $1 AND merchant_reference = $2
)`, string(record.Command.TenantID), string(record.Command.MerchantReference)).Scan(&referenceExists); err != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("check merchant reference: %w", err))
	}
	if referenceExists {
		return paymentlifecycle.StoreResult{}, paymentlifecycle.ErrMerchantReferenceConflict
	}

	var addressID int64
	var address string
	err = tx.QueryRowContext(ctx, `
SELECT candidate.address_id, candidate.address
FROM payment_lifecycle_deposit_addresses AS candidate
WHERE candidate.tenant_id = $1
  AND candidate.chain_id = $2
  AND candidate.asset_id = $3
  AND NOT EXISTS (
      SELECT 1 FROM payment_lifecycle_address_assignments AS assignment
      WHERE assignment.address_id = candidate.address_id
  )
ORDER BY candidate.address_id
FOR UPDATE OF candidate SKIP LOCKED
LIMIT 1`, string(record.Command.TenantID), string(record.Command.PaymentMethod.ChainID), string(record.Command.PaymentMethod.AssetID)).Scan(&addressID, &address)
	if errors.Is(err, sql.ErrNoRows) {
		return paymentlifecycle.StoreResult{}, paymentlifecycle.ErrDepositAddressUnavailable
	}
	if err != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("allocate deposit address: %w", err))
	}

	invoiceID := paymentlifecycle.InvoiceID("invoice-" + uuid.NewString())
	assignmentID := paymentlifecycle.AddressAssignmentID("assignment-" + uuid.NewString())
	historyID := paymentlifecycle.LifecycleEventID("history-" + uuid.NewString())
	outboxID := paymentlifecycle.OutboxEventID("event-" + uuid.NewString())
	view := paymentlifecycle.PaymentView{
		InvoiceID:         invoiceID,
		TenantID:          record.Command.TenantID,
		MerchantReference: record.Command.MerchantReference,
		InvoiceAmount:     record.Command.InvoiceAmount,
		PaymentMethod:     record.Command.PaymentMethod,
		Quote:             record.Quote,
		DepositAddress:    paymentlifecycle.DepositAddress{AssignmentID: assignmentID, Address: address},
		State:             record.State,
		Revision:          record.Revision,
		OpenedAt:          record.OpenedAt.UTC(),
		ExpiresAt:         record.Command.ExpiresAt.UTC(),
	}
	payload, err := encodePaymentViewV1(view)
	if err != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("encode payment view: %w", err))
	}

	if _, err = tx.ExecContext(ctx, `
INSERT INTO payment_lifecycle_invoices
    (invoice_id, tenant_id, merchant_reference, invoice_currency, invoice_minor_units, state, revision, expires_at, opened_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		string(invoiceID), string(record.Command.TenantID), string(record.Command.MerchantReference),
		record.Command.InvoiceAmount.Currency, record.Command.InvoiceAmount.MinorUnits,
		string(record.State), record.Revision, record.Command.ExpiresAt.UTC(), record.OpenedAt.UTC(),
	); err != nil {
		if isConstraint(err, "payment_lifecycle_invoices_tenant_id_merchant_reference_key") {
			return paymentlifecycle.StoreResult{}, paymentlifecycle.ErrMerchantReferenceConflict
		}
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("insert invoice: %w", err))
	}
	if _, err = tx.ExecContext(ctx, `
INSERT INTO payment_lifecycle_payment_methods
    (invoice_id, tenant_id, chain_id, asset_id, selected_at)
VALUES ($1, $2, $3, $4, $5)`, string(invoiceID), string(record.Command.TenantID),
		string(record.Command.PaymentMethod.ChainID), string(record.Command.PaymentMethod.AssetID), record.OpenedAt.UTC()); err != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("insert payment method: %w", err))
	}
	if _, err = tx.ExecContext(ctx, `
INSERT INTO payment_lifecycle_quotes
    (quote_id, invoice_id, tenant_id, invoice_currency, invoice_minor_units, chain_id, asset_id,
     required_atomic_units, asset_decimals, rate_numerator, rate_denominator, source, quoted_at, expires_at, rounding)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		string(record.Quote.ID), string(invoiceID), string(record.Command.TenantID), record.Quote.InvoiceCurrency,
		record.Quote.InvoiceMinorUnits, string(record.Quote.ChainID), string(record.Quote.AssetID),
		record.Quote.RequiredAtomicUnits, int16(record.Quote.AssetDecimals), record.Quote.RateNumerator,
		record.Quote.RateDenominator, record.Quote.Source, record.Quote.QuotedAt.UTC(),
		record.Quote.ExpiresAt.UTC(), record.Quote.Rounding,
	); err != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("insert quote: %w", err))
	}
	if _, err = tx.ExecContext(ctx, `
INSERT INTO payment_lifecycle_address_assignments
    (assignment_id, invoice_id, tenant_id, address_id, chain_id, asset_id, address, assigned_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, string(assignmentID), string(invoiceID),
		string(record.Command.TenantID), addressID, string(record.Command.PaymentMethod.ChainID),
		string(record.Command.PaymentMethod.AssetID), address, record.OpenedAt.UTC()); err != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("insert address assignment: %w", err))
	}
	if _, err = tx.ExecContext(ctx, `
INSERT INTO payment_lifecycle_history
    (history_id, tenant_id, invoice_id, event_type, revision, occurred_at, payload)
VALUES ($1, $2, $3, $4, $5, $6, $7)`, string(historyID), string(record.Command.TenantID),
		string(invoiceID), record.LifecycleEvent.Type, record.LifecycleEvent.Revision,
		record.LifecycleEvent.OccurredAt.UTC(), payload); err != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("insert lifecycle history: %w", err))
	}
	if _, err = tx.ExecContext(ctx, `
INSERT INTO payment_lifecycle_outbox_events
    (event_id, tenant_id, aggregate_type, aggregate_id, event_type, aggregate_revision, schema_version, occurred_at, payload)
VALUES ($1, $2, 'invoice', $3, $4, $5, $6, $7, $8)`, string(outboxID),
		string(record.Command.TenantID), string(invoiceID), record.OutboxEvent.Type, record.Revision,
		record.OutboxEvent.SchemaVersion, record.OutboxEvent.OccurredAt.UTC(), payload); err != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("insert outbox event: %w", err))
	}
	if _, err = tx.ExecContext(ctx, `
INSERT INTO payment_lifecycle_idempotency_receipts
    (tenant_id, operation, idempotency_key, request_hash, invoice_id, response, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`, string(record.Command.TenantID), record.Operation,
		string(record.Command.IdempotencyKey), record.RequestHash, string(invoiceID), payload, record.OpenedAt.UTC()); err != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("insert idempotency receipt: %w", err))
	}

	if err = tx.Commit(); err != nil {
		return paymentlifecycle.StoreResult{}, storageFailure(fmt.Errorf("commit open payment: %w", err))
	}
	return paymentlifecycle.StoreResult{Disposition: paymentlifecycle.StoreCreated, View: view}, nil
}

func storageFailure(err error) error {
	return &paymentlifecycle.Error{Code: paymentlifecycle.CodeStorageUnavailable, Cause: err}
}

func isConstraint(err error, name string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == name
}

func validateReceiptIdentity(operation, requestHash string) error {
	if operation != paymentlifecycle.OperationOpen {
		return fmt.Errorf("unsupported receipt operation %q", operation)
	}
	if len(requestHash) != 64 {
		return fmt.Errorf("request hash must be 64 lowercase hexadecimal characters")
	}
	for _, character := range requestHash {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return fmt.Errorf("request hash must be 64 lowercase hexadecimal characters")
		}
	}
	return nil
}
