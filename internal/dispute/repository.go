package dispute

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, dispute *Dispute) error {
	return r.db.WithContext(ctx).Create(dispute).Error
}

func (r *Repository) ExistsOpenByBookingAndReporter(ctx context.Context, bookingID, reporterID uuid.UUID) (bool, error) {
	var count int64
	countError := r.db.WithContext(ctx).
		Model(&Dispute{}).
		Where("booking_id = ? AND reporter_id = ? AND status IN ?",
			bookingID, reporterID, []string{"open", "investigating"}).
		Count(&count).Error
	if countError != nil {
		return false, countError
	}
	return count > 0, nil
}

func (r *Repository) GetByReporter(ctx context.Context, reporterID uuid.UUID, limit, offset int) ([]Dispute, error) {
	var disputes []Dispute
	findError := r.db.WithContext(ctx).
		Where("reporter_id = ?", reporterID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&disputes).Error
	return disputes, findError
}
