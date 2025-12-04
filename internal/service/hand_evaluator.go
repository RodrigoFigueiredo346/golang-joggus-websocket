package service

import (
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

type HandScore struct {
	Rank     HandRank
	Tiebreak []int
	Cards    []model.Card
}

type EvaluatedHand struct {
	PlayerID string
	Rank     HandRank
	HighCard int
	Cards    []model.Card
}

var rankOrder = map[string]int{
	"2": 2, "3": 3, "4": 4, "5": 5, "6": 6, "7": 7, "8": 8, "9": 9, "10": 10, "J": 11, "Q": 12, "K": 13, "A": 14,
}

func valueSlice(cs []model.Card) []int {
	v := make([]int, len(cs))
	for i, c := range cs {
		v[i] = rankOrder[c.Rank]
	}
	sort.Ints(v)
	return v
}

func uniqDesc(vals []int) []int {
	m := map[int]bool{}
	out := []int{}
	for i := len(vals) - 1; i >= 0; i-- {
		if !m[vals[i]] {
			m[vals[i]] = true
			out = append(out, vals[i])
		}
	}
	return out
}

func straightHigh(vals []int) (bool, int) {
	u := uniqDesc(vals)
	sort.Ints(u)
	// wheel
	hasA, has2, has3, has4, has5 := false, false, false, false, false
	for _, x := range u {
		if x == 14 {
			hasA = true
		}
		if x == 2 {
			has2 = true
		}
		if x == 3 {
			has3 = true
		}
		if x == 4 {
			has4 = true
		}
		if x == 5 {
			has5 = true
		}
	}
	if hasA && has2 && has3 && has4 && has5 {
		return true, 5
	}
	// normal
	if len(u) < 5 {
		return false, 0
	}
	for i := 0; i <= len(u)-5; i++ {
		if u[i+4]-u[i] == 4 {
			return true, u[i+4]
		}
	}
	return false, 0
}

func countRanks(cs []model.Card) (map[int]int, []int) {
	cnt := map[int]int{}
	for _, c := range cs {
		cnt[rankOrder[c.Rank]]++
	}
	keys := []int{}
	for k := range cnt {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if cnt[keys[i]] == cnt[keys[j]] {
			return keys[i] > keys[j]
		}
		return cnt[keys[i]] > cnt[keys[j]]
	})
	return cnt, keys
}

func pickFlush(cs []model.Card) ([]model.Card, bool) {
	suitCnt := map[string][]model.Card{}
	for _, c := range cs {
		suitCnt[c.Suit] = append(suitCnt[c.Suit], c)
	}
	for _, group := range suitCnt {
		if len(group) >= 5 {
			sort.Slice(group, func(i, j int) bool { return rankOrder[group[i].Rank] > rankOrder[group[j].Rank] })
			return group[:5], true
		}
	}
	return nil, false
}

func isStraightFlush(cs []model.Card) (bool, int) {
	suitCnt := map[string][]int{}
	for _, c := range cs {
		suitCnt[c.Suit] = append(suitCnt[c.Suit], rankOrder[c.Rank])
	}
	for _, vals := range suitCnt {
		sort.Ints(vals)
		ok, hi := straightHigh(vals)
		if ok {
			return true, hi
		}
	}
	return false, 0
}

func eval5(cs []model.Card) HandScore {
	vals := valueSlice(cs)
	cnt, order := countRanks(cs)
	okSF, hiSF := isStraightFlush(cs)
	if okSF {
		return HandScore{StraightFlush, []int{hiSF}, nil}
	}
	for _, k := range order {
		if cnt[k] == 4 {
			kickers := []int{}
			for _, v := range vals {
				if v != k {
					kickers = append(kickers, v)
				}
			}
			sort.Slice(kickers, func(i, j int) bool { return kickers[i] > kickers[j] })
			return HandScore{FourOfKind, []int{k, kickers[0]}, nil}
		}
	}
	three := -1
	pair := -1
	for _, k := range order {
		if cnt[k] == 3 && three == -1 {
			three = k
		}
	}
	for _, k := range order {
		if cnt[k] >= 2 && k != three && pair == -1 {
			pair = k
		}
	}
	if three != -1 && pair != -1 {
		return HandScore{FullHouse, []int{three, pair}, nil}
	}
	if f, ok := pickFlush(cs); ok {
		v := valueSlice(f)
		sort.Slice(v, func(i, j int) bool { return v[i] > v[j] })
		return HandScore{Flush, v, nil}
	}
	if ok, hi := straightHigh(vals); ok {
		return HandScore{Straight, []int{hi}, nil}
	}
	if three != -1 {
		kickers := []int{}
		for i := len(vals) - 1; i >= 0; i-- {
			if vals[i] != three {
				kickers = append(kickers, vals[i])
			}
		}
		return HandScore{ThreeOfKind, []int{three, kickers[0], kickers[1]}, nil}
	}
	pairs := []int{}
	for _, k := range order {
		if cnt[k] == 2 {
			pairs = append(pairs, k)
		}
	}
	if len(pairs) >= 2 {
		sort.Slice(pairs, func(i, j int) bool { return pairs[i] > pairs[j] })
		kicker := -1
		for i := len(vals) - 1; i >= 0; i-- {
			if vals[i] != pairs[0] && vals[i] != pairs[1] {
				kicker = vals[i]
				break
			}
		}
		return HandScore{TwoPair, []int{pairs[0], pairs[1], kicker}, nil}
	}
	if len(pairs) == 1 {
		kickers := []int{}
		for i := len(vals) - 1; i >= 0; i-- {
			if vals[i] != pairs[0] {
				kickers = append(kickers, vals[i])
			}
		}
		return HandScore{OnePair, []int{pairs[0], kickers[0], kickers[1], kickers[2]}, nil}
	}
	v := uniqDesc(vals)
	for len(v) < 5 {
		v = append(v, 0)
	}
	return HandScore{HighCard, v[:5], nil}
}

