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

	// Rebuild PlayerOrder from ALL players who have chips
	// This is important because folded players are removed from PlayerOrder during the round
	// but should be re-added if they still have chips for the next round

	// First, preserve the order of players still in PlayerOrder who have chips
	newPlayerOrder := []string{}
	inOrder := make(map[string]bool)

	for _, playerID := range room.PlayerOrder {
		p := room.Players[playerID]
		if p.Chips > 0 {
			p.Active = true
			newPlayerOrder = append(newPlayerOrder, playerID)
			inOrder[playerID] = true
		} else {
			p.Active = false
		}
	}

	// Then, add any players who were removed (folded) but still have chips
	// We need to iterate through ALL players, not just PlayerOrder
	for playerID, p := range room.Players {
		if p.Chips > 0 && !inOrder[playerID] {
			p.Active = true
			newPlayerOrder = append(newPlayerOrder, playerID)
			log.Printf("Re-adding player %s (had folded but still has %d chips)\n", p.Name, p.Chips)
		}
	}

	room.PlayerOrder = newPlayerOrder
	log.Printf("PlayerOrder rebuilt with %d players\n", len(room.PlayerOrder))

	// Rotate dealer button: move first player to the end
	if len(room.PlayerOrder) > 1 {
		room.PlayerOrder = append(room.PlayerOrder[1:], room.PlayerOrder[0])
		log.Println("Dealer button rotated. New order:", room.PlayerOrder)
	}
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
