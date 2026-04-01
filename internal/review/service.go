package review
 
import (
	"context"
	"errors"
	"fmt"
	"time"
 
	"kazi-backend/internal/auth"
	"kazi-backend/internal/booking"
 
	"github.com/google/uuid"
)
 
type Service struct {
	repo        *Repository
	authRepo    *auth.Repository
	bookingRepo *booking.Repository
}
 
func NewService(repo *Repository, authRepo *auth.Repository, bookingRepo *booking.Repository) *Service {
	return &Service{
		repo:        repo,
		authRepo:    authRepo,
		bookingRepo: bookingRepo,
	}
}
 
func (s *Service) CreateReview(ctx context.Context, reviewerID uuid.UUID, req *CreateReviewRequest) (*ReviewResponse, error) {
	bookingUUID, err := uuid.Parse(req.BookingID)
	if err != nil {
		return nil, errors.New("invalid booking ID")
	}
 
	b, err := s.bookingRepo.GetBookingByID(ctx, bookingUUID)
	if err != nil {
		return nil, errors.New("booking not found")
	}
 
	if b.CustomerID != reviewerID {
		return nil, errors.New("only the customer can review this booking")
	}
 
	if b.BookingStatus != "completed" {
		return nil, errors.New("can only review completed bookings")
	}
 
	exists, err := s.repo.ExistsByBookingID(ctx, bookingUUID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("booking already reviewed")
	}
 
	review := &Review{
		BookingID:  bookingUUID,
		ReviewerID: reviewerID,
		RevieweeID: b.MaidID,
		Rating:     req.Rating,
		Comment:    req.Comment,
		Tags:       EncodeTags(req.Tags),
	}
 
	if err := s.repo.Create(ctx, review); err != nil {
		return nil, fmt.Errorf("failed to create review: %w", err)
	}
 
	reviewer, err := s.authRepo.FindUserByID(ctx, reviewerID)
	if err != nil {
		return nil, err
	}
 
	return &ReviewResponse{
		ID:               review.ID.String(),
		BookingID:        review.BookingID.String(),
		ReviewerID:       review.ReviewerID.String(),
		ReviewerName:     reviewer.FullName,
		ReviewerPhotoURL: reviewer.ProfilePhotoURL,
		Rating:           review.Rating,
		Comment:          review.Comment,
		Tags:             DecodeTags(review.Tags),
		CreatedAt:        review.CreatedAt.Format(time.RFC3339),
	}, nil
}
 
func (s *Service) GetMaidReviews(ctx context.Context, maidID uuid.UUID, limit, offset int) ([]ReviewResponse, error) {
	reviews, err := s.repo.GetByMaidID(ctx, maidID, limit, offset)
	if err != nil {
		return nil, err
	}
 
	var responses []ReviewResponse
	for _, r := range reviews {
		reviewer, err := s.authRepo.FindUserByID(ctx, r.ReviewerID)
		if err != nil {
			continue
		}
		responses = append(responses, ReviewResponse{
			ID:               r.ID.String(),
			BookingID:        r.BookingID.String(),
			ReviewerID:       r.ReviewerID.String(),
			ReviewerName:     reviewer.FullName,
			ReviewerPhotoURL: reviewer.ProfilePhotoURL,
			Rating:           r.Rating,
			Comment:          r.Comment,
			Tags:             DecodeTags(r.Tags),
			CreatedAt:        r.CreatedAt.Format(time.RFC3339),
		})
	}
 
	return responses, nil
}