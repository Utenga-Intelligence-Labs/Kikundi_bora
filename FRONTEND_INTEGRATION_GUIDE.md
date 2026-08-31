# Kikundibora Frontend Integration Guide
## Role-Scoped Dashboard Endpoints

This guide helps the frontend team integrate the 5 new dashboard endpoints into the React/TanStack application.

---

## Quick Reference

| Endpoint | Purpose | Access | Response Type |
|----------|---------|--------|----------------|
| `GET /api/v1/members/{id}/dashboard-summary` | Personal savings view | Self/admin/leader | MemberDashboardSummary |
| `GET /api/v1/groups/{id}/dashboard-summary` | Group-wide leadership view | Leadership/admin | GroupDashboardSummary |
| `GET /api/v1/groups/{id}/dashboard-summary/katibu` | Secretary insights | Katibu role | KatibuDashboardSummary |
| `GET /api/v1/groups/{id}/dashboard-summary/mweka-hazina` | Treasurer cash-flow | Hazina role | HazinaDashboardSummary |
| `GET /api/v1/users/{id}/roles` | User roles (role-switch) | Self/admin/leader | UserRolesResponse |

---

## Base URL
```
http://localhost:3000/api/v1
```
(Update for production environment)

---

## Authentication
All endpoints require Bearer token authentication:

```typescript
const response = await fetch(`/api/v1/members/${memberId}/dashboard-summary`, {
  headers: {
    "Authorization": `Bearer ${authToken}`
  }
});
```

---

## 1. Member Personal Dashboard

### Fetch Member Summary
```typescript
async function getMemberDashboard(memberId: string, token: string) {
  const response = await fetch(
    `/api/v1/members/${memberId}/dashboard-summary`,
    {
      headers: { "Authorization": `Bearer ${token}` }
    }
  );
  
  if (!response.ok) {
    if (response.status === 404) throw new Error("Member not found");
    if (response.status === 403) throw new Error("Access denied");
  }
  
  const { data } = await response.json();
  return data as MemberDashboardSummary;
}
```

### Display Logic
```typescript
// Member Card Component
<div>
  <h2>{member.full_name} ({member.member_no})</h2>
  
  {/* Savings Section */}
  <div>
    <h3>Akiba Yangu</h3>
    <p className="amount">{formatMoney(summary.total_contributions)} TZS</p>
    <small>{summary.contributions_count} michango</small>
  </div>
  
  {/* Welfare Section (optional, separate) */}
  {summary.welfare_contributions_total > 0 && (
    <div>
      <h3>Mfuko wa Kijamii</h3>
      <p>{formatMoney(summary.welfare_contributions_total)} TZS</p>
    </div>
  )}
  
  {/* Pending Submissions */}
  {summary.pending_contributions_count > 0 && (
    <Alert type="info">
      {summary.pending_contributions_count} michango yangu inasubiri uthibitisho
    </Alert>
  )}
  {summary.rejected_contributions_count > 0 && (
    <Alert type="warning">
      {summary.rejected_contributions_count} michango yalikataliwa
    </Alert>
  )}
  
  {/* Loans Section */}
  <div>
    <h3>Mikopo</h3>
    {summary.outstanding_loans_count > 0 && (
      <p>Deni: {formatMoney(summary.outstanding_loans_balance)} TZS</p>
    )}
    {summary.closed_loans_count > 0 && (
      <p>Mikopo iliyofungwa: {summary.closed_loans_count}</p>
    )}
  </div>
  
  {/* Recent Contributions */}
  <div>
    <h3>Historia ya Michango (10 Mipango)</h3>
    {summary.recent_contributions.map(contrib => (
      <div key={contrib.id}>
        <p>
          {contrib.amount} TZS — {contrib.period_label}
        </p>
        <small>
          {contrib.source === "member_contribution" ? "Nilijaza" : "Nimepokea"} 
          • {contrib.status}
        </small>
      </div>
    ))}
  </div>
</div>
```

### TypeScript Types
```typescript
interface MemberDashboardSummary {
  member_id: string;
  member_no: string;
  full_name: string;
  total_contributions: string;  // Decimal as string
  contributions_count: number;
  welfare_contributions_total: string;
  welfare_contributions_count: number;
  pending_contributions_count: number;
  rejected_contributions_count: number;
  outstanding_loans_count: number;
  outstanding_loans_balance: string;
  closed_loans_count: number;
  recent_contributions: RecentContribution[];
}

interface RecentContribution {
  id: string;
  source: "contribution" | "member_contribution";
  contribution_type: "AKIBA" | "MFUKO_WA_KIJAMII";
  period_label: string;
  amount: string;
  status: "PAID" | "CONFIRMED" | "PENDING_VERIFICATION" | "REJECTED";
  paid_at?: string;
  created_at: string;
}
```

