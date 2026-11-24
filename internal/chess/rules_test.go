package chess

import "testing"

func TestStatus(t *testing.T) {
	tests := []struct {
		name string
		fen  string
		side Color
		want string
	}{
		{"start position", StartingFEN, White, ""},
		{"fool's mate", "rnb1kbnr/pppp1ppp/8/4p3/6Pq/5P2/PPPPP2P/RNBQKBNR w KQkq - 1 3", White, "checkmate"},
		{"scholar's mate", "r1bqkb1r/pppp1Qpp/2n2n2/4p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 4", Black, "checkmate"},
		{"back rank mate", "6k1/5ppp/8/8/8/8/8/R3K3 b - - 0 1", Black, ""},
		{"back rank mate delivered", "R5k1/5ppp/8/8/8/8/8/4K3 b - - 0 1", Black, "checkmate"},
		{"stalemate", "7k/5Q2/6K1/8/8/8/8/8 b - - 0 1", Black, "stalemate"},
		{"stalemate with pawns", "k7/P7/K7/8/8/8/8/8 b - - 0 1", Black, "stalemate"},
		{"in check is not mate when a reply exists", "4k3/8/8/8/8/8/4r3/4K3 w - - 0 1", White, ""},
		{"king versus king", "4k3/8/8/8/8/8/8/4K3 w - - 0 1", White, "insufficient material"},
		{"king and bishop versus king", "4k3/8/8/8/8/8/8/3BK3 w - - 0 1", White, "insufficient material"},
		{"king and knight versus king", "4k3/8/8/8/8/8/8/3NK3 w - - 0 1", White, "insufficient material"},
		{"bishop each is not auto-drawn here", "4kb2/8/8/8/8/8/8/3BK3 w - - 0 1", White, ""},
		{"two knights are not insufficient", "4k3/8/8/8/8/8/8/2NNK3 w - - 0 1", White, ""},
		{"a pawn is sufficient material", "4k3/8/8/8/8/8/P7/4K3 w - - 0 1", White, ""},
		{"a rook is sufficient material", "4k3/8/8/8/8/8/8/3RK3 w - - 0 1", White, ""},
		{"fifty-move rule", "4k3/8/8/8/8/8/P7/4K3 w - - 100 80", White, "fifty-move rule"},
		{"just under fifty-move rule", "4k3/8/8/8/8/8/P7/4K3 w - - 99 80", White, ""},
		{"checkmate beats fifty-move rule", "R5k1/5ppp/8/8/8/8/8/4K3 b - - 120 80", Black, "checkmate"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)

			if got := p.Status(tc.side); got != tc.want {
				t.Fatalf("Status = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHasLegalMove(t *testing.T) {
	p := mustFEN(t, StartingFEN)

	if !p.HasLegalMove(White) || !p.HasLegalMove(Black) {
		t.Fatal("both sides have moves at the start")
	}

	mated := mustFEN(t, "R5k1/5ppp/8/8/8/8/8/4K3 b - - 0 1")

	if mated.HasLegalMove(Black) {
		t.Fatal("black is checkmated and has no legal move")
	}

	if !mated.HasLegalMove(White) {
		t.Fatal("white still has moves")
	}
}

func TestInCheck(t *testing.T) {
	tests := []struct {
		name  string
		fen   string
		color Color
		want  bool
	}{
		{"start", StartingFEN, White, false},
		{"rook check", "4r2k/8/8/8/8/8/8/4K3 w - - 0 1", White, true},
		{"bishop check", "7k/8/8/8/8/8/3b4/4K3 w - - 0 1", White, true},
		{"knight check", "7k/8/8/8/8/5n2/8/4K3 w - - 0 1", White, true},
		{"pawn check", "7k/8/8/8/8/8/3p4/4K3 w - - 0 1", White, true},
		{"blocked rook is no check", "4r2k/8/8/8/8/8/4P3/4K3 w - - 0 1", White, false},
		{"black in check", "4k3/8/8/8/8/8/8/4R2K b - - 0 1", Black, true},
		{"missing king is never in check", "8/8/8/8/8/8/8/R7 w - - 0 1", White, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)

			if got := p.InCheck(tc.color); got != tc.want {
				t.Fatalf("InCheck(%v) = %v, want %v", tc.color, got, tc.want)
			}
		})
	}
}

func TestPositionKey(t *testing.T) {
	a := mustFEN(t, StartingFEN)
	b := mustFEN(t, StartingFEN)

	if a.Key() != b.Key() {
		t.Fatal("identical positions must have identical keys")
	}

	b.SideToMove = Black

	if a.Key() == b.Key() {
		t.Fatal("side to move must change the key")
	}

	c := mustFEN(t, StartingFEN)
	c.CastlingRights = WhiteKingSide

	if a.Key() == c.Key() {
		t.Fatal("castling rights must change the key")
	}

	d := mustFEN(t, StartingFEN)
	d.EnPassant = sqOf("e3")

	if a.Key() == d.Key() {
		t.Fatal("en passant square must change the key")
	}

	e := mustFEN(t, StartingFEN)
	e.HalfmoveClock, e.FullmoveNumber = 17, 40

	if a.Key() != e.Key() {
		t.Fatal("move counters must not change the key (repetition detection)")
	}

	f := mustFEN(t, StartingFEN)
	f.MakeMove(findMove(t, &f, "g1f3"))

	if a.Key() == f.Key() {
		t.Fatal("a different board must change the key")
	}
}

func TestPositionKeyDetectsRepetition(t *testing.T) {
	p := mustFEN(t, StartingFEN)
	start := p.Key()

	for _, name := range []string{"g1f3", "g8f6", "f3g1", "f6g8"} {
		p.MakeMove(findMove(t, &p, name))
	}

	if p.Key() != start {
		t.Fatal("knights out and back should repeat the starting position")
	}
}

func TestPlayMove(t *testing.T) {
	t.Run("legal move is played", func(t *testing.T) {
		p := mustFEN(t, StartingFEN)

		if !p.PlayMove(sqOf("e2"), sqOf("e4"), Empty) {
			t.Fatal("e2e4 should be playable")
		}

		if p.Board.PieceAt(sqOf("e4")).Type != WhitePawn || p.SideToMove != Black {
			t.Fatalf("e2e4 not applied: %s", p.FEN())
		}
	})

	t.Run("illegal move is refused and changes nothing", func(t *testing.T) {
		p := mustFEN(t, StartingFEN)
		before := p

		if p.PlayMove(sqOf("e2"), sqOf("e5"), Empty) {
			t.Fatal("e2e5 is illegal")
		}

		if p != before {
			t.Fatal("a refused move must not change the position")
		}
	})

	t.Run("cannot move the other side's piece", func(t *testing.T) {
		p := mustFEN(t, StartingFEN)

		if p.PlayMove(sqOf("e7"), sqOf("e5"), Empty) {
			t.Fatal("black cannot move on white's turn")
		}
	})

	t.Run("empty square and out of range squares", func(t *testing.T) {
		p := mustFEN(t, StartingFEN)

		for _, c := range [][2]int{{sqOf("e4"), sqOf("e5")}, {-1, 20}, {20, -1}, {64, 20}, {20, 64}} {
			if p.PlayMove(c[0], c[1], Empty) {
				t.Fatalf("PlayMove(%d, %d) should fail", c[0], c[1])
			}
		}
	})

	t.Run("promotion defaults to a queen", func(t *testing.T) {
		p := mustFEN(t, "7k/P7/8/8/8/8/8/K7 w - - 0 1")

		if !p.PlayMove(sqOf("a7"), sqOf("a8"), Empty) {
			t.Fatal("promotion should be playable")
		}

		if got := p.Board.PieceAt(sqOf("a8")).Type; got != WhiteQueen {
			t.Fatalf("default promotion = %v, want queen", got)
		}
	})

	t.Run("explicit promotion piece", func(t *testing.T) {
		p := mustFEN(t, "7k/P7/8/8/8/8/8/K7 w - - 0 1")

		if !p.PlayMove(sqOf("a7"), sqOf("a8"), WhiteKnight) {
			t.Fatal("knight promotion should be playable")
		}

		if got := p.Board.PieceAt(sqOf("a8")).Type; got != WhiteKnight {
			t.Fatalf("promotion = %v, want knight", got)
		}
	})

	t.Run("black default promotion is a black queen", func(t *testing.T) {
		p := mustFEN(t, "k7/8/8/8/8/8/p7/7K b - - 0 1")

		if !p.PlayMove(sqOf("a2"), sqOf("a1"), Empty) {
			t.Fatal("promotion should be playable")
		}

		if got := p.Board.PieceAt(sqOf("a1")).Type; got != BlackQueen {
			t.Fatalf("default promotion = %v, want black queen", got)
		}
	})

	t.Run("castling through PlayMove", func(t *testing.T) {
		p := mustFEN(t, "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1")

		if !p.PlayMove(sqOf("e1"), sqOf("g1"), Empty) {
			t.Fatal("O-O should be playable")
		}

		if p.Board.PieceAt(sqOf("f1")).Type != WhiteRook {
			t.Fatalf("rook should be on f1: %s", p.FEN())
		}
	})
}

func TestFoolsMateSequence(t *testing.T) {
	p := mustFEN(t, StartingFEN)

	for _, c := range [][2]string{{"f2", "f3"}, {"e7", "e5"}, {"g2", "g4"}, {"d8", "h4"}} {
		if !p.PlayMove(sqOf(c[0]), sqOf(c[1]), Empty) {
			t.Fatalf("%s%s should be legal", c[0], c[1])
		}
	}

	if got := p.Status(p.SideToMove); got != "checkmate" {
		t.Fatalf("after fool's mate status = %q, want checkmate", got)
	}
}

func TestFullGameKeepsBoardConsistent(t *testing.T) {
	game := []string{
		"e2e4", "e7e5", "g1f3", "d7d6", "d2d4", "c8g4", "d4e5", "g4f3",
		"d1f3", "d6e5", "f1c4", "g8f6", "f3b3", "d8e7", "b1c3", "c7c6",
		"c1g5", "b7b5", "c3b5", "c6b5", "c4b5", "b8d7", "e1c1", "a8d8",
		"d1d7", "d8d7", "h1d1", "e7e6", "b5d7", "f6d7", "b3b8", "d7b8", "d1d8",
	}

	p := mustFEN(t, StartingFEN)

	for _, name := range game {
		m := findMove(t, &p, name)
		p.MakeMove(m)

		checkBoardConsistent(t, &p)
	}

	if got := p.Status(p.SideToMove); got != "checkmate" {
		t.Fatalf("the Opera Game ends in mate, status = %q", got)
	}
}
