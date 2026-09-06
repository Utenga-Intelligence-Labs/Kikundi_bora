package handlers

import (
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
)

// loadObligationGroup resolves :id to this deployment's group or writes 404.
func loadObligationGroup(c *fiber.Ctx) *models.Group {
	id := c.Params("id")
	if ok, err := database.IsCurrentGroup(id); err != nil || !ok {
		c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
		return nil
	}
	var g models.Group
	if err := database.DB.First(&g, "id = ?", id).Error; err != nil {
		c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Kikundi hakijapatikana"})
		return nil
	}
	return &g
}

// ── Summaries ────────────────────────────────────────────────────────────────

// GET /api/v1/members/:id/obligations/summary — self or leadership.
func ObligationsMemberSummary(c *fiber.Ctx) error {
	g, err := database.GetCurrentGroup()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupata kikundi"})
	}
	out, err := services.GetMemberObligations(g.ID, c.Params("id"), time.Now())
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama hajapatikana"})
	}
	return c.JSON(fiber.Map{"data": out})
}

// GET /api/v1/groups/:id/obligations/summary — leadership aggregate.
func ObligationsGroupSummary(c *fiber.Ctx) error {
	g := loadObligationGroup(c)
	if g == nil {
		return nil
	}
	out, err := services.GetGroupObligations(g.ID, time.Now())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukokotoa"})
	}
	return c.JSON(fiber.Map{"data": out})
}

// GET /api/v1/groups/:id/collection-queue — mweka hazina working view:
// every owing member with itemized arrears + fines, biggest debt first.
func CollectionQueue(c *fiber.Ctx) error {
	g := loadObligationGroup(c)
	if g == nil {
		return nil
	}
	group, err := services.GetGroupObligations(g.ID, time.Now())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukokotoa"})
	}
	type queueItem struct {
		Member  services.MemberObligationRollup `json:"member"`
		Arrears []services.ArrearsItem          `json:"arrears"`
		Fines   []services.FineItem             `json:"fines"`
	}
	queue := []queueItem{}
	for _, roll := range group.Members {
		if roll.GrandTotal.IsZero() {
			continue
		}
		mo, err := services.GetMemberObligations(g.ID, roll.MemberID, time.Now())
		if err != nil {
			continue
		}
		queue = append(queue, queueItem{Member: roll, Arrears: mo.ItemizedArrears, Fines: mo.ItemizedFines})
	}
	// Biggest debt first.
	for i := 0; i < len(queue); i++ {
		for j := i + 1; j < len(queue); j++ {
			if queue[j].Member.GrandTotal.GreaterThan(queue[i].Member.GrandTotal) {
				queue[i], queue[j] = queue[j], queue[i]
			}
		}
	}
	return c.JSON(fiber.Map{"data": queue, "total": len(queue)})
}

// ── Notification settings (SMS channel) ────────────────────────────────────

type notificationSettingsResponse struct {
	SMSEnabled       bool              `json:"sms_enabled"`
	Provider         string            `json:"provider"`
	ProviderReal     bool              `json:"provider_real"`
	Types            map[string]bool   `json:"types"`
}

// GET /api/v1/groups/:id/notification-settings — mwenyekiti/admin.
func GetNotificationSettings(c *fiber.Ctx) error {
	g := loadObligationGroup(c)
	if g == nil {
		return nil
	}
	types := map[string]bool{}
	for _, t := range []models.NotificationType{
		models.NotifContributionDue, models.NotifFineIssued,
		models.NotifLoanRequest, models.NotifLoanApproved, models.NotifLoanDisbursed,
		models.NotifRepayment, models.NotifContribution, models.NotifWelfarePayment,
		models.NotifUserCreated, models.NotifSystem,
	} {
		types[string(t)] = services.SMDPrefOrDefault(g.ID, t)
	}
	return c.JSON(fiber.Map{"data": notificationSettingsResponse{
		SMSEnabled:   g.SMSNotificationsEnabled,
		Provider:     services.SMSProviderName(),
		ProviderReal: services.SMSProviderReal(),
		Types:        types,
	}})
}

type notificationSettingsRequest struct {
	SMSEnabled *bool           `json:"sms_enabled"`
	Types      map[string]bool `json:"types"`
}

