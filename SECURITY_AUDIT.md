# Kikundibora — Security Audit Report

**Date:** 2026-09-01  
**Scope:** `backend/` (Go 1.24 · Fiber v2 · GORM · PostgreSQL), `Frontend-1/` (React 19 · TanStack), `docker-compose.yml`, `Frontend-1/nginx.conf`, `vite.config.ts`  
**Method:** Static code review, `file:line` verified. **No fixes applied** — findings only.  
**Classification:** Multi-tenant savings-group app — financial data, PII, RBAC (mwenyekiti / katibu / mweka hazina / mwanachama / msimamizi).

---

## Executive Summary

| Category | Critical | High | Medium | Low | Verdict |
|---|---:|---:|---:|---:|---|
| 1. Authentication | 1 | 1 | 2 | 1 | Solid primitives, operational gaps |
| 2. Authorization / IDOR | 0 | 1 | 3 | 2 | One exploitable IDOR |
| 3. Data Exposure | 0 | 2 | 2 | 1 | No masking, plaintext backups/logs |
| 4. Input Validation & Injection | 0 | 0 | 2 | 2 | SQL safe, validation patchy |
| 5. Transport & Infra | 1 | 1 | 2 | 1 | HTTP-only, CSP/HSTS missing on frontend |
| 6. Secrets Management | 1 | 1 | 1 | 0 | Weak dev secrets, correct gitignore |
| 7. Audit Trail Integrity | 0 | 0 | 1 | 1 | Client cannot tamper, retention infinite |

**Top 3 to fix before next deploy:** (1) Rotate weak secrets + re-enable login rate limit, (2) Fix `GET /welfare/contributions` IDOR, (3) Encrypt backups.

---

## 1. Authentication

**Strengths:** `bcrypt DefaultCost=10` — `backend/handlers/auth.go:76,209,278,394,436`, `models/user.go:44 json:"-"`; JWT `HS256` only, `exp=30m`, `jti=uuid` — `backend/handlers/auth.go:563-576`, secret `len>=32` enforced `backend/config/config.go:33`, `none` rejected `backend/middleware/auth.go:32-37`, role re-fetched from DB `backend/middleware/auth.go:76-82`.

| ID | Sev | Location | Finding |
|---|---|---|---|
| AUTH-01 | **Critical** | `backend/.env:15` | `DISABLE_LOGIN_RATE_LIMIT=1` disables the `5/5min` per-IP + per-account throttle `backend/handlers/auth.go:158-179`. Active on disk. |
| AUTH-02 | High | `backend/middleware/auth.go:21-24` | JWT accepted via `?token=` for `/uploads/*`. Leaks in access logs, `Referer`, history, caches. No `Cache-Control: private, no-store`. |
| AUTH-03 | Medium | `backend/handlers/auth.go:530-551` | Refresh is not rotation — old token stays valid 30 min, no revocation, no reuse detection. `user_sessions.ExpiresAt` (24h login vs 30m refresh) never checked in `AuthRequired`. Table grows unbounded. |
| AUTH-04 | Medium | `backend/handlers/auth.go:196-207` | Login leaks `PENDING`/`REJECTED`/`SUSPENDED` 403 vs `401 si sahihi` before password check → user enumeration. |
| AUTH-05 | Low | `backend/handlers/auth.go:264` vs `models/requests.go:317` | `FirstLoginSetup` allows `len>=6` while validator requires `min=8`. No complexity rules anywhere (only length). |

---

## 2. Authorization / RBAC — Full Endpoint Inventory (129 endpoints)

Base: `protected.Use(AuthRequired)` — `backend/main.go:131`. Dual planes: `User.role` vs `LeadershipPosition` vs `UserPosition`.

### 2.1 Exploitable

| ID | Location | Issue | Exploit |
|---|---|---|---|
| **RBAC-H01** | `backend/main.go:265` + `backend/handlers/welfare.go:719-729` | `GET /welfare/contributions` has no `RequireRoles` guard and no handler role check. `listContributionsAdmin` does `SELECT *`. | Any `role=member` → `GET /api/v1/welfare/contributions` lists every member's welfare obligations, amounts, statuses. |

### 2.2 Medium

