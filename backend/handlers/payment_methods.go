package handlers

import (
	"kikundibora/database"
	"kikundibora/middleware"
	"kikundibora/models"
	"kikundibora/services"

	"github.com/gofiber/fiber/v2"
)

type PaymentMethodHandler struct{}

func NewPaymentMethodHandler() *PaymentMethodHandler {
	return &PaymentMethodHandler{}
}

type paymentMethodRequest struct {
	Type          string `json:"type"`
	ProviderName  string `json:"provider_name"`
	AccountNumber string `json:"account_number"`
	AccountName   string `json:"account_name"`
	IsActive      *bool  `json:"is_active"`
}

func validatePaymentMethodRequest(req *paymentMethodRequest, requireAll bool) error {
	if requireAll || req.Type != "" {
		if !models.IsValidPaymentMethodType(req.Type) {
			return fiber.NewError(fiber.StatusBadRequest, "Aina si sahihi. Chagua lipa_namba au bank")
		}
	}
	if requireAll || req.ProviderName != "" {
		if req.ProviderName == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Jina la mtandao/benki linahitajika")
		}
	}
	if requireAll || req.AccountNumber != "" {
		if req.AccountNumber == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Namba ya malipo inahitajika")
		}
	}
	if requireAll || req.AccountName != "" {
		if req.AccountName == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Jina lililosajiliwa linahitajika")
		}
	}
	return nil
}

// loadGroupForPaymentMethods resolves the :id group or sends a 404.
// Returns nil if the response was already written. RBAC-M01 tenant check
// included.
func loadGroupForPaymentMethods(c *fiber.Ctx) *models.Group {
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

// List returns the group's payment methods. Members (and any non-leadership
// role) see only active ones; leadership sees all (including deactivated).
// GET /api/v1/groups/:id/payment-methods
func (h *PaymentMethodHandler) List(c *fiber.Ctx) error {
	g := loadGroupForPaymentMethods(c)
	if g == nil {
		return nil
	}

	role := middleware.GetUserRole(c)
	query := database.DB.Where("group_id = ?", g.ID)
	if role != models.RoleChair && role != models.RoleTreasurer && role != models.RoleAdmin {
		query = query.Where("is_active = TRUE")
	}

	var methods []models.PaymentMethod
	query.Order("type ASC, provider_name ASC").Find(&methods)

	return c.JSON(fiber.Map{"data": methods, "total": len(methods)})
}

// Create adds a payment method. Mwenyekiti / Mweka Hazina only (route guard).
// POST /api/v1/groups/:id/payment-methods
func (h *PaymentMethodHandler) Create(c *fiber.Ctx) error {
	g := loadGroupForPaymentMethods(c)
	if g == nil {
		return nil
	}

	var req paymentMethodRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validatePaymentMethodRequest(&req, true); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	userID := middleware.GetUserID(c)
	pm := models.PaymentMethod{
		GroupID:       g.ID,
		Type:          models.PaymentMethodType(req.Type),
		ProviderName:  req.ProviderName,
		AccountNumber: req.AccountNumber,
		AccountName:   req.AccountName,
		IsActive:      true,
		CreatedBy:     userID,
	}
	if err := database.DB.Create(&pm).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuongeza njia ya malipo"})
	}

	services.LogAudit(c, &userID, models.AuditCreate, "payment_methods", &pm.ID, nil, map[string]interface{}{
		"group_id": g.ID, "type": pm.Type, "provider": pm.ProviderName, "account": pm.AccountNumber,
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Njia ya malipo imeongezwa", "data": pm})
}

// Update edits a payment method (fields or is_active toggle).
// Mwenyekiti / Mweka Hazina only (route guard).
// PATCH /api/v1/groups/:id/payment-methods/:pmId
func (h *PaymentMethodHandler) Update(c *fiber.Ctx) error {
	g := loadGroupForPaymentMethods(c)
	if g == nil {
		return nil
	}

	var pm models.PaymentMethod
	if err := database.DB.Where("id = ? AND group_id = ?", c.Params("pmId"), g.ID).
		First(&pm).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Njia ya malipo haijapatikana"})
	}

	var req paymentMethodRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Data si sahihi"})
	}
	if err := validatePaymentMethodRequest(&req, false); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	if req.Type != "" {
		pm.Type = models.PaymentMethodType(req.Type)
	}
	if req.ProviderName != "" {
		pm.ProviderName = req.ProviderName
	}
	if req.AccountNumber != "" {
		pm.AccountNumber = req.AccountNumber
	}
	if req.AccountName != "" {
		pm.AccountName = req.AccountName
	}
	if req.IsActive != nil {
		pm.IsActive = *req.IsActive
	}

	if err := database.DB.Save(&pm).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kuhifadhi mabadiliko"})
	}

	userID := middleware.GetUserID(c)
	services.LogAudit(c, &userID, models.AuditUpdate, "payment_methods", &pm.ID, nil, map[string]interface{}{
		"provider": pm.ProviderName, "account": pm.AccountNumber, "is_active": pm.IsActive,
	})

	return c.JSON(fiber.Map{"message": "Mabadiliko yamehifadhiwa", "data": pm})
}

// Delete removes a payment method. Mwenyekiti / Mweka Hazina only (route guard).
// DELETE /api/v1/groups/:id/payment-methods/:pmId
func (h *PaymentMethodHandler) Delete(c *fiber.Ctx) error {
	g := loadGroupForPaymentMethods(c)
	if g == nil {
		return nil
	}

	res := database.DB.Where("id = ? AND group_id = ?", c.Params("pmId"), g.ID).
		Delete(&models.PaymentMethod{})
	if res.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Imeshindikana kufuta"})
	}
	if res.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "Njia ya malipo haijapatikana"})
	}

	userID := middleware.GetUserID(c)
	pmID := c.Params("pmId")
	services.LogAudit(c, &userID, models.AuditDelete, "payment_methods", &pmID, nil, map[string]interface{}{
		"group_id": g.ID,
	})

	return c.JSON(fiber.Map{"message": "Njia ya malipo imefutwa"})
}
