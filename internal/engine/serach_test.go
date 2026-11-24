package engine

import (
	"testing"
	"time"

	"vajra/internal/chess"
)

const noEnPassant = -1

var (
	startRows = []string{
		"rnbqkbnr",
		"pppppppp",
		"........",
		"........",
		"........",
		"........",
		"PPPPPPPP",
		"RNBQKBNR",
	}

	sparseRows = []string{
		"r...k...",
		"pp...ppp",
		"........",
		"........",
		"....N...",
		"........",
		"PPP..PPP",
		"R...K..R",
	}

	mateInOneWhiteRows = []string{
		"......k.",
		".....ppp",
		"........",
		"........",
		"........",
		"........",
		"........",
		"R.....K.",
	}

	mateInOneBlackRows = []string{
		"r.....k.",
		"........",
		"........",
		"........",
		"........",
		"........",
		".....PPP",
		"......K.",
	}

	blackMatedRows = []string{
		"R.....k.",
		".....ppp",
		"........",
		"........",
		"........",
		"........",
		"........",
		"......K.",
	}

	stalemateRows = []string{
		"k.......",
		"..K.....",
		".Q......",
		"........",
		"........",
		"........",
		"........",
		"........",
	}

	queenUpRows = []string{
		"....k...",
		"........",
		"........",
		"........",
		"........",
		"........",
		"........",
		"...QK...",
	}

	knightDownRows = []string{
		"....k...",
		"........",
		"........",
		"........",
		"........",
		"........",
		"........",
		"....K..n",
	}

	kingVsKingRows = []string{
		"....k...",
		"........",
		"........",
		"........",
		"........",
		"........",
		"........",
		"....K...",
	}

	rookWinsKnightWhiteRows = []string{
		".......k",
		"........",
		"........",
		"n.......",
		"........",
		"........",
		"........",
		"R......K",
	}

	rookWinsKnightBlackRows = []string{
		"r......k",
		"........",
		"........",
		"........",
		"N.......",
		"........",
		"........",
		".......K",
	}
)

func sq(name string) int {
	return int(name[0]-'a') + int(name[1]-'1')*8
}

func fromRows(side chess.Color, rows []string) *chess.Position {
	p := &chess.Position{
		SideToMove:     side,
		EnPassant:      noEnPassant,
		FullmoveNumber: 1,
	}

	for i, row := range rows {
		for file, ch := range row {
			if ch == '.' {
				continue
			}

			pieceType := chess.PieceTypeMap[string(ch)]
			p.Board.SetPiece(pieceType, (7-i)*8+file)
		}
	}

	return p
}

func reset(t *testing.T) {
	t.Helper()

	oldOrdering, oldLMR := UseMoveOrdering, UseLMR

	TT.Clear()
	history = nil
	GameHistory = nil
	UseMoveOrdering = true

	// LMR is lossy, so exact-score tests run with it off. LMR tests
	// turn it on explicitly after reset.
	UseLMR = false
	Nodes = 0

	t.Cleanup(func() {
		TT.Clear()
		history = nil
		GameHistory = nil
		UseMoveOrdering = oldOrdering
		UseLMR = oldLMR
	})
}

func samePosition(a, b chess.Position) bool {
	return a.Board == b.Board &&
		a.SideToMove == b.SideToMove &&
		a.CastlingRights == b.CastlingRights &&
		a.EnPassant == b.EnPassant &&
		a.HalfmoveClock == b.HalfmoveClock &&
		a.FullmoveNumber == b.FullmoveNumber
}

func containsMove(moves []chess.Move, m chess.Move) bool {
	for _, x := range moves {
		if sameMove(x, m) {
			return true
		}
	}
	return false
}

func testMove(from, to string) chess.Move {
	return chess.Move{
		From:  sq(from),
		To:    sq(to),
		Piece: chess.WhitePawn,
	}
}

// func bruteQ(p *chess.Position, ply int) int {
// 	moves := allMoves(p)

// 	// Checkmate / stalemate.
// 	if len(moves) == 0 {
// 		if p.InCheck(p.SideToMove) {
// 			return -MateScore + ply
// 		}
// 		return 0
// 	}

// 	// If we're in check, stand-pat is illegal.
// 	// We must search every legal evasion.
// 	if p.InCheck(p.SideToMove) {
// 		best := -Infinity

// 		for _, m := range moves {
// 			undo := p.MakeMove(m)
// 			score := -bruteQ(p, ply+1)
// 			p.UnmakeMove(m, undo)

// 			if score > best {
// 				best = score
// 			}
// 		}

// 		return best
// 	}

// 	// Stand-pat evaluation.
// 	best := evalSide(p)

// 	// Only continue with tactical moves.
// 	for _, m := range moves {
// 		if !isQuiescenceMove(m) {
// 			continue
// 		}

// 		undo := p.MakeMove(m)
// 		score := -bruteQ(p, ply+1)
// 		p.UnmakeMove(m, undo)

// 		if score > best {
// 			best = score
// 		}
// 	}

//		return best
//	}

func TestNegamaxMatchesBruteForceAtMultipleDepths(t *testing.T) {
	tests := []struct {
		name string
		side chess.Color
		rows []string
	}{
		{"white", chess.White, sparseRows},
		{"black", chess.Black, sparseRows},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for depth := 1; depth <= 3; depth++ {
				reset(t)

				p := fromRows(tt.side, tt.rows)

				want := brute(p, depth, 0)

				reset(t)

				got, ok := negamax(
					p,
					depth,
					0,
					-Infinity,
					Infinity,
					time.Time{},
					&SearchState{},
				)

				t.Logf(
					"depth=%d brute=%d negamax=%d ok=%v",
					depth,
					want,
					got,
					ok,
				)

				if !ok || got != want {
					t.Fatalf("depth=%d: negamax=%d ok=%v, want %d", depth, got, ok, want)
				}
			}
		})
	}
}

func brute(p *chess.Position, depth, ply int) int {
	if depth <= 0 {
		return bruteQ(p, ply)
	}

	moves := allMoves(p)

	if len(moves) == 0 {
		if p.InCheck(p.SideToMove) {
			return -MateScore + ply
		}

		return 0
	}

	best := -Infinity

	for _, m := range moves {
		undo := p.MakeMove(m)

		score := -brute(p, depth-1, ply+1)

		p.UnmakeMove(m, undo)

		if score > best {
			best = score
		}
	}

	return best
}

func TestAllMoves(t *testing.T) {
	t.Run("start position has 20 moves", func(t *testing.T) {
		p := fromRows(chess.White, startRows)

		if got := len(allMoves(p)); got != 20 {
			t.Fatalf("got %d moves, want 20", got)
		}
	})

	t.Run("start position has 20 moves for black", func(t *testing.T) {
		p := fromRows(chess.Black, startRows)

		if got := len(allMoves(p)); got != 20 {
			t.Fatalf("got %d moves, want 20", got)
		}
	})

	t.Run("lone black king ignores empty squares", func(t *testing.T) {
		rows := []string{
			"........",
			"........",
			"........",
			"....k...",
			"........",
			"........",
			"........",
			"K.......",
		}
		p := fromRows(chess.Black, rows)

		if got := len(allMoves(p)); got != 8 {
			t.Fatalf("got %d moves, want 8", got)
		}
	})

	t.Run("every move starts on a friendly piece", func(t *testing.T) {
		p := fromRows(chess.Black, sparseRows)

		for _, m := range allMoves(p) {
			piece := p.Board.PieceAt(m.From)

			if piece.Type == chess.Empty || piece.Color != chess.Black {
				t.Fatalf("move from %s is not a black piece", SquareName(m.From))
			}
		}
	})

	t.Run("checkmated side has no moves", func(t *testing.T) {
		p := fromRows(chess.Black, blackMatedRows)

		if got := len(allMoves(p)); got != 0 {
			t.Fatalf("got %d moves, want 0", got)
		}
	})

	t.Run("stalemated side has no moves", func(t *testing.T) {
		p := fromRows(chess.Black, stalemateRows)

		if got := len(allMoves(p)); got != 0 {
			t.Fatalf("got %d moves, want 0", got)
		}
	})
}

