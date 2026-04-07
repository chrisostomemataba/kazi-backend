package main

import (
	"log"

	"kazi-backend/config"
	"kazi-backend/internal/admin"
	"kazi-backend/internal/auth"
	"kazi-backend/internal/booking"
	"kazi-backend/internal/customer"
	"kazi-backend/internal/common/database"
	"kazi-backend/internal/common/middleware"
	"kazi-backend/internal/common/sms"
	"kazi-backend/internal/common/storage"
	wsHub "kazi-backend/internal/common/websocket"
	"kazi-backend/internal/maid"
	"kazi-backend/internal/notification"
	"kazi-backend/internal/review"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.Connect(cfg.DatabaseURL, cfg.Environment == "development")
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	// Auto-migrate all models
	if err := database.AutoMigrate(db,
		// Auth
		&auth.User{},
		&auth.UserRole{},
		&auth.OTPCode{},
		// Maid
		&maid.MaidProfile{},
		&maid.MaidService{},
		&maid.MaidVerificationDocument{},
		&maid.MaidStatistics{},
		&maid.MaidWallet{},
		&maid.WalletTransaction{},
		// Customer
		&customer.CustomerProfile{},
		&customer.CustomerLocation{},
		&customer.CustomerStatistics{},
		// Booking
		&booking.Booking{},
		&booking.BookingLocation{},
		&booking.BookingPricing{},
		&booking.BookingTimeline{},
		&booking.Payment{},
		// Review
		&review.Review{},
		// Notifications
		&notification.Notification{},
		// Admin
		&admin.AdminUser{},
		&admin.AuditLog{},
	); err != nil {
		log.Fatal("Migration failed:", err)
	}

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

	hub := wsHub.NewHub()
	go hub.Run()

	// Repositories
	authRepo := auth.NewRepository(db)
	maidRepo := maid.NewRepository(db)
	customerRepo := customer.NewRepository(db)
	bookingRepo := booking.NewRepository(db)
	reviewRepo    := review.NewRepository(db)
	adminRepo := admin.NewRepository(db)

	// Services
	notificationService := notification.NewService(db)
	isDev := cfg.Environment == "development"
	authService := auth.NewService(authRepo, smsService, cfg.JWTSecret, isDev)
	maidService := maid.NewService(maidRepo, authRepo, minioService, notificationService)
	customerService := customer.NewService(customerRepo, authRepo)
	bookingService := booking.NewService(bookingRepo, authRepo, maidRepo, customerRepo, notificationService)
	reviewService := review.NewService(reviewRepo, authRepo, bookingRepo)
	adminService := admin.NewService(adminRepo, minioService, notificationService, cfg.JWTSecret)

	// Handlers
	authHandler := auth.NewHandler(authService)
	maidHandler := maid.NewHandler(maidService)
	customerHandler := customer.NewHandler(customerService)
	bookingHandler := booking.NewHandler(bookingService)
	reviewHandler := review.NewHandler(reviewService)
	adminHandler := admin.NewHandler(adminService)
	notificationHandler := notification.NewHandler(notificationService)
	wsHandler := notification.NewWebSocketHandler(hub, cfg.JWTSecret)

	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
		BodyLimit:    50 * 1024 * 1024,
	})

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := app.Group("/api/v1")

	// Auth routes (public)
	authRoutes := api.Group("/auth")
	authRoutes.Post("/request-otp", authHandler.RequestOTP)
	authRoutes.Post("/verify-otp", authHandler.VerifyOTP)
	authRoutes.Post("/complete-profile", authHandler.CompleteProfile)
	authRoutes.Post("/login", authHandler.Login)

	// Customer routes (protected)
	customerRoutes := api.Group("/customer", middleware.RequireAuth(cfg.JWTSecret))
	customerRoutes.Get("/profile", middleware.RequireRole("customer"), customerHandler.GetProfile)
	customerRoutes.Get("/locations", middleware.RequireRole("customer"), customerHandler.GetLocations)
	customerRoutes.Post("/locations", middleware.RequireRole("customer"), customerHandler.AddLocation)
	customerRoutes.Delete("/locations/:location_id", middleware.RequireRole("customer"), customerHandler.DeleteLocation)

	// Maid routes (protected)
	maidRoutes := api.Group("/maid", middleware.RequireAuth(cfg.JWTSecret))
	maidRoutes.Post("/verification/submit", middleware.RequireRole("maid"), maidHandler.SubmitVerification)
	maidRoutes.Post("/verification/upload-video", middleware.RequireRole("maid"), maidHandler.UploadVerificationVideo)
	maidRoutes.Post("/verification/upload-id", middleware.RequireRole("maid"), maidHandler.UploadIDPhoto)
	maidRoutes.Get("/profile", middleware.RequireRole("maid"), maidHandler.GetMyProfile)
	maidRoutes.Put("/profile/location", middleware.RequireRole("maid"), maidHandler.UpdateLocation)
	maidRoutes.Put("/profile/contract-rate", middleware.RequireRole("maid"), maidHandler.UpdateContractRate)
	maidRoutes.Get("/wallet", middleware.RequireRole("maid"), maidHandler.GetWallet)

	// Maid search routes (public - no auth needed)
	maidsRoutes := api.Group("/maids")
	maidsRoutes.Get("/search", maidHandler.SearchMaids)
	maidsRoutes.Get("/:maid_id", maidHandler.GetMaidByID)

	// Customer booking routes
	bookingRoutes := api.Group("/bookings", middleware.RequireAuth(cfg.JWTSecret))
	bookingRoutes.Post("/validate", middleware.RequireRole("customer"), bookingHandler.ValidateBooking)
	bookingRoutes.Post("/create", middleware.RequireRole("customer"), bookingHandler.CreateBooking)
	bookingRoutes.Get("/my-bookings", middleware.RequireRole("customer"), bookingHandler.GetMyBookings)
	bookingRoutes.Post("/:id/initiate-payment", middleware.RequireRole("customer"), bookingHandler.InitiatePayment)
	bookingRoutes.Post("/:id/confirm", middleware.RequireRole("customer"), bookingHandler.ConfirmCompletion) 

	// Maid booking routes
	maidBookingRoutes := api.Group("/maid/bookings", middleware.RequireAuth(cfg.JWTSecret))
	maidBookingRoutes.Get("/requests", middleware.RequireRole("maid"), bookingHandler.GetMaidBookings)         // list with ?status=pending_maid
	maidBookingRoutes.Post("/:id/accept", middleware.RequireRole("maid"), bookingHandler.AcceptBooking)        // Workflow C3
	maidBookingRoutes.Post("/:id/decline", middleware.RequireRole("maid"), bookingHandler.DeclineBooking)      // Workflow C3
	maidBookingRoutes.Post("/:id/arrive", middleware.RequireRole("maid"), bookingHandler.MarkArrival)          // Workflow E1
	maidBookingRoutes.Post("/:id/complete", middleware.RequireRole("maid"), bookingHandler.MarkComplete)       // Workflow E2

	// Review routes (protected)
	reviewRoutes := api.Group("/reviews", middleware.RequireAuth(cfg.JWTSecret))
	reviewRoutes.Post("/", middleware.RequireRole("customer"), reviewHandler.CreateReview)
	reviewRoutes.Get("/maid/:maid_id", reviewHandler.GetMaidReviews)	

	// Notification routes (protected)
	notificationRoutes := api.Group("/notifications", middleware.RequireAuth(cfg.JWTSecret))
	notificationRoutes.Get("/", notificationHandler.GetMyNotifications)
	notificationRoutes.Put("/:id/read", notificationHandler.MarkAsRead)

	// WebSocket route
	app.Use("/ws", wsHandler.UpgradeMiddleware)
	app.Get("/ws", websocket.New(wsHandler.HandleConnection))

	// Admin routes
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