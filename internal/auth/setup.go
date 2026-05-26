package auth

import (
	"kazi-backend/internal/common/sms"

	"gorm.io/gorm"
)

type Module struct {
	Handler    *Handler
	Repository *Repository
}

func NewModule(db *gorm.DB, smsService *sms.SMSService, jwtSecret string, isDev bool) *Module {
	repo    := NewRepository(db)
	service := NewService(repo, smsService, jwtSecret, isDev)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Repository: repo,
	}
}