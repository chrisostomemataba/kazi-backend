package chat

import (
	"kazi-backend/internal/auth"
	"kazi-backend/internal/booking"
	wsHub "kazi-backend/internal/common/websocket"
	"kazi-backend/internal/notification"

	"gorm.io/gorm"
)

type Module struct {
	Handler *Handler
}

func NewModule(
	db *gorm.DB,
	bookingRepo *booking.Repository,
	authRepo *auth.Repository,
	hub *wsHub.Hub,
	notificationService *notification.Service,
) *Module {
	repo    := NewRepository(db)
	service := NewService(repo, bookingRepo, authRepo, hub, notificationService)
	handler := NewHandler(service)

	return &Module{Handler: handler}
}
