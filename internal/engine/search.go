package engine

import (
	"math/bits"
	"time"

	"vajra/internal/chess"
)

const (
	Infinity  = 999999
	MateScore = 100000
)

var Nodes uint64

var UseMoveOrdering = true

var TT = NewTranspositionTable(64)

var GameHistory []uint64

var history []uint64

const MaxPly = 128

type SearchState struct {
	killers KillerTable
	history HistoryTable
}

type KillerTable struct {
	moves [MaxPly][2]chess.Move
}

func allMoves(p *chess.Position) []chess.Move {
	var moves []chess.Move

	for sq := 0; sq < 64; sq++ {
		piece := p.Board.PieceAt(sq)

		if piece.Type != chess.Empty && piece.Color == p.SideToMove {
			moves = append(moves, p.LegalMoves(sq)...)
		}
	}

	return moves
}

func evalSide(p *chess.Position) int {
	score := Evaluate(p)

	if p.SideToMove == chess.Black {
		return -score
	}

	return score
}

func expired(deadline time.Time) bool {
	return !deadline.IsZero() && time.Now().After(deadline)
}

func isRepetition(hash uint64, halfmove int) bool {
	limit := max(len(history)-halfmove, 0)

	for i := len(history) - 2; i >= limit; i -= 2 {
		if history[i] == hash {
			return true
		}
	}

	return false
}

func insufficientMaterial(p *chess.Position) bool {
	b := &p.Board

	if b.WhitePawns|b.BlackPawns|b.WhiteRooks|b.BlackRooks|b.WhiteQueens|b.BlackQueens != 0 {
		return false
	}

	minors := b.WhiteKnights | b.BlackKnights | b.WhiteBishops | b.BlackBishops

	return bits.OnesCount64(uint64(minors)) <= 1
}

func startSearch(timeLimit time.Duration) time.Time {
	Nodes = 0
	TT.Clear()
	history = append(history[:0], GameHistory...)

	if timeLimit > 0 {
		return time.Now().Add(timeLimit)
	}

	return time.Time{}
}

func searchRoot(
	p *chess.Position,
	depth int,
	deadline time.Time,
	prevBest chess.Move,
	state *SearchState,
) (move chess.Move, found bool, completed bool) {

	moves := allMoves(p)

	if len(moves) == 0 {
		return move, false, true
	}

	if UseMoveOrdering {
		orderMovesHeuristic(moves, p.SideToMove, 0, state)
	}

	moves = putTTMoveFirst(moves, prevBest)

	hash := HashPosition(p)

	history = append(history, hash)
	defer func() { history = history[:len(history)-1] }()

	alpha := -Infinity

	for i, m := range moves {
		if expired(deadline) {
			return move, found, false
		}

		score, ok := searchMove(p, m, depth-1, 1, alpha, Infinity, i, deadline, state)

		if !ok {
			return move, found, false
		}

		if score > alpha {
			alpha = score
			move = m
			found = true
		}
	}

	TT.Store(hash, depth, scoreToTT(alpha, 0), TTExact, move)

	return move, found, true
}

func Search(
	p *chess.Position,
	depth int,
	timeLimit time.Duration,
) (chess.Move, bool) {

	deadline := startSearch(timeLimit)
	state := &SearchState{}

	move, found, _ := searchRoot(p, max(depth, 1), deadline, chess.Move{}, state)

	return move, found
}

func negamax(
	p *chess.Position,
	depth int,
	ply int,
	alpha int,
	beta int,
	deadline time.Time,
	state *SearchState,
) (int, bool) {

	Nodes++

	if expired(deadline) {
		return 0, false
	}

	hash := HashPosition(p)

	if p.HalfmoveClock >= 100 || isRepetition(hash, p.HalfmoveClock) {
		return 0, true
	}

	if p.HalfmoveClock == 0 && insufficientMaterial(p) {
		return 0, true
	}

	history = append(history, hash)
	defer func() { history = history[:len(history)-1] }()

	originalAlpha := alpha

	entry, hit := TT.Probe(hash)

	if hit && entry.Depth >= depth {
		score := scoreFromTT(entry.Score, ply)

		switch entry.Flag {
		case TTExact:
			return score, true
		case TTLowerBound:
			alpha = max(alpha, score)
		case TTUpperBound:
			beta = min(beta, score)
		}

		if alpha >= beta {
			return score, true
		}
	}

	moves := allMoves(p)

	if len(moves) == 0 {
		if p.InCheck(p.SideToMove) {
			return -MateScore + ply, true
		}

		return 0, true
	}

	if depth <= 0 {
		return quiescence(p, ply, alpha, beta, deadline)
	}

	if UseMoveOrdering {
		orderMovesHeuristic(moves, p.SideToMove, ply, state)

		if hit {
			moves = putTTMoveFirst(moves, entry.BestMove)
		}
	}

	bestScore := -Infinity
	var bestMove chess.Move

	for i, move := range moves {
		score, ok := searchMove(p, move, depth-1, ply+1, alpha, beta, i, deadline, state)

		if !ok {
			return 0, false
		}

		if score > bestScore {
			bestScore = score
			bestMove = move
		}

		alpha = max(alpha, bestScore)

		if alpha >= beta {
			if isQuiet(move) {
				if UseHistory {
					state.history.update(p.SideToMove, move, depth)
				}

				if UseKillers {
					state.killers.store(ply, move)
				}
			}

			break
		}
	}

	flag := TTExact

	if bestScore <= originalAlpha {
		flag = TTUpperBound
	} else if bestScore >= beta {
		flag = TTLowerBound
	}

	TT.Store(hash, depth, scoreToTT(bestScore, ply), flag, bestMove)

	return bestScore, true
}

func IterativeSearch(
	p *chess.Position,
	maxDepth int,
	timeLimit time.Duration,
) (chess.Move, bool) {

	deadline := startSearch(timeLimit)

	var best chess.Move
	found := false
	state := &SearchState{}

	for depth := 1; depth <= maxDepth; depth++ {
		move, ok, completed := searchRoot(p, depth, deadline, best, state)

		if ok {
			best, found = move, true
		}

		if !completed {
			break
		}
	}

	if !found {
		if moves := allMoves(p); len(moves) > 0 {
			return moves[0], true
		}
	}

	return best, found
}

func putTTMoveFirst(
	moves []chess.Move,
	ttMove chess.Move,
) []chess.Move {

	if ttMove.From == ttMove.To {
		return moves
	}

	for i := range moves {
		if sameMove(moves[i], ttMove) {
			move := moves[i]
			copy(moves[1:i+1], moves[:i])
			moves[0] = move
			break
		}
	}

	return moves
}

func sameMove(a, b chess.Move) bool {
	return a.From == b.From &&
		a.To == b.To &&
		a.Piece == b.Piece &&
		a.Promotion == b.Promotion
}

func scoreToTT(score int, ply int) int {
	if score > MateScore-1000 {
		return score + ply
	}

	if score < -MateScore+1000 {
		return score - ply
	}

	return score
}

func scoreFromTT(score int, ply int) int {
	if score > MateScore-1000 {
		return score - ply
	}

	if score < -MateScore+1000 {
		return score + ply
	}

	return score
}

func moveScore(move chess.Move) int {
	score := 0

	if move.Flags&chess.Capture != 0 {
		score += 10*pieceValue(move.Captured) - pieceValue(move.Piece)
	}

	if move.Flags&chess.Promotion != 0 {
		score += pieceValue(move.Promotion)
	}

	return score
}

func SquareName(sq int) string {
	files := "abcdefgh"
	ranks := "12345678"

	return string(files[sq%8]) + string(ranks[sq/8])
}
