package engine

import (
	"fmt"
	"testing"
	"time"

	"vajra/internal/chess"
)

// The reference quiescence search is intentionally full-width.
// Memoization prevents identical tactical positions reached through
// different capture orders from being searched repeatedly.
const bruteQMaxPly = 32

type bruteQKey struct {
	fen string
	ply int
}

const maxBruteQNodes = 1_000_000

var bruteQCache map[string]int
var bruteQNodes int

func bruteQ(p *chess.Position, ply int) int {
	if bruteQCache == nil {
		bruteQCache = make(map[string]int)
	}
	bruteQNodes++

	if bruteQNodes > maxBruteQNodes {
		panic(fmt.Sprintf(
			"bruteQ exceeded %d nodes at ply %d, position=%s",
			maxBruteQNodes,
			ply,
			p.FEN(),
		))
	}

	// Ply is part of the key because mate scores contain ply:
	// -MateScore + ply
	key := fmt.Sprintf("%s|ply=%d", p.FEN(), ply)

	if score, ok := bruteQCache[key]; ok {
		return score
	}

	moves := allMoves(p)

	// No legal moves:
	// checkmate or stalemate.
	if len(moves) == 0 {
		if p.InCheck(p.SideToMove) {
			score := -MateScore + ply
			bruteQCache[key] = score
			return score
		}

		bruteQCache[key] = 0
		return 0
	}

	best := evalSide(p)

	// Only tactical moves belong in q-search.
	for _, m := range moves {
		if !isQuiescenceMove(m) {
			continue
		}

		undo := p.MakeMove(m)

		score := -bruteQ(p, ply+1)

		p.UnmakeMove(m, undo)

		if score > best {
			best = score
		}
	}

	bruteQCache[key] = best

	return best
}

func resetBruteQ() {
	bruteQCache = make(map[string]int)
	bruteQNodes = 0
}

func bruteQMemo(
	p *chess.Position,
	ply int,
	memo map[bruteQKey]int,
) int {
	// Hard safety limit for the reference implementation.
	//
	// Normal capture sequences should terminate much earlier because
	// captures remove material from the board.
	if ply >= bruteQMaxPly {
		return evalSide(p)
	}

	key := bruteQKey{
		fen: p.FEN(),
		ply: ply,
	}

	if score, ok := memo[key]; ok {
		return score
	}

	moves := allMoves(p)

	// Checkmate / stalemate.
	if len(moves) == 0 {
		var score int

		if p.InCheck(p.SideToMove) {
			score = -MateScore + ply
		} else {
			score = 0
		}

		memo[key] = score
		return score
	}

	// If the side to move is in check, stand-pat is illegal.
	// Every legal move must therefore be searched.
	if p.InCheck(p.SideToMove) {
		best := -Infinity

		for _, move := range moves {
			undo := p.MakeMove(move)

			score := -bruteQMemo(
				p,
				ply+1,
				memo,
			)

			p.UnmakeMove(move, undo)

			if score > best {
				best = score
			}
		}

		memo[key] = best
		return best
	}

	// Stand-pat.
	best := evalSide(p)

	// Search captures and promotions only.
	for _, move := range moves {
		if !isQuiescenceMove(move) {
			continue
		}

		undo := p.MakeMove(move)

		score := -bruteQMemo(
			p,
			ply+1,
			memo,
		)

		p.UnmakeMove(move, undo)

		if score > best {
			best = score
		}
	}

	memo[key] = best

	return best
}

func TestQuiescenceStandPat(t *testing.T) {
	reset(t)

	p, err := chess.ParseFEN(
		"8/8/8/8/8/8/4K3/7k w - - 0 1",
	)
	if err != nil {
		t.Fatal(err)
	}

	score, completed := quiescence(
		&p,
		0,
		-Infinity,
		Infinity,
		time.Time{},
	)

	if !completed {
		t.Fatal("quiescence did not complete")
	}

	if want := evalSide(&p); score != want {
		t.Fatalf("got score %d, want stand-pat %d", score, want)
	}
}

func TestQuiescenceFindsWinningCapture(t *testing.T) {
	p, err := chess.ParseFEN(
		"4k3/8/8/8/8/3q4/4Q3/4K3 w - - 0 1",
	)
	if err != nil {
		t.Fatal(err)
	}

	Nodes = 0

	score, completed := quiescence(
		&p,
		0,
		-Infinity,
		Infinity,
		time.Time{},
	)

	if !completed {
		t.Fatal("quiescence did not complete")
	}

	// White can capture the black queen.
	if score < 900 {
		t.Fatalf(
			"expected tactical score >= 900, got %d",
			score,
		)
	}
}

func TestQuiescenceRespectsDeadline(t *testing.T) {
	p, err := chess.ParseFEN(chess.StartingFEN)
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(-time.Second)

	_, completed := quiescence(
		&p,
		0,
		-Infinity,
		Infinity,
		deadline,
	)

	if completed {
		t.Fatal("expected quiescence to stop on expired deadline")
	}
}

