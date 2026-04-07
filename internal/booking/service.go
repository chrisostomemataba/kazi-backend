package booking

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"kazi-backend/internal/auth"
	"kazi-backend/internal/customer"
	"kazi-backend/internal/maid"
	"kazi-backend/internal/notification"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PlatformCommissionRate = 15.0
	GPSVerificationRadius  = 500
)

type Service struct {
	repo            *Repository
	authRepo        *auth.Repository
	maidRepo        *maid.Repository
	customerRepo    *customer.Repository
	notificationSvc *notification.Service
}

func NewService(
	repo *Repository,
	authRepo *auth.Repository,
	maidRepo *maid.Repository,
	customerRepo *customer.Repository,
	notificationSvc *notification.Service,
) *Service {
	return &Service{
		repo:            repo,
		authRepo:        authRepo,
		maidRepo:        maidRepo,
		customerRepo:    customerRepo,
		notificationSvc: notificationSvc,
	}
}

// ── Workflow C2: Create booking ───────────────────────────────────────────────

func (s *Service) ValidateBooking(ctx context.Context, customerID uuid.UUID, req *ValidateBookingRequest) (*ValidateBookingResponse, error) {
	response := &ValidateBookingResponse{
		CanBook:       false,
		MaidAvailable: false,
		CustomerReady: false,
		Issues:        []string{},
	}

	maidUUID, err := uuid.Parse(req.MaidID)
	if err != nil {
		response.Issues = append(response.Issues, "Invalid maid ID")
		return response, nil
	}

	maidProfile, err := s.maidRepo.GetMaidProfileByUserID(ctx, maidUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Issues = append(response.Issues, "Maid not found")
			return response, nil
		}
		return nil, fmt.Errorf("get maid profile: %w", err)
	}

	if maidProfile.VerificationStatus != "approved" {
		response.Issues = append(response.Issues, "Maid is not verified")
		return response, nil
	}

	bookingDate, err := time.Parse("2006-01-02", req.BookingDate)
	if err != nil {
		response.Issues = append(response.Issues, "Invalid date format, use YYYY-MM-DD")
		return response, nil
	}

	today := time.Now().Truncate(24 * time.Hour)
	if bookingDate.Before(today) {
		response.Issues = append(response.Issues, "Cannot book for past dates")
		return response, nil
	}

	available, err := s.repo.CheckMaidAvailability(ctx, maidUUID, bookingDate, req.StartTime, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("check availability: %w", err)
	}

	response.MaidAvailable = available
	if !available {
		response.Issues = append(response.Issues, "Msaidizi ana booking nyingine wakati huu")
	}

	user, err := s.authRepo.FindUserByID(ctx, customerID)
	if err != nil {
		response.Issues = append(response.Issues, "Customer not found")
		return response, nil
	}

	response.CustomerReady = user.IsPhoneVerified
	response.CanBook = response.MaidAvailable && response.CustomerReady

	return response, nil
}