func TestEvalSide(t *testing.T) {
	white := fromRows(chess.White, queenUpRows)
	black := fromRows(chess.Black, queenUpRows)

	if evalSide(white) <= 0 {
		t.Fatalf("white to move with extra queen should be positive, got %d", evalSide(white))
	}

	if evalSide(black) >= 0 {
		t.Fatalf("black to move facing extra queen should be negative, got %d", evalSide(black))
	}

	if evalSide(white) != Evaluate(white) {
		t.Fatal("white side score should equal Evaluate")
	}

	if evalSide(black) != -Evaluate(black) {
		t.Fatal("black side score should equal -Evaluate")
	}
}

func TestExpired(t *testing.T) {
	if expired(time.Time{}) {
		t.Fatal("zero deadline must never expire")
	}

	if !expired(time.Now().Add(-time.Second)) {
		t.Fatal("past deadline must be expired")
	}

	if expired(time.Now().Add(time.Hour)) {
		t.Fatal("future deadline must not be expired")
	}
}

func TestScoreTTRoundTrip(t *testing.T) {
	scores := []int{
		0, 1, -1, 350, -350, 99000, -99000,
		MateScore - 1, MateScore - 7, -MateScore + 1, -MateScore + 9,
	}

	for _, ply := range []int{0, 1, 5, 40} {
		for _, s := range scores {
			if got := scoreFromTT(scoreToTT(s, ply), ply); got != s {
				t.Fatalf("round trip ply=%d score=%d got %d", ply, s, got)
			}
		}
	}

	t.Run("normal scores are unchanged", func(t *testing.T) {
		if scoreToTT(350, 9) != 350 || scoreFromTT(-350, 9) != -350 {
			t.Fatal("non-mate scores must not be adjusted")
		}
	})

	t.Run("mate distance is relative to the node", func(t *testing.T) {
		stored := scoreToTT(MateScore-3, 2)

		if got := scoreFromTT(stored, 4); got != MateScore-5 {
			t.Fatalf("got %d, want %d", got, MateScore-5)
		}

		stored = scoreToTT(-MateScore+3, 2)

		if got := scoreFromTT(stored, 4); got != -MateScore+5 {
			t.Fatalf("got %d, want %d", got, -MateScore+5)
		}
	})
}

func TestPutTTMoveFirst(t *testing.T) {
	a, b, c, d := testMove("a2", "a3"), testMove("b2", "b3"), testMove("c2", "c3"), testMove("d2", "d3")

	t.Run("moves middle move to front and keeps order", func(t *testing.T) {
		moves := []chess.Move{a, b, c, d}
		moves = putTTMoveFirst(moves, c)

		want := []chess.Move{c, a, b, d}

		for i := range want {
			if !sameMove(moves[i], want[i]) {
				t.Fatalf("index %d wrong", i)
			}
		}
	})

	t.Run("moves last move to front", func(t *testing.T) {
		moves := putTTMoveFirst([]chess.Move{a, b, c, d}, d)

		want := []chess.Move{d, a, b, c}

		for i := range want {
			if !sameMove(moves[i], want[i]) {
				t.Fatalf("index %d wrong", i)
			}
		}
	})

	t.Run("already first is unchanged", func(t *testing.T) {
		moves := putTTMoveFirst([]chess.Move{a, b, c}, a)

		want := []chess.Move{a, b, c}

		for i := range want {
			if !sameMove(moves[i], want[i]) {
				t.Fatalf("index %d wrong", i)
			}
		}
	})

	t.Run("missing move is a no-op", func(t *testing.T) {
		moves := putTTMoveFirst([]chess.Move{a, b, c}, d)

		want := []chess.Move{a, b, c}

		for i := range want {
			if !sameMove(moves[i], want[i]) {
				t.Fatalf("index %d wrong", i)
			}
		}
	})

	t.Run("empty move is a no-op", func(t *testing.T) {
		moves := putTTMoveFirst([]chess.Move{a, b, c}, chess.Move{})

		want := []chess.Move{a, b, c}

		for i := range want {
			if !sameMove(moves[i], want[i]) {
				t.Fatalf("index %d wrong", i)
			}
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		if got := putTTMoveFirst(nil, a); len(got) != 0 {
			t.Fatal("expected empty result")
		}
	})

	t.Run("single move", func(t *testing.T) {
		got := putTTMoveFirst([]chess.Move{a}, a)

		if len(got) != 1 || !sameMove(got[0], a) {
			t.Fatal("single move changed")
		}
	})
}

func TestSameMove(t *testing.T) {
	base := chess.Move{From: 8, To: 16, Piece: chess.WhitePawn}

	if !sameMove(base, base) {
		t.Fatal("identical moves must match")
	}

	if sameMove(base, chess.Move{From: 9, To: 16, Piece: chess.WhitePawn}) {
		t.Fatal("different From must not match")
	}

	if sameMove(base, chess.Move{From: 8, To: 24, Piece: chess.WhitePawn}) {
		t.Fatal("different To must not match")
	}

	if sameMove(base, chess.Move{From: 8, To: 16, Piece: chess.BlackPawn}) {
		t.Fatal("different Piece must not match")
	}

	if sameMove(
		chess.Move{From: 48, To: 56, Piece: chess.WhitePawn, Promotion: chess.WhiteQueen},
		chess.Move{From: 48, To: 56, Piece: chess.WhitePawn, Promotion: chess.WhiteKnight},
	) {
		t.Fatal("different promotion must not match")
	}
}

func TestIsRepetition(t *testing.T) {
	t.Cleanup(func() { history = nil })

	t.Run("empty history", func(t *testing.T) {
		history = nil

		if isRepetition(1, 10) {
			t.Fatal("empty history cannot repeat")
		}
	})

	t.Run("same side to move entry matches", func(t *testing.T) {
		history = []uint64{1, 2, 3, 4}

		if !isRepetition(3, 4) {
			t.Fatal("expected repetition at n-2")
		}

		if !isRepetition(1, 4) {
			t.Fatal("expected repetition at n-4")
		}
	})

	t.Run("opposite side entries are ignored", func(t *testing.T) {
		history = []uint64{1, 2, 3, 4}

		if isRepetition(4, 4) {
			t.Fatal("parent position must not count")
		}

		if isRepetition(2, 4) {
			t.Fatal("opposite side position must not count")
		}
	})

	t.Run("halfmove clock limits the window", func(t *testing.T) {
		history = []uint64{1, 2, 3, 4}

		if isRepetition(3, 1) {
			t.Fatal("clock 1 leaves no reachable entries")
		}

		if isRepetition(1, 3) {
			t.Fatal("clock 3 should not reach index 0")
		}

		if !isRepetition(1, 4) {
			t.Fatal("clock 4 should reach index 0")
		}
	})

	t.Run("zero clock never repeats", func(t *testing.T) {
		history = []uint64{1, 2, 3, 4}

		if isRepetition(3, 0) {
			t.Fatal("clock 0 means irreversible move just happened")
		}
	})

	t.Run("short history", func(t *testing.T) {
		history = []uint64{7}

		if isRepetition(7, 50) {
			t.Fatal("single entry is the parent, not a repetition")
		}
	})

	t.Run("huge clock does not go out of range", func(t *testing.T) {
		history = []uint64{1, 2, 3}

		if !isRepetition(2, 1000) {
			t.Fatal("expected repetition at n-2")
		}

		if isRepetition(1, 1000) || isRepetition(3, 1000) {
			t.Fatal("opposite side entries must not count, even with a huge clock")
		}
	})
}

func TestInsufficientMaterial(t *testing.T) {
	tests := []struct {
		name string
		rows []string
		want bool
	}{
		{"king vs king", kingVsKingRows, true},
		{"king and knight vs king", []string{
			"....k...", "........", "........", "........",
			"........", "........", "........", "...NK...",
		}, true},
		{"king and bishop vs king", []string{
			"....k...", "........", "........", "........",
			"........", "........", "........", "...BK...",
		}, true},
		{"king vs king and knight", knightDownRows, true},
		{"two knights", []string{
			"....k...", "........", "........", "........",
			"........", "........", "........", "..NNK...",
		}, false},
		{"knight each", []string{
			"....k...", "...n....", "........", "........",
			"........", "........", "........", "...NK...",
		}, false},
		{"bishop each", []string{
			"....k...", "...b....", "........", "........",
			"........", "........", "........", "...BK...",
		}, false},
		{"pawn", []string{
			"....k...", "........", "........", "........",
			"........", "........", "...P....", "....K...",
		}, false},
		{"rook", []string{
			"....k...", "........", "........", "........",
			"........", "........", "........", "...RK...",
		}, false},
		{"queen", queenUpRows, false},
		{"black pawn", []string{
			"....k...", "...p....", "........", "........",
			"........", "........", "........", "....K...",
		}, false},
		{"black rook", []string{
			"...rk...", "........", "........", "........",
			"........", "........", "........", "....K...",
		}, false},
		{"black queen", []string{
			"...qk...", "........", "........", "........",
			"........", "........", "........", "....K...",
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := fromRows(chess.White, tt.rows)

			if got := insufficientMaterial(p); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStartSearch(t *testing.T) {
	reset(t)

	Nodes = 99
	history = []uint64{5, 6, 7, 8}
	GameHistory = []uint64{1, 2, 3}

	TT.Store(42, 3, 10, TTExact, chess.Move{})

	deadline := startSearch(0)

	if !deadline.IsZero() {
		t.Fatal("zero limit must give zero deadline")
	}

	if Nodes != 0 {
		t.Fatal("nodes must reset")
	}

	if _, ok := TT.Probe(42); ok {
		t.Fatal("TT must be cleared")
	}

	if len(history) != 3 || history[0] != 1 || history[2] != 3 {
		t.Fatalf("history must be seeded from GameHistory, got %v", history)
	}

	history = append(history, 9)

	if len(GameHistory) != 3 || GameHistory[0] != 1 {
		t.Fatal("GameHistory must not be modified by search history")
	}

	before := time.Now()
	deadline = startSearch(50 * time.Millisecond)

	if deadline.Before(before.Add(40*time.Millisecond)) || deadline.After(time.Now().Add(time.Second)) {
		t.Fatalf("deadline looks wrong: %v", deadline)
	}
}

func TestNegamaxTerminalScores(t *testing.T) {
	t.Run("checkmate returns mate score relative to ply", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.Black, blackMatedRows)

		for _, ply := range []int{0, 1, 3, 10} {
			score, ok := negamax(p, 2, ply, -Infinity, Infinity, time.Time{}, &SearchState{})

			if !ok || score != -MateScore+ply {
				t.Fatalf("ply %d: got %d ok=%v, want %d", ply, score, ok, -MateScore+ply)
			}
		}
	})

	t.Run("checkmate is detected even at depth zero", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.Black, blackMatedRows)

		score, ok := negamax(p, 0, 2, -Infinity, Infinity, time.Time{}, &SearchState{})

		if !ok || score != -MateScore+2 {
			t.Fatalf("got %d ok=%v", score, ok)
		}
	})

	t.Run("stalemate scores zero", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.Black, stalemateRows)

		score, ok := negamax(p, 3, 1, -Infinity, Infinity, time.Time{}, &SearchState{})

		if !ok || score != 0 {
			t.Fatalf("got %d ok=%v, want 0", score, ok)
		}
	})

	t.Run("depth zero without captures returns static evaluation", func(t *testing.T) {

		reset(t)
		p := fromRows(chess.White, queenUpRows)

		score, ok := negamax(p, 0, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if !ok || score != evalSide(p) {
			t.Fatalf("got %d, want %d", score, evalSide(p))
		}
	})

	t.Run("mate in one is seen from the attacker", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, mateInOneWhiteRows)

		score, ok := negamax(p, 1, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if !ok || score != MateScore-1 {
			t.Fatalf("got %d, want %d", score, MateScore-1)
		}
	})

	t.Run("mate in one as black", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.Black, mateInOneBlackRows)

		score, ok := negamax(p, 1, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if !ok || score != MateScore-1 {
			t.Fatalf("got %d, want %d", score, MateScore-1)
		}
	})
}

