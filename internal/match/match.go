package match

import (
	"fmt"
	"strings"

	"vajra/internal/chess"
	"vajra/internal/engine"
	"vajra/internal/uci"
)

type Outcome int

const (
	Draw Outcome = iota
	WhiteWins
	BlackWins
)

type Player struct {
	Name  string
	Depth int
	Setup func()
}

type Tally struct {
	Wins, Draws, Losses int
}

func (t Tally) Games() int {
	return t.Wins + t.Draws + t.Losses
}

var Openings = [][]string{
	strings.Fields("e2e4 e7e5 g1f3 b8c6"),
	strings.Fields("e2e4 c7c5 g1f3 d7d6"),
	strings.Fields("d2d4 d7d5 c2c4 e7e6"),
	strings.Fields("d2d4 g8f6 c2c4 g7g6"),
	strings.Fields("e2e4 e7e6 d2d4 d7d5"),
	strings.Fields("c2c4 e7e5 b1c3 g8f6"),
	strings.Fields("g1f3 d7d5 g2g3 g8f6"),
	strings.Fields("e2e4 c7c6 d2d4 d7d5"),
	strings.Fields("d2d4 d7d5 g1f3 g8f6"),
	strings.Fields("e2e4 e7e5 f1c4 g8f6"),
	strings.Fields("e2e4 d7d6 d2d4 g8f6"),
	strings.Fields("c2c4 c7c5 g1f3 g8f6"),
}

func Play(white, black Player, opening []string, maxPlies int) (Outcome, error) {
	engine.ClearStop()

	pos, err := chess.ParseFEN(chess.StartingFEN)
	if err != nil {
		return Draw, err
	}

	var history []uint64

	seen := map[string]int{pos.Key(): 1}

	for _, text := range opening {
		move, err := uci.ParseMove(&pos, text)
		if err != nil {
			return Draw, fmt.Errorf("opening move %s: %w", text, err)
		}

		history = append(history, engine.HashPosition(&pos))

		pos.MakeMove(move)
		seen[pos.Key()]++
	}

	for ply := len(opening); ply < maxPlies; ply++ {
		switch pos.Status(pos.SideToMove) {
		case "checkmate":
			if pos.SideToMove == chess.White {
				return BlackWins, nil
			}

			return WhiteWins, nil
		case "":
		default:
			return Draw, nil
		}

		if seen[pos.Key()] >= 3 {
			return Draw, nil
		}

		player := black
		if pos.SideToMove == chess.White {
			player = white
		}

		if player.Setup != nil {
			player.Setup()
		}

		engine.GameHistory = append([]uint64(nil), history...)

		move, ok := engine.IterativeSearch(&pos, player.Depth, 0)
		if !ok {
			return Draw, nil
		}

		history = append(history, engine.HashPosition(&pos))

		pos.MakeMove(move)
		seen[pos.Key()]++
	}

	return Draw, nil
}

func Run(a, b Player, openings [][]string, maxPlies int) (Tally, error) {
	var tally Tally

	for _, opening := range openings {
		for _, aIsWhite := range []bool{true, false} {
			white, black := a, b
			if !aIsWhite {
				white, black = b, a
			}

			outcome, err := Play(white, black, opening, maxPlies)
			if err != nil {
				return tally, err
			}

			switch {
			case outcome == Draw:
				tally.Draws++
			case (outcome == WhiteWins) == aIsWhite:
				tally.Wins++
			default:
				tally.Losses++
			}
		}
	}

	return tally, nil
}
