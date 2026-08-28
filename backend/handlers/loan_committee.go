package handlers

import (
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

type LoanCommitteeHandler struct{}

func NewLoanCommitteeHandler() *LoanCommitteeHandler {
	return &LoanCommitteeHandler{}
}

// ListMembers returns all active committee members (including automatic ones by role/position).
// De-duplicates by user_id so leaders who are also appointed appear once.
func (h *LoanCommitteeHandler) ListMembers(c *fiber.Ctx) error {
	var appointed []models.LoanCommitteeMember
	database.DB.
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, email, role")
		}).
		Preload("Appointer", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		}).
		Where("is_active = TRUE").
		Order("appointed_at ASC").
		Find(&appointed)

	var result []models.LoanCommitteeMemberResponse
	seen := make(map[string]struct{})

	// Automatic members from leadership positions
	var leaderPositions []models.UserPosition
	database.DB.
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, email, role")
		}).
		Where("position_type IN ? AND is_active = TRUE", leadershipPositions).
		Find(&leaderPositions)

	for _, p := range leaderPositions {
		if p.User.ID == "" {
			continue
		}
		if _, ok := seen[p.User.ID]; ok {
			continue
		}
		seen[p.User.ID] = struct{}{}
		email := ""
		if p.User.Email != nil {
			email = *p.User.Email
		}
		result = append(result, models.LoanCommitteeMemberResponse{
			UserID:    p.User.ID,
			UserName:  p.User.Name,
			UserEmail: email,
			UserRole:  string(p.PositionType),
			IsActive:  true,
		})
	}

	// Active users with leadership roles (legacy, not yet synced to positions)
	var leaders []models.User
	database.DB.Where("role IN ? AND status = ? AND deleted_at IS NULL AND is_active = TRUE",
		[]models.Role{models.RoleChair, models.RoleSecretary, models.RoleTreasurer},
		models.UserStatusActive,
	).Find(&leaders)
	for _, u := range leaders {
		if _, ok := seen[u.ID]; ok {
			continue
		}
		seen[u.ID] = struct{}{}
		email := ""
		if u.Email != nil {
			email = *u.Email
		}
		result = append(result, models.LoanCommitteeMemberResponse{
			UserID:    u.ID,
			UserName:  u.Name,
			UserEmail: email,
			UserRole:  string(u.Role),
			IsActive:  true,
		})
	}

	for _, m := range appointed {
		if m.User == nil {
			continue
		}
		if _, ok := seen[m.UserID]; ok {
			continue
		}
		seen[m.UserID] = struct{}{}
		email := ""
		if m.User.Email != nil {
			email = *m.User.Email
		}
		r := models.LoanCommitteeMemberResponse{
			ID:          m.ID,
			UserID:      m.UserID,
			UserName:    m.User.Name,
			UserEmail:   email,
			UserRole:    string(m.User.Role),
			AppointedAt: m.AppointedAt.Format("2006-01-02 15:04:05"),
			IsActive:    m.IsActive,
		}
		if m.Appointer != nil {
			r.AppointedBy = &m.Appointer.Name
		}
		result = append(result, r)
	}

	return c.JSON(fiber.Map{"data": result})
}

