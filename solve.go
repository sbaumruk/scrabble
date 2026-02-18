package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"unsafe"
)

// ── Terminal ──────────────────────────────────────────────────────────────────

var origTermios syscall.Termios

func initTerminal() {
	syscall.Syscall(syscall.SYS_IOCTL, 0, syscall.TIOCGETA,
		uintptr(unsafe.Pointer(&origTermios)))
}

func enableRaw() {
	raw := origTermios
	raw.Lflag &^= syscall.ECHO | syscall.ICANON
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0
	syscall.Syscall(syscall.SYS_IOCTL, 0, syscall.TIOCSETA,
		uintptr(unsafe.Pointer(&raw)))
	fmt.Print("\x1b[?25l") // hide cursor
}

func disableRaw() {
	syscall.Syscall(syscall.SYS_IOCTL, 0, syscall.TIOCSETA,
		uintptr(unsafe.Pointer(&origTermios)))
	fmt.Print("\x1b[?25h") // show cursor
}

// ── Key input ─────────────────────────────────────────────────────────────────

type key int

const (
	keyUp key = iota
	keyDown
	keyEnter
	keyQ
	keyOther
)

func readKey() key {
	var buf [3]byte
	n, _ := os.Stdin.Read(buf[:1])
	if n == 0 {
		return keyOther
	}
	switch buf[0] {
	case '\r', '\n':
		return keyEnter
	case 'q', 'Q':
		return keyQ
	case 0x03: // Ctrl+C
		disableRaw()
		fmt.Println()
		os.Exit(0)
	case 0x1b:
		n, _ = os.Stdin.Read(buf[1:3])
		if n == 2 && buf[1] == '[' {
			switch buf[2] {
			case 'A':
				return keyUp
			case 'B':
				return keyDown
			}
		}
	}
	return keyOther
}

// ── Board rendering ───────────────────────────────────────────────────────────

func buildBoardLines(b *Board, highlight map[int]bool) []string {
	lines := make([]string, 15)
	for y := 0; y < 15; y++ {
		var sb strings.Builder
		for x := 0; x < 15; x++ {
			idx := cti(x, y)
			if b.board[x][y] == 0 {
				switch {
				case dw[idx]:
					sb.WriteString("\x1b[31;1m")
				case tw[idx]:
					sb.WriteString("\x1b[33;1m")
				case dl[idx]:
					sb.WriteString("\x1b[34;1m")
				case tl[idx]:
					sb.WriteString("\x1b[32;1m")
				}
				sb.WriteByte('.')
			} else {
				if highlight[idx] {
					sb.WriteString("\x1b[42;1m")
				}
				sb.WriteByte(b.board[x][y])
			}
			sb.WriteString("\x1b[0m ")
		}
		lines[y] = sb.String()
	}
	return lines
}

// previewMove returns a deep copy of b's board with m applied, plus the set of
// newly-placed positions. The original board is not modified.
func previewMove(b *Board, m BestMove) ([][]byte, map[int]bool) {
	board := make([][]byte, 15)
	for i := range board {
		board[i] = make([]byte, 15)
		copy(board[i], b.board[i])
	}
	h := make(map[int]bool)
	tiles := m.tiles
	if m.dir == DIR_VERT {
		for i := m.y; len(tiles) > 0; i++ {
			if board[m.x][i] != 0 {
				continue
			}
			board[m.x][i] = tiles[0]
			h[cti(m.x, i)] = true
			tiles = tiles[1:]
		}
	} else {
		for i := m.x; len(tiles) > 0; i++ {
			if board[i][m.y] != 0 {
				continue
			}
			board[i][m.y] = tiles[0]
			h[cti(i, m.y)] = true
			tiles = tiles[1:]
		}
	}
	return board, h
}

// ── Side-by-side UI ───────────────────────────────────────────────────────────

const leftWidth = 30

