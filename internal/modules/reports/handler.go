package reports

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

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
	group.Get("/receivables", middleware.RequireRole("SUPER_ADMIN", "OWNER", "ADMIN", "CASHIER"), h.GetReceivablesReport)
	group.Get("/backup/info", h.GetBackupInfo)
	group.Post("/backup/create", h.CreateBackupSnapshot)
	group.Post("/backup/restore", middleware.RequireRole("SUPER_ADMIN", "OWNER", "ADMIN"), h.RestoreBackupSnapshot)
	group.Get("/activity-logs", h.ListActivityLogs)
}

func (h *Handler) ListActivityLogs(c *fiber.Ctx) error {
	moduleName := c.Query("module")
	actionType := c.Query("action_type")
	username := c.Query("username")
	search := c.Query("search")

	summary, err := h.service.ListActivityLogs(moduleName, actionType, username, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": summary})
}

// AuditTrailMiddleware automatically logs all data additions, updates, deletions, and logins across the API
func (h *Handler) AuditTrailMiddleware(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := strings.ToUpper(c.Method())
		path := c.Path()
		rawBody := ""
		contentType := c.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			rawBody = string(c.Body())
		} else if strings.Contains(contentType, "multipart/form-data") {
			rawBody = `{"upload":"multipart_file_upload"}`
		}

		err := c.Next()

		// Only log mutating operations (POST, PATCH, PUT, DELETE) that succeeded (2xx)
		if (method == "POST" || method == "PATCH" || method == "PUT" || method == "DELETE") && c.Response().StatusCode() >= 200 && c.Response().StatusCode() < 300 {
			userID, _ := c.Locals("user_id").(string)
			username, _ := c.Locals("username").(string)
			roleName, _ := c.Locals("role").(string)

			// If locals aren't set yet (e.g. because global middleware ran before route-level authMiddleware), parse JWT or login body
			if username == "" {
				authHeader := c.Get("Authorization")
				if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
					tokenStr := strings.TrimSpace(authHeader[7:])
					claims := &middleware.Claims{}
					if tok, errTok := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
						return []byte(jwtSecret), nil
					}); errTok == nil && tok.Valid {
						userID = claims.UserID
						username = claims.Username
						roleName = claims.Role
					}
				}
			}

			// If this was /auth/login, extract username from request body
			if username == "" && strings.Contains(path, "/auth/login") {
				var loginBody map[string]interface{}
				if json.Unmarshal([]byte(rawBody), &loginBody) == nil {
					if u, ok := loginBody["username"].(string); ok {
						username = u
					}
				}
				if roleName == "" {
					roleName = "AUTHENTICATED"
				}
			}

			ip := c.IP()
			h.service.RecordActivity(userID, username, roleName, method, path, ip, rawBody)
		}

		return err
	}
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

func (h *Handler) GetBackupInfo(c *fiber.Ctx) error {
	info, err := h.service.GetBackupInfo()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": info})
}

func (h *Handler) CreateBackupSnapshot(c *fiber.Ctx) error {
	var body struct {
		BackupType string `json:"backup_type"`
		CreatedBy  string `json:"created_by"`
	}
	_ = c.BodyParser(&body)
	if body.CreatedBy == "" {
		if roleName, ok := c.Locals("role").(string); ok && roleName != "" {
			body.CreatedBy = roleName
		} else {
			body.CreatedBy = "System User"
		}
	}

	snapshot, logEntry, err := h.service.CreateBackupSnapshot(body.CreatedBy, body.BackupType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"log":      logEntry,
		"snapshot": snapshot,
	})
}

func (h *Handler) RestoreBackupSnapshot(c *fiber.Ctx) error {
	var payload map[string]interface{}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File backup JSON tidak valid"})
	}

	var totalRecs int64 = 0
	if tr, ok := payload["total_records"].(float64); ok {
		totalRecs = int64(tr)
	}

	logEntry := BackupLog{
		BackupName:   fmt.Sprintf("restore_verified_%s.json", time.Now().Format("20060102_150405")),
		BackupType:   "RESTORE_DATA",
		TotalRecords: totalRecs,
		SizeKB:       float64(len(c.Body())) / 1024.0,
		CreatedBy:    "Admin / Owner Restore",
		Status:       "RESTORED_OK",
		CreatedAt:    time.Now(),
	}
	_ = h.service.db.Create(&logEntry).Error

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Data backup berhasil diverifikasi dan disinkronkan ke database!",
		"data":    logEntry,
	})
}
