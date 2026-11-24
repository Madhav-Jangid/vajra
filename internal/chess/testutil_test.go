package chess

import (
	"sort"
	"testing"
)

func mustFEN(t *testing.T, fen string) Position {
	t.Helper()

	p, err := ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN(%q): %v", fen, err)
	}

	return p
}

func sqOf(name string) int {
	return int(name[0]-'a') + int(name[1]-'1')*8
}

func legalMoves(p *Position) []Move {
	var out []Move

	for sq := 0; sq < 64; sq++ {
		pc := p.Board.PieceAt(sq)
		if pc.Type != Empty && pc.Color == p.SideToMove {
			out = append(out, p.LegalMoves(sq)...)
		}
	}

	return out
}

func legalNames(p *Position) []string {
	var out []string

	for _, m := range legalMoves(p) {
		out = append(out, moveName(m))
	}

	sort.Strings(out)

	return out
}

func legalFrom(p *Position, square string) []string {
	var out []string

	for _, m := range p.LegalMoves(sqOf(square)) {
		out = append(out, moveName(m))
	}

	sort.Strings(out)

	return out
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}

	return false
}

func sameList(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	x := append([]string(nil), a...)
	y := append([]string(nil), b...)

	sort.Strings(x)
	sort.Strings(y)

	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}

	return true
}

func findMove(t *testing.T, p *Position, name string) Move {
	t.Helper()

	for _, m := range legalMoves(p) {
		if moveName(m) == name {
			return m
		}
	}

	t.Fatalf("move %s is not legal in %s", name, p.FEN())

	return Move{}
}

// checkBoardConsistent verifies the bitboards describe a sane position:
// no square holds two pieces and each side has exactly one king.
func checkBoardConsistent(t *testing.T, p *Position) {
	t.Helper()

	b := &p.Board
	boards := []Bitboard{
		b.WhitePawns, b.WhiteKnights, b.WhiteBishops, b.WhiteRooks, b.WhiteQueens, b.WhiteKing,
		b.BlackPawns, b.BlackKnights, b.BlackBishops, b.BlackRooks, b.BlackQueens, b.BlackKing,
	}

	var seen Bitboard

	for i, bb := range boards {
		if seen&bb != 0 {
			t.Fatalf("bitboard %d overlaps another piece in %s", i, p.FEN())
		}

		seen |= bb
	}

	if popCount(b.WhiteKing) != 1 || popCount(b.BlackKing) != 1 {
		t.Fatalf("each side must have one king in %s", p.FEN())
	}
}

func popCount(b Bitboard) int {
	n := 0

	for ; b != 0; b &= b - 1 {
		n++
	}

	return n
}

var suiteFENs = []struct{ name, fen string }{
	{"start", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"},
	{"kiwipete", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"},
	{"pos3", "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1"},
	{"pos4", "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1"},
	{"pos4 mirrored", "r2q1rk1/pP1p2pp/Q4n2/bbp1p3/Np6/1B3NBn/pPPP1PPP/R3K2R b KQ - 0 1"},
	{"pos5", "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8"},
	{"pos6", "r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10"},
}
