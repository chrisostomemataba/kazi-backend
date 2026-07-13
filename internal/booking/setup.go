package booking

import (
	"kazi-backend/internal/auth"
	"kazi-backend/internal/common/storage"
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
	minioService *storage.MinIOService,
) *Module {
	repo := NewRepository(db)
	service := NewService(repo, authRepo, maidRepo, customerRepo, notificationService, paymentClient, minioService)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Repository: repo,
		Service:    service,
	}
}
