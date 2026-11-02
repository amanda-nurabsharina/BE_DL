package reports

import (
	"github.com/gofiber/fiber/v2"

	"dental-lab-backend/internal/platform/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	group := router.Group("/reports", authMiddleware)
	group.Get("/dashboard", h.GetDashboardSummary)
	group.Get("/receivables", middleware.RequireRole("OWNER", "ADMIN", "CASHIER"), h.GetReceivablesReport)
}

func (h *Handler) GetDashboardSummary(c *fiber.Ctx) error {
	summary, err := h.service.GetDashboardSummary()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": summary})
}

func (h *Handler) GetReceivablesReport(c *fiber.Ctx) error {
	report, err := h.service.GetReceivablesReport()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": report})
}
