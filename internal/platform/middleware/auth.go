package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	TenantID string `json:"tenant_id,omitempty"`
	jwt.RegisteredClaims
}

func AuthMiddleware(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "UNAUTHORIZED",
				"message": "Authorization header missing",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "UNAUTHORIZED",
				"message": "Invalid token format",
			})
		}

		tokenString := parts[1]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "UNAUTHORIZED",
				"message": "Invalid or expired token",
			})
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("username", claims.Username)
		c.Locals("role", claims.Role)
		c.Locals("tenant_id", claims.TenantID)

		return c.Next()
	}
}

func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("role").(string)
		if !ok || userRole == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "FORBIDDEN",
				"message": "User role is not determined",
			})
		}

		// Owner and Super Admin have universal access
		if strings.ToUpper(userRole) == "OWNER" || strings.ToUpper(userRole) == "SUPER_ADMIN" {
			return c.Next()
		}

		for _, role := range allowedRoles {
			if strings.EqualFold(role, userRole) {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "FORBIDDEN",
			"message": "You do not have permission to perform this action",
		})
	}
}
