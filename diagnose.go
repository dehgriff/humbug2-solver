package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	diagnosisStateLimit = 2_000_000
	diagnosisTimeLimit  = 30 * time.Second
)

type puzzleDiagnosis struct {
	nodes            []searchNode
	best             int
	states           int
	deepest          int
	exhaustive       bool
	limitedByStates  bool
	limitedByTime    bool
	initialByKind    map[BugType]int
	minimumByKind    map[BugType]int
	positionsByKind  map[BugType]map[Pos]bool
	successfulByKind map[BugType]int
	minimumStars     [BoardSize][BoardSize]uint8
}

// DiagnosePuzzle explores the reachable state graph up to max-moves and
// reports the closest state plus evidence about bugs and stars that can never
// be cleared. It does not alter the puzzle.
func DiagnosePuzzle(p Puzzle, out io.Writer) error {
	d := diagnosePuzzle(p)
	best := d.nodes[d.best]
	remaining := remainingGoalUnits(best.state)
	stars := remainingStarCount(best.state)

	fmt.Fprintf(out, "Diagnosis: searched %d unique state(s) through depth %d", d.states, d.deepest)
	switch {
	case d.exhaustive:
		fmt.Fprintln(out, "; the reachable state graph was exhausted.")
	case d.limitedByStates:
		fmt.Fprintf(out, "; stopped at the %d-state safety limit.\n", diagnosisStateLimit)
	case d.limitedByTime:
		fmt.Fprintf(out, "; stopped at the %s time limit.\n", diagnosisTimeLimit)
	default:
		fmt.Fprintf(out, "; reached the puzzle's max-moves limit of %d.\n", p.MaxMoves)
	}

	if remaining == 0 {
		moves := pathToNode(d.nodes, d.best)
		fmt.Fprintf(out, "The puzzle is solvable in %d move(s).\n", len(moves))
		writeDiagnosticMoves(out, moves)
		return nil
	}

	moves := pathToNode(d.nodes, d.best)
	fmt.Fprintf(out, "Closest state: %d goal bug(s) and %d star(s) remain after %d move(s).\n", remaining, stars, len(moves))
	writeDiagnosticMoves(out, moves)
	printBoardState(out, p, best.state, "Closest reachable state")

	fmt.Fprintln(out, "\nRemaining goal bugs:")
	for _, bug := range best.state.Bugs {
		if bug.Kind == Puck || bug.Kind == Scorpion {
			continue
		}
		extra := ""
		if bug.Egg {
			extra = " (egg)"
		} else if bug.Kind == Ant && antCount(bug) > 1 {
			extra = fmt.Sprintf(" (group of %d)", antCount(bug))
		}
		fmt.Fprintf(out, "- %s at (%d,%d)%s\n", bug.Kind, bug.Pos.R+1, bug.Pos.C+1, extra)
	}
	fmt.Fprintln(out, "Remaining stars:")
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			if best.state.Stars[r][c] > 0 {
				fmt.Fprintf(out, "- (%d,%d): %d star(s), %s platform\n", r+1, c+1, best.state.Stars[r][c], terrainName(p.Terrain[r][c]))
			}
		}
	}

	kinds := sortedGoalKinds(d.initialByKind)
	fmt.Fprintln(out, "Likely cause:")
	foundBlocker := false
	for _, kind := range kinds {
		minimum := d.minimumByKind[kind]
		if minimum == 0 {
			continue
		}
		foundBlocker = true
		fmt.Fprintf(out, "- %s cannot be completely cleared: the search always leaves at least %d", kind, minimum)
		if !positionsIncludeInitialStar(d.positionsByKind[kind], p.Stars) {
			fmt.Fprint(out, ", and none of its reachable positions is an initial star square")
		}
		fmt.Fprintln(out, ".")
	}
	if !foundBlocker {
		fmt.Fprintln(out, "- Every bug type can score in some branch; the obstruction is the ordering or interaction between them rather than one individually unreachable type.")
	}

	fmt.Fprintln(out, "Evidence:")
	for _, kind := range kinds {
		minimum := d.minimumByKind[kind]
		if minimum > 0 {
			positions := sortedPositions(d.positionsByKind[kind])
			initial := d.initialByKind[kind]
			if minimum == initial {
				if initial == 1 {
					fmt.Fprintf(out, "- The %s never merges with a star and remains in every explored state.", kind)
				} else {
					fmt.Fprintf(out, "- No %s ever merges with a star; all %d remain in every explored state.", kind, initial)
				}
			} else {
				fmt.Fprintf(out, "- Some %s can score, but at least %d of the initial %d remain in every explored state.", kind, minimum, initial)
			}
			if len(positions) > 0 {
				fmt.Fprintf(out, " Reachable positions: %s.", formatPositions(positions, 16))
			}
			fmt.Fprintf(out, " Successful initiated moves observed: %d.\n", d.successfulByKind[kind])
		} else {
			fmt.Fprintf(out, "- %s can merge with a star in at least one branch, so it is not individually unreachable.\n", kind)
		}
	}
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			if p.Stars[r][c] > 0 && d.minimumStars[r][c] == p.Stars[r][c] {
				fmt.Fprintf(out, "- The star at (%d,%d) is never consumed in any explored state.\n", r+1, c+1)
			}
		}
	}
	if !d.exhaustive {
		fmt.Fprintln(out, "These are bounded-search findings, not a proof beyond the explored move/state limits.")
	}
	return nil
}

