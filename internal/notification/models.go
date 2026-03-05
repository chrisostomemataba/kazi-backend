package notification

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Notification struct {
	ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID             uuid.UUID `gorm:"type:uuid;not null;index:idx_user_notifications"`
	Title              string    `gorm:"type:varchar(100);not null"`
	Message            string    `gorm:"type:text;not null"`
	NotificationType   string    `gorm:"type:varchar(30);not null"`
	RelatedBookingID   *uuid.UUID `gorm:"type:uuid"`
	IsRead             bool      `gorm:"default:false;index:idx_user_notifications"`
	SentViaSMS         bool      `gorm:"default:false"`
	CreatedAt          time.Time `gorm:"index:idx_user_notifications"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}