package handlers

import (
	"fmt"
	"strings"
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type UserManagementHandler struct{}

func NewUserManagementHandler() *UserManagementHandler {
	return &UserManagementHandler{}
}

// CreateUser creates a new user with minimal details (Chair only).
// User is created in PENDING status with a one-time temp password returned to the chair.
// Breaking response field: temp_password (plaintext, shown once to chair only).
func (h *UserManagementHandler) CreateUser(c *fiber.Ctx) error {
	actorID := middleware.GetUserID(c)

	var req models.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	phone := strings.TrimSpace(req.Phone)

	// Check phone uniqueness
	var count int64
	database.DB.Model(&models.User{}).Where("phone = ? AND deleted_at IS NULL", phone).Count(&count)
	if count > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Nambari ya simu hii tayari imesajiliwa"})
	}

	// Default role to member if not specified, and validate role
	role := req.Role
	if role == "" {
		role = models.RoleMember
	}
	// Prevent chair from creating admin or elevated roles
	allowedRoles := map[models.Role]bool{
		models.RoleMember:    true,
		models.RoleTreasurer: true,
		models.RoleSecretary: true,
		models.RoleChair:     true,
	}
	if !allowedRoles[role] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Jukumu la mtumiaji si sahihi. Chagua: member, treasurer, secretary, au chair.",
		})
	}

	// Generate temp password once; hash only that value; return plaintext to chair once
	tempPassword := models.DefaultTempPassword()
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Hitilafu ya mfumo"})
	}

	user := models.User{
		Name:               strings.TrimSpace(req.FullName),
		Phone:              phone,
		Password:           string(hashedPwd),
		Role:               role,
		Status:             models.UserStatusPending,
		MustChangePassword: true,
		IsActive:           true,
		CreatedBy:          &actorID,
	}

	tx := database.DB.Begin()

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunda mtumiaji"})
	}

	// Sync UserPosition for leadership roles so RequirePosition money routes work
	if err := upsertUserPosition(tx, user.ID, role); err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuweka nafasi ya mtumiaji"})
	}

	// Auto-create linked member so they appear on /wanachama
	if err := database.EnsureMemberForUser(tx, user, actorID); err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuunda mtumiaji"})
	}

	// Log audit
	services.LogAudit(c, &actorID, models.AuditUserCreated, "users", &user.ID, nil, map[string]interface{}{
		"name": user.Name, "phone": user.Phone, "role": user.Role, "status": user.Status,
	})

	// Notify all secretaries about new user
	services.NotifyRole(models.RoleSecretary, models.NotifUserCreated,
		"Mtumiaji Mpya Ameundwa",
		fmt.Sprintf("Mtumiaji mpya \"%s\" ameundwa na Mwenyekiti. Anahitaji kuidhinishwa.", user.Name),
		user.ID,
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":       "Mtumiaji ameundwa. Anasubiri kuidhinishwa na Katibu. Mpe nenosiri la muda moja kwa moja (halitaonekana tena).",
		"data":          user,
		"temp_password": tempPassword,
	})
}

// ListPending returns all users with PENDING status (Secretary only).
func (h *UserManagementHandler) ListPending(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	offset := pq.GetOffset()

	var users []models.User
	var total int64

	query := database.DB.Model(&models.User{}).Where("status = ? AND deleted_at IS NULL", models.UserStatusPending)
	query.Count(&total)

	query.Order("created_at DESC").
		Offset(offset).
		Limit(pq.Limit).
		Find(&users)

	return c.JSON(fiber.Map{
		"data":  users,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

// ListUsers returns all users with optional filters (Chair/Secretary).
func (h *UserManagementHandler) ListUsers(c *fiber.Ctx) error {
	var pq models.PaginationQuery
	if err := c.QueryParser(&pq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	offset := pq.GetOffset()

	status := c.Query("status")
	role := c.Query("role")
	search := c.Query("q")

	query := database.DB.Model(&models.User{}).Where("deleted_at IS NULL")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if search != "" {
		searchPattern := likePattern(search)
		query = query.Where("LOWER(name) LIKE ? ESCAPE '\\' OR phone LIKE ? ESCAPE '\\' OR LOWER(email) LIKE ? ESCAPE '\\'", searchPattern, searchPattern, searchPattern)
	}

	var total int64
	query.Count(&total)

	var users []models.User
	query.Order("created_at DESC").
		Offset(offset).
		Limit(pq.Limit).
		Find(&users)

	return c.JSON(fiber.Map{
		"data":  users,
		"total": total,
		"page":  pq.Page,
		"limit": pq.Limit,
	})
}

// ApproveUser approves a pending user (Secretary only).
func (h *UserManagementHandler) ApproveUser(c *fiber.Ctx) error {
	actorID := middleware.GetUserID(c)
	userID := c.Params("id")

	var req models.ApproveUserRequest
	c.BodyParser(&req) // optional, no error if empty

	var user models.User
	if err := database.DB.Where("deleted_at IS NULL").First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mtumiaji hajapatikana"})
	}

	if user.Status != models.UserStatusPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mtumiaji hana hali ya kusubiri. Hali ya sasa: " + user.Status})
	}

	tx := database.DB.Begin()

	// Update user status
	updates := map[string]interface{}{
		"status":      models.UserStatusActive,
		"approved_by": actorID,
		"is_active":   true,
	}
	if err := tx.Model(&user).Updates(updates).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuidhinisha"})
	}

	// Ensure leadership positions exist for elevated roles (covers legacy users)
	if err := upsertUserPosition(tx, user.ID, user.Role); err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuweka nafasi ya mtumiaji"})
	}

	// Ensure member row exists (self-register path + legacy users)
	if err := database.EnsureMemberForUser(tx, user, actorID); err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
	}

	// Create approval record
	approval := models.UserApproval{
		UserID:     user.ID,
		ApprovedBy: actorID,
		Status:     "APPROVED",
		Remarks:    req.Remarks,
		ApprovedAt: time.Now(),
	}
	if err := tx.Create(&approval).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi idhini"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuidhinisha"})
	}

	// Log audit
	services.LogAudit(c, &actorID, models.AuditUserApproved, "users", &user.ID, nil, map[string]interface{}{
		"user_id": user.ID, "name": user.Name, "approved_by": actorID,
	})

	// Self-registered users chose their own password; chair-created users got a temp password
	var notifBody string
	if user.MustChangePassword || user.CreatedBy != nil {
		notifBody = "Akaunti yako imeidhinishwa na Katibu. Ingia kwa nenosiri la muda ulilopewa na Mwenyekiti wakati wa kuunda akaunti, kisha utakazwa kuweka nenosiri jipya."
	} else {
		notifBody = "Akaunti yako imeidhinishwa na Katibu. Ingia kwa nenosiri uliloweka wakati wa kujisajili."
	}
	services.NotifyUser(user.ID, models.NotifUserApproved,
		"Akaunti Yako Imeidhinishwa",
		notifBody,
	)

	return c.JSON(fiber.Map{"message": "Mtumiaji ameidhinishwa"})
}

