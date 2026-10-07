-- Versioned fee rules and per-attempt fee snapshots (ticket 02, internal/fees/README.md).
-- Idempotent: fees.Migrate runs it in dev/test and migration 2026100702 repeats it verbatim.

-- btree_gist backs the no-overlap exclusion constraint; trusted since PG13 (docs/OPERATIONS.md, Fee rules).
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS fee_rules (
    id bigserial PRIMARY KEY,
    lineage_id uuid NOT NULL,
    version integer NOT NULL CHECK (version >= 1),
    method varchar(16) NOT NULL CHECK (method IN ('card', 'upi', 'bank', 'crypto')),
    connector varchar(64),
    card_type varchar(16) CHECK (card_type IN ('credit', 'debit', 'prepaid')),
    region varchar(8),
    currency varchar(16) NOT NULL,
    minor_units smallint NOT NULL CHECK (minor_units BETWEEN 0 AND 18),
    percent numeric(9, 6) NOT NULL DEFAULT 0 CHECK (percent >= 0 AND percent <= 100),
    flat numeric(38, 18) NOT NULL DEFAULT 0 CHECK (flat >= 0),
    slabs jsonb CHECK (slabs IS NULL OR jsonb_typeof(slabs) = 'array'),
    min_fee numeric(38, 18) CHECK (min_fee >= 0),
    max_fee numeric(38, 18) CHECK (max_fee >= 0),
    taxable boolean NOT NULL DEFAULT false,
    tax_percent numeric(9, 6) NOT NULL DEFAULT 0,
    fee_bearer varchar(16) NOT NULL CHECK (fee_bearer IN ('merchant', 'customer')),
    effective_from timestamptz NOT NULL,
    effective_to timestamptz,
    created_by varchar(128) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT fee_rules_lineage_version_key UNIQUE (lineage_id, version),
    CONSTRAINT fee_rules_id_version_key UNIQUE (id, version),
    CONSTRAINT fee_rules_min_max_check CHECK (min_fee IS NULL OR max_fee IS NULL OR min_fee <= max_fee),
    CONSTRAINT fee_rules_tax_check CHECK (
        (taxable AND tax_percent > 0 AND tax_percent <= 100) OR (NOT taxable AND tax_percent = 0)),
    -- Equal bounds are allowed: superseding a not-yet-effective version leaves it an empty window.
    CONSTRAINT fee_rules_window_check CHECK (effective_to IS NULL OR effective_to >= effective_from),
    CONSTRAINT fee_rules_scope_check CHECK (
        (card_type IS NULL OR (connector IS NOT NULL AND method = 'card'))
        AND (region IS NULL OR card_type IS NOT NULL)),
    -- No two rules for one scope are active at the same instant; empty windows never conflict.
    CONSTRAINT fee_rules_no_overlap EXCLUDE USING gist (
        method WITH =, currency WITH =,
        (coalesce(connector, '')) WITH =, (coalesce(card_type, '')) WITH =, (coalesce(region, '')) WITH =,
        tstzrange(effective_from, effective_to, '[)') WITH &&)
);
CREATE INDEX IF NOT EXISTS fee_rules_resolve_idx ON fee_rules (method, currency, effective_from);

-- Rules are never mutated. Only fee_rules_close_lineage (SECURITY DEFINER, owned by ledger_owner where the
-- migrator can hand it over) may change effective_to, only earlier, and never into the past (docs/OPERATIONS.md).
DO $fees$
DECLARE
    fn text := format('%I.fee_rules_close_lineage(uuid,timestamptz)', current_schema());
