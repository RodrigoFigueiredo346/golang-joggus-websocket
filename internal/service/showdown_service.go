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
	names := []string{}
	for _, id := range winnerIDs {
		room.Players[id].Chips += split
		names = append(names, room.Players[id].Name)
	}

	// Get winning hand details from the first winner (all winners have the same score)
	winningScore := scores[winnerIDs[0]]

	msg := map[string]any{
		"method": "showdown",
		"params": map[string]any{
			"winners": names,
			"amount":  room.Pot,
			"split":   split,
			"hand":    winningScore.Rank.String(),
			"cards":   GetRelevantCards(winningScore),
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

	// Rebuild PlayerOrder to include only players with chips
	newPlayerOrder := []string{}
	for _, p := range room.Players {
		p.CurrentBet = 0
		p.TotalBet = 0
		p.Hand = []model.Card{}
		if p.Chips > 0 {
			p.Active = true
			newPlayerOrder = append(newPlayerOrder, p.ID)
		} else {
			p.Active = false
		}
	}
	room.PlayerOrder = newPlayerOrder
	log.Printf("PlayerOrder rebuilt with %d players\n", len(room.PlayerOrder))
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
