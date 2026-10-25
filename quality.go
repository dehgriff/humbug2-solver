package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const analysisVersion = 3

// PuzzleAnalysis combines a replay of the supplied solution with bounded
// searches of the alternative moves encountered along it. It never mutates
// the puzzle or tries removing board features. ANALYSIS.md defines every field
// and the quality and difficulty formulas which consume them.
type PuzzleAnalysis struct {
	Quality, Tightness, Interactions, AbilityUse, Manageability, SetupBalance int
	Difficulty                                                                float64
	DifficultyDepth, DifficultyChoices, DifficultyObjects                     int
	DifficultySetup, DifficultyMechanics, DifficultyScorpions                 int
	DifficultyDecisionLandscape, DifficultySequencing                         int
	Moves, Objects, Platforms, VisitedPlatforms                               int
	OpeningLegal, MaximumLegal                                                int
	AverageLegal                                                              float64
	ScoringMoves, SetupMoves, InteractionMoves                                int
	GoalObjects, MovesBeforeFirstScore, LongestSetupRun, MoverTypeSwitches    int
	AverageMovesPerGoal                                                       float64
	MechanicsUsed, MechanicOpportunities                                      int
	EggsHatched, WallsDestroyed                                               int
	PeakActiveScorpions, ScorpionSteps, ScorpionTurns, MultiScorpionTurns     int
	ScorpionDirectionChanges                                                  int
	AverageActiveScorpions                                                    float64
	OptimalMoves, DetourMoves, DeadEndMoves                                   int
	UnresolvedMoves                                                           int
	BranchingSteps, ForcedChoiceSteps, MultipleOptimalSteps, MisleadingSteps  int
	DetourPenalty, MaximumDetourPenalty, LandscapeStates                      int
	AverageOptimalMoves                                                       float64
	AnalysisTimedOut                                                          bool
	Mechanics                                                                 []string
}

