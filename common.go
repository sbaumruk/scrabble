package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type FNV struct {
	v uint64
}

func NewFNV() FNV {
	return FNV{v: 0xcbf29ce484222325}
}

func (h *FNV) Add(b byte) {
	h.v *= 0x100000001b3
	h.v ^= uint64(b & ^byte(32))
}

func (h *FNV) AddString(s string) {
	for _, c := range s {
		h.v *= 0x100000001b3
		h.v ^= uint64(c)
	}
}

func (h *FNV) Val() uint64 {
	return h.v
}

// https://en.wikipedia.org/wiki/Scrabble_letter_distributions
var tilePoints = [255]int{'A': 1, 'B': 4, 'C': 3, 'D': 2, 'E': 1, 'F': 4, 'G': 4, 'H': 3, 'I': 1, 'J': 10, 'K': 6, 'L': 2, 'M': 3, 'N': 1, 'O': 1, 'P': 3, 'Q': 10, 'R': 1, 'S': 1, 'T': 1, 'U': 2, 'V': 6, 'W': 5, 'X': 8, 'Y': 4, 'Z': 10}
var startTiles = "AAAAAAAAABBCCDDDDEEEEEEEEEEEEFFGGGHHIIIIIIIIIJKLLLLMMNNNNNNOOOOOOOOPPQRRRRRRSSSSTTTTTTUUUUVVWWXYYZ**"

var bingoBonus = 40 // default: NYT Crossplay; Standard Scrabble uses 50

var tw = [225]bool{3: true, 11: true, 45: true, 59: true, 165: true, 179: true, 213: true, 221: true}
var dw = [225]bool{16: true, 28: true, 52: true, 108: true, 116: true, 172: true, 196: true, 208: true}
var tl = [225]bool{0: true, 14: true, 21: true, 23: true, 65: true, 69: true, 79: true, 85: true, 91: true, 103: true, 121: true, 133: true, 139: true, 145: true, 155: true, 159: true, 201: true, 203: true, 210: true, 224: true}
var dl = [225]bool{7: true, 34: true, 40: true, 48: true, 56: true, 62: true, 72: true, 82: true, 105: true, 110: true, 114: true, 119: true, 142: true, 152: true, 162: true, 168: true, 176: true, 184: true, 190: true, 217: true}

type direction int

var DIR_VERT direction = 0
var DIR_HORIZ direction = 1

type TrieNode struct {
	children [26]*TrieNode
	isEnd    bool
}

type Board struct {
	board    [][]byte
	tiles    []byte
	wordlist map[uint64]struct{}
	trie     *TrieNode
	pscore   [2]int
	ptiles   [2][]byte
}

func cti(x, y int) int {
	return y*15 + x
}

func buildTrie(filename string) (*TrieNode, error) {
	root := &TrieNode{}
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for line, _, err := r.ReadLine(); err == nil; line, _, err = r.ReadLine() {
		word := strings.TrimRight(string(line), "\r\n")
		if len(word) < 2 {
			continue
		}
		node := root
		valid := true
		for i := 0; i < len(word); i++ {
			idx := int(word[i]&^32) - int('A')
			if idx < 0 || idx >= 26 {
				valid = false
				break
			}
			if node.children[idx] == nil {
				node.children[idx] = &TrieNode{}
			}
			node = node.children[idx]
		}
		if valid {
			node.isEnd = true
		}
	}
	return root, nil
}

func (b *Board) addWord(word string) {
	f := NewFNV()
	f.AddString(word)
	b.wordlist[f.Val()] = struct{}{}
}

func loadDictionary(filename string) (map[uint64]struct{}, error) {
	wordlist := make(map[uint64]struct{})
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for line, _, err := r.ReadLine(); err == nil; line, _, err = r.ReadLine() {
		word := strings.TrimRight(string(line), "\r\n")
		if len(word) > 1 {
			h := NewFNV()
			h.AddString(word)
			wordlist[h.Val()] = struct{}{}
		}
	}
	return wordlist, nil
}

