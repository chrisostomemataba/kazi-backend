package maid

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"kazi-backend/internal/common/storage"
	"kazi-backend/internal/notification"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	repo            *Repository
	minioService    *storage.MinIOService
	notificationSvc *notification.Service
}

func NewService(repo *Repository, minioService *storage.MinIOService, notificationSvc *notification.Service) *Service {
	return &Service{
		repo:            repo,
		minioService:    minioService,
		notificationSvc: notificationSvc,
	}
}

func (s *Service) SubmitVerification(ctx context.Context, req *VerificationSubmitRequest) error {
	// Parse date of birth
	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		return fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
	}

	// Validate contract rate if offers_contracts is true
	if req.OffersContracts && req.MonthlyContractRate == nil {
		return errors.New("monthly_contract_rate is required when offers_contracts is true")
	}

	profile, err := s.repo.GetMaidProfileByUserID(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new profile
			profile = &MaidProfile{
				UserID:              req.UserID,
				Bio:                 req.Bio,
				Gender:              req.Gender,
				DateOfBirth:         &dob,
				HomeAddress:         req.HomeAddress,
				HomeLocationLat:     req.HomeLocationLat,
				HomeLocationLng:     req.HomeLocationLng,
				District:            req.District,
				Ward:                req.Ward,
				HourlyRate:          req.HourlyRate,
				OffersContracts:     req.OffersContracts,
				MonthlyContractRate: req.MonthlyContractRate,
				VerificationStatus:  "pending",
				IDNumber:            req.IDNumber,
				IDType:              req.IDType,
			}
			if err := s.repo.CreateMaidProfile(ctx, profile); err != nil {
				return fmt.Errorf("failed to create maid profile: %w", err)
			}
		} else {
			return err
		}
	} else {
		// Update existing profile
		profile.Bio = req.Bio
		profile.Gender = req.Gender
		profile.DateOfBirth = &dob
		profile.HomeAddress = req.HomeAddress
		profile.HomeLocationLat = req.HomeLocationLat
		profile.HomeLocationLng = req.HomeLocationLng
		profile.District = req.District
		profile.Ward = req.Ward
		profile.HourlyRate = req.HourlyRate
		profile.OffersContracts = req.OffersContracts
		profile.MonthlyContractRate = req.MonthlyContractRate
		profile.IDNumber = req.IDNumber
		profile.IDType = req.IDType
		profile.VerificationStatus = "pending"
		if err := s.repo.UpdateMaidProfile(ctx, profile); err != nil {
			return fmt.Errorf("failed to update maid profile: %w", err)
		}
	}

	// Delete existing services and recreate
	s.repo.DeleteMaidServices(ctx, req.UserID)
	for _, serviceType := range req.Services {
		maidService := &MaidService{
			MaidID:      req.UserID,
			ServiceType: serviceType,
		}
		s.repo.CreateMaidService(ctx, maidService)
	}

	if err := s.notificationSvc.NotifyMaidVerificationPending(ctx, req.UserID); err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateLocation(ctx context.Context, userID uuid.UUID, req *UpdateLocationRequest) error {
	profile, err := s.repo.GetMaidProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("maid profile not found: %w", err)
	}

	profile.HomeAddress = req.HomeAddress
	profile.HomeLocationLat = req.HomeLocationLat
	profile.HomeLocationLng = req.HomeLocationLng
	profile.District = req.District
	profile.Ward = req.Ward

	return s.repo.UpdateMaidProfile(ctx, profile)
}

func (s *Service) UpdateContractRate(ctx context.Context, userID uuid.UUID, req *UpdateContractRateRequest) error {
	profile, err := s.repo.GetMaidProfileByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("maid profile not found: %w", err)
	}

	if req.OffersContracts && req.MonthlyContractRate == nil {
		return errors.New("monthly_contract_rate is required when offers_contracts is true")
	}

	profile.OffersContracts = req.OffersContracts
	profile.MonthlyContractRate = req.MonthlyContractRate

	return s.repo.UpdateMaidProfile(ctx, profile)
}

func (s *Service) UploadVerificationVideo(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	fileURL, err := s.minioService.UploadVerificationVideo(ctx, userID, src, file.Size)
	if err != nil {
		return err
	}

	doc := &MaidVerificationDocument{
		MaidID:       userID,
		DocumentType: "selfie_video",
		FileURL:      fileURL,
	}
	return s.repo.CreateVerificationDocument(ctx, doc)
}

func (s *Service) UploadIDPhoto(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	fileURL, err := s.minioService.UploadIDPhoto(ctx, userID, src, file.Size, file.Header.Get("Content-Type"))
	if err != nil {
		return err
	}

	doc := &MaidVerificationDocument{
		MaidID:       userID,
		DocumentType: "id_photo",
		FileURL:      fileURL,
	}
	return s.repo.CreateVerificationDocument(ctx, doc)
}

func (s *Service) GetMaidProfile(ctx context.Context, userID uuid.UUID) (*MaidProfileResponse, error) {
	profile, err := s.repo.GetMaidProfileByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	services, err := s.repo.GetMaidServices(ctx, userID)
	if err != nil {
		return nil, err
	}

	stats, err := s.repo.GetMaidStatistics(ctx, userID)
	if err != nil {
		return nil, err
	}

	docs, err := s.repo.GetVerificationDocuments(ctx, userID)
	if err != nil {
		return nil, err
	}

	var videoURL, idPhotoURL string
	for _, doc := range docs {
		if doc.DocumentType == "selfie_video" {
			videoURL, _ = s.minioService.GetPresignedURL(ctx, doc.FileURL, 3600)
		} else if doc.DocumentType == "id_photo" {
			idPhotoURL, _ = s.minioService.GetPresignedURL(ctx, doc.FileURL, 3600)
		}
	}

	var dobStr string
	if profile.DateOfBirth != nil {
		dobStr = profile.DateOfBirth.Format("2006-01-02")
	}

	return &MaidProfileResponse{
		ID:                  profile.ID.String(),
		UserID:              profile.UserID.String(),
		Bio:                 profile.Bio,
		Gender:              profile.Gender,
		DateOfBirth:         dobStr,
		HomeAddress:         profile.HomeAddress,
		HomeLocationLat:     profile.HomeLocationLat,
		HomeLocationLng:     profile.HomeLocationLng,
		District:            profile.District,
		Ward:                profile.Ward,
		HourlyRate:          profile.HourlyRate,
		OffersContracts:     profile.OffersContracts,
		MonthlyContractRate: profile.MonthlyContractRate,
		Services:            services,
		VerificationStatus:  profile.VerificationStatus,
		IDNumber:            profile.IDNumber,
		IDType:              profile.IDType,
		RejectionReason:     profile.RejectionReason,
		VideoURL:            videoURL,
		IDPhotoURL:          idPhotoURL,
		Statistics: &MaidStatsResponse{
			AverageRating:           stats.AverageRating,
			TotalReviews:            stats.TotalReviews,
			TotalJobsCompleted:      stats.TotalJobsCompleted,
			TotalContractsCompleted: stats.TotalContractsCompleted,
		},
	}, nil
}

func (s *Service) SearchMaids(ctx context.Context, req *SearchMaidsRequest) ([]MaidSearchResult, error) {
	return s.repo.SearchMaids(ctx, req)
}