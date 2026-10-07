-- Double-entry ledger: additive tables plus the append-only and balance guarantees.
-- Idempotent so a development database already shaped by AutoMigrate converges.
-- Design: .scratch/payments-v1/issues/01-ledger.md

CREATE TABLE IF NOT EXISTS ledger_accounts (
    id bigserial PRIMARY KEY,
    owner_type varchar(16) NOT NULL,
    owner_id varchar(128) NOT NULL,
    asset varchar(32) NOT NULL,
    kind varchar(16) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE UNIQUE INDEX IF NOT EXISTS ledger_accounts_owner_asset_kind_key
    ON ledger_accounts (owner_type, owner_id, asset, kind);

CREATE TABLE IF NOT EXISTS ledger_journals (
    id bigserial PRIMARY KEY,
    kind varchar(16) NOT NULL,
    reference_type varchar(128) NOT NULL,
    reference_id varchar(128) NOT NULL,
    idempotency_key varchar(128) NOT NULL,
    request_hash char(64) NOT NULL,
    posted_at timestamptz NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    posting_txid bigint NOT NULL DEFAULT 0,
    posting_started_at timestamptz NOT NULL DEFAULT '1970-01-01 00:00:00+00',
    created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE UNIQUE INDEX IF NOT EXISTS ledger_journals_idempotency_key_key
    ON ledger_journals (idempotency_key);
CREATE INDEX IF NOT EXISTS ledger_journals_reference_idx
    ON ledger_journals (reference_type, reference_id);
CREATE INDEX IF NOT EXISTS idx_ledger_journals_posted_at
    ON ledger_journals (posted_at);

CREATE TABLE IF NOT EXISTS ledger_lines (
    id bigserial PRIMARY KEY,
    journal_id bigint NOT NULL,
    account_id bigint NOT NULL,
    asset varchar(32) NOT NULL,
    amount numeric(38, 18) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX IF NOT EXISTS idx_ledger_lines_journal_id ON ledger_lines (journal_id);
CREATE INDEX IF NOT EXISTS idx_ledger_lines_account_id ON ledger_lines (account_id);

COMMENT ON TABLE ledger_lines IS
    'Signed amounts: positive is a debit, negative a credit. Balances are derived, never stored.';

-- Postgres-only guarantees the GORM models cannot express. Idempotent: applied by
-- AutoMigrate in dev/test and repeated verbatim in the checksummed migration.
-- Trigger functions pin search_path and qualify every reference with the schema they were
-- installed in, so a session that can SET search_path cannot shadow a table or txid_current().

ALTER TABLE ledger_journals ADD COLUMN IF NOT EXISTS posting_txid bigint NOT NULL DEFAULT 0;
ALTER TABLE ledger_journals ADD COLUMN IF NOT EXISTS posting_started_at timestamptz NOT NULL DEFAULT '1970-01-01 00:00:00+00';
ALTER TABLE ledger_accounts ALTER COLUMN asset TYPE varchar(32);
ALTER TABLE ledger_lines ALTER COLUMN asset TYPE varchar(32);

DO $$
BEGIN
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

-- Ownership moves to a NOLOGIN role so the application role cannot disable or replace the
-- guarantees above. The migrator keeps SELECT, INSERT and becomes a member of ledger_owner (SET and
-- INHERIT) so later ledger migrations can SET ROLE to it; narrow other roles with cmd/migrate
-- --ledger-app-role. Without the right to SET ROLE ledger_owner this is skipped with a NOTICE and
-- must be done by hand (docs/OPERATIONS.md, Ledger roles).
DO $$
DECLARE
    is_super boolean;
    can_create_role boolean;
BEGIN
    SELECT rolsuper, rolcreaterole INTO is_super, can_create_role FROM pg_roles WHERE rolname = current_user;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ledger_owner') THEN
        IF COALESCE(is_super, false) OR COALESCE(can_create_role, false) THEN
            CREATE ROLE ledger_owner NOLOGIN;
        ELSE
            RAISE NOTICE 'ledger: % cannot create roles; ledger tables stay owned by it. See docs/OPERATIONS.md, Ledger roles.', current_user;
            RETURN;
        END IF;
    END IF;
    IF NOT COALESCE(is_super, false) AND NOT pg_has_role(current_user, 'ledger_owner', 'SET') THEN
        BEGIN
            EXECUTE format('GRANT ledger_owner TO %I WITH SET TRUE, INHERIT TRUE', current_user);
        EXCEPTION WHEN insufficient_privilege THEN
            RAISE NOTICE 'ledger: % is not an admin of ledger_owner; grant membership by hand. See docs/OPERATIONS.md, Ledger roles.', current_user;
        END;
    END IF;
    IF NOT COALESCE(is_super, false) AND NOT pg_has_role(current_user, 'ledger_owner', 'SET') THEN
        RAISE NOTICE 'ledger: % cannot SET ROLE ledger_owner; ledger tables stay owned by it. See docs/OPERATIONS.md, Ledger roles.', current_user;
        RETURN;
    END IF;
    -- A new owner must hold CREATE on the schema; the schema owner (database owner) can grant it.
    IF NOT has_schema_privilege('ledger_owner', current_schema(), 'CREATE') THEN
        BEGIN
            EXECUTE format('GRANT USAGE, CREATE ON SCHEMA %I TO ledger_owner', current_schema());
        EXCEPTION WHEN OTHERS THEN
            NULL;
        END;
    END IF;
    IF NOT has_schema_privilege('ledger_owner', current_schema(), 'CREATE') THEN
        RAISE NOTICE 'ledger: ledger_owner has no CREATE on schema %; ledger tables stay owned by %. See docs/OPERATIONS.md, Ledger roles.', current_schema(), current_user;
        RETURN;
    END IF;
    ALTER TABLE ledger_accounts OWNER TO ledger_owner;
    ALTER TABLE ledger_journals OWNER TO ledger_owner;
    ALTER TABLE ledger_lines OWNER TO ledger_owner;
    ALTER FUNCTION ledger_reject_mutation() OWNER TO ledger_owner;
    ALTER FUNCTION ledger_check_journal_balance() OWNER TO ledger_owner;
    ALTER FUNCTION ledger_check_journal_has_lines() OWNER TO ledger_owner;
    ALTER FUNCTION ledger_stamp_journal_txid() OWNER TO ledger_owner;
    ALTER FUNCTION ledger_check_line_same_transaction() OWNER TO ledger_owner;
    EXECUTE format('GRANT SELECT, INSERT ON ledger_accounts, ledger_journals, ledger_lines TO %I', current_user);
    EXECUTE format('GRANT USAGE, SELECT ON SEQUENCE ledger_accounts_id_seq, ledger_journals_id_seq, ledger_lines_id_seq TO %I', current_user);
END;
$$;
