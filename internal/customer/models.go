package customer

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CustomerProfile struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID            uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	PreferredLanguage string     `gorm:"type:varchar(10);default:'sw'"` // sw, en
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type CustomerLocation struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CustomerID  uuid.UUID `gorm:"type:uuid;not null;index:idx_customer_locations"`
	Label       string    `gorm:"type:varchar(50);not null"` // Home, Work, Other
	Address     string    `gorm:"type:text;not null"`
	Latitude    float64   `gorm:"type:decimal(10,8);not null"`
	Longitude   float64   `gorm:"type:decimal(11,8);not null"`
	District    string    `gorm:"type:varchar(50)"`
	Ward        string    `gorm:"type:varchar(50)"`
	IsDefault   bool      `gorm:"default:false"`
	CreatedAt   time.Time
}

type CustomerStatistics struct {
	ID                   uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	CustomerID           uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	AverageMaidRating    float64   `gorm:"type:decimal(2,1);default:0.0"` // maids rate customers
	TotalBookings        int       `gorm:"default:0"`
	CompletedBookings    int       `gorm:"default:0"`
	CancelledBookings    int       `gorm:"default:0"`
	PaymentOnTimeRate    float64   `gorm:"type:decimal(4,2);default:100.0"` // percentage
	TotalSpent           int64     `gorm:"default:0"`
	LastBookingAt        *time.Time
	LastCalculatedAt     time.Time `gorm:"default:now()"`
}

func (cp *CustomerProfile) BeforeCreate(tx *gorm.DB) error {
	if cp.ID == uuid.Nil {
		cp.ID = uuid.New()
	}
	return nil
}

func (cl *CustomerLocation) BeforeCreate(tx *gorm.DB) error {
	if cl.ID == uuid.Nil {
		cl.ID = uuid.New()
	}
	return nil
}

func (cs *CustomerStatistics) BeforeCreate(tx *gorm.DB) error {
	if cs.ID == uuid.Nil {
		cs.ID = uuid.New()
	}
	return nil
}