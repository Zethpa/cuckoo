package realtime

import (
	"net/http"
	"sync"
	"time"

	"cuckoo/backend/internal/auth"
	"cuckoo/backend/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Event struct {
	Type     string      `json:"type"`
	RoomCode string      `json:"roomCode"`
	Payload  interface{} `json:"payload"`
	SentAt   time.Time   `json:"sentAt"`
}

type Client struct {
	roomCode string
	userID   uint
	conn     *websocket.Conn
	send     chan Event
	hub      *Hub
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{clients: map[string]map[*Client]bool{}}
}

func (h *Hub) Broadcast(roomCode, eventType string, payload interface{}) {
	event := Event{Type: eventType, RoomCode: roomCode, Payload: payload, SentAt: time.Now().UTC()}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients[roomCode] {
		select {
		case client.send <- event:
		default:
		}
	}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.roomCode] == nil {
		h.clients[c.roomCode] = map[*Client]bool{}
	}
	h.clients[c.roomCode][c] = true
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.roomCode] != nil {
		delete(h.clients[c.roomCode], c)
		close(c.send)
	}
}

func (h *Hub) Handler(cfg config.Config) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return func(c *gin.Context) {
		claimsAny, ok := c.Get("claims")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		claims := claimsAny.(*auth.Claims)
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		client := &Client{
			roomCode: c.Param("code"),
			userID:   claims.UserID,
			conn:     conn,
			send:     make(chan Event, 16),
			hub:      h,
		}
		h.register(client)
		go client.writePump()
		client.readPump()
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister(c)
		_ = c.conn.Close()
	}()
	for {
		if _, _, err := c.conn.NextReader(); err != nil {
			return
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		_ = c.conn.Close()
	}()
	for event := range c.send {
		if err := c.conn.WriteJSON(event); err != nil {
			return
		}
	}
}
