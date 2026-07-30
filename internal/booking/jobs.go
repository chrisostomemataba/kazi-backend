package booking

import (
	"context"
	"log/slog"
	"time"

	"kazi-backend/internal/notification"
)

const (
	staleCollectionRecheckAfter = 30 * time.Minute
	collectionExpiryAfter       = 24 * time.Hour
	autoConfirmAfter            = 24 * time.Hour
	backgroundJobInterval       = 1 * time.Hour
)

// StartBackgroundJobs runs the booking lifecycle safety nets on a fixed
// interval: recovering payments whose webhook never arrived, expiring
// bookings that were never paid, and auto-confirming jobs the customer
// forgot to confirm so the maid still gets paid.
func (s *Service) StartBackgroundJobs(ctx context.Context) {
	slog.Info("background jobs: started",
		"interval", backgroundJobInterval.String(),
		"collection_recheck_after", staleCollectionRecheckAfter.String(),
		"collection_expiry_after", collectionExpiryAfter.String(),
		"auto_confirm_after", autoConfirmAfter.String())

	ticker := time.NewTicker(backgroundJobInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("background jobs: stopped")
			return
		case <-ticker.C:
			s.recoverStaleCollections(ctx)
			s.autoConfirmForgottenCompletions(ctx)
		}
	}
}

// recoverStaleCollections handles payments stuck in collection_pending.
// First it asks the payment service for the real status (covers a lost
// webhook). If the payment is genuinely dead for over 24 hours the booking
// is cancelled so the maid's slot frees up.
func (s *Service) recoverStaleCollections(ctx context.Context) {
	recheckCutoff := time.Now().Add(-staleCollectionRecheckAfter)

	staleBookings, err := s.repo.FindStaleCollectionPending(ctx, recheckCutoff)
	if err != nil {
		slog.Error("background jobs: could not load stale pending payments", "error", err)
		return
	}

	if len(staleBookings) == 0 {
		return
	}

	slog.Info("background jobs: checking payments that never got a webhook",
		"count", len(staleBookings))

	for _, staleBooking := range staleBookings {
		if staleBooking.PaymentCollectionTransactionID == nil {
			continue
		}

		transactionID := *staleBooking.PaymentCollectionTransactionID

		transactionStatus, err := s.paymentClient.GetTransactionStatus(transactionID)
		if err != nil {
			slog.Warn("background jobs: payment service did not answer for a stale transaction",
				"booking_reference", staleBooking.ReferenceNumber,
				"transaction_id", transactionID,
				"error", err)
			continue
		}

		switch transactionStatus {
		case "completed", "success":
			slog.Info("background jobs: found a completed payment whose webhook was lost, recovering it",
				"booking_reference", staleBooking.ReferenceNumber,
				"transaction_id", transactionID)
			if err := s.HandlePaymentCompleted(ctx, transactionID); err != nil {
				slog.Error("background jobs: recovery of completed payment failed",
					"booking_reference", staleBooking.ReferenceNumber,
					"error", err)
			}
		case "failed", "cancelled":
			slog.Info("background jobs: stale payment actually failed, cleaning it up",
				"booking_reference", staleBooking.ReferenceNumber,
				"transaction_id", transactionID)
			if err := s.HandlePaymentFailed(ctx, transactionID); err != nil {
				slog.Error("background jobs: cleanup of failed payment failed",
					"booking_reference", staleBooking.ReferenceNumber,
					"error", err)
			}
		default:
			expiryCutoff := time.Now().Add(-collectionExpiryAfter)
			if staleBooking.UpdatedAt.Before(expiryCutoff) {
				s.expireUnpaidBooking(ctx, &staleBooking)
			}
		}
	}
}

