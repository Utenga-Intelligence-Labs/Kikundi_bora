# Kikundi Bora Backend Developer Guide

## Quick orientation

The backend is a Go 1.26+ HTTP API built with Fiber v2, GORM, and PostgreSQL.
The application entry point, startup sequence, and complete URL registration are
in [backend/main.go](../backend/main.go). There is no separate `controllers`
directory: the HTTP controller layer is the [backend/handlers](../backend/handlers)
package.

The API base URL is:

```text
http://<host>:<port>/api/v1
```

The health endpoint is outside that prefix: `GET /health`.

## Where URLs live

All production routes are registered in [backend/main.go](../backend/main.go),
inside `main()`:

1. `app` contains global middleware, `GET /health`, and the authenticated
   `/uploads` static-file route.
2. `api := app.Group("/api/v1")` defines the API prefix.
3. `auth := api.Group("/auth")` contains public login and OTP routes.
4. `protected := api.Group("")` applies `middleware.AuthRequired` to all
   authenticated routes.
5. Feature groups such as `members`, `contribs`, `loans`, `welfare`, and
   `ledgerRoutes` add their own URL prefixes and authorization middleware.

### URL family to controller mapping

| URL family | Handler/controller file | Main responsibilities |
|---|---|---|
| `/auth`, `/me` | [handlers/auth.go](../backend/handlers/auth.go) | Login, OTP, sessions, profile, password operations |
| `/dashboard`, `/members/*/dashboard-summary`, `/groups/*/dashboard-summary` | [handlers/dashboard.go](../backend/handlers/dashboard.go), [handlers/dashboard_scoped.go](../backend/handlers/dashboard_scoped.go) | Personal and group dashboards |
| `/groups/*`, contribution settings | [handlers/group_settings.go](../backend/handlers/group_settings.go) | Group settings and proposal approval |
| `/members/*` | [handlers/members.go](../backend/handlers/members.go) | Member CRUD and approval lifecycle |
| `/users/*` | [handlers/user_management.go](../backend/handlers/user_management.go) | User creation, approval, roles, password reset |
| `/contributions/*` | [handlers/contributions.go](../backend/handlers/contributions.go) | Treasurer-recorded contributions |
| `/michango/*` | [handlers/member_contribution.go](../backend/handlers/member_contribution.go) | Member-submitted contributions and verification |
| `/welfare/*` | [handlers/welfare.go](../backend/handlers/welfare.go) | Mfuko wa Kijamii events, obligations, payments |
| `/loans/*`, `/repayments/*` | [handlers/loans.go](../backend/handlers/loans.go), [handlers/repayments.go](../backend/handlers/repayments.go) | Loan lifecycle and repayments |
| `/loan-committee/*` | [handlers/loan_committee.go](../backend/handlers/loan_committee.go) | Committee membership and loan review |
| `/uongozi/*` | [handlers/leadership.go](../backend/handlers/leadership.go) | Leadership dashboard, reports, loan approvals |
| `/notifications/*` | [handlers/notifications.go](../backend/handlers/notifications.go) | User notifications |
| `/audit-logs/*` | [handlers/audit.go](../backend/handlers/audit.go) | Audit and login activity views |
| `/pending-actions/*` | [handlers/pending_actions.go](../backend/handlers/pending_actions.go) | Chairperson approval queue |
| `/admin/*` | [handlers/admin.go](../backend/handlers/admin.go), [handlers/backup.go](../backend/handlers/backup.go), [handlers/ledger.go](../backend/handlers/ledger.go) | Admin operations, backups, accounting API |
| `/reports/*` | [handlers/reports.go](../backend/handlers/reports.go) | Chair-only reports |
| `/upload/*` | [handlers/upload.go](../backend/handlers/upload.go) | Avatar and document uploads |
| `/dissolution-*` | [handlers/dissolution.go](../backend/handlers/dissolution.go) | Group dissolution proposals and payouts |
| `/import/*` | [handlers/import.go](../backend/handlers/import.go) | Historical contribution and loan imports |
| `/announcements` | [handlers/announcement.go](../backend/handlers/announcement.go) | Leadership announcements |
| Fine and obligation routes | [handlers/fine_settings.go](../backend/handlers/fine_settings.go), [handlers/obligations.go](../backend/handlers/obligations.go) | Fine types, fines, arrears, collection queues |
| `/payment-methods/*` | [handlers/payment_methods.go](../backend/handlers/payment_methods.go) | Lipa Namba and bank payment methods |

