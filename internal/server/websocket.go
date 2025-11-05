package server

import (
	"encoding/json"
	"log"
	"net/http"

	"joggus/internal/controller"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Message struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type CreateRoomParams struct {
	PlayerName string `json:"player_name"`
}

type JoinRoomParams struct {
	RoomID     string `json:"room_id"`
	PlayerName string `json:"player_name"`
}

type ChatParams struct {
	RoomID  string `json:"room_id"`
	Message string `json:"message"`
}

type RoomOnlyParams struct {
	RoomID string `json:"room_id"`
}

var p = 0

var first = false

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer conn.Close()

	// playerID := uuid.NewString()
	var playerID string

	switch p {
	case 0:
		playerID = "1111"
		p++
	case 1:
		playerID = "2222"
		p++
	case 2:
		playerID = "3333"
		p++
	case 3:
		playerID = "4444"
		p++
	}

	log.Println("playerID: ", playerID)
	log.Println("new connection:", conn.RemoteAddr())

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			break
		}

		var m Message
		if err := json.Unmarshal(raw, &m); err != nil {
			log.Println("invalid json:", err)
			continue
		}

		switch m.Method {
		case "create_room":
			var p CreateRoomParams
			if err := json.Unmarshal(m.Params, &p); err != nil {
				log.Println("invalid params:", err)
				continue
			}
			controller.CreateRoom(conn, playerID, p.PlayerName)

		case "join_room":
			var p JoinRoomParams
			if err := json.Unmarshal(m.Params, &p); err != nil {
				log.Println("invalid params:", err)
				continue
			}
			controller.JoinRoom(conn, playerID, p.RoomID, p.PlayerName)

		case "chat_message":
			var p ChatParams
			if err := json.Unmarshal(m.Params, &p); err != nil {
				log.Println("invalid chat params:", err)
				continue
			}
			controller.ChatMessage(p.RoomID, p.Message)

		case "start_game":
			var p RoomOnlyParams
			if err := json.Unmarshal(m.Params, &p); err != nil {
				log.Println("invalid params:", err)
				continue
			}
			controller.StartGame(p.RoomID)
		case "player_action":
			var p struct {
				RoomID   string `json:"room_id"`
				PlayerID string `json:"player_id"`
				Action   string `json:"action"`
				Amount   int    `json:"amount"`
			}
			if err := json.Unmarshal(m.Params, &p); err != nil {
				log.Println("invalid action params:", err)
				continue
			}
			controller.PlayerAction(p.RoomID, p.PlayerID, p.Action, p.Amount)
		case "reconnect":
			var p struct {
				PlayerID   string `json:"player_id"`
				RoomID     string `json:"room_id"`
				PlayerName string `json:"player_name"`
			}
			if err := json.Unmarshal(m.Params, &p); err != nil {
				log.Println("invalid params:", err)
				continue
			}
			controller.ReconnectPlayer(conn, p.PlayerID, p.RoomID, p.PlayerName)

		default:
			log.Println("unknown method:", m.Method)
		}
	}
}
