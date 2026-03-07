package main

import (
	"log"

	"kazi-backend/config"
	"kazi-backend/internal/admin"
	"kazi-backend/internal/auth"
	"kazi-backend/internal/common/database"
	"kazi-backend/internal/common/middleware"
	"kazi-backend/internal/common/sms"
	"kazi-backend/internal/common/storage"
	wsHub "kazi-backend/internal/common/websocket"
	"kazi-backend/internal/maid"
	"kazi-backend/internal/notification"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

func main() {
	cfg := config.LoadConfig()

	// Database connection
	db, err := database.Connect(cfg.DatabaseURL, cfg.Environment == "development")
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	// Auto-migrate all models
	if err := database.AutoMigrate(db,
		&auth.User{},
		&auth.UserRole{},
		&auth.OTPCode{},
		&maid.MaidProfile{},
		&maid.MaidService{},
		&maid.MaidVerificationDocument{},
		&notification.Notification{},
		&admin.AdminUser{},
		&admin.AuditLog{},
	); err != nil {
		log.Fatal("Migration failed:", err)
	}

	// Initialize services
	smsService := sms.NewSMSService(cfg.SMSAPIToken, cfg.SMSSenderID, cfg.SMSBaseURL)
	
	minioService, err := storage.NewMinIOService(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOUseSSL,
	)
	if err != nil {
		log.Fatal("MinIO initialization failed:", err)
	}

	// WebSocket Hub
	hub := wsHub.NewHub()
	go hub.Run()

	// Repositories
	authRepo := auth.NewRepository(db)
	maidRepo := maid.NewRepository(db)
	adminRepo := admin.NewRepository(db)

	// Services
	notificationService := notification.NewService(db)
	authService := auth.NewService(authRepo, smsService, cfg.JWTSecret)
	maidService := maid.NewService(maidRepo, minioService, notificationService)
	adminService := admin.NewService(adminRepo, minioService, notificationService, cfg.JWTSecret)

	// Handlers
	authHandler := auth.NewHandler(authService)
	maidHandler := maid.NewHandler(maidService)
	adminHandler := admin.NewHandler(adminService)
	notificationHandler := notification.NewHandler(notificationService)
	wsHandler := notification.NewWebSocketHandler(hub, cfg.JWTSecret)

	// Setup Fiber
	app := fiber.New(fiber.Config{
		ErrorHandler:  customErrorHandler,
		BodyLimit:     50 * 1024 * 1024, // 50MB for video uploads
	})

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// API routes
	api := app.Group("/api/v1")

	// Auth routes (public)
	authRoutes := api.Group("/auth")
	authRoutes.Post("/request-otp", authHandler.RequestOTP)
	authRoutes.Post("/verify-otp", authHandler.VerifyOTP)
	authRoutes.Post("/complete-profile", authHandler.CompleteProfile)
	authRoutes.Post("/login", authHandler.Login)

	// Maid routes (protected)
	maidRoutes := api.Group("/maid", middleware.RequireAuth(cfg.JWTSecret))
	maidRoutes.Post("/verification/submit", middleware.RequireRole("maid"), maidHandler.SubmitVerification)
	maidRoutes.Post("/verification/upload-video", middleware.RequireRole("maid"), maidHandler.UploadVerificationVideo)
	maidRoutes.Post("/verification/upload-id", middleware.RequireRole("maid"), maidHandler.UploadIDPhoto)
	maidRoutes.Get("/profile", middleware.RequireRole("maid"), maidHandler.GetMyProfile)

	// Notification routes (protected)
	notificationRoutes := api.Group("/notifications", middleware.RequireAuth(cfg.JWTSecret))
	notificationRoutes.Get("/", notificationHandler.GetMyNotifications)
	notificationRoutes.Put("/:id/read", notificationHandler.MarkAsRead)

	// WebSocket route (protected via query token)
	app.Use("/ws", wsHandler.UpgradeMiddleware)
	app.Get("/ws", websocket.New(wsHandler.HandleConnection))

	// Admin routes (protected - admin only)
	adminRoutes := api.Group("/admin")
	adminRoutes.Post("/login", adminHandler.Login)
	
	adminProtected := adminRoutes.Use(middleware.RequireAdmin(cfg.JWTSecret))
	adminProtected.Get("/verifications/pending", adminHandler.GetPendingVerifications)
	adminProtected.Get("/verifications/:maid_id", adminHandler.GetVerificationDetails)
	adminProtected.Post("/verifications/approve", adminHandler.ApproveVerification)
	adminProtected.Post("/verifications/reject", adminHandler.RejectVerification)

	log.Printf("🚀 Server starting on port %s", cfg.Port)
	log.Printf("📁 MinIO endpoint: %s", cfg.MinIOEndpoint)
	log.Printf("🔔 WebSocket available at: ws://localhost:%s/ws", cfg.Port)
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