// AppointMember adds an ordinary member to the loan committee.
func (h *LoanCommitteeHandler) AppointMember(c *fiber.Ctx) error {
	var req models.AppointCommitteeMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	var targetUser models.User
	if err := database.DB.Where("id = ? AND is_active = TRUE AND deleted_at IS NULL", req.UserID).First(&targetUser).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mtumiaji hajapatikana"})
	}

	if targetUser.Role == models.RoleChair || targetUser.Role == models.RoleSecretary || targetUser.Role == models.RoleTreasurer {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Kiongozi huyu tayari ni mwenyeji wa kamati ya mikopo kwa jukumu lake",
		})
	}

	var existing models.LoanCommitteeMember
	err := database.DB.Where("user_id = ? AND is_active = TRUE", req.UserID).First(&existing).Error
	if err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"message": "Mtumiaji huyu tayari ni mwanachama wa kamati ya mikopo",
		})
	}

	var removed models.LoanCommitteeMember
	err = database.DB.Where("user_id = ? AND is_active = FALSE", req.UserID).First(&removed).Error
	if err == nil {
		userID := middleware.GetUserID(c)
		now := time.Now()
		removed.IsActive = true
		removed.RemovedAt = nil
		removed.AppointedBy = userID
		removed.AppointedAt = now
		database.DB.Save(&removed)

		services.LogAudit(c, &userID, models.AuditCommitteeAppoint, "loan_committee_members", &removed.ID, nil, map[string]interface{}{
			"user_id": req.UserID, "action": "re-appointed",
		})
		services.NotifyUser(req.UserID, models.NotifCommitteeAppoint, "Uteuzi wa Kamati ya Mikopo",
			"Umeuteuliwa tena kuwa mwanachama wa kamati ya mikopo.")

		return c.JSON(fiber.Map{
			"message": "Mwanachama ameuteuliwa tena kwenye kamati ya mikopo",
			"data":    removed,
		})
	}

	userID := middleware.GetUserID(c)
	member := models.LoanCommitteeMember{
		UserID:      req.UserID,
		AppointedBy: userID,
		IsActive:    true,
	}

	if err := database.DB.Create(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Imeshindikana kuteua mwanachama",
		})
	}

	services.LogAudit(c, &userID, models.AuditCommitteeAppoint, "loan_committee_members", &member.ID, nil, map[string]interface{}{
		"user_id": req.UserID, "action": "appointed",
	})
	services.NotifyUser(req.UserID, models.NotifCommitteeAppoint, "Uteuzi wa Kamati ya Mikopo",
		"Umeuteuliwa kuwa mwanachama wa kamati ya mikopo. Sasa unaweza kupitia na kuidhinisha maombi ya mikopo.")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Mwanachama ameuteuliwa kwenye kamati ya mikopo",
		"data":    member,
	})
}

// RemoveMember deactivates an appointed committee member.
func (h *LoanCommitteeHandler) RemoveMember(c *fiber.Ctx) error {
	id := c.Params("id")

	var member models.LoanCommitteeMember
	if err := database.DB.Where("id = ? AND is_active = TRUE", id).First(&member).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Mwanachama wa kamati hajapatikana",
		})
	}

	userID := middleware.GetUserID(c)
	now := time.Now()
	member.IsActive = false
	member.RemovedAt = &now

	if err := database.DB.Save(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Imeshindikana kumuondoa mwanachama",
		})
	}

	services.LogAudit(c, &userID, models.AuditCommitteeRemove, "loan_committee_members", &member.ID,
		map[string]interface{}{"is_active": true},
		map[string]interface{}{"is_active": false, "removed_at": now.Format(time.RFC3339)},
	)
	services.NotifyUser(member.UserID, models.NotifCommitteeRemove, "Uondoaji Kamati ya Mikopo",
		"Umeondolewa kutoka kamati ya mikopo.")

	return c.JSON(fiber.Map{
		"message": "Mwanachama ameondolewa kutoka kamati ya mikopo",
		"data":    member,
	})
}

// ListLoans returns loans that are pending review or under review by the committee.
func (h *LoanCommitteeHandler) ListLoans(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 20}
	}

	status := c.Query("status")

	query := database.DB.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name, phone")
		}).
		Preload("Reviewer", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		})

	if status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status IN ?", []models.LoanStatus{models.LoanPending, models.LoanUnderReview})
	}

	var total int64
	query.Model(&models.Loan{}).Count(&total)

	var loans []models.Loan
	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("applied_at DESC").Find(&loans)

	return c.JSON(fiber.Map{
		"data":  loans,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

// GetLoan returns a loan with its review history.
func (h *LoanCommitteeHandler) GetLoan(c *fiber.Ctx) error {
	id := c.Params("id")

	var loan models.Loan
	if err := database.DB.
		Preload("Member", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, member_no, full_name, phone")
		}).
		Preload("Reviewer", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		}).
		First(&loan, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mkopo haujapatikana"})
	}

	var reviews []models.LoanReview
	database.DB.
		Preload("Reviewer", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		}).
		Where("loan_id = ?", loan.ID).
		Order("created_at ASC").
		Find(&reviews)

	var contributions []models.Contribution
	database.DB.Where("member_id = ?", loan.MemberID).Order("paid_at DESC").Limit(12).Find(&contributions)

	var previousLoans []models.Loan
	database.DB.Where("member_id = ? AND id != ?", loan.MemberID, loan.ID).Order("applied_at DESC").Limit(5).Find(&previousLoans)

	var outstandingBalance float64
	database.DB.Model(&models.Loan{}).
		Where("member_id = ? AND status = ?", loan.MemberID, models.LoanOutstanding).
		Select("COALESCE(SUM(balance_remaining), 0)").
		Scan(&outstandingBalance)

	var reviewResponses []models.LoanReviewResponse
	for _, r := range reviews {
		rp := models.LoanReviewResponse{
			ID:         r.ID,
			LoanID:     r.LoanID,
			ReviewerID: r.ReviewerID,
			Decision:   string(r.Decision),
			Comments:   r.Comments,
		}
		if r.Reviewer != nil {
			rp.ReviewerName = r.Reviewer.Name
		}
		if r.ReviewedAt != nil {
			t := r.ReviewedAt.Format("2006-01-02 15:04:05")
			rp.ReviewedAt = &t
		}
		reviewResponses = append(reviewResponses, rp)
	}

	return c.JSON(fiber.Map{
		"data":                loan,
		"reviews":             reviewResponses,
		"contributions":       contributions,
		"previous_loans":      previousLoans,
		"outstanding_balance": outstandingBalance,
	})
}

