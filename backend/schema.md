# Database Schema — Kikundi cha Kuweka na Kukopa

## Tables

```
kikundi_db/
├── users
├── members
├── contributions
├── contribution_edits
├── loans
├── repayments
├── audit_logs
├── user_sessions
├── failed_logins
└── notifications
```

---

## users
| Column | Type | Constraints |
|---|---|---|
| id | bigint | PK, AUTO |
| name | varchar(100) | NOT NULL |
| email | varchar(150) | NOT NULL, UNIQUE |
| password | varchar(255) | NOT NULL |
| role | enum('chair','treasurer','secretary') | NOT NULL |
| is_active | boolean | DEFAULT true |
| last_login_at | timestamp | NULLABLE |
| deleted_at | timestamp | NULLABLE (soft delete) |
| created_at | timestamp | AUTO |
| updated_at | timestamp | AUTO |

---

## members
| Column | Type | Constraints |
|---|---|---|
| id | bigint | PK, AUTO |
| member_no | varchar(20) | NOT NULL, UNIQUE |
| full_name | varchar(150) | NOT NULL |
| phone | varchar(15) | NOT NULL, UNIQUE |
| address | text | NULLABLE |
| joined_at | date | NOT NULL |
| is_active | boolean | DEFAULT true |
| registered_by | bigint | FK → users.id |
| deleted_at | timestamp | NULLABLE (soft delete) |
| created_at | timestamp | AUTO |
| updated_at | timestamp | AUTO |

---

## contributions
| Column | Type | Constraints |
|---|---|---|
| id | bigint | PK, AUTO |
| member_id | bigint | FK → members.id |
| recorded_by | bigint | FK → users.id |
| amount | decimal(15,2) | NOT NULL, CHECK(amount > 0) |
| month | date | NOT NULL (YYYY-MM-01) |
| paid_at | date | NOT NULL |
| notes | text | NULLABLE |
| created_at | timestamp | AUTO |

**Constraints:**
- `UNIQUE(member_id, month)`
- `CHECK(amount > 0)`
- `INDEX(member_id)`
- `INDEX(month)`

---

## contribution_edits
| Column | Type | Constraints |
|---|---|---|
| id | bigint | PK, AUTO |
| contribution_id | bigint | FK → contributions.id |
| edited_by | bigint | FK → users.id |
| old_amount | decimal(15,2) | NOT NULL |
| new_amount | decimal(15,2) | NOT NULL |
| reason | text | NOT NULL |
| created_at | timestamp | AUTO (immutable) |

**Note:** Append-only. No UPDATE or DELETE allowed.

---

## loans
| Column | Type | Constraints |
|---|---|---|
| id | bigint | PK, AUTO |
| member_id | bigint | FK → members.id |
| reviewed_by | bigint | FK → users.id, NULLABLE |
| amount | decimal(15,2) | NOT NULL, CHECK(amount > 0) |
| approved_amount | decimal(15,2) | NULLABLE, CHECK(approved_amount <= amount) |
| balance_remaining | decimal(15,2) | NULLABLE, CHECK(balance_remaining >= 0) |
| purpose | text | NULLABLE |
| due_date | date | NOT NULL |
| status | enum('PENDING','APPROVED','OUTSTANDING','REJECTED','CLOSED') | NOT NULL, DEFAULT 'PENDING' |
| rejection_reason | text | NULLABLE |
| applied_at | timestamp | DEFAULT now() |
| reviewed_at | timestamp | NULLABLE |
| updated_at | timestamp | AUTO |

**Constraints:**
- `CHECK(amount > 0)`
- `CHECK(approved_amount <= amount)`
- `CHECK(balance_remaining >= 0)`
- `INDEX(member_id, status)`
- `INDEX(status)`
- `INDEX(due_date)`

**Status flow:**
```
PENDING → APPROVED → OUTSTANDING → CLOSED
PENDING → REJECTED
```

---