// PUT /api/v1/groups/:id/notification-settings — mwenyekiti/admin.
func UpdateNotificationSettings(c *fiber.Ctx) error {
	g := loadObligationGroup(c)
	if g == nil {
		return nil
	}
	var req notificationSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	userID := middleware.GetUserID(c)
	if req.SMSEnabled != nil {
		g.SMSNotificationsEnabled = *req.SMSEnabled
		if err := database.DB.Save(g).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi"})
		}
	}
	for typeName, enabled := range req.Types {
		var pref models.NotificationSMSPref
		err := database.DB.Where("group_id = ? AND notif_type = ?", g.ID, typeName).First(&pref).Error
		if err == nil {
			pref.Enabled = enabled
			pref.UpdatedBy = &userID
			database.DB.Save(&pref)
		} else {
			database.DB.Create(&models.NotificationSMSPref{
				GroupID: g.ID, NotifType: typeName, Enabled: enabled, UpdatedBy: &userID,
			})
		}
	}
	services.LogAudit(c, &userID, models.AuditUpdate, "notification_settings", &g.ID, nil, map[string]interface{}{
		"sms_enabled": g.SMSNotificationsEnabled,
	})
	return GetNotificationSettings(c)
}

type offenceTypeRequest struct {
	Kind            string           `json:"kind"`
	Name            string           `json:"name"`
	FineType        string           `json:"fine_type"`
	FineAmount      *decimal.Decimal `json:"fine_amount"`
	FinePercentage  *decimal.Decimal `json:"fine_percentage"`
	GracePeriodDays *int             `json:"grace_period_days"`
}

func validateOffenceRequest(req *offenceTypeRequest) error {
	if !models.IsValidOffenceKind(req.Kind) {
		return fiber.NewError(fiber.StatusBadRequest, "Aina ya kosa si sahihi")
	}
	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Jina la kosa linahitajika")
	}
	ft := req.FineType
	if ft == "" {
		ft = models.FineTypeFixed
	}
	grace := 0
	if req.GracePeriodDays != nil {
		grace = *req.GracePeriodDays
	}
	if err := services.ValidateFineSettingsSpec(true, ft, req.FineAmount, req.FinePercentage, grace); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return nil
}

// GET /api/v1/groups/:id/fine-offence-types — leadership read.
func ListOffenceTypes(c *fiber.Ctx) error {
	g := loadObligationGroup(c)
	if g == nil {
		return nil
	}
	var ots []models.FineOffenceType
	database.DB.Where("group_id = ?", g.ID).Order("kind ASC, name ASC").Find(&ots)
	return c.JSON(fiber.Map{"data": ots, "total": len(ots)})
}

// POST /api/v1/groups/:id/fine-offence-types — mwenyekiti proposes (pending).
func CreateOffenceType(c *fiber.Ctx) error {
	g := loadObligationGroup(c)
	if g == nil {
		return nil
	}
	var req offenceTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validateOffenceRequest(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	ft := req.FineType
	if ft == "" {
		ft = models.FineTypeFixed
	}
	grace := 0
	if req.GracePeriodDays != nil {
		grace = *req.GracePeriodDays
	}
	userID := middleware.GetUserID(c)
	ot := models.FineOffenceType{
		GroupID: g.ID, Kind: req.Kind, Name: req.Name, FineType: ft,
		FineAmount: req.FineAmount, FinePercentage: req.FinePercentage,
		GracePeriodDays: grace, Status: models.OffencePending, CreatedBy: userID,
	}
	if err := database.DB.Create(&ot).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunda"})
	}
	services.LogAudit(c, &userID, models.AuditCreate, "fine_offence_types", &ot.ID, nil, map[string]interface{}{
		"kind": ot.Kind, "name": ot.Name,
	})
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Pendekezo limetumwa kwa Katibu", "data": ot})
}

