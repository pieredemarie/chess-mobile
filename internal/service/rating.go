package service

import "math"

const eloK = 32

// EloRatings returns both ratings after one finished game.
func EloRatings(white, black int, result string) (int, int) {
	score := 0.5
	switch result {
	case "1-0":
		score = 1
	case "0-1":
		score = 0
	}
	expected := 1 / (1 + math.Pow(10, float64(black-white)/400))
	delta := int(math.Round(eloK * (score - expected)))
	nextWhite, nextBlack := white+delta, black-delta
	if nextWhite < 0 {
		nextWhite = 0
	}
	if nextBlack < 0 {
		nextBlack = 0
	}
	return nextWhite, nextBlack
}