func AnalyzePuzzle(p Puzzle, moves []Move) (PuzzleAnalysis, error) {
	a := PuzzleAnalysis{Moves: len(moves), Objects: len(p.Bugs)}
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			if p.Terrain[r][c] != Void {
				a.Platforms++
			}
		}
	}
	opportunities := analysisOpportunities(p)
	used := make(map[string]bool)
	visited := make(map[Pos]bool)
	state := initialState(p)
	a.GoalObjects = remainingBugCount(state)
	states := []State{state}
	markOccupiedPlatforms(p, state, visited)
	legalTotal := 0
	activeScorpionTotal := 0
	currentSetupRun := 0
	firstScoreSeen := false
	var previousMoverType BugType
	hasPreviousMover := false
	for step, move := range moves {
		if hasPreviousMover && move.Kind != previousMoverType {
			a.MoverTypeSwitches++
		}
		previousMoverType = move.Kind
		hasPreviousMover = true
		legal := countSafeMoves(p, state)
		if step == 0 {
			a.OpeningLegal = legal
		}
		legalTotal += legal
		if legal > a.MaximumLegal {
			a.MaximumLegal = legal
		}
		index := bugAt(state.Bugs, move.From)
		if index < 0 || state.Bugs[index].Kind != move.Kind {
			return PuzzleAnalysis{}, fmt.Errorf("move %d does not match the replayed state", step+1)
		}
		before := state
		activeScorpions := activeScorpionCount(before.Bugs)
		activeScorpionTotal += activeScorpions
		if activeScorpions > a.PeakActiveScorpions {
			a.PeakActiveScorpions = activeScorpions
		}
		outcome := applyMoveDetailed(p, state, index, move.Dir)
		if outcome.status != moveApplied {
			return PuzzleAnalysis{}, fmt.Errorf("move %d cannot be replayed: %s", step+1, outcome.reason)
		}
		state = outcome.state
		states = append(states, state)
		scorpionSteps, directionChanges := scorpionChanges(before.Bugs, state.Bugs)
		a.ScorpionSteps += scorpionSteps
		a.ScorpionDirectionChanges += directionChanges
		if scorpionSteps > 0 {
			a.ScorpionTurns++
		}
		if scorpionSteps > 1 {
			a.MultiScorpionTurns++
		}
		markMovePath(p, outcome.move, visited)
		markOccupiedPlatforms(p, state, visited)

		starsBefore, starsAfter := starCount(before.Stars), starCount(state.Stars)
		if starsAfter < starsBefore {
			a.ScoringMoves++
			if !firstScoreSeen {
				a.MovesBeforeFirstScore = step
				firstScoreSeen = true
			}
			currentSetupRun = 0
		} else {
			a.SetupMoves++
			currentSetupRun++
			if currentSetupRun > a.LongestSetupRun {
				a.LongestSetupRun = currentSetupRun
			}
		}
		eggs := eggCount(before.Bugs) - eggCount(state.Bugs)
		if eggs > 0 {
			a.EggsHatched += eggs
			used["egg-hatching"] = true
		}
		walls := wallCount(before.Walls) - wallCount(state.Walls)
		if walls > 0 {
			a.WallsDestroyed += walls
			used["wall-special"] = true
		}
		passive := passiveStateChanged(before.Bugs, state.Bugs, outcome.move)
		if passive || eggs > 0 || walls > 0 {
			a.InteractionMoves++
		}
		classifyMechanics(p, before, state, outcome.move, passive, used)
	}
	if remainingBugCount(state) != 0 {
		return PuzzleAnalysis{}, fmt.Errorf("solution ends after %d move(s) but the puzzle is not solved", len(moves))
	}
	if !firstScoreSeen {
		a.MovesBeforeFirstScore = a.Moves
	}

	a.VisitedPlatforms = len(visited)
	if a.Platforms > 0 {
		a.Tightness = percent(a.VisitedPlatforms, a.Platforms)
	}
	if a.Moves > 0 {
		a.AverageLegal = float64(legalTotal) / float64(a.Moves)
		a.AverageActiveScorpions = float64(activeScorpionTotal) / float64(a.Moves)
		a.Interactions = percent(a.InteractionMoves, a.Moves)
		a.SetupBalance = setupBalance(float64(a.SetupMoves) / float64(a.Moves))
	} else {
		a.SetupBalance = 40
	}
	if a.GoalObjects > 0 {
		a.AverageMovesPerGoal = float64(a.Moves) / float64(a.GoalObjects)
	}
	for mechanic := range used {
		if opportunities[mechanic] {
			a.Mechanics = append(a.Mechanics, mechanic)
		}
	}
	sort.Strings(a.Mechanics)
	a.MechanicsUsed = len(a.Mechanics)
	a.MechanicOpportunities = len(opportunities)
	if a.MechanicOpportunities == 0 {
		a.AbilityUse = 50
	} else {
		a.AbilityUse = percent(a.MechanicsUsed, a.MechanicOpportunities)
	}
	a.Manageability = manageability(a.Moves, a.Objects)
	a.Quality = int(math.Round(.30*float64(a.Tightness) + .25*float64(a.Interactions) + .20*float64(a.AbilityUse) + .15*float64(a.Manageability) + .10*float64(a.SetupBalance)))

	a.DifficultyDepth = int(math.Round(clamp(float64(a.Moves-3)/27*100, 0, 100)))
	a.DifficultyChoices = int(math.Round(clamp((a.AverageLegal-1)/7*100, 0, 100)))
	a.DifficultyObjects = int(math.Round(clamp(float64(a.Objects-1)/9*100, 0, 100)))
	if a.Moves > 0 {
		a.DifficultySetup = int(math.Round(float64(a.SetupMoves) / float64(a.Moves) * 100))
	}
	a.DifficultyMechanics = int(math.Round(clamp(float64(a.MechanicsUsed)/5*100, 0, 100)))
	if a.Moves > 0 {
		setupLoad := float64(a.DifficultySetup)
		delayedScoring := float64(a.MovesBeforeFirstScore) / float64(a.Moves) * 100
		longestSetup := float64(a.LongestSetupRun) / float64(a.Moves) * 100
		moveLoad := clamp((a.AverageMovesPerGoal-1)/5*100, 0, 100)
		switchLoad := 0.0
		if a.Moves > 1 {
			switchLoad = float64(a.MoverTypeSwitches) / float64(a.Moves-1) * 100
		}
		a.DifficultySequencing = int(math.Round(clamp(.25*setupLoad+.25*delayedScoring+.25*moveLoad+.15*switchLoad+.10*longestSetup, 0, 100)))
	}
	if a.PeakActiveScorpions > 0 && a.Moves > 0 {
		peakLoad := clamp(float64(a.PeakActiveScorpions)/4*100, 0, 100)
		averageLoad := clamp(a.AverageActiveScorpions/2*100, 0, 100)
		movementCoverage := float64(a.ScorpionTurns) / float64(a.Moves) * 100
		multiCoverage := float64(a.MultiScorpionTurns) / float64(a.Moves) * 100
		turningLoad := 0.0
		if a.ScorpionSteps > 0 {
			turningLoad = float64(a.ScorpionDirectionChanges) / float64(a.ScorpionSteps) * 100
		}
		a.DifficultyScorpions = int(math.Round(clamp(.30*peakLoad+.30*averageLoad+.20*movementCoverage+.20*multiCoverage+.10*turningLoad, 0, 100)))
	}
	analyzeDecisionLandscape(p, states, &a)
	rawDifficulty := .25*float64(a.DifficultyDepth) + .10*float64(a.DifficultyChoices) + .10*float64(a.DifficultyObjects) + .10*float64(a.DifficultySetup) + .10*float64(a.DifficultyMechanics)
	// Autonomous movement is extra cognitive work rather than another form of
	// ordinary branching. Add up to two difficulty points without reducing
	// the score of otherwise identical puzzles that contain no scorpions.
	rawDifficulty += .20 * float64(a.DifficultyScorpions)
	// The decision landscape is the strongest branch-analysis signal: it measures whether
	// plausible alternatives preserve an optimal solution, cause a detour, or
	// become dead ends within the puzzle's move allowance.
	rawDifficulty += .35 * float64(a.DifficultyDecisionLandscape)
	// Long sequences of non-scoring repositioning and repeated coordination
	// between bug types are an additional human planning burden.
	rawDifficulty += .25 * float64(a.DifficultySequencing)
	a.Difficulty = math.Round(clamp(rawDifficulty/10, 1, 10)*10) / 10
	return a, nil
}

