package engine

import (
	"math/bits"
	"math/rand"

	"vajra/internal/chess"
)

const (
	zobristPieceCount  = 13
	zobristSquareCount = 64
)

var (
	zobristPiece  [zobristPieceCount][zobristSquareCount]uint64
	zobristSide   uint64
	zobristCastle [16]uint64
	zobristEP     [65]uint64
)

func init() {
	r := rand.New(rand.NewSource(0x123456789ABCDEF))

	for piece := 0; piece < zobristPieceCount; piece++ {
		for square := 0; square < zobristSquareCount; square++ {
			zobristPiece[piece][square] = r.Uint64()
		}
	}

	zobristSide = r.Uint64()

	for i := range zobristCastle {
		zobristCastle[i] = r.Uint64()
	}

	for i := range zobristEP {
		zobristEP[i] = r.Uint64()
	}
}

func HashPosition(p *chess.Position) uint64 {
	var hash uint64
	b := &p.Board

	hashPieces := func(bb chess.Bitboard, pt chess.PieceType) {
		table := &zobristPiece[int(pt)]
		for x := uint64(bb); x != 0; x &= x - 1 {
			hash ^= table[bits.TrailingZeros64(x)]
		}
	}

	hashPieces(b.WhiteKing, chess.WhiteKing)
	hashPieces(b.BlackKing, chess.BlackKing)
	hashPieces(b.WhiteQueens, chess.WhiteQueen)
	hashPieces(b.BlackQueens, chess.BlackQueen)
	hashPieces(b.WhiteRooks, chess.WhiteRook)
	hashPieces(b.BlackRooks, chess.BlackRook)
	hashPieces(b.WhiteKnights, chess.WhiteKnight)
	hashPieces(b.BlackKnights, chess.BlackKnight)
	hashPieces(b.WhiteBishops, chess.WhiteBishop)
	hashPieces(b.BlackBishops, chess.BlackBishop)
	hashPieces(b.WhitePawns, chess.WhitePawn)
	hashPieces(b.BlackPawns, chess.BlackPawn)

	if p.SideToMove == chess.Black {
		hash ^= zobristSide
	}

	hash ^= zobristCastle[int(p.CastlingRights)]

	ep := 64

	if p.EnPassant >= 0 && p.EnPassant < 64 {
		ep = p.EnPassant
	}

	hash ^= zobristEP[ep]
	return hash
}
