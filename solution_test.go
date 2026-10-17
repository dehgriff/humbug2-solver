package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func twoSnailPuzzle(t *testing.T) Puzzle {
	t.Helper()
	return puzzle(t, "max-moves 2\n..........\n..........\nn*n*......\n")
}

func TestFormatSolutionUsesCoordinatesOnlyWhileTypeIsAmbiguous(t *testing.T) {
	p := twoSnailPuzzle(t)
	moves := []Move{
		{Kind: Snail, From: Pos{2, 0}, To: Pos{2, 1}, Dir: Right},
		{Kind: Snail, From: Pos{2, 2}, To: Pos{2, 3}, Dir: Right},
	}
	got, err := formatSolution(p, moves)
	if err != nil {
		t.Fatal(err)
	}
	if want := "1 n 3 1 r # snail merged with star at (3,2)\n2 n r # snail merged with star at (3,4)\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFormatSolutionNotesPassiveObjectsFallingOffBoard(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "puck", text: "max-moves 1\nboard:\ntu\n*\n", want: "1 t r # puck fell off board\n"},
		{name: "scorpion", text: "max-moves 1\nboard:\ntq\n*\n", want: "1 t r # scorpion 1 fell off board\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := puzzle(t, test.text)
			got, err := formatSolution(p, []Move{{Kind: Beetle, From: Pos{0, 0}, Dir: Right}})
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("solution = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSolutionInlineFallNoteRemainsReplayable(t *testing.T) {
	p := puzzle(t, "max-moves 1\nboard:\nn*\nsolution:\n1 n r # puck fell off board\n")
	if len(p.Solution) != 1 || p.Solution[0] != "n r" {
		t.Fatalf("parsed solution = %v", p.Solution)
	}
	if _, err := readSolution(p); err != nil {
		t.Fatal(err)
	}
}

func TestSolutionMoveNotesForStdout(t *testing.T) {
	p := puzzle(t, "max-moves 2\nboard:\ntu*\n")
	moves := []Move{
		{Kind: Beetle, From: Pos{0, 0}, Dir: Right},
		{Kind: Beetle, From: Pos{0, 1}, Dir: Right},
	}
	notes, err := solutionMoveNotes(p, moves)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 || len(notes[0]) != 0 || len(notes[1]) != 2 || notes[1][0] != "puck fell off board" || notes[1][1] != "beetle merged with star at (1,3)" {
		t.Fatalf("unexpected stdout notes: %v", notes)
	}
}

func TestParseStateMoveRequiresCoordinatesForDuplicateType(t *testing.T) {
	p := twoSnailPuzzle(t)
	state := initialState(p)
	if _, _, err := parseStateMove("n r", state); err == nil {
		t.Fatal("expected ambiguous shorthand to fail")
	}
	index, dir, err := parseStateMove("n 3 3 r", state)
	if err != nil || state.Bugs[index].Pos != (Pos{2, 2}) || dir != Right {
		t.Fatalf("index=%d dir=%v err=%v", index, dir, err)
	}
}

func TestReadAndNavigateSolution(t *testing.T) {
	p := twoSnailPuzzle(t)
	p.HasSolution = true
	p.Solution = []string{"n 3 1 r", "n r"}
	replay, err := readSolution(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(replay.moves) != 2 || len(replay.states) != 3 || remainingBugCount(replay.states[2]) != 0 {
		t.Fatalf("unexpected replay: %+v", replay)
	}

	var output bytes.Buffer
	err = PlaySolution(p, strings.NewReader("right\nr\nleft\nquit\n"), &output)
	if err != nil {
		t.Fatal(err)
	}
	text := output.String()
	for _, want := range []string{
		"Solution step 0/2. Next move: snail at (3,1) right -> (3,2)",
		"\x1b[1;31mn>\x1b[0m",
		"Applied: n 3 1 r",
		"Solution step 1/2. Next move: snail at (3,3) right -> (3,4)",
		"Solution step 2/2. Puzzle solved.",
		"Rewound: n r",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q:\n%s", want, text)
		}
	}
}

func TestPlaySolutionPrintsScorpionMovementOrder(t *testing.T) {
	p := puzzle(t, "max-moves 1\nboard:\nn*\nQQ\n")
	p.HasSolution = true
	p.Solution = []string{"n r"}
	var output bytes.Buffer
	if err := PlaySolution(p, strings.NewReader("right\nquit\n"), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Scorpion movement order: 1, 2") {
		t.Fatalf("movement order diagnostic missing:\n%s", output.String())
	}
}

func TestS22MoveFourUsesBoardPositionBeforeStableNumber(t *testing.T) {
	data, err := os.ReadFile("puzzles/s22.puzzle")
	if err != nil {
		t.Fatal(err)
	}
	p := puzzle(t, string(data))
	state := initialState(p)
	var order []uint8
	for moveNumber, line := range p.Solution[:4] {
		index, dir, err := parseStateMove(line, state)
		if err != nil {
			t.Fatalf("solution move %d: %v", moveNumber+1, err)
		}
		outcome := applyMoveDetailed(p, state, index, dir)
		if outcome.status != moveApplied {
			t.Fatalf("solution move %d: %s", moveNumber+1, outcome.reason)
		}
		state = outcome.state
		order = outcome.scorpionOrder
	}
	if len(order) == 0 || order[0] != 2 {
		t.Fatalf("move 4 scorpion order = %v, want scorpion 3 first", order)
	}
}

func TestS22SolutionThroughMoveNineteenUsesLeftTurn(t *testing.T) {
	data, err := os.ReadFile("puzzles/s22.puzzle")
	if err != nil {
		t.Fatal(err)
	}
	p := puzzle(t, string(data))
	state := initialState(p)
	for moveNumber, line := range p.Solution[:19] {
		index, dir, err := parseStateMove(line, state)
		if err != nil {
			t.Fatalf("solution move %d: %v", moveNumber+1, err)
		}
		outcome := applyMoveDetailed(p, state, index, dir)
		if outcome.status != moveApplied {
			t.Fatalf("solution move %d: %s", moveNumber+1, outcome.reason)
		}
		state = outcome.state
	}
	for _, bug := range state.Bugs {
		if bug.Kind == Scorpion && bug.ID == 0 {
			if bug.Pos != (Pos{4, 4}) {
				t.Fatalf("scorpion 1 after move 19 = (%d,%d), want (5,5)", bug.Pos.R+1, bug.Pos.C+1)
			}
			return
		}
	}
	t.Fatal("scorpion 1 not found after move 19")
}

func TestSolutionHighlightShowsNextMoveDirectionOnly(t *testing.T) {
	p := twoSnailPuzzle(t)
	p.HasSolution = true
	p.Solution = []string{"n 3 1 r", "n r"}
	replay, err := readSolution(p)
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	printSolutionState(&output, p, replay, 0)
	text := output.String()
	if !strings.Contains(text, "\x1b[1;31mn>\x1b[0m") {
		t.Fatalf("next bug is not bold red with its direction arrow:\n%s", text)
	}
	if !strings.Contains(text, "\x1b[1;32mn\x1b[0m") {
		t.Fatalf("other bug is not bold green:\n%s", text)
	}
	if !strings.Contains(text, "\x1b[33m*\x1b[0m") {
		t.Fatalf("star is not yellow:\n%s", text)
	}
	if count := strings.Count(text, "\x1b[1;31m"); count != 1 {
		t.Fatalf("got %d highlighted cells, want 1:\n%s", count, text)
	}

	output.Reset()
	printSolutionState(&output, p, replay, len(replay.moves))
	if strings.Contains(output.String(), "\x1b[1;31m") {
		t.Fatalf("solved board should not contain a next-move highlight:\n%s", output.String())
	}
}

func TestReadNavigationKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  navigationKey
	}{
		{name: "right arrow", input: "\x1b[C", want: navigationRight},
		{name: "left arrow", input: "\x1b[D", want: navigationLeft},
		{name: "application right arrow", input: "\x1bOC", want: navigationRight},
		{name: "modified left arrow", input: "\x1b[1;5D", want: navigationLeft},
		{name: "quit", input: "q", want: navigationQuit},
		{name: "control c", input: "\x03", want: navigationQuit},
		{name: "ignored", input: "r", want: navigationNone},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := readNavigationKey(strings.NewReader(test.input))
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("got %d, want %d", got, test.want)
			}
		})
	}
}

