# Scrabble Engine — Algorithm Notes

These notes exist so a future port (e.g. a JavaScript webapp) can re-implement the engine
without needing to reverse-engineer the Go source.

---

## 1. Board representation

The board is a **15×15 grid** stored as `board[x][y]` (column-major, x = col, y = row).
Empty cells are `0`. Occupied cells hold an ASCII byte:

| Value | Meaning |
|---|---|
| `'A'`–`'Z'` (uppercase) | Normal tile |
| `'a'`–`'z'` (lowercase) | Blank tile used as that letter (scores 0) |

The helper `cti(x, y) = y*15 + x` converts (x,y) to a flat index used by the
multiplier arrays (`tw`, `dw`, `tl`, `dl`), which are pre-computed `[225]bool` arrays
indexed by flat position.

In JavaScript you'd use a `Uint8Array(225)` for the board and parallel `Uint8Array` or
`boolean[]` arrays for multipliers.

---

## 2. Tile / wildcard representation

- The tile bag uses `'*'` for blank tiles.
- A player's rack is a `[]byte` of 1–7 tiles, mix of `'A'`–`'Z'` and `'*'`.
- When a blank is placed on the board it is stored as the **lowercase** version of the
  letter it represents (e.g. blank played as E → stored as `'e'`).
- Scoring: `tilePoints['e'] == 0` because the `tilePoints` lookup table only has
  uppercase entries — lowercase just falls through to 0.

---

## 3. Dictionary lookup — two parallel structures

The engine maintains **two** representations of the word list, used for different purposes:

### 3a. FNV hash map (`wordlist map[uint64]struct{}`)
Used for **cross-word validation** only (see §6). Keys are FNV-1a hashes of uppercase
words. O(1) lookup. Built by `loadDictionary`.

### 3b. Trie (`*TrieNode`)
Used for **main-word validation during DFS**. Each node has 26 children (A–Z) and an
`isEnd` flag. Built by `buildTrie`, which reads the same file and walks/creates nodes
using `(byte &^ 32) - 'A'` as the child index (this converts any case to 0–25).

**Why two structures?**
The trie drives the DFS and prunes dead branches immediately. Cross-word validation
happens at each empty cell individually — a single FNV hash lookup is the cheapest
option there, and building a 2-D trie for cross-words would be complex.

In JavaScript: use a plain object / `Map` for the hash set, and a tree of objects
(or a flat array of `[26]` child indices into a node array) for the trie.

---

## 4. Play space — `getPlaySpace(x, y, dir)`

Before searching, the engine needs to know what a word placed at (x,y) in direction
`dir` would look like from the **beginning** of its run.

**What it does:**
1. **Back up** from (x,y) along `dir` through any existing tiles already on the board.
   The backed-up position is `(startX, startY)`.
2. **Walk forward** from `(startX, startY)` to the edge of the board, collecting:
   - `play[]` — a byte slice where `0` = empty cell, non-zero = existing tile.
   - `crossPlays[]` — parallel slice; for each *empty* cell, if there are existing tiles
     orthogonally adjacent (forming a potential cross-word), store them as a byte slice
     with a `0` placeholder where the new tile would go. If no cross-word context, `nil`.
   - `room` — count of empty cells (max tiles we could place).

**Return values:** `startX, startY, play, crossPlays, room`

**Why the caller needs `startX`/`startY`:**
The anchor (x,y) passed in is somewhere *inside* `play[]`. The offset
`offset = x - startX` (horiz) or `offset = y - startY` (vert) tells the DFS
where in `play[]` the anchor sits — i.e. where the search starts placing rack tiles.

---

## 5. Trie DFS — `searchPlay`

This is the heart of the engine. It walks the trie and `play[]` simultaneously, building
a word character by character.

### Signature
```
searchPlay(node, play, crossPlays, playIdx, rack, placed,
           anchorX, anchorY, dir, rackLen, seen, moves)
```

