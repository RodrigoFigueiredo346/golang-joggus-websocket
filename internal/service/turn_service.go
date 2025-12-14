package service

import (
	"encoding/json"
	"log"

	"joggus/internal/model"
)

// ação do jogador
func PlayerAction(room *model.Room, playerID, action string, amount int) {
	log.Println("PlayerAction...")
	p, ok := room.Players[playerID]
	if !ok {
		log.Println("action error: player not found")
		return
	}
	if !p.Active {
		log.Printf("ignored action from %s (inactive)\n", p.Name)
		return
	}
	if room.CurrentPlayer != playerID {
		log.Printf("ignored action from %s (not their turn)\n", p.Name)
		return
	}
	log.Println("room.State: ", room.State)
	switch action {
	case "fold":
		p.Active = false
		log.Printf("%s folded\n", p.Name)
		// Do NOT remove from PlayerOrder, otherwise NextPlayer loses track of position
		// NextPlayer will skip inactive players automatically
	case "check":
		log.Println("room.CurrentBet: ", room.CurrentBet)
		if room.CurrentBet > 0 {
			// se há aposta, check é inválido → deve ser call
			diff := room.CurrentBet - p.CurrentBet
			if diff > 0 {
				if diff > p.Chips {
					diff = p.Chips
				}
				p.Chips -= diff
				p.CurrentBet += diff
				room.Pot += diff
				amount = diff
				p.TotalBet += diff
				log.Printf("%s auto-called (%d) instead of check\n", p.Name, diff)
			} else {
				amount = 0
				//ver se é necessario setar p.currentbet = 0?
			}
		}
		log.Println("p.CurrentBet", p.CurrentBet)
		// NextPlayer(room)
	case "call":
		log.Println("p.CurrentBet: ", p.CurrentBet)
		diff := room.CurrentBet - p.CurrentBet
		if diff > 0 {
			if diff > p.Chips {
				diff = p.Chips
			}
			p.Chips -= diff
			p.CurrentBet += diff
			room.Pot += diff
			amount = diff
			p.TotalBet += diff
		}
		log.Printf("%d called\n", p.CurrentBet)
		// NextPlayer(room)
	case "bet", "raise":
		if amount <= 0 {
			log.Printf("invalid bet amount from %s\n", p.Name)
			return
		}
		// Calculate the actual amount to add (total desired bet - current bet)
		diff := amount - p.CurrentBet
		if diff <= 0 {
			log.Printf("invalid raise amount from %s (must be higher than current bet)\n", p.Name)
			return
		}
		if diff > p.Chips {
			diff = p.Chips
		}
		p.Chips -= diff
		p.CurrentBet += diff
		p.TotalBet += diff
		room.Pot += diff
		if p.CurrentBet > room.CurrentBet {
			room.CurrentBet = p.CurrentBet
			// se houve aumento, outros devem agir novamente
			for _, other := range room.Players {
				if other.Active && other.ID != p.ID {
					other.HasActed = false
				}
			}
		}
		log.Printf("%s bet/raised to %d (added %d)\n", p.Name, p.CurrentBet, diff)
	case "allin":
		amount = p.Chips
		p.Chips = 0
		p.CurrentBet += amount
		room.Pot += amount
		if p.CurrentBet > room.CurrentBet {
			room.CurrentBet = p.CurrentBet
		}
		log.Printf("%s went all-in with %d\n", p.Name, amount)
		// NextPlayer(room)
	default:
		log.Printf("unknown action from %s: %s\n", p.Name, action)
		return
	}

	// marca como tendo agido
	p.HasActed = true

	nextPlayer := NextPlayer(room)

	// broadcast da ação
	msg := map[string]any{
		"method": "player_action",
		"params": map[string]any{
			"player_id":   p.ID,
			"action":      action,
			"amount":      amount,
			"pot":         room.Pot,
			"next_player": nextPlayer,
		},
	}
	// count := 0
	// for _, pl := range room.Players {
	// 	if pl.Connected {
	// 		count++
	// 	}
	// }
	// log.Printf("broadcasting (player action) action to %d connected players:", count)

	b, _ := json.Marshal(msg)
	room.Broadcast <- b

	// Check if only one active player remains (automatic win)
	activeCount := 0
	var lastActive *model.Player
	for _, player := range room.Players {
		if player.Active {
			activeCount++
			lastActive = player
		}
	}

	if activeCount == 1 && lastActive != nil {
		log.Printf("only one active player remains: %s wins automatically\n", lastActive.Name)
		// Award pot to the winner
		lastActive.Chips += room.Pot

		// Construir array de winners com informações detalhadas
		winners := []map[string]any{
			{
				"player_id": lastActive.ID,
				"hand":      "win by fold",
				"cards":     []model.Card{},
				"amount":    room.Pot,
			},
		}

		winMsg := map[string]any{
			"method": "showdown",
			"params": map[string]any{
				"winners": winners,
				"pot":     0,
			},
		}
		winBytes, _ := json.Marshal(winMsg)
		select {
		case room.Broadcast <- winBytes:
		default:
		}

		// Reset and start new round
		resetRoomShowDown(room)
		checkGameOverShowDown(room)
		// if !checkGameOverShowDown(room) {
		// 	go startNewRoundShowDown(room)
		// }
		return
	}

	// checa se rodada terminou
	if AllPlayersActed(room) {
		log.Printf("all active players acted in %s phase\n", room.State)
		// NextPhase(room)
		switch room.State {
		case model.StatePreflop:
			log.Println("calling the flop phase")
			DealFlop(room)
		case model.StateFlop:
			log.Println("calling the turn phase")
			DealTurn(room)
		case model.StateTurn:
			log.Println("calling the river phase")
			DealRiver(room)
		case model.StateRiver:
			log.Println("calling the showdown phase")
			Showdown(room)
			// case model.StateShowdown:

		}
		return
	}
}

