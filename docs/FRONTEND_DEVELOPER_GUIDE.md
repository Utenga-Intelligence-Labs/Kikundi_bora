# Frontend Implementation - Quick Reference Guide

## For Developers

### Understanding the Role-Switch Flow

```
User with multiple roles (e.g., mwenyekiti + member)
    ↓
AppShell wraps with RoleSwitchProvider
    ↓
RoleSwitchToggle renders dropdown in sidebar
    ↓
User selects role from dropdown
    ↓
useRoleSwitch() updates currentRole state
    ↓
Dashibodi component re-renders with new role
    ↓
Role-specific view renders (ChairmanView, MemberView, etc.)
    ↓
View fetches data from role-scoped endpoint
    ↓
Dashboard cards display data (or loading/error state)
```

### Adding a New Role View

1. **Create view component** in `dashibodi.tsx`:
```typescript
function MyRoleView({ groupId }: { groupId?: string }) {
  const groupIdVal = groupId || "1";
  const { data, isLoading, error } = useMyDashboardSummary(groupIdVal);

  if (isLoading) return <LoadingSkeleton />;
  if (error || !data) return <ErrorAlert />;

  return <YourCardsHere />;
}
```

2. **Add routing** in Dashibodi component:
```typescript
{displayRole === "MyRole" && <MyRoleView groupId={user.group_id} />}
```

3. **Create API endpoint** in backend (if not exists)
4. **Add TypeScript type** to `api/types.ts`
5. **Create API function** in `api/dashboard.ts`
6. **Create React hook** in `hooks/use-scoped-dashboard.ts`
7. **Create card components** if needed
8. **Write tests** in `components/__tests__/`

### Using the Role-Switch Context

```typescript
import { useRoleSwitch } from "@/lib/role-context";

function MyComponent() {
  const { currentRole, availableRoles, switchRole } = useRoleSwitch();

  return (
    <div>
      <p>Currently viewing as: {currentRole}</p>
      <p>Available roles: {availableRoles.join(", ")}</p>
      <button onClick={() => switchRole("Mwanachama")}>
        Switch to Member
      </button>
    </div>
  );
}
```

### Fetching Role-Scoped Data

```typescript
import { useMemberDashboardSummary } from "@/hooks/use-scoped-dashboard";

function MyCard() {
  const { data, isLoading, error } = useMemberDashboardSummary(memberId);

  if (isLoading) return <Skeleton />;
  if (error) return <ErrorAlert message="Data load failed" />;
  if (!data) return null;

  return <div>{data.total_contributions}</div>;
}
```

### Type Safety Tips

```typescript
// ✅ Good - Matches API response
const amount: string = data.total_contributions; // Snake case from API
const count: number = data.contributions_count;

// ❌ Bad - Wrong types
const amount: number = data.total_contributions; // Should be string
const count: string = data.contributions_count; // Should be number

// ✅ Good - Convert for display
const formatted = tzs(Number(data.total_contributions));

// ❌ Bad - Direct "0" fallback
const amount = data.total_contributions || "0"; // Don't do this!
```

### Error Handling Pattern

```typescript
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AlertCircle } from "lucide-react";

if (error || !data) {
  return (
    <Alert className="border-destructive/50 bg-destructive/5">
      <AlertCircle className="h-5 w-5 text-destructive" />
      <AlertDescription className="text-destructive">
        Imeshindikana kupakia. Tafadhali jaribu tena.
      </AlertDescription>
    </Alert>
  );
}
```

### Loading State Pattern

```typescript
if (isLoading) {
  return (
    <div className="card-surface animate-pulse">
      <div className="h-12 bg-muted rounded w-40 mb-4" />
      <div className="h-6 bg-muted rounded w-32" />
      <div className="grid grid-cols-2 gap-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-16 bg-muted rounded" />
        ))}
      </div>
    </div>
  );
}
```

## Common Patterns

### Never Show Default "0"

```typescript
// ❌ Wrong - Misleading zero
<p>{data?.total_contributions ?? 0}</p>

// ✅ Right - Show nothing or message
{data?.total_contributions && (
  <p>{tzs(Number(data.total_contributions))}</p>
)}

// ✅ Right - Show in loading state
if (isLoading) return <SkeletonLoader />;
if (error) return <ErrorAlert />;
```