- `node` — current trie node (starts pre-walked through existing tiles before anchor)
- `play` — the play space array from `getPlaySpace`
- `crossPlays` — parallel cross-word context array
- `playIdx` — current position within `play[]`
- `rack` — remaining rack tiles (mutated in-place but always restored on return)
- `placed` — tiles placed from rack so far (only new tiles, not existing board tiles)
- `anchorX, anchorY` — the empty cell where `placed[0]` will go
- `rackLen` — original rack length before any placement (for bingo detection)

### Logic (pseudo-code)
```
canStop = (playIdx >= len(play)) OR (play[playIdx] == 0)
if canStop AND node.isEnd AND len(placed) > 0:
    recordMove(...)   // found a valid word

if playIdx >= len(play) OR len(rack) == 0:
    return            // no more board space or tiles

curr = play[playIdx]
if curr != 0:
    // Existing board tile — must follow this edge or dead end
    idx = uppercase(curr) - 'A'
    if node.children[idx] != nil:
        searchPlay(node.children[idx], ..., playIdx+1, ...)
    return

// Empty cell — try each rack tile
tried = [26]bool{}
for each tile t in rack:
    isWild = (t == '*')
    for letter = 'A' to 'Z':
        if not isWild and uppercase(t) != letter: continue
        if tried[letter]: continue
        if node.children[letter-'A'] == nil: continue   // trie prune
        if crossPlays[playIdx] != nil:
            validate cross-word via FNV hash; skip if invalid
        tried[letter] = true
        stored = letter (uppercase) or letter+32 (lowercase if blank)
        // swap tile to end of rack, shrink rack
        rack[rackIdx] ↔ rack[last]; rack = rack[:last]
        placed.push(stored)
        searchPlay(node.children[letter-'A'], ..., playIdx+1, ...)
        placed.pop()
        rack = rack[:last+1]; rack[rackIdx] ↔ rack[last]  // restore
```

### Key design points

**`canStop` condition:** We can end the word here only if the *next* position is either
off the board or an empty cell. If the next position has an existing tile, we *must*
keep going — the rules forbid a gap between your tiles and an existing tile in the same
direction.

**`tried[26]` dedup:** Prevents trying the same letter twice (e.g. two `'A'` tiles in
the rack, or two wildcards that both map to `'A'`). Without this, you'd generate
duplicate moves.

**Rack mutation / restore:** The swap-to-end trick (`rack[i] ↔ rack[last]; rack = rack[:last]`)
is a standard O(1) "remove from unordered slice" idiom. Restoring swaps in reverse
order brings the rack back to its original state. In JavaScript you can do the same
with a plain array.

**`placed` vs full word:** `placed` only contains the *new* tiles from the rack. The
full word also includes existing tiles that were pre-walked or encountered at non-zero
positions in `play[]`. `scoreMove` reconstructs the full word by reading the board.

---

## 6. Cross-word validation

When placing a tile at an empty cell, it may form a new word in the *orthogonal*
direction with tiles already on the board.

`crossPlays[playIdx]` contains the orthogonal context: a byte array of existing tiles
with a single `0` at the position where the new tile goes. Example: if placing
horizontally and there's `B` above and `T` below the empty cell, `crossPlays[i] = [B, 0, T]`.

Validation: build the FNV-1a hash of this array (substituting the candidate letter for
the `0`) and look it up in `wordlist`. If not found, skip this letter.

FNV-1a in JavaScript:
```js
function fnv1a(bytes) {
    let h = 0xcbf29ce484222325n;  // BigInt (64-bit)
    for (const b of bytes) {
        h = BigInt.asUintN(64, h * 0x100000001b3n);
        h ^= BigInt(b & ~32);     // uppercase
    }
    return h;
}
```
Note: JavaScript's native numbers are 64-bit floats and can't represent 64-bit integers
exactly. Use `BigInt` for the FNV hash, or use a different dictionary lookup strategy
(e.g. a `Set` of uppercase strings) for the cross-word check.

---

## 7. Scoring

### `scoreWord(x, y, dir, plays)`

Scores a single word. `plays` is a scratch `[225]byte` array showing which cells have
newly placed tiles (non-zero = this move's tile, board tiles are read directly from
`b.board`).