func diagnosePuzzle(p Puzzle) puzzleDiagnosis {
	start := initialState(p)
	d := puzzleDiagnosis{
		nodes: []searchNode{{state: start, parent: -1}}, best: 0, states: 1,
		initialByKind: make(map[BugType]int), minimumByKind: make(map[BugType]int),
		positionsByKind: make(map[BugType]map[Pos]bool), successfulByKind: make(map[BugType]int),
		minimumStars: start.Stars,
	}
	d.observe(start)
	includeWalls := hasGoldBeetle(p)
	seen := map[string]struct{}{searchStateKey(start, includeWalls): {}}
	frontier := []int{0}
	started := time.Now()
	for depth := 0; depth < p.MaxMoves && len(frontier) > 0; depth++ {
		nextFrontier := make([]int, 0)
		for _, parent := range frontier {
			state := d.nodes[parent].state
			for i, bug := range state.Bugs {
				if !canInitiateMove(bug) {
					continue
				}
				for _, dir := range directions {
					outcome := applyMoveDetailed(p, state, i, dir)
					if outcome.status != moveApplied || violatesMoveConstraints(p, bug, outcome.state) {
						continue
					}
					d.successfulByKind[bug.Kind]++
					key := searchStateKey(outcome.state, includeWalls)
					if _, exists := seen[key]; exists {
						continue
					}
					seen[key] = struct{}{}
					d.nodes = append(d.nodes, searchNode{state: outcome.state, parent: parent, move: outcome.move, depth: depth + 1})
					index := len(d.nodes) - 1
					nextFrontier = append(nextFrontier, index)
					d.states++
					d.deepest = depth + 1
					d.observe(outcome.state)
					if diagnosisStateBetter(outcome.state, d.nodes[d.best].state, depth+1, d.nodes[d.best].depth) {
						d.best = index
					}
					if d.states >= diagnosisStateLimit {
						d.limitedByStates = true
						return d
					}
				}
			}
			if time.Since(started) >= diagnosisTimeLimit {
				d.limitedByTime = true
				return d
			}
		}
		frontier = nextFrontier
	}
	d.exhaustive = len(frontier) == 0
	return d
}