type landscapeDistance struct {
	exact         int
	hasExact      bool
	failedThrough int
}

type landscapeSearch struct {
	p              Puzzle
	includeWalls   bool
	useHeuristic   bool
	successors     map[string][]successor
	distances      map[string]landscapeDistance
	expandedStates int
	deadline       time.Time
	timedOut       bool
}

const analysisStateLimit = 2_000_000
const analysisTimeLimit = 30 * time.Second

type landscapeStatus byte

const (
	landscapeUnsolved landscapeStatus = iota
	landscapeSolved
	landscapeUnresolved
)

func analyzeDecisionLandscape(p Puzzle, states []State, a *PuzzleAnalysis) {
	if len(states) != a.Moves+1 || a.Moves == 0 {
		return
	}
	search := landscapeSearch{
		p:            p,
		includeWalls: hasGoldBeetle(p),
		// With a pushing Beetle, avoid the solver's simplifying lower-bound
		// assumption so this classification remains exact.
		useHeuristic: !hasPushingBeetle(p),
		successors:   make(map[string][]successor),
		distances:    make(map[string]landscapeDistance),
		deadline:     time.Now().Add(analysisTimeLimit),
	}
	// Every suffix of a globally shortest solution is itself shortest. Seeding
	// these distances avoids re-searching the selected route and alternatives
	// which converge back onto it.
	for step, state := range states {
		key := searchStateKey(state, search.includeWalls)
		search.distances[key] = landscapeDistance{exact: a.Moves - step, hasExact: true}
	}

	optimalTotal := 0
	for step := 0; step < a.Moves; step++ {
		state := states[step]
		key := searchStateKey(state, search.includeWalls)
		successors := cachedSuccessors(p, state, key, search.includeWalls, search.successors)
		if len(successors) > 1 {
			a.BranchingSteps++
		}
		optimalRemaining := a.Moves - step - 1
		allowedRemaining := p.MaxMoves - step - 1
		if allowedRemaining < optimalRemaining {
			allowedRemaining = optimalRemaining
		}
		optimalHere := 0
		badHere := 0
		for _, next := range successors {
			distance, status := search.shortestDistance(next.state, next.key, allowedRemaining)
			switch {
			case status == landscapeSolved && distance == optimalRemaining:
				a.OptimalMoves++
				optimalHere++
			case status == landscapeSolved:
				a.DetourMoves++
				badHere++
				penalty := distance - optimalRemaining
				if penalty > 0 {
					a.DetourPenalty += penalty
					if penalty > a.MaximumDetourPenalty {
						a.MaximumDetourPenalty = penalty
					}
				}
			case status == landscapeUnsolved:
				a.DeadEndMoves++
				badHere++
			default:
				a.UnresolvedMoves++
			}
		}
		optimalTotal += optimalHere
		if len(successors) > 1 && optimalHere == 1 {
			a.ForcedChoiceSteps++
		}
		if optimalHere > 1 {
			a.MultipleOptimalSteps++
		}
		if len(successors) > 1 && badHere > 0 {
			a.MisleadingSteps++
		}
	}
	a.LandscapeStates = search.expandedStates
	a.AnalysisTimedOut = search.timedOut
	a.AverageOptimalMoves = float64(optimalTotal) / float64(a.Moves)

	totalMoves := a.OptimalMoves + a.DetourMoves + a.DeadEndMoves
	badMoves := a.DetourMoves + a.DeadEndMoves
	badRatio := 0.0
	deadEndRatio := 0.0
	if totalMoves > 0 {
		badRatio = float64(badMoves) / float64(totalMoves) * 100
	}
	if badMoves > 0 {
		deadEndRatio = float64(a.DeadEndMoves) / float64(badMoves) * 100
	}
	forcedRatio := 0.0
	misleadingRatio := 0.0
	if a.BranchingSteps > 0 {
		forcedRatio = float64(a.ForcedChoiceSteps) / float64(a.BranchingSteps) * 100
		misleadingRatio = float64(a.MisleadingSteps) / float64(a.BranchingSteps) * 100
	}
	detourSeverity := clamp(float64(a.MaximumDetourPenalty)/6*100, 0, 100)
	unresolvedRatio := 0.0
	allMoves := totalMoves + a.UnresolvedMoves
	if allMoves > 0 {
		unresolvedRatio = float64(a.UnresolvedMoves) / float64(allMoves) * 100
	}
	searchEffort := clamp(math.Log10(float64(a.LandscapeStates)+1)/math.Log10(float64(analysisStateLimit)+1)*100, 0, 100)
	timeoutBonus := 0.0
	if a.AnalysisTimedOut {
		timeoutBonus = 10
	}
	// Unresolved choices represent uncertainty rather than proven bad moves,
	// but they must not make an expensive puzzle appear artificially easy.
	a.DifficultyDecisionLandscape = int(math.Round(clamp(.30*badRatio+.20*forcedRatio+.15*misleadingRatio+.10*deadEndRatio+.10*detourSeverity+.30*unresolvedRatio+.05*searchEffort+timeoutBonus, 0, 100)))
}

