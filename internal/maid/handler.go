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

func (h *Handler) UpdateLocation(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req UpdateLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	if err := h.service.UpdateLocation(c.Context(), userID, &req); err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, nil, "Location updated successfully")
}

func (h *Handler) UpdateContractRate(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	var req UpdateContractRateRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	if err := h.service.UpdateContractRate(c.Context(), userID, &req); err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, nil, "Contract rate updated successfully")
}

func (h *Handler) UploadVerificationVideo(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	file, err := c.FormFile("video")
	if err != nil {
		return util.ValidationErrorResponse(c, "Video file is required")
	}

	if file.Size > 50*1024*1024 {
		return util.ValidationErrorResponse(c, "Video file too large (max 50MB)")
	}

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

	if file.Size > 5*1024*1024 {
		return util.ValidationErrorResponse(c, "Image file too large (max 5MB)")
	}

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

func (h *Handler) SearchMaids(c *fiber.Ctx) error {
	var req SearchMaidsRequest

	// Parse query parameters
	if lat := c.QueryFloat("latitude", 0); lat != 0 {
		req.Latitude = &lat
	}
	if lng := c.QueryFloat("longitude", 0); lng != 0 {
		req.Longitude = &lng
	}
	req.RadiusKM = c.QueryFloat("radius_km", 10)
	req.ServiceType = c.Query("service_type")
	
	if minRate := c.QueryInt("min_hourly_rate", 0); minRate > 0 {
		req.MinHourlyRate = &minRate
	}
	if maxRate := c.QueryInt("max_hourly_rate", 0); maxRate > 0 {
		req.MaxHourlyRate = &maxRate
	}
	
	if offersContracts := c.Query("offers_contracts"); offersContracts != "" {
		val := offersContracts == "true"
		req.OffersContracts = &val
	}
	
	req.Gender = c.Query("gender")
	
	if minRating := c.QueryFloat("min_rating", 0); minRating > 0 {
		req.MinRating = &minRating
	}
	
	req.Page = c.QueryInt("page", 1)
	req.Limit = c.QueryInt("limit", 20)

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	results, err := h.service.SearchMaids(c.Context(), &req)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, fiber.Map{
		"maids": results,
		"page":  req.Page,
		"limit": req.Limit,
		"total": len(results),
	}, "Maids retrieved successfully")
}

func (h *Handler) GetMaidByID(c *fiber.Ctx) error {
	maidIDStr := c.Params("maid_id")
	maidID, err := uuid.Parse(maidIDStr)
	if err != nil {
		return util.ValidationErrorResponse(c, "Invalid maid ID")
	}

	profile, err := h.service.GetMaidProfile(c.Context(), maidID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusNotFound, "Maid not found")
	}

	return util.SuccessResponse(c, profile, "Maid profile retrieved successfully")
}

func (h *Handler) GetWallet(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uuid.UUID)

	wallet, err := h.service.GetWallet(c.Context(), userID)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, wallet, "Wallet retrieved successfully")
}