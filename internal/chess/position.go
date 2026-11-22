package chess

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