func TestQuiescenceRestoresPosition(t *testing.T) {
	p, err := chess.ParseFEN(
		"4k3/8/8/8/3q4/8/4Q3/4K3 w - - 0 1",
	)
	if err != nil {
		t.Fatal(err)
	}

	before := p.FEN()

	_, completed := quiescence(
		&p,
		0,
		-Infinity,
		Infinity,
		time.Time{},
	)

	if !completed {
		t.Fatal("quiescence did not complete")
	}

	after := p.FEN()

	if before != after {
		t.Fatalf(
			"position changed after quiescence\nbefore: %s\nafter:  %s",
			before,
			after,
		)
	}
}

var quiescenceRows = []string{
	"r...k..r",
	"ppp..ppp",
	"..n.b...",
	"...q....",
	"....N...",
	"..N.B...",
	"PPP..PPP",
	"R..QK..R",
}

func TestIsQuiescenceMove(t *testing.T) {
	tests := []struct {
		name string
		move chess.Move
		want bool
	}{
		{
			name: "quiet",
			move: chess.Move{
				From:  8,
				To:    16,
				Piece: chess.WhitePawn,
			},
			want: false,
		},
		{
			name: "capture",
			move: chess.Move{
				Flags: chess.Capture,
			},
			want: true,
		},
		{
			name: "promotion",
			move: chess.Move{
				Flags: chess.Promotion,
			},
			want: true,
		},
		{
			name: "capture promotion",
			move: chess.Move{
				Flags: chess.Capture | chess.Promotion,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isQuiescenceMove(tt.move); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQuiescenceTerminalScores(t *testing.T) {
	t.Run("checkmate scores relative to ply", func(t *testing.T) {
		reset(t)

		p := fromRows(chess.Black, blackMatedRows)

		for _, ply := range []int{0, 2, 7} {
			score, ok := quiescence(
				p,
				ply,
				-Infinity,
				Infinity,
				time.Time{},
			)

			if !ok || score != -MateScore+ply {
				t.Fatalf(
					"ply %d: got %d ok=%v",
					ply,
					score,
					ok,
				)
			}
		}
	})

	t.Run("stalemate scores zero", func(t *testing.T) {
		reset(t)

		p := fromRows(chess.Black, stalemateRows)

		score, ok := quiescence(
			p,
			0,
			-Infinity,
			Infinity,
			time.Time{},
		)

		if !ok || score != 0 {
			t.Fatalf("got %d ok=%v", score, ok)
		}
	})

	t.Run("expired deadline aborts", func(t *testing.T) {
		reset(t)

		p := fromRows(chess.White, quiescenceRows)

		score, ok := quiescence(
			p,
			0,
			-Infinity,
			Infinity,
			time.Now().Add(-time.Second),
		)

		if ok || score != 0 {
			t.Fatalf(
				"got (%d,%v), want (0,false)",
				score,
				ok,
			)
		}
	})
}

func TestQuiescenceTactics(t *testing.T) {
	t.Run("captures a hanging queen", func(t *testing.T) {
		reset(t)

		p := fromRows(chess.White, []string{
			"q......k",
			"........",
			"........",
			"........",
			"........",
			"........",
			"........",
			"R......K",
		})

		score, ok := quiescence(
			p,
			0,
			-Infinity,
			Infinity,
			time.Time{},
		)

		if !ok || score <= evalSide(p) {
			t.Fatalf(
				"got %d, want more than stand pat %d",
				score,
				evalSide(p),
			)
		}
	})

	t.Run("declines a poisoned pawn", func(t *testing.T) {
		reset(t)

		p := fromRows(chess.White, []string{
			"...r...k",
			"........",
			"........",
			"...p....",
			"........",
			"........",
			"........",
			"...Q..K.",
		})

		score, ok := quiescence(
			p,
			0,
			-Infinity,
			Infinity,
			time.Time{},
		)

		if !ok || score != evalSide(p) {
			t.Fatalf(
				"got %d, want stand pat %d",
				score,
				evalSide(p),
			)
		}
	})

	t.Run("searches promotions", func(t *testing.T) {
		reset(t)

		p := fromRows(chess.White, []string{
			"........",
			"P.......",
			"........",
			"........",
			"........",
			".......k",
			"........",
			"....K...",
		})

		score, ok := quiescence(
			p,
			0,
			-Infinity,
			Infinity,
			time.Time{},
		)

		if !ok || score <= evalSide(p) {
			t.Fatalf(
				"got %d, want more than stand pat %d",
				score,
				evalSide(p),
			)
		}
	})

	t.Run("sees recapture sequences", func(t *testing.T) {
		reset(t)

		p := fromRows(chess.White, quiescenceRows)

		resetBruteQ()
		want := bruteQ(p, 0)

		score, ok := quiescence(
			p, 0, -Infinity, Infinity, time.Time{},
		)

		if !ok || score != want {
			t.Fatalf(
				"got %d, want %d, brute nodes=%d",
				score,
				want,
				bruteQNodes,
			)
		}
	})
}

func TestQuiescenceMatchesBruteForce(t *testing.T) {
	tests := []struct {
		name string
		side chess.Color
		rows []string
	}{
		{
			name: "capture rich white",
			side: chess.White,
			rows: quiescenceRows,
		},
		{
			name: "capture rich black",
			side: chess.Black,
			rows: quiescenceRows,
		},
		{
			name: "start white",
			side: chess.White,
			rows: startRows,
		},
		{
			name: "sparse white",
			side: chess.White,
			rows: sparseRows,
		},
		{
			name: "sparse black",
			side: chess.Black,
			rows: sparseRows,
		},
		{
			name: "rook wins knight white",
			side: chess.White,
			rows: rookWinsKnightWhiteRows,
		},
		{
			name: "rook wins knight black",
			side: chess.Black,
			rows: rookWinsKnightBlackRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reset(t)

			p := fromRows(tt.side, tt.rows)

			resetBruteQ()
			want := bruteQ(p, 0)

			got, ok := quiescence(
				p, 0, -Infinity, Infinity, time.Time{},
			)

			if !ok || got != want {
				t.Fatalf(
					"quiescence %d, brute force %d, brute nodes=%d",
					got,
					want,
					bruteQNodes,
				)
			}
		})
	}

	t.Run("move ordering does not change the score", func(t *testing.T) {
		reset(t)

		p := fromRows(chess.White, quiescenceRows)

		UseMoveOrdering = true

		withOrdering, ok := quiescence(
			p,
			0,
			-Infinity,
			Infinity,
			time.Time{},
		)

		if !ok {
			t.Fatal("quiescence with ordering did not complete")
		}

		UseMoveOrdering = false

		withoutOrdering, ok := quiescence(
			p,
			0,
			-Infinity,
			Infinity,
			time.Time{},
		)

		if !ok {
			t.Fatal("quiescence without ordering did not complete")
		}

		if withOrdering != withoutOrdering {
			t.Fatalf(
				"%d with ordering, %d without",
				withOrdering,
				withoutOrdering,
			)
		}
	})
}

func TestQuiescenceBookkeeping(t *testing.T) {
	t.Run("restores position", func(t *testing.T) {
		reset(t)

		p := fromRows(chess.White, quiescenceRows)

		snapshot := *p

		quiescence(
			p,
			0,
			-Infinity,
			Infinity,
			time.Time{},
		)

		if !samePosition(snapshot, *p) {
			t.Fatal("position not restored")
		}
	})

	t.Run("restores position after timeout", func(t *testing.T) {
		reset(t)

		p := fromRows(chess.White, quiescenceRows)

		snapshot := *p

		quiescence(
			p,
			0,
			-Infinity,
			Infinity,
			time.Now().Add(-time.Second),
		)

		if !samePosition(snapshot, *p) {
			t.Fatal("position not restored after timeout")
		}
	})

	t.Run("counts nodes", func(t *testing.T) {
		reset(t)

		p := fromRows(chess.White, quiescenceRows)

		quiescence(
			p,
			0,
			-Infinity,
			Infinity,
			time.Time{},
		)

		if Nodes < 2 {
			t.Fatalf(
				"expected captures to be searched, nodes=%d",
				Nodes,
			)
		}
	})

	t.Run("quiet position counts one node", func(t *testing.T) {
		reset(t)

		p := fromRows(chess.White, queenUpRows)

		quiescence(
			p,
			0,
			-Infinity,
			Infinity,
			time.Time{},
		)

		if Nodes != 1 {
			t.Fatalf("nodes=%d, want 1", Nodes)
		}
	})

	t.Run("does not touch history", func(t *testing.T) {
		reset(t)

		history = []uint64{1, 2, 3}

		p := fromRows(chess.White, quiescenceRows)

		quiescence(
			p,
			0,
			-Infinity,
			Infinity,
			time.Time{},
		)

		if len(history) != 3 {
			t.Fatalf(
				"history length %d, want 3",
				len(history),
			)
		}
	})
}

func TestNegamaxDepthZeroUsesQuiescence(t *testing.T) {
	reset(t)

	p := fromRows(chess.White, quiescenceRows)
	resetBruteQ()

	want := bruteQ(p, 0)

	got, ok := negamax(
		p,
		0,
		0,
		-Infinity,
		Infinity,
		time.Time{},
		&SearchState{},
	)

	if !ok || got != want {
		t.Fatalf(
			"got %d, want %d",
			got,
			want,
		)
	}

	if got == evalSide(p) {
		t.Fatal(
			"depth zero returned the raw static eval, " +
				"so quiescence did not run",
		)
	}
}
