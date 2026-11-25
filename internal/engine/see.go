package engine

import (
	"math/bits"

	"vajra/internal/chess"
)

func seePieceValue(pt chess.PieceType) int {
	switch pt {
	case chess.WhitePawn, chess.BlackPawn:
		return PawnValue
	case chess.WhiteKnight, chess.BlackKnight:
		return KnightValue
	case chess.WhiteBishop, chess.BlackBishop:
		return BishopValue
	case chess.WhiteRook, chess.BlackRook:
		return RookValue
	case chess.WhiteQueen, chess.BlackQueen:
		return QueenValue
	case chess.WhiteKing, chess.BlackKing:
		return KingValue
	default:
		return 0
	}
}

func pawnAttackersTo(sq int, pawns uint64, color chess.Color) uint64 {
	var mask uint64
	r, f := sq/8, sq%8
	srcRank := r + 1
	if color == chess.White {
		srcRank = r - 1
	}
	if srcRank < 0 || srcRank > 7 {
		return 0
	}
	if f > 0 {
		mask |= uint64(1) << uint(srcRank*8+f-1)
	}
	if f < 7 {
		mask |= uint64(1) << uint(srcRank*8+f+1)
	}
	return pawns & mask
}

func attackersToSquare(b *chess.PieceBoards, sq int, occ uint64) uint64 {
	var atk uint64
	atk |= pawnAttackersTo(sq, uint64(b.WhitePawns), chess.White)
	atk |= pawnAttackersTo(sq, uint64(b.BlackPawns), chess.Black)
	atk |= knightAttackBB(sq) & uint64(b.WhiteKnights|b.BlackKnights)
	atk |= kingRingOnly(sq) & uint64(b.WhiteKing|b.BlackKing)

	diag := slideAttacks(sq, occ, bishopDirs)
	ortho := slideAttacks(sq, occ, rookDirs)
	atk |= diag & uint64(b.WhiteBishops|b.BlackBishops|b.WhiteQueens|b.BlackQueens)
	atk |= ortho & uint64(b.WhiteRooks|b.BlackRooks|b.WhiteQueens|b.BlackQueens)
	return atk
}

func knightAttackBB(sq int) uint64 {
	return knightAttacks[sq]
}

func kingRingOnly(sq int) uint64 {
	return kingRing[sq] &^ (uint64(1) << uint(sq))
}

func leastValuableAttacker(b *chess.PieceBoards, atk uint64, color chess.Color) (uint64, int) {
	if color == chess.White {
		if bb := atk & uint64(b.WhitePawns); bb != 0 {
			return bb & -bb, PawnValue
		}
		if bb := atk & uint64(b.WhiteKnights); bb != 0 {
			return bb & -bb, KnightValue
		}
		if bb := atk & uint64(b.WhiteBishops); bb != 0 {
			return bb & -bb, BishopValue
		}
		if bb := atk & uint64(b.WhiteRooks); bb != 0 {
			return bb & -bb, RookValue
		}
		if bb := atk & uint64(b.WhiteQueens); bb != 0 {
			return bb & -bb, QueenValue
		}
		if bb := atk & uint64(b.WhiteKing); bb != 0 {
			return bb & -bb, KingValue
		}
		return 0, 0
	}
	if bb := atk & uint64(b.BlackPawns); bb != 0 {
		return bb & -bb, PawnValue
	}
	if bb := atk & uint64(b.BlackKnights); bb != 0 {
		return bb & -bb, KnightValue
	}
	if bb := atk & uint64(b.BlackBishops); bb != 0 {
		return bb & -bb, BishopValue
	}
	if bb := atk & uint64(b.BlackRooks); bb != 0 {
		return bb & -bb, RookValue
	}
	if bb := atk & uint64(b.BlackQueens); bb != 0 {
		return bb & -bb, QueenValue
	}
	if bb := atk & uint64(b.BlackKing); bb != 0 {
		return bb & -bb, KingValue
	}
	return 0, 0
}

