package booking

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Booking struct {
	ID                               uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ReferenceNumber                  string    `gorm:"type:varchar(20);uniqueIndex;not null"`
	CustomerID                       uuid.UUID `gorm:"type:uuid;not null;index:idx_customer_bookings"`
	MaidID                           uuid.UUID `gorm:"type:uuid;not null;index:idx_maid_bookings"`
	ServiceType                      string    `gorm:"type:varchar(50);not null"`
	BookingDate                      time.Time `gorm:"type:date;not null"`
	StartTime                        string    `gorm:"type:varchar(5);not null"` // HH:MM format
	EndTime                          string    `gorm:"type:varchar(5);not null"`
	DurationHours                    float64   `gorm:"type:decimal(4,2);not null"`
	SpecialInstructions              string    `gorm:"type:text"`
	BookingStatus                    string    `gorm:"type:varchar(30);default:'pending_maid';index:idx_booking_status"`
	PaymentStatus                    string    `gorm:"type:varchar(30);default:'unpaid';index:idx_payment_status"`
	ServiceStartedAt                 *time.Time
	ServiceCompletedAt               *time.Time
	MaidCurrentLat                   *float64 `gorm:"type:decimal(10,8)"`
	MaidCurrentLng                   *float64 `gorm:"type:decimal(11,8)"`
	MaidLocationUpdatedAt            *time.Time
	PaymentCollectionTransactionID   *string `gorm:"type:text"`
	PaymentDisbursementTransactionID *string `gorm:"type:text"`
	CreatedAt                        time.Time
	UpdatedAt                        time.Time
}

type BookingLocation struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	BookingID           uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	CustomerAddress     string    `gorm:"type:text;not null"`
	CustomerLocationLat float64   `gorm:"type:decimal(10,8);not null"`
	CustomerLocationLng float64   `gorm:"type:decimal(11,8);not null"`
	District            string    `gorm:"type:varchar(50)"`
	Ward                string    `gorm:"type:varchar(50)"`
	ArrivalVerifiedLat  *float64  `gorm:"type:decimal(10,8)"`
	ArrivalVerifiedLng  *float64  `gorm:"type:decimal(11,8)"`
	ArrivalVerifiedAt   *time.Time
}

type BookingPricing struct {
	ID                       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	BookingID                uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	HourlyRate               int       `gorm:"not null"` // snapshot at booking time
	SubtotalAmount           int       `gorm:"not null"`
	PlatformCommissionRate   float64   `gorm:"type:decimal(4,2);not null"`
	PlatformCommissionAmount int       `gorm:"not null"`
	TotalAmount              int       `gorm:"not null"`
	MaidPayoutAmount         int       `gorm:"not null"`
}

type BookingTimeline struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	BookingID      uuid.UUID  `gorm:"type:uuid;not null;index:idx_booking_timeline"`
	EventType      string     `gorm:"type:varchar(50);not null"`
	EventTimestamp time.Time  `gorm:"not null"`
	TriggeredBy    *uuid.UUID `gorm:"type:uuid"` // user_id who triggered event
	Notes          string     `gorm:"type:text"`
}

type Payment struct {
	ID                   uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	BookingID            uuid.UUID `gorm:"type:uuid;not null;index:idx_payment_booking"`
	UserID               uuid.UUID `gorm:"type:uuid;not null"`
	TransactionType      string    `gorm:"type:varchar(20);not null"` // booking_payment, contract_payment, withdrawal
	Amount               int       `gorm:"not null"`
	Provider             string    `gorm:"type:varchar(20)"` // Mpesa, TigoPesa, AirtelMoney, Halopesa
	AccountNumber        string    `gorm:"type:varchar(20)"`
	Status               string    `gorm:"type:varchar(20);default:'pending'"`
	AzampayTransactionID string    `gorm:"type:varchar(100);uniqueIndex"`
	AzampayReference     string    `gorm:"type:varchar(100)"`
	FailureReason        string    `gorm:"type:text"`
	InitiatedAt          time.Time `gorm:"default:now()"`
	CompletedAt          *time.Time
}

func (b *Booking) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.ReferenceNumber == "" {
		b.ReferenceNumber = generateBookingReference()
	}
	return nil
}

func (bl *BookingLocation) BeforeCreate(tx *gorm.DB) error {
	if bl.ID == uuid.Nil {
		bl.ID = uuid.New()
	}
	return nil
}

func (bp *BookingPricing) BeforeCreate(tx *gorm.DB) error {
	if bp.ID == uuid.Nil {
		bp.ID = uuid.New()
	}
	return nil
}

func (bt *BookingTimeline) BeforeCreate(tx *gorm.DB) error {
	if bt.ID == uuid.Nil {
		bt.ID = uuid.New()
	}
	return nil
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func generateBookingReference() string {
	now := time.Now()
	return fmt.Sprintf("BK%d%02d%02d%04d", now.Year(), now.Month(), now.Day(), rand.Intn(10000))
}

type MaidLocationRow struct {
	MaidLat       *float64
	MaidLng       *float64
	UpdatedAt     *time.Time
	CustomerLat   float64
	CustomerLng   float64
	MaidID        string
	CustomerID    string
	BookingStatus string
}

type UpdateLocationRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type MaidLocationResponse struct {
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	UpdatedAt  time.Time `json:"updated_at"`
	DistanceKm float64   `json:"distance_km"`
	ETAMinutes int       `json:"eta_minutes"`
}
