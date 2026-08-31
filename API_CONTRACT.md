# Kikundibora API Contract - Role-Scoped Dashboard Endpoints

## Overview
This document specifies the 5 new/updated role-scoped dashboard endpoints. All endpoints:
- Use **snake_case** for JSON field names
- Return data in `{ "data": {...} }` envelope
- Support multi-role users (a member can be chair + member simultaneously)
- Are **multi-tenant safe** (single group per deployment)
- Require authentication (Bearer token in `Authorization: Bearer <token>` header)

---

## 1. Member Personal Dashboard

**Endpoint:**
```
GET /api/v1/members/{id}/dashboard-summary
```

**Access Control:**
- Self (the member themselves)
- Admin users
- Users with leadership positions (MWENYEKITI, KATIBU, HAZINA)
- Users with leadership roles (chair, secretary, treasurer)

**Response Schema:**
```json
{
  "data": {
    "member_id": "uuid",
    "member_no": "string",
    "full_name": "string",
    "total_contributions": "decimal",
    "contributions_count": "int64",
    "welfare_contributions_total": "decimal",
    "welfare_contributions_count": "int64",
    "pending_contributions_count": "int64",
    "rejected_contributions_count": "int64",
    "outstanding_loans_count": "int64",
    "outstanding_loans_balance": "decimal",
    "closed_loans_count": "int64",
    "recent_contributions": [
      {
        "id": "uuid",
        "source": "contribution|member_contribution",
        "contribution_type": "AKIBA|MFUKO_WA_KIJAMII",
        "period_label": "YYYY-MM",
        "amount": "decimal",
        "status": "PAID|CONFIRMED|PENDING_VERIFICATION|REJECTED",
        "paid_at": "YYYY-MM-DD",
        "created_at": "RFC3339"
      }
    ]
  }
}
```

**Key Features:**
- Aggregates contributions from **TWO sources**: 
  - `contributions` table (treasurer-recorded via "Pokea Mchango")
  - `member_contributions` table (member self-submitted via "Weka Mchango", status=CONFIRMED)
- Shows breakdown: AKIBA vs MFUKO_WA_KIJAMII (welfare)
- Tracks pending and rejected self-submissions separately
- Loan summaries: outstanding balance + count, closed loans count
- Recent contributions sorted by date (newest first, max 10)

