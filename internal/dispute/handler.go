package dispute

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

func (h *Handler) CreateDispute(c *fiber.Ctx) error {
	reporterID := c.Locals("userID").(uuid.UUID)

	var req CreateDisputeRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	resp, err := h.service.CreateDispute(c.Context(), reporterID, &req)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, resp, "Report submitted successfully")
}

func (h *Handler) GetMyDisputes(c *fiber.Ctx) error {
	reporterID := c.Locals("userID").(uuid.UUID)

	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	disputes, err := h.service.GetMyDisputes(c.Context(), reporterID, limit, offset)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, disputes, "Reports retrieved successfully")
}

func (h *Handler) UploadEvidence(c *fiber.Ctx) error {
	reporterID := c.Locals("userID").(uuid.UUID)

	file, err := c.FormFile("evidence")
	if err != nil {
		return util.ValidationErrorResponse(c, "Evidence photo is required")
	}

	if file.Size > 5*1024*1024 {
		return util.ValidationErrorResponse(c, "Image file too large (max 5MB)")
	}

	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" {
		return util.ValidationErrorResponse(c, "Only JPEG and PNG image formats are allowed")
	}

	objectName, err := h.service.UploadEvidence(c.Context(), reporterID, file)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, EvidenceUploadResponse{ObjectName: objectName}, "Evidence uploaded successfully")
}
