package engine

import (
	"math/bits"

	"vajra/internal/chess"
)

const (
	PawnValue   = 100
	KnightValue = 320
	BishopValue = 330
	RookValue   = 500
	QueenValue  = 900
	KingValue   = 20000
)

type EvalBreakdown struct {
	Material      int
	PieceSquare   int
	PawnStructure int
	PassedPawns   int
	Pieces        int
	Mobility      int
	Threats       int
	KingSafety    int
}

func (e EvalBreakdown) Total() int {
	return e.Material +
		e.PieceSquare +
		e.PawnStructure +
		e.PassedPawns +
		e.Pieces +
		e.Mobility +
		e.Threats +
		e.KingSafety
}

var pawnPST = [64]int{
	0, 0, 0, 0, 0, 0, 0, 0,
	5, 10, 10, -20, -20, 10, 10, 5,
	5, -5, -10, 0, 0, -10, -5, 5,
	0, 0, 0, 20, 20, 0, 0, 0,
	5, 5, 10, 25, 25, 10, 5, 5,
	10, 10, 20, 30, 30, 20, 10, 10,
	50, 50, 50, 50, 50, 50, 50, 50,
	0, 0, 0, 0, 0, 0, 0, 0,
}

var knightPST = [64]int{
	-50, -40, -30, -30, -30, -30, -40, -50,
	-40, -20, 0, 0, 0, 0, -20, -40,
	-30, 0, 10, 15, 15, 10, 0, -30,
	-30, 5, 15, 20, 20, 15, 5, -30,
	-30, 0, 15, 20, 20, 15, 0, -30,
	-30, 5, 10, 15, 15, 10, 5, -30,
	-40, -20, 0, 5, 5, 0, -20, -40,
	-50, -40, -30, -30, -30, -30, -40, -50,
}

var bishopPST = [64]int{
	-20, -10, -10, -10, -10, -10, -10, -20,
	-10, 5, 0, 0, 0, 0, 5, -10,
	-10, 10, 10, 10, 10, 10, 10, -10,
	-10, 0, 10, 10, 10, 10, 0, -10,
	-10, 5, 5, 10, 10, 5, 5, -10,
	-10, 0, 5, 10, 10, 5, 0, -10,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-20, -10, -10, -10, -10, -10, -10, -20,
}

var rookPST = [64]int{
	0, 0, 0, 5, 5, 0, 0, 0,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	-5, 0, 0, 0, 0, 0, 0, -5,
	5, 10, 10, 10, 10, 10, 10, 5,
	0, 0, 0, 0, 0, 0, 0, 0,
}

var queenPST = [64]int{
	-20, -10, -10, -5, -5, -10, -10, -20,
	-10, 0, 5, 0, 0, 0, 0, -10,
	-10, 5, 5, 5, 5, 5, 0, -10,
	0, 0, 5, 5, 5, 5, 0, -5,
	-5, 0, 5, 5, 5, 5, 0, -5,
	-10, 0, 5, 5, 5, 5, 0, -10,
	-10, 0, 0, 0, 0, 0, 0, -10,
	-20, -10, -10, -5, -5, -10, -10, -20,
}

var kingMGPST = [64]int{
	20, 30, 10, 0, 0, 10, 30, 20,
	20, 20, 0, 0, 0, 0, 20, 20,
	-10, -20, -20, -20, -20, -20, -20, -10,
	-20, -30, -30, -40, -40, -30, -30, -20,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
	-30, -40, -40, -50, -50, -40, -40, -30,
}

var kingEGPST = [64]int{
	-50, -30, -30, -30, -30, -30, -30, -50,
	-30, -30, 0, 0, 0, 0, -30, -30,
	-30, -10, 20, 30, 30, 20, -10, -30,
	-30, -10, 30, 40, 40, 30, -10, -30,
	-30, -10, 30, 40, 40, 30, -10, -30,
	-30, -10, 20, 30, 30, 20, -10, -30,
	-30, -20, -10, 0, 0, -10, -20, -30,
	-50, -40, -30, -20, -20, -30, -40, -50,
}

