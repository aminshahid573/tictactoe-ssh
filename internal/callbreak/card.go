package callbreak

import (
	"fmt"
	"math/rand"
	"sort"
)

type Suit int

const (
	Clubs    Suit = 0
	Diamonds Suit = 1
	Hearts   Suit = 2
	Spades   Suit = 3
)

var (
	SuitSymbols = [4]string{"♣", "♦", "♥", "♠"}
	SuitNames   = [4]string{"Clubs", "Diamonds", "Hearts", "Spades"}
	SuitColors  = [4]string{"#44AAFF", "#FFD700", "#FF6B6B", "#50FA7B"}
)

func (s Suit) Symbol() string { return SuitSymbols[s] }
func (s Suit) Name() string   { return SuitNames[s] }
func (s Suit) Color() string  { return SuitColors[s] }

type Rank int

func (r Rank) String() string {
	switch r {
	case 11:
		return "J"
	case 12:
		return "Q"
	case 13:
		return "K"
	case 14:
		return "A"
	}
	return fmt.Sprintf("%d", int(r))
}

type Card struct {
	Suit Suit `json:"suit"`
	Rank Rank `json:"rank"`
}

func (c Card) String() string {
	if c.Rank == 0 {
		return ""
	}
	return c.Rank.String() + c.Suit.Symbol()
}

func (c Card) IsZero() bool { return c.Rank == 0 }

func NewDeck() []Card {
	var deck []Card
	for s := Clubs; s <= Spades; s++ {
		for r := Rank(2); r <= 14; r++ {
			deck = append(deck, Card{s, r})
		}
	}
	return deck
}

func ShuffleDeck(deck []Card) []Card {
	d := make([]Card, len(deck))
	copy(d, deck)
	rand.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
	return d
}

func SortHand(hand []Card) []Card {
	h := make([]Card, len(hand))
	copy(h, hand)
	sort.Slice(h, func(i, j int) bool {
		if h[i].Suit != h[j].Suit {
			return h[i].Suit > h[j].Suit
		}
		return h[i].Rank > h[j].Rank
	})
	return h
}
