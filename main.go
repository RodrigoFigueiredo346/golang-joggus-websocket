package main

import (
	"log"
	"net/http"

	"joggus/internal/server"
)

func main() {
	http.HandleFunc("/ws", server.HandleWebSocket)
	addr := ":8080"
	log.Println("Joggus server running on:", addr)
	http.ListenAndServe(addr, nil)
}
