package maid

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

func (h *Handler) SubmitVerification(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req VerificationSubmitRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	req.UserID = userID

	if err := h.service.SubmitVerification(c.Context(), &req); err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, nil, "Verification submitted successfully")
}

func (h *Handler) UploadVerificationVideo(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	file, err := c.FormFile("video")
	if err != nil {
		return util.ValidationErrorResponse(c, "Video file is required")
	}

	// Validate file size (max 50MB)
	if file.Size > 50*1024*1024 {
		return util.ValidationErrorResponse(c, "Video file too large (max 50MB)")
	}

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	if contentType != "video/mp4" && contentType != "video/quicktime" {
		return util.ValidationErrorResponse(c, "Only MP4 and MOV video formats are allowed")
	}

	if err := h.service.UploadVerificationVideo(c.Context(), userID, file); err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, nil, "Verification video uploaded successfully")
}

func (h *Handler) UploadIDPhoto(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	file, err := c.FormFile("id_photo")
	if err != nil {
		return util.ValidationErrorResponse(c, "ID photo is required")
	}

	// Validate file size (max 5MB)
	if file.Size > 5*1024*1024 {
		return util.ValidationErrorResponse(c, "Image file too large (max 5MB)")
	}

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" {
		return util.ValidationErrorResponse(c, "Only JPEG and PNG image formats are allowed")
	}

	if err := h.service.UploadIDPhoto(c.Context(), userID, file); err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, nil, "ID photo uploaded successfully")
}

func (h *Handler) GetMyProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	profile, err := h.service.GetMaidProfile(c.Context(), userID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusNotFound, "Maid profile not found")
	}

	return util.SuccessResponse(c, profile, "Maid profile retrieved successfully")
}