package util

import "github.com/gofiber/fiber/v2"

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func SuccessResponse(c *fiber.Ctx, data interface{}, message string) error {
	return c.JSON(APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func ErrorResponse(c *fiber.Ctx, statusCode int, errMsg string) error {
	return c.Status(statusCode).JSON(APIResponse{
		Success: false,
		Error:   errMsg,
	})
}

func ValidationErrorResponse(c *fiber.Ctx, errMsg string) error {
	return ErrorResponse(c, fiber.StatusBadRequest, errMsg)
}