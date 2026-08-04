package booking

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"mime/multipart"
	"time"

	"strings"

	"kazi-backend/internal/auth"
	"kazi-backend/internal/common/storage"
	"kazi-backend/internal/customer"
	"kazi-backend/internal/maid"
	"kazi-backend/internal/notification"
	"kazi-backend/internal/payment"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrBookingNotFound      = errors.New("booking not found")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrLocationNotAvailable = errors.New("location not available yet")
	ErrInvalidLocation      = errors.New("invalid coordinates")
	ErrInvalidState         = errors.New("booking not in trackable state")
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
	paymentClient   *payment.PaymentClient
	minioService    *storage.MinIOService
}

func NewService(
	repo *Repository,
	authRepo *auth.Repository,
	maidRepo *maid.Repository,
	customerRepo *customer.Repository,
	notificationSvc *notification.Service,
	paymentClient *payment.PaymentClient,
	minioService *storage.MinIOService,
) *Service {
	return &Service{
		repo:            repo,
		authRepo:        authRepo,
		maidRepo:        maidRepo,
		customerRepo:    customerRepo,
		notificationSvc: notificationSvc,
		paymentClient:   paymentClient,
		minioService:    minioService,
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
		slog.Warn("booking: maid was not notified about the new request",
			"booking_reference", booking.ReferenceNumber,
			"error", err)
	}

	slog.Info("booking: new booking created, waiting for maid to accept",
		"booking_reference", booking.ReferenceNumber,
		"customer_id", customerID.String(),
		"maid_id", maidUUID.String(),
		"service_type", booking.ServiceType,
		"total_tzs", total)

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

	slotFree, err := s.repo.CheckMaidSlotFreeForAccept(ctx, maidID, booking.BookingDate, booking.StartTime, booking.EndTime, bookingUUID)
	if err != nil {
		return nil, fmt.Errorf("check slot availability: %w", err)
	}
	if !slotFree {
		slog.Warn("booking: maid tried to accept an overlapping booking",
			"booking_reference", booking.ReferenceNumber,
			"maid_id", maidID.String())
		return nil, errors.New("umeshakubali kazi nyingine kwa muda huu — you already have another booking at this time")
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

	slog.Info("booking: maid accepted, customer can now pay",
		"booking_reference", booking.ReferenceNumber,
		"maid_id", maidID.String())

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

	slog.Info("booking: maid declined the request",
		"booking_reference", booking.ReferenceNumber,
		"maid_id", maidID.String(),
		"reason", reason)

	return nil
}

// ── Workflow D: Payment via payment microservice (Snippe) ────────────────────

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

	if booking.PaymentStatus == "collection_pending" {
		return nil, errors.New("a payment is already in progress for this booking — please wait for it to resolve before retrying")
	}

	pricing, err := s.repo.GetBookingPricing(ctx, bookingUUID)
	if err != nil {
		return nil, fmt.Errorf("get pricing: %w", err)
	}

	customerUser, err := s.authRepo.FindUserByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}

	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "session"
	}

	slog.Info("payment: customer is initiating payment",
		"booking_reference", booking.ReferenceNumber,
		"payment_method", paymentMethod,
		"amount_tzs", pricing.TotalAmount)

	if paymentMethod == "session" {
		return s.initiateSessionPayment(ctx, booking, pricing, customerUser, req)
	}

	if paymentMethod == "card" {
		return s.initiateCardPayment(ctx, booking, pricing, customerUser.PhoneNumber, req)
	}

	if req.Provider == "" {
		return nil, errors.New("provider is required for mobile money payment")
	}

	firstName, lastName := splitFullName(customerUser.FullName)

	transactionID, err := s.paymentClient.CollectFromCustomer(
		req.PhoneNumber,
		customerUser.PhoneNumber,
		pricing.TotalAmount,
		firstName,
		lastName,
		placeholderCustomerEmail(customerUser.PhoneNumber),
		map[string]interface{}{"booking_id": bookingUUID.String()},
	)
	if err != nil {
		slog.Error("payment: mobile money collection could not be started",
			"booking_reference", booking.ReferenceNumber,
			"error", err)
		return nil, errors.New("payment initiation failed")
	}

	if err := s.repo.UpdateBookingPaymentCollectionTxID(ctx, bookingUUID, transactionID); err != nil {
		return nil, fmt.Errorf("store transaction id: %w", err)
	}

	if err := s.repo.UpdatePaymentStatus(ctx, bookingUUID, "collection_pending"); err != nil {
		return nil, fmt.Errorf("update payment status: %w", err)
	}

	slog.Info("payment: mobile money collection started, waiting for customer PIN and webhook",
		"booking_reference", booking.ReferenceNumber,
		"transaction_id", transactionID)

	return &PaymentResponse{
		PaymentInitiated: true,
		TransactionID:    transactionID,
		Message:          "Malipo yanaendelea",
	}, nil
}

