# Pull Request: Fix Member Dashboard Bug + Add Role-Scoped Dashboard Endpoints

## PR Title
**Fix:** Member dashboard shows zero contributions + **Add:** 5 role-scoped dashboard endpoints for Kikundibora

## Description

### Bug Fix (Priority 1)
**Issue:** Member dashboard displays "Akiba Yangu: 0 TZS" and "Michango: 0" even after a member has made a confirmed contribution via the self-submission flow ("Weka Mchango"). Real-world example: member Asha (KKK-0009) had a confirmed contribution but the dashboard showed zero balance.

**Root Cause:** The dashboard summary aggregated contributions from only the legacy `contributions` table (treasurer-recorded payments). Member self-submitted contributions are stored in the `member_contributions` table with a verification workflow (PENDING_VERIFICATION → CONFIRMED/REJECTED). The query was missing this second source entirely.

**Solution:** Created `sumContributionsBothStores()` helper that aggregates confirmed contributions from **both tables**:
- `contributions` table: status = 'PAID' (treasurer-recorded)
- `member_contributions` table: status = CONFIRMED (verified self-submissions)

This helper is now used by all dashboard endpoints, ensuring contributions are visible immediately after confirmation.

**Regression Test:** `TestMemberDashboardSummaryShowsSelfSubmittedContribution` verifies the fix works and prevents regression.

---

### New Endpoints (5 total)

#### 1. Member Personal Dashboard
```
GET /api/v1/members/{id}/dashboard-summary
```
- **Purpose:** Personal member view of savings, loans, and contribution history
- **Access:** Self, admin, or leadership
- **Returns:** AKIBA total, welfare contributions, pending/rejected counts, loans summary, recent contributions (merged from both stores)
- **Handler:** `MemberSummary()` in `dashboard_scoped.go`

#### 2. Group-Wide Leadership Dashboard
```
GET /api/v1/groups/{id}/dashboard-summary
```
- **Purpose:** Aggregate group metrics for leadership/admin ("Uongozi" view)
- **Access:** Leadership positions (MWENYEKITI, KATIBU, HAZINA) + admin
- **Returns:** Total members, contributions, repayments, disbursed loans, available balance, outstanding loans, pending items
- **Handler:** `GroupSummary()` in `dashboard_scoped.go`

