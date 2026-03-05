package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"kazi-backend/internal/common/storage"
	"kazi-backend/internal/common/util"
	"kazi-backend/internal/notification"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	repo            *Repository
	minioService    *storage.MinIOService
	notificationSvc *notification.Service
	jwtSecret       string
}

func NewService(repo *Repository, minioService *storage.MinIOService, notificationSvc *notification.Service, jwtSecret string) *Service {
	return &Service{
		repo:            repo,
		minioService:    minioService,
		notificationSvc: notificationSvc,
		jwtSecret:       jwtSecret,
	}
}

func (s *Service) Login(ctx context.Context, username, password string) (*AdminAuthResponse, error) {
	admin, err := s.repo.FindAdminByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	if !admin.IsActive {
		return nil, errors.New("account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := s.repo.UpdateLastLogin(ctx, admin.ID); err != nil {
		return nil, err
	}

	token, err := util.GenerateJWT(admin.ID, []string{"admin"}, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AdminAuthResponse{
		Token: token,
		Admin: &AdminData{
			ID:       admin.ID.String(),
			Username: admin.Username,
			FullName: admin.FullName,
			Role:     admin.Role,
		},
	}, nil
}

func (s *Service) CreateAdmin(ctx context.Context, username, password, fullName, role string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	admin := &AdminUser{
		Username:     username,
		PasswordHash: string(hashedPassword),
		FullName:     fullName,
		Role:         role,
		IsActive:     true,
	}

	return s.repo.CreateAdmin(ctx, admin)
}

func (s *Service) GetPendingVerifications(ctx context.Context, page, limit int) (*PendingVerificationsResponse, error) {
	offset := (page - 1) * limit
	profiles, err := s.repo.GetPendingVerifications(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	var items []VerificationItem
	for _, profile := range profiles {
		items = append(items, VerificationItem{
			MaidID:             profile.UserID.String(),
			FullName:           "", // Will be joined from users table in production
			PhoneNumber:        "", // Will be joined from users table
			Bio:                profile.Bio,
			HourlyRate:         profile.HourlyRate,
			IDNumber:           profile.IDNumber,
			IDType:             profile.IDType,
			VerificationStatus: profile.VerificationStatus,
			SubmittedAt:        profile.CreatedAt,
		})
	}

	return &PendingVerificationsResponse{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: len(items),
	}, nil
}

func (s *Service) GetVerificationDetails(ctx context.Context, maidID uuid.UUID) (*VerificationDetailsResponse, error) {
	profile, err := s.repo.GetMaidProfileByUserID(ctx, maidID)
	if err != nil {
		return nil, err
	}

	docs, err := s.repo.GetVerificationDocuments(ctx, maidID)
	if err != nil {
		return nil, err
	}

	var videoURL, idPhotoURL string
	for _, doc := range docs {
		url, _ := s.minioService.GetPresignedURL(ctx, doc.FileURL, 3600)
		if doc.DocumentType == "selfie_video" {
			videoURL = url
		} else if doc.DocumentType == "id_photo" {
			idPhotoURL = url
		}
	}

	return &VerificationDetailsResponse{
		MaidID:             profile.UserID.String(),
		Bio:                profile.Bio,
		HourlyRate:         profile.HourlyRate,
		IDNumber:           profile.IDNumber,
		IDType:             profile.IDType,
		VerificationStatus: profile.VerificationStatus,
		VideoURL:           videoURL,
		IDPhotoURL:         idPhotoURL,
		SubmittedAt:        profile.CreatedAt,
	}, nil
}

func (s *Service) ApproveVerification(ctx context.Context, adminID, maidID uuid.UUID, ipAddress string) error {
	if err := s.repo.UpdateVerificationStatus(ctx, maidID, "approved", ""); err != nil {
		return err
	}

	if err := s.notificationSvc.NotifyMaidVerificationApproved(ctx, maidID); err != nil {
		return err
	}

	changes := map[string]interface{}{
		"verification_status": "approved",
	}
	changesJSON, _ := json.Marshal(changes)

	auditLog := &AuditLog{
		ActorID:          adminID,
		ActorType:        "admin",
		ActionType:       "verified_maid",
		TargetEntityType: "maid_profile",
		TargetEntityID:   &maidID,
		Changes:          string(changesJSON),
		IPAddress:        ipAddress,
	}
	return s.repo.CreateAuditLog(ctx, auditLog)
}

func (s *Service) RejectVerification(ctx context.Context, adminID, maidID uuid.UUID, reason, ipAddress string) error {
	if reason == "" {
		return errors.New("rejection reason is required")
	}

	if err := s.repo.UpdateVerificationStatus(ctx, maidID, "rejected", reason); err != nil {
		return err
	}

	if err := s.notificationSvc.NotifyMaidVerificationRejected(ctx, maidID, reason); err != nil {
		return err
	}

	changes := map[string]interface{}{
		"verification_status": "rejected",
		"rejection_reason":    reason,
	}
	changesJSON, _ := json.Marshal(changes)

	auditLog := &AuditLog{
		ActorID:          adminID,
		ActorType:        "admin",
		ActionType:       "rejected_maid",
		TargetEntityType: "maid_profile",
		TargetEntityID:   &maidID,
		Changes:          string(changesJSON),
		IPAddress:        ipAddress,
	}
	return s.repo.CreateAuditLog(ctx, auditLog)
}