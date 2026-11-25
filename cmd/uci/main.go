package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"vajra/internal/book"
	"vajra/internal/uci"
)

func main() {
	if len(os.Args) > 1 {
		os.Exit(command(os.Args[1:]))
	}

	e := uci.New(os.Stdin, os.Stdout)

	e.LoadBook = func(path string) (uci.Book, error) {
		return book.Load(path, time.Now().UnixNano())
	}

	if err := e.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func command(args []string) int {
	switch args[0] {
	case "version", "-version", "--version":
		fmt.Printf("%s %s\n", uci.Name, uci.Version)

		return 0
	case "bench":
		depth := uci.DefaultBenchDepth

		if len(args) > 1 {
			n, err := strconv.Atoi(args[1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "bench depth must be a number, got %q\n", args[1])

				return 2
			}

			depth = n
		}

		if _, err := uci.Bench(os.Stdout, depth); err != nil {
			fmt.Fprintln(os.Stderr, err)

			return 2
		}

		return 0
	}

	fmt.Fprintf(os.Stderr, "usage: %s [version | bench [depth]]\n", os.Args[0])

	return 2
}
