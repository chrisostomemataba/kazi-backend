package customer

import (
	"kazi-backend/internal/auth"

	"gorm.io/gorm"
)

type Module struct {
	Handler    *Handler
	Repository *Repository
}

func NewModule(db *gorm.DB, authRepo *auth.Repository) *Module {
	repo    := NewRepository(db)
	service := NewService(repo, authRepo)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Repository: repo,
	}
}