// SubmitReview allows a committee member to approve or reject a loan.
func (h *LoanCommitteeHandler) SubmitReview(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.SubmitLoanReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	// Same eligibility as countActiveCommitteeMembers: leadership role/position or appointed
	if !h.isEligibleCommitteeVoter(database.DB, userID, role) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Wanachama wa kamati ya mikopo pekee ndio wanaweza kufanya ukaguzi",
		})
	}

	tx := database.DB.Begin()

	var loan models.Loan
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&loan, "id = ?", id).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mkopo haujapatikana"})
	}

	if loan.Status != models.LoanPending && loan.Status != models.LoanUnderReview {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Mkopo huu hauwezi kupitiwa. Hali yake ni: " + string(loan.Status),
		})
	}

	var existingReview models.LoanReview
	err := tx.Where("loan_id = ? AND reviewer_id = ?", loan.ID, userID).First(&existingReview).Error
	if err == nil && existingReview.Decision != models.ReviewPending {
		tx.Rollback()
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"message": "Tayari umetoa maamuzi yako kwa mkopo huu",
		})
	}

	now := time.Now()
	decision := models.LoanReviewDecision(req.Decision)

	if err == nil {
		existingReview.Decision = decision
		existingReview.Comments = req.Comments
		existingReview.ReviewedAt = &now
		if err := tx.Save(&existingReview).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Imeshindikana kuhifadhi ukaguzi",
			})
		}
	} else {
		review := models.LoanReview{
			LoanID:     loan.ID,
			ReviewerID: userID,
			Decision:   decision,
			Comments:   req.Comments,
			ReviewedAt: &now,
		}
		if err := tx.Create(&review).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Imeshindikana kuhifadhi ukaguzi",
			})
		}
	}

	if loan.Status == models.LoanPending {
		loan.Status = models.LoanUnderReview
		if err := tx.Save(&loan).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Imeshindikana kusasisha hali ya mkopo",
			})
		}
	}

	if decision == models.ReviewReject {
		reason := "Umekataliwa na kamati ya mikopo"
		if req.Comments != nil && *req.Comments != "" {
			reason = *req.Comments
		}
		loan.Status = models.LoanRejected
		loan.RejectionReason = &reason
		loan.ReviewedBy = &userID
		loan.ReviewedAt = &now
		if err := tx.Save(&loan).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Imeshindikana kukataa mkopo",
			})
		}
		if err := tx.Commit().Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Imeshindikana kukataa mkopo",
			})
		}

		services.LogAudit(c, &userID, models.AuditReject, "loans", &loan.ID,
			map[string]interface{}{"status": string(models.LoanUnderReview)},
			map[string]interface{}{"status": "REJECTED", "reason": reason},
		)
		h.notifyLoanApplicant(loan, models.NotifLoanRejected, "Mkopo Umekataliwa",
			"Mkopo wako wa TZS "+formatMoney(loan.Amount)+" umekataliwa na kamati ya mikopo. Sababu: "+reason)

		return c.JSON(fiber.Map{
			"message": "Ukaguzi umehifadhiwa. Mkopo umekataliwa.",
			"data":    loan,
		})
	}

	// Count eligible voters inside the same locked transaction
	totalCommittee := h.countActiveCommitteeMembers(tx)
	var approveCount int64
	tx.Model(&models.LoanReview{}).
		Where("loan_id = ? AND decision = ?", loan.ID, models.ReviewApprove).
		Count(&approveCount)

	if approveCount >= totalCommittee {
		loan.Status = models.LoanApproved
		loan.ApprovedAmount = &loan.Amount
		loan.ReviewedBy = &userID
		loan.ReviewedAt = &now
		if err := tx.Save(&loan).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Imeshindikana kuidhinisha mkopo",
			})
		}
		if err := tx.Commit().Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Imeshindikana kuidhinisha mkopo",
			})
		}

		services.LogAudit(c, &userID, models.AuditApprove, "loans", &loan.ID,
			map[string]interface{}{"status": string(models.LoanUnderReview)},
			map[string]interface{}{"status": "APPROVED", "approved_amount": loan.Amount},
		)
		h.notifyLoanApplicant(loan, models.NotifLoanApproved, "Mkopo Umeidhinishwa",
			"Mkopo wako wa TZS "+formatMoney(loan.Amount)+" umeidhinishwa na kamati nzima ya mikopo.")

		return c.JSON(fiber.Map{
			"message": "Ukaguzi umehifadhiwa. Mkopo umeidhinishwa na kamati nzima!",
			"data":    loan,
		})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Imeshindikana kuhifadhi ukaguzi",
		})
	}

	// Notify other committee members
	msg := "Mkopo wa TZS " + formatMoney(loan.Amount) + " umepitiwa na mwanachama mwingine wa kamati."
	services.NotifyRole(models.RoleChair, models.NotifLoanUnderReview, "Ukaguzi Mpya wa Mkopo", msg, userID)
	services.NotifyRole(models.RoleSecretary, models.NotifLoanUnderReview, "Ukaguzi Mpya wa Mkopo", msg, userID)
	services.NotifyRole(models.RoleTreasurer, models.NotifLoanUnderReview, "Ukaguzi Mpya wa Mkopo", msg, userID)

	var appointed []models.LoanCommitteeMember
	database.DB.Where("is_active = TRUE AND user_id != ?", userID).Find(&appointed)
	for _, m := range appointed {
		services.NotifyUser(m.UserID, models.NotifLoanUnderReview, "Ukaguzi Mpya wa Mkopo", msg)
	}

	return c.JSON(fiber.Map{
		"message": "Ukaguzi wako umehifadhiwa. Mkopo bado unasubiri ukaguzi wa wanachama wengine.",
		"data":    loan,
	})
}

