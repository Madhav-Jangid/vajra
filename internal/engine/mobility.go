package engine

import (
	"math/bits"

	"vajra/internal/chess"
)

var (
	knightMobility = mobilityTerm{mg: 4, eg: 4, baseline: 4}
	bishopMobility = mobilityTerm{mg: 3, eg: 3, baseline: 5}
	rookMobility   = mobilityTerm{mg: 2, eg: 4, baseline: 6}
	queenMobility  = mobilityTerm{mg: 1, eg: 2, baseline: 9}
)

type mobilityTerm struct {
	mg, eg, baseline int
}

func (m mobilityTerm) score(squares, phase int) int {
	d := squares - m.baseline

	return taper(m.mg*d, m.eg*d, phase)
}

const (
	notFileA uint64 = ^fileA
	notFileH uint64 = ^(fileA << 7)
)

var (
	knightAttacks [64]uint64

	bishopDirs = [4][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	rookDirs   = [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
)

func init() {
	jumps := [8][2]int{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}}

	for sq := 0; sq < 64; sq++ {
		for _, j := range jumps {
			f, r := sq%8+j[0], sq/8+j[1]

			if f >= 0 && f < 8 && r >= 0 && r < 8 {
				knightAttacks[sq] |= 1 << uint(r*8+f)
			}
		}
	}
}

func slideAttacks(sq int, occ uint64, dirs [4][2]int) uint64 {
	var attacks uint64

	for _, d := range dirs {
		f, r := sq%8+d[0], sq/8+d[1]

		for f >= 0 && f < 8 && r >= 0 && r < 8 {
			bit := uint64(1) << uint(r*8+f)
			attacks |= bit

			if occ&bit != 0 {
				break
			}

			f, r = f+d[0], r+d[1]
		}
	}

	return attacks
}

func pawnAttacks(pawns uint64, color chess.Color) uint64 {
	if color == chess.White {
		return (pawns&notFileH)<<9 | (pawns&notFileA)<<7
	}

	return (pawns&notFileH)>>7 | (pawns&notFileA)>>9
}

const (
	pawnThreatDivisor = 4
	trappedDivisor    = 3
)

func threatPenalty(value, safeSquares int) int {
	p := (value - PawnValue) / pawnThreatDivisor

	if safeSquares == 0 {
		p += value / trappedDivisor
	}

	return p
}

func mobilityScore(
	knights, bishops, rooks, queens chess.Bitboard,
	ownOcc, occ, enemyPawn, enemyPawnOcc uint64,
	phase int,
) (mobility, threats int) {

	allowed := ^(ownOcc | enemyPawn)

	visit := func(sq int, attacks uint64, term mobilityTerm, value int) {
		n := bits.OnesCount64(attacks & allowed)
		mobility += term.score(n, phase)

		if enemyPawn&(uint64(1)<<uint(sq)) != 0 {
			escapes := bits.OnesCount64(attacks & allowed &^ enemyPawnOcc)
			threats -= threatPenalty(value, escapes)
		}
	}

	forEachSquare(knights, func(sq int) {
		visit(sq, knightAttacks[sq], knightMobility, KnightValue)
	})

	forEachSquare(bishops, func(sq int) {
		visit(sq, slideAttacks(sq, occ, bishopDirs), bishopMobility, BishopValue)
	})

	forEachSquare(rooks, func(sq int) {
		visit(sq, slideAttacks(sq, occ, rookDirs), rookMobility, RookValue)
	})

	forEachSquare(queens, func(sq int) {
		visit(sq, slideAttacks(sq, occ, rookDirs)|slideAttacks(sq, occ, bishopDirs), queenMobility, QueenValue)
	})

	return mobility, threats
}

var kingRing [64]uint64

func init() {
	for sq := 0; sq < 64; sq++ {
		for df := -1; df <= 1; df++ {
			for dr := -1; dr <= 1; dr++ {
				f, r := sq%8+df, sq/8+dr

				if f >= 0 && f < 8 && r >= 0 && r < 8 {
					kingRing[sq] |= 1 << uint(r*8+f)
				}
			}
		}
	}
}

const maxKingDanger = 400

func kingDanger(
	knights, bishops, rooks, queens chess.Bitboard,
	occ uint64,
	kingSq int,
) int {

	if kingSq < 0 || kingSq > 63 {
		return 0
	}

	ring := kingRing[kingSq]
	units, attackers := 0, 0

	hit := func(attacks uint64, weight int) {
		if attacks&ring != 0 {
			units += weight
			attackers++
		}
	}

	forEachSquare(knights, func(sq int) { hit(knightAttacks[sq], 2) })
	forEachSquare(bishops, func(sq int) { hit(slideAttacks(sq, occ, bishopDirs), 2) })
	forEachSquare(rooks, func(sq int) { hit(slideAttacks(sq, occ, rookDirs), 3) })
	forEachSquare(queens, func(sq int) {
		hit(slideAttacks(sq, occ, rookDirs)|slideAttacks(sq, occ, bishopDirs), 5)
	})

	if attackers < 2 {
		return units
	}

	return min(units*attackers*4, maxKingDanger)
}

func kingSquareOf(king chess.Bitboard) int {
	if king == 0 {
		return -1
	}

	return bits.TrailingZeros64(uint64(king))
}