// renderSideBySide clears the screen and prints leftLines (plain text, padded
// to leftWidth) next to rightLines (ANSI board). selIdx highlights one left row.
func renderSideBySide(header string, leftLines []string, selIdx int, rightLines []string) {
	fmt.Print("\x1b[2J\x1b[H")
	fmt.Print(header + "\r\n\r\n")

	rows := len(leftLines)
	if len(rightLines) > rows {
		rows = len(rightLines)
	}
	for i := 0; i < rows; i++ {
		var left string
		if i < len(leftLines) {
			s := leftLines[i]
			if len(s) > leftWidth {
				s = s[:leftWidth]
			}
			// Pad to exact leftWidth BEFORE applying ANSI (fmt.Sprintf counts bytes, not columns).
			padded := fmt.Sprintf("%-*s", leftWidth, s)
			if i == selIdx {
				left = "\x1b[7m" + padded + "\x1b[0m"
			} else {
				left = padded
			}
		} else {
			left = strings.Repeat(" ", leftWidth)
		}
		right := ""
		if i < len(rightLines) {
			right = rightLines[i]
		}
		fmt.Print(left + " | " + right + "\r\n")
	}
}

// ── Board file I/O ────────────────────────────────────────────────────────────

func parseBoardFile(path string) ([][]byte, error) {
	board := make([][]byte, 15)
	for i := range board {
		board[i] = make([]byte, 15)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for y := 0; y < 15; y++ {
		line, _, err := r.ReadLine()
		if err != nil {
			break
		}
		for x := 0; x < 15 && x < len(line); x++ {
			c := line[x]
			if c != '.' {
				board[x][y] = c &^ byte(32) // uppercase
			}
		}
	}
	return board, nil
}

func saveBoard(board [][]byte, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for y := 0; y < 15; y++ {
		for x := 0; x < 15; x++ {
			if board[x][y] == 0 {
				w.WriteByte('.')
			} else {
				w.WriteByte(board[x][y])
			}
		}
		w.WriteByte('\n')
	}
	return w.Flush()
}

func createBlankBoard(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for y := 0; y < 15; y++ {
		for x := 0; x < 15; x++ {
			w.WriteByte('.')
		}
		w.WriteByte('\n')
	}
	return w.Flush()
}

// ── Move finding ──────────────────────────────────────────────────────────────

// fullWord reconstructs the complete word formed by m, including tiles already
// on the board before and after the new tiles.
func fullWord(b *Board, m BestMove) string {
	var sb strings.Builder
	tileIdx := 0
	if m.dir == DIR_VERT {
		startY := m.y
		for startY > 0 && b.board[m.x][startY-1] != 0 {
			startY--
		}
		for i := startY; i < 15; i++ {
			if b.board[m.x][i] != 0 {
				sb.WriteByte(b.board[m.x][i])
			} else if tileIdx < len(m.tiles) {
				sb.WriteByte(m.tiles[tileIdx])
				tileIdx++
			} else {
				break
			}
		}
	} else {
		startX := m.x
		for startX > 0 && b.board[startX-1][m.y] != 0 {
			startX--
		}
		for i := startX; i < 15; i++ {
			if b.board[i][m.y] != 0 {
				sb.WriteByte(b.board[i][m.y])
			} else if tileIdx < len(m.tiles) {
				sb.WriteByte(m.tiles[tileIdx])
				tileIdx++
			} else {
				break
			}
		}
	}
	return strings.ToUpper(sb.String())
}

type BestMove struct {
	x, y  int
	dir   direction
	tiles string
	score int
}

// findTopNMoves finds all valid moves for rack, deduplicates by visual placement,
// sorts by score descending, and returns the top n.
func (b *Board) findTopNMoves(rack []byte, n int) []BestMove {
	var moves []BestMove
	seen := make(map[string]bool)
	tilesets := permute(rack)
	rackLen := len(rack)

	for x := 0; x < 15; x++ {
		for y := 0; y < 15; y++ {
			if b.board[x][y] != 0 {
				continue
			}
			for _, dir := range []direction{DIR_HORIZ, DIR_VERT} {
				play, crossPlays, room := b.getPlaySpace(x, y, dir)
				if room > rackLen {
					room = rackLen
				}
				for tilecount := 1; tilecount <= room; tilecount++ {
					if !b.checkCenterPlayed(x, y, tilecount, dir) || !b.checkContiguous(x, y, tilecount, dir) {
						continue
					}
				TILESETLIST:
					for _, tileset := range tilesets {
						if len(tileset) != tilecount {
							continue
						}
						f := NewFNV()
						j := 0
						for i, v := range play {
							if v == 0 {
								if j == len(tileset) {
									break
								}
								f.Add(tileset[j])
								if crossPlays[i] != nil {
									f2 := NewFNV()
									for _, v2 := range crossPlays[i] {
										if v2 == 0 {
											f2.Add(tileset[j])
										} else {
											f2.Add(v2)
										}
									}
									if _, ok := b.wordlist[f2.Val()]; !ok {
										continue TILESETLIST
									}
								}
								j++
							} else {
								f.Add(v)
							}
						}
						if _, ok := b.wordlist[f.Val()]; !ok {
							continue TILESETLIST
						}
						score := b.scoreMove(x, y, tileset, dir)
						if rackLen == 7 && len(tileset) == 7 {
							score += bingoBonus
						}
						// Deduplicate on position + direction + uppercase tiles
						key := fmt.Sprintf("%d,%d,%d,%s", x, y, int(dir), strings.ToUpper(tileset))
						if !seen[key] {
							seen[key] = true
							moves = append(moves, BestMove{x: x, y: y, dir: dir, tiles: tileset, score: score})
						}
					}
				}
			}
		}
	}

	sort.Slice(moves, func(i, j int) bool {
		return moves[i].score > moves[j].score
	})
	if len(moves) > n {
		moves = moves[:n]
	}
	return moves
}

// findOpponentPlacements finds all valid board positions where word could have
// been played. Returns moves where x,y is the first NEW tile position and tiles
// contains only the letters that weren't already on the board.
func (b *Board) findOpponentPlacements(word string) []BestMove {
	word = strings.ToUpper(word)
	n := len(word)
	var placements []BestMove

	for _, dir := range []direction{DIR_HORIZ, DIR_VERT} {
		for startX := 0; startX < 15; startX++ {
			for startY := 0; startY < 15; startY++ {
				// Check word fits on board
				if dir == DIR_HORIZ && startX+n > 15 {
					continue
				}
				if dir == DIR_VERT && startY+n > 15 {
					continue
				}

				// Check no tile immediately before the word
				if dir == DIR_HORIZ && startX > 0 && b.board[startX-1][startY] != 0 {
					continue
				}
				if dir == DIR_VERT && startY > 0 && b.board[startX][startY-1] != 0 {
					continue
				}

				// Check no tile immediately after the word
				if dir == DIR_HORIZ && startX+n < 15 && b.board[startX+n][startY] != 0 {
					continue
				}
				if dir == DIR_VERT && startY+n < 15 && b.board[startX][startY+n] != 0 {
					continue
				}

				// Scan word positions: check for conflicts, collect new tiles
				valid := true
				newTiles := ""
				firstNewX, firstNewY := -1, -1
				touches := false

				for i := 0; i < n; i++ {
					var bx, by int
					if dir == DIR_HORIZ {
						bx, by = startX+i, startY
					} else {
						bx, by = startX, startY+i
					}
					if b.board[bx][by] != 0 {
						if b.board[bx][by] != word[i] {
							valid = false
							break
						}
						touches = true // using an existing tile counts as connected
					} else {
						newTiles += string(word[i])
						if firstNewX == -1 {
							firstNewX, firstNewY = bx, by
						}
						// Check orthogonal neighbors for connectivity
						if bx > 0 && b.board[bx-1][by] != 0 {
							touches = true
						}
						if bx < 14 && b.board[bx+1][by] != 0 {
							touches = true
						}
						if by > 0 && b.board[bx][by-1] != 0 {
							touches = true
						}
						if by < 14 && b.board[bx][by+1] != 0 {
							touches = true
						}
					}
				}

				if !valid || len(newTiles) == 0 {
					continue
				}

				// Must connect to existing tiles (unless this is the very first word)
				if b.board[7][7] != 0 && !touches {
					continue
				}

				// Validate cross-words formed by each new tile
				crossValid := true
				for i := 0; i < n; i++ {
					var bx, by int
					if dir == DIR_HORIZ {
						bx, by = startX+i, startY
					} else {
						bx, by = startX, startY+i
					}
					if b.board[bx][by] != 0 {
						continue // existing tile, no new cross-word here
					}
					c := word[i]
					if dir == DIR_HORIZ {
						// Cross direction is vertical
						cy1, cy2 := by, by
						for cy1 > 0 && b.board[bx][cy1-1] != 0 {
							cy1--
						}
						for cy2 < 14 && b.board[bx][cy2+1] != 0 {
							cy2++
						}
						if cy1 < by || cy2 > by { // touches existing tiles vertically
							f := NewFNV()
							for j := cy1; j <= cy2; j++ {
								if j == by {
									f.Add(c)
								} else {
									f.Add(b.board[bx][j])
								}
							}
							if _, ok := b.wordlist[f.Val()]; !ok {
								crossValid = false
								break
							}
						}
					} else {
						// Cross direction is horizontal
						cx1, cx2 := bx, bx
						for cx1 > 0 && b.board[cx1-1][by] != 0 {
							cx1--
						}
						for cx2 < 14 && b.board[cx2+1][by] != 0 {
							cx2++
						}
						if cx1 < bx || cx2 > bx { // touches existing tiles horizontally
							f := NewFNV()
							for j := cx1; j <= cx2; j++ {
								if j == bx {
									f.Add(c)
								} else {
									f.Add(b.board[j][by])
								}
							}
							if _, ok := b.wordlist[f.Val()]; !ok {
								crossValid = false
								break
							}
						}
					}
				}
				if !crossValid {
					continue
				}

				// First word must cover the center square
				if b.board[7][7] == 0 {
					coversCentre := false
					for i := 0; i < n; i++ {
						var bx, by int
						if dir == DIR_HORIZ {
							bx, by = startX+i, startY
						} else {
							bx, by = startX, startY+i
						}
						if bx == 7 && by == 7 {
							coversCentre = true
							break
						}
					}
					if !coversCentre {
						continue
					}
				}

				score := b.scoreMove(firstNewX, firstNewY, newTiles, dir)
				placements = append(placements, BestMove{
					x: firstNewX, y: firstNewY,
					dir: dir, tiles: newTiles, score: score,
				})
			}
		}
	}
	return placements
}

// ── Screen: board picker ──────────────────────────────────────────────────────

// boardPickerScreen shows the boards/ directory with a "+ New board" option.
// Manages raw mode internally. Returns the selected file path and whether to proceed.
func boardPickerScreen(reader *bufio.Reader) (string, bool) {
	loadFiles := func() []string {
		var files []string
		entries, err := os.ReadDir("boards")
		if err != nil {
			return files
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".txt") {
				files = append(files, e.Name())
			}
		}
		return files
	}

	files := loadFiles()
	sel := 0

	for {
		totalItems := len(files) + 1 // files + "+ New board"

		displayLines := make([]string, totalItems)
		for i, f := range files {
			name := strings.TrimSuffix(f, ".txt")
			if len(name) > leftWidth-4 {
				name = name[:leftWidth-7] + "..."
			}
			displayLines[i] = "  " + name
		}
		displayLines[len(files)] = "  + New board"

		previews := make([][]string, totalItems)
		for i, f := range files {
			if board, err := parseBoardFile("boards/" + f); err == nil {
				previews[i] = buildBoardLines(&Board{board: board}, nil)
			} else {
				previews[i] = make([]string, 15)
			}
		}
		previews[len(files)] = make([]string, 15) // blank preview for new board

		if sel >= totalItems {
			sel = totalItems - 1
		}

		renderSideBySide(
			"Select a board  (\x1b[1m\xe2\x86\x91\xe2\x86\x93\x1b[0m navigate, \x1b[1mEnter\x1b[0m select, \x1b[1mq\x1b[0m quit)",
			displayLines, sel, previews[sel],
		)

		switch readKey() {
		case keyUp:
			if sel > 0 {
				sel--
			}
		case keyDown:
			if sel < totalItems-1 {
				sel++
			}
		case keyEnter:
			if sel == len(files) {
				// Create a new board
				disableRaw()
				fmt.Print("\x1b[2J\x1b[H")
				fmt.Print("New board name: ")
				name, _ := reader.ReadString('\n')
				name = strings.TrimSpace(strings.TrimRight(name, "\r\n"))
				if name != "" {
					path := "boards/" + name + ".txt"
					if err := createBlankBoard(path); err != nil {
						fmt.Printf("Error creating board: %v\n", err)
					} else {
						fmt.Printf("Created %s\n", path)
					}
					files = loadFiles()
					// Select the newly created file
					target := name + ".txt"
					for i, f := range files {
						if f == target {
							sel = i
							break
						}
					}
				}
				enableRaw()
				// Continue outer loop → re-render picker
			} else {
				disableRaw()
				return "boards/" + files[sel], true
			}
		case keyQ:
			disableRaw()
			return "", false
		}
	}
}