## repayments
| Column | Type | Constraints |
|---|---|---|
| id | bigint | PK, AUTO |
| loan_id | bigint | FK → loans.id |
| member_id | bigint | FK → members.id |
| recorded_by | bigint | FK → users.id |
| amount | decimal(15,2) | NOT NULL, CHECK(amount > 0) |
| balance_after | decimal(15,2) | NOT NULL, CHECK(balance_after >= 0) |
| paid_at | date | NOT NULL |
| notes | text | NULLABLE |
| created_at | timestamp | AUTO |

**Constraints:**
- `CHECK(amount > 0)`
- `CHECK(balance_after >= 0)`
- `INDEX(loan_id)`
- `INDEX(member_id)`

---

## audit_logs
| Column | Type | Constraints |
|---|---|---|
| id | bigint | PK, AUTO |
| user_id | bigint | FK → users.id, NULLABLE |
| action | enum('CREATE','UPDATE','DELETE','LOGIN','LOGOUT','APPROVE','REJECT') | NOT NULL |
| table_name | varchar(50) | NOT NULL |
| record_id | bigint | NULLABLE |
| old_values | json | NULLABLE |
| new_values | json | NULLABLE |
| ip_address | varchar(45) | NULLABLE |
| user_agent | text | NULLABLE |
| created_at | timestamp | AUTO (immutable) |

**Note:** Append-only. No UPDATE or DELETE allowed.

**Indexes:**
- `INDEX(user_id)`
- `INDEX(table_name, record_id)`
- `INDEX(created_at)`

---

## user_sessions
| Column | Type | Constraints |
|---|---|---|
| id | bigint | PK, AUTO |
| user_id | bigint | FK → users.id |
| token_hash | varchar(64) | NOT NULL, UNIQUE (sha256) |
| ip_address | varchar(45) | NOT NULL |
| user_agent | text | NULLABLE |
| last_active_at | timestamp | NOT NULL |
| expires_at | timestamp | NOT NULL |
| revoked_at | timestamp | NULLABLE |
| created_at | timestamp | AUTO |

**Indexes:**
- `INDEX(user_id)`
- `INDEX(token_hash)`
- `INDEX(expires_at)`

---

## failed_logins
| Column | Type | Constraints |
|---|---|---|
| id | bigint | PK, AUTO |
| email_attempted | varchar(150) | NOT NULL |
| ip_address | varchar(45) | NOT NULL |
| user_agent | text | NULLABLE |
| attempted_at | timestamp | NOT NULL |

**Note:** Auto-lock after 5 attempts within 15 minutes.

**Indexes:**
- `INDEX(email_attempted)`
- `INDEX(ip_address)`
- `INDEX(attempted_at)`

---

## notifications
| Column | Type | Constraints |
|---|---|---|
| id | bigint | PK, AUTO |
| user_id | bigint | FK → users.id |
| type | enum('LOAN_REQUEST','LOAN_APPROVED','LOAN_REJECTED','REPAYMENT','CONTRIBUTION','SYSTEM') | NOT NULL |
| title | varchar(200) | NOT NULL |
| message | text | NOT NULL |
| data | json | NULLABLE |
| read_at | timestamp | NULLABLE |
| created_at | timestamp | AUTO |

**Indexes:**
- `INDEX(user_id, read_at)`

---

## Relationships Summary

| From | To | Type | Via |
|---|---|---|---|
| members | contributions | 1 → N | member_id |
| members | loans | 1 → N | member_id |
| members | repayments | 1 → N | member_id |
| loans | repayments | 1 → N | loan_id |
| contributions | contribution_edits | 1 → N | contribution_id |
| users | audit_logs | 1 → N | user_id |
| users | user_sessions | 1 → N | user_id |
| users | notifications | 1 → N | user_id |
| users | loans | 1 → N | reviewed_by |
| users | members | 1 → N | registered_by |
| users | contributions | 1 → N | recorded_by |
| users | repayments | 1 → N | recorded_by |
