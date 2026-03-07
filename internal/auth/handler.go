package auth

import (
	"kazi-backend/internal/common/util"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RequestOTP(c *fiber.Ctx) error {
	var req RequestOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	if err := h.service.RequestOTP(req.PhoneNumber); err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, nil, "OTP sent successfully to your phone")
}

func (h *Handler) VerifyOTP(c *fiber.Ctx) error {
	var req VerifyOTPRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	isNewUser, err := h.service.VerifyOTP(req.PhoneNumber, req.Code)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	if isNewUser {
		// New user - needs to complete profile
		return util.SuccessResponse(c, fiber.Map{
			"is_new_user":  true,
			"phone_number": util.FormatPhoneNumber(req.PhoneNumber),
		}, "OTP verified. Please complete your profile")
	}

	// Existing user - log them in
	authResp, err := h.service.Login(req.PhoneNumber)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}

	return util.SuccessResponse(c, authResp, "Login successful")
}

func (h *Handler) CompleteProfile(c *fiber.Ctx) error {
	var req struct {
		PhoneNumber string   `json:"phone_number" validate:"required,len=12"`
		FullName    string   `json:"full_name" validate:"required,min=2,max=100"`
		Roles       []string `json:"roles" validate:"required,min=1,dive,oneof=customer maid"`
	}

	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	authResp, err := h.service.CompleteProfile(req.PhoneNumber, &CompleteProfileRequest{
		FullName: req.FullName,
		Roles:    req.Roles,
	})
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, authResp, "Profile completed successfully")
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}

	authResp, err := h.service.LoginWithOTP(req.PhoneNumber, req.OTPCode)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}

	return util.SuccessResponse(c, authResp, "Login successful")
}