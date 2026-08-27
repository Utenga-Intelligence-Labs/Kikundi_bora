# MASTER PROMPT — Kikundibora Event-Sourced Double-Entry Ledger Core

Paste this whole document as the task/system prompt for the coding agent (Claude Code, etc.). It is self-contained.

---

## 1. Role & Context

You are implementing the **accounting core** for Kikundibora, a savings-group (SACCOS) management platform. This core must be a standalone, well-isolated module: `ledger/` (Go package or microservice — your call, but keep it decoupled from the rest of Kikundibora's domain code so it could theoretically be extracted as its own service later).

Think **Beancount / Ledger-CLI semantics**, implemented as an **event-sourced** system rather than a flat-file/batch one. Every financial fact enters the system as an immutable event. Current balances, statements, and reports are **derived (projected) state**, never the source of truth. The event log is the source of truth and must be replayable from empty state to reproduce any account balance at any point in time.

This is the highest-integrity part of the whole product — group members' real money is tracked here. Correctness, auditability, and immutability matter more than developer convenience anywhere else in the codebase.

---

## 2. Non-negotiable invariants

1. **Events are append-only and immutable.** No `UPDATE` or `DELETE` is ever issued against the event store. Corrections happen via new compensating events (e.g. `TransactionReversed`, `TransactionAdjusted`), never by mutating history.
2. **Every financial transaction is double-entry.** For any transaction event, the sum of debit entries MUST equal the sum of credit entries, in the same currency. This is enforced at the domain layer *before* the event is appended — reject, don't record, invalid transactions.
3. **Balances are always derived, never stored as the source of truth.** A balance is `fold(events, initialState=0)`. Cached/materialized balances (for read speed) are allowed but must be regenerable from the event stream at any time and must be tagged with the event offset/version they were computed at.
4. **Full replay must be deterministic.** Replaying the entire event log from scratch, in order, must always produce identical account balances and ledger state, byte for byte.
5. **Every event is attributable and timestamped.** `actor_id` (who), `occurred_at` (business time), `recorded_at` (system time) are mandatory on every event. These can differ (e.g. backdated correction entries) — never conflate them.
6. **No silent money creation or destruction.** A system-wide invariant check must exist: sum of all account balances in a closed ledger (or per currency) reconciles to zero (assets = liabilities + equity, or however Kikundibora's chart of accounts is structured for group savings).

---

## 3. Domain model

### 3.1 Chart of accounts (Kikundibora-specific)
Model group savings accounting, minimum viable chart of accounts:
- `member_savings:{member_id}` (liability — group owes member)
- `member_loans_receivable:{member_id}` (asset — member owes group)
- `group_cash` / `group_bank:{account_id}` (asset)
- `interest_income` (income)
- `fines_income` (income)
- `loan_loss_provision` (contra-asset, optional v1)
- `group_equity` / `retained_earnings` (equity)

Accounts belong to a `group_id` (the savings group/SACCOS). Ledger is scoped per group — no cross-group commingling of events or balances.

### 3.2 Core entities
- **Account**: `id`, `group_id`, `type` (asset/liability/income/expense/equity), `name`, `currency`, `owner_member_id` (nullable), `opened_at`, `closed_at` (nullable).
- **Transaction**: a logical business event (e.g. "member deposit", "loan disbursement", "interest posting") composed of ≥2 **Entries**.
- **Entry**: one leg of a transaction — `account_id`, `direction` (debit/credit), `amount`, `currency`.
- **Event**: the append-only fact. A `TransactionRecorded` event embeds its entries. Other event types: `TransactionReversed`, `TransactionAdjusted`, `AccountOpened`, `AccountClosed`.

### 3.3 Event envelope (every event has this shape)
```
event_id        UUID, primary key
group_id        UUID, partition key for the stream
stream_id       UUID  (e.g. the transaction id, or account id for account-lifecycle events)
sequence_no     BIGINT, monotonic per stream_id (optimistic concurrency)
event_type      TEXT   ("TransactionRecorded", "TransactionReversed", ...)
event_version   INT    (schema version of the payload, for future migrations)
payload         JSONB  (the actual event data — entries, amounts, memo, etc.)
actor_id        UUID   (member/admin who caused this)
occurred_at     TIMESTAMPTZ (business-effective time)
recorded_at     TIMESTAMPTZ (system append time, server-generated, never client-supplied)
causation_id    UUID nullable (event that caused this one, e.g. reversal -> original)
correlation_id  UUID nullable (groups events from one user-initiated command)
```

---

## 4. Architecture

```
Command (HTTP/CLI/API) 
   -> Command Handler (validates business rules, e.g. debits==credits, sufficient balance for withdrawal)
   -> Domain logic produces one or more Events
   -> Event Store.Append(events) [optimistic concurrency on sequence_no]
   -> Event Store publishes to Projector(s)
   -> Projectors update read models (account_balances, member_statements, trial_balance)
   -> Query side reads ONLY from projections, never replays live for normal reads
```

**Event Store**: PostgreSQL table (see §3.3), append-only, with a unique constraint on `(stream_id, sequence_no)` for optimistic concurrency control. Do NOT use an ORM's generic "update" path anywhere near this table — write raw, explicit, append-only SQL/queries for it.

**Projections**: separate tables, rebuildable at any time by replaying events from sequence 0. Minimum projections:
- `account_balances` (account_id, balance, currency, as_of_sequence)
- `trial_balance` (group_id snapshot, for auditing debits=credits globally)
- `member_statement_lines` (denormalized, for fast statement rendering)

Provide a `ledger replay --group-id=X --rebuild-projections` operation (CLI subcommand or admin endpoint) that:
1. Truncates the projection tables for that group (or all groups).
2. Streams all events for the group in `sequence_no` order.
3. Re-applies each to the projector.
4. Asserts the final trial balance nets to zero before committing.

**Snapshots** (optional but recommended for perf once event count grows): periodic materialized balance snapshots at a known `sequence_no`, so replay for a single account can start from the last snapshot instead of sequence 0. Snapshots are a cache, not truth — must be provably reconstructible.

---

## 5. Tech stack constraints (match existing Kikundibora stack)

- **Language**: Go (matches Kikundibora's planned backend). Package it as `internal/ledger/` or a standalone module `ledger/` with a clean public API (`ledger.RecordTransaction(...)`, `ledger.GetBalance(...)`, `ledger.Replay(...)`).
- **Storage**: PostgreSQL (existing planned DB). Use `pgx` or `database/sql`, explicit SQL — avoid a heavy ORM for the event store table specifically; a lighter query builder is fine for projections.
- **Money type**: never use `float64` for amounts. Use integer minor units (cents) or a decimal type (`shopspring/decimal`). All arithmetic must be exact.
- **API layer**: expose via whatever Kikundibora's existing API framework is (REST/gRPC) — ledger package itself should have zero HTTP dependencies, so it's testable and portable.
- **Concurrency**: use DB-level optimistic concurrency (unique constraint + retry-on-conflict) for appending events to a stream, not application-level locks.

---

## 6. Required operations (public API of the ledger package)

1. `OpenAccount(groupID, accountType, name, currency, ownerMemberID) (AccountID, error)`
2. `RecordTransaction(groupID, actorID, occurredAt, memo string, entries []Entry) (TransactionID, error)`
   - Validates: ≥2 entries, all same currency (or explicit multi-currency handling if in scope), sum(debits) == sum(credits), all accounts exist and belong to groupID, no closed accounts.
3. `ReverseTransaction(transactionID, actorID, reason string) (ReversalEventID, error)`
   - Emits a new event with mirrored entries; never deletes/mutates the original.
4. `GetBalance(accountID, asOf *time.Time) (Money, error)` — reads from projection; if `asOf` given and no snapshot covers it, replay from the last snapshot ≤ asOf.
5. `GetLedgerEntries(accountID, from, to) ([]Entry, error)` — for statements.
6. `GetTrialBalance(groupID, asOf *time.Time) (TrialBalance, error)` — must net to zero; return an error/flag if it doesn't (this should never happen if invariants hold — treat a nonzero trial balance as a critical bug alarm, not a display quirk).
7. `Replay(groupID *string) error` — full or per-group rebuild of all projections from the event log.

---

## 7. Testing requirements — do not skip these

1. **Unit tests** for the double-entry validator: reject unbalanced entries, reject wrong currency mixes, reject entries against nonexistent/closed accounts.
2. **Property-based test**: generate random sequences of valid transactions; assert that after every single event, `sum(all account balances weighted by normal balance side)` reconciles to zero.
3. **Replay determinism test**: record N random transactions, snapshot the resulting balances, wipe projections, replay from event 0, assert identical balances.
4. **Concurrency test**: fire concurrent `RecordTransaction` calls against overlapping accounts; assert no lost updates and that sequence numbers stay strictly monotonic per stream with no gaps or duplicates.
5. **Reversal test**: reverse a transaction, assert original event is untouched in the store, and net balance effect of original+reversal is zero.
6. **Golden-file / audit test**: given a fixed event log fixture, assert the exported trial balance and member statement match a checked-in expected output — this is your regression guard for the audit trail.

---

## 8. Explicit non-goals for v1 (call these out so scope doesn't creep)

- Multi-currency FX conversion logic (assume single base currency, e.g. TZS, unless told otherwise).
- Full accrual-basis complex interest schedules — start with simple posted-interest events; accrual engines can be a separate module that *emits* ledger events, not something baked into the ledger core.
- A general-ledger UI — this task is the core + API only, not the frontend.

---

## 9. Deliverables checklist for the agent

- [ ] `ledger` Go package with event store, domain types, and public API listed in §6
- [ ] PostgreSQL migration(s) for `events`, `account_balances`, `trial_balance`, `member_statement_lines` tables
- [ ] Projector(s) that consume events and update read models, idempotently (safe to re-run)
- [ ] `Replay` / rebuild-projections operation, exposed as CLI command and/or admin API endpoint
- [ ] Full test suite per §7, runnable via `go test ./ledger/...`
- [ ] A short `ledger/README.md` documenting the event schema, chart of accounts, and how to run a replay
- [ ] No `float64` anywhere in money-handling code (grep-check this before calling it done)

---

## 10. Working instructions for the agent

- Work incrementally: (1) event envelope + append-only store + optimistic concurrency, (2) domain validation + `RecordTransaction`, (3) projectors + `GetBalance`/`GetTrialBalance`, (4) `Replay`, (5) reversal flow, (6) full test suite from §7. Get each stage green before moving to the next.
- Ask before making an irreversible schema choice (e.g. final chart-of-accounts naming) if it's not already fixed elsewhere in the Kikundibora codebase — check for existing account/member models first and align with them rather than inventing parallel ones.
- Every function touching money must have a matching test. If you can't write a property test for an invariant in §2, treat that as a signal the design needs rethinking, not that the test can be skipped.
