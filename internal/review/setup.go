package review

import (
	"kazi-backend/internal/auth"
	"kazi-backend/internal/booking"

	"gorm.io/gorm"
)

type Module struct {
	Handler *Handler
}

func NewModule(db *gorm.DB, authRepo *auth.Repository, bookingRepo *booking.Repository) *Module {
	repo    := NewRepository(db)
	service := NewService(repo, authRepo, bookingRepo)
	handler := NewHandler(service)

	return &Module{Handler: handler}
}