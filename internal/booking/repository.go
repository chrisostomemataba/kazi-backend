package booking

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateBooking(ctx context.Context, booking *Booking) error {
	return r.db.WithContext(ctx).Create(booking).Error
}

func (r *Repository) CreateBookingLocation(ctx context.Context, location *BookingLocation) error {
	return r.db.WithContext(ctx).Create(location).Error
}

func (r *Repository) CreateBookingPricing(ctx context.Context, pricing *BookingPricing) error {
	return r.db.WithContext(ctx).Create(pricing).Error
}

func (r *Repository) AddTimelineEvent(ctx context.Context, event *BookingTimeline) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *Repository) GetBookingByID(ctx context.Context, bookingID uuid.UUID) (*Booking, error) {
	var booking Booking
	err := r.db.WithContext(ctx).Where("id = ?", bookingID).First(&booking).Error
	return &booking, err
}

func (r *Repository) GetBookingLocation(ctx context.Context, bookingID uuid.UUID) (*BookingLocation, error) {
	var location BookingLocation
	err := r.db.WithContext(ctx).Where("booking_id = ?", bookingID).First(&location).Error
	return &location, err
}

func (r *Repository) GetBookingPricing(ctx context.Context, bookingID uuid.UUID) (*BookingPricing, error) {
	var pricing BookingPricing
	err := r.db.WithContext(ctx).Where("booking_id = ?", bookingID).First(&pricing).Error
	return &pricing, err
}

func (r *Repository) GetBookingTimeline(ctx context.Context, bookingID uuid.UUID) ([]BookingTimeline, error) {
	var timeline []BookingTimeline
	err := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Order("event_timestamp ASC").
		Find(&timeline).Error
	return timeline, err
}

func (r *Repository) CheckMaidAvailability(ctx context.Context, maidID uuid.UUID, bookingDate time.Time, startTime, endTime string) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&Booking{}).
		Where("maid_id = ?", maidID).
		Where("booking_date = ?", bookingDate).
		Where("booking_status IN ?", []string{"confirmed", "in_progress"}).
		Where(
			"(start_time < ? AND end_time > ?)",
			endTime, startTime,
		).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count == 0, nil
}

func (r *Repository) UpdateBookingStatus(ctx context.Context, bookingID uuid.UUID, status string) error {
	return r.db.WithContext(ctx).
		Model(&Booking{}).
		Where("id = ?", bookingID).
		Update("booking_status", status).Error
}

func (r *Repository) UpdatePaymentStatus(ctx context.Context, bookingID uuid.UUID, status string) error {
	return r.db.WithContext(ctx).
		Model(&Booking{}).
		Where("id = ?", bookingID).
		Update("payment_status", status).Error
}

func (r *Repository) CreatePayment(ctx context.Context, payment *Payment) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

func (r *Repository) UpdatePayment(ctx context.Context, payment *Payment) error {
	return r.db.WithContext(ctx).Save(payment).Error
}

func (r *Repository) GetCustomerBookings(ctx context.Context, customerID uuid.UUID, status string, page, limit int) ([]Booking, error) {
	var bookings []Booking
	offset := (page - 1) * limit
	
	query := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("created_at DESC")
	
	if status != "" {
		query = query.Where("booking_status = ?", status)
	}
	
	err := query.Limit(limit).Offset(offset).Find(&bookings).Error
	return bookings, err
}

func (r *Repository) GetMaidBookings(ctx context.Context, maidID uuid.UUID, status string, page, limit int) ([]Booking, error) {
	var bookings []Booking
	offset := (page - 1) * limit
	
	query := r.db.WithContext(ctx).
		Where("maid_id = ?", maidID).
		Order("created_at DESC")
	
	if status != "" {
		query = query.Where("booking_status = ?", status)
	}
	
	err := query.Limit(limit).Offset(offset).Find(&bookings).Error
	return bookings, err
}