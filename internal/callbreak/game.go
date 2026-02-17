package callbreak

func Beats(challenger, currentWinner Card) bool {
	if challenger.IsZero() || currentWinner.IsZero() {
		return false
	}
	if challenger.Suit == Spades && currentWinner.Suit != Spades {
		return true
	}
	if challenger.Suit == currentWinner.Suit && challenger.Rank > currentWinner.Rank {
		return true
	}
	return false
}

func ValidPlays(hand []Card, leadCard Card) []bool {
	valid := make([]bool, len(hand))
	if leadCard.IsZero() {
		for i := range valid {
			valid[i] = true
		}
		return valid
	}
	hasSuit := false
	for _, c := range hand {
		if c.Suit == leadCard.Suit {
			hasSuit = true
			break
		}
	}
	for i, c := range hand {
		if hasSuit {
			valid[i] = c.Suit == leadCard.Suit
		} else {
			valid[i] = true
		}
	}
	return valid
}

func FindTrickWinner(tableCards [4]Card, tablePlayed [4]bool, leader int) int {
	winner := leader
	for i := 1; i < 4; i++ {
		p := (leader + i) % 4
		if tablePlayed[p] && Beats(tableCards[p], tableCards[winner]) {
			winner = p
		}
	}
	return winner
}

func CalcScore(bid, won int) float64 {
	if won >= bid {
		return float64(bid) + float64(won-bid)*0.1
	}
	return -float64(bid)
}

func CountSuit(hand []Card, suit Suit) int {
	n := 0
	for _, c := range hand {
		if c.Suit == suit {
			n++
		}
	}
	return n
}

func AIBid(hand []Card) int {
	bid := 0
	for _, c := range hand {
		if c.Suit == Spades {
			switch {
			case c.Rank >= 13:
				bid += 2
			case c.Rank >= 10:
				bid++
			case c.Rank >= 7 && CountSuit(hand, Spades) >= 4:
				bid++
			}
		} else {
			if c.Rank == 14 {
				bid++
			} else if c.Rank == 13 && CountSuit(hand, c.Suit) <= 2 {
				bid++
			}
		}
	}
	if bid < 1 {
		bid = 1
	}
	if bid > 8 {
		bid = 8
	}
	return bid
}

func AIPlay(hand []Card, tableCards [4]Card, tablePlayed [4]bool, leader int) int {
	leadCard := tableCards[leader]
	valid := ValidPlays(hand, leadCard)

	var validIdx []int
	for i, v := range valid {
		if v {
			validIdx = append(validIdx, i)
		}
	}
	if len(validIdx) == 0 {
		return 0
	}
	if len(validIdx) == 1 {
		return validIdx[0]
	}

	if leadCard.IsZero() {
		best := validIdx[0]
		for _, i := range validIdx {
			if hand[best].Suit == Spades && hand[i].Suit != Spades {
				best = i
			} else if hand[i].Suit == hand[best].Suit && hand[i].Rank > hand[best].Rank {
				best = i
			}
		}
		return best
	}

	winner := FindTrickWinner(tableCards, tablePlayed, leader)
	winnerCard := tableCards[winner]

	bestWin := -1
	for _, i := range validIdx {
		if Beats(hand[i], winnerCard) {
			if bestWin == -1 || hand[i].Rank < hand[bestWin].Rank {
				bestWin = i
			}
		}
	}
	if bestWin != -1 {
		return bestWin
	}

	lowest := validIdx[0]
	for _, i := range validIdx {
		if hand[i].Suit != Spades && hand[lowest].Suit == Spades {
			lowest = i
		} else if hand[i].Suit == hand[lowest].Suit && hand[i].Rank < hand[lowest].Rank {
			lowest = i
		}
	}
	return lowest
}
