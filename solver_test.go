package main

import (
	"os"
	"strings"
	"testing"
)

func puzzle(t *testing.T, text string) Puzzle {
	t.Helper()
	p, err := ParsePuzzle(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLadybirdShortestSolution(t *testing.T) {
	p := puzzle(t, "max-moves 3\n..........\n..........\n..lo*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	m, ok := Solve(p)
	if !ok || len(m) != 1 || m[0].Dir != Right {
		t.Fatalf("got ok=%v moves=%v", ok, m)
	}
}

func TestSolveReportsBreadthFirstDepths(t *testing.T) {
	p := puzzle(t, "max-moves 2\n..........\n..........\n..n*n*....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	start := State{Bugs: append([]Bug(nil), p.Bugs...), Stars: p.Stars, Walls: p.Walls}
	canonicalize(&start)
	var progress []SearchProgress
	moves, ok := solveBFS(p, start, func(update SearchProgress) {
		progress = append(progress, update)
	})
	if !ok || len(moves) != 2 {
		t.Fatalf("got ok=%v moves=%v", ok, moves)
	}
	if len(progress) != 2 || progress[0].Algorithm != "bfs" || progress[0].Depth != 0 || progress[0].States != 1 || progress[1].Depth != 1 {
		t.Fatalf("unexpected progress: %+v", progress)
	}
}

func TestSolveUsesAStarWhenBugCountIsSafeLowerBound(t *testing.T) {
	p := puzzle(t, "max-moves 2\n..........\n..........\n..n*n*....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	var progress []SearchProgress
	moves, ok := SolveWithProgress(p, func(update SearchProgress) { progress = append(progress, update) })
	if !ok || len(moves) != 2 || len(progress) == 0 || progress[0].Algorithm != "astar" || progress[0].Depth != 2 {
		t.Fatalf("unexpected A* result: ok=%v moves=%v progress=%+v", ok, moves, progress)
	}
}

func TestSolveUsesAStarForEggPuzzle(t *testing.T) {
	p := puzzle(t, "max-moves 2\n..........\n..........\n.gn*......\n..*.......\n..........\n..........\n..........\n..........\n..........\n..........\negg 3 3\n")
	var progress []SearchProgress
	moves, ok := SolveWithProgress(p, func(update SearchProgress) { progress = append(progress, update) })
	if !ok || len(moves) != 2 || len(progress) == 0 || progress[0].Algorithm != "astar" {
		t.Fatalf("unexpected egg A* result: ok=%v moves=%v progress=%+v", ok, moves, progress)
	}
}

func TestEggOnStarContributesZeroToSearchLowerBound(t *testing.T) {
	state := State{Bugs: []Bug{
		{Kind: Snail, Pos: Pos{2, 2}, Egg: true},
		{Kind: Ladybird, Pos: Pos{3, 3}},
	}}
	state.Stars[2][2] = 1
	if got := searchLowerBound(state); got != 1 {
		t.Fatalf("got lower bound %d, want 1", got)
	}
	if got := remainingBugCount(state); got != 2 {
		t.Fatalf("egg on star must still count as an unsolved bug; got %d", got)
	}
}

func TestAStarDoesNotMistakeEggOnStarForSolvedPuzzle(t *testing.T) {
	var p Puzzle
	p.MaxMoves = 1
	for c := 1; c <= 3; c++ {
		p.Terrain[2][c] = Low
	}
	start := State{Bugs: []Bug{
		{Kind: Grasshopper, Pos: Pos{2, 1}},
		{Kind: Snail, Pos: Pos{2, 2}, Egg: true},
	}}
	start.Stars[2][2], start.Stars[2][3] = 1, 1
	p.Stars = start.Stars
	canonicalize(&start)
	moves, ok := solveAStar(p, start, nil)
	if !ok || len(moves) != 1 || moves[0].Kind != Grasshopper || moves[0].Dir != Right {
		t.Fatalf("egg-on-star A* result: ok=%v moves=%v", ok, moves)
	}
}

func TestSolveFallsBackToBFSForPushingBeetle(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..t*......\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	var progress []SearchProgress
	moves, ok := SolveWithProgress(p, func(update SearchProgress) { progress = append(progress, update) })
	if !ok || len(moves) != 1 || len(progress) == 0 || progress[0].Algorithm != "bfs" {
		t.Fatalf("unexpected BFS fallback: ok=%v moves=%v progress=%+v", ok, moves, progress)
	}
}

func TestParseScorpionsInBoardOrder(t *testing.T) {
	p := puzzle(t, "max-moves 1\nqQn*\n")
	if len(p.Bugs) != 3 || p.Bugs[0].Kind != Scorpion || p.Bugs[0].ID != 0 || p.Bugs[0].Direction != Right || p.Bugs[1].Kind != Scorpion || p.Bugs[1].ID != 1 {
		t.Fatalf("unexpected scorpions: %+v", p.Bugs)
	}
}

func TestScorpionMovesTowardReachableBug(t *testing.T) {
	var p Puzzle
	for c := 0; c < 5; c++ {
		p.Terrain[2][c] = Low
	}
	s := State{Bugs: []Bug{{Kind: Scorpion, Pos: Pos{2, 0}, Direction: Right}, {Kind: Snail, Pos: Pos{2, 4}}}}
	if !moveScorpions(p, &s) || s.Bugs[0].Pos != (Pos{2, 1}) || s.Bugs[0].Direction != Right {
		t.Fatalf("scorpion did not move toward bug: %+v", s.Bugs)
	}
}

func TestScorpionDirectionTieBreaking(t *testing.T) {
	var p Puzzle
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			p.Terrain[r][c] = Low
		}
	}
	bugs := []Bug{
		{Kind: Scorpion, Pos: Pos{2, 2}, Direction: Down},
		{Kind: Snail, Pos: Pos{2, 0}},
		{Kind: Ladybird, Pos: Pos{0, 2}},
	}
	if dir, ok := scorpionDirection(p, bugs, 0); !ok || dir != Left {
		t.Fatalf("expected left/up tie to choose left, got %v, %v", dir, ok)
	}
	bugs[0].Direction = Up
	if dir, ok := scorpionDirection(p, bugs, 0); !ok || dir != Up {
		t.Fatalf("expected current direction to win tie, got %v, %v", dir, ok)
	}
}

func TestScorpionPrefersRightTurnOverLeftTurn(t *testing.T) {
	var p Puzzle
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			p.Terrain[r][c] = Low
		}
	}
	bugs := []Bug{
		{Kind: Scorpion, Pos: Pos{2, 2}, Direction: Up},
		{Kind: Snail, Pos: Pos{2, 4}},
		{Kind: Ladybird, Pos: Pos{2, 0}},
	}
	if dir, ok := scorpionDirection(p, bugs, 0); !ok || dir != Right {
		t.Fatalf("expected right turn to win equal right/left paths, got %v, %v", dir, ok)
	}
}

func TestS22ScorpionPrefersDownOverReverseTurn(t *testing.T) {
	data, err := os.ReadFile("puzzles/s22.puzzle")
	if err != nil {
		t.Fatal(err)
	}
	p := puzzle(t, string(data))
	state := State{Bugs: append([]Bug(nil), p.Bugs...), Stars: p.Stars, Walls: p.Walls}
	canonicalize(&state)
	grasshopper := bugAt(state.Bugs, Pos{5, 7})
	next, _, ok := applyMove(p, state, grasshopper, Left)
	if !ok {
		t.Fatal("grasshopper move should be valid")
	}
	for _, bug := range next.Bugs {
		if bug.Kind == Scorpion && bug.ID == 1 {
			if bug.Pos != (Pos{3, 5}) || bug.Direction != Down {
				t.Fatalf("scorpion 2 should turn down, got %+v", bug)
			}
			return
		}
	}
	t.Fatal("scorpion 2 not found")
}

func TestScorpionDirectionAndIdentityArePartOfSearchState(t *testing.T) {
	a := State{Bugs: []Bug{{Kind: Scorpion, Pos: Pos{2, 3}, Direction: Left, ID: 0}}}
	b := State{Bugs: []Bug{{Kind: Scorpion, Pos: Pos{2, 3}, Direction: Right, ID: 0}}}
	c := State{Bugs: []Bug{{Kind: Scorpion, Pos: Pos{2, 3}, Direction: Left, ID: 1}}}
	if stateKey(a) == stateKey(b) || stateKey(a) == stateKey(c) {
		t.Fatal("scorpion direction and identity must affect the search state")
	}
}

func TestScorpionKillsAdjacentBug(t *testing.T) {
	var p Puzzle
	p.Terrain[2][1], p.Terrain[2][2] = Low, Low
	s := State{Bugs: []Bug{{Kind: Scorpion, Pos: Pos{2, 1}, Direction: Right}, {Kind: Snail, Pos: Pos{2, 2}}}}
	if moveScorpions(p, &s) {
		t.Fatal("an unstunned scorpion should kill an adjacent bug")
	}
}

func TestScorpionCannotKillAdjacentBugOnDifferentLevel(t *testing.T) {
	var p Puzzle
	p.Terrain[2][1], p.Terrain[2][2] = Low, High
	s := State{Bugs: []Bug{{Kind: Scorpion, Pos: Pos{2, 1}, Direction: Right}, {Kind: Snail, Pos: Pos{2, 2}}}}
	if !moveScorpions(p, &s) {
		t.Fatal("scorpion must not kill an adjacent bug on another level")
	}
}

func TestScorpionCannotKillThroughWall(t *testing.T) {
	for _, test := range []struct {
		name     string
		scorpion Pos
		bug      Pos
	}{
		{name: "wall to right", scorpion: Pos{2, 1}, bug: Pos{2, 2}},
		{name: "wall to left", scorpion: Pos{2, 2}, bug: Pos{2, 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var p Puzzle
			p.Terrain[2][1], p.Terrain[2][2] = Low, Low
			p.Walls[2][1] = 1 << Right
			p.Walls[2][2] = 1 << Left
			s := State{Bugs: []Bug{{Kind: Scorpion, Pos: test.scorpion, Direction: Right}, {Kind: Snail, Pos: test.bug}}}
			if !moveScorpions(p, &s) {
				t.Fatal("scorpion must not kill a bug through a wall")
			}
		})
	}
}

func TestScorpionsDoNotTargetOrKillEachOther(t *testing.T) {
	var p Puzzle
	for c := 1; c <= 3; c++ {
		p.Terrain[2][c] = Low
	}
	s := State{Bugs: []Bug{
		{Kind: Scorpion, Pos: Pos{2, 1}, Direction: Right, ID: 0},
		{Kind: Scorpion, Pos: Pos{2, 2}, Direction: Left, ID: 1},
	}}
	if !moveScorpions(p, &s) || len(s.Bugs) != 2 || s.Bugs[0].Pos != (Pos{2, 1}) || s.Bugs[1].Pos != (Pos{2, 2}) {
		t.Fatalf("scorpions should ignore one another as targets: %+v", s.Bugs)
	}
}

func TestLandingOnScorpionStunsItForMove(t *testing.T) {
	var p Puzzle
	for c := 0; c < 4; c++ {
		p.Terrain[2][c] = Low
	}
	s := State{Bugs: []Bug{{Kind: Grasshopper, Pos: Pos{2, 0}}, {Kind: Scorpion, Pos: Pos{2, 1}, Direction: Right}}}
	next, _, ok := applyMove(p, s, 0, Right)
	if !ok {
		t.Fatal("landing bug should pass a stunned scorpion safely")
	}
	positions := occupancy(next.Bugs)
	if _, exists := positions[Pos{2, 1}]; !exists {
		t.Fatalf("stunned scorpion should remain in place: %+v", next.Bugs)
	}
	if _, exists := positions[Pos{2, 2}]; !exists {
		t.Fatalf("grasshopper should bounce past scorpion: %+v", next.Bugs)
	}
}

func TestEggIsNotScorpionTarget(t *testing.T) {
	var p Puzzle
	for c := 0; c < 4; c++ {
		p.Terrain[2][c] = Low
	}
	s := State{Bugs: []Bug{{Kind: Scorpion, Pos: Pos{2, 0}, Direction: Right}, {Kind: Snail, Pos: Pos{2, 3}, Egg: true}}}
	if !moveScorpions(p, &s) || s.Bugs[0].Pos != (Pos{2, 0}) {
		t.Fatalf("scorpion should not pursue an egg: %+v", s.Bugs)
	}
}

func TestScorpionBouncesOnTrampolineAndHatchesEgg(t *testing.T) {
	var p Puzzle
	for r := 0; r < 3; r++ {
		for c := 0; c < 4; c++ {
			p.Terrain[r][c] = Low
		}
	}
	p.Trampolines[2][1] = true
	s := State{Bugs: []Bug{
		{Kind: Scorpion, Pos: Pos{2, 0}, Direction: Right},
		{Kind: Snail, Pos: Pos{2, 2}, Egg: true},
		{Kind: Ladybird, Pos: Pos{0, 1}},
	}}
	if !moveScorpions(p, &s) {
		t.Fatal("scorpion trampoline move unexpectedly lost puzzle")
	}
	if s.Bugs[0].Pos != (Pos{2, 3}) || s.Bugs[1].Egg {
		t.Fatalf("scorpion should bounce over and hatch egg: %+v", s.Bugs)
	}
}

func TestS17BeeMoveMakesScorpionApproachAcrossPerceivedTrampolinePath(t *testing.T) {
	data, err := os.ReadFile("puzzles/s17.puzzle")
	if err != nil {
		t.Fatal(err)
	}
	p := puzzle(t, string(data))
	state := State{Bugs: append([]Bug(nil), p.Bugs...), Stars: p.Stars, Walls: p.Walls}
	canonicalize(&state)
	bee := bugAt(state.Bugs, Pos{1, 1})
	next, _, ok := applyMove(p, state, bee, Right)
	if !ok {
		t.Fatal("bee move should be valid")
	}
	scorpion := bugAt(next.Bugs, Pos{1, 1})
	if scorpion < 0 || next.Bugs[scorpion].Kind != Scorpion {
		t.Fatalf("scorpion should move right into the bee's old square: %+v", next.Bugs)
	}
}

func TestPuzzleCanSolveWithScorpionRemaining(t *testing.T) {
	p := puzzle(t, "max-moves 1\nq.n*\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 || moves[0].Kind != Snail || moves[0].Dir != Right {
		t.Fatalf("scorpion should not count toward victory: ok=%v moves=%v", ok, moves)
	}
}

func TestSnailMovesOneSquare(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..n*......\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 {
		t.Fatalf("got ok=%v moves=%v", ok, moves)
	}
	if moves[0].Kind != Snail || moves[0].Dir != Right || moves[0].To != (Pos{2, 3}) {
		t.Fatalf("unexpected snail move: %+v", moves[0])
	}
}

func TestSnailIsNonFlying(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..nO*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars}
	if _, _, ok := applyMove(p, s, 0, Right); ok {
		t.Fatal("low snail should not climb onto a high platform")
	}
}

