package websocket

import (
	"log"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type Client struct {
	UserID uuid.UUID
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	clients    map[uuid.UUID]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = client
			h.mu.Unlock()
			log.Printf("WebSocket client registered: %s", client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("WebSocket client unregistered: %s", client.UserID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client.UserID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) SendToUser(userID uuid.UUID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	if client, ok := h.clients[userID]; ok {
		select {
		case client.Send <- message:
			log.Printf("Message sent to user %s", userID)
		default:
			log.Printf("Failed to send message to user %s", userID)
		}
	}
}

func (h *Hub) ReadPump(client *Client) {
	defer func() {
		h.unregister <- client
		client.Conn.Close()
	}()

	for {
		_, _, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (h *Hub) WritePump(client *Client) {
	defer client.Conn.Close()

	for message := range client.Send {
		if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}