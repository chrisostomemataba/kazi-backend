package customer

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

func (h *Handler) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	profile, err := h.service.GetProfile(c.Context(), userID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, profile, "Profile retrieved successfully")
}

func (h *Handler) AddLocation(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req AddLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	if err := h.service.AddLocation(c.Context(), userID, &req); err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, nil, "Location added successfully")
}

func (h *Handler) GetLocations(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	locations, err := h.service.GetLocations(c.Context(), userID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, fiber.Map{
		"locations": locations,
		"total":     len(locations),
	}, "Locations retrieved successfully")
}

func (h *Handler) DeleteLocation(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	locationID := c.Params("location_id")

	if err := h.service.DeleteLocation(c.Context(), userID, locationID); err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, nil, "Location deleted successfully")
}