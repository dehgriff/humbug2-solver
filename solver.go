package main

import (
	"fmt"
	"sort"
	"strings"
)

type Direction byte

const (
	Up Direction = iota
	Right
	Down
	Left
)

var directions = []Direction{Up, Right, Down, Left}
var deltas = map[Direction]Pos{Up: {-1, 0}, Right: {0, 1}, Down: {1, 0}, Left: {0, -1}}

func (d Direction) String() string { return []string{"up", "right", "down", "left"}[d] }

type Move struct {
	Kind BugType
	From Pos
	Dir  Direction
	To   Pos
}

func (m Move) String() string {
	return fmt.Sprintf("%s at (%d,%d) %s -> (%d,%d)", m.Kind, m.From.R+1, m.From.C+1, m.Dir, m.To.R+1, m.To.C+1)
}

type State struct {
	Bugs  []Bug
	Stars [BoardSize][BoardSize]bool
}

type searchNode struct {
	state  State
	parent int
	move   Move
	depth  int
}

func Solve(p Puzzle) ([]Move, bool) {
	start := State{Bugs: append([]Bug(nil), p.Bugs...), Stars: p.Stars}
	canonicalize(&start)
	if len(start.Bugs) == 0 {
		return nil, true
	}
	nodes := []searchNode{{state: start, parent: -1}}
	seen := map[string]struct{}{stateKey(start): {}}
	for head := 0; head < len(nodes); head++ {
		cur := nodes[head]
		if cur.depth >= p.MaxMoves {
			continue
		}
		for i := range cur.state.Bugs {
			for _, dir := range directions {
				next, move, ok := applyMove(p, cur.state, i, dir)
				if !ok { // includes blocked/no-op and falling branches
					continue
				}
				if len(next.Bugs) == 0 {
					return buildPath(nodes, head, move), true
				}
				key := stateKey(next)
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				nodes = append(nodes, searchNode{state: next, parent: head, move: move, depth: cur.depth + 1})
			}
		}
	}
	return nil, false
}

func buildPath(nodes []searchNode, parent int, last Move) []Move {
	path := make([]Move, nodes[parent].depth+1)
	path[len(path)-1] = last
	for i := len(path) - 2; i >= 0; i-- {
		path[i] = nodes[parent].move
		parent = nodes[parent].parent
	}
	return path
}

func applyMove(p Puzzle, old State, index int, dir Direction) (State, Move, bool) {
	s := cloneState(old)
	bug := s.Bugs[index]
	from := bug.Pos
	occupied := occupancy(s.Bugs)
	delete(occupied, from)
	var to Pos
	var ok, fell bool
	switch bug.Kind {
	case Bee:
		to, ok, fell = fly(p, bug.Pos, dir, 2, occupied)
	case Ladybird:
		to, ok, fell = walk(p, bug, dir, 2, occupied)
	case Spider:
		to, ok, fell = spider(p, bug, dir, occupied)
	case Beetle:
		to, ok, fell = beetle(p, &s, index, dir)
	}
	if fell || !ok || to == from {
		return State{}, Move{}, false
	}
	s.Bugs[index].Pos = to
	consumeStars(&s)
	canonicalize(&s)
	return s, Move{Kind: bug.Kind, From: from, Dir: dir, To: to}, true
}

func walk(p Puzzle, bug Bug, dir Direction, limit int, occupied map[Pos]int) (Pos, bool, bool) {
	cur := bug.Pos
	for step := 0; step < limit; step++ {
		next := add(cur, deltas[dir])
		if !inside(next) || p.Terrain[next.R][next.C] == Void {
			return Pos{}, false, true
		}
		if _, blocked := occupied[next]; blocked || (p.Terrain[cur.R][cur.C] == Low && p.Terrain[next.R][next.C] == High) {
			return cur, cur != bug.Pos, false
		}
		// Dropping from high to low changes to landing/bounce behaviour.
		if p.Terrain[cur.R][cur.C] == High && p.Terrain[next.R][next.C] == Low {
			return land(p, next, dir, occupied)
		}
		cur = next
	}
	return cur, true, false
}

func fly(p Puzzle, start Pos, dir Direction, distance int, occupied map[Pos]int) (Pos, bool, bool) {
	landing := start
	for i := 0; i < distance; i++ {
		landing = add(landing, deltas[dir])
	}
	return land(p, landing, dir, occupied)
}