// initiateCardPayment starts a hosted Snippe card checkout. The mobile app
// must open the returned payment_url in a WebView or browser; completion
// still arrives asynchronously via the payment.completed webhook.
func (s *Service) initiateCardPayment(
	ctx context.Context,
	booking *Booking,
	pricing *BookingPricing,
	customerPhone string,
	req *InitiatePaymentRequest,
) (*PaymentResponse, error) {
	billingAddress := req.BillingAddress
	if billingAddress == "" {
		billingAddress = "Dar es Salaam"
	}
	billingCity := req.BillingCity
	if billingCity == "" {
		billingCity = "Dar es Salaam"
	}
	billingState := req.BillingState
	if billingState == "" {
		billingState = "Dar es Salaam"
	}
	billingPostcode := req.BillingPostcode
	if billingPostcode == "" {
		billingPostcode = "00000"
	}
	billingCountry := req.BillingCountry
	if billingCountry == "" {
		billingCountry = "TZ"
	}
	redirectURL := req.RedirectURL
	if redirectURL == "" {
		redirectURL = "kazi://payment/success"
	}
	cancelURL := req.CancelURL
	if cancelURL == "" {
		cancelURL = "kazi://payment/cancel"
	}

	phoneNumber := req.PhoneNumber
	if phoneNumber == "" {
		phoneNumber = customerPhone
	}

	transactionID, paymentURL, err := s.paymentClient.CollectFromCustomerByCard(
		phoneNumber,
		pricing.TotalAmount,
		billingAddress,
		billingCity,
		billingState,
		billingPostcode,
		billingCountry,
		redirectURL,
		cancelURL,
		map[string]interface{}{"booking_id": booking.ID.String()},
	)
	if err != nil {
		slog.Error("payment: card checkout could not be created",
			"booking_reference", booking.ReferenceNumber,
			"error", err)
		return nil, errors.New("card payment initiation failed")
	}

	if err := s.repo.UpdateBookingPaymentCollectionTxID(ctx, booking.ID, transactionID); err != nil {
		return nil, fmt.Errorf("store transaction id: %w", err)
	}

	if err := s.repo.UpdatePaymentStatus(ctx, booking.ID, "collection_pending"); err != nil {
		return nil, fmt.Errorf("update payment status: %w", err)
	}

	slog.Info("payment: card checkout created, customer must complete it on the payment page",
		"booking_reference", booking.ReferenceNumber,
		"transaction_id", transactionID)

	return &PaymentResponse{
		PaymentInitiated: true,
		TransactionID:    transactionID,
		PaymentURL:       paymentURL,
		Message:          "Fungua ukurasa wa malipo kukamilisha",
	}, nil
}

