package admin

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminUser struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Username     string    `gorm:"type:varchar(50);uniqueIndex;not null"`
	PasswordHash string    `gorm:"type:text;not null"`
	FullName     string    `gorm:"type:varchar(100)"`
	Role         string    `gorm:"type:varchar(20);not null"`
	IsActive     bool      `gorm:"default:true"`
	LastLoginAt  *time.Time
	CreatedAt    time.Time
}

type AuditLog struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ActorID          uuid.UUID `gorm:"type:uuid;index:idx_audit_actor"`
	ActorType        string    `gorm:"type:varchar(10);not null"`
	ActionType       string    `gorm:"type:varchar(30);not null"`
	TargetEntityType string    `gorm:"type:varchar(30);index:idx_audit_target"`
	TargetEntityID   *uuid.UUID `gorm:"type:uuid;index:idx_audit_target"`
	Changes          string    `gorm:"type:jsonb"`
	IPAddress        string    `gorm:"type:inet"`
	CreatedAt        time.Time `gorm:"index:idx_audit_actor"`
}

func (au *AdminUser) BeforeCreate(tx *gorm.DB) error {
	if au.ID == uuid.Nil {
		au.ID = uuid.New()
	}
	return nil
}

func (al *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if al.ID == uuid.Nil {
		al.ID = uuid.New()
	}
	return nil
}