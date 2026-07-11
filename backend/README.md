# Kikundi Bora API

Backend API for managing a **Kikundi cha Kuweka na Kukopa** (savings and loans group). Built with Go, Fiber, and PostgreSQL.

## Tech Stack

- **Go 1.26+**
- **Fiber v2** — HTTP web framework
- **GORM** — ORM for PostgreSQL
- **JWT** — Authentication (HS256)
- **PostgreSQL 14+** — Database
- **gomail** — Email delivery (optional)

## Features

- User authentication (login, logout, password change)
- Self-registration with approval workflow (Secretary approves)
- Role-based access control (Admin, Chair, Treasurer, Secretary, Member)
- Position-based access control (Chairperson, Treasurer, Secretary positions)
- Member management (CRUD)
- Contributions tracking (monthly contributions per member)
- Loan management (apply, committee review, approve, reject, disburse, track outstanding)
- Loan committee workflow (appointed members review loans)
- Welfare events (Mfuko wa Kijamii)
- Repayment recording
- Dashboard summary
- Notifications system
- Comprehensive audit logging
- Failed login tracking with rate limiting (5 attempts / 5 min)
- File uploads (avatars, documents) with magic byte validation
- Database backups (pg_dump with zip)
- Security headers (CSP, HSTS, X-Frame-Options, etc.)

## Project Structure

```
backend/
├── config/          # App configuration (env loading)
├── database/        # DB connection, migrations, seeding
├── handlers/        # HTTP request handlers
├── middleware/       # Auth, CORS, security headers, position middleware
├── models/          # GORM models & request DTOs
├── services/        # Business logic (audit, backup, email, notifications)
├── uploads/         # Uploaded files (avatars, docs, reports)
├── backups/         # Database backup archives
├── main.go          # Entry point & route definitions
├── Makefile         # Build & run commands
└── schema.md        # Database schema documentation
```

## Getting Started

### Prerequisites

- Go 1.26+
- PostgreSQL 14+
- pg_dump (for backup feature)

### Setup

1. Clone the repository:
   ```bash
   git clone <repo-url>
   cd backend
   ```

2. Copy `.env.example` to `.env` and configure:
   ```bash
   cp .env.example .env
   ```

3. Update `.env` with your credentials:
   ```
   DB_HOST=127.0.0.1
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=yourpassword
   DB_NAME=kikundi_db
   DB_SSLMODE=disable
   JWT_SECRET=<generate-with-openssl-rand-hex-32>
   ADMIN_PASSWORD=<your-admin-password-min-8-chars>
   PORT=8080
   CORS_ORIGINS=http://localhost:3000,http://localhost:5173
   ENVIRONMENT=development
   ```

4. Create the database:
   ```sql
   CREATE DATABASE kikundi_db;
   ```

5. Run migrations and seed:
   ```bash
   go run . -migrate
   ```

6. Start the server:
   ```bash
   make run
   # or
   go run .
   ```

The API will be available at `http://localhost:8080`.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DB_HOST` | Yes | `127.0.0.1` | PostgreSQL host |
| `DB_PORT` | Yes | `5432` | PostgreSQL port |
| `DB_USER` | Yes | `postgres` | Database user |
| `DB_PASSWORD` | Yes | — | Database password |
| `DB_NAME` | Yes | `kikundi_db` | Database name |
| `DB_SSLMODE` | Yes | `disable` | SSL mode (`disable`, `require`, `verify-full`) |
| `JWT_SECRET` | Yes | — | JWT signing secret (min 32 chars) |
| `ADMIN_PASSWORD` | Yes (for seed) | — | Admin account password (min 8 chars) |
| `PORT` | No | `8080` | Server port |
| `CORS_ORIGINS` | No | `localhost:3000,5173` | Comma-separated allowed origins |
| `ENVIRONMENT` | No | `development` | `development` or `production` |
| `SMTP_HOST` | No | — | SMTP server for email delivery |
| `SMTP_PORT` | No | `587` | SMTP port |
| `SMTP_USERNAME` | No | — | SMTP username |
| `SMTP_PASSWORD` | No | — | SMTP password |
| `SMTP_FROM` | No | — | Sender email address |

## Deployment

### Production Checklist

#### 1. Environment Variables

```bash
# Generate a secure JWT secret (min 32 characters)
openssl rand -hex 32