func TestPinkLadybirdMovesThreeSquares(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..poo*....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 {
		t.Fatalf("got ok=%v moves=%v", ok, moves)
	}
	if moves[0].Kind != PinkLadybird || moves[0].Dir != Right || moves[0].To != (Pos{2, 5}) {
		t.Fatalf("unexpected Pink Ladybird move: %+v", moves[0])
	}
}

func TestGrasshopperFliesOneSquareAcrossWall(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..g*......\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwall 3 3 right\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 {
		t.Fatalf("got ok=%v moves=%v", ok, moves)
	}
	if moves[0].Kind != Grasshopper || moves[0].Dir != Right || moves[0].To != (Pos{2, 3}) {
		t.Fatalf("unexpected grasshopper move: %+v", moves[0])
	}
}

func TestButterflyFliesThreeSquaresAcrossWall(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n.f..*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwall 3 2 right\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 || moves[0].Kind != Butterfly || moves[0].To != (Pos{2, 4}) {
		t.Fatalf("unexpected butterfly solution: ok=%v moves=%v", ok, moves)
	}
}

func TestFlyTravelsToFirstPlatform(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n.y..*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwall 3 2 right\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 || moves[0].Kind != Fly || moves[0].To != (Pos{2, 4}) {
		t.Fatalf("unexpected fly solution: ok=%v moves=%v", ok, moves)
	}
}

