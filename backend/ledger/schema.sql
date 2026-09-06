-- Kikundibora ledger schema (v1)
-- Single currency: TZS.

CREATE TABLE IF NOT EXISTS ledger_groups (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name          text NOT NULL UNIQUE,
    base_currency char(3) NOT NULL DEFAULT 'TZS',
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- THE event log. Source of truth. Append-only; guarded below against UPDATE/DELETE.
CREATE TABLE IF NOT EXISTS ledger_events (
    event_id       uuid PRIMARY KEY,
    global_seq     bigserial NOT NULL UNIQUE,        -- total order for deterministic replay
    group_id       uuid NOT NULL REFERENCES ledger_groups(id),
    stream_id      uuid NOT NULL,                    -- transaction id / account id
    sequence_no    bigint NOT NULL CHECK (sequence_no >= 1), -- per-stream, gapless
    event_type     text NOT NULL,
    event_version  int  NOT NULL DEFAULT 1,
    payload        jsonb NOT NULL,
    actor_id       text NOT NULL,                    -- who caused it
    occurred_at    timestamptz NOT NULL,             -- business time
    recorded_at    timestamptz NOT NULL DEFAULT now(), -- system time, server-generated
    causation_id   uuid,                             -- event that caused this one
    correlation_id uuid,                             -- user-initiated command grouping
    UNIQUE (stream_id, sequence_no)                  -- optimistic concurrency control
);

CREATE INDEX IF NOT EXISTS idx_ledger_events_group_global ON ledger_events (group_id, global_seq);
CREATE INDEX IF NOT EXISTS idx_ledger_events_stream ON ledger_events (stream_id, sequence_no);
CREATE INDEX IF NOT EXISTS idx_ledger_events_type ON ledger_events (group_id, event_type);

-- Immutability guard (spec §2 invariant 1): reject any UPDATE or DELETE.
CREATE OR REPLACE FUNCTION ledger_events_immutable_guard() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'ledger_events is append-only: % not permitted', TG_OP;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_ledger_events_immutable ON ledger_events;
CREATE TRIGGER trg_ledger_events_immutable
    BEFORE UPDATE OR DELETE ON ledger_events
    FOR EACH ROW EXECUTE FUNCTION ledger_events_immutable_guard();

-- ---------------------------------------------------------------------------
-- Projections (derived state — rebuildable via Replay, never source of truth)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS ledger_accounts (
    group_id         uuid NOT NULL REFERENCES ledger_groups(id),
    account_name     text NOT NULL,               -- e.g. 'akiba_ya_mwanachama:42'
    type             text NOT NULL,
    currency         char(3) NOT NULL DEFAULT 'TZS',
    owner_member_ref text,                           -- member_no/id reference when member-owned
    opened_at_seq    bigint NOT NULL,                -- global_seq of AccountOpened
    closed_at_seq    bigint,                         -- global_seq of AccountClosed, NULL if open
    opened_at        timestamptz NOT NULL,
    closed_at        timestamptz,
    PRIMARY KEY (group_id, account_name)
);

CREATE TABLE IF NOT EXISTS ledger_account_balances (
    group_id         uuid NOT NULL,
    account_name     text NOT NULL,
    balance_minor    bigint NOT NULL,                 -- normalized (+ = normal side), cached
    currency         char(3) NOT NULL DEFAULT 'TZS',
    as_of_global_seq bigint NOT NULL,                 -- event offset this cache value was computed at
    PRIMARY KEY (group_id, account_name),
    FOREIGN KEY (group_id, account_name) REFERENCES ledger_accounts (group_id, account_name)
);

CREATE TABLE IF NOT EXISTS ledger_statement_lines (
    line_id          bigserial PRIMARY KEY,
    group_id         uuid NOT NULL,
    account_name     text NOT NULL,
    transaction_id   uuid NOT NULL,
    event_type       text NOT NULL,                  -- TransactionRecorded | TransactionReversed | ...
    direction        text NOT NULL,                  -- debit | credit as posted to THIS account
    amount_minor     bigint NOT NULL,
    currency         char(3) NOT NULL DEFAULT 'TZS',
    occurred_at      timestamptz NOT NULL,           -- business-effective time (statement ordering)
    memo             text,
    actor_id         text NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ledger_stmt_acct_time ON ledger_statement_lines (account_name, occurred_at, line_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_ledger_stmt_line ON ledger_statement_lines (transaction_id, account_name, direction);

CREATE TABLE IF NOT EXISTS ledger_trial_balance (
    group_id       uuid NOT NULL,
    snapshot_at    timestamptz NOT NULL DEFAULT now(),
    total_debit_minor  bigint NOT NULL,
    total_credit_minor bigint NOT NULL,
    net_minor          bigint NOT NULL,              -- MUST be 0; nonzero => critical bug alarm
    as_of_global_seq   bigint NOT NULL,
    PRIMARY KEY (group_id, as_of_global_seq)
);