### Multi-Role User Detection

```typescript
// User has multiple roles if:
const isMultiRole = availableRoles.length > 1;

// Or in raw User data:
const hasLeadership = user.leadership && user.leadership.length > 0;
const hasMultipleRoles = hasLeadership || user.member_id;
```

### Responsive Grid Patterns

```typescript
// 2 columns on mobile, 4 on desktop
<div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
  <Card />
  <Card />
  <Card />
  <Card />
</div>

// 3 columns on mobile, 6 on desktop
<div className="grid grid-cols-3 gap-2 lg:grid-cols-6">
  <Item />
</div>
```

### Swahili Translation Map

```typescript
const roles: Record<Jukumu, string> = {
  "Mwenyekiti": "Mwenyekiti (Chair)",
  "Mweka Hazina": "Mweka Hazina (Treasurer)",
  "Katibu": "Katibu (Secretary)",
  "Mwanachama": "Mwanachama (Member)",
  "Msimamizi": "Msimamizi (Admin)",
};

const statuses: Record<string, string> = {
  "CONFIRMED": "Kuthibitishwa",
  "PENDING_VERIFICATION": "Kinasub.",
  "PAID": "Kilicholipwa",
  "REJECTED": "Kilikataliwa",
};
```

## Debugging Tips

### Check Role Context is Provided

```typescript
// If you get "useRoleSwitch must be used within RoleSwitchProvider"
// Make sure parent component is wrapped:
<RoleSwitchProvider primaryRole={...} leadershipRoles={[...]}>
  <YourComponent />
</RoleSwitchProvider>
```

### Verify API Response Format

```typescript
// Backend returns:
{ data: { total_contributions: "40000", ... } }

// Component receives:
const { data } = useMemberDashboardSummary(memberId);
console.log(data); // { total_contributions: "40000", ... }
```

### Check Query Cache

```typescript
// In browser dev tools console:
// Find React Query Devtools (usually bottom-right)
// or check Network tab for API calls
// Look for /api/v1/members/:id/dashboard-summary
```

### Test Role Switching

```typescript
// Browser console:
// Find element with class "role-switch-toggle"
// Select different role from dropdown
// Verify sidebar and dashboard update
// Verify no page reload occurs
```

## Performance Considerations

1. **Query Keys:** Uses standard React Query patterns
   - Automatic caching per member/group ID
   - Manual invalidation if needed: `queryClient.invalidateQueries({ queryKey: ["member-dashboard-summary", memberId] })`

2. **Component Re-renders:**
   - Only dashboard re-renders on role switch
   - Header and sidebar maintain state
   - Unnecessary re-renders minimized

3. **API Calls:**
   - One call per role change (not per component)
   - Cached automatically by React Query
   - No refetch on role switch (cached)

## Accessibility Notes

- [ ] Role dropdown labeled "Jukumu" (Swahili for "Role/Duty")
- [ ] Icons used to differentiate roles visually
- [ ] Loading skeletons maintain layout (no layout shift)
- [ ] Error messages clear and actionable
- [ ] Role switch visible only when applicable (multi-role users)

## Browser Compatibility

- Chrome/Chromium: ✅
- Firefox: ✅
- Safari: ✅
- Edge: ✅
- Mobile browsers: ✅ (tested with viewport)

## Production Checklist

- [ ] Environment variables set (API URL, token)
- [ ] Backend endpoints responding correctly
- [ ] Tests all passing (npm run test)
- [ ] No console errors or warnings
- [ ] Role switching tested with real multi-role users
- [ ] Asha (KKK-0009) contribution shows correctly
- [ ] Loading states display properly
- [ ] Error states are clear
- [ ] Mobile layout responsive
- [ ] Keyboard navigation works
- [ ] Performance acceptable (no lag on switch)

---

**Need Help?** 
- Check test files for examples: `src/components/__tests__/`
- Review dashboard implementation: `src/routes/dashibodi.tsx`
- See API contract: `FRONTEND_INTEGRATION_GUIDE.md` (from backend PR)
