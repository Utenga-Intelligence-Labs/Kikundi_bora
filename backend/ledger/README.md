# Kikundibora Ledger Core

Event-sourced double-entry accounting core for group savings (SACCOS).
Beancount/Ledger-CLI semantics, implemented as an immutable event log whose
balances, statements and reports are **derived projections** that can be
rebuilt at any time. This package is deliberately decoupled from the rest of
Kikundibora (no GORM, no HTTP, no `models` imports) so it can be extracted as
its own service later.

## Invariants (non-negotiable)

1. **Append-only events** — the store issues no UPDATE/DELETE; a DB trigger
   (`trg_ledger_events_immutable`) rejects them at engine level.
2. **Double entry** — every transaction validates `sum(debits) == sum(credits)`
   in one currency *before* any append. Rejected, never recorded.
3. **Balances are derived** — cached values are tagged with the event offset
   (`as_of_global_seq`) they were computed at, and must be reproducible by
   replay.
4. **Deterministic replay** — re-applying `ledger_events` in `global_seq`
   order from empty projections always yields identical state.
5. **Attribution & time separation** — every event carries `actor_id`,
   business-effective `occurred_at` and server-generated `recorded_at`.
6. **No silent money creation/destruction** — weighted account balances must
   net to zero per group; the replay operation refuses to commit otherwise.

## Money

`int64` minor units only — **never float64**. v1 is single-currency **TZS**
(`ledger.CurrencyTZS`); mixed-currency transactions are rejected.

## Chart of accounts (Swahili naming)

| Name | Kind | Type |
|---|---|---|
| `akiba_ya_mwanachama:{member_ref}` | member savings | liability |
| `dai_la_mkopo:{member_ref}` | loan receivable | asset |
| `hazina_taslimu` | cash on hand | asset |
| `hazina_benki:{bank_ref}` | bank account | asset |
| `mapato_ya_riba` | interest income | income |
| `mapato_ya_faini` | fines income | income |
| `hifadhi_ya_hasara_ya_mkopo` | loan-loss provision | expense/contra-asset |
| `mtaji_wa_kikundi` | group capital | equity |
| `faida_tulizo` | retained earnings | equity |

Account names are unique **per group** (`PRIMARY KEY (group_id, account_name)`);
account ids are deterministic UUIDv5 hashes of `(group_id, name)`.

Balance convention: stored `balance_minor` is positive on the **normal side**
(assets/expenses grow on debit; liabilities/income/equity on credit).

## Event envelope (`ledger_events`)

```
event_id UUID PK · global_seq bigserial UNIQUE (replay order)
group_id → ledger_groups · stream_id (txn/account id) · sequence_no (per-stream,
gapless, UNIQUE(stream_id, sequence_no) = optimistic concurrency)
event_type ("AccountOpened" | "AccountClosed" | "TransactionRecorded" |
"TransactionReversed" | "TransactionAdjusted") · event_version · payload JSONB
actor_id · occurred_at · recorded_at (server-generated) · causation_id · correlation_id
```

Event streams: account lifecycle events stream on the account id; each
transaction/reversal is its own stream keyed by its transaction id.

## Public API (`backend/ledger`)

```go
lg, _ := ledger.New(pool)                                  // pool: *pgxpool.Pool
gid, err := ledger.CreateGroup(ctx, pool, name, "TZS")
accountID, err := lg.OpenAccount(ctx, gid, actor, occurredAt,
    ledger.MemberSavingsName("42"), ledger.Liability, "42", 0 /*OCC expected*/)
txID, err := lg.RecordTransaction(ctx, gid, actor, occurredAt, memo, entries)
eventID, err := lg.ReverseTransaction(ctx, gid, actor, txID, occurredAt, reason)
money, err := lg.GetBalance(ctx, gid, "hazina_taslimu", nil /* or &asOf */)
lines, err := lg.GetLedgerEntries(ctx, gid, "akiba_ya_mwanachama:42", from, to)
tb, err     := lg.GetTrialBalance(ctx, gid, nil)           // errors if net != 0
err         = lg.CheckGroupReconciliation(ctx, gid)
err         = lg.RebuildProjections(ctx, &gid /* or nil = all */)
```

Errors are typed sentinel values (`ErrUnbalancedTransaction`, …) in `errors.go`.

## Projections

Rebuildable read models written in the same transaction as their events:

- `ledger_accounts` — account registry (open/closed state, type)
- `ledger_account_balances` — cached balances tagged `as_of_global_seq`
- `ledger_statement_lines` — denormalized statement feed per account
- `ledger_trial_balance` — debit/credit checkpoints (net MUST be 0)

## Running a replay

CLI (preferred for ops):

```bash
cd backend
./server -replay-ledger=group -serve=false   # this deployment's group
./server -replay-ledger=all   -serve=false   # every group
# add -migrate first if the schema isn't applied yet
```

HTTP (treasurer/admin token): `POST /api/v1/admin/ledger/replay?scope=all`.

A failed replay (unbalanced result) aborts without committing anything.

## Other endpoints

All under `/api/v1/admin/ledger` (role: treasurer or admin):

```
POST /accounts                       {name,type,owner_member_ref}
POST /transactions                   {memo,occurred_at?,entries:[{account_name,direction,amount_minor}]}
POST /transactions/:id/reverse       {reason}
GET  /balance?account=…&as_of=RFC3339
GET  /statement?account=…&from=…&to=…
GET  /trial-balance                  (417 Expectation Failed if unbalanced)
```

## Testing

```bash
cd backend
LEDGER_TEST_DSN="postgres://user:pass@host:port/kikundi_db" \
LEDGER_TEST_RESET=1 go test ./ledger/ -count=1 -v
```

- `LEDGER_TEST_DSN` enables DB-backed tests (skipped when unset).
- `LEDGER_TEST_RESET=1` truncates all ledger tables once at suite start
  (safe against shared dev DBs — tests use uniquely-named groups anyway).
- `LEDGER_UPDATE_GOLDEN=1 go test ./ledger/ -run TestGoldenAuditExport`
  regenerates the checked-in audit golden file — review diffs like code.

Coverage includes: double-entry validator units, property-based reconciliation
(2000 random multi-leg transactions keep cumulative weighted balance zero),
concurrent appends (exactly-one-winner per sequence), event immutability,
reversal flow (original byte-identical, net-zero effect), full-replay
determinism, and the golden audit export.

## Out of scope for v1

FX conversion (single TZS base), accrual interest schedules (an external
module should *emit* ledger events), general-ledger UI, snapshots (balances
are cheap to derive until history grows; `as_of_global_seq` tagging makes
adding them non-breaking).

## Float64 check

```bash
grep -rn "float64" backend/ledger/    # matches comments only, intentionally
```