func (s *Service) CreateBooking(ctx context.Context, customerID uuid.UUID, req *CreateBookingRequest) (*BookingResponse, error) {
	maidUUID, err := uuid.Parse(req.MaidID)
	if err != nil {
		return nil, errors.New("invalid maid ID")
	}

	validateReq := &ValidateBookingRequest{
		MaidID:      req.MaidID,
		BookingDate: req.BookingDate,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
	}
	validation, err := s.ValidateBooking(ctx, customerID, validateReq)
	if err != nil {
		return nil, err
	}

	if !validation.CanBook {
		if len(validation.Issues) > 0 {
			return nil, errors.New(validation.Issues[0])
		}
		return nil, errors.New("unable to create booking")
	}

	maidProfile, err := s.maidRepo.GetMaidProfileByUserID(ctx, maidUUID)
	if err != nil {
		return nil, fmt.Errorf("get maid profile: %w", err)
	}

	duration, err := calculateDuration(req.StartTime, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("calculate duration: %w", err)
	}
	if duration <= 0 {
		return nil, errors.New("end time must be after start time")
	}

	subtotal := int(math.Round(float64(maidProfile.HourlyRate) * duration))
	platformFee := int(math.Round(float64(subtotal) * PlatformCommissionRate / 100))
	total := subtotal + platformFee
	maidPayout := subtotal - platformFee

	bookingDate, _ := time.Parse("2006-01-02", req.BookingDate)

	booking := &Booking{
		CustomerID:          customerID,
		MaidID:              maidUUID,
		ServiceType:         req.ServiceType,
		BookingDate:         bookingDate,
		StartTime:           req.StartTime,
		EndTime:             req.EndTime,
		DurationHours:       duration,
		SpecialInstructions: req.SpecialInstructions,
		BookingStatus:       "pending_maid",
		PaymentStatus:       "unpaid",
	}

	if err := s.repo.CreateBooking(ctx, booking); err != nil {
		return nil, fmt.Errorf("create booking: %w", err)
	}

	location := &BookingLocation{
		BookingID:           booking.ID,
		CustomerAddress:     req.ServiceLocation.Address,
		CustomerLocationLat: req.ServiceLocation.Latitude,
		CustomerLocationLng: req.ServiceLocation.Longitude,
		District:            req.ServiceLocation.District,
		Ward:                req.ServiceLocation.Ward,
	}
	if err := s.repo.CreateBookingLocation(ctx, location); err != nil {
		return nil, fmt.Errorf("create location: %w", err)
	}

	pricing := &BookingPricing{
		BookingID:                booking.ID,
		HourlyRate:               maidProfile.HourlyRate,
		SubtotalAmount:           subtotal,
		PlatformCommissionRate:   PlatformCommissionRate,
		PlatformCommissionAmount: platformFee,
		TotalAmount:              total,
		MaidPayoutAmount:         maidPayout,
	}
	if err := s.repo.CreateBookingPricing(ctx, pricing); err != nil {
		return nil, fmt.Errorf("create pricing: %w", err)
	}

	s.repo.AddTimelineEvent(ctx, &BookingTimeline{
		BookingID:      booking.ID,
		EventType:      "booking_created",
		EventTimestamp: time.Now(),
		TriggeredBy:    &customerID,
		Notes:          "Customer created booking",
	})

	// Get customer name for notification
	customerUser, _ := s.authRepo.FindUserByID(ctx, customerID)
	customerName := "Mteja"
	if customerUser != nil {
		customerName = customerUser.FullName
	}

	if err := s.notificationSvc.NotifyMaidNewBooking(ctx, maidUUID, booking.ReferenceNumber, customerName); err != nil {
		log.Printf("[Booking] Warning: notification failed: %v", err)
	}

	log.Printf("[Booking] Created %s for customer %s → maid %s", booking.ReferenceNumber, customerID, maidUUID)

	return s.buildBookingResponse(ctx, booking, location, pricing, nil)
}

// ── Workflow C3: Maid accepts booking ─────────────────────────────────────────