// PATCH /api/v1/groups/:id/fine-offence-types/:typeId — mwenyekiti edits
// (active types drop back to pending for re-approval).
func UpdateOffenceType(c *fiber.Ctx) error {
	g := loadObligationGroup(c)
	if g == nil {
		return nil
	}
	var ot models.FineOffenceType
	if err := database.DB.Where("id = ? AND group_id = ?", c.Params("typeId"), g.ID).First(&ot).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Aina ya kosa haijapatikana"})
	}
	var req offenceTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if req.Kind != "" && !models.IsValidOffenceKind(req.Kind) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Aina ya kosa si sahihi"})
	}
	if req.Kind != "" {
		ot.Kind = req.Kind
	}
	if req.Name != "" {
		ot.Name = req.Name
	}
	if req.FineType != "" {
		if !models.IsValidFineType(req.FineType) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Aina ya faini si sahihi"})
		}
		ot.FineType = req.FineType
	}
	if req.FineAmount != nil {
		ot.FineAmount = req.FineAmount
	}
	if req.FinePercentage != nil {
		ot.FinePercentage = req.FinePercentage
	}
	if req.GracePeriodDays != nil {
		if *req.GracePeriodDays < 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Siku za neema haziwezi kuwa hasi"})
		}
		ot.GracePeriodDays = *req.GracePeriodDays
	}
	if err := services.ValidateFineSettingsSpec(true, ot.FineType, ot.FineAmount, ot.FinePercentage, ot.GracePeriodDays); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	if ot.Status == models.OffenceActive {
		ot.Status = models.OffencePending
		ot.ApprovedBy = nil
	}
	if err := database.DB.Save(&ot).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi"})
	}
	userID := middleware.GetUserID(c)
	services.LogAudit(c, &userID, models.AuditUpdate, "fine_offence_types", &ot.ID, nil, map[string]interface{}{
		"status": ot.Status,
	})
	msg := "Mabadiliko yamehifadhiwa"
	if ot.Status == models.OffencePending {
		msg = "Mabadiliko yametumwa kwa Katibu kuidhinisha"
	}
	return c.JSON(fiber.Map{"message": msg, "data": ot})
}

// POST /api/v1/groups/:id/fine-offence-types/:typeId/approve — katibu.
func ApproveOffenceType(c *fiber.Ctx) error {
	g := loadObligationGroup(c)
	if g == nil {
		return nil
	}
	var ot models.FineOffenceType
	if err := database.DB.Where("id = ? AND group_id = ?", c.Params("typeId"), g.ID).First(&ot).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Aina ya kosa haijapatikana"})
	}
	userID := middleware.GetUserID(c)
	ot.Status = models.OffenceActive
	ot.ApprovedBy = &userID
	if err := database.DB.Save(&ot).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuidhinisha"})
	}
	services.LogAudit(c, &userID, models.AuditApprove, "fine_offence_types", &ot.ID, nil, map[string]interface{}{
		"name": ot.Name,
	})
	return c.JSON(fiber.Map{"message": "Aina ya kosa imeidhinishwa", "data": ot})
}

// POST /api/v1/groups/:id/fine-offence-types/:typeId/deactivate —
// mwenyekiti or katibu. Turning off stops new fines; old ones stand.
func DeactivateOffenceType(c *fiber.Ctx) error {
	g := loadObligationGroup(c)
	if g == nil {
		return nil
	}
	var ot models.FineOffenceType
	if err := database.DB.Where("id = ? AND group_id = ?", c.Params("typeId"), g.ID).First(&ot).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Aina ya kosa haijapatikana"})
	}
	ot.Status = models.OffenceInactive
	if err := database.DB.Save(&ot).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana"})
	}
	userID := middleware.GetUserID(c)
	services.LogAudit(c, &userID, models.AuditUpdate, "fine_offence_types", &ot.ID, nil, map[string]interface{}{
		"status": "inactive",
	})
	return c.JSON(fiber.Map{"message": "Aina ya kosa imezimwa — faini zilizotolewa hazijaguswa", "data": ot})
}

// ── Fines ────────────────────────────────────────────────────────────────────