The registration file is the source of truth for the HTTP method, path,
handler method, and route guard. Comments in handler files document individual
endpoint paths, but should be treated as secondary to `main.go`.

## Request flow

```text
HTTP request
  -> global CORS/security/rate-limit/request logging middleware
  -> route group middleware (JWT authentication, role/position/leadership guard)
  -> handler in backend/handlers
  -> backend/services and/or backend/database
  -> GORM models or the backend/ledger package
  -> JSON response
```

### Authentication and authorization

- [middleware/auth.go](../backend/middleware/auth.go) validates Bearer JWTs,
  checks the server-side session, refreshes activity, reloads the user role from
  the database, and stores `user_id` and `role` in Fiber locals.
- [middleware/leadership.go](../backend/middleware/leadership.go) checks active
  leadership positions. This is separate from a user's canonical role.
- [middleware/position.go](../backend/middleware/position.go) checks position-
  based permissions such as treasurer or chairperson.
- `RequireRoles`, committee access, and related authorization helpers are also
  in [middleware/auth.go](../backend/middleware/auth.go).
- Global middleware is installed before route registration in `main()`:
  CORS, security headers, rate limiting, and request logging.

When adding a route, register it in `main.go` and attach the narrowest suitable
guard there. Do not rely on frontend visibility for authorization.

## Business and data layers

- [backend/services](../backend/services): reusable business operations such as
  treasury calculations, notifications, audit logging, obligations, reports,
  SMS/email, scheduling, and automatic ledger posting.
- [backend/database](../backend/database): PostgreSQL/GORM connection,
  migrations, seed data, group setup, and synchronization helpers.
- [backend/models](../backend/models): GORM persistence models, enums, and
  request/response DTOs.
- [backend/ledger](../backend/ledger): append-only event-sourced accounting,
  projections, replay, validation, and ledger queries. It uses a raw pgx pool
  in addition to the GORM application database.
- [backend/config/config.go](../backend/config/config.go): environment-backed
  application configuration.

Handlers generally receive `*fiber.Ctx`, validate/authorize the request, call
`database.DB` or a service, record an audit event where appropriate, and return
a Fiber JSON response. Keep database-heavy or cross-handler rules in services
when they are reused.

## LOC snapshot

Measured on 2026-09-06 with `wc -l` over Go files:

| Area | Lines |
|---|---:|
| All backend Go files | 25,668 |
| Production Go files (`*_test.go` excluded) | 19,433 |
| Test Go files | 6,235 |
| `handlers` total | 14,924 |
| `handlers` production files | 10,674 |
| `middleware` | 823 |
| `services` | 3,821 |
| `database` | 865 |
| `models` | 1,730 |
| `ledger` | 2,751 |
| `config` | 92 |
| `main.go` | 530 |

`main.go` currently contains 171 concrete HTTP method registrations (`GET`,
`POST`, `PUT`, `PATCH`, and `DELETE`), plus route-group and middleware
declarations. The count includes the health and upload routes.

## Working conventions

1. Start route work in [backend/main.go](../backend/main.go), then follow the
   handler symbol passed to the Fiber method.
2. Check the route's middleware before changing behavior; many permissions are
   intentionally enforced at registration time.
3. Use request DTOs and validation patterns already present in
   [backend/models/requests.go](../backend/models/requests.go) and handlers.
4. Preserve the JSON envelope and naming conventions documented in
   [docs/API_CONTRACT.md](API_CONTRACT.md).
5. Add or update focused tests beside the affected package. Handler tests use
   the repository's test configuration and helpers.
6. Run `gofmt` on changed Go files and verify with:

```bash
cd backend
go test ./...
```

For local startup, configure PostgreSQL and environment variables as described
in [backend/README.md](../backend/README.md), then run `go run .` or `make run`.