func land(p Puzzle, landing Pos, dir Direction, occupied map[Pos]int) (Pos, bool, bool) {
	for {
		if !inside(landing) || p.Terrain[landing.R][landing.C] == Void {
			return Pos{}, false, true
		}
		if _, taken := occupied[landing]; !taken {
			return landing, true, false
		}
		landing = add(landing, deltas[dir])
	}
}

func spider(p Puzzle, bug Bug, dir Direction, occupied map[Pos]int) (Pos, bool, bool) {
	cur := bug.Pos
	for {
		next := add(cur, deltas[dir])
		if !inside(next) || p.Terrain[next.R][next.C] == Void {
			return Pos{}, false, true
		}
		if _, blocked := occupied[next]; blocked || (p.Terrain[cur.R][cur.C] == Low && p.Terrain[next.R][next.C] == High) {
			return cur, cur != bug.Pos, false
		}
		if p.Terrain[cur.R][cur.C] == High && p.Terrain[next.R][next.C] == Low {
			return land(p, next, dir, occupied)
		}
		cur = next
	}
}

func beetle(p Puzzle, s *State, index int, dir Direction) (Pos, bool, bool) {
	start := s.Bugs[index].Pos
	next := add(start, deltas[dir])
	if !inside(next) || p.Terrain[next.R][next.C] == Void {
		return Pos{}, false, true
	}
	if p.Terrain[start.R][start.C] == Low && p.Terrain[next.R][next.C] == High {
		return start, false, false
	}
	occ := occupancy(s.Bugs)
	if _, taken := occ[next]; !taken {
		if p.Terrain[start.R][start.C] == High && p.Terrain[next.R][next.C] == Low {
			delete(occ, start)
			return land(p, next, dir, occ)
		}
		return next, true, false
	}
	// Find the whole contiguous chain and validate its destination before
	// changing anything. Each pushed bug obeys the low-to-high restriction.
	chain := []int{}
	cur := next
	for {
		j, taken := occ[cur]
		if !taken {
			break
		}
		chain = append(chain, j)
		dest := add(cur, deltas[dir])
		if !inside(dest) || p.Terrain[dest.R][dest.C] == Void {
			return Pos{}, false, true
		}
		if p.Terrain[cur.R][cur.C] == Low && p.Terrain[dest.R][dest.C] == High {
			return start, false, false
		}
		cur = dest
	}
	for i := len(chain) - 1; i >= 0; i-- {
		s.Bugs[chain[i]].Pos = add(s.Bugs[chain[i]].Pos, deltas[dir])
	}
	return next, true, false
}

func consumeStars(s *State) {
	kept := s.Bugs[:0]
	for _, b := range s.Bugs {
		if s.Stars[b.Pos.R][b.Pos.C] {
			s.Stars[b.Pos.R][b.Pos.C] = false
			continue
		}
		kept = append(kept, b)
	}
	s.Bugs = kept
}

func occupancy(bugs []Bug) map[Pos]int {
	m := make(map[Pos]int, len(bugs))
	for i, b := range bugs {
		m[b.Pos] = i
	}
	return m
}

func cloneState(s State) State {
	s.Bugs = append([]Bug(nil), s.Bugs...)
	return s
}

func canonicalize(s *State) {
	sort.Slice(s.Bugs, func(i, j int) bool {
		if s.Bugs[i].Kind != s.Bugs[j].Kind {
			return s.Bugs[i].Kind < s.Bugs[j].Kind
		}
		if s.Bugs[i].Pos.R != s.Bugs[j].Pos.R {
			return s.Bugs[i].Pos.R < s.Bugs[j].Pos.R
		}
		return s.Bugs[i].Pos.C < s.Bugs[j].Pos.C
	})
}

func stateKey(s State) string {
	var b strings.Builder
	for _, bug := range s.Bugs {
		b.WriteByte(byte(bug.Kind))
		b.WriteByte(byte(bug.Pos.R))
		b.WriteByte(byte(bug.Pos.C))
	}
	b.WriteByte('|')
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			if s.Stars[r][c] {
				b.WriteByte(byte(r*BoardSize + c + 1))
			}
		}
	}
	return b.String()
}

func add(a, b Pos) Pos  { return Pos{a.R + b.R, a.C + b.C} }
func inside(p Pos) bool { return p.R >= 0 && p.R < BoardSize && p.C >= 0 && p.C < BoardSize }