func (b *Board) checkCenterPlayed(x, y, tiles int, dir direction) bool {
	if b.board[7][7] != 0 {
		return true
	}
	if dir == DIR_VERT {
		return x == 7 && y <= 7 && (y+tiles) > 7
	}
	return y == 7 && x <= 7 && (x+tiles) > 7
}

func (b *Board) checkContiguous(x, y, tiles int, dir direction) bool {
	if b.board[7][7] == 0 {
		return true
	}
	if dir == DIR_VERT {
		for i := y; tiles > 0; i++ {
			if b.board[x][i] == 0 {
				tiles--
			}
			if (x > 0 && b.board[x-1][i] != 0) || (x < 14 && b.board[x+1][i] != 0) || (i > 0 && b.board[x][i-1] != 0) || (i < 14 && b.board[x][i+1] != 0) {
				return true
			}
		}
	} else {
		for i := x; tiles > 0; i++ {
			if b.board[i][y] == 0 {
				tiles--
			}
			if (i > 0 && b.board[i-1][y] != 0) || (i < 14 && b.board[i+1][y] != 0) || (y > 0 && b.board[i][y-1] != 0) || (y < 14 && b.board[i][y+1] != 0) {
				return true
			}
		}
	}
	return false
}

func (b *Board) scoreWord(x, y int, dir direction, plays []byte) int {
	points := 0
	wordMult := 1
	var x2, y2 int
	wordLen := 0

	if dir == DIR_VERT {
		for y2 = y; y2 > 0 && (plays[cti(x, y2-1)] != 0 || b.board[x][y2-1] != 0); y2-- {
		}
		for ; y2 < 15; y2++ {
			idx := cti(x, y2)
			if b.board[x][y2] != 0 {
				wordLen++
				points += tilePoints[b.board[x][y2]]
			} else if plays[idx] != 0 {
				wordLen++
				points += tilePoints[plays[idx]]
				if dw[idx] {
					wordMult *= 2
				} else if tw[idx] {
					wordMult *= 3
				} else if dl[idx] {
					points += tilePoints[plays[idx]]
				} else if tl[idx] {
					points += tilePoints[plays[idx]] * 2
				}
			} else {
				break
			}
		}
	} else {
		for x2 = x; x2 > 0 && (plays[cti(x2-1, y)] != 0 || b.board[x2-1][y] != 0); x2-- {
		}
		for ; x2 < 15; x2++ {
			idx := cti(x2, y)
			if b.board[x2][y] != 0 {
				wordLen++
				points += tilePoints[b.board[x2][y]]
			} else if plays[idx] != 0 {
				wordLen++
				points += tilePoints[plays[idx]]
				if dw[idx] {
					wordMult *= 2
				} else if tw[idx] {
					wordMult *= 3
				} else if dl[idx] {
					points += tilePoints[plays[idx]]
				} else if tl[idx] {
					points += tilePoints[plays[idx]] * 2
				}
			} else {
				break
			}
		}
	}
	if wordLen == 1 {
		return 0
	}
	return points * wordMult
}

func (b *Board) scoreMove(x, y int, tiles string, dir direction) int {
	playPoints := 0
	tilei := 0
	plays := make([]byte, 225)

	if dir == DIR_VERT {
		for i := y; len(tiles) > tilei; i++ {
			if b.board[x][i] == 0 {
				plays[cti(x, i)] = tiles[tilei]
				tilei++
				playPoints += b.scoreWord(x, i, DIR_HORIZ, plays)
			}
		}
	} else {
		for i := x; len(tiles) > tilei; i++ {
			if b.board[i][y] == 0 {
				plays[cti(i, y)] = tiles[tilei]
				tilei++
				playPoints += b.scoreWord(i, y, DIR_VERT, plays)
			}
		}
	}
	return playPoints + b.scoreWord(x, y, dir, plays)
}

