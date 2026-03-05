package notification

import (
	"encoding/json"
	"log"

	"kazi-backend/internal/common/util"
	wsHub "kazi-backend/internal/common/websocket"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type WebSocketHandler struct {
	hub       *wsHub.Hub
	jwtSecret string
}

func NewWebSocketHandler(hub *wsHub.Hub, jwtSecret string) *WebSocketHandler {
	return &WebSocketHandler{
		hub:       hub,
		jwtSecret: jwtSecret,
	}
}

func (h *WebSocketHandler) UpgradeMiddleware(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		token := c.Query("token")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token is required",
			})
		}

		claims, err := util.ValidateJWT(token, h.jwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		c.Locals("userID", claims.UserID)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func (h *WebSocketHandler) HandleConnection(c *websocket.Conn) {
	userID := c.Locals("userID").(uuid.UUID)

	client := &wsHub.Client{
		UserID: userID,
		Conn:   c,
		Send:   make(chan []byte, 256),
	}

	h.hub.Register <- client

	go h.hub.WritePump(client)
	h.hub.ReadPump(client)
}

func (h *WebSocketHandler) BroadcastNotification(userID uuid.UUID, notification *Notification) {
	message, err := json.Marshal(fiber.Map{
		"type": "notification",
		"data": notification,
	})
	if err != nil {
		log.Printf("Failed to marshal notification: %v", err)
		return
	}

	h.hub.SendToUser(userID, message)
}