func NextPlayer(room *model.Room) string {
	if len(room.PlayerOrder) == 0 {
		return ""
	}
	// Find current player index
	currentIdx := -1
	for i, id := range room.PlayerOrder {
		if id == room.CurrentPlayer {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 {
		// Current player not found, start from beginning
		currentIdx = 0
	}

	// Find next active player
	for i := 1; i <= len(room.PlayerOrder); i++ {
		nextIdx := (currentIdx + i) % len(room.PlayerOrder)
		nextID := room.PlayerOrder[nextIdx]
		if room.Players[nextID].Active {
			room.CurrentPlayer = nextID
			return room.CurrentPlayer
		}
	}

	// No active players found
	room.CurrentPlayer = ""
	return ""
}

func AllPlayersActed(room *model.Room) bool {
	activeCount := 0
	actedCount := 0

	for _, p := range room.Players {
		if p.Active {
			activeCount++
			if p.HasActed {
				actedCount++
			}
		}
	}
	return activeCount > 0 && actedCount == activeCount
}

// func StartTurn(room *model.Room) {
// 	log.Println("StartTurn...")
// 	if len(room.PlayerOrder) == 0 {
// 		for id := range room.Players {
// 			room.PlayerOrder = append(room.PlayerOrder, id)
// 		}
// 	}
// 	room.CurrentPlayer = room.PlayerOrder[0]
// 	broadcastTurn(room)
// }

// func broadcastTurn(room *model.Room) {
// 	msg := map[string]any{
// 		"method": "turn_start",
// 		"params": map[string]any{
// 			"player_id": room.CurrentPlayer,
// 			"player":    room.Players[room.CurrentPlayer].ID,
// 			"pot":       room.Pot,
// 		},
// 	}
// 	data, _ := json.Marshal(msg)
// 	log.Println("sending => msg:", string(data))
// 	b, _ := json.Marshal(msg)
// 	room.Broadcast <- b
// }
