package booking

import (
	"kazi-backend/internal/common/util"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ── Workflow C2: Customer creates booking ─────────────────────────────────────

func (h *Handler) ValidateBooking(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req ValidateBookingRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}
	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	result, err := h.service.ValidateBooking(c.Context(), userID, &req)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, result, "Validation complete")
}

func (h *Handler) CreateBooking(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req CreateBookingRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}
	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	booking, err := h.service.CreateBooking(c.Context(), userID, &req)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, booking, "Booking created successfully")
}

// ── Workflow C3: Maid accepts booking ─────────────────────────────────────────

func (h *Handler) AcceptBooking(c *fiber.Ctx) error {
	maidID := c.Locals("userID").(uuid.UUID)
	bookingID := c.Params("id")

	booking, err := h.service.AcceptBooking(c.Context(), maidID, bookingID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, booking, "Booking accepted")
}

// ── Workflow C3: Maid declines booking ────────────────────────────────────────

func (h *Handler) DeclineBooking(c *fiber.Ctx) error {
	maidID := c.Locals("userID").(uuid.UUID)
	bookingID := c.Params("id")

	var req DeclineBookingRequest
	if err := c.BodyParser(&req); err != nil {
		req.Reason = "Msaidizi hapatikani"
	}

	if err := h.service.DeclineBooking(c.Context(), maidID, bookingID, req.Reason); err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, nil, "Booking declined")
}

// ── Workflow D: Customer initiates payment ────────────────────────────────────

func (h *Handler) InitiatePayment(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	bookingID := c.Params("id")

	var req InitiatePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}
	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	payment, err := h.service.InitiatePayment(c.Context(), userID, bookingID, &req)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, payment, "Payment initiated successfully")
}

// ── Workflow E1: Maid marks arrival ──────────────────────────────────────────

func (h *Handler) MarkArrival(c *fiber.Ctx) error {
	maidID := c.Locals("userID").(uuid.UUID)
	bookingID := c.Params("id")

	var req ArrivalRequest
	// GPS is optional — don't fail if not provided
	c.BodyParser(&req)

	booking, err := h.service.MarkArrival(c.Context(), maidID, bookingID, &req)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, booking, "Arrival marked, work started")
}

// ── Workflow E2: Maid marks work complete ─────────────────────────────────────

func (h *Handler) MarkComplete(c *fiber.Ctx) error {
	maidID := c.Locals("userID").(uuid.UUID)
	bookingID := c.Params("id")

	booking, err := h.service.MarkComplete(c.Context(), maidID, bookingID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, booking, "Work marked as complete, awaiting customer confirmation")
}

// ── Workflow E2+F: Customer confirms completion → releases payment ────────────

func (h *Handler) ConfirmCompletion(c *fiber.Ctx) error {
	customerID := c.Locals("userID").(uuid.UUID)
	bookingID := c.Params("id")

	booking, err := h.service.ConfirmCompletion(c.Context(), customerID, bookingID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, booking, "Kazi imethibitishwa, malipo yametumwa kwa msaidizi")
}

// ── Fetch endpoints ───────────────────────────────────────────────────────────

func (h *Handler) GetBookingByID(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	bookingID := c.Params("id")

	booking, err := h.service.GetBookingByID(c.Context(), userID, bookingID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusNotFound, err.Error())
	}

	return util.SuccessResponse(c, booking, "Booking retrieved")
}

func (h *Handler) GetMyBookings(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	status := c.Query("status", "")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	bookings, err := h.service.GetCustomerBookings(c.Context(), userID, status, page, limit)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, fiber.Map{
		"bookings": bookings,
		"page":     page,
		"limit":    limit,
		"total":    len(bookings),
	}, "Bookings retrieved")
}

func (h *Handler) GetMaidBookings(c *fiber.Ctx) error {
	maidID := c.Locals("userID").(uuid.UUID)
	status := c.Query("status", "")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	bookings, err := h.service.GetMaidBookings(c.Context(), maidID, status, page, limit)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, fiber.Map{
		"bookings": bookings,
		"page":     page,
		"limit":    limit,
		"total":    len(bookings),
	}, "Bookings retrieved")
}