func (s *Service) expireUnpaidBooking(ctx context.Context, expiredBooking *Booking) {
	if err := s.repo.UpdatePaymentStatus(ctx, expiredBooking.ID, "expired"); err != nil {
		slog.Error("background jobs: could not mark payment as expired",
			"booking_reference", expiredBooking.ReferenceNumber,
			"error", err)
		return
	}

	if err := s.repo.UpdateBookingStatus(ctx, expiredBooking.ID, "cancelled_by_system"); err != nil {
		slog.Error("background jobs: could not cancel expired booking",
			"booking_reference", expiredBooking.ReferenceNumber,
			"error", err)
		return
	}

	s.repo.AddTimelineEvent(ctx, &BookingTimeline{
		BookingID:      expiredBooking.ID,
		EventType:      "cancelled_by_system",
		EventTimestamp: time.Now(),
		Notes:          "Payment was never completed within 24 hours",
	})

	s.notificationSvc.CreateNotification(ctx, &notification.Notification{
		UserID:           expiredBooking.CustomerID,
		Title:            "Booking Imeghairiwa",
		Message:          "Booking " + expiredBooking.ReferenceNumber + " imeghairiwa kwa sababu malipo hayakukamilika. Tafadhali book tena.",
		NotificationType: "booking_expired",
	})

	s.notificationSvc.CreateNotification(ctx, &notification.Notification{
		UserID:           expiredBooking.MaidID,
		Title:            "Booking Imeghairiwa",
		Message:          "Booking " + expiredBooking.ReferenceNumber + " imeghairiwa kwa sababu mteja hakulipa.",
		NotificationType: "booking_expired",
	})

	slog.Info("background jobs: unpaid booking expired and cancelled, maid slot is free again",
		"booking_reference", expiredBooking.ReferenceNumber)
}

// autoConfirmForgottenCompletions completes bookings the maid finished more
// than 24 hours ago but the customer never confirmed, then releases escrow
// so the maid is not left unpaid. Bookings under an open dispute are skipped
// by the repository query, so a customer's complaint blocks auto-payout.
func (s *Service) autoConfirmForgottenCompletions(ctx context.Context) {
	cutoff := time.Now().Add(-autoConfirmAfter)

	forgottenBookings, err := s.repo.FindUnconfirmedCompleted(ctx, cutoff)
	if err != nil {
		slog.Error("background jobs: could not load unconfirmed completed bookings", "error", err)
		return
	}

	if len(forgottenBookings) == 0 {
		return
	}

	slog.Info("background jobs: auto-confirming bookings the customer forgot to confirm",
		"count", len(forgottenBookings))

	for _, forgottenBooking := range forgottenBookings {
		if err := s.repo.UpdateBookingStatus(ctx, forgottenBooking.ID, "completed"); err != nil {
			slog.Error("background jobs: could not auto-complete booking",
				"booking_reference", forgottenBooking.ReferenceNumber,
				"error", err)
			continue
		}

		s.repo.AddTimelineEvent(ctx, &BookingTimeline{
			BookingID:      forgottenBooking.ID,
			EventType:      "auto_confirmed",
			EventTimestamp: time.Now(),
			Notes:          "System auto-confirmed after 24 hours without customer response",
		})

		pricing, err := s.repo.GetBookingPricing(ctx, forgottenBooking.ID)
		if err != nil {
			slog.Error("background jobs: pricing missing for auto-confirmed booking, escrow not released",
				"booking_reference", forgottenBooking.ReferenceNumber,
				"error", err)
			continue
		}

		if err := s.releaseEscrowPayment(ctx, &forgottenBooking, pricing); err != nil {
			slog.Error("background jobs: escrow release failed after auto-confirm, payout is stuck",
				"booking_reference", forgottenBooking.ReferenceNumber,
				"error", err)
		}

		s.notificationSvc.CreateNotification(ctx, &notification.Notification{
			UserID:           forgottenBooking.CustomerID,
			Title:            "Booking Imekamilika",
			Message:          "Booking " + forgottenBooking.ReferenceNumber + " imethibitishwa kiotomatiki baada ya masaa 24. Tafadhali toa tathmini.",
			NotificationType: "booking_auto_confirmed",
		})

		slog.Info("background jobs: booking auto-confirmed and payout started",
			"booking_reference", forgottenBooking.ReferenceNumber)
	}
}
