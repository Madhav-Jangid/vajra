package engine

import (
	"time"

	"vajra/internal/chess"
)

// quiescence extends the search beyond the normal depth limit
// while the position contains tactical moves.
//
// For v1 we search:
//   - captures
//   - promotions
//
// Checks are intentionally not included yet.
func quiescence(
	p *chess.Position,
	ply int,
	alpha int,
	beta int,
	deadline time.Time,
) (int, bool) {

	// Always respect the search deadline.
	if expired(deadline) {
		return 0, false
	}

	Nodes++

	// Generate all legal moves so that checkmate/stalemate
	// are handled correctly.
	moves := allMoves(p)

	// If there are no legal moves, this is either checkmate
	// or stalemate.
	if len(moves) == 0 {
		if p.InCheck(p.SideToMove) {
			return -MateScore + ply, true
		}

		return 0, true
	}

	// Static evaluation of the current position.
	//
	// This is called the "stand pat" score. We assume that
	// the side to move can choose to make no tactical capture
	// and keep this evaluation.
	standPat := evalSide(p)

	// If the current position is already good enough to exceed
	// beta, the opponent would never allow us to continue here.
	if standPat >= beta {
		return beta, true
	}

	if standPat > alpha {
		alpha = standPat
	}
	if UseMoveOrdering {
		orderMoves(moves)
	}

	// Only search tactical moves.
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

// isQuiescenceMove determines whether a move is tactical
// enough to continue searching.
//
// v1:
//   - captures
//   - promotions
func isQuiescenceMove(move chess.Move) bool {
	if move.Flags&chess.Capture != 0 {
		return true
	}

	if move.Flags&chess.Promotion != 0 {
		return true
	}

	return false
}
