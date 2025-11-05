package controller

import (
	"log"

	"joggus/internal/service"
)

func PlayerAction(roomID, playerID, action string, amount int) {
	Server.Mu.RLock()
	room, exists := Server.Rooms[roomID]
	Server.Mu.RUnlock()
	if !exists {
		log.Println("player_action error: room not found")
		return
	}
	service.PlayerAction(room, playerID, action, amount)
}
