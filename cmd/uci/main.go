package main

import (
	"fmt"
	"os"

	"vajra/internal/uci"
)

func main() {
	if err := uci.New(os.Stdin, os.Stdout).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