func (s *Service) AcceptBooking(ctx context.Context, maidID uuid.UUID, bookingID string) (*BookingResponse, error) {
	bookingUUID, err := uuid.Parse(bookingID)
	if err != nil {
		return nil, errors.New("invalid booking ID")
	}

	booking, err := s.repo.GetBookingByID(ctx, bookingUUID)
	if err != nil {
		return nil, errors.New("booking not found")
	}

	if booking.MaidID != maidID {
		return nil, errors.New("unauthorized")
	}

	if booking.BookingStatus != "pending_maid" {
		return nil, fmt.Errorf("cannot accept booking in status: %s", booking.BookingStatus)
	}

	if err := s.repo.UpdateBookingStatus(ctx, bookingUUID, "maid_accepted"); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	s.repo.AddTimelineEvent(ctx, &BookingTimeline{
		BookingID:      bookingUUID,
		EventType:      "maid_accepted",
		EventTimestamp: time.Now(),
		TriggeredBy:    &maidID,
		Notes:          "Maid accepted the booking",
	})

	// Notify customer to proceed with payment
	maidUser, _ := s.authRepo.FindUserByID(ctx, maidID)
	maidName := "Msaidizi"
	if maidUser != nil {
		maidName = maidUser.FullName
	}

	s.notificationSvc.NotifyCustomerMaidAccepted(ctx, booking.CustomerID, maidName, booking.ReferenceNumber)

	log.Printf("[Booking] %s accepted by maid %s", booking.ReferenceNumber, maidID)

	booking.BookingStatus = "maid_accepted"
	location, _ := s.repo.GetBookingLocation(ctx, bookingUUID)
	pricing, _ := s.repo.GetBookingPricing(ctx, bookingUUID)
	return s.buildBookingResponse(ctx, booking, location, pricing, nil)
}

// ── Workflow C3: Maid declines booking ────────────────────────────────────────

func (s *Service) DeclineBooking(ctx context.Context, maidID uuid.UUID, bookingID string, reason string) error {
	bookingUUID, err := uuid.Parse(bookingID)
	if err != nil {
		return errors.New("invalid booking ID")
	}

	booking, err := s.repo.GetBookingByID(ctx, bookingUUID)
	if err != nil {
		return errors.New("booking not found")
	}

	if booking.MaidID != maidID {
		return errors.New("unauthorized")
	}

	if booking.BookingStatus != "pending_maid" {
		return fmt.Errorf("cannot decline booking in status: %s", booking.BookingStatus)
	}

	if err := s.repo.UpdateBookingStatus(ctx, bookingUUID, "cancelled_by_maid"); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	s.repo.AddTimelineEvent(ctx, &BookingTimeline{
		BookingID:      bookingUUID,
		EventType:      "maid_declined",
		EventTimestamp: time.Now(),
		TriggeredBy:    &maidID,
		Notes:          fmt.Sprintf("Maid declined: %s", reason),
	})

	maidUser, _ := s.authRepo.FindUserByID(ctx, maidID)
	maidName := "Msaidizi"
	if maidUser != nil {
		maidName = maidUser.FullName
	}

	s.notificationSvc.NotifyCustomerMaidDeclined(ctx, booking.CustomerID, maidName, booking.ReferenceNumber)

	log.Printf("[Booking] %s declined by maid %s", booking.ReferenceNumber, maidID)

	return nil
}

// ── Workflow D: Payment (dev mode auto-success, replace with AzamPay later) ──