**Example Usage:**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:3000/api/v1/members/abc-123/dashboard-summary
```

---

## 2. Group-Wide Dashboard (Leadership/Admin View)

**Endpoint:**
```
GET /api/v1/groups/{id}/dashboard-summary
```

**Access Control:**
- Leadership positions: MWENYEKITI, KATIBU, HAZINA (via middleware.RequireLeadership)
- Admin users (automatic pass)

**Response Schema:**
```json
{
  "data": {
    "group_id": "uuid",
    "group_name": "string",
    "total_active_members": "int64",
    "total_contributions": "decimal",
    "total_repayments": "decimal",
    "total_disbursed": "decimal",
    "available_balance": "decimal",
    "outstanding_loans_count": "int64",
    "outstanding_loans_balance": "decimal",
    "pending_loans_count": "int64",
    "pending_contributions_count": "int64",
    "contributions_this_period": "decimal",
    "contribution_interval": "weekly|monthly|semi_annual|yearly",
    "next_due_date": "YYYY-MM-DD"
  }
}
```

**Key Features:**
- **Available Balance** = total_contributions + total_repayments − total_disbursed
- Aggregates contributions from **BOTH stores** (treasurer + self-submitted CONFIRMED)
- Includes pending verification count (awaiting review)
- Period-specific metrics (current month/cycle)
- Contribution settings (interval, next due date)

---

## 3. Secretary (Katibu) Dashboard

**Endpoint:**
```
GET /api/v1/groups/{id}/dashboard-summary/katibu
```

**Access Control:**
- KATIBU leadership position only (via middleware.RequireLeadership)

**Response Schema:**
```json
{
  "data": {
    "group_id": "uuid",
    "total_active_members": "int64",
    "members_joined_this_month": "int64",
    "members_left_this_month": "int64",
    "pending_user_approvals": "int64",
    "announcements_this_month": "int64",
    "pending_contributions_count": "int64",
    "current_period_label": "YYYY-MM",
    "next_due_date": "YYYY-MM-DD",
    "late_payments_count": "int64",
    "late_payments": [
      {
        "member_id": "uuid",
        "member_no": "string",
        "full_name": "string",
        "phone": "string",
        "period_label": "YYYY-MM",
        "expected_amount": "decimal"
      }
    ]
  }
}
```

**Key Features:**
- Membership tracking: joins, departures, deactivations this month
- Pending user approvals (new members waiting for secretary approval)
- Announcements activity (announcements made this month)
- **Late payments list**: Active members who have NOT contributed (AKIBA only) in the current period
  - Checked in both stores: no PAID in `contributions` AND no CONFIRMED AKIBA in `member_contributions`
- Expected amount from group settings (if fixed contribution is configured)

---

## 4. Treasurer (Mweka Hazina) Dashboard

**Endpoint:**
```
GET /api/v1/groups/{id}/dashboard-summary/mweka-hazina
```

**Access Control:**
- HAZINA leadership position only (via middleware.RequireLeadership)

**Response Schema:**
```json
{
  "data": {
    "group_id": "uuid",
    "cash_in_confirmed": "decimal",
    "cash_in_pending": "decimal",
    "cash_in_pending_count": "int64",
    "cash_in_this_period": "decimal",
    "expected_this_period": "decimal|null",
    "repayments_total": "decimal",
    "repayments_this_month": "decimal",
    "disbursements_total": "decimal",
    "disbursements_count": "int64",
    "recent_disbursements": [
      {
        "loan_id": "uuid",
        "member_no": "string",
        "full_name": "string",
        "amount": "decimal",
        "status": "OUTSTANDING|CLOSED",
        "disbursed_at": "YYYY-MM-DD"
      }
    ],
    "available_balance": "decimal"
  }
}
```

**Key Features:**
- **Cash flow dashboard**:
  - `cash_in_confirmed`: all PAID contributions + CONFIRMED self-submissions (both stores)
  - `cash_in_pending`: contributions awaiting verification (status=PENDING_VERIFICATION)
  - `cash_in_this_period`: contributions for current cycle only
  - `expected_this_period`: fixed_amount × active_members (null if not configured)
- Loan disbursements: total out, recent 10 (ordered by disbursed_at DESC)
- **Available Balance** = cash_in_confirmed + repayments_total − disbursements_total
- Period-scoped repayments this month

---

## 5. User Roles List

**Endpoint:**
```
GET /api/v1/users/{id}/roles
```

**Access Control:**
- Self (the user whose roles are being fetched)
- Admin users
- Users with leadership positions or roles

**Response Schema:**
```json
{
  "data": {
    "user_id": "uuid",
    "member_id": "uuid|null",
    "primary_role": "chair|secretary|treasurer|member|admin",
    "leadership_positions": [
      "MWENYEKITI|KATIBU|HAZINA"
    ],
    "roles": [
      "mwenyekiti|katibu|mweka-hazina|mwanachama|msimamizi"
    ]
  }
}
```

**Key Features:**
- Returns **Swahili role names** for frontend role-switch toggle:
  - `mwenyekiti` = chair
  - `katibu` = secretary
  - `mweka-hazina` = treasurer
  - `mwanachama` = member (implicit for linked members)
  - `msimamizi` = admin
- **Role precedence** (order in `roles` array):
  1. Leadership positions (MWENYEKITI, KATIBU, HAZINA) — if any
  2. Primary user role (chair/secretary/treasurer/member/admin)
  3. Implicit member role (if user has a linked member row)
- `member_id` is null for users without a linked member (e.g., admins)
- Enables frontend to display role-switch toggle when `roles.length > 1`

**Example (multi-role user):**
```json
{
  "data": {
    "user_id": "asha-uuid",
    "member_id": "member-uuid-123",
    "primary_role": "chair",
    "leadership_positions": ["MWENYEKITI", "KATIBU"],
    "roles": ["mwenyekiti", "katibu", "mwanachama"]
  }
}
```

---

## Error Responses

All endpoints return standard error envelopes:

**404 Not Found:**
```json
{
  "message": "Mwanachama hajapatikana" | "Kikundi hakijapatikana" | "Mtumiaji hajapatikana"
}
```

**403 Forbidden:**
```json
{
  "message": "Huna ruhusa ya kuona data ya mwanachama huu" | "Huna ruhusa ya kuona majukumu ya mtumiaji huyu"
}
```

**500 Internal Server Error:**
```json
{
  "message": "Imeshindikana kupata hesabu za michango"
}
```

---

## Testing Notes

### Regression Test (Bug Fix)
The bug where member contributions weren't visible in the dashboard summary is covered by:
```
TestMemberDashboardSummaryShowsSelfSubmittedContribution
```
This test verifies:
1. Member submits a contribution (PENDING_VERIFICATION)
2. Contribution does NOT count toward akiba yet
3. Treasurer confirms (→ CONFIRMED)
4. Contribution is visible IMMEDIATELY in the member summary
5. Recent contributions list includes the entry

### Multi-Role Test
```
TestUserRolesMultiRole
```
Verifies a member holding multiple leadership positions (e.g., chair + secretary) correctly returns all roles.

---

## Database Schema References

**Contributions (two tables):**
- `contributions` — treasurer-recorded (legacy), status='PAID'
- `member_contributions` — member self-submitted, status ∈ {PENDING_VERIFICATION, CONFIRMED, REJECTED}

**Leadership:**
- `leadership_positions(member_id, role, is_current)` — current positions per member

**Users:**
- `users(id, role, status)` — primary role + approval status
- `user_positions(user_id, position_type, is_active)` — historical tracking (deprecated, use LeadershipPosition)

---

## Migration / Rollout Notes

1. **No schema changes** — all endpoints read from existing tables
2. **Dashboard handlers** are stateless; responses computed on-demand
3. **Access control** via existing middleware (RequireLeadership, RequireRoles, GetUserID)
4. **Backward compatible** — new endpoints do not affect existing API

