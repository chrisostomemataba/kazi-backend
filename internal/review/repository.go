package review
 
import (
	"context"
	"encoding/json"
	"strings"
 
	"github.com/google/uuid"
	"gorm.io/gorm"
)
 
type Repository struct {
	db *gorm.DB
}
 
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}
 
func (r *Repository) Create(ctx context.Context, review *Review) error {
	return r.db.WithContext(ctx).Create(review).Error
}
 
func (r *Repository) ExistsByBookingID(ctx context.Context, bookingID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Review{}).
		Where("booking_id = ?", bookingID).
		Count(&count).Error
	return count > 0, err
}
 
func (r *Repository) GetByMaidID(ctx context.Context, maidID uuid.UUID, limit, offset int) ([]Review, error) {
	var reviews []Review
	err := r.db.WithContext(ctx).
		Where("reviewee_id = ? AND is_visible = ?", maidID, true).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&reviews).Error
	return reviews, err
}
 
func EncodeTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	b, _ := json.Marshal(tags)
	return string(b)
}
 
func DecodeTags(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return strings.Split(raw, ",")
	}
	return tags
}