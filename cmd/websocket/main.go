package main

import (
	"log"

	"github.com/AlexsanderHamir/GoFlow/pkg/websocket"
)

func main() {
	server := websocket.NewServer()
	log.Fatal(server.Start(":8080"))
}