package engine

import (
	"strings"
	"testing"

	"vajra/internal/chess"
)

func TestHashRestoredAfterMakeUnmake(t *testing.T) {
	for _, tc := range prodFENs {
		t.Run(tc.name, func(t *testing.T) {
			p := prodPos(t, tc.fen)
			want := HashPosition(&p)

			for _, m := range allMoves(&p) {
				undo := p.MakeMove(m)
				reply := allMoves(&p)

				for _, r := range reply {
					undo2 := p.MakeMove(r)
					p.UnmakeMove(r, undo2)
				}

				p.UnmakeMove(m, undo)

				if got := HashPosition(&p); got != want {
					t.Fatalf("%s: hash %x after make/unmake, want %x", prodMoveName(m), got, want)
				}
			}
		})
	}
}

func TestHashIdenticalForTranspositions(t *testing.T) {
	play := func(moves ...string) uint64 {
		p := prodPos(t, chess.StartingFEN)

		for _, name := range moves {
			var found bool

			for _, m := range allMoves(&p) {
				if prodMoveName(m) == name {
					p.MakeMove(m)

					found = true

					break
				}
			}

			if !found {
				t.Fatalf("%s is not legal in %s", name, p.FEN())
			}
		}

		return HashPosition(&p)
	}

	a := play("g1f3", "g8f6", "b1c3", "b8c6")
	b := play("b1c3", "b8c6", "g1f3", "g8f6")

	if a != b {
		t.Fatal("the same position reached by two move orders must hash identically")
	}

	c := play("g1f3", "g8f6", "b1c3", "b8c6", "c3b1", "c6b8", "b1c3", "b8c6")

	if a != c {
		t.Fatal("a repeated position must hash identically, repetition detection depends on it")
	}

	d := play("g1f3", "g8f6", "b1c3")

	if a == d {
		t.Fatal("different positions must not hash identically")
	}
}

func TestHashSeparatesEveryPositionInATree(t *testing.T) {
	for _, tc := range []struct{ name, fen string }{
		{"start", chess.StartingFEN},
		{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := prodPos(t, tc.fen)
			seen := map[uint64]string{}

			var walk func(depth int)

			walk = func(depth int) {
				key := p.Key()
				h := HashPosition(&p)

				if other, ok := seen[h]; ok && other != key {
					t.Fatalf("hash collision between two different positions: %x", h)
				}

				seen[h] = key

				if depth == 0 {
					return
				}

				for _, m := range allMoves(&p) {
					undo := p.MakeMove(m)
					walk(depth - 1)
					p.UnmakeMove(m, undo)
				}
			}

			walk(3)

			if len(seen) < 100 {
				t.Fatalf("only %d distinct positions seen, tree walk is broken", len(seen))
			}
		})
	}
}

func TestHashDistinguishesPieceColourAndType(t *testing.T) {
	boards := []string{
		"4k3/8/8/8/3N4/8/8/4K3 w - - 0 1",
		"4k3/8/8/8/3n4/8/8/4K3 w - - 0 1",
		"4k3/8/8/8/3B4/8/8/4K3 w - - 0 1",
		"4k3/8/8/8/3b4/8/8/4K3 w - - 0 1",
		"4k3/8/8/8/3N4/8/8/4K3 b - - 0 1",
		"4k3/8/8/8/4N3/8/8/4K3 w - - 0 1",
	}

	seen := map[uint64]string{}

	for _, fen := range boards {
		p := prodPos(t, fen)
		h := HashPosition(&p)

		if other, ok := seen[h]; ok {
			t.Fatalf("%q and %q hash the same", fen, other)
		}

		seen[h] = fen
	}
}

