package maid

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"

	"kazi-backend/internal/common/storage"
	"kazi-backend/internal/notification"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	repo              *Repository
	minioService      *storage.MinIOService
	notificationSvc   *notification.Service
}

func NewService(repo *Repository, minioService *storage.MinIOService, notificationSvc *notification.Service) *Service {
	return &Service{
		repo:            repo,
		minioService:    minioService,
		notificationSvc: notificationSvc,
	}
}

func (s *Service) SubmitVerification(ctx context.Context, req *VerificationSubmitRequest) error {
	profile, err := s.repo.GetMaidProfileByUserID(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			profile = &MaidProfile{
				UserID:             req.UserID,
				Bio:                req.Bio,
				HourlyRate:         req.HourlyRate,
				VerificationStatus: "pending",
				IDNumber:           req.IDNumber,
				IDType:             req.IDType,
			}
			if err := s.repo.CreateMaidProfile(ctx, profile); err != nil {
				return fmt.Errorf("failed to create maid profile: %w", err)
			}
		} else {
			return err
		}
	} else {
		profile.Bio = req.Bio
		profile.HourlyRate = req.HourlyRate
		profile.IDNumber = req.IDNumber
		profile.IDType = req.IDType
		profile.VerificationStatus = "pending"
		if err := s.repo.UpdateMaidProfile(ctx, profile); err != nil {
			return fmt.Errorf("failed to update maid profile: %w", err)
		}
	}

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

	return &MaidProfileResponse{
		ID:                 profile.ID.String(),
		UserID:             profile.UserID.String(),
		Bio:                profile.Bio,
		HourlyRate:         profile.HourlyRate,
		VerificationStatus: profile.VerificationStatus,
		IDNumber:           profile.IDNumber,
		IDType:             profile.IDType,
		RejectionReason:    profile.RejectionReason,
		VideoURL:           videoURL,
		IDPhotoURL:         idPhotoURL,
	}, nil
}