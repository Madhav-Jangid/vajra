package utils

import (
	"fmt"
	"math/bits"
	"slices"
	"strings"
	"vajra/internal/chess"
)

func PrintBoard() {
	for rank := 7; rank >= 0; rank-- {
		fmt.Printf("%d ", rank+1)
		for file := 0; file <= 7; file++ {
			square := rank*8 + file
			piece := GetPieceAt(square)
			fmt.Print(piece.Face, " ")
		}
		fmt.Println()
	}
	fmt.Println("  a b c d e f g h")
}

func GetPieceAt(square int) chess.Piece {
	return PieceAtBoard(chess.BOARD, square)
}

// PieceAtBoard returns the piece on `square` of the given board.
func PieceAtBoard(BOARD chess.PieceBoards, square int) chess.Piece {
	mask := chess.Bitboard(1) << square
	var piece string
	switch {
	case BOARD.WhiteKing&mask != 0:
		piece = "K"
	case BOARD.BlackKing&mask != 0:
		piece = "k"
	case BOARD.WhiteQueens&mask != 0:
		piece = "Q"
	case BOARD.BlackQueens&mask != 0:
		piece = "q"
	case BOARD.WhiteKnights&mask != 0:
		piece = "N"
	case BOARD.BlackKnights&mask != 0:
		piece = "n"
	case BOARD.WhiteBishops&mask != 0:
		piece = "B"
	case BOARD.BlackBishops&mask != 0:
		piece = "b"
	case BOARD.WhiteRooks&mask != 0:
		piece = "R"
	case BOARD.BlackRooks&mask != 0:
		piece = "r"
	case BOARD.WhitePawns&mask != 0:
		piece = "P"
	case BOARD.BlackPawns&mask != 0:
		piece = "p"
	default:
		piece = "."
	}

	pieceType := chess.PieceTypeMap[piece]

	return chess.Piece{
		Face:  piece,
		Type:  pieceType,
		Color: GetColor(pieceType),
	}
}

func GetColor(pieceType chess.PieceType) chess.Color {
	switch pieceType {
	case chess.WhiteKing,
		chess.WhiteQueen,
		chess.WhiteRook,
		chess.WhiteKnight,
		chess.WhiteBishop,
		chess.WhitePawn:
		return chess.White

	default:
		return chess.Black
	}
}

// SquareName converts a square index to algebraic notation, e.g. 12 -> "e2".
func SquareName(sq int) string {
	return string(rune('a'+sq%8)) + string(rune('1'+sq/8))
}

func ParseMove(move string) []int {
	move = strings.Trim(move, " ")
	if len(move) != 4 {
		return nil
	}

	fileMapings := map[string]int{
		"a": 0,
		"b": 1,
		"c": 2,
		"d": 3,
		"e": 4,
		"f": 5,
		"g": 6,
		"h": 7,
	}
	rankMapings := map[string]int{
		"1": 0,
		"2": 1,
		"3": 2,
		"4": 3,
		"5": 4,
		"6": 5,
		"7": 6,
		"8": 7,
	}

	var moves []string
	moves = append(moves, move[:2], move[2:])

	var fromIdx, toIdx int
	for i, move := range moves {
		file := fileMapings[move[:1]]
		rank := rankMapings[move[1:]]
		if i == 0 {
			fromIdx = rank*8 + file
		} else {
			toIdx = rank*8 + file
		}
	}
	return []int{
		fromIdx,
		toIdx,
	}
}

func SetPiece(square int, piece *chess.Bitboard) {
	if square < 0 || square >= 64 {
		return
	}

	*piece |= chess.Bitboard(1) << square
}

func MakeMove(from, to int) bool {
	if from < 0 || from >= 64 || to < 0 || to >= 64 {
		return false
	}

	if GetPieceAt(from).Type == chess.Empty || !slices.Contains(LegalMoves(from), to) {
		return false
	}
	applyMove(from, to)
	return true
}

