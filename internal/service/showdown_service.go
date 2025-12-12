package service

import (
	"encoding/json"
	"log"

	"joggus/internal/model"
)

func Showdown(room *model.Room) {
	if room.GameOver {
		return
	}
	room.State = model.StateShowdown

	active := []*model.Player{}
	for _, p := range room.Players {
		if p.Active && p.Chips >= 0 {
			active = append(active, p)
		}
	}
	if len(active) == 0 {
		return
	}

	scores := map[string]HandScore{}
	for _, p := range active {
		all := append(append([]model.Card{}, room.CommunityCards...), p.Hand...)
		scores[p.ID] = EvaluateBest5From7(all)
	}

	winnerIDs := GetWinnersByScore(scores)
	split := room.Pot / len(winnerIDs)

	// Atualizar chips dos vencedores
	for _, id := range winnerIDs {
		room.Players[id].Chips += split
	}

	/// Get winning hand details from the first winner (all winners have the same score)
	winningScore := scores[winnerIDs[0]]

	// Construir array de winners com informações detalhadas
	winners := []map[string]any{}
	for _, id := range winnerIDs {
		winner := map[string]any{
			"player_id": id,
			"hand":      winningScore.Rank.String(),
			"cards":     GetRelevantCards(winningScore),
			"amount":    split,
		}
		winners = append(winners, winner)
	}

	// Construir array de players com estado atualizado de todos os jogadores
	players := []map[string]any{}
	for _, p := range room.Players {
		player := map[string]any{
			"player_id": p.ID,
			"name":      p.Name,
			"chips":     p.Chips,
		}
		if p.Active {
			player["cards"] = p.Hand
		}
		players = append(players, player)
	}

	msg := map[string]any{
		"method": "showdown",
		"params": map[string]any{
			"winners": winners,
			"players": players,
			"pot":     0, // Pot é zerado após o showdown
		},
	}
	b, _ := json.Marshal(msg)
	select {
	case room.Broadcast <- b:
	default:
	}

	resetRoomShowDown(room)
	checkGameOverShowDown(room)
}

func resetRoomShowDown(room *model.Room) {
	log.Println("resetRoomShowDown...")
	room.Pot = 0
	room.CurrentBet = 0
	room.CommunityCards = []model.Card{}
	room.Deck = nil
	room.State = model.StateWaiting

	// Reset player state
	for _, p := range room.Players {
		p.CurrentBet = 0
		p.TotalBet = 0
		p.Hand = []model.Card{}
	}

	// Rebuild PlayerOrder using OriginalPlayerOrder as master reference
	// This preserves the original join order and ensures correct dealer rotation

	// 1. Identify who started the previous round (Dealer/Button)
	var lastStarterID string
	if len(room.PlayerOrder) > 0 {
		lastStarterID = room.PlayerOrder[0]
	}

	// 2. Build list of currently active players (chips > 0) in their original relative order
	activePlayers := []string{}
	for _, playerID := range room.OriginalPlayerOrder {
		p, exists := room.Players[playerID]
		if !exists {
			continue
		}
		if p.Chips > 0 {
			p.Active = true
			activePlayers = append(activePlayers, playerID)
		} else {
			p.Active = false
		}
	}

	if len(activePlayers) == 0 {
		log.Println("No active players left to rebuild order")
		return
	}

	// 3. Find the new starter (next active player after lastStarterID)
	var newStarterID string

	// Find index of lastStarter in OriginalPlayerOrder
	lastStarterIndex := -1
	for i, pid := range room.OriginalPlayerOrder {
		if pid == lastStarterID {
			lastStarterIndex = i
			break
		}
	}

	if lastStarterIndex == -1 {
		// Fallback: if last starter not found (shouldn't happen), use first active
		newStarterID = activePlayers[0]
	} else {
		// Search forward from lastStarterIndex + 1 to find the first active player
		found := false
		for i := 1; i <= len(room.OriginalPlayerOrder); i++ {
			idx := (lastStarterIndex + i) % len(room.OriginalPlayerOrder)
			candidateID := room.OriginalPlayerOrder[idx]

			// Check if this candidate is in activePlayers
			isActive := false
			for _, ap := range activePlayers {
				if ap == candidateID {
					isActive = true
					break
				}
			}

			if isActive {
				newStarterID = candidateID
				found = true
				break
			}
		}
		if !found {
			newStarterID = activePlayers[0]
		}
	}

	// 4. Rotate activePlayers so newStarterID is at index 0
	newPlayerOrder := []string{}
	startIndex := -1
	for i, pid := range activePlayers {
		if pid == newStarterID {
			startIndex = i
			break
		}
	}

	if startIndex != -1 {
		newPlayerOrder = append(newPlayerOrder, activePlayers[startIndex:]...)
		newPlayerOrder = append(newPlayerOrder, activePlayers[:startIndex]...)
	} else {
		newPlayerOrder = activePlayers
	}

	room.PlayerOrder = newPlayerOrder
	log.Printf("PlayerOrder rebuilt. Last Starter: %s, New Starter: %s, Order: %v\n",
		lastStarterID, newStarterID, room.PlayerOrder)
}

func startNewRoundShowDown(room *model.Room) {
	log.Println("startNewRoundShowDown...")
	room.RoundNumber++
	// msg := map[string]any{
	// 	"method": "new_round",
	// 	"params": map[string]any{
	// 		"round": room.RoundNumber,
	// 	},
	// }
	// b, _ := json.Marshal(msg)

	// select {
	// case room.Broadcast <- b:
	// default:
	// 	log.Println("new_round broadcast skipped (buffer full)")
	// }

	go StartGame(room)
}

func checkGameOverShowDown(room *model.Room) bool {
	log.Println("checkGameOverShowDown...")
	active := 0
	var lastPlayer *model.Player

	for _, p := range room.Players {
		if p.Chips > 0 {
			active++
			lastPlayer = p
		}
	}

	if active <= 1 {
		room.GameOver = true
		msg := map[string]any{
			"method": "game_over",
			"params": map[string]any{
				"winner": lastPlayer.Name,
				"chips":  lastPlayer.Chips,
				"rounds": room.RoundNumber,
			},
		}
		b, _ := json.Marshal(msg)

		select {
		case room.Broadcast <- b:
		default:
			log.Println("game_over broadcast skipped (buffer full)")
		}

		log.Printf("game over in room %s, winner: %s with %d chips after %d rounds\n",
			room.ID, lastPlayer.Name, lastPlayer.Chips, room.RoundNumber)
		return true
	}

	return false
}
