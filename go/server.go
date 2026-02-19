package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

// ── API response types ───────────────────────────────────────────────────────

type MoveResponse struct {
	X            int      `json:"x"`
	Y            int      `json:"y"`
	Dir          string   `json:"dir"`
	Tiles        string   `json:"tiles"`
	Word         string   `json:"word"`
	Score        int      `json:"score"`
	NewPositions [][2]int `json:"newPositions"`
}

type RulesetResponse struct {
	Name         string         `json:"name"`
	BingoBonus   int            `json:"bingoBonus"`
	LetterPoints map[string]int `json:"letterPoints"`
	TripleWord   [][2]int       `json:"tripleWord"`
	DoubleWord   [][2]int       `json:"doubleWord"`
	TripleLetter [][2]int       `json:"tripleLetter"`
	DoubleLetter [][2]int       `json:"doubleLetter"`
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func boardToStrings(board [][]byte) []string {
	rows := make([]string, 15)
	for y := 0; y < 15; y++ {
		var sb strings.Builder
		for x := 0; x < 15; x++ {
			if board[x][y] == 0 {
				sb.WriteByte('.')
			} else {
				sb.WriteByte(board[x][y])
			}
		}
		rows[y] = sb.String()
	}
	return rows
}

func stringsToBoard(rows []string) [][]byte {
	board := make([][]byte, 15)
	for i := range board {
		board[i] = make([]byte, 15)
	}
	for y := 0; y < 15 && y < len(rows); y++ {
		for x := 0; x < 15 && x < len(rows[y]); x++ {
			c := rows[y][x]
			if c != '.' {
				board[x][y] = c
			}
		}
	}
	return board
}

func bestMoveToResponse(b *Board, m BestMove) MoveResponse {
	dirStr := "H"
	if m.dir == DIR_VERT {
		dirStr = "V"
	}
	word := fullWord(b, m)

	var newPos [][2]int
	tileIdx := 0
	if m.dir == DIR_VERT {
		for i := m.y; tileIdx < len(m.tiles); i++ {
			if b.board[m.x][i] != 0 {
				continue
			}
			newPos = append(newPos, [2]int{m.x, i})
			tileIdx++
		}
	} else {
		for i := m.x; tileIdx < len(m.tiles); i++ {
			if b.board[i][m.y] != 0 {
				continue
			}
			newPos = append(newPos, [2]int{i, m.y})
			tileIdx++
		}
	}

	return MoveResponse{
		X:            m.x,
		Y:            m.y,
		Dir:          dirStr,
		Tiles:        m.tiles,
		Word:         word,
		Score:        m.score,
		NewPositions: newPos,
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func handleListBoards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "method not allowed")
		return
	}
	entries, err := os.ReadDir("boards")
	if err != nil {
		writeJSON(w, 200, map[string][]string{"boards": {}})
		return
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".txt") {
			names = append(names, strings.TrimSuffix(e.Name(), ".txt"))
		}
	}
	if names == nil {
		names = []string{}
	}
	writeJSON(w, 200, map[string][]string{"boards": names})
}

func handleGetBoard(w http.ResponseWriter, r *http.Request, name string) {
	path := filepath.Join("boards", name+".txt")
	board, err := parseBoardFile(path)
	if err != nil {
		writeError(w, 404, "board not found")
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"name":  name,
		"board": boardToStrings(board),
	})
}

func handleSaveBoard(w http.ResponseWriter, r *http.Request, name string) {
	var req struct {
		Board []string `json:"board"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if len(req.Board) != 15 {
		writeError(w, 400, "board must have 15 rows")
		return
	}
	board := stringsToBoard(req.Board)
	path := filepath.Join("boards", name+".txt")
	if err := saveBoard(board, path); err != nil {
		writeError(w, 500, "failed to save board")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func handleCreateBoard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, 400, "name is required")
		return
	}
	path := filepath.Join("boards", req.Name+".txt")
	if err := createBlankBoard(path); err != nil {
		writeError(w, 500, "failed to create board")
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}

func handleSolve(wordlist map[uint64]struct{}, trie *TrieNode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, 405, "method not allowed")
			return
		}
		var req struct {
			Board []string `json:"board"`
			Rack  string   `json:"rack"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid JSON")
			return
		}
		if len(req.Board) != 15 {
			writeError(w, 400, "board must have 15 rows")
			return
		}
		board := stringsToBoard(req.Board)
		rack := parseRack(req.Rack)

		b := &Board{board: board, wordlist: wordlist, trie: trie}
		moves := b.findTopNMoves(rack, 20)

		results := make([]MoveResponse, len(moves))
		for i, m := range moves {
			results[i] = bestMoveToResponse(b, m)
		}
		writeJSON(w, 200, map[string]interface{}{"moves": results})
	}
}

