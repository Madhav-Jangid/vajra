package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"vajra/internal/chess"
	"vajra/internal/utils"
)

func name(c chess.Color) string {
	if c == chess.White {
		return "White"
	}
	return "Black"
}

func start() {
	fmt.Println("Welcome to cli chess")
	fmt.Println("Enter moves like e2e4 (e7e8q to promote). Commands: moves e2, new, exit")
	fmt.Println()

	pos := &chess.CurrentPosition
	seen := map[string]int{pos.Key(): 1}
	utils.PrintBoard()

	in := bufio.NewScanner(os.Stdin)
	for {
		turn := pos.SideToMove
		fmt.Println()
		if pos.InCheck(turn) {
			fmt.Println("Check!")
		}
		fmt.Printf("%s to move:- ", name(turn))
		if !in.Scan() {
			if err := in.Err(); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			return
		}
		input := strings.ToLower(strings.TrimSpace(in.Text()))
		fmt.Println()

		switch {
		case input == "":
			continue
		case input == "exit" || input == "quit":
			return
		case input == "new":
			if err := chess.SetUpInitialPosition(); err != nil {
				fmt.Println(err)
				return
			}
			seen = map[string]int{pos.Key(): 1}
			utils.PrintBoard()
			continue
		case strings.HasPrefix(input, "moves "):
			sq, err := utils.ParseSquare(strings.TrimSpace(input[len("moves "):]))
			if err != nil {
				fmt.Println("Usage: moves e2")
				continue
			}
			for _, mv := range pos.LegalMoves(sq) {
				fmt.Print(utils.SquareName(mv.To), " ")
			}
			fmt.Println()
			continue
		}

		from, to, promo, err := utils.ParseMove(input, turn)
		if err != nil {
			fmt.Println("Invalid input. Use a move like e2e4.")
			continue
		}
		if p := utils.GetPieceAt(from); p.Type == chess.Empty || p.Color != turn {
			fmt.Println("No", strings.ToLower(name(turn)), "piece on that square.")
			continue
		}
		if !pos.PlayMove(from, to, promo) {
			fmt.Println("Illegal move.")
			continue
		}

		turn = pos.SideToMove
		utils.PrintBoard()

		status := pos.Status(turn)
		key := pos.Key()
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

func main() {
	if err := chess.SetUpInitialPosition(); err != nil {
		fmt.Println(err)
		return
	}
	start()
}