func TestNegamaxDraws(t *testing.T) {
	t.Run("fifty move rule", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)
		p.HalfmoveClock = 100

		score, ok := negamax(p, 0, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if !ok || score != 0 {
			t.Fatalf("got %d, want 0", score)
		}
	})

	t.Run("clock above 100 is still a draw", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)
		p.HalfmoveClock = 130

		score, _ := negamax(p, 2, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if score != 0 {
			t.Fatalf("got %d, want 0", score)
		}
	})

	t.Run("clock 99 is not yet a draw", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)
		p.HalfmoveClock = 99

		score, _ := negamax(p, 0, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if score <= 0 {
			t.Fatalf("got %d, want positive", score)
		}
	})

	t.Run("quiet move onto clock 100 scores zero", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)
		p.HalfmoveClock = 99

		score, _ := negamax(p, 1, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if score != 0 {
			t.Fatalf("got %d, want 0", score)
		}
	})

	t.Run("quiet move onto clock 99 keeps advantage", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)
		p.HalfmoveClock = 98

		score, _ := negamax(p, 1, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if score <= 0 {
			t.Fatalf("got %d, want positive", score)
		}
	})

	t.Run("insufficient material right after capture", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, knightDownRows)
		p.HalfmoveClock = 0

		score, ok := negamax(p, 0, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if !ok || score != 0 {
			t.Fatalf("got %d, want 0", score)
		}
	})

	t.Run("insufficient material is a draw regardless of clock", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, knightDownRows)
		p.HalfmoveClock = 5

		score, _ := negamax(p, 0, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if score != 0 {
			t.Fatalf("expected draw score at any halfmove clock, got %d", score)
		}
	})

	t.Run("king vs king", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, kingVsKingRows)

		score, _ := negamax(p, 3, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if score != 0 {
			t.Fatalf("got %d, want 0", score)
		}
	})

	t.Run("repetition in history", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)
		p.HalfmoveClock = 4

		history = []uint64{HashPosition(p), 0}

		score, ok := negamax(p, 0, 2, -Infinity, Infinity, time.Time{}, &SearchState{})

		if !ok || score != 0 {
			t.Fatalf("got %d, want 0", score)
		}
	})

	t.Run("repetition ignored when clock window excludes it", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)
		p.HalfmoveClock = 1

		history = []uint64{HashPosition(p), 0}

		score, _ := negamax(p, 0, 2, -Infinity, Infinity, time.Time{}, &SearchState{})

		if score <= 0 {
			t.Fatalf("got %d, want positive", score)
		}
	})

	t.Run("history is balanced after draw return", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)
		p.HalfmoveClock = 100
		history = []uint64{1, 2}

		negamax(p, 2, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if len(history) != 2 {
			t.Fatalf("history length %d, want 2", len(history))
		}
	})
}

