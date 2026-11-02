package production

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"dental-lab-backend/internal/platform/middleware"
)

type Handler struct {
	service   *Service
	uploadDir string
}

func NewHandler(service *Service, uploadDir string) *Handler {
	return &Handler{service: service, uploadDir: uploadDir}
}

func (h *Handler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	group := router.Group("/production", authMiddleware)

	group.Get("/denture-types", h.ListDentureTypes)
	group.Get("/orders", h.ListOrders)
	group.Post("/orders", middleware.RequireRole("OWNER", "ADMIN"), h.CreateOrder)
	group.Get("/orders/:id", h.GetOrderByID)
	group.Post("/orders/:id/assign", middleware.RequireRole("OWNER", "ADMIN", "TECHNICIAN"), h.AssignTechnician)
	group.Patch("/orders/:id/stages/:runId", middleware.RequireRole("OWNER", "ADMIN", "TECHNICIAN"), h.UpdateStageRun)
	group.Patch("/orders/:id/status", middleware.RequireRole("OWNER", "ADMIN", "TECHNICIAN", "CASHIER"), h.UpdateOrderStatus)
	group.Post("/orders/:id/advance-stage", middleware.RequireRole("OWNER", "ADMIN", "TECHNICIAN"), h.AdvanceStage)
	group.Post("/orders/:id/qc", middleware.RequireRole("OWNER", "ADMIN", "TECHNICIAN"), h.SubmitQualityCheck)
	group.Patch("/orders/:id/feedback", middleware.RequireRole("OWNER", "ADMIN", "TECHNICIAN"), h.UpdateOrderFeedback)
	group.Post("/orders/:id/upload", h.UploadAttachment)
}

func (h *Handler) ListDentureTypes(c *fiber.Ctx) error {
	types, err := h.service.ListDentureTypes()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": types})
}

func (h *Handler) ListOrders(c *fiber.Ctx) error {
	status := c.Query("status")
	customerID := c.Query("customer_id")
	search := c.Query("search")

	orders, err := h.service.ListOrders(status, customerID, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": orders})
}

func (h *Handler) GetOrderByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid order ID"})
	}

	order, err := h.service.GetOrderByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Order not found"})
	}
	return c.JSON(fiber.Map{"success": true, "data": order})
}

func (h *Handler) CreateOrder(c *fiber.Ctx) error {
	var input CreateOrderInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	userIDStr, _ := c.Locals("user_id").(string)
	if uid, err := uuid.Parse(userIDStr); err == nil {
		input.CreatedBy = &uid
	}

	order, err := h.service.CreateOrder(input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": order})
}

type AssignInput struct {
	TechnicianID uuid.UUID `json:"technician_id"`
}

func (h *Handler) AssignTechnician(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid order ID"})
	}

	var input AssignInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	if err := h.service.AssignTechnician(id, input.TechnicianID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Technician assigned successfully"})
}

func (h *Handler) UpdateStageRun(c *fiber.Ctx) error {
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid order ID"})
	}
	runID, err := uuid.Parse(c.Params("runId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid stage run ID"})
	}

	var input UpdateStageRunInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	order, err := h.service.UpdateStageRun(orderID, runID, input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": order})
}

type UpdateOrderStatusInput struct {
	Status        string `json:"status"`
	InternalNotes string `json:"internal_notes"`
}

func (h *Handler) UpdateOrderStatus(c *fiber.Ctx) error {
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid order ID"})
	}

	var input UpdateOrderStatusInput
	if err := c.BodyParser(&input); err != nil || input.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Status is required"})
	}

	order, err := h.service.UpdateOrderStatus(orderID, input.Status, input.InternalNotes)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": order})
}

type AdvanceStageInput struct {
	CurrentRunID uuid.UUID  `json:"current_run_id"`
	Notes        string     `json:"notes"`
	TechnicianID *uuid.UUID `json:"technician_id"`
	NextTechID   *uuid.UUID `json:"next_technician_id"`
}

func (h *Handler) AdvanceStage(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid order ID"})
	}

	var input AdvanceStageInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	if err := h.service.AdvanceStage(id, input.CurrentRunID, input.Notes, input.TechnicianID, input.NextTechID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Stage progressed successfully"})
}

type QCInput struct {
	Passed            bool   `json:"passed"`
	ChecklistResults  string `json:"checklist_results"`
	DefectNotes       string `json:"defect_notes"`
	ReworkTargetStage string `json:"rework_target_stage"`
}

func (h *Handler) SubmitQualityCheck(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid order ID"})
	}

	var input QCInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	userIDStr, _ := c.Locals("user_id").(string)
	inspectorID, _ := uuid.Parse(userIDStr)

	qc, err := h.service.SubmitQualityCheck(id, inspectorID, input.Passed, input.ChecklistResults, input.DefectNotes, input.ReworkTargetStage)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": qc})
}

func (h *Handler) UploadAttachment(c *fiber.Ctx) error {
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid order ID"})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File attachment is required"})
	}

	attachmentType := c.FormValue("attachment_type", "QC_PHOTO")
	description := c.FormValue("description", "")

	filename := fmt.Sprintf("ord_%s_%d%s", orderID.String()[:8], time.Now().UnixNano(), filepath.Ext(file.Filename))
	destination := filepath.Join(h.uploadDir, filename)

	if err := c.SaveFile(file, destination); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save file"})
	}

	publicPath := fmt.Sprintf("/uploads/%s", filename)

	userIDStr, _ := c.Locals("user_id").(string)
	userID, _ := uuid.Parse(userIDStr)

	att := OrderAttachment{
		ProductionOrderID: orderID,
		FileName:          file.Filename,
		StoredFilePath:    publicPath,
		FileSize:          file.Size,
		MimeType:          file.Header.Get("Content-Type"),
		AttachmentType:    attachmentType,
		Description:       description,
		UploadedBy:        &userID,
		CreatedAt:         time.Now(),
	}

	if err := h.service.db.Create(&att).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": att})
}

func (h *Handler) UpdateOrderFeedback(c *fiber.Ctx) error {
	orderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid order ID"})
	}
	var body struct {
		FeedbackRating int    `json:"feedback_rating"`
		FeedbackNotes  string `json:"feedback_notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}
	ord, err := h.service.UpdateOrderFeedback(orderID, body.FeedbackRating, body.FeedbackNotes)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": ord})
}

