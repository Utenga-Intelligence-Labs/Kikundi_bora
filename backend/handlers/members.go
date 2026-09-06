package handlers

import (
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type MemberHandler struct{}

func NewMemberHandler() *MemberHandler {
	return &MemberHandler{}
}

func (h *MemberHandler) List(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		pq = models.PaginationQuery{Page: 1, Limit: 20}
	}

	q := c.Query("q")
	search := c.Query("search")
	if search != "" {
		q = search
	}

	var members []models.Member
	query := database.DB.Where("deleted_at IS NULL").
		Preload("Registrar", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, email, role")
		}).
		Preload("Approver", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, role")
		}).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, phone, role, status")
		})

	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	// Approval-status filter: status=pending is the katibu approval queue
	statusFilter := c.Query("status")
	if statusFilter != "" {
		query = query.Where("approval_status = ?", statusFilter)
	}

	if q != "" {
		searchPattern := likePattern(q)
		query = query.Where("LOWER(full_name) LIKE ? ESCAPE '\\' OR phone LIKE ? ESCAPE '\\' OR LOWER(member_no) LIKE ? ESCAPE '\\'",
			searchPattern, searchPattern, searchPattern)
	}

	// Clone query for counting to preserve all filters
	countQuery := database.DB.Model(&models.Member{}).Where("deleted_at IS NULL")
	if userID := c.Query("user_id"); userID != "" {
		countQuery = countQuery.Where("user_id = ?", userID)
	}
	if statusFilter != "" {
		countQuery = countQuery.Where("approval_status = ?", statusFilter)
	}
	if q != "" {
		searchPattern := likePattern(q)
		countQuery = countQuery.Where("LOWER(full_name) LIKE ? ESCAPE '\\' OR phone LIKE ? ESCAPE '\\' OR LOWER(member_no) LIKE ? ESCAPE '\\'",
			searchPattern, searchPattern, searchPattern)
	}
	var total int64
	countQuery.Count(&total)

	query.Offset(pq.GetOffset()).Limit(pq.Limit).Order("created_at DESC").Find(&members)

	return c.JSON(fiber.Map{
		"data":  members,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

func (h *MemberHandler) Get(c *fiber.Ctx) error {
	id := c.Params("id")

	var member models.Member
	if err := database.DB.Where("deleted_at IS NULL").
		Preload("Registrar", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, email, role")
		}).
		Where("id = ?", id).First(&member).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama hajapatikana"})
	}

	return c.JSON(member)
}