func TestFlyStopsAtFirstPlatform(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n.y.o*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars, Walls: p.Walls}
	next, move, ok := applyMove(p, s, 0, Right)
	if !ok || move.To != (Pos{2, 3}) || len(next.Bugs) != 1 {
		t.Fatalf("fly did not stop at first platform: ok=%v move=%+v bugs=%+v", ok, move, next.Bugs)
	}
}

func TestFlyBouncesWhenFirstPlatformIsOccupied(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n.y.u*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 || moves[0].To != (Pos{2, 4}) {
		t.Fatalf("fly should bounce from puck onto star: ok=%v moves=%v", ok, moves)
	}
}

func TestFlyFallsWhenNoPlatformAhead(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n.y........\n..........\n..........\n..........\n..........\n..........\n......*...\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars, Walls: p.Walls}
	if _, _, ok := applyMove(p, s, 0, Right); ok {
		t.Fatal("fly should fall when no platform exists before the edge")
	}
}

func TestCockroachWalksTwoSquaresThroughWalls(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..co*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwall 3 3 right\nwall 3 4 right\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 || moves[0].Kind != Cockroach || moves[0].To != (Pos{2, 4}) {
		t.Fatalf("unexpected cockroach solution: ok=%v moves=%v", ok, moves)
	}
}

func TestCockroachStillCannotClimb(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..cO*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars}
	if _, _, ok := applyMove(p, s, 0, Right); ok {
		t.Fatal("low cockroach should not climb onto a high platform")
	}
}

