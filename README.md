# scrabble

A two-player Scrabble AI and personal game assistant written in pure Go.

## Modes

**AI simulation** — two AI players play a full game against each other using a greedy strategy (always picks the highest-scoring valid move):

```
go run .
```

**Solver** — interactive tool for your real Scrabble games. Reads board state from a file in `boards/`, takes your rack, and finds the top 10 legal moves with a live preview. Supports continuous play: enter your move, then your opponent's word, loop.

```
go run . solve
```

## Setup

Requires `dictionary.txt` (178K-word TWL/SOWPODS list) in the working directory. Create a `boards/` directory for your game files — each is a 15×15 text file with letters and `.` for empty squares.

The solver can create new blank board files for you from within the UI.

## Build

```
go build -o scrabble .
./scrabble
./scrabble solve
```

No external dependencies. Requires Go 1.18+.
