package handlers

import (
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"time"

	"kikundibora/config"
	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var validate = validator.New()

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req models.RegisterRequest
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

	// Check email uniqueness if provided
	if req.Email != "" {
		email := strings.ToLower(strings.TrimSpace(req.Email))
		database.DB.Model(&models.User{}).Where("email = ? AND deleted_at IS NULL", email).Count(&count)
		if count > 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Barua pepe hii tayari imesajiliwa"})
		}
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Hitilafu ya mfumo"})
	}

	var emailPtr *string
	if req.Email != "" {
		e := strings.ToLower(strings.TrimSpace(req.Email))
		emailPtr = &e
	}

	// SECURITY: Force role to member and status to pending
	// Users cannot self-assign elevated roles
	user := models.User{
		Name:               strings.TrimSpace(req.Name),
		Email:              emailPtr,
		Phone:              phone,
		Password:           string(hashedPwd),
		Role:               models.RoleMember, // Always member
		Status:             models.UserStatusPending, // Requires approval
		MustChangePassword: false,
		IsActive:           true,
	}

	tx := database.DB.Begin()
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusajili"})
	}

	// Linked member row (appears on wanachama; login still requires approval)
	if err := database.EnsureMemberForUser(tx, user, user.ID); err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": err.Error()})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusajili"})
	}

	services.LogAudit(c, &user.ID, models.AuditCreate, "users", &user.ID, nil, map[string]interface{}{
		"name": user.Name, "phone": user.Phone, "role": user.Role, "status": user.Status,
	})

	// Notify secretaries about new registration
	services.NotifyRole(models.RoleSecretary, models.NotifUserCreated,
		"Usajili Mpya",
		"Mtumiaji mpya \""+user.Name+"\" amesajiliwa na anahitaji kuidhinishwa.",
		user.ID,
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Usajili umekamilika. Akaunti yako inasubiri kuidhinishwa na Katibu.",
		"data":    user,
	})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	loginID := strings.TrimSpace(req.Email) // can be email or phone
	ip := c.IP()

	// SECURITY: Rate limiting based on failed login attempts
	var recentAttempts int64
	fiveMinAgo := time.Now().Add(-5 * time.Minute)
	database.DB.Model(&models.FailedLogin{}).
		Where("ip_address = ? AND attempted_at > ?", ip, fiveMinAgo).
		Count(&recentAttempts)
	if recentAttempts >= 5 {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"message": "Majaribio mengi mno ya kuingia. Jaribu tena baada ya dakika 5.",
		})
	}

	// Find user by email or phone
	var user models.User
	query := database.DB.Where("deleted_at IS NULL")
	if strings.Contains(loginID, "@") {
		query = query.Where("email = ?", strings.ToLower(loginID))
	} else {
		query = query.Where("phone = ?", loginID)
	}

	if err := query.First(&user).Error; err != nil {
		h.recordFailedLogin(loginID, ip, c.Get("User-Agent"))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Nambari ya simu/barua pepe au nenosiri si sahihi"})
	}

	// Check user status
	switch user.Status {
	case models.UserStatusPending:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Akaunti yako bado haijakaguliwa na Katibu. Subiri uidhinishwe."})
	case models.UserStatusRejected:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Akaunti yako imekataliwa. Wasiliana na Mwenyekiti."})
	case models.UserStatusSuspended:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Akaunti yamesimamishwa. Wasiliana na msimamizi."})
	}

	if !user.IsActive {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"message": "Akaunti hii imezimwa"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		h.recordFailedLogin(loginID, ip, c.Get("User-Agent"))
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Nambari ya simu/barua pepe au nenosiri si sahihi"})
	}

	now := time.Now()
	database.DB.Model(&user).Update("last_login_at", now)

	services.LogAudit(c, &user.ID, models.AuditLogin, "users", &user.ID, nil, map[string]interface{}{
		"ip_address": ip,
		"user_agent": c.Get("User-Agent"),
	})

	token, expiresAt := h.generateToken(user.ID, user.Role)
	if token == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutengeneza tokeni"})
	}

	session := models.UserSession{
		UserID:       user.ID,
		TokenHash:    sha256Hex(token),
		IPAddress:    ip,
		UserAgent:    c.Get("User-Agent"),
		LastActiveAt: now,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}
	if err := database.DB.Create(&session).Error; err != nil {
		log.Printf("ERROR: Failed to create session: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuanzisha kikao"})
	}

	return c.JSON(models.AuthResponse{
		Token:              token,
		User:               &user,
		ExpiresAt:          expiresAt,
		FirstLoginRequired: user.MustChangePassword,
	})
}

