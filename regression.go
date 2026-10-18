package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type puzzleTestSummary struct {
	Files            int
	PassedSolutions  int
	MissingSolutions int
	InvalidPuzzles   int
	FailedSolutions  int
}

// testPuzzleDirectory parses every .puzzle file and replays every embedded
// solution. Files without a solution are reported separately so an unfinished
// puzzle cannot be mistaken for a passing regression test.
func testPuzzleDirectory(dir string, out io.Writer) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read puzzle directory %s: %w", dir, err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".puzzle") {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(paths)

	summary := puzzleTestSummary{Files: len(paths)}
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			summary.InvalidPuzzles++
			fmt.Fprintf(out, "INVALID %s: %v\n", path, err)
			continue
		}
		p, parseErr := ParsePuzzle(f)
		closeErr := f.Close()
		if parseErr != nil {
			summary.InvalidPuzzles++
			fmt.Fprintf(out, "INVALID %s: %v\n", path, parseErr)
			continue
		}
		if closeErr != nil {
			summary.InvalidPuzzles++
			fmt.Fprintf(out, "INVALID %s: %v\n", path, closeErr)
			continue
		}
		if !p.HasSolution {
			summary.MissingSolutions++
			fmt.Fprintf(out, "MISSING %s: no solution section\n", path)
			continue
		}
		replay, err := readSolution(p)
		if err != nil {
			summary.FailedSolutions++
			fmt.Fprintf(out, "FAIL %s: %v\n", path, err)
			continue
		}
		summary.PassedSolutions++
		fmt.Fprintf(out, "PASS %s: %d move(s)\n", path, len(replay.moves))
	}

	fmt.Fprintf(out, "Summary: %d file(s), %d solution(s) passed, %d missing solution(s), %d invalid puzzle(s), %d failed solution(s)\n",
		summary.Files, summary.PassedSolutions, summary.MissingSolutions, summary.InvalidPuzzles, summary.FailedSolutions)
	if summary.InvalidPuzzles > 0 || summary.FailedSolutions > 0 {
		return fmt.Errorf("puzzle regression test failed")
	}
	return nil
}
