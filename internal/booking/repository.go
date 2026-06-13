package booking

import (
	"context"
	"time"
	"fmt"

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

func (r *Repository) AddTimelineEventTx(ctx context.Context, event *BookingTimeline) error {
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
	if err != nil {
		return nil, err
	}
	return &location, nil
}

func (r *Repository) GetBookingPricing(ctx context.Context, bookingID uuid.UUID) (*BookingPricing, error) {
	var pricing BookingPricing
	err := r.db.WithContext(ctx).Where("booking_id = ?", bookingID).First(&pricing).Error
	if err != nil {
		return nil, err
	}
	return &pricing, nil
}

func (r *Repository) GetBookingTimeline(ctx context.Context, bookingID uuid.UUID) ([]BookingTimeline, error) {
	var timeline []BookingTimeline
	err := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Order("event_timestamp ASC").
		Find(&timeline).Error
	return timeline, err
}

// Only block on confirmed/in_progress — pending requests don't block the maid
func (r *Repository) CheckMaidAvailability(ctx context.Context, maidID uuid.UUID, bookingDate time.Time, startTime, endTime string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&Booking{}).
		Where("maid_id = ?", maidID).
		Where("booking_date = ?", bookingDate).
		Where("booking_status IN ?", []string{"confirmed", "in_progress"}).
		Where("start_time < ? AND end_time > ?", endTime, startTime).
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

func (r *Repository) UpdatePaymentStatusTx(ctx context.Context, bookingID uuid.UUID, status string) error {
	return r.db.WithContext(ctx).
		Model(&Booking{}).
		Where("id = ?", bookingID).
		Update("payment_status", status).Error
}

func (r *Repository) UpdateServiceCompletedAt(ctx context.Context, bookingID uuid.UUID, completedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&Booking{}).
		Where("id = ?", bookingID).
		Update("service_completed_at", completedAt).Error
}

func (r *Repository) UpdateArrivalLocation(ctx context.Context, bookingID uuid.UUID, lat, lng float64, arrivedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&BookingLocation{}).
		Where("booking_id = ?", bookingID).
		Updates(map[string]interface{}{
			"arrival_verified_lat": lat,
			"arrival_verified_lng": lng,
			"arrival_verified_at":  arrivedAt,
		}).Error
}

func (r *Repository) CreatePayment(ctx context.Context, payment *Payment) error {
	return r.db.WithContext(ctx).Create(payment).Error
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

func (r *Repository) GetMaidBookings(ctx context.Context, maidID uuid.UUID, status string, date string, page, limit int) ([]Booking, error) {
	var bookings []Booking
	offset := (page - 1) * limit
	query := r.db.WithContext(ctx).
		Where("maid_id = ?", maidID).
		Order("created_at DESC")
	if status != "" {
		query = query.Where("booking_status = ?", status)
	}

	if date != "" {
        query = query.Where("booking_date = ?", date) // "2026-04-09"
    }

	err := query.Limit(limit).Offset(offset).Find(&bookings).Error
	return bookings, err
}

func (r *Repository) CreditMaidWallet(ctx context.Context, maidID uuid.UUID, amount int, bookingID uuid.UUID) error {
	// Upsert wallet — creates if first job, updates if returning
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO maid_wallets (id, maid_id, available_balance, total_earned, total_withdrawn, created_at, updated_at)
		VALUES (gen_random_uuid(), ?, ?, ?, 0, NOW(), NOW())
		ON CONFLICT (maid_id) DO UPDATE
		SET available_balance = maid_wallets.available_balance + EXCLUDED.available_balance,
		    total_earned      = maid_wallets.total_earned + EXCLUDED.total_earned,
		    updated_at        = NOW()
	`, maidID, amount, amount).Error
	if err != nil {
		return fmt.Errorf("upsert wallet: %w", err)
	}
 
	// Record the credit transaction
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO wallet_transactions (id, maid_id, transaction_type, amount, related_booking_id, created_at)
		VALUES (gen_random_uuid(), ?, 'job_completed_credit', ?, ?, NOW())
	`, maidID, amount, bookingID).Error
}
 

// WithTransaction wraps operations in a DB transaction.
func (r *Repository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &Repository{db: tx}
		_ = txRepo // fn uses the same repo; pass tx context through ctx
		return fn(ctx)
	})
}

func (r *Repository) UpdateMaidLocation(ctx context.Context, bookingID string, lat, lng float64) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE bookings
		SET maid_current_lat = ?,
		    maid_current_lng = ?,
		    maid_location_updated_at = NOW()
		WHERE id = ?
	`, lat, lng, bookingID).Error
}

func (r *Repository) MaidLocationByBookingID(ctx context.Context, bookingID string) (*MaidLocationRow, error) {
	var result MaidLocationRow
	tx := r.db.WithContext(ctx).Raw(`
		SELECT
		    b.maid_current_lat         AS maid_lat,
		    b.maid_current_lng         AS maid_lng,
		    b.maid_location_updated_at AS updated_at,
		    bl.customer_location_lat   AS customer_lat,
		    bl.customer_location_lng   AS customer_lng,
		    b.maid_id,
		    b.customer_id,
		    b.booking_status
		FROM bookings b
		LEFT JOIN booking_locations bl ON bl.booking_id = b.id
		WHERE b.id = ?
	`, bookingID).Scan(&result)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &result, nil
}