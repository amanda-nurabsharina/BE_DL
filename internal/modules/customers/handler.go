package customers

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
	group := router.Group("/customers", authMiddleware)
	group.Get("/", h.List)
	group.Post("/", middleware.RequireRole("OWNER", "ADMIN"), h.Create)
	group.Get("/:id", h.GetByID)
	group.Patch("/:id", middleware.RequireRole("OWNER", "ADMIN"), h.Update)
	group.Delete("/:id", middleware.RequireRole("OWNER", "ADMIN"), h.Delete)

	patGroup := router.Group("/patients", authMiddleware)
	patGroup.Get("/", h.ListPatients)
	patGroup.Post("/", middleware.RequireRole("OWNER", "ADMIN", "TECHNICIAN"), h.CreatePatient)
	patGroup.Patch("/:id", middleware.RequireRole("OWNER", "ADMIN", "TECHNICIAN"), h.UpdatePatient)
	patGroup.Delete("/:id", middleware.RequireRole("OWNER", "ADMIN"), h.DeletePatient)
}

func (h *Handler) List(c *fiber.Ctx) error {
	search := c.Query("search")
	list, err := h.service.List(search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "SERVER_ERROR",
			"message": err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    list,
	})
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "INVALID_ID",
			"message": "Invalid customer ID format",
		})
	}

	cust, err := h.service.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "NOT_FOUND",
			"message": "Customer not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    cust,
	})
}

func (h *Handler) Create(c *fiber.Ctx) error {
	var req CreateCustomerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "BAD_REQUEST",
			"message": "Invalid payload",
		})
	}

	if req.Name == "" || req.Phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "Doctor name and phone number are required",
		})
	}

	cust, err := h.service.Create(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "SERVER_ERROR",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    cust,
	})
}

func (h *Handler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "INVALID_ID",
			"message": "Invalid customer ID format",
		})
	}

	var req UpdateCustomerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "BAD_REQUEST",
			"message": "Invalid payload",
		})
	}

	cust, err := h.service.Update(id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "SERVER_ERROR",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    cust,
	})
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "INVALID_ID",
			"message": "Invalid customer ID format",
		})
	}

	if err := h.service.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "SERVER_ERROR",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Customer archived successfully",
	})
}

func (h *Handler) ListPatients(c *fiber.Ctx) error {
	search := c.Query("search")
	customerID := c.Query("customer_id")
	list, err := h.service.ListPatients(search, customerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "SERVER_ERROR",
			"message": err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    list,
	})
}

func (h *Handler) CreatePatient(c *fiber.Ctx) error {
	var req CreatePatientRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "BAD_REQUEST",
			"message": "Invalid payload",
		})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "Patient name is required",
		})
	}

	pat, err := h.service.CreatePatient(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "SERVER_ERROR",
			"message": err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    pat,
	})
}

func (h *Handler) UpdatePatient(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "INVALID_ID",
			"message": "Invalid patient ID format",
		})
	}
	var req UpdatePatientRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "BAD_REQUEST",
			"message": "Invalid payload",
		})
	}
	pat, err := h.service.UpdatePatient(id, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "SERVER_ERROR",
			"message": err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    pat,
	})
}

func (h *Handler) DeletePatient(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "INVALID_ID",
			"message": "Invalid patient ID format",
		})
	}
	if err := h.service.DeletePatient(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "SERVER_ERROR",
			"message": err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Patient archived successfully",
	})
}
