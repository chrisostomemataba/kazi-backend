package maid

import (
	"kazi-backend/internal/auth"
	"kazi-backend/internal/common/storage"
	"kazi-backend/internal/notification"

	"gorm.io/gorm"
)

type Module struct {
	Handler    *Handler
	Repository *Repository
}

func NewModule(db *gorm.DB, authRepo *auth.Repository, minioService *storage.MinIOService, notificationService *notification.Service) *Module {
	repo    := NewRepository(db)
	service := NewService(repo, authRepo, minioService, notificationService)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Repository: repo,
	}
}