// ── Screen: move / placement picker ──────────────────────────────────────────

// movePickerScreen shows a list of moves with a live board preview.
// header should include tile/context info. Returns (selected index, ok).
func movePickerScreen(b *Board, moves []BestMove, header string) (int, bool) {
	sel := 0
	for {
		leftLines := make([]string, len(moves))
		for i, m := range moves {
			dirStr := "H"
			if m.dir == DIR_VERT {
				dirStr = "V"
			}
			word := fullWord(b, m)
			leftLines[i] = fmt.Sprintf("  %d. %-7s%4dpts (%2d,%2d) %s",
				i+1, word, m.score, m.x+1, m.y+1, dirStr)
		}

		previewBoard, highlight := previewMove(b, moves[sel])
		rightLines := buildBoardLines(&Board{board: previewBoard}, highlight)

		renderSideBySide(header, leftLines, sel, rightLines)

		switch readKey() {
		case keyUp:
			if sel > 0 {
				sel--
			}
		case keyDown:
			if sel < len(moves)-1 {
				sel++
			}
		case keyEnter:
			return sel, true
		case keyQ:
			return 0, false
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func applyMove(b *Board, m BestMove) {
	tiles := m.tiles
	if m.dir == DIR_VERT {
		for i := m.y; len(tiles) > 0; i++ {
			if b.board[m.x][i] != 0 {
				continue
			}
			b.board[m.x][i] = tiles[0]
			tiles = tiles[1:]
		}
	} else {
		for i := m.x; len(tiles) > 0; i++ {
			if b.board[i][m.y] != 0 {
				continue
			}
			b.board[i][m.y] = tiles[0]
			tiles = tiles[1:]
		}
	}
}

func parseRack(input string) []byte {
	input = strings.TrimRight(input, "\r\n ")
	rack := make([]byte, 0, len(input))
	for i := 0; i < len(input); i++ {
		if input[i] == '*' {
			rack = append(rack, '*')
		} else {
			rack = append(rack, input[i]&^byte(32))
		}
	}
	return rack
}

// promptSave asks whether to save (default yes). Once the user says yes,
// autoSave is set to true and subsequent calls save silently without asking.
func promptSave(reader *bufio.Reader, board [][]byte, path string, autoSave *bool) {
	if *autoSave {
		if err := saveBoard(board, path); err != nil {
			fmt.Printf("Error saving: %v\n", err)
		}
		return
	}
	fmt.Print("Save board? [Y/n]: ")
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(ans)
	if ans == "" || strings.HasPrefix(strings.ToLower(ans), "y") {
		*autoSave = true
		if err := saveBoard(board, path); err != nil {
			fmt.Printf("Error saving: %v\n", err)
		} else {
			fmt.Println("Saved.")
		}
	}
}

// ── Main ──────────────────────────────────────────────────────────────────────

func runSolve() {
	initTerminal()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	go func() {
		<-sigCh
		disableRaw()
		fmt.Println()
		os.Exit(0)
	}()

	ruleset := loadRuleset()

	wordlist, err := loadDictionary("dictionary.txt")
	if err != nil {
		fmt.Println("Unable to open dictionary:", err)
		return
	}

	if err := os.MkdirAll("boards", 0755); err != nil {
		fmt.Println("Cannot create boards/ directory:", err)
		return
	}

	reader := bufio.NewReader(os.Stdin)

	// Pick (or create) a board
	enableRaw()
	boardFile, ok := boardPickerScreen(reader) // disables raw before returning
	if !ok {
		return
	}

	boardData, err := parseBoardFile(boardFile)
	if err != nil {
		fmt.Println("Failed to load board:", err)
		return
	}
	b := &Board{board: boardData, wordlist: wordlist}

	// Ask whose turn is first
	fmt.Print("\x1b[2J\x1b[H")
	for _, line := range buildBoardLines(b, nil) {
		fmt.Println(line)
	}
	fmt.Printf("\nBoard: %s | Ruleset: %s\n", boardFile, ruleset)
	fmt.Print("Whose turn is it first? [M]ine / [O]pponent's: ")
	firstInput, _ := reader.ReadString('\n')
	skipMyTurn := strings.HasPrefix(strings.TrimSpace(strings.ToLower(firstInput)), "o")

	autoSave := false

	// Continuous game loop ────────────────────────────────────────────────────
	for {
		if !skipMyTurn {
			// Show current board and prompt for rack
			fmt.Print("\x1b[2J\x1b[H")
			for _, line := range buildBoardLines(b, nil) {
				fmt.Println(line)
			}
			fmt.Printf("\nBoard: %s\n", boardFile)
			fmt.Print("Your tiles (blank to quit): ")

			input, _ := reader.ReadString('\n')
			rack := parseRack(input)
			if len(rack) == 0 {
				fmt.Println("Goodbye!")
				break
			}

			// Find best moves
			fmt.Printf("Searching for top moves for %s...\n", string(rack))
			moves := b.findTopNMoves(rack, 10)
			if len(moves) == 0 {
				fmt.Println("No valid moves found.")
				fmt.Print("Press Enter to continue...")
				reader.ReadString('\n')
				continue
			}

			// Move picker
			myHeader := fmt.Sprintf(
				"Your tiles: \x1b[1m%s\x1b[0m   (\x1b[1m\xe2\x86\x91\xe2\x86\x93\x1b[0m navigate, \x1b[1mEnter\x1b[0m confirm, \x1b[1mq\x1b[0m back)",
				string(rack))
			enableRaw()
			selMove, ok := movePickerScreen(b, moves, myHeader)
			disableRaw()
			if !ok {
				continue
			}

			// Apply and display my move
			m := moves[selMove]
			dirStr := "horizontal"
			if m.dir == DIR_VERT {
				dirStr = "vertical"
			}
			bonusNote := ""
			if len(rack) == 7 && len(m.tiles) == 7 {
				bonusNote = fmt.Sprintf(" (includes %dpt bingo bonus)", bingoBonus)
			}

			_, highlight := previewMove(b, m)
			applyMove(b, m)

			fmt.Print("\x1b[2J\x1b[H")
			for _, line := range buildBoardLines(b, highlight) {
				fmt.Println(line)
			}
			fmt.Printf("\nPlayed: %s at (%d,%d) %s — %d points%s\n\n",
				fullWord(b, m), m.x+1, m.y+1, dirStr, m.score, bonusNote)

			promptSave(reader, b.board, boardFile, &autoSave)
			fmt.Println()
		}
		skipMyTurn = false

		// Opponent's turn
		fmt.Print("Opponent's word (blank to skip to next turn): ")
		oppInput, _ := reader.ReadString('\n')
		oppWord := strings.TrimSpace(strings.TrimRight(oppInput, "\r\n"))
		if oppWord == "" {
			continue
		}

		placements := b.findOpponentPlacements(oppWord)
		if len(placements) == 0 {
			fmt.Printf("Could not find a valid placement for %q on the board.\n",
				strings.ToUpper(oppWord))
			fmt.Print("Press Enter to continue...")
			reader.ReadString('\n')
			continue
		}

		// Sort placements by score for display
		sort.Slice(placements, func(i, j int) bool {
			return placements[i].score > placements[j].score
		})

		oppHeader := fmt.Sprintf(
			"Where did opponent play \x1b[1m%s\x1b[0m?   (\x1b[1m\xe2\x86\x91\xe2\x86\x93\x1b[0m navigate, \x1b[1mEnter\x1b[0m confirm, \x1b[1mq\x1b[0m skip)",
			strings.ToUpper(oppWord))
		enableRaw()
		selOpp, ok := movePickerScreen(b, placements, oppHeader)
		disableRaw()
		if !ok {
			continue
		}

		// Apply opponent's move
		oppM := placements[selOpp]
		_, oppHighlight := previewMove(b, oppM)
		applyMove(b, oppM)

		fmt.Print("\x1b[2J\x1b[H")
		for _, line := range buildBoardLines(b, oppHighlight) {
			fmt.Println(line)
		}
		fmt.Printf("\nOpponent played %s\n\n", strings.ToUpper(oppWord))

		promptSave(reader, b.board, boardFile, &autoSave)
		fmt.Println()
	}
}
