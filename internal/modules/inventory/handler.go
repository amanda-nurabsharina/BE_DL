package inventory

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"dental-lab-backend/internal/platform/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	group := router.Group("/inventory", authMiddleware)

	// Units & Suppliers
	group.Get("/units", h.ListUnits)
	group.Get("/suppliers", h.ListSuppliers)
	group.Post("/suppliers", middleware.RequireRole("OWNER", "ADMIN"), h.CreateSupplier)

	// Master Data Items
	group.Get("/items", h.ListItems)
	group.Post("/items", middleware.RequireRole("OWNER", "ADMIN"), h.CreateItem)
	group.Patch("/items/:id", middleware.RequireRole("OWNER", "ADMIN"), h.UpdateItem)

	// Batches, Stock Movements & Inventory Report
	group.Get("/batches", h.ListBatches)
	group.Get("/movements", h.ListMovements)
	group.Get("/report", h.GetInventoryReport)

	// Stock In (Receive) & Stock Out (Issue / Usage)
	group.Post("/receive", middleware.RequireRole("OWNER", "ADMIN"), h.ReceiveMaterial)
	group.Post("/issue", middleware.RequireRole("OWNER", "ADMIN", "TECHNICIAN"), h.IssueMaterial)

	// Purchase Requests & Owner Approvals
	group.Get("/purchase-requests", h.ListPurchaseRequests)
	group.Post("/purchase-requests", middleware.RequireRole("OWNER", "ADMIN", "TECHNICIAN"), h.CreatePurchaseRequest)
	group.Post("/purchase-requests/:id/approve", middleware.RequireRole("OWNER"), h.ApprovePurchaseRequest)
	group.Post("/purchase-requests/:id/reject", middleware.RequireRole("OWNER"), h.RejectPurchaseRequest)
	group.Post("/purchase-requests/:id/receive", middleware.RequireRole("OWNER", "ADMIN"), h.ReceiveApprovedPR)
}

func getTenantID(c *fiber.Ctx) *uuid.UUID {
	role, _ := c.Locals("role").(string)
	if role == "SUPER_ADMIN" {
		if tQuery := c.Query("tenant_id"); tQuery != "" {
			if parsed, err := uuid.Parse(tQuery); err == nil {
				return &parsed
			}
		}
	}
	if tid, ok := c.Locals("tenant_id").(*uuid.UUID); ok {
		return tid
	}
	return nil
}

func (h *Handler) ListUnits(c *fiber.Ctx) error {
	units, err := h.service.ListUnits()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": units})
}

func (h *Handler) ListSuppliers(c *fiber.Ctx) error {
	suppliers, err := h.service.ListSuppliers(getTenantID(c))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": suppliers})
}

func (h *Handler) CreateSupplier(c *fiber.Ctx) error {
	var sup Supplier
	if err := c.BodyParser(&sup); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}
	if sup.TenantID == nil {
		sup.TenantID = getTenantID(c)
	}
	if err := h.service.CreateSupplier(&sup); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": sup})
}

func (h *Handler) ListItems(c *fiber.Ctx) error {
	search := c.Query("search")
	category := c.Query("category")
	items, err := h.service.ListItems(search, category, getTenantID(c))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": items})
}

func (h *Handler) CreateItem(c *fiber.Ctx) error {
	var input UpsertItemInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}
	if input.TenantID == nil {
		input.TenantID = getTenantID(c)
	}
	userIDStr, _ := c.Locals("user_id").(string)
	input.UserID, _ = uuid.Parse(userIDStr)

	item, err := h.service.CreateItem(input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item})
}

func (h *Handler) UpdateItem(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid item ID"})
	}

	var input UpsertItemInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	item, err := h.service.UpdateItem(id, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": item})
}

func (h *Handler) ListBatches(c *fiber.Ctx) error {
	itemID := c.Query("item_id")
	batches, err := h.service.ListBatches(itemID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": batches})
}

func (h *Handler) ListMovements(c *fiber.Ctx) error {
	itemID := c.Query("item_id")
	movementType := c.Query("type")
	movements, err := h.service.ListMovements(itemID, movementType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": movements})
}

func (h *Handler) GetInventoryReport(c *fiber.Ctx) error {
	report, err := h.service.GetInventoryReport(getTenantID(c))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": report})
}

func (h *Handler) ReceiveMaterial(c *fiber.Ctx) error {
	var input ReceiveMaterialInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	userIDStr, _ := c.Locals("user_id").(string)
	input.UserID, _ = uuid.Parse(userIDStr)

	movement, err := h.service.ReceiveMaterial(input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": movement})
}

func (h *Handler) IssueMaterial(c *fiber.Ctx) error {
	var input IssueMaterialInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	userIDStr, _ := c.Locals("user_id").(string)
	input.UserID, _ = uuid.Parse(userIDStr)

	movement, err := h.service.IssueMaterial(input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": movement})
}

func (h *Handler) ListPurchaseRequests(c *fiber.Ctx) error {
	status := c.Query("status")
	prs, err := h.service.ListPurchaseRequests(status, getTenantID(c))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": prs})
}

func (h *Handler) CreatePurchaseRequest(c *fiber.Ctx) error {
	var pr PurchaseRequest
	if err := c.BodyParser(&pr); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}
	if pr.TenantID == nil {
		pr.TenantID = getTenantID(c)
	}

	userIDStr, _ := c.Locals("user_id").(string)
	pr.RequestedBy, _ = uuid.Parse(userIDStr)

	if err := h.service.CreatePurchaseRequest(&pr); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": pr})
}

func (h *Handler) ApprovePurchaseRequest(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid PR ID"})
	}

	userIDStr, _ := c.Locals("user_id").(string)
	ownerID, _ := uuid.Parse(userIDStr)

	if err := h.service.ApprovePurchaseRequest(id, ownerID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Purchase request approved by Owner"})
}

type RejectRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) RejectPurchaseRequest(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid PR ID"})
	}

	var req RejectRequest
	c.BodyParser(&req)

	userIDStr, _ := c.Locals("user_id").(string)
	ownerID, _ := uuid.Parse(userIDStr)

	if err := h.service.RejectPurchaseRequest(id, ownerID, req.Reason); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Purchase request rejected"})
}

func (h *Handler) ReceiveApprovedPR(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid PR ID"})
	}

	userIDStr, _ := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	if err := h.service.ReceiveApprovedPR(id, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Material received and stock updated"})
}
