package chess

import "testing"

func TestPieceMoveCounts(t *testing.T) {
	tests := []struct {
		name   string
		fen    string
		square string
		want   int
	}{
		{"king in centre", "7k/8/8/8/3K4/8/8/8 w - - 0 1", "d4", 8},
		{"king in corner", "K6k/8/8/8/8/8/8/8 w - - 0 1", "a8", 3},
		{"king on edge", "7k/8/8/8/8/8/8/3K4 w - - 0 1", "d1", 5},
		{"knight in centre", "7k/8/8/8/3N4/8/8/K7 w - - 0 1", "d4", 8},
		{"knight in corner", "7k/8/8/8/8/8/8/N6K w - - 0 1", "a1", 2},
		{"knight next to edge", "7k/8/8/8/8/8/8/1N5K w - - 0 1", "b1", 3},
		{"rook open board", "7k/8/8/8/3R4/8/8/K7 w - - 0 1", "d4", 14},
		{"rook blocked by own pawn", "7k/8/8/3P4/3R4/8/8/K7 w - - 0 1", "d4", 10},
		{"rook captures enemy pawn and stops", "7k/8/8/3p4/3R4/8/8/K7 w - - 0 1", "d4", 11},
		{"bishop open board", "k7/8/8/8/3B4/8/8/7K w - - 0 1", "d4", 13},
		{"bishop in corner", "7k/8/8/8/8/8/8/B6K w - - 0 1", "a1", 7},
		{"queen open board", "k7/8/8/8/3Q4/8/8/7K w - - 0 1", "d4", 27},
		{"queen boxed in by own pieces", "7k/8/8/8/8/PPP5/PQP5/PKP5 w - - 0 1", "b2", 0},
		{"black knight", "7K/8/8/8/3n4/8/8/k7 b - - 0 1", "d4", 8},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)

			if got := len(legalFrom(&p, tc.square)); got != tc.want {
				t.Fatalf("%s has %d legal moves, want %d: %v", tc.square, got, tc.want, legalFrom(&p, tc.square))
			}
		})
	}
}

func TestSlidersStopAtBlockers(t *testing.T) {
	p := mustFEN(t, "7k/8/8/3p4/8/3R4/8/K7 w - - 0 1")
	moves := legalFrom(&p, "d3")

	if !contains(moves, "d3d5") {
		t.Fatal("rook should be able to capture the pawn on d5")
	}

	if contains(moves, "d3d6") {
		t.Fatal("rook must not jump past the pawn it captures")
	}

	p = mustFEN(t, "7k/8/8/3P4/8/3R4/8/K7 w - - 0 1")
	moves = legalFrom(&p, "d3")

	if contains(moves, "d3d5") {
		t.Fatal("rook must not capture its own pawn")
	}

	if !contains(moves, "d3d4") {
		t.Fatal("rook should reach d4")
	}
}

func TestMoveFlagsAndCapturedPiece(t *testing.T) {
	p := mustFEN(t, "4k3/8/8/3p4/8/2N5/8/4K3 w - - 0 1")

	var capture, quiet Move

	for _, m := range p.LegalMoves(sqOf("c3")) {
		switch moveName(m) {
		case "c3d5":
			capture = m
		case "c3e4":
			quiet = m
		}
	}

	if capture.Flags&Capture == 0 || capture.Captured != BlackPawn || capture.Piece != WhiteKnight {
		t.Fatalf("capture move not described correctly: %+v", capture)
	}

	if quiet.Flags != Quiet || quiet.Captured != Empty {
		t.Fatalf("quiet move not described correctly: %+v", quiet)
	}
}

