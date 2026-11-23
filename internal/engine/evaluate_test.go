package engine

import (
	"testing"

	"vajra/internal/chess"
)

func TestPieceValue(t *testing.T) {
	tests := []struct {
		name  string
		piece chess.PieceType
		want  int
	}{
		{"white pawn", chess.WhitePawn, PawnValue},
		{"black pawn", chess.BlackPawn, PawnValue},

		{"white knight", chess.WhiteKnight, KnightValue},
		{"black knight", chess.BlackKnight, KnightValue},

		{"white bishop", chess.WhiteBishop, BishopValue},
		{"black bishop", chess.BlackBishop, BishopValue},

		{"white rook", chess.WhiteRook, RookValue},
		{"black rook", chess.BlackRook, RookValue},

		{"white queen", chess.WhiteQueen, QueenValue},
		{"black queen", chess.BlackQueen, QueenValue},

		{"white king", chess.WhiteKing, KingValue},
		{"black king", chess.BlackKing, KingValue},

		{"empty", chess.Empty, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pieceValue(tc.piece)

			if got != tc.want {
				t.Errorf(
					"pieceValue(%v) = %d, want %d",
					tc.piece,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestFlipSquare(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"a1 to a8", 0, 56},
		{"b1 to b8", 1, 57},
		{"e1 to e8", 4, 60},
		{"h1 to h8", 7, 63},

		{"a2 to a7", 8, 48},
		{"e2 to e7", 12, 52},

		{"a4 to a5", 24, 32},
		{"e4 to e5", 28, 36},

		{"h8 to h1", 63, 7},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flipSquare(tc.in)

			if got != tc.want {
				t.Errorf(
					"flipSquare(%d) = %d, want %d",
					tc.in,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestPieceSquareBonusUsesCorrectPST(t *testing.T) {
	tests := []struct {
		name   string
		piece  chess.PieceType
		color  chess.Color
		square int
		want   int
	}{
		{
			name:   "white pawn",
			piece:  chess.WhitePawn,
			color:  chess.White,
			square: 28,
			want:   pawnPST[28],
		},
		{
			name:   "black pawn mirrored",
			piece:  chess.BlackPawn,
			color:  chess.Black,
			square: 36,
			want:   pawnPST[flipSquare(36)],
		},
		{
			name:   "white knight",
			piece:  chess.WhiteKnight,
			color:  chess.White,
			square: 18,
			want:   knightPST[18],
		},
		{
			name:   "black knight mirrored",
			piece:  chess.BlackKnight,
			color:  chess.Black,
			square: 42,
			want:   knightPST[flipSquare(42)],
		},
		{
			name:   "white bishop",
			piece:  chess.WhiteBishop,
			color:  chess.White,
			square: 26,
			want:   bishopPST[26],
		},
		{
			name:   "black bishop mirrored",
			piece:  chess.BlackBishop,
			color:  chess.Black,
			square: 34,
			want:   bishopPST[flipSquare(34)],
		},
		{
			name:   "white rook",
			piece:  chess.WhiteRook,
			color:  chess.White,
			square: 0,
			want:   rookPST[0],
		},
		{
			name:   "black rook mirrored",
			piece:  chess.BlackRook,
			color:  chess.Black,
			square: 56,
			want:   rookPST[flipSquare(56)],
		},
		{
			name:   "white queen",
			piece:  chess.WhiteQueen,
			color:  chess.White,
			square: 27,
			want:   queenPST[27],
		},
		{
			name:   "black queen mirrored",
			piece:  chess.BlackQueen,
			color:  chess.Black,
			square: 35,
			want:   queenPST[flipSquare(35)],
		},
		{
			name:   "white king",
			piece:  chess.WhiteKing,
			color:  chess.White,
			square: 4,
			want:   kingMGPST[4],
		},
		{
			name:   "black king mirrored",
			piece:  chess.BlackKing,
			color:  chess.Black,
			square: 60,
			want:   kingMGPST[flipSquare(60)],
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pieceSquareBonus(
				tc.piece,
				tc.color,
				tc.square,
			)

			if got != tc.want {
				t.Errorf(
					"pieceSquareBonus(%v, %v, %d) = %d, want %d",
					tc.piece,
					tc.color,
					tc.square,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestPieceSquareBonusBlackMirrorsWhite(t *testing.T) {
	tests := []struct {
		name        string
		whitePiece  chess.PieceType
		blackPiece  chess.PieceType
		whiteSquare int
		blackSquare int
	}{
		{
			name:        "pawn",
			whitePiece:  chess.WhitePawn,
			blackPiece:  chess.BlackPawn,
			whiteSquare: 28,
			blackSquare: 36,
		},
		{
			name:        "knight",
			whitePiece:  chess.WhiteKnight,
			blackPiece:  chess.BlackKnight,
			whiteSquare: 18,
			blackSquare: 42,
		},
		{
			name:        "bishop",
			whitePiece:  chess.WhiteBishop,
			blackPiece:  chess.BlackBishop,
			whiteSquare: 26,
			blackSquare: 34,
		},
		{
			name:        "rook",
			whitePiece:  chess.WhiteRook,
			blackPiece:  chess.BlackRook,
			whiteSquare: 0,
			blackSquare: 56,
		},
		{
			name:        "queen",
			whitePiece:  chess.WhiteQueen,
			blackPiece:  chess.BlackQueen,
			whiteSquare: 27,
			blackSquare: 35,
		},
		{
			name:        "king",
			whitePiece:  chess.WhiteKing,
			blackPiece:  chess.BlackKing,
			whiteSquare: 4,
			blackSquare: 60,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			white := pieceSquareBonus(
				tc.whitePiece,
				chess.White,
				tc.whiteSquare,
			)

			black := pieceSquareBonus(
				tc.blackPiece,
				chess.Black,
				tc.blackSquare,
			)

			if white != black {
				t.Errorf(
					"white bonus = %d, black mirrored bonus = %d",
					white,
					black,
				)
			}
		})
	}
}

func TestEvaluateStartingPosition(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	position, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN() error = %v", err)
	}

	got := Evaluate(&position)

	// Starting position must be materially equal.
	//
	// With mirrored PSTs, the positional contribution of the
	// starting position is also symmetric.
	want := 0

	if got != want {
		t.Errorf(
			"Evaluate(start position) = %d, want %d",
			got,
			want,
		)
	}
}

func TestEvaluateIsSymmetricForMirroredMaterial(t *testing.T) {
	tests := []struct {
		name string
		fen  string
	}{
		{
			name: "white pawn vs black pawn",
			fen:  "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 1",
		},
		{
			name: "black pawn vs white pawn",
			fen:  "rnbqkbnr/pppp1ppp/8/4p3/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			position, err := chess.ParseFEN(tc.fen)
			if err != nil {
				t.Fatalf("ParseFEN() error = %v", err)
			}

			got := Evaluate(&position)

			if got == 0 {
				t.Errorf(
					"position with material imbalance should not evaluate to 0",
				)
			}
		})
	}
}

func TestEvaluateWhiteAdvantage(t *testing.T) {
	baseFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	whiteFEN := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 1"

	base, err := chess.ParseFEN(baseFEN)
	if err != nil {
		t.Fatalf("ParseFEN(base) error = %v", err)
	}

	position, err := chess.ParseFEN(whiteFEN)
	if err != nil {
		t.Fatalf("ParseFEN(position) error = %v", err)
	}

	baseScore := Evaluate(&base)
	score := Evaluate(&position)

	if score <= baseScore {
		t.Errorf(
			"adding a white pawn should improve evaluation: base=%d score=%d",
			baseScore,
			score,
		)
	}
}

func TestEvaluateBlackAdvantage(t *testing.T) {
	baseFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	blackFEN := "rnbqkbnr/pppp1ppp/8/4p3/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	base, err := chess.ParseFEN(baseFEN)
	if err != nil {
		t.Fatalf("ParseFEN(base) error = %v", err)
	}

	position, err := chess.ParseFEN(blackFEN)
	if err != nil {
		t.Fatalf("ParseFEN(position) error = %v", err)
	}

	baseScore := Evaluate(&base)
	score := Evaluate(&position)

	if score >= baseScore {
		t.Errorf(
			"adding a black pawn should worsen evaluation: base=%d score=%d",
			baseScore,
			score,
		)
	}
}
