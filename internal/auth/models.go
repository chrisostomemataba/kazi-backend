package auth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	PhoneNumber       string    `gorm:"type:varchar(13);uniqueIndex;not null"`
	FullName          string    `gorm:"type:varchar(100)"`
	ProfilePhotoURL   string    `gorm:"type:text"`
	IsActive          bool      `gorm:"default:true"`
	IsPhoneVerified   bool      `gorm:"default:true"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type UserRole struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_user_roles_lookup"`
	RoleType  string    `gorm:"type:varchar(10);not null;index:idx_user_roles_lookup"`
	IsActive  bool      `gorm:"default:true"`
	CreatedAt time.Time

	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

type OTPCode struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	PhoneNumber string    `gorm:"type:varchar(13);not null;index:idx_otp_lookup"`
	Code        string    `gorm:"type:varchar(6);not null"`
	Purpose     string    `gorm:"type:varchar(20);not null"`
	IsUsed      bool      `gorm:"default:false;index:idx_otp_lookup"`
	Attempts    int       `gorm:"default:0"`
	ExpiresAt   time.Time `gorm:"not null;index:idx_otp_lookup"`
	CreatedAt   time.Time
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (ur *UserRole) BeforeCreate(tx *gorm.DB) error {
	if ur.ID == uuid.Nil {
		ur.ID = uuid.New()
	}
	return nil
}

func (o *OTPCode) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}