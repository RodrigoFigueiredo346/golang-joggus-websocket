package controller

import (
	"log"

	"joggus/internal/service"
)

func Showdown(roomID string) {
	Server.Mu.RLock()
	room, exists := Server.Rooms[roomID]
	Server.Mu.RUnlock()
	if !exists {
		log.Println("showdown error: room not found")
		return
	}
	service.Showdown(room)
}