func TestTranspositionTableEdgeCases(t *testing.T) {
	t.Run("nil table is safe", func(t *testing.T) {
		var tt *TranspositionTable

		if _, ok := tt.Probe(42); ok {
			t.Fatal("nil table must miss")
		}

		tt.Store(42, 1, 1, TTExact, chess.Move{})
	})

	t.Run("a zero size still works", func(t *testing.T) {
		tt := NewTranspositionTable(0)

		tt.Store(7, 3, 55, TTExact, chess.Move{From: 1, To: 2})

		entry, ok := tt.Probe(7)
		if !ok || entry.Score != 55 || entry.Depth != 3 {
			t.Fatalf("probe = %+v, %v", entry, ok)
		}
	})

	t.Run("a colliding key replaces the old entry", func(t *testing.T) {
		tt := NewTranspositionTable(1)
		a := uint64(12345)
		b := a + uint64(len(tt.entries))

		tt.Store(a, 8, 10, TTExact, chess.Move{})
		tt.Store(b, 1, 20, TTExact, chess.Move{})

		if _, ok := tt.Probe(a); ok {
			t.Fatal("the older colliding entry should be gone")
		}

		if entry, ok := tt.Probe(b); !ok || entry.Score != 20 {
			t.Fatalf("the newer entry should be found, got %+v %v", entry, ok)
		}
	})

	t.Run("a different key never returns another position's entry", func(t *testing.T) {
		tt := NewTranspositionTable(1)

		tt.Store(100, 4, 9, TTLowerBound, chess.Move{})

		if _, ok := tt.Probe(101); ok {
			t.Fatal("probing an unrelated key must miss")
		}
	})

	t.Run("deeper result replaces shallower for the same key", func(t *testing.T) {
		tt := NewTranspositionTable(1)

		tt.Store(9, 2, 1, TTExact, chess.Move{})
		tt.Store(9, 6, 2, TTUpperBound, chess.Move{})

		entry, _ := tt.Probe(9)
		if entry.Depth != 6 || entry.Flag != TTUpperBound || entry.Score != 2 {
			t.Fatalf("entry = %+v", entry)
		}
	})

	t.Run("clear forgets everything", func(t *testing.T) {
		tt := NewTranspositionTable(1)

		for k := uint64(1); k < 500; k++ {
			tt.Store(k, 1, int(k), TTExact, chess.Move{})
		}

		tt.Clear()

		for k := uint64(1); k < 500; k++ {
			if _, ok := tt.Probe(k); ok {
				t.Fatalf("key %d survived Clear", k)
			}
		}
	})

	t.Run("the best move survives a round trip", func(t *testing.T) {
		tt := NewTranspositionTable(1)
		want := chess.Move{From: 12, To: 28, Piece: chess.WhitePawn, Flags: chess.DoublePawnPush}

		tt.Store(77, 5, 33, TTExact, want)

		entry, ok := tt.Probe(77)
		if !ok || entry.BestMove != want {
			t.Fatalf("best move = %+v, want %+v", entry.BestMove, want)
		}
	})
}

// An untouched slot holds the zero entry, whose key is 0. Probing key 0 must
// still miss, and a real entry stored under key 0 must be found.
func TestTranspositionTableEmptySlotIsAMiss(t *testing.T) {
	tt := NewTranspositionTable(1)

	if _, ok := tt.Probe(0); ok {
		t.Fatal("an empty slot must not count as a hit")
	}
}

func TestMateScoresSurviveTheTableAtAnyPly(t *testing.T) {
	for _, base := range []int{MateScore - 1, MateScore - 7, -MateScore + 1, -MateScore + 9, 0, 250, -250} {
		for ply := 0; ply < 20; ply++ {
			if got := scoreFromTT(scoreToTT(base, ply), ply); got != base {
				t.Fatalf("score %d at ply %d came back as %d", base, ply, got)
			}
		}
	}
}

