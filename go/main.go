package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "solve":
			runSolve()
		case "serve":
			runServer()
		default:
			fmt.Fprintf(os.Stderr, "usage: scrabble [solve|serve]\n")
			os.Exit(1)
		}
	} else {
		runGame()
	}
}