func compareScore(a, b HandScore) int {
	if a.Rank != b.Rank {
		if a.Rank > b.Rank {
			return 1
		}
		return -1
	}
	n := min(len(b.Tiebreak), len(a.Tiebreak))
	for i := range n {
		if a.Tiebreak[i] != b.Tiebreak[i] {
			if a.Tiebreak[i] > b.Tiebreak[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

func EvaluateBest5From7(cards []model.Card) HandScore {
	best := HandScore{Rank: HighCard, Tiebreak: []int{0}, Cards: nil}
	n := len(cards)
	for a := 0; a < n-4; a++ {
		for b := a + 1; b < n-3; b++ {
			for c := b + 1; c < n-2; c++ {
				for d := c + 1; d < n-1; d++ {
					for e := d + 1; e < n; e++ {
						hand := []model.Card{cards[a], cards[b], cards[c], cards[d], cards[e]}
						s := eval5(hand)
						s.Cards = hand
						if compareScore(s, best) > 0 {
							best = s
						}
					}
				}
			}
		}
	}
	return best
}

func GetWinnersByScore(scores map[string]HandScore) []string {
	type pair struct {
		id string
		s  HandScore
	}
	arr := []pair{}
	for id, s := range scores {
		arr = append(arr, pair{id, s})
	}
	sort.Slice(arr, func(i, j int) bool { return compareScore(arr[i].s, arr[j].s) > 0 })
	winners := []string{arr[0].id}
	for i := 1; i < len(arr); i++ {
		if compareScore(arr[i].s, arr[0].s) == 0 {
			winners = append(winners, arr[i].id)
		} else {
			break
		}
	}
	return winners
}

func (h HandRank) String() string {
	switch h {
	case HighCard:
		return "High Card"
	case OnePair:
		return "One Pair"
	case TwoPair:
		return "Two Pair"
	case ThreeOfKind:
		return "Three of a Kind"
	case Straight:
		return "Straight"
	case Flush:
		return "Flush"
	case FullHouse:
		return "Full House"
	case FourOfKind:
		return "Four of a Kind"
	case StraightFlush:
		return "Straight Flush"
	default:
		return "Unknown"
	}
}

// GetRelevantCards returns only the cards that form the actual hand
// For example, for "One Pair", it returns only the 2 cards of the pair
func GetRelevantCards(score HandScore) []model.Card {
	if len(score.Cards) == 0 {
		return []model.Card{}
	}

	switch score.Rank {
	case StraightFlush, Straight, Flush:
		// All 5 cards are part of the hand
		return score.Cards

	case FourOfKind:
		// Return the 4 cards that match
		if len(score.Tiebreak) == 0 {
			return score.Cards
		}
		fourValue := score.Tiebreak[0]
		result := []model.Card{}
		for _, card := range score.Cards {
			if rankOrder[card.Rank] == fourValue {
				result = append(result, card)
			}
		}
		return result

	case FullHouse:
		// Return the 3 cards of the three of a kind and the 2 cards of the pair
		if len(score.Tiebreak) < 2 {
			return score.Cards
		}
		threeValue := score.Tiebreak[0]
		pairValue := score.Tiebreak[1]
		result := []model.Card{}
		for _, card := range score.Cards {
			cardValue := rankOrder[card.Rank]
			if cardValue == threeValue || cardValue == pairValue {
				result = append(result, card)
			}
		}
		return result

	case ThreeOfKind:
		// Return the 3 cards that match
		if len(score.Tiebreak) == 0 {
			return score.Cards
		}
		threeValue := score.Tiebreak[0]
		result := []model.Card{}
		for _, card := range score.Cards {
			if rankOrder[card.Rank] == threeValue {
				result = append(result, card)
			}
		}
		return result

	case TwoPair:
		// Return the 4 cards that form the two pairs
		if len(score.Tiebreak) < 2 {
			return score.Cards
		}
		pair1Value := score.Tiebreak[0]
		pair2Value := score.Tiebreak[1]
		result := []model.Card{}
		for _, card := range score.Cards {
			cardValue := rankOrder[card.Rank]
			if cardValue == pair1Value || cardValue == pair2Value {
				result = append(result, card)
			}
		}
		return result

	case OnePair:
		// Return only the 2 cards that form the pair
		if len(score.Tiebreak) == 0 {
			return score.Cards
		}
		pairValue := score.Tiebreak[0]
		result := []model.Card{}
		for _, card := range score.Cards {
			if rankOrder[card.Rank] == pairValue {
				result = append(result, card)
			}
		}
		return result

	case HighCard:
		// For high card, return only the highest card
		if len(score.Cards) == 0 {
			return []model.Card{}
		}
		highest := score.Cards[0]
		highestValue := rankOrder[highest.Rank]
		for _, card := range score.Cards {
			if rankOrder[card.Rank] > highestValue {
				highest = card
				highestValue = rankOrder[card.Rank]
			}
		}
		return []model.Card{highest}

	default:
		return score.Cards
	}
}
