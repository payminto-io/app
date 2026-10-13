-- Tempo memo credits: one row per TransferWithMemo log the Tempo watcher decided on, unique on
-- (tx_hash, log_index) so a log is credited at most once. Outcome is credited or the rejection reason.
-- Additive; idempotent for databases shaped by AutoMigrate.

CREATE TABLE IF NOT EXISTS tempo_memo_credits (
    id bigserial PRIMARY KEY,
    created_at timestamptz,
    tx_hash varchar(66) NOT NULL,
    log_index bigint NOT NULL,
    block_number bigint,
    token varchar(42) NOT NULL,
    from_address varchar(42),
    to_address varchar(42) NOT NULL,
    amount numeric(38,18) NOT NULL,
    memo varchar(66) NOT NULL,
    outcome varchar(32) NOT NULL,
    payment_request_id bigint,
    deposit_id bigint
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tempo_memo_credit_log ON tempo_memo_credits (tx_hash, log_index);
CREATE INDEX IF NOT EXISTS idx_tempo_memo_credits_memo ON tempo_memo_credits (memo);
