package chess

import "testing"

func sanOf(t *testing.T, fen, move string) string {
	t.Helper()

	p := mustFEN(t, fen)
	m := findMove(t, &p, move)
	before := p

	text := p.SAN(m)

	if p != before {
		t.Fatalf("SAN changed the position for %s in %s", move, fen)
	}

	return text
}

func TestSAN(t *testing.T) {
	tests := []struct {
		name string
		fen  string
		move string
		want string
	}{
		{"pawn push", StartingFEN, "e2e4", "e4"},
		{"knight", StartingFEN, "g1f3", "Nf3"},
		{"pawn capture", "4k3/8/8/3p4/4P3/8/8/4K3 w - - 0 1", "e4d5", "exd5"},
		{"piece capture", "4k3/8/8/3p4/8/2N5/8/4K3 w - - 0 1", "c3d5", "Nxd5"},
		{"king move", "4k3/8/8/8/8/8/8/4K3 w - - 0 1", "e1e2", "Ke2"},
		{"white king-side castle", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", "e1g1", "O-O"},
		{"white queen-side castle", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", "e1c1", "O-O-O"},
		{"black king-side castle", "r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1", "e8g8", "O-O"},
		{"castle that gives check", "5k2/8/8/8/8/8/8/4K2R w K - 0 1", "e1g1", "O-O+"},
		{"promotion", "7k/P7/8/8/8/8/8/K7 w - - 0 1", "a7a8q", "a8=Q+"},
		{"underpromotion", "7k/P7/8/8/8/8/8/K7 w - - 0 1", "a7a8n", "a8=N"},
		{"promotion with capture", "1n5k/P7/8/8/8/8/8/K7 w - - 0 1", "a7b8q", "axb8=Q+"},
		{"black promotion", "k7/8/8/8/8/8/p7/7K b - - 0 1", "a2a1q", "a1=Q+"},
		{"en passant", "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1", "e5d6", "exd6"},
		{"check", "4k3/8/8/8/8/8/8/R3K3 w - - 0 1", "a1a8", "Ra8+"},
		{"checkmate", "6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1", "a1a8", "Ra8#"},
		{"stalemate is not mate", "7k/8/5Q2/8/8/8/8/6K1 w - - 0 1", "f6f7", "Qf7"},
		{"disambiguate by file", "4k3/8/8/8/8/5N2/8/1N2K3 w - - 0 1", "b1d2", "Nbd2"},
		{"the other knight by file", "4k3/8/8/8/8/5N2/8/1N2K3 w - - 0 1", "f3d2", "Nfd2"},
		{"no disambiguation when one piece can move", "4k3/8/8/2p5/8/8/8/N3K1N1 w - - 0 1", "a1b3", "Nb3"},
		{"disambiguate by rank", "4k3/8/8/8/R7/8/8/R3K3 w - - 0 1", "a1a3", "R1a3"},
		{"disambiguate by file and rank", "4k3/8/8/8/8/Q7/8/Q1Q1K3 w - - 0 1", "a1b2", "Qa1b2"},
		{"disambiguate two queens on a rank", "4k3/8/8/8/Q6Q/8/8/4K3 w - - 0 1", "a4e4", "Qae4+"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanOf(t, tc.fen, tc.move); got != tc.want {
				t.Fatalf("SAN = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSANForEveryLegalMoveIsUniqueAndRestores(t *testing.T) {
	for _, tc := range suiteFENs {
		t.Run(tc.name, func(t *testing.T) {
			p := mustFEN(t, tc.fen)
			before := p
			seen := map[string]string{}

			for _, m := range legalMoves(&p) {
				text := p.SAN(m)

				if text == "" {
					t.Fatalf("%s has no notation", moveName(m))
				}

				if other, dup := seen[text]; dup {
					t.Fatalf("%s and %s are both written %q", moveName(m), other, text)
				}

				seen[text] = moveName(m)
			}

			if p != before {
				t.Fatal("SAN changed the position")
			}
		})
	}
}
