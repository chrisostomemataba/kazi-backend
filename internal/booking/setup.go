package booking

import (
	"kazi-backend/internal/auth"
	"kazi-backend/internal/customer"
	"kazi-backend/internal/maid"
	"kazi-backend/internal/notification"
	"kazi-backend/internal/payment"

	"gorm.io/gorm"
)

type Module struct {
	Handler    *Handler
	Repository *Repository
	Service    *Service
}

func NewModule(
	db *gorm.DB,
	authRepo *auth.Repository,
	maidRepo *maid.Repository,
	customerRepo *customer.Repository,
	notificationService *notification.Service,
	paymentClient *payment.PaymentClient,
) *Module {
	repo := NewRepository(db)
	service := NewService(repo, authRepo, maidRepo, customerRepo, notificationService, paymentClient)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Repository: repo,
		Service:    service,
	}
}
