-- Expand-only foundation for PaymentLifecycle.Open.
-- This migration neither reads nor rewrites legacy payment_requests data.

CREATE TABLE payment_lifecycle_invoices (
    invoice_id text PRIMARY KEY CHECK (length(invoice_id) BETWEEN 1 AND 128),
    tenant_id text NOT NULL CHECK (length(tenant_id) BETWEEN 1 AND 128),
    merchant_reference text NOT NULL CHECK (length(merchant_reference) BETWEEN 1 AND 128),
    invoice_currency char(3) NOT NULL CHECK (invoice_currency ~ '^[A-Z]{3}$'),
    invoice_minor_units numeric(78, 0) NOT NULL CHECK (invoice_minor_units > 0),
    state text NOT NULL CHECK (state IN ('open')),
    revision bigint NOT NULL CHECK (revision = 1),
    expires_at timestamptz NOT NULL,
    opened_at timestamptz NOT NULL,
    UNIQUE (tenant_id, merchant_reference),
    UNIQUE (invoice_id, tenant_id),
    UNIQUE (invoice_id, tenant_id, invoice_currency, invoice_minor_units),
    CHECK (expires_at > opened_at)
);

CREATE TABLE payment_lifecycle_payment_methods (
    invoice_id text PRIMARY KEY,
    tenant_id text NOT NULL,
    chain_id text NOT NULL CHECK (length(chain_id) BETWEEN 1 AND 128),
    asset_id text NOT NULL CHECK (length(asset_id) BETWEEN 1 AND 256),
    selected_at timestamptz NOT NULL,
    UNIQUE (invoice_id, tenant_id),
    UNIQUE (invoice_id, tenant_id, chain_id, asset_id),
    FOREIGN KEY (invoice_id, tenant_id)
        REFERENCES payment_lifecycle_invoices (invoice_id, tenant_id) ON DELETE RESTRICT
);

