package controller

import (
	"encoding/json"
	"log"

	"joggus/internal/model"

	"github.com/gorilla/websocket"
)

var Server = &model.Server{
	Rooms: make(map[string]*model.Room),
}

func CreateRoom(conn *websocket.Conn, playerID, playerName string) {
	// roomID := uuid.NewString()[:5]
	roomID := "12345"

	player := &model.Player{
		ID:        playerID,
		Name:      playerName,
		Conn:      conn,
		RoomID:    roomID,
		SendChan:  make(chan []byte, 10),
		Connected: true,
		Active:    true,
		Chips:     1000,
	}

	go playerWriter(player)

	room := &model.Room{
		ID:             roomID,
		Players:        make(map[string]*model.Player),
		Broadcast:      make(chan []byte, 10),
		Deck:           nil,
		CommunityCards: []model.Card{},
		State:          model.StateWaiting,
		Pot:            0,
		CurrentBet:     0,
		MinBet:         10,
		RoundNumber:    1,
		PlayerOrder:    []string{playerID},
	}

	room.Players[playerID] = player
	Server.Mu.Lock()
	Server.Rooms[roomID] = room
	Server.Mu.Unlock()

	go room.Run()

	resp := map[string]any{
		"method": "room_created",
		"params": map[string]any{
			"room_id":   roomID,
			"player":    playerName,
			"player_id": playerID,
		},
	}
	b, _ := json.Marshal(resp)
	player.SendChan <- b
	// conn.WriteMessage(websocket.TextMessage, b)

	log.Printf("room created: %s by %s\n", roomID, playerName)
}

func playerWriter(p *model.Player) {
	log.Printf("writer started for %s", p.Name)
	for msg := range p.SendChan {
		// log.Printf("sending to %s: %s", p.Name, string(msg))
		if err := p.Conn.WriteMessage(1, msg); err != nil {
			log.Printf("connection lost: %s (%v)\n", p.Name, err)
			p.Connected = false
			p.Active = false
			return
		}
	}
}

func JoinRoom(conn *websocket.Conn, playerID, roomID, playerName string) {
	Server.Mu.Lock()
	room, exists := Server.Rooms[roomID]
	Server.Mu.Unlock()

	if !exists {
		resp := map[string]any{
			"method": "error",
			"params": map[string]string{"message": "room not found"},
		}
		b, _ := json.Marshal(resp)
		conn.WriteMessage(websocket.TextMessage, b)
		return
	}

	player := &model.Player{
		ID:        playerID,
		Name:      playerName,
		Conn:      conn,
		RoomID:    roomID,
		SendChan:  make(chan []byte, 10),
		Connected: true,
		Active:    true,
		Chips:     1000,
	}

	go playerWriter(player)

	// Adiciona o jogador na sala antes de criar o snapshot
	room.Players[playerID] = player

	// (se quiser, mantém ordem também)
	room.PlayerOrder = append(room.PlayerOrder, playerID)

	// Agora sim monta snapshot completo
	playersSnapshot := []map[string]any{}
	for _, p := range room.Players {
		playersSnapshot = append(playersSnapshot, map[string]any{
			"player_id":   p.ID,
			"player_name": p.Name,
			"chips":       p.Chips,
		})
	}

	resp := map[string]any{
		"method": "joined_room",
		"params": map[string]any{
			"room_id":   roomID,
			"player":    playerName,
			"player_id": playerID,
			"players":   playersSnapshot,
		},
	}
	b, _ := json.Marshal(resp)
	player.SendChan <- b

	msg := map[string]any{
		"method": "player_joined",
		"params": map[string]any{
			"player_name": playerName,
			"player_id":   playerID,
			"chips":       1000,
		},
	}
	m, _ := json.Marshal(msg)
	room.Broadcast <- m

	log.Printf("%s joined room %s (%d players total)\n", playerName, roomID, len(room.Players))
}

func ChatMessage(roomID, message string) {
	Server.Mu.RLock()
	room, exists := Server.Rooms[roomID]
	Server.Mu.RUnlock()
	if !exists {
		log.Println("chat error: room not found")
		return
	}

	msg := map[string]any{
		"method": "chat_broadcast",
		"params": map[string]string{"message": message},
	}
	b, _ := json.Marshal(msg)
	room.Broadcast <- b
}

func ReconnectPlayer(conn *websocket.Conn, playerID, roomID, playerName string) {
	Server.Mu.RLock()
	room, exists := Server.Rooms[roomID]
	Server.Mu.RUnlock()
	if !exists {
		resp := map[string]any{
			"method": "error",
			"params": map[string]string{"message": "room not found"},
		}
		b, _ := json.Marshal(resp)
		conn.WriteMessage(websocket.TextMessage, b)
		return
	}

	player, ok := room.Players[playerID]
	if !ok {
		resp := map[string]any{
			"method": "error",
			"params": map[string]string{"message": "player not found"},
		}
		b, _ := json.Marshal(resp)
		conn.WriteMessage(websocket.TextMessage, b)
		return
	}

	player.Conn = conn
	player.Connected = true
	player.Active = player.Chips > 0

	resp := map[string]any{
		"method": "reconnected",
		"params": map[string]any{
			"player": player.Name,
			"chips":  player.Chips,
			"pot":    room.Pot,
			"state":  room.State,
		},
	}
	b, _ := json.Marshal(resp)
	conn.WriteMessage(websocket.TextMessage, b)

	msg := map[string]any{
		"method": "player_reconnected",
		"params": map[string]string{"player": player.Name},
	}
	m, _ := json.Marshal(msg)
	room.Broadcast <- m

	log.Printf("%s reconnected to room %s\n", playerName, roomID)
}