func (s *Service) initiateSessionPayment(
	ctx context.Context,
	booking *Booking,
	pricing *BookingPricing,
	customerUser *auth.User,
	req *InitiatePaymentRequest,
) (*PaymentResponse, error) {
	redirectURL := req.RedirectURL
	if redirectURL == "" {
		redirectURL = "kazi://payment/success"
	}

	phoneNumber := req.PhoneNumber
	if phoneNumber == "" {
		phoneNumber = customerUser.PhoneNumber
	}

	transactionID, checkoutURL, err := s.paymentClient.CreateCheckoutSession(
		pricing.TotalAmount,
		customerUser.FullName,
		phoneNumber,
		placeholderCustomerEmail(customerUser.PhoneNumber),
		0,
		redirectURL,
		map[string]interface{}{"booking_id": booking.ID.String()},
	)
	if err != nil {
		slog.Error("payment: checkout session could not be created",
			"booking_reference", booking.ReferenceNumber,
			"error", err)
		return nil, errors.New("session payment initiation failed")
	}

	if err := s.repo.UpdateBookingPaymentCollectionTxID(ctx, booking.ID, transactionID); err != nil {
		return nil, fmt.Errorf("store transaction id: %w", err)
	}

	if err := s.repo.UpdatePaymentStatus(ctx, booking.ID, "collection_pending"); err != nil {
		return nil, fmt.Errorf("update payment status: %w", err)
	}

	slog.Info("payment: checkout session created, customer must complete it on the payment page",
		"booking_reference", booking.ReferenceNumber,
		"transaction_id", transactionID)

	return &PaymentResponse{
		PaymentInitiated: true,
		TransactionID:    transactionID,
		PaymentURL:       checkoutURL,
		Message:          "Fungua ukurasa wa malipo kukamilisha",
	}, nil
}

// ── Payment webhook events (invoked via payment.WebhookEventHandler) ─────────

func (s *Service) HandlePaymentCompleted(ctx context.Context, transactionID string) error {
	booking, err := s.repo.GetBookingByCollectionTransactionID(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("find booking by collection transaction: %w", err)
	}

	// Webhook retries must not hold escrow twice.
	if booking.PaymentStatus == "paid_held_escrow" || booking.PaymentStatus == "disbursement_pending" || booking.PaymentStatus == "released_to_maid" {
		slog.Info("payment: duplicate payment.completed webhook ignored, booking is already paid",
			"booking_reference", booking.ReferenceNumber,
			"transaction_id", transactionID)
		return nil
	}

	pricing, err := s.repo.GetBookingPricing(ctx, booking.ID)
	if err != nil {
		return fmt.Errorf("get pricing: %w", err)
	}

	if _, err := s.paymentClient.HoldEscrow(
		transactionID,
		booking.ReferenceNumber,
		pricing.TotalAmount,
		map[string]interface{}{"booking_id": booking.ID.String()},
	); err != nil {
		return fmt.Errorf("hold escrow: %w", err)
	}

	if err := s.repo.UpdatePaymentStatus(ctx, booking.ID, "paid_held_escrow"); err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}

	if err := s.repo.UpdateBookingStatus(ctx, booking.ID, "confirmed"); err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}

	s.notificationSvc.NotifyMaidBookingConfirmed(ctx, booking.MaidID, booking.ReferenceNumber)

	slog.Info("payment: customer money received and held in escrow, booking is now confirmed",
		"booking_reference", booking.ReferenceNumber,
		"amount_tzs", pricing.TotalAmount)

	return nil
}

func (s *Service) HandlePayoutCompleted(ctx context.Context, transactionID string) error {
	booking, err := s.repo.GetBookingByDisbursementTransactionID(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("find booking by disbursement transaction: %w", err)
	}

	// Webhook retries must not credit the display wallet twice.
	if booking.PaymentStatus == "released_to_maid" {
		slog.Info("payment: duplicate payout.completed webhook ignored, maid was already paid",
			"booking_reference", booking.ReferenceNumber,
			"transaction_id", transactionID)
		return nil
	}

	if err := s.repo.UpdatePaymentStatus(ctx, booking.ID, "released_to_maid"); err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}

	pricing, err := s.repo.GetBookingPricing(ctx, booking.ID)
	if err != nil {
		return fmt.Errorf("get pricing: %w", err)
	}

	if err := s.repo.CreditMaidWallet(ctx, booking.MaidID, pricing.MaidPayoutAmount, booking.ID); err != nil {
		slog.Warn("payment: maid wallet display balance was not updated — money already sent by Snippe, fix the wallet row manually",
			"booking_reference", booking.ReferenceNumber,
			"maid_id", booking.MaidID.String(),
			"amount_tzs", pricing.MaidPayoutAmount,
			"error", err)
	}

	s.notificationSvc.NotifyMaidPaymentReleased(ctx, booking.MaidID, pricing.MaidPayoutAmount, booking.ReferenceNumber)

	slog.Info("payment: payout confirmed, maid has been paid on her phone",
		"booking_reference", booking.ReferenceNumber,
		"maid_id", booking.MaidID.String(),
		"amount_tzs", pricing.MaidPayoutAmount)

	return nil
}

