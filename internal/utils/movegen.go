package utils

import (
	"vajra/internal/chess"
)

type Moves []int
type Directions [][2]int

func GenerateMoves(square int) Moves {
	chess.HasCheck = false
	if square < 0 || square >= 64 {
		return nil
	}

	var directions Directions
	var moves Moves
	rank := square / 8
	file := square % 8
	piece := GetPieceAt(square)

	switch {
	case piece.Type == chess.BlackKing && piece.Color == chess.Black:
		return GetKingsMoves(square, piece.Color)
	case piece.Type == chess.WhiteKing && piece.Color == chess.White:
		return GetKingsMoves(square, piece.Color)
	case piece.Type == chess.BlackQueen && piece.Color == chess.Black:
		directions = GetQueensDirections()
	case piece.Type == chess.WhiteQueen && piece.Color == chess.White:
		directions = GetQueensDirections()
	case piece.Type == chess.BlackBishop && piece.Color == chess.Black:
		directions = GetBishopsDirections()
	case piece.Type == chess.WhiteBishop && piece.Color == chess.White:
		directions = GetBishopsDirections()
	case piece.Type == chess.BlackKnight && piece.Color == chess.Black:
		directions = GetKnightsDirections()
	case piece.Type == chess.WhiteKnight && piece.Color == chess.White:
		directions = GetKnightsDirections()
	case piece.Type == chess.BlackRook && piece.Color == chess.Black:
		directions = GetRooksDirections()
	case piece.Type == chess.WhiteRook && piece.Color == chess.White:
		directions = GetRooksDirections()
	case piece.Type == chess.BlackPawn && piece.Color == chess.Black:
		return GetPawnsMoves(square, piece.Color)
	case piece.Type == chess.WhitePawn && piece.Color == chess.White:
		return GetPawnsMoves(square, piece.Color)
	default:
		directions = nil
	}

	for _, direction := range directions {
		r := rank + direction[0]
		f := file + direction[1]

		if piece.Type == chess.BlackKnight || piece.Type == chess.WhiteKnight {
			if r < 0 || r >= 8 || f < 0 || f >= 8 {
				continue
			}

			moveSquare := r*8 + f
			squarePiece := GetPieceAt(moveSquare)

			if !IsSquareOccupied(&chess.BOARD, moveSquare) {
				moves = append(moves, moveSquare)
			} else if squarePiece.Color != piece.Color {
				moves = append(moves, moveSquare)
			}

			continue
		}

		for r >= 0 && r < 8 && f >= 0 && f < 8 {
			moveSquare := r*8 + f
			squarePiece := GetPieceAt(moveSquare)
			if IsSquareOccupied(&chess.BOARD, moveSquare) {
				if squarePiece.Color != piece.Color {
					moves = append(moves, moveSquare)
				}
				break
			}

			moves = append(moves, moveSquare)
			r += direction[0]
			f += direction[1]
		}
	}
	moves = FilterMoves(moves)
	return moves
}
func GetQueensDirections() Directions {
	directions := Directions{
		{-1, 1},
		{0, 1},
		{1, 1},
		{1, 0},

		{1, -1},
		{0, -1},
		{-1, -1},
		{-1, 0},
	}
	return directions
}
func GetKingsDirections() Directions {
	directions := Directions{
		{-1, 1},
		{0, 1},
		{1, 1},
		{1, 0},

		{1, -1},
		{0, -1},
		{-1, -1},
		{-1, 0},
	}
	return directions
}
func GetKingsMoves(square int, color chess.Color) Moves {
	var moves Moves

	rank := square / 8
	file := square % 8

	directions := GetKingsDirections()

	for _, direction := range directions {
		r := rank + direction[0]
		f := file + direction[1]

		if r < 0 || r >= 8 || f < 0 || f >= 8 {
			continue
		}

		moveSquare := r*8 + f

		if IsSquareOccupied(&chess.BOARD, moveSquare) {
			target := GetPieceAt(moveSquare)

			if target.Color == color {
				continue
			}
		}

		enemy := chess.White
		if color == chess.White {
			enemy = chess.Black
		}

		if IsSquareAttacked(moveSquare, enemy, chess.BOARD) {
			continue
		}

		moves = append(moves, moveSquare)
	}

	return moves
}
func GetRooksDirections() Directions {
	directions := Directions{
		{0, 1},
		{1, 0},
		{0, -1},
		{-1, 0},
	}
	return directions
}
func GetBishopsDirections() Directions {
	directions := Directions{
		{-1, 1},
		{1, 1},
		{1, -1},
		{-1, -1},
	}
	return directions
}
func GetKnightsDirections() Directions {
	directions := Directions{
		{2, 1},
		{2, -1},
		{-2, 1},
		{-2, -1},

		{1, 2},
		{1, -2},
		{-1, 2},
		{-1, -2},
	}
	return directions
}
func GetPawnsMoves(square int, color chess.Color) Moves {
	var moves Moves

	rank := square / 8
	file := square % 8

	forward := 8
	startRank := 1

	if color == chess.Black {
		forward = -8
		startRank = 6
	}

	// 1 square forward
	one := square + forward

	if one >= 0 && one < 64 && !IsSquareOccupied(&chess.BOARD, one) {
		moves = append(moves, one)

		// 2 squares forward
		if rank == startRank {
			two := square + (2 * forward)

			if !IsSquareOccupied(&chess.BOARD, two) {
				moves = append(moves, two)
			}
		}
	}

	// Capture left
	if file > 0 {
		capture := square + forward - 1

		if capture >= 0 && capture < 64 {
			if IsSquareOccupied(&chess.BOARD, capture) {
				target := GetPieceAt(capture)

				if target.Color != color {
					moves = append(moves, capture)
				}
			}
		}
	}

	// Capture right
	if file < 7 {
		capture := square + forward + 1

		if capture >= 0 && capture < 64 {
			if IsSquareOccupied(&chess.BOARD, capture) {
				target := GetPieceAt(capture)

				if target.Color != color {
					moves = append(moves, capture)
				}
			}
		}
	}

	return moves
}