func (h *AuthHandler) FirstLoginSetup(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req models.FirstLoginSetupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	if req.NewPassword != req.ConfirmPassword {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Nenosiri hazifanani"})
	}

	if len(req.NewPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Nenosiri lazima liwe na angalau herufi 6"})
	}

	var user models.User
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mtumiaji hajapatikana"})
	}

	// Prevent reusing the current (temp) password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.NewPassword)); err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Huwezi kutumia nenosiri la sasa. Chagua nenosiri jipya."})
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Hitilafu ya mfumo"})
	}

	updates := map[string]interface{}{
		"password":             string(hashedPwd),
		"must_change_password": false,
	}
	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusasisha nenosiri"})
	}

	// Reload user from DB so the response has the updated must_change_password = false
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupakia mtumiaji"})
	}

	services.LogAudit(c, &userID, models.AuditPasswordSet, "users", &userID, nil, map[string]interface{}{
		"action": "first_login_password_setup",
	})

	token, expiresAt := h.generateToken(user.ID, user.Role)
	if token == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kutengeneza tokeni"})
	}

	session := models.UserSession{
		UserID:       user.ID,
		TokenHash:    sha256Hex(token),
		IPAddress:    c.IP(),
		UserAgent:    c.Get("User-Agent"),
		LastActiveAt: time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}
	if err := database.DB.Create(&session).Error; err != nil {
		log.Printf("ERROR: Failed to create session: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuanzisha kikao"})
	}

	return c.JSON(models.AuthResponse{
		Token:              token,
		User:               &user,
		ExpiresAt:          expiresAt,
		FirstLoginRequired: user.MustChangePassword,
	})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var user models.User
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mtumiaji hajapatikana"})
	}

	resp := models.MeResponse{
		User:       &user,
		Leadership: []string{},
	}

	// Admin has no member row
	if user.Role == models.RoleAdmin {
		return c.JSON(resp)
	}

	// Find linked member
	var member models.Member
	if err := database.DB.Where("user_id = ? AND deleted_at IS NULL", userID).
		First(&member).Error; err == nil {
		resp.MemberID = &member.ID
		resp.MemberCode = &member.MemberNo

		// Find leadership positions
		var positions []models.LeadershipPosition
		database.DB.Where("member_id = ? AND is_current = TRUE", member.ID).
			Find(&positions)
		for _, p := range positions {
			resp.Leadership = append(resp.Leadership, string(p.Role))
		}
	}

	return c.JSON(resp)
}

func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	// SECURITY: Require admin role to reset passwords
	// This prevents unauthenticated account takeover
	actorRole, ok := c.Locals("role").(models.Role)
	if !ok || actorRole != models.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "Ruhusa imekataliwa. Msimamizi pekee anaweza kubadilisha nenosiri.",
		})
	}

	var req models.ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if err := validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": formatValidationErrors(err)})
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	var user models.User
	if err := database.DB.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Akaunti haijapatikana"})
	}

	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Nenosiri lazima liwe na angalau herufi 8"})
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Hitilafu ya mfumo"})
	}

	adminID := middleware.GetUserID(c)
	database.DB.Model(&user).Updates(map[string]interface{}{
		"password":             string(hashedPwd),
		"must_change_password": false,
	})

	services.LogAudit(c, &adminID, models.AuditPasswordSet, "users", &user.ID, nil, map[string]interface{}{
		"action": "admin_reset_password",
	})

	return c.JSON(fiber.Map{"message": "Nenosiri limebadilishwa"})
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var body struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	if len(body.NewPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Nenosiri jipya lazima liwe na angalau herufi 6"})
	}

	var user models.User
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mtumiaji hajapatikana"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.OldPassword)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Nenosiri la zamani si sahihi"})
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Hitilafu ya mfumo"})
	}
	database.DB.Model(&user).Updates(map[string]interface{}{
		"password":             string(hashedPwd),
		"must_change_password": false,
	})

	services.LogAudit(c, &userID, models.AuditPasswordSet, "users", &userID, nil, map[string]interface{}{
		"action": "change_password",
	})

	return c.JSON(fiber.Map{"message": "Nenosiri limebadilishwa"})
}

func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req models.UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}

	var user models.User
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mtumiaji hajapatikana"})
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		user.Name = strings.TrimSpace(*req.Name)
		if user.Name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Jina haliwezi kuwa tupu"})
		}
		updates["name"] = user.Name
	}
	if req.Phone != nil {
		user.Phone = strings.TrimSpace(*req.Phone)
		if user.Phone == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Nambari ya simu haliwezi kuwa tupu"})
		}
		var count int64
		database.DB.Model(&models.User{}).Where("phone = ? AND id != ? AND deleted_at IS NULL", user.Phone, userID).Count(&count)
		if count > 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Nambari ya simu tayari imetumika na mtumiaji mwingine"})
		}
		updates["phone"] = user.Phone
	}
	if req.AvatarURL != nil {
		user.AvatarURL = strings.TrimSpace(*req.AvatarURL)
		updates["avatar_url"] = user.AvatarURL
	}
	if req.Bio != nil {
		user.Bio = strings.TrimSpace(*req.Bio)
		updates["bio"] = user.Bio
	}

	if len(updates) == 0 {
		return c.JSON(fiber.Map{"message": "Hakuna mabadiliko", "data": user})
	}

	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kubadilisha"})
	}

	// Reload user from DB to get all updated fields
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kupakia mtumiaji"})
	}

	services.LogAudit(c, &userID, models.AuditUpdate, "users", &userID, nil, updates)

	return c.JSON(fiber.Map{"message": "Wasifu umebadilishwa", "data": user})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	// Invalidate token by storing its hash in the session's revoked_at field
	header := c.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		tokenHash := sha256Hex(tokenStr)
		now := time.Now()
		database.DB.Model(&models.UserSession{}).
			Where("user_id = ? AND token_hash = ? AND revoked_at IS NULL", userID, tokenHash).
			Update("revoked_at", now)
	}

	services.LogAudit(c, &userID, models.AuditLogout, "users", &userID, nil, nil)
	return c.JSON(fiber.Map{"message": "Umetoka kwenye akaunti"})
}

func sha256Hex(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}

func (h *AuthHandler) generateToken(userID string, role models.Role) (string, string) {
	expiresAt := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     expiresAt.Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(config.AppConfig.JWTSecret))
	if err != nil {
		log.Printf("ERROR: Failed to sign JWT: %v", err)
		return "", ""
	}
	return tokenStr, expiresAt.Format(time.RFC3339)
}

func (h *AuthHandler) recordFailedLogin(email, ip, ua string) {
	failed := models.FailedLogin{
		EmailAttempted: email,
		IPAddress:      ip,
		UserAgent:      ua,
		AttemptedAt:    time.Now(),
	}
	database.DB.Create(&failed)
}