func TestEvaluateSurvivesOddPositions(t *testing.T) {
	var empty chess.Position

	empty.EnPassant = -1

	if got := Evaluate(&empty); got != 0 {
		t.Fatalf("an empty board should evaluate to 0, got %d", got)
	}

	for _, fen := range []string{
		"4k3/8/8/8/8/8/8/4K3 w - - 0 1",
		"8/8/8/8/8/8/8/R7 w - - 0 1",
		"8/8/8/8/8/8/8/r7 w - - 0 1",
		"QQQQQQQQ/QQQQQQQQ/8/8/8/8/qqqqqqqq/qqqqqqqq w - - 0 1",
		"k7/8/8/8/8/8/8/K7 w - - 0 1",
		"7k/8/8/8/8/8/8/7K w - - 0 1",
	} {
		p, err := chess.ParseFEN(fen)
		if err != nil {
			t.Fatalf("ParseFEN(%q): %v", fen, err)
		}

		first := Evaluate(&p)

		if second := Evaluate(&p); first != second {
			t.Fatalf("%q evaluates differently on repeat: %d vs %d", fen, first, second)
		}

		if EvaluateBreakdown(&p).Total() != first {
			t.Fatalf("%q: breakdown total disagrees with Evaluate", fen)
		}
	}
}

func TestEvaluateIsColourSymmetricAcrossTheSuite(t *testing.T) {
	for _, tc := range prodFENs {
		t.Run(tc.name, func(t *testing.T) {
			placement := strings.Fields(tc.fen)[0]

			white, err := chess.ParseFEN(placement + " w - - 0 1")
			if err != nil {
				t.Fatal(err)
			}

			black, err := chess.ParseFEN(mirrorPlacement(placement) + " w - - 0 1")
			if err != nil {
				t.Fatal(err)
			}

			if a, b := Evaluate(&white), Evaluate(&black); a != -b {
				t.Fatalf("eval %d but mirrored eval %d, want %d", a, b, -a)
			}
		})
	}
}

func TestEvaluateRewardsMaterial(t *testing.T) {
	base := "4k3/8/8/8/8/8/8/4K3 w - - 0 1"

	extras := map[string]int{
		"4k3/8/8/8/8/8/8/3NK3 w - - 0 1": KnightValue,
		"4k3/8/8/8/8/8/8/3BK3 w - - 0 1": BishopValue,
		"4k3/8/8/8/8/8/8/3RK3 w - - 0 1": RookValue,
		"4k3/8/8/8/8/8/8/3QK3 w - - 0 1": QueenValue,
	}

	b, _ := chess.ParseFEN(base)
	baseScore := Evaluate(&b)

	var last int

	for _, order := range []string{
		"4k3/8/8/8/8/8/8/3NK3 w - - 0 1",
		"4k3/8/8/8/8/8/8/3RK3 w - - 0 1",
		"4k3/8/8/8/8/8/8/3QK3 w - - 0 1",
	} {
		p, _ := chess.ParseFEN(order)
		gain := Evaluate(&p) - baseScore

		if gain < extras[order]-150 || gain > extras[order]+150 {
			t.Fatalf("%q adds %d, expected about %d", order, gain, extras[order])
		}

		if gain <= last {
			t.Fatalf("%q should be worth more than the previous piece (%d <= %d)", order, gain, last)
		}

		last = gain
	}
}

