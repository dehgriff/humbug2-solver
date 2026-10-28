package main

import (
	"strings"
	"testing"
)

func TestMinimiseRemovesUnusedPlatformAndWall(t *testing.T) {
	p := puzzle(t, "max-moves 1\nboard:\nn*o\nwalls:\nwall 1 3 right\nsolution:\n1 n r\n")
	var output strings.Builder
	if err := MinimisePuzzle(p, &output); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "removed 1 platform(s) and 1 wall(s)") {
		t.Fatalf("unexpected summary:\n%s", text)
	}
	minimised, err := ParsePuzzle(strings.NewReader(text))
	if err != nil {
		t.Fatalf("minimised output does not parse: %v\n%s", err, text)
	}
	if minimised.Terrain[0][2] != Void || uniqueWallCount(minimised.Walls) != 0 {
		t.Fatalf("unused board elements remain: terrain=%v walls=%d", minimised.Terrain[0][2], uniqueWallCount(minimised.Walls))
	}
	if _, err := readSolution(minimised); err != nil {
		t.Fatalf("retained solution does not replay: %v", err)
	}
}

func TestMinimiseKeepsPlatformCrossedBySolution(t *testing.T) {
	p := puzzle(t, "max-moves 1\nboard:\nlo*\nsolution:\n1 l r\n")
	var output strings.Builder
	if err := MinimisePuzzle(p, &output); err != nil {
		t.Fatal(err)
	}
	minimised, err := ParsePuzzle(strings.NewReader(output.String()))
	if err != nil {
		t.Fatal(err)
	}
	if minimised.Terrain[0][1] == Void {
		t.Fatal("removed an intermediate platform required by the solution")
	}
}

func TestMinimiseRequiresEmbeddedSolution(t *testing.T) {
	p := puzzle(t, "max-moves 1\nboard:\nn*\n")
	if err := MinimisePuzzle(p, &strings.Builder{}); err == nil {
		t.Fatal("expected missing solution error")
	}
}

func TestRemovalRejectedWhenItCreatesShorterSolution(t *testing.T) {
	p := puzzle(t, "max-moves 3\nboard:\nooo\nso*\nwalls:\nwall 1 1 up\nwall 1 3 right\nwall 2 1 right\nwall 2 3 right\nwall 2 3 down\nsolution:\n1 s u\n2 s r\n3 s d\n")
	candidate := p
	knockDownWall(&candidate.Walls, Pos{1, 0}, Right)
	if _, err := readSolution(candidate); err != nil {
		t.Fatalf("original three-move route should still work after removal: %v", err)
	}
	if removalPreservesShortestSolution(candidate, 3) {
		t.Fatal("accepted a wall removal that creates a one-move solution")
	}
}