const (
	maxPhase = 24

	bishopPairBonus   = 30
	rookOpenFile      = 20
	rookSemiOpenFile  = 10
	doubledPawnPenalt = 12
	isolatedPawnPen   = 12

	shieldNear    = 10
	shieldFar     = 5
	shieldMissing = -15

	exposedKingPenalty = 20
)

var (
	passedMG = [8]int{0, 5, 10, 20, 35, 60, 100, 0}
	passedEG = [8]int{0, 10, 20, 40, 70, 120, 200, 0}
)

const fileA uint64 = 0x0101010101010101

func fileMask(file int) uint64 {
	if file < 0 || file > 7 {
		return 0
	}

	return fileA << uint(file)
}

func gamePhase(b *chess.PieceBoards) int {
	phase := bits.OnesCount64(uint64(b.WhiteKnights|b.BlackKnights|b.WhiteBishops|b.BlackBishops)) +
		2*bits.OnesCount64(uint64(b.WhiteRooks|b.BlackRooks)) +
		4*bits.OnesCount64(uint64(b.WhiteQueens|b.BlackQueens))

	return min(phase, maxPhase)
}

func taper(mg, eg, phase int) int {
	return (mg*phase + eg*(maxPhase-phase)) / maxPhase
}

func forEachSquare(bb chess.Bitboard, f func(sq int)) {
	for b := uint64(bb); b != 0; b &= b - 1 {
		f(bits.TrailingZeros64(b))
	}
}

func relRank(sq int, color chess.Color) int {
	if color == chess.White {
		return sq / 8
	}

	return 7 - sq/8
}

func evalColor(
	color chess.Color,
	pawns, enemyPawns, knights, bishops, rooks, queens, king chess.Bitboard,
	phase int,
	occ, enemyPawnAttacks uint64,
) EvalBreakdown {

	var e EvalBreakdown

	count := func(bb chess.Bitboard, value int, piece chess.PieceType) {
		forEachSquare(bb, func(sq int) {
			e.Material += value
			e.PieceSquare += pieceSquareBonus(piece, color, sq)
		})
	}

	pawnType, knightType, bishopType, rookType, queenType := chess.WhitePawn, chess.WhiteKnight, chess.WhiteBishop, chess.WhiteRook, chess.WhiteQueen
	if color == chess.Black {
		pawnType, knightType, bishopType, rookType, queenType = chess.BlackPawn, chess.BlackKnight, chess.BlackBishop, chess.BlackRook, chess.BlackQueen
	}

	count(pawns, PawnValue, pawnType)
	count(knights, KnightValue, knightType)
	count(bishops, BishopValue, bishopType)
	count(rooks, RookValue, rookType)
	count(queens, QueenValue, queenType)

	forEachSquare(king, func(sq int) {
		psq := sq
		if color == chess.Black {
			psq = flipSquare(sq)
		}

		e.PieceSquare += taper(kingMGPST[psq], kingEGPST[psq], phase)
	})

	own, enemy := uint64(pawns), uint64(enemyPawns)

	for file := 0; file < 8; file++ {
		onFile := bits.OnesCount64(own & fileMask(file))
		if onFile > 1 {
			e.PawnStructure -= (onFile - 1) * doubledPawnPenalt
		}

		if onFile > 0 && own&(fileMask(file-1)|fileMask(file+1)) == 0 {
			e.PawnStructure -= onFile * isolatedPawnPen
		}
	}

	forEachSquare(pawns, func(sq int) {
		file, rank := sq%8, sq/8

		var ahead uint64
		if color == chess.White {
			if rank < 7 {
				ahead = ^uint64(0) << uint(8*(rank+1))
			}
		} else {
			ahead = (uint64(1) << uint(8*rank)) - 1
		}

		front := (fileMask(file-1) | fileMask(file) | fileMask(file+1)) & ahead

		if enemy&front == 0 {
			r := relRank(sq, color)
			e.PassedPawns += taper(passedMG[r], passedEG[r], phase)
		}
	})

	if bits.OnesCount64(uint64(bishops)) >= 2 {
		e.Pieces += bishopPairBonus
	}

	forEachSquare(rooks, func(sq int) {
		mask := fileMask(sq % 8)

		switch {
		case (own|enemy)&mask == 0:
			e.Pieces += rookOpenFile
		case own&mask == 0:
			e.Pieces += rookSemiOpenFile
		}
	})

	ownOcc := uint64(pawns | knights | bishops | rooks | queens | king)
	e.Mobility, e.Threats = mobilityScore(knights, bishops, rooks, queens, ownOcc, occ, enemyPawnAttacks, uint64(enemyPawns), phase)

	forEachSquare(king, func(ksq int) {
		kr := relRank(ksq, color)

		if kr > 1 {
			e.KingSafety -= taper(exposedKingPenalty*(kr-1), 0, phase)
			return
		}

		shield := 0

		for f := ksq%8 - 1; f <= ksq%8+1; f++ {
			if f < 0 || f > 7 {
				continue
			}

			switch {
			case pawnAt(own, f, kr+1, color):
				shield += shieldNear
			case pawnAt(own, f, kr+2, color):
				shield += shieldFar
			default:
				shield += shieldMissing
			}
		}

		e.KingSafety += taper(shield, 0, phase)
	})

	return e
}

