package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "solve" {
		runSolve()
	} else if len(os.Args) > 1 {
		fmt.Fprintf(os.Stderr, "usage: scrabble [solve]\n")
		os.Exit(1)
	} else {
		runGame()
	}
}