---

## 2. Group Leadership Dashboard

### Fetch Group Summary
```typescript
async function getGroupDashboard(groupId: string, token: string) {
  const response = await fetch(
    `/api/v1/groups/${groupId}/dashboard-summary`,
    {
      headers: { "Authorization": `Bearer ${token}` }
    }
  );
  
  if (!response.ok) {
    if (response.status === 403) throw new Error("Leadership access required");
  }
  
  const { data } = await response.json();
  return data as GroupDashboardSummary;
}
```

### Display "Uongozi" View
```typescript
// Leadership Dashboard Component
<div className="leadership-dashboard">
  <h2>Muhtasari wa {group.group_name}</h2>
  
  {/* Key Metrics */}
  <div className="grid">
    <Card>
      <h4>Wanachama Wenye Hazina</h4>
      <p className="big">{summary.total_active_members}</p>
    </Card>
    
    <Card>
      <h4>Salio la Kikundi</h4>
      <p className="big amount">{formatMoney(summary.available_balance)} TZS</p>
      <small>
        Michango: {formatMoney(summary.total_contributions)}
        + Malipo: {formatMoney(summary.total_repayments)}
        - Zimepewa: {formatMoney(summary.total_disbursed)}
      </small>
    </Card>
    
    <Card>
      <h4>Michango Hii Mwezi</h4>
      <p className="big">{formatMoney(summary.contributions_this_period)} TZS</p>
    </Card>
    
    <Card>
      <h4>Deni (Mikopo Inayolipwa)</h4>
      <p className="big">{summary.outstanding_loans_count}</p>
      <small>{formatMoney(summary.outstanding_loans_balance)} TZS</small>
    </Card>
  </div>
  
  {/* Period Info */}
  <div className="period-info">
    <p>Kipindi: {summary.contribution_interval}</p>
    {summary.next_due_date && (
      <p>Tarehe ya Mpokeaji: {summary.next_due_date}</p>
    )}
  </div>
  
  {/* Pending Items Alert */}
  {summary.pending_contributions_count > 0 && (
    <Alert type="warning">
      {summary.pending_contributions_count} michango inasubiri uthibitisho
    </Alert>
  )}
  {summary.pending_loans_count > 0 && (
    <Alert type="info">
      {summary.pending_loans_count} mikopo inasubiri usambazaji
    </Alert>
  )}
</div>
```

### TypeScript Types
```typescript
interface GroupDashboardSummary {
  group_id: string;
  group_name: string;
  total_active_members: number;
  total_contributions: string;
  total_repayments: string;
  total_disbursed: string;
  available_balance: string;
  outstanding_loans_count: number;
  outstanding_loans_balance: string;
  pending_loans_count: number;
  pending_contributions_count: number;
  contributions_this_period: string;
  contribution_interval: string;
  next_due_date?: string;
}
```

---

## 3. Secretary (Katibu) Dashboard

### Fetch Secretary Summary
```typescript
async function getKatibuDashboard(groupId: string, token: string) {
  const response = await fetch(
    `/api/v1/groups/${groupId}/dashboard-summary/katibu`,
    {
      headers: { "Authorization": `Bearer ${token}` }
    }
  );
  
  if (!response.ok) {
    if (response.status === 403) throw new Error("Secretary access required");
  }
  
  const { data } = await response.json();
  return data as KatibuDashboardSummary;
}
```

### Display Secretary View
```typescript
// Secretary Dashboard
<div>
  <h2>Muhtasari wa Katibu</h2>
  
  {/* Membership Activity */}
  <Card>
    <h3>Harakati ya Wanachama</h3>
    <p>Wajanja Wakati wa Mwezi: {summary.members_joined_this_month}</p>
    <p>Waliondoka: {summary.members_left_this_month}</p>
    <p>Jumla Aktibishi: {summary.total_active_members}</p>
  </Card>
  
  {/* Pending Approvals */}
  {summary.pending_user_approvals > 0 && (
    <Alert type="warning">
      {summary.pending_user_approvals} watumiaji wanasuhiriaji usambazaji
    </Alert>
  )}
  
  {/* Records */}
  <Card>
    <h3>Rekodi</h3>
    <p>Tangazo la Mwezi: {summary.announcements_this_month}</p>
  </Card>
  
  {/* Late Payments */}
  {summary.late_payments_count > 0 && (
    <Card type="warning">
      <h3>Waliochelewa ({summary.late_payments_count})</h3>
      <table>
        <thead>
          <tr>
            <th>Jina</th>
            <th>Simu</th>
            <th>Inatarajiwa</th>
          </tr>
        </thead>
        <tbody>
          {summary.late_payments.map(payment => (
            <tr key={payment.member_id}>
              <td>{payment.full_name} ({payment.member_no})</td>
              <td>{payment.phone}</td>
              <td>
                {payment.expected_amount 
                  ? formatMoney(payment.expected_amount) + " TZS"
                  : "Haijaelezwa"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  )}
  
  {/* Pending Contributions */}
  {summary.pending_contributions_count > 0 && (
    <Alert type="info">
      {summary.pending_contributions_count} michango inasubiri uthibitisho
    </Alert>
  )}
</div>
```

