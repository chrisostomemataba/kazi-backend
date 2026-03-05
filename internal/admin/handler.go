package admin

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

func (h *Handler) Login(c *fiber.Ctx) error {
	var req AdminLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	authResp, err := h.service.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusUnauthorized, err.Error())
	}

	return util.SuccessResponse(c, authResp, "Login successful")
}

func (h *Handler) GetPendingVerifications(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	response, err := h.service.GetPendingVerifications(c.Context(), page, limit)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, response, "Pending verifications retrieved successfully")
}

func (h *Handler) GetVerificationDetails(c *fiber.Ctx) error {
	maidIDStr := c.Params("maid_id")
	maidID, err := uuid.Parse(maidIDStr)
	if err != nil {
		return util.ValidationErrorResponse(c, "Invalid maid ID")
	}

	details, err := h.service.GetVerificationDetails(c.Context(), maidID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusNotFound, "Maid verification not found")
	}

	return util.SuccessResponse(c, details, "Verification details retrieved successfully")
}

func (h *Handler) ApproveVerification(c *fiber.Ctx) error {
	adminID := c.Locals("userID").(uuid.UUID)

	var req ApproveVerificationRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	maidID, err := uuid.Parse(req.MaidID)
	if err != nil {
		return util.ValidationErrorResponse(c, "Invalid maid ID")
	}

	ipAddress := c.IP()
	if err := h.service.ApproveVerification(c.Context(), adminID, maidID, ipAddress); err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, nil, "Verification approved successfully")
}

func (h *Handler) RejectVerification(c *fiber.Ctx) error {
	adminID := c.Locals("userID").(uuid.UUID)

	var req RejectVerificationRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	maidID, err := uuid.Parse(req.MaidID)
	if err != nil {
		return util.ValidationErrorResponse(c, "Invalid maid ID")
	}

	ipAddress := c.IP()
	if err := h.service.RejectVerification(c.Context(), adminID, maidID, req.Reason, ipAddress); err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, nil, "Verification rejected successfully")
}