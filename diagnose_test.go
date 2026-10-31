package main

import (
	"strings"
	"testing"
)

func TestDiagnoseReportsBugThatCanNeverScore(t *testing.T) {
	p := puzzle(t, "max-moves 4\nboard:\nf*\n")
	var output strings.Builder
	if err := DiagnosePuzzle(p, &output); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "Closest state: 1 goal bug(s)") ||
		!strings.Contains(text, "butterfly never merges with a star") ||
		!strings.Contains(text, "reachable state graph was exhausted") {
		t.Fatalf("unexpected diagnosis:\n%s", text)
	}
}

func TestDiagnoseFindsClosestStateAndPath(t *testing.T) {
	p := puzzle(t, "max-moves 3\nboard:\nn*n.*\n")
	var output strings.Builder
	if err := DiagnosePuzzle(p, &output); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "Closest state: 1 goal bug(s) and 1 star(s) remain after 1 move(s)") ||
		!strings.Contains(text, "1. snail") {
		t.Fatalf("unexpected diagnosis:\n%s", text)
	}
}
