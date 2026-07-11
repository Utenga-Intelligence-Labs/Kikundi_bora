# Kikundi Bora

**Mfumo wa Ushirika wa Akiba na Mikopo** — A digital platform for managing savings and loan groups (vikundi), built for Tanzanian communities.

[![Tests](https://img.shields.io/badge/tests-112%20passing-brightgreen)](#)

---

## Features

- **Member Management** — Register, edit, activate/deactivate members with unique membership numbers
- **Contributions** — Record monthly contributions with duplicate detection
- **Loan Processing** — Apply, committee review, disburse, track repayments, auto-close
- **Loan Committee** — Multi-member review with unanimous approval gating
- **Welfare Fund (Mfuko wa Kijamii)** — Event creation, approval, per-member contribution tracking
- **Reports** — CSV exports for members, contributions, loans, income/expense summaries
- **Audit Trail** — Immutable activity log for every action
- **Admin Panel** — User override, password reset, system health monitoring
- **PWA Support** — Installable on mobile, offline-capable
- **100% Kiswahili UI** — Built for Swahili-speaking users

---

## Architecture

```
Kikundi_bora/
├── backend/                 # Go API server
│   ├── config/              # Environment configuration
│   ├── database/            # GORM migrations & seed data
│   ├── handlers/            # HTTP handlers (16 controllers)
│   ├── middleware/           # Auth, CORS, security, role guards
│   ├── models/              # GORM models & request DTOs
│   ├── services/            # Email, audit, backup, notifications
│   └── main.go              # Entry point, route definitions
│
├── Frontend-1/              # React SPA
│   └── src/
│       ├── api/             # Fetch-based API client (typed)
│       ├── components/      # shadcn/ui + custom components
│       ├── hooks/           # TanStack Query wrappers (15 hooks)
│       ├── lib/             # Auth, roles, formatting utilities
│       └── routes/          # File-based TanStack Router pages
│
└── start.sh                 # One-command startup script
```

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| **Backend** | Go 1.24 · Fiber v2 · GORM · PostgreSQL |
| **Frontend** | React 19 · TanStack Start · TanStack Router · TanStack Query v5 |
| **Styling** | Tailwind CSS v4 · shadcn/ui (Radix) |
| **Auth** | JWT (HS256) · bcrypt · session revocation |
| **Testing** | Go `testing` · Vitest · Testing Library · MSW |
| **PWA** | vite-plugin-pwa · Workbox |

---

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 18+
- PostgreSQL 16+ (running locally)

### 1. Clone

```bash
git clone https://github.com/Startup101-Bongo/Kikundi_bora.git
cd Kikundi_bora
```

### 2. Configure

```bash
cp backend/.env.example backend/.env
```

Edit `backend/.env` with your PostgreSQL credentials and a JWT secret:

```env
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=your_user
DB_PASSWORD=your_password
DB_NAME=kikundi_db
JWT_SECRET=generate-with-openssl-rand-hex-32
ADMIN_PASSWORD=YourAdminPassword123
```

### 3. Run

```bash
./start.sh
```

This will:
1. Create the database tables and seed demo data
2. Start the backend on `http://localhost:8080`
3. Install frontend dependencies and start on `http://localhost:8081`

### Demo Accounts

| Role | Phone | Password |
|------|-------|----------|
| Chairperson (Mwenyekiti) | 0710000001 | demo123 |
| Treasurer (Mweka Hazina) | 0710000002 | demo123 |
| Secretary (Katibu) | 0710000003 | demo123 |
| Member (Mwanachama) | 0710000004 | demo123 |

---

## API Overview

Base URL: `http://localhost:8080/api/v1`

### Authentication

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/auth/login` | No | Login with phone/email + password |
| GET | `/me` | Yes | Current user profile |
| POST | `/auth/logout` | Yes | Revoke token |
| POST | `/auth/change-password` | Yes | Change own password |
| POST | `/auth/first-login-setup` | Yes | Set password on first login |

### Members

| Method | Endpoint | Roles | Description |
|--------|----------|-------|-------------|
| GET | `/members` | All | List members (paginated, searchable) |
| GET | `/members/:id` | All | Get member details |
| POST | `/members` | Chair/Secretary/Treasurer | Register new member |
| PUT | `/members/:id` | Chair/Secretary | Update member |
| DELETE | `/members/:id` | Chair only | Soft-delete member |

### Loans

| Method | Endpoint | Roles | Description |
|--------|----------|-------|-------------|
| POST | `/loans/apply` | All | Submit loan application |
| POST | `/loan-committee/loans/:id/review` | Committee | Approve/reject (committee review) |
| POST | `/loans/:id/disburse` | Treasurer | Disburse approved loan |
| POST | `/repayments` | Treasurer | Record repayment |

### Welfare

| Method | Endpoint | Roles | Description |
|--------|----------|-------|-------------|
| POST | `/welfare/events` | Treasurer | Create welfare event |
| POST | `/welfare/events/:id/approve` | Chair | Approve event |
| POST | `/welfare/events/:id/contributions/:mid/pay` | Treasurer | Record member payment |

---

## Testing

### Run all tests

```bash
# Frontend (112 tests)
cd Frontend-1 && npm test

# Backend (22 tests)
cd backend && go test ./... -v
```

### Test coverage

| Category | Tests | Covers |
|----------|-------|--------|
| Pure functions | 30 | Currency formatting, JWT decode, initials, validation helpers |
| Auth storage | 9 | Token set/get/clear/role extraction |
| Role guards & nav | 27 | hasRole, sidebar nav per role, mobile nav, subtitles |
| Hooks | 18 | useMembers, useLoans, useWelfare, useLoanCommittee |
| API client (MSW) | 8 | GET/POST/PUT/DELETE, auth headers, error handling |
| Auth provider | 2 | Login flow, logout flow |
| Route smoke | 3 | Auth gating, role-based rendering |
| Backend unit | 12 | escapeLike, formatMoney, detectContentType, magic bytes |
| Backend lifecycle | 10 | Full HTTP loan (6-state), welfare event, user approval, rate limiting, negative scenarios |

---

## Project Status

- [x] Member CRUD with soft-delete
- [x] Contribution recording with duplicate detection
- [x] Loan lifecycle (Apply → Committee → Disburse → Repay → Close)
- [x] Loan committee with unanimous approval
- [x] Welfare events with member contributions
- [x] Dashboard with role-based views
- [x] CSV report exports
- [x] Audit logging
- [x] JWT auth with session revocation
- [x] Role-based access control
- [x] Rate limiting (5 attempts/5 min)
- [x] Magic byte verification on uploads
- [x] PWA with offline caching
- [x] 112 automated tests
