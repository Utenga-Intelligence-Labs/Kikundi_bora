# MASTER TASK — Kikundi Web App V1 (Fanya Kwa Mpangilio Huu, Usiruke Hatua)

Backend: Go (Gin), Postgres `kikundi_db` (local, `DB_HOST=127.0.0.1`), API `localhost:8080`.
Frontend: existing web app, tab `/wanachama` tayari inasoma `members` table (bug hii
imeshafixiwa jana — usiiguse tena).
Muktadha: `users` = auth; `members` = wanachama halisi (KKK-xxxx); admin/overseer si member.

Fanya kazi hii kwa awamu (PHASE) zilizopangwa. Baada ya kila phase, run build/test na uripoti
kabla ya kuendelea na phase inayofuata — usisubiri mpaka mwisho kuripoti kila kitu kwa mara moja.

---

## PHASE 1 — Dual Plane: Mwenyekiti/Hazina/Katibu kama Member + Kiongozi (Msingi wa kila kitu kingine)

### 1.1 Schema
```sql
CREATE TABLE leadership_positions (
    id SERIAL PRIMARY KEY,
    member_id INT NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('MWENYEKITI', 'HAZINA', 'KATIBU')),
    term_start DATE NOT NULL DEFAULT CURRENT_DATE,
    term_end DATE,
    is_current BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_one_current_role_per_member
    ON leadership_positions(member_id, role) WHERE is_current = true;
```
Backfill: wale walio chair/hazina/katibu waliokwisha pata `members` row (jana) — wape row
kwenye `leadership_positions` sasa (`is_current = true`). Admin/overseer HAWAPATI row yoyote
ndani ya `members` wala `leadership_positions`.

### 1.2 Backend — Auth Context
```go
type UserContext struct {
    UserID     int
    MemberID   *int
    IsAdmin    bool
    Leadership []string // e.g. ["MWENYEKITI"]
}
```
Jenga hii wakati wa login/token-refresh kwa join `users -> members -> leadership_positions
(is_current=true)`.

### 1.3 Middleware
- `RequireMember()` — anahitaji `MemberID != nil`.
- `RequireLeadership(roles ...string)` — anahitaji `Leadership` kuingiliana na `roles`.
- Endpoints za member (omba mkopo, akiba, historia) zisiwe na `if isLeadership` branching —
  zote zinatumia `MemberID` tu.

### 1.4 Endpoints mpya
- `GET /api/me` → `{ memberId, memberCode, isAdmin, leadership: [] }`

### 1.5 Frontend
- `AuthContext` inafetch `/api/me`, inahifadhi `{ memberId, isAdmin, leadership }`.
- `RequireMember` / `RequireLeadership(roles)` route guards.
- Navbar badges: `🟢 Mwanachama · KKK-xxxx` + `👑 {role}` kama ana leadership. Admin anaona
  label tofauti kabisa ("Overseer/Admin"), si badge ya member.
- Dashboard: tabs "Dashboard Yangu" (default) na "Uongozi" (DOM-removed kabisa kama
  `leadership.length === 0`, si tu disabled).
- Sidebar: menu ya kawaida juu, divider, kisha "UONGOZI" section chini (tu ikiwa ana
  leadership).

### ✅ Phase 1 Checklist
- [ ] Login kama dadi (KKK-0010) → `/api/me` inarudisha memberId sahihi.
- [ ] `/wanachama` bado inaonyesha dadi (hakuna regression).
- [ ] Login kama mwenyekiti → tabs zote mbili zinaonekana.
- [ ] Login kama member wa kawaida → tab "Uongozi" haipo kabisa kwenye DOM (check inspector).
- [ ] `curl` moja kwa moja kwa endpoint ya uongozi kama member wa kawaida → 403.
- [ ] Admin/overseer haonekani na badge ya member, hana row kwenye `members`.

---

## PHASE 2 — Hazina Cap kwenye Mikopo (Data Integrity — Muhimu Sana)

- Tengeneza function moja (service/repo layer) inayokokotoa `hazina_balance_available`:
  jumla ya michango zilizothibitishwa (CONFIRMED) + malipo ya mikopo yaliyorejeshwa − mikopo
  yaliyotolewa tayari (ACTIVE/DISBURSED).
