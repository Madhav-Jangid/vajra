package engine

import (
	"math/bits"
	"sync"

	"vajra/internal/chess"
)

const (
	kpkInvalid uint8 = 0
	kpkUnknown uint8 = 1
	kpkDraw    uint8 = 2
	kpkWin     uint8 = 4

	kpkSize = 2 * 24 * 64 * 64

	kpkBaseScore    = 800
	kpkAdvanceScore = 20
)

var (
	kpkOnce    sync.Once
	kpkResults [kpkSize]uint8
)

func kpkIndex(weakToMove, weakKing, strongKing, pawn int) int {
	return strongKing | weakKing<<6 | weakToMove<<12 | (pawn%8)<<13 | (6-pawn/8)<<15
}

func squareDistance(a, b int) int {
	df, dr := a%8-b%8, a/8-b/8
	if df < 0 {
		df = -df
	}

	if dr < 0 {
		dr = -dr
	}

	return max(df, dr)
}

func kingNeighbours(sq int) uint64 {
	return kingRing[sq] &^ (1 << uint(sq))
}

func whitePawnAttacks(sq int) uint64 {
	var attacks uint64

	if sq%8 > 0 {
		attacks |= 1 << uint(sq+7)
	}

	if sq%8 < 7 {
		attacks |= 1 << uint(sq+9)
	}

	return attacks
}

func kpkDecode(idx int) (weakToMove, weakKing, strongKing, pawn int) {
	strongKing = idx & 63
	weakKing = (idx >> 6) & 63
	weakToMove = (idx >> 12) & 1
	pawn = (6-(idx>>15)&7)*8 + (idx>>13)&3

	return
}

func kpkInitial(idx int) uint8 {
	weakToMove, bk, wk, pawn := kpkDecode(idx)
	rank := pawn / 8
	promotion := pawn + 8

	switch {
	case squareDistance(wk, bk) <= 1 || wk == pawn || bk == pawn ||
		(weakToMove == 0 && whitePawnAttacks(pawn)&(1<<uint(bk)) != 0):
		return kpkInvalid
	case weakToMove == 0 && rank == 6 && wk != promotion &&
		(squareDistance(bk, promotion) > 1 || squareDistance(wk, promotion) == 1):
		return kpkWin
	case weakToMove == 1:
		weakMoves := kingNeighbours(bk) &^ (kingNeighbours(wk) | whitePawnAttacks(pawn))

		if weakMoves == 0 || kingNeighbours(bk)&(1<<uint(pawn))&^kingNeighbours(wk) != 0 {
			return kpkDraw
		}
	}

	return kpkUnknown
}

func kpkClassify(idx int) uint8 {
	weakToMove, bk, wk, pawn := kpkDecode(idx)

	good, bad := kpkDraw, kpkWin

	if weakToMove == 0 {
		good, bad = kpkWin, kpkDraw
	}

	reply := kpkInvalid

	if weakToMove == 0 {
		for b := kingNeighbours(wk); b != 0; b &= b - 1 {
			reply |= kpkResults[kpkIndex(1, bk, bits.TrailingZeros64(b), pawn)]
		}

		if pawn/8 < 6 {
			reply |= kpkResults[kpkIndex(1, bk, wk, pawn+8)]
		}

		if pawn/8 == 1 && pawn+8 != wk && pawn+8 != bk {
			reply |= kpkResults[kpkIndex(1, bk, wk, pawn+16)]
		}
	} else {
		for b := kingNeighbours(bk); b != 0; b &= b - 1 {
			reply |= kpkResults[kpkIndex(0, bits.TrailingZeros64(b), wk, pawn)]
		}
	}

	switch {
	case reply&good != 0:
		return good
	case reply&kpkUnknown != 0:
		return kpkUnknown
	}

	return bad
}

func kpkBuild() {
	for idx := range kpkResults {
		kpkResults[idx] = kpkInitial(idx)
	}

	for changed := true; changed; {
		changed = false

		for idx := range kpkResults {
			if kpkResults[idx] != kpkUnknown {
				continue
			}

			if result := kpkClassify(idx); result != kpkUnknown {
				kpkResults[idx] = result
				changed = true
			}
		}
	}
}

func kpkLookup(p *chess.Position) (strongWhite, win, ok bool) {
	b := &p.Board

	if b.WhiteKnights|b.BlackKnights|b.WhiteBishops|b.BlackBishops|b.WhiteRooks|b.BlackRooks|b.WhiteQueens|b.BlackQueens != 0 {
		return false, false, false
	}

	white, black := bits.OnesCount64(uint64(b.WhitePawns)), bits.OnesCount64(uint64(b.BlackPawns))
	if white+black != 1 || b.WhiteKing == 0 || b.BlackKing == 0 {
		return false, false, false
	}

	kpkOnce.Do(kpkBuild)

	strongWhite = white == 1

	whiteKing, blackKing := bits.TrailingZeros64(uint64(b.WhiteKing)), bits.TrailingZeros64(uint64(b.BlackKing))

	var strongKing, weakKing, pawn int

	if strongWhite {
		strongKing, weakKing, pawn = whiteKing, blackKing, bits.TrailingZeros64(uint64(b.WhitePawns))
	} else {
		strongKing, weakKing, pawn = blackKing^56, whiteKing^56, bits.TrailingZeros64(uint64(b.BlackPawns))^56
	}

	if pawn%8 >= 4 {
		strongKing, weakKing, pawn = strongKing^7, weakKing^7, pawn^7
	}

	weakToMove := 0

	if (p.SideToMove == chess.White) != strongWhite {
		weakToMove = 1
	}

	return strongWhite, kpkResults[kpkIndex(weakToMove, weakKing, strongKing, pawn)] == kpkWin, true
}

func kpkScore(p *chess.Position) (int, bool) {
	strongWhite, win, ok := kpkLookup(p)
	if !ok {
		return 0, false
	}

	if !win {
		return 0, true
	}

	pawns := p.Board.WhitePawns
	advance := bits.TrailingZeros64(uint64(pawns)) / 8

	if !strongWhite {
		advance = 7 - bits.TrailingZeros64(uint64(p.Board.BlackPawns))/8
	}

	score := kpkBaseScore + kpkAdvanceScore*advance

	if !strongWhite {
		return -score, true
	}

	return score, true
}