func handleOpponent(wordlist map[uint64]struct{}, trie *TrieNode) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, 405, "method not allowed")
			return
		}
		var req struct {
			Board []string `json:"board"`
			Word  string   `json:"word"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid JSON")
			return
		}
		if len(req.Board) != 15 {
			writeError(w, 400, "board must have 15 rows")
			return
		}
		board := stringsToBoard(req.Board)

		b := &Board{board: board, wordlist: wordlist, trie: trie}
		placements := b.findOpponentPlacements(req.Word)
		sort.Slice(placements, func(i, j int) bool {
			return placements[i].score > placements[j].score
		})

		results := make([]MoveResponse, len(placements))
		for i, m := range placements {
			results[i] = bestMoveToResponse(b, m)
		}
		writeJSON(w, 200, map[string]interface{}{"placements": results})
	}
}

func handleRuleset(rulesetName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, 405, "method not allowed")
			return
		}

		letterPoints := make(map[string]int)
		for i := byte('A'); i <= 'Z'; i++ {
			if tilePoints[i] > 0 {
				letterPoints[string(i)] = tilePoints[i]
			}
		}

		var tripleWord, doubleWord, tripleLetter, doubleLetter [][2]int
		for i := 0; i < 225; i++ {
			x, y := i%15, i/15
			if tw[i] {
				tripleWord = append(tripleWord, [2]int{x, y})
			}
			if dw[i] {
				doubleWord = append(doubleWord, [2]int{x, y})
			}
			if tl[i] {
				tripleLetter = append(tripleLetter, [2]int{x, y})
			}
			if dl[i] {
				doubleLetter = append(doubleLetter, [2]int{x, y})
			}
		}

		writeJSON(w, 200, RulesetResponse{
			Name:         rulesetName,
			BingoBonus:   bingoBonus,
			LetterPoints: letterPoints,
			TripleWord:   tripleWord,
			DoubleWord:   doubleWord,
			TripleLetter: tripleLetter,
			DoubleLetter: doubleLetter,
		})
	}
}

// ── Server ───────────────────────────────────────────────────────────────────

func runServer() {
	rulesetName := loadRuleset()

	fmt.Println("Loading dictionary...")
	wordlist, err := loadDictionary("dictionary.txt")
	if err != nil {
		fmt.Println("Unable to load dictionary:", err)
		os.Exit(1)
	}

	fmt.Println("Building trie...")
	trie, err := buildTrie("dictionary.txt")
	if err != nil {
		fmt.Println("Unable to build trie:", err)
		os.Exit(1)
	}

	if err := os.MkdirAll("boards", 0755); err != nil {
		fmt.Println("Cannot create boards/ directory:", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/solve", handleSolve(wordlist, trie))
	mux.HandleFunc("/api/opponent", handleOpponent(wordlist, trie))
	mux.HandleFunc("/api/ruleset", handleRuleset(rulesetName))

	mux.HandleFunc("/api/boards", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleListBoards(w, r)
		} else if r.Method == http.MethodPost {
			handleCreateBoard(w, r)
		} else {
			writeError(w, 405, "method not allowed")
		}
	})

	mux.HandleFunc("/api/boards/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/boards/")
		if name == "" {
			writeError(w, 400, "board name required")
			return
		}
		if r.Method == http.MethodGet {
			handleGetBoard(w, r, name)
		} else if r.Method == http.MethodPost {
			handleSaveBoard(w, r, name)
		} else {
			writeError(w, 405, "method not allowed")
		}
	})

	// Static files with SPA fallback
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		fmt.Println("Static files not embedded (run in dev mode with Vite):", err)
		// In dev mode, only serve API routes; Vite handles static files
		staticFS = nil
	}

	if staticFS != nil {
		fileServer := http.FileServer(http.FS(staticFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Try to serve the file directly
			path := r.URL.Path
			if path == "/" {
				path = "/index.html"
			}
			// Check if file exists in embedded FS
			f, err := staticFS.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
			// SPA fallback: serve index.html for non-file paths
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
		})
	}

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	// CORS wrapper around entire mux for dev mode (Vite on :5173 → API on :8080)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(204)
				return
			}
		}
		mux.ServeHTTP(w, r)
	})

	fmt.Printf("Scrabble server running on http://localhost:%s (ruleset: %s)\n", port, rulesetName)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		fmt.Println("Server error:", err)
		os.Exit(1)
	}
}
