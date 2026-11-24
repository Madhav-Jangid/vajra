package engine

import (
	"fmt"
	"testing"
	"time"

	"vajra/internal/chess"
)

var prodFENs = []struct{ name, fen string }{
	{"start", chess.StartingFEN},
	{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"},
	{"pos3", "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1"},
	{"pos4", "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1"},
	{"pos5", "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8"},
	{"middlegame", "r1bq1rk1/ppp2ppp/2np1n2/2b1p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 7"},
	{"endgame", "8/5pk1/6p1/2P5/8/1P4P1/5PK1/8 w - - 0 1"},
	{"promotion race", "1nb5/Pnk1K3/2p5/8/8/8/8/8 w - - 0 1"},
}

func prodPos(t *testing.T, fen string) chess.Position {
	t.Helper()

	p, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN(%q): %v", fen, err)
	}

	return p
}

func prodMoveName(m chess.Move) string {
	name := SquareName(m.From) + SquareName(m.To)

	if m.Flags&chess.Promotion != 0 {
		switch m.Promotion {
		case chess.WhiteQueen, chess.BlackQueen:
			name += "q"
		case chess.WhiteRook, chess.BlackRook:
			name += "r"
		case chess.WhiteBishop, chess.BlackBishop:
			name += "b"
		case chess.WhiteKnight, chess.BlackKnight:
			name += "n"
		}
	}

	return name
}

func prodIsLegal(p *chess.Position, m chess.Move) bool {
	for _, l := range allMoves(p) {
		if l == m {
			return true
		}
	}

	return false
}

type prodFlags struct {
	ordering, history, killers, pvs, lmr bool
}

func (f prodFlags) String() string {
	return fmt.Sprintf("ord=%v hist=%v kill=%v pvs=%v lmr=%v", f.ordering, f.history, f.killers, f.pvs, f.lmr)
}

// prodSetFlags clears the search state, then applies the flags. It restores
// every flag when the test ends.
func prodSetFlags(t *testing.T, f prodFlags) {
	t.Helper()

	reset(t)

	oldO, oldH, oldK, oldP, oldL := UseMoveOrdering, UseHistory, UseKillers, UsePVS, UseLMR

	t.Cleanup(func() {
		UseMoveOrdering, UseHistory, UseKillers, UsePVS, UseLMR = oldO, oldH, oldK, oldP, oldL
	})

	UseMoveOrdering, UseHistory, UseKillers, UsePVS, UseLMR = f.ordering, f.history, f.killers, f.pvs, f.lmr
}

var prodDefaultFlags = prodFlags{ordering: true, history: true, killers: true, pvs: true, lmr: true}

func prodRootScore(t *testing.T, p chess.Position, depth int, f prodFlags) int {
	t.Helper()

	prodSetFlags(t, f)

	score, ok := negamax(&p, depth, 0, -Infinity, Infinity, time.Time{}, &SearchState{})
	if !ok {
		t.Fatal("search did not complete")
	}

	return score
}

func TestMatesAreFound(t *testing.T) {
	tests := []struct {
		name      string
		fen       string
		move      string
		depth     int
		mateScore int
	}{
		{"back rank mate in 1", "6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1", "a1a8", 2, MateScore - 1},
		{"smothered mate in 1", "6rk/6pp/8/6N1/8/8/8/K7 w - - 0 1", "g5f7", 2, MateScore - 1},
		{"queen and king mate in 1", "7k/8/5K2/8/8/8/8/6Q1 w - - 0 1", "g1g7", 2, MateScore - 1},
		{"black delivers mate in 1", "r3k3/8/8/8/8/8/5PPP/6K1 b - - 0 1", "a8a1", 2, MateScore - 1},
		{"rook sacrifice mate in 2", "kbK5/pp6/1P6/8/8/8/8/R7 w - - 0 1", "a1a6", 4, MateScore - 3},
		{"knight promotion mate in 1", "1nb5/Pnk1K3/2p5/8/8/8/8/8 w - - 0 1", "a7a8n", 2, MateScore - 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for depth := tc.depth; depth <= tc.depth+1; depth++ {
				p := prodPos(t, tc.fen)

				prodSetFlags(t, prodDefaultFlags)

				move, ok := Search(&p, depth, 0)
				if !ok {
					t.Fatalf("depth %d: no move found", depth)
				}

				if got := prodMoveName(move); got != tc.move {
					t.Fatalf("depth %d: played %s, want %s", depth, got, tc.move)
				}

				if got := prodRootScore(t, prodPos(t, tc.fen), depth, prodDefaultFlags); got != tc.mateScore {
					t.Fatalf("depth %d: score %d, want %d", depth, got, tc.mateScore)
				}
			}
		})
	}
}

