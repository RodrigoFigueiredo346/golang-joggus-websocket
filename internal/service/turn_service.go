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
		NextPlayer(room)
	case "check":
		log.Println("p.CurrentBet: ", p.CurrentBet)
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
				log.Printf("%s auto-called (%d) instead of check\n", p.Name, diff)
			} else {
				amount = 0
			}
		}
		log.Printf("%d checked\n", p.CurrentBet)
		NextPlayer(room)
	case "call":
		diff := room.CurrentBet - p.CurrentBet
		if diff > 0 {
			if diff > p.Chips {
				diff = p.Chips
			}
			p.Chips -= diff
			p.CurrentBet += diff
			room.Pot += diff
			amount = diff
		}
		log.Printf("%d called\n", p.CurrentBet)
		NextPlayer(room)
	case "bet", "raise":
		if amount <= 0 {
			log.Printf("invalid bet amount from %s\n", p.Name)
			return
		}
		if amount > p.Chips {
			amount = p.Chips
		}
		p.Chips -= amount
		p.CurrentBet += amount
		room.Pot += amount
		if p.CurrentBet > room.CurrentBet {
			room.CurrentBet = p.CurrentBet
			// se houve aumento, outros devem agir novamente
			for _, other := range room.Players {
				if other.Active && other.ID != p.ID {
					other.HasActed = false
				}
			}
		}
		log.Printf("%s bet %d\n", p.Name, amount)
		NextPlayer(room)
	case "allin":
		amount = p.Chips
		p.Chips = 0
		p.CurrentBet += amount
		room.Pot += amount
		if p.CurrentBet > room.CurrentBet {
			room.CurrentBet = p.CurrentBet
		}
		log.Printf("%s went all-in with %d\n", p.Name, amount)
		NextPlayer(room)
	default:
		log.Printf("unknown action from %s: %s\n", p.Name, action)
		return
	}

	// marca como tendo agido
	p.HasActed = true

	// broadcast da ação
	msg := map[string]any{
		"method": "player_action",
		"params": map[string]any{
			"player_id": p.ID,
			"action":    action,
			"amount":    amount,
			"pot":       room.Pot,
		},
	}
	count := 0
	for _, pl := range room.Players {
		if pl.Connected {
			count++
		}
	}
	log.Printf("broadcasting (player action) action to %d connected players:", count)

	b, _ := json.Marshal(msg)
	room.Broadcast <- b

	// checa se rodada terminou
	if AllPlayersActed(room) {
		log.Printf("all active players acted in %s phase\n", room.State)
		NextPhase(room)
		switch room.State {
		case model.StateFlop:
			log.Println("calling the flop phase")
			DealFlop(room)
		case model.StateTurn:
			log.Println("calling the turn phase")
			DealTurn(room)
		case model.StateRiver:
			log.Println("calling the river phase")
			DealRiver(room)
		case model.StateShowdown:
			log.Println("calling the showdown phase")
			Showdown(room)
		}
		return
	}

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
func StartTurn(room *model.Room) {
	log.Println("StartTurn...")
	if len(room.PlayerOrder) == 0 {
		for id := range room.Players {
			room.PlayerOrder = append(room.PlayerOrder, id)
		}
	}
	room.CurrentPlayer = room.PlayerOrder[0]
	broadcastTurn(room)
}

func NextPlayer(room *model.Room) {
	if len(room.PlayerOrder) == 0 {
		return
	}
	for i, id := range room.PlayerOrder {
		if id == room.CurrentPlayer {
			room.CurrentPlayer = room.PlayerOrder[(i+1)%len(room.PlayerOrder)]
			break
		}
	}
	broadcastTurn(room)
}

func broadcastTurn(room *model.Room) {
	msg := map[string]any{
		"method": "turn_start",
		"params": map[string]any{
			"player_id": room.CurrentPlayer,
			"player":    room.Players[room.CurrentPlayer].ID,
			"pot":       room.Pot,
		},
	}
	data, _ := json.Marshal(msg)
	log.Println("sending => msg:", string(data))
	b, _ := json.Marshal(msg)
	room.Broadcast <- b
}
