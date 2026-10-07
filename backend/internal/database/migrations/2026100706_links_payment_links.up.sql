-- Payment links, their line items and questions, and each use with its answers (ticket 03, internal/links/README.md).
-- Idempotent: links.Migrate runs it in dev/test and migration 2026100706 repeats it verbatim.

CREATE TABLE IF NOT EXISTS payment_links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id bigint NOT NULL REFERENCES members (id) ON DELETE RESTRICT,
    external_platform_id bigint NOT NULL REFERENCES external_platforms (id) ON DELETE RESTRICT,
    environment varchar(8) NOT NULL DEFAULT 'test' CHECK (environment IN ('live', 'test')),
    title varchar(200) NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    amount_mode varchar(16) NOT NULL DEFAULT 'fixed' CHECK (amount_mode IN ('fixed', 'customer', 'line_items')),
    amount numeric(38, 18) CHECK (amount >= 0),
    amount_min numeric(38, 18) CHECK (amount_min > 0),
    amount_max numeric(38, 18) CHECK (amount_max > 0),
    currency varchar(16) NOT NULL DEFAULT '',
    reference_id varchar(100) NOT NULL DEFAULT '',
    metadata jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(metadata) = 'object'),
    category varchar(64) NOT NULL DEFAULT '',
    customer_field_policy jsonb NOT NULL CHECK (jsonb_typeof(customer_field_policy) = 'object'),
    billing_required boolean NOT NULL DEFAULT false,
    shipping_required boolean NOT NULL DEFAULT false,
    multi_use boolean NOT NULL DEFAULT false,
    use_limit integer CHECK (use_limit >= 1),
    methods jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(methods) = 'array'),
    capture_mode varchar(16) NOT NULL DEFAULT 'automatic' CHECK (capture_mode IN ('automatic', 'manual')),
    three_ds_policy varchar(16) NOT NULL DEFAULT 'inherit' CHECK (three_ds_policy IN ('inherit', 'force')),
    chain_tolerance_bps integer NOT NULL DEFAULT 0 CHECK (chain_tolerance_bps BETWEEN 0 AND 1000),
    quote_expiry_seconds integer NOT NULL DEFAULT 900 CHECK (quote_expiry_seconds BETWEEN 60 AND 86400),
    fee_bearer varchar(16) NOT NULL DEFAULT 'merchant' CHECK (fee_bearer IN ('merchant', 'customer')),
    success_mode varchar(16) NOT NULL DEFAULT 'message' CHECK (success_mode IN ('message', 'redirect')),
    success_url text NOT NULL DEFAULT '',
    success_message text NOT NULL DEFAULT '',
    receipt_email boolean NOT NULL DEFAULT false,
    receipt_note text NOT NULL DEFAULT '',
    webhook_id bigint REFERENCES webhooks (id) ON DELETE RESTRICT,
    failure_retry boolean NOT NULL DEFAULT true,
    failure_message text NOT NULL DEFAULT '',
    settlement_override jsonb CHECK (settlement_override IS NULL OR jsonb_typeof(settlement_override) = 'object'),
    hold_in_asset boolean NOT NULL DEFAULT false,
    settlement_timing varchar(16) NOT NULL DEFAULT 'cycle' CHECK (settlement_timing IN ('cycle', 'immediate')),
    expires_at timestamptz,
    expires_after_payments integer CHECK (expires_after_payments >= 1),
    status varchar(16) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'paused', 'archived')),
    logo_url text NOT NULL DEFAULT '',
    accent_color varchar(7) NOT NULL DEFAULT '' CHECK (accent_color = '' OR accent_color ~ '^#[0-9A-Fa-f]{6}$'),
    language varchar(16) NOT NULL DEFAULT 'en',
    short_code varchar(32) CHECK (short_code ~ '^[A-Za-z0-9]{8,32}$'),
    uses_count integer NOT NULL DEFAULT 0 CHECK (uses_count >= 0),
    revision integer NOT NULL DEFAULT 1 CHECK (revision >= 1),
    published_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT payment_links_short_code_key UNIQUE (short_code),
    CONSTRAINT payment_links_published_have_code CHECK (status = 'draft' OR status = 'archived' OR short_code IS NOT NULL),
    CONSTRAINT payment_links_amount_range CHECK (amount_min IS NULL OR amount_max IS NULL OR amount_min <= amount_max),
    -- The last line of defence for the use limit; the service also checks it under a row lock.
    CONSTRAINT payment_links_single_use CHECK (multi_use OR uses_count <= 1),
    CONSTRAINT payment_links_use_limit CHECK (use_limit IS NULL OR uses_count <= use_limit),
    CONSTRAINT payment_links_payment_cap CHECK (expires_after_payments IS NULL OR uses_count <= expires_after_payments)
);

