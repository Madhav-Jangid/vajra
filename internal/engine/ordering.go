package engine

import (
	"sort"
	"vajra/internal/chess"
)

func orderMoves(moves []chess.Move) {
	sort.Slice(moves, func(i, j int) bool {
		return moveScore(moves[i]) > moveScore(moves[j])
	})
}