### TypeScript Types
```typescript
interface KatibuDashboardSummary {
  group_id: string;
  total_active_members: number;
  members_joined_this_month: number;
  members_left_this_month: number;
  pending_user_approvals: number;
  announcements_this_month: number;
  pending_contributions_count: number;
  current_period_label: string;
  next_due_date?: string;
  late_payments_count: number;
  late_payments: LatePaymentRow[];
}

interface LatePaymentRow {
  member_id: string;
  member_no: string;
  full_name: string;
  phone: string;
  period_label: string;
  expected_amount?: string;
}
```

---

## 4. Treasurer (Mweka Hazina) Dashboard

### Fetch Treasurer Summary
```typescript
async function getHazinaDashboard(groupId: string, token: string) {
  const response = await fetch(
    `/api/v1/groups/${groupId}/dashboard-summary/mweka-hazina`,
    {
      headers: { "Authorization": `Bearer ${token}` }
    }
  );
  
  if (!response.ok) {
    if (response.status === 403) throw new Error("Treasurer access required");
  }
  
  const { data } = await response.json();
  return data as HazinaDashboardSummary;
}
```

### Display Treasurer Cash-Flow View
```typescript
// Treasurer Dashboard
<div>
  <h2>Muhtasari wa Hazina</h2>
  
  {/* Available Balance (Salio) */}
  <Card className="prominent">
    <h3>Salio Halisi</h3>
    <p className="huge">{formatMoney(summary.available_balance)} TZS</p>
  </Card>
  
  {/* Cash In */}
  <Card>
    <h3>Pesa Zilinopokea</h3>
    <div>
      <p>
        <strong>Zilizo Thibitishwa:</strong>
        {formatMoney(summary.cash_in_confirmed)} TZS
      </p>
      <small>
        Michango yote iliyolipwa + iliyothibitishwa
      </small>
    </div>
    
    {summary.cash_in_pending > 0 && (
      <Alert type="info">
        <strong>Inassubiri Uthibitisho:</strong>
        {formatMoney(summary.cash_in_pending)} TZS
        ({summary.cash_in_pending_count} michango)
      </Alert>
    )}
    
    <div>
      <p>
        <strong>Hii Mwezi:</strong>
        {formatMoney(summary.cash_in_this_period)} TZS
      </p>
      {summary.expected_this_period && (
        <small>
          Inatarajiwa: {formatMoney(summary.expected_this_period)} TZS
        </small>
      )}
    </div>
  </Card>
  
  {/* Repayments */}
  <Card>
    <h3>Malipo ya Mikopo</h3>
    <p>Jumla: {formatMoney(summary.repayments_total)} TZS</p>
    <p>Mwezi Huu: {formatMoney(summary.repayments_this_month)} TZS</p>
  </Card>
  
  {/* Disbursements */}
  <Card>
    <h3>Zimepewa (Mikopo)</h3>
    <p>Jumla: {formatMoney(summary.disbursements_total)} TZS</p>
    <p>Idadi: {summary.disbursements_count}</p>
    
    {summary.recent_disbursements.length > 0 && (
      <div>
        <h4>Zinazosika 10</h4>
        <table>
          <thead>
            <tr>
              <th>Jina</th>
              <th>Kiasi</th>
              <th>Tarehe</th>
              <th>Hali</th>
            </tr>
          </thead>
          <tbody>
            {summary.recent_disbursements.map(d => (
              <tr key={d.loan_id}>
                <td>{d.full_name} ({d.member_no})</td>
                <td>{formatMoney(d.amount)} TZS</td>
                <td>{d.disbursed_at}</td>
                <td>{d.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )}
  </Card>
</div>
```

### TypeScript Types
```typescript
interface HazinaDashboardSummary {
  group_id: string;
  cash_in_confirmed: string;
  cash_in_pending: string;
  cash_in_pending_count: number;
  cash_in_this_period: string;
  expected_this_period?: string;
  repayments_total: string;
  repayments_this_month: string;
  disbursements_total: string;
  disbursements_count: number;
  recent_disbursements: DisbursementRow[];
  available_balance: string;
}

interface DisbursementRow {
  loan_id: string;
  member_no: string;
  full_name: string;
  amount: string;
  status: string;
  disbursed_at: string;
}
```

---

## 5. Role-Switch Toggle

