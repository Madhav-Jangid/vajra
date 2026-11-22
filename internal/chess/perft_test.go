package chess

import (
	"sort"
	"testing"
)

func perft(p *Position, depth int) int {
	if depth == 0 {
		return 1
	}

	side := p.SideToMove
	nodes := 0

	for sq := 0; sq < 64; sq++ {
		pc := p.Board.PieceAt(sq)

		if pc.Type == Empty || pc.Color != side {
			continue
		}

		for _, move := range p.LegalMoves(sq) {
			undo := p.MakeMove(move)

			nodes += perft(p, depth-1)

			p.UnmakeMove(move, undo)
		}
	}

	return nodes
}

// perftDivide calculates the node count for every legal root move.
//
// Example:
//
//	e2e4  600
//	d2d4  560
//	g1f3  440
//
// This is useful when a normal perft count is wrong because it tells
// you which root move causes the mismatch.
func perftDivide(p *Position, depth int) map[string]int {
	result := make(map[string]int)

	side := p.SideToMove

	for sq := 0; sq < 64; sq++ {
		pc := p.Board.PieceAt(sq)

		if pc.Type == Empty || pc.Color != side {
			continue
		}

		for _, move := range p.LegalMoves(sq) {
			undo := p.MakeMove(move)

			nodes := perft(p, depth-1)

			p.UnmakeMove(move, undo)

			result[moveName(move)] = nodes
		}
	}

	return result
}

// moveName converts a move to coordinate notation.
//
// Examples:
//
//	12 -> 28 = e2e4
//	6  -> 21 = g1f3
//
// Promotions are represented as:
//
//	e7e8q
func moveName(move Move) string {
	name := squareName(move.From) + squareName(move.To)

	if move.Promotion != Empty {
		switch move.Promotion {
		case WhiteQueen, BlackQueen:
			name += "q"
		case WhiteRook, BlackRook:
			name += "r"
		case WhiteBishop, BlackBishop:
			name += "b"
		case WhiteKnight, BlackKnight:
			name += "n"
		}
	}

	return name
}

func squareName(square int) string {
	file := square % 8
	rank := square / 8

	return string(rune('a'+file)) +
		string(rune('1'+rank))
}

func TestFENRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		fen  string
	}{
		{
			name: "start position",
			fen:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		},
		{
			name: "kiwipete",
			fen:  "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
		},
		{
			name: "position 3",
			fen:  "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
		},
		{
			name: "position 4",
			fen:  "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1",
		},
		{
			name: "position 5",
			fen:  "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ParseFEN(tc.fen)
			if err != nil {
				t.Fatalf("ParseFEN() error: %v", err)
			}

			got := p.FEN()

			if got != tc.fen {
				t.Fatalf(
					"FEN round-trip mismatch\nwant: %s\ngot:  %s",
					tc.fen,
					got,
				)
			}
		})
	}
}

func TestPerft(t *testing.T) {
	tests := []struct {
		name  string
		fen   string
		depth int
		want  int
	}{
		{
			name:  "start d3",
			fen:   "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			depth: 3,
			want:  8902,
		},
		{
			name:  "start d4",
			fen:   "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			depth: 4,
			want:  197281,
		},
		{
			name:  "kiwipete d3",
			fen:   "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
			depth: 3,
			want:  97862,
		},
		{
			name:  "pos3 d4",
			fen:   "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
			depth: 4,
			want:  43238,
		},
		{
			name:  "pos4 d3",
			fen:   "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1",
			depth: 3,
			want:  9467,
		},
		{
			name:  "pos5 d3",
			fen:   "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
			depth: 3,
			want:  62379,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ParseFEN(tc.fen)
			if err != nil {
				t.Fatalf("ParseFEN() error: %v", err)
			}

			before := p.FEN()

			got := perft(&p, tc.depth)

			if got != tc.want {
				t.Errorf(
					"perft depth %d = %d, want %d",
					tc.depth,
					got,
					tc.want,
				)
			}

			after := p.FEN()

			if after != before {
				t.Errorf(
					"position not restored\nbefore: %s\nafter:  %s",
					before,
					after,
				)
			}
		})
	}
}

// This test checks the root move breakdown for the starting position.
//
// These values are the standard perft divide values for depth 3.
func TestPerftDivideStartPosition(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	p, err := ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN() error: %v", err)
	}

	want := map[string]int{
		"a2a3": 380,
		"a2a4": 420,
		"b1a3": 400,
		"b1c3": 440,
		"b2b3": 420,
		"b2b4": 421,
		"c2c3": 420,
		"c2c4": 441,
		"d2d3": 539,
		"d2d4": 560,
		"e2e3": 599,
		"e2e4": 600,
		"f2f3": 380,
		"f2f4": 401,
		"g1f3": 440,
		"g1h3": 400,
		"g2g3": 420,
		"g2g4": 421,
		"h2h3": 380,
		"h2h4": 420,
	}

	got := perftDivide(&p, 3)

	if len(got) != len(want) {
		t.Fatalf(
			"perft divide returned %d moves, want %d",
			len(got),
			len(want),
		)
	}

	for move, wantNodes := range want {
		gotNodes, ok := got[move]

		if !ok {
			t.Errorf("missing move %s", move)
			continue
		}

		if gotNodes != wantNodes {
			t.Errorf(
				"%s: got %d nodes, want %d",
				move,
				gotNodes,
				wantNodes,
			)
		}
	}
}

// This is mainly useful while debugging.
// It prints the divide in a stable order so the output is easy to compare.
func TestPerftDivideKiwipete(t *testing.T) {
	fen := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"

	p, err := ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN() error: %v", err)
	}

	got := perftDivide(&p, 3)

	moves := make([]string, 0, len(got))

	for move := range got {
		moves = append(moves, move)
	}

	sort.Strings(moves)

	for _, move := range moves {
		t.Logf("%s: %d", move, got[move])
	}
}