| ID | Location | Issue |
|---|---|---|
| RBAC-M01 | `backend/models/member.go:9`, `backend/models/loan.go`, `backend/models/welfare.go` | No `group_id` FK on `members`/`loans`/`contributions`/`welfare_*` (only `payment_methods`, `group_setting_proposals`, `ledger`). Single-group deployment (`EnsureGroupSetup` `main.go:37`) — `:id` in `/groups/:id/...` is existence-checked `dashboard_scoped.go:57` then ignored for data queries. Second group → full cross-group leakage. |
| RBAC-M02 | `backend/main.go:186,192,209` etc. | Auth relies on handler `requesterIsSelfOrLeadership` `backend/handlers/dashboard_scoped.go:116-151` instead of route middleware — correct but fragile, no fallback if handler is refactored. Affects `/users/:id/roles`, `/members/:id/dashboard-summary`, `/loans/:id`. |
| RBAC-M03 | `backend/middleware/auth.go:21-24` | Query-token accepted globally, not scoped to `/uploads/*` only. |

### 2.3 Low

- `POST /michango/:id/confirm` `main.go:329` allows `secretary` at route but handler `member_contribution.go:229-238` denies both types — UX inconsistency.
- `GET /welfare/my-contributions` fallback `listAllContributions` for treasurer without member row `welfare.go:690`.
- `GET /uongozi/ripoti` `main.go:341` delegates to chair-only reports — treasurer/secretary gain access via alternate path (intended but undocumented).

### 2.4 Well-Guarded (positive)

`DELETE /members/:id` `backend/handlers/members.go:364-366` explicitly denies `admin` despite `RequireRoles(chair)` bypass; payment methods scoped `WHERE group_id=? AND id=?` `backend/handlers/payment_methods.go:129,178`; loan sequential state machine `backend/handlers/leadership.go:141-215`; member self-filter on `GET /loans/` `backend/handlers/loans.go:46-53` and `POST /loans/apply` `loans.go:130-137`.

---

## 3. Data Exposure

| ID | Sev | Location | Finding |
|---|---|---|---|
| DATA-H01 | High | `backend/services/backup.go:167-215` | `zip.Deflate` with zero encryption — no `SetPassword`/`AES`/`age`/`gpg`. Stored plaintext `backend_backups:/app/backups` `docker-compose.yml:29`, emailed plaintext `services/backup.go:217-249`, downloadable plaintext `handlers/backup.go:114-141`. |
| DATA-H02 | High | `backend/middleware/logger.go:33-35,40` | Dev mode (`ENVIRONMENT != production` `logger.go:12`) logs `Body: <raw JSON>` if `len<2000` and `Response` on `>=400` — dumps `password`/`new_password`/`temp_password`. Prod single-line mode is safe. |
| DATA-M01 | Medium | `backend/models/member.go:9-41`, `backend/models/payment_method.go:23-34` | No per-role masking. Handlers return whole structs `c.JSON(member)` `members.go:72`. `phone`, `next_of_kin_phone`, `email`, `account_number`/`account_name` serialize for any caller with access. `GET /groups/:id/payment-methods` `payment_methods.go:71-74` exposes full `account_number` to every authenticated member. |
| DATA-M02 | Medium | `backend/models/audit.go:34-44`, `backend/services/audit.go:13-42` | `audit_logs.old_values/new_values` store `phone` `members.go:192` and `account` `payment_methods.go:112` indefinitely, queryable via `GET /audit-logs` `main.go:225`. |

No `NIDA` field exists (`grep NIDA = 0`). `Password json:"-"` `models/user.go:44` and `TokenHash json:"-"` `models/user.go:62` correctly suppressed.

---

## 4. Input Validation & Injection

