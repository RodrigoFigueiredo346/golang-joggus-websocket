package service

import (
	"log"
	"sort"

	"joggus/internal/model"
)

type HandRank int

const (
	HighCard HandRank = iota
	OnePair
	TwoPair
	ThreeOfKind
	Straight
	Flush
	FullHouse
	FourOfKind
	StraightFlush
)

type EvaluatedHand struct {
	PlayerID string
	Rank     HandRank
	HighCard int
	Cards    []model.Card
}

var rankOrder = map[string]int{
	"2": 2, "3": 3, "4": 4, "5": 5, "6": 6,
	"7": 7, "8": 8, "9": 9, "10": 10,
	"J": 11, "Q": 12, "K": 13, "A": 14,
}

func EvaluateHand(cards []model.Card) (HandRank, int) {
	log.Println("EvaluateHand...")
	ranks := map[int]int{}
	suits := map[string]int{}
	values := []int{}

	for _, c := range cards {
		val := rankOrder[c.Rank]
		ranks[val]++
		suits[c.Suit]++
		values = append(values, val)
	}

	sort.Ints(values)
	isFlush := false
	for _, count := range suits {
		if count >= 5 {
			isFlush = true
			break
		}
	}

	isStraight := false
	for i := 0; i <= len(values)-5; i++ {
		if values[i+4]-values[i] == 4 &&
			values[i] != values[i+1] &&
			values[i+1] != values[i+2] {
			isStraight = true
			break
		}
	}

	four, three, pairs := false, false, 0
	for _, c := range ranks {
		switch c {
		case 4:
			four = true
		case 3:
			three = true
		case 2:
			pairs++
		}
	}

	switch {
	case isStraight && isFlush:
		return StraightFlush, values[len(values)-1]
	case four:
		return FourOfKind, values[len(values)-1]
	case three && pairs == 1:
		return FullHouse, values[len(values)-1]
	case isFlush:
		return Flush, values[len(values)-1]
	case isStraight:
		return Straight, values[len(values)-1]
	case three:
		return ThreeOfKind, values[len(values)-1]
	case pairs == 2:
		return TwoPair, values[len(values)-1]
	case pairs == 1:
		return OnePair, values[len(values)-1]
	default:
		return HighCard, values[len(values)-1]
	}
}

func CompareHands(h1, h2 EvaluatedHand) int {
	log.Println("CompareHands...")
	if h1.Rank != h2.Rank {
		return int(h1.Rank) - int(h2.Rank)
	}
	return h1.HighCard - h2.HighCard
}

func GetWinnersHand(hands []EvaluatedHand) []EvaluatedHand {
	log.Println("GetWinners...")
	if len(hands) == 0 {
		return nil
	}

	sort.Slice(hands, func(i, j int) bool {
		if hands[i].Rank == hands[j].Rank {
			return hands[i].HighCard > hands[j].HighCard
		}
		return hands[i].Rank > hands[j].Rank
	})

	top := hands[0]
	winners := []EvaluatedHand{top}

	for i := 1; i < len(hands); i++ {
		if hands[i].Rank == top.Rank && hands[i].HighCard == top.HighCard {
			winners = append(winners, hands[i])
		} else {
			break
		}
	}

	return winners
}
