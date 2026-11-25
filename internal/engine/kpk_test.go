package engine

import (
	"testing"

	"vajra/internal/chess"
)

func kpkPosition(t *testing.T, weakToMove, weakKing, strongKing, pawn int) chess.Position {
	t.Helper()

	p := chess.Position{SideToMove: chess.White, EnPassant: -1, FullmoveNumber: 1}

	p.Board.SetPiece(chess.WhiteKing, strongKing)
	p.Board.SetPiece(chess.BlackKing, weakKing)
	p.Board.SetPiece(chess.WhitePawn, pawn)

	if weakToMove == 1 {
		p.SideToMove = chess.Black
	}

	return p
}

func mirrorColours(p chess.Position) chess.Position {
	m := chess.Position{SideToMove: chess.Black, EnPassant: -1, FullmoveNumber: 1}

	if p.SideToMove == chess.Black {
		m.SideToMove = chess.White
	}

	for sq := 0; sq < 64; sq++ {
		piece := p.Board.PieceAt(sq)

		switch piece.Type {
		case chess.WhiteKing:
			m.Board.SetPiece(chess.BlackKing, sq^56)
		case chess.BlackKing:
			m.Board.SetPiece(chess.WhiteKing, sq^56)
		case chess.WhitePawn:
			m.Board.SetPiece(chess.BlackPawn, sq^56)
		case chess.BlackPawn:
			m.Board.SetPiece(chess.WhitePawn, sq^56)
		}
	}

	return m
}

func TestKPKKnownPositions(t *testing.T) {
	tests := []struct {
		name string
		fen  string
		win  bool
	}{
		{"king on the key square wins, white to move", "4k3/8/4K3/4P3/8/8/8/8 w - - 0 1", true},
		{"king on the key square wins, black to move", "4k3/8/4K3/4P3/8/8/8/8 b - - 0 1", true},
		{"rook pawn with the defender in the corner", "k7/8/8/8/8/8/P7/K7 w - - 0 1", false},
		{"defending king in front of the pawn", "8/8/8/8/8/4k3/4P3/4K3 w - - 0 1", false},
		{"pawn on the seventh promotes safely", "8/3P4/8/8/8/8/k7/4K3 w - - 0 1", true},
		{"pawn on the seventh is captured", "3k4/3P4/3K4/8/8/8/8/8 b - - 0 1", false},
		{"black pawn wins by the same rules", "8/8/8/8/8/8/4p3/4K1k1 b - - 0 1", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := prodPos(t, tc.fen)

			_, win, ok := kpkLookup(&p)
			if !ok {
				t.Fatal("not recognised as king and pawn against king")
			}

			if win != tc.win {
				t.Fatalf("win = %v, want %v", win, tc.win)
			}
		})
	}
}

func TestKPKOnlyAppliesToKingAndOnePawn(t *testing.T) {
	for _, fen := range []string{
		chess.StartingFEN,
		"4k3/8/8/8/8/8/8/4K3 w - - 0 1",
		"4k3/8/8/8/8/8/PP6/4K3 w - - 0 1",
		"4k3/p7/8/8/8/8/P7/4K3 w - - 0 1",
		"4k3/8/8/8/8/8/P7/3NK3 w - - 0 1",
		"4k3/8/8/8/8/8/P7/3RK3 w - - 0 1",
		"4k3/8/8/8/8/8/P7/3QK3 w - - 0 1",
		"4k3/8/8/8/8/8/P7/3BK3 w - - 0 1",
	} {
		p := prodPos(t, fen)

		if _, _, ok := kpkLookup(&p); ok {
			t.Fatalf("%s must not use the KPK table", fen)
		}

		if _, ok := kpkScore(&p); ok {
			t.Fatalf("%s must not get a KPK score", fen)
		}
	}
}

