-- Live and test isolation: the database stamp, and an environment on api_keys, ledger_accounts
-- and ledger_journals. Additive and idempotent; existing rows are test money until adopt-live.
-- Design: .scratch/payments-v1/issues/13-environments.md

CREATE TABLE IF NOT EXISTS gateway_environment (
    id smallint PRIMARY KEY,
    environment varchar(8) NOT NULL,
    stamped_at timestamptz NOT NULL,
    adopted_from varchar(8),
    adopted_at timestamptz,
    CONSTRAINT gateway_environment_singleton CHECK (id = 1),
    CONSTRAINT gateway_environment_environment_check CHECK (environment IN ('test', 'live'))
);

ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS environment varchar(8) NOT NULL DEFAULT 'test';
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS prefix varchar(16) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_api_keys_environment ON api_keys (environment);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'api_keys'::regclass AND conname = 'api_keys_environment_check') THEN
        ALTER TABLE api_keys ADD CONSTRAINT api_keys_environment_check
            CHECK (environment IN ('test', 'live')) NOT VALID;
        ALTER TABLE api_keys VALIDATE CONSTRAINT api_keys_environment_check;
    END IF;
END;
$$;

-- Ledger: the environment joins the account uniqueness key through a second index. The
-- four-column index stays so binaries that still infer it keep posting during a rolling
-- deploy; a follow-up migration drops it once no such binary runs.
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
END;
$$;

-- The only sanctioned rewrite of ledger rows: relabelling a database that is being adopted as live
-- (cmd/migrate adopt-live). SECURITY DEFINER so the table owner's right to pause the append-only
-- triggers is exercised here and nowhere else; it refuses once any live row exists.
CREATE OR REPLACE FUNCTION ledger_adopt_environment(target text)
RETURNS TABLE (accounts bigint, journals bigint)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public
AS $$
DECLARE
    relabelled_accounts bigint;
    relabelled_journals bigint;
BEGIN
    IF target <> 'live' THEN
        RAISE EXCEPTION 'ledger: adoption only moves test rows to live, not to %', target;
    END IF;
    IF EXISTS (SELECT 1 FROM ledger_accounts WHERE environment = 'live')
       OR EXISTS (SELECT 1 FROM ledger_journals WHERE environment = 'live') THEN
        RAISE EXCEPTION 'ledger: live rows already exist; this database was adopted or served live before';
    END IF;
    ALTER TABLE ledger_accounts DISABLE TRIGGER ledger_accounts_append_only;
    ALTER TABLE ledger_journals DISABLE TRIGGER ledger_journals_append_only;
    UPDATE ledger_accounts SET environment = 'live' WHERE environment = 'test';
    GET DIAGNOSTICS relabelled_accounts = ROW_COUNT;
    UPDATE ledger_journals SET environment = 'live' WHERE environment = 'test';
    GET DIAGNOSTICS relabelled_journals = ROW_COUNT;
    ALTER TABLE ledger_accounts ENABLE TRIGGER ledger_accounts_append_only;
    ALTER TABLE ledger_journals ENABLE TRIGGER ledger_journals_append_only;
    RETURN QUERY SELECT relabelled_accounts, relabelled_journals;
END;
$$;
REVOKE ALL ON FUNCTION ledger_adopt_environment(text) FROM PUBLIC;

-- The relabel function belongs to the ledger owner (migration 2026100701) and only the migrator may call it.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ledger_owner')
       AND (SELECT rolsuper OR rolcreaterole FROM pg_roles WHERE rolname = current_user) THEN
        ALTER FUNCTION ledger_adopt_environment(text) OWNER TO ledger_owner;
    END IF;
    EXECUTE format('GRANT EXECUTE ON FUNCTION ledger_adopt_environment(text) TO %I', current_user);
END;
$$;