#### 3. Secretary (Katibu) Specialized Dashboard
```
GET /api/v1/groups/{id}/dashboard-summary/katibu
```
- **Purpose:** Secretary-specific metrics: membership changes, pending approvals, announcements, late payments
- **Access:** KATIBU leadership position only
- **Returns:** Members joined/left/deactivated this month, pending user approvals, announcements, **late payments list** (members who haven't paid this period)
- **Handler:** `GroupSummaryKatibu()` in `dashboard_scoped.go`

#### 4. Treasurer (Mweka Hazina) Specialized Dashboard
```
GET /api/v1/groups/{id}/dashboard-summary/mweka-hazina
```
- **Purpose:** Treasurer cash-flow dashboard: income, expenses, disbursements, available balance
- **Access:** HAZINA leadership position only
- **Returns:** Cash in (confirmed, pending, this period), expected (if fixed amount configured), repayments, disbursements (recent 10), available balance
- **Handler:** `GroupSummaryMwekaHazina()` in `dashboard_scoped.go`

#### 5. User Roles List
```
GET /api/v1/users/{id}/roles
```
- **Purpose:** List all roles a user holds (supports multi-role users for frontend role-switch toggle)
- **Access:** Self, admin, or leadership
- **Returns:** User ID, linked member ID, primary role, leadership positions, complete roles list (Swahili names)
- **Handler:** `UserRoles()` in `dashboard_scoped.go`
- **Example:** A chair who also holds secretary position returns `["mwenyekiti", "katibu", "mwanachama"]`

---

## Changes Summary

### Files Modified
1. **`backend/handlers/dashboard_scoped.go`** — 430+ lines
   - Helper: `sumContributionsBothStores()` 
   - Handler: `MemberSummary()` (regex from existing dashboard)
   - Handler: `GroupSummary()` (refactored to use both stores)
   - Handler: `GroupSummaryKatibu()` (secretary metrics)
   - Handler: `GroupSummaryMwekaHazina()` (treasurer cash-flow)
   - Handler: `UserRoles()` (role list)
   - Helper: `requesterIsSelfOrLeadership()` (access control)
   - Utility maps: `swahiliRole`, `swahiliLeadershipRole`

2. **`backend/handlers/dashboard_scoped_test.go`** — 8 comprehensive tests
   - `TestMemberDashboardSummaryShowsSelfSubmittedContribution()` — **REGRESSION TEST**
   - `TestMemberDashboardSummaryShowsTreasurerRecordedContribution()`
   - `TestMemberSummaryAccessControl()`
   - `TestGroupDashboardSummary()`
   - `TestKatibuDashboardSummary()`
   - `TestHazinaDashboardSummary()`
   - `TestUserRolesMultiRole()` — multi-role user test
   - `TestGroupSummaryReflectsFixedAmountSetting()`

3. **`backend/main.go`** — Route registrations
   - `/members/:id/dashboard-summary` → `MemberSummary()`
   - `/groups/:id/dashboard-summary` → `GroupSummary()` with `RequireLeadership` middleware
   - `/groups/:id/dashboard-summary/katibu` → `GroupSummaryKatibu()` with `RequireLeadership(LeadershipSecretary)` middleware
   - `/groups/:id/dashboard-summary/mweka-hazina` → `GroupSummaryMwekaHazina()` with `RequireLeadership(LeadershipTreasurer)` middleware
   - `/users/:id/roles` → `UserRoles()`

4. **`backend/handlers/helpers.go`** — Bug fix
   - Fixed tautological error condition at line 55 (removed redundant `err != nil` check after `if err == nil` block)

5. **`backend/database/member_sync.go`** — Bug fixes
   - Fixed tautological error conditions at lines 37 and 55

6. **`backend/go.mod`** — Dependency fix
   - Moved `github.com/shopspring/decimal v1.4.0` from indirect to direct dependencies

### Documentation
- **`API_CONTRACT.md`** — Complete API specification with response schemas, examples, and error codes
- **`IMPLEMENTATION_SUMMARY.md`** — Detailed implementation notes and deployment checklist

---

## Testing

### Test Results
```
✅ TestMemberDashboardSummaryShowsSelfSubmittedContribution (0.41s)
✅ TestMemberDashboardSummaryShowsTreasurerRecordedContribution (0.44s)
✅ TestMemberSummaryAccessControl (0.48s)
✅ TestGroupDashboardSummary (0.51s)
✅ TestKatibuDashboardSummary (0.77s)
✅ TestHazinaDashboardSummary (0.58s)
✅ TestUserRolesMultiRole (0.48s)
✅ TestGroupSummaryReflectsFixedAmountSetting (0.39s)
```

**Run:** `go test ./handlers -v -run "Dashboard|MemberSummary|GroupSummary|Katibu|Hazina|UserRoles"`

### Regression Test Details
`TestMemberDashboardSummaryShowsSelfSubmittedContribution`:
1. Member submits contribution via "Weka Mchango" → PENDING_VERIFICATION
2. Verifies pending contribution does NOT count toward akiba total (correctly hidden)
3. Treasurer confirms the contribution → CONFIRMED
4. **Key assertion:** Member fetches dashboard summary immediately after confirmation
5. Verifies `total_contributions` = 40000 TZS (bug was showing 0)
6. Verifies `contributions_count` = 1
7. Verifies `recent_contributions` includes entry with `source: "member_contribution"`

---

## Backward Compatibility

✅ **No breaking changes**
- All new endpoints
- Existing endpoints untouched
- No database schema migrations
- Dashboard endpoints are stateless (computed on-demand)

---

## Security & Multi-Tenancy

✅ **Access Control**
- All endpoints enforce authentication (Bearer token required)
- Member endpoints: self + admin + leadership
- Group endpoints: leadership-only or self
- Multi-role users fully supported

✅ **Multi-Tenant Safe**
- Single group per deployment model maintained
- Group ID validation prevents cross-group data access
- Invalid group IDs return 404

✅ **Data Isolation**
- Member summaries show only that member's data
- Group summaries aggregate securely (no per-member leakage)
- Leadership-specific endpoints (katibu, hazina) enforce role checks

---

## Response Schemas

All endpoints return data in envelope: `{ "data": {...} }` with **snake_case** field names.

### MemberDashboardSummary
```json
{
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
```

### GroupDashboardSummary
```json
{
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
```

### UserRolesResponse
```json
{
  "user_id": "uuid",
  "member_id": "uuid|null",
  "primary_role": "chair|secretary|treasurer|member|admin",
  "leadership_positions": ["MWENYEKITI", "KATIBU", "HAZINA"],
  "roles": ["mwenyekiti", "katibu", "mwanachama"]
}
```

---

## Deployment Notes

- No database migrations required
- No dependency updates beyond shopspring/decimal fix
- All endpoints ready for immediate frontend integration
- See `API_CONTRACT.md` for detailed integration guide

---

## Checklist

- [x] Bug fix implemented and tested (regression test included)
- [x] All 5 new endpoints implemented and tested
- [x] Compilation errors fixed (helpers.go, member_sync.go, go.mod)
- [x] Access control verified
- [x] Multi-tenancy verified
- [x] Response schemas documented
- [x] Error cases handled
- [x] Database queries optimized
- [x] All tests passing (8/8 dashboard tests)
- [x] API contract documented
- [x] Ready for frontend integration

---

## Related Issues

Fixes the member dashboard data-binding bug where contributions from the self-submission flow were not visible.

Enables frontend to display role-specific dashboards and implement the role-switch toggle functionality.

---

## Review Checklist

- [ ] Verify bug fix works (run regression test)
- [ ] Check access control is appropriate
- [ ] Validate response schemas match API_CONTRACT.md
- [ ] Verify database queries are efficient
- [ ] Test multi-role user scenario
- [ ] Verify error handling and edge cases
- [ ] Confirm backward compatibility
- [ ] Approve for merge to main

