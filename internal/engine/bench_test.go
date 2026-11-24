package engine

import (
	"testing"
	"time"

	"vajra/internal/chess"
)

// The search has no hit or cutoff counters, so two of the requested metrics are
// measured indirectly:
//
//	TT effectiveness   nodes with a normal table vs. nodes with a one-slot table
//	pruning ratio      nodes searched vs. the full move tree (perft) at the same depth
//
// Exact TT hit rate and beta-cutoff rate need counters inside search.go and
// transposition.go.

var benchPositions = []struct{ name, fen string }{
	{"start", chess.StartingFEN},
	{"middlegame", "r1bq1rk1/ppp2ppp/2np1n2/2b1p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 7"},
	{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"},
	{"endgame", "8/5pk1/6p1/2P5/8/1P4P1/5PK1/8 w - - 0 1"},
	{"tactical", "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1"},
}

type searchStats struct {
	nodes        uint64
	elapsed      time.Duration
	depthReached int
	ttFill       float64
	move         chess.Move
}

func (s searchStats) nps() float64 {
	if s.elapsed <= 0 {
		return 0
	}

	return float64(s.nodes) / s.elapsed.Seconds()
}

func ttOccupancy() float64 {
	used := 0

	for _, e := range TT.entries {
		if e.Valid {
			used++
		}
	}

	return float64(used) / float64(len(TT.entries))
}

// runSearch is IterativeSearch with the bookkeeping a benchmark needs: it
// records how deep the search completed and how full the table became.
func runSearch(p chess.Position, maxDepth int, limit time.Duration) searchStats {
	start := time.Now()
	deadline := startSearch(limit)
	state := &SearchState{}

	var best chess.Move

	reached := 0

	for depth := 1; depth <= maxDepth; depth++ {
		move, ok, completed := searchRoot(&p, depth, deadline, best, state)

		if ok {
			best = move
		}

		if !completed {
			break
		}

		reached = depth
	}

	return searchStats{
		nodes:        Nodes,
		elapsed:      time.Since(start),
		depthReached: reached,
		ttFill:       ttOccupancy(),
		move:         best,
	}
}

func perftCount(p *chess.Position, depth int) uint64 {
	if depth == 0 {
		return 1
	}

	var n uint64

	for _, m := range allMoves(p) {
		undo := p.MakeMove(m)
		n += perftCount(p, depth-1)
		p.UnmakeMove(m, undo)
	}

	return n
}

func withOneSlotTable(f func()) {
	saved := TT

	TT = &TranspositionTable{entries: make([]TTEntry, 1)}

	defer func() { TT = saved }()

	f()
}

func benchParse(tb testing.TB, fen string) chess.Position {
	tb.Helper()

	p, err := chess.ParseFEN(fen)
	if err != nil {
		tb.Fatalf("ParseFEN(%q): %v", fen, err)
	}

	return p
}

// BenchmarkSearchFixedDepth searches each position to depth 4 and reports the
// work done per search.
func BenchmarkSearchFixedDepth(b *testing.B) {
	for _, tc := range benchPositions {
		b.Run(tc.name, func(b *testing.B) {
			var nodes, reached uint64

			var fill float64

			var elapsed time.Duration

			for i := 0; i < b.N; i++ {
				s := runSearch(benchParse(b, tc.fen), 4, 0)

				nodes += s.nodes
				reached += uint64(s.depthReached)
				fill += s.ttFill
				elapsed += s.elapsed
			}

			n := float64(b.N)

			b.ReportMetric(float64(nodes)/n, "nodes/op")
			b.ReportMetric(float64(nodes)/elapsed.Seconds(), "nps")
			b.ReportMetric(float64(reached)/n, "depth")
			b.ReportMetric(100*fill/n, "tt-fill-%")
			b.ReportMetric(float64(elapsed.Milliseconds())/n, "search-ms")
		})
	}
}

// BenchmarkSearchTimeBudget gives every position the same 300ms and reports how
// deep it got.
func BenchmarkSearchTimeBudget(b *testing.B) {
	for _, tc := range benchPositions {
		b.Run(tc.name, func(b *testing.B) {
			var nodes, reached uint64

			var elapsed time.Duration

			for i := 0; i < b.N; i++ {
				s := runSearch(benchParse(b, tc.fen), 64, 300*time.Millisecond)

				nodes += s.nodes
				reached += uint64(s.depthReached)
				elapsed += s.elapsed
			}

			n := float64(b.N)

			b.ReportMetric(float64(reached)/n, "depth")
			b.ReportMetric(float64(nodes)/n, "nodes/op")
			b.ReportMetric(float64(nodes)/elapsed.Seconds(), "nps")
		})
	}
}

func BenchmarkEvaluate(b *testing.B) {
	for _, tc := range benchPositions {
		b.Run(tc.name, func(b *testing.B) {
			p := benchParse(b, tc.fen)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				Evaluate(&p)
			}
		})
	}
}

