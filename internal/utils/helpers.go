package utils

import (
	"fmt"
	"strings"
	"vajra/internal/chess"
)

func PrintBoard() {
	for rank := 7; rank >= 0; rank-- {
		fmt.Printf("%d ", rank+1)
		for file := 0; file <= 7; file++ {
			piece := GetPieceAt(rank*8 + file)
			fmt.Print(piece.Face, " ")
		}
		fmt.Println()
	}
	fmt.Println("  a b c d e f g h")
}

func GetPieceAt(square int) chess.Piece {
	return chess.CurrentPosition.Board.PieceAt(square)
}

func SquareName(sq int) string {
	return string(rune('a'+sq%8)) + string(rune('1'+sq/8))
}

func ParseSquare(s string) (int, error) {
	if len(s) != 2 || s[0] < 'a' || s[0] > 'h' || s[1] < '1' || s[1] > '8' {
		return -1, fmt.Errorf("invalid square %q", s)
	}
	return int(s[1]-'1')*8 + int(s[0]-'a'), nil
}

func ParseMove(s string, side chess.Color) (from, to int, promo chess.PieceType, err error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) != 4 && len(s) != 5 {
		return 0, 0, chess.Empty, fmt.Errorf("invalid move: expected e.g. e2e4")
	}
	if from, err = ParseSquare(s[:2]); err != nil {
		return
	}
	if to, err = ParseSquare(s[2:4]); err != nil {
		return
	}
	if len(s) == 5 {
		letter := s[4:]
		if side == chess.White {
			letter = strings.ToUpper(letter)
		}
		switch letter {
		case "q", "r", "b", "n", "Q", "R", "B", "N":
			promo = chess.PieceTypeMap[letter]
		default:
			return 0, 0, chess.Empty, fmt.Errorf("invalid promotion piece %q", s[4:])
		}
	}
	return from, to, promo, nil
}