func (h *MemberHandler) Create(c *fiber.Ctx) error {
	var req models.CreateMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	// Normalize to E.164 at entry so SMS delivery never fails on formatting.
	// Invalid numbers are rejected here, not silently stored.
	phoneE164, err := services.NormalizeTanzanianPhone(req.Phone)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Namba ya simu si sahihi — tumia 07... (Tanzania)"})
	}
	req.Phone = phoneE164

	var phoneCount int64
	database.DB.Model(&models.Member{}).Where("phone = ? AND deleted_at IS NULL", req.Phone).Count(&phoneCount)
	if phoneCount > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Namba ya simu hiyo tayari imesajiliwa"})
	}

	joinedAt, err := time.Parse("2006-01-02", req.JoinedAt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Tarehe si sahihi"})
	}

	userID := middleware.GetUserID(c)

	tx := database.DB.Begin()
	defer tx.Rollback()

	memberNo, err := database.NextMemberNo(tx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutengeneza namba ya uanachama"})
	}

	// Consolidated flow: the member submission ALSO creates the linked login
	// account (PENDING) in the same transaction — one approval by the katibu
	// then activates both.
	tempPassword := models.DefaultTempPassword()
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Hitilafu ya mfumo"})
	}
	newUser := models.User{
		Name:               req.FullName,
		Phone:              req.Phone,
		Email:              req.Email,
		Password:           string(hashedPwd),
		Role:               models.RoleMember,
		Status:             models.UserStatusPending,
		MustChangePassword: true,
		IsActive:           true,
		CreatedBy:          &userID,
	}
	if err := tx.Create(&newUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunda akaunti ya kuingia"})
	}
	if err := upsertUserPosition(tx, newUser.ID, models.RoleMember); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuweka nafasi ya mtumiaji"})
	}

	member := models.Member{
		MemberNo:       memberNo,
		UserID:         &newUser.ID,
		FullName:       req.FullName,
		Phone:          req.Phone,
		Address:        req.Address,
		Gender:         req.Gender,
		Occupation:     req.Occupation,
		Email:          req.Email,
		NextOfKinName:  req.NextOfKinName,
		NextOfKinPhone: req.NextOfKinPhone,
		PhotoURL:       req.PhotoURL,
		JoinedAt:       joinedAt,
		IsActive:       false, // activated on katibu approval
		RegisteredBy:   userID,
		ApprovalStatus: models.MemberApprovalPending,
	}

	if err := tx.Create(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusajili mwanachama"})
	}

	// GORM omits false on insert (column default true) — force is_active off
	// until katibu approval.
	if err := tx.Model(&member).Update("is_active", false).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusajili mwanachama"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusajili mwanachama"})
	}

	services.LogAudit(c, &userID, models.AuditCreate, "members", &member.ID, nil, map[string]interface{}{
		"member_no": memberNo, "full_name": member.FullName, "phone": member.Phone,
		"approval_status": member.ApprovalStatus, "linked_user_id": newUser.ID,
	})

	// Notify katibu — approval queue
	services.NotifyRole(models.RoleSecretary, models.NotifSystem,
		"Mwanachama Mpya Unasubiri",
		"Mwanachama mpya "+member.FullName+" ("+memberNo+") amesajiliwa na unasubiri idhini yako.",
		"",
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Mwanachama amesajiliwa. Unasubiri idhini ya Katibu.",
		"data":    member,
	})
}

// ApproveMember marks a pending member approved (katibu only).
// PATCH /api/v1/members/:id/approve
func (h *MemberHandler) ApproveMember(c *fiber.Ctx) error {
	var member models.Member
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", c.Params("id")).
		First(&member).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama hajapatikana"})
	}

	if member.ApprovalStatus != models.MemberApprovalPending {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"message": "Mwanachama huu hashughulikiwi. Hali yake: " + member.ApprovalStatus,
		})
	}

	userID := middleware.GetUserID(c)
	now := time.Now()
	member.ApprovalStatus = models.MemberApprovalApproved
	member.ApprovedBy = &userID
	member.ApprovedAt = &now
	member.IsActive = true
	if err := database.DB.Save(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuidhinisha"})
	}

	// One approval activates BOTH the member record and its linked login
	// account (consolidated flow).
	if member.UserID != nil {
		database.DB.Model(&models.User{}).
			Where("id = ?", *member.UserID).
			Updates(map[string]interface{}{
				"status":             models.UserStatusActive,
				"is_active":          true,
				"must_change_password": true,
			})
	}

	services.LogAudit(c, &userID, models.AuditApprove, "members", &member.ID,
		map[string]interface{}{"approval_status": models.MemberApprovalPending},
		map[string]interface{}{"approval_status": models.MemberApprovalApproved},
	)

	// Notify the registrar (mwenyekiti who submitted)
	services.NotifyUser(member.RegisteredBy, models.NotifSystem,
		"Mwanachama Ameidhinishwa",
		member.FullName+" ("+member.MemberNo+") ameidhinishwa na Katibu. Sasa anatumika kikamilifu.",
	)

	return c.JSON(fiber.Map{"message": "Mwanachama ameidhinishwa.", "data": member})
}

