package main

import (
	"fmt"
	"vajra/internal/chess"
	"vajra/internal/utils"
)

func main() {
	fmt.Println("Welcome to cli chess")
	fmt.Println()
	chess.InitializeBitBoard()
	utils.PrintBoard()
	var move string
	for {
		fmt.Println()
		fmt.Printf("Enter a move:- ")
		_, err := fmt.Scan(&move)
		if err != nil {
			fmt.Println("Error reading move:", err)
			return
		}
		fmt.Println()

		if move == "exit" {
			return
		}
		idx := utils.ParseMove(move)
		utils.MakeMove(idx[0], idx[1])
		utils.PrintBoard()

	}
}