func TestPushedCockroachIsBlockedByWall(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..tc*.....\n..........\n..........\n..........\n..........\n..........\n......*...\n..........\nwall 3 4 right\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars, Walls: p.Walls}
	canonicalize(&s)
	var beetleIndex int
	for i, b := range s.Bugs {
		if b.Kind == Beetle {
			beetleIndex = i
		}
	}
	if _, _, ok := applyMove(p, s, beetleIndex, Right); ok {
		t.Fatal("pushed cockroach must not pass under wall")
	}
}

func TestGoldBeetleMovesTwoSquaresAndKnocksDownWalls(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..do*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwall 3 3 right\nwall 3 4 right\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars, Walls: p.Walls}
	next, move, ok := applyMove(p, s, 0, Right)
	if !ok || move.Kind != GoldBeetle || move.To != (Pos{2, 4}) || len(next.Bugs) != 0 {
		t.Fatalf("unexpected Gold Beetle move: ok=%v move=%+v bugs=%+v", ok, move, next.Bugs)
	}
	if next.Walls[2][2] != 0 || next.Walls[2][3] != 0 || next.Walls[2][4] != 0 {
		t.Fatalf("crossed walls were not removed: %+v", next.Walls[2])
	}
}

func TestGoldBeetleDoesNotDestroyWallDuringBounce(t *testing.T) {
	p := Puzzle{}
	p.Terrain[3][2] = High
	p.Terrain[3][3] = Low
	p.Terrain[3][4] = Low
	p.Walls[3][3] |= 1 << Right
	p.Walls[3][4] |= 1 << Left
	s := State{
		Bugs:  []Bug{{Kind: GoldBeetle, Pos: Pos{3, 2}}, {Kind: Grasshopper, Pos: Pos{3, 3}}},
		Walls: p.Walls,
	}
	next, move, ok := applyMove(p, s, 0, Right)
	if !ok || move.To != (Pos{3, 4}) {
		t.Fatalf("Gold Beetle should bounce over grasshopper: ok=%v move=%+v", ok, move)
	}
	if next.Walls[3][3]&(1<<Right) == 0 || next.Walls[3][4]&(1<<Left) == 0 {
		t.Fatal("wall crossed during bounce was incorrectly destroyed")
	}
}

func TestGoldBeetleDestroysWallWhenBugBlocksDestination(t *testing.T) {
	p := Puzzle{}
	p.Terrain[2][2] = Low
	p.Terrain[2][3] = Low
	p.Walls[2][2] |= 1 << Right
	p.Walls[2][3] |= 1 << Left
	s := State{
		Bugs:  []Bug{{Kind: GoldBeetle, Pos: Pos{2, 2}}, {Kind: Snail, Pos: Pos{2, 3}}},
		Walls: p.Walls,
	}
	next, move, ok := applyMove(p, s, 0, Right)
	if !ok || move.From != move.To || move.To != (Pos{2, 2}) {
		t.Fatalf("wall-only Gold Beetle move failed: ok=%v move=%+v", ok, move)
	}
	if hasWall(Puzzle{Walls: next.Walls}, Pos{2, 2}, Right) {
		t.Fatal("Gold Beetle did not destroy wall in front of blocking bug")
	}
	if len(next.Bugs) != 2 || next.Bugs[0].Pos != (Pos{2, 2}) {
		t.Fatalf("Gold Beetle should remain before obstacle: %+v", next.Bugs)
	}
}

func TestGoldBeetleDestroysWallWhenSecondSquareIsVoid(t *testing.T) {
	p := Puzzle{}
	p.Terrain[4][2] = Low
	p.Terrain[5][2] = Low
	p.Walls[4][2] |= 1 << Down
	p.Walls[5][2] |= 1 << Up
	s := State{Bugs: []Bug{{Kind: GoldBeetle, Pos: Pos{4, 2}}}, Walls: p.Walls}
	next, move, ok := applyMove(p, s, 0, Down)
	if !ok || move.To != (Pos{5, 2}) {
		t.Fatalf("wall-only move before void failed: ok=%v move=%+v", ok, move)
	}
	if hasWall(Puzzle{Walls: next.Walls}, Pos{4, 2}, Down) {
		t.Fatal("Gold Beetle did not destroy wall when second square was void")
	}
}

func TestGoldBeetleStaysPutWhenWallBeforeBlockedSecondSquare(t *testing.T) {
	p := Puzzle{}
	for c := 2; c <= 4; c++ {
		p.Terrain[2][c] = Low
	}
	p.Walls[2][3] |= 1 << Right
	p.Walls[2][4] |= 1 << Left
	s := State{
		Bugs:  []Bug{{Kind: GoldBeetle, Pos: Pos{2, 2}}, {Kind: Ant, Pos: Pos{2, 4}, Count: 1}},
		Walls: p.Walls,
	}
	next, move, ok := applyMove(p, s, 0, Right)
	if !ok || move.To != (Pos{2, 3}) {
		t.Fatalf("blocked two-square wall strike failed: ok=%v move=%+v", ok, move)
	}
	if hasWall(Puzzle{Walls: next.Walls}, Pos{2, 3}, Right) {
		t.Fatal("Gold Beetle did not destroy wall before blocked second square")
	}
}

func TestPushedGoldBeetleIsBlockedByWall(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..td*.....\n..........\n..........\n..........\n..........\n..........\n......*...\n..........\nwall 3 4 right\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars, Walls: p.Walls}
	canonicalize(&s)
	var beetleIndex int
	for i, b := range s.Bugs {
		if b.Kind == Beetle {
			beetleIndex = i
		}
	}
	if _, _, ok := applyMove(p, s, beetleIndex, Right); ok {
		t.Fatal("pushed Gold Beetle must not pass through or destroy wall")
	}
}

func TestWallLayoutIsPartOfSearchState(t *testing.T) {
	a := State{}
	b := State{}
	b.Walls[2][3] = 1 << Right
	if stateKey(a) == stateKey(b) {
		t.Fatal("states with different remaining walls must have different keys")
	}
}

