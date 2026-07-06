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
	log.Printf("Notification → user %s: %s", notification.UserID, notification.Title)
	return nil
}

// ── Verification ──────────────────────────────────────────────────────────────

func (s *Service) NotifyMaidVerificationPending(ctx context.Context, maidID uuid.UUID) error {
	return s.CreateNotification(ctx, &Notification{
		UserID:           maidID,
		Title:            "Verification Imewasilishwa",
		Message:          "Hongera! Tumepokea nyaraka zako. Tutaangalia ndani ya masaa 24-48.",
		NotificationType: "verification_submitted",
	})
}

func (s *Service) NotifyMaidVerificationApproved(ctx context.Context, maidID uuid.UUID) error {
	return s.CreateNotification(ctx, &Notification{
		UserID:           maidID,
		Title:            "Hongera! Umeidhinishwa",
		Message:          "Account yako imethibitishwa. Unaweza kuanza kupokea bookings sasa.",
		NotificationType: "verification_approved",
	})
}

func (s *Service) NotifyMaidVerificationRejected(ctx context.Context, maidID uuid.UUID, reason string) error {
	return s.CreateNotification(ctx, &Notification{
		UserID:           maidID,
		Title:            "Verification Haikupitishwa",
		Message:          fmt.Sprintf("Samahani, verification yako haikupitishwa. Sababu: %s. Tafadhali submit upya.", reason),
		NotificationType: "verification_rejected",
	})
}

// ── Workflow C2: New booking request → notify maid ────────────────────────────

func (s *Service) NotifyMaidNewBooking(ctx context.Context, maidID uuid.UUID, bookingRef, customerName string) error {
	return s.CreateNotification(ctx, &Notification{
		UserID:           maidID,
		Title:            "Ombi Jipya la Kazi!",
		Message:          fmt.Sprintf("%s amekutumia ombi la kazi. Namba: %s. Kubali au kataa.", customerName, bookingRef),
		NotificationType: "new_booking_request",
	})
}

// ── Workflow C3: Maid accepted → notify customer to pay ───────────────────────

func (s *Service) NotifyCustomerMaidAccepted(ctx context.Context, customerID uuid.UUID, maidName, bookingRef string) error {
	return s.CreateNotification(ctx, &Notification{
		UserID:           customerID,
		Title:            "Ombi Limepokelewa!",
		Message:          fmt.Sprintf("%s amekubali booking yako %s. Tafadhali lipia sasa ili kuthibitisha.", maidName, bookingRef),
		NotificationType: "maid_accepted",
	})
}

// ── Workflow C3: Maid declined → notify customer to rebook ───────────────────

func (s *Service) NotifyCustomerMaidDeclined(ctx context.Context, customerID uuid.UUID, maidName, bookingRef string) error {
	return s.CreateNotification(ctx, &Notification{
		UserID:           customerID,
		Title:            "Ombi Limekataliwa",
		Message:          fmt.Sprintf("Samahani, %s hapatikani kwa booking %s. Tafadhali chagua msaidizi mwingine.", maidName, bookingRef),
		NotificationType: "maid_declined",
	})
}

// ── Workflow D: Payment confirmed → notify both parties ───────────────────────

func (s *Service) NotifyMaidBookingConfirmed(ctx context.Context, maidID uuid.UUID, bookingRef string) error {
	return s.CreateNotification(ctx, &Notification{
		UserID:           maidID,
		Title:            "Booking Imethibitishwa!",
		Message:          fmt.Sprintf("Mteja amelipa. Booking %s imethibitishwa. Kuwa tayari siku ya kazi.", bookingRef),
		NotificationType: "booking_confirmed",
	})
}

func (s *Service) NotifyCustomerPaymentConfirmed(ctx context.Context, customerID uuid.UUID, bookingRef string) error {
	return s.CreateNotification(ctx, &Notification{
		UserID:           customerID,
		Title:            "Malipo Yamekamilika!",
		Message:          fmt.Sprintf("Malipo ya booking %s yamepokewa. Msaidizi atakuja siku uliyochagua.", bookingRef),
		NotificationType: "payment_confirmed",
	})
}

// ── Workflow E1: Maid arrived → notify customer ───────────────────────────────

func (s *Service) NotifyCustomerMaidArrived(ctx context.Context, customerID uuid.UUID, maidName string) error {
	return s.CreateNotification(ctx, &Notification{
		UserID:           customerID,
		Title:            "Msaidizi Amefika!",
		Message:          fmt.Sprintf("%s amefika na ameanza kazi.", maidName),
		NotificationType: "maid_arrived",
	})
}

// ── Workflow E2: Maid marked complete → notify customer to confirm ────────────

func (s *Service) NotifyCustomerWorkComplete(ctx context.Context, customerID uuid.UUID, maidName, bookingRef string) error {
	return s.CreateNotification(ctx, &Notification{
		UserID:           customerID,
		Title:            "Kazi Imekamilika!",
		Message:          fmt.Sprintf("%s amesema kazi imekamilika kwa booking %s. Thibitisha ili msaidizi apokee malipo.", maidName, bookingRef),
		NotificationType: "work_complete",
	})
}

// ── Workflow F: Payment released → notify maid ────────────────────────────────

func (s *Service) NotifyMaidPaymentReleased(ctx context.Context, maidID uuid.UUID, amount int, bookingRef string) error {
	return s.CreateNotification(ctx, &Notification{
		UserID:           maidID,
		Title:            "Malipo Yametumwa!",
		Message:          fmt.Sprintf("Hongera! Umepokea TZS %s kwa booking %s. Angalia wallet yako.", formatAmount(amount), bookingRef),
		NotificationType: "payment_released",
	})
}

func (s *Service) NotifyCustomerPaymentFailed(ctx context.Context, customerID uuid.UUID, bookingRef string) error {
	return s.CreateNotification(ctx, &Notification{
		UserID:           customerID,
		Title:            "Malipo Hayajafanikiwa",
		Message:          fmt.Sprintf("Samahani, malipo ya booking %s hayajafanikiwa. Tafadhali jaribu tena.", bookingRef),
		NotificationType: "payment_failed",
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

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

func formatAmount(amount int) string {
	if amount >= 1000 {
		return fmt.Sprintf("%d,%03d", amount/1000, amount%1000)
	}
	return fmt.Sprintf("%d", amount)
}
