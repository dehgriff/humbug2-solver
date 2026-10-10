package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParsePlayMove(t *testing.T) {
	state := State{Bugs: []Bug{{Kind: Snail, Pos: Pos{3, 5}}}}
	index, dir, err := parseStateMove("n u", state)
	if err != nil || index != 0 || dir != Up {
		t.Fatalf("got index=%d dir=%v err=%v", index, dir, err)
	}
	_, _, err = parseStateMove("4 6 u", state)
	if err == nil {
		t.Fatal("coordinate-only notation must be rejected")
	}
}

func TestInteractivePlaySolveAndUndo(t *testing.T) {
	p := puzzle(t, "max-moves 2\n..........\n..........\n..n*......\n")
	var output bytes.Buffer
	Play(p, strings.NewReader("n r\nundo\nquit\n"), &output)
	text := output.String()
	for _, want := range []string{
		"Move 0/2 — Puzzle in progress.",
		"\x1b[1;32mn\x1b[0m",
		"\x1b[33m*\x1b[0m",
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
	Play(p, strings.NewReader("n l\nundo\nquit\n"), &output)
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

func TestBoardRenderingOmitsOnlyTrailingRowsAndColumns(t *testing.T) {
	var p Puzzle
	p.Terrain[2][3] = Low
	rows, cols := renderedBoardSize(p)
	if rows != 3 || cols != 4 {
		t.Fatalf("got rendered size %dx%d, want 3x4", rows, cols)
	}
	var output bytes.Buffer
	printBoardState(&output, p, State{}, "Board")
	lines := strings.Split(output.String(), "\n")
	if len(lines) < 6 || !strings.Contains(lines[2], "1") || !strings.Contains(lines[3], ".") || !strings.Contains(lines[5], "o") {
		t.Fatalf("unexpected trimmed board:\n%s", output.String())
	}
	if strings.Contains(output.String(), "\n 4 ") || strings.Contains(lines[2], "5") {
		t.Fatalf("render included trailing rows or columns:\n%s", output.String())
	}
}
