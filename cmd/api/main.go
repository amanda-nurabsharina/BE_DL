package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"dental-lab-backend/internal/config"
	"dental-lab-backend/internal/modules/auth"
	"dental-lab-backend/internal/modules/customers"
	"dental-lab-backend/internal/modules/finance"
	"dental-lab-backend/internal/modules/inventory"
	"dental-lab-backend/internal/modules/production"
	"dental-lab-backend/internal/modules/reports"
	"dental-lab-backend/internal/platform/database"
	"dental-lab-backend/internal/platform/middleware"
)

func main() {
	// 1. Load Configuration
	cfg := config.LoadConfig()

	// Ensure upload directory exists
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	// 2. Connect to Database
	db, err := database.Connect(cfg.DatabaseURL, cfg.AppEnv == "development")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 3. Initialize Fiber App
	app := fiber.New(fiber.Config{
		AppName:      "Dental Laboratory Management System API",
		BodyLimit:    int(cfg.UploadMaxBytes),
		ServerHeader: "Fiber",
	})

	// Middlewares
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AppPublicURL + ", http://localhost:3000, http://127.0.0.1:3000",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// Serve static files for uploads
	app.Static("/uploads", cfg.UploadDir)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "dental-lab-api",
			"env":     cfg.AppEnv,
		})
	})

	// 4. Register Services & Handlers
	authMiddleware := middleware.AuthMiddleware(cfg.JWTSecret)

	apiV1 := app.Group("/api/v1")

	// Auth Module
	authService := auth.NewService(db, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService, cfg.UploadDir)
	authHandler.RegisterRoutes(apiV1, authMiddleware)

	// Customers Module
	custService := customers.NewService(db)
	custHandler := customers.NewHandler(custService)
	custHandler.RegisterRoutes(apiV1, authMiddleware)

	// Inventory Module
	invService := inventory.NewService(db)
	invHandler := inventory.NewHandler(invService)
	invHandler.RegisterRoutes(apiV1, authMiddleware)

	// Production Module
	prodService := production.NewService(db)
	prodHandler := production.NewHandler(prodService, cfg.UploadDir)
	prodHandler.RegisterRoutes(apiV1, authMiddleware)

	// Finance Module
	finService := finance.NewService(db)
	finHandler := finance.NewHandler(finService)
	finHandler.RegisterRoutes(apiV1, authMiddleware)

	// Reports Module
	repService := reports.NewService(db)
	repHandler := reports.NewHandler(repService)
	repHandler.RegisterRoutes(apiV1, authMiddleware)

	// 5. Start Server
	log.Printf("Dental Lab API server running on %s", cfg.AppAddr)
	if err := app.Listen(cfg.AppAddr); err != nil {
		log.Fatalf("Server stopped with error: %v", err)
	}
}
