package elo

import "math"

const confidenceZ = 1.96

func Score(wins, draws, losses int) float64 {
	games := wins + draws + losses
	if games == 0 {
		return 0.5
	}

	return (float64(wins) + float64(draws)/2) / float64(games)
}

func Difference(score float64) float64 {
	switch {
	case score <= 0:
		return math.Inf(-1)
	case score >= 1:
		return math.Inf(1)
	}

	return -400 * math.Log10(1/score-1)
}

func Estimate(wins, draws, losses int) (diff, margin float64) {
	games := wins + draws + losses
	if games == 0 {
		return 0, math.Inf(1)
	}

	n := float64(games)
	p := Score(wins, draws, losses)

	variance := (float64(wins)*math.Pow(1-p, 2) + float64(draws)*math.Pow(0.5-p, 2) + float64(losses)*p*p) / n
	deviation := math.Sqrt(variance / n)

	diff = Difference(p)
	margin = (Difference(math.Min(p+confidenceZ*deviation, 1)) - Difference(math.Max(p-confidenceZ*deviation, 0))) / 2

	return diff, margin
}

func LikelihoodOfSuperiority(wins, losses int) float64 {
	if wins+losses == 0 {
		return 0.5
	}

	return 0.5 * (1 + math.Erf(float64(wins-losses)/math.Sqrt(2*float64(wins+losses))))
}

func Rating(anchor, diff float64) float64 {
	return anchor + diff
}
