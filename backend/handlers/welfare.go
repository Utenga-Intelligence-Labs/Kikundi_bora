package handlers

import (
	"fmt"
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WelfareHandler struct{}

func NewWelfareHandler() *WelfareHandler {
	return &WelfareHandler{}
}

// ---------- TREASURER: Create welfare event ----------

func (h *WelfareHandler) CreateEvent(c *fiber.Ctx) error {
	var req models.CreateWelfareEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	// Verify member exists
	var member models.Member
	if err := database.DB.Where("id = ? AND deleted_at IS NULL AND is_active = TRUE", req.MemberID).First(&member).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama hajapatikana au si hai"})
	}

	// Validate funding source amounts
	if req.FundingSource == string(models.FundTreasury) {
		if req.TreasuryAmount.LessThanOrEqual(decimal.Zero) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kiasi cha hazina lazima kiwe zaidi ya sifuri"})
		}
		req.MemberAmount = decimal.Zero
	} else if req.FundingSource == string(models.FundMemberContribution) {
		if req.MemberAmount.LessThanOrEqual(decimal.Zero) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kiasi cha wanachama lazima kiwe zaidi ya sifuri"})
		}
		req.TreasuryAmount = decimal.Zero
	} else if req.FundingSource == string(models.FundBoth) {
		if req.TreasuryAmount.LessThanOrEqual(decimal.Zero) || req.MemberAmount.LessThanOrEqual(decimal.Zero) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kiasi cha hazina na wanachama lazima viwe zaidi ya sifuri"})
		}
	}

	userID := middleware.GetUserID(c)

	event := models.WelfareEvent{
		MemberID:        req.MemberID,
		EventType:       models.WelfareEventType(req.EventType),
		Description:     req.Description,
		AmountRequested: req.AmountRequested,
		FundingSource:   models.WelfareFundingSource(req.FundingSource),
		TreasuryAmount:  req.TreasuryAmount,
		MemberAmount:    req.MemberAmount,
		Status:          models.WelfarePending,
		CreatedBy:       userID,
	}

	if err := database.DB.Create(&event).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutunda tukio"})
	}

	services.LogAudit(c, &userID, models.AuditCreateWelfareEvent, "welfare_events", &event.ID, nil, map[string]interface{}{
		"member_id": req.MemberID, "event_type": req.EventType, "amount": req.AmountRequested, "funding_source": req.FundingSource,
	})

	// Notify chairperson
	services.NotifyRole(models.RoleChair, models.NotifWelfareCreated,
		"Tukio Jipya la Kijamii",
		"Kuna tukio jipya la aina ya "+string(req.EventType)+" lenye kiasi cha TZS "+formatMoney(req.AmountRequested)+" linahitaji idhini yako.",
		"",
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Tukio la kijamii limetundwa. Linasubiri idhini ya Mwenyekiti.",
		"data":    event,
	})
}

// ---------- CHAIRPERSON: Approve welfare event ----------

func (h *WelfareHandler) ApproveEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.ApproveWelfareEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	userID := middleware.GetUserID(c)

	tx := database.DB.Begin()

	var event models.WelfareEvent
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&event, "id = ?", id).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Tukio la kijamii halijapatikana"})
	}

	if event.Status != models.WelfarePending {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Tukio haliwezi kuidhinishwa. Hali yake ni: " + string(event.Status)})
	}

	if req.ApprovedAmount.LessThanOrEqual(decimal.Zero) {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kiasi kilichoidhinishwa lazima kiwe zaidi ya sifuri"})
	}

	now := time.Now()
	event.Status = models.WelfareApproved
	event.AmountApproved = &req.ApprovedAmount
	event.ApprovedBy = &userID

	if err := tx.Save(&event).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuidhinisha"})
	}

	// If funding involves members, generate contribution obligations
	if event.FundingSource == models.FundMemberContribution || event.FundingSource == models.FundBoth {
		if err := h.generateMemberContributions(tx, event); err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutengeneza michango ya wanachama: " + err.Error()})
		}
	}

	// If funding is treasury-only, mark as completed immediately
	if event.FundingSource == models.FundTreasury {
		event.Status = models.WelfareCompleted
		event.CompletedAt = &now
		if err := tx.Save(&event).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukamilisha tukio"})
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuidhinisha"})
	}

	services.LogAudit(c, &userID, models.AuditApproveWelfareEvent, "welfare_events", &event.ID,
		map[string]interface{}{"status": "PENDING"},
		map[string]interface{}{"status": string(event.Status), "approved_amount": req.ApprovedAmount},
	)

	// Notify treasurer
	services.NotifyRole(models.RoleTreasurer, models.NotifWelfareApproved,
		"Tukio la Kijamii Limeidhinishwa",
		"Tukio la aina ya "+string(event.EventType)+" limeidhinishwa na Mwenyekiti kwa TZS "+formatMoney(req.ApprovedAmount)+".",
		"",
	)

	// Notify affected member
	var member models.Member
	if err := database.DB.First(&member, "id = ?", event.MemberID).Error; err == nil {
		var notifUserID string
		if member.UserID != nil {
			notifUserID = *member.UserID
		}
		if notifUserID == "" {
			notifUserID = member.RegisteredBy
		}
		if notifUserID != "" {
			services.NotifyUser(notifUserID, models.NotifWelfareApproved,
				"Tukio Lako la Kijamii Limeidhinishwa",
				"Tukio lako la aina ya "+string(event.EventType)+" limeidhinishwa kwa TZS "+formatMoney(req.ApprovedAmount)+".",
			)
		}
	}

	return c.JSON(fiber.Map{
		"message": "Tukio la kijamii limeidhinishwa",
		"data":    event,
	})
}