func TestOrderingEdgeCases(t *testing.T) {
	state := &SearchState{}

	orderMovesHeuristic(nil, chess.White, 0, state)
	orderMovesHeuristic([]chess.Move{}, chess.White, 0, state)

	single := []chess.Move{{From: 1, To: 2}}

	orderMovesHeuristic(single, chess.White, 0, state)

	if single[0].From != 1 {
		t.Fatal("a single move must stay put")
	}

	moves := []chess.Move{{From: 8, To: 16, Piece: chess.WhitePawn}, {From: 9, To: 17, Piece: chess.WhitePawn}}

	orderMovesHeuristic(moves, chess.White, 0, nil)

	pawnTakesQueen := chess.Move{Piece: chess.WhitePawn, Captured: chess.BlackQueen, Flags: chess.Capture}
	rookTakesPawn := chess.Move{Piece: chess.WhiteRook, Captured: chess.BlackPawn, Flags: chess.Capture}
	quiet := chess.Move{From: 8, To: 16, Piece: chess.WhitePawn}

	mixed := []chess.Move{quiet, rookTakesPawn, pawnTakesQueen}

	orderMovesHeuristic(mixed, chess.White, 0, state)

	if mixed[0] != pawnTakesQueen || mixed[1] != rookTakesPawn || mixed[2] != quiet {
		t.Fatalf("captures must come first, best victim first: %+v", mixed)
	}

	equalQuiet := []chess.Move{{From: 1, To: 2}, {From: 3, To: 4}, {From: 5, To: 6}}

	orderMovesHeuristic(equalQuiet, chess.White, 0, state)

	if equalQuiet[0].From != 1 || equalQuiet[1].From != 3 || equalQuiet[2].From != 5 {
		t.Fatalf("equal-scored moves must keep their order: %+v", equalQuiet)
	}
}

func TestKillerAndHistoryBoundsAreSafe(t *testing.T) {
	var k KillerTable

	m := chess.Move{From: 1, To: 2, Piece: chess.WhiteKnight}

	for _, ply := range []int{-5, -1, MaxPly, MaxPly + 1, 10000} {
		k.store(ply, m)

		if k.isKiller(ply, m) != -1 {
			t.Fatalf("ply %d is out of range and must never report a killer", ply)
		}
	}

	var h HistoryTable

	for i := 0; i < 5000; i++ {
		h.update(chess.Black, m, 30)
	}

	if got := h.get(chess.Black, m); got != maxHistory {
		t.Fatalf("history = %d, want it capped at %d", got, maxHistory)
	}
}

func TestInsufficientMaterialEdgeCases(t *testing.T) {
	tests := []struct {
		fen  string
		want bool
	}{
		{"4k3/8/8/8/8/8/8/4K3 w - - 0 1", true},
		{"4k3/8/8/8/8/8/8/3BK3 w - - 0 1", true},
		{"4k3/8/8/8/8/8/8/3NK3 w - - 0 1", true},
		{"4kb2/8/8/8/8/8/8/3BK3 w - - 0 1", false},
		{"4k3/8/8/8/8/8/8/2NNK3 w - - 0 1", false},
		{"4k3/8/8/8/8/8/P7/4K3 w - - 0 1", false},
		{"4k3/8/8/8/8/8/8/3RK3 w - - 0 1", false},
		{"4k3/8/8/8/8/8/8/3QK3 w - - 0 1", false},
		{"4k3/p7/8/8/8/8/8/4K3 w - - 0 1", false},
	}

	for _, tc := range tests {
		p, err := chess.ParseFEN(tc.fen)
		if err != nil {
			t.Fatal(err)
		}

		if got := insufficientMaterial(&p); got != tc.want {
			t.Fatalf("%s: insufficientMaterial = %v, want %v", tc.fen, got, tc.want)
		}
	}
}

func TestTranspositionTableKeyZeroIsUsable(t *testing.T) {
	tt := NewTranspositionTable(1)

	tt.Store(0, 4, 77, TTExact, chess.Move{From: 3, To: 4})

	entry, ok := tt.Probe(0)
	if !ok || entry.Score != 77 || entry.Depth != 4 {
		t.Fatalf("an entry stored under key 0 must be found, got %+v %v", entry, ok)
	}

	tt.Clear()

	if _, ok := tt.Probe(0); ok {
		t.Fatal("Clear must remove the key 0 entry")
	}

	tt.Store(0, 6, 1, TTExact, chess.Move{})
	tt.Store(0, 2, 2, TTExact, chess.Move{})

	if entry, _ := tt.Probe(0); entry.Depth != 6 {
		t.Fatalf("a shallower result must not replace a deeper one for key 0, got depth %d", entry.Depth)
	}
}
