package service

import (
	"encoding/json"
	"log"

	"joggus/internal/model"
)

func Showdown(room *model.Room) {
	log.Println("Showdown...")
	if room.GameOver {
		return
	}

	activePlayers := []*model.Player{}
	for _, p := range room.Players {
		if p.Active && p.Chips > 0 {
			activePlayers = append(activePlayers, p)
		}
	}

	if len(activePlayers) == 0 {
		log.Println("showdown error: no active players")
		return
	}

	evaluated := []EvaluatedHand{}
	for _, p := range activePlayers {
		playerCards := append(append([]model.Card{}, room.CommunityCards...), p.Hand...)
		rank, high := EvaluateHand(playerCards)
		evaluated = append(evaluated, EvaluatedHand{
			PlayerID: p.ID,
			Rank:     rank,
			HighCard: high,
		})
	}

	winners := GetWinnersHand(evaluated)

	split := room.Pot / len(winners)
	for _, w := range winners {
		room.Players[w.PlayerID].Chips += split
		log.Printf("%s wins %d chips (split)\n", room.Players[w.PlayerID].Name, split)
	}

	names := []string{}
	for _, w := range winners {
		names = append(names, room.Players[w.PlayerID].Name)
	}

	msg := map[string]any{
		"method": "showdown",
		"params": map[string]any{
			"winners": names,
			"amount":  room.Pot,
			"split":   split,
		},
	}
	b, _ := json.Marshal(msg)

	select {
	case room.Broadcast <- b:
	default:
		log.Println("showdown broadcast skipped (buffer full)")
	}

	resetRoomShowDown(room)

	if !checkGameOverShowDown(room) {
		go startNewRoundShowDown(room) // async to avoid blocking writes
	}
}

func resetRoomShowDown(room *model.Room) {
	log.Println("resetRoomShowDown...")
	room.Pot = 0
	room.CurrentBet = 0
	room.MinBet = room.MinBet + 10
	room.CommunityCards = []model.Card{}
	room.Deck = nil
	room.State = model.StateWaiting
	for _, p := range room.Players {
		p.CurrentBet = 0
		p.TotalBet = 0
		p.Active = p.Chips > 0
		p.Hand = []model.Card{}
	}
}

func startNewRoundShowDown(room *model.Room) {
	log.Println("startNewRoundShowDown...")
	room.RoundNumber++
	msg := map[string]any{
		"method": "new_round",
		"params": map[string]any{
			"round": room.RoundNumber,
		},
	}
	b, _ := json.Marshal(msg)

	select {
	case room.Broadcast <- b:
	default:
		log.Println("new_round broadcast skipped (buffer full)")
	}

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