func TestNegamaxTranspositionTable(t *testing.T) {
	t.Run("exact entry is returned", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)

		TT.Store(HashPosition(p), 10, 1234, TTExact, chess.Move{})

		score, ok := negamax(p, 3, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if !ok || score != 1234 {
			t.Fatalf("got %d, want 1234", score)
		}
	})

	t.Run("shallow entry is ignored", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)

		TT.Store(HashPosition(p), 0, 7777, TTExact, chess.Move{})

		score, _ := negamax(p, 2, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if score == 7777 {
			t.Fatal("entry with insufficient depth must not be used")
		}
	})

	t.Run("entry with equal depth is used", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)

		TT.Store(HashPosition(p), 2, 4321, TTExact, chess.Move{})

		score, _ := negamax(p, 2, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if score != 4321 {
			t.Fatalf("got %d, want 4321", score)
		}
	})

	t.Run("lower bound causes cutoff", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)

		TT.Store(HashPosition(p), 10, 5000, TTLowerBound, chess.Move{})

		score, ok := negamax(p, 3, 0, -Infinity, 4000, time.Time{}, &SearchState{})

		if !ok || score != 5000 {
			t.Fatalf("got %d, want 5000", score)
		}
	})

	t.Run("upper bound causes cutoff", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)

		TT.Store(HashPosition(p), 10, -5000, TTUpperBound, chess.Move{})

		score, ok := negamax(p, 3, 0, -4000, Infinity, time.Time{}, &SearchState{})

		if !ok || score != -5000 {
			t.Fatalf("got %d, want -5000", score)
		}
	})

	t.Run("mate score is adjusted for ply", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)

		TT.Store(HashPosition(p), 10, scoreToTT(MateScore-3, 2), TTExact, chess.Move{})

		score, _ := negamax(p, 3, 4, -Infinity, Infinity, time.Time{}, &SearchState{})

		if score != MateScore-5 {
			t.Fatalf("got %d, want %d", score, MateScore-5)
		}
	})

	t.Run("search stores its result", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)

		score, _ := negamax(p, 2, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		entry, ok := TT.Probe(HashPosition(p))

		if !ok {
			t.Fatal("expected entry in TT")
		}

		if entry.Depth != 2 || entry.Flag != TTExact || entry.Score != score {
			t.Fatalf("bad entry: %+v (score %d)", entry, score)
		}
	})

	t.Run("fail low stores upper bound", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)

		negamax(p, 1, 0, Infinity-1, Infinity, time.Time{}, &SearchState{})

		entry, ok := TT.Probe(HashPosition(p))

		if !ok || entry.Flag != TTUpperBound {
			t.Fatalf("expected upper bound, got %+v ok=%v", entry, ok)
		}
	})

	t.Run("fail high stores lower bound", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, queenUpRows)

		negamax(p, 1, 0, -Infinity, -MateScore, time.Time{}, &SearchState{})

		entry, ok := TT.Probe(HashPosition(p))

		if !ok || entry.Flag != TTLowerBound {
			t.Fatalf("expected lower bound, got %+v ok=%v", entry, ok)
		}
	})

	t.Run("stored best move is legal", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, sparseRows)

		negamax(p, 2, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		entry, ok := TT.Probe(HashPosition(p))

		if !ok || !containsMove(allMoves(p), entry.BestMove) {
			t.Fatalf("stored move is not legal: %+v", entry.BestMove)
		}
	})
}

func TestNegamaxDeadline(t *testing.T) {
	reset(t)
	p := fromRows(chess.White, startRows)

	before := *p
	score, ok := negamax(p, 4, 0, -Infinity, Infinity, time.Now().Add(-time.Second), &SearchState{})

	if ok || score != 0 {
		t.Fatalf("expired deadline must return (0,false), got (%d,%v)", score, ok)
	}

	if Nodes != 1 {
		t.Fatalf("expected one node counted, got %d", Nodes)
	}

	if !samePosition(before, *p) {
		t.Fatal("position modified")
	}

	if len(history) != 0 {
		t.Fatal("history must be untouched")
	}
}

func TestNegamaxMatchesBruteForce(t *testing.T) {
	tests := []struct {
		name  string
		side  chess.Color
		rows  []string
		depth int
	}{
		{"start white d3", chess.White, startRows, 3},
		{"start black d3", chess.Black, startRows, 3},
		{"sparse white d3", chess.White, sparseRows, 3},
		{"sparse black d3", chess.Black, sparseRows, 3},
		{"queen up d3", chess.White, queenUpRows, 3},
		{"rook wins knight d3", chess.White, rookWinsKnightWhiteRows, 3},
		{"mate position d2", chess.White, mateInOneWhiteRows, 2},
		{"black mate position d2", chess.Black, mateInOneBlackRows, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset(t)
			p := fromRows(tt.side, tt.rows)
			snapshot := *p

			t.Logf("starting brute force")
			resetBruteQ()

			want := brute(p, tt.depth, 0)

			t.Logf("brute force finished: %d", want)

			t.Logf("starting negamax")

			got, ok := negamax(
				p,
				tt.depth,
				0,
				-Infinity,
				Infinity,
				time.Time{},
				&SearchState{},
			)

			t.Logf("negamax finished: %d", got)

			if !ok || got != want {
				t.Fatalf("alpha-beta %d, brute force %d", got, want)
			}

			if !samePosition(snapshot, *p) {
				t.Fatal("position not restored")
			}

			if len(history) != 0 {
				t.Fatalf("history length %d, want 0", len(history))
			}
		})
	}

	t.Run("ordering off gives the same score", func(t *testing.T) {
		reset(t)
		UseMoveOrdering = false

		p := fromRows(chess.White, sparseRows)
		want := brute(p, 3, 0)

		got, _ := negamax(p, 3, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
	})

	t.Run("warm TT gives the same score", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, sparseRows)
		want := brute(p, 3, 0)

		negamax(p, 2, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		got, _ := negamax(p, 3, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		if got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
	})
}

func TestSearchRoot(t *testing.T) {
	t.Run("finds mate in one", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, mateInOneWhiteRows)

		move, found, completed := searchRoot(p, 1, time.Time{}, chess.Move{}, &SearchState{})

		if !found || !completed {
			t.Fatalf("found=%v completed=%v", found, completed)
		}

		if move.From != sq("a1") || move.To != sq("a8") {
			t.Fatalf("got %s%s, want a1a8", SquareName(move.From), SquareName(move.To))
		}
	})

	t.Run("no legal moves is complete but not found", func(t *testing.T) {
		reset(t)

		for _, rows := range [][]string{blackMatedRows, stalemateRows} {
			p := fromRows(chess.Black, rows)

			_, found, completed := searchRoot(p, 3, time.Time{}, chess.Move{}, &SearchState{})

			if found || !completed {
				t.Fatalf("found=%v completed=%v", found, completed)
			}
		}
	})

	t.Run("expired deadline stops immediately", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, startRows)

		_, found, completed := searchRoot(p, 3, time.Now().Add(-time.Second), chess.Move{}, &SearchState{})

		if found || completed {
			t.Fatalf("found=%v completed=%v", found, completed)
		}
	})

	t.Run("root is stored in TT when completed", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, sparseRows)

		move, _, completed := searchRoot(p, 2, time.Time{}, chess.Move{}, &SearchState{})

		entry, ok := TT.Probe(HashPosition(p))

		if !completed || !ok {
			t.Fatal("expected completed search with TT entry")
		}

		if entry.Depth != 2 || entry.Flag != TTExact || !sameMove(entry.BestMove, move) {
			t.Fatalf("bad root entry: %+v", entry)
		}
	})

	t.Run("root is not stored when interrupted", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, startRows)

		searchRoot(p, 3, time.Now().Add(-time.Second), chess.Move{}, &SearchState{})

		if _, ok := TT.Probe(HashPosition(p)); ok {
			t.Fatal("interrupted search must not store the root")
		}
	})

	t.Run("position and history are restored", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, sparseRows)
		snapshot := *p
		history = []uint64{1, 2, 3}

		searchRoot(p, 3, time.Time{}, chess.Move{}, &SearchState{})

		if !samePosition(snapshot, *p) {
			t.Fatal("position not restored")
		}

		if len(history) != 3 {
			t.Fatalf("history length %d, want 3", len(history))
		}
	})

	t.Run("history is restored after interruption", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, startRows)
		history = []uint64{1, 2, 3}

		searchRoot(p, 3, time.Now().Add(-time.Second), chess.Move{}, &SearchState{})

		if len(history) != 3 {
			t.Fatalf("history length %d, want 3", len(history))
		}
	})
}