// GetDashboard returns summary stats for the committee dashboard.
func (h *LoanCommitteeHandler) GetDashboard(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var dash models.LoanCommitteeDashboard

	database.DB.Model(&models.LoanReview{}).
		Where("reviewer_id = ? AND decision = 'PENDING'", userID).
		Count(&dash.PendingReviews)

	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanUnderReview).
		Count(&dash.LoansUnderReview)

	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanApproved).
		Count(&dash.ApprovedLoans)

	database.DB.Model(&models.Loan{}).
		Where("status = ?", models.LoanRejected).
		Count(&dash.RejectedLoans)

	database.DB.Model(&models.LoanReview{}).
		Where("reviewer_id = ? AND decision != 'PENDING'", userID).
		Count(&dash.MyReviews)

	// Distinct eligible voters (same denominator as SubmitReview unanimous check)
	dash.CommitteeMembers = h.countActiveCommitteeMembers(database.DB)

	return c.JSON(fiber.Map{"data": dash})
}

// GetHistory returns the review history for all loans.
func (h *LoanCommitteeHandler) GetHistory(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 50}
	}

	type Row struct {
		LoanID        string  `json:"loan_id"`
		ApplicantName string          `json:"applicant_name"`
		MemberNo      string          `json:"member_no"`
		Amount        decimal.Decimal `json:"amount"`
		Status        string          `json:"status"`
		ReviewedBy    string          `json:"reviewed_by"`
		Decision      string          `json:"decision"`
		Comments      *string         `json:"comments,omitempty"`
		ReviewedAt    string          `json:"reviewed_at"`
	}

	var total int64
	database.DB.Model(&models.LoanReview{}).Where("decision != 'PENDING'").Count(&total)

	var rows []Row
	database.DB.Raw(`
		SELECT lr.loan_id,
		       m.full_name AS applicant_name,
		       m.member_no,
		       l.amount,
		       l.status,
		       u.name AS reviewed_by,
		       lr.decision,
		       lr.comments,
		       lr.reviewed_at
		FROM loan_reviews lr
		JOIN loans l ON l.id = lr.loan_id
		JOIN members m ON m.id = l.member_id
		JOIN users u ON u.id = lr.reviewer_id
		WHERE lr.decision != 'PENDING'
		ORDER BY lr.reviewed_at DESC
		LIMIT ? OFFSET ?
	`, pq.Limit, pq.GetOffset()).Scan(&rows)

	return c.JSON(fiber.Map{
		"data":  rows,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

// GetReport generates the committee activity report.
func (h *LoanCommitteeHandler) GetReport(c *fiber.Ctx) error {
	var report models.CommitteeActivityReport

	database.DB.Model(&models.LoanReview{}).Where("decision != 'PENDING'").Count(&report.TotalReviews)

	var approvals, rejections int64
	database.DB.Model(&models.LoanReview{}).Where("decision = 'APPROVE'").Count(&approvals)
	database.DB.Model(&models.LoanReview{}).Where("decision = 'REJECT'").Count(&rejections)

	if report.TotalReviews > 0 {
		report.ApprovalRate = float64(approvals) / float64(report.TotalReviews) * 100
		report.RejectionRate = float64(rejections) / float64(report.TotalReviews) * 100
	}

	type MemberCount struct {
		UserID     string
		UserName   string
		Reviews    int64
		Approvals  int64
		Rejections int64
	}
	var memberCounts []MemberCount
	database.DB.Raw(`
		SELECT lr.reviewer_id AS user_id,
		       u.name AS user_name,
		       COUNT(*) AS reviews,
		       SUM(CASE WHEN lr.decision = 'APPROVE' THEN 1 ELSE 0 END) AS approvals,
		       SUM(CASE WHEN lr.decision = 'REJECT' THEN 1 ELSE 0 END) AS rejections
		FROM loan_reviews lr
		JOIN users u ON u.id = lr.reviewer_id
		WHERE lr.decision != 'PENDING'
		GROUP BY lr.reviewer_id, u.name
		ORDER BY reviews DESC
	`).Scan(&memberCounts)

	for _, mc := range memberCounts {
		report.ReviewsByMember = append(report.ReviewsByMember, models.CommitteeMemberReviewCount{
			UserID:     mc.UserID,
			UserName:   mc.UserName,
			Reviews:    mc.Reviews,
			Approvals:  mc.Approvals,
			Rejections: mc.Rejections,
		})
	}

	var leaders []models.User
	database.DB.Where("role IN ? AND is_active = TRUE AND deleted_at IS NULL",
		[]models.Role{models.RoleChair, models.RoleSecretary, models.RoleTreasurer}).
		Find(&leaders)

	for _, u := range leaders {
		report.CommitteeComposition = append(report.CommitteeComposition, models.CommitteeCompositionEntry{
			UserID:      u.ID,
			UserName:    u.Name,
			Role:        string(u.Role),
			AppointedAt: u.CreatedAt.Format("2006-01-02"),
			Type:        "automatic",
		})
	}

	var appointed []models.LoanCommitteeMember
	database.DB.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, name, role")
	}).Where("is_active = TRUE AND deleted_at IS NULL").Find(&appointed)

	for _, m := range appointed {
		if m.User == nil {
			continue
		}
		report.CommitteeComposition = append(report.CommitteeComposition, models.CommitteeCompositionEntry{
			UserID:      m.UserID,
			UserName:    m.User.Name,
			Role:        string(m.User.Role),
			AppointedAt: m.AppointedAt.Format("2006-01-02"),
			Type:        "appointed",
		})
	}

	type HistoryRow struct {
		LoanID        string  `json:"loan_id"`
		ApplicantName string          `json:"applicant_name"`
		MemberNo      string          `json:"member_no"`
		Amount        decimal.Decimal `json:"amount"`
		Status        string          `json:"status"`
		ReviewedBy    string          `json:"reviewed_by"`
		Decision      string          `json:"decision"`
		Comments      *string         `json:"comments,omitempty"`
		ReviewedAt    string          `json:"reviewed_at"`
	}

	var history []HistoryRow
	database.DB.Raw(`
		SELECT lr.loan_id,
		       m.full_name AS applicant_name,
		       m.member_no,
		       l.amount,
		       l.status,
		       u.name AS reviewed_by,
		       lr.decision,
		       lr.comments,
		       lr.reviewed_at
		FROM loan_reviews lr
		JOIN loans l ON l.id = lr.loan_id
		JOIN members m ON m.id = l.member_id
		JOIN users u ON u.id = lr.reviewer_id
		WHERE lr.decision != 'PENDING'
		ORDER BY lr.reviewed_at DESC
		LIMIT 50
	`).Scan(&history)

	for _, hr := range history {
		report.ReviewHistory = append(report.ReviewHistory, models.LoanCommitteeHistoryRow{
			LoanID:        hr.LoanID,
			ApplicantName: hr.ApplicantName,
			MemberNo:      hr.MemberNo,
			Amount:        hr.Amount,
			Status:        hr.Status,
			ReviewedBy:    hr.ReviewedBy,
			Decision:      hr.Decision,
			Comments:      hr.Comments,
			ReviewedAt:    hr.ReviewedAt,
		})
	}

	return c.JSON(fiber.Map{"data": report})
}

