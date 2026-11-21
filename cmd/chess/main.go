package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"vajra/internal/chess"
	"vajra/internal/utils"
)

var (
	squareRe = regexp.MustCompile(`^[a-h][1-8]$`)
	moveRe   = regexp.MustCompile(`^[a-h][1-8][a-h][1-8]$`)
)

func name(c chess.Color) string {
	if c == chess.White {
		return "White"
	}
	return "Black"
}

func main() {
	fmt.Println("Welcome to cli chess")
	fmt.Println("Enter moves like e2e4. Commands: moves e2, new, exit")
	fmt.Println()

	chess.InitializeBitBoard()
	turn := chess.White
	seen := map[string]int{utils.PositionKey(turn): 1}
	utils.PrintBoard()

	in := bufio.NewScanner(os.Stdin)
	if in.Err() != nil {
		return
	}
	for {
		fmt.Println()
		if utils.InCheck(turn) {
			fmt.Println("Check!")
		}
		fmt.Printf("%s to move:- ", name(turn))
		if !in.Scan() {
			return
		}
		move := strings.ToLower(strings.TrimSpace(in.Text()))
		fmt.Println()

		switch {
		case move == "":
			continue
		case move == "exit" || move == "quit":
			return
		case move == "new":
			chess.InitializeBitBoard()
			turn = chess.White
			seen = map[string]int{utils.PositionKey(turn): 1}
			utils.PrintBoard()
			continue
		case strings.HasPrefix(move, "moves "):
			sq := strings.TrimSpace(move[len("moves "):])
			if !squareRe.MatchString(sq) {
				fmt.Println("Usage: moves e2")
				continue
			}
			idx := utils.ParseMove(sq + sq)
			if idx == nil {
				fmt.Println("Usage: moves e2")
				continue
			}
			for _, to := range utils.LegalMoves(idx[0]) {
				fmt.Print(utils.SquareName(to), " ")
			}
			fmt.Println()
			continue
		}

		if !moveRe.MatchString(move) {
			fmt.Println("Invalid input. Use a move like e2e4.")
			continue
		}
		idx := utils.ParseMove(move)
		if idx == nil {
			fmt.Println("Invalid input. Use a move like e2e4.")
			continue
		}
		if p := utils.GetPieceAt(idx[0]); p.Type == chess.Empty || p.Color != turn {
			fmt.Println("No", strings.ToLower(name(turn)), "piece on that square.")
			continue
		}
		if !utils.MakeMove(idx[0], idx[1]) {
			fmt.Println("Illegal move.")
			continue
		}

		if turn == chess.White {
			turn = chess.Black
		} else {
			turn = chess.White
		}
		utils.PrintBoard()

		status := utils.Status(turn)
		key := utils.PositionKey(turn)
		seen[key]++
		if status == "" && seen[key] >= 3 {
			status = "threefold repetition"
		}
		if status == "checkmate" {
			fmt.Printf("\nCheckmate! %s wins.\n", name(chess.Color(1-int(turn))))
			return
		} else if status != "" {
			fmt.Printf("\nDraw by %s.\n", status)
			return
		}
	}
}
