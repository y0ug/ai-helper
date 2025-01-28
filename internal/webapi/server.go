package webapi

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/y0ug/ai-helper/internal/assistant"
	"github.com/y0ug/ai-helper/internal/assistant/eventbus"
)

type WebServer struct {
	assistant *assistant.AssistantOrchestrator
	eventBus  *eventbus.EventBus
	upgrader  websocket.Upgrader
	clients   sync.Map // thread-safe map for websocket clients
}

func NewWebServer(assistant *assistant.AssistantOrchestrator, bus *eventbus.EventBus) *WebServer {
	return &WebServer{
		assistant: assistant,
		eventBus:  bus,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Configure as needed
			},
		},
	}
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Status   string `json:"status"`
}

func (s *WebServer) Start(addr string) error {
	// Using Go 1.22 routing patterns
	mux := http.NewServeMux()

	// Chat endpoints
	mux.HandleFunc("POST /api/chat", s.handleChat)

	// File management
	mux.HandleFunc("POST /api/files", s.handleAddFiles)
	mux.HandleFunc("DELETE /api/files/{filename}", s.handleRemoveFile)
	mux.HandleFunc("GET /api/files", s.handleListFiles)

	// WebSocket endpoint
	mux.HandleFunc("GET /ws", s.handleWebSocket)

	// Start the server
	server := &http.Server{
		Addr:         addr,
		Handler:      s.withMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

func generateRequestID() string {
	return uuid.New().String()
}

func (s *WebServer) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Add request ID
		ctx := context.WithValue(r.Context(), "requestID", generateRequestID())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *WebServer) handleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create a channel to receive the response
	// responseChan := make(chan string, 1)

	// Subscribe to output events
	sub := s.eventBus.Subscribe(100)
	defer s.eventBus.Unsubscribe(sub)

	// Publish input event
	s.eventBus.Publish(eventbus.NewEvent(
		eventbus.EventInput,
		eventbus.UserInput{
			Source:  "api",
			Content: req.Message,
		},
	))

	// Wait for response with timeout
	select {
	case event := <-sub:
		if event.Type == eventbus.EventOutput {
			if output, ok := event.Payload.(string); ok {
				json.NewEncoder(w).Encode(ChatResponse{
					Response: output,
					Status:   s.assistant.GetStatus().Current(),
				})
				return
			}
		}
	case <-time.After(30 * time.Second):
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		return
	}
}

func (s *WebServer) handleAddFiles(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	fileNames := make([]string, 0, len(files))

	for _, fileHeader := range files {
		// Save file to temp location
		fileName := fileHeader.Filename
		fileNames = append(fileNames, fileName)
	}

	s.eventBus.Publish(eventbus.NewEvent(
		eventbus.EventAddFile,
		eventbus.FileOperation{
			Files: fileNames,
		},
	))

	w.WriteHeader(http.StatusOK)
}

func (s *WebServer) handleRemoveFile(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")

	s.eventBus.Publish(eventbus.NewEvent(
		eventbus.EventRemoveFile,
		eventbus.FileOperation{
			Files: []string{filename},
		},
	))

	w.WriteHeader(http.StatusOK)
}

func (s *WebServer) handleListFiles(w http.ResponseWriter, r *http.Request) {
	files := s.assistant.GetRM().GetFM().List(0)
	json.NewEncoder(w).Encode(files)
}

// WebSocket handling
type WSClient struct {
	conn   *websocket.Conn
	server *WebServer
	send   chan []byte
}

func (s *WebServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &WSClient{
		conn:   conn,
		server: s,
		send:   make(chan []byte, 256),
	}

	// Store client
	s.clients.Store(client, struct{}{})

	// Start client routines
	go client.writePump()
	go client.readPump()

	// Subscribe to events
	sub := s.eventBus.Subscribe(100)
	go func() {
		defer s.eventBus.Unsubscribe(sub)
		for event := range sub {
			// Convert event to JSON and send to client
			if data, err := json.Marshal(event); err == nil {
				client.send <- data
			}
		}
	}()
}

func (c *WSClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		c.server.clients.Delete(c)
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WSClient) readPump() {
	defer func() {
		c.conn.Close()
		c.server.clients.Delete(c)
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		// Handle incoming WebSocket messages
		var input struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}

		if err := json.Unmarshal(message, &input); err != nil {
			continue
		}

		switch input.Type {
		case "chat":
			var chatReq ChatRequest
			if err := json.Unmarshal(input.Payload, &chatReq); err != nil {
				continue
			}
			c.server.eventBus.Publish(eventbus.NewEvent(
				eventbus.EventInput,
				eventbus.UserInput{
					Source:  "websocket",
					Content: chatReq.Message,
				},
			))
		}
	}
}