func TestIterativeSearchFindsMates(t *testing.T) {
	tests := []struct {
		name, fen, move string
		maxDepth        int
	}{
		{"back rank", "6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1", "a1a8", 4},
		{"smothered", "6rk/6pp/8/6N1/8/8/8/K7 w - - 0 1", "g5f7", 4},
		{"mate in 2", "kbK5/pp6/1P6/8/8/8/8/R7 w - - 0 1", "a1a6", 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := prodPos(t, tc.fen)

			prodSetFlags(t, prodDefaultFlags)

			move, ok := IterativeSearch(&p, tc.maxDepth, 0)
			if !ok || prodMoveName(move) != tc.move {
				t.Fatalf("IterativeSearch = %s (found=%v), want %s", prodMoveName(move), ok, tc.move)
			}
		})
	}
}

func TestShortestMateIsPreferred(t *testing.T) {
	p := prodPos(t, "6k1/5ppp/8/8/8/8/Q7/R3K3 w - - 0 1")

	prodSetFlags(t, prodDefaultFlags)

	move, ok := Search(&p, 4, 0)
	if !ok {
		t.Fatal("no move found")
	}

	if got := prodRootScore(t, prodPos(t, "6k1/5ppp/8/8/8/8/Q7/R3K3 w - - 0 1"), 4, prodDefaultFlags); got != MateScore-1 {
		t.Fatalf("score %d, want mate in 1 (%d)", got, MateScore-1)
	}

	undo := p.MakeMove(move)
	mated := p.InCheck(chess.Black) && len(allMoves(&p)) == 0
	p.UnmakeMove(move, undo)

	if !mated {
		t.Fatalf("%s does not mate immediately", prodMoveName(move))
	}
}

func TestNoLegalMovesMeansNoMove(t *testing.T) {
	tests := []struct{ name, fen string }{
		{"checkmate", "R5k1/5ppp/8/8/8/8/8/4K3 b - - 0 1"},
		{"stalemate", "7k/5Q2/6K1/8/8/8/8/8 b - - 0 1"},
		{"fool's mate", "rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := prodPos(t, tc.fen)

			prodSetFlags(t, prodDefaultFlags)

			if _, ok := Search(&p, 3, 0); ok {
				t.Fatal("Search returned a move in a position with no legal moves")
			}

			if _, ok := IterativeSearch(&p, 3, 0); ok {
				t.Fatal("IterativeSearch returned a move in a position with no legal moves")
			}
		})
	}
}

func TestTerminalScoresAtTheRoot(t *testing.T) {
	prodSetFlags(t, prodDefaultFlags)

	mated := prodPos(t, "R5k1/5ppp/8/8/8/8/8/4K3 b - - 0 1")

	if got := prodRootScore(t, mated, 3, prodDefaultFlags); got != -MateScore {
		t.Fatalf("checkmated side scores %d, want %d", got, -MateScore)
	}

	stalemate := prodPos(t, "7k/5Q2/6K1/8/8/8/8/8 b - - 0 1")

	if got := prodRootScore(t, stalemate, 3, prodDefaultFlags); got != 0 {
		t.Fatalf("stalemated side scores %d, want 0", got)
	}
}

func TestSingleLegalMoveIsPlayed(t *testing.T) {
	p := prodPos(t, "1r5k/8/8/8/8/8/8/K7 w - - 0 1")

	if got := len(allMoves(&p)); got != 1 {
		t.Fatalf("setup: %d legal moves, want 1", got)
	}

	for _, depth := range []int{1, 3} {
		prodSetFlags(t, prodDefaultFlags)

		move, ok := Search(&p, depth, 0)
		if !ok || prodMoveName(move) != "a1a2" {
			t.Fatalf("depth %d: got %s (found=%v), want a1a2", depth, prodMoveName(move), ok)
		}
	}
}

func TestWinsHangingPiece(t *testing.T) {
	p := prodPos(t, "4k3/8/8/3q4/8/8/8/3RK3 w - - 0 1")

	prodSetFlags(t, prodDefaultFlags)

	move, ok := Search(&p, 3, 0)
	if !ok || prodMoveName(move) != "d1d5" {
		t.Fatalf("got %s (found=%v), want d1d5 winning the queen", prodMoveName(move), ok)
	}
}

