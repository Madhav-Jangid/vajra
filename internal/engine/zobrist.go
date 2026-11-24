package engine

import (
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

	for sq := 0; sq < 64; sq++ {
		piece := p.Board.PieceAt(sq)

		if piece.Type == chess.Empty {
			continue
		}

		hash ^= zobristPiece[int(piece.Type)][sq]
	}

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