func (b *Board) getPlaySpace(x, y int, dir direction) (startX, startY int, play []byte, crossPlays [][]byte, room int) {
	play = make([]byte, 0)
	crossPlays = make([][]byte, 0)
	if dir == DIR_VERT {
		for y > 0 && b.board[x][y-1] != 0 {
			y--
		}
		startX, startY = x, y
		for i := y; i < 15; i++ {
			play = append(play, b.board[x][i])
			var crossPlay []byte
			if b.board[x][i] == 0 {
				room++
				x2, x3 := x, x
				for x2 > 0 && b.board[x2-1][i] != 0 {
					x2--
				}
				for x3 < 14 && b.board[x3+1][i] != 0 {
					x3++
				}
				if x2 < x3 {
					for j := x2; j <= x3; j++ {
						crossPlay = append(crossPlay, b.board[j][i])
					}
				}
			}
			crossPlays = append(crossPlays, crossPlay)
		}
	} else {
		for x > 0 && b.board[x-1][y] != 0 {
			x--
		}
		startX, startY = x, y
		for i := x; i < 15; i++ {
			play = append(play, b.board[i][y])
			var crossPlay []byte
			if b.board[i][y] == 0 {
				room++
				y2, y3 := y, y
				for y2 > 0 && b.board[i][y2-1] != 0 {
					y2--
				}
				for y3 < 14 && b.board[i][y3+1] != 0 {
					y3++
				}
				if y2 < y3 {
					for j := y2; j <= y3; j++ {
						crossPlay = append(crossPlay, b.board[i][j])
					}
				}
			}
			crossPlays = append(crossPlays, crossPlay)
		}
	}
	return
}

func (b *Board) recordMove(placed []byte, anchorX, anchorY int, dir direction,
	rackLen int, seen map[string]bool, moves *[]BestMove) {
	if !b.checkCenterPlayed(anchorX, anchorY, len(placed), dir) {
		return
	}
	if !b.checkContiguous(anchorX, anchorY, len(placed), dir) {
		return
	}
	score := b.scoreMove(anchorX, anchorY, string(placed), dir)
	if rackLen == 7 && len(placed) == 7 {
		score += bingoBonus
	}
	key := fmt.Sprintf("%d,%d,%d,%s", anchorX, anchorY, int(dir), strings.ToUpper(string(placed)))
	if !seen[key] {
		seen[key] = true
		*moves = append(*moves, BestMove{x: anchorX, y: anchorY, dir: dir, tiles: string(placed), score: score})
	}
}

func (b *Board) searchPlay(node *TrieNode, play []byte, crossPlays [][]byte,
	playIdx int, rack []byte, placed []byte,
	anchorX, anchorY int, dir direction,
	rackLen int, seen map[string]bool, moves *[]BestMove) {
	// Can we record a word here? Only if the next position is not an existing tile we must include.
	canStop := playIdx >= len(play) || play[playIdx] == 0
	if canStop && node.isEnd && len(placed) > 0 {
		b.recordMove(placed, anchorX, anchorY, dir, rackLen, seen, moves)
	}
	if playIdx >= len(play) || len(rack) == 0 {
		return
	}

	curr := play[playIdx]
	if curr != 0 {
		// Existing tile on board: must follow this trie edge.
		idx := int(curr&^32) - int('A')
		if idx >= 0 && idx < 26 && node.children[idx] != nil {
			b.searchPlay(node.children[idx], play, crossPlays, playIdx+1, rack, placed,
				anchorX, anchorY, dir, rackLen, seen, moves)
		}
		return
	}

	// Empty slot: try placing each rack tile here.
	var tried [26]bool
	for rackIdx := 0; rackIdx < len(rack); rackIdx++ {
		t := rack[rackIdx]
		isWild := t == '*'
		for letter := byte('A'); letter <= 'Z'; letter++ {
			if !isWild && (t&^32) != letter {
				continue
			}
			if tried[letter-'A'] {
				continue
			}
			child := node.children[letter-'A']
			if child == nil {
				continue
			}
			// Cross-word check via FNV wordlist.
			if crossPlays[playIdx] != nil {
				f2 := NewFNV()
				for _, v := range crossPlays[playIdx] {
					if v == 0 {
						f2.Add(letter)
					} else {
						f2.Add(v)
					}
				}
				if _, ok := b.wordlist[f2.Val()]; !ok {
					continue
				}
			}
			tried[letter-'A'] = true
			stored := letter
			if isWild {
				stored = letter + 32 // lowercase = blank tile on board
			}
			// Remove tile from rack (swap to end, shrink).
			rack[rackIdx], rack[len(rack)-1] = rack[len(rack)-1], rack[rackIdx]
			rack = rack[:len(rack)-1]
			placed = append(placed, stored)
			b.searchPlay(child, play, crossPlays, playIdx+1, rack, placed,
				anchorX, anchorY, dir, rackLen, seen, moves)
			placed = placed[:len(placed)-1]
			rack = rack[:len(rack)+1]
			rack[rackIdx], rack[len(rack)-1] = rack[len(rack)-1], rack[rackIdx]
		}
	}
}