func (s *landscapeSearch) shortestDistance(state State, key string, maximum int) (int, landscapeStatus) {
	if s.limitReached() {
		return 0, landscapeUnresolved
	}
	if remainingBugCount(state) == 0 {
		return 0, landscapeSolved
	}
	if maximum < 1 {
		return 0, landscapeUnsolved
	}
	record := s.distances[key]
	if record.hasExact {
		if record.exact <= maximum {
			return record.exact, landscapeSolved
		}
		return 0, landscapeUnsolved
	}
	lower := 1
	if s.useHeuristic {
		lower = searchLowerBound(state)
	}
	if record.failedThrough >= lower {
		lower = record.failedThrough + 1
	}
	for bound := lower; bound <= maximum; bound++ {
		bestDepth := map[string]int{key: 0}
		status := s.reachableAtDepth(state, key, 0, bound, bestDepth)
		if status == landscapeUnresolved {
			return 0, landscapeUnresolved
		}
		if status == landscapeSolved {
			s.distances[key] = landscapeDistance{exact: bound, hasExact: true}
			return bound, landscapeSolved
		}
		record.failedThrough = bound
		s.distances[key] = record
	}
	return 0, landscapeUnsolved
}

func (s *landscapeSearch) reachableAtDepth(state State, key string, depth, bound int, bestDepth map[string]int) landscapeStatus {
	if s.limitReached() {
		return landscapeUnresolved
	}
	s.expandedStates++
	for _, next := range cachedSuccessors(s.p, state, key, s.includeWalls, s.successors) {
		if depth+1 > bound || s.useHeuristic && depth+1+next.h > bound {
			continue
		}
		if remainingBugCount(next.state) == 0 {
			return landscapeSolved
		}
		nextDepth := depth + 1
		if oldDepth, exists := bestDepth[next.key]; exists && oldDepth <= nextDepth {
			continue
		}
		bestDepth[next.key] = nextDepth
		status := s.reachableAtDepth(next.state, next.key, nextDepth, bound, bestDepth)
		if status != landscapeUnsolved {
			return status
		}
	}
	return landscapeUnsolved
}

