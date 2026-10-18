package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPuzzleDirectoryReportsPassingAndMissingSolutions(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"passing.puzzle": "max-moves 1\nboard:\nn*\nsolution:\n1 n r\n",
		"missing.puzzle": "max-moves 1\nboard:\nn*\n",
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	if err := testPuzzleDirectory(dir, &output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"PASS ", "MISSING ", "1 solution(s) passed", "1 missing solution(s)"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("output does not contain %q:\n%s", want, output.String())
		}
	}
}

func TestPuzzleDirectoryFailsForInvalidPuzzleOrSolution(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"invalid.puzzle": "max-moves 1\nboard:\nn\n",
		"failed.puzzle":  "max-moves 1\nboard:\nn*\nsolution:\n1 n l\n",
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	if err := testPuzzleDirectory(dir, &output); err == nil {
		t.Fatal("invalid puzzle directory should fail")
	}
	if !strings.Contains(output.String(), "INVALID ") || !strings.Contains(output.String(), "FAIL ") {
		t.Fatalf("unexpected output:\n%s", output.String())
	}
}
