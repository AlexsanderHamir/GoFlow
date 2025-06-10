package websocket

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"
)

var addr = flag.String("addr", ":8080", "http service address")

func InitializeServer(ctx context.Context) *Server {
	flag.Parse()
	server := NewServer()
	go server.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(server, w, r)
	})

	srv := &http.Server{
		Addr:    *addr,
		Handler: nil,
	}

	go func() {
		log.Println("Starting websocket server on port", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		log.Println("Shutting down websocket server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}

		server.Shutdown()
	}()

	return server
}
