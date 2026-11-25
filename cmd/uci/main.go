package main

import (
	"fmt"
	"os"
	"time"

	"vajra/internal/book"
	"vajra/internal/uci"
)

func main() {
	e := uci.New(os.Stdin, os.Stdout)

	e.LoadBook = func(path string) (uci.Book, error) {
		return book.Load(path, time.Now().UnixNano())
	}

	if err := e.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
