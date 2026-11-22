package chess

import (
	"math/bits"
)

type Directions [][2]int

type Move struct {
	From      int
	To        int
	Piece     PieceType
	Captured  PieceType
	Promotion PieceType
	Flags     MoveFlags
}

type MoveFlags uint8

const (
	Quiet          MoveFlags = 0
	Capture        MoveFlags = 1 << 0
	DoublePawnPush MoveFlags = 1 << 1
	KingCastle     MoveFlags = 1 << 2
	QueenCastle    MoveFlags = 1 << 3
	EnPassant      MoveFlags = 1 << 4
	Promotion      MoveFlags = 1 << 5
)

type Moves []Move

func (p *Position) GenerateMoves(square int) Moves {
	if square < 0 || square >= 64 {
		return nil
	}

	piece := p.Board.PieceAt(square)

	switch piece.Type {
	case WhiteKing, BlackKing:
		return p.GetKingsMoves(square, piece.Color)
	case WhitePawn, BlackPawn:
		return p.GetPawnsMoves(square, piece.Color)
	case WhiteKnight, BlackKnight:
		return p.stepMoves(square, piece, GetKnightsDirections())
	case WhiteQueen, BlackQueen:
		return p.slideMoves(square, piece, GetAllDirections())
	case WhiteBishop, BlackBishop:
		return p.slideMoves(square, piece, GetBishopsDirections())
	case WhiteRook, BlackRook:
		return p.slideMoves(square, piece, GetRooksDirections())
	}
	return nil
}

func (p *Position) stepMoves(square int, piece Piece, dirs Directions) Moves {
	var moves Moves
	rank, file := square/8, square%8
	for _, d := range dirs {
		r, f := rank+d[0], file+d[1]
		if r < 0 || r >= 8 || f < 0 || f >= 8 {
			continue
		}
		to := r*8 + f
		target := p.Board.PieceAt(to)
		switch {
		case target.Type == Empty:
			moves = append(moves, Move{From: square, To: to, Piece: piece.Type, Flags: Quiet})
		case target.Color != piece.Color:
			moves = append(moves, Move{From: square, To: to, Piece: piece.Type, Captured: target.Type, Flags: Capture})
		}
	}
	return moves
}

func (p *Position) slideMoves(square int, piece Piece, dirs Directions) Moves {
	var moves Moves
	rank, file := square/8, square%8
	for _, d := range dirs {
		r, f := rank+d[0], file+d[1]
		for r >= 0 && r < 8 && f >= 0 && f < 8 {
			to := r*8 + f
			target := p.Board.PieceAt(to)
			if target.Type != Empty {
				if target.Color != piece.Color {
					moves = append(moves, Move{From: square, To: to, Piece: piece.Type, Captured: target.Type, Flags: Capture})
				}
				break
			}
			moves = append(moves, Move{From: square, To: to, Piece: piece.Type, Flags: Quiet})
			r += d[0]
			f += d[1]
		}
	}
	return moves
}

func GetAllDirections() Directions {
	return Directions{
		{-1, 1},
		{0, 1},
		{1, 1},
		{1, 0},
		{1, -1},
		{0, -1},
		{-1, -1},
		{-1, 0},
	}
}

func (p *Position) GetKingsMoves(square int, color Color) Moves {
	piece := p.Board.PieceAt(square)
	moves := p.stepMoves(square, piece, GetAllDirections())

	home, enemy := 4, Black
	kingSide, queenSide := WhiteKingSide, WhiteQueenSide
	rook := WhiteRook
	if color == Black {
		home, enemy = 60, White
		kingSide, queenSide = BlackKingSide, BlackQueenSide
		rook = BlackRook
	}
	empty := func(sq int) bool { return !p.Board.IsSquareOccupied(sq) }
	hasRook := func(sq int) bool {
		r := p.Board.PieceAt(sq)
		return r.Type == rook
	}
	if square == home && !p.Board.IsSquareAttacked(home, enemy) {
		if p.CastlingRights&kingSide != 0 && hasRook(home+3) &&
			empty(home+1) && empty(home+2) &&
			!p.Board.IsSquareAttacked(home+1, enemy) {
			moves = append(moves, Move{From: square, To: home + 2, Piece: piece.Type, Flags: KingCastle})
		}
		if p.CastlingRights&queenSide != 0 && hasRook(home-4) &&
			empty(home-1) && empty(home-2) && empty(home-3) &&
			!p.Board.IsSquareAttacked(home-1, enemy) {
			moves = append(moves, Move{From: square, To: home - 2, Piece: piece.Type, Flags: QueenCastle})
		}
	}

	return moves
}

func GetRooksDirections() Directions {
	return Directions{
		{0, 1},
		{1, 0},
		{0, -1},
		{-1, 0},
	}
}

func GetBishopsDirections() Directions {
	return Directions{
		{-1, 1},
		{1, 1},
		{1, -1},
		{-1, -1},
	}
}

func GetKnightsDirections() Directions {
	return Directions{
		{2, 1},
		{2, -1},
		{-2, 1},
		{-2, -1},

		{1, 2},
		{1, -2},
		{-1, 2},
		{-1, -2},
	}
}

func (p *Position) GetPawnsMoves(square int, color Color) Moves {
	var moves Moves

	rank, file := square/8, square%8

	forward, startRank, promoRank, epRank := 8, 1, 7, 5
	promos := []PieceType{WhiteQueen, WhiteRook, WhiteBishop, WhiteKnight}
	enemyPawn := BlackPawn
	if color == Black {
		forward, startRank, promoRank, epRank = -8, 6, 0, 2
		promos = []PieceType{BlackQueen, BlackRook, BlackBishop, BlackKnight}
		enemyPawn = WhitePawn
	}
	piece := p.Board.PieceAt(square)

	add := func(to int, captured PieceType, flags MoveFlags) {
		if to/8 == promoRank {
			for _, promo := range promos {
				moves = append(moves, Move{From: square, To: to, Piece: piece.Type,
					Captured: captured, Promotion: promo, Flags: flags | Promotion})
			}
			return
		}
		moves = append(moves, Move{From: square, To: to, Piece: piece.Type,
			Captured: captured, Flags: flags})
	}

	one := square + forward
	if one >= 0 && one < 64 && !p.Board.IsSquareOccupied(one) {
		add(one, Empty, Quiet)

		if rank == startRank {
			two := square + 2*forward
			if !p.Board.IsSquareOccupied(two) {
				add(two, Empty, DoublePawnPush)
			}
		}
	}

	for _, df := range []int{-1, 1} {
		if file+df < 0 || file+df > 7 {
			continue
		}
		capture := square + forward + df
		if capture < 0 || capture >= 64 {
			continue
		}
		target := p.Board.PieceAt(capture)
		if target.Type != Empty {
			if target.Color != color {
				add(capture, target.Type, Capture)
			}
		} else if capture == p.EnPassant && capture/8 == epRank {
			add(capture, enemyPawn, Capture|EnPassant)
		}
	}

	return moves
}

func (p *Position) kingSquare(c Color) int {
	king := p.Board.BlackKing
	if c == White {
		king = p.Board.WhiteKing
	}
	if king == 0 {
		return -1
	}
	return bits.TrailingZeros64(uint64(king))
}

func opponent(c Color) Color {
	if c == White {
		return Black
	}
	return White
}

func (p *Position) InCheck(c Color) bool {
	sq := p.kingSquare(c)
	return sq >= 0 && p.Board.IsSquareAttacked(sq, opponent(c))
}