// ---------- CHAIRPERSON: Reject welfare event ----------

func (h *WelfareHandler) RejectEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.RejectWelfareEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	userID := middleware.GetUserID(c)

	tx := database.DB.Begin()

	var event models.WelfareEvent
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&event, "id = ?", id).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Tukio la kijamii halijapatikana"})
	}

	if event.Status != models.WelfarePending {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Tukio haliwezi kukataliwa. Hali yake ni: " + string(event.Status)})
	}

	event.Status = models.WelfareRejected
	event.RejectedBy = &userID
	event.RejectionReason = &req.Reason

	if err := tx.Save(&event).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukataa"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukataa"})
	}

	services.LogAudit(c, &userID, models.AuditRejectWelfareEvent, "welfare_events", &event.ID,
		map[string]interface{}{"status": "PENDING"},
		map[string]interface{}{"status": "REJECTED", "reason": req.Reason},
	)

	// Notify treasurer
	services.NotifyRole(models.RoleTreasurer, models.NotifWelfareRejected,
		"Tukio la Kijamii Limekataliwa",
		"Tukio la aina ya "+string(event.EventType)+" limekataliwa na Mwenyekiti. Sababu: "+req.Reason,
		"",
	)

	// Notify affected member
	var member models.Member
	if err := database.DB.First(&member, "id = ?", event.MemberID).Error; err == nil {
		var notifUserID string
		if member.UserID != nil {
			notifUserID = *member.UserID
		}
		if notifUserID == "" {
			notifUserID = member.RegisteredBy
		}
		if notifUserID != "" {
			services.NotifyUser(notifUserID, models.NotifWelfareRejected,
				"Tukio Lako la Kijamii Limekataliwa",
				"Tukio lako la aina ya "+string(event.EventType)+" limekataliwa. Sababu: "+req.Reason,
			)
		}
	}

	return c.JSON(fiber.Map{"message": "Tukio la kijamii limekataliwa", "data": event})
}

// ---------- TREASURER: Disburse welfare event to member ----------