func (s *Service) HandlePaymentFailed(ctx context.Context, transactionID string) error {
	booking, err := s.repo.GetBookingByCollectionTransactionID(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("find booking by collection transaction: %w", err)
	}

	// A late failure webhook must never undo a payment that already succeeded.
	if booking.PaymentStatus == "paid_held_escrow" || booking.PaymentStatus == "disbursement_pending" || booking.PaymentStatus == "released_to_maid" {
		slog.Warn("payment: payment.failed webhook arrived for a booking that is already paid, ignoring it",
			"booking_reference", booking.ReferenceNumber,
			"transaction_id", transactionID)
		return nil
	}
	if booking.PaymentStatus == "failed" {
		slog.Info("payment: duplicate payment.failed webhook ignored",
			"booking_reference", booking.ReferenceNumber,
			"transaction_id", transactionID)
		return nil
	}

	if err := s.repo.UpdatePaymentStatus(ctx, booking.ID, "failed"); err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}

	if err := s.repo.UpdateBookingStatus(ctx, booking.ID, "cancelled_payment_failed"); err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}

	s.notificationSvc.NotifyCustomerPaymentFailed(ctx, booking.CustomerID, booking.ReferenceNumber)

	slog.Warn("payment: customer payment failed, booking was cancelled",
		"booking_reference", booking.ReferenceNumber,
		"customer_id", booking.CustomerID.String())

	return nil
}

func (s *Service) HandlePaymentExpired(ctx context.Context, transactionID string) error {
	booking, err := s.repo.GetBookingByCollectionTransactionID(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("find booking by collection transaction: %w", err)
	}

	if booking.PaymentStatus == "paid_held_escrow" || booking.PaymentStatus == "disbursement_pending" || booking.PaymentStatus == "released_to_maid" {
		slog.Warn("payment: payment.expired webhook arrived for a booking that is already paid, ignoring it",
			"booking_reference", booking.ReferenceNumber,
			"transaction_id", transactionID)
		return nil
	}
	if booking.PaymentStatus == "failed" {
		slog.Info("payment: duplicate payment.expired webhook ignored",
			"booking_reference", booking.ReferenceNumber,
			"transaction_id", transactionID)
		return nil
	}

	if err := s.repo.UpdatePaymentStatus(ctx, booking.ID, "failed"); err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}

	if err := s.repo.UpdateBookingStatus(ctx, booking.ID, "cancelled_payment_failed"); err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}

	s.notificationSvc.NotifyCustomerPaymentExpired(ctx, booking.CustomerID, booking.ReferenceNumber)

	slog.Warn("payment: checkout session expired before payment, booking was cancelled",
		"booking_reference", booking.ReferenceNumber,
		"customer_id", booking.CustomerID.String())

	return nil
}

