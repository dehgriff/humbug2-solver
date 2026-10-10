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
	if want := "1 n 3 1 r\n2 n r\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
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
	want := "# retained comment\nmax-moves 1\nboard:\n..n*\nsolution:\n1 n r\n"
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
