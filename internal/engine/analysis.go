package engine

import (
	"sync/atomic"
	"time"

	"vajra/internal/chess"
)

var stopRequested atomic.Bool

// RequestStop makes the running search return as soon as it can, keeping the
// best move found so far. It stays in effect until ClearStop is called.
func RequestStop() { stopRequested.Store(true) }

// ClearStop re-arms the engine for a new search.
func ClearStop() { stopRequested.Store(false) }

// SearchInfo describes one completed iteration of IterativeSearchWithInfo.
type SearchInfo struct {
	Depth   int
	Score   int
	Nodes   uint64
	Elapsed time.Duration
	Best    chess.Move
}

// MateIn returns how many moves away a mate score is (positive when the side to
// move delivers it, negative when it is mated) and whether the score is a mate.
func MateIn(score int) (int, bool) {
	switch {
	case score >= MateScore-1000:
		return (MateScore - score + 1) / 2, true
	case score <= -MateScore+1000:
		return -(MateScore + score) / 2, true
	}

	return 0, false
}

// IterativeSearchWithInfo behaves like IterativeSearch and calls report after
// every completed iteration. The score is from the side to move's point of view.
// A zero timeLimit means no time limit; RequestStop still ends the search.
func IterativeSearchWithInfo(
	p *chess.Position,
	maxDepth int,
	timeLimit time.Duration,
	report func(SearchInfo),
) (chess.Move, bool) {

	start := time.Now()
	deadline := startSearch(timeLimit)
	state := &SearchState{}

	var best chess.Move

	found := false

	for depth := 1; depth <= maxDepth; depth++ {
		move, ok, completed, score := searchRootScored(p, depth, deadline, best, state)

		if ok {
			best, found = move, true
		}

		if !completed {
			break
		}

		if report != nil && ok {
			report(SearchInfo{
				Depth:   depth,
				Score:   score,
				Nodes:   Nodes,
				Elapsed: time.Since(start),
				Best:    best,
			})
		}

		if ok && mateProven(score, depth) {
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

// mateProven reports whether a mate score has been searched one ply deeper than
// the mate itself, so searching further cannot change the answer.
func mateProven(score, depth int) bool {
	if _, ok := MateIn(score); !ok {
		return false
	}

	if score < 0 {
		score = -score
	}

	return depth > MateScore-score
}
