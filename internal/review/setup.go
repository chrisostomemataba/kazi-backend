package review

import (
	"kazi-backend/internal/auth"
	"kazi-backend/internal/booking"
	"kazi-backend/internal/common/storage"

	"gorm.io/gorm"
)

type Module struct {
	Handler *Handler
}

func NewModule(db *gorm.DB, authRepo *auth.Repository, bookingRepo *booking.Repository, minioService *storage.MinIOService) *Module {
	repo    := NewRepository(db)
	service := NewService(repo, authRepo, bookingRepo, minioService)
	handler := NewHandler(service)

	return &Module{Handler: handler}
}