package maid

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MaidProfile struct {
	ID                   uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID               uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	Bio                  string     `gorm:"type:text"`
	Gender               string     `gorm:"type:varchar(10)"` // male, female, other
	DateOfBirth          *time.Time `gorm:"type:date"`
	HomeAddress          string     `gorm:"type:text"`
	HomeLocationLat      *float64   `gorm:"type:decimal(10,8)"`
	HomeLocationLng      *float64   `gorm:"type:decimal(11,8)"`
	District             string     `gorm:"type:varchar(50)"`
	Ward                 string     `gorm:"type:varchar(50)"`
	HourlyRate           int        `gorm:"not null"` // in TZS, for regular bookings
	OffersContracts      bool       `gorm:"default:false"`
	MonthlyContractRate  *int       `gorm:"type:integer"` // nullable, only if offers_contracts=true
	VerificationStatus   string     `gorm:"type:varchar(20);default:'pending';index:idx_maid_status"`
	IDNumber             string     `gorm:"type:varchar(50)"`
	IDType               string     `gorm:"type:varchar(20)"`
	RejectionReason      string     `gorm:"type:text"`
	IsAvailableNow       bool       `gorm:"default:true"`
	VerifiedAt           *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type MaidService struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MaidID      uuid.UUID `gorm:"type:uuid;not null;index:idx_maid_service"`
	ServiceType string    `gorm:"type:varchar(50);not null;index:idx_maid_service"`
}

type MaidVerificationDocument struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MaidID       uuid.UUID `gorm:"type:uuid;not null;index:idx_maid_docs"`
	DocumentType string    `gorm:"type:varchar(20);not null;index:idx_maid_docs"`
	FileURL      string    `gorm:"type:text;not null"`
	UploadedAt   time.Time `gorm:"default:now()"`
}

type MaidStatistics struct {
	ID                       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MaidID                   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	AverageRating            float64   `gorm:"type:decimal(2,1);default:0.0"`
	TotalReviews             int       `gorm:"default:0"`
	TotalJobsCompleted       int       `gorm:"default:0"`
	TotalContractsCompleted  int       `gorm:"default:0"`
	TotalEarnings            int64     `gorm:"default:0"`
	LastCalculatedAt         time.Time `gorm:"default:now()"`
}

type MaidWallet struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MaidID           uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"` 
	AvailableBalance int       `gorm:"not null;default:0"`             
	TotalEarned      int       `gorm:"not null;default:0"`             
	TotalWithdrawn   int       `gorm:"not null;default:0"`             
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
 
type WalletTransaction struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MaidID            uuid.UUID  `gorm:"type:uuid;not null;index:idx_wallet_tx"`
	TransactionType   string     `gorm:"type:varchar(30);not null"` // job_completed_credit | withdrawal_debit | withdrawal_refund
	Amount            int        `gorm:"not null"`                  // positive = credit, negative = debit
	RelatedBookingID  *uuid.UUID `gorm:"type:uuid"`
	CreatedAt         time.Time  `gorm:"index:idx_wallet_tx"`
}

func (mp *MaidProfile) BeforeCreate(tx *gorm.DB) error {
	if mp.ID == uuid.Nil {
		mp.ID = uuid.New()
	}
	return nil
}

func (ms *MaidService) BeforeCreate(tx *gorm.DB) error {
	if ms.ID == uuid.Nil {
		ms.ID = uuid.New()
	}
	return nil
}

func (mvd *MaidVerificationDocument) BeforeCreate(tx *gorm.DB) error {
	if mvd.ID == uuid.Nil {
		mvd.ID = uuid.New()
	}
	return nil
}

func (mst *MaidStatistics) BeforeCreate(tx *gorm.DB) error {
	if mst.ID == uuid.Nil {
		mst.ID = uuid.New()
	}
	return nil
}