func (s *Service) InitiatePayment(ctx context.Context, customerID uuid.UUID, bookingID string, req *InitiatePaymentRequest) (*PaymentResponse, error) {
	bookingUUID, err := uuid.Parse(bookingID)
	if err != nil {
		return nil, errors.New("invalid booking ID")
	}

	booking, err := s.repo.GetBookingByID(ctx, bookingUUID)
	if err != nil {
		return nil, errors.New("booking not found")
	}

	if booking.CustomerID != customerID {
		return nil, errors.New("unauthorized")
	}

	// Only allow payment after maid has accepted
	if booking.BookingStatus != "maid_accepted" {
		return nil, fmt.Errorf("cannot pay booking in status: %s — maid must accept first", booking.BookingStatus)
	}

	if booking.PaymentStatus == "paid_held_escrow" {
		return nil, errors.New("booking already paid")
	}

	pricing, err := s.repo.GetBookingPricing(ctx, bookingUUID)
	if err != nil {
		return nil, fmt.Errorf("get pricing: %w", err)
	}

	// DEV MODE: auto-success. Replace this block with AzamPay API call.
	completedAt := time.Now()
	payment := &Payment{
		BookingID:            bookingUUID,
		UserID:               customerID,
		TransactionType:      "booking_payment",
		Amount:               pricing.TotalAmount,
		Provider:             req.Provider,
		AccountNumber:        req.PhoneNumber,
		Status:               "success",
		AzampayTransactionID: fmt.Sprintf("DEV_%s_%d", bookingID[:8], time.Now().Unix()),
		AzampayReference:     fmt.Sprintf("REF_%s", bookingID[:8]),
		CompletedAt:          &completedAt,
	}

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		return nil, fmt.Errorf("create payment: %w", err)
	}

	if err := s.repo.UpdatePaymentStatus(ctx, bookingUUID, "paid_held_escrow"); err != nil {
		return nil, fmt.Errorf("update payment status: %w", err)
	}

	if err := s.repo.UpdateBookingStatus(ctx, bookingUUID, "confirmed"); err != nil {
		return nil, fmt.Errorf("update booking status: %w", err)
	}

	s.repo.AddTimelineEvent(ctx, &BookingTimeline{
		BookingID:      bookingUUID,
		EventType:      "payment_confirmed",
		EventTimestamp: time.Now(),
		TriggeredBy:    &customerID,
		Notes:          fmt.Sprintf("Payment via %s (DEV MODE)", req.Provider),
	})

	// Notify both parties
	s.notificationSvc.NotifyMaidBookingConfirmed(ctx, booking.MaidID, booking.ReferenceNumber)
	s.notificationSvc.NotifyCustomerPaymentConfirmed(ctx, customerID, booking.ReferenceNumber)

	log.Printf("[Payment] DEV: %s paid, status → confirmed", booking.ReferenceNumber)

	return &PaymentResponse{
		PaymentInitiated: true,
		TransactionID:    payment.AzampayTransactionID,
		Message:          "Malipo yamekamilika",
	}, nil
}

// ── Workflow E1: Maid marks arrival ──────────────────────────────────────────

func (s *Service) MarkArrival(ctx context.Context, maidID uuid.UUID, bookingID string, req *ArrivalRequest) (*BookingResponse, error) {
	bookingUUID, err := uuid.Parse(bookingID)
	if err != nil {
		return nil, errors.New("invalid booking ID")
	}

	booking, err := s.repo.GetBookingByID(ctx, bookingUUID)
	if err != nil {
		return nil, errors.New("booking not found")
	}

	if booking.MaidID != maidID {
		return nil, errors.New("unauthorized")
	}

	if booking.BookingStatus != "confirmed" {
		return nil, fmt.Errorf("cannot mark arrival for booking in status: %s", booking.BookingStatus)
	}

	now := time.Now()
	if err := s.repo.UpdateBookingStatus(ctx, bookingUUID, "in_progress"); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	// Store arrival GPS on booking location
	if req.Latitude != 0 && req.Longitude != 0 {
		s.repo.UpdateArrivalLocation(ctx, bookingUUID, req.Latitude, req.Longitude, now)
	}

	s.repo.AddTimelineEvent(ctx, &BookingTimeline{
		BookingID:      bookingUUID,
		EventType:      "maid_arrived",
		EventTimestamp: now,
		TriggeredBy:    &maidID,
		Notes:          "Maid marked arrival, work started",
	})

	maidUser, _ := s.authRepo.FindUserByID(ctx, maidID)
	maidName := "Msaidizi"
	if maidUser != nil {
		maidName = maidUser.FullName
	}

	s.notificationSvc.NotifyCustomerMaidArrived(ctx, booking.CustomerID, maidName)

	log.Printf("[Booking] %s → in_progress, maid arrived", booking.ReferenceNumber)

	booking.BookingStatus = "in_progress"
	location, _ := s.repo.GetBookingLocation(ctx, bookingUUID)
	pricing, _ := s.repo.GetBookingPricing(ctx, bookingUUID)
	return s.buildBookingResponse(ctx, booking, location, pricing, nil)
}

// ── Workflow E2: Maid marks work complete ─────────────────────────────────────