### Fetch User Roles
```typescript
async function getUserRoles(userId: string, token: string) {
  const response = await fetch(
    `/api/v1/users/${userId}/roles`,
    {
      headers: { "Authorization": `Bearer ${token}` }
    }
  );
  
  const { data } = await response.json();
  return data as UserRolesResponse;
}
```

### Implement Role-Switch Logic
```typescript
interface UserRolesResponse {
  user_id: string;
  member_id?: string;
  primary_role: string;
  leadership_positions: string[];
  roles: string[];
}

// Component: Role Switch Toggle
function RoleSwitcher({ userId, token }: Props) {
  const [roles, setRoles] = useState<UserRolesResponse | null>(null);
  const [currentRole, setCurrentRole] = useState<string>("");
  
  useEffect(() => {
    getUserRoles(userId, token).then(setRoles);
  }, [userId, token]);
  
  // Only show toggle if user has multiple roles
  if (!roles || roles.roles.length <= 1) {
    return <div>Jukumu: {roles?.roles[0] || roles?.primary_role}</div>;
  }
  
  // Map Swahili names to Kiswahili labels
  const roleLabels: Record<string, string> = {
    "mwenyekiti": "Mwenyekiti (Rais)",
    "katibu": "Katibu (Sekretari)",
    "mweka-hazina": "Mweka Hazina (Hazina)",
    "mwanachama": "Mwanachama (Member)",
    "msimamizi": "Msimamizi (Admin)"
  };
  
  return (
    <div>
      <label>Jukumu:</label>
      <select value={currentRole} onChange={(e) => setCurrentRole(e.target.value)}>
        {roles.roles.map(role => (
          <option key={role} value={role}>
            {roleLabels[role] || role}
          </option>
        ))}
      </select>
      <p>Jukumu lile lako: {roleLabels[currentRole] || currentRole}</p>
    </div>
  );
}

// Use it
<RoleSwitcher userId={authUser.id} token={authToken} />
```

### TypeScript Types
```typescript
interface UserRolesResponse {
  user_id: string;
  member_id: string | null;
  primary_role: string;
  leadership_positions: string[];  // ["MWENYEKITI", "KATIBU", ...]
  roles: string[];  // ["mwenyekiti", "mwanachama", ...]
}
```

---

## Error Handling

All endpoints return structured error responses:

```typescript
async function handleDashboardError(error: any) {
  if (error.status === 404) {
    return "Hakuna data inayotafutwa";
  }
  if (error.status === 403) {
    return "Huna ruhusa ya kuona hii data";
  }
  if (error.status === 500) {
    return "Kosa la seva. Tafadhali jaribu tena.";
  }
  return "Kosa lilijitokeza";
}

// Usage
try {
  const summary = await getMemberDashboard(memberId, token);
} catch (error) {
  const message = await handleDashboardError(error);
  showNotification(message, "error");
}
```

---

## Utilities

### Format Money
```typescript
function formatMoney(amount: string | number): string {
  const num = typeof amount === "string" ? parseFloat(amount) : amount;
  return new Intl.NumberFormat("sw-TZ", {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2
  }).format(num);
}

// Usage
<p>{formatMoney(summary.total_contributions)} TZS</p>
```

### Parse Period Label
```typescript
function parsePeriodLabel(label: string): string {
  const [year, month] = label.split("-");
  const date = new Date(parseInt(year), parseInt(month) - 1, 1);
  return new Intl.DateTimeFormat("sw-TZ", {
    month: "long",
    year: "numeric"
  }).format(date);
}

// Usage
<p>{parsePeriodLabel(contribution.period_label)}</p>
```

---

## React Query / TanStack Query Integration

```typescript
// Query hooks
import { useQuery } from "@tanstack/react-query";

export function useMemberDashboard(memberId: string, token: string) {
  return useQuery({
    queryKey: ["member-dashboard", memberId],
    queryFn: () => getMemberDashboard(memberId, token),
    enabled: !!memberId && !!token
  });
}

export function useGroupDashboard(groupId: string, token: string) {
  return useQuery({
    queryKey: ["group-dashboard", groupId],
    queryFn: () => getGroupDashboard(groupId, token),
    enabled: !!groupId && !!token
  });
}

// Component usage
function MemberView({ memberId, token }: Props) {
  const { data: summary, isLoading, error } = useMemberDashboard(memberId, token);
  
  if (isLoading) return <Spinner />;
  if (error) return <ErrorAlert error={error} />;
  
  return <MemberDashboardCard summary={summary} />;
}
```

---

## Summary

✅ All 5 endpoints are ready for integration
✅ TypeScript types provided
✅ Response schemas documented
✅ Error handling examples included
✅ React Query integration pattern shown
✅ Swahili labels and translations provided

Start with the member dashboard, then add leadership/specialist dashboards based on user roles.

