package chess

import "math/bits"

const (
	dirN = iota
	dirNE
	dirE
	dirSE
	dirS
	dirSW
	dirW
	dirNW
)

var (
	knightAttacks [64]Bitboard
	kingAttacks   [64]Bitboard
	pawnAttackers [2][64]Bitboard
	rays          [8][64]Bitboard
)

var rayStep = [8][2]int{
	dirN:  {0, 1},
	dirNE: {1, 1},
	dirE:  {1, 0},
	dirSE: {1, -1},
	dirS:  {0, -1},
	dirSW: {-1, -1},
	dirW:  {-1, 0},
	dirNW: {-1, 1},
}

func init() {
	jumps := [8][2]int{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}}

	for sq := 0; sq < 64; sq++ {
		file, rank := sq%8, sq/8

		for _, j := range jumps {
			if f, r := file+j[0], rank+j[1]; f >= 0 && f < 8 && r >= 0 && r < 8 {
				knightAttacks[sq] |= 1 << uint(r*8+f)
			}
		}

		for df := -1; df <= 1; df++ {
			for dr := -1; dr <= 1; dr++ {
				if f, r := file+df, rank+dr; (df != 0 || dr != 0) && f >= 0 && f < 8 && r >= 0 && r < 8 {
					kingAttacks[sq] |= 1 << uint(r*8+f)
				}
			}
		}

		for _, df := range []int{-1, 1} {
			if f := file + df; f >= 0 && f < 8 {
				if rank > 0 {
					pawnAttackers[White][sq] |= 1 << uint((rank-1)*8+f)
				}

				if rank < 7 {
					pawnAttackers[Black][sq] |= 1 << uint((rank+1)*8+f)
				}
			}
		}

		for dir, step := range rayStep {
			for f, r := file+step[0], rank+step[1]; f >= 0 && f < 8 && r >= 0 && r < 8; f, r = f+step[0], r+step[1] {
				rays[dir][sq] |= 1 << uint(r*8+f)
			}
		}
	}
}

func rayAttacks(sq int, occ Bitboard, dir int) Bitboard {
	attacks := rays[dir][sq]

	blockers := attacks & occ
	if blockers == 0 {
		return attacks
	}

	var first int

	if dir == dirN || dir == dirNE || dir == dirE || dir == dirNW {
		first = bits.TrailingZeros64(uint64(blockers))
	} else {
		first = 63 - bits.LeadingZeros64(uint64(blockers))
	}

	return attacks ^ rays[dir][first]
}

func bishopAttacks(sq int, occ Bitboard) Bitboard {
	return rayAttacks(sq, occ, dirNE) | rayAttacks(sq, occ, dirSE) | rayAttacks(sq, occ, dirSW) | rayAttacks(sq, occ, dirNW)
}

func rookAttacks(sq int, occ Bitboard) Bitboard {
	return rayAttacks(sq, occ, dirN) | rayAttacks(sq, occ, dirE) | rayAttacks(sq, occ, dirS) | rayAttacks(sq, occ, dirW)
}
