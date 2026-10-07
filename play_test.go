package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParsePlayMove(t *testing.T) {
	row, col, dir, err := parsePlayMove("4 6 u")
	if err != nil || row != 3 || col != 5 || dir != Up {
		t.Fatalf("got row=%d col=%d dir=%v err=%v", row, col, dir, err)
	}
	_, _, _, err = parsePlayMove("4 6 sideways")
	if err == nil {
		t.Fatal("expected invalid direction")
	}
}

func TestInteractivePlaySolveAndUndo(t *testing.T) {
	p := puzzle(t, "max-moves 2\n..........\n..........\n..n*......\n")
	var output bytes.Buffer
	Play(p, strings.NewReader("3 3 r\nundo\nquit\n"), &output)
	text := output.String()
	for _, want := range []string{
		"Move 0/2 — Puzzle in progress.",
		"Played: snail at (3,3) right -> (3,4)",
		"Move 1/2 — Puzzle solved.",
		"Move 0/2 — Move undone. Puzzle in progress.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func TestInteractivePlayCanUndoTerminalFall(t *testing.T) {
	p := puzzle(t, "max-moves 2\nn.........\n*.........\n")
	var output bytes.Buffer
	Play(p, strings.NewReader("1 1 l\nundo\nquit\n"), &output)
	text := output.String()
	if !strings.Contains(text, "Move 1/2 — Puzzle over: snail fell off the board.") ||
		!strings.Contains(text, "Move 0/2 — Move undone. Puzzle in progress.") {
		t.Fatalf("unexpected output:\n%s", text)
	}
}

func TestPlayCellTokenShowsScorpionState(t *testing.T) {
	var p Puzzle
	p.Terrain[2][3] = Low
	state := State{Bugs: []Bug{{Kind: Scorpion, Pos: Pos{2, 3}, ID: 1, Direction: Left}}}
	if got := playCellToken(p, state, Pos{2, 3}); got != "q2<" {
		t.Fatalf("got %q", got)
	}
	state.Bugs[0].Egg = true
	if got := playCellToken(p, state, Pos{2, 3}); got != "q2e" {
		t.Fatalf("got egg token %q", got)
	}
}
