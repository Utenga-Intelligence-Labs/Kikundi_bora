package handlers

import (
	"time"

	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
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
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name, phone, role, status")
		})

	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
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
		First(&member, id).Error; err != nil {
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

	member := models.Member{
		MemberNo:     memberNo,
		FullName:     req.FullName,
		Phone:        req.Phone,
		Address:      req.Address,
		JoinedAt:     joinedAt,
		IsActive:     true,
		RegisteredBy: userID,
	}

	if err := tx.Create(&member).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusajili mwanachama"})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kusajili mwanachama"})
	}

	services.LogAudit(c, &userID, models.AuditCreate, "members", &member.ID, nil, map[string]interface{}{
		"member_no": memberNo, "full_name": member.FullName, "phone": member.Phone,
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Mwanachama amesajiliwa",
		"data":    member,
	})
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
		var phoneCount int64
		database.DB.Model(&models.Member{}).Where("phone = ? AND id != ? AND deleted_at IS NULL", *req.Phone, member.ID).Count(&phoneCount)
		if phoneCount > 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"message": "Namba ya simu hiyo tayari imesajiliwa"})
		}
		member.Phone = *req.Phone
	}
	if req.Address != nil {
		member.Address = req.Address
	}
	if req.IsActive != nil {
		member.IsActive = *req.IsActive
	}

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
	if err := database.DB.Where("deleted_at IS NULL").First(&member, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Mwanachama hajapatikana"})
	}

	now := time.Now()
	database.DB.Model(&member).Update("deleted_at", now)

	userID := middleware.GetUserID(c)
	services.LogAudit(c, &userID, models.AuditDelete, "members", &member.ID, nil, nil)

	return c.JSON(fiber.Map{"message": "Mwanachama amefutwa"})
}