func (h *WelfareHandler) DisburseEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := middleware.GetUserID(c)

	tx := database.DB.Begin()

	var event models.WelfareEvent
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&event, "id = ?", id).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Tukio la kijamii halijapatikana"})
	}

	if event.Status != models.WelfareApproved {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Tukio haliwezi kutolewa. Hali yake ni: " + string(event.Status)})
	}

	// Check if all contributions are collected (for member-funded events)
	if event.FundingSource == models.FundMemberContribution || event.FundingSource == models.FundBoth {
		var totalCollected decimal.Decimal
		tx.Model(&models.WelfareContribution{}).
			Where("event_id = ? AND status = ?", event.ID, models.WelfareContribPaid).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&totalCollected)

		var totalRequired decimal.Decimal
		tx.Model(&models.WelfareContribution{}).
			Where("event_id = ?", event.ID).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&totalRequired)

		if totalCollected.LessThan(totalRequired) {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": fmt.Sprintf("Michango haijakamilika. Imekusanywa: TZS %s, Inahitajika: TZS %s", formatMoney(totalCollected), formatMoney(totalRequired)),
			})
		}
	}

	// Calculate disbursement amount
	disbursementAmount := event.AmountApproved
	if disbursementAmount == nil || disbursementAmount.LessThanOrEqual(decimal.Zero) {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Kiasi cha kutolewa si sahihi"})
	}

	// Update event status to completed
	now := time.Now()
	event.Status = models.WelfareCompleted
	event.CompletedAt = &now

	if err := tx.Save(&event).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutoa fedha"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutoa fedha"})
	}

	services.LogAudit(c, &userID, "WELFARE_DISBURSE", "welfare_events", &event.ID,
		map[string]interface{}{"status": "APPROVED"},
		map[string]interface{}{"status": "COMPLETED", "amount": *disbursementAmount},
	)

	// Notify affected member
	var member models.Member
	if err := database.DB.First(&member, "id = ?", event.MemberID).Error; err == nil {
		var notifUserID string
		if member.UserID != nil {
			notifUserID = *member.UserID
		}
		if notifUserID == "" {
			notifUserID = member.RegisteredBy
		}
		if notifUserID != "" {
			services.NotifyUser(notifUserID, models.NotifWelfareCompleted,
				"Fedha za Kijamii Zimetolewa",
				"Fedha za TZS "+formatMoney(*disbursementAmount)+" kwa tukio lako la "+string(event.EventType)+" zimetolewa. Wasiliana na Mweka Hazina.",
			)
		}
	}

	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Fedha za TZS %s zimetolewa kwa mwanachama.", formatMoney(*disbursementAmount)),
		"data":    event,
	})
}

// ---------- TREASURER: Record welfare contribution payment ----------

func (h *WelfareHandler) RecordPayment(c *fiber.Ctx) error {
	eventID := c.Params("id")
	memberID := c.Params("memberId")

	var req models.RecordWelfarePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	userID := middleware.GetUserID(c)

	tx := database.DB.Begin()

	var contrib models.WelfareContribution
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("event_id = ? AND member_id = ?", eventID, memberID).
		First(&contrib).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mchango haujapatikana"})
	}

	if contrib.Status == models.WelfareContribPaid {
		tx.Rollback()
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Mchango huu tayari umelipwa"})
	}

	if contrib.Status == models.WelfareContribWaived {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mchango huu umesamehewa"})
	}

	now := time.Now()
	contrib.Status = models.WelfareContribPaid
	contrib.PaidAt = &now
	contrib.RecordedBy = &userID
	contrib.Amount = req.Amount

	if err := tx.Save(&contrib).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kurekodi malipo"})
	}

	// Shared completion check (pay path)
	if err := h.maybeCompleteWelfareEvent(tx, eventID); err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukamilisha tukio"})
	}

	// Reload event for completion notifications
	var event models.WelfareEvent
	tx.First(&event, "id = ?", eventID)

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kurekodi malipo"})
	}

	services.LogAudit(c, &userID, models.AuditRecordWelfarePayment, "welfare_contributions", &contrib.ID,
		map[string]interface{}{"status": "PENDING"},
		map[string]interface{}{"status": "PAID", "amount": req.Amount},
	)

	// Notify affected member
	var contribMember models.Member
	if err := database.DB.First(&contribMember, "id = ?", contrib.MemberID).Error; err == nil {
		var notifUserID string
		if contribMember.UserID != nil {
			notifUserID = *contribMember.UserID
		}
		if notifUserID == "" {
			notifUserID = contribMember.RegisteredBy
		}
		if notifUserID != "" {
			services.NotifyUser(notifUserID, models.NotifWelfarePayment,
				"Malipo ya Mchango wa Kijamii",
				"Malipo yako ya TZS "+formatMoney(req.Amount)+" kwa tukio la kijamii yamerekodiwa.",
			)
		}
	}

	// If event completed, notify everyone
	if event.Status == models.WelfareCompleted {
		var eventMember models.Member
		if err := database.DB.First(&eventMember, "id = ?", event.MemberID).Error; err == nil {
			var eventNotifUserID string
			if eventMember.UserID != nil {
				eventNotifUserID = *eventMember.UserID
			}
			if eventNotifUserID == "" {
				eventNotifUserID = eventMember.RegisteredBy
			}
			if eventNotifUserID != "" {
				services.NotifyUser(eventNotifUserID, models.NotifWelfareCompleted,
					"Tukio la Kijamii Limekamilika",
					"Tukio lako la aina ya "+string(event.EventType)+" limekamilika.",
				)
			}
		}
	}

	return c.JSON(fiber.Map{"message": "Malipo yamerekodiwa", "data": contrib})
}