// RejectMember marks a pending member rejected — reason required (katibu only).
// PATCH /api/v1/members/:id/reject
func (h *MemberHandler) RejectMember(c *fiber.Ctx) error {
	var req models.RejectMemberRequest
	if err := c.BodyParser(&req); err != nil || req.Reason == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Sababu ya kukataa inahitajika"})
	}

	var member models.Member
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", c.Params("id")).
		First(&member).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama hajapatikana"})
	}

	if member.ApprovalStatus != models.MemberApprovalPending {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"message": "Mwanachama huu hashughulikiwi. Hali yake: " + member.ApprovalStatus,
		})
	}

	userID := middleware.GetUserID(c)
	now := time.Now()
	member.ApprovalStatus = models.MemberApprovalRejected
	member.ApprovedBy = &userID
	member.ApprovedAt = &now
	member.RejectionReason = &req.Reason
	member.IsActive = false
	if err := database.DB.Save(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukataa"})
	}

	// Reject also disarms the linked login account so it can never be
	// activated separately.
	if member.UserID != nil {
		database.DB.Model(&models.User{}).
			Where("id = ?", *member.UserID).
			Updates(map[string]interface{}{"status": models.UserStatusRejected, "is_active": false})
	}

	services.LogAudit(c, &userID, models.AuditReject, "members", &member.ID,
		map[string]interface{}{"approval_status": models.MemberApprovalPending},
		map[string]interface{}{"approval_status": models.MemberApprovalRejected, "reason": req.Reason},
	)

	services.NotifyUser(member.RegisteredBy, models.NotifSystem,
		"Mwanachama Amekataliwa",
		member.FullName+" ("+member.MemberNo+") amekataliwa na Katibu. Sababu: "+req.Reason,
	)

	return c.JSON(fiber.Map{"message": "Mwanachama amekataliwa.", "data": member})
}

func (h *MemberHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.UpdateMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	var member models.Member
	if err := database.DB.Where("deleted_at IS NULL").First(&member, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama hajapatikana"})
	}

	oldVals := map[string]interface{}{
		"full_name": member.FullName,
		"phone":     member.Phone,
		"is_active": member.IsActive,
	}

	if req.FullName != nil {
		member.FullName = *req.FullName
	}
	if req.Phone != nil {
		phoneE164, err := services.NormalizeTanzanianPhone(*req.Phone)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Namba ya simu si sahihi — tumia 07... (Tanzania)"})
		}
		var phoneCount int64
		database.DB.Model(&models.Member{}).Where("phone = ? AND id != ? AND deleted_at IS NULL", phoneE164, member.ID).Count(&phoneCount)
		if phoneCount > 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Namba ya simu hiyo tayari imesajiliwa"})
		}
		member.Phone = phoneE164
	}
	if req.Address != nil {
		member.Address = req.Address
	}
	// NOTE: is_active is intentionally NOT handled here. Activating /
	// deactivating members is a katibu-only power via POST
	// /members/:id/toggle-active, so it cannot be smuggled through edits.

	if err := database.DB.Save(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kubadilisha"})
	}

	userID := middleware.GetUserID(c)
	services.LogAudit(c, &userID, models.AuditUpdate, "members", &member.ID, oldVals, map[string]interface{}{
		"full_name": member.FullName, "phone": member.Phone, "is_active": member.IsActive,
	})

	return c.JSON(fiber.Map{"message": "Mwanachama amebadilishwa", "data": member})
}

func (h *MemberHandler) Delete(c *fiber.Ctx) error {
	// Admin is NOT allowed to delete members — only Chair can
	role := middleware.GetUserRole(c)
	if role == models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Msimamizi hana ruhusa ya kufuta mwanachama"})
	}

	id := c.Params("id")

	var member models.Member
	if err := database.DB.First(&member, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama hajapatikana"})
	}

	database.DB.Delete(&member)

	userID := middleware.GetUserID(c)
	services.LogAudit(c, &userID, models.AuditDelete, "members", &member.ID, nil, nil)

	return c.JSON(fiber.Map{"message": "Mwanachama amefutwa"})
}