// ── Ruleset loading ───────────────────────────────────────────────────────────

type rulesetDef struct {
	Name         string         `json:"name"`
	BingoBonus   int            `json:"bingo_bonus"`
	LetterPoints map[string]int `json:"letter_points"`
	TripleWord   [][2]int       `json:"triple_word"`
	DoubleWord   [][2]int       `json:"double_word"`
	TripleLetter [][2]int       `json:"triple_letter"`
	DoubleLetter [][2]int       `json:"double_letter"`
}

// loadRuleset reads config.json and rulesets.json and applies the selected
// ruleset to the global board/scoring variables. Returns the active ruleset
// name. On any error it prints a warning and keeps the compiled-in crossplay
// defaults unchanged.
func loadRuleset() string {
	const defaultName = "NYT Crossplay (default)"

	cfgBytes, err := os.ReadFile("config.json")
	if err != nil {
		// No config file — silently use defaults.
		return defaultName
	}

	var cfg struct {
		Ruleset string `json:"ruleset"`
	}
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config.json is malformed (%v) — using crossplay defaults\n", err)
		return defaultName
	}

	rsBytes, err := os.ReadFile("rulesets.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: rulesets.json not found — using crossplay defaults\n")
		return defaultName
	}

	var rulesets map[string]rulesetDef
	if err := json.Unmarshal(rsBytes, &rulesets); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: rulesets.json is malformed (%v) — using crossplay defaults\n", err)
		return defaultName
	}

	def, ok := rulesets[cfg.Ruleset]
	if !ok {
		names := make([]string, 0, len(rulesets))
		for k := range rulesets {
			names = append(names, k)
		}
		fmt.Fprintf(os.Stderr, "Warning: ruleset %q not found in rulesets.json (available: %s) — using crossplay defaults\n",
			cfg.Ruleset, strings.Join(names, ", "))
		return defaultName
	}

	applyRuleset(def)
	return def.Name
}

func applyRuleset(def rulesetDef) {
	if def.BingoBonus > 0 {
		bingoBonus = def.BingoBonus
	}
	tilePoints = [255]int{}
	for letter, pts := range def.LetterPoints {
		if len(letter) == 1 {
			tilePoints[letter[0]&^32] = pts // uppercase
		}
	}
	tw = [225]bool{}
	for _, pos := range def.TripleWord {
		tw[cti(pos[0], pos[1])] = true
	}
	dw = [225]bool{}
	for _, pos := range def.DoubleWord {
		dw[cti(pos[0], pos[1])] = true
	}
	tl = [225]bool{}
	for _, pos := range def.TripleLetter {
		tl[cti(pos[0], pos[1])] = true
	}
	dl = [225]bool{}
	for _, pos := range def.DoubleLetter {
		dl[cti(pos[0], pos[1])] = true
	}
}

func (b *Board) PrintBoard() {
	for y := 0; y < 15; y++ {
		line := ""
		for x := 0; x < 15; x++ {
			if b.board[x][y] == 0 {
				if dw[cti(x, y)] {
					line += "\x1b[31;1m"
				} else if tw[cti(x, y)] {
					line += "\x1b[33;1m"
				} else if dl[cti(x, y)] {
					line += "\x1b[34;1m"
				} else if tl[cti(x, y)] {
					line += "\x1b[32;1m"
				}
				line += "."
			} else {
				line += string(b.board[x][y])
			}
			line += "\x1b[0m "
		}
		fmt.Println(line)
	}
	fmt.Println()
	fmt.Println("Player 1:", b.pscore[0])
	fmt.Println("Player 2:", b.pscore[1])
}
