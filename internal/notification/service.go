package notification

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateNotification(ctx context.Context, notification *Notification) error {
	if err := s.db.WithContext(ctx).Create(notification).Error; err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	log.Printf("Notification created for user %s: %s", notification.UserID, notification.Title)
	return nil
}

func (s *Service) NotifyMaidVerificationPending(ctx context.Context, maidID uuid.UUID) error {
	notification := &Notification{
		UserID:           maidID,
		Title:            "Verification Pending",
		Message:          "Hongera! Verification documents submitted. Tutaangalia ndani ya masaa 24-48.",
		NotificationType: "verification_submitted",
	}
	return s.CreateNotification(ctx, notification)
}

func (s *Service) NotifyMaidVerificationApproved(ctx context.Context, maidID uuid.UUID) error {
	notification := &Notification{
		UserID:           maidID,
		Title:            "Verification Approved!",
		Message:          "Hongera! Account yako imethibitishwa. Unaweza kuanza kupokea bookings sasa.",
		NotificationType: "verification_approved",
	}
	return s.CreateNotification(ctx, notification)
}

func (s *Service) NotifyMaidVerificationRejected(ctx context.Context, maidID uuid.UUID, reason string) error {
	notification := &Notification{
		UserID:           maidID,
		Title:            "Verification Rejected",
		Message:          fmt.Sprintf("Samahani, verification yako haijapitishwa. Sababu: %s. Tafadhali submit upya.", reason),
		NotificationType: "verification_rejected",
	}
	return s.CreateNotification(ctx, notification)
}

func (s *Service) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Notification, error) {
	var notifications []Notification
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	return notifications, err
}

func (s *Service) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	return s.db.WithContext(ctx).
		Model(&Notification{}).
		Where("id = ?", notificationID).
		Update("is_read", true).Error
}