CREATE INDEX IF NOT EXISTS payment_links_platform_created_idx ON payment_links (external_platform_id, created_at DESC);

CREATE TABLE IF NOT EXISTS payment_link_line_items (
    id bigserial PRIMARY KEY,
    link_id uuid NOT NULL REFERENCES payment_links (id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position >= 0),
    name varchar(200) NOT NULL,
    quantity integer NOT NULL CHECK (quantity BETWEEN 1 AND 100000),
    unit_price numeric(38, 18) NOT NULL CHECK (unit_price >= 0),
    tax_rate numeric(9, 6) NOT NULL DEFAULT 0 CHECK (tax_rate BETWEEN 0 AND 100),
    CONSTRAINT payment_link_line_items_position_key UNIQUE (link_id, position)
);

CREATE TABLE IF NOT EXISTS payment_link_questions (
    id bigserial PRIMARY KEY,
    link_id uuid NOT NULL REFERENCES payment_links (id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position >= 0),
    key varchar(64) NOT NULL CHECK (key ~ '^[a-z0-9_]{1,64}$'),
    label varchar(200) NOT NULL,
    type varchar(16) NOT NULL CHECK (type IN ('text', 'select', 'checkbox')),
    options jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(options) = 'array'),
    required boolean NOT NULL DEFAULT false,
    per_order boolean NOT NULL DEFAULT true,
    CONSTRAINT payment_link_questions_key_key UNIQUE (link_id, key),
    CONSTRAINT payment_link_questions_position_key UNIQUE (link_id, position)
);

-- One use of a link: pending while the payment is being created, created once it exists.
CREATE TABLE IF NOT EXISTS payment_link_payments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id uuid NOT NULL REFERENCES payment_links (id) ON DELETE RESTRICT,
    idempotency_key varchar(128) NOT NULL CHECK (length(idempotency_key) >= 1),
    request_hash char(64) NOT NULL,
    status varchar(16) NOT NULL CHECK (status IN ('pending', 'created')),
    environment varchar(8) NOT NULL CHECK (environment IN ('live', 'test')),
    method varchar(16) NOT NULL CHECK (method IN ('card', 'upi', 'bank', 'crypto')),
    chain varchar(20) NOT NULL DEFAULT '',
    asset varchar(16) NOT NULL DEFAULT '',
    connector varchar(64) NOT NULL DEFAULT '',
    amount numeric(38, 18) NOT NULL CHECK (amount > 0),
    currency varchar(16) NOT NULL,
    fee numeric(38, 18) CHECK (fee >= 0),
    tax numeric(38, 18) CHECK (tax >= 0),
    customer_total numeric(38, 18) NOT NULL CHECK (customer_total >= amount),
    fee_bearer varchar(16) NOT NULL CHECK (fee_bearer IN ('merchant', 'customer')),
    fee_rule_id bigint,
    fee_rule_version integer,
    customer_name varchar(200) NOT NULL DEFAULT '',
    customer_email varchar(254) NOT NULL DEFAULT '',
    customer_phone varchar(16) NOT NULL DEFAULT '',
    billing_address jsonb,
    shipping_address jsonb,
    payment_reference varchar(128),
    processor_response jsonb,
    -- sha256 of the payer's IP, for the per-client open-payment cap.
    client_key varchar(64) NOT NULL DEFAULT '',
    -- The lease of a pending use; the resolver looks the payment up only after it.
    reserved_until timestamptz NOT NULL,
    -- When this use stops counting as an open payment: the lease, then the payment's expiry.
    open_until timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT payment_link_payments_key_key UNIQUE (link_id, idempotency_key),
    CONSTRAINT payment_link_payments_fee_rule_fkey FOREIGN KEY (fee_rule_id, fee_rule_version)
        REFERENCES fee_rules (id, version) ON DELETE RESTRICT,
    CONSTRAINT payment_link_payments_rule_pair CHECK ((fee_rule_id IS NULL) = (fee_rule_version IS NULL)),
    CONSTRAINT payment_link_payments_created_have_ref CHECK (status = 'pending' OR payment_reference IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS payment_link_payments_link_open_idx ON payment_link_payments (link_id, open_until);
CREATE INDEX IF NOT EXISTS payment_link_payments_pending_idx ON payment_link_payments (reserved_until) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS payment_link_answers (
    id bigserial PRIMARY KEY,
    link_payment_id uuid NOT NULL REFERENCES payment_link_payments (id) ON DELETE CASCADE,
    link_id uuid NOT NULL REFERENCES payment_links (id) ON DELETE RESTRICT,
    question_key varchar(64) NOT NULL,
    question_label varchar(200) NOT NULL,
    value text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT payment_link_answers_question_key UNIQUE (link_payment_id, question_key)
);