func (s *landscapeSearch) limitReached() bool {
	if s.expandedStates >= analysisStateLimit {
		return true
	}
	if s.timedOut || time.Now().After(s.deadline) {
		s.timedOut = true
		return true
	}
	return false
}

func countSafeMoves(p Puzzle, state State) int {
	count := 0
	for i, bug := range state.Bugs {
		if !canInitiateMove(bug) {
			continue
		}
		for _, dir := range directions {
			outcome := applyMoveDetailed(p, state, i, dir)
			if outcome.status == moveApplied && !violatesMoveConstraints(p, bug, outcome.state) {
				count++
			}
		}
	}
	return count
}

func analysisOpportunities(p Puzzle) map[string]bool {
	m := make(map[string]bool)
	for _, bug := range p.Bugs {
		if bug.Egg {
			m["egg-hatching"] = true
		}
		switch bug.Kind {
		case Ant:
			m["ant-merging"] = true
		case Beetle:
			m["beetle-pushing"] = true
		case Caterpillar:
			m["caterpillar-nudging"] = true
		case Puck:
			m["puck-use"] = true
		case Scorpion:
			m["scorpion-action"] = true
		case GoldBeetle, Cockroach:
			m["wall-special"] = true
		case Bee, Grasshopper, Butterfly, Fly:
			m["flight"] = true
		case Ladybird, PinkLadybird:
			m["distance-adjustment"] = true
		}
	}
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			if p.Trampolines[r][c] {
				m["trampoline-use"] = true
			}
		}
	}
	return m
}

func classifyMechanics(p Puzzle, before, after State, move Move, passive bool, used map[string]bool) {
	if passive {
		switch move.Kind {
		case Beetle:
			used["beetle-pushing"] = true
		}
	}
	if kindChanged(before.Bugs, after.Bugs, Scorpion) {
		used["scorpion-action"] = true
	}
	if kindChanged(before.Bugs, after.Bugs, Puck) || pathContainsKind(before.Bugs, move, Puck) {
		used["puck-use"] = true
	}
	if move.Kind != Caterpillar && kindChanged(before.Bugs, after.Bugs, Caterpillar) {
		used["caterpillar-nudging"] = true
	}
	if antGroups(after.Bugs) > antGroups(before.Bugs) {
		used["ant-merging"] = true
	}
	if pathContainsTrampoline(p, move) {
		used["trampoline-use"] = true
	}
	distance := abs(move.To.R-move.From.R) + abs(move.To.C-move.From.C)
	switch move.Kind {
	case Ladybird:
		if distance != 2 {
			used["distance-adjustment"] = true
		}
	case PinkLadybird:
		if distance != 3 {
			used["distance-adjustment"] = true
		}
	case Bee, Grasshopper, Butterfly, Fly:
		if pathHasSpecialTraversal(p, before, move) {
			used["flight"] = true
		}
	case Cockroach:
		if pathCrossesWall(before.Walls, move) {
			used["wall-special"] = true
		}
	}
}

