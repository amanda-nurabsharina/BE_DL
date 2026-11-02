package finance

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
	group := router.Group("/finance", authMiddleware)

	// Invoices
	group.Get("/invoices", h.ListInvoices)
	group.Get("/invoices/:id", h.GetInvoiceByID)
	group.Post("/invoices", middleware.RequireRole("OWNER", "ADMIN", "CASHIER"), h.CreateInvoiceFromOrder)

	// Payments
	group.Post("/payments", middleware.RequireRole("OWNER", "ADMIN", "CASHIER"), h.RecordPayment)

	// Expenses
	group.Get("/expense-categories", h.ListExpenseCategories)
	group.Get("/expenses", h.ListExpenses)
	group.Post("/expenses", middleware.RequireRole("OWNER", "ADMIN", "CASHIER"), h.CreateExpense)
}

func (h *Handler) ListInvoices(c *fiber.Ctx) error {
	status := c.Query("status")
	customerID := c.Query("customer_id")
	search := c.Query("search")

	invoices, err := h.service.ListInvoices(status, customerID, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": invoices})
}

func (h *Handler) GetInvoiceByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid invoice ID"})
	}

	inv, err := h.service.GetInvoiceByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invoice not found"})
	}
	return c.JSON(fiber.Map{"success": true, "data": inv})
}

func (h *Handler) CreateInvoiceFromOrder(c *fiber.Ctx) error {
	var input CreateInvoiceInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	userIDStr, _ := c.Locals("user_id").(string)
	if uid, err := uuid.Parse(userIDStr); err == nil {
		input.CreatedBy = &uid
	}

	inv, err := h.service.CreateInvoiceFromOrder(input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": inv})
}

func (h *Handler) RecordPayment(c *fiber.Ctx) error {
	var input RecordPaymentInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	userIDStr, _ := c.Locals("user_id").(string)
	if uid, err := uuid.Parse(userIDStr); err == nil {
		input.ReceivedBy = &uid
	}

	pay, err := h.service.RecordPayment(input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": pay})
}

func (h *Handler) ListExpenseCategories(c *fiber.Ctx) error {
	cats, err := h.service.ListExpenseCategories()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": cats})
}

func (h *Handler) ListExpenses(c *fiber.Ctx) error {
	expenses, err := h.service.ListExpenses()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": expenses})
}

func (h *Handler) CreateExpense(c *fiber.Ctx) error {
	var input CreateExpenseInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	userIDStr, _ := c.Locals("user_id").(string)
	if uid, err := uuid.Parse(userIDStr); err == nil {
		input.RecordedBy = &uid
	}

	exp, err := h.service.CreateExpense(input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": exp})
}
