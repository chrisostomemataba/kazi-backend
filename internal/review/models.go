package review
 
import (
	"time"
 
	"github.com/google/uuid"
	"gorm.io/gorm"
)
 
type Review struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	BookingID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	ReviewerID  uuid.UUID `gorm:"type:uuid;not null;index"`
	RevieweeID  uuid.UUID `gorm:"type:uuid;not null;index"`
	Rating      int       `gorm:"not null"`
	Comment     string    `gorm:"type:text"`
	Tags        string    `gorm:"type:text"`
	IsVisible   bool      `gorm:"default:true"`
	CreatedAt   time.Time
}
 
func (r *Review) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
 
type CreateReviewRequest struct {
	BookingID string   `json:"booking_id" validate:"required,uuid"`
	Rating    int      `json:"rating" validate:"required,min=1,max=5"`
	Comment   string   `json:"comment" validate:"max=1000"`
	Tags      []string `json:"tags" validate:"omitempty,max=5,dive,oneof=punctual thorough friendly professional fast"`
}
 
type ReviewResponse struct {
	ID           string   `json:"id"`
	BookingID    string   `json:"booking_id"`
	ReviewerID   string   `json:"reviewer_id"`
	ReviewerName string   `json:"reviewer_name"`
	ReviewerPhotoURL string `json:"reviewer_photo_url,omitempty"`
	Rating       int      `json:"rating"`
	Comment      string   `json:"comment"`
	Tags         []string `json:"tags"`
	CreatedAt    string   `json:"created_at"`
}