package engine

import (
	"time"

	"vajra/internal/chess"
)

var UsePVS = true

var UseLMR = true

const (
	lmrFullMoves = 4

	lmrMinChildDepth = 2
)

func lmrReduction(index, childDepth int, quiet bool) int {
	if !UseLMR || !quiet || index < lmrFullMoves || childDepth < lmrMinChildDepth {
		return 0
	}

	r := 1

	if index >= 2*lmrFullMoves && childDepth >= 4 {
		r = 2
	}

	return min(r, childDepth-1)
}

func searchMove(
	p *chess.Position,
	move chess.Move,
	depth, ply, alpha, beta int,
	index int,
	deadline time.Time,
	state *SearchState,
) (int, bool) {

	reduce := 0

	if index >= lmrFullMoves {
		quiet := isQuiet(move)

		if quiet && !p.InCheck(p.SideToMove) {
			reduce = lmrReduction(index, depth, quiet)
		}
	}

	undo := p.MakeMove(move)
	defer p.UnmakeMove(move, undo)

	if reduce > 0 && p.InCheck(p.SideToMove) {
		reduce = 0
	}

	search := func(d, a, b int) (int, bool) {
		s, ok := negamax(p, d, ply, -b, -a, deadline, state)
		return -s, ok
	}

	if index == 0 || (!UsePVS && reduce == 0) {
		return search(depth, alpha, beta)
	}

	probeBeta := alpha + 1
	if !UsePVS {
		probeBeta = beta
	}

	score, ok := search(depth-reduce, alpha, probeBeta)
	if !ok {
		return 0, false
	}

	if reduce > 0 && score > alpha {
		score, ok = search(depth, alpha, probeBeta)
		if !ok {
			return 0, false
		}
	}

	if probeBeta < beta && score > alpha && score < beta {
		return search(depth, alpha, beta)
	}

	return score, true
}
