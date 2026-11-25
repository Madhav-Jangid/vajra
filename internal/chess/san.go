package chess

import "strings"

func (p *Position) SAN(m Move) string {
	var s strings.Builder

	switch {
	case m.Flags&KingCastle != 0:
		s.WriteString("O-O")
	case m.Flags&QueenCastle != 0:
		s.WriteString("O-O-O")
	case m.Piece == WhitePawn || m.Piece == BlackPawn:
		if m.Flags&Capture != 0 {
			s.WriteByte(byte('a' + m.From%8))
			s.WriteByte('x')
		}

		s.WriteString(squareText(m.To))

		if m.Flags&Promotion != 0 {
			s.WriteByte('=')
			s.WriteString(pieceLetter(m.Promotion))
		}
	default:
		s.WriteString(pieceLetter(m.Piece))
		s.WriteString(p.disambiguation(m))

		if m.Flags&Capture != 0 {
			s.WriteByte('x')
		}

		s.WriteString(squareText(m.To))
	}

	undo := p.MakeMove(m)

	if opponentColor := p.SideToMove; p.InCheck(opponentColor) {
		if p.HasLegalMove(opponentColor) {
			s.WriteByte('+')
		} else {
			s.WriteByte('#')
		}
	}

	p.UnmakeMove(m, undo)

	return s.String()
}

func (p *Position) disambiguation(m Move) string {
	sameFile, sameRank, ambiguous := false, false, false

	for _, other := range p.AllLegalMoves() {
		if other.To != m.To || other.From == m.From || other.Piece != m.Piece {
			continue
		}

		ambiguous = true
		sameFile = sameFile || other.From%8 == m.From%8
		sameRank = sameRank || other.From/8 == m.From/8
	}

	switch {
	case !ambiguous:
		return ""
	case !sameFile:
		return string(rune('a' + m.From%8))
	case !sameRank:
		return string(rune('1' + m.From/8))
	}

	return squareText(m.From)
}

func squareText(sq int) string {
	return string([]rune{rune('a' + sq%8), rune('1' + sq/8)})
}

func pieceLetter(t PieceType) string {
	switch t {
	case WhiteKing, BlackKing:
		return "K"
	case WhiteQueen, BlackQueen:
		return "Q"
	case WhiteRook, BlackRook:
		return "R"
	case WhiteBishop, BlackBishop:
		return "B"
	case WhiteKnight, BlackKnight:
		return "N"
	}

	return ""
}
