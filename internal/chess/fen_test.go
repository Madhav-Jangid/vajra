package chess

import "testing"

func TestParseFENFields(t *testing.T) {
	p := mustFEN(t, "r3k2r/8/8/3pP3/8/8/8/R3K2R b Kq d6 7 23")

	if p.SideToMove != Black {
		t.Fatalf("side to move = %v, want Black", p.SideToMove)
	}

	if p.CastlingRights != WhiteKingSide|BlackQueenSide {
		t.Fatalf("castling rights = %04b, want K and q", p.CastlingRights)
	}

	if p.EnPassant != sqOf("d6") {
		t.Fatalf("en passant = %d, want d6 (%d)", p.EnPassant, sqOf("d6"))
	}

	if p.HalfmoveClock != 7 || p.FullmoveNumber != 23 {
		t.Fatalf("clocks = %d/%d, want 7/23", p.HalfmoveClock, p.FullmoveNumber)
	}
}

func TestParseFENPlacesPiecesOnTheRightSquares(t *testing.T) {
	p := mustFEN(t, "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	want := map[string]PieceType{
		"a1": WhiteRook, "e1": WhiteKing, "h1": WhiteRook,
		"d2": WhiteBishop, "e2": WhiteBishop,
		"c3": WhiteKnight, "f3": WhiteQueen, "h3": BlackPawn,
		"e4": WhitePawn, "b4": BlackPawn,
		"d5": WhitePawn, "e5": WhiteKnight,
		"a6": BlackBishop, "b6": BlackKnight,
		"e7": BlackQueen, "g7": BlackBishop,
		"a8": BlackRook, "e8": BlackKing, "h8": BlackRook,
		"d4": Empty, "a4": Empty, "h5": Empty,
	}

	for sq, piece := range want {
		if got := p.Board.PieceAt(sqOf(sq)).Type; got != piece {
			t.Fatalf("%s holds %v, want %v", sq, got, piece)
		}
	}
}

func TestParseFENRejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		fen  string
	}{
		{"empty string", ""},
		{"only placement", "8/8/8/8/8/8/8/8"},
		{"five fields", "8/8/8/8/8/8/8/8 w - - 0"},
		{"seven fields", "8/8/8/8/8/8/8/8 w - - 0 1 extra"},
		{"bad side to move", "8/8/8/8/8/8/8/8 x - - 0 1"},
		{"uppercase side to move", "8/8/8/8/8/8/8/8 W - - 0 1"},
		{"seven ranks", "8/8/8/8/8/8/8 w - - 0 1"},
		{"nine ranks", "8/8/8/8/8/8/8/8/8 w - - 0 1"},
		{"rank too short", "7/8/8/8/8/8/8/8 w - - 0 1"},
		{"rank too long", "9/8/8/8/8/8/8/8 w - - 0 1"},
		{"rank overflows with pieces", "ppppppppp/8/8/8/8/8/8/8 w - - 0 1"},
		{"unknown piece letter", "x7/8/8/8/8/8/8/8 w - - 0 1"},
		{"bad en passant file", "8/8/8/8/8/8/8/8 w - z3 0 1"},
		{"bad en passant rank", "8/8/8/8/8/8/8/8 w - e9 0 1"},
		{"en passant too long", "8/8/8/8/8/8/8/8 w - e33 0 1"},
		{"halfmove not a number", "8/8/8/8/8/8/8/8 w - - x 1"},
		{"fullmove not a number", "8/8/8/8/8/8/8/8 w - - 0 y"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseFEN(tc.fen); err == nil {
				t.Fatalf("ParseFEN(%q) should fail", tc.fen)
			}
		})
	}
}

func TestFENRoundTripCoversEveryField(t *testing.T) {
	fens := []string{
		"8/8/8/8/8/8/8/k6K w - - 0 1",
		"8/8/8/8/8/8/8/k6K b - - 49 80",
		"r3k2r/8/8/3pP3/8/8/8/R3K2R w KQkq d6 0 12",
		"r3k2r/8/8/8/3Pp3/8/8/R3K2R b KQkq d3 0 12",
		"r3k2r/8/8/8/8/8/8/R3K2R w Kq - 3 9",
		"r3k2r/8/8/8/8/8/8/R3K2R w Qk - 3 9",
		"4k3/8/8/8/8/8/8/4K3 w - - 99 120",
	}

	for _, fen := range fens {
		p := mustFEN(t, fen)

		if got := p.FEN(); got != fen {
			t.Fatalf("round trip mismatch\nwant %s\ngot  %s", fen, got)
		}
	}
}