// IsCommitteeMember checks if the current user is an eligible committee voter
// (same rules as SubmitReview / RequireLoanCommitteeMember).
func (h *LoanCommitteeHandler) IsCommitteeMember(c *fiber.Ctx) error {
	role := middleware.GetUserRole(c)
	userID := middleware.GetUserID(c)
	return c.JSON(fiber.Map{
		"is_committee_member": h.isEligibleCommitteeVoter(database.DB, userID, role),
	})
}

// GetPendingLoansCount returns the count of loans pending committee review.
func (h *LoanCommitteeHandler) GetPendingLoansCount(c *fiber.Ctx) error {
	var count int64
	database.DB.Model(&models.Loan{}).
		Where("status IN ?", []models.LoanStatus{models.LoanPending, models.LoanUnderReview}).
		Count(&count)

	return c.JSON(fiber.Map{"count": count})
}

// --- Helper functions ---

var leadershipPositions = []models.PositionType{
	models.PositionChairperson, models.PositionSecretary, models.PositionTreasurer,
}

// isEligibleCommitteeVoter matches who may vote: leadership role, leadership position, or appointed.
// System admin is not a committee voter (may still access committee routes via middleware for oversight).
func (h *LoanCommitteeHandler) isEligibleCommitteeVoter(db *gorm.DB, userID string, role models.Role) bool {
	if role == models.RoleChair || role == models.RoleSecretary || role == models.RoleTreasurer {
		return true
	}
	var posCount int64
	db.Model(&models.UserPosition{}).
		Where("user_id = ? AND position_type IN ? AND is_active = TRUE", userID, leadershipPositions).
		Count(&posCount)
	if posCount > 0 {
		return true
	}
	var appointed int64
	db.Model(&models.LoanCommitteeMember{}).
		Where("user_id = ? AND is_active = TRUE", userID).
		Count(&appointed)
	return appointed > 0
}

