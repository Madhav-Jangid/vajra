package engine

import (
	"math/bits"
	"sync/atomic"
	"time"

	"vajra/internal/chess"
)

const (
	Infinity  = 999999
	MateScore = 100000
)

var Nodes uint64

var NodeLimit uint64

var UseMoveOrdering = true

var PreserveTT = false

var TT = NewTranspositionTable(64)

var GameHistory []uint64

var history []uint64

const MaxPly = 128

type SearchState struct {
	killers KillerTable
	history HistoryTable
	path    *[]uint64
}

func (s *SearchState) pathRef() *[]uint64 {
	if s.path != nil {
		return s.path
	}

	return &history
}

type KillerTable struct {
	moves [MaxPly][2]chess.Move
}

func allMoves(p *chess.Position) []chess.Move {
	return p.AllLegalMoves()
}

func evalSide(p *chess.Position) int {
	score := Evaluate(p)

	if p.SideToMove == chess.Black {
		return -score
	}

	return score
}

func expired(deadline time.Time) bool {
	if stopRequested.Load() || helpersStop.Load() {
		return true
	}

	if NodeLimit > 0 && atomic.LoadUint64(&Nodes) >= NodeLimit {
		return true
	}

	return !deadline.IsZero() && time.Now().After(deadline)
}

func isRepetition(hash uint64, halfmove int) bool {
	return repeatedIn(history, hash, halfmove)
}

