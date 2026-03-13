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
	PlatformCommissionRate = 15.0 // 15%
	GPSVerificationRadius  = 500  // 500 meters
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

func (s *Service) ValidateBooking(ctx context.Context, customerID uuid.UUID, req *ValidateBookingRequest) (*ValidateBookingResponse, error) {
	log.Printf("[Booking] Validating booking for customer %s with maid %s", customerID, req.MaidID)

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

	// Check maid exists and is verified
	maidProfile, err := s.maidRepo.GetMaidProfileByUserID(ctx, maidUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Issues = append(response.Issues, "Maid not found")
			return response, nil
		}
		return nil, fmt.Errorf("failed to get maid profile: %w", err)
	}

	if maidProfile.VerificationStatus != "approved" {
		response.Issues = append(response.Issues, "Maid is not verified")
		return response, nil
	}

	// Parse booking date
	bookingDate, err := time.Parse("2006-01-02", req.BookingDate)
	if err != nil {
		response.Issues = append(response.Issues, "Invalid date format, use YYYY-MM-DD")
		return response, nil
	}

	// Check if date is in the past
	today := time.Now().Truncate(24 * time.Hour)
	if bookingDate.Before(today) {
		response.Issues = append(response.Issues, "Cannot book for past dates")
		return response, nil
	}

	// Check maid availability (no conflicting bookings)
	available, err := s.repo.CheckMaidAvailability(ctx, maidUUID, bookingDate, req.StartTime, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to check availability: %w", err)
	}

	response.MaidAvailable = available
	if !available {
		response.Issues = append(response.Issues, "Maid is not available at this time")
	}

	// Customer validation - ensure they have verified phone
	user, err := s.authRepo.FindUserByID(ctx, customerID)
	if err != nil {
		response.Issues = append(response.Issues, "Customer not found")
		return response, nil
	}

	if !user.IsPhoneVerified {
		response.Issues = append(response.Issues, "Please verify your phone number")
		return response, nil
	}

	response.CustomerReady = user.IsPhoneVerified
	response.CanBook = response.MaidAvailable && response.CustomerReady

	if response.CanBook {
		log.Printf("[Booking] Validation passed for customer %s", customerID)
	} else {
		log.Printf("[Booking] Validation failed: %v", response.Issues)
	}

	return response, nil
}

func (s *Service) CreateBooking(ctx context.Context, customerID uuid.UUID, req *CreateBookingRequest) (*BookingResponse, error) {
	log.Printf("[Booking] Creating booking for customer %s with maid %s", customerID, req.MaidID)

	maidUUID, err := uuid.Parse(req.MaidID)
	if err != nil {
		return nil, errors.New("invalid maid ID")
	}

	// Validate booking first
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
		return nil, fmt.Errorf("booking validation failed: %v", validation.Issues)
	}

	// Get maid profile for pricing
	maidProfile, err := s.maidRepo.GetMaidProfileByUserID(ctx, maidUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get maid profile: %w", err)
	}

	// Calculate duration
	duration, err := calculateDuration(req.StartTime, req.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate duration: %w", err)
	}

	if duration <= 0 {
		return nil, errors.New("end time must be after start time")
	}

	// Calculate pricing
	subtotal := int(math.Round(float64(maidProfile.HourlyRate) * duration))
	platformFee := int(math.Round(float64(subtotal) * PlatformCommissionRate / 100))
	total := subtotal + platformFee
	maidPayout := subtotal

	// Parse booking date
	bookingDate, _ := time.Parse("2006-01-02", req.BookingDate)

	// Create booking
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
		log.Printf("[Booking] Failed to create booking: %v", err)
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}

	log.Printf("[Booking] Created booking %s with reference %s", booking.ID, booking.ReferenceNumber)

	// Create booking location
	location := &BookingLocation{
		BookingID:           booking.ID,
		CustomerAddress:     req.ServiceLocation.Address,
		CustomerLocationLat: req.ServiceLocation.Latitude,
		CustomerLocationLng: req.ServiceLocation.Longitude,
		District:            req.ServiceLocation.District,
		Ward:                req.ServiceLocation.Ward,
	}

	if err := s.repo.CreateBookingLocation(ctx, location); err != nil {
		log.Printf("[Booking] Failed to create location: %v", err)
		return nil, fmt.Errorf("failed to create booking location: %w", err)
	}

	// Create booking pricing
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
		log.Printf("[Booking] Failed to create pricing: %v", err)
		return nil, fmt.Errorf("failed to create booking pricing: %w", err)
	}

	// Add timeline event
	timelineEvent := &BookingTimeline{
		BookingID:      booking.ID,
		EventType:      "booking_created",
		EventTimestamp: time.Now(),
		TriggeredBy:    &customerID,
		Notes:          "Customer created booking",
	}

	if err := s.repo.AddTimelineEvent(ctx, timelineEvent); err != nil {
		log.Printf("[Booking] Warning: Failed to add timeline event: %v", err)
	}

	// Send notification to maid
	if err := s.notificationSvc.NotifyMaidNewBooking(ctx, maidUUID, booking.ReferenceNumber); err != nil {
		log.Printf("[Booking] Warning: Failed to send notification to maid: %v", err)
	}

	log.Printf("[Booking] Booking %s created successfully", booking.ReferenceNumber)

	return s.buildBookingResponse(ctx, booking, location, pricing, nil)
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
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}

	// Verify user is part of this booking
	if booking.CustomerID != userID && booking.MaidID != userID {
		return nil, errors.New("unauthorized to view this booking")
	}

	location, err := s.repo.GetBookingLocation(ctx, bookingUUID)
	if err != nil {
		log.Printf("[Booking] Warning: Failed to get location: %v", err)
		location = nil
	}

	pricing, err := s.repo.GetBookingPricing(ctx, bookingUUID)
	if err != nil {
		log.Printf("[Booking] Warning: Failed to get pricing: %v", err)
		pricing = nil
	}

	timeline, err := s.repo.GetBookingTimeline(ctx, bookingUUID)
	if err != nil {
		log.Printf("[Booking] Warning: Failed to get timeline: %v", err)
		timeline = nil
	}

	return s.buildBookingResponse(ctx, booking, location, pricing, timeline)
}

