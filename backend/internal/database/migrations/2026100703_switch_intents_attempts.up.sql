-- Switch core: payment intents, attempts, refunds, the webhook replay guard and the transition audit trail.
-- Additive and idempotent so a development database already shaped by AutoMigrate converges.
-- Design: .scratch/payments-v1/issues/05-switch-core.md

CREATE TABLE IF NOT EXISTS switch_payment_intents (
    id varchar(64) PRIMARY KEY,
    merchant_id varchar(128) NOT NULL,
    platform_id varchar(128) NOT NULL DEFAULT '',
    idempotency_key varchar(128) NOT NULL,
    request_hash char(64) NOT NULL,
    status varchar(32) NOT NULL,
    amount numeric(38, 18) NOT NULL,
    asset varchar(16) NOT NULL,
    amount_captured numeric(38, 18) NOT NULL DEFAULT 0,
    amount_refunded numeric(38, 18) NOT NULL DEFAULT 0,
    capture_method varchar(16) NOT NULL,
    payment_method_type varchar(16) NOT NULL DEFAULT '',
    payment_method jsonb NOT NULL DEFAULT '{}'::jsonb,
    connector_code varchar(32) NOT NULL DEFAULT '',
    active_attempt_id varchar(64) NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    return_url text NOT NULL DEFAULT '',
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    next_action jsonb,
    last_error_code varchar(64) NOT NULL DEFAULT '',
    last_error_message text NOT NULL DEFAULT '',
    version bigint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE UNIQUE INDEX IF NOT EXISTS switch_intents_merchant_idempotency_key
    ON switch_payment_intents (merchant_id, idempotency_key);
CREATE INDEX IF NOT EXISTS idx_switch_payment_intents_merchant_id ON switch_payment_intents (merchant_id);
CREATE INDEX IF NOT EXISTS idx_switch_payment_intents_status ON switch_payment_intents (status);

CREATE TABLE IF NOT EXISTS switch_payment_attempts (
    id varchar(64) PRIMARY KEY,
    intent_id varchar(64) NOT NULL,
    merchant_id varchar(128) NOT NULL,
    connector_code varchar(32) NOT NULL,
    status varchar(32) NOT NULL,
    raw_status varchar(64) NOT NULL DEFAULT '',
    amount numeric(38, 18) NOT NULL,
    asset varchar(16) NOT NULL,
    amount_captured numeric(38, 18) NOT NULL DEFAULT 0,
    amount_received numeric(38, 18) NOT NULL DEFAULT 0,
    connector_transaction_id varchar(128),
    selection_reason text NOT NULL DEFAULT '',
    error_code varchar(64) NOT NULL DEFAULT '',
    error_message text NOT NULL DEFAULT '',
    next_action jsonb,
    version bigint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE UNIQUE INDEX IF NOT EXISTS switch_attempts_connector_tx_key
    ON switch_payment_attempts (connector_code, connector_transaction_id);
CREATE INDEX IF NOT EXISTS idx_switch_payment_attempts_intent_id ON switch_payment_attempts (intent_id);
CREATE INDEX IF NOT EXISTS idx_switch_payment_attempts_merchant_id ON switch_payment_attempts (merchant_id);
CREATE INDEX IF NOT EXISTS idx_switch_payment_attempts_status ON switch_payment_attempts (status);

CREATE TABLE IF NOT EXISTS switch_refunds (
    id varchar(64) PRIMARY KEY,
    intent_id varchar(64) NOT NULL,
    attempt_id varchar(64) NOT NULL,
    merchant_id varchar(128) NOT NULL,
    connector_code varchar(32) NOT NULL,
    idempotency_key varchar(128) NOT NULL,
    request_hash char(64) NOT NULL,
    status varchar(32) NOT NULL,
    raw_status varchar(64) NOT NULL DEFAULT '',
    amount numeric(38, 18) NOT NULL,
    asset varchar(16) NOT NULL,
    connector_refund_id varchar(128),
    reason text NOT NULL DEFAULT '',
    error_code varchar(64) NOT NULL DEFAULT '',
    error_message text NOT NULL DEFAULT '',
    version bigint NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE UNIQUE INDEX IF NOT EXISTS switch_refunds_merchant_idempotency_key
    ON switch_refunds (merchant_id, idempotency_key);
CREATE UNIQUE INDEX IF NOT EXISTS switch_refunds_connector_refund_key
    ON switch_refunds (connector_code, connector_refund_id);
CREATE INDEX IF NOT EXISTS idx_switch_refunds_intent_id ON switch_refunds (intent_id);
CREATE INDEX IF NOT EXISTS idx_switch_refunds_attempt_id ON switch_refunds (attempt_id);
CREATE INDEX IF NOT EXISTS idx_switch_refunds_status ON switch_refunds (status);

CREATE TABLE IF NOT EXISTS switch_webhook_events (
    id bigserial PRIMARY KEY,
    connector_code varchar(32) NOT NULL,
    event_id varchar(255) NOT NULL,
    received_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE UNIQUE INDEX IF NOT EXISTS switch_webhook_events_connector_event_key
    ON switch_webhook_events (connector_code, event_id);

CREATE TABLE IF NOT EXISTS switch_status_transitions (
    id bigserial PRIMARY KEY,
    entity varchar(16) NOT NULL,
    entity_id varchar(64) NOT NULL,
    from_status varchar(32) NOT NULL,
    to_status varchar(32) NOT NULL,
    reason text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX IF NOT EXISTS idx_switch_status_transitions_entity_id ON switch_status_transitions (entity_id);

COMMENT ON TABLE switch_payment_attempts IS
    'One row per connector try. raw_status is the connector''s own word; status is the switch vocabulary from paymentswitch/status_map.go.';

-- Postgres-only guarantees the GORM models cannot express. Idempotent: applied by
-- AutoMigrate in dev/test and repeated verbatim in migration 2026100703.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_intents'::regclass AND conname = 'switch_payment_intents_status_check') THEN
        ALTER TABLE switch_payment_intents ADD CONSTRAINT switch_payment_intents_status_check
            CHECK (status IN ('requires_payment_method', 'requires_confirmation', 'requires_action', 'processing',
                              'requires_capture', 'partially_captured', 'succeeded', 'failed', 'cancelled'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_intents'::regclass AND conname = 'switch_payment_intents_capture_method_check') THEN
        ALTER TABLE switch_payment_intents ADD CONSTRAINT switch_payment_intents_capture_method_check
            CHECK (capture_method IN ('automatic', 'manual'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_intents'::regclass AND conname = 'switch_payment_intents_amounts_check') THEN
        ALTER TABLE switch_payment_intents ADD CONSTRAINT switch_payment_intents_amounts_check
            CHECK (amount > 0 AND amount_captured >= 0 AND amount_refunded >= 0 AND amount_refunded <= amount_captured);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_intents'::regclass AND conname = 'switch_payment_intents_asset_check') THEN
        ALTER TABLE switch_payment_intents ADD CONSTRAINT switch_payment_intents_asset_check
            CHECK (length(asset) BETWEEN 1 AND 16 AND asset = btrim(asset));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_intents'::regclass AND conname = 'switch_payment_intents_request_hash_check') THEN
        ALTER TABLE switch_payment_intents ADD CONSTRAINT switch_payment_intents_request_hash_check
            CHECK (request_hash ~ '^[0-9a-f]{64}$');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_attempts'::regclass AND conname = 'switch_payment_attempts_status_check') THEN
        ALTER TABLE switch_payment_attempts ADD CONSTRAINT switch_payment_attempts_status_check
            CHECK (status IN ('started', 'pending', 'authentication_pending', 'authorized', 'capture_initiated', 'charged',
                              'partial_charged', 'partially_paid', 'overpaid', 'capture_failed', 'authorization_failed',
                              'void_initiated', 'voided', 'void_failed', 'failure'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_attempts'::regclass AND conname = 'switch_payment_attempts_amounts_check') THEN
        ALTER TABLE switch_payment_attempts ADD CONSTRAINT switch_payment_attempts_amounts_check
            CHECK (amount > 0 AND amount_captured >= 0 AND amount_received >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_attempts'::regclass AND conname = 'switch_payment_attempts_intent_id_fkey') THEN
        ALTER TABLE switch_payment_attempts ADD CONSTRAINT switch_payment_attempts_intent_id_fkey
            FOREIGN KEY (intent_id) REFERENCES switch_payment_intents (id) ON DELETE RESTRICT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_refunds'::regclass AND conname = 'switch_refunds_status_check') THEN
        ALTER TABLE switch_refunds ADD CONSTRAINT switch_refunds_status_check
            CHECK (status IN ('pending', 'succeeded', 'failed'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_refunds'::regclass AND conname = 'switch_refunds_amount_check') THEN
        ALTER TABLE switch_refunds ADD CONSTRAINT switch_refunds_amount_check CHECK (amount > 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_refunds'::regclass AND conname = 'switch_refunds_intent_id_fkey') THEN
        ALTER TABLE switch_refunds ADD CONSTRAINT switch_refunds_intent_id_fkey
            FOREIGN KEY (intent_id) REFERENCES switch_payment_intents (id) ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_refunds'::regclass AND conname = 'switch_refunds_attempt_id_fkey') THEN
        ALTER TABLE switch_refunds ADD CONSTRAINT switch_refunds_attempt_id_fkey
            FOREIGN KEY (attempt_id) REFERENCES switch_payment_attempts (id) ON DELETE RESTRICT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_status_transitions'::regclass AND conname = 'switch_status_transitions_entity_check') THEN
        ALTER TABLE switch_status_transitions ADD CONSTRAINT switch_status_transitions_entity_check
            CHECK (entity IN ('intent', 'attempt', 'refund'));
    END IF;
END;
$$;