// countActiveCommitteeMembers returns distinct eligible voters (positions ∪ role leaders ∪ appointed).
// Must be called with the same DB/tx used for the review transaction when finalizing.
func (h *LoanCommitteeHandler) countActiveCommitteeMembers(db *gorm.DB) int64 {
	if db == nil {
		db = database.DB
	}
	eligible := make(map[string]struct{})

	// Leadership positions
	var positions []models.UserPosition
	db.Where("position_type IN ? AND is_active = TRUE", leadershipPositions).Find(&positions)
	for _, p := range positions {
		eligible[p.UserID] = struct{}{}
	}

	// Active users with leadership roles (covers any not yet synced to positions)
	var leaders []models.User
	db.Where("role IN ? AND status = ? AND deleted_at IS NULL AND is_active = TRUE",
		[]models.Role{models.RoleChair, models.RoleSecretary, models.RoleTreasurer},
		models.UserStatusActive,
	).Select("id").Find(&leaders)
	for _, u := range leaders {
		eligible[u.ID] = struct{}{}
	}

	// Appointed committee members
	var appointed []models.LoanCommitteeMember
	db.Where("is_active = TRUE AND deleted_at IS NULL").Find(&appointed)
	for _, a := range appointed {
		eligible[a.UserID] = struct{}{}
	}

	return int64(len(eligible))
}

func (h *LoanCommitteeHandler) notifyLoanApplicant(loan models.Loan, notifType models.NotificationType, title, message string) {
	var member models.Member
	if err := database.DB.First(&member, "id = ?", loan.MemberID).Error; err != nil {
		return
	}
	var notifUserID string
	if member.UserID != nil {
		notifUserID = *member.UserID
	}
	if notifUserID == "" {
		notifUserID = member.RegisteredBy
	}
	if notifUserID != "" {
		services.NotifyUser(notifUserID, notifType, title, message)
	}
}