func (s *Service) MarkComplete(ctx context.Context, maidID uuid.UUID, bookingID string) (*BookingResponse, error) {
	bookingUUID, err := uuid.Parse(bookingID)
	if err != nil {
		return nil, errors.New("invalid booking ID")
	}

	booking, err := s.repo.GetBookingByID(ctx, bookingUUID)
	if err != nil {
		return nil, errors.New("booking not found")
	}

	if booking.MaidID != maidID {
		return nil, errors.New("unauthorized")
	}

	if booking.BookingStatus != "in_progress" {
		return nil, fmt.Errorf("cannot mark complete for booking in status: %s", booking.BookingStatus)
	}

	now := time.Now()
	if err := s.repo.UpdateServiceCompletedAt(ctx, bookingUUID, now); err != nil {
		return nil, fmt.Errorf("update completion time: %w", err)
	}

	s.repo.AddTimelineEvent(ctx, &BookingTimeline{
		BookingID:      bookingUUID,
		EventType:      "maid_marked_complete",
		EventTimestamp: now,
		TriggeredBy:    &maidID,
		Notes:          "Maid marked work as complete, awaiting customer confirmation",
	})

	maidUser, _ := s.authRepo.FindUserByID(ctx, maidID)
	maidName := "Msaidizi"
	if maidUser != nil {
		maidName = maidUser.FullName
	}

	s.notificationSvc.NotifyCustomerWorkComplete(ctx, booking.CustomerID, maidName, booking.ReferenceNumber)

	log.Printf("[Booking] %s marked complete by maid, awaiting customer confirm", booking.ReferenceNumber)

	booking.BookingStatus = "in_progress"
	location, _ := s.repo.GetBookingLocation(ctx, bookingUUID)
	pricing, _ := s.repo.GetBookingPricing(ctx, bookingUUID)
	return s.buildBookingResponse(ctx, booking, location, pricing, nil)
}

// ── Workflow E2 + F: Customer confirms completion → triggers escrow release ──

func (s *Service) ConfirmCompletion(ctx context.Context, customerID uuid.UUID, bookingID string) (*BookingResponse, error) {
	bookingUUID, err := uuid.Parse(bookingID)
	if err != nil {
		return nil, errors.New("invalid booking ID")
	}

	booking, err := s.repo.GetBookingByID(ctx, bookingUUID)
	if err != nil {
		return nil, errors.New("booking not found")
	}

	if booking.CustomerID != customerID {
		return nil, errors.New("unauthorized")
	}

	if booking.BookingStatus != "in_progress" {
		return nil, fmt.Errorf("cannot confirm booking in status: %s", booking.BookingStatus)
	}

	if err := s.repo.UpdateBookingStatus(ctx, bookingUUID, "completed"); err != nil {
		return nil, fmt.Errorf("update status: %w", err)
	}

	s.repo.AddTimelineEvent(ctx, &BookingTimeline{
		BookingID:      bookingUUID,
		EventType:      "customer_confirmed",
		EventTimestamp: time.Now(),
		TriggeredBy:    &customerID,
		Notes:          "Customer confirmed work completion",
	})

	booking.BookingStatus = "completed"
	location, _ := s.repo.GetBookingLocation(ctx, bookingUUID)
	pricing, _ := s.repo.GetBookingPricing(ctx, bookingUUID)

	// Trigger Workflow F: release payment to maid
	if err := s.releaseEscrowPayment(ctx, booking, pricing); err != nil {
		log.Printf("[Payment] Warning: escrow release failed for %s: %v", booking.ReferenceNumber, err)
	}

	log.Printf("[Booking] %s completed and confirmed by customer", booking.ReferenceNumber)

	return s.buildBookingResponse(ctx, booking, location, pricing, nil)
}

// ── Workflow F: Release escrow to maid wallet ─────────────────────────────────

