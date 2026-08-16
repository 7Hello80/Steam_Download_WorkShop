package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev; restrict in production
	},
}

// Message types sent from server to client
const (
	MsgTaskUpdate    = "task_update"
	MsgQueueUpdate   = "queue_update"
	MsgPtyOutput     = "pty_output"
	MsgPtyOutputBatch = "pty_output_batch"
	MsgPtyPrompt     = "pty_prompt"
	MsgPtyInputAck   = "pty_input_ack"
	MsgPtyError      = "pty_error"
)

// Message types sent from client to server
const (
	MsgPtyInput  = "pty_input"
	MsgSubscribe = "subscribe"
)

// Message represents a WebSocket message.
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Client represents a single WebSocket connection.
type Client struct {
	conn   *websocket.Conn
	userID string
	send   chan []byte
	hub    *Hub
}

// Hub maintains the set of active clients grouped by user ID.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]bool // userID -> set of connections
	// Callbacks for handling client messages
	onPtyInput  func(userID, taskID, input string)
}

// NewHub creates a new WebSocket hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]bool),
	}
}

// OnPtyInput registers a callback for pty_input messages.
func (h *Hub) OnPtyInput(cb func(userID, taskID, input string)) {
	h.onPtyInput = cb
}

// HandleUpgrade handles WebSocket upgrade requests.
func (h *Hub) HandleUpgrade(w http.ResponseWriter, r *http.Request, userID string) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		conn:   conn,
		userID: userID,
		send:   make(chan []byte, 64),
		hub:    h,
	}

	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]bool)
	}
	h.clients[userID][client] = true
	h.mu.Unlock()

	go client.writePump()
	go client.readPump()

	log.Printf("WebSocket client connected: user=%s", userID)
	return nil
}

// SendToUser sends a message to all connections of a specific user.
func (h *Hub) SendToUser(userID string, msgType string, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("WebSocket marshal error: %v", err)
		return
	}

	msg := Message{
		Type: msgType,
		Data: dataBytes,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket message marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if userClients, ok := h.clients[userID]; ok {
		for client := range userClients {
			select {
			case client.send <- msgBytes:
			default:
				// Client buffer full, skip
				log.Printf("WebSocket client send buffer full for user=%s", userID)
			}
		}
	}
}

// SendToUserBatch sends batched PTY output lines to reduce mutex contention.
func (h *Hub) SendToUserBatch(userID string, msgType string, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("WebSocket marshal error: %v", err)
		return
	}

	msg := Message{
		Type: msgType,
		Data: dataBytes,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket message marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if userClients, ok := h.clients[userID]; ok {
		for client := range userClients {
			select {
			case client.send <- msgBytes:
			default:
				// Client buffer full, skip
				log.Printf("WebSocket client send buffer full for user=%s", userID)
			}
		}
	}
}

// SendPtyInputAck sends confirmation that 2FA input was received.
func (h *Hub) SendPtyInputAck(userID, taskID, status string) {
	h.SendToUser(userID, MsgPtyInputAck, map[string]interface{}{
		"task_id": taskID,
		"status":  status,
	})
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msgType string, data interface{}) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("WebSocket marshal error: %v", err)
		return
	}

	msg := Message{
		Type: msgType,
		Data: dataBytes,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket message marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, userClients := range h.clients {
		for client := range userClients {
			select {
			case client.send <- msgBytes:
			default:
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("WebSocket unmarshal error: %v", err)
			continue
		}

		switch msg.Type {
		case MsgPtyInput:
			var data struct {
				TaskID string `json:"task_id"`
				Input  string `json:"input"`
			}
			if err := json.Unmarshal(msg.Data, &data); err != nil {
				log.Printf("WebSocket pty_input parse error: %v", err)
				continue
			}
			if c.hub.onPtyInput != nil {
				c.hub.onPtyInput(c.userID, data.TaskID, data.Input)
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
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

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
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

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[c.userID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.clients, c.userID)
		}
	}
	close(c.send)
	log.Printf("WebSocket client disconnected: user=%s", c.userID)
}
