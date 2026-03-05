package notification

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

func (h *Handler) GetMyNotifications(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)
	
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	notifications, err := h.service.GetUserNotifications(c.Context(), userID, limit, offset)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, fiber.Map{
		"notifications": notifications,
		"page":          page,
		"limit":         limit,
	}, "Notifications retrieved successfully")
}

func (h *Handler) MarkAsRead(c *fiber.Ctx) error {
	notificationIDStr := c.Params("id")
	notificationID, err := uuid.Parse(notificationIDStr)
	if err != nil {
		return util.ValidationErrorResponse(c, "Invalid notification ID")
	}

	if err := h.service.MarkAsRead(c.Context(), notificationID); err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, nil, "Notification marked as read")
}