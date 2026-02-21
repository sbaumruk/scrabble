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

A Scrabble AI solver with three modes: AI-vs-AI simulation, interactive terminal solver,
and a web UI. The web UI uses Keycloak OIDC for authentication and PostgreSQL for
board storage (with a file-based fallback for local dev). Two AI players play against
each other using a greedy strategy (always picks the highest-scoring valid move).
An interactive solver mode (`./scrabble solve`) lets a human player get move suggestions.
A web UI mode (`./scrabble serve`) starts an HTTP server with a SvelteKit frontend
for the same solver workflow in the browser.

## Build & Run

```bash
cd go
go build -o scrabble .
./scrabble        # AI vs AI simulation
./scrabble solve  # Interactive solver UI
./scrabble serve  # Web UI on http://localhost:8080
```

All Go runtime files live in `go/`. The `boards/` directory is in the repo root and
symlinked into `go/boards` so the code can reference it as a plain relative path.
If you clone just the `go/` directory standalone, create a `go/boards/` folder there.

**Go dependencies:** `github.com/jackc/pgx/v5` (PostgreSQL driver + connection pool),
`github.com/coreos/go-oidc/v3` (OIDC discovery + JWT verification).
The `solve` and AI simulation modes have no external deps; pgx and go-oidc are only
used by `serve`.

### Web UI development

The SvelteKit frontend lives in `web/`. During development, run the Go API and
Vite dev server in two terminals:

```bash
cd go && go run . serve          # API on :8080
cd web && npm run dev            # Vite on :5173 (api.ts points to :8080 in dev mode)
```

Production build (single binary with embedded frontend):

```bash
cd web && npm run build
rm -rf ../go/static && cp -r build ../go/static
cd ../go && go build -o scrabble .
./scrabble serve
```

### Docker deployment

The `docker-compose.yml` is copy-pasted into Portainer as a stack for deployment.
The compose file includes a PostgreSQL service and passes `DATABASE_URL` to the app.

```bash
docker-compose build             # Build locally (or deploy via Portainer)
docker-compose up                # Web UI on http://localhost:8031
```

### Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | No | — | PostgreSQL connection string. If unset, uses file-based `boards/` storage. |
| `PORT` | No | `8080` | HTTP listen port inside the container |
| `OIDC_ISSUER_URL` | No | — | Keycloak OIDC issuer URL (e.g. `https://auth.spencerbaumruk.com/realms/master`) |
| `OIDC_CLIENT_ID` | No | — | Keycloak OIDC client ID (e.g. `scrabble`) |
| `VITE_OIDC_AUTHORITY` | No | `https://auth.spencerbaumruk.com/realms/master` | Frontend OIDC authority (build-time) |
| `VITE_OIDC_CLIENT_ID` | No | `scrabble` | Frontend OIDC client ID (build-time) |

### Testing

There is no test suite. The AI simulation mode (`./scrabble`) serves as an integration-level check — if moves are generated and scored correctly through a full game, the engine is working.

## File layout

```
/
├── boards/          # Saved board files (shared; symlinked from go/boards)
├── docs/            # Detailed documentation (see index above)
├── CLAUDE.md        # This file
├── README.md        # Project readme
├── go/              # All Go source and runtime data
│   ├── main.go          # Entry point; dispatches to runGame, runSolve, runServer, or runMigrateBoards
│   ├── common.go        # Shared engine: Board/Trie, scoring, searchPlay, getPlaySpace
│   ├── scrabble.go      # AI vs AI game loop (NewBoard, DoTurn, runGame)
│   ├── solve.go         # Interactive solver UI, findTopNMoves, terminal rendering
│   ├── server.go        # HTTP server, JSON API handlers, static file serving, DB/file routing
│   ├── db.go            # PostgreSQL connection, migration, board CRUD with ownership
│   ├── auth.go          # OIDC token verification, auth middleware, /api/me endpoint
│   ├── go.mod           # Go module file (pgx/v5 dependency)
│   ├── go.sum           # Go dependency checksums
│   ├── dictionary.txt   # 178K-word dictionary (required at runtime)
│   ├── rulesets.json    # Ruleset definitions (NYT Crossplay, Standard Scrabble)
│   ├── config.json      # Active ruleset selection (optional; defaults to NYT Crossplay)
│   ├── static/          # Embedded SvelteKit build (populated by web build)
│   └── boards -> ../boards  # Symlink to root boards/
└── web/             # SvelteKit frontend (TypeScript + Svelte 5)
    ├── svelte.config.js     # adapter-static, SPA fallback
    ├── vite.config.ts       # Vite config
    ├── package.json
    ├── src/
    │   ├── app.html
    │   ├── app.css          # CSS custom properties, light/dark themes
    │   ├── lib/
    │   │   ├── types.ts     # Move, Ruleset, BoardMeta, BoardRecord interfaces
    │   │   ├── api.ts       # Fetch wrappers for all API endpoints (auto-attaches Bearer token)
    │   │   ├── auth.ts      # OIDC client (oidc-client-ts): login, logout, token management
    │   │   └── components/
    │   │       ├── Board.svelte      # 15×15 CSS grid board
    │   │       ├── MoveList.svelte   # Scrollable move list with selection
    │   │       └── RackInput.svelte  # Tile input with visual slots
    │   └── routes/
    │       ├── +layout.svelte  # App shell: header, auth UI, dark mode toggle
    │       ├── +layout.ts      # SSR disabled
    │       ├── +page.svelte    # Board picker (auth-aware: boards if logged in, login prompt if not)
    │       ├── auth/callback/
    │       │   └── +page.svelte  # OIDC redirect callback handler
    │       └── game/
    │           └── +page.svelte  # Main game view (solve + opponent, shared boards read-only)
    └── build/               # Production build output (gitignored)
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

**Board Storage (`db.go` / file-based):**
- If `DATABASE_URL` is set: boards stored in PostgreSQL (`boards` table) with UUID primary keys, per-user ownership (`user_id`), and optional share tokens for public read-only links.
- If `DATABASE_URL` is not set: falls back to file-based storage in `boards/*.txt` (original behavior, used for local dev and CLI modes).
- The `solve` and `runGame` CLI commands always use file-based storage.
- API endpoints use UUID-based board IDs when DB-backed, name-based when file-backed.

**Authentication (`auth.go` / `auth.ts`):**
- Backend: If `OIDC_ISSUER_URL` + `OIDC_CLIENT_ID` env vars are set, OIDC is active. The server performs JWKS discovery on startup and validates JWT access tokens on each API request.
- Board mutations (create, save, delete, share) require authentication when OIDC is configured. Solver/ruleset endpoints are always public. Board list returns empty for unauthenticated users.
- Frontend: Uses `oidc-client-ts` with Authorization Code + PKCE flow. Login redirects to Keycloak, callback handled at `/auth/callback`. Access tokens stored in localStorage and auto-renewed. All `fetchJSON` calls include `Authorization: Bearer` header when a token is available.
- Keycloak setup: Public OIDC client `scrabble` in the master realm. Valid redirect URIs include the production, Tailscale, and localhost origins.
