package engine

import (
	"vajra/internal/chess"
)

const (
	Infinity  = 999999
	MateScore = 100000
)

var Nodes uint64
var UseMoveOrdering = true

func allMoves(p *chess.Position) []chess.Move {
	var moves []chess.Move
	for sq := 0; sq < 64; sq++ {
		if p.Board.PieceAt(sq).Color == p.SideToMove {
			moves = append(moves, p.LegalMoves(sq)...)
		}
	}
	return moves
}

// Evaluate is White-positive, so flip it for Black.
func evalSide(p *chess.Position) int {
	s := Evaluate(p)
	if p.SideToMove == chess.Black {
		return -s
	}
	return s
}

func Search(p *chess.Position, depth int) (chess.Move, bool) {
	Nodes = 0

	if depth < 1 {
		depth = 1
	}

	var bestMove chess.Move
	found := false
	alpha, beta := -Infinity, Infinity
	moves := allMoves(p)
	if UseMoveOrdering {
		orderMoves(moves)
	}
	for _, move := range moves {
		undo := p.MakeMove(move)
		score := -negamax(p, depth-1, 1, -beta, -alpha)
		p.UnmakeMove(move, undo)

		if !found || score > alpha {
			found = true
			bestMove = move
			if score > alpha {
				alpha = score
			}
		}
	}
	return bestMove, found
}

func negamax(p *chess.Position, depth, ply, alpha, beta int) int {
	Nodes++
	moves := allMoves(p)

	if len(moves) == 0 {
		if p.InCheck(p.SideToMove) {
			return -MateScore + ply
		}
		return 0
	}

	if depth <= 0 {
		return evalSide(p)
	}

	if UseMoveOrdering {
		orderMoves(moves)
	}
	for _, move := range moves {
		undo := p.MakeMove(move)

		score := -negamax(
			p,
			depth-1,
			ply+1,
			-beta,
			-alpha,
		)

		p.UnmakeMove(move, undo)

		if score >= beta {
			return beta
		}

		if score > alpha {
			alpha = score
		}
	}

	return alpha
}

func moveScore(move chess.Move) int {
	score := 0

	if move.Flags&chess.Capture != 0 {
		score += 10 * pieceValue(move.Captured)
		score -= pieceValue(move.Piece)
	}

	if move.Flags&chess.Promotion != 0 {
		score += pieceValue(move.Promotion)
	}

	return score
}