// ---------- TREASURER: Waive a welfare contribution ----------

func (h *WelfareHandler) WaiveContribution(c *fiber.Ctx) error {
	eventID := c.Params("id")
	memberID := c.Params("memberId")
	userID := middleware.GetUserID(c)

	tx := database.DB.Begin()

	var contrib models.WelfareContribution
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("event_id = ? AND member_id = ?", eventID, memberID).
		First(&contrib).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mchango haujapatikana"})
	}

	if contrib.Status != models.WelfareContribPending {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mchango huu hauwezi kusamehewa. Hali yake ni: " + string(contrib.Status)})
	}

	contrib.Status = models.WelfareContribWaived
	if err := tx.Save(&contrib).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusamehe"})
	}

	// Same completion check as pay path: no PENDING remaining → COMPLETED
	if err := h.maybeCompleteWelfareEvent(tx, eventID); err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukamilisha tukio"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusamehe"})
	}

	services.LogAudit(c, &userID, models.AuditUpdate, "welfare_contributions", &contrib.ID,
		map[string]interface{}{"status": "PENDING"},
		map[string]interface{}{"status": "WAIVED"},
	)

	return c.JSON(fiber.Map{"message": "Mchango umesamehewa", "data": contrib})
}

// maybeCompleteWelfareEvent marks the event COMPLETED when no PENDING contributions remain.
// Shared by pay and waive paths. Locks the event row so concurrent pay/waive cannot both skip completion.
func (h *WelfareHandler) maybeCompleteWelfareEvent(tx *gorm.DB, eventID string) error {
	var event models.WelfareEvent
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&event, "id = ?", eventID).Error; err != nil {
		return err
	}

	var unpaidCount int64
	tx.Model(&models.WelfareContribution{}).
		Where("event_id = ? AND status = ?", eventID, models.WelfareContribPending).
		Count(&unpaidCount)

	if unpaidCount == 0 {
		if event.FundingSource == models.FundMemberContribution || event.FundingSource == models.FundBoth {
			if event.Status == models.WelfareApproved {
				now := time.Now()
				event.Status = models.WelfareCompleted
				event.CompletedAt = &now
				return tx.Save(&event).Error
			}
		}
	}
	return nil
}

// ---------- LIST: Events members can contribute to ----------

// ListContributeEvents returns APPROVED events funded (fully or partly) by
// member contributions — these are the "mifuko" a member can submit a
// contribution against from the Weka Mchango page.
// GET /api/v1/welfare/contribute-events
func (h *WelfareHandler) ListContributeEvents(c *fiber.Ctx) error {	var events []models.WelfareEvent
	database.DB.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name")
		}).
		Where("status = ? AND funding_source IN ?",
			models.WelfareApproved,
			[]models.WelfareFundingSource{models.FundMemberContribution, models.FundBoth}).
		Order("created_at DESC").
		Find(&events)

	return c.JSON(fiber.Map{"data": events, "total": len(events)})
}

// MyObligation returns the calling member's own contribution obligation
// (fixed per-member amount) for one welfare event — this is the exact
// amount the Weka Mchango form prefills and locks.
// GET /api/v1/welfare/events/:id/my-obligation
func (h *WelfareHandler) MyObligation(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var member models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).First(&member).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Huna usajili wa mwanachama"})
	}
	if !member.IsActive || member.ApprovalStatus != "approved" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Akaunti yako ya mwanachama haijaidhinishwa bado — subiri Katibu"})
	}

	var contrib models.WelfareContribution
	if err := database.DB.
		Preload("Event", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, event_type, description, status")
		}).
		Where("event_id = ? AND member_id = ?", c.Params("id"), member.ID).
		First(&contrib).Error; err != nil {
		// Distinguish "event not approved yet" from "no row for you" so the
		// UI can explain instead of showing a dead end.
		var event models.WelfareEvent
		if dbErr := database.DB.Where("id = ?", c.Params("id")).First(&event).Error; dbErr != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mfuko huu haujapatikana"})
		}
		if event.Status != models.WelfareApproved {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Mfuko huu bado haujafanyiwa approval — subiri kidogo"})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Huna kiwango kilichowekwa kwa mfuko huu"})
	}

	return c.JSON(fiber.Map{"data": contrib})
}