BEGIN
    EXECUTE format($f$
CREATE OR REPLACE FUNCTION %1$I.fee_rules_close_lineage(p_lineage uuid, p_at timestamptz) RETURNS integer
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, pg_temp AS $body$
DECLARE
    n integer;
BEGIN
    IF p_at IS NULL OR p_at < pg_catalog.transaction_timestamp() THEN
        RAISE EXCEPTION 'fee_rules: cannot close a version in the past (%%)', p_at USING ERRCODE = 'restrict_violation';
    END IF;
    UPDATE %1$I.fee_rules SET effective_to = GREATEST(effective_from, p_at)
     WHERE lineage_id = p_lineage AND (effective_to IS NULL OR effective_to > p_at);
    GET DIAGNOSTICS n = ROW_COUNT;
    RETURN n;
END
$body$;
$f$, current_schema());

    EXECUTE format($f$
CREATE OR REPLACE FUNCTION %1$I.fee_rules_append_only() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog, pg_temp AS $body$
DECLARE
    closer name;
BEGIN
    IF TG_OP IN ('DELETE', 'TRUNCATE') THEN
        RAISE EXCEPTION 'fee_rules is append-only: %% rejected', TG_OP USING ERRCODE = 'restrict_violation';
    END IF;
    IF (pg_catalog.to_jsonb(NEW) - 'effective_to') IS DISTINCT FROM (pg_catalog.to_jsonb(OLD) - 'effective_to') THEN
        RAISE EXCEPTION 'fee_rules is append-only: only effective_to may change' USING ERRCODE = 'restrict_violation';
    END IF;
    IF NEW.effective_to IS DISTINCT FROM OLD.effective_to THEN
        IF NEW.effective_to IS NULL OR (OLD.effective_to IS NOT NULL AND NEW.effective_to > OLD.effective_to) THEN
            RAISE EXCEPTION 'fee_rules is append-only: effective_to may only move earlier' USING ERRCODE = 'restrict_violation';
        END IF;
        IF NEW.effective_to < pg_catalog.transaction_timestamp() THEN
            RAISE EXCEPTION 'fee_rules is append-only: a version cannot be closed in the past' USING ERRCODE = 'restrict_violation';
        END IF;
        SELECT pg_catalog.pg_get_userbyid(p.proowner) INTO closer
          FROM pg_catalog.pg_proc p WHERE p.oid = pg_catalog.to_regprocedure(%2$L);
        IF closer IS NULL OR current_user <> closer THEN
            RAISE EXCEPTION 'fee_rules is append-only: only fee_rules_close_lineage may close a version' USING ERRCODE = 'restrict_violation';
        END IF;
    END IF;
    RETURN NEW;
END
$body$;
$f$, current_schema(), fn);
END
$fees$;
CREATE OR REPLACE TRIGGER fee_rules_append_only
    BEFORE UPDATE OR DELETE ON fee_rules FOR EACH ROW EXECUTE FUNCTION fee_rules_append_only();
CREATE OR REPLACE TRIGGER fee_rules_no_truncate
    BEFORE TRUNCATE ON fee_rules FOR EACH STATEMENT EXECUTE FUNCTION fee_rules_append_only();

-- One row per payment attempt: the rule version it is priced under, its merchant and ledger asset.
CREATE TABLE IF NOT EXISTS fee_snapshots (
    id bigserial PRIMARY KEY,
    attempt_id varchar(64) NOT NULL,
    payment_request_id bigint NOT NULL REFERENCES payment_requests (id),
    merchant_id bigint NOT NULL,
    fee_rule_id bigint NOT NULL,
    fee_rule_version integer NOT NULL,
    method varchar(16) NOT NULL,
    connector varchar(64) NOT NULL DEFAULT '',
    card_type varchar(16) NOT NULL DEFAULT '',
    region varchar(8) NOT NULL DEFAULT '',
    chain varchar(20) NOT NULL DEFAULT '',
    currency varchar(16) NOT NULL,
    ledger_asset varchar(32) NOT NULL,
    fee_bearer varchar(16) NOT NULL CHECK (fee_bearer IN ('merchant', 'customer')),
    resolved_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT fee_snapshots_attempt_key UNIQUE (attempt_id),
    CONSTRAINT fee_snapshots_rule_fkey FOREIGN KEY (fee_rule_id, fee_rule_version) REFERENCES fee_rules (id, version)
);
CREATE INDEX IF NOT EXISTS fee_snapshots_payment_idx ON fee_snapshots (payment_request_id);