func (s *Service) GetCustomerBookings(ctx context.Context, customerID uuid.UUID, status string, page, limit int) ([]BookingResponse, error) {
	bookings, err := s.repo.GetCustomerBookings(ctx, customerID, status, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get bookings: %w", err)
	}

	var responses []BookingResponse
	for _, booking := range bookings {
		location, _ := s.repo.GetBookingLocation(ctx, booking.ID)
		pricing, _ := s.repo.GetBookingPricing(ctx, booking.ID)

		resp, err := s.buildBookingResponse(ctx, &booking, location, pricing, nil)
		if err != nil {
			log.Printf("[Booking] Warning: Failed to build response for booking %s: %v", booking.ID, err)
			continue
		}
		responses = append(responses, *resp)
	}

	return responses, nil
}

func (s *Service) InitiatePayment(ctx context.Context, customerID uuid.UUID, bookingID string, req *InitiatePaymentRequest) (*PaymentResponse, error) {
	log.Printf("[Payment] Initiating payment for booking %s by customer %s", bookingID, customerID)

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

	if booking.PaymentStatus == "paid" {
		return nil, errors.New("booking already paid")
	}

	pricing, err := s.repo.GetBookingPricing(ctx, bookingUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing: %w", err)
	}

	// DEV MODE: Simulate successful payment
	payment := &Payment{
		BookingID:            bookingUUID,
		UserID:               customerID,
		TransactionType:      "booking_payment",
		Amount:               pricing.TotalAmount,
		Provider:             req.Provider,
		AccountNumber:        req.PhoneNumber,
		Status:               "success", // DEV: Auto-success
		AzampayTransactionID: fmt.Sprintf("DEV_%s_%d", bookingID, time.Now().Unix()),
		AzampayReference:     fmt.Sprintf("REF_%s", bookingID),
	}

	completedAt := time.Now()
	payment.CompletedAt = &completedAt

	if err := s.repo.CreatePayment(ctx, payment); err != nil {
		log.Printf("[Payment] Failed to create payment record: %v", err)
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Update booking payment status
	if err := s.repo.UpdatePaymentStatus(ctx, bookingUUID, "paid"); err != nil {
		log.Printf("[Payment] Failed to update payment status: %v", err)
		return nil, fmt.Errorf("failed to update payment status: %w", err)
	}

	// Update booking status
	if err := s.repo.UpdateBookingStatus(ctx, bookingUUID, "confirmed"); err != nil {
		log.Printf("[Payment] Failed to update booking status: %v", err)
	}

	// Add timeline event
	timelineEvent := &BookingTimeline{
		BookingID:      bookingUUID,
		EventType:      "payment_received",
		EventTimestamp: time.Now(),
		TriggeredBy:    &customerID,
		Notes:          fmt.Sprintf("Payment via %s", req.Provider),
	}
	s.repo.AddTimelineEvent(ctx, timelineEvent)

	// Notify maid
	s.notificationSvc.NotifyMaidBookingConfirmed(ctx, booking.MaidID, booking.ReferenceNumber)

	log.Printf("[Payment] Payment successful for booking %s (DEV MODE)", booking.ReferenceNumber)

	return &PaymentResponse{
		PaymentInitiated: true,
		TransactionID:    payment.AzampayTransactionID,
		Message:          "Payment successful (DEV MODE)",
	}, nil
}

// Helper functions

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

	// Get maid info
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

	// Get customer info
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

	// Add location
	if location != nil {
		locationData := &BookingLocationData{
			Address:         location.CustomerAddress,
			Latitude:        location.CustomerLocationLat,
			Longitude:       location.CustomerLocationLng,
			District:        location.District,
			Ward:            location.Ward,
			ArrivalVerified: location.ArrivalVerifiedAt != nil,
		}
		if location.ArrivalVerifiedAt != nil {
			arrivalTime := location.ArrivalVerifiedAt.Format(time.RFC3339)
			locationData.ArrivalVerifiedAt = &arrivalTime
		}
		response.Location = locationData
	}

	// Add pricing
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

	// Add timeline
	if timeline != nil {
		var timelineItems []BookingTimelineItem
		for _, event := range timeline {
			item := BookingTimelineItem{
				EventType:      event.EventType,
				EventTimestamp: event.EventTimestamp.Format(time.RFC3339),
				Notes:          event.Notes,
			}
			if event.TriggeredBy != nil {
				triggeredBy := event.TriggeredBy.String()
				item.TriggeredBy = &triggeredBy
			}
			timelineItems = append(timelineItems, item)
		}
		response.Timeline = timelineItems
	}

	return response, nil
}

func calculateDuration(startTime, endTime string) (float64, error) {
	start, err := time.Parse("15:04", startTime)
	if err != nil {
		return 0, err
	}

	end, err := time.Parse("15:04", endTime)
	if err != nil {
		return 0, err
	}

	duration := end.Sub(start)
	return duration.Hours(), nil
}