func TestFENRoundTripAfterMoves(t *testing.T) {
	p := mustFEN(t, StartingFEN)

	for _, name := range []string{"e2e4", "c7c5", "g1f3", "d7d6", "f1b5"} {
		p.MakeMove(findMove(t, &p, name))

		again, err := ParseFEN(p.FEN())
		if err != nil {
			t.Fatalf("FEN %q does not reparse: %v", p.FEN(), err)
		}

		if again != p {
			t.Fatalf("reparsed position differs after %s\n%s", name, p.FEN())
		}
	}
}

func TestValidateFen(t *testing.T) {
	for _, tc := range suiteFENs {
		p := mustFEN(t, tc.fen)

		ok, err := p.ValidateFen()
		if err != nil || !ok {
			t.Fatalf("%s: ValidateFen() = %v, %v", tc.name, ok, err)
		}
	}
}

func TestSetUpInitialPosition(t *testing.T) {
	saved := CurrentPosition
	defer func() { CurrentPosition = saved }()

	if err := SetUpInitialPosition(); err != nil {
		t.Fatalf("SetUpInitialPosition: %v", err)
	}

	if got := CurrentPosition.FEN(); got != StartingFEN {
		t.Fatalf("current position = %s, want the start position", got)
	}
}

func TestPieceAtEveryStartingSquare(t *testing.T) {
	p := mustFEN(t, StartingFEN)
	back := []PieceType{WhiteRook, WhiteKnight, WhiteBishop, WhiteQueen, WhiteKing, WhiteBishop, WhiteKnight, WhiteRook}
	blackBack := []PieceType{BlackRook, BlackKnight, BlackBishop, BlackQueen, BlackKing, BlackBishop, BlackKnight, BlackRook}

	for file := 0; file < 8; file++ {
		checks := []struct {
			sq   int
			want PieceType
		}{
			{file, back[file]},
			{8 + file, WhitePawn},
			{48 + file, BlackPawn},
			{56 + file, blackBack[file]},
		}

		for _, c := range checks {
			if got := p.Board.PieceAt(c.sq).Type; got != c.want {
				t.Fatalf("square %d holds %v, want %v", c.sq, got, c.want)
			}
		}

		for rank := 2; rank <= 5; rank++ {
			if p.Board.PieceAt(rank*8+file).Type != Empty {
				t.Fatalf("square %d should be empty", rank*8+file)
			}
		}
	}

	if p.Board.PieceAt(sqOf("e1")).Color != White || p.Board.PieceAt(sqOf("e8")).Color != Black {
		t.Fatal("piece colours are wrong")
	}
}

func TestSquareAttacked(t *testing.T) {
	tests := []struct {
		name   string
		fen    string
		square string
		by     Color
		want   bool
	}{
		{"pawn attacks diagonally", "4k3/8/8/8/3P4/8/8/4K3 w - - 0 1", "e5", White, true},
		{"pawn does not attack straight ahead", "4k3/8/8/8/3P4/8/8/4K3 w - - 0 1", "d5", White, false},
		{"black pawn attacks downward", "4k3/8/8/3p4/8/8/8/4K3 w - - 0 1", "c4", Black, true},
		{"knight attack", "4k3/8/8/8/3N4/8/8/4K3 w - - 0 1", "e6", White, true},
		{"knight does not attack adjacent", "4k3/8/8/8/3N4/8/8/4K3 w - - 0 1", "d5", White, false},
		{"rook along file", "4k3/8/8/8/8/8/8/R3K3 w - - 0 1", "a8", White, true},
		{"rook blocked by a piece", "4k3/8/8/8/8/P7/8/R3K3 w - - 0 1", "a8", White, false},
		{"bishop on diagonal", "4k3/8/8/8/8/8/8/2B1K3 w - - 0 1", "h6", White, true},
		{"queen on both lines", "4k3/8/8/8/3Q4/8/8/4K3 w - - 0 1", "h8", White, true},
		{"king attacks neighbours", "4k3/8/8/8/8/8/8/4K3 w - - 0 1", "d2", White, true},
		{"out of range low", "4k3/8/8/8/8/8/8/4K3 w - - 0 1", "e1", Black, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)

			if got := p.Board.IsSquareAttacked(sqOf(tc.square), tc.by); got != tc.want {
				t.Fatalf("IsSquareAttacked(%s, %v) = %v, want %v", tc.square, tc.by, got, tc.want)
			}
		})
	}

	p := mustFEN(t, StartingFEN)

	if p.Board.IsSquareAttacked(-1, White) || p.Board.IsSquareAttacked(64, White) {
		t.Fatal("out-of-range squares are never attacked")
	}

	if p.Board.IsSquareOccupied(-1) || p.Board.IsSquareOccupied(64) {
		t.Fatal("out-of-range squares are never occupied")
	}
}
