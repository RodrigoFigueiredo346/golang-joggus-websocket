package controller

import (
	"log"

	"joggus/internal/service"
)

func StartGame(roomID string) {

	Server.Mu.RLock()
	room, exists := Server.Rooms[roomID]
	Server.Mu.RUnlock()
	if !exists {
		log.Println("start_game error: room not found")
		return
	}
	service.StartGame(room)
}

func DealFlop(roomID string) {
	Server.Mu.RLock()
	room, exists := Server.Rooms[roomID]
	Server.Mu.RUnlock()
	if !exists {
		log.Println("flop error: room not found")
		return
	}
	service.DealFlop(room)
}

func DealTurn(roomID string) {
	Server.Mu.RLock()
	room, exists := Server.Rooms[roomID]
	Server.Mu.RUnlock()
	if !exists {
		log.Println("turn error: room not found")
		return
	}
	service.DealTurn(room)
}

func DealRiver(roomID string) {
	Server.Mu.RLock()
	room, exists := Server.Rooms[roomID]
	Server.Mu.RUnlock()
	if !exists {
		log.Println("river error: room not found")
		return
	}
	service.DealRiver(room)
}