func TestDoesNotWalkIntoCapture(t *testing.T) {
	p := prodPos(t, "4k3/8/8/3p4/8/8/8/3QK3 w - - 0 1")

	prodSetFlags(t, prodDefaultFlags)

	move, ok := Search(&p, 3, 0)
	if !ok {
		t.Fatal("no move found")
	}

	undo := p.MakeMove(move)
	captured := false

	for _, m := range allMoves(&p) {
		if m.Flags&chess.Capture != 0 && m.Captured == chess.WhiteQueen {
			captured = true
		}
	}

	p.UnmakeMove(move, undo)

	if captured {
		t.Fatalf("%s leaves the queen to be captured", prodMoveName(move))
	}
}

func TestAvoidsStalematingTheWinningPosition(t *testing.T) {
	t.Run("mate in 1 is delivered", func(t *testing.T) {
		p := prodPos(t, "7k/8/5K2/8/8/8/8/6Q1 w - - 0 1")

		prodSetFlags(t, prodDefaultFlags)

		move, ok := Search(&p, 4, 0)
		if !ok {
			t.Fatal("no move found")
		}

		undo := p.MakeMove(move)
		status := p.Status(chess.Black)
		p.UnmakeMove(move, undo)

		if status != "checkmate" {
			t.Fatalf("%s gives status %q, want checkmate", prodMoveName(move), status)
		}
	})

	t.Run("a forced mate is found without stalemating", func(t *testing.T) {
		fen := "7k/8/6K1/8/8/8/8/6Q1 w - - 0 1"
		p := prodPos(t, fen)

		prodSetFlags(t, prodDefaultFlags)

		move, ok := Search(&p, 4, 0)
		if !ok {
			t.Fatal("no move found")
		}

		undo := p.MakeMove(move)
		status := p.Status(chess.Black)
		p.UnmakeMove(move, undo)

		if status == "stalemate" {
			t.Fatalf("%s stalemates a won position", prodMoveName(move))
		}

		if score := prodRootScore(t, prodPos(t, fen), 4, prodDefaultFlags); score < MateScore-1000 {
			t.Fatalf("score %d, expected a forced mate", score)
		}
	})
}

func TestUnderpromotionToAvoidStalemate(t *testing.T) {
	const fen = "8/1P6/8/8/8/8/8/5K1k w - - 0 1"

	p := prodPos(t, fen)

	var queen chess.Move

	for _, m := range allMoves(&p) {
		if m.From == sq("b7") && m.To == sq("b8") && m.Promotion == chess.WhiteQueen {
			queen = m
		}
	}

	undo := p.MakeMove(queen)
	stale := p.Status(chess.Black) == "stalemate"
	p.UnmakeMove(queen, undo)

	if !stale {
		t.Fatal("setup: b8=Q should stalemate in this position")
	}

	for depth := 2; depth <= 5; depth++ {
		p := prodPos(t, fen)

		prodSetFlags(t, prodDefaultFlags)

		move, ok := Search(&p, depth, 0)
		if !ok {
			t.Fatalf("depth %d: no move", depth)
		}

		if move.From == sq("b7") && move.To == sq("b8") && move.Promotion == chess.WhiteQueen {
			t.Fatalf("depth %d: promoted to a queen and stalemated", depth)
		}

		if score := prodRootScore(t, prodPos(t, fen), depth, prodDefaultFlags); score <= 300 {
			t.Fatalf("depth %d: score %d, the position is winning", depth, score)
		}
	}
}

func TestSearchAlwaysReturnsALegalMove(t *testing.T) {
	flagSets := []prodFlags{
		prodDefaultFlags,
		{ordering: true},
		{ordering: true, history: true},
		{ordering: true, killers: true},
		{ordering: true, pvs: true},
		{ordering: true, lmr: true},
		{ordering: true, history: true, killers: true, pvs: true},
	}

	for _, tc := range prodFENs {
		for _, f := range flagSets {
			t.Run(tc.name+"/"+f.String(), func(t *testing.T) {
				p := prodPos(t, tc.fen)

				prodSetFlags(t, f)

				move, ok := Search(&p, 3, 0)
				if !ok {
					t.Fatal("no move found")
				}

				if !prodIsLegal(&p, move) {
					t.Fatalf("illegal move %s in %s", prodMoveName(move), tc.fen)
				}
			})
		}
	}
}

func TestSearchLeavesPositionUntouched(t *testing.T) {
	for _, tc := range prodFENs {
		t.Run(tc.name, func(t *testing.T) {
			p := prodPos(t, tc.fen)
			before := p

			prodSetFlags(t, prodDefaultFlags)

			Search(&p, 3, 0)

			if p != before {
				t.Fatalf("Search changed the position:\n%s\n%s", before.FEN(), p.FEN())
			}

			IterativeSearch(&p, 3, 0)

			if p != before {
				t.Fatalf("IterativeSearch changed the position:\n%s\n%s", before.FEN(), p.FEN())
			}
		})
	}
}