func (s *Service) HandlePaymentVoided(ctx context.Context, transactionID string) error {
	booking, err := s.repo.GetBookingByCollectionTransactionID(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("find booking by collection transaction: %w", err)
	}

	if booking.PaymentStatus == "paid_held_escrow" || booking.PaymentStatus == "disbursement_pending" || booking.PaymentStatus == "released_to_maid" {
		slog.Warn("payment: payment.voided webhook arrived for a booking that is already paid, ignoring it",
			"booking_reference", booking.ReferenceNumber,
			"transaction_id", transactionID)
		return nil
	}
	if booking.PaymentStatus == "failed" {
		slog.Info("payment: duplicate payment.voided webhook ignored",
			"booking_reference", booking.ReferenceNumber,
			"transaction_id", transactionID)
		return nil
	}

	if err := s.repo.UpdatePaymentStatus(ctx, booking.ID, "failed"); err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}

	if err := s.repo.UpdateBookingStatus(ctx, booking.ID, "cancelled_payment_failed"); err != nil {
		return fmt.Errorf("update booking status: %w", err)
	}

	s.notificationSvc.NotifyCustomerPaymentVoided(ctx, booking.CustomerID, booking.ReferenceNumber)

	slog.Warn("payment: customer cancelled the checkout session, booking was cancelled",
		"booking_reference", booking.ReferenceNumber,
		"customer_id", booking.CustomerID.String())

	return nil
}

func (s *Service) HandlePayoutFailed(ctx context.Context, transactionID string) error {
	booking, err := s.repo.GetBookingByDisbursementTransactionID(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("find booking by disbursement transaction: %w", err)
	}

	if booking.PaymentStatus == "released_to_maid" {
		slog.Info("payment: duplicate payout.failed webhook ignored, maid was already paid",
			"booking_reference", booking.ReferenceNumber,
			"transaction_id", transactionID)
		return nil
	}
	if booking.PaymentStatus == "payout_failed" {
		slog.Info("payment: duplicate payout.failed webhook ignored",
			"booking_reference", booking.ReferenceNumber,
			"transaction_id", transactionID)
		return nil
	}

	if err := s.repo.UpdatePaymentStatus(ctx, booking.ID, "payout_failed"); err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}

	s.repo.AddTimelineEvent(ctx, &BookingTimeline{
		BookingID:      booking.ID,
		EventType:      "payout_failed",
		EventTimestamp: time.Now(),
		Notes:          "Snippe could not deliver the disbursement — escrow funds are not confirmed on the maid's phone, needs manual reconciliation",
	})

	slog.Error("payment: ACTION REQUIRED — payout to maid failed, escrow funds are unaccounted for",
		"booking_reference", booking.ReferenceNumber,
		"maid_id", booking.MaidID.String(),
		"transaction_id", transactionID)

	return nil
}

func (s *Service) HandlePayoutReversed(ctx context.Context, transactionID string) error {
	booking, err := s.repo.GetBookingByDisbursementTransactionID(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("find booking by disbursement transaction: %w", err)
	}

	if booking.PaymentStatus == "payout_reversed" {
		slog.Info("payment: duplicate payout.reversed webhook ignored",
			"booking_reference", booking.ReferenceNumber,
			"transaction_id", transactionID)
		return nil
	}

	if err := s.repo.UpdatePaymentStatus(ctx, booking.ID, "payout_reversed"); err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}

	s.repo.AddTimelineEvent(ctx, &BookingTimeline{
		BookingID:      booking.ID,
		EventType:      "payout_reversed",
		EventTimestamp: time.Now(),
		Notes:          "A completed payout was reversed after the fact — the maid's wallet display balance was already credited, needs manual reconciliation",
	})

	slog.Error("payment: ACTION REQUIRED — a completed payout was reversed, maid wallet display balance is now wrong",
		"booking_reference", booking.ReferenceNumber,
		"maid_id", booking.MaidID.String(),
		"transaction_id", transactionID)

	return nil
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

	// Work duration is measured from arrival, shown on the completion summary
	if err := s.repo.UpdateServiceStartedAt(ctx, bookingUUID, now); err != nil {
		slog.Warn("booking: failed to record service start time", "error", err)
	}
	booking.ServiceStartedAt = &now

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

	slog.Info("booking: maid arrived at the customer, work has started",
		"booking_reference", booking.ReferenceNumber,
		"maid_id", maidID.String())

	booking.BookingStatus = "in_progress"
	location, _ := s.repo.GetBookingLocation(ctx, bookingUUID)
	pricing, _ := s.repo.GetBookingPricing(ctx, bookingUUID)
	return s.buildBookingResponse(ctx, booking, location, pricing, nil)
}

