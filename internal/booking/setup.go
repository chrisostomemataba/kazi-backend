package booking

import (
	"kazi-backend/internal/auth"
	"kazi-backend/internal/customer"
	"kazi-backend/internal/maid"
	"kazi-backend/internal/notification"

	"gorm.io/gorm"
)

type Module struct {
	Handler    *Handler
	Repository *Repository
}

func NewModule(
	db *gorm.DB,
	authRepo *auth.Repository,
	maidRepo *maid.Repository,
	customerRepo *customer.Repository,
	notificationService *notification.Service,
) *Module {
	repo    := NewRepository(db)
	service := NewService(repo, authRepo, maidRepo, customerRepo, notificationService)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Repository: repo,
	}
}