func markOccupiedPlatforms(p Puzzle, state State, visited map[Pos]bool) {
	for _, b := range state.Bugs {
		if inside(b.Pos) && p.Terrain[b.Pos.R][b.Pos.C] != Void {
			visited[b.Pos] = true
		}
	}
}
func markMovePath(p Puzzle, move Move, visited map[Pos]bool) {
	walkLine(move, func(pos Pos) {
		if inside(pos) && p.Terrain[pos.R][pos.C] != Void {
			visited[pos] = true
		}
	})
}
func walkLine(move Move, fn func(Pos)) {
	pos := move.From
	fn(pos)
	for pos != move.To {
		next := add(pos, deltas[move.Dir])
		if !inside(next) {
			return
		}
		pos = next
		fn(pos)
	}
}
func pathContainsTrampoline(p Puzzle, move Move) bool {
	found := false
	walkLine(move, func(pos Pos) { found = found || p.Trampolines[pos.R][pos.C] })
	return found
}
func pathContainsKind(bugs []Bug, move Move, kind BugType) bool {
	found := false
	first := true
	walkLine(move, func(pos Pos) {
		if first {
			first = false
			return
		}
		for _, b := range bugs {
			if b.Pos == pos && b.Kind == kind {
				found = true
			}
		}
	})
	return found
}
func pathHasSpecialTraversal(p Puzzle, state State, move Move) bool {
	special := false
	first := true
	walkLine(move, func(pos Pos) {
		if first {
			first = false
			return
		}
		if p.Terrain[pos.R][pos.C] == Void {
			special = true
		}
		for _, b := range state.Bugs {
			if b.Pos == pos {
				special = true
			}
		}
	})
	return special || pathCrossesWall(state.Walls, move)
}
func pathCrossesWall(walls [BoardSize][BoardSize]uint8, move Move) bool {
	crossed := false
	pos := move.From
	for pos != move.To {
		if walls[pos.R][pos.C]&(1<<move.Dir) != 0 {
			crossed = true
		}
		next := add(pos, deltas[move.Dir])
		if !inside(next) {
			break
		}
		pos = next
	}
	return crossed
}

func starCount(stars [BoardSize][BoardSize]uint8) int {
	n := 0
	for _, row := range stars {
		for _, v := range row {
			n += int(v)
		}
	}
	return n
}
func eggCount(bugs []Bug) int {
	n := 0
	for _, b := range bugs {
		if b.Egg {
			n += int(antCount(b))
		}
	}
	return n
}
func antGroups(bugs []Bug) int {
	n := 0
	for _, b := range bugs {
		if b.Kind == Ant && antCount(b) > 1 {
			n++
		}
	}
	return n
}

func activeScorpionCount(bugs []Bug) int {
	n := 0
	for _, bug := range bugs {
		if bug.Kind == Scorpion && !bug.Egg && inside(bug.Pos) {
			n++
		}
	}
	return n
}

func scorpionChanges(before, after []Bug) (steps, directionChanges int) {
	remaining := make(map[uint8]Bug)
	for _, bug := range after {
		if bug.Kind == Scorpion {
			remaining[bug.ID] = bug
		}
	}
	for _, bug := range before {
		if bug.Kind != Scorpion || bug.Egg {
			continue
		}
		next, exists := remaining[bug.ID]
		if !exists || next.Pos != bug.Pos {
			steps++
		}
		if exists && next.Direction != bug.Direction {
			directionChanges++
		}
	}
	return steps, directionChanges
}
func wallCount(walls [BoardSize][BoardSize]uint8) int {
	n := 0
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			if walls[r][c]&(1<<Right) != 0 {
				n++
			}
			if walls[r][c]&(1<<Down) != 0 {
				n++
			}
			if c == 0 && walls[r][c]&(1<<Left) != 0 {
				n++
			}
			if r == 0 && walls[r][c]&(1<<Up) != 0 {
				n++
			}
		}
	}
	return n
}

func passiveStateChanged(before, after []Bug, move Move) bool {
	a := append([]Bug(nil), before...)
	b := append([]Bug(nil), after...)
	a = removeAnalysisBug(a, move.From, move.Kind)
	b = removeAnalysisBug(b, move.To, move.Kind)
	return bugMultiset(a) != bugMultiset(b)
}
func removeAnalysisBug(bugs []Bug, pos Pos, kind BugType) []Bug {
	for i, b := range bugs {
		if b.Pos == pos && b.Kind == kind {
			return append(bugs[:i], bugs[i+1:]...)
		}
	}
	return bugs
}
func bugMultiset(bugs []Bug) string {
	keys := make([]string, len(bugs))
	for i, b := range bugs {
		keys[i] = fmt.Sprintf("%d/%d/%d/%t/%d/%d/%d", b.Kind, b.Pos.R, b.Pos.C, b.Egg, b.Count, b.Direction, b.ID)
	}
	sort.Strings(keys)
	return strings.Join(keys, "|")
}
func kindChanged(before, after []Bug, kind BugType) bool {
	var a, b []Bug
	for _, x := range before {
		if x.Kind == kind {
			a = append(a, x)
		}
	}
	for _, x := range after {
		if x.Kind == kind {
			b = append(b, x)
		}
	}
	return bugMultiset(a) != bugMultiset(b)
}

