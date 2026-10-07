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
	Kind  BugType
	Count uint8
	From  Pos
	Dir   Direction
	To    Pos
}

func (m Move) String() string {
	if m.Kind == Ant && m.Count > 1 {
		return fmt.Sprintf("%d ants at (%d,%d) %s -> (%d,%d)", m.Count, m.From.R+1, m.From.C+1, m.Dir, m.To.R+1, m.To.C+1)
	}
	return fmt.Sprintf("%s at (%d,%d) %s -> (%d,%d)", m.Kind, m.From.R+1, m.From.C+1, m.Dir, m.To.R+1, m.To.C+1)
}

type State struct {
	Bugs  []Bug
	Stars [BoardSize][BoardSize]uint8
	Walls [BoardSize][BoardSize]uint8
}

type searchNode struct {
	state  State
	parent int
	move   Move
	depth  int
}

// SearchProgress describes a search frontier as it is reached.
type SearchProgress struct {
	Algorithm   string
	Depth       int
	States      int
	TotalStates int
}

func Solve(p Puzzle) ([]Move, bool) {
	return SolveWithProgress(p, nil)
}

// SolveWithProgress finds a shortest solution. It uses A* when the number of
// remaining bugs is a safe lower bound, and otherwise falls back to BFS.
func SolveWithProgress(p Puzzle, progress func(SearchProgress)) ([]Move, bool) {
	start := State{Bugs: append([]Bug(nil), p.Bugs...), Stars: p.Stars, Walls: p.Walls}
	canonicalize(&start)
	if remainingBugCount(start) == 0 {
		return nil, true
	}
	if canUseBugCountHeuristic(p) {
		return solveAStar(p, start, progress)
	}
	return solveBFS(p, start, progress)
}

func solveBFS(p Puzzle, start State, progress func(SearchProgress)) ([]Move, bool) {
	nodes := []searchNode{{state: start, parent: -1}}
	includeWalls := hasGoldBeetle(p)
	seen := map[string]struct{}{searchStateKey(start, includeWalls): {}}
	frontier := []int{0}
	for depth := 0; depth <= p.MaxMoves && len(frontier) > 0; depth++ {
		if progress != nil {
			progress(SearchProgress{Algorithm: "bfs", Depth: depth, States: len(frontier), TotalStates: len(nodes)})
		}
		if depth == p.MaxMoves {
			break
		}
		nextFrontier := make([]int, 0)
		for _, head := range frontier {
			cur := nodes[head]
			for i := range cur.state.Bugs {
				if !canInitiateMove(cur.state.Bugs[i]) {
					continue
				}
				for _, dir := range directions {
					next, move, ok := applyMove(p, cur.state, i, dir)
					if !ok { // includes blocked/no-op and falling branches
						continue
					}
					if remainingBugCount(next) == 0 {
						return buildPath(nodes, head, move), true
					}
					key := searchStateKey(next, includeWalls)
					if _, exists := seen[key]; exists {
						continue
					}
					seen[key] = struct{}{}
					nodes = append(nodes, searchNode{state: next, parent: head, move: move, depth: depth + 1})
					nextFrontier = append(nextFrontier, len(nodes)-1)
				}
			}
		}
		frontier = nextFrontier
	}
	return nil, false
}

func solveAStar(p Puzzle, start State, progress func(SearchProgress)) ([]Move, bool) {
	includeWalls := hasGoldBeetle(p)
	startKey := searchStateKey(start, includeWalls)
	totalExpanded := 0
	for bound := remainingBugCount(start); bound <= p.MaxMoves; bound++ {
		bestDepth := map[string]int{startKey: 0}
		path := make([]Move, 0, bound)
		expanded := 0
		if idaSearch(p, start, 0, bound, includeWalls, bestDepth, &path, &expanded) {
			if progress != nil {
				progress(SearchProgress{Algorithm: "astar", Depth: bound, States: expanded, TotalStates: totalExpanded + expanded})
			}
			return path, true
		}
		totalExpanded += expanded
		if progress != nil {
			progress(SearchProgress{Algorithm: "astar", Depth: bound, States: expanded, TotalStates: totalExpanded})
		}
	}
	return nil, false
}

