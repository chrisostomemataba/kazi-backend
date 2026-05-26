package admin

import (
	"kazi-backend/internal/common/storage"
	"kazi-backend/internal/notification"

	"gorm.io/gorm"
)

type Module struct {
	Handler *Handler
}

func NewModule(db *gorm.DB, minioService *storage.MinIOService, notificationService *notification.Service, jwtSecret string) *Module {
	repo    := NewRepository(db)
	service := NewService(repo, minioService, notificationService, jwtSecret)
	handler := NewHandler(service)

	return &Module{Handler: handler}
}