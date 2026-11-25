package uci

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"

	"vajra/internal/chess"
	"vajra/internal/engine"
)

func TestBenchPositionsAreValid(t *testing.T) {
	seen := map[string]bool{}

	for _, fen := range benchFENs {
		p, err := chess.ParseFEN(fen)
		if err != nil {
			t.Fatalf("%s: %v", fen, err)
		}

		if p.Board.WhiteKing == 0 || p.Board.BlackKing == 0 {
			t.Fatalf("%s: a king is missing", fen)
		}

		other := chess.White
		if p.SideToMove == chess.White {
			other = chess.Black
		}

		if p.InCheck(other) {
			t.Fatalf("%s: the side not to move is in check", fen)
		}

		if p.Status(p.SideToMove) == "checkmate" || p.Status(p.SideToMove) == "stalemate" {
			t.Fatalf("%s: the position is already over", fen)
		}

		if seen[fen] {
			t.Fatalf("%s is listed twice", fen)
		}

		seen[fen] = true
	}

	if len(benchFENs) < 10 {
		t.Fatalf("only %d bench positions", len(benchFENs))
	}
}

func TestBenchIsDeterministic(t *testing.T) {
	restoreEngineState(t)

	var a, b bytes.Buffer

	first, err := Bench(&a, 4)
	if err != nil {
		t.Fatal(err)
	}

	second, err := Bench(&b, 4)
	if err != nil {
		t.Fatal(err)
	}

	if first.Nodes == 0 || first.Nodes != second.Nodes {
		t.Fatalf("node counts differ between runs: %d and %d", first.Nodes, second.Nodes)
	}

	if first.Positions != len(benchFENs) {
		t.Fatalf("searched %d positions, want %d", first.Positions, len(benchFENs))
	}
}

func TestBenchIgnoresTheThreadsSetting(t *testing.T) {
	restoreEngineState(t)

	engine.Threads = 1

	var a bytes.Buffer

	single, err := Bench(&a, 3)
	if err != nil {
		t.Fatal(err)
	}

	engine.Threads = 4

	var b bytes.Buffer

	multi, err := Bench(&b, 3)
	if err != nil {
		t.Fatal(err)
	}

	if single.Nodes != multi.Nodes {
		t.Fatalf("bench must always run single threaded: %d vs %d nodes", single.Nodes, multi.Nodes)
	}

	if engine.Threads != 4 {
		t.Fatalf("Bench changed Threads to %d", engine.Threads)
	}
}

func TestBenchRestoresTheTable(t *testing.T) {
	restoreEngineState(t)

	before := engine.TT

	var out bytes.Buffer

	if _, err := Bench(&out, 2); err != nil {
		t.Fatal(err)
	}

	if engine.TT != before {
		t.Fatal("Bench must put the original transposition table back")
	}
}

func TestBenchOutput(t *testing.T) {
	restoreEngineState(t)

	var out bytes.Buffer

	result, err := Bench(&out, 3)
	if err != nil {
		t.Fatal(err)
	}

	text := out.String()

	for _, want := range []string{"position  1/12", "position 12/12", "depth           : 3", "nodes searched  : ", "nodes/second    : ", "total time (ms) : "} {
		if !strings.Contains(text, want) {
			t.Fatalf("output lacks %q:\n%s", want, text)
		}
	}

	if !strings.Contains(text, "nodes searched  : "+strconv.FormatUint(result.Nodes, 10)) {
		t.Fatalf("the printed total does not match the result %d:\n%s", result.Nodes, text)
	}
}

func TestBenchRejectsABadDepth(t *testing.T) {
	var out bytes.Buffer

	for _, depth := range []int{0, -1, -50} {
		if _, err := Bench(&out, depth); err == nil {
			t.Fatalf("depth %d should be refused", depth)
		}
	}
}

func TestBenchResultSpeed(t *testing.T) {
	if (BenchResult{Nodes: 100}).NodesPerSecond() != 0 {
		t.Fatal("no elapsed time means no speed")
	}

	if got := (BenchResult{Nodes: 5000, Elapsed: 2 * time.Second}).NodesPerSecond(); got != 2500 {
		t.Fatalf("got %d, want 2500", got)
	}
}

func TestVersionIsInTheHandshake(t *testing.T) {
	old := Version

	t.Cleanup(func() { Version = old })

	Version = "1.2.3"

	lines := session(t, "uci\n")

	if len(lines) == 0 || lines[0] != "id name Vajra 1.2.3" {
		t.Fatalf("got %v", lines)
	}
}
