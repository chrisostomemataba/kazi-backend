package auth

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


func (r *Repository) FindUserByPhone(phoneNumber string) (*User, error) {
	var user User
	err := r.db.Where("phone_number = ?", phoneNumber).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindUserByID(ctx context.Context, userID uuid.UUID) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	return &user, err
}

func (r *Repository) CreateUser(user *User) error {
	return r.db.Create(user).Error
}

func (r *Repository) CreateOTP(otp *OTPCode) error {
	return r.db.Create(otp).Error
}

func (r *Repository) FindValidOTP(phoneNumber, code string) (*OTPCode, error) {
	var otp OTPCode
	err := r.db.Where(
		"phone_number = ? AND code = ? AND is_used = ? AND expires_at > ?",
		phoneNumber, code, false, time.Now(),
	).First(&otp).Error
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

func (r *Repository) MarkOTPAsUsed(otpID uuid.UUID) error {
	return r.db.Model(&OTPCode{}).Where("id = ?", otpID).Update("is_used", true).Error
}

func (r *Repository) CreateUserRole(userRole *UserRole) error {
	return r.db.Create(userRole).Error
}

func (r *Repository) GetUserRoles(userID uuid.UUID) ([]string, error) {
	var roles []string
	err := r.db.Model(&UserRole{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Pluck("role_type", &roles).Error
	return roles, err
}

func (r *Repository) CountRecentOTPs(phoneNumber string, since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&OTPCode{}).
		Where("phone_number = ? AND created_at > ?", phoneNumber, since).
		Count(&count).Error
	return count, err
}

func (r *Repository) IncrementOTPAttempts(otpID uuid.UUID) error {
	return r.db.Model(&OTPCode{}).Where("id = ?", otpID).
		UpdateColumn("attempts", gorm.Expr("attempts + ?", 1)).Error
}

func (r *Repository) UpdateUser(user *User) error {
	return r.db.Save(user).Error
}