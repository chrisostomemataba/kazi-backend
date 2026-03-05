package admin

import (
	"context"

	"kazi-backend/internal/maid"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindAdminByUsername(ctx context.Context, username string) (*AdminUser, error) {
	var admin AdminUser
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&admin).Error
	return &admin, err
}

func (r *Repository) CreateAdmin(ctx context.Context, admin *AdminUser) error {
	return r.db.WithContext(ctx).Create(admin).Error
}

func (r *Repository) UpdateLastLogin(ctx context.Context, adminID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&AdminUser{}).
		Where("id = ?", adminID).
		Update("last_login_at", gorm.Expr("NOW()")).Error
}

func (r *Repository) GetPendingVerifications(ctx context.Context, limit, offset int) ([]maid.MaidProfile, error) {
	var profiles []maid.MaidProfile
	err := r.db.WithContext(ctx).
		Where("verification_status = ?", "pending").
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&profiles).Error
	return profiles, err
}

func (r *Repository) GetMaidProfileByUserID(ctx context.Context, userID uuid.UUID) (*maid.MaidProfile, error) {
	var profile maid.MaidProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
	return &profile, err
}

func (r *Repository) UpdateVerificationStatus(ctx context.Context, userID uuid.UUID, status, reason string) error {
	updates := map[string]interface{}{
		"verification_status": status,
		"rejection_reason":    reason,
	}
	if status == "approved" {
		updates["verified_at"] = gorm.Expr("NOW()")
	}
	return r.db.WithContext(ctx).
		Model(&maid.MaidProfile{}).
		Where("user_id = ?", userID).
		Updates(updates).Error
}

func (r *Repository) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *Repository) GetVerificationDocuments(ctx context.Context, maidID uuid.UUID) ([]maid.MaidVerificationDocument, error) {
	var docs []maid.MaidVerificationDocument
	err := r.db.WithContext(ctx).Where("maid_id = ?", maidID).Find(&docs).Error
	return docs, err
}