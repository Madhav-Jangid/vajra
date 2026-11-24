package chess

import "math/bits"

func (p *Position) LegalMoves(square int) Moves {
	piece := p.Board.PieceAt(square)

	if piece.Type == Empty {
		return nil
	}

	var legal Moves

	for _, move := range p.GenerateMoves(square) {
		undo := p.MakeMove(move)

		if !p.InCheck(piece.Color) {
			legal = append(legal, move)
		}

		p.UnmakeMove(move, undo)
	}

	return legal
}

func (p *Position) pinnedPieces(color Color, kingSq int) Bitboard {
	if kingSq < 0 || kingSq >= 64 {
		return 0
	}

	b := &p.Board
	var ownOcc, enemyRQ, enemyBQ Bitboard
	if color == White {
		ownOcc = b.WhitePawns | b.WhiteKnights | b.WhiteBishops | b.WhiteRooks | b.WhiteQueens | b.WhiteKing
		enemyRQ = b.BlackRooks | b.BlackQueens
		enemyBQ = b.BlackBishops | b.BlackQueens
	} else {
		ownOcc = b.BlackPawns | b.BlackKnights | b.BlackBishops | b.BlackRooks | b.BlackQueens | b.BlackKing
		enemyRQ = b.WhiteRooks | b.WhiteQueens
		enemyBQ = b.WhiteBishops | b.WhiteQueens
	}

	occ := b.Occupied()
	kr, kf := kingSq/8, kingSq%8
	var pinned Bitboard

	scan := func(dirs [4][2]int, sliders Bitboard) {
		if sliders == 0 {
			return
		}
		for _, d := range dirs {
			r, f := kr+d[0], kf+d[1]
			var blocker Bitboard
			for r >= 0 && r < 8 && f >= 0 && f < 8 {
				bit := Bitboard(1) << uint(r*8+f)
				if occ&bit != 0 {
					if blocker == 0 {
						if ownOcc&bit == 0 {
							break
						}
						blocker = bit
					} else {
						if sliders&bit != 0 {
							pinned |= blocker
						}
						break
					}
				}
				r += d[0]
				f += d[1]
			}
		}
	}

	scan(rookDirs, enemyRQ)
	scan(bishopDirs, enemyBQ)
	return pinned
}

func (p *Position) AllLegalMoves() Moves {
	b := &p.Board
	side := p.SideToMove

	var ownOcc Bitboard
	if side == White {
		ownOcc = b.WhitePawns | b.WhiteKnights | b.WhiteBishops | b.WhiteRooks | b.WhiteQueens | b.WhiteKing
	} else {
		ownOcc = b.BlackPawns | b.BlackKnights | b.BlackBishops | b.BlackRooks | b.BlackQueens | b.BlackKing
	}

	kingSq := p.kingSquare(side)
	inCheck := kingSq >= 0 && b.IsSquareAttacked(kingSq, opponent(side))
	var pinned Bitboard
	if !inCheck {
		pinned = p.pinnedPieces(side, kingSq)
	}

	moves := make(Moves, 0, 48)
	for bb := uint64(ownOcc); bb != 0; bb &= bb - 1 {
		sq := bits.TrailingZeros64(bb)
		needCheck := inCheck || sq == kingSq || (pinned&(Bitboard(1)<<uint(sq))) != 0
		for _, move := range p.GenerateMoves(sq) {
			if !needCheck && move.Flags&EnPassant == 0 {
				moves = append(moves, move)
				continue
			}
			undo := p.MakeMove(move)
			if !p.InCheck(side) {
				moves = append(moves, move)
			}
			p.UnmakeMove(move, undo)
		}
	}
	return moves
}

func (p *Position) PlayMove(from, to int, promo PieceType) bool {
	if from < 0 || from >= 64 || to < 0 || to >= 64 {
		return false
	}
	piece := p.Board.PieceAt(from)
	if piece.Type == Empty || piece.Color != p.SideToMove {
		return false
	}
	for _, m := range p.LegalMoves(from) {
		if m.To != to {
			continue
		}
		if m.Flags&Promotion != 0 {
			want := promo
			if want == Empty {
				want = WhiteQueen
				if piece.Color == Black {
					want = BlackQueen
				}
			}
			if m.Promotion != want {
				continue
			}
		}
		p.MakeMove(m)
		return true
	}
	return false
}
