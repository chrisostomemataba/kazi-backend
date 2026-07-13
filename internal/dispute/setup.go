package dispute

import (
	"kazi-backend/internal/booking"
	"kazi-backend/internal/common/storage"
	"kazi-backend/internal/notification"

	"gorm.io/gorm"
)

type Module struct {
	Handler *Handler
}

func NewModule(
	db *gorm.DB,
	bookingRepo *booking.Repository,
	minioService *storage.MinIOService,
	notificationService *notification.Service,
) *Module {
	repo    := NewRepository(db)
	service := NewService(repo, bookingRepo, minioService, notificationService)
	handler := NewHandler(service)

	return &Module{Handler: handler}
}