CREATE OR REPLACE FUNCTION fee_snapshots_append_only() RETURNS trigger
LANGUAGE plpgsql SET search_path = pg_catalog, pg_temp AS $$
BEGIN
    RAISE EXCEPTION '% is append-only: % rejected', TG_TABLE_NAME, TG_OP USING ERRCODE = 'restrict_violation';
END
$$;
CREATE OR REPLACE TRIGGER fee_snapshots_append_only
    BEFORE UPDATE OR DELETE ON fee_snapshots FOR EACH ROW EXECUTE FUNCTION fee_snapshots_append_only();
CREATE OR REPLACE TRIGGER fee_snapshots_no_truncate
    BEFORE TRUNCATE ON fee_snapshots FOR EACH STATEMENT EXECUTE FUNCTION fee_snapshots_append_only();

-- At most one posted fee per payment request and environment; a second successful attempt is refused (README).
CREATE TABLE IF NOT EXISTS fee_postings (
    id bigserial PRIMARY KEY,
    payment_request_id bigint NOT NULL REFERENCES payment_requests (id),
    environment varchar(16) NOT NULL,
    attempt_id varchar(64) NOT NULL REFERENCES fee_snapshots (attempt_id),
    captured numeric(38, 18) NOT NULL CHECK (captured > 0),
    base numeric(38, 18) NOT NULL CHECK (base > 0),
    fee numeric(38, 18) NOT NULL CHECK (fee >= 0),
    tax numeric(38, 18) NOT NULL CHECK (tax >= 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT fee_postings_attempt_key UNIQUE (attempt_id),
    CONSTRAINT fee_postings_payment_key UNIQUE (payment_request_id, environment)
);
CREATE OR REPLACE TRIGGER fee_postings_append_only
    BEFORE UPDATE OR DELETE ON fee_postings FOR EACH ROW EXECUTE FUNCTION fee_snapshots_append_only();
CREATE OR REPLACE TRIGGER fee_postings_no_truncate
    BEFORE TRUNCATE ON fee_postings FOR EACH STATEMENT EXECUTE FUNCTION fee_snapshots_append_only();

-- Hand the close function to ledger_owner so the application role can never satisfy the trigger's owner check.
DO $own$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ledger_owner')
        AND ((SELECT rolsuper FROM pg_roles WHERE rolname = current_user) OR pg_has_role(current_user, 'ledger_owner', 'SET'))
        AND has_schema_privilege('ledger_owner', current_schema(), 'CREATE') THEN
        EXECUTE format('GRANT SELECT, UPDATE ON %I.fee_rules TO ledger_owner', current_schema());
        EXECUTE format('ALTER FUNCTION %I.fee_rules_close_lineage(uuid, timestamptz) OWNER TO ledger_owner', current_schema());
    ELSE
        RAISE NOTICE 'fees: fee_rules_close_lineage stays owned by %; see docs/OPERATIONS.md, Fee rules', current_user;
    END IF;
END
$own$;

-- Legacy columns on payment_requests: additive, nullable, set by PostFee from the latest successful attempt.
ALTER TABLE payment_requests ADD COLUMN IF NOT EXISTS fee_rule_id bigint;
ALTER TABLE payment_requests ADD COLUMN IF NOT EXISTS fee_rule_version integer;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'payment_requests'::regclass AND conname = 'payment_requests_fee_rule_fkey') THEN
        ALTER TABLE payment_requests ADD CONSTRAINT payment_requests_fee_rule_fkey
            FOREIGN KEY (fee_rule_id, fee_rule_version) REFERENCES fee_rules (id, version);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'payment_requests'::regclass AND conname = 'payment_requests_fee_rule_pair_check') THEN
        ALTER TABLE payment_requests ADD CONSTRAINT payment_requests_fee_rule_pair_check
            CHECK ((fee_rule_id IS NULL) = (fee_rule_version IS NULL));
    END IF;
END
$$;