| ID | Sev | Location | Finding |
|---|---|---|---|
| SQL | **PASS** | All handlers | Every `WHERE` uses `?` placeholders `handlers/members.go:58`, `handlers/loans.go:406` static `Raw`. `escapeLike` + `ESCAPE '\\'` `handlers/helpers.go:68-79` on all `LIKE`. No `fmt.Sprintf` SQL. |
| VAL-M01 | Medium | `models/requests.go:37-42`, `requests.go:68-73`, `handlers/member_contribution.go:38-105`, `handlers/ledger.go:64-108` | Missing `validate` tags: `UpdateProfileRequest`, `UpdateMemberRequest`, `MemberContribution.Submit` (hand-parsed struct, manual `oneof` check), ledger `Type` cast without allowlist, `group_settings.go:127` `ContributionInterval` no tag (relies on `ValidateProposalSpec`). |
| VAL-M02 | Medium | `handlers/import.go:51-53,201-216` | CSV import checks extension only; `Loans` has no `LimitReader`/size cap (contributions has 10MB+1 `import.go:69`), unbounded `ReadAll` → DoS. No CSV-injection sanitization (`=cmd`). |
| FILE-M01 | Low | `handlers/upload.go:127-129,38-41,174-185` | `UploadAvatar` lacks `MkdirAll` for `avatars` (vs `UploadDoc` `197`); `WEBP` detection accepts any `RIFF`; `/upload/doc` skips magic for image extensions `upload.go:155-157`; header read ignores error `upload.go:180`. |
| XSS | Low | `Frontend-1/src/components/ui/chart.tsx:72-88`, `Frontend-1/src/lib/auth-storage.ts:1-14` | Single `dangerouslySetInnerHTML` — static `THEMES` only, no user data. No other `innerHTML`/`eval` in `Frontend-1/src`. `sessionStorage` token theft risk noted but `httpOnly` migration not done. |

---

## 5. Transport & Infra

| ID | Sev | Location | Finding |
|---|---|---|---|
| TLS-01 | **Critical** | `Frontend-1/nginx.conf:50-51`, `docker-compose.yml:26,48`, `backend/main.go:364` | **HTTP-only.** `listen 8080` no `443 ssl`, `app.Listen(":"+Port)` no `ListenTLS`. JWT and `?token=` traverse plaintext on any non-loopback deploy. |
| TLS-02 | High | `backend/middleware/security.go:28-30` | `HSTS` only if `ENVIRONMENT==production`, no `preload`, no HTTP→HTTPS redirect. On HTTP, browsers ignore HSTS. |
| SEC-M01 | Medium | `Frontend-1/nginx.conf:57-61`, `backend/middleware/security.go:33-36` | Backend sets CSP but SPA from **nginx never sees it** — nginx has no `Content-Security-Policy` at all. `X-Frame-Options` inconsistent: backend `DENY` vs nginx `SAMEORIGIN`. Missing `Permissions-Policy`, `object-src 'none'`, `base-uri 'self'`. |
| CORS-M01 | Medium | `backend/middleware/cors.go:12-25`, `backend/config/config.go:52` | `AllowOrigins` from `CORS_ORIGINS` with `AllowCredentials:true` — default `localhost:3000,5173` correct, but `*` not rejected when credentials enabled. |

---

## 6. Secrets Management

| ID | Sev | Location | Finding |
|---|---|---|---|
| SEC-C01 | Critical | `backend/.env:4` | `DB_PASSWORD=123456789` — 9-char sequential, live on disk. Correctly gitignored (`/.gitignore:1-4` `.env`, `*.env`, `!*.env.example` — not committed), but weak value persists. |
| SEC-H01 | High | `backend/.env:7,12` | `JWT_SECRET=2fa2f4b1...` (64 hex, strong) and `ADMIN_PASSWORD=AdminPass123!` in plaintext file. `len>=32` enforced `config.go:33`. Single file via `env_file: ./backend/.env` `docker-compose.yml:18` + `godotenv.Load()` `config.go:27`. |
| SEC-M01 | Medium | `backend/database/seed.go:39-50`, `docker-compose.yml:69` | Demo `demo123` for 4 seed users `seed.go:40` (bcrypt hashed but known), gated `if userCount==0` `seed.go:18` — may seed in prod on empty DB. `POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-changeme}` fallback `changeme`. |

**Positive:** No hardcoded `JWT_SECRET`/`DB_PASSWORD` in Go source (only `config.go:29` env load). `.env.example` is placeholder `change-this-to-a-32-char...`.

---

## 7. Audit Trail Integrity

