package chat

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

func (h *Handler) SendMessage(c *fiber.Ctx) error {
	senderID := c.Locals("userID").(uuid.UUID)
	bookingID := c.Params("booking_id")

	var req SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	message, err := h.service.SendMessage(c.Context(), senderID, bookingID, &req)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, message, "Message sent")
}

func (h *Handler) GetMessages(c *fiber.Ctx) error {
	readerID := c.Locals("userID").(uuid.UUID)
	bookingID := c.Params("booking_id")

	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	messages, err := h.service.GetMessages(c.Context(), readerID, bookingID, limit, offset)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, messages, "Messages retrieved")
}

func (h *Handler) MarkRead(c *fiber.Ctx) error {
	readerID := c.Locals("userID").(uuid.UUID)
	bookingID := c.Params("booking_id")

	if err := h.service.MarkRead(c.Context(), readerID, bookingID); err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, nil, "Conversation marked read")
}

func (h *Handler) GetConversations(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	conversations, err := h.service.GetConversations(c.Context(), userID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, conversations, "Conversations retrieved")
}

func (h *Handler) GetUnreadCount(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	count, err := h.service.GetUnreadCount(c.Context(), userID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, count, "Unread count retrieved")
}
