# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## ⚠️ CRITICAL: Keep /docs up to date

The `docs/` directory contains detailed documentation of how this application works.
**Any time you make a change to the codebase — however small — you MUST review every
file in `docs/` and update any section that is affected by your change.**

This is not optional. The docs are the primary reference for understanding this codebase
and for porting it to other platforms (a JavaScript webapp is planned). Stale docs are
worse than no docs.

**Specifically:**
- Changed how move search works? Update `docs/ALGORITHM.md` §5 and §9.
- Changed board representation or data structures? Update `docs/ALGORITHM.md` §1–§2.
- Changed scoring logic? Update `docs/ALGORITHM.md` §7.
- Added a new file or changed the file layout? Update the File layout section below
  and any relevant doc files.
- Added a new concept that isn't covered? Add a new section to the appropriate doc
  (or create a new file in `docs/`).

When in doubt: read the relevant doc section, check whether it still accurately describes
the code, and fix anything that doesn't match.

---

## Documentation index

All detailed documentation lives in `docs/`:

| File | What it covers |
|---|---|
| `docs/ALGORITHM.md` | Full engine deep-dive: board layout, trie DFS, play space, cross-word validation, scoring, JS port checklist |

---

## Project Overview

A two-player Scrabble AI simulation written in pure Go with no external dependencies.
Two AI players play against each other using a greedy strategy (always picks the
highest-scoring valid move). Console-only output with ANSI-colored board display.
An interactive solver mode (`./scrabble solve`) lets a human player get move suggestions.

## Build & Run

```bash
cd go
go build -o scrabble .
./scrabble        # AI vs AI simulation
./scrabble solve  # Interactive solver UI
```

All Go runtime files live in `go/`. The `boards/` directory is in the repo root and
symlinked into `go/boards` so the code can reference it as a plain relative path.
If you clone just the `go/` directory standalone, create a `go/boards/` folder there.

There are no external Go dependencies.

## File layout

```
/
├── boards/          # Saved board files (shared; symlinked from go/boards)
├── docs/            # Detailed documentation (see index above)
├── CLAUDE.md        # This file
├── README.md        # Project readme
└── go/              # All Go source and runtime data
    ├── main.go          # Entry point; dispatches to runGame or runSolve
    ├── common.go        # Shared engine: Board/Trie, scoring, searchPlay, getPlaySpace
    ├── scrabble.go      # AI vs AI game loop (NewBoard, DoTurn, runGame)
    ├── solve.go         # Interactive solver UI, findTopNMoves, terminal rendering
    ├── go.mod           # Go module file
    ├── dictionary.txt   # 178K-word dictionary (required at runtime)
    ├── rulesets.json    # Ruleset definitions (NYT Crossplay, Standard Scrabble)
    ├── config.json      # Active ruleset selection (optional; defaults to NYT Crossplay)
    └── boards -> ../boards  # Symlink to root boards/
```

## Architecture summary

For full details see `docs/ALGORITHM.md`. High-level overview:

**Board & Game State (`Board` struct):**
- `board`: 15×15 `[][]byte`, column-major (`board[x][y]`), indexed flat via `cti(x, y) = y*15 + x`
- `tiles`: Remaining tile pool
- `wordlist`: Dictionary as FNV-1a hash map for O(1) cross-word lookups
- `trie`: Prefix trie used by the move-search DFS for main-word validation and pruning
- `pscore`/`ptiles`: Per-player scores and tile hands (2 players, 7 tiles each)
- Board multipliers: flat `[225]bool` arrays `tw`, `dw`, `tl`, `dl`

**Move Search (`DoTurn` / `findTopNMoves`):**
1. Iterate every empty cell as a potential anchor; call `getPlaySpace` to get the run of tiles in that direction
2. Pre-walk the trie through any existing tiles before the anchor
3. Run `searchPlay` DFS: walk trie and play-space simultaneously, placing rack tiles at empty cells, pruning when no trie edge exists
4. Cross-words at each empty cell are validated via FNV hash lookup
5. When `node.isEnd`, call `recordMove` to validate geometry, score, and collect the move

**Wildcards:** `'*'` in the rack. DFS expands to all 26 letters but only follows existing trie edges. Placed blanks stored as lowercase on the board (score 0).

**Scoring:** `scoreWord` handles letter/word multipliers (only new tiles activate multiplier squares). Cross-words scored in `scoreMove`. Single-letter words score 0.
