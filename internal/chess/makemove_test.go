package chess

import "testing"

func TestMakeUnmakeRestoresEveryMove(t *testing.T) {
	for _, tc := range suiteFENs {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)
			before := p

			for _, m := range legalMoves(&p) {
				undo := p.MakeMove(m)
				p.UnmakeMove(m, undo)

				if p != before {
					t.Fatalf("%s did not restore the position:\nbefore %s\nafter  %s", moveName(m), before.FEN(), p.FEN())
				}
			}
		})
	}
}

func TestMakeUnmakeRestoresDeepLines(t *testing.T) {
	for _, tc := range suiteFENs {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)
			before := p

			var walk func(depth int)

			walk = func(depth int) {
				if depth == 0 {
					return
				}

				for _, m := range legalMoves(&p) {
					snapshot := p
					undo := p.MakeMove(m)

					checkBoardConsistent(t, &p)
					walk(depth - 1)

					p.UnmakeMove(m, undo)

					if p != snapshot {
						t.Fatalf("%s did not restore %s", moveName(m), snapshot.FEN())
					}
				}
			}

			walk(3)

			if p != before {
				t.Fatalf("position changed after the walk: %s", p.FEN())
			}
		})
	}
}

func TestMakeMoveSwitchesSideAndCountsMoves(t *testing.T) {
	p := mustFEN(t, StartingFEN)

	p.MakeMove(findMove(t, &p, "e2e4"))

	if p.SideToMove != Black || p.FullmoveNumber != 1 {
		t.Fatalf("after white's move: side=%v fullmove=%d, want Black/1", p.SideToMove, p.FullmoveNumber)
	}

	p.MakeMove(findMove(t, &p, "e7e5"))

	if p.SideToMove != White || p.FullmoveNumber != 2 {
		t.Fatalf("after black's move: side=%v fullmove=%d, want White/2", p.SideToMove, p.FullmoveNumber)
	}
}

func TestHalfmoveClock(t *testing.T) {
	p := mustFEN(t, "4k3/8/8/8/8/8/P3N3/4K3 w - - 5 10")

	p.MakeMove(findMove(t, &p, "e2c3"))

	if p.HalfmoveClock != 6 {
		t.Fatalf("quiet piece move: clock = %d, want 6", p.HalfmoveClock)
	}

	p = mustFEN(t, "4k3/8/8/8/8/8/P3N3/4K3 w - - 5 10")
	p.MakeMove(findMove(t, &p, "a2a3"))

	if p.HalfmoveClock != 0 {
		t.Fatalf("pawn move must reset the clock, got %d", p.HalfmoveClock)
	}

	p = mustFEN(t, "4k3/8/8/3p4/8/2N5/8/4K3 w - - 5 10")
	p.MakeMove(findMove(t, &p, "c3d5"))

	if p.HalfmoveClock != 0 {
		t.Fatalf("capture must reset the clock, got %d", p.HalfmoveClock)
	}
}

func TestEnPassantSquareLifecycle(t *testing.T) {
	p := mustFEN(t, StartingFEN)

	p.MakeMove(findMove(t, &p, "e2e4"))

	if p.EnPassant != sqOf("e3") {
		t.Fatalf("after e2e4 en passant square = %d, want e3", p.EnPassant)
	}

	p.MakeMove(findMove(t, &p, "g8f6"))

	if p.EnPassant != -1 {
		t.Fatalf("a non-double-push must clear en passant, got %d", p.EnPassant)
	}

	p.MakeMove(findMove(t, &p, "d2d3"))

	if p.EnPassant != -1 {
		t.Fatalf("a single push must not set en passant, got %d", p.EnPassant)
	}

	p.MakeMove(findMove(t, &p, "d7d5"))

	if p.EnPassant != sqOf("d6") {
		t.Fatalf("after d7d5 en passant square = %d, want d6", p.EnPassant)
	}
}

