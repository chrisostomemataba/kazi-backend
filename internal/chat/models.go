package chat

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatMessage struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	BookingID   uuid.UUID `gorm:"type:uuid;not null;index:idx_chat_booking"`
	SenderID    uuid.UUID `gorm:"type:uuid;not null;index:idx_chat_sender"`
	RecipientID uuid.UUID `gorm:"type:uuid;not null;index:idx_chat_recipient"`
	Body        string    `gorm:"type:text;not null"`
	CreatedAt   time.Time `gorm:"index:idx_chat_booking"`
	ReadAt      *time.Time
}

func (m *ChatMessage) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

type SendMessageRequest struct {
	Body string `json:"body" validate:"required,min=1,max=2000"`
}

type MessageResponse struct {
	ID        string  `json:"id"`
	BookingID string  `json:"booking_id"`
	SenderID  string  `json:"sender_id"`
	Body      string  `json:"body"`
	CreatedAt string  `json:"created_at"`
	ReadAt    *string `json:"read_at,omitempty"`
}

type ConversationUser struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	PhotoURL string `json:"photo_url,omitempty"`
}

type ConversationResponse struct {
	BookingID     string           `json:"booking_id"`
	BookingRef    string           `json:"booking_ref"`
	ServiceType   string           `json:"service_type"`
	BookingStatus string           `json:"booking_status"`
	BookingDate   string           `json:"booking_date"`
	OtherUser     ConversationUser `json:"other_user"`
	LastMessage   string           `json:"last_message,omitempty"`
	LastMessageAt *string          `json:"last_message_at,omitempty"`
	UnreadCount   int64            `json:"unread_count"`
}

type UnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}
