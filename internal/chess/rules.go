package chess

import (
	"fmt"
	"math/bits"
)

func (p *Position) HasLegalMove(c Color) bool {
	for sq := 0; sq < 64; sq++ {
		pc := p.Board.PieceAt(sq)
		if pc.Type != Empty && pc.Color == c && len(p.LegalMoves(sq)) > 0 {
			return true
		}
	}
	return false
}

func (p *Position) Status(c Color) string {
	if !p.HasLegalMove(c) {
		if p.InCheck(c) {
			return "checkmate"
		}
		return "stalemate"
	}
	b := p.Board
	if b.WhitePawns|b.BlackPawns|b.WhiteRooks|b.BlackRooks|b.WhiteQueens|b.BlackQueens == 0 &&
		bits.OnesCount64(uint64(b.WhiteKnights|b.BlackKnights|b.WhiteBishops|b.BlackBishops)) <= 1 {
		return "insufficient material"
	}
	if p.HalfmoveClock >= 100 {
		return "fifty-move rule"
	}
	return ""
}

func (p *Position) Key() string {
	return fmt.Sprintf("%v|%d|%d|%d", p.Board, p.SideToMove, p.CastlingRights, p.EnPassant)
}