func TestLandingOnEggHatchesIt(t *testing.T) {
	p := puzzle(t, "max-moves 2\n..........\n..........\n.gn*......\n..*.......\n..........\n..........\n..........\n..........\n..........\n..........\neggs:\negg 3 3\n")
	if !p.Bugs[1].Egg {
		t.Fatal("snail was not parsed as an egg")
	}
	moves, ok := Solve(p)
	if !ok || len(moves) != 2 {
		t.Fatalf("egg puzzle: ok=%v moves=%v", ok, moves)
	}
	if moves[0].Kind != Grasshopper || moves[0].Dir != Right || moves[1].Kind != Snail || moves[1].Dir != Down {
		t.Fatalf("unexpected egg solution: %+v", moves)
	}
}

func TestEggCannotMoveBeforeHatching(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..n*......\n..........\n..........\n..........\n..........\n..........\n..........\n..........\negg 3 3\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars, Walls: p.Walls}
	if _, _, ok := applyMove(p, s, 0, Right); ok {
		t.Fatal("egg initiated a move")
	}
	if _, ok := Solve(p); ok {
		t.Fatal("egg-only puzzle should not be solvable")
	}
}

func TestPushedEggRemainsAnEgg(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..tno*....\n..........\n..........\n..........\n..........\n..........\n......*...\n..........\negg 3 4\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars, Walls: p.Walls}
	canonicalize(&s)
	var beetleIndex int
	for i, b := range s.Bugs {
		if b.Kind == Beetle {
			beetleIndex = i
		}
	}
	next, _, ok := applyMove(p, s, beetleIndex, Right)
	if !ok {
		t.Fatal("beetle should be able to push an egg")
	}
	found := false
	for _, b := range next.Bugs {
		if b.Kind == Snail {
			found = true
			if !b.Egg || b.Pos != (Pos{2, 4}) {
				t.Fatalf("pushed egg changed unexpectedly: %+v", b)
			}
		}
	}
	if !found {
		t.Fatal("pushed egg disappeared")
	}
}

func TestPushedEggDoesNotConsumeStar(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..tn*.....\n..........\n..........\n..........\n..........\n..........\n......*...\n..........\negg 3 4\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars, Walls: p.Walls}
	canonicalize(&s)
	var beetleIndex int
	for i, b := range s.Bugs {
		if b.Kind == Beetle {
			beetleIndex = i
		}
	}
	next, _, ok := applyMove(p, s, beetleIndex, Right)
	if !ok {
		t.Fatal("beetle should push egg onto star")
	}
	star := Pos{2, 4}
	if next.Stars[star.R][star.C] != 1 {
		t.Fatal("unhatched egg consumed star")
	}
	found := false
	for _, b := range next.Bugs {
		if b.Kind == Snail && b.Pos == star && b.Egg {
			found = true
		}
	}
	if !found {
		t.Fatalf("egg did not remain on star: %+v", next.Bugs)
	}
}

func TestEggOnStarDisappearsWhenLandedOn(t *testing.T) {
	p := Puzzle{}
	for c := 1; c <= 3; c++ {
		p.Terrain[2][c] = Low
	}
	p.Stars[2][2] = 1
	s := State{Bugs: []Bug{
		{Kind: Grasshopper, Pos: Pos{2, 1}},
		{Kind: Snail, Pos: Pos{2, 2}, Egg: true},
	}, Stars: p.Stars}
	next, _, ok := applyMove(p, s, 0, Right)
	if !ok {
		t.Fatal("grasshopper should land on egg and bounce")
	}
	if next.Stars[2][2] != 0 {
		t.Fatal("hatched egg did not consume star")
	}
	if len(next.Bugs) != 1 || next.Bugs[0].Kind != Grasshopper || next.Bugs[0].Pos != (Pos{2, 3}) {
		t.Fatalf("hatched egg should disappear while landing bug continues: %+v", next.Bugs)
	}
}

func TestEggStatusIsPartOfSearchState(t *testing.T) {
	a := State{Bugs: []Bug{{Kind: Snail, Pos: Pos{2, 3}, Egg: true}}}
	b := State{Bugs: []Bug{{Kind: Snail, Pos: Pos{2, 3}}}}
	if stateKey(a) == stateKey(b) {
		t.Fatal("egg and hatched states must have different keys")
	}
}

func TestRejectEggWithoutBug(t *testing.T) {
	_, err := ParsePuzzle(strings.NewReader("max-moves 1\n..........\n..........\n..n*......\n..........\n..........\n..........\n..........\n..........\n..........\n..........\negg 1 1\n"))
	if err == nil {
		t.Fatal("expected egg-position validation error")
	}
}

func TestPuckDoesNotNeedToFinishOrConsumeStar(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..n*u.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 || moves[0].Kind != Snail {
		t.Fatalf("puck should not prevent victory: ok=%v moves=%v", ok, moves)
	}
}

func TestFlyingBugBouncesOnPuck(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..gu*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 || moves[0].To != (Pos{2, 4}) {
		t.Fatalf("grasshopper should bounce on puck: ok=%v moves=%v", ok, moves)
	}
}

func TestPuckBouncesFromTrampolineWithoutConsumingStar(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..tux*....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars, Walls: p.Walls}
	canonicalize(&s)
	var beetleIndex int
	for i, b := range s.Bugs {
		if b.Kind == Beetle {
			beetleIndex = i
		}
	}
	next, _, ok := applyMove(p, s, beetleIndex, Right)
	if !ok || len(next.Bugs) != 2 {
		t.Fatalf("puck trampoline push failed: ok=%v bugs=%+v", ok, next.Bugs)
	}
	star := Pos{2, 5}
	if next.Stars[star.R][star.C] != 1 {
		t.Fatal("puck consumed a star")
	}
	found := false
	for _, b := range next.Bugs {
		if b.Kind == Puck && b.Pos == star {
			found = true
		}
	}
	if !found {
		t.Fatalf("puck did not bounce onto star: %+v", next.Bugs)
	}
}

