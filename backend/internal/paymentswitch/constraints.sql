-- Postgres-only guarantees the GORM models cannot express. Idempotent: applied by
-- AutoMigrate in dev/test and repeated verbatim in migration 2026100703.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_intents'::regclass AND conname = 'switch_payment_intents_status_check') THEN
        ALTER TABLE switch_payment_intents ADD CONSTRAINT switch_payment_intents_status_check
            CHECK (status IN ('requires_payment_method', 'requires_confirmation', 'requires_action', 'processing',
                              'requires_capture', 'partially_captured', 'partially_paid', 'succeeded', 'failed', 'cancelled'));
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
                              'partial_charged', 'partially_paid', 'underpaid', 'overpaid', 'capture_failed', 'authorization_failed',
                              'void_initiated', 'voided', 'void_failed', 'failure'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_attempts'::regclass AND conname = 'switch_payment_attempts_amounts_check') THEN
        ALTER TABLE switch_payment_attempts ADD CONSTRAINT switch_payment_attempts_amounts_check
            CHECK (amount > 0 AND amount_to_capture >= 0 AND amount_captured >= 0 AND (amount_received IS NULL OR amount_received >= 0));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_attempts'::regclass AND conname = 'switch_payment_attempts_intent_id_fkey') THEN
        ALTER TABLE switch_payment_attempts ADD CONSTRAINT switch_payment_attempts_intent_id_fkey
            FOREIGN KEY (intent_id) REFERENCES switch_payment_intents (id) ON DELETE RESTRICT;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_refunds'::regclass AND conname = 'switch_refunds_status_check') THEN
        ALTER TABLE switch_refunds ADD CONSTRAINT switch_refunds_status_check
            CHECK (status IN ('initiated', 'pending', 'succeeded', 'failed'));
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

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_intents'::regclass AND conname = 'switch_payment_intents_environment_check') THEN
        ALTER TABLE switch_payment_intents ADD CONSTRAINT switch_payment_intents_environment_check CHECK (environment IN ('test', 'live'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_payment_attempts'::regclass AND conname = 'switch_payment_attempts_environment_check') THEN
        ALTER TABLE switch_payment_attempts ADD CONSTRAINT switch_payment_attempts_environment_check CHECK (environment IN ('test', 'live'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_refunds'::regclass AND conname = 'switch_refunds_environment_check') THEN
        ALTER TABLE switch_refunds ADD CONSTRAINT switch_refunds_environment_check CHECK (environment IN ('test', 'live'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_anomalies'::regclass AND conname = 'switch_anomalies_entity_check') THEN
        ALTER TABLE switch_anomalies ADD CONSTRAINT switch_anomalies_entity_check
            CHECK (entity IN ('intent', 'attempt', 'refund'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'switch_status_transitions'::regclass AND conname = 'switch_status_transitions_entity_check') THEN
        ALTER TABLE switch_status_transitions ADD CONSTRAINT switch_status_transitions_entity_check
            CHECK (entity IN ('intent', 'attempt', 'refund'));
    END IF;
END;
$$;
