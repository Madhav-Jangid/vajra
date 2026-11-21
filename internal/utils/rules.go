package utils

import (
	"fmt"
	"math/bits"
	"vajra/internal/chess"
)

const (
	CastleWK uint8 = 1
	CastleWQ uint8 = 2
	CastleBK uint8 = 4
	CastleBQ uint8 = 8
)

func opponent(c chess.Color) chess.Color {
	if c == chess.White {
		return chess.Black
	}
	return chess.White
}

type Snapshot struct {
	board     chess.PieceBoards
	castling  uint8
	enPassant int
	halfmove  int
}

func Save() Snapshot {
	return Snapshot{chess.BOARD, chess.Castling, chess.EnPassant, chess.Halfmove}
}

func (s Snapshot) Restore() {
	chess.BOARD, chess.Castling, chess.EnPassant, chess.Halfmove = s.board, s.castling, s.enPassant, s.halfmove
}

func castleMask(sq int) uint8 {
	switch sq {
	case 0:
		return CastleWQ
	case 7:
		return CastleWK
	case 4:
		return CastleWK | CastleWQ
	case 56:
		return CastleBQ
	case 63:
		return CastleBK
	case 60:
		return CastleBK | CastleBQ
	}
	return 0
}

func applyMove(from, to int) {
	piece := GetPieceAt(from)
	target := GetPieceAt(to)
	white := piece.Color == chess.White
	isPawn := piece.Type == chess.WhitePawn || piece.Type == chess.BlackPawn
	isKing := piece.Type == chess.WhiteKing || piece.Type == chess.BlackKing

	if isPawn && from%8 != to%8 && target.Type == chess.Empty {
		if white {
			ClearSquare(&chess.BOARD, to-8)
		} else {
			ClearSquare(&chess.BOARD, to+8)
		}
	}

	ClearSquare(&chess.BOARD, to)
	ClearSquare(&chess.BOARD, from)

	placed := piece.Type
	if isPawn && (to/8 == 7 || to/8 == 0) {
		placed = chess.BlackQueen
		if white {
			placed = chess.WhiteQueen
		}
	}
	SetPiece(to, getPieceBoard(chess.Piece{Type: placed}))

	if isKing && (to-from == 2 || from-to == 2) {
		rookFrom, rookTo := from+3, from+1
		if to < from {
			rookFrom, rookTo = from-4, from-1
		}
		rook := chess.BlackRook
		if white {
			rook = chess.WhiteRook
		}
		ClearSquare(&chess.BOARD, rookFrom)
		SetPiece(rookTo, getPieceBoard(chess.Piece{Type: rook}))
	}

	chess.EnPassant = -1
	if isPawn && (to-from == 16 || from-to == 16) {
		chess.EnPassant = (from + to) / 2
	}

	chess.Castling &^= castleMask(from) | castleMask(to)

	if isPawn || target.Type != chess.Empty {
		chess.Halfmove = 0
	} else {
		chess.Halfmove++
	}
}

func kingSquare(c chess.Color) int {
	king := chess.BOARD.BlackKing
	if c == chess.White {
		king = chess.BOARD.WhiteKing
	}
	if king == 0 {
		return -1
	}
	return bits.TrailingZeros64(uint64(king))
}

func InCheck(c chess.Color) bool {
	sq := kingSquare(c)
	return sq >= 0 && IsSquareAttacked(sq, opponent(c), chess.BOARD)
}

func LegalMoves(square int) Moves {
	piece := GetPieceAt(square)
	if piece.Type == chess.Empty {
		return nil
	}
	var legal Moves
	for _, to := range GenerateMoves(square) {
		s := Save()
		applyMove(square, to)
		if !InCheck(piece.Color) {
			legal = append(legal, to)
		}
		s.Restore()
	}
	return legal
}

func HasLegalMove(c chess.Color) bool {
	for sq := 0; sq < 64; sq++ {
		p := GetPieceAt(sq)
		if p.Type != chess.Empty && p.Color == c && len(LegalMoves(sq)) > 0 {
			return true
		}
	}
	return false
}

func Status(c chess.Color) string {
	if !HasLegalMove(c) {
		if InCheck(c) {
			return "checkmate"
		}
		return "stalemate"
	}
	b := chess.BOARD
	if b.WhitePawns|b.BlackPawns|b.WhiteRooks|b.BlackRooks|b.WhiteQueens|b.BlackQueens == 0 &&
		bits.OnesCount64(uint64(b.WhiteKnights|b.BlackKnights|b.WhiteBishops|b.BlackBishops)) <= 1 {
		return "insufficient material"
	}
	if chess.Halfmove >= 100 {
		return "fifty-move rule"
	}
	return ""
}

func PositionKey(c chess.Color) string {
	return fmt.Sprintf("%v|%d|%d|%d", chess.BOARD, c, chess.Castling, chess.EnPassant)
}