func getPieceBoard(piece chess.Piece) *chess.Bitboard {
	switch piece.Type {
	case chess.WhiteKing:
		return &chess.BOARD.WhiteKing
	case chess.WhiteQueen:
		return &chess.BOARD.WhiteQueens
	case chess.WhiteRook:
		return &chess.BOARD.WhiteRooks
	case chess.WhiteKnight:
		return &chess.BOARD.WhiteKnights
	case chess.WhiteBishop:
		return &chess.BOARD.WhiteBishops
	case chess.WhitePawn:
		return &chess.BOARD.WhitePawns

	case chess.BlackKing:
		return &chess.BOARD.BlackKing
	case chess.BlackQueen:
		return &chess.BOARD.BlackQueens
	case chess.BlackRook:
		return &chess.BOARD.BlackRooks
	case chess.BlackKnight:
		return &chess.BOARD.BlackKnights
	case chess.BlackBishop:
		return &chess.BOARD.BlackBishops
	case chess.BlackPawn:
		return &chess.BOARD.BlackPawns
	}

	return nil
}

func ClearSquare(b *chess.PieceBoards, square int) {
	if square < 0 || square >= 64 {
		return
	}

	mask := ^(chess.Bitboard(1) << square)

	b.WhitePawns &= mask
	b.WhiteKnights &= mask
	b.WhiteBishops &= mask
	b.WhiteRooks &= mask
	b.WhiteQueens &= mask
	b.WhiteKing &= mask

	b.BlackPawns &= mask
	b.BlackKnights &= mask
	b.BlackBishops &= mask
	b.BlackRooks &= mask
	b.BlackQueens &= mask
	b.BlackKing &= mask
}

func IsSquareOccupied(b *chess.PieceBoards, square int) bool {
	if square < 0 || square >= 64 {
		return false
	}
	return b.Occupied()&(chess.Bitboard(1)<<square) != 0
}

func IsSquareAttacked(square int, by chess.Color, b chess.PieceBoards) bool {
	if square < 0 || square >= 64 {
		return false
	}
	rank := square / 8
	file := square % 8

	at := func(r, f int) (chess.Piece, bool) {
		if r < 0 || r >= 8 || f < 0 || f >= 8 {
			return chess.Piece{}, false
		}
		p := PieceAtBoard(b, r*8+f)
		if p.Type == chess.Empty || p.Color != by {
			return chess.Piece{}, false
		}
		return p, true
	}

	pawn, knight, king := chess.BlackPawn, chess.BlackKnight, chess.BlackKing
	rook, bishop, queen := chess.BlackRook, chess.BlackBishop, chess.BlackQueen
	pawnRank := rank + 1
	if by == chess.White {
		pawn, knight, king = chess.WhitePawn, chess.WhiteKnight, chess.WhiteKing
		rook, bishop, queen = chess.WhiteRook, chess.WhiteBishop, chess.WhiteQueen
		pawnRank = rank - 1
	}

	for _, df := range []int{-1, 1} {
		if p, ok := at(pawnRank, file+df); ok && p.Type == pawn {
			return true
		}
	}
	for _, d := range GetKnightsDirections() {
		if p, ok := at(rank+d[0], file+d[1]); ok && p.Type == knight {
			return true
		}
	}
	for _, d := range GetKingsDirections() {
		if p, ok := at(rank+d[0], file+d[1]); ok && p.Type == king {
			return true
		}
	}

	ray := func(dirs Directions, slider chess.PieceType) bool {
		for _, d := range dirs {
			r, f := rank+d[0], file+d[1]
			for r >= 0 && r < 8 && f >= 0 && f < 8 {
				p := PieceAtBoard(b, r*8+f)
				if p.Type != chess.Empty {
					if p.Color == by && (p.Type == slider || p.Type == queen) {
						return true
					}
					break
				}
				r += d[0]
				f += d[1]
			}
		}
		return false
	}
	return ray(GetRooksDirections(), rook) || ray(GetBishopsDirections(), bishop)
}

func CheckTurn() chess.Color {
	if chess.NumberOfMoves%2 != 0 {
		return chess.White
	} else {
		return chess.Black
	}
}

func ListenCheck() int {
	king, enemy := chess.BOARD.BlackKing, chess.White
	turn := CheckTurn()
	if turn == chess.White {
		king, enemy = chess.BOARD.WhiteKing, chess.Black
	}
	if king == 0 {
		return -1
	}
	sq := bits.TrailingZeros64(uint64(king))
	if IsSquareAttacked(sq, enemy, chess.BOARD) {
		return sq
	}
	return -1
}
