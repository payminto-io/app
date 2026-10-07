-- CRE attestation records (ticket 21, internal/cre/README.md).
-- Idempotent: cre.Migrate runs it in dev/test and migration 2026100707 repeats it verbatim.

CREATE TABLE IF NOT EXISTS cre_attestations (
    id uuid PRIMARY KEY,
    kind varchar(32) NOT NULL CHECK (kind IN ('solvency', 'deposit_finality', 'conversion_reference')),
    subject_type varchar(32) NOT NULL,
    subject_id varchar(128) NOT NULL,
    payload_hash bytea NOT NULL,
    payload bytea NOT NULL,
    chain varchar(64) NOT NULL,
    tx_hash bytea,
    block_number bigint NOT NULL DEFAULT 0,
    workflow_id bytea NOT NULL,
    workflow_owner bytea NOT NULL,
    report_id bytea NOT NULL,
    observed_at timestamptz NOT NULL,
    recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    status varchar(16) NOT NULL CHECK (status IN ('pending', 'attested', 'failed', 'stale', 'mismatch', 'ignored')),
    provider varchar(16) NOT NULL CHECK (provider IN ('mock', 'chainlink')),
    simulated boolean NOT NULL DEFAULT false,
    reason text NOT NULL DEFAULT '',
    item jsonb NOT NULL,
    execution_id varchar(128) NOT NULL DEFAULT '',
    item_index smallint NOT NULL DEFAULT 0,
    -- One row per item per report; a replayed report is refused before it gets here.
    CONSTRAINT cre_attestations_payload_item_key UNIQUE (payload_hash, item_index)
);
CREATE INDEX IF NOT EXISTS cre_attestations_kind_subject_idx ON cre_attestations (kind, subject_id);
CREATE INDEX IF NOT EXISTS cre_attestations_status_recorded_idx ON cre_attestations (status, recorded_at);
CREATE INDEX IF NOT EXISTS cre_attestations_kind_recorded_idx ON cre_attestations (kind, recorded_at DESC);

-- What the gateway asked each workflow about; an attestation for anything else is refused (SPEC section 6).
CREATE TABLE IF NOT EXISTS cre_subjects (
    kind varchar(32) NOT NULL CHECK (kind IN ('solvency', 'deposit_finality', 'conversion_reference')),
    subject_key bytea NOT NULL,
    subject_id varchar(128) NOT NULL,
    facts jsonb NOT NULL,
    asked_at timestamptz NOT NULL,
    PRIMARY KEY (kind, subject_key)
);
CREATE INDEX IF NOT EXISTS cre_subjects_kind_asked_idx ON cre_subjects (kind, asked_at DESC);

-- One row per trigger the gateway sent (or the mock ran).
CREATE TABLE IF NOT EXISTS cre_runs (
    id bigserial PRIMARY KEY,
    kind varchar(32) NOT NULL CHECK (kind IN ('solvency', 'deposit_finality', 'conversion_reference')),
    provider varchar(16) NOT NULL,
    execution_id varchar(128) NOT NULL DEFAULT '',
    status varchar(16) NOT NULL CHECK (status IN ('accepted', 'failed')),
    detail text NOT NULL DEFAULT '',
    started_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX IF NOT EXISTS cre_runs_kind_started_idx ON cre_runs (kind, started_at DESC);

-- Poll position per workflow.
CREATE TABLE IF NOT EXISTS cre_cursors (
    kind varchar(32) PRIMARY KEY CHECK (kind IN ('solvency', 'deposit_finality', 'conversion_reference')),
    block_number bigint NOT NULL DEFAULT 0,
    seq bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