func (d *puzzleDiagnosis) observe(state State) {
	counts := goalCountsByKind(state)
	if len(d.initialByKind) == 0 {
		for kind, count := range counts {
			d.initialByKind[kind] = count
			d.minimumByKind[kind] = count
		}
	}
	for kind := range d.initialByKind {
		if counts[kind] < d.minimumByKind[kind] {
			d.minimumByKind[kind] = counts[kind]
		}
	}
	for _, bug := range state.Bugs {
		if bug.Kind == Puck || bug.Kind == Scorpion {
			continue
		}
		if d.positionsByKind[bug.Kind] == nil {
			d.positionsByKind[bug.Kind] = make(map[Pos]bool)
		}
		d.positionsByKind[bug.Kind][bug.Pos] = true
	}
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			if state.Stars[r][c] < d.minimumStars[r][c] {
				d.minimumStars[r][c] = state.Stars[r][c]
			}
		}
	}
}

func goalCountsByKind(state State) map[BugType]int {
	counts := make(map[BugType]int)
	for _, bug := range state.Bugs {
		if bug.Kind != Puck && bug.Kind != Scorpion {
			counts[bug.Kind]++
		}
	}
	return counts
}

func remainingGoalUnits(state State) int {
	total := 0
	for _, bug := range state.Bugs {
		if bug.Kind == Puck || bug.Kind == Scorpion {
			continue
		}
		if bug.Kind == Ant {
			total += int(antCount(bug))
		} else {
			total++
		}
	}
	return total
}

func remainingStarCount(state State) int {
	total := 0
	for _, row := range state.Stars {
		for _, stars := range row {
			total += int(stars)
		}
	}
	return total
}

func diagnosisStateBetter(candidate, current State, candidateDepth, currentDepth int) bool {
	candidateBugs, currentBugs := remainingGoalUnits(candidate), remainingGoalUnits(current)
	if candidateBugs != currentBugs {
		return candidateBugs < currentBugs
	}
	candidateStars, currentStars := remainingStarCount(candidate), remainingStarCount(current)
	if candidateStars != currentStars {
		return candidateStars < currentStars
	}
	return candidateDepth < currentDepth
}

func pathToNode(nodes []searchNode, index int) []Move {
	path := make([]Move, nodes[index].depth)
	for i := len(path) - 1; i >= 0; i-- {
		path[i] = nodes[index].move
		index = nodes[index].parent
	}
	return path
}

func writeDiagnosticMoves(out io.Writer, moves []Move) {
	if len(moves) == 0 {
		fmt.Fprintln(out, "Moves to closest state: none")
		return
	}
	fmt.Fprintln(out, "Moves to closest state:")
	for i, move := range moves {
		fmt.Fprintf(out, "%d. %s\n", i+1, move)
	}
}

func sortedGoalKinds(counts map[BugType]int) []BugType {
	kinds := make([]BugType, 0, len(counts))
	for kind := range counts {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })
	return kinds
}

func sortedPositions(set map[Pos]bool) []Pos {
	positions := make([]Pos, 0, len(set))
	for pos := range set {
		positions = append(positions, pos)
	}
	sort.Slice(positions, func(i, j int) bool {
		return positions[i].R < positions[j].R || positions[i].R == positions[j].R && positions[i].C < positions[j].C
	})
	return positions
}

func formatPositions(positions []Pos, limit int) string {
	parts := make([]string, 0, min(len(positions), limit))
	for i, pos := range positions {
		if i == limit {
			parts = append(parts, fmt.Sprintf("and %d more", len(positions)-limit))
			break
		}
		parts = append(parts, fmt.Sprintf("(%d,%d)", pos.R+1, pos.C+1))
	}
	return strings.Join(parts, ", ")
}

func terrainName(terrain Terrain) string {
	if terrain == High {
		return "high"
	}
	return "low"
}

func positionsIncludeInitialStar(positions map[Pos]bool, stars [BoardSize][BoardSize]uint8) bool {
	for pos := range positions {
		if stars[pos.R][pos.C] > 0 {
			return true
		}
	}
	return false
}
