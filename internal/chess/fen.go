package chess

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

type Position struct {
	Board          PieceBoards
	SideToMove     Color
	CastlingRights uint8
	EnPassant      int
	HalfmoveClock  int
	FullmoveNumber int
}

var CurrentPosition Position

const (
	WhiteKingSide uint8 = 1 << iota
	WhiteQueenSide
	BlackKingSide
	BlackQueenSide
)

func parseColor(s string) (Color, error) {
	switch s {
	case "w":
		return White, nil
	case "b":
		return Black, nil
	default:
		return 0, fmt.Errorf("invalid side to move: %q", s)
	}
}

func ParseFEN(fen string) (Position, error) {
	board := PieceBoards{}
	fenComponents := strings.Fields(fen)

	if len(fenComponents) != 6 {
		return Position{}, fmt.Errorf("invalid FEN: expected 6 fields")
	}

	pos := fenComponents[0]
	sideToMove, err := parseColor(fenComponents[1])
	if err != nil {
		return Position{}, err
	}
	castlingRights := fenComponents[2]
	enPassant, err := parseEnPassant(fenComponents[3])
	if err != nil {
		return Position{}, err
	}
	halfmoveClock, err := strconv.Atoi(fenComponents[4])
	if err != nil {
		return Position{}, fmt.Errorf("invalid halfmove clock: %w", err)
	}
	fullmoveNumber, err := strconv.Atoi(fenComponents[5])
	if err != nil {
		return Position{}, fmt.Errorf("invalid fullmove number: %w", err)
	}

	castling := uint8(0)
	for _, c := range castlingRights {
		switch c {
		case 'K':
			castling |= WhiteKingSide
		case 'Q':
			castling |= WhiteQueenSide
		case 'k':
			castling |= BlackKingSide
		case 'q':
			castling |= BlackQueenSide
		case '-':
		}
	}

	ranks := strings.Split(pos, "/")
	if len(ranks) != 8 {
		return Position{}, fmt.Errorf("invalid FEN position: expected 8 ranks, got %d", len(ranks))
	}
	for fenRank, rankData := range ranks {
		rank := 7 - fenRank
		file := 0
		for _, char := range rankData {
			if char >= '1' && char <= '8' {
				file += int(char - '0')
				continue
			}
			piece, ok := PieceTypeMap[string(char)]
			if !ok {
				return Position{}, fmt.Errorf("invalid piece: %c", char)
			}
			if file >= 8 {
				return Position{}, fmt.Errorf("Too many square in rank")
			}
			square := rank*8 + file
			board.SetPiece(piece, square)
			file++
		}
		if file != 8 {
			return Position{}, fmt.Errorf("rank does not contain 8 squares")
		}
	}

	return Position{Board: board, SideToMove: sideToMove, CastlingRights: castling,
		EnPassant: enPassant, HalfmoveClock: halfmoveClock, FullmoveNumber: fullmoveNumber}, nil
}

func (b *PieceBoards) SetPiece(piece PieceType, square int) {
	if square < 0 || square >= 64 {
		return
	}
	mask := Bitboard(1) << square
	switch piece {
	case WhiteKing:
		b.WhiteKing |= mask
	case BlackKing:
		b.BlackKing |= mask
	case WhiteQueen:
		b.WhiteQueens |= mask
	case BlackQueen:
		b.BlackQueens |= mask
	case WhiteRook:
		b.WhiteRooks |= mask
	case BlackRook:
		b.BlackRooks |= mask
	case WhiteBishop:
		b.WhiteBishops |= mask
	case BlackBishop:
		b.BlackBishops |= mask
	case WhiteKnight:
		b.WhiteKnights |= mask
	case BlackKnight:
		b.BlackKnights |= mask
	case WhitePawn:
		b.WhitePawns |= mask
	case BlackPawn:
		b.BlackPawns |= mask
	}
}

func SetUpInitialPosition() error {
	startingPosition := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	position, err := ParseFEN(startingPosition)
	if err != nil {
		return err
	}

	CurrentPosition = position
	return nil
}

func parseEnPassant(s string) (int, error) {
	if s == "-" {
		return -1, nil
	}

	if len(s) != 2 {
		return -1, fmt.Errorf("invalid en passant square: %q", s)
	}

	file := int(s[0] - 'a')
	rank := int(s[1] - '1')

	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return -1, fmt.Errorf("invalid en passant square: %q", s)
	}

	return rank*8 + file, nil
}

func (p Position) boardFen() string {
	var result strings.Builder
	for rank := 7; rank >= 0; rank-- {
		nEmpty := 0
		for file := 0; file <= 7; file++ {
			square := rank*8 + file
			piece := p.Board.PieceAt(square)
			if piece.Face == "." {
				nEmpty++
				continue
			}
			if nEmpty > 0 {
				result.Write([]byte(strconv.Itoa(nEmpty)))
				nEmpty = 0
			}
			result.Write([]byte(piece.Face))
		}
		if nEmpty > 0 {
			result.Write([]byte(strconv.Itoa(nEmpty)))
		}
		if rank != 0 {
			result.Write([]byte("/"))
		}
	}
	return result.String()
}

func (p Position) FEN() string {
	sideToMove := "b"
	if p.SideToMove == White {
		sideToMove = "w"
	}
	return strings.Join([]string{
		p.boardFen(),
		sideToMove,
		p.castlingFEN(),
		indexToSquare(p.EnPassant),
		strconv.Itoa(p.HalfmoveClock),
		strconv.Itoa(p.FullmoveNumber),
	}, " ")
}

func indexToSquare(idx int) string {
	var square string
	if idx < 0 || idx >= 64 {
		return "-"
	}
	rank := idx/8 + 1
	file := idx % 8
	fileMapings := map[int]string{
		0: "a",
		1: "b",
		2: "c",
		3: "d",
		4: "e",
		5: "f",
		6: "g",
		7: "h",
	}
	square += fileMapings[file]
	square += strconv.Itoa(rank)
	return square
}

func (p Position) castlingFEN() string {
	if p.CastlingRights == 0 {
		return "-"
	}

	var b strings.Builder

	if p.CastlingRights&WhiteKingSide != 0 {
		b.WriteByte('K')
	}
	if p.CastlingRights&WhiteQueenSide != 0 {
		b.WriteByte('Q')
	}
	if p.CastlingRights&BlackKingSide != 0 {
		b.WriteByte('k')
	}
	if p.CastlingRights&BlackQueenSide != 0 {
		b.WriteByte('q')
	}

	return b.String()
}

func (p Position) ValidateFen() (bool, error) {
	cFen := p.FEN()
	pos, err := ParseFEN(cFen)
	if err != nil {
		return false, err
	}
	if pos != p {
		return false, nil
	}
	log.Println(cFen)
	return true, nil
}
