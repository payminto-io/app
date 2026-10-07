-- Live and test isolation: api_keys and ledger_accounts carry an environment.
-- Additive and idempotent; existing rows are test money. Design: .scratch/payments-v1/issues/13-environments.md

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
            CHECK (environment IN ('test', 'live'));
    END IF;
END;
$$;

-- Ledger accounts: the environment joins the uniqueness key. The old four-column
-- index is replaced; the append-only triggers do not fire on DDL.
ALTER TABLE ledger_accounts ADD COLUMN IF NOT EXISTS environment varchar(8) NOT NULL DEFAULT 'test';
DROP INDEX IF EXISTS ledger_accounts_owner_asset_kind_key;
CREATE UNIQUE INDEX IF NOT EXISTS ledger_accounts_env_owner_asset_kind_key
    ON ledger_accounts (environment, owner_type, owner_id, asset, kind);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'ledger_accounts'::regclass AND conname = 'ledger_accounts_environment_check') THEN
        ALTER TABLE ledger_accounts ADD CONSTRAINT ledger_accounts_environment_check
            CHECK (environment IN ('test', 'live'));
    END IF;
END;
$$;