# Generate a strong admin password
openssl rand -base64 24
```

Set these in your production environment:

```bash
# REQUIRED — Never use defaults in production
JWT_SECRET=<your-generated-secret>
ADMIN_PASSWORD=<your-generated-password>
DB_PASSWORD=<strong-database-password>

# Database — Use SSL in production
DB_HOST=your-db-host
DB_PORT=5432
DB_USER=your-db-user
DB_NAME=kikundi_db
DB_SSLMODE=require

# Server
PORT=8080
ENVIRONMENT=production

# CORS — Restrict to your production domain(s)
CORS_ORIGINS=https://yourdomain.co.tz

# SMTP — Configure for email notifications
SMTP_HOST=smtp.yourprovider.com
SMTP_PORT=587
SMTP_USERNAME=your-email
SMTP_PASSWORD=your-email-password
SMTP_FROM=noreply@yourdomain.co.tz
```

#### 2. Database Setup

```bash
# Create database
createdb kikundi_db

# Run migrations and create admin account
go run . -migrate
```

The admin account is created with:
- Phone: `0000000000`
- Password: value of `ADMIN_PASSWORD` env var

**Change the admin password immediately after first login.**

#### 3. Build and Run

```bash
# Build binary
make build
# or
go build -o kikundi-api .

# Run
./kikundi-api

# Or with systemd (see below)
```

#### 4. Systemd Service (Linux)

Create `/etc/systemd/system/kikundi-api.service`:

```ini
[Unit]
Description=Kikundi Bora API
After=network.target postgresql.service

[Service]
Type=simple
User=kikundi
Group=kikundi
WorkingDirectory=/opt/kikundi-api
ExecStart=/opt/kikundi-api/kikundi-api
Restart=always
RestartSec=5