func TestCastlingRightsLoss(t *testing.T) {
	const start = "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1"

	tests := []struct {
		name string
		fen  string
		move string
		want uint8
	}{
		{"king move loses both white rights", start, "e1d1", BlackKingSide | BlackQueenSide},
		{"kingside rook move loses K", start, "h1g1", WhiteQueenSide | BlackKingSide | BlackQueenSide},
		{"queenside rook move loses Q", start, "a1b1", WhiteKingSide | BlackKingSide | BlackQueenSide},
		{"castling loses both rights", start, "e1g1", BlackKingSide | BlackQueenSide},
		{"black king move loses both black rights", "r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1", "e8d8", WhiteKingSide | WhiteQueenSide},
		{"black rook move loses its right", "r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1", "h8g8", WhiteKingSide | WhiteQueenSide | BlackQueenSide},
		{"capturing a rook removes that right", "r3k2r/8/8/8/8/8/6b1/R3K2R b KQkq - 0 1", "g2h1", WhiteQueenSide | BlackKingSide | BlackQueenSide},
		{"other moves keep rights", "r3k2r/8/8/8/8/8/P7/R3K2R w KQkq - 0 1", "a2a3", WhiteKingSide | WhiteQueenSide | BlackKingSide | BlackQueenSide},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)
			p.MakeMove(findMove(t, &p, tc.move))

			if p.CastlingRights != tc.want {
				t.Fatalf("rights = %04b, want %04b", p.CastlingRights, tc.want)
			}
		})
	}
}

func TestPromotionPlacesChosenPiece(t *testing.T) {
	tests := []struct {
		move string
		want PieceType
	}{
		{"a7a8q", WhiteQueen},
		{"a7a8r", WhiteRook},
		{"a7a8b", WhiteBishop},
		{"a7a8n", WhiteKnight},
	}

	for _, tc := range tests {
		p := mustFEN(t, "7k/P7/8/8/8/8/8/K7 w - - 0 1")
		p.MakeMove(findMove(t, &p, tc.move))

		if got := p.Board.PieceAt(sqOf("a8")).Type; got != tc.want {
			t.Fatalf("%s: a8 holds %v, want %v", tc.move, got, tc.want)
		}

		if p.Board.PieceAt(sqOf("a7")).Type != Empty {
			t.Fatalf("%s: a7 should be empty", tc.move)
		}
	}

	p := mustFEN(t, "k7/8/8/8/8/8/p7/7K b - - 0 1")
	p.MakeMove(findMove(t, &p, "a2a1n"))

	if got := p.Board.PieceAt(sqOf("a1")).Type; got != BlackKnight {
		t.Fatalf("black underpromotion: a1 holds %v, want BlackKnight", got)
	}
}

func TestUnmakePromotionRestoresPawn(t *testing.T) {
	p := mustFEN(t, "1n5k/P7/8/8/8/8/8/K7 w - - 0 1")
	before := p

	for _, name := range []string{"a7a8q", "a7b8q", "a7b8n"} {
		m := findMove(t, &p, name)
		undo := p.MakeMove(m)
		p.UnmakeMove(m, undo)

		if p != before {
			t.Fatalf("%s: position not restored\n%s\n%s", name, before.FEN(), p.FEN())
		}
	}
}

func TestUnmakeRestoresCapturedPiece(t *testing.T) {
	p := mustFEN(t, "4k3/8/8/3q4/8/2N5/8/4K3 w - - 0 1")
	before := p

	m := findMove(t, &p, "c3d5")
	undo := p.MakeMove(m)

	if p.Board.PieceAt(sqOf("d5")).Type != WhiteKnight {
		t.Fatal("knight should stand on d5 after the capture")
	}

	if p.Board.BlackQueens != 0 {
		t.Fatal("captured queen must leave the board")
	}

	p.UnmakeMove(m, undo)

	if p != before {
		t.Fatalf("capture not undone: %s", p.FEN())
	}
}