type successor struct {
	state State
	move  Move
	h     int
}

func idaSearch(p Puzzle, state State, depth, bound int, includeWalls bool, bestDepth map[string]int, path *[]Move, expanded *int) bool {
	*expanded++
	successors := make([]successor, 0, len(state.Bugs)*len(directions))
	for i := range state.Bugs {
		if !canInitiateMove(state.Bugs[i]) {
			continue
		}
		for _, dir := range directions {
			next, move, ok := applyMove(p, state, i, dir)
			if !ok {
				continue
			}
			h := remainingBugCount(next)
			if depth+1+h <= bound {
				successors = append(successors, successor{state: next, move: move, h: h})
			}
		}
	}
	// Scoring moves first makes a solution quick to find in the final iteration.
	sort.SliceStable(successors, func(i, j int) bool { return successors[i].h < successors[j].h })
	for _, next := range successors {
		*path = append(*path, next.move)
		if next.h == 0 {
			return true
		}
		nextDepth := depth + 1
		key := searchStateKey(next.state, includeWalls)
		if oldDepth, exists := bestDepth[key]; exists && oldDepth <= nextDepth {
			*path = (*path)[:len(*path)-1]
			continue
		}
		bestDepth[key] = nextDepth
		if idaSearch(p, next.state, nextDepth, bound, includeWalls, bestDepth, path, expanded) {
			return true
		}
		*path = (*path)[:len(*path)-1]
	}
	return false
}

func canUseBugCountHeuristic(p Puzzle) bool {
	for _, bug := range p.Bugs {
		if bug.Egg || bug.Kind == Beetle {
			return false
		}
	}
	return true
}

func hasGoldBeetle(p Puzzle) bool {
	for _, bug := range p.Bugs {
		if bug.Kind == GoldBeetle {
			return true
		}
	}
	return false
}

func canInitiateMove(bug Bug) bool {
	return !bug.Egg && bug.Kind != Puck && bug.Kind != Scorpion
}

func canFallWithoutLosing(bug Bug) bool {
	return bug.Kind == Puck || bug.Kind == Scorpion
}

func updatePushedScorpionDirections(before, after []Bug) {
	oldPositions := make(map[uint8]Pos)
	for _, bug := range before {
		if bug.Kind == Scorpion {
			oldPositions[bug.ID] = bug.Pos
		}
	}
	for i := range after {
		bug := &after[i]
		if bug.Kind != Scorpion {
			continue
		}
		old, exists := oldPositions[bug.ID]
		if !exists || old == bug.Pos || !inside(bug.Pos) {
			continue
		}
		switch {
		case bug.Pos.C < old.C:
			bug.Direction = Left
		case bug.Pos.R < old.R:
			bug.Direction = Up
		case bug.Pos.C > old.C:
			bug.Direction = Right
		case bug.Pos.R > old.R:
			bug.Direction = Down
		}
	}
}

func moveScorpions(p Puzzle, s *State) bool {
	maxID := -1
	for _, bug := range s.Bugs {
		if bug.Kind == Scorpion && int(bug.ID) > maxID {
			maxID = int(bug.ID)
		}
	}
	for id := 0; id <= maxID; id++ {
		index := -1
		for i := range s.Bugs {
			if s.Bugs[i].Kind == Scorpion && int(s.Bugs[i].ID) == id {
				index = i
				break
			}
		}
		if index < 0 || s.Bugs[index].Egg {
			continue
		}
		if !s.Bugs[index].Stunned && scorpionAdjacentToBug(p, s.Bugs, index) {
			return false
		}
		dir, moves := scorpionDirection(p, s.Bugs, index)
		if !moves {
			continue
		}
		start := s.Bugs[index].Pos
		next := add(start, deltas[dir])
		s.Bugs[index].Direction = dir
		if p.Trampolines[next.R][next.C] {
			occupied := occupancy(s.Bugs)
			delete(occupied, start)
			landing, ok, fell := land(p, next, dir, occupied, s.Bugs)
			if fell {
				s.Bugs[index].Pos = Pos{-1, -1}
			} else if ok {
				s.Bugs[index].Pos = landing
			}
		} else {
			s.Bugs[index].Pos = next
		}
		removeGoneEntities(s)
		consumeStars(s)
	}
	return true
}

