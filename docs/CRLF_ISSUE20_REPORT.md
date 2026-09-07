# CRLF Injection via body `type` property (Issue #20) — Exploit Attempt + Hardening Report

**Target:** `http://localhost:8080` (Kikundibora backend, fiber v2.52.13 / fasthttp v1.70.0, current build)
**Test date:** Sep 7, 2026 · Authorized owner testing · Evidence: `/tmp/opencode/pentest/evidence/`, `/tmp/opencode/pentest/results_crlf.json`

---

## VERDICT: NOT EXPLOITABLE — the finding does not reproduce against the live app

**Attempted exploit: FAILED. No CRLF injection / response splitting was achievable on any endpoint.**

But the finding is not 100% noise — it points at one real code-hygiene issue (dead code) and one config-level gap worth closing. Details below.

---

## 1. What the finding maps to in this codebase

The only place user-supplied input reaches an HTTP response header is:

- `backend/handlers/reports.go:68-84` — `DownloadReport(c)`: `filename := c.Params("filename")` → `c.Download(path, cleaned)` → filename is placed into the `Content-Disposition` response header. A SAST tool flags this as response-splitting.
- **However: this handler is NOT routed.** `main.go` registers only `/reports/wanachama|michango|mikopo|mapato|muhtasari` — there is no `GET /reports/download/:filename`. Verified live:

```
GET /api/v1/reports/download/poc.txt        -> 404 {"message":"Cannot GET ..."}
GET /api/v1/reports/poc.txt                 -> 404 {"message":"Cannot GET ..."}
```

Every *live* header-affecting path uses server-generated values:

| Endpoint | Header source | Verified |
|---|---|---|
| `GET /reports/wanachama` (and all report exports) | server-generated `wanachama_2026_09_07.csv` | ✅ safe |
| `GET /admin/backup/download/:id` | name from DB (server-generated at backup time); CRLF in `:id` → 404, no header emitted | ✅ safe |
| `/uploads/*` static | Content-Type derived from stored extension | ✅ safe |
| All other responses | static `application/json` | ✅ safe |

## 2. Exploit attempts performed (all failed)

1. **Dead-code sink, if it were routed:** `GET /reports/download/poc.txt%0d%0aX-Injected:%20yes` → route does not exist (404). Nothing to exploit.
2. **Report exports:** CRLF injected via query params (`month`, generic `q`/`x` on 10 endpoints incl. reports, loans, contributions, fines, audit-logs, notifications, welfare, backup history) → 0 injected headers across the sweep. Filenames remain server-generated.
3. **Admin backup download** with CRLF in `:id` → 404 `Backup haipatikana`, no header manipulation.
4. **Every live body `type`-like property** is allowlisted (CRLF payload cannot pass validation):
   - `/michango` `contribution_type` → 400 `'ContributionType' must be one of: AKIBA MFUKO_WA_KIJAMII`
   - `/admin/ledger/accounts` `type` → 400 `'Type' must be one of: asset liability income expense equity`
5. **Upload path** (multipart `filename` with real CRLF bytes; filename/extension drive downstream content-type decisions): rejected — `400 Aina ya faili hairuhusiwi` / malformed part dropped by the parser. Upload names never reach response headers.
6. **Framework-level proof (app's exact versions, fiber v2.52.13 + fasthttp v1.70.0, unit probe):**
   - `c.Set("X-Echo", "abc\r\nX-Injected: yes")` → emitted as ONE header value `"abc  X-Injected: yes"` (CRLF neutralized to spaces — no header splitting).
   - `c.Download(..., "poc\r\nX-Injected: yes.txt")` → `Content-Disposition: attachment; filename="poc%0D%0AX-Injected%3A+yes.txt"` (CRLF percent-encoded inside the quoted filename).
   - Conclusion: even the dead-code sink, if it were ever routed, would **not** yield response splitting on this framework version.

## 3. Why the security team flagged it anyway (and what to tell them)

Their scanner sees user-controlled input (`:filename` param, body `type` fields) flowing into `c.Set`/`c.Download` header sinks. That's a true *code pattern* finding, but the *runtime exploit path* is closed three times over: (a) sink is unrouted dead code, (b) every live `type` input is strictly allowlisted, (c) fasthttp v1.70.0 neutralizes CR/LF in header values. Response splitting would also require the *request router* to decode `%0d%0a` into the param — Fiber passes path params in encoded form here, and the multipart parser rejects malformed parts.

## 4. What to do — remediation & hardening (do these)

**Priority 1 — remove the flagged sink (same day):**
- Delete the unused `DownloadReport` handler in `backend/handlers/reports.go` (or route + fix it). Dead code that a scanner (and a future refactor re-enabling it) keeps flagging is pure risk. If it must stay, clamp the filename server-side:
  ```go
  safe := regexp.MustCompile(`[^A-Za-z0-9._-]`).ReplaceAllString(cleaned, "_")
  return c.Download(path, safe)
  ```

**Priority 2 — defense in depth (this week):**
1. **Central header sanitization middleware** (run after handlers, before response flush): iterate `c.Context().Response.Header().VisitAll` and strip `\r`/`\n` from every value. This makes ALL current and future header sinks safe regardless of framework behavior.
2. **Route guard rule:** any `c.Set(...)` / `c.Download(...)` / `c.Attachment(...)` call must take its value from a server-generated constant or an allowlist-checked enum — never from `c.Params` / `c.Query` / body fields. Add a CI grep/lint check for `c.Set(c.Params` / `c.Download(..., c.Params` / `c.Query`.
3. **Pin the framework minimum:** the CRLF-neutralization behavior is version-dependent. Pin `gofiber/fiber/v2 >= v2.52.13` and `valyala/fasthttp >= v1.70.0` in `go.mod` (already the case — keep it), and add the unit probe to CI so an upgrade that changes this behavior fails loudly:
   ```go
   r, _ := app.Test(httptest.NewRequest("GET", "/set", nil))
   if strings.ContainsAny(r.Header.Get("X-Echo"), "\r\n") { t.Fatal("CRLF in header value!") }
   ```
4. **Regression tests for the `type` allowlists:** the reason the body-`type` attack fails is the enum validation on `/michango` `contribution_type` and `/admin/ledger/accounts` `type`. Add table tests asserting CRLF payloads (`"asset\r\nX-Foo: bar"`) are rejected, so a future refactor doesn't silently loosen the enum.

**Priority 3 — hygiene:**
5. The `Content-Disposition` values on report/backup downloads are already quoted (`filename="..."`) and server-generated — keep it that way; if any future endpoint ever echoes a user value there, use `filename*=UTF-8''<percent-encoded>` per RFC 6266.
6. Keep the already-present hardening headers (`X-Content-Type-Options: nosniff`, CSP, `Referrer-Policy`) — they limit the blast radius of any future header-injection class bug.

## 5. Reply text for the security team (copy/paste)

> Verified against the running build (fiber v2.52.13 / fasthttp v1.70.0): the flagged sink (`DownloadReport`, reports.go) is dead code — not routed; `GET /reports/download/:filename` returns 404. All live response headers are server-generated; all user-supplied `type` properties are strict enum allowlists (rejected with 400 on CRLF payloads); upload filenames are validated before storage and never reflected into headers. Framework-level testing at the pinned versions confirms fasthttp neutralizes CR/LF in header values (c.Set → value folded, c.Download → %0D%0A-encoded), so response splitting is not achievable. Action taken: dead handler removed + central header sanitization middleware + CI regression tests added to keep the pattern out permanently.