func TestSearch(t *testing.T) {
	t.Run("white mate in one at several depths", func(t *testing.T) {
		for _, depth := range []int{1, 2, 3, 4} {
			reset(t)
			p := fromRows(chess.White, mateInOneWhiteRows)

			move, found := Search(p, depth, 0)

			if !found || move.From != sq("a1") || move.To != sq("a8") {
				t.Fatalf("depth %d: got %s%s found=%v", depth, SquareName(move.From), SquareName(move.To), found)
			}
		}
	})

	t.Run("black mate in one at several depths", func(t *testing.T) {
		for _, depth := range []int{1, 2, 3, 4} {
			reset(t)
			p := fromRows(chess.Black, mateInOneBlackRows)

			move, found := Search(p, depth, 0)

			if !found || move.From != sq("a8") || move.To != sq("a1") {
				t.Fatalf("depth %d: got %s%s found=%v", depth, SquareName(move.From), SquareName(move.To), found)
			}
		}
	})

	t.Run("mate found without move ordering", func(t *testing.T) {
		reset(t)
		UseMoveOrdering = false
		p := fromRows(chess.White, mateInOneWhiteRows)

		move, found := Search(p, 3, 0)

		if !found || move.From != sq("a1") || move.To != sq("a8") {
			t.Fatalf("got %s%s", SquareName(move.From), SquareName(move.To))
		}
	})

	t.Run("wins loose knight as white", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, rookWinsKnightWhiteRows)

		move, found := Search(p, 2, 0)

		if !found || move.From != sq("a1") || move.To != sq("a5") {
			t.Fatalf("got %s%s", SquareName(move.From), SquareName(move.To))
		}
	})

	t.Run("wins loose knight as black", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.Black, rookWinsKnightBlackRows)

		move, found := Search(p, 2, 0)

		if !found || move.From != sq("a8") || move.To != sq("a4") {
			t.Fatalf("got %s%s", SquareName(move.From), SquareName(move.To))
		}
	})

	t.Run("returns a legal move from the start position", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, startRows)

		move, found := Search(p, 2, 0)

		if !found || !containsMove(allMoves(p), move) {
			t.Fatal("expected a legal move")
		}
	})

	t.Run("no legal moves", func(t *testing.T) {
		reset(t)

		if _, found := Search(fromRows(chess.Black, blackMatedRows), 3, 0); found {
			t.Fatal("checkmated side should find nothing")
		}

		if _, found := Search(fromRows(chess.Black, stalemateRows), 3, 0); found {
			t.Fatal("stalemated side should find nothing")
		}
	})

	t.Run("depth below one is clamped", func(t *testing.T) {
		for _, depth := range []int{0, -1, -50} {
			reset(t)
			p := fromRows(chess.White, mateInOneWhiteRows)

			move, found := Search(p, depth, 0)

			if !found || move.To != sq("a8") {
				t.Fatalf("depth %d: expected clamped search to find mate", depth)
			}
		}
	})

	t.Run("position is restored", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, sparseRows)
		snapshot := *p

		Search(p, 3, 0)

		if !samePosition(snapshot, *p) {
			t.Fatal("position not restored")
		}
	})

	t.Run("is deterministic", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, sparseRows)

		m1, _ := Search(p, 3, 0)
		n1 := Nodes

		m2, _ := Search(p, 3, 0)
		n2 := Nodes

		if !sameMove(m1, m2) || n1 != n2 {
			t.Fatalf("runs differ: %d vs %d nodes", n1, n2)
		}

		if n1 == 0 {
			t.Fatal("nodes not counted")
		}
	})

	t.Run("nodes reset every call", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, sparseRows)

		Search(p, 3, 0)
		big := Nodes

		Search(p, 1, 0)

		if Nodes >= big {
			t.Fatalf("nodes not reset: %d then %d", big, Nodes)
		}
	})

	t.Run("tiny time limit stops quickly", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, startRows)

		start := time.Now()
		Search(p, 8, time.Millisecond)

		if time.Since(start) > 2*time.Second {
			t.Fatal("search ignored its time limit")
		}
	})

	t.Run("search path history is balanced", func(t *testing.T) {
		reset(t)
		GameHistory = []uint64{11, 22, 33}
		p := fromRows(chess.White, sparseRows)

		Search(p, 3, 0)

		if len(history) != 3 || history[0] != 11 || history[2] != 33 {
			t.Fatalf("history corrupted: %v", history)
		}
	})
}

func TestSearchAvoidsRepetitionFromGameHistory(t *testing.T) {
	reset(t)

	p := fromRows(chess.White, queenUpRows)
	p.HalfmoveClock = 2

	best, found := Search(p, 2, 0)

	if !found {
		t.Fatal("no move found")
	}

	undo := p.MakeMove(best)
	childHash := HashPosition(p)
	p.UnmakeMove(best, undo)

	GameHistory = []uint64{childHash}

	again, found := Search(p, 2, 0)

	if !found {
		t.Fatal("no move found with game history")
	}

	if sameMove(best, again) {
		t.Fatal("engine repeated a position already in game history while winning")
	}
}

func TestIterativeSearch(t *testing.T) {
	t.Run("finds mate in one", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, mateInOneWhiteRows)

		move, found := IterativeSearch(p, 4, 0)

		if !found || move.From != sq("a1") || move.To != sq("a8") {
			t.Fatalf("got %s%s", SquareName(move.From), SquareName(move.To))
		}
	})

	t.Run("finds mate in one as black", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.Black, mateInOneBlackRows)

		move, found := IterativeSearch(p, 4, 0)

		if !found || move.From != sq("a8") || move.To != sq("a1") {
			t.Fatalf("got %s%s", SquareName(move.From), SquareName(move.To))
		}
	})

	t.Run("matches fixed depth search", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, rookWinsKnightWhiteRows)

		fixed, _ := Search(p, 3, 0)

		reset(t)

		iter, found := IterativeSearch(p, 3, 0)

		if !found || !sameMove(fixed, iter) {
			t.Fatalf("iterative %s%s differs from fixed %s%s",
				SquareName(iter.From), SquareName(iter.To),
				SquareName(fixed.From), SquareName(fixed.To))
		}
	})

	t.Run("returns a legal move", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, startRows)

		move, found := IterativeSearch(p, 3, 0)

		if !found || !containsMove(allMoves(p), move) {
			t.Fatal("expected a legal move")
		}
	})

	t.Run("max depth zero falls back to a legal move", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, startRows)

		move, found := IterativeSearch(p, 0, 0)

		if !found || !containsMove(allMoves(p), move) {
			t.Fatal("expected fallback legal move")
		}
	})

	t.Run("expired time still returns a legal move", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, startRows)

		move, found := IterativeSearch(p, 20, time.Nanosecond)

		if !found || !containsMove(allMoves(p), move) {
			t.Fatal("expected fallback legal move")
		}
	})

	t.Run("time limit is respected and a move is returned", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, sparseRows)

		start := time.Now()
		move, found := IterativeSearch(p, 30, 150*time.Millisecond)
		elapsed := time.Since(start)

		if !found || !containsMove(allMoves(p), move) {
			t.Fatal("expected a legal move")
		}

		if elapsed > 2*time.Second {
			t.Fatalf("took %v", elapsed)
		}
	})

	t.Run("no legal moves", func(t *testing.T) {
		reset(t)

		if _, found := IterativeSearch(fromRows(chess.Black, blackMatedRows), 5, 0); found {
			t.Fatal("checkmated side should find nothing")
		}

		if _, found := IterativeSearch(fromRows(chess.Black, stalemateRows), 5, 0); found {
			t.Fatal("stalemated side should find nothing")
		}

		if _, found := IterativeSearch(fromRows(chess.Black, blackMatedRows), 0, 0); found {
			t.Fatal("checkmated side with depth 0 should find nothing")
		}
	})

	t.Run("root best move ends up in the TT", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, sparseRows)

		move, _ := IterativeSearch(p, 3, 0)

		entry, ok := TT.Probe(HashPosition(p))

		if !ok || !sameMove(entry.BestMove, move) {
			t.Fatalf("TT root move %+v differs from result %+v", entry.BestMove, move)
		}
	})

	t.Run("TT survives between iterations", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, sparseRows)

		IterativeSearch(p, 2, 0)
		small := Nodes

		reset(t)
		Search(p, 2, 0)
		direct := Nodes

		if small == 0 || direct == 0 {
			t.Fatal("nodes not counted")
		}

		reset(t)
		IterativeSearch(p, 4, 0)
		iterative := Nodes

		reset(t)
		Search(p, 4, 0)
		fixed := Nodes

		if iterative > 3*fixed {
			t.Fatalf("iterative deepening did far more work (%d) than a direct search (%d)", iterative, fixed)
		}
	})

	t.Run("position is restored", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, sparseRows)
		snapshot := *p

		IterativeSearch(p, 3, 0)

		if !samePosition(snapshot, *p) {
			t.Fatal("position not restored")
		}
	})

	t.Run("position is restored after timeout", func(t *testing.T) {
		reset(t)
		p := fromRows(chess.White, sparseRows)
		snapshot := *p

		IterativeSearch(p, 30, 50*time.Millisecond)

		if !samePosition(snapshot, *p) {
			t.Fatal("position not restored after timeout")
		}

		if len(history) != 0 {
			t.Fatalf("history length %d after timeout, want 0", len(history))
		}
	})

	t.Run("game history is preserved", func(t *testing.T) {
		reset(t)
		GameHistory = []uint64{101, 202, 303}
		p := fromRows(chess.White, sparseRows)

		IterativeSearch(p, 3, 0)

		if len(history) != 3 || len(GameHistory) != 3 {
			t.Fatalf("history=%v GameHistory=%v", history, GameHistory)
		}
	})
}