func TestKPKScore(t *testing.T) {
	win := prodPos(t, "4k3/8/4K3/4P3/8/8/8/8 w - - 0 1")

	if got := Evaluate(&win); got < kpkBaseScore {
		t.Fatalf("a won white position scores %d, want at least %d", got, kpkBaseScore)
	}

	draw := prodPos(t, "k7/8/8/8/8/8/P7/K7 w - - 0 1")

	if got := Evaluate(&draw); got != 0 {
		t.Fatalf("a drawn position scores %d, want 0", got)
	}

	far := prodPos(t, "8/3P4/8/8/8/8/k7/4K3 w - - 0 1")
	near := prodPos(t, "8/8/8/8/3P4/8/k7/4K3 w - - 0 1")

	if Evaluate(&far) <= Evaluate(&near) {
		t.Fatal("a more advanced winning pawn must score higher")
	}

	if Evaluate(&win) >= MateScore-1000 {
		t.Fatal("a KPK win must stay outside the mate range")
	}
}

func TestKPKColourSymmetry(t *testing.T) {
	checked := 0

	for idx := 0; idx < kpkSize; idx += 13 {
		weakToMove, weakKing, strongKing, pawn := kpkDecode(idx)

		kpkOnce.Do(kpkBuild)

		if kpkResults[idx] == kpkInvalid {
			continue
		}

		p := kpkPosition(t, weakToMove, weakKing, strongKing, pawn)
		m := mirrorColours(p)

		if a, b := Evaluate(&p), Evaluate(&m); a != -b {
			t.Fatalf("%s evaluates to %d but its colour mirror to %d", p.FEN(), a, b)
		}

		checked++
	}

	if checked < 1000 {
		t.Fatalf("only %d positions checked", checked)
	}
}

func TestKPKFilesMirror(t *testing.T) {
	kpkOnce.Do(kpkBuild)

	for idx := 0; idx < kpkSize; idx += 11 {
		if kpkResults[idx] == kpkInvalid {
			continue
		}

		weakToMove, weakKing, strongKing, pawn := kpkDecode(idx)

		p := kpkPosition(t, weakToMove, weakKing, strongKing, pawn)
		flipped := kpkPosition(t, weakToMove, weakKing^7, strongKing^7, pawn^7)

		_, w1, ok1 := kpkLookup(&p)
		_, w2, ok2 := kpkLookup(&flipped)

		if !ok1 || !ok2 || w1 != w2 {
			t.Fatalf("%s and its left-right mirror disagree", p.FEN())
		}
	}
}

func TestKPKIsAFixedPointOfTheRealMoveGenerator(t *testing.T) {
	kpkOnce.Do(kpkBuild)

	wins := func(p *chess.Position) bool {
		_, win, ok := kpkLookup(p)

		return ok && win
	}

	checked, winCount := 0, 0

	for idx := 0; idx < kpkSize; idx += 5 {
		if kpkResults[idx] == kpkInvalid {
			continue
		}

		weakToMove, weakKing, strongKing, pawn := kpkDecode(idx)

		p := kpkPosition(t, weakToMove, weakKing, strongKing, pawn)

		if weakToMove == 0 && pawn/8 == 6 {
			continue
		}

		isWin := kpkResults[idx] == kpkWin
		moves := allMoves(&p)

		var anyWin, allWin bool

		allWin = true

		for _, m := range moves {
			if m.Flags&chess.Promotion != 0 {
				continue
			}

			undo := p.MakeMove(m)
			w := wins(&p)
			p.UnmakeMove(m, undo)

			anyWin = anyWin || w
			allWin = allWin && w
		}

		if weakToMove == 0 {
			if isWin != anyWin {
				t.Fatalf("%s: win=%v but a winning move exists=%v", p.FEN(), isWin, anyWin)
			}
		} else {
			if isWin && (len(moves) == 0 || !allWin) {
				t.Fatalf("%s: marked a win but black has an escape", p.FEN())
			}

			if !isWin && len(moves) > 0 && allWin {
				t.Fatalf("%s: marked a draw but every black move loses", p.FEN())
			}
		}

		checked++

		if isWin {
			winCount++
		}
	}

	if checked < 10000 || winCount < 1000 || winCount > checked-1000 {
		t.Fatalf("checked %d positions with %d wins, the table looks wrong", checked, winCount)
	}
}

func TestKPKImprovesPlay(t *testing.T) {
	p := prodPos(t, "8/8/8/8/8/4k3/4P3/4K3 w - - 0 1")

	prodSetFlags(t, prodDefaultFlags)

	if score := prodRootScore(t, p, 4, prodDefaultFlags); score != 0 {
		t.Fatalf("a drawn king and pawn ending scores %d at depth 4, want 0", score)
	}
}