func TestPushedPuckCanFallOffBoard(t *testing.T) {
	p := puzzle(t, "max-moves 2\n..........\n..........\n........tu\n..........\n..........\n..........\n..........\n..........\n......*...\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars, Walls: p.Walls}
	canonicalize(&s)
	var beetleIndex int
	for i, b := range s.Bugs {
		if b.Kind == Beetle {
			beetleIndex = i
		}
	}
	next, _, ok := applyMove(p, s, beetleIndex, Right)
	if !ok || len(next.Bugs) != 1 || next.Bugs[0].Kind != Beetle || next.Bugs[0].Pos != (Pos{2, 9}) {
		t.Fatalf("puck fall should leave beetle in play: ok=%v bugs=%+v", ok, next.Bugs)
	}
}

func TestPuckCannotInitiateMove(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..n*u.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars, Walls: p.Walls}
	var puckIndex int
	for i, b := range s.Bugs {
		if b.Kind == Puck {
			puckIndex = i
		}
	}
	if _, _, ok := applyMove(p, s, puckIndex, Right); ok {
		t.Fatal("puck initiated a move")
	}
}

func TestAntsCombineThenMoveAsOneGroup(t *testing.T) {
	p := puzzle(t, "max-moves 2\n..........\n..........\n. . a a o *2 . . . .\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwall 3 6 right\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 2 {
		t.Fatalf("combined-ant puzzle: ok=%v moves=%v", ok, moves)
	}
	if moves[0].Kind != Ant || moves[0].Count != 1 || moves[0].To != (Pos{2, 3}) {
		t.Fatalf("unexpected combining move: %+v", moves[0])
	}
	if moves[1].Kind != Ant || moves[1].Count != 2 || moves[1].To != (Pos{2, 5}) {
		t.Fatalf("combined ants did not move as one: %+v", moves[1])
	}
}

func TestThreeAntsHittingOneLeaveOneBehind(t *testing.T) {
	p := Puzzle{}
	for c := 2; c <= 4; c++ {
		p.Terrain[2][c] = Low
	}
	s := State{Bugs: []Bug{{Kind: Ant, Pos: Pos{2, 2}, Count: 3}, {Kind: Ant, Pos: Pos{2, 4}, Count: 1}}}
	next, _, ok := applyMove(p, s, 0, Right)
	if !ok || len(next.Bugs) != 2 {
		t.Fatalf("partial ant merge failed: ok=%v bugs=%+v", ok, next.Bugs)
	}
	want := map[Pos]uint8{{2, 3}: 1, {2, 4}: 3}
	for _, b := range next.Bugs {
		if want[b.Pos] != b.Count {
			t.Fatalf("unexpected ant group after merge: %+v", next.Bugs)
		}
	}
}

func TestThreeAntsHittingTwoLeaveTwoBehind(t *testing.T) {
	p := Puzzle{}
	for c := 2; c <= 4; c++ {
		p.Terrain[2][c] = Low
	}
	s := State{Bugs: []Bug{{Kind: Ant, Pos: Pos{2, 2}, Count: 3}, {Kind: Ant, Pos: Pos{2, 4}, Count: 2}}}
	next, _, ok := applyMove(p, s, 0, Right)
	if !ok || len(next.Bugs) != 2 {
		t.Fatalf("partial ant merge failed: ok=%v bugs=%+v", ok, next.Bugs)
	}
	want := map[Pos]uint8{{2, 3}: 2, {2, 4}: 3}
	for _, b := range next.Bugs {
		if want[b.Pos] != b.Count {
			t.Fatalf("unexpected ant group after merge: %+v", next.Bugs)
		}
	}
}

func TestLandingAntsBounceWithoutCombining(t *testing.T) {
	p := Puzzle{}
	p.Terrain[2][2] = High
	p.Terrain[2][3] = Low
	p.Terrain[2][4] = Low
	s := State{Bugs: []Bug{{Kind: Ant, Pos: Pos{2, 2}, Count: 2}, {Kind: Ant, Pos: Pos{2, 3}, Count: 1}}}
	next, _, ok := applyMove(p, s, 0, Right)
	if !ok || len(next.Bugs) != 2 {
		t.Fatalf("landing ant bounce failed: ok=%v bugs=%+v", ok, next.Bugs)
	}
	want := map[Pos]uint8{{2, 3}: 1, {2, 4}: 2}
	for _, b := range next.Bugs {
		if want[b.Pos] != b.Count {
			t.Fatalf("landing ants combined unexpectedly: %+v", next.Bugs)
		}
	}
}

func TestStarsConsumeIndividualAntsFromGroup(t *testing.T) {
	s := State{Bugs: []Bug{{Kind: Ant, Pos: Pos{2, 3}, Count: 3}}}
	s.Stars[2][3] = 2
	consumeStars(&s)
	if len(s.Bugs) != 1 || s.Bugs[0].Count != 1 || s.Stars[2][3] != 0 {
		t.Fatalf("partial ant consumption failed: bugs=%+v stars=%d", s.Bugs, s.Stars[2][3])
	}
	s.Stars[2][3] = 1
	consumeStars(&s)
	if len(s.Bugs) != 0 || s.Stars[2][3] != 0 {
		t.Fatalf("final ant consumption failed: bugs=%+v stars=%d", s.Bugs, s.Stars[2][3])
	}
}

func TestAntCountIsPartOfSearchState(t *testing.T) {
	a := State{Bugs: []Bug{{Kind: Ant, Pos: Pos{2, 3}, Count: 1}}}
	b := State{Bugs: []Bug{{Kind: Ant, Pos: Pos{2, 3}, Count: 2}}}
	if stateKey(a) == stateKey(b) {
		t.Fatal("ant groups of different sizes must have different keys")
	}
}

func TestWalkingBugBouncesAsSoonAsItCrossesTrampoline(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n.px*......\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 || moves[0].To != (Pos{2, 3}) {
		t.Fatalf("Pink Ladybird should bounce early onto star: ok=%v moves=%v", ok, moves)
	}
}

func TestFlyingBugLandsOnTrampolineAndBouncesAcrossWall(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..gx@.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwall 3 4 right\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 || moves[0].To != (Pos{2, 4}) {
		t.Fatalf("grasshopper should bounce from trampoline across wall: ok=%v moves=%v", ok, moves)
	}
}