CREATE TABLE payment_lifecycle_quotes (
    quote_id text NOT NULL CHECK (length(quote_id) BETWEEN 1 AND 128),
    invoice_id text NOT NULL,
    tenant_id text NOT NULL,
    invoice_currency char(3) NOT NULL,
    invoice_minor_units numeric(78, 0) NOT NULL CHECK (invoice_minor_units > 0),
    chain_id text NOT NULL,
    asset_id text NOT NULL,
    required_atomic_units numeric(78, 0) NOT NULL CHECK (required_atomic_units > 0),
    asset_decimals smallint NOT NULL CHECK (asset_decimals BETWEEN 0 AND 77),
    rate_numerator numeric(78, 0) NOT NULL CHECK (rate_numerator > 0),
    rate_denominator numeric(78, 0) NOT NULL CHECK (rate_denominator > 0),
    source text NOT NULL CHECK (length(source) BETWEEN 1 AND 128),
    quoted_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    rounding text NOT NULL CHECK (rounding IN ('ceil', 'floor', 'half_up', 'half_even')),
    PRIMARY KEY (tenant_id, quote_id),
    UNIQUE (invoice_id),
    FOREIGN KEY (invoice_id, tenant_id)
        REFERENCES payment_lifecycle_invoices (invoice_id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (invoice_id, tenant_id, invoice_currency, invoice_minor_units)
        REFERENCES payment_lifecycle_invoices
            (invoice_id, tenant_id, invoice_currency, invoice_minor_units) ON DELETE RESTRICT,
    FOREIGN KEY (invoice_id, tenant_id, chain_id, asset_id)
        REFERENCES payment_lifecycle_payment_methods
            (invoice_id, tenant_id, chain_id, asset_id) ON DELETE RESTRICT,
    CHECK (expires_at > quoted_at)
);

CREATE TABLE payment_lifecycle_idempotency_receipts (
    tenant_id text NOT NULL CHECK (length(tenant_id) BETWEEN 1 AND 128),
    operation text NOT NULL CHECK (operation = 'payment_lifecycle.open'),
    idempotency_key text NOT NULL CHECK (length(idempotency_key) BETWEEN 16 AND 128),
    request_hash char(64) NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    invoice_id text NOT NULL,
    response jsonb NOT NULL CHECK (jsonb_typeof(response) = 'object'),
    created_at timestamptz NOT NULL,
    PRIMARY KEY (tenant_id, operation, idempotency_key),
    FOREIGN KEY (invoice_id, tenant_id)
        REFERENCES payment_lifecycle_invoices (invoice_id, tenant_id) ON DELETE RESTRICT
);

CREATE TABLE payment_lifecycle_deposit_addresses (
    address_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id text NOT NULL CHECK (length(tenant_id) BETWEEN 1 AND 128),
    chain_id text NOT NULL CHECK (length(chain_id) BETWEEN 1 AND 128),
    asset_id text NOT NULL CHECK (length(asset_id) BETWEEN 1 AND 256),
    address text NOT NULL CHECK (length(address) BETWEEN 1 AND 512),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (tenant_id, chain_id, asset_id, address),
    UNIQUE (chain_id, asset_id, address),
    UNIQUE (address_id, tenant_id),
    UNIQUE (address_id, tenant_id, chain_id, asset_id, address)
);

CREATE INDEX payment_lifecycle_deposit_addresses_available_idx
    ON payment_lifecycle_deposit_addresses (tenant_id, chain_id, asset_id, address_id);

CREATE TABLE payment_lifecycle_address_assignments (
    assignment_id text NOT NULL UNIQUE CHECK (length(assignment_id) BETWEEN 1 AND 128),
    invoice_id text PRIMARY KEY,
    tenant_id text NOT NULL,
    address_id bigint NOT NULL UNIQUE,
    chain_id text NOT NULL,
    asset_id text NOT NULL,
    address text NOT NULL,
    assigned_at timestamptz NOT NULL,
    UNIQUE (invoice_id, tenant_id),
    FOREIGN KEY (invoice_id, tenant_id)
        REFERENCES payment_lifecycle_invoices (invoice_id, tenant_id) ON DELETE RESTRICT,
    FOREIGN KEY (invoice_id, tenant_id, chain_id, asset_id)
        REFERENCES payment_lifecycle_payment_methods
            (invoice_id, tenant_id, chain_id, asset_id) ON DELETE RESTRICT,
    FOREIGN KEY (address_id, tenant_id, chain_id, asset_id, address)
        REFERENCES payment_lifecycle_deposit_addresses
            (address_id, tenant_id, chain_id, asset_id, address) ON DELETE RESTRICT
);

CREATE TABLE payment_lifecycle_history (
    history_id text PRIMARY KEY CHECK (length(history_id) BETWEEN 1 AND 128),
    tenant_id text NOT NULL,
    invoice_id text NOT NULL,
    event_type text NOT NULL CHECK (event_type = 'invoice.opened'),
    revision bigint NOT NULL CHECK (revision = 1),
    occurred_at timestamptz NOT NULL,
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    UNIQUE (tenant_id, invoice_id, revision),
    FOREIGN KEY (invoice_id, tenant_id)
        REFERENCES payment_lifecycle_invoices (invoice_id, tenant_id) ON DELETE RESTRICT
);

CREATE TABLE payment_lifecycle_outbox_events (
    event_id text PRIMARY KEY CHECK (length(event_id) BETWEEN 1 AND 128),
    tenant_id text NOT NULL,
    aggregate_type text NOT NULL CHECK (aggregate_type = 'invoice'),
    aggregate_id text NOT NULL,
    event_type text NOT NULL CHECK (event_type = 'invoice.opened'),
    aggregate_revision bigint NOT NULL CHECK (aggregate_revision = 1),
    schema_version smallint NOT NULL CHECK (schema_version > 0),
    occurred_at timestamptz NOT NULL,
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    UNIQUE (tenant_id, aggregate_type, aggregate_id, aggregate_revision),
    FOREIGN KEY (aggregate_id, tenant_id)
        REFERENCES payment_lifecycle_invoices (invoice_id, tenant_id) ON DELETE RESTRICT
);

CREATE OR REPLACE FUNCTION payment_lifecycle_reject_immutable_change()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION '% is immutable; append a compensating domain fact instead', TG_TABLE_NAME;
END;
$$;

CREATE OR REPLACE FUNCTION payment_lifecycle_validate_quote_expiry()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    invoice_expiry timestamptz;
    invoice_opened_at timestamptz;
BEGIN
    SELECT expires_at, opened_at INTO STRICT invoice_expiry, invoice_opened_at
      FROM payment_lifecycle_invoices
     WHERE invoice_id = NEW.invoice_id AND tenant_id = NEW.tenant_id;
    IF NEW.quoted_at > invoice_opened_at THEN
        RAISE EXCEPTION 'Quote time must not be after Invoice open time';
    END IF;
    IF NEW.expires_at <= invoice_opened_at THEN
        RAISE EXCEPTION 'Quote must remain valid after Invoice open time';
    END IF;
    IF NEW.expires_at > invoice_expiry THEN
        RAISE EXCEPTION 'Quote expiry must not exceed Invoice expiry';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER payment_lifecycle_quotes_validate_expiry
    BEFORE INSERT OR UPDATE ON payment_lifecycle_quotes
    FOR EACH ROW EXECUTE FUNCTION payment_lifecycle_validate_quote_expiry();

CREATE TRIGGER payment_lifecycle_quotes_immutable
    BEFORE UPDATE OR DELETE ON payment_lifecycle_quotes
    FOR EACH ROW EXECUTE FUNCTION payment_lifecycle_reject_immutable_change();

CREATE TRIGGER payment_lifecycle_payment_methods_immutable
    BEFORE UPDATE OR DELETE ON payment_lifecycle_payment_methods
    FOR EACH ROW EXECUTE FUNCTION payment_lifecycle_reject_immutable_change();

CREATE TRIGGER payment_lifecycle_idempotency_receipts_immutable
    BEFORE UPDATE OR DELETE ON payment_lifecycle_idempotency_receipts
    FOR EACH ROW EXECUTE FUNCTION payment_lifecycle_reject_immutable_change();

CREATE TRIGGER payment_lifecycle_address_assignments_immutable
    BEFORE UPDATE OR DELETE ON payment_lifecycle_address_assignments
    FOR EACH ROW EXECUTE FUNCTION payment_lifecycle_reject_immutable_change();

CREATE TRIGGER payment_lifecycle_history_immutable
    BEFORE UPDATE OR DELETE ON payment_lifecycle_history
    FOR EACH ROW EXECUTE FUNCTION payment_lifecycle_reject_immutable_change();

CREATE TRIGGER payment_lifecycle_outbox_events_immutable
    BEFORE UPDATE OR DELETE ON payment_lifecycle_outbox_events
    FOR EACH ROW EXECUTE FUNCTION payment_lifecycle_reject_immutable_change();

COMMENT ON TABLE payment_lifecycle_invoices IS
    'Expand-only PaymentLifecycle.Open write model; not a legacy payment_requests replacement yet.';
COMMENT ON COLUMN payment_lifecycle_quotes.required_atomic_units IS
    'Exact unsigned chain amount in atomic units; never a display decimal or floating-point value.';