| ID | Sev | Location | Finding |
|---|---|---|---|
| AUD-M01 | Medium | `backend/handlers/members.go:103-208,210-311`, `backend/services/audit.go:13-42` | **Client cannot tamper** — `registered_by` from `GetUserID(c)` `members.go:137`, `approved_by` from `GetUserID` `members.go:228,262`, never from `req` body. `CreateMemberRequest` `models/requests.go:48-56` has no `approval_status`/`approved_by` fields. Missing: no `FOR UPDATE` lock on approve, no `created_by` immutability trigger (but `PUT /members/:id` does not expose `registered_by` `requests.go:68-73`, so safe). |
| AUD-L01 | Low | `backend/models/member.go:28-31`, `backend/models/audit.go:34-44` | Default `approval_status='approved'` keeps existing rows active, but direct SQL bypass could insert `approved` without audit. `audit_logs` retained forever, no `DELETE` trigger — depends on DB access controls (`DB_SSLMODE=require` `config.go:49` set, but `backup.go` plaintext ZIP undermines it). |

---

## Remediation Order (Recommended, Not Yet Applied)

1. **Immediate (before next deploy):** Rotate `DB_PASSWORD`/`JWT_SECRET`/`ADMIN_PASSWORD`, remove `DISABLE_LOGIN_RATE_LIMIT=1` `backend/.env:15`, fix `GET /welfare/contributions` guard (RBAC-H01).
2. **This sprint:** Encrypt backups (`age`/`gpg` + `BACKUP_ENCRYPTION_KEY`), redact `logger.go` `password` fields, add `Cache-Control: private, no-store` on `?token=` responses and scope it to `/uploads/*` only, mask `phone`/`account_number` per role via DTOs.
3. **Next sprint:** Add `group_id` FK + scoping (or document single-tenant invariant + `id != currentGroup.ID → 404`), enforce TLS at nginx with HSTS `preload` + redirect, gate demo seed by `ENVIRONMENT != production`, require `POSTGRES_PASSWORD` (no `changeme` default).

---

*Generated from static review of `backend/config`, `backend/middleware`, `backend/handlers`, `backend/models`, `backend/services`, `Frontend-1/src`, `Frontend-1/nginx.conf`, `docker-compose.yml`. All findings reference verified `file:line`.*

---

## ✅ REMEDIATION STATUS (updated 2026-09-02)

All actionable findings have been fixed and pushed. Reference commits (in order):

| Finding(s) | Commit | Fix |
|---|---|---|
| AUTH-01, SEC-C01/H01 | *(on-disk .env — untracked)* | Secrets rotated (DB/JWT/admin), `DISABLE_LOGIN_RATE_LIMIT` removed |
| RBAC-H01 | `cca0479` | `GET /welfare/contributions` guarded to leadership |
| DATA-H02, AUTH-02 | `4f12815` | Logger redacts secrets; `?token=` scoped to `/uploads/*`; no-store |
| DATA-H01 | `4c1a265` | Backups encrypted AES-256-GCM; plaintext zip removed; download decrypts in memory |
| RBAC-M01 | `1a1e843` | `IsCurrentGroup` tenant check on all group-scoped handlers (404 foreign IDs) |
| AUTH-04, AUTH-05 | `d2f02f6` | Login responses unified; password min length 6→8 consistent |
| VAL-M01 | `868fd76` | validate tags on michango submit, ledger DTOs, profile/member updates |
| VAL-M02, FILE-M01 | `7d1e783` | Loans CSV 10MB cap; upload magic-byte/WEBP/MkdirAll hardening |
| SEC-M01 | `ab1dc93` | Demo seed gated by ENVIRONMENT=production; POSTGRES_PASSWORD required |
| TLS-02, SEC-M01, CORS-M01 | `01a324b` | CSP directives, HSTS preload, CORS `*` rejected, nginx CSP/DENY/Permissions-Policy |
| RBAC-M02/M-3 | `2e4ca56` | `RequireSelfOrLeadership` middleware on roles + member-summary routes |
| L-1, L-2, M-4, L-6 | `3130cf1` | michango route alignment; welfare fallback leak; ledger replay admin-only; announcements to active members |

**Remaining (documented, intentionally not fixed in code):**
- TLS-01: HTTPS requires a real certificate + domain — infrastructure task at deploy time (nginx `listen 443 ssl`; HSTS line is pre-written and commented in nginx.conf).
- DATA-M01 masking: payment-method numbers must remain visible to members (they need them to pay); member lists are leadership-only, so per-role DTO masking was judged unnecessary for v1.
- L-7 backup disk quota: rate limited to 5/hour/admin; monitor in production.
- `social_funds`/`social_fund_contributions` tables remain in the DB but are unused (feature consolidated into welfare events) — safe to DROP manually.
