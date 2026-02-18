package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"runtime"
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
	var playX, playY, playPoints int
	var playTiles string
	var playDir direction
	tilesets := permute(b.ptiles[player])

	for _, x := range rand.Perm(15) {
		for _, y := range rand.Perm(15) {
			if b.board[x][y] != 0 {
				continue
			}
			for _, dir := range []direction{DIR_HORIZ, DIR_VERT} {
				play, crossPlays, room := b.getPlaySpace(x, y, dir)
				if room > len(b.ptiles[player]) {
					room = len(b.ptiles[player])
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
									for _, v := range crossPlays[i] {
										if v == 0 {
											f2.Add(tileset[j])
										} else {
											f2.Add(v)
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
						if startCount == 7 && len(tileset) == 7 {
							score += bingoBonus
						}
						if score > playPoints {
							playX = x
							playY = y
							playTiles = tileset
							playPoints = score
							playDir = dir
						}
					}
				}
			}
		}
	}
	if playTiles == "" {
		fmt.Println("NO WORD FOUND - PASSING")
		return
	}
	b.play(playX, playY, playTiles, playDir)
	if startCount == 7 && len(playTiles) == 7 {
		fmt.Printf("Play %s for %d points (includes %dpt bingo bonus)\n", playTiles, playPoints, bingoBonus)
	} else {
		fmt.Println("Play", playTiles, "for", playPoints, "points")
	}
	for _, c := range playTiles {
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
	b.pscore[player] += playPoints
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
