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
// User is created in PENDING status with default temp password.
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

	// Hash default temp password
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(models.DefaultTempPassword()), bcrypt.DefaultCost)
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

	if err := database.DB.Create(&user).Error; err != nil {
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
		"message": "Mtumiaji ameundwa. Anasubiri kuidhinishwa na Katibu.",
		"data":    user,
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

	// Update user status
	updates := map[string]interface{}{
		"status":      models.UserStatusActive,
		"approved_by": actorID,
		"is_active":   true,
	}
	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuidhinisha"})
	}

	// Create approval record
	approval := models.UserApproval{
		UserID:     user.ID,
		ApprovedBy: actorID,
		Status:     "APPROVED",
		Remarks:    req.Remarks,
		ApprovedAt: time.Now(),
	}
	database.DB.Create(&approval)

	// Log audit
	services.LogAudit(c, &actorID, models.AuditUserApproved, "users", &user.ID, nil, map[string]interface{}{
		"user_id": user.ID, "name": user.Name, "approved_by": actorID,
	})

	// Notify user that they are approved
	services.NotifyUser(user.ID, models.NotifUserApproved,
		"Akaunti Yako Imeidhinishwa",
		"Akaunti yako imeidhinishwa na Katibu. Ingia kwa nenosiri la mfumo \"1-9\" na utakazwa kuweka nenosiri jipya.",
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