// ── Workflow E2: Maid marks work complete ─────────────────────────────────────

func (s *Service) MarkComplete(ctx context.Context, maidID uuid.UUID, bookingID string, req *CompleteRequest) (*BookingResponse, error) {
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
	booking.ServiceCompletedAt = &now

	if req != nil {
		if err := s.repo.UpdateCompletionDetails(ctx, bookingUUID, req.Notes, req.BeforePhotoURL, req.AfterPhotoURL); err != nil {
			slog.Warn("booking: failed to save completion details", "error", err)
		} else {
			booking.CompletionNotes = req.Notes
			if req.BeforePhotoURL != "" {
				booking.BeforePhotoURL = &req.BeforePhotoURL
			}
			if req.AfterPhotoURL != "" {
				booking.AfterPhotoURL = &req.AfterPhotoURL
			}
		}
	}

	timelineNotes := "Maid marked work as complete, awaiting customer confirmation"
	if req != nil && req.Notes != "" {
		timelineNotes = req.Notes
	}

	s.repo.AddTimelineEvent(ctx, &BookingTimeline{
		BookingID:      bookingUUID,
		EventType:      "maid_marked_complete",
		EventTimestamp: now,
		TriggeredBy:    &maidID,
		Notes:          timelineNotes,
	})

	maidUser, _ := s.authRepo.FindUserByID(ctx, maidID)
	maidName := "Msaidizi"
	if maidUser != nil {
		maidName = maidUser.FullName
	}

	s.notificationSvc.NotifyCustomerWorkComplete(ctx, booking.CustomerID, maidName, booking.ReferenceNumber)

	slog.Info("booking: maid finished the work, waiting for the customer to confirm",
		"booking_reference", booking.ReferenceNumber,
		"maid_id", maidID.String())

	booking.BookingStatus = "in_progress"
	location, _ := s.repo.GetBookingLocation(ctx, bookingUUID)
	pricing, _ := s.repo.GetBookingPricing(ctx, bookingUUID)
	return s.buildBookingResponse(ctx, booking, location, pricing, nil)
}

// UploadJobPhoto stores a before/after work photo for the completion summary.
func (s *Service) UploadJobPhoto(ctx context.Context, maidID uuid.UUID, bookingID string, file *multipart.FileHeader) (string, error) {
	bookingUUID, parseError := uuid.Parse(bookingID)
	if parseError != nil {
		return "", errors.New("invalid booking ID")
	}

	jobBooking, findError := s.repo.GetBookingByID(ctx, bookingUUID)
	if findError != nil {
		return "", errors.New("booking not found")
	}

	if jobBooking.MaidID != maidID {
		return "", errors.New("unauthorized")
	}

	openedFile, openError := file.Open()
	if openError != nil {
		return "", fmt.Errorf("failed to open photo: %w", openError)
	}
	defer openedFile.Close()

	contentType := file.Header.Get("Content-Type")
	objectName, uploadError := s.minioService.UploadImage(ctx, "jobs/photos", maidID, openedFile, file.Size, contentType)
	if uploadError != nil {
		return "", fmt.Errorf("failed to upload job photo: %w", uploadError)
	}

	return objectName, nil
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
		slog.Error("payment: escrow release did not go through, maid payout is stuck — needs retry or manual action",
			"booking_reference", booking.ReferenceNumber,
			"error", err)
	}

	slog.Info("booking: customer confirmed completion, booking is done",
		"booking_reference", booking.ReferenceNumber,
		"customer_id", customerID.String())

	return s.buildBookingResponse(ctx, booking, location, pricing, nil)
}

// ── Workflow F: Release escrow to maid wallet ─────────────────────────────────

