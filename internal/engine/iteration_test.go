package engine

import (
	"testing"
	"time"
	"vajra/internal/chess"
)

func TestIterativeSearchCustomPosition(t *testing.T) {
	pos, err := chess.ParseFEN(
		"r1bq1rk1/ppp2ppp/2n5/8/2B1P3/2N5/PPP2PPP/R1BQ1RK1 w - - 0 1",
	)
	if err != nil {
		t.Fatalf("invalid position: %v", err)
	}
	t.Log("using:- IterativeSearch(&pos, 5, 100*time.Millisecond)")
	move, found := IterativeSearch(&pos, 5, 100*time.Millisecond)

	if !found {
		t.Fatal("iterative search returned no move")
	}

	legal := allMoves(&pos)

	if !containsMove(legal, move) {
		t.Fatalf(
			"iterative search returned illegal move: %s -> %s",
			SquareName(move.From), SquareName(move.To),
		)
	}

	t.Logf("Vajra chose: %s -> %s", SquareName(move.From), SquareName(move.To))
}

func TestCustomPositionMoves(t *testing.T) {
	pos, err := chess.ParseFEN(
		"r1bq1rk1/ppp2ppp/2n5/8/2B1P3/2N5/PPP2PPP/R1BQ1RK1 w - - 0 1",
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, move := range allMoves(&pos) {
		t.Logf("%s -> %s", SquareName(move.From), SquareName(move.To))
	}
}
