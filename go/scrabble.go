package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"time"
)

func NewBoard(dict string) *Board {
	board := &Board{}
	board.board = make([][]byte, 15)
	for i := 0; i < 15; i++ {
		board.board[i] = make([]byte, 15)
	}
	board.ptiles = [2][]byte{{}, {}}
	board.tiles = []byte(startTiles)
	for i := range board.tiles {
		j := rand.Intn(i + 1)
		board.tiles[i], board.tiles[j] = board.tiles[j], board.tiles[i]
	}
	var err error
	board.wordlist, err = loadDictionary(dict)
	if err != nil {
		fmt.Println("Unable to open dictionary", err)
		return nil
	}
	board.trie, err = buildTrie(dict)
	if err != nil {
		fmt.Println("Unable to build trie", err)
		return nil
	}
	board.ptiles[0], board.tiles = board.tiles[:7], board.tiles[7:]
	board.ptiles[1], board.tiles = board.tiles[:7], board.tiles[7:]
	return board
}

func (b *Board) play(x, y int, word string, dir direction) {
	if dir == DIR_VERT {
		for i := y; len(word) > 0; i++ {
			if b.board[x][i] != 0 {
				continue
			}
			b.board[x][i] = word[0]
			word = word[1:]
		}
	} else {
		for i := x; len(word) > 0; i++ {
			if b.board[i][y] != 0 {
				continue
			}
			b.board[i][y] = word[0]
			word = word[1:]
		}
	}
}

func (b *Board) DoTurn(player int) {
	startCount := len(b.ptiles[player])
	var moves []BestMove
	seen := make(map[string]bool)
	rack := make([]byte, startCount)
	copy(rack, b.ptiles[player])
	rackLen := startCount

	for x := 0; x < 15; x++ {
		for y := 0; y < 15; y++ {
			if b.board[x][y] != 0 {
				continue
			}
			for _, dir := range []direction{DIR_HORIZ, DIR_VERT} {
				startX, startY, play, crossPlays, room := b.getPlaySpace(x, y, dir)
				if room == 0 {
					continue
				}
				var offset int
				if dir == DIR_HORIZ {
					offset = x - startX
				} else {
					offset = y - startY
				}
				node := b.trie
				valid := true
				for i := 0; i < offset; i++ {
					idx := int(play[i]&^32) - int('A')
					if idx < 0 || idx >= 26 || node.children[idx] == nil {
						valid = false
						break
					}
					node = node.children[idx]
				}
				if !valid {
					continue
				}
				b.searchPlay(node, play, crossPlays, offset, rack,
					make([]byte, 0, 7), x, y, dir, rackLen, seen, &moves)
			}
		}
	}

	if len(moves) == 0 {
		fmt.Println("NO WORD FOUND - PASSING")
		return
	}
	sort.Slice(moves, func(i, j int) bool { return moves[i].score > moves[j].score })
	m := moves[0]

	b.play(m.x, m.y, m.tiles, m.dir)
	if startCount == 7 && len(m.tiles) == 7 {
		fmt.Printf("Play %s for %d points (includes %dpt bingo bonus)\n", m.tiles, m.score, bingoBonus)
	} else {
		fmt.Println("Play", m.tiles, "for", m.score, "points")
	}
	for _, c := range m.tiles {
		if c >= 'a' && c <= 'z' {
			c = '*'
		}
		idx := bytes.IndexRune(b.ptiles[player], c)
		b.ptiles[player] = append(b.ptiles[player][:idx], b.ptiles[player][idx+1:]...)
	}
	for len(b.ptiles[player]) < 7 && len(b.tiles) > 0 {
		b.ptiles[player] = append(b.ptiles[player], b.tiles[0])
		b.tiles = b.tiles[1:]
	}
	b.pscore[player] += m.score
}

func runGame() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().Unix())

	ruleset := loadRuleset()
	fmt.Printf("Ruleset: %s\n", ruleset)

	b := NewBoard("dictionary.txt")

	// Game ends when the bag empties. After the bag depletes, each player
	// gets one more turn (starting from the player after whoever drew the last tile).
	bagDepleted := false
	finalPlayer := -1

	for !bagDepleted {
		for p := 0; p < 2; p++ {
			b.DoTurn(p)
			if !bagDepleted && len(b.tiles) == 0 {
				bagDepleted = true
				finalPlayer = p
				break
			}
		}
	}

	// Each player gets one more turn in order.
	for i := 0; i < 2; i++ {
		b.DoTurn((finalPlayer + 1 + i) % 2)
	}

	b.PrintBoard()
}