func TestMoveScore(t *testing.T) {
	quiet := chess.Move{From: 8, To: 16, Piece: chess.WhitePawn}

	if moveScore(quiet) != 0 {
		t.Fatal("quiet move should score zero")
	}

	pawnTakesQueen := chess.Move{
		Piece: chess.WhitePawn, Captured: chess.BlackQueen, Flags: chess.Capture,
	}
	queenTakesPawn := chess.Move{
		Piece: chess.WhiteQueen, Captured: chess.BlackPawn, Flags: chess.Capture,
	}

	if moveScore(pawnTakesQueen) <= moveScore(queenTakesPawn) {
		t.Fatal("pawn taking queen must outrank queen taking pawn")
	}

	if moveScore(pawnTakesQueen) <= moveScore(quiet) {
		t.Fatal("captures must outrank quiet moves")
	}

	promo := chess.Move{
		Piece: chess.WhitePawn, Promotion: chess.WhiteQueen, Flags: chess.Promotion,
	}

	if moveScore(promo) <= moveScore(quiet) {
		t.Fatal("promotion must outrank quiet move")
	}

	queenPromo := moveScore(promo)
	knightPromo := moveScore(chess.Move{
		Piece: chess.WhitePawn, Promotion: chess.WhiteKnight, Flags: chess.Promotion,
	})

	if queenPromo <= knightPromo {
		t.Fatal("queen promotion must outrank knight promotion")
	}
}

func TestSquareName(t *testing.T) {
	tests := map[int]string{
		0: "a1", 7: "h1", 8: "a2", 27: "d4", 36: "e5", 56: "a8", 63: "h8",
	}

	for square, want := range tests {
		if got := SquareName(square); got != want {
			t.Fatalf("SquareName(%d) = %s, want %s", square, got, want)
		}
	}

	for square := 0; square < 64; square++ {
		if sq(SquareName(square)) != square {
			t.Fatalf("square %d does not round trip", square)
		}
	}
}

func bit(square int) chess.Bitboard {
	return chess.Bitboard(1) << uint(square)
}

func setPiece(
	p *chess.Position,
	piece chess.PieceType,
	square int,
) {
	b := bit(square)

	switch piece {
	case chess.WhiteKing:
		p.Board.WhiteKing |= b
	case chess.WhiteQueen:
		p.Board.WhiteQueens |= b
	case chess.WhiteRook:
		p.Board.WhiteRooks |= b
	case chess.WhiteBishop:
		p.Board.WhiteBishops |= b
	case chess.WhiteKnight:
		p.Board.WhiteKnights |= b
	case chess.WhitePawn:
		p.Board.WhitePawns |= b

	case chess.BlackKing:
		p.Board.BlackKing |= b
	case chess.BlackQueen:
		p.Board.BlackQueens |= b
	case chess.BlackRook:
		p.Board.BlackRooks |= b
	case chess.BlackBishop:
		p.Board.BlackBishops |= b
	case chess.BlackKnight:
		p.Board.BlackKnights |= b
	case chess.BlackPawn:
		p.Board.BlackPawns |= b

	default:
		panic("unsupported piece type")
	}
}

func newPosition(
	side chess.Color,
	pieces map[chess.PieceType][]int,
) chess.Position {
	var p chess.Position

	p.SideToMove = side
	p.EnPassant = -1

	for piece, squares := range pieces {
		for _, square := range squares {
			setPiece(&p, piece, square)
		}
	}

	return p
}

func TestHistoryMovePerformance(t *testing.T) {
	fens := map[string]string{
		"start":      chess.StartingFEN,
		"middlegame": "r1bq1rk1/ppp2ppp/2np1n2/2b1p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 7",
		"kiwipete":   "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
	}

	const depth = 4

	run := func(p chess.Position, useHistory bool) (int, uint64) {
		old := UseHistory
		UseHistory = useHistory
		defer func() { UseHistory = old }()

		reset(t)

		score, ok := negamax(&p, depth, 0, -Infinity, Infinity, time.Time{}, &SearchState{})
		if !ok {
			t.Fatal("search did not complete")
		}

		return score, Nodes
	}

	for name, fen := range fens {
		pos, err := chess.ParseFEN(fen)
		if err != nil {
			t.Fatalf("%s: bad FEN: %v", name, err)
		}

		offScore, offNodes := run(pos, false)
		onScore, onNodes := run(pos, true)

		t.Logf("%s: history disabled nodes: %d", name, offNodes)
		t.Logf("%s: history enabled  nodes: %d", name, onNodes)

		if onScore != offScore {
			t.Fatalf("%s: history changed the score: enabled=%d disabled=%d", name, onScore, offScore)
		}
	}
}

func TestHistoryTableUpdate(t *testing.T) {
	var h HistoryTable

	m := chess.Move{From: 12, To: 28, Piece: chess.WhitePawn}

	h.update(chess.White, m, 3)

	if got := h.get(chess.White, m); got != 9 {
		t.Fatalf("history = %d, want 9", got)
	}

	if got := h.get(chess.Black, m); got != 0 {
		t.Fatalf("history leaked to other side: %d", got)
	}

	for i := 0; i < 1000; i++ {
		h.update(chess.White, m, 10)
	}

	if got := h.get(chess.White, m); got != maxHistory {
		t.Fatalf("history = %d, want cap %d", got, maxHistory)
	}
}

func withFlags(ordering, history, pvs bool) func() {
	oldO, oldH, oldP := UseMoveOrdering, UseHistory, UsePVS
	UseMoveOrdering, UseHistory, UsePVS = ordering, history, pvs

	return func() { UseMoveOrdering, UseHistory, UsePVS = oldO, oldH, oldP }
}

