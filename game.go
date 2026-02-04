package main

import (
	"math/rand"
	"time"
)

const (
	boardWidth  = 10
	boardHeight = 20
)

type Point struct {
	X int
	Y int
}

type Game struct {
	Board    [][]int
	X        int
	Y        int
	Rotation int
	Current  int
	Next     int
	HoldKind int
	HasHold  bool
	CanHold  bool
	Score    int
	Lines    int
	Level    int
	Over     bool
	Paused   bool
	bag      []int
	rng      *rand.Rand
}

func NewGame() Game {
	board := make([][]int, boardHeight)
	for i := range board {
		board[i] = make([]int, boardWidth)
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	game := Game{
		Board:   board,
		HoldKind: -1,
		rng:     rng,
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

func (g *Game) Move(dx int) {
	if g.Over || g.Paused {
		return
	}
	if !g.collides(g.X+dx, g.Y, g.Rotation) {
		g.X += dx
	}
}

func (g *Game) SoftDrop() {
	if g.Over || g.Paused {
		return
	}
	if !g.collides(g.X, g.Y+1, g.Rotation) {
		g.Y++
		g.Score++
	}
}

func (g *Game) HardDrop() (bool, int) {
	if g.Over || g.Paused {
		return false, 0
	}
	distance := 0
	for !g.collides(g.X, g.Y+1, g.Rotation) {
		g.Y++
		distance++
	}
	if distance > 0 {
		g.Score += distance * 2
	}
	cleared := g.lockAndSpawn()
	return true, cleared
}

func (g *Game) Rotate(dir int) {
	if g.Over || g.Paused {
		return
	}
	newRot := (g.Rotation + dir + 4) % 4
	if !g.collides(g.X, g.Y, newRot) {
		g.Rotation = newRot
		return
	}
	for _, dx := range []int{-1, 1, -2, 2} {
		if !g.collides(g.X+dx, g.Y, newRot) {
			g.X += dx
			g.Rotation = newRot
			return
		}
	}
}

func (g *Game) Hold() {
	if g.Over || g.Paused || !g.CanHold {
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
	g.CanHold = false
}

func (g *Game) Step() (bool, int) {
	if g.Over || g.Paused {
		return false, 0
	}
	if !g.collides(g.X, g.Y+1, g.Rotation) {
		g.Y++
		return false, 0
	}
	cleared := g.lockAndSpawn()
	return true, cleared
}

func (g *Game) lockAndSpawn() int {
	g.lockPiece()
	cleared := g.clearLines()
	if cleared > 0 {
		scoreTable := []int{0, 100, 300, 500, 800}
		g.Score += scoreTable[cleared] * (g.Level + 1)
		g.Lines += cleared
		g.Level = g.Lines / 10
	}
	g.spawnNext()
	return cleared
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
	if g.collides(g.X, g.Y, g.Rotation) {
		g.Over = true
	}
}

func (g *Game) clearLines() int {
	cleared := 0
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
			for pull := y; pull > 0; pull-- {
				copy(g.Board[pull], g.Board[pull-1])
			}
			for x := 0; x < boardWidth; x++ {
				g.Board[0][x] = 0
			}
			y++
		}
	}
	return cleared
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
