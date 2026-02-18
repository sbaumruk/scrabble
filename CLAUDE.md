# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A two-player Scrabble AI simulation written in pure Go with no external dependencies. Two AI players play against each other using a greedy strategy (always picks the highest-scoring valid move). Console-only output with ANSI-colored board display.

## Build & Run

```bash
go build -o scrabble scrabble.go
./scrabble
```

Requires `dictionary.txt` (178K words) in the working directory at runtime.

There are no tests, no go.mod, and no external dependencies.

## Architecture

Single-file project (`scrabble.go`, ~464 lines). Key components:

**Board & Game State (`Board` struct):**
- `board`: 15x15 byte array (`[225]byte`, indexed via `cti(x, y) = y*15 + x`)
- `tiles`: Remaining tile pool
- `wordlist`: Dictionary stored as FNV-1a hash map for O(1) lookups
- `pscore`/`ptiles`: Per-player scores and tile hands (2 players, 7 tiles each)
- Board multipliers defined as coordinate arrays: `tw`, `dw`, `tl`, `dl`

**Game Loop (`main` → `DoTurn`):**
1. For each player, randomly iterate board positions trying both directions
2. Generate all permutations/subsequences of player's hand tiles via `permute()`
3. Validate candidate words against dictionary, check board geometry (`checkContiguous`, `checkCenterPlayed`)
4. Score valid moves (`scoreMove`/`scoreWord`) including cross-words and multipliers
5. Play the highest-scoring move, draw replacement tiles

**Wildcards:** Blank tiles are `*` in the tile pool. `permute()` expands them to all 26 letters. Wildcard tiles placed on the board are stored as lowercase letters (score 0 points).

**Scoring:** `scoreWord` handles letter/word multipliers. Cross-words formed by a placement are scored in `scoreMove`. Single-letter words score 0.