func TestPushedBugBouncesFromTrampoline(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..tlx*....\n..........\n..........\n..........\n..........\n..........\n......*...\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars}
	canonicalize(&s)
	var beetleIndex int
	for i, b := range s.Bugs {
		if b.Kind == Beetle {
			beetleIndex = i
		}
	}
	next, _, ok := applyMove(p, s, beetleIndex, Right)
	if !ok || len(next.Bugs) != 1 || next.Bugs[0].Kind != Beetle {
		t.Fatalf("pushed bug should bounce from trampoline onto star: ok=%v bugs=%+v", ok, next.Bugs)
	}
}

func TestMultipleStarsOnOnePlatformAreConsumedIndividually(t *testing.T) {
	p := puzzle(t, "max-moves 6\n. . . . . . . . . .\n. . . . . . . . . .\nn n n *3 . . . . . .\n. . . . . . . . . .\n. . . . . . . . . .\n. . . . . . . . . .\n. . . . . . . . . .\n. . . . . . . . . .\n. . . . . . . . . .\n. . . . . . . . . .\n")
	star := Pos{2, 3}
	if p.Stars[star.R][star.C] != 3 {
		t.Fatalf("parsed star count = %d, want 3", p.Stars[star.R][star.C])
	}
	s := State{Stars: p.Stars}
	for want := uint8(2); ; want-- {
		s.Bugs = []Bug{{Kind: Snail, Pos: star}}
		consumeStars(&s)
		if len(s.Bugs) != 0 || s.Stars[star.R][star.C] != want {
			t.Fatalf("after consumption: bugs=%v stars=%d, want 0 bugs and %d stars", s.Bugs, s.Stars[star.R][star.C], want)
		}
		if want == 0 {
			break
		}
	}
	moves, ok := Solve(p)
	if !ok || len(moves) != 6 {
		t.Fatalf("stacked-star puzzle: ok=%v moves=%v", ok, moves)
	}
}

func TestWallStopsLadybirdOnStar(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..l*o.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwalls:\nwall 3 4 right\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 {
		t.Fatalf("wall-stopped puzzle: ok=%v moves=%v", ok, moves)
	}
	if moves[0].To != (Pos{2, 3}) {
		t.Fatalf("ladybird crossed wall: %+v", moves[0])
	}
	if !hasWall(p, Pos{2, 3}, Right) || !hasWall(p, Pos{2, 4}, Left) {
		t.Fatal("wall was not stored on both sides of its boundary")
	}
}

func TestFlyingBugCrossesWall(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..bo*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwall 3 3 right\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 || moves[0].To != (Pos{2, 4}) {
		t.Fatalf("bee should fly through wall: ok=%v moves=%v", ok, moves)
	}
}

func TestFlyingBugBouncesAcrossWall(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..b.l@....\n..........\n..........\n..........\n..........\n..........\n......*...\n..........\nwall 3 5 right\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars}
	canonicalize(&s)
	var beeIndex int
	for i, b := range s.Bugs {
		if b.Kind == Bee {
			beeIndex = i
		}
	}
	next, _, ok := applyMove(p, s, beeIndex, Right)
	if !ok || len(next.Bugs) != 1 || next.Bugs[0].Kind != Ladybird {
		t.Fatalf("bee should bounce across wall onto star: ok=%v bugs=%+v", ok, next.Bugs)
	}
}

func TestNonFlyingBugBouncesAcrossWall(t *testing.T) {
	p := puzzle(t, "max-moves 2\np x C *2 . . . . . .\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwall 1 3 right\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars, Walls: p.Walls}
	canonicalize(&s)
	var pinkIndex int
	for i, b := range s.Bugs {
		if b.Kind == PinkLadybird {
			pinkIndex = i
		}
	}
	next, move, ok := applyMove(p, s, pinkIndex, Right)
	if !ok || move.To != (Pos{0, 3}) || len(next.Bugs) != 1 || next.Bugs[0].Kind != Cockroach {
		t.Fatalf("Pink Ladybird should bounce over wall onto one star: ok=%v move=%+v bugs=%+v", ok, move, next.Bugs)
	}
	if next.Stars[0][3] != 1 {
		t.Fatalf("stacked star count after bounce = %d, want 1", next.Stars[0][3])
	}
}

func TestRejectWallBetweenTwoVoidSquares(t *testing.T) {
	_, err := ParsePuzzle(strings.NewReader("max-moves 1\n..........\n..........\n..n*......\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwall 1 1 right\n"))
	if err == nil {
		t.Fatal("expected wall validation error")
	}
}

func TestWallAtBoardEdgeStopsBugFromFalling(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n*l........\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwall 3 1 left\n")
	moves, ok := Solve(p)
	if !ok || len(moves) != 1 || moves[0].To != (Pos{2, 0}) {
		t.Fatalf("edge wall should stop ladybird on star: ok=%v moves=%v", ok, moves)
	}
}

func TestRejectEdgeWallBesideVoid(t *testing.T) {
	_, err := ParsePuzzle(strings.NewReader("max-moves 1\n..........\n..........\n..n*......\n..........\n..........\n..........\n..........\n..........\n..........\n..........\nwall 1 1 up\n"))
	if err == nil {
		t.Fatal("expected edge-wall validation error")
	}
}

func TestBeeFliesOverVoid(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..b.*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	m, ok := Solve(p)
	if !ok || len(m) != 1 {
		t.Fatalf("got ok=%v moves=%v", ok, m)
	}
}

func TestBounceCanLandOnHighPlatform(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n.b.l@.....\n..........\n..........\n..........\n..........\n..........\n......*...\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars}
	canonicalize(&s)
	var beeIndex int
	for i, b := range s.Bugs {
		if b.Kind == Bee {
			beeIndex = i
		}
	}
	next, _, ok := applyMove(p, s, beeIndex, Right)
	if !ok {
		t.Fatal("bee should bounce from the occupied low platform onto the high platform")
	}
	if len(next.Bugs) != 1 {
		t.Fatalf("bee should disappear on the high-platform star: %+v", next.Bugs)
	}
	if next.Bugs[0].Kind != Ladybird || next.Bugs[0].Pos != (Pos{2, 3}) {
		t.Fatalf("the bug that caused the bounce should not move: %+v", next.Bugs)
	}
}