- Kwenye endpoint ya "omba mkopo" (kabla ya kuunda record au kabla ya approval — chagua
  hatua moja iliyo wazi, pendekezo: validate wote wawili), angalia:
  `if loan_amount > hazina_balance_available { reject na error: "Kiasi kinazidi hazina ya
  kikundi (TZS X iliyopo)" }`.
- Function hii itatumika tena kwenye Phase 3 (dashboard) na Phase 5 (mikopo page).

### ✅ Phase 2 Checklist
- [ ] Jaribu kuomba mkopo unaozidi hazina → umezuiwa na error wazi.
- [ ] Jaribu kuomba mkopo ulio ndani ya hazina → unapita kawaida.

---

## PHASE 3 — Fix "Takwimu za Haraka" (Haifetch Data)

1. Chunguza frontend — endpoint gani inaitwa? Angalia network tab / API call.
2. Test hiyo endpoint moja kwa moja na curl kabla ya kudhania frontend ndiyo tatizo.
3. Kama endpoint haipo/ina bug, tengeneza/rekebisha `GET /api/dashboard/quick-stats`
   inayorudisha: jumla ya wanachama, jumla ya michango ya mwezi huu, jumla ya mikopo
   active, hazina balance ya sasa (tumia function ya Phase 2).
4. Hakikisha response field names zinalingana kabisa na frontend inavyotarajia.
5. Ongeza loading + error states frontend — isikae blank kimya.

### ✅ Phase 3 Checklist
- [ ] "Takwimu za Haraka" inaonyesha namba halisi, si blank.

---

## PHASE 4 — Mfumo wa Kuweka Mchango (Haipo — Jenga Mpya)

### 4.1 Schema
```sql
CREATE TABLE contributions (
    id SERIAL PRIMARY KEY,
    member_id INT NOT NULL REFERENCES members(id),
    contribution_type VARCHAR(20) NOT NULL CHECK (contribution_type IN ('AKIBA','MFUKO_WA_KIJAMII')),
    period_label VARCHAR(30) NOT NULL,
    amount NUMERIC(12,2) NOT NULL,
    proof_image_url TEXT,
    proof_message TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING_VERIFICATION'
        CHECK (status IN ('PENDING_VERIFICATION','CONFIRMED','REJECTED')),
    reviewed_by_member_id INT REFERENCES members(id),
    review_reason TEXT,
    is_historical_import BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    reviewed_at TIMESTAMP
);
```
Ongeza kwenye kikundi settings: `contribution_period_type VARCHAR(10) CHECK (IN
('WEEKLY','MONTHLY'))` — admin anaiseti hii.

### 4.2 Mzunguko
1. Member anatuma mchango (kiasi + period ya sasa kutokana na setting ya admin) na proof —
   picha AU ujumbe wa muamala (moja required).
2. Status huanza `PENDING_VERIFICATION`.
3. `AKIBA` → hazina anaidhinisha. `MFUKO_WA_KIJAMII` → mwenyekiti anaidhinisha.
4. Approve → `CONFIRMED` (sasa inahesabika kwenye hazina balance). Reject → `REJECTED` na
   `review_reason`, member anaweza kutuma tena.

### 4.3 Endpoints
- `POST /api/michango` (multipart kwa picha, au JSON kwa message-only)
- `GET /api/michango/mine` — member anaona zake tu
- `GET /api/michango` (all, filterable) — `RequireLeadership`
- `GET /api/michango/pending` — `RequireLeadership`
- `POST /api/michango/:id/confirm`
- `POST /api/michango/:id/reject` (+ reason)

Storage ya picha: local disk `/uploads/` kwa v1 (si lazima cloud sasa).

### 4.4 Frontend
- Fomu ya "Weka Mchango": kiasi, upload picha AU andika ujumbe, chagua type (AKIBA/MFUKO).
- Ukurasa wa hazina/mwenyekiti wa "Michango Yanayosubiri" — anaona proof, ana-approve/reject.
- Member wa kawaida anaona "Michango Yangu" tu. Leadership wanaona tab ya ziada "Michango
  ya Wote" (tumia `RequireLeadership` guard ile ile ya Phase 1).

