-- Postgres-only guarantees the GORM models cannot express. Idempotent: applied by
-- AutoMigrate in dev/test and repeated verbatim in the checksummed migration.
-- Trigger functions pin search_path and qualify every reference with the schema they were
-- installed in, so a session that can SET search_path cannot shadow a table or txid_current().

ALTER TABLE ledger_journals ADD COLUMN IF NOT EXISTS posting_txid bigint NOT NULL DEFAULT 0;
ALTER TABLE ledger_journals ADD COLUMN IF NOT EXISTS posting_started_at timestamptz NOT NULL DEFAULT '1970-01-01 00:00:00+00';
ALTER TABLE ledger_accounts ALTER COLUMN asset TYPE varchar(32);
ALTER TABLE ledger_lines ALTER COLUMN asset TYPE varchar(32);

-- Environment isolation (ticket 13). The four-column index stays until every binary
-- infers the five-column one; ticket 17 drops it.
ALTER TABLE ledger_accounts ADD COLUMN IF NOT EXISTS environment varchar(8) NOT NULL DEFAULT 'test';
CREATE UNIQUE INDEX IF NOT EXISTS ledger_accounts_env_owner_asset_kind_key
    ON ledger_accounts (environment, owner_type, owner_id, asset, kind);
ALTER TABLE ledger_journals ADD COLUMN IF NOT EXISTS environment varchar(8) NOT NULL DEFAULT 'test';
CREATE INDEX IF NOT EXISTS ledger_journals_environment_idx ON ledger_journals (environment);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_accounts'::regclass AND conname = 'ledger_accounts_environment_check') THEN
        ALTER TABLE ledger_accounts ADD CONSTRAINT ledger_accounts_environment_check
            CHECK (environment IN ('test', 'live')) NOT VALID;
        ALTER TABLE ledger_accounts VALIDATE CONSTRAINT ledger_accounts_environment_check;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_journals'::regclass AND conname = 'ledger_journals_environment_check') THEN
        ALTER TABLE ledger_journals ADD CONSTRAINT ledger_journals_environment_check
            CHECK (environment IN ('test', 'live')) NOT VALID;
        ALTER TABLE ledger_journals VALIDATE CONSTRAINT ledger_journals_environment_check;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_accounts'::regclass AND conname = 'ledger_accounts_id_asset_key') THEN
        ALTER TABLE ledger_accounts ADD CONSTRAINT ledger_accounts_id_asset_key UNIQUE (id, asset);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_accounts'::regclass AND conname = 'ledger_accounts_owner_type_check') THEN
        ALTER TABLE ledger_accounts ADD CONSTRAINT ledger_accounts_owner_type_check
            CHECK (owner_type IN ('member', 'platform', 'connector', 'chain', 'fees', 'reserve'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_accounts'::regclass AND conname = 'ledger_accounts_kind_check') THEN
        ALTER TABLE ledger_accounts ADD CONSTRAINT ledger_accounts_kind_check
            CHECK (kind IN ('asset', 'liability', 'income', 'expense'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_accounts'::regclass AND conname = 'ledger_accounts_owner_id_check') THEN
        ALTER TABLE ledger_accounts ADD CONSTRAINT ledger_accounts_owner_id_check
            CHECK (length(owner_id) BETWEEN 1 AND 128 AND owner_id = btrim(owner_id));
    END IF;
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_accounts'::regclass AND conname = 'ledger_accounts_asset_check') THEN
        ALTER TABLE ledger_accounts DROP CONSTRAINT ledger_accounts_asset_check;
    END IF;
    ALTER TABLE ledger_accounts ADD CONSTRAINT ledger_accounts_asset_check
        CHECK (length(asset) BETWEEN 1 AND 32 AND asset ~ '^[A-Z0-9_-]+(\.[A-Z0-9_-]+)*$');
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_lines'::regclass AND conname = 'ledger_lines_asset_check') THEN
        ALTER TABLE ledger_lines DROP CONSTRAINT ledger_lines_asset_check;
    END IF;
    ALTER TABLE ledger_lines ADD CONSTRAINT ledger_lines_asset_check
        CHECK (length(asset) BETWEEN 1 AND 32 AND asset ~ '^[A-Z0-9_-]+(\.[A-Z0-9_-]+)*$');

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_journals'::regclass AND conname = 'ledger_journals_kind_check') THEN
        ALTER TABLE ledger_journals ADD CONSTRAINT ledger_journals_kind_check
            CHECK (kind IN ('payment', 'fee', 'conversion', 'settlement', 'refund', 'adjustment', 'transfer'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_journals'::regclass AND conname = 'ledger_journals_idempotency_key_check') THEN
        ALTER TABLE ledger_journals ADD CONSTRAINT ledger_journals_idempotency_key_check
            CHECK (length(idempotency_key) BETWEEN 1 AND 128);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_journals'::regclass AND conname = 'ledger_journals_request_hash_check') THEN
        ALTER TABLE ledger_journals ADD CONSTRAINT ledger_journals_request_hash_check
            CHECK (request_hash ~ '^[0-9a-f]{64}$');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_journals'::regclass AND conname = 'ledger_journals_metadata_check') THEN
        ALTER TABLE ledger_journals ADD CONSTRAINT ledger_journals_metadata_check
            CHECK (jsonb_typeof(metadata) = 'object');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_lines'::regclass AND conname = 'ledger_lines_amount_check') THEN
        ALTER TABLE ledger_lines ADD CONSTRAINT ledger_lines_amount_check CHECK (amount <> 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_lines'::regclass AND conname = 'ledger_lines_journal_id_fkey') THEN
        ALTER TABLE ledger_lines ADD CONSTRAINT ledger_lines_journal_id_fkey
            FOREIGN KEY (journal_id) REFERENCES ledger_journals (id) ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_lines'::regclass AND conname = 'ledger_lines_account_asset_fkey') THEN
        ALTER TABLE ledger_lines ADD CONSTRAINT ledger_lines_account_asset_fkey
            FOREIGN KEY (account_id, asset) REFERENCES ledger_accounts (id, asset) ON DELETE RESTRICT;
    END IF;
END;
$$;

-- Functions are created through format() so the ledger schema is baked into their bodies.
DO $install$
DECLARE
    s text := current_schema();
BEGIN
    EXECUTE format($f$
        CREATE OR REPLACE FUNCTION %1$I.ledger_reject_mutation()
        RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog, pg_temp AS $body$
        BEGIN
            RAISE EXCEPTION 'ledger: %% is append-only; post a compensating journal instead', TG_TABLE_NAME;
        END;
        $body$$f$, s);

    EXECUTE format($f$
        CREATE OR REPLACE FUNCTION %1$I.ledger_check_journal_balance()
        RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog, pg_temp AS $body$
        DECLARE
            total numeric(38, 18);
        BEGIN
            SELECT COALESCE(pg_catalog.sum(amount), 0::numeric) INTO total
              FROM %1$I.ledger_lines
             WHERE journal_id = NEW.journal_id AND asset = NEW.asset;
            IF total <> 0 THEN
                RAISE EXCEPTION 'ledger: journal %% does not balance for asset %% (sum %%)', NEW.journal_id, NEW.asset, total;
            END IF;
            RETURN NULL;
        END;
        $body$$f$, s);

    EXECUTE format($f$
        CREATE OR REPLACE FUNCTION %1$I.ledger_stamp_journal_txid()
        RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog, pg_temp AS $body$
        BEGIN
            NEW.posting_txid := pg_catalog.txid_current();
            NEW.posting_started_at := pg_catalog.transaction_timestamp();
            RETURN NEW;
        END;
        $body$$f$, s);

    EXECUTE format($f$
        CREATE OR REPLACE FUNCTION %1$I.ledger_check_line_same_transaction()
        RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog, pg_temp AS $body$
        DECLARE
            journal_txid bigint;
            journal_started timestamptz;
        BEGIN
            SELECT posting_txid, posting_started_at INTO journal_txid, journal_started
              FROM %1$I.ledger_journals WHERE id = NEW.journal_id;
            IF journal_txid IS NULL THEN
                RAISE EXCEPTION 'ledger: journal %% does not exist', NEW.journal_id;
            END IF;
            IF journal_txid <> pg_catalog.txid_current() OR journal_started <> pg_catalog.transaction_timestamp() THEN
                RAISE EXCEPTION 'ledger: journal %% is sealed; it was posted in another transaction', NEW.journal_id;
            END IF;
            RETURN NEW;
        END;
        $body$$f$, s);

    EXECUTE format($f$
        CREATE OR REPLACE FUNCTION %1$I.ledger_check_journal_has_lines()
        RETURNS trigger LANGUAGE plpgsql SET search_path = pg_catalog, pg_temp AS $body$
        BEGIN
            IF NOT EXISTS (SELECT 1 FROM %1$I.ledger_lines WHERE journal_id = NEW.id) THEN
                RAISE EXCEPTION 'ledger: journal %% has no lines', NEW.id;
            END IF;
            RETURN NULL;
        END;
        $body$$f$, s);

    -- The only sanctioned rewrite of ledger rows: relabelling a database adopted as live (cmd/migrate
    -- adopt-live). SECURITY DEFINER so the table owner's right to pause the append-only triggers is
    -- exercised here and nowhere else; it refuses once any live row exists.
    EXECUTE format($f$
        CREATE OR REPLACE FUNCTION %1$I.ledger_adopt_environment(target text)
        RETURNS TABLE (accounts bigint, journals bigint)
        LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, pg_temp AS $body$
        DECLARE
            relabelled_accounts bigint;
            relabelled_journals bigint;
        BEGIN
            IF target <> 'live' THEN
                RAISE EXCEPTION 'ledger: adoption only moves test rows to live, not to %%', target;
            END IF;
            IF EXISTS (SELECT 1 FROM %1$I.ledger_accounts WHERE environment = 'live')
               OR EXISTS (SELECT 1 FROM %1$I.ledger_journals WHERE environment = 'live') THEN
                RAISE EXCEPTION 'ledger: live rows already exist; this database was adopted or served live before';
            END IF;
            ALTER TABLE %1$I.ledger_accounts DISABLE TRIGGER ledger_accounts_append_only;
            ALTER TABLE %1$I.ledger_journals DISABLE TRIGGER ledger_journals_append_only;
            UPDATE %1$I.ledger_accounts SET environment = 'live' WHERE environment = 'test';
            GET DIAGNOSTICS relabelled_accounts = ROW_COUNT;
            UPDATE %1$I.ledger_journals SET environment = 'live' WHERE environment = 'test';
            GET DIAGNOSTICS relabelled_journals = ROW_COUNT;
            ALTER TABLE %1$I.ledger_accounts ENABLE TRIGGER ledger_accounts_append_only;
            ALTER TABLE %1$I.ledger_journals ENABLE TRIGGER ledger_journals_append_only;
            RETURN QUERY SELECT relabelled_accounts, relabelled_journals;
        END;
        $body$$f$, s);
    EXECUTE format('REVOKE ALL ON FUNCTION %I.ledger_adopt_environment(text) FROM PUBLIC', s);
END;
$install$;

DROP TRIGGER IF EXISTS ledger_accounts_append_only ON ledger_accounts;
CREATE TRIGGER ledger_accounts_append_only
    BEFORE UPDATE OR DELETE ON ledger_accounts
    FOR EACH ROW EXECUTE FUNCTION ledger_reject_mutation();

DROP TRIGGER IF EXISTS ledger_journals_append_only ON ledger_journals;
CREATE TRIGGER ledger_journals_append_only
    BEFORE UPDATE OR DELETE ON ledger_journals
    FOR EACH ROW EXECUTE FUNCTION ledger_reject_mutation();

DROP TRIGGER IF EXISTS ledger_lines_append_only ON ledger_lines;
CREATE TRIGGER ledger_lines_append_only
    BEFORE UPDATE OR DELETE ON ledger_lines
    FOR EACH ROW EXECUTE FUNCTION ledger_reject_mutation();

DROP TRIGGER IF EXISTS ledger_accounts_no_truncate ON ledger_accounts;
CREATE TRIGGER ledger_accounts_no_truncate
    BEFORE TRUNCATE ON ledger_accounts
    FOR EACH STATEMENT EXECUTE FUNCTION ledger_reject_mutation();

DROP TRIGGER IF EXISTS ledger_journals_no_truncate ON ledger_journals;
CREATE TRIGGER ledger_journals_no_truncate
    BEFORE TRUNCATE ON ledger_journals
    FOR EACH STATEMENT EXECUTE FUNCTION ledger_reject_mutation();

DROP TRIGGER IF EXISTS ledger_lines_no_truncate ON ledger_lines;
CREATE TRIGGER ledger_lines_no_truncate
    BEFORE TRUNCATE ON ledger_lines
    FOR EACH STATEMENT EXECUTE FUNCTION ledger_reject_mutation();

DROP TRIGGER IF EXISTS ledger_journals_stamp_txid ON ledger_journals;
CREATE TRIGGER ledger_journals_stamp_txid
    BEFORE INSERT ON ledger_journals
    FOR EACH ROW EXECUTE FUNCTION ledger_stamp_journal_txid();

DROP TRIGGER IF EXISTS ledger_lines_same_transaction ON ledger_lines;
CREATE TRIGGER ledger_lines_same_transaction
    BEFORE INSERT ON ledger_lines
    FOR EACH ROW EXECUTE FUNCTION ledger_check_line_same_transaction();

DROP TRIGGER IF EXISTS ledger_lines_balance ON ledger_lines;
CREATE CONSTRAINT TRIGGER ledger_lines_balance
    AFTER INSERT ON ledger_lines
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION ledger_check_journal_balance();

DROP TRIGGER IF EXISTS ledger_journals_has_lines ON ledger_journals;
CREATE CONSTRAINT TRIGGER ledger_journals_has_lines
    AFTER INSERT ON ledger_journals
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION ledger_check_journal_has_lines();