func scorpionAdjacentToBug(p Puzzle, bugs []Bug, scorpion int) bool {
	from := bugs[scorpion].Pos
	level := p.Terrain[from.R][from.C]
	for _, bug := range bugs {
		if bug.Kind == Puck || bug.Kind == Scorpion || bug.Egg {
			continue
		}
		if p.Terrain[bug.Pos.R][bug.Pos.C] != level || abs(bug.Pos.R-from.R)+abs(bug.Pos.C-from.C) != 1 {
			continue
		}
		for _, dir := range directions {
			if add(from, deltas[dir]) == bug.Pos && !hasWall(p, from, dir) {
				return true
			}
		}
	}
	return false
}

// scorpionDirection chooses a first step on a shortest unobstructed path to a
// bug on the same level. The current direction wins ties, followed by L/U/R/D.
func scorpionDirection(p Puzzle, bugs []Bug, scorpion int) (Direction, bool) {
	from := bugs[scorpion].Pos
	level := p.Terrain[from.R][from.C]
	occupied := occupancy(bugs)
	delete(occupied, from)
	order := []Direction{bugs[scorpion].Direction, Left, Up, Right, Down}
	seenDirection := [4]bool{}
	bestDistance := BoardSize*BoardSize + 1
	bestDirection := Up
	found := false
	for _, dir := range order {
		if seenDirection[dir] {
			continue
		}
		seenDirection[dir] = true
		if hasWall(p, from, dir) {
			continue
		}
		first := add(from, deltas[dir])
		if !inside(first) || p.Terrain[first.R][first.C] != level {
			continue
		}
		if _, blocked := occupied[first]; blocked {
			continue
		}
		distance, ok := distanceToReachableBug(p, bugs, first, level, occupied)
		if ok && distance+1 < bestDistance {
			bestDistance = distance + 1
			bestDirection = dir
			found = true
		}
	}
	return bestDirection, found
}