func TestSearchIsDeterministic(t *testing.T) {
	for _, tc := range prodFENs {
		t.Run(tc.name, func(t *testing.T) {
			a, b := prodPos(t, tc.fen), prodPos(t, tc.fen)

			prodSetFlags(t, prodDefaultFlags)
			m1, ok1 := IterativeSearch(&a, 3, 0)
			n1 := Nodes

			prodSetFlags(t, prodDefaultFlags)
			m2, ok2 := IterativeSearch(&b, 3, 0)
			n2 := Nodes

			if m1 != m2 || ok1 != ok2 || n1 != n2 {
				t.Fatalf("two identical searches differ: %s/%d vs %s/%d", prodMoveName(m1), n1, prodMoveName(m2), n2)
			}
		})
	}
}

// PVS, history and killers only change the order and the windows, never the
// result. With LMR off, every combination must return the baseline score.
func TestExactHeuristicsNeverChangeTheScore(t *testing.T) {
	cases := []struct {
		name  string
		fen   string
		depth int
	}{
		{"start", chess.StartingFEN, 4},
		{"middlegame", "r1bq1rk1/ppp2ppp/2np1n2/2b1p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 7", 3},
		{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 3},
		{"endgame", "8/5pk1/6p1/2P5/8/1P4P1/5PK1/8 w - - 0 1", 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := prodRootScore(t, prodPos(t, tc.fen), tc.depth, prodFlags{ordering: true})

			for mask := 0; mask < 8; mask++ {
				f := prodFlags{ordering: true, history: mask&1 != 0, killers: mask&2 != 0, pvs: mask&4 != 0}

				if got := prodRootScore(t, prodPos(t, tc.fen), tc.depth, f); got != want {
					t.Fatalf("%s: score %d, want %d", f, got, want)
				}
			}
		})
	}
}

func TestLMRKeepsMatesAndSmallScoreError(t *testing.T) {
	for _, tc := range prodFENs {
		t.Run(tc.name, func(t *testing.T) {
			off := prodRootScore(t, prodPos(t, tc.fen), 4, prodFlags{ordering: true, history: true, killers: true, pvs: true})
			on := prodRootScore(t, prodPos(t, tc.fen), 4, prodDefaultFlags)

			if off >= MateScore-1000 && on != off {
				t.Fatalf("LMR lost a forced mate: off=%d on=%d", off, on)
			}

			diff := on - off
			if diff < 0 {
				diff = -diff
			}

			if diff > 200 {
				t.Fatalf("LMR moved the score by %d (off=%d on=%d)", diff, off, on)
			}
		})
	}
}

func TestWarmTranspositionTableGivesTheSameScore(t *testing.T) {
	for _, tc := range prodFENs {
		t.Run(tc.name, func(t *testing.T) {
			f := prodFlags{ordering: true, history: true, killers: true, pvs: true}

			cold := prodRootScore(t, prodPos(t, tc.fen), 3, f)

			p := prodPos(t, tc.fen)
			state := &SearchState{}

			warm, ok := negamax(&p, 3, 0, -Infinity, Infinity, time.Time{}, state)
			if !ok {
				t.Fatal("search did not complete")
			}

			if cold != warm {
				t.Fatalf("a table filled by an earlier search changed the score: cold=%d warm=%d", cold, warm)
			}
		})
	}
}

