package routes

import (
	"kazi-backend/internal/admin"
	"kazi-backend/internal/auth"
	"kazi-backend/internal/booking"
	"kazi-backend/internal/common/middleware"
	"kazi-backend/internal/maid"
	"kazi-backend/internal/notification"
	"kazi-backend/internal/review"
	"kazi-backend/internal/customer"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type Handlers struct {
	Auth         *auth.Handler
	Maid         *maid.Handler
	Customer     *customer.Handler
	Booking      *booking.Handler
	Review       *review.Handler
	Admin        *admin.Handler
	Notification *notification.Handler
	WebSocket    *notification.WebSocketHandler
}

func Register(app *fiber.App, h Handlers, jwtSecret string) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := app.Group("/api/v1")

	registerAuthRoutes(api, h.Auth)
	registerCustomerRoutes(api, h.Customer, jwtSecret)
	registerMaidRoutes(api, h.Maid, jwtSecret)
	registerBookingRoutes(api, h.Booking, jwtSecret)
	registerReviewRoutes(api, h.Review, jwtSecret)
	registerNotificationRoutes(api, h.Notification, jwtSecret)
	registerAdminRoutes(api, h.Admin, jwtSecret)
	registerWebSocketRoute(app, h.WebSocket)
}

func registerAuthRoutes(api fiber.Router, h *auth.Handler) {
	g := api.Group("/auth")
	g.Post("/request-otp", h.RequestOTP)
	g.Post("/verify-otp", h.VerifyOTP)
	g.Post("/complete-profile", h.CompleteProfile)
	g.Post("/login", h.Login)
}

func registerCustomerRoutes(api fiber.Router, h *customer.Handler, jwtSecret string) {
	g := api.Group("/customer", middleware.RequireAuth(jwtSecret))
	g.Get("/profile", middleware.RequireRole("customer"), h.GetProfile)
	g.Get("/locations", middleware.RequireRole("customer"), h.GetLocations)
	g.Post("/locations", middleware.RequireRole("customer"), h.AddLocation)
	g.Delete("/locations/:location_id", middleware.RequireRole("customer"), h.DeleteLocation)
}

func registerMaidRoutes(api fiber.Router, h *maid.Handler, jwtSecret string) {
	// Protected maid-only routes
	protected := api.Group("/maid", middleware.RequireAuth(jwtSecret))
	protected.Post("/verification/submit", middleware.RequireRole("maid"), h.SubmitVerification)
	protected.Post("/verification/upload-video", middleware.RequireRole("maid"), h.UploadVerificationVideo)
	protected.Post("/verification/upload-id", middleware.RequireRole("maid"), h.UploadIDPhoto)
	protected.Get("/profile", middleware.RequireRole("maid"), h.GetMyProfile)
	protected.Put("/profile/location", middleware.RequireRole("maid"), h.UpdateLocation)
	protected.Put("/profile/contract-rate", middleware.RequireRole("maid"), h.UpdateContractRate)
	protected.Get("/wallet", middleware.RequireRole("maid"), h.GetWallet)

	// Public maid search routes — no auth required
	public := api.Group("/maids")
	public.Get("/search", h.SearchMaids)
	public.Get("/:maid_id", h.GetMaidByID)
}

func registerBookingRoutes(api fiber.Router, h *booking.Handler, jwtSecret string) {
	// Customer booking actions
	customer := api.Group("/bookings", middleware.RequireAuth(jwtSecret))
	customer.Post("/validate", middleware.RequireRole("customer"), h.ValidateBooking)
	customer.Post("/create", middleware.RequireRole("customer"), h.CreateBooking)
	customer.Get("/my-bookings", middleware.RequireRole("customer"), h.GetMyBookings)
	customer.Get("/:id", middleware.RequireRole("customer"), h.GetBookingByID)     
	customer.Post("/:id/initiate-payment", middleware.RequireRole("customer"), h.InitiatePayment)
	customer.Post("/:id/confirm", middleware.RequireRole("customer"), h.ConfirmCompletion)

	// Maid booking actions (Workflows C3, E1, E2)
	maidBookings := api.Group("/maid/bookings", middleware.RequireAuth(jwtSecret))
	maidBookings.Get("/requests", middleware.RequireRole("maid"), h.GetMaidBookings)
	maidBookings.Post("/:id/accept", middleware.RequireRole("maid"), h.AcceptBooking)
	maidBookings.Post("/:id/decline", middleware.RequireRole("maid"), h.DeclineBooking)
	maidBookings.Post("/:id/arrive", middleware.RequireRole("maid"), h.MarkArrival)
	maidBookings.Post("/:id/complete", middleware.RequireRole("maid"), h.MarkComplete)

	// Live location tracking
	customer.Post("/:id/location", h.UpdateMaidLocation)
	customer.Get("/:id/location", h.GetMaidLocation)
}

func registerReviewRoutes(api fiber.Router, h *review.Handler, jwtSecret string) {
	g := api.Group("/reviews", middleware.RequireAuth(jwtSecret))
	g.Post("/", middleware.RequireRole("customer"), h.CreateReview)
	g.Get("/maid/:maid_id", h.GetMaidReviews)
}

func registerNotificationRoutes(api fiber.Router, h *notification.Handler, jwtSecret string) {
	g := api.Group("/notifications", middleware.RequireAuth(jwtSecret))
	g.Get("/", h.GetMyNotifications)
	g.Put("/:id/read", h.MarkAsRead)
}

func registerAdminRoutes(api fiber.Router, h *admin.Handler, jwtSecret string) {
	g := api.Group("/admin")
	g.Post("/login", h.Login)

	protected := g.Use(middleware.RequireAdmin(jwtSecret))
	protected.Get("/verifications/pending", h.GetPendingVerifications)
	protected.Get("/verifications/:maid_id", h.GetVerificationDetails)
	protected.Post("/verifications/approve", h.ApproveVerification)
	protected.Post("/verifications/reject", h.RejectVerification)
}

func registerWebSocketRoute(app *fiber.App, h *notification.WebSocketHandler) {
	app.Use("/ws", h.UpgradeMiddleware)
	app.Get("/ws", websocket.New(h.HandleConnection))
}