1. Back up to find the start of the word (skip before first tile in this direction).
2. Walk forward, summing letter values and tracking word multipliers.
3. Letter multipliers (`dl` = double letter, `tl` = triple letter) apply only to newly
   placed tiles (i.e. `plays[idx] != 0`).
4. Word multipliers (`dw`, `tw`) also only apply to newly placed tiles.
5. If the word is 1 letter long, return 0 (single letters don't score).

### `scoreMove(x, y, tiles, dir)`

Scores the *complete* placement: main word + all cross-words.

1. Place `tiles[0]`, `tiles[1]`, ... into a scratch `plays[]` array, skipping cells
   where `board[x][y] != 0` (existing tiles; keep iterating to fill next empty).
2. For each newly placed tile, call `scoreWord` in the *orthogonal* direction to score
   any cross-word formed.
3. Finally call `scoreWord` in the *primary* direction to score the main word.
4. Return the sum.

**Bingo bonus:** If the player used all 7 tiles, add `bingoBonus` (40 for NYT Crossplay,
50 for Standard Scrabble) *after* `scoreMove` returns.

---

## 8. Move validation — `recordMove`

Called whenever `node.isEnd && len(placed) > 0`.

1. **`checkCenterPlayed(anchorX, anchorY, len(placed), dir)`**
   On the very first move, the placement must cover (7,7). If the board already has
   a tile at (7,7), this check always passes.

2. **`checkContiguous(anchorX, anchorY, len(placed), dir)`**
   After the first move, the placement must touch at least one existing tile (adjacency
   in any direction). If the board is empty, this always passes.

3. **Score + dedup:** Compute `scoreMove`, optionally add bingo bonus, build a string
   key `"x,y,dir,TILES"` (uppercase), and only add to the move list if unseen.

---

## 9. Outer loop — `findTopNMoves` / `DoTurn`

```
for each empty cell (x, y):
    for each direction (HORIZ, VERT):
        (startX, startY, play, crossPlays, room) = getPlaySpace(x, y, dir)
        if room == 0: skip

        offset = x - startX  (or y - startY for VERT)

        // Pre-walk trie through existing tiles before the anchor
        node = trie.root
        for i in 0..offset-1:
            node = node.children[play[i] uppercased - 'A']
            if node == nil: break (dead end, skip this anchor)

        searchPlay(node, play, crossPlays, offset, rack, [], x, y, dir, ...)

sort moves by score descending
return top N  (findTopNMoves) / pick moves[0]  (DoTurn)
```

**Why iterate only empty cells?**
Every valid Scrabble word must place at least one new tile. An anchor is the leftmost/
topmost new tile in the word. By iterating only empty cells, every anchor is exactly
one new tile — the DFS then explores all continuations.

**Why pre-walk the trie?**
When an anchor has existing tiles before it (backed-up by `getPlaySpace`), those tiles
are already determined. Pre-walking the trie through them before the DFS begins means
the DFS inherits a partially-walked trie state and never re-explores prefixes that are
impossible.

---

## 10. JavaScript port checklist

- [ ] `board`: `Uint8Array(225)`, column-major (`board[x*15+y]` or use `cti(x,y) = y*15+x`)
      Actually keep `board[x][y]` as a 2D array for clarity, or use `board[cti(x,y)]`.
- [ ] Multipliers: four `Uint8Array(225)` or four `Set<number>` of flat indices.
- [ ] Trie: array of `{children: Int32Array(26), isEnd: boolean}` nodes, or a nested
      object tree (simpler, slower).
- [ ] Dictionary hash set: `Set<bigint>` with FNV-1a using `BigInt`, or just a
      `Set<string>` of uppercase words (simpler, ~5× more memory).
- [ ] Rack: plain `number[]` (ASCII codes), or `string[]` of single chars.
- [ ] The swap-restore idiom works identically in JS arrays.
- [ ] `getPlaySpace` returns the same 5 values; cross-word context arrays work the same.
- [ ] `searchPlay` is naturally expressed as a recursive function; JS engines handle
      recursion depth up to ~7 tiles × 15 board positions without issue.
- [ ] For the webapp, run the search in a **Web Worker** so the UI thread stays
      responsive. Post the rack + board state to the worker, get back the move list.
