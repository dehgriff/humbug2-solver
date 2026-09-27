package main

import (
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
