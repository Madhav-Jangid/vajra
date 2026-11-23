package engine

import (
	"vajra/internal/chess"
)

const (
	PawnValue   = 100
	KnightValue = 320
	BishopValue = 330
	RookValue   = 500
	QueenValue  = 900
	KingValue   = 20000
)

var pawnPST = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0,
	5, 10, 10, -20, -20, 10, 10, 5,
	5, -5, -10, 0, 0, -10, -5, 5,
	0, 0, 0, 20, 20, 0, 0, 0,
	5, 5, 10, 25, 25, 10, 5, 5,
	10, 10, 20, 30, 30, 20, 10, 10,
	50, 50, 50, 50, 50, 50, 50, 50,
	0, 0, 0, 0, 0, 0, 0, 0,
}

var knightPST = [64]int{
	-50, -40, -30, -30, -30, -30, -40, -50,
	-40, -20, 0, 0, 0, 0, -20, -40,
	-30, 0, 10, 15, 15, 10, 0, -30,
	-30, 5, 15, 20, 20, 15, 5, -30,
	-30, 0, 15, 20, 20, 15, 0, -30,
	-30, 5, 10, 15, 15, 10, 5, -30,
	-40, -20, 0, 5, 5, 0, -20, -40,
	-50, -40, -30, -30, -30, -30, -40, -50,
}

var bishopPST = [64]int{
	-20, -10, -10, -10, -10, -10, -10, -20,
	-10, 5, 0, 0, 0, 0, 5, -10,
	-10, 10, 10, 10, 10, 10, 10, -10,
	-10, 0, 10, 10, 10, 10, 0, -10,
	-10, 5, 5, 10, 10, 5, 5, -10,
	-10, 0, 5, 10, 10, 5, 0, -10,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-20, -10, -10, -10, -10, -10, -10, -20,
}

var rookPST = [64]int{
	0, 0, 0, 5, 5, 0, 0, 0,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	5, 10, 10, 10, 10, 10, 10, 5,
	0, 0, 0, 0, 0, 0, 0, 0,
}

var queenPST = [64]int{
	-20, -10, -10, -5, -5, -10, -10, -20,
	-10, 0, 5, 0, 0, 0, 0, -10,
	-10, 5, 5, 5, 5, 5, 0, -10,
	0, 0, 5, 5, 5, 5, 0, -5,
	-5, 0, 5, 5, 5, 5, 0, -5,
	-10, 0, 5, 5, 5, 5, 0, -10,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-20, -10, -10, -5, -5, -10, -10, -20,
}

var kingMGPST = [64]int{
	20, 30, 10, 0, 0, 10, 30, 20,
	20, 20, 0, 0, 0, 0, 20, 20,
	-10, -20, -20, -20, -20, -20, -20, -10,
	-20, -30, -30, -40, -40, -30, -30, -20,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
}

// var kingEGPST = [64]int{
// 	-50, -30, -30, -30, -30, -30, -30, -50,
// 	-30, -30, 0, 0, 0, 0, -30, -30,
// 	-30, -10, 20, 30, 30, 20, -10, -30,
// 	-30, -10, 30, 40, 40, 30, -10, -30,
// 	-30, -10, 30, 40, 40, 30, -10, -30,
// 	-30, -10, 20, 30, 30, 20, -10, -30,
// 	-30, -20, -10, 0, 0, -10, -20, -30,
// 	-50, -40, -30, -20, -20, -30, -40, -50,
// }

func Evaluate(p *chess.Position) int {
	board := p.Board
	score := 0

	for square := 0; square < 64; square++ {
		piece := board.PieceAt(square)

		if piece.Type == chess.Empty {
			continue
		}
		mValue := pieceValue(piece.Type)
		pstValue := pieceSquareBonus(piece.Type, piece.Color, square)

		if piece.Color == chess.White {
			score += mValue + pstValue
		} else {
			score -= mValue + pstValue
		}
	}

	return score
}

func pieceValue(piece chess.PieceType) int {
	switch piece {
	case chess.WhitePawn, chess.BlackPawn:
		return PawnValue

	case chess.WhiteKnight, chess.BlackKnight:
		return KnightValue

	case chess.WhiteBishop, chess.BlackBishop:
		return BishopValue

	case chess.WhiteRook, chess.BlackRook:
		return RookValue

	case chess.WhiteQueen, chess.BlackQueen:
		return QueenValue

	case chess.WhiteKing, chess.BlackKing:

		return KingValue

	default:
		return 0
	}
}

func pieceSquareBonus(
	piece chess.PieceType,
	color chess.Color,
	square int,
) int {
	if color == chess.Black {
		square = flipSquare(square)
	}

	switch piece {
	case chess.WhitePawn, chess.BlackPawn:
		return pawnPST[square]

	case chess.WhiteKnight, chess.BlackKnight:
		return knightPST[square]

	case chess.WhiteBishop, chess.BlackBishop:
		return bishopPST[square]

	case chess.WhiteRook, chess.BlackRook:
		return rookPST[square]

	case chess.WhiteQueen, chess.BlackQueen:
		return queenPST[square]

	case chess.WhiteKing, chess.BlackKing:
		return kingMGPST[square]

	default:
		return 0
	}
}

func flipSquare(square int) int {
	file := square % 8
	rank := square / 8

	return (7-rank)*8 + file
}

// const maxPhase = 24

// var phaseWeights = [6]int{
// 	0, // Pawn
// 	1, // Knight
// 	1, // Bishop
// 	2, // Rook
// 	4, // Queen
// 	0, // King
// }

// func calculatePhaseScore() int {
// 	pawns := bits.OnesCount64(chess.WhitePawn) + bits.OnesCount64(chess.BlackPawn)
// 	knights := bits.OnesCount64(chess.WhiteKnight) + bits.OnesCount64(chess.BlackKnight)
// 	bishops := bits.OnesCount64(chess.WhiteBishop) + bits.OnesCount64(chess.BlackBishop)
// 	rooks := bits.OnesCount64(chess.WhiteRook) + bits.OnesCount64(chess.BlackRook)
// 	queens := bits.OnesCount64(chess.WhiteQueen) + bits.OnesCount64(chess.BlackQueen)

// 	return pawns + knights + bishops + rooks + queens
// }

// func taperedEval(phase int) int {
// 	if phase > maxPhase {
// 		phase = maxPhase
// 	}
// 	return (mgScore*phase + egScore*(maxPhase-phase)) / maxPhase
// }
