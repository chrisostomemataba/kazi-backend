package review
 
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
 
func (h *Handler) CreateReview(c *fiber.Ctx) error {
	reviewerID := c.Locals("userID").(uuid.UUID)
 
	var req CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return util.ValidationErrorResponse(c, "Invalid request body")
	}
 
	if err := util.ValidateStruct(&req); err != nil {
		return util.ValidationErrorResponse(c, err.Error())
	}
 
	resp, err := h.service.CreateReview(c.Context(), reviewerID, &req)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusBadRequest, err.Error())
	}
 
	return util.SuccessResponse(c, resp, "Review submitted successfully")
}
 
func (h *Handler) GetMaidReviews(c *fiber.Ctx) error {
	maidIDStr := c.Params("maid_id")
	maidID, err := uuid.Parse(maidIDStr)
	if err != nil {
		return util.ValidationErrorResponse(c, "Invalid maid ID")
	}
 
	limit := c.QueryInt("limit", 10)
	offset := c.QueryInt("offset", 0)
 
	reviews, err := h.service.GetMaidReviews(c.Context(), maidID, limit, offset)
	if err != nil {
		return util.ErrorResponse(c, fiber.StatusInternalServerError, err.Error())
	}
 
	return util.SuccessResponse(c, reviews, "Reviews retrieved successfully")
}