var benchFENs = []struct{ name, fen string }{
	{"start", chess.StartingFEN},
	{"middlegame", "r1bq1rk1/ppp2ppp/2np1n2/2b1p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 7"},
	{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"},
}

// searchScore runs a fresh full-window search and returns score and nodes.
func searchScore(t *testing.T, p chess.Position, depth int, ordering, history, pvs bool) (int, uint64) {
	t.Helper()

	reset(t)

	// reset forces ordering on, so flags are applied after it.
	restore := withFlags(ordering, history, pvs)
	defer restore()

	score, ok := negamax(&p, depth, 0, -Infinity, Infinity, time.Time{}, &SearchState{})
	if !ok {
		t.Fatal("search did not complete")
	}

	return score, Nodes
}

func TestPVSMatchesBruteForce(t *testing.T) {
	for _, side := range []chess.Color{chess.White, chess.Black} {
		for depth := 1; depth <= 3; depth++ {
			for _, pvs := range []bool{false, true} {
				reset(t)
				want := brute(fromRows(side, sparseRows), depth, 0)

				got, _ := searchScore(t, *fromRows(side, sparseRows), depth, true, true, pvs)
				ok := true

				if !ok || got != want {
					t.Fatalf("side=%v depth=%d pvs=%v: negamax=%d brute=%d ok=%v", side, depth, pvs, got, want, ok)
				}
			}
		}
	}
}

func TestPVSMatchesNormalSearch(t *testing.T) {
	for _, tc := range benchFENs {
		pos, err := chess.ParseFEN(tc.fen)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		offScore, _ := searchScore(t, pos, 4, true, true, false)
		onScore, _ := searchScore(t, pos, 4, true, true, true)

		if onScore != offScore {
			t.Fatalf("%s: PVS changed score: pvs=%d normal=%d", tc.name, onScore, offScore)
		}
	}
}

func TestPVSPerformance(t *testing.T) {
	var offTotal, onTotal uint64

	for _, tc := range benchFENs {
		pos, err := chess.ParseFEN(tc.fen)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		offScore, offNodes := searchScore(t, pos, 4, true, true, false)
		onScore, onNodes := searchScore(t, pos, 4, true, true, true)

		t.Logf("%s: PVS disabled nodes: %d", tc.name, offNodes)
		t.Logf("%s: PVS enabled  nodes: %d", tc.name, onNodes)

		if onScore != offScore {
			t.Fatalf("%s: PVS changed score: pvs=%d normal=%d", tc.name, onScore, offScore)
		}

		offTotal += offNodes
		onTotal += onNodes
	}

	t.Logf("total: PVS disabled %d, PVS enabled %d", offTotal, onTotal)
}

type orderCfg struct {
	name                     string
	ordering, killers, histo bool
}

// searchScoreCfg runs a fresh full-window search (PVS off, so only
// the ordering heuristics differ) and returns score and nodes.
func searchScoreCfg(t *testing.T, p chess.Position, depth int, c orderCfg) (int, uint64) {
	t.Helper()

	reset(t)

	oldO, oldK, oldH, oldP := UseMoveOrdering, UseKillers, UseHistory, UsePVS
	UseMoveOrdering, UseKillers, UseHistory, UsePVS = c.ordering, c.killers, c.histo, false

	defer func() { UseMoveOrdering, UseKillers, UseHistory, UsePVS = oldO, oldK, oldH, oldP }()

	score, ok := negamax(&p, depth, 0, -Infinity, Infinity, time.Time{}, &SearchState{})
	if !ok {
		t.Fatal("search did not complete")
	}

	return score, Nodes
}

func TestKillerTableStore(t *testing.T) {
	var k KillerTable

	a := chess.Move{From: 12, To: 28, Piece: chess.WhitePawn}
	b := chess.Move{From: 6, To: 21, Piece: chess.WhiteKnight}
	c := chess.Move{From: 1, To: 18, Piece: chess.WhiteKnight}

	k.store(3, a)
	k.store(3, a) // duplicate must not push itself into slot 1
	if k.isKiller(3, a) != 0 || k.isKiller(3, b) != -1 {
		t.Fatal("first killer should sit in slot 0 only")
	}

	k.store(3, b)
	if k.isKiller(3, b) != 0 || k.isKiller(3, a) != 1 {
		t.Fatal("new killer should shift the old one to slot 1")
	}

	k.store(3, c)
	if k.isKiller(3, c) != 0 || k.isKiller(3, b) != 1 || k.isKiller(3, a) != -1 {
		t.Fatal("only the two most recent killers should be kept")
	}

	if k.isKiller(4, c) != -1 {
		t.Fatal("killers must be per ply")
	}

	// Out of range plies must be ignored, not panic.
	k.store(-1, a)
	k.store(MaxPly, a)

	if k.isKiller(MaxPly, a) != -1 || k.isKiller(-1, a) != -1 {
		t.Fatal("out of range ply must report no killer")
	}
}

func TestKillerOrderingRanksBetweenCapturesAndQuiet(t *testing.T) {
	state := &SearchState{}

	quiet := chess.Move{From: 8, To: 16, Piece: chess.WhitePawn}
	killer := chess.Move{From: 9, To: 17, Piece: chess.WhitePawn}
	lowCapture := chess.Move{From: 3, To: 51, Piece: chess.WhiteQueen, Captured: chess.BlackPawn, Flags: chess.Capture}
	promo := chess.Move{From: 48, To: 56, Piece: chess.WhitePawn, Promotion: chess.WhiteKnight, Flags: chess.Promotion}
	capture := chess.Move{From: 10, To: 35, Piece: chess.WhitePawn, Captured: chess.BlackQueen, Flags: chess.Capture}

	state.killers.store(2, killer)
	// Saturate history on the quiet move to ensure maxHistory never outranks captures/promotions or killers.
	state.history.update(chess.White, quiet, 100)

	moves := []chess.Move{quiet, killer, promo, lowCapture, capture}

	oldK, oldH := UseKillers, UseHistory
	UseKillers, UseHistory = true, true

	defer func() { UseKillers, UseHistory = oldK, oldH }()

	orderMovesHeuristic(moves, chess.White, 2, state)

	if !sameMove(moves[0], capture) ||
		!sameMove(moves[1], promo) ||
		!sameMove(moves[2], lowCapture) ||
		!sameMove(moves[3], killer) ||
		!sameMove(moves[4], quiet) {
		t.Fatalf("want capture, promo, lowCapture, killer, quiet; got %+v", moves)
	}

	// Different ply: killer no longer applies.
	moves = []chess.Move{killer, quiet, capture}
	orderMovesHeuristic(moves, chess.White, 5, state)

	if !sameMove(moves[1], quiet) {
		t.Fatalf("killer must not apply at another ply; got %+v", moves)
	}
}

// TestKillerMovePerformance measures killers alone against the same
// ordering without them. Scores must be identical.
func TestKillerMovePerformance(t *testing.T) {
	for _, tc := range benchFENs {
		pos, err := chess.ParseFEN(tc.fen)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		offScore, offNodes := searchScoreCfg(t, pos, 4, orderCfg{"captures", true, false, false})
		onScore, onNodes := searchScoreCfg(t, pos, 4, orderCfg{"captures+killers", true, true, false})

		t.Logf("%s: killers disabled nodes: %d", tc.name, offNodes)
		t.Logf("%s: killers enabled  nodes: %d", tc.name, onNodes)

		if onScore != offScore {
			t.Fatalf("%s: killers changed score: on=%d off=%d", tc.name, onScore, offScore)
		}
	}
}

// TestOrderingContributions measures each ordering stage on its own.
func TestOrderingContributions(t *testing.T) {
	configs := []orderCfg{
		{"no ordering", false, false, false},
		{"captures", true, false, false},
		{"captures+killers", true, true, false},
		{"captures+history", true, false, true},
		{"all", true, true, true},
	}

	for _, tc := range benchFENs {
		pos, err := chess.ParseFEN(tc.fen)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		var want int

		haveWant := false

		for _, c := range configs {
			// Unordered search explodes on tactical positions, so the
			// no-ordering baseline only runs on the start position.
			if !c.ordering && tc.name != "start" {
				continue
			}

			score, nodes := searchScoreCfg(t, pos, 4, c)

			t.Logf("%-11s %-17s nodes: %d", tc.name, c.name, nodes)

			if !haveWant {
				want, haveWant = score, true
			} else if score != want {
				t.Fatalf("%s/%s: score %d != baseline %d", tc.name, c.name, score, want)
			}
		}
	}
}

// lmrRun searches with every other heuristic on and only LMR varied.
func lmrRun(t *testing.T, p chess.Position, depth int, lmr bool) (int, uint64) {
	t.Helper()

	reset(t)

	oldL := UseLMR
	UseLMR = lmr

	defer func() { UseLMR = oldL }()

	score, ok := negamax(&p, depth, 0, -Infinity, Infinity, time.Time{}, &SearchState{})
	if !ok {
		t.Fatal("search did not complete")
	}

	return score, Nodes
}

func TestLMRReductionRules(t *testing.T) {
	old := UseLMR
	defer func() { UseLMR = old }()

	UseLMR = true

	for index := 0; index < lmrFullMoves; index++ {
		if lmrReduction(index, 6, true) != 0 {
			t.Fatalf("early move %d must not be reduced", index)
		}
	}

	if lmrReduction(lmrFullMoves, 6, false) != 0 {
		t.Fatal("captures/promotions must not be reduced")
	}

	if lmrReduction(lmrFullMoves, lmrMinChildDepth-1, true) != 0 {
		t.Fatal("shallow moves must not be reduced")
	}

	if got := lmrReduction(lmrFullMoves, 6, true); got != 1 {
		t.Fatalf("late move reduction = %d, want 1", got)
	}

	if got := lmrReduction(2*lmrFullMoves, 6, true); got != 2 {
		t.Fatalf("very late move reduction = %d, want 2", got)
	}

	for depth := lmrMinChildDepth; depth < 8; depth++ {
		if got := lmrReduction(100, depth, true); depth-got < 1 {
			t.Fatalf("depth %d reduced by %d, must leave at least 1 ply", depth, got)
		}
	}

	UseLMR = false

	if lmrReduction(100, 6, true) != 0 {
		t.Fatal("no reduction when LMR is off")
	}
}

// With LMR off the search is exact, so it must still agree with brute force.
func TestLMRDisabledMatchesBruteForce(t *testing.T) {
	for _, side := range []chess.Color{chess.White, chess.Black} {
		for depth := 1; depth <= 3; depth++ {
			reset(t)

			want := brute(fromRows(side, sparseRows), depth, 0)
			got, _ := lmrRun(t, *fromRows(side, sparseRows), depth, false)

			if got != want {
				t.Fatalf("side=%v depth=%d: negamax=%d brute=%d", side, depth, got, want)
			}
		}
	}
}

// LMR is lossy in general, but it must never lose a forced mate.
func TestLMRStillFindsMates(t *testing.T) {
	tests := []struct {
		name  string
		fen   string
		depth int
	}{
		{"mate in 1", "6k1/5ppp/8/8/8/8/5PPP/3R2K1 w - - 0 1", 3},
		{"mate in 2", "kbK5/pp6/1P6/8/8/8/8/R7 w - - 0 1", 5},
	}

	for _, tt := range tests {
		pos, err := chess.ParseFEN(tt.fen)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}

		offScore, _ := lmrRun(t, pos, tt.depth, false)
		onScore, _ := lmrRun(t, pos, tt.depth, true)

		if offScore < MateScore-1000 {
			t.Fatalf("%s: baseline did not find the mate (score %d)", tt.name, offScore)
		}

		if onScore != offScore {
			t.Fatalf("%s: LMR changed mate score: on=%d off=%d", tt.name, onScore, offScore)
		}
	}
}

