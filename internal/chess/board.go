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

var (
	knightAttackBB [64]Bitboard
	kingAttackBB   [64]Bitboard
	pawnAttackBB   [2][64]Bitboard
)

func init() {
	knightJumps := [8][2]int{{2, 1}, {2, -1}, {-2, 1}, {-2, -1}, {1, 2}, {1, -2}, {-1, 2}, {-1, -2}}
	kingSteps := [8][2]int{{-1, 1}, {0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}, {-1, 0}}

	for sq := 0; sq < 64; sq++ {
		r, f := sq/8, sq%8
		for _, d := range knightJumps {
			nr, nf := r+d[0], f+d[1]
			if nr >= 0 && nr < 8 && nf >= 0 && nf < 8 {
				knightAttackBB[sq] |= Bitboard(1) << uint(nr*8+nf)
			}
		}
		for _, d := range kingSteps {
			nr, nf := r+d[0], f+d[1]
			if nr >= 0 && nr < 8 && nf >= 0 && nf < 8 {
				kingAttackBB[sq] |= Bitboard(1) << uint(nr*8+nf)
			}
		}
		for _, df := range [2]int{-1, 1} {
			nf := f + df
			if nf < 0 || nf >= 8 {
				continue
			}
			if r > 0 {
				pawnAttackBB[White][sq] |= Bitboard(1) << uint((r-1)*8+nf)
			}
			if r < 7 {
				pawnAttackBB[Black][sq] |= Bitboard(1) << uint((r+1)*8+nf)
			}
		}
	}
}

func (b *PieceBoards) IsSquareAttacked(square int, by Color) bool {
	if square < 0 || square >= 64 {
		return false
	}

	var pawns, knights, bishops, rooks, queens, king Bitboard
	if by == White {
		pawns, knights, bishops, rooks, queens, king = b.WhitePawns, b.WhiteKnights, b.WhiteBishops, b.WhiteRooks, b.WhiteQueens, b.WhiteKing
	} else {
		pawns, knights, bishops, rooks, queens, king = b.BlackPawns, b.BlackKnights, b.BlackBishops, b.BlackRooks, b.BlackQueens, b.BlackKing
	}

	if pawnAttackBB[by][square]&pawns != 0 {
		return true
	}
	if knightAttackBB[square]&knights != 0 {
		return true
	}
	if kingAttackBB[square]&king != 0 {
		return true
	}

	occ := b.Occupied()
	rank, file := square/8, square%8

	if rq := rooks | queens; rq != 0 {
		for _, d := range rookDirs {
			r, f := rank+d[0], file+d[1]
			for r >= 0 && r < 8 && f >= 0 && f < 8 {
				bit := Bitboard(1) << uint(r*8+f)
				if occ&bit != 0 {
					if rq&bit != 0 {
						return true
					}
					break
				}
				r += d[0]
				f += d[1]
			}
		}
	}

	if bq := bishops | queens; bq != 0 {
		for _, d := range bishopDirs {
			r, f := rank+d[0], file+d[1]
			for r >= 0 && r < 8 && f >= 0 && f < 8 {
				bit := Bitboard(1) << uint(r*8+f)
				if occ&bit != 0 {
					if bq&bit != 0 {
						return true
					}
					break
				}
				r += d[0]
				f += d[1]
			}
		}
	}

	return false
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
