package main

import (
	"log"

	"kazi-backend/config"
	"kazi-backend/internal/auth"
	"kazi-backend/internal/common/database"
	"kazi-backend/internal/common/sms"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.Connect(cfg.DatabaseURL, cfg.Environment == "development")
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	// Auto-migrate models
	if err := database.AutoMigrate(db,
		&auth.User{},
		&auth.UserRole{},
		&auth.OTPCode{},
	); err != nil {
		log.Fatal("Migration failed:", err)
	}

	// Initialize services with Notify Africa credentials
	smsService := sms.NewSMSService(cfg.SMSAPIToken, cfg.SMSSenderID, cfg.SMSBaseURL)
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, smsService, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService)

	// Setup Fiber
	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
	})

	app.Use(logger.New())
	app.Use(cors.New())

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Routes
	api := app.Group("/api/v1")
	
	authRoutes := api.Group("/auth")
	authRoutes.Post("/request-otp", authHandler.RequestOTP)
	authRoutes.Post("/verify-otp", authHandler.VerifyOTP)
	authRoutes.Post("/complete-profile", authHandler.CompleteProfile)

	log.Printf("🚀 Server starting on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error":   err.Error(),
	})
}