func pawnAt(pawns uint64, file, relativeRank int, color chess.Color) bool {
	if relativeRank < 0 || relativeRank > 7 {
		return false
	}

	rank := relativeRank
	if color == chess.Black {
		rank = 7 - relativeRank
	}

	return pawns&(uint64(1)<<uint(rank*8+file)) != 0
}

func EvaluateBreakdown(p *chess.Position) EvalBreakdown {
	b := &p.Board
	phase := gamePhase(b)
	occ := uint64(b.Occupied())

	w := evalColor(chess.White, b.WhitePawns, b.BlackPawns, b.WhiteKnights, b.WhiteBishops, b.WhiteRooks, b.WhiteQueens, b.WhiteKing, phase, occ, pawnAttacks(uint64(b.BlackPawns), chess.Black))
	k := evalColor(chess.Black, b.BlackPawns, b.WhitePawns, b.BlackKnights, b.BlackBishops, b.BlackRooks, b.BlackQueens, b.BlackKing, phase, occ, pawnAttacks(uint64(b.WhitePawns), chess.White))

	dangerOnWhite := kingDanger(b.BlackKnights, b.BlackBishops, b.BlackRooks, b.BlackQueens, occ, kingSquareOf(b.WhiteKing))
	dangerOnBlack := kingDanger(b.WhiteKnights, b.WhiteBishops, b.WhiteRooks, b.WhiteQueens, occ, kingSquareOf(b.BlackKing))

	w.KingSafety -= taper(dangerOnWhite, 0, phase)
	k.KingSafety -= taper(dangerOnBlack, 0, phase)

	return EvalBreakdown{
		Material:      w.Material - k.Material,
		PieceSquare:   w.PieceSquare - k.PieceSquare,
		PawnStructure: w.PawnStructure - k.PawnStructure,
		PassedPawns:   w.PassedPawns - k.PassedPawns,
		Pieces:        w.Pieces - k.Pieces,
		Mobility:      w.Mobility - k.Mobility,
		Threats:       w.Threats - k.Threats,
		KingSafety:    w.KingSafety - k.KingSafety,
	}
}

func Evaluate(p *chess.Position) int {
	return EvaluateBreakdown(p).Total()
}

func pieceValue(piece chess.PieceType) int {
	switch piece {
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

func pieceSquareBonus(
	piece chess.PieceType,
	color chess.Color,
	square int,
) int {
	if color == chess.Black {
		square = flipSquare(square)
	}

	switch piece {
	case chess.WhitePawn, chess.BlackPawn:
		return pawnPST[square]

	case chess.WhiteKnight, chess.BlackKnight:
		return knightPST[square]

	case chess.WhiteBishop, chess.BlackBishop:
		return bishopPST[square]

	case chess.WhiteRook, chess.BlackRook:
		return rookPST[square]

	case chess.WhiteQueen, chess.BlackQueen:
		return queenPST[square]

	case chess.WhiteKing, chess.BlackKing:
		return kingMGPST[square]

	default:
		return 0
	}
}

func flipSquare(square int) int {
	file := square % 8
	rank := square / 8

	return (7-rank)*8 + file
}