func TestWalkingBugDropsOntoBugAndBouncesToHighPlatform(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..........\n..LlO*....\n..........\n..........\n..........\n..........\n......*...\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars}
	canonicalize(&s)
	var highLadybird int
	for i, b := range s.Bugs {
		if b.Kind == Ladybird && p.Terrain[b.Pos.R][b.Pos.C] == High {
			highLadybird = i
		}
	}
	next, _, ok := applyMove(p, s, highLadybird, Right)
	if !ok {
		t.Fatal("high ladybird should enter landing mode, bounce, and land high")
	}
	positions := occupancy(next.Bugs)
	if _, ok := positions[Pos{3, 4}]; !ok {
		t.Fatalf("ladybird did not bounce onto high platform: %+v", next.Bugs)
	}
}

func TestSpiderStopsAtBug(t *testing.T) {
	p := puzzle(t, "max-moves 2\n..........\n..........\n.toos*....\n..........\n..........\n..........\n..........\n..........\n..........\n.....*....\n")
	// The parser validation is what matters here; use direct movement to avoid
	// asserting a full-puzzle solution for a movement-unit test.
	s := State{Bugs: p.Bugs, Stars: p.Stars}
	canonicalize(&s)
	var spiderIndex int
	for i, b := range s.Bugs {
		if b.Kind == Spider {
			spiderIndex = i
		}
	}
	next, _, ok := applyMove(p, s, spiderIndex, Left)
	if !ok {
		t.Fatal("spider move should be valid")
	}
	for _, b := range next.Bugs {
		if b.Kind == Spider && b.Pos != (Pos{2, 2}) {
			t.Fatalf("spider stopped at %+v", b.Pos)
		}
	}
}

func TestNoSolutionAtMoveLimit(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..looo*...\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	if _, ok := Solve(p); ok {
		t.Fatal("unexpected solution")
	}
}

func TestBeetlePushesChainOntoStar(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..tll*....\n..........\n..........\n..........\n..........\n..........\n......**..\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars}
	canonicalize(&s)
	var beetleIndex int
	for i, b := range s.Bugs {
		if b.Kind == Beetle {
			beetleIndex = i
		}
	}
	next, _, ok := applyMove(p, s, beetleIndex, Right)
	if !ok {
		t.Fatal("push should be valid")
	}
	if len(next.Bugs) != 2 {
		t.Fatalf("pushed bug on star was not removed: %+v", next.Bugs)
	}
}

func TestHighBeetlePushDropsLastBugIntoLandingMode(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..TLl*....\n..........\n..........\n..........\n..........\n..........\n......**..\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars}
	canonicalize(&s)
	var beetleIndex int
	for i, b := range s.Bugs {
		if b.Kind == Beetle {
			beetleIndex = i
		}
	}
	next, _, ok := applyMove(p, s, beetleIndex, Right)
	if !ok {
		t.Fatal("high push and landing bounce should be valid")
	}
	if len(next.Bugs) != 2 {
		t.Fatalf("high bug should bounce over low bug and disappear on star: %+v", next.Bugs)
	}
	positions := occupancy(next.Bugs)
	if _, ok := positions[Pos{2, 3}]; !ok {
		t.Fatalf("beetle did not move into vacated high platform: %+v", next.Bugs)
	}
	if _, ok := positions[Pos{2, 4}]; !ok {
		t.Fatalf("low-level bug should not have been pushed: %+v", next.Bugs)
	}
}

func TestHighBeetleBouncesOverRatherThanPushesLowBug(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..Tl*.....\n..........\n..........\n..........\n..........\n..........\n......*...\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars}
	canonicalize(&s)
	var beetleIndex int
	for i, b := range s.Bugs {
		if b.Kind == Beetle {
			beetleIndex = i
		}
	}
	next, _, ok := applyMove(p, s, beetleIndex, Right)
	if !ok {
		t.Fatal("high beetle should enter landing mode")
	}
	if len(next.Bugs) != 1 || next.Bugs[0].Kind != Ladybird || next.Bugs[0].Pos != (Pos{2, 3}) {
		t.Fatalf("high beetle should bounce over, not push, the low bug: %+v", next.Bugs)
	}
}

func TestLowBugCannotClimb(t *testing.T) {
	p := puzzle(t, "max-moves 1\n..........\n..........\n..lO*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	s := State{Bugs: p.Bugs, Stars: p.Stars}
	if _, _, ok := applyMove(p, s, 0, Right); ok {
		t.Fatal("low ladybird should not climb onto a high platform")
	}
}

func TestZeroMoveLimitIsValid(t *testing.T) {
	p := puzzle(t, "max-moves 0\n..........\n..........\n..l.*.....\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n")
	if _, ok := Solve(p); ok {
		t.Fatal("unexpected solution")
	}
}

func TestRejectMismatchedStars(t *testing.T) {
	_, err := ParsePuzzle(strings.NewReader("max-moves 2\nl*........\n*.........\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n..........\n"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParserPadsOmittedTrailingCellsAndRows(t *testing.T) {
	p := puzzle(t, "max-moves 3\nn n *2\n")
	if len(p.Bugs) != 2 || p.Stars[0][2] != 2 {
		t.Fatalf("short spaced row parsed incorrectly: bugs=%+v stars=%d", p.Bugs, p.Stars[0][2])
	}
	for r := 1; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			if p.Terrain[r][c] != Void {
				t.Fatalf("omitted row %d column %d was not void", r+1, c+1)
			}
		}
	}
}

func TestSectionEndsShortenedBoard(t *testing.T) {
	p := puzzle(t, "max-moves 2\nn*\neggs:\negg 1 1\n")
	if len(p.Bugs) != 1 || !p.Bugs[0].Egg || p.Terrain[0][2] != Void || p.Terrain[1][0] != Void {
		t.Fatalf("short board with section parsed incorrectly: %+v", p)
	}
}
