package engine

import (
	"testing"

	"vajra/internal/chess"
)

func benchmarkPosition() chess.Position {
	return newPosition(chess.White, map[chess.PieceType][]int{
		chess.WhiteKing:   {4},
		chess.WhiteQueen:  {3},
		chess.WhiteRook:   {0, 7},
		chess.WhiteBishop: {2, 5},
		chess.WhiteKnight: {1, 6},
		chess.WhitePawn: {
			8, 9, 10, 11, 12, 13, 14, 15,
		},

		chess.BlackKing:   {60},
		chess.BlackQueen:  {59},
		chess.BlackRook:   {56, 63},
		chess.BlackBishop: {58, 61},
		chess.BlackKnight: {57, 62},
		chess.BlackPawn: {
			48, 49, 50, 51, 52, 53, 54, 55,
		},
	})
}

func setOrdering(enabled bool) (restore func()) {
	old := UseMoveOrdering
	UseMoveOrdering = enabled

	return func() { UseMoveOrdering = old }
}

func BenchmarkSearchOrdering(b *testing.B) {
	for _, ordering := range []bool{false, true} {
		name := "WithoutOrdering"
		if ordering {
			name = "WithOrdering"
		}

		b.Run(name, func(b *testing.B) {
			defer setOrdering(ordering)()

			var totalNodes uint64

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				p := benchmarkPosition()

				Search(&p, 4, 0)

				totalNodes += Nodes
			}

			b.ReportMetric(float64(totalNodes)/float64(b.N), "nodes/search")
		})
	}
}
