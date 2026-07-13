package dispute

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"time"

	"kazi-backend/internal/booking"
	"kazi-backend/internal/common/storage"
	"kazi-backend/internal/notification"

	"github.com/google/uuid"
)

// Booking states in which a participant may still open a complaint.
var disputableStatuses = map[string]bool{
	"confirmed":   true,
	"in_progress": true,
	"completed":   true,
}

type Service struct {
	repo            *Repository
	bookingRepo     *booking.Repository
	minioService    *storage.MinIOService
	notificationSvc *notification.Service
}

func NewService(
	repo *Repository,
	bookingRepo *booking.Repository,
	minioService *storage.MinIOService,
	notificationSvc *notification.Service,
) *Service {
	return &Service{
		repo:            repo,
		bookingRepo:     bookingRepo,
		minioService:    minioService,
		notificationSvc: notificationSvc,
	}
}

func (s *Service) CreateDispute(ctx context.Context, reporterID uuid.UUID, req *CreateDisputeRequest) (*DisputeResponse, error) {
	bookingUUID, parseError := uuid.Parse(req.BookingID)
	if parseError != nil {
		return nil, errors.New("invalid booking ID")
	}

	reportedBooking, findBookingError := s.bookingRepo.GetBookingByID(ctx, bookingUUID)
	if findBookingError != nil {
		return nil, errors.New("booking not found")
	}

	isCustomer := reportedBooking.CustomerID == reporterID
	isMaid := reportedBooking.MaidID == reporterID
	if !isCustomer && !isMaid {
		return nil, errors.New("only booking participants can report a problem")
	}

	if !disputableStatuses[reportedBooking.BookingStatus] {
		return nil, fmt.Errorf("cannot report a problem for booking in status: %s", reportedBooking.BookingStatus)
	}

	hasOpenDispute, existsError := s.repo.ExistsOpenByBookingAndReporter(ctx, bookingUUID, reporterID)
	if existsError != nil {
		return nil, fmt.Errorf("failed to check existing disputes: %w", existsError)
	}
	if hasOpenDispute {
		return nil, errors.New("you already have an open report for this booking")
	}

	reportedUserID := reportedBooking.MaidID
	if isMaid {
		reportedUserID = reportedBooking.CustomerID
	}

	newDispute := &Dispute{
		BookingID:       bookingUUID,
		ReporterID:      reporterID,
		ReportedUserID:  reportedUserID,
		DisputeType:     req.DisputeType,
		Description:     req.Description,
		EvidenceURLs:    encodeEvidenceURLs(req.EvidenceURLs),
		RefundRequested: req.RefundRequested,
		Status:          "open",
	}

	createError := s.repo.Create(ctx, newDispute)
	if createError != nil {
		return nil, fmt.Errorf("failed to create dispute: %w", createError)
	}

	s.bookingRepo.AddTimelineEvent(ctx, &booking.BookingTimeline{
		BookingID:      bookingUUID,
		EventType:      "dispute_opened",
		EventTimestamp: time.Now(),
		TriggeredBy:    &reporterID,
		Notes:          fmt.Sprintf("Problem reported: %s", req.DisputeType),
	})

	s.notificationSvc.NotifyDisputeOpened(ctx, reporterID, reportedBooking.ReferenceNumber)

	slog.Info("dispute: a problem was reported on a booking",
		"booking_reference", reportedBooking.ReferenceNumber,
		"dispute_type", req.DisputeType,
		"refund_requested", req.RefundRequested,
		"reporter_id", reporterID.String())

	return s.buildResponse(newDispute, reportedBooking.ReferenceNumber), nil
}

func (s *Service) GetMyDisputes(ctx context.Context, reporterID uuid.UUID, limit, offset int) ([]DisputeResponse, error) {
	disputes, findError := s.repo.GetByReporter(ctx, reporterID, limit, offset)
	if findError != nil {
		return nil, fmt.Errorf("failed to fetch disputes: %w", findError)
	}

	responses := make([]DisputeResponse, 0, len(disputes))
	for i := range disputes {
		bookingRef := ""
		relatedBooking, findBookingError := s.bookingRepo.GetBookingByID(ctx, disputes[i].BookingID)
		if findBookingError == nil && relatedBooking != nil {
			bookingRef = relatedBooking.ReferenceNumber
		}
		responses = append(responses, *s.buildResponse(&disputes[i], bookingRef))
	}

	return responses, nil
}

func (s *Service) UploadEvidence(ctx context.Context, reporterID uuid.UUID, file *multipart.FileHeader) (string, error) {
	openedFile, openError := file.Open()
	if openError != nil {
		return "", fmt.Errorf("failed to open evidence file: %w", openError)
	}
	defer openedFile.Close()

	contentType := file.Header.Get("Content-Type")
	objectName, uploadError := s.minioService.UploadImage(ctx, "disputes/evidence", reporterID, openedFile, file.Size, contentType)
	if uploadError != nil {
		return "", fmt.Errorf("failed to upload evidence: %w", uploadError)
	}

	return objectName, nil
}

func (s *Service) buildResponse(d *Dispute, bookingRef string) *DisputeResponse {
	evidenceObjectNames := decodeEvidenceURLs(d.EvidenceURLs)

	presignedURLs := make([]string, 0, len(evidenceObjectNames))
	for _, objectName := range evidenceObjectNames {
		presignedURL, presignError := s.minioService.GetPresignedURL(context.Background(), objectName, time.Hour)
		if presignError != nil {
			continue
		}
		presignedURLs = append(presignedURLs, presignedURL)
	}

	return &DisputeResponse{
		ID:              d.ID.String(),
		BookingID:       d.BookingID.String(),
		BookingRef:      bookingRef,
		DisputeType:     d.DisputeType,
		Description:     d.Description,
		EvidenceURLs:    presignedURLs,
		RefundRequested: d.RefundRequested,
		Status:          d.Status,
		Resolution:      d.Resolution,
		CreatedAt:       d.CreatedAt.Format(time.RFC3339),
	}
}