func TestUnmakeEnPassantRestoresCapturedPawn(t *testing.T) {
	p := mustFEN(t, "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1")
	before := p

	m := findMove(t, &p, "e5d6")
	undo := p.MakeMove(m)
	p.UnmakeMove(m, undo)

	if p != before {
		t.Fatalf("en passant not undone:\n%s\n%s", before.FEN(), p.FEN())
	}

	p = mustFEN(t, "4k3/8/8/8/3pP3/8/8/4K3 b - e3 0 1")
	before = p

	m = findMove(t, &p, "d4e3")
	undo = p.MakeMove(m)
	p.UnmakeMove(m, undo)

	if p != before {
		t.Fatalf("black en passant not undone:\n%s\n%s", before.FEN(), p.FEN())
	}
}

func TestUnmakeCastlingRestoresRooks(t *testing.T) {
	for _, fen := range []string{
		"r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",
		"r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1",
	} {
		p := mustFEN(t, fen)
		before := p

		for _, m := range legalMoves(&p) {
			if m.Flags&(KingCastle|QueenCastle) == 0 {
				continue
			}

			undo := p.MakeMove(m)
			p.UnmakeMove(m, undo)

			if p != before {
				t.Fatalf("%s not undone:\n%s\n%s", moveName(m), before.FEN(), p.FEN())
			}
		}
	}
}

func TestApplyMoveOnEmptySquareDoesNothing(t *testing.T) {
	p := mustFEN(t, StartingFEN)
	before := p.Board

	p.Board.ApplyMove(Move{From: sqOf("e4"), To: sqOf("e5")})

	if p.Board != before {
		t.Fatal("moving from an empty square must not change the board")
	}
}

func TestClearAndSetPiece(t *testing.T) {
	var b PieceBoards

	b.SetPiece(WhiteQueen, sqOf("d4"))

	if b.PieceAt(sqOf("d4")).Type != WhiteQueen {
		t.Fatal("SetPiece did not place the queen")
	}

	b.ClearSquare(sqOf("d4"))

	if b.PieceAt(sqOf("d4")).Type != Empty || b.Occupied() != 0 {
		t.Fatal("ClearSquare did not empty the square")
	}

	b.SetPiece(WhitePawn, -1)
	b.SetPiece(WhitePawn, 64)

	if b.Occupied() != 0 {
		t.Fatal("SetPiece must ignore out-of-range squares")
	}
}

// Moves made for the side that is not on turn must still unmake cleanly:
// the castling rook and the en passant pawn belong to the mover, not to
// SideToMove.
func TestUnmakeUsesTheMovingPiecesColour(t *testing.T) {
	cases := []struct {
		name string
		fen  string
		move Move
	}{
		{
			name: "black king-side castle while white is to move",
			fen:  "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",
			move: Move{From: sqOf("e8"), To: sqOf("g8"), Piece: BlackKing, Flags: KingCastle},
		},
		{
			name: "black queen-side castle while white is to move",
			fen:  "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",
			move: Move{From: sqOf("e8"), To: sqOf("c8"), Piece: BlackKing, Flags: QueenCastle},
		},
		{
			name: "white king-side castle while black is to move",
			fen:  "r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1",
			move: Move{From: sqOf("e1"), To: sqOf("g1"), Piece: WhiteKing, Flags: KingCastle},
		},
		{
			name: "white queen-side castle while black is to move",
			fen:  "r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1",
			move: Move{From: sqOf("e1"), To: sqOf("c1"), Piece: WhiteKing, Flags: QueenCastle},
		},
		{
			name: "black en passant while white is to move",
			fen:  "4k3/8/8/8/3pP3/8/8/4K3 w - e3 0 1",
			move: Move{From: sqOf("d4"), To: sqOf("e3"), Piece: BlackPawn, Captured: WhitePawn, Flags: Capture | EnPassant},
		},
		{
			name: "white en passant while black is to move",
			fen:  "4k3/8/8/3pP3/8/8/8/4K3 b - d6 0 1",
			move: Move{From: sqOf("e5"), To: sqOf("d6"), Piece: WhitePawn, Captured: BlackPawn, Flags: Capture | EnPassant},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)
			before := p

			undo := p.MakeMove(tc.move)
			p.UnmakeMove(tc.move, undo)

			if p != before {
				t.Fatalf("not restored:\nbefore %s\nafter  %s", before.FEN(), p.FEN())
			}
		})
	}
}
