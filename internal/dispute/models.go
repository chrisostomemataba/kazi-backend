package dispute

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Dispute struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	BookingID       uuid.UUID `gorm:"type:uuid;not null;index:idx_dispute_booking"`
	ReporterID      uuid.UUID `gorm:"type:uuid;not null;index:idx_dispute_reporter"`
	ReportedUserID  uuid.UUID `gorm:"type:uuid;not null"`
	DisputeType     string    `gorm:"type:varchar(40);not null"`
	Description     string    `gorm:"type:text;not null"`
	EvidenceURLs    string    `gorm:"type:text"` // JSON-encoded array of object names
	RefundRequested bool      `gorm:"not null;default:false"`
	Status          string    `gorm:"type:varchar(20);not null;default:'open'"` // open | investigating | resolved | rejected
	Resolution      string    `gorm:"type:text"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (d *Dispute) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

type CreateDisputeRequest struct {
	BookingID       string   `json:"booking_id" validate:"required,uuid"`
	DisputeType     string   `json:"dispute_type" validate:"required,oneof=service_not_completed poor_quality payment_issue safety_concern no_show behavior other"`
	Description     string   `json:"description" validate:"required,min=10,max=1000"`
	EvidenceURLs    []string `json:"evidence_urls" validate:"omitempty,max=4"`
	RefundRequested bool     `json:"refund_requested"`
}

type DisputeResponse struct {
	ID              string   `json:"id"`
	BookingID       string   `json:"booking_id"`
	BookingRef      string   `json:"booking_ref"`
	DisputeType     string   `json:"dispute_type"`
	Description     string   `json:"description"`
	EvidenceURLs    []string `json:"evidence_urls"`
	RefundRequested bool     `json:"refund_requested"`
	Status          string   `json:"status"`
	Resolution      string   `json:"resolution,omitempty"`
	CreatedAt       string   `json:"created_at"`
}

type EvidenceUploadResponse struct {
	ObjectName string `json:"object_name"`
}

func encodeEvidenceURLs(urls []string) string {
	if len(urls) == 0 {
		return "[]"
	}
	encoded, encodeError := json.Marshal(urls)
	if encodeError != nil {
		return "[]"
	}
	return string(encoded)
}

func decodeEvidenceURLs(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var urls []string
	decodeError := json.Unmarshal([]byte(raw), &urls)
	if decodeError != nil {
		return []string{}
	}
	return urls
}
