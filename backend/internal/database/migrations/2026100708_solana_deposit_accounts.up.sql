-- Solana deposit accounts: owner keypair address and associated token account per deposit address,
-- the watch window and cadence, and the watcher's signature cursors; sweep attempts (every
-- signature a sweep ever broadcast) and the deposits each sweep claimed.
-- Additive; idempotent for databases shaped by AutoMigrate.
-- Design: .scratch/payments-v1/issues/09-solana-usdc.md

CREATE TABLE IF NOT EXISTS solana_deposit_accounts (
    id bigserial PRIMARY KEY,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz,
    deposit_address_id bigint NOT NULL,
    payment_request_id bigint,
    blockchain_currency_id bigint NOT NULL,
    owner_address varchar(64) NOT NULL,
    token_account varchar(64) NOT NULL,
    mint varchar(64) NOT NULL,
    token_program varchar(64) NOT NULL,
    decimals smallint NOT NULL,
    status varchar(20) NOT NULL DEFAULT 'watching',
    token_account_cursor varchar(128),
    owner_cursor varchar(128),
    last_polled_at timestamptz,
    last_seen_slot bigint DEFAULT 0,
    payment_expires_at timestamptz,
    watch_until timestamptz,
    last_balance_raw varchar(40),
    token_poll_after timestamptz,
    owner_poll_after timestamptz,
    held_signature varchar(128),
    held_attempts integer DEFAULT 0,
    unresolved_signatures text,
    balance_hold_attempts integer DEFAULT 0,
    unresolved_attempts integer DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_solana_deposit_accounts_deposit_address_id ON solana_deposit_accounts (deposit_address_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_solana_deposit_accounts_token_account ON solana_deposit_accounts (token_account);
CREATE INDEX IF NOT EXISTS idx_solana_deposit_accounts_payment_request_id ON solana_deposit_accounts (payment_request_id);
CREATE INDEX IF NOT EXISTS idx_solana_deposit_accounts_blockchain_currency_id ON solana_deposit_accounts (blockchain_currency_id);
CREATE INDEX IF NOT EXISTS idx_solana_deposit_accounts_owner_address ON solana_deposit_accounts (owner_address);
CREATE INDEX IF NOT EXISTS idx_solana_deposit_accounts_status ON solana_deposit_accounts (status);
CREATE INDEX IF NOT EXISTS idx_solana_deposit_accounts_watch_until ON solana_deposit_accounts (watch_until);
CREATE INDEX IF NOT EXISTS idx_solana_deposit_accounts_token_poll_after ON solana_deposit_accounts (token_poll_after);
CREATE INDEX IF NOT EXISTS idx_solana_deposit_accounts_deleted_at ON solana_deposit_accounts (deleted_at);

CREATE TABLE IF NOT EXISTS solana_sweep_attempts (
    id bigserial PRIMARY KEY,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz,
    sweep_id bigint NOT NULL,
    attempt_no integer NOT NULL DEFAULT 1,
    signature varchar(128) NOT NULL,
    blockhash varchar(64) NOT NULL,
    last_valid_block_height bigint NOT NULL,
    status varchar(20) NOT NULL DEFAULT 'sent'
);
CREATE INDEX IF NOT EXISTS idx_solana_sweep_attempts_sweep_id ON solana_sweep_attempts (sweep_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_solana_sweep_attempts_signature ON solana_sweep_attempts (signature);
CREATE UNIQUE INDEX IF NOT EXISTS idx_solana_sweep_attempts_sweep_attempt ON solana_sweep_attempts (sweep_id, attempt_no);
CREATE INDEX IF NOT EXISTS idx_solana_sweep_attempts_deleted_at ON solana_sweep_attempts (deleted_at);

CREATE TABLE IF NOT EXISTS solana_sweep_deposits (
    id bigserial PRIMARY KEY,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz,
    sweep_id bigint NOT NULL,
    deposit_id bigint NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_solana_sweep_deposits_sweep_deposit ON solana_sweep_deposits (sweep_id, deposit_id);
CREATE INDEX IF NOT EXISTS idx_solana_sweep_deposits_deposit_id ON solana_sweep_deposits (deposit_id);
CREATE INDEX IF NOT EXISTS idx_solana_sweep_deposits_deleted_at ON solana_sweep_deposits (deleted_at);

-- One sweep in flight per token account, enforced by the database.
CREATE TABLE IF NOT EXISTS solana_sweep_locks (
    id bigserial PRIMARY KEY,
    created_at timestamptz,
    token_account varchar(64) NOT NULL,
    sweep_id bigint NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_solana_sweep_locks_token_account ON solana_sweep_locks (token_account);
CREATE INDEX IF NOT EXISTS idx_solana_sweep_locks_sweep_id ON solana_sweep_locks (sweep_id);