func distanceToReachableBug(p Puzzle, bugs []Bug, start Pos, level Terrain, occupied map[Pos]int) (int, bool) {
	type step struct {
		pos      Pos
		distance int
	}
	queue := []step{{pos: start}}
	visited := [BoardSize][BoardSize]bool{}
	visited[start.R][start.C] = true
	for head := 0; head < len(queue); head++ {
		cur := queue[head]
		for _, dir := range directions {
			if hasWall(p, cur.pos, dir) {
				continue
			}
			next := add(cur.pos, deltas[dir])
			if !inside(next) || p.Terrain[next.R][next.C] != level {
				continue
			}
			if index, blocked := occupied[next]; blocked {
				bug := bugs[index]
				if bug.Kind != Puck && bug.Kind != Scorpion && !bug.Egg {
					return cur.distance + 1, true
				}
				continue
			}
			if visited[next.R][next.C] {
				continue
			}
			visited[next.R][next.C] = true
			queue = append(queue, step{pos: next, distance: cur.distance + 1})
		}
	}
	return 0, false
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
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

type moveStatus byte

const (
	moveInvalid moveStatus = iota
	moveApplied
	moveLost
)

type moveOutcome struct {
	state  State
	move   Move
	status moveStatus
	reason string
}

func applyMove(p Puzzle, old State, index int, dir Direction) (State, Move, bool) {
	outcome := applyMoveDetailed(p, old, index, dir)
	return outcome.state, outcome.move, outcome.status == moveApplied
}

func applyMoveDetailed(p Puzzle, old State, index int, dir Direction) moveOutcome {
	if !canInitiateMove(old.Bugs[index]) {
		return moveOutcome{state: old, status: moveInvalid, reason: "that object cannot initiate a move"}
	}
	s := cloneState(old)
	for i := range s.Bugs {
		s.Bugs[i].Stunned = false
	}
	p.Walls = s.Walls
	bug := s.Bugs[index]
	from := bug.Pos
	occupied := occupancy(s.Bugs)
	delete(occupied, from)
	var to Pos
	var ok, fell bool
	positionHandled := false
	switch bug.Kind {
	case Bee:
		to, ok, fell = fly(p, bug.Pos, dir, 2, occupied, s.Bugs)
	case Grasshopper:
		to, ok, fell = fly(p, bug.Pos, dir, 1, occupied, s.Bugs)
	case Butterfly:
		to, ok, fell = fly(p, bug.Pos, dir, 3, occupied, s.Bugs)
	case Fly:
		to, ok, fell = flyToPlatform(p, bug.Pos, dir, occupied, s.Bugs)
	case Ant:
		to, ok, fell = moveAnt(p, &s, index, dir, occupied)
		positionHandled = true
	case Ladybird:
		to, ok, fell = walk(p, bug, dir, 2, occupied, false, s.Bugs)
	case PinkLadybird:
		to, ok, fell = walk(p, bug, dir, 3, occupied, false, s.Bugs)
	case Snail:
		to, ok, fell = walk(p, bug, dir, 1, occupied, false, s.Bugs)
	case Cockroach:
		to, ok, fell = walk(p, bug, dir, 2, occupied, true, s.Bugs)
	case GoldBeetle:
		to, ok, fell = walkGoldBeetle(p, &s, bug, dir, occupied)
	case Spider:
		to, ok, fell = spider(p, bug, dir, occupied, s.Bugs)
	case Beetle:
		to, ok, fell = beetle(p, &s, index, dir)
	}
	wallsChanged := s.Walls != old.Walls
	move := Move{Kind: bug.Kind, Count: antCount(bug), From: from, Dir: dir, To: to}
	if fell {
		return moveOutcome{state: old, move: move, status: moveLost, reason: fmt.Sprintf("%s fell off the board", bug.Kind)}
	}
	if !ok || (to == from && !wallsChanged) {
		return moveOutcome{state: old, move: move, status: moveInvalid, reason: "the bug cannot move in that direction"}
	}
	if !positionHandled {
		s.Bugs[index].Pos = to
	}
	removeGoneEntities(&s)
	consumeStars(&s)
	updatePushedScorpionDirections(old.Bugs, s.Bugs)
	// A Gold Beetle may have changed the walls during the player move.
	p.Walls = s.Walls
	if !moveScorpions(p, &s) {
		return moveOutcome{state: s, move: move, status: moveLost, reason: "a scorpion killed an adjacent bug"}
	}
	removeGoneEntities(&s)
	for i := range s.Bugs {
		s.Bugs[i].Stunned = false
	}
	canonicalize(&s)
	return moveOutcome{state: s, move: move, status: moveApplied}
}

func moveAnt(p Puzzle, s *State, index int, dir Direction, occupied map[Pos]int) (Pos, bool, bool) {
	start := s.Bugs[index].Pos
	cur := start
	s.Bugs[index].Count = antCount(s.Bugs[index])
	for {
		if hasWall(p, cur, dir) {
			s.Bugs[index].Pos = cur
			return cur, cur != start, false
		}
		next := add(cur, deltas[dir])
		if !inside(next) || p.Terrain[next.R][next.C] == Void {
			return Pos{}, false, true
		}
		if p.Terrain[cur.R][cur.C] == High && p.Terrain[next.R][next.C] == Low {
			to, ok, fell := land(p, next, dir, occupied, s.Bugs)
			if ok {
				s.Bugs[index].Pos = to
			}
			return to, ok, fell
		}
		if p.Terrain[cur.R][cur.C] == Low && p.Terrain[next.R][next.C] == High {
			s.Bugs[index].Pos = cur
			return cur, cur != start, false
		}
		if p.Trampolines[next.R][next.C] {
			to, ok, fell := land(p, next, dir, occupied, s.Bugs)
			if ok {
				s.Bugs[index].Pos = to
			}
			return to, ok, fell
		}
		if other, blocked := occupied[next]; blocked {
			if s.Bugs[other].Kind == Ant && !s.Bugs[other].Egg {
				destinationCount := antCount(s.Bugs[other])
				capacity := uint8(3) - destinationCount
				moving := s.Bugs[index].Count
				joined := moving
				if joined > capacity {
					joined = capacity
				}
				if joined > 0 {
					s.Bugs[other].Count = destinationCount + joined
					s.Bugs[index].Count = moving - joined
					if s.Bugs[index].Count == 0 {
						s.Bugs[index].Pos = Pos{-1, -1}
					} else {
						s.Bugs[index].Pos = cur
					}
					return next, true, false
				}
			}
			s.Bugs[index].Pos = cur
			return cur, cur != start, false
		}
		cur = next
	}
}

func walk(p Puzzle, bug Bug, dir Direction, limit int, occupied map[Pos]int, crossesWalls bool, bugs []Bug) (Pos, bool, bool) {
	cur := bug.Pos
	for step := 0; step < limit; step++ {
		if !crossesWalls && hasWall(p, cur, dir) {
			return cur, cur != bug.Pos, false
		}
		next := add(cur, deltas[dir])
		if !inside(next) || p.Terrain[next.R][next.C] == Void {
			return Pos{}, false, true
		}
		// Dropping from high to low changes to landing/bounce behaviour.
		if p.Terrain[cur.R][cur.C] == High && p.Terrain[next.R][next.C] == Low {
			return land(p, next, dir, occupied, bugs)
		}
		if p.Terrain[cur.R][cur.C] == Low && p.Terrain[next.R][next.C] == High {
			return cur, cur != bug.Pos, false
		}
		if p.Trampolines[next.R][next.C] {
			return land(p, next, dir, occupied, bugs)
		}
		if _, blocked := occupied[next]; blocked {
			return cur, cur != bug.Pos, false
		}
		cur = next
	}
	return cur, true, false
}

func walkGoldBeetle(p Puzzle, s *State, bug Bug, dir Direction, occupied map[Pos]int) (Pos, bool, bool) {
	cur := bug.Pos
	if struckWalls, stop, blocked := goldBeetleBlockedMove(p, cur, dir, occupied); blocked {
		for _, wall := range struckWalls {
			knockDownWall(&s.Walls, wall, dir)
		}
		return stop, true, false
	}
	for step := 0; step < 2; step++ {
		next := add(cur, deltas[dir])
		if !inside(next) || p.Terrain[next.R][next.C] == Void {
			return Pos{}, false, true
		}
		if p.Terrain[cur.R][cur.C] == Low && p.Terrain[next.R][next.C] == High {
			return cur, cur != bug.Pos, false
		}
		// The beetle crosses this boundary under its own power, so only this
		// walking portion of the move destroys a wall. Later bounce boundaries
		// are airborne and must remain intact.
		if p.Terrain[cur.R][cur.C] == High && p.Terrain[next.R][next.C] == Low {
			knockDownWall(&s.Walls, cur, dir)
			return land(p, next, dir, occupied, s.Bugs)
		}
		if p.Trampolines[next.R][next.C] {
			knockDownWall(&s.Walls, cur, dir)
			return land(p, next, dir, occupied, s.Bugs)
		}
		if _, blocked := occupied[next]; blocked {
			// Even though the obstacle prevents entry, the Gold Beetle has
			// struck this boundary and knocks down any wall on it.
			hadWall := hasWall(p, cur, dir)
			knockDownWall(&s.Walls, cur, dir)
			return cur, cur != bug.Pos || hadWall, false
		}
		knockDownWall(&s.Walls, cur, dir)
		cur = next
	}
	return cur, true, false
}

func goldBeetleBlockedMove(p Puzzle, start Pos, dir Direction, occupied map[Pos]int) ([]Pos, Pos, bool) {
	cur := start
	var struck []Pos
	for step := 0; step < 2; step++ {
		if hasWall(p, cur, dir) {
			struck = append(struck, cur)
		}
		next := add(cur, deltas[dir])
		if !inside(next) || p.Terrain[next.R][next.C] == Void {
			return struck, cur, len(struck) > 0
		}
		if p.Terrain[cur.R][cur.C] == Low && p.Terrain[next.R][next.C] == High {
			return struck, cur, len(struck) > 0
		}
		if _, blocked := occupied[next]; blocked {
			return struck, cur, len(struck) > 0
		}
		// Landing mode begins here, so no later walking boundary is attempted.
		if p.Terrain[cur.R][cur.C] == High && p.Terrain[next.R][next.C] == Low || p.Trampolines[next.R][next.C] {
			return nil, start, false
		}
		cur = next
	}
	return nil, start, false
}

func fly(p Puzzle, start Pos, dir Direction, distance int, occupied map[Pos]int, bugs []Bug) (Pos, bool, bool) {
	landing := start
	for i := 0; i < distance; i++ {
		landing = add(landing, deltas[dir])
	}
	return land(p, landing, dir, occupied, bugs)
}

func flyToPlatform(p Puzzle, start Pos, dir Direction, occupied map[Pos]int, bugs []Bug) (Pos, bool, bool) {
	landing := add(start, deltas[dir])
	for inside(landing) {
		if p.Terrain[landing.R][landing.C] != Void {
			return land(p, landing, dir, occupied, bugs)
		}
		landing = add(landing, deltas[dir])
	}
	return Pos{}, false, true
}

func land(p Puzzle, landing Pos, dir Direction, occupied map[Pos]int, bugs []Bug) (Pos, bool, bool) {
	for {
		if !inside(landing) || p.Terrain[landing.R][landing.C] == Void {
			return Pos{}, false, true
		}
		j, taken := occupied[landing]
		if !taken && !p.Trampolines[landing.R][landing.C] {
			return landing, true, false
		}
		if taken {
			bugs[j].Egg = false
			if bugs[j].Kind == Scorpion {
				bugs[j].Stunned = true
			}
		}
		landing = add(landing, deltas[dir])
	}
}

func spider(p Puzzle, bug Bug, dir Direction, occupied map[Pos]int, bugs []Bug) (Pos, bool, bool) {
	cur := bug.Pos
	for {
		if hasWall(p, cur, dir) {
			return cur, cur != bug.Pos, false
		}
		next := add(cur, deltas[dir])
		if !inside(next) || p.Terrain[next.R][next.C] == Void {
			return Pos{}, false, true
		}
		if p.Terrain[cur.R][cur.C] == High && p.Terrain[next.R][next.C] == Low {
			return land(p, next, dir, occupied, bugs)
		}
		if p.Terrain[cur.R][cur.C] == Low && p.Terrain[next.R][next.C] == High {
			return cur, cur != bug.Pos, false
		}
		if p.Trampolines[next.R][next.C] {
			return land(p, next, dir, occupied, bugs)
		}
		if _, blocked := occupied[next]; blocked {
			return cur, cur != bug.Pos, false
		}
		cur = next
	}
}

func beetle(p Puzzle, s *State, index int, dir Direction) (Pos, bool, bool) {
	start := s.Bugs[index].Pos
	if hasWall(p, start, dir) {
		return start, false, false
	}
	next := add(start, deltas[dir])
	if !inside(next) || p.Terrain[next.R][next.C] == Void {
		return Pos{}, false, true
	}
	if p.Terrain[start.R][start.C] == Low && p.Terrain[next.R][next.C] == High {
		return start, false, false
	}
	occ := occupancy(s.Bugs)
	if p.Terrain[start.R][start.C] == High && p.Terrain[next.R][next.C] == Low {
		delete(occ, start)
		return land(p, next, dir, occ, s.Bugs)
	}
	if _, taken := occ[next]; !taken {
		if p.Trampolines[next.R][next.C] {
			delete(occ, start)
			return land(p, next, dir, occ, s.Bugs)
		}
		return next, true, false
	}
	if p.Terrain[start.R][start.C] == High {
		return highBeetlePush(p, s, start, next, dir, occ)
	}
	// Find the whole contiguous chain and validate its destination before
	// changing anything. This is a low-level chain, so a high platform blocks it.
	chain := []int{}
	cur := next
	for {
		j, taken := occ[cur]
		if !taken {
			break
		}
		chain = append(chain, j)
		dest := add(cur, deltas[dir])
		if hasWall(p, cur, dir) {
			return start, false, false
		}
		if !inside(dest) || p.Terrain[dest.R][dest.C] == Void {
			if canFallWithoutLosing(s.Bugs[j]) {
				movePushChain(s, chain[:len(chain)-1], dir, cur)
				s.Bugs[j].Pos = Pos{-1, -1}
				return next, true, false
			}
			return Pos{}, false, true
		}
		if p.Terrain[cur.R][cur.C] == Low && p.Terrain[dest.R][dest.C] == High {
			return start, false, false
		}
		cur = dest
	}
	landing := cur
	if p.Trampolines[cur.R][cur.C] {
		var ok, fell bool
		landing, ok, fell = land(p, cur, dir, occ, s.Bugs)
		if fell || !ok {
			last := chain[len(chain)-1]
			if fell && canFallWithoutLosing(s.Bugs[last]) {
				movePushChain(s, chain[:len(chain)-1], dir, s.Bugs[last].Pos)
				s.Bugs[last].Pos = Pos{-1, -1}
				return next, true, false
			}
			return Pos{}, false, fell
		}
	}
	movePushChain(s, chain, dir, landing)
	return next, true, false
}

// highBeetlePush pushes only bugs standing at the beetle's high level. When
// the last one is pushed over an edge onto a low platform, that bug enters
// landing mode: it lands if the square is free or bounces over occupants.
func highBeetlePush(p Puzzle, s *State, start, next Pos, dir Direction, occ map[Pos]int) (Pos, bool, bool) {
	chain := []int{}
	cur := next
	for inside(cur) && p.Terrain[cur.R][cur.C] == High {
		j, taken := occ[cur]
		if !taken {
			landing := cur
			if p.Trampolines[cur.R][cur.C] {
				var ok, fell bool
				landing, ok, fell = land(p, cur, dir, occ, s.Bugs)
				if fell || !ok {
					return Pos{}, false, fell
				}
			}
			movePushChain(s, chain, dir, landing)
			return next, true, false
		}
		chain = append(chain, j)
		if hasWall(p, cur, dir) {
			return start, false, false
		}
		cur = add(cur, deltas[dir])
	}
	if len(chain) == 0 {
		// The adjacent bug is on a low platform, so it is not at the high
		// beetle's level and cannot be pushed.
		return start, false, false
	}
	landing, ok, fell := land(p, cur, dir, occ, s.Bugs)
	if fell || !ok {
		last := chain[len(chain)-1]
		if fell && canFallWithoutLosing(s.Bugs[last]) {
			movePushChain(s, chain[:len(chain)-1], dir, s.Bugs[last].Pos)
			s.Bugs[last].Pos = Pos{-1, -1}
			return next, true, false
		}
		return Pos{}, false, fell
	}
	for i := 0; i < len(chain)-1; i++ {
		s.Bugs[chain[i]].Pos = add(s.Bugs[chain[i]].Pos, deltas[dir])
	}
	s.Bugs[chain[len(chain)-1]].Pos = landing
	return next, true, false
}

func movePushChain(s *State, chain []int, dir Direction, lastLanding Pos) {
	if len(chain) == 0 {
		return
	}
	for i := 0; i < len(chain)-1; i++ {
		s.Bugs[chain[i]].Pos = add(s.Bugs[chain[i]].Pos, deltas[dir])
	}
	s.Bugs[chain[len(chain)-1]].Pos = lastLanding
}

func consumeStars(s *State) {
	kept := s.Bugs[:0]
	for _, b := range s.Bugs {
		stars := s.Stars[b.Pos.R][b.Pos.C]
		if b.Kind != Puck && b.Kind != Scorpion && !b.Egg && stars > 0 {
			if b.Kind == Ant {
				count := antCount(b)
				if count > stars {
					b.Count = count - stars
					s.Stars[b.Pos.R][b.Pos.C] = 0
					kept = append(kept, b)
				} else {
					s.Stars[b.Pos.R][b.Pos.C] -= count
				}
				continue
			}
			s.Stars[b.Pos.R][b.Pos.C]--
			continue
		}
		kept = append(kept, b)
	}
	s.Bugs = kept
}

func removeGoneEntities(s *State) {
	kept := s.Bugs[:0]
	for _, b := range s.Bugs {
		if b.Kind == Ant && b.Count == 0 {
			continue
		}
		if !canFallWithoutLosing(b) || inside(b.Pos) {
			kept = append(kept, b)
		}
	}
	s.Bugs = kept
}

func remainingBugCount(s State) int {
	count := 0
	for _, b := range s.Bugs {
		if b.Kind != Puck && b.Kind != Scorpion {
			count++
		}
	}
	return count
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
		if s.Bugs[i].Pos.C != s.Bugs[j].Pos.C {
			return s.Bugs[i].Pos.C < s.Bugs[j].Pos.C
		}
		if s.Bugs[i].ID != s.Bugs[j].ID {
			return s.Bugs[i].ID < s.Bugs[j].ID
		}
		return antCount(s.Bugs[i]) < antCount(s.Bugs[j])
	})
}

