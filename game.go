package main

import (
	"math/rand"
	"time"
)

const (
	boardWidth  = 10
	boardHeight = 20
	lockDelay   = 250 * time.Millisecond
)

type Point struct {
	X int
	Y int
}

type Game struct {
	Board       [][]int
	X           int
	Y           int
	Rotation    int
	Current     int
	Next        int
	HoldKind    int
	HasHold     bool
	CanHold     bool
	Score       int
	Lines       int
	Level       int
	Over        bool
	Paused      bool
	lockStart   time.Time
	lastRotate  bool
	bag         []int
	rng         *rand.Rand
	pendingRows []int
}

type LockResult struct {
	Locked      bool
	Cleared     int
	ScoreDelta  int
	TSpin       bool
	ClearedRows []int
}

func NewGame() Game {
	board := make([][]int, boardHeight)
	for i := range board {
		board[i] = make([]int, boardWidth)
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	game := Game{
		Board:    board,
		HoldKind: -1,
		rng:      rng,
	}
	game.refillBag()
	game.Current = game.popBag()
	game.Next = game.popBag()
	game.spawn()
	return game
}

func (g *Game) FallInterval() time.Duration {
	base := 800 * time.Millisecond
	step := 60 * time.Millisecond
	interval := base - time.Duration(g.Level)*step
	if interval < 100*time.Millisecond {
		return 100 * time.Millisecond
	}
	return interval
}

func (g *Game) Move(dx int) bool {
	if g.Over || g.Paused || g.hasPendingLineClear() {
		return false
	}
	if !g.collides(g.X+dx, g.Y, g.Rotation) {
		g.X += dx
		g.resetLock()
		g.lastRotate = false
		return true
	}
	return false
}

func (g *Game) SoftDrop() {
	if g.Over || g.Paused || g.hasPendingLineClear() {
		return
	}
	if !g.collides(g.X, g.Y+1, g.Rotation) {
		g.Y++
		g.Score++
		g.resetLock()
		g.lastRotate = false
	}
}

func (g *Game) HardDrop() LockResult {
	if g.Over || g.Paused || g.hasPendingLineClear() {
		return LockResult{}
	}
	distance := 0
	for !g.collides(g.X, g.Y+1, g.Rotation) {
		g.Y++
		distance++
	}
	if distance > 0 {
		g.Score += distance * 2
	}
	result := g.lockAndSpawn()
	result.Locked = true
	return result
}

func (g *Game) Rotate(dir int) {
	if g.Over || g.Paused || g.hasPendingLineClear() {
		return
	}
	newRot := (g.Rotation + dir + 4) % 4
	if !g.collides(g.X, g.Y, newRot) {
		g.Rotation = newRot
		g.lastRotate = true
		g.resetLock()
		return
	}
	for _, dx := range []int{-1, 1, -2, 2} {
		if !g.collides(g.X+dx, g.Y, newRot) {
			g.X += dx
			g.Rotation = newRot
			g.lastRotate = true
			g.resetLock()
			return
		}
	}
}

func (g *Game) Hold() {
	if g.Over || g.Paused || !g.CanHold || g.hasPendingLineClear() {
		return
	}
	if !g.HasHold {
		g.HoldKind = g.Current
		g.HasHold = true
		g.Current = g.Next
		g.Next = g.popBag()
	} else {
		temp := g.Current
		g.Current = g.HoldKind
		g.HoldKind = temp
	}
	g.spawn()
	g.lastRotate = false
	g.CanHold = false
}

func (g *Game) Step() LockResult {
	if g.Over || g.Paused || g.hasPendingLineClear() {
		return LockResult{}
	}
	if !g.collides(g.X, g.Y+1, g.Rotation) {
		g.Y++
		g.resetLock()
		return LockResult{}
	}
	if g.lockStart.IsZero() {
		g.lockStart = time.Now()
		return LockResult{}
	}
	if time.Since(g.lockStart) < lockDelay {
		return LockResult{}
	}
	result := g.lockAndSpawn()
	result.Locked = true
	return result
}

func (g *Game) lockAndSpawn() LockResult {
	result := LockResult{}
	result.TSpin = g.isTSpin()
	g.lockPiece()
	rows := g.fullRows()
	cleared := len(rows)
	result.Cleared = cleared
	result.ClearedRows = rows
	if result.TSpin {
		scoreTable := []int{400, 800, 1200, 1600}
		if cleared >= 0 && cleared < len(scoreTable) {
			result.ScoreDelta = scoreTable[cleared] * (g.Level + 1)
		}
	} else if cleared > 0 {
		scoreTable := []int{0, 100, 300, 500, 800}
		result.ScoreDelta = scoreTable[cleared] * (g.Level + 1)
	}
	if result.ScoreDelta > 0 {
		g.Score += result.ScoreDelta
	}
	if cleared > 0 {
		g.Lines += cleared
		g.Level = g.Lines / 10
		g.pendingRows = append([]int{}, rows...)
	} else {
		g.spawnNext()
	}
	g.resetLock()
	g.lastRotate = false
	return result
}

func (g *Game) lockPiece() {
	for _, p := range pieceRotations[g.Current][g.Rotation] {
		bx := g.X + p.X
		by := g.Y + p.Y
		if by >= 0 && by < boardHeight && bx >= 0 && bx < boardWidth {
			g.Board[by][bx] = g.Current + 1
		}
	}
}

func (g *Game) spawnNext() {
	g.Current = g.Next
	g.Next = g.popBag()
	g.spawn()
}

func (g *Game) spawn() {
	g.X = 3
	g.Y = 0
	g.Rotation = 0
	g.CanHold = true
	g.resetLock()
	if g.collides(g.X, g.Y, g.Rotation) {
		g.Over = true
	}
}

func (g *Game) clearLines() (int, []int) {
	cleared := 0
	rows := []int{}
	for y := boardHeight - 1; y >= 0; y-- {
		full := true
		for x := 0; x < boardWidth; x++ {
			if g.Board[y][x] == 0 {
				full = false
				break
			}
		}
		if full {
			cleared++
			rows = append(rows, y)
			for pull := y; pull > 0; pull-- {
				copy(g.Board[pull], g.Board[pull-1])
			}
			for x := 0; x < boardWidth; x++ {
				g.Board[0][x] = 0
			}
			y++
		}
	}
	return cleared, rows
}

func (g *Game) clearRows(rows []int) {
	if len(rows) == 0 {
		return
	}
	rowsMap := make(map[int]struct{}, len(rows))
	for _, row := range rows {
		if row >= 0 && row < boardHeight {
			rowsMap[row] = struct{}{}
		}
	}
	if len(rowsMap) == 0 {
		return
	}
	dst := boardHeight - 1
	for src := boardHeight - 1; src >= 0; src-- {
		if _, remove := rowsMap[src]; remove {
			continue
		}
		if dst != src {
			copy(g.Board[dst], g.Board[src])
		}
		dst--
	}
	for ; dst >= 0; dst-- {
		for x := 0; x < boardWidth; x++ {
			g.Board[dst][x] = 0
		}
	}
}

func (g *Game) fullRows() []int {
	rows := []int{}
	for y := boardHeight - 1; y >= 0; y-- {
		full := true
		for x := 0; x < boardWidth; x++ {
			if g.Board[y][x] == 0 {
				full = false
				break
			}
		}
		if full {
			rows = append(rows, y)
		}
	}
	return rows
}

func (g *Game) ResolveLineClear() {
	if !g.hasPendingLineClear() {
		return
	}
	g.clearRows(g.pendingRows)
	g.pendingRows = nil
	g.spawnNext()
}

func (g *Game) hasPendingLineClear() bool {
	return len(g.pendingRows) > 0
}

func (g *Game) resetLock() {
	g.lockStart = time.Time{}
}

func (g *Game) isTSpin() bool {
	if g.Current != 2 || !g.lastRotate {
		return false
	}
	cx := g.X + 1
	cy := g.Y + 1
	corners := [][2]int{
		{cx - 1, cy - 1},
		{cx + 1, cy - 1},
		{cx - 1, cy + 1},
		{cx + 1, cy + 1},
	}
	filled := 0
	for _, c := range corners {
		x := c[0]
		y := c[1]
		if x < 0 || x >= boardWidth || y < 0 || y >= boardHeight {
			filled++
			continue
		}
		if g.Board[y][x] != 0 {
			filled++
		}
	}
	return filled >= 3
}

func (g *Game) collides(x, y, rotation int) bool {
	for _, p := range pieceRotations[g.Current][rotation] {
		bx := x + p.X
		by := y + p.Y
		if bx < 0 || bx >= boardWidth || by < 0 || by >= boardHeight {
			return true
		}
		if g.Board[by][bx] != 0 {
			return true
		}
	}
	return false
}

func (g *Game) popBag() int {
	if len(g.bag) == 0 {
		g.refillBag()
	}
	kind := g.bag[0]
	g.bag = g.bag[1:]
	return kind
}

func (g *Game) refillBag() {
	bag := []int{0, 1, 2, 3, 4, 5, 6}
	g.rng.Shuffle(len(bag), func(i, j int) {
		bag[i], bag[j] = bag[j], bag[i]
	})
	g.bag = bag
}

func (g *Game) GhostY() int {
	y := g.Y
	for !g.collides(g.X, y+1, g.Rotation) {
		y++
	}
	return y
}

var pieceRotations = [7][4][]Point{
	// I
	{
		{{0, 1}, {1, 1}, {2, 1}, {3, 1}},
		{{2, 0}, {2, 1}, {2, 2}, {2, 3}},
		{{0, 2}, {1, 2}, {2, 2}, {3, 2}},
		{{1, 0}, {1, 1}, {1, 2}, {1, 3}},
	},
	// O
	{
		{{1, 0}, {2, 0}, {1, 1}, {2, 1}},
		{{1, 0}, {2, 0}, {1, 1}, {2, 1}},
		{{1, 0}, {2, 0}, {1, 1}, {2, 1}},
		{{1, 0}, {2, 0}, {1, 1}, {2, 1}},
	},
	// T
	{
		{{1, 0}, {0, 1}, {1, 1}, {2, 1}},
		{{1, 0}, {1, 1}, {2, 1}, {1, 2}},
		{{0, 1}, {1, 1}, {2, 1}, {1, 2}},
		{{1, 0}, {0, 1}, {1, 1}, {1, 2}},
	},
	// S
	{
		{{1, 0}, {2, 0}, {0, 1}, {1, 1}},
		{{1, 0}, {1, 1}, {2, 1}, {2, 2}},
		{{1, 1}, {2, 1}, {0, 2}, {1, 2}},
		{{0, 0}, {0, 1}, {1, 1}, {1, 2}},
	},
	// Z
	{
		{{0, 0}, {1, 0}, {1, 1}, {2, 1}},
		{{2, 0}, {1, 1}, {2, 1}, {1, 2}},
		{{0, 1}, {1, 1}, {1, 2}, {2, 2}},
		{{1, 0}, {0, 1}, {1, 1}, {0, 2}},
	},
	// J
	{
		{{0, 0}, {0, 1}, {1, 1}, {2, 1}},
		{{1, 0}, {2, 0}, {1, 1}, {1, 2}},
		{{0, 1}, {1, 1}, {2, 1}, {2, 2}},
		{{1, 0}, {1, 1}, {0, 2}, {1, 2}},
	},
	// L
	{
		{{2, 0}, {0, 1}, {1, 1}, {2, 1}},
		{{1, 0}, {1, 1}, {1, 2}, {2, 2}},
		{{0, 1}, {1, 1}, {2, 1}, {0, 2}},
		{{0, 0}, {1, 0}, {1, 1}, {1, 2}},
	},
}
