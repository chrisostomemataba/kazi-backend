package maid

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MaidProfile struct {
	ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID             uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Bio                string    `gorm:"type:text"`
	HourlyRate         int       `gorm:"not null"` // in TZS
	VerificationStatus string    `gorm:"type:varchar(20);default:'pending';index:idx_maid_status"`
	IDNumber           string    `gorm:"type:varchar(50)"`
	IDType             string    `gorm:"type:varchar(20)"`
	RejectionReason    string    `gorm:"type:text"`
	IsAvailableNow     bool      `gorm:"default:true"`
	VerifiedAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type MaidService struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MaidID      uuid.UUID `gorm:"type:uuid;not null;index:idx_maid_service"`
	ServiceType string    `gorm:"type:varchar(50);not null;index:idx_maid_service"`
}

type MaidVerificationDocument struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MaidID       uuid.UUID `gorm:"type:uuid;not null;index:idx_maid_docs"`
	DocumentType string    `gorm:"type:varchar(20);not null;index:idx_maid_docs"` // selfie_video, id_photo
	FileURL      string    `gorm:"type:text;not null"`
	UploadedAt   time.Time `gorm:"default:now()"`
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