func stateKey(s State) string {
	return searchStateKey(s, true)
}

func searchStateKey(s State, includeWalls bool) string {
	var b strings.Builder
	for _, bug := range s.Bugs {
		b.WriteByte(byte(bug.Kind))
		b.WriteByte(byte(bug.Pos.R))
		b.WriteByte(byte(bug.Pos.C))
		if bug.Egg {
			b.WriteByte(1)
		} else {
			b.WriteByte(0)
		}
		b.WriteByte(antCount(bug))
		if bug.Kind == Scorpion {
			b.WriteByte(byte(bug.Direction))
			b.WriteByte(bug.ID)
		}
	}
	b.WriteByte('|')
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			if s.Stars[r][c] > 0 {
				b.WriteByte(byte(r*BoardSize + c + 1))
				b.WriteByte(s.Stars[r][c])
			}
		}
	}
	if includeWalls {
		b.WriteByte('|')
		for r := 0; r < BoardSize; r++ {
			for c := 0; c < BoardSize; c++ {
				b.WriteByte(s.Walls[r][c])
			}
		}
	}
	return b.String()
}

func antCount(bug Bug) uint8 {
	if bug.Kind != Ant {
		return 1
	}
	if bug.Count == 0 {
		return 1
	}
	return bug.Count
}

func add(a, b Pos) Pos  { return Pos{a.R + b.R, a.C + b.C} }
func inside(p Pos) bool { return p.R >= 0 && p.R < BoardSize && p.C >= 0 && p.C < BoardSize }

func hasWall(p Puzzle, from Pos, dir Direction) bool {
	return inside(from) && p.Walls[from.R][from.C]&(1<<dir) != 0
}

func knockDownWall(walls *[BoardSize][BoardSize]uint8, from Pos, dir Direction) {
	next := add(from, deltas[dir])
	walls[from.R][from.C] &^= 1 << dir
	if inside(next) {
		walls[next.R][next.C] &^= 1 << Direction((int(dir)+2)%4)
	}
}
