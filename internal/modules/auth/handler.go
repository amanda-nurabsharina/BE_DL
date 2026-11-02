package auth

import (
	"fmt"
	"path/filepath"
	"strconv"
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
	authGroup := router.Group("/auth")
	authGroup.Post("/login", h.Login)
	authGroup.Get("/me", authMiddleware, h.GetMe)
	authGroup.Patch("/profile", authMiddleware, h.UpdateMyProfile)
	authGroup.Post("/profile/avatar", authMiddleware, h.UploadMyAvatar)

	// Roles
	router.Get("/roles", authMiddleware, h.ListRoles)
	router.Patch("/roles/:id", authMiddleware, middleware.RequireRole("SUPER_ADMIN", "OWNER"), h.UpdateRoleMenus)

	// Users CRUD & HR Staff Directory
	usersGroup := router.Group("/users", authMiddleware)
	usersGroup.Get("/", h.ListUsers)
	usersGroup.Post("/", middleware.RequireRole("SUPER_ADMIN", "OWNER", "ADMIN"), h.CreateUser)
	usersGroup.Patch("/:id", middleware.RequireRole("SUPER_ADMIN", "OWNER", "ADMIN"), h.UpdateUser)
	usersGroup.Post("/:id/avatar", middleware.RequireRole("SUPER_ADMIN", "OWNER", "ADMIN"), h.UploadUserAvatar)
	usersGroup.Delete("/:id", middleware.RequireRole("SUPER_ADMIN", "OWNER"), h.DeleteUser)

	// HR Attendance & Overtime
	hrGroup := router.Group("/hr", authMiddleware)
	hrGroup.Get("/attendances", h.ListAttendances)
	hrGroup.Post("/attendances", h.CreateAttendance)
	hrGroup.Delete("/attendances/:id", middleware.RequireRole("SUPER_ADMIN", "OWNER", "ADMIN"), h.DeleteAttendance)

	// System Settings
	settingsGroup := router.Group("/settings", authMiddleware)
	settingsGroup.Get("/", h.GetSettings)
	settingsGroup.Post("/", middleware.RequireRole("SUPER_ADMIN", "OWNER"), h.UpdateSetting)
	settingsGroup.Post("/logo", middleware.RequireRole("SUPER_ADMIN", "OWNER"), h.UploadLogo)
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "BAD_REQUEST",
			"message": "Invalid request payload",
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"message": "Username and password are required",
		})
	}

	res, err := h.service.Login(req)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "AUTH_FAILED",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    res,
	})
}

func (h *Handler) GetMe(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "UNAUTHORIZED",
			"message": "Session expired or invalid",
		})
	}

	user, err := h.service.GetMe(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "NOT_FOUND",
			"message": "User not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

func (h *Handler) ListUsers(c *fiber.Ctx) error {
	users, err := h.service.ListUsers()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "SERVER_ERROR",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    users,
	})
}

func (h *Handler) CreateUser(c *fiber.Ctx) error {
	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	user, err := h.service.CreateUser(req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

func (h *Handler) UpdateUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	user, err := h.service.UpdateUser(id, req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

func (h *Handler) DeleteUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	if err := h.service.DeleteUser(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User deactivated successfully",
	})
}

func (h *Handler) ListRoles(c *fiber.Ctx) error {
	roles, err := h.service.ListRoles()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": roles})
}

func (h *Handler) UpdateRoleMenus(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid role ID"})
	}

	var req UpdateRoleMenusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	if err := h.service.UpdateRoleMenus(id, req.AllowedMenus); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Role permissions updated successfully"})
}

func (h *Handler) GetSettings(c *fiber.Ctx) error {
	settings, err := h.service.GetSettings()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": settings})
}

type SettingUpdateRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (h *Handler) UpdateSetting(c *fiber.Ctx) error {
	var req SettingUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	if err := h.service.UpdateSetting(req.Key, req.Value); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Setting saved successfully"})
}

func (h *Handler) UploadLogo(c *fiber.Ctx) error {
	file, err := c.FormFile("logo")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File logo is required"})
	}

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("logo_%d%s", time.Now().UnixNano(), ext)
	destination := filepath.Join(h.uploadDir, filename)

	if err := c.SaveFile(file, destination); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save logo file"})
	}

	logoURL := fmt.Sprintf("/uploads/%s", filename)
	if err := h.service.UpdateSetting("header_logo", logoURL); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success":   true,
		"logo_url":  logoURL,
		"message":   "Logo header uploaded successfully",
	})
}

func (h *Handler) UpdateMyProfile(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	id, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}
	// Do not allow changing role or active status from self profile update
	req.RoleID = nil
	req.IsActive = nil

	user, err := h.service.UpdateUser(id, req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

func (h *Handler) UploadMyAvatar(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	id, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	return h.saveAvatarForUser(c, id)
}

func (h *Handler) UploadUserAvatar(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	return h.saveAvatarForUser(c, id)
}

func (h *Handler) saveAvatarForUser(c *fiber.Ctx, userID uuid.UUID) error {
	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File avatar is required"})
	}

	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("avatar_%s_%d%s", userID.String()[:8], time.Now().UnixNano(), ext)
	destination := filepath.Join(h.uploadDir, filename)

	if err := c.SaveFile(file, destination); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save avatar file"})
	}

	avatarURL := fmt.Sprintf("/uploads/%s", filename)
	user, err := h.service.UpdateUser(userID, UpdateUserRequest{AvatarURL: &avatarURL})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"avatar_url": avatarURL,
		"data":       user,
		"message":    "Foto profil berhasil diperbarui",
	})
}

func (h *Handler) ListAttendances(c *fiber.Ctx) error {
	userID := c.Query("user_id")
	list, err := h.service.ListAttendances(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": list})
}

func (h *Handler) CreateAttendance(c *fiber.Ctx) error {
	var req CreateAttendanceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}
	if req.UserID == "" {
		if userIDStr, ok := c.Locals("user_id").(string); ok {
			req.UserID = userIDStr
		}
	}
	item, err := h.service.CreateAttendance(req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": item})
}

func (h *Handler) DeleteAttendance(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID"})
	}
	if err := h.service.DeleteAttendance(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Data absensi dihapus"})
}