func see(p *chess.Position, m chess.Move) int {
	if m.Flags&chess.Capture == 0 && m.Flags&chess.Promotion == 0 {
		return 0
	}

	b := &p.Board
	occ := uint64(b.Occupied())
	fromBit := uint64(1) << uint(m.From)
	occ &^= fromBit

	if m.Flags&chess.EnPassant != 0 {
		capSq := m.To + 8
		if p.SideToMove == chess.White {
			capSq = m.To - 8
		}
		occ &^= uint64(1) << uint(capSq)
	}

	var gain [32]int
	d := 0
	gain[0] = seePieceValue(m.Captured)
	attackerVal := seePieceValue(m.Piece)
	if m.Flags&chess.Promotion != 0 {
		promoVal := seePieceValue(m.Promotion)
		gain[0] += promoVal - PawnValue
		attackerVal = promoVal
	}

	side := chess.Black
	if p.SideToMove == chess.Black {
		side = chess.White
	}

	diagSliders := uint64(b.WhiteBishops | b.BlackBishops | b.WhiteQueens | b.BlackQueens)
	orthoSliders := uint64(b.WhiteRooks | b.BlackRooks | b.WhiteQueens | b.BlackQueens)

	atk := attackersToSquare(b, m.To, occ) & occ

	for d < 31 {
		lvaBit, lvaVal := leastValuableAttacker(b, atk, side)
		if lvaBit == 0 {
			break
		}

		d++
		gain[d] = attackerVal - gain[d-1]
		if max(-gain[d-1], gain[d]) < 0 {
			break
		}

		attackerVal = lvaVal
		occ &^= lvaBit

		if lvaVal == PawnValue || lvaVal == BishopValue || lvaVal == QueenValue {
			atk |= slideAttacks(m.To, occ, bishopDirs) & diagSliders
		}
		if lvaVal == RookValue || lvaVal == QueenValue {
			atk |= slideAttacks(m.To, occ, rookDirs) & orthoSliders
		}
		atk &= occ

		if side == chess.White {
			side = chess.Black
		} else {
			side = chess.White
		}
	}

	for d > 0 {
		d--
		if -gain[d+1] < gain[d] {
			gain[d] = -gain[d+1]
		}
	}

	return gain[0]
}

func nonPawnPieceCount(p *chess.Position, c chess.Color) int {
	b := &p.Board
	if c == chess.White {
		return bits.OnesCount64(uint64(b.WhiteKnights | b.WhiteBishops | b.WhiteRooks | b.WhiteQueens))
	}
	return bits.OnesCount64(uint64(b.BlackKnights | b.BlackBishops | b.BlackRooks | b.BlackQueens))
}

func kingCornerMopUp(b *chess.PieceBoards, phase, wMat, kMat int) int {
	if phase > 6 {
		return 0
	}
	diff := wMat - kMat
	if diff >= RookValue && (b.BlackPawns|b.BlackKnights|b.BlackBishops|b.BlackRooks|b.BlackQueens) == 0 {
		wk := kingSquareOf(b.WhiteKing)
		bk := kingSquareOf(b.BlackKing)
		if wk < 0 || bk < 0 {
			return 0
		}
		return mopUpBonus(wk, bk, maxPhase-phase)
	}
	if -diff >= RookValue && (b.WhitePawns|b.WhiteKnights|b.WhiteBishops|b.WhiteRooks|b.WhiteQueens) == 0 {
		wk := kingSquareOf(b.WhiteKing)
		bk := kingSquareOf(b.BlackKing)
		if wk < 0 || bk < 0 {
			return 0
		}
		return -mopUpBonus(bk, wk, maxPhase-phase)
	}
	return 0
}

func mopUpBonus(winKing, loseKing, egWeight int) int {
	lf, lr := loseKing%8, loseKing/8
	wf, wr := winKing%8, winKing/8
	fd := lf - 3
	if lf < 4 {
		fd = 3 - lf
	} else {
		fd = lf - 4
	}
	rd := lr - 3
	if lr < 4 {
		rd = 3 - lr
	} else {
		rd = lr - 4
	}
	centerDist := fd + rd

	df := wf - lf
	if df < 0 {
		df = -df
	}
	dr := wr - lr
	if dr < 0 {
		dr = -dr
	}
	kingProximity := 14 - (df + dr)

	return ((10*centerDist + 4*kingProximity) * egWeight) / maxPhase
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
