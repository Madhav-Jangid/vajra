package uci

import (
	"fmt"
	"io"
	"time"

	"vajra/internal/chess"
	"vajra/internal/engine"
)

const DefaultBenchDepth = 10

var benchFENs = []string{
	"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
	"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
	"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
	"r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1",
	"rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
	"r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10",
	"r1bqkbnr/pp1ppppp/2n5/2p5/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3",
	"rnbq1rk1/pp2ppbp/3p1np1/2p5/2PP4/2N2NP1/PP2PPBP/R1BQ1RK1 w - - 0 8",
	"2rr3k/pp3pp1/1nnqbN1p/3pN3/2pP4/2P3Q1/PPB4P/R4RK1 w - - 0 1",
	"r1bq1rk1/ppp2ppp/2np1n2/2b1p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 7",
	"8/8/4k3/3r4/3P4/4K3/8/3R4 w - - 0 1",
	"8/5pk1/6p1/2P5/8/1P4P1/5PK1/8 w - - 0 1",
}

type BenchResult struct {
	Positions int
	Nodes     uint64
	Elapsed   time.Duration
}

func (r BenchResult) NodesPerSecond() uint64 {
	if r.Elapsed <= 0 {
		return 0
	}

	return uint64(float64(r.Nodes) / r.Elapsed.Seconds())
}

func Bench(w io.Writer, depth int) (BenchResult, error) {
	if depth < 1 {
		return BenchResult{}, fmt.Errorf("bench depth must be at least 1, got %d", depth)
	}

	savedThreads, savedTT := engine.Threads, engine.TT

	defer func() { engine.Threads, engine.TT = savedThreads, savedTT }()

	engine.Threads = 1
	engine.TT = engine.NewTranspositionTable(defaultHashMB)

	engine.ClearStop()

	var result BenchResult

	for i, fen := range benchFENs {
		pos, err := chess.ParseFEN(fen)
		if err != nil {
			return result, fmt.Errorf("bench position %d: %w", i+1, err)
		}

		start := time.Now()

		engine.IterativeSearch(&pos, depth, 0)

		elapsed := time.Since(start)

		result.Positions++
		result.Nodes += engine.Nodes
		result.Elapsed += elapsed

		fmt.Fprintf(w, "position %2d/%d  nodes %10d  time %6d ms  nps %9d\n",
			i+1, len(benchFENs), engine.Nodes, elapsed.Milliseconds(),
			BenchResult{Nodes: engine.Nodes, Elapsed: elapsed}.NodesPerSecond())
	}

	fmt.Fprintf(w, "\n===========================\n")
	fmt.Fprintf(w, "depth           : %d\n", depth)
	fmt.Fprintf(w, "total time (ms) : %d\n", result.Elapsed.Milliseconds())
	fmt.Fprintf(w, "nodes searched  : %d\n", result.Nodes)
	fmt.Fprintf(w, "nodes/second    : %d\n", result.NodesPerSecond())

	return result, nil
}
