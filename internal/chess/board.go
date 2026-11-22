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

func (b *PieceBoards) PieceAt(square int) Piece {
	mask := Bitboard(1) << square
	var piece string
	switch {
	case b.WhiteKing&mask != 0:
		piece = "K"
	case b.BlackKing&mask != 0:
		piece = "k"
	case b.WhiteQueens&mask != 0:
		piece = "Q"
	case b.BlackQueens&mask != 0:
		piece = "q"
	case b.WhiteKnights&mask != 0:
		piece = "N"
	case b.BlackKnights&mask != 0:
		piece = "n"
	case b.WhiteBishops&mask != 0:
		piece = "B"
	case b.BlackBishops&mask != 0:
		piece = "b"
	case b.WhiteRooks&mask != 0:
		piece = "R"
	case b.BlackRooks&mask != 0:
		piece = "r"
	case b.WhitePawns&mask != 0:
		piece = "P"
	case b.BlackPawns&mask != 0:
		piece = "p"
	default:
		piece = "."
	}

	pieceType := PieceTypeMap[piece]

	return Piece{
		Face:  piece,
		Type:  pieceType,
		Color: GetColor(pieceType),
	}
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
	rank := square / 8
	file := square % 8

	at := func(r, f int) (Piece, bool) {
		if r < 0 || r >= 8 || f < 0 || f >= 8 {
			return Piece{}, false
		}
		p := b.PieceAt(r*8 + f)
		if p.Type == Empty || p.Color != by {
			return Piece{}, false
		}
		return p, true
	}

	pawn, knight, king := BlackPawn, BlackKnight, BlackKing
	rook, bishop, queen := BlackRook, BlackBishop, BlackQueen
	pawnRank := rank + 1
	if by == White {
		pawn, knight, king = WhitePawn, WhiteKnight, WhiteKing
		rook, bishop, queen = WhiteRook, WhiteBishop, WhiteQueen
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
	for _, d := range GetAllDirections() {
		if p, ok := at(rank+d[0], file+d[1]); ok && p.Type == king {
			return true
		}
	}

	ray := func(dirs Directions, slider PieceType) bool {
		for _, d := range dirs {
			r, f := rank+d[0], file+d[1]
			for r >= 0 && r < 8 && f >= 0 && f < 8 {
				p := b.PieceAt(r*8 + f)
				if p.Type != Empty {
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
