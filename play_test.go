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

func TestInteractivePlayPrintsScorpionMovementOrder(t *testing.T) {
	p := puzzle(t, "max-moves 1\nboard:\nn*\nQQ\n")
	var output bytes.Buffer
	Play(p, strings.NewReader("n r\nquit\n"), &output)
	if !strings.Contains(output.String(), "Scorpion movement order: 1, 2") {
		t.Fatalf("movement order diagnostic missing:\n%s", output.String())
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

func TestScorpionsRenderBoldBlue(t *testing.T) {
	var p Puzzle
	p.Terrain[1][1] = Low
	state := State{Bugs: []Bug{{Kind: Scorpion, Pos: Pos{1, 1}, Direction: Right}}}
	var output bytes.Buffer
	printBoardStateWithMove(&output, p, state, "Board", nil)
	if !strings.Contains(output.String(), "\x1b[1;34mq1>\x1b[0m") {
		t.Fatalf("scorpion is not bold blue:\n%s", output.String())
	}
}

func TestBoardRendersWallsBetweenAndAroundCells(t *testing.T) {
	p := puzzle(t, "max-moves 1\nboard:\nn*\nwalls:\nwall 1 1 up\nwall 1 1 left\nwall 1 1 down\nwall 1 1 right\nwall 1 2 up\nwall 1 2 right\nwall 1 2 down\n")
	state := initialState(p)
	var output bytes.Buffer
	printBoardState(&output, p, state, "Board")
	text := output.String()
	for _, want := range []string{"┌────┬────┐", "│ n  │ *  │", "└────┴────┘"} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered board missing %q:\n%s", want, text)
		}
	}
}

func TestIsolatedWallsDoNotExtendIntoAdjacentCells(t *testing.T) {
	p := puzzle(t, "max-moves 1\nboard:\nno*\nwalls:\nwall 1 1 right\nwall 1 3 down\n")
	state := initialState(p)
	var output bytes.Buffer
	printBoardState(&output, p, state, "Board")
	text := output.String()
	if strings.Count(text, "│") != 1 {
		t.Fatalf("isolated vertical wall should occupy one content row:\n%s", text)
	}
	if strings.Count(text, "─") != 4 {
		t.Fatalf("isolated horizontal wall should occupy one cell width:\n%s", text)
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
	text := output.String()
	if !strings.Contains(text, "     1    2    3    4") || !strings.Contains(text, "\n 1   .") || !strings.Contains(text, "\n 3   .    .    .    o") {
		t.Fatalf("unexpected trimmed board:\n%s", text)
	}
	if strings.Contains(text, "\n 4 ") || strings.Contains(text, "    5    ") {
		t.Fatalf("render included trailing rows or columns:\n%s", text)
	}
}
