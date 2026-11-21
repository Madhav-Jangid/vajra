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

	Empty Bitboard
}

var BOARD PieceBoards

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

var HasCheck bool
var NumberOfMoves int

func InitializeBitBoard() {
	BOARD = PieceBoards{}

	BOARD.WhitePawns = 0x000000000000FF00
	BOARD.WhiteRooks = Bitboard(1)<<0 | Bitboard(1)<<7
	BOARD.WhiteKnights = Bitboard(1)<<1 | Bitboard(1)<<6
	BOARD.WhiteBishops = Bitboard(1)<<2 | Bitboard(1)<<5
	BOARD.WhiteQueens = Bitboard(1) << 3
	BOARD.WhiteKing = Bitboard(1) << 4

	BOARD.BlackPawns = 0x00FF000000000000
	BOARD.BlackRooks = Bitboard(1)<<56 | Bitboard(1)<<63
	BOARD.BlackKnights = Bitboard(1)<<57 | Bitboard(1)<<62
	BOARD.BlackBishops = Bitboard(1)<<58 | Bitboard(1)<<61
	BOARD.BlackQueens = Bitboard(1) << 59
	BOARD.BlackKing = Bitboard(1) << 60

	BOARD.Empty = 0

	HasCheck = false
	NumberOfMoves = 0
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
