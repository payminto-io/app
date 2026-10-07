-- Solana deposit accounts: owner keypair address and associated token account per deposit address,
-- with the watcher's signature cursors. Additive; idempotent for databases shaped by AutoMigrate.
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
    last_seen_slot bigint DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_solana_deposit_accounts_deposit_address_id ON solana_deposit_accounts (deposit_address_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_solana_deposit_accounts_token_account ON solana_deposit_accounts (token_account);
CREATE INDEX IF NOT EXISTS idx_solana_deposit_accounts_payment_request_id ON solana_deposit_accounts (payment_request_id);
CREATE INDEX IF NOT EXISTS idx_solana_deposit_accounts_blockchain_currency_id ON solana_deposit_accounts (blockchain_currency_id);
CREATE INDEX IF NOT EXISTS idx_solana_deposit_accounts_owner_address ON solana_deposit_accounts (owner_address);
CREATE INDEX IF NOT EXISTS idx_solana_deposit_accounts_status ON solana_deposit_accounts (status);
CREATE INDEX IF NOT EXISTS idx_solana_deposit_accounts_deleted_at ON solana_deposit_accounts (deleted_at);
