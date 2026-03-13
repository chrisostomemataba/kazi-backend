package customer

import (
	"context"
	"fmt"

	"kazi-backend/internal/auth"

	"github.com/google/uuid"
)

type Service struct {
	repo     *Repository
	authRepo *auth.Repository
}

func NewService(repo *Repository, authRepo *auth.Repository) *Service {
	return &Service{
		repo:     repo,
		authRepo: authRepo,
	}
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*CustomerProfileResponse, error) {
	// Get user info
	user, err := s.authRepo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Get or create customer profile
	profile, err := s.repo.GetOrCreateProfile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer profile: %w", err)
	}

	// Get saved locations
	locations, err := s.repo.GetCustomerLocations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get locations: %w", err)
	}

	var locationResponses []CustomerLocationResponse
	for _, loc := range locations {
		locationResponses = append(locationResponses, CustomerLocationResponse{
			ID:        loc.ID.String(),
			Label:     loc.Label,
			Address:   loc.Address,
			Latitude:  loc.Latitude,
			Longitude: loc.Longitude,
			District:  loc.District,
			Ward:      loc.Ward,
			IsDefault: loc.IsDefault,
		})
	}

	// Get statistics
	stats, err := s.repo.GetOrCreateStatistics(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return &CustomerProfileResponse{
		ID:                profile.ID.String(),
		UserID:            user.ID.String(),
		FullName:          user.FullName,
		PhoneNumber:       user.PhoneNumber,
		PreferredLanguage: profile.PreferredLanguage,
		SavedLocations:    locationResponses,
		Statistics: &CustomerStatsResponse{
			AverageMaidRating: stats.AverageMaidRating,
			TotalBookings:     stats.TotalBookings,
			CompletedBookings: stats.CompletedBookings,
			PaymentOnTimeRate: stats.PaymentOnTimeRate,
		},
	}, nil
}

func (s *Service) AddLocation(ctx context.Context, userID uuid.UUID, req *AddLocationRequest) error {
	location := &CustomerLocation{
		CustomerID: userID,
		Label:      req.Label,
		Address:    req.Address,
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		District:   req.District,
		Ward:       req.Ward,
		IsDefault:  req.IsDefault,
	}

	if err := s.repo.AddLocation(ctx, location); err != nil {
		return fmt.Errorf("failed to add location: %w", err)
	}

	return nil
}

func (s *Service) GetLocations(ctx context.Context, userID uuid.UUID) ([]CustomerLocationResponse, error) {
	locations, err := s.repo.GetCustomerLocations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get locations: %w", err)
	}

	var responses []CustomerLocationResponse
	for _, loc := range locations {
		responses = append(responses, CustomerLocationResponse{
			ID:        loc.ID.String(),
			Label:     loc.Label,
			Address:   loc.Address,
			Latitude:  loc.Latitude,
			Longitude: loc.Longitude,
			District:  loc.District,
			Ward:      loc.Ward,
			IsDefault: loc.IsDefault,
		})
	}

	return responses, nil
}

func (s *Service) DeleteLocation(ctx context.Context, userID uuid.UUID, locationID string) error {
	locUUID, err := uuid.Parse(locationID)
	if err != nil {
		return fmt.Errorf("invalid location ID: %w", err)
	}

	if err := s.repo.DeleteLocation(ctx, locUUID, userID); err != nil {
		return fmt.Errorf("failed to delete location: %w", err)
	}

	return nil
}