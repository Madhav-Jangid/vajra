package chess

import "testing"

func BenchmarkPerft(b *testing.B) {
	tests := []struct {
		name  string
		fen   string
		depth int
		nodes int
	}{
		{"start d3", StartingFEN, 3, 8902},
		{"kiwipete d3", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", 3, 97862},
		{"pos3 d4", "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1", 4, 43238},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			p, err := ParseFEN(tc.fen)
			if err != nil {
				b.Fatal(err)
			}

			total := 0

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if got := perft(&p, tc.depth); got != tc.nodes {
					b.Fatalf("perft = %d, want %d", got, tc.nodes)
				}

				total += tc.nodes
			}

			b.ReportMetric(float64(total)/b.Elapsed().Seconds(), "nodes/s")
		})
	}
}

func BenchmarkParseFEN(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := ParseFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLegalMoves(b *testing.B) {
	p, _ := ParseFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	for i := 0; i < b.N; i++ {
		legalMoves(&p)
	}
}
