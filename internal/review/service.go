package review
 
import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"kazi-backend/internal/auth"
	"kazi-backend/internal/booking"
	"kazi-backend/internal/common/storage"

	"github.com/google/uuid"
)

type Service struct {
	repo         *Repository
	authRepo     *auth.Repository
	bookingRepo  *booking.Repository
	minioService *storage.MinIOService
}

func NewService(repo *Repository, authRepo *auth.Repository, bookingRepo *booking.Repository, minioService *storage.MinIOService) *Service {
	return &Service{
		repo:         repo,
		authRepo:     authRepo,
		bookingRepo:  bookingRepo,
		minioService: minioService,
	}
}

func (s *Service) UploadReviewPhoto(ctx context.Context, reviewerID uuid.UUID, file *multipart.FileHeader) (string, error) {
	openedFile, openError := file.Open()
	if openError != nil {
		return "", fmt.Errorf("failed to open photo: %w", openError)
	}
	defer openedFile.Close()

	contentType := file.Header.Get("Content-Type")
	objectName, uploadError := s.minioService.UploadImage(ctx, "reviews/photos", reviewerID, openedFile, file.Size, contentType)
	if uploadError != nil {
		return "", fmt.Errorf("failed to upload review photo: %w", uploadError)
	}

	return objectName, nil
}

func (s *Service) presignPhotoURLs(ctx context.Context, raw string) []string {
	objectNames := DecodeTags(raw)
	presignedURLs := make([]string, 0, len(objectNames))
	for _, objectName := range objectNames {
		presignedURL, presignError := s.minioService.GetPresignedURL(ctx, objectName, time.Hour)
		if presignError != nil {
			continue
		}
		presignedURLs = append(presignedURLs, presignedURL)
	}
	return presignedURLs
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
		PhotoURLs:  EncodeTags(req.PhotoURLs),
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
		PhotoURLs:        s.presignPhotoURLs(ctx, review.PhotoURLs),
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
			PhotoURLs:        s.presignPhotoURLs(ctx, r.PhotoURLs),
			CreatedAt:        r.CreatedAt.Format(time.RFC3339),
		})
	}
 
	return responses, nil
}