// GET /api/v1/fines?member_id=&status= — members see own only.
func ListFines(c *fiber.Ctx) error {
	role := middleware.GetUserRole(c)
	query := database.DB.Preload("OffenceType").Preload("Member").Order("occurrence_date DESC")
	if role == models.RoleMember {
		var mem models.Member
		if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", middleware.GetUserID(c)).First(&mem).Error; err != nil {
			return c.JSON(fiber.Map{"data": []models.Fine{}, "total": 0})
		}
		query = query.Where("member_id = ?", mem.ID)
	} else if mid := c.Query("member_id"); mid != "" {
		query = query.Where("member_id = ?", mid)
	}
	if st := c.Query("status"); st != "" {
		if !models.IsValidFineStatus(st) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Hali si sahihi"})
		}
		query = query.Where("status = ?", st)
	}
	var fines []models.Fine
	query.Find(&fines)
	return c.JSON(fiber.Map{"data": fines, "total": len(fines)})
}

// POST /api/v1/fines/:id/collect — mweka hazina ONLY.
func CollectFine(c *fiber.Ctx) error {
	g, err := database.GetCurrentGroup()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupata kikundi"})
	}
	var f models.Fine
	if err := database.DB.Where("id = ? AND group_id = ?", c.Params("id"), g.ID).First(&f).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Faini haijapatikana"})
	}
	if f.Status != models.FineUnpaid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Faini hii tayari imeshughulikiwa"})
	}
	userID := middleware.GetUserID(c)
	now := time.Now()
	f.Status = models.FinePaid
	f.CollectedBy = &userID
	f.CollectedAt = &now
	if err := database.DB.Save(&f).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana"})
	}
	services.LogAudit(c, &userID, models.AuditUpdate, "fines", &f.ID, nil, map[string]interface{}{
		"status": "paid",
	})
	return c.JSON(fiber.Map{"message": "Faini imemarkiwa imelipwa", "data": f})
}

type waiveRequest struct {
	Reason string `json:"reason"`
}

// POST /api/v1/fines/:id/waive-propose — mwenyekiti proposes.
func ProposeFineWaiver(c *fiber.Ctx) error {
	g, err := database.GetCurrentGroup()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupata kikundi"})
	}
	var f models.Fine
	if err := database.DB.Where("id = ? AND group_id = ?", c.Params("id"), g.ID).First(&f).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Faini haijapatikana"})
	}
	if f.Status != models.FineUnpaid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Faini hii tayari imeshughulikiwa"})
	}
	var req waiveRequest
	if err := c.BodyParser(&req); err != nil || req.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Sababu inahitajika"})
	}
	userID := middleware.GetUserID(c)
	f.WaiverStatus = models.WaiverPending
	f.WaiverRequestedBy = &userID
	f.WaiverRequestReason = &req.Reason
	if err := database.DB.Save(&f).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana"})
	}
	services.LogAudit(c, &userID, models.AuditCreate, "fine_waivers", &f.ID, nil, map[string]interface{}{
		"reason": req.Reason,
	})
	return c.JSON(fiber.Map{"message": "Ombi la msamaha limetumwa kwa Katibu", "data": f})
}

// POST /api/v1/fines/:id/waive-approve — katibu approves.
func ApproveFineWaiver(c *fiber.Ctx) error {
	g, err := database.GetCurrentGroup()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupata kikundi"})
	}
	var f models.Fine
	if err := database.DB.Where("id = ? AND group_id = ?", c.Params("id"), g.ID).First(&f).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Faini haijapatikana"})
	}
	if f.WaiverStatus != models.WaiverPending || f.Status != models.FineUnpaid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Hakuna ombi linalosubiri"})
	}
	userID := middleware.GetUserID(c)
	now := time.Now()
	_ = now
	f.Status = models.FineWaived
	f.WaivedBy = &userID
	if f.WaiverRequestReason != nil {
		f.WaivedReason = f.WaiverRequestReason
	}
	f.WaiverStatus = models.WaiverApproved
	if err := database.DB.Save(&f).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana"})
	}
	services.LogAudit(c, &userID, models.AuditApprove, "fine_waivers", &f.ID, nil, nil)
	return c.JSON(fiber.Map{"message": "Msamaha umeidhinishwa", "data": f})
}

