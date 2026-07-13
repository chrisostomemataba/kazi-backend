package chat

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

func (r *Repository) Create(ctx context.Context, message *ChatMessage) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *Repository) ListByBooking(ctx context.Context, bookingID uuid.UUID, limit, offset int) ([]ChatMessage, error) {
	var messages []ChatMessage
	findError := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, findError
}

func (r *Repository) GetLastMessage(ctx context.Context, bookingID uuid.UUID) (*ChatMessage, error) {
	var message ChatMessage
	findError := r.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Order("created_at DESC").
		First(&message).Error
	if findError != nil {
		if findError == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, findError
	}
	return &message, nil
}

// MarkAllRead stamps every unread message addressed to the reader in this
// conversation and returns how many rows changed, so the caller knows
// whether a read receipt is worth broadcasting.
func (r *Repository) MarkAllRead(ctx context.Context, bookingID, readerID uuid.UUID, readAt time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&ChatMessage{}).
		Where("booking_id = ? AND recipient_id = ? AND read_at IS NULL", bookingID, readerID).
		Update("read_at", readAt)
	return result.RowsAffected, result.Error
}

func (r *Repository) CountUnreadByBooking(ctx context.Context, bookingID, recipientID uuid.UUID) (int64, error) {
	var count int64
	countError := r.db.WithContext(ctx).
		Model(&ChatMessage{}).
		Where("booking_id = ? AND recipient_id = ? AND read_at IS NULL", bookingID, recipientID).
		Count(&count).Error
	return count, countError
}

func (r *Repository) CountUnreadForUser(ctx context.Context, recipientID uuid.UUID) (int64, error) {
	var count int64
	countError := r.db.WithContext(ctx).
		Model(&ChatMessage{}).
		Where("recipient_id = ? AND read_at IS NULL", recipientID).
		Count(&count).Error
	return count, countError
}