func BenchmarkMoveGeneration(b *testing.B) {
	for _, tc := range benchPositions {
		b.Run(tc.name, func(b *testing.B) {
			p := benchParse(b, tc.fen)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				allMoves(&p)
			}
		})
	}
}

func BenchmarkHashPosition(b *testing.B) {
	p := benchParse(b, benchPositions[2].fen)

	for i := 0; i < b.N; i++ {
		HashPosition(&p)
	}
}

func BenchmarkMakeUnmake(b *testing.B) {
	p := benchParse(b, benchPositions[2].fen)
	moves := allMoves(&p)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m := moves[i%len(moves)]
		undo := p.MakeMove(m)
		p.UnmakeMove(m, undo)
	}
}

func BenchmarkQuiescence(b *testing.B) {
	for _, tc := range benchPositions {
		b.Run(tc.name, func(b *testing.B) {
			p := benchParse(b, tc.fen)

			for i := 0; i < b.N; i++ {
				quiescence(&p, 0, -Infinity, Infinity, time.Time{})
			}
		})
	}
}

// TestSearchStatsReport prints one row per position and fails on regressions
// that are large enough to be real: a search that stops completing its depth, a
// table that is never used, or a pruning collapse. The node bounds are
// deliberately loose (about 3x the figures measured when this was written), so
// ordinary evaluation tuning does not trip them.
func TestSearchStatsReport(t *testing.T) {
	const depth = 4

	// Upper bounds on nodes at depth 4 with every heuristic on.
	maxNodes := map[string]uint64{
		"start":      8_000,
		"middlegame": 60_000,
		"kiwipete":   55_000,
		"endgame":    3_000,
		"tactical":   32_000,
	}

	t.Logf("%-11s %9s %9s %10s %6s %8s %9s %9s", "position", "nodes", "ms", "nps", "depth", "tt fill", "tt gain", "prune")

	for _, tc := range benchPositions {
		reset(t)

		oldLMR := UseLMR
		UseLMR = true

		t.Cleanup(func() { UseLMR = oldLMR })

		s := runSearch(benchParse(t, tc.fen), depth, 0)

		if s.depthReached != depth {
			t.Fatalf("%s: completed depth %d, want %d", tc.name, s.depthReached, depth)
		}

		if s.nodes == 0 {
			t.Fatalf("%s: no nodes searched", tc.name)
		}

		if s.ttFill <= 0 {
			t.Fatalf("%s: the transposition table was never written", tc.name)
		}

		if limit := maxNodes[tc.name]; s.nodes > limit {
			t.Fatalf("%s: %d nodes at depth %d, over the guard of %d: pruning or ordering regressed", tc.name, s.nodes, depth, limit)
		}

		if s.nps() < 2000 {
			t.Fatalf("%s: %.0f nodes/second is far below normal", tc.name, s.nps())
		}

		var noTT searchStats

		withOneSlotTable(func() {
			reset(t)

			UseLMR = true
			noTT = runSearch(benchParse(t, tc.fen), depth, 0)
		})

		// A table can cost a few percent (different bounds, different best moves),
		// but it must never make the search much worse.
		if s.nodes > noTT.nodes+noTT.nodes/10 {
			t.Fatalf("%s: a real table searched %d nodes, a one-slot table only %d", tc.name, s.nodes, noTT.nodes)
		}

		reset(t)

		p := benchParse(t, tc.fen)
		full := perftCount(&p, 3)

		shallow := runSearch(benchParse(t, tc.fen), 3, 0)

		ttGain := 100 * (1 - float64(s.nodes)/float64(noTT.nodes))
		prune := 100 * (1 - float64(shallow.nodes)/float64(full))

		t.Logf("%-11s %9d %9d %10.0f %6d %7.3f%% %8.1f%% %8.1f%%",
			tc.name, s.nodes, s.elapsed.Milliseconds(), s.nps(), s.depthReached, 100*s.ttFill, ttGain, prune)
	}
}
