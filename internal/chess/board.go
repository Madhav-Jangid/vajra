package chess

type Bitboard uint64

type PieceBoards struct {
	WhitePawns   Bitboard
	WhiteKnights Bitboard
	WhiteBishops Bitboard
	WhiteRooks   Bitboard
	WhiteQueens  Bitboard
	WhiteKing    Bitboard

	BlackPawns   Bitboard
	BlackKnights Bitboard
	BlackBishops Bitboard
	BlackRooks   Bitboard
	BlackQueens  Bitboard
	BlackKing    Bitboard
}

func (p *PieceBoards) Occupied() Bitboard {
	return p.WhitePawns |
		p.WhiteKnights |
		p.WhiteBishops |
		p.WhiteRooks |
		p.WhiteQueens |
		p.WhiteKing |
		p.BlackPawns |
		p.BlackKnights |
		p.BlackBishops |
		p.BlackRooks |
		p.BlackQueens |
		p.BlackKing
}

type PieceType int
type Color int

const (
	Empty PieceType = iota
	WhiteKing
	BlackKing
	WhiteQueen
	BlackQueen
	WhiteRook
	BlackRook
	WhiteKnight
	BlackKnight
	WhiteBishop
	BlackBishop
	WhitePawn
	BlackPawn
)

const (
	Black Color = iota
	White
)

type Piece struct {
	Face  string
	Type  PieceType
	Color Color
}

var PieceTypeMap = map[string]PieceType{
	".": Empty,
	"K": WhiteKing,
	"k": BlackKing,
	"Q": WhiteQueen,
	"q": BlackQueen,
	"R": WhiteRook,
	"r": BlackRook,
	"N": WhiteKnight,
	"n": BlackKnight,
	"B": WhiteBishop,
	"b": BlackBishop,
	"P": WhitePawn,
	"p": BlackPawn,
}

func (b *PieceBoards) ClearSquare(square int) {
	if square < 0 || square >= 64 {
		return
	}

	mask := ^(Bitboard(1) << square)

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

var pieceFaces = [13]string{".", "K", "k", "Q", "q", "R", "r", "N", "n", "B", "b", "P", "p"}

func (b *PieceBoards) PieceAt(square int) Piece {
	mask := Bitboard(1) << square
	pieceType := Empty

	switch {
	case b.WhiteKing&mask != 0:
		pieceType = WhiteKing
	case b.BlackKing&mask != 0:
		pieceType = BlackKing
	case b.WhiteQueens&mask != 0:
		pieceType = WhiteQueen
	case b.BlackQueens&mask != 0:
		pieceType = BlackQueen
	case b.WhiteKnights&mask != 0:
		pieceType = WhiteKnight
	case b.BlackKnights&mask != 0:
		pieceType = BlackKnight
	case b.WhiteBishops&mask != 0:
		pieceType = WhiteBishop
	case b.BlackBishops&mask != 0:
		pieceType = BlackBishop
	case b.WhiteRooks&mask != 0:
		pieceType = WhiteRook
	case b.BlackRooks&mask != 0:
		pieceType = BlackRook
	case b.WhitePawns&mask != 0:
		pieceType = WhitePawn
	case b.BlackPawns&mask != 0:
		pieceType = BlackPawn
	}

	return Piece{
		Face:  pieceFaces[pieceType],
		Type:  pieceType,
		Color: GetColor(pieceType),
	}
}

func (b *PieceBoards) Side(color Color) Bitboard {
	if color == White {
		return b.WhitePawns | b.WhiteKnights | b.WhiteBishops | b.WhiteRooks | b.WhiteQueens | b.WhiteKing
	}

	return b.BlackPawns | b.BlackKnights | b.BlackBishops | b.BlackRooks | b.BlackQueens | b.BlackKing
}

func GetColor(pieceType PieceType) Color {
	switch pieceType {
	case WhiteKing,
		WhiteQueen,
		WhiteRook,
		WhiteKnight,
		WhiteBishop,
		WhitePawn:
		return White

	default:
		return Black
	}
}

func (b *PieceBoards) IsSquareOccupied(square int) bool {
	if square < 0 || square >= 64 {
		return false
	}
	return b.Occupied()&(Bitboard(1)<<square) != 0
}

func (b *PieceBoards) IsSquareAttacked(square int, by Color) bool {
	if square < 0 || square >= 64 {
		return false
	}

	occ := b.Occupied()

	if by == White {
		return pawnAttackers[White][square]&b.WhitePawns != 0 ||
			knightAttacks[square]&b.WhiteKnights != 0 ||
			kingAttacks[square]&b.WhiteKing != 0 ||
			bishopAttacks(square, occ)&(b.WhiteBishops|b.WhiteQueens) != 0 ||
			rookAttacks(square, occ)&(b.WhiteRooks|b.WhiteQueens) != 0
	}

	return pawnAttackers[Black][square]&b.BlackPawns != 0 ||
		knightAttacks[square]&b.BlackKnights != 0 ||
		kingAttacks[square]&b.BlackKing != 0 ||
		bishopAttacks(square, occ)&(b.BlackBishops|b.BlackQueens) != 0 ||
		rookAttacks(square, occ)&(b.BlackRooks|b.BlackQueens) != 0
}

func (b *PieceBoards) ApplyMove(move Move) {
	from := move.From
	to := move.To

	piece := b.PieceAt(from)
	target := b.PieceAt(to)

	if piece.Type == Empty {
		return
	}

	white := piece.Color == White
	isPawn := piece.Type == WhitePawn || piece.Type == BlackPawn
	isKing := piece.Type == WhiteKing || piece.Type == BlackKing

	if isPawn && move.Flags&EnPassant != 0 && target.Type == Empty {
		if white {
			b.ClearSquare(to - 8)
		} else {
			b.ClearSquare(to + 8)
		}
	}

	b.ClearSquare(from)
	b.ClearSquare(to)

	placed := piece.Type

	if isPawn && (to/8 == 7 || to/8 == 0) {
		switch {
		case move.Promotion != Empty:
			placed = move.Promotion
		case white:
			placed = WhiteQueen
		default:
			placed = BlackQueen
		}
	}

	b.SetPiece(placed, to)

	if isKing && move.Flags&(KingCastle|QueenCastle) != 0 {
		rookFrom := from + 3
		rookTo := from + 1

		if to < from {
			rookFrom = from - 4
			rookTo = from - 1
		}

		rook := WhiteRook
		if !white {
			rook = BlackRook
		}

		b.ClearSquare(rookFrom)
		b.SetPiece(rook, rookTo)
	}
}
