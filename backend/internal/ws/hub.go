package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type MessageType string

const (
	TypeTopologyUpdate MessageType = "topology_update"
	TypeContainerAdd   MessageType = "container_add"
	TypeContainerDel   MessageType = "container_del"
	TypeContainerUpd   MessageType = "container_update"
	TypeNetworkAdd     MessageType = "network_add"
	TypeNetworkDel     MessageType = "network_del"
	TypeStatusChange   MessageType = "status_change"
)

type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

type Client struct {
	conn         *websocket.Conn
	orgID        uuid.UUID
	connectionID uuid.UUID
	send         chan []byte
	hub          *Hub
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *BroadcastMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *slog.Logger
}

type BroadcastMessage struct {
	OrgID        uuid.UUID
	ConnectionID uuid.UUID
	Message      *Message
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *BroadcastMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Info("websocket client connected",
				"org_id", client.orgID,
				"connection_id", client.connectionID,
			)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Info("websocket client disconnected",
				"org_id", client.orgID,
				"connection_id", client.connectionID,
			)

		case msg := <-h.broadcast:
			data, err := json.Marshal(msg.Message)
			if err != nil {
				h.logger.Error("marshaling message", "error", err)
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				if client.orgID == msg.OrgID &&
					(msg.ConnectionID == uuid.Nil || client.connectionID == msg.ConnectionID) {
					select {
					case client.send <- data:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(orgID, connectionID uuid.UUID, msg *Message) {
	h.broadcast <- &BroadcastMessage{
		OrgID:        orgID,
		ConnectionID: connectionID,
		Message:      msg,
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, orgID, connectionID uuid.UUID) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("upgrading websocket", "error", err)
		return
	}

	client := &Client{
		conn:         conn,
		orgID:        orgID,
		connectionID: connectionID,
		send:         make(chan []byte, 256),
		hub:          h,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Error("websocket read error", "error", err)
			}
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
