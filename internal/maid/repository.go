package maid

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

func (r *Repository) CreateMaidProfile(ctx context.Context, profile *MaidProfile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *Repository) GetMaidProfileByUserID(ctx context.Context, userID uuid.UUID) (*MaidProfile, error) {
	var profile MaidProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
	return &profile, err
}

func (r *Repository) UpdateMaidProfile(ctx context.Context, profile *MaidProfile) error {
	return r.db.WithContext(ctx).Save(profile).Error
}

func (r *Repository) CreateMaidService(ctx context.Context, service *MaidService) error {
	return r.db.WithContext(ctx).Create(service).Error
}

func (r *Repository) CreateVerificationDocument(ctx context.Context, doc *MaidVerificationDocument) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *Repository) GetVerificationDocuments(ctx context.Context, maidID uuid.UUID) ([]MaidVerificationDocument, error) {
	var docs []MaidVerificationDocument
	err := r.db.WithContext(ctx).Where("maid_id = ?", maidID).Find(&docs).Error
	return docs, err
}

func (r *Repository) GetPendingVerifications(ctx context.Context, limit, offset int) ([]MaidProfile, error) {
	var profiles []MaidProfile
	err := r.db.WithContext(ctx).
		Where("verification_status = ?", "pending").
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&profiles).Error
	return profiles, err
}

func (r *Repository) UpdateVerificationStatus(ctx context.Context, maidID uuid.UUID, status, reason string) error {
	updates := map[string]interface{}{
		"verification_status": status,
		"rejection_reason":    reason,
	}
	if status == "approved" {
		updates["verified_at"] = gorm.Expr("NOW()")
	}
	return r.db.WithContext(ctx).
		Model(&MaidProfile{}).
		Where("user_id = ?", maidID).
		Updates(updates).Error
}