### ✅ Phase 4 Checklist
- [ ] Member anatuma mchango na picha → PENDING_VERIFICATION.
- [ ] Member anatuma mchango na ujumbe tu (bila picha) → inakubalika pia.
- [ ] Hazina anaidhinisha AKIBA → CONFIRMED, inaongezeka kwenye hazina balance.
- [ ] Mwenyekiti anaidhinisha MFUKO_WA_KIJAMII.
- [ ] Member wa kawaida akijaribu `GET /api/michango` (all) → 403.
- [ ] Reject inarudisha reason kwa member.

---

## PHASE 5 — Historia Yangu + Mikopo (Pages Tofauti)

- **`/historia-yangu`**: feed ya actions zote za huyu member — michango (zote aina mbili) +
  mikopo (maombi na malipo) — pamoja kwa mpangilio wa tarehe. Kama muda mfupi: fanya
  frontend-side merge ya endpoints zilizopo (`/api/michango/mine` + `/api/mikopo/mine`)
  badala ya kutengeneza unified backend endpoint mpya.
- **`/mikopo`**: ukurasa maalum — history ya maombi yake ya mikopo (status: PENDING/
  APPROVED/REJECTED/ACTIVE/PAID) + fomu ya "Omba Mkopo Mpya" (inayotumia validation ya
  Phase 2).

### ✅ Phase 5 Checklist
- [ ] "Historia Yangu" inaonyesha michango na mikopo pamoja, sorted by date.
- [ ] "Mikopo" ni ukurasa tofauti wenye fomu ya kuomba mkopo mpya.

---

## PHASE 6 — Kupakia Data ya Zamani (Vitabu)

- CSV import script/endpoint: `member_code, amount, type, date, status`.
- Ingiza moja kwa moja kama `CONFIRMED`/`PAID` (bila kupitia PENDING workflow — tayari
  zilithibitishwa kimwili zamani), na weka `is_historical_import = true`.
- Kama muda mfupi: fanya kama script ya one-off (Go script au raw SQL), si UI kamili.

### ✅ Phase 6 Checklist
- [ ] Import inafanya kazi kwenye dev DB bila kuvunja workflow ya kawaida.
- [ ] Records za zamani zina `is_historical_import = true`.

---

## PHASE 7 — Notifications

```sql
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    member_id INT REFERENCES members(id),  -- null = broadcast kwa wote
    title VARCHAR(200) NOT NULL,
    body TEXT,
    type VARCHAR(30),
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```
- Trigger kwenye: mchango CONFIRMED/REJECTED, mkopo APPROVED/REJECTED, na announcement/ripoti
  yoyote leadership wanayo-"publish" (broadcast, `member_id = null`).
- Frontend: bell icon navbar na unread badge count, dropdown/page ya notifications,
  `PATCH /api/notifications/:id/read`. In-app polling tu kwa v1 — hakuna SMS/push.

### ✅ Phase 7 Checklist
- [ ] Notification inatengenezwa mchango/mkopo inapoidhinishwa/kukataliwa.
- [ ] Bell badge inaongezeka/inapungua sahihi.

---

## MUHIMU — Kanuni za Jumla Kwa Awamu Zote
- Usifanye member-facing endpoints (mkopo, akiba) ku-branch kwa leadership status — key off
  `MemberID` tu.
- Admin/overseer HAWEZI kuwa member — angalia hili kila phase.
- Tumia middleware `RequireMember`/`RequireLeadership` moja iliyojengwa Phase 1 kila
  mahali — usiandike checks mpya za ad-hoc.
- Baada ya kila phase: `go build ./...` na frontend build lazima zipite bila error kabla ya
  kuendelea na phase inayofuata.
- Ripoti baada ya kila phase: nini kimebadilika (files), na majibu ya checklist husika.

## Non-Goals kwa V1
- Hakuna dual login/dual account kwa leadership.
- Hakuna push notifications za simu (SMS/FCM) — in-app tu.
- Hakuna cloud storage kwa picha — local disk inatosha.
- Hakuna automated recurring reminders.

## Mwisho wa Kazi — Deliverables
1. Migrations zote (leadership_positions, contributions, notifications, settings field).
2. Backend: middleware, endpoints, hazina-balance function, dashboard fix.
3. Frontend: AuthContext, guards, badges, tabs, sidebar, mchango forms, historia/mikopo
   pages, notification bell.
4. Import script ya data ya zamani.
5. Muhtasari wa mwisho: files zote zilizoguswa + matokeo ya checklist za phase zote 1-7.