func (s *Service) releaseEscrowPayment(ctx context.Context, booking *Booking, pricing *BookingPricing) error {
	if pricing == nil {
		return errors.New("pricing not found for escrow release")
	}

	// Use DB transaction — all or nothing
	return s.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := s.repo.UpdatePaymentStatusTx(txCtx, booking.ID, "released_to_maid"); err != nil {
			return fmt.Errorf("update payment status: %w", err)
		}

		if err := s.repo.CreditMaidWallet(txCtx, booking.MaidID, pricing.MaidPayoutAmount, booking.ID); err != nil {
			return fmt.Errorf("credit maid wallet: %w", err)
		}

		maidSystemID := booking.ID // use booking ID as trigger reference
		s.repo.AddTimelineEventTx(txCtx, &BookingTimeline{
			BookingID:      booking.ID,
			EventType:      "payment_released",
			EventTimestamp: time.Now(),
			TriggeredBy:    &maidSystemID,
			Notes:          fmt.Sprintf("TZS %d released to maid wallet", pricing.MaidPayoutAmount),
		})

		s.notificationSvc.NotifyMaidPaymentReleased(ctx, booking.MaidID, pricing.MaidPayoutAmount, booking.ReferenceNumber)

		log.Printf("[Payment] Escrow released: TZS %d → maid %s for booking %s",
			pricing.MaidPayoutAmount, booking.MaidID, booking.ReferenceNumber)

		return nil
	})
}

// ── Maid: get their booking requests ─────────────────────────────────────────

func (s *Service) GetMaidBookings(ctx context.Context, maidID uuid.UUID, status string, date string, page, limit int) ([]BookingResponse, error) {
	bookings, err := s.repo.GetMaidBookings(ctx, maidID, status, date, page, limit)
	if err != nil {
		return nil, fmt.Errorf("get maid bookings: %w", err)
	}

	var responses []BookingResponse
	for _, booking := range bookings {
		location, _ := s.repo.GetBookingLocation(ctx, booking.ID)
		pricing, _ := s.repo.GetBookingPricing(ctx, booking.ID)
		resp, err := s.buildBookingResponse(ctx, &booking, location, pricing, nil)
		if err != nil {
			log.Printf("[Booking] Warning: build response failed for %s: %v", booking.ID, err)
			continue
		}
		responses = append(responses, *resp)
	}

	return responses, nil
}

// ── Customer: get their bookings ──────────────────────────────────────────────

func (s *Service) GetCustomerBookings(ctx context.Context, customerID uuid.UUID, status string, page, limit int) ([]BookingResponse, error) {
	bookings, err := s.repo.GetCustomerBookings(ctx, customerID, status, page, limit)
	if err != nil {
		return nil, fmt.Errorf("get customer bookings: %w", err)
	}

	var responses []BookingResponse
	for _, booking := range bookings {
		location, _ := s.repo.GetBookingLocation(ctx, booking.ID)
		pricing, _ := s.repo.GetBookingPricing(ctx, booking.ID)
		resp, err := s.buildBookingResponse(ctx, &booking, location, pricing, nil)
		if err != nil {
			log.Printf("[Booking] Warning: build response failed for %s: %v", booking.ID, err)
			continue
		}
		responses = append(responses, *resp)
	}

	return responses, nil
}

func (s *Service) GetBookingByID(ctx context.Context, userID uuid.UUID, bookingID string) (*BookingResponse, error) {
	bookingUUID, err := uuid.Parse(bookingID)
	if err != nil {
		return nil, errors.New("invalid booking ID")
	}

	booking, err := s.repo.GetBookingByID(ctx, bookingUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("booking not found")
		}
		return nil, fmt.Errorf("get booking: %w", err)
	}

	if booking.CustomerID != userID && booking.MaidID != userID {
		return nil, errors.New("unauthorized to view this booking")
	}

	location, _ := s.repo.GetBookingLocation(ctx, bookingUUID)
	pricing, _ := s.repo.GetBookingPricing(ctx, bookingUUID)
	timeline, _ := s.repo.GetBookingTimeline(ctx, bookingUUID)

	return s.buildBookingResponse(ctx, booking, location, pricing, timeline)
}

