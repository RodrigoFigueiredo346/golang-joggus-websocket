package service

import (
	"encoding/json"
	"log"

	"joggus/internal/model"
)

func StartGame(room *model.Room) {
	log.Println("StartGame...")

	resetAct(room)

	if len(room.Players) < 2 {
		log.Println("start_game error: not enough players")
		return
	}
	if room.State != model.StateWaiting {
		log.Println("start_game error: game already in progress")
		return
	}

	room.Deck = model.NewDeck()
	room.Deck.Shuffle()
	room.CommunityCards = []model.Card{}
	room.State = model.StatePreflop
	// RoundNumber already initialized to 0 at room creation
	// MinBet already set to initial value (10) at room creation
	log.Printf("Starting first game, MinBet: %d\n", room.MinBet)

	// for _, p := range room.Players {
	// 	p.Chips = 1000
	// 	p.Active = true
	// }

	sb, bb := &model.Player{}, &model.Player{}

	if len(room.PlayerOrder) >= 2 {
		sb = room.Players[room.PlayerOrder[len(room.PlayerOrder)-2]]
		bb = room.Players[room.PlayerOrder[len(room.PlayerOrder)-1]]
		log.Println("len player order: ", len(room.PlayerOrder))

		sbBet := room.MinBet / 2
		bbBet := room.MinBet
		sb.Chips -= sbBet
		sb.CurrentBet = sbBet

		bb.Chips -= bbBet
		bb.CurrentBet = bbBet

		room.Pot += sbBet + bbBet
		room.CurrentBet = bbBet
		room.MinBet = bbBet

		log.Printf("blinds set: %s(SB=%d), %s(BB=%d)\n", sb.Name, sbBet, bb.Name, bbBet)
	}

	// if len(room.PlayerOrder) >= 3 {
	// 	room.CurrentPlayer = room.PlayerOrder[2%len(room.PlayerOrder)]
	// } else {
	// 	room.CurrentPlayer = room.PlayerOrder[0]
	// }
	room.CurrentPlayer = room.PlayerOrder[0]

	playersInfo := []map[string]any{}
	for _, pl := range room.Players {
		info := map[string]any{
			// "id":    pl.ID,
			"name":      pl.Name,
			"chips":     pl.Chips,
			"player_id": pl.ID,
		}
		if pl.ID == sb.ID {
			info["blind"] = "sb"
		}
		if pl.ID == bb.ID {
			info["blind"] = "bb"
		}
		playersInfo = append(playersInfo, info)
	}
	for _, player := range room.Players {
		hand := room.Deck.Draw(2)
		player.Hand = hand
		// broadcast de início de jogo
		msg := map[string]any{
			"method": "game_started",
			"params": map[string]any{
				"room_id":        room.ID,
				"players":        playersInfo,
				"your_hand":      player.Hand,
				"pot":            room.Pot,
				"current_player": room.Players[room.CurrentPlayer].Name,
				"min_bet":        room.MinBet,
			},
		}
		bytes, _ := json.Marshal(msg)

		player.SendChan <- bytes

	}

	// count := 0
	// for _, pl := range room.Players {
	// 	if pl.Connected {
	// 		count++
	// 	}
	// }

	log.Printf("game started in room %s with %d players\n", room.ID, len(room.Players))
}