func TestWriteSolutionSectionIntoPuzzleFile(t *testing.T) {
	dir := t.TempDir()
	puzzlePath := filepath.Join(dir, "d12.puzzle")
	puzzleText := "# retained comment\nmax-moves 1\nboard:\n..n*\nsolution:\n1 n l\n"
	if err := os.WriteFile(puzzlePath, []byte(puzzleText), 0o644); err != nil {
		t.Fatal(err)
	}
	p := puzzle(t, puzzleText)
	moves := []Move{{Kind: Snail, From: Pos{0, 2}, To: Pos{0, 3}, Dir: Right}}
	if err := writePuzzleSolution(puzzlePath, p, moves); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(puzzlePath)
	if err != nil {
		t.Fatal(err)
	}
	want := "# retained comment\nmax-moves 1\nboard:\n..n*\nsolution:\n1 n r # snail merged with star at (1,4)\n"
	if string(data) != want {
		t.Fatalf("unexpected puzzle file %q, want %q", data, want)
	}
}

func TestReadSolutionRejectsUnsolvedMoveList(t *testing.T) {
	p := twoSnailPuzzle(t)
	p.HasSolution = true
	p.Solution = []string{"n 3 1 r"}
	if _, err := readSolution(p); err == nil {
		t.Fatal("expected incomplete solution to fail")
	}
}

func TestParseNumberedSolutionSection(t *testing.T) {
	p := puzzle(t, "max-moves 2\nboard:\nn*n*\nsolution:\n1 n 1 1 r\n2 n r\n")
	if !p.HasSolution || len(p.Solution) != 2 || p.Solution[0] != "n 1 1 r" || p.Solution[1] != "n r" {
		t.Fatalf("unexpected parsed solution: %+v", p.Solution)
	}
}

func TestD23EmbeddedSolutionReplays(t *testing.T) {
	data, err := os.ReadFile("puzzles/d23.puzzle")
	if err != nil {
		t.Fatal(err)
	}
	p := puzzle(t, string(data))
	replay, err := readSolution(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(replay.moves) != 10 || remainingBugCount(replay.states[len(replay.states)-1]) != 0 {
		t.Fatalf("unexpected d23 replay: %d moves", len(replay.moves))
	}
}

func TestPuzzleRejectsOutOfSequenceSolutionNumber(t *testing.T) {
	_, err := ParsePuzzle(strings.NewReader("max-moves 1\nboard:\nn*\nsolution:\n2 n r\n"))
	if err == nil || !strings.Contains(err.Error(), "expected solution move number 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}