// ── Build response ────────────────────────────────────────────────────────────

func (s *Service) buildBookingResponse(
	ctx context.Context,
	booking *Booking,
	location *BookingLocation,
	pricing *BookingPricing,
	timeline []BookingTimeline,
) (*BookingResponse, error) {
	response := &BookingResponse{
		ID:              booking.ID.String(),
		ReferenceNumber: booking.ReferenceNumber,
		Status:          booking.BookingStatus,
		PaymentStatus:   booking.PaymentStatus,
		ServiceType:     booking.ServiceType,
		BookingDate:     booking.BookingDate.Format("2006-01-02"),
		StartTime:       booking.StartTime,
		EndTime:         booking.EndTime,
		DurationHours:   booking.DurationHours,
	}

	maidUser, err := s.authRepo.FindUserByID(ctx, booking.MaidID)
	if err == nil {
		maidStats, _ := s.maidRepo.GetMaidStatistics(ctx, booking.MaidID)
		response.Maid = &BookingMaidData{
			ID:              maidUser.ID.String(),
			FullName:        maidUser.FullName,
			PhoneNumber:     maidUser.PhoneNumber,
			ProfilePhotoURL: maidUser.ProfilePhotoURL,
			AverageRating:   0.0,
		}
		if maidStats != nil {
			response.Maid.AverageRating = maidStats.AverageRating
		}
	}

	customerUser, err := s.authRepo.FindUserByID(ctx, booking.CustomerID)
	if err == nil {
		customerStats, _ := s.customerRepo.GetOrCreateStatistics(ctx, booking.CustomerID)
		response.Customer = &BookingCustomerData{
			ID:                customerUser.ID.String(),
			FullName:          customerUser.FullName,
			PhoneNumber:       customerUser.PhoneNumber,
			AverageMaidRating: 0.0,
		}
		if customerStats != nil {
			response.Customer.AverageMaidRating = customerStats.AverageMaidRating
		}
	}

	if location != nil {
		ld := &BookingLocationData{
			Address:         location.CustomerAddress,
			Latitude:        location.CustomerLocationLat,
			Longitude:       location.CustomerLocationLng,
			District:        location.District,
			Ward:            location.Ward,
			ArrivalVerified: location.ArrivalVerifiedAt != nil,
		}
		if location.ArrivalVerifiedAt != nil {
			t := location.ArrivalVerifiedAt.Format(time.RFC3339)
			ld.ArrivalVerifiedAt = &t
		}
		response.Location = ld
	}

	if pricing != nil {
		response.Pricing = &BookingPricingData{
			HourlyRate:      pricing.HourlyRate,
			Subtotal:        pricing.SubtotalAmount,
			PlatformFee:     pricing.PlatformCommissionAmount,
			PlatformFeeRate: pricing.PlatformCommissionRate,
			Total:           pricing.TotalAmount,
			MaidPayout:      pricing.MaidPayoutAmount,
		}
	}

	if timeline != nil {
		var items []BookingTimelineItem
		for _, event := range timeline {
			item := BookingTimelineItem{
				EventType:      event.EventType,
				EventTimestamp: event.EventTimestamp.Format(time.RFC3339),
				Notes:          event.Notes,
			}
			if event.TriggeredBy != nil {
				tb := event.TriggeredBy.String()
				item.TriggeredBy = &tb
			}
			items = append(items, item)
		}
		response.Timeline = items
	}

	return response, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func calculateDuration(startTime, endTime string) (float64, error) {
	start, err := time.Parse("15:04", startTime)
	if err != nil {
		return 0, err
	}
	end, err := time.Parse("15:04", endTime)
	if err != nil {
		return 0, err
	}
	return end.Sub(start).Hours(), nil
}