// ---------- LIST: Events (role-filtered) ----------

func (h *WelfareHandler) ListEvents(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 20}
	}

	status := c.Query("status")
	eventType := c.Query("event_type")
	role := middleware.GetUserRole(c)
	userID := middleware.GetUserID(c)

	query := database.DB.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name, phone")
		}).
		Preload("Creator", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}

	// Members can only see events affecting them
	if role == models.RoleMember {
		query = query.Where("member_id = (SELECT id FROM members WHERE user_id = ? AND deleted_at IS NULL LIMIT 1)", userID)
	}

	var total int64
	query.Model(&models.WelfareEvent{}).Count(&total)

	var events []models.WelfareEvent
	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("created_at DESC").Find(&events)

	return c.JSON(fiber.Map{
		"data":  events,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

// ---------- GET: Single event with contributions ----------

func (h *WelfareHandler) GetEvent(c *fiber.Ctx) error {
	id := c.Params("id")
	role := middleware.GetUserRole(c)
	userID := middleware.GetUserID(c)

	var event models.WelfareEvent
	if err := database.DB.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name, phone, user_id")
		}).
		Preload("Creator", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		}).
		Preload("Approver", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		}).
		Where("id = ?", id).First(&event).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Tukio la kijamii halijapatikana"})
	}

	// Members can only see events affecting them
	if role == models.RoleMember {
		var member models.Member
		if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).First(&member).Error; err != nil || member.ID != event.MemberID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Huna ruhusa ya kuona tukio hili"})
		}
	}

	// Get contributions
	var contributions []models.WelfareContribution
	database.DB.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name, phone")
		}).
		Where("event_id = ?", event.ID).
		Order("created_at ASC").
		Find(&contributions)

	// Calculate stats
	var totalPaid, totalPending decimal.Decimal
	var paidCount, pendingCount int64
	for _, c := range contributions {
		if c.Status == models.WelfareContribPaid {
			totalPaid = totalPaid.Add(c.Amount)
			paidCount++
		} else if c.Status == models.WelfareContribPending {
			totalPending = totalPending.Add(c.Amount)
			pendingCount++
		}
	}

	return c.JSON(fiber.Map{
		"data": event,
		"contributions": contributions,
		"stats": fiber.Map{
			"total_paid":     totalPaid,
			"total_pending":  totalPending,
			"paid_count":     paidCount,
			"pending_count":  pendingCount,
		},
	})
}

// ---------- LIST: My contributions (for members) ----------

func (h *WelfareHandler) MyContributions(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 20}
	}

	status := c.Query("status")

	// Find the member record for this user
	var member models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).First(&member).Error; err != nil {
		// L-2: non-members get an EMPTY list — the previous fallback leaked
		// all contributions to any non-member role hitting /my-contributions.
		return c.JSON(fiber.Map{"data": []models.WelfareContribution{}, "total": 0, "page": pq.Page, "limit": pq.Limit})
	}

	query := database.DB.
		Preload("Event", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, event_type, description, status")
		}).
		Where("member_id = ?", member.ID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&models.WelfareContribution{}).Count(&total)

	var contribs []models.WelfareContribution
	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("created_at DESC").Find(&contribs)

	return c.JSON(fiber.Map{
		"data":  contribs,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

// ---------- LIST: All contributions (for treasurer/chair/secretary) ----------

func (h *WelfareHandler) ListContributions(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 50}
	}

	status := c.Query("status")
	eventID := c.Query("event_id")

	return h.listContributionsAdmin(c, pq, status, eventID)
}

// ---------- DASHBOARD: Role-specific stats ----------

