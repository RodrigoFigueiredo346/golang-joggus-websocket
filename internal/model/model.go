package model

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type GameState string

const (
	StateWaiting  GameState = "waiting"
	StatePreflop  GameState = "preflop"
	StateFlop     GameState = "flop"
	StateTurn     GameState = "turn"
	StateRiver    GameState = "river"
	StateShowdown GameState = "showdown"
)

type Player struct {
	ID         string
	Name       string
	Conn       *websocket.Conn
	RoomID     string
	SendChan   chan []byte
	Chips      int
	Active     bool
	CurrentBet int
	TotalBet   int
	Connected  bool
	Hand       []Card
	HasActed   bool
}

type Room struct {
	ID                  string
	Players             map[string]*Player
	PlayerOrder         []string
	OriginalPlayerOrder []string // Master order, never changes after all players join
	CurrentPlayer       string
	Pot                 int
	CurrentBet          int
	MinBet              int
	RoundNumber         int
	GameOver            bool
	Broadcast           chan []byte
	Deck                *Deck
	CommunityCards      []Card
	State               GameState
}

type Server struct {
	Rooms map[string]*Room
	Mu    sync.RWMutex
}

type Card struct {
	Rank string `json:"rank"`
	Suit string `json:"suit"`
}

type Deck struct {
	Cards []Card
}

func NewDeck() *Deck {
	suits := []string{"♠", "♥", "♦", "♣"}
	ranks := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
	cards := make([]Card, 0, 52)
	for _, s := range suits {
		for _, r := range ranks {
			cards = append(cards, Card{Rank: r, Suit: s})
		}
	}
	return &Deck{Cards: cards}
}

func (d *Deck) Shuffle() {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(d.Cards), func(i, j int) { d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i] })
}

func (d *Deck) Draw(n int) []Card {
	if n > len(d.Cards) {
		n = len(d.Cards)
	}
	drawn := d.Cards[:n]
	d.Cards = d.Cards[n:]
	return drawn
}

func (r *Room) Run() {
	for msg := range r.Broadcast {
		for _, p := range r.Players {
			if p.Connected {
				select {
				case p.SendChan <- msg:
				default:
					log.Printf("player %s send buffer full, skipping", p.Name)
				}
			}
		}
	}
}
