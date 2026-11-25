package engine

import (
	"sync"
	"sync/atomic"
	"time"

	"vajra/internal/chess"
)

var (
	stopRequested atomic.Bool
	helpersStop   atomic.Bool

	Threads = 1
)

func RequestStop() { stopRequested.Store(true) }

func ClearStop() { stopRequested.Store(false) }

type SearchInfo struct {
	Depth    int
	Score    int
	Nodes    uint64
	HashFull int
	Elapsed  time.Duration
	Best     chess.Move
	PV       []chess.Move
}

func MateIn(score int) (int, bool) {
	switch {
	case score >= MateScore-1000:
		return (MateScore - score + 1) / 2, true
	case score <= -MateScore+1000:
		return -(MateScore + score) / 2, true
	}

	return 0, false
}

func IterativeSearchWithInfo(
	p *chess.Position,
	maxDepth int,
	timeLimit time.Duration,
	report func(SearchInfo),
) (chess.Move, bool) {
	defer func() { NodeLimit = 0 }()

	start := time.Now()
	deadline := startSearch(timeLimit)
	state := &SearchState{}

	defer startHelpers(p, maxDepth, deadline)()

	var best chess.Move

	found := false
	prevScore := 0

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

		if report != nil && ok {
			report(SearchInfo{
				Depth:    depth,
				Score:    score,
				Nodes:    Nodes,
				HashFull: TT.HashFull(),
				Elapsed:  time.Since(start),
				Best:     best,
				PV:       extractPV(p, best, max(depth, 1)),
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

func extractPV(p *chess.Position, rootMove chess.Move, maxLen int) []chess.Move {
	if rootMove.From == rootMove.To || maxLen <= 0 {
		return nil
	}
	pos := *p
	pv := make([]chess.Move, 0, maxLen)
	seen := make(map[uint64]bool, maxLen)

	cur := rootMove
	for i := 0; i < maxLen; i++ {
		legal := false
		for _, m := range allMoves(&pos) {
			if sameMove(m, cur) {
				cur = m
				legal = true
				break
			}
		}
		if !legal {
			break
		}
		pv = append(pv, cur)
		pos.MakeMove(cur)
		h := HashPosition(&pos)
		if seen[h] {
			break
		}
		seen[h] = true

		entry, hit := TT.Probe(transpositionKey(&pos))
		if !hit || entry.BestMove.From == entry.BestMove.To {
			break
		}
		cur = entry.BestMove
	}
	return pv
}
func mateProven(score, depth int) bool {
	if _, ok := MateIn(score); !ok {
		return false
	}

	if score < 0 {
		score = -score
	}

	return depth > MateScore-score
}

func startHelpers(p *chess.Position, maxDepth int, deadline time.Time) func() {
	helpersStop.Store(false)

	var wg sync.WaitGroup

	for id := 1; id < Threads; id++ {
		wg.Add(1)

		go func(id int, pos chess.Position) {
			defer wg.Done()

			path := append([]uint64(nil), GameHistory...)
			state := &SearchState{path: &path}

			var best chess.Move

			for depth := 1 + id%2; depth <= maxDepth; depth++ {
				move, ok, completed, _ := searchRootScored(&pos, depth, deadline, best, state)

				if ok {
					best = move
				}

				if !completed {
					return
				}
			}
		}(id, *p)
	}

	return func() {
		helpersStop.Store(true)
		wg.Wait()
		helpersStop.Store(false)
	}
}