func (h *WelfareHandler) Dashboard(c *fiber.Ctx) error {
	role := middleware.GetUserRole(c)
	userID := middleware.GetUserID(c)

	var dash models.WelfareDashboard

	database.DB.Model(&models.WelfareEvent{}).Count(&dash.TotalEvents)
	database.DB.Model(&models.WelfareEvent{}).Where("status = ?", models.WelfarePending).Count(&dash.PendingApproval)
	database.DB.Model(&models.WelfareEvent{}).Where("status = ?", models.WelfareApproved).Count(&dash.ActiveEvents)
	database.DB.Model(&models.WelfareEvent{}).Where("status = ?", models.WelfareCompleted).Count(&dash.CompletedEvents)
	database.DB.Model(&models.WelfareEvent{}).Where("status = ?", models.WelfareRejected).Count(&dash.RejectedEvents)

	// Total collected from member contributions
	database.DB.Model(&models.WelfareContribution{}).
		Where("status = ?", models.WelfareContribPaid).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&dash.TotalCollected)

	// Total from treasury (sum of treasury_amount for completed/approved events with treasury funding)
	database.DB.Model(&models.WelfareEvent{}).
		Where("status IN ? AND funding_source IN ?", []string{"APPROVED", "COMPLETED"}, []string{"TREASURY", "BOTH"}).
		Select("COALESCE(SUM(treasury_amount), 0)").
		Scan(&dash.TotalFromTreasury)

	// Member-specific stats
	if role == models.RoleMember {
		var member models.Member
		if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).First(&member).Error; err == nil {
			database.DB.Model(&models.WelfareContribution{}).
				Where("member_id = ? AND status = ?", member.ID, models.WelfareContribPending).
				Count(&dash.MyPendingContributions)
			database.DB.Model(&models.WelfareContribution{}).
				Where("member_id = ? AND status = ?", member.ID, models.WelfareContribPaid).
				Count(&dash.MyPaidContributions)
		}
	}

	return c.JSON(fiber.Map{"data": dash})
}

// ---------- INTERNAL HELPERS ----------

// generateMemberContributions creates contribution obligations for all active members.
func (h *WelfareHandler) generateMemberContributions(tx *gorm.DB, event models.WelfareEvent) error {
	var activeMembers []models.Member
	if err := tx.Where("is_active = TRUE AND deleted_at IS NULL AND approval_status = 'approved'").Find(&activeMembers).Error; err != nil {
		return err
	}

	if len(activeMembers) == 0 {
		return fmt.Errorf("hakuna wanachama hai")
	}

	memberTotal := event.MemberAmount
	if event.FundingSource == models.FundMemberContribution {
		memberTotal = *event.AmountApproved
	}

	perMember := memberTotal.Div(decimal.NewFromInt(int64(len(activeMembers))))

	for _, m := range activeMembers {
		contrib := models.WelfareContribution{
			EventID:  event.ID,
			MemberID: m.ID,
			Amount:   perMember,
			Status:   models.WelfareContribPending,
		}
		if err := tx.Create(&contrib).Error; err != nil {
			return err
		}
	}

	// Notify members after contributions are created (notifications use DB, not tx)
	for _, m := range activeMembers {
		var notifUserID string
		if m.UserID != nil {
			notifUserID = *m.UserID
		}
		if notifUserID == "" {
			notifUserID = m.RegisteredBy
		}
		if notifUserID != "" {
			services.NotifyUser(notifUserID, models.NotifWelfareCreated,
				"Mchango Mpya wa Kijamii",
				"Unadaiwa TZS "+formatMoney(perMember)+" kwa tukio la aina ya "+string(event.EventType)+".",
			)
		}
	}

	return nil
}

func (h *WelfareHandler) listAllContributions(c *fiber.Ctx, pq models.PaginationQuery, status string) error {
	query := database.DB.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name, phone")
		}).
		Preload("Event", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, event_type, description, status")
		})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&models.WelfareContribution{}).Count(&total)

	var contribs []models.WelfareContribution
	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("created_at DESC").Find(&contribs)

	return c.JSON(fiber.Map{
		"data":  contribs,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

func (h *WelfareHandler) listContributionsAdmin(c *fiber.Ctx, pq models.PaginationQuery, status, eventID string) error {
	query := database.DB.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name, phone")
		}).
		Preload("Event", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, event_type, description, status")
		})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if eventID != "" {
		query = query.Where("event_id = ?", eventID)
	}

	var total int64
	query.Model(&models.WelfareContribution{}).Count(&total)

	var contribs []models.WelfareContribution
	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("created_at DESC").Find(&contribs)

	return c.JSON(fiber.Map{
		"data":  contribs,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}
