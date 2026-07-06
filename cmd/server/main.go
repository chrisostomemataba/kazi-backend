package main

import (
	"log/slog"
	"os"

	"kazi-backend/config"
	"kazi-backend/internal/admin"
	"kazi-backend/internal/auth"
	"kazi-backend/internal/booking"
	"kazi-backend/internal/common/database"
	"kazi-backend/internal/common/sms"
	"kazi-backend/internal/common/storage"
	wsHub "kazi-backend/internal/common/websocket"
	"kazi-backend/internal/customer"
	"kazi-backend/internal/maid"
	"kazi-backend/internal/notification"
	"kazi-backend/internal/payment"
	"kazi-backend/internal/review"
	"kazi-backend/internal/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	cfg := config.LoadConfig()

	setupLogger(cfg.Environment)

	db, err := database.Connect(cfg.DatabaseURL, cfg.Environment == "development")
	if err != nil {
		slog.Error("Database connection failed", "error", err)
		os.Exit(1)
	}

	if err := database.AutoMigrate(db,
		&auth.User{},
		&auth.UserRole{},
		&auth.OTPCode{},
		&maid.MaidProfile{},
		&maid.MaidService{},
		&maid.MaidVerificationDocument{},
		&maid.MaidStatistics{},
		&maid.MaidWallet{},
		&maid.WalletTransaction{},
		&customer.CustomerProfile{},
		&customer.CustomerLocation{},
		&customer.CustomerStatistics{},
		&booking.Booking{},
		&booking.BookingLocation{},
		&booking.BookingPricing{},
		&booking.BookingTimeline{},
		&booking.Payment{},
		&review.Review{},
		&notification.Notification{},
		&admin.AdminUser{},
		&admin.AuditLog{},
	); err != nil {
		slog.Error("Migration failed", "error", err)
		os.Exit(1)
	}

	// Infrastructure
	smsService := sms.NewSMSService(cfg.SMSAPIToken, cfg.SMSSenderID, cfg.SMSBaseURL)

	minioService, err := storage.NewMinIOService(
		cfg.MinIOEndpoint,
		cfg.MinIOAccessKey,
		cfg.MinIOSecretKey,
		cfg.MinIOBucket,
		cfg.MinIOUseSSL,
	)
	if err != nil {
		slog.Error("MinIO initialization failed", "error", err)
		os.Exit(1)
	}

	hub := wsHub.NewHub()
	go hub.Run()

	// Domain modules — order matters: shared deps built first
	isDev := cfg.Environment == "development"
	authModule := auth.NewModule(db, smsService, cfg.JWTSecret, isDev)
	notifModule := notification.NewModule(db, hub, cfg.JWTSecret)
	maidModule := maid.NewModule(db, authModule.Repository, minioService, notifModule.Service)
	customerModule := customer.NewModule(db, authModule.Repository)
	paymentClient := payment.NewPaymentClient(cfg.PaymentServiceURL)
	bookingModule := booking.NewModule(db, authModule.Repository, maidModule.Repository, customerModule.Repository, notifModule.Service, paymentClient)
	paymentWebhookHandler := payment.NewWebhookHandler(bookingModule.Service, cfg.PaymentWebhookSecret)
	reviewModule := review.NewModule(db, authModule.Repository, bookingModule.Repository)
	adminModule := admin.NewModule(db, minioService, notifModule.Service, cfg.JWTSecret)

	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
		BodyLimit:    50 * 1024 * 1024,
	})

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	routes.Register(app, routes.Handlers{
		Auth:           authModule.Handler,
		Maid:           maidModule.Handler,
		Customer:       customerModule.Handler,
		Booking:        bookingModule.Handler,
		Review:         reviewModule.Handler,
		Admin:          adminModule.Handler,
		Notification:   notifModule.Handler,
		WebSocket:      notifModule.WebSocketHandler,
		PaymentWebhook: paymentWebhookHandler,
	}, cfg.JWTSecret)

	slog.Info("Server starting", "port", cfg.Port)
	slog.Info("MinIO endpoint", "endpoint", cfg.MinIOEndpoint)
	slog.Info("WebSocket available", "url", "ws://localhost:"+cfg.Port+"/ws")

	if err := app.Listen(":" + cfg.Port); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func setupLogger(env string) {
	if env == "development" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	} else {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"error":   err.Error(),
	})
}