// POST /api/v1/fines/:id/waive-reject — katibu rejects.
func RejectFineWaiver(c *fiber.Ctx) error {
	g, err := database.GetCurrentGroup()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupata kikundi"})
	}
	var f models.Fine
	if err := database.DB.Where("id = ? AND group_id = ?", c.Params("id"), g.ID).First(&f).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Faini haijapatikana"})
	}
	if f.WaiverStatus != models.WaiverPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Hakuna ombi linalosubiri"})
	}
	userID := middleware.GetUserID(c)
	f.WaiverStatus = models.WaiverRejected
	if err := database.DB.Save(&f).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana"})
	}
	services.LogAudit(c, &userID, models.AuditUpdate, "fine_waivers", &f.ID, nil, nil)
	return c.JSON(fiber.Map{"message": "Ombi la msamaha limekataliwa", "data": f})
}

// ── Meetings ─────────────────────────────────────────────────────────────────

type meetingRequest struct {
	Title       string  `json:"title"`
	MeetingDate string  `json:"meeting_date"`
	Notes       *string `json:"notes"`
}

// GET /api/v1/groups/:id/meetings — leadership read.
func ListMeetings(c *fiber.Ctx) error {
	g := loadObligationGroup(c)
	if g == nil {
		return nil
	}
	var meetings []models.Meeting
	database.DB.Where("group_id = ?", g.ID).Order("meeting_date DESC").Find(&meetings)
	return c.JSON(fiber.Map{"data": meetings, "total": len(meetings)})
}

// POST /api/v1/groups/:id/meetings — mwenyekiti or katibu.
func CreateMeeting(c *fiber.Ctx) error {
	g := loadObligationGroup(c)
	if g == nil {
		return nil
	}
	var req meetingRequest
	if err := c.BodyParser(&req); err != nil || req.Title == "" || req.MeetingDate == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kichwa na tarehe vinahitajika"})
	}
	day, err := time.Parse("2006-01-02", req.MeetingDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Tarehe si sahihi (YYYY-MM-DD)"})
	}
	userID := middleware.GetUserID(c)
	mtg := models.Meeting{
		GroupID: g.ID, Title: req.Title, MeetingDate: day,
		Notes: req.Notes, CreatedBy: userID,
	}
	if err := database.DB.Create(&mtg).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunda"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Mkutano umeundwa", "data": mtg})
}

type attendanceRow struct {
	MemberID string `json:"member_id"`
	Status   string `json:"status"`
}

// GET /api/v1/meetings/:id/attendance — leadership read.
func GetAttendance(c *fiber.Ctx) error {
	var rows []models.MeetingAttendance
	database.DB.Preload("Member").Where("meeting_id = ?", c.Params("id")).Find(&rows)
	return c.JSON(fiber.Map{"data": rows, "total": len(rows)})
}

// PUT /api/v1/meetings/:id/attendance — katibu marks attendance.
func SetAttendance(c *fiber.Ctx) error {
	var req []attendanceRow
	if err := c.BodyParser(&req); err != nil || len(req) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Orodha ya mahudhurio inahitajika"})
	}
	for _, r := range req {
		if !models.IsValidAttendanceStatus(r.Status) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Hali si sahihi: " + r.Status})
		}
		var m models.Member
		if err := database.DB.First(&m, "id = ?", r.MemberID).Error; err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mwanachama hajapatikana"})
		}
		var existing models.MeetingAttendance
		err := database.DB.Where("meeting_id = ? AND member_id = ?", c.Params("id"), r.MemberID).First(&existing).Error
		if err == nil {
			existing.Status = r.Status
			database.DB.Save(&existing)
		} else {
			database.DB.Create(&models.MeetingAttendance{
				MeetingID: c.Params("id"), MemberID: r.MemberID, Status: r.Status,
			})
		}
	}
	return c.JSON(fiber.Map{"message": "Mahudhurio yamehifadhiwa"})
}

// POST /api/v1/meetings/:id/trigger-fines — katibu only.
func TriggerMeetingFines(c *fiber.Ctx) error {
	g, err := database.GetCurrentGroup()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana"})
	}
	userID := middleware.GetUserID(c)
	n, err := services.TriggerMeetingFines(g.ID, c.Params("id"), userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	services.LogAudit(c, &userID, models.AuditCreate, "meeting_fines", nil, nil, map[string]interface{}{
		"meeting_id": c.Params("id"), "created": n,
	})
	return c.JSON(fiber.Map{"message": "Faini zimetolewa", "created": n})
}
