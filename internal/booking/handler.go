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

func (h *Handler) ValidateBooking(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req ValidateBookingRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	validation, err := h.service.ValidateBooking(c.Context(), userID, &req)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, validation, "Validation complete")
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

func (h *Handler) GetBookingByID(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	bookingID := c.Params("id")

	booking, err := h.service.GetBookingByID(c.Context(), userID, bookingID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusNotFound, err.Error())
	}

	return util.SuccessResponse(c, booking, "Booking retrieved successfully")
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
	}, "Bookings retrieved successfully")
}

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