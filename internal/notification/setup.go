package notification

import (
	wsHub "kazi-backend/internal/common/websocket"

	"gorm.io/gorm"
)

type Module struct {
	Handler          *Handler
	WebSocketHandler *WebSocketHandler
	Service          *Service
}

func NewModule(db *gorm.DB, hub *wsHub.Hub, jwtSecret string) *Module {
	service          := NewService(db)
	handler          := NewHandler(service)
	webSocketHandler := NewWebSocketHandler(hub, jwtSecret)

	return &Module{
		Handler:          handler,
		WebSocketHandler: webSocketHandler,
		Service:          service,
	}
}