func percent(part, whole int) int {
	if whole == 0 {
		return 0
	}
	return int(math.Round(float64(part) * 100 / float64(whole)))
}
func clamp(v, low, high float64) float64 { return math.Max(low, math.Min(high, v)) }
func setupBalance(r float64) int         { return int(math.Round(clamp(100-math.Abs(r-.60)*100, 40, 100))) }
func manageability(moves, objects int) int {
	moveScore := 100.0
	if moves < 8 {
		moveScore = 20 + float64(moves)*10
	} else if moves > 25 {
		moveScore = clamp(100-float64(moves-25)*3.3, 50, 100)
	}
	objectScore := 100.0
	if objects < 3 {
		objectScore = 40 + float64(objects)*20
	} else if objects > 8 {
		objectScore = 100 - float64(objects-8)*15
	}
	return int(math.Round((moveScore + objectScore) / 2))
}

func (a PuzzleAnalysis) lines() []string {
	mechanics := "none"
	if len(a.Mechanics) > 0 {
		mechanics = strings.Join(a.Mechanics, ",")
	}
	return []string{
		fmt.Sprintf("version %d", analysisVersion), fmt.Sprintf("quality %d", a.Quality), fmt.Sprintf("difficulty %.1f", a.Difficulty),
		fmt.Sprintf("moves %d", a.Moves), fmt.Sprintf("objects %d", a.Objects), fmt.Sprintf("platforms %d", a.Platforms),
		fmt.Sprintf("visited-platforms %d", a.VisitedPlatforms), fmt.Sprintf("tightness %d", a.Tightness),
		fmt.Sprintf("average-legal-moves %.2f", a.AverageLegal), fmt.Sprintf("opening-legal-moves %d", a.OpeningLegal), fmt.Sprintf("maximum-legal-moves %d", a.MaximumLegal),
		fmt.Sprintf("scoring-moves %d", a.ScoringMoves), fmt.Sprintf("setup-moves %d", a.SetupMoves), fmt.Sprintf("interaction-moves %d", a.InteractionMoves),
		fmt.Sprintf("sequencing-load %d", a.DifficultySequencing), fmt.Sprintf("goal-objects %d", a.GoalObjects), fmt.Sprintf("moves-per-goal-object %.2f", a.AverageMovesPerGoal),
		fmt.Sprintf("moves-before-first-score %d", a.MovesBeforeFirstScore), fmt.Sprintf("longest-setup-run %d", a.LongestSetupRun), fmt.Sprintf("mover-type-switches %d", a.MoverTypeSwitches),
		fmt.Sprintf("mechanics-used %d/%d", a.MechanicsUsed, a.MechanicOpportunities), fmt.Sprintf("mechanics %s", mechanics),
		fmt.Sprintf("eggs-hatched %d", a.EggsHatched), fmt.Sprintf("walls-destroyed %d", a.WallsDestroyed),
		fmt.Sprintf("scorpion-load %d", a.DifficultyScorpions), fmt.Sprintf("peak-active-scorpions %d", a.PeakActiveScorpions), fmt.Sprintf("average-active-scorpions %.2f", a.AverageActiveScorpions),
		fmt.Sprintf("scorpion-steps %d", a.ScorpionSteps), fmt.Sprintf("scorpion-turns %d", a.ScorpionTurns), fmt.Sprintf("multi-scorpion-turns %d", a.MultiScorpionTurns), fmt.Sprintf("scorpion-direction-changes %d", a.ScorpionDirectionChanges),
		fmt.Sprintf("decision-landscape %d", a.DifficultyDecisionLandscape), fmt.Sprintf("optimal-moves %d", a.OptimalMoves), fmt.Sprintf("detour-moves %d", a.DetourMoves), fmt.Sprintf("dead-end-moves %d", a.DeadEndMoves), fmt.Sprintf("unresolved-moves %d", a.UnresolvedMoves),
		fmt.Sprintf("branching-steps %d", a.BranchingSteps), fmt.Sprintf("forced-choice-steps %d", a.ForcedChoiceSteps), fmt.Sprintf("multiple-optimal-steps %d", a.MultipleOptimalSteps), fmt.Sprintf("misleading-steps %d", a.MisleadingSteps),
		fmt.Sprintf("average-optimal-moves %.2f", a.AverageOptimalMoves), fmt.Sprintf("total-detour-penalty %d", a.DetourPenalty), fmt.Sprintf("maximum-detour-penalty %d", a.MaximumDetourPenalty), fmt.Sprintf("landscape-states-searched %d", a.LandscapeStates),
		fmt.Sprintf("analysis-timed-out %t", a.AnalysisTimedOut),
		fmt.Sprintf("quality-components tightness=%d interactions=%d ability-use=%d manageability=%d setup-balance=%d", a.Tightness, a.Interactions, a.AbilityUse, a.Manageability, a.SetupBalance),
		fmt.Sprintf("difficulty-components depth=%d choices=%d objects=%d setup=%d mechanics=%d scorpions=%d decision-landscape=%d sequencing=%d", a.DifficultyDepth, a.DifficultyChoices, a.DifficultyObjects, a.DifficultySetup, a.DifficultyMechanics, a.DifficultyScorpions, a.DifficultyDecisionLandscape, a.DifficultySequencing),
	}
}