func repeatedIn(path []uint64, hash uint64, halfmove int) bool {
	limit := max(len(path)-halfmove, 0)

	for i := len(path) - 2; i >= limit; i -= 2 {
		if path[i] == hash {
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

func transpositionKey(p *chess.Position) uint64 {
	hash := HashPosition(p)
	if p.HalfmoveClock <= 0 {
		return hash
	}

	x := uint64(p.HalfmoveClock) + 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	x ^= x >> 31
	return hash ^ x
}

func startSearch(timeLimit time.Duration) time.Time {
	atomic.StoreUint64(&Nodes, 0)
	if !PreserveTT {
		TT.Clear()
	}
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

	move, found, completed, _ = searchRootScored(p, depth, deadline, prevBest, state)

	return move, found, completed
}

func searchRootScored(
	p *chess.Position,
	depth int,
	deadline time.Time,
	prevBest chess.Move,
	state *SearchState,
) (move chess.Move, found bool, completed bool, score int) {
	return searchRootWindow(p, depth, -Infinity, Infinity, deadline, prevBest, state)
}

func searchRootWindow(
	p *chess.Position,
	depth int,
	alpha int,
	beta int,
	deadline time.Time,
	prevBest chess.Move,
	state *SearchState,
) (move chess.Move, found bool, completed bool, score int) {

	moves := allMoves(p)

	if len(moves) == 0 {
		return move, false, true, 0
	}

	if UseMoveOrdering {
		orderMovesHeuristic(moves, p.SideToMove, 0, state)
	}

	moves = putTTMoveFirst(moves, prevBest)

	hash := HashPosition(p)

	path := state.pathRef()
	*path = append(*path, hash)
	defer func() { *path = (*path)[:len(*path)-1] }()

	for i, m := range moves {
		if expired(deadline) {
			return move, found, false, alpha
		}

		child, ok := searchMove(p, m, depth-1, 1, alpha, beta, i, deadline, state)

		if !ok {
			return move, found, false, alpha
		}

		if child > alpha {
			alpha = child
			move = m
			found = true
			if alpha >= beta {
				return move, found, true, alpha
			}
		}
	}

	TT.Store(transpositionKey(p), depth, scoreToTT(alpha, 0), TTExact, move)

	return move, found, true, alpha
}

func searchRootAspiration(
	p *chess.Position,
	depth int,
	prevScore int,
	havePrev bool,
	deadline time.Time,
	prevBest chess.Move,
	state *SearchState,
) (move chess.Move, found bool, completed bool, score int) {
	if depth < 4 || !havePrev || absInt(prevScore) >= MateScore-1000 {
		return searchRootScored(p, depth, deadline, prevBest, state)
	}

	delta := 50
	alpha := max(prevScore-delta, -Infinity)
	beta := min(prevScore+delta, Infinity)

	for {
		move, f, ok, s := searchRootWindow(p, depth, alpha, beta, deadline, prevBest, state)
		if !ok {
			if f {
				return move, true, false, s
			}
			return prevBest, false, false, s
		}

		if s <= alpha && alpha > -Infinity {
			alpha = max(s-delta, -Infinity)
			delta *= 2
			if delta > 1000 {
				alpha = -Infinity
			}
			continue
		}
		if s >= beta && beta < Infinity {
			beta = min(s+delta, Infinity)
			delta *= 2
			if delta > 1000 {
				beta = Infinity
			}
			if f {
				prevBest = move
			}
			continue
		}
		return move, f, true, s
	}
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
	return negamaxSearch(p, depth, ply, alpha, beta, deadline, state, true)
}

func negamaxSearch(
	p *chess.Position,
	depth int,
	ply int,
	alpha int,
	beta int,
	deadline time.Time,
	state *SearchState,
	allowNull bool,
) (int, bool) {

	atomic.AddUint64(&Nodes, 1)

	if expired(deadline) {
		return 0, false
	}

	hash := HashPosition(p)

	if p.HalfmoveClock >= 100 {
		if len(allMoves(p)) == 0 && p.InCheck(p.SideToMove) {
			return -MateScore + ply, true
		}
		return 0, true
	}

	path := state.pathRef()

	if repeatedIn(*path, hash, p.HalfmoveClock) || insufficientMaterial(p) {
		return 0, true
	}

	*path = append(*path, hash)
	defer func() { *path = (*path)[:len(*path)-1] }()

	originalAlpha := alpha

	entry, hit := TT.Probe(transpositionKey(p))

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

	if depth <= 0 {
		return quiescence(p, ply, alpha, beta, deadline)
	}

	moves := allMoves(p)
	if len(moves) == 0 {
		if p.InCheck(p.SideToMove) {
			return -MateScore + ply, true
		}
		return 0, true
	}

	inCheck := p.InCheck(p.SideToMove)

	if UseLMR && !inCheck && depth <= 3 && absInt(beta) < MateScore-1000 && nonPawnPieceCount(p, p.SideToMove) > 0 {
		staticEval := evalSide(p)
		if staticEval-120*depth >= beta {
			return staticEval - 120*depth, true
		}
	}

	if UseLMR && allowNull && depth >= 3 && !inCheck &&
		nonPawnPieceCount(p, p.SideToMove) >= 2 && beta < MateScore-1000 && alpha > -MateScore+1000 &&
		evalSide(p) >= beta {
		r := 2
		if depth >= 6 {
			r = 3
		}
		savedEP := p.EnPassant
		p.EnPassant = -1
		if p.SideToMove == chess.White {
			p.SideToMove = chess.Black
		} else {
			p.SideToMove = chess.White
		}
		nullScore, ok := negamaxSearch(p, depth-1-r, ply+1, -beta, -beta+1, deadline, state, false)
		if p.SideToMove == chess.White {
			p.SideToMove = chess.Black
		} else {
			p.SideToMove = chess.White
		}
		p.EnPassant = savedEP
		if !ok {
			return 0, false
		}
		if -nullScore >= beta {
			return beta, true
		}
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

	TT.Store(transpositionKey(p), depth, scoreToTT(bestScore, ply), flag, bestMove)

	return bestScore, true
}

func IterativeSearch(
	p *chess.Position,
	maxDepth int,
	timeLimit time.Duration,
) (chess.Move, bool) {
	defer func() { NodeLimit = 0 }()

	start := time.Now()
	deadline := startSearch(timeLimit)

	var best chess.Move
	found := false
	prevScore := 0
	state := &SearchState{}

	for depth := 1; depth <= maxDepth; depth++ {
		if timeLimit > 0 && found && depth >= 2 && time.Since(start) >= (timeLimit*2)/3 {
			break
		}

		move, ok, completed, score := searchRootAspiration(p, depth, prevScore, found, deadline, best, state)

		if ok {
			best, found = move, true
			if completed {
				prevScore = score
			}
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
