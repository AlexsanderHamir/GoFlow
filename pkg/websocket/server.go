package websocket

import (
	"fmt"
	"log"
	"time"

	"github.com/AlexsanderHamir/ringbuffer"
	"github.com/AlexsanderHamir/ringbuffer/config"
	"github.com/gorilla/websocket"
)

type Server struct {
	client     *Client
	register   chan *Client
	unregister chan *Client
	done       chan struct{}
	ringBuffer *ringbuffer.RingBuffer[[]byte]
}

func NewServer() *Server {
	config := config.RingBufferConfig[[]byte]{
		Block: true,
	}

	ringBuffer, err := ringbuffer.NewWithConfig(1000, &config)
	if err != nil {
		log.Fatalf("Failed to create ring buffer: %v", err)
	}

	return &Server{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		done:       make(chan struct{}),
		ringBuffer: ringBuffer,
	}
}

func (h *Server) Run() {
	for {
		select {
		case <-h.done:
			if h.client != nil {
				close(h.client.send)
				h.client = nil
			}
			return
		case client := <-h.register:
			if h.client != nil {
				close(h.client.send)
			}
			h.client = client

			for {
				message, err := h.ringBuffer.GetOne()
				if err != nil {
					break
				}
				h.client.send <- message
			}

		case client := <-h.unregister:
			if h.client == client {
				close(h.client.send)
				h.client = nil
			}
		}
	}
}

// Shutdown gracefully shuts down the server
func (h *Server) Shutdown() {
	close(h.done)
}

// GetClient returns the current client if one is connected
func (h *Server) GetClient() *Client {
	return h.client
}

// GetActiveClient returns the connected client if it's active
func (h *Server) GetActiveClient() (*Client, error) {
	if h.client == nil {
		return nil, fmt.Errorf("no client connected")
	}

	// Try to send a ping message to verify connection
	err := h.client.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait))
	if err != nil {
		// If ping fails, unregister the client
		h.unregister <- h.client
		return nil, fmt.Errorf("client connection is not active")
	}
	return h.client, nil
}

// SendMessage sends a message to the connected client
func (h *Server) SendMessage(message []byte) error {
	if h.client == nil {
		log.Println("Client not connected, adding message to queue")
		h.ringBuffer.Write(message)
		log.Println("Message added to queue")
		return nil
	}

	select {
	case h.client.send <- message:
		return nil
	default:
		close(h.client.send)
		h.client = nil
		return fmt.Errorf("failed to send message to client")
	}
}