func printPuzzleAnalysis(out io.Writer, a PuzzleAnalysis) {
	fmt.Fprintf(out, "Quality: %d/100  Difficulty: %.1f/10\n", a.Quality, a.Difficulty)
	fmt.Fprintf(out, "Tightness: %d/%d platforms visited (%d%%); interactions: %d/%d moves; safe choices: %.2f average\n", a.VisitedPlatforms, a.Platforms, a.Tightness, a.InteractionMoves, a.Moves, a.AverageLegal)
	fmt.Fprintf(out, "Mechanics used: %d/%d (%s)\n", a.MechanicsUsed, a.MechanicOpportunities, func() string {
		if len(a.Mechanics) == 0 {
			return "none"
		}
		return strings.Join(a.Mechanics, ", ")
	}())
	if a.PeakActiveScorpions > 0 {
		fmt.Fprintf(out, "Scorpion load: %d/100; peak %d active, %.2f active on average, %d autonomous step(s)\n", a.DifficultyScorpions, a.PeakActiveScorpions, a.AverageActiveScorpions, a.ScorpionSteps)
	}
	fmt.Fprintf(out, "Sequencing load: %d/100; %d setup move(s), first score after %d move(s), %.2f moves per goal object\n", a.DifficultySequencing, a.SetupMoves, a.MovesBeforeFirstScore, a.AverageMovesPerGoal)
	fmt.Fprintf(out, "Decision landscape: %d/100; %d optimal, %d detour, %d dead-end, %d unresolved move(s); %d state(s) searched\n", a.DifficultyDecisionLandscape, a.OptimalMoves, a.DetourMoves, a.DeadEndMoves, a.UnresolvedMoves, a.LandscapeStates)
	if a.AnalysisTimedOut {
		fmt.Fprintf(out, "Decision analysis stopped after the %s time limit; unresolved moves were not treated as dead ends.\n", analysisTimeLimit)
	}
}

func writePuzzleAnalysis(path string, a PuzzleAnalysis) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	solutionAt := solutionSectionOffset(data)
	analysisAt := firstSectionOffset(data, "analysis")
	prefixEnd := len(data)
	if solutionAt >= 0 {
		prefixEnd = solutionAt
	}
	if analysisAt >= 0 && analysisAt < prefixEnd {
		prefixEnd = analysisAt
	}
	prefix := data[:prefixEnd]
	var suffix []byte
	if solutionAt >= 0 {
		suffix = data[solutionAt:]
	}
	var text strings.Builder
	text.Write(prefix)
	if text.Len() > 0 && !strings.HasSuffix(text.String(), "\n") {
		text.WriteByte('\n')
	}
	text.WriteString("analysis:\n")
	for _, line := range a.lines() {
		text.WriteString(line)
		text.WriteByte('\n')
	}
	if len(suffix) > 0 {
		text.Write(suffix)
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".humbug2-analysis-*")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err = temp.Chmod(info.Mode().Perm()); err == nil {
		_, err = io.WriteString(temp, text.String())
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}