# Environment variables (or use EnvironmentFile)
EnvironmentFile=/opt/kikundi-api/.env

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/kikundi-api/uploads /opt/kikundi-api/backups

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable kikundi-api
sudo systemctl start kikundi-api
sudo systemctl status kikundi-api
```

#### 5. Nginx Reverse Proxy

```nginx
server {
    listen 80;
    server_name api.yourdomain.co.tz;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.yourdomain.co.tz;

    ssl_certificate /etc/letsencrypt/live/api.yourdomain.co.tz/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.yourdomain.co.tz/privkey.pem;

    # Security headers (backup — app also sets these)
    add_header X-Frame-Options "DENY" always;
    add_header X-Content-Type-Options "nosniff" always;

    # Upload size limit
    client_max_body_size 10M;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

#### 6. Backups

Backups are generated via the admin API and stored in `./backups/`. Configure automatic cleanup (files older than 7 days are deleted).

To manually trigger a backup:
```bash
curl -X POST https://api.yourdomain.co.tz/api/v1/admin/backup/generate \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"backup_type": "DATABASE"}'
```

To set up automatic email backups, configure SMTP and save backup settings via:
```bash
curl -X POST https://api.yourdomain.co.tz/api/v1/admin/backup/settings \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@yourdomain.co.tz", "backup_type": "DATABASE", "frequency": "DAILY"}'
```

#### 7. Monitoring

Health check endpoint:
```bash
curl https://api.yourdomain.co.tz/health
# {"status":"ok","service":"kikundi-api"}
```

System health (admin only):
```bash
curl https://api.yourdomain.co.tz/api/v1/admin/health \
  -H "Authorization: Bearer <admin-token>"
```

## API Endpoints

### Public

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/login` | Login |
| GET | `/health` | Health check |

### Auth (requires JWT)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/me` | Get current user |
| PUT | `/api/v1/me` | Update profile |
| POST | `/api/v1/auth/change-password` | Change password |
| POST | `/api/v1/auth/logout` | Logout |
| POST | `/api/v1/auth/first-login-setup` | Set new password on first login |
| POST | `/api/v1/auth/register` | Self-register (requires approval) |

### User Management (Chair/Secretary)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/users/create` | Create user (Chair) |
| GET | `/api/v1/users/pending` | List pending users (Secretary) |
| GET | `/api/v1/users/` | List all users (Chair/Secretary) |
| POST | `/api/v1/users/:id/approve` | Approve user (Secretary) |
| POST | `/api/v1/users/:id/reject` | Reject user (Secretary) |

### Members

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/members` | List members |
| GET | `/api/v1/members/:id` | Get member |
| POST | `/api/v1/members` | Create member (Chair/Secretary/Treasurer) |
| PUT | `/api/v1/members/:id` | Update member (Chair/Secretary) |
| DELETE | `/api/v1/members/:id` | Delete member (Chair only) |

### Contributions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/contributions` | List contributions |
| POST | `/api/v1/contributions` | Record contribution (Treasurer) |
| PUT | `/api/v1/contributions/:id` | Edit contribution (Treasurer) |
| GET | `/api/v1/contributions/monthly-report` | Monthly report |

### Loans

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/loans` | List loans |
| GET | `/api/v1/loans/:id` | Get loan with repayments |
| POST | `/api/v1/loans/apply` | Apply for loan |
| POST | `/api/v1/loans/:id/approve` | Approve loan (Chair) |
| POST | `/api/v1/loans/:id/reject` | Reject loan (Chair/Treasurer) |
| POST | `/api/v1/loans/:id/disburse` | Disburse loan (Treasurer) |
| GET | `/api/v1/loans/outstanding-report` | Outstanding loans report |

### Repayments

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/repayments` | List repayments |
| POST | `/api/v1/repayments` | Record repayment (Treasurer) |

### Loan Committee

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/loan-committee/check` | Check if current user is committee member |
| GET | `/api/v1/loan-committee/members` | List committee members |
| POST | `/api/v1/loan-committee/members` | Appoint member (Chair) |
| DELETE | `/api/v1/loan-committee/members/:id` | Remove member (Chair) |
| GET | `/api/v1/loan-committee/loans` | List loans for review |
| GET | `/api/v1/loan-committee/loans/:id` | Get loan details |
| POST | `/api/v1/loan-committee/loans/:id/review` | Submit review (Approve/Reject) |
| GET | `/api/v1/loan-committee/dashboard` | Committee dashboard |
| GET | `/api/v1/loan-committee/history` | Review history |
| GET | `/api/v1/loan-committee/report` | Activity report |
| GET | `/api/v1/loan-committee/pending-count` | Pending loans count |

### Welfare (Mfuko wa Kijamii)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/welfare/dashboard` | Welfare dashboard |
| GET | `/api/v1/welfare/events` | List events |
| GET | `/api/v1/welfare/events/:id` | Get event |
| POST | `/api/v1/welfare/events` | Create event (Treasurer) |
| POST | `/api/v1/welfare/events/:id/approve` | Approve event (Chair) |
| POST | `/api/v1/welfare/events/:id/reject` | Reject event (Chair) |
| GET | `/api/v1/welfare/contributions` | List contributions |
| GET | `/api/v1/welfare/my-contributions` | My contributions |
| POST | `/api/v1/welfare/events/:id/contributions/:memberId/pay` | Record payment (Treasurer) |
| POST | `/api/v1/welfare/events/:id/contributions/:memberId/waive` | Waive contribution (Treasurer) |

### Notifications

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/notifications` | List notifications |
| POST | `/api/v1/notifications/read` | Mark as read |

### Audit Logs (Chair/Admin)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/audit-logs` | All audit logs (filterable) |
| GET | `/api/v1/audit-logs/login-activity` | Login/logout activity |
| GET | `/api/v1/audit-logs/failed-logins` | Failed login attempts |
| GET | `/api/v1/audit-logs/summary` | Audit statistics summary |

#### Audit Log Filters

All audit endpoints support these query parameters:

| Parameter | Example | Description |
|-----------|---------|-------------|
| `page` | `1` | Page number |
| `limit` | `20` | Results per page (max 500) |
| `table` | `users` | Filter by table name |
| `action` | `LOGIN` | Filter by action type |
| `user_id` | `uuid` | Filter by user |
| `q` | `192.168` | Search IP or user agent |
| `from` | `2024-01-01` | Start date |
| `to` | `2024-12-31` | End date |
| `ip` | `192.168.1.1` | Filter by IP (login endpoints) |
| `email` | `user@` | Filter by email (failed logins) |
| `days` | `7` | Lookback period (summary endpoint) |

### Admin (Admin only)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/admin/users` | List all users |
| GET | `/api/v1/admin/logs` | Admin action logs |
| POST | `/api/v1/admin/users/:id/override` | Activate/deactivate/suspend user |
| POST | `/api/v1/admin/users/:id/reset-password` | Reset user password |
| POST | `/api/v1/admin/auth/reset-password` | Reset password by email |
| GET | `/api/v1/admin/health` | System health stats |
| POST | `/api/v1/admin/backup/generate` | Generate backup |
| GET | `/api/v1/admin/backup/history` | Backup history |
| GET | `/api/v1/admin/backup/settings` | Get backup settings |
| POST | `/api/v1/admin/backup/settings` | Save backup settings |
| GET | `/api/v1/admin/backup/download/:id` | Download backup |

### Reports (Chair only)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/reports/wanachama` | Members report |
| GET | `/api/v1/reports/michango` | Contributions report |
| GET | `/api/v1/reports/mikopo` | Loans report |
| GET | `/api/v1/reports/mapato` | Income/expense report |
| GET | `/api/v1/reports/muhtasari` | Summary report |

### Pending Actions (Chairperson)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/pending-actions/` | List pending actions |
| GET | `/api/v1/pending-actions/:id` | Get pending action |
| POST | `/api/v1/pending-actions/:id/approve` | Approve action |
| POST | `/api/v1/pending-actions/:id/reject` | Reject action |

### File Uploads

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/upload/avatar` | Upload avatar (max 5MB, images only) |
| POST | `/api/v1/upload/doc` | Upload document (max 5MB, docs/images) |

## Commands

```bash
make run      # Run the application
make build    # Build binary
make tidy     # Tidy go modules
make vet      # Run go vet
make test     # Run tests
```

## Roles

| Role | Permissions |
|------|-------------|
| **Admin** | Full system access — user management, backups, system health, all role bypasses |
| **Chair** | Manage members, approve loans, view audit logs, reports, pending actions |
| **Treasurer** | Record contributions, disburse loans, record repayments, welfare events |
| **Secretary** | Approve/reject user registrations, manage members |
| **Member** | View own data, apply for loans, view notifications |

## Positions

Positions are assigned to users and grant additional permissions:

| Position | Permissions |
|----------|-------------|
| **Chairperson** | Approve pending actions |
| **Treasurer** | Financial operations (contributions, disbursements, repayments) |
| **Secretary** | User approval workflow |

## Security

### Authentication
- JWT tokens with 24-hour expiry
- Bearer token in `Authorization` header
- Rate limiting on login (5 attempts per 5 minutes per IP)
- Failed login tracking and audit logging

### Authorization
- Role-based access control (RBAC)
- Position-based access control
- Admin role bypasses all checks

### Data Protection
- Passwords hashed with bcrypt
- SQL injection prevention (parameterized queries + LIKE escaping)
- File upload validation (extension + magic bytes)
- Path traversal protection on file downloads

### HTTP Security Headers
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Strict-Transport-Security` (production only)
- `Content-Security-Policy`
- `Referrer-Policy`
- `Permissions-Policy`

## License

Proprietary — Startup101-Bongo