func TestPawnMoves(t *testing.T) {
	tests := []struct {
		name   string
		fen    string
		square string
		want   []string
	}{
		{"white pawn start rank", "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1", "e2", []string{"e2e3", "e2e4"}},
		{"white pawn off start rank", "4k3/8/8/8/8/4P3/8/4K3 w - - 0 1", "e3", []string{"e3e4"}},
		{"white pawn single push blocked", "4k3/8/8/8/8/4p3/4P3/4K3 w - - 0 1", "e2", nil},
		{"white pawn double push blocked at e4", "4k3/8/8/8/4p3/8/4P3/4K3 w - - 0 1", "e2", []string{"e2e3"}},
		{"black pawn start rank", "4k3/4p3/8/8/8/8/8/4K3 b - - 0 1", "e7", []string{"e7e5", "e7e6"}},
		{"black pawn moves down", "4k3/8/4p3/8/8/8/8/4K3 b - - 0 1", "e6", []string{"e6e5"}},
		{"white pawn captures both sides", "4k3/8/8/3p1p2/4P3/8/8/4K3 w - - 0 1", "e4", []string{"e4d5", "e4e5", "e4f5"}},
		{"a-file pawn does not wrap to h-file", "4k3/8/8/7p/P7/8/8/4K3 w - - 0 1", "a4", []string{"a4a5"}},
		{"h-file pawn does not wrap to a-file", "4k3/8/8/p7/7P/8/8/4K3 w - - 0 1", "h4", []string{"h4h5"}},
		{"pawn cannot capture forward", "4k3/8/8/4p3/4P3/8/8/4K3 w - - 0 1", "e4", nil},
		{"pawn cannot capture own piece", "4k3/8/8/3N4/4P3/8/8/4K3 w - - 0 1", "e4", []string{"e4e5"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)

			if got := legalFrom(&p, tc.square); !sameList(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDoublePawnPushSetsFlag(t *testing.T) {
	p := mustFEN(t, "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1")

	if findMove(t, &p, "e2e4").Flags&DoublePawnPush == 0 {
		t.Fatal("e2e4 must carry the DoublePawnPush flag")
	}

	if findMove(t, &p, "e2e3").Flags&DoublePawnPush != 0 {
		t.Fatal("e2e3 must not carry the DoublePawnPush flag")
	}
}

func TestPromotionMoves(t *testing.T) {
	t.Run("push gives four promotions", func(t *testing.T) {
		p := mustFEN(t, "7k/P7/8/8/8/8/8/K7 w - - 0 1")
		want := []string{"a7a8q", "a7a8r", "a7a8b", "a7a8n"}

		if got := legalFrom(&p, "a7"); !sameList(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}

		for _, m := range p.LegalMoves(sqOf("a7")) {
			if m.Flags&Promotion == 0 {
				t.Fatalf("%s lacks the Promotion flag", moveName(m))
			}
		}
	})

	t.Run("capture promotion gives four per target", func(t *testing.T) {
		p := mustFEN(t, "1n5k/P7/8/8/8/8/8/K7 w - - 0 1")
		moves := legalFrom(&p, "a7")

		if len(moves) != 8 {
			t.Fatalf("want 8 moves (4 push + 4 capture), got %d: %v", len(moves), moves)
		}

		for _, want := range []string{"a7b8q", "a7b8r", "a7b8b", "a7b8n"} {
			if !contains(moves, want) {
				t.Fatalf("missing %s in %v", want, moves)
			}
		}
	})

	t.Run("black promotes downward with black pieces", func(t *testing.T) {
		p := mustFEN(t, "k7/8/8/8/8/8/p7/7K b - - 0 1")

		for _, m := range p.LegalMoves(sqOf("a2")) {
			switch m.Promotion {
			case BlackQueen, BlackRook, BlackBishop, BlackKnight:
			default:
				t.Fatalf("black pawn promoted to %v", m.Promotion)
			}
		}

		if got := len(p.LegalMoves(sqOf("a2"))); got != 4 {
			t.Fatalf("want 4 promotions, got %d", got)
		}
	})

	t.Run("blocked pawn on seventh cannot promote", func(t *testing.T) {
		p := mustFEN(t, "n6k/P7/8/8/8/8/8/K7 w - - 0 1")

		if got := legalFrom(&p, "a7"); len(got) != 0 {
			t.Fatalf("blocked pawn must not move, got %v", got)
		}
	})
}

func TestEnPassant(t *testing.T) {
	t.Run("white captures en passant", func(t *testing.T) {
		p := mustFEN(t, "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1")
		m := findMove(t, &p, "e5d6")

		if m.Flags&EnPassant == 0 || m.Flags&Capture == 0 || m.Captured != BlackPawn {
			t.Fatalf("e5d6 not flagged as en passant capture: %+v", m)
		}
	})

	t.Run("black captures en passant", func(t *testing.T) {
		p := mustFEN(t, "4k3/8/8/8/3pP3/8/8/4K3 b - e3 0 1")
		m := findMove(t, &p, "d4e3")

		if m.Flags&EnPassant == 0 || m.Captured != WhitePawn {
			t.Fatalf("d4e3 not flagged as en passant capture: %+v", m)
		}
	})

	t.Run("no en passant without the target square", func(t *testing.T) {
		p := mustFEN(t, "4k3/8/8/3pP3/8/8/8/4K3 w - - 0 1")

		if contains(legalFrom(&p, "e5"), "e5d6") {
			t.Fatal("e5d6 must not be legal without an en passant square")
		}
	})

	t.Run("en passant removes the captured pawn", func(t *testing.T) {
		p := mustFEN(t, "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1")
		p.MakeMove(findMove(t, &p, "e5d6"))

		if p.Board.PieceAt(sqOf("d5")).Type != Empty {
			t.Fatal("captured pawn on d5 should be gone")
		}

		if p.Board.PieceAt(sqOf("d6")).Type != WhitePawn {
			t.Fatal("capturing pawn should stand on d6")
		}
	})

	t.Run("en passant that exposes the king along the rank is illegal", func(t *testing.T) {
		p := mustFEN(t, "7k/8/8/KPp4r/8/8/8/8 w - c6 0 1")

		if contains(legalFrom(&p, "b5"), "b5c6") {
			t.Fatal("b5c6 en passant would leave the white king in check")
		}
	})

	t.Run("en passant that resolves a pawn check is legal", func(t *testing.T) {
		p := mustFEN(t, "8/8/8/2k5/3Pp3/8/8/4K3 b - d3 0 1")

		if !contains(legalFrom(&p, "e4"), "e4d3") {
			t.Fatal("e4d3 should capture the checking pawn en passant")
		}
	})
}

func TestCastlingMoves(t *testing.T) {
	both := "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1"

	t.Run("white can castle both ways", func(t *testing.T) {
		p := mustFEN(t, both)
		moves := legalFrom(&p, "e1")

		if !contains(moves, "e1g1") || !contains(moves, "e1c1") {
			t.Fatalf("expected both castles, got %v", moves)
		}
	})

	t.Run("black can castle both ways", func(t *testing.T) {
		p := mustFEN(t, "r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1")
		moves := legalFrom(&p, "e8")

		if !contains(moves, "e8g8") || !contains(moves, "e8c8") {
			t.Fatalf("expected both castles, got %v", moves)
		}
	})

	t.Run("castle moves carry castle flags", func(t *testing.T) {
		p := mustFEN(t, both)

		if findMove(t, &p, "e1g1").Flags&KingCastle == 0 {
			t.Fatal("e1g1 should be KingCastle")
		}

		if findMove(t, &p, "e1c1").Flags&QueenCastle == 0 {
			t.Fatal("e1c1 should be QueenCastle")
		}
	})

	t.Run("no castling without rights", func(t *testing.T) {
		p := mustFEN(t, "r3k2r/8/8/8/8/8/8/R3K2R w - - 0 1")
		moves := legalFrom(&p, "e1")

		if contains(moves, "e1g1") || contains(moves, "e1c1") {
			t.Fatalf("castling without rights, got %v", moves)
		}
	})

	t.Run("only the right that remains", func(t *testing.T) {
		p := mustFEN(t, "r3k2r/8/8/8/8/8/8/R3K2R w Kkq - 0 1")
		moves := legalFrom(&p, "e1")

		if !contains(moves, "e1g1") || contains(moves, "e1c1") {
			t.Fatalf("want kingside only, got %v", moves)
		}
	})

	t.Run("blocked by a piece between king and rook", func(t *testing.T) {
		p := mustFEN(t, "r3k2r/8/8/8/8/8/8/R3KB1R w KQkq - 0 1")

		if contains(legalFrom(&p, "e1"), "e1g1") {
			t.Fatal("kingside castle through the bishop on f1")
		}

		p = mustFEN(t, "r3k2r/8/8/8/8/8/8/RN2K2R w KQkq - 0 1")

		if contains(legalFrom(&p, "e1"), "e1c1") {
			t.Fatal("queenside castle through the knight on b1")
		}
	})

	t.Run("not out of check", func(t *testing.T) {
		p := mustFEN(t, "4r2k/8/8/8/8/8/8/R3K2R w KQ - 0 1")
		moves := legalFrom(&p, "e1")

		if contains(moves, "e1g1") || contains(moves, "e1c1") {
			t.Fatalf("castling out of check, got %v", moves)
		}
	})

	t.Run("not through an attacked square", func(t *testing.T) {
		p := mustFEN(t, "5r1k/8/8/8/8/8/8/R3K2R w KQ - 0 1")

		if contains(legalFrom(&p, "e1"), "e1g1") {
			t.Fatal("kingside castle through the attacked f1 square")
		}

		if !contains(legalFrom(&p, "e1"), "e1c1") {
			t.Fatal("queenside castle should still be legal")
		}
	})

	t.Run("not into an attacked square", func(t *testing.T) {
		p := mustFEN(t, "6rk/8/8/8/8/8/8/R3K2R w KQ - 0 1")

		if contains(legalFrom(&p, "e1"), "e1g1") {
			t.Fatal("kingside castle into the attacked g1 square")
		}
	})

	t.Run("queenside castle allowed when only b1 is attacked", func(t *testing.T) {
		p := mustFEN(t, "1r5k/8/8/8/8/8/8/R3K2R w KQ - 0 1")

		if !contains(legalFrom(&p, "e1"), "e1c1") {
			t.Fatal("a rook attacking b1 must not stop queenside castling")
		}
	})

	t.Run("rook missing from its corner", func(t *testing.T) {
		p := mustFEN(t, "r3k2r/8/8/8/8/8/8/4K2R w KQkq - 0 1")

		if contains(legalFrom(&p, "e1"), "e1c1") {
			t.Fatal("queenside castle without a rook")
		}
	})

	t.Run("castling moves the rook", func(t *testing.T) {
		p := mustFEN(t, both)
		p.MakeMove(findMove(t, &p, "e1g1"))

		if p.Board.PieceAt(sqOf("g1")).Type != WhiteKing || p.Board.PieceAt(sqOf("f1")).Type != WhiteRook {
			t.Fatalf("king/rook not on g1/f1 after O-O: %s", p.FEN())
		}

		if p.Board.PieceAt(sqOf("h1")).Type != Empty || p.Board.PieceAt(sqOf("e1")).Type != Empty {
			t.Fatalf("h1/e1 should be empty after O-O: %s", p.FEN())
		}

		p = mustFEN(t, both)
		p.MakeMove(findMove(t, &p, "e1c1"))

		if p.Board.PieceAt(sqOf("c1")).Type != WhiteKing || p.Board.PieceAt(sqOf("d1")).Type != WhiteRook {
			t.Fatalf("king/rook not on c1/d1 after O-O-O: %s", p.FEN())
		}

		if p.Board.PieceAt(sqOf("a1")).Type != Empty {
			t.Fatalf("a1 should be empty after O-O-O: %s", p.FEN())
		}
	})
}

func TestPinnedPieces(t *testing.T) {
	t.Run("pinned rook moves only along the pin", func(t *testing.T) {
		p := mustFEN(t, "4k3/4r3/8/8/8/8/4R3/4K3 w - - 0 1")
		want := []string{"e2e3", "e2e4", "e2e5", "e2e6", "e2e7"}

		if got := legalFrom(&p, "e2"); !sameList(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("pinned knight cannot move", func(t *testing.T) {
		p := mustFEN(t, "4k3/4r3/8/8/8/8/4N3/4K3 w - - 0 1")

		if got := legalFrom(&p, "e2"); len(got) != 0 {
			t.Fatalf("pinned knight moved: %v", got)
		}
	})

	t.Run("pinned bishop can capture the pinner", func(t *testing.T) {
		p := mustFEN(t, "7k/8/8/8/8/2b5/3B4/4K3 w - - 0 1")
		moves := legalFrom(&p, "d2")

		if !contains(moves, "d2c3") {
			t.Fatalf("bishop should be free to take along the pin, got %v", moves)
		}
	})

	t.Run("pinned pawn cannot capture off the pin line", func(t *testing.T) {
		p := mustFEN(t, "4k3/4r3/8/8/3p4/4P3/8/4K3 w - - 0 1")

		if contains(legalFrom(&p, "e3"), "e3d4") {
			t.Fatal("pinned pawn captured off the pin line")
		}
	})
}

func TestCheckEvasion(t *testing.T) {
	t.Run("only legal replies to a rook check", func(t *testing.T) {
		p := mustFEN(t, "4r2k/8/8/8/8/8/3B4/4K3 w - - 0 1")
		got := legalNames(&p)
		want := []string{"d2e3", "e1d1", "e1f1", "e1f2"}

		if !sameList(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("double check allows only king moves", func(t *testing.T) {
		p := mustFEN(t, "4r2k/8/8/8/8/5n2/8/R3K2R w - - 0 1")

		for _, name := range legalNames(&p) {
			if name[:2] != "e1" {
				t.Fatalf("non-king move %s in double check", name)
			}
		}
	})

	t.Run("king cannot step next to the enemy king", func(t *testing.T) {
		p := mustFEN(t, "8/8/8/8/8/5k2/8/4K3 w - - 0 1")
		moves := legalFrom(&p, "e1")

		for _, bad := range []string{"e1e2", "e1f2"} {
			if contains(moves, bad) {
				t.Fatalf("king moved adjacent to the enemy king with %s", bad)
			}
		}
	})

	t.Run("king cannot capture a protected piece", func(t *testing.T) {
		p := mustFEN(t, "4k3/8/8/7b/8/8/4r3/4K3 w - - 0 1")

		if contains(legalFrom(&p, "e1"), "e1e2") {
			t.Fatal("king captured a rook protected by the bishop")
		}
	})

	t.Run("discovered check is detected", func(t *testing.T) {
		p := mustFEN(t, "4k3/8/8/8/4N3/8/8/4RK2 w - - 0 1")
		p.MakeMove(findMove(t, &p, "e4c5"))

		if !p.InCheck(Black) {
			t.Fatal("moving the knight should discover a check from the rook")
		}
	})
}

func TestGenerateMovesInvalidInput(t *testing.T) {
	p := mustFEN(t, StartingFEN)

	for _, sq := range []int{-1, 64, 1000, -50} {
		if got := p.GenerateMoves(sq); got != nil {
			t.Fatalf("GenerateMoves(%d) = %v, want nil", sq, got)
		}
	}

	if got := p.GenerateMoves(sqOf("e4")); got != nil {
		t.Fatalf("empty square should generate nothing, got %v", got)
	}

	if got := p.LegalMoves(sqOf("e4")); got != nil {
		t.Fatalf("empty square should have no legal moves, got %v", got)
	}
}

func TestStartingPositionMoveCount(t *testing.T) {
	p := mustFEN(t, StartingFEN)

	if got := len(legalMoves(&p)); got != 20 {
		t.Fatalf("start position has %d legal moves, want 20", got)
	}

	p.SideToMove = Black

	if got := len(legalMoves(&p)); got != 20 {
		t.Fatalf("black has %d legal moves at the start, want 20", got)
	}
}

func TestGenerateMovesDoesNotChangePosition(t *testing.T) {
	for _, tc := range suiteFENs {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)
			before := p.FEN()

			for sq := 0; sq < 64; sq++ {
				p.GenerateMoves(sq)

				if pc := p.Board.PieceAt(sq); pc.Type != Empty && pc.Color == p.SideToMove {
					p.LegalMoves(sq)
				}
			}

			if after := p.FEN(); after != before {
				t.Fatalf("position changed:\n%s\n%s", before, after)
			}
		})
	}
}

// LegalMoves makes and unmakes each candidate move, so asking about a piece
// that does not belong to the side to move must still leave the board intact.
// UnmakeMove once took the castling colour from SideToMove, which corrupted the
// rooks in exactly this case.
func TestLegalMovesForSideNotToMoveRestoresPosition(t *testing.T) {
	for _, tc := range suiteFENs {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)
			before := p.FEN()

			for sq := 0; sq < 64; sq++ {
				p.LegalMoves(sq)
			}

			if after := p.FEN(); after != before {
				t.Fatalf("position changed:\n%s\n%s", before, after)
			}
		})
	}
}

func TestLegalMovesNeverLeaveKingInCheck(t *testing.T) {
	for _, tc := range suiteFENs {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)
			side := p.SideToMove

			for _, m := range legalMoves(&p) {
				undo := p.MakeMove(m)

				if p.InCheck(side) {
					t.Fatalf("%s leaves the king in check", moveName(m))
				}

				p.UnmakeMove(m, undo)
			}
		})
	}
}
