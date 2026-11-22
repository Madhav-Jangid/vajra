package chess

type Undo struct {
	CapturedPiece  PieceType
	CastlingRights uint8
	EnPassant      int
	HalfmoveClock  int
	FullmoveNumber int
}

func isPawn(t PieceType) bool { return t == WhitePawn || t == BlackPawn }

func castleMask(sq int) uint8 {
	switch sq {
	case 0:
		return WhiteQueenSide
	case 7:
		return WhiteKingSide
	case 4:
		return WhiteKingSide | WhiteQueenSide
	case 56:
		return BlackQueenSide
	case 63:
		return BlackKingSide
	case 60:
		return BlackKingSide | BlackQueenSide
	}
	return 0
}

func (p *Position) MakeMove(move Move) Undo {
	undo := Undo{
		CapturedPiece:  move.Captured,
		CastlingRights: p.CastlingRights,
		EnPassant:      p.EnPassant,
		HalfmoveClock:  p.HalfmoveClock,
		FullmoveNumber: p.FullmoveNumber,
	}

	p.Board.ApplyMove(move)

	p.CastlingRights &^= castleMask(move.From) | castleMask(move.To)

	p.EnPassant = -1
	if move.Flags&DoublePawnPush != 0 {
		p.EnPassant = (move.From + move.To) / 2
	}

	if isPawn(move.Piece) || move.Captured != Empty {
		p.HalfmoveClock = 0
	} else {
		p.HalfmoveClock++
	}

	if p.SideToMove == White {
		p.SideToMove = Black
	} else {
		p.SideToMove = White
		p.FullmoveNumber++
	}

	return undo
}

func (p *Position) UnmakeMove(move Move, undo Undo) {
	from, to := move.From, move.To

	p.SideToMove = oppositeColor(p.SideToMove)
	p.CastlingRights = undo.CastlingRights
	p.EnPassant = undo.EnPassant
	p.HalfmoveClock = undo.HalfmoveClock
	p.FullmoveNumber = undo.FullmoveNumber

	color := p.SideToMove

	p.Board.ClearSquare(to)
	p.Board.SetPiece(move.Piece, from)

	if move.Flags&EnPassant != 0 {
		capturedSquare := to + 8
		if color == White {
			capturedSquare = to - 8
		}
		p.Board.SetPiece(undo.CapturedPiece, capturedSquare)
	} else if undo.CapturedPiece != Empty {
		p.Board.SetPiece(undo.CapturedPiece, to)
	}

	if move.Flags&KingCastle != 0 {
		if color == White {
			p.Board.ClearSquare(5)
			p.Board.SetPiece(WhiteRook, 7)
		} else {
			p.Board.ClearSquare(61)
			p.Board.SetPiece(BlackRook, 63)
		}
	}

	if move.Flags&QueenCastle != 0 {
		if color == White {
			p.Board.ClearSquare(3)
			p.Board.SetPiece(WhiteRook, 0)
		} else {
			p.Board.ClearSquare(59)
			p.Board.SetPiece(BlackRook, 56)
		}
	}
}

func oppositeColor(color Color) Color {
	if color == White {
		return Black
	}
	return White
}
