package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for development
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Server represents a WebSocket server
type Server struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

// NewServer creates a new WebSocket server
func NewServer() *Server {
	return &Server{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Start starts the WebSocket server
func (s *Server) Start(addr string) error {
	// Start the message handler
	go s.handleMessages()

	http.HandleFunc("/ws", s.handleWebSocket)
	log.Printf("WebSocket server starting on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// SendMessage sends a message to all connected clients
func (s *Server) SendMessage(message []byte) {
	s.broadcast <- message
}

// handleWebSocket handles incoming WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}

	// Set up ping handler
	conn.SetPingHandler(func(string) error {
		return conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(time.Second))
	})

	// Set up pong handler
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	// Set initial read deadline
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Register the new connection
	s.register <- conn

	// Start a goroutine to handle client disconnection
	go func() {
		defer func() {
			s.unregister <- conn
			conn.Close()
		}()

		// Keep the connection alive with ping/pong
		for {
			time.Sleep(30 * time.Second)
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second)); err != nil {
				return
			}
		}
	}()
}

// handleMessages processes incoming and outgoing messages
func (s *Server) handleMessages() {
	for {
		select {
		case client := <-s.register:
			s.clients[client] = true
			log.Printf("Client connected. Total clients: %d", len(s.clients))

		case client := <-s.unregister:
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				client.Close()
				log.Printf("Client disconnected. Total clients: %d", len(s.clients))
			}

		case message := <-s.broadcast:
			for client := range s.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Error writing message: %v", err)
					client.Close()
					s.unregister <- client
				}
			}
		}
	}
}