// RejectUser rejects a pending user (Secretary only).
func (h *UserManagementHandler) RejectUser(c *fiber.Ctx) error {
	actorID := middleware.GetUserID(c)
	userID := c.Params("id")

	var req models.RejectUserRequest
	c.BodyParser(&req)

	var user models.User
	if err := database.DB.Where("deleted_at IS NULL").First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mtumiaji hajapatikana"})
	}

	if user.Status != models.UserStatusPending {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mtumiaji hana hali ya kusubiri. Hali ya sasa: " + user.Status})
	}

	updates := map[string]interface{}{
		"status":      models.UserStatusRejected,
		"approved_by": actorID,
	}
	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kukataa"})
	}

	// Create rejection record
	approval := models.UserApproval{
		UserID:     user.ID,
		ApprovedBy: actorID,
		Status:     "REJECTED",
		Remarks:    req.Remarks,
		ApprovedAt: time.Now(),
	}
	database.DB.Create(&approval)

	// Log audit
	services.LogAudit(c, &actorID, models.AuditUserRejected, "users", &user.ID, nil, map[string]interface{}{
		"user_id": user.ID, "name": user.Name, "rejected_by": actorID, "remarks": req.Remarks,
	})

	// Notify user
	services.NotifyUser(user.ID, models.NotifUserRejected,
		"Akaunti Yako Imekataliwa",
		"Akaunti yako imekataliwa na Katibu. Wasiliana na Mwenyekiti kwa maelezo zaidi.",
	)

	return c.JSON(fiber.Map{"message": "Mtumiaji amekataliwa"})
}

// ResetUserPassword lets the Chair reset a non-admin user's password.
// Generates a one-time temp password, returns plaintext once as temp_password.
// Does not grant admin override powers — only password reset for group users.
func (h *UserManagementHandler) ResetUserPassword(c *fiber.Ctx) error {
	actorID := middleware.GetUserID(c)
	targetID := c.Params("id")

	if actorID == targetID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Huwezi kuweka upya nenosiri lako hapa. Tumia badilisha nenosiri kwenye wasifu.",
		})
	}

	var user models.User
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", targetID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mtumiaji hajapatikana"})
	}

	// Chairs may not reset system admin accounts
	if user.Role == models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Huwezi kuweka upya nenosiri la msimamizi wa mfumo",
		})
	}

	tempPassword := models.DefaultTempPassword()
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Hitilafu ya mfumo"})
	}

	if err := database.DB.Model(&user).Updates(map[string]interface{}{
		"password":             string(hashedPwd),
		"must_change_password": true,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kubadilisha nenosiri"})
	}

	services.LogAudit(c, &actorID, models.AuditPasswordSet, "users", &user.ID, nil, map[string]interface{}{
		"action": "chair_reset_password", "target_name": user.Name,
	})

	services.NotifyUser(user.ID, models.NotifSystem,
		"Nenosiri Limewekwa Upya",
		"Nenosiri lako limewekwa upya na Mwenyekiti. Tumia nenosiri la muda ulilopewa kuingia; utakazwa kuweka nenosiri jipya.",
	)

	return c.JSON(fiber.Map{
		"message":       "Nenosiri limewekwa upya. Mpe mtumiaji nenosiri la muda moja kwa moja (halitaonekana tena).",
		"temp_password": tempPassword,
	})
}