func (s *Service) releaseEscrowPayment(ctx context.Context, booking *Booking, pricing *BookingPricing) error {
	if pricing == nil {
		return errors.New("pricing not found for escrow release")
	}

	if booking.PaymentCollectionTransactionID == nil || *booking.PaymentCollectionTransactionID == "" {
		return errors.New("no collection transaction found for escrow release")
	}

	maidUser, err := s.authRepo.FindUserByID(ctx, booking.MaidID)
	if err != nil {
		return fmt.Errorf("get maid: %w", err)
	}

	disbursementTransactionID, err := s.paymentClient.ReleaseEscrow(
		*booking.PaymentCollectionTransactionID,
		maidUser.PhoneNumber,
		maidUser.FullName,
		"Job payment "+booking.ReferenceNumber,
	)
	if err != nil {
		return fmt.Errorf("release escrow via payment service: %w", err)
	}

	if err := s.repo.UpdateBookingPaymentDisbursementTxID(ctx, booking.ID, disbursementTransactionID); err != nil {
		return fmt.Errorf("store disbursement transaction id: %w", err)
	}

	if err := s.repo.UpdatePaymentStatus(ctx, booking.ID, "disbursement_pending"); err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}

	slog.Info("payment: escrow release started, money is on its way to the maid",
		"booking_reference", booking.ReferenceNumber,
		"disbursement_transaction_id", disbursementTransactionID,
		"maid_payout_tzs", pricing.MaidPayoutAmount)

	return nil
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
			slog.Warn("booking: skipped one booking in the list because its response could not be built",
				"booking_id", booking.ID.String(),
				"error", err)
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
			slog.Warn("booking: skipped one booking in the list because its response could not be built",
				"booking_id", booking.ID.String(),
				"error", err)
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
		CompletionNotes: booking.CompletionNotes,
	}

	if booking.ServiceStartedAt != nil {
		startedAt := booking.ServiceStartedAt.Format(time.RFC3339)
		response.ServiceStartedAt = &startedAt
	}
	if booking.ServiceCompletedAt != nil {
		completedAt := booking.ServiceCompletedAt.Format(time.RFC3339)
		response.ServiceCompletedAt = &completedAt
	}
	if booking.BeforePhotoURL != nil {
		presignedBefore, presignError := s.minioService.GetPresignedURL(ctx, *booking.BeforePhotoURL, time.Hour)
		if presignError == nil {
			response.BeforePhotoURL = presignedBefore
		}
	}
	if booking.AfterPhotoURL != nil {
		presignedAfter, presignError := s.minioService.GetPresignedURL(ctx, *booking.AfterPhotoURL, time.Hour)
		if presignError == nil {
			response.AfterPhotoURL = presignedAfter
		}
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

func (s *Service) UpdateMaidLocation(ctx context.Context, bookingID, callerID string, lat, lng float64) error {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return ErrInvalidLocation
	}

	row, err := s.repo.MaidLocationByBookingID(ctx, bookingID)
	if err != nil {
		return ErrBookingNotFound
	}

	if row.MaidID != callerID {
		return ErrUnauthorized
	}

	if row.BookingStatus != "confirmed" && row.BookingStatus != "in_progress" {
		return ErrInvalidState
	}

	return s.repo.UpdateMaidLocation(ctx, bookingID, lat, lng)
}

func (s *Service) MaidLocationForCustomer(ctx context.Context, bookingID, callerID string) (*MaidLocationResponse, error) {
	row, err := s.repo.MaidLocationByBookingID(ctx, bookingID)
	if err != nil {
		return nil, ErrBookingNotFound
	}

	if row.CustomerID != callerID {
		return nil, ErrUnauthorized
	}

	if row.MaidLat == nil || row.MaidLng == nil || row.UpdatedAt == nil {
		return nil, ErrLocationNotAvailable
	}

	dist := haversineKm(*row.MaidLat, *row.MaidLng, row.CustomerLat, row.CustomerLng)
	eta := int((dist / 4.0) * 60)

	return &MaidLocationResponse{
		Lat:        *row.MaidLat,
		Lng:        *row.MaidLng,
		UpdatedAt:  *row.UpdatedAt,
		DistanceKm: math.Round(dist*100) / 100,
		ETAMinutes: eta,
	}, nil
}

func placeholderCustomerEmail(phoneNumber string) string {
	return phoneNumber + "@kazi.noemail"
}

func splitFullName(fullName string) (firstName, lastName string) {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