// LMR may change scores slightly. Log the difference and require it
// to stay small so a broken reduction is caught.
func TestLMRScoreDeviation(t *testing.T) {
	const maxDeviation = 150 // centipawns

	for _, tc := range benchFENs {
		pos, err := chess.ParseFEN(tc.fen)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		offScore, _ := lmrRun(t, pos, 5, false)
		onScore, _ := lmrRun(t, pos, 5, true)

		diff := onScore - offScore
		if diff < 0 {
			diff = -diff
		}

		t.Logf("%s: LMR off score %d, LMR on score %d (diff %d)", tc.name, offScore, onScore, diff)

		if diff > maxDeviation {
			t.Fatalf("%s: LMR score deviates by %d cp (max %d)", tc.name, diff, maxDeviation)
		}
	}
}

func TestLMRPerformance(t *testing.T) {
	var offTotal, onTotal uint64

	for _, tc := range benchFENs {
		pos, err := chess.ParseFEN(tc.fen)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		_, offNodes := lmrRun(t, pos, 5, false)
		_, onNodes := lmrRun(t, pos, 5, true)

		t.Logf("%s: LMR disabled nodes: %d", tc.name, offNodes)
		t.Logf("%s: LMR enabled  nodes: %d", tc.name, onNodes)

		offTotal += offNodes
		onTotal += onNodes
	}

	t.Logf("total: LMR disabled %d, LMR enabled %d", offTotal, onTotal)
}

const underpromotionMateFEN = "1nb5/Pnk1K3/2p5/8/8/8/8/8 w - - 0 1"

func TestUnderpromotionToKnightMate(t *testing.T) {
	pos, err := chess.ParseFEN(underpromotionMateFEN)
	if err != nil {
		t.Fatalf("ParseFEN: %v", err)
	}

	promotions := map[chess.PieceType]bool{}

	for _, m := range allMoves(&pos) {
		if m.From == sq("a7") && m.To == sq("a8") {
			promotions[m.Promotion] = true
		}
	}

	for _, want := range []chess.PieceType{chess.WhiteQueen, chess.WhiteRook, chess.WhiteBishop, chess.WhiteKnight} {
		if !promotions[want] {
			t.Fatalf("a7-a8 promotion to piece %v was not generated (got %v)", want, promotions)
		}
	}

	var knight chess.Move

	for _, m := range allMoves(&pos) {
		if m.From == sq("a7") && m.To == sq("a8") && m.Promotion == chess.WhiteKnight {
			knight = m
		}
	}

	undo := pos.MakeMove(knight)
	mated := pos.InCheck(chess.Black) && len(allMoves(&pos)) == 0
	pos.UnmakeMove(knight, undo)

	if !mated {
		t.Fatal("a8=N should be checkmate")
	}

	for depth := 1; depth <= 5; depth++ {
		p, _ := chess.ParseFEN(underpromotionMateFEN)

		reset(t)

		UseLMR = true

		move, ok := Search(&p, depth, 0)
		nodes := Nodes

		if !ok {
			t.Fatalf("depth %d: no move found", depth)
		}

		p2, _ := chess.ParseFEN(underpromotionMateFEN)

		reset(t)

		score, _ := negamax(&p2, depth, 0, -Infinity, Infinity, time.Time{}, &SearchState{})

		t.Logf("depth %d: %s%s promotion=%v score=%d nodes=%d",
			depth, SquareName(move.From), SquareName(move.To), move.Promotion, score, nodes)

		if move.From != sq("a7") || move.To != sq("a8") {
			t.Fatalf("depth %d: played %s%s, want a7a8", depth, SquareName(move.From), SquareName(move.To))
		}

		if move.Promotion != chess.WhiteKnight {
			t.Fatalf("depth %d: promoted to %v, want knight", depth, move.Promotion)
		}

		if score != MateScore-1 {
			t.Fatalf("depth %d: score %d, want mate in 1 (%d)", depth, score, MateScore-1)
		}
	}
}

func TestQueenPromotionIsNotMate(t *testing.T) {
	pos, err := chess.ParseFEN(underpromotionMateFEN)
	if err != nil {
		t.Fatalf("ParseFEN: %v", err)
	}

	for _, m := range allMoves(&pos) {
		if m.From != sq("a7") || m.To != sq("a8") || m.Promotion != chess.WhiteQueen {
			continue
		}

		undo := pos.MakeMove(m)
		mated := pos.InCheck(chess.Black) && len(allMoves(&pos)) == 0
		pos.UnmakeMove(m, undo)

		if mated {
			t.Fatal("a8=Q must not be checkmate in this position")
		}

		return
	}

	t.Fatal("a8=Q was not generated")
}