func StartNextRound(room *model.Room) {
	log.Println("StartNextRound...")

	if len(room.Players) < 2 {
		log.Println("start_next_round error: not enough players")
		return
	}
	if room.State != model.StateWaiting {
		log.Println("start_next_round error: game already in progress")
		return
	}

	// Increment round number
	room.RoundNumber++

	// Increment MinBet every 3 rounds
	if room.RoundNumber > 0 && room.RoundNumber%3 == 0 {
		room.MinBet = room.MinBet + 10
	}
	log.Printf("Round %d, MinBet: %d\n", room.RoundNumber, room.MinBet)

	// Create new deck and shuffle
	room.Deck = model.NewDeck()
	room.Deck.Shuffle()
	room.CommunityCards = []model.Card{}
	room.State = model.StatePreflop

	resetAct(room)

	// Set small blind and big blind
	sb, bb := &model.Player{}, &model.Player{}
	if len(room.PlayerOrder) >= 2 {
		sb = room.Players[room.PlayerOrder[len(room.PlayerOrder)-2]]
		bb = room.Players[room.PlayerOrder[len(room.PlayerOrder)-1]]

		sbBet := room.MinBet / 2
		bbBet := room.MinBet
		sb.Chips -= sbBet
		sb.CurrentBet = sbBet

		bb.Chips -= bbBet
		bb.CurrentBet = bbBet

		room.Pot += sbBet + bbBet
		room.CurrentBet = bbBet

		log.Printf("blinds set: %s(SB=%d), %s(BB=%d)\n", sb.Name, sbBet, bb.Name, bbBet)
	}

	room.CurrentPlayer = room.PlayerOrder[0]

	// Prepare player information
	playersInfo := []map[string]any{}
	for _, pl := range room.Players {
		info := map[string]any{
			"name":      pl.Name,
			"chips":     pl.Chips,
			"player_id": pl.ID,
		}
		if pl.ID == sb.ID {
			info["blind"] = "sb"
		}
		if pl.ID == bb.ID {
			info["blind"] = "bb"
		}
		playersInfo = append(playersInfo, info)
	}

	// Deal cards and send game_started to each player
	for _, player := range room.Players {
		hand := room.Deck.Draw(2)
		player.Hand = hand

		msg := map[string]any{
			"method": "game_started",
			"params": map[string]any{
				"room_id":        room.ID,
				"players":        playersInfo,
				"your_hand":      player.Hand,
				"pot":            room.Pot,
				"current_player": room.Players[room.CurrentPlayer].Name,
				"min_bet":        room.MinBet,
			},
		}
		bytes, _ := json.Marshal(msg)
		player.SendChan <- bytes
	}

	log.Printf("next round started in room %s with %d players\n", room.ID, len(room.Players))
}

func DealFlop(room *model.Room) {
	log.Println("DealFlop...")
	resetAct(room)
	if room.State != model.StatePreflop {
		log.Println("flop error: wrong state, current:", room.State)
		return
	}
	room.State = model.StateFlop

	cards := room.Deck.Draw(3)
	room.CommunityCards = append(room.CommunityCards, cards...)

	resp := map[string]any{
		"method": "deal_flop",
		"params": cards,
	}
	b, _ := json.Marshal(resp)
	room.Broadcast <- b

	log.Printf("flop dealt in room %s\n", room.ID)
	// NextPhase(room)
}

func DealTurn(room *model.Room) {
	log.Println("DealTurn...")
	resetAct(room)
	if room.State != model.StateFlop {
		log.Println("turn error: wrong state, current:", room.State)
		return
	}
	room.State = model.StateTurn

	card := room.Deck.Draw(1)
	room.CommunityCards = append(room.CommunityCards, card...)

	resp := map[string]any{
		"method": "deal_turn",
		"params": card,
	}
	b, _ := json.Marshal(resp)
	room.Broadcast <- b

	log.Printf("turn dealt in room %s\n", room.ID)
	// NextPhase(room)
}

func DealRiver(room *model.Room) {
	log.Println("DealRiver...")
	resetAct(room)
	if room.State != model.StateTurn {
		log.Println("river error: wrong state, current:", room.State)
		return
	}
	room.State = model.StateRiver

	card := room.Deck.Draw(1)
	room.CommunityCards = append(room.CommunityCards, card...)

	resp := map[string]any{
		"method": "deal_river",
		"params": card,
	}
	b, _ := json.Marshal(resp)
	room.Broadcast <- b

	log.Printf("river dealt in room %s\n", room.ID)
	// NextPhase(room)

}

// func NextPhase(room *model.Room) {
// 	log.Println("TROCANDO DE FASE:")
// 	log.Println("fase atual: ", room.State)
// 	switch room.State {
// 	case model.StateWaiting:
// 		room.State = model.StatePreflop
// 	case model.StatePreflop:
// 		room.State = model.StateFlop
// 	case model.StateFlop:
// 		room.State = model.StateTurn
// 	case model.StateTurn:
// 		room.State = model.StateRiver
// 	case model.StateRiver:
// 		room.State = model.StateShowdown
// 	case model.StateShowdown:
// 		room.State = model.StateWaiting
// 	}
// 	log.Println("fase nova: ", room.State)

// }

func resetAct(room *model.Room) {
	for _, p := range room.Players {
		if p.Active {
			p.HasActed = false
			p.CurrentBet = 0
		}
	}
	room.CurrentBet = 0
}