// CreateLogin creates a linked login account for a member that has none
// (e.g. seeded or imported members). Chair only.
// POST /api/v1/members/:id/create-login
func (h *MemberHandler) CreateLogin(c *fiber.Ctx) error {
	id := c.Params("id")

	var member models.Member
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&member).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama hajapatikana"})
	}
	if member.UserID != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Mwanachama huyu tayari ana akaunti ya kuingia"})
	}

	actorID := middleware.GetUserID(c)

	// If a user with the same phone already exists, link it instead of
	// creating a duplicate.
	var existing models.User
	if err := database.DB.Where("phone = ? AND deleted_at IS NULL", member.Phone).First(&existing).Error; err == nil {
		member.UserID = &existing.ID
		if err := database.DB.Save(&member).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunganisha akaunti"})
		}
		services.LogAudit(c, &actorID, models.AuditUpdate, "members", &member.ID, nil, map[string]interface{}{
			"action": "link_login", "user_id": existing.ID,
		})
		return c.JSON(fiber.Map{"message": "Akaunti iliyopo imeunganishwa na mwanachama. Tumia 'Weka upya nenosiri' kumpa nenosiri la muda."})
	}

	tempPassword := models.DefaultTempPassword()
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Hitilafu ya mfumo"})
	}

	// Mirror the member's standing: already-approved members get an active
	// login right away, others go through katibu approval like new members.
	userStatus := models.UserStatusPending
	if member.ApprovalStatus == "approved" && member.IsActive {
		userStatus = models.UserStatusActive
	}

	tx := database.DB.Begin()
	defer tx.Rollback()

	newUser := models.User{
		Name:               member.FullName,
		Phone:              member.Phone,
		Email:              member.Email,
		Password:           string(hashedPwd),
		Role:               models.RoleMember,
		Status:             userStatus,
		MustChangePassword: true,
		IsActive:           true,
		CreatedBy:          &actorID,
	}
	if err := tx.Create(&newUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunda akaunti ya kuingia"})
	}
	if err := upsertUserPosition(tx, newUser.ID, models.RoleMember); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuweka nafasi ya mtumiaji"})
	}
	member.UserID = &newUser.ID
	if err := tx.Save(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunganisha akaunti"})
	}
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi"})
	}

	services.LogAudit(c, &actorID, models.AuditCreate, "users", &newUser.ID, nil, map[string]interface{}{
		"action": "chair_create_login", "member_id": member.ID,
	})
	services.NotifyUser(newUser.ID, models.NotifSystem,
		"Akaunti ya Kuingia",
		"Mwenyekiti amekutengenezea akaunti ya kuingia. Tumia nenosiri la muda ulilopewa; utakazwa kuweka nenosiri jipya.",
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":       "Akaunti ya kuingia imetengenezwa. Mpe mwanachama nenosiri la muda (halitaonekana tena).",
		"temp_password": tempPassword,
	})
}

// ToggleActive flips a member's active flag. Katibu ONLY — neither
// mwenyekiti nor anyone else may disable/enable members.
// POST /api/v1/members/:id/toggle-active
func (h *MemberHandler) ToggleActive(c *fiber.Ctx) error {
	id := c.Params("id")

	var member models.Member
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", id).First(&member).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama hajapatikana"})
	}

	member.IsActive = !member.IsActive
	if err := database.DB.Save(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kubadilisha hali"})
	}

	userID := middleware.GetUserID(c)
	services.LogAudit(c, &userID, models.AuditUpdate, "members", &member.ID,
		nil, map[string]interface{}{"is_active": member.IsActive})

	msg := "Mwanachama ameilishwa."
	if !member.IsActive {
		msg = "Mwanachama amezimwa."
	}
	return c.JSON(fiber.Map{"message": msg, "data": member})
}
