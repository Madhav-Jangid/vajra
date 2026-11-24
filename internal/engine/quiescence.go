package engine

import (
	"time"

	"vajra/internal/chess"
)

func quiescence(
	p *chess.Position,
	ply int,
	alpha int,
	beta int,
	deadline time.Time,
) (int, bool) {

	if expired(deadline) {
		return 0, false
	}

	Nodes++

	moves := allMoves(p)

	if len(moves) == 0 {
		if p.InCheck(p.SideToMove) {
			return -MateScore + ply, true
		}

		return 0, true
	}

	standPat := evalSide(p)

	if standPat >= beta {
		return beta, true
	}

	if standPat > alpha {
		alpha = standPat
	}
	if UseMoveOrdering {
		orderMoves(moves)
	}

	for _, move := range moves {
		if !isQuiescenceMove(move) {
			continue
		}

		if expired(deadline) {
			return 0, false
		}

		undo := p.MakeMove(move)

		score, completed := quiescence(
			p,
			ply+1,
			-beta,
			-alpha,
			deadline,
		)

		p.UnmakeMove(move, undo)

		if !completed {
			return 0, false
		}

		score = -score

		if score >= beta {
			return beta, true
		}

		if score > alpha {
			alpha = score
		}
	}

	return alpha, true
}

func isQuiescenceMove(move chess.Move) bool {
	if move.Flags&chess.Capture != 0 {
		return true
	}

	if move.Flags&chess.Promotion != 0 {
		return true
	}

	return false
}