func TestIterativeSearchRespectsTimeLimit(t *testing.T) {
	p := prodPos(t, "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	prodSetFlags(t, prodDefaultFlags)

	start := time.Now()
	move, ok := IterativeSearch(&p, 30, 200*time.Millisecond)
	elapsed := time.Since(start)

	if !ok || !prodIsLegal(&p, move) {
		t.Fatalf("expected a legal move, got %s (found=%v)", prodMoveName(move), ok)
	}

	if elapsed > 2*time.Second {
		t.Fatalf("search took %v with a 200ms limit", elapsed)
	}
}

func TestIterativeSearchAlwaysFallsBackToALegalMove(t *testing.T) {
	for _, tc := range prodFENs {
		t.Run(tc.name, func(t *testing.T) {
			p := prodPos(t, tc.fen)

			prodSetFlags(t, prodDefaultFlags)

			move, ok := IterativeSearch(&p, 10, time.Nanosecond)
			if !ok {
				t.Fatal("an expired deadline must still produce a move")
			}

			if !prodIsLegal(&p, move) {
				t.Fatalf("illegal fallback move %s", prodMoveName(move))
			}
		})
	}
}

func TestSearchDepthIsClamped(t *testing.T) {
	p := prodPos(t, chess.StartingFEN)

	for _, depth := range []int{0, -1, -50} {
		prodSetFlags(t, prodDefaultFlags)

		move, ok := Search(&p, depth, 0)
		if !ok || !prodIsLegal(&p, move) {
			t.Fatalf("depth %d: got %s (found=%v)", depth, prodMoveName(move), ok)
		}
	}
}

func TestDrawnPositionsScoreZero(t *testing.T) {
	tests := []struct{ name, fen string }{
		{"king vs king", "4k3/8/8/8/8/8/8/4K3 w - - 0 1"},
		{"king and bishop vs king", "4k3/8/8/8/8/8/8/3BK3 w - - 0 1"},
		{"king and knight vs king", "4k3/8/8/8/8/8/8/3NK3 w - - 0 1"},
		{"fifty-move rule even a queen up", "4k3/8/8/8/8/8/8/3QK3 w - - 100 80"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := prodRootScore(t, prodPos(t, tc.fen), 3, prodDefaultFlags); got != 0 {
				t.Fatalf("score %d, want 0", got)
			}
		})
	}
}

func TestRepetitionInGameHistoryIsAvoidedWhenWinning(t *testing.T) {
	p := prodPos(t, "4k3/8/8/8/8/8/8/3QK3 w - - 8 40")

	prodSetFlags(t, prodDefaultFlags)

	GameHistory = []uint64{HashPosition(&p)}

	t.Cleanup(func() { GameHistory = nil })

	move, ok := Search(&p, 4, 0)
	if !ok {
		t.Fatal("no move found")
	}

	if !prodIsLegal(&p, move) {
		t.Fatalf("illegal move %s", prodMoveName(move))
	}
}

func prodSelfPlay(t *testing.T, fen string, plies int, depth int) {
	t.Helper()

	p := prodPos(t, fen)

	for ply := 0; ply < plies; ply++ {
		status := p.Status(p.SideToMove)
		if status == "checkmate" || status == "stalemate" || status == "insufficient material" || status == "fifty-move rule" {
			return
		}

		prodSetFlags(t, prodDefaultFlags)

		move, ok := Search(&p, depth, 0)
		if !ok {
			t.Fatalf("ply %d: no move in %s", ply, p.FEN())
		}

		if !prodIsLegal(&p, move) {
			t.Fatalf("ply %d: illegal move %s in %s", ply, prodMoveName(move), p.FEN())
		}

		p.MakeMove(move)

		b := &p.Board
		if b.WhiteKing == 0 || b.BlackKing == 0 {
			t.Fatalf("ply %d: a king was captured (%s)", ply, p.FEN())
		}

		if round, err := chess.ParseFEN(p.FEN()); err != nil || round != p {
			t.Fatalf("ply %d: position does not round trip: %s", ply, p.FEN())
		}
	}
}

func TestSelfPlayStaysLegal(t *testing.T) {
	for _, tc := range []struct {
		name, fen string
		plies     int
	}{
		{"from the start", chess.StartingFEN, 40},
		{"from kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 24},
		{"from a promotion race", "8/P6k/8/8/8/8/p6K/8 w - - 0 1", 20},
		{"from pos4 with castling and promotions", "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1", 24},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prodSelfPlay(t, tc.fen, tc.plies, 2)
		})
	}
}

func TestAllMovesMatchesTheMoveGenerator(t *testing.T) {
	for _, tc := range prodFENs {
		t.Run(tc.name, func(t *testing.T) {
			p := prodPos(t, tc.fen)

			count := 0

			for sq := 0; sq < 64; sq++ {
				piece := p.Board.PieceAt(sq)
				if piece.Type != chess.Empty && piece.Color == p.SideToMove {
					count += len(p.LegalMoves(sq))
				}
			}

			if got := len(allMoves(&p)); got != count {
				t.Fatalf("allMoves = %d, want %d", got, count)
			}
		})
	}
}

func TestSquareNameCoversTheBoard(t *testing.T) {
	seen := map[string]bool{}

	for s := 0; s < 64; s++ {
		name := SquareName(s)

		if len(name) != 2 || seen[name] {
			t.Fatalf("square %d has bad or duplicate name %q", s, name)
		}

		seen[name] = true

		if sq(name) != s {
			t.Fatalf("%q does not round trip to %d", name, s)
		}
	}
}
