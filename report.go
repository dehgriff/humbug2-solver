package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

type storedAnalysis struct {
	Version, Quality, Moves, Tightness, InteractionMoves int
	DecisionLandscape, Optimal, Detours, DeadEnds        int
	Unresolved, ScorpionLoad                             int
	Difficulty                                           float64
	TimedOut                                             bool
	ParseError                                           error
}

type reportRow struct {
	Name        string
	Puzzle      Puzzle
	Analysis    storedAnalysis
	Invalid     error
	Flags       []string
	Unresolved  bool
	HasAnalysis bool
}

type collectionSummary struct {
	Files, Valid, Solved, Version2, MissingSolutions int
	MissingAnalysis, StaleAnalysis, BadAnalysis      int
	Invalid                                          int
	TimedOut, UnresolvedHeavy, LowQuality            int
}

func reportPuzzleDirectory(dir string, out io.Writer) error {
	return reportPuzzleDirectoryWithCSV(dir, out, "")
}

func reportPuzzleDirectoryWithCSV(dir string, out io.Writer, csvPath string) error {
	paths, err := puzzlePathsRecursive(dir)
	if err != nil {
		return err
	}
	rows := make([]reportRow, 0, len(paths))
	summary := collectionSummary{Files: len(paths)}
	for _, path := range paths {
		name, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			name = path
		}
		row := reportRow{Name: name}
		f, openErr := os.Open(path)
		if openErr != nil {
			row.Invalid = openErr
			row.Flags = append(row.Flags, "INVALID")
			summary.Invalid++
			rows = append(rows, row)
			continue
		}
		p, parseErr := ParsePuzzle(f)
		closeErr := f.Close()
		if parseErr != nil {
			row.Invalid = parseErr
		} else if closeErr != nil {
			row.Invalid = closeErr
		}
		if row.Invalid != nil {
			row.Flags = append(row.Flags, "INVALID")
			summary.Invalid++
			rows = append(rows, row)
			continue
		}
		row.Puzzle = p
		summary.Valid++
		if !p.HasSolution {
			row.Flags = append(row.Flags, "NO-SOLUTION")
			summary.MissingSolutions++
		} else {
			summary.Solved++
		}
		if !p.HasAnalysis {
			row.Flags = append(row.Flags, "NO-ANALYSIS")
			summary.MissingAnalysis++
			rows = append(rows, row)
			continue
		}
		row.HasAnalysis = true
		row.Analysis = parseStoredAnalysis(p.Analysis)
		if row.Analysis.ParseError != nil {
			row.Flags = append(row.Flags, "BAD-ANALYSIS")
			summary.BadAnalysis++
		} else if row.Analysis.Version != analysisVersion {
			row.Flags = append(row.Flags, fmt.Sprintf("V%d", row.Analysis.Version))
			summary.StaleAnalysis++
		} else {
			summary.Version2++
		}
		if row.Analysis.TimedOut {
			row.Flags = append(row.Flags, "TIMEOUT")
			summary.TimedOut++
		}
		totalAlternatives := row.Analysis.Optimal + row.Analysis.Detours + row.Analysis.DeadEnds + row.Analysis.Unresolved
		row.Unresolved = totalAlternatives > 0 && row.Analysis.Unresolved*5 > totalAlternatives
		if row.Unresolved {
			row.Flags = append(row.Flags, "UNRESOLVED")
			summary.UnresolvedHeavy++
		}
		if row.Analysis.Quality > 0 && row.Analysis.Quality < 60 {
			row.Flags = append(row.Flags, "LOW-Q")
			summary.LowQuality++
		}
		if row.Analysis.Difficulty >= 8 {
			row.Flags = append(row.Flags, "HARD")
		} else if row.Analysis.Difficulty > 0 && row.Analysis.Difficulty <= 3 {
			row.Flags = append(row.Flags, "EASY")
		}
		rows = append(rows, row)
	}

	printCollectionTable(out, rows)
	fmt.Fprintf(out, "\nSummary: %d puzzle(s), %d valid, %d solved, %d with version %d analysis\n",
		summary.Files, summary.Valid, summary.Solved, summary.Version2, analysisVersion)
	fmt.Fprintf(out, "Attention: %d missing solution, %d missing analysis, %d stale analysis, %d malformed analysis, %d invalid, %d timed out, %d heavily unresolved, %d low quality\n",
		summary.MissingSolutions, summary.MissingAnalysis, summary.StaleAnalysis, summary.BadAnalysis, summary.Invalid, summary.TimedOut, summary.UnresolvedHeavy, summary.LowQuality)
	printCollectionExtremes(out, rows)
	printSimilarProfiles(out, rows)
	fmt.Fprintln(out, "Flags: NO-SOLUTION, NO-ANALYSIS, Vn=stale version, BAD-ANALYSIS, INVALID, TIMEOUT, UNRESOLVED=>20%, LOW-Q=<60, HARD=>=8, EASY=<=3")
	if csvPath != "" {
		if err := writeCollectionCSV(csvPath, rows); err != nil {
			return fmt.Errorf("write CSV report: %w", err)
		}
		fmt.Fprintf(out, "CSV written to %s\n", csvPath)
	}
	return nil
}

func writeCollectionCSV(path string, rows []reportRow) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".humbug2-report-*.csv")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(statErr) {
		temp.Close()
		return statErr
	}
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return err
	}

	w := csv.NewWriter(temp)
	header := []string{
		"puzzle", "valid", "has_solution", "has_analysis", "analysis_version",
		"quality", "difficulty", "moves", "tightness_percent", "interaction_moves",
		"decision_landscape", "optimal_moves", "detour_moves", "dead_end_moves",
		"unresolved_moves", "scorpion_load", "analysis_timed_out", "flags", "error",
	}
	if err := w.Write(header); err != nil {
		temp.Close()
		return err
	}
	for _, row := range rows {
		record := collectionCSVRecord(row)
		if err := w.Write(record); err != nil {
			temp.Close()
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}

func collectionCSVRecord(row reportRow) []string {
	valid := row.Invalid == nil
	record := []string{
		row.Name,
		strconv.FormatBool(valid),
		strconv.FormatBool(valid && row.Puzzle.HasSolution),
		strconv.FormatBool(valid && row.HasAnalysis),
		"", "", "", "", "", "", "", "", "", "", "", "", "",
		strings.Join(row.Flags, ";"),
		"",
	}
	if row.Invalid != nil {
		record[len(record)-1] = row.Invalid.Error()
		return record
	}
	if !row.HasAnalysis {
		if row.Puzzle.HasSolution {
			record[7] = strconv.Itoa(len(row.Puzzle.Solution))
		}
		return record
	}
	a := row.Analysis
	record[4] = strconv.Itoa(a.Version)
	record[5] = strconv.Itoa(a.Quality)
	record[6] = strconv.FormatFloat(a.Difficulty, 'f', 1, 64)
	record[7] = strconv.Itoa(a.Moves)
	record[8] = strconv.Itoa(a.Tightness)
	record[9] = strconv.Itoa(a.InteractionMoves)
	record[10] = strconv.Itoa(a.DecisionLandscape)
	record[11] = strconv.Itoa(a.Optimal)
	record[12] = strconv.Itoa(a.Detours)
	record[13] = strconv.Itoa(a.DeadEnds)
	record[14] = strconv.Itoa(a.Unresolved)
	record[15] = strconv.Itoa(a.ScorpionLoad)
	record[16] = strconv.FormatBool(a.TimedOut)
	if a.ParseError != nil {
		record[len(record)-1] = a.ParseError.Error()
	}
	return record
}

func puzzlePathsRecursive(dir string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".puzzle") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk puzzle directory %s: %w", dir, err)
	}
	sort.Strings(paths)
	return paths, nil
}

func parseStoredAnalysis(lines []string) storedAnalysis {
	values := make(map[string]string)
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			values[strings.ToLower(fields[0])] = fields[1]
		}
	}
	var a storedAnalysis
	var errors []string
	parseInt := func(key string, target *int) {
		value, exists := values[key]
		if !exists {
			return
		}
		n, err := strconv.Atoi(value)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s=%q", key, value))
			return
		}
		*target = n
	}
	parseInt("version", &a.Version)
	parseInt("quality", &a.Quality)
	parseInt("moves", &a.Moves)
	parseInt("tightness", &a.Tightness)
	parseInt("interaction-moves", &a.InteractionMoves)
	parseInt("decision-landscape", &a.DecisionLandscape)
	parseInt("optimal-moves", &a.Optimal)
	parseInt("detour-moves", &a.Detours)
	parseInt("dead-end-moves", &a.DeadEnds)
	parseInt("unresolved-moves", &a.Unresolved)
	parseInt("scorpion-load", &a.ScorpionLoad)
	if value, exists := values["difficulty"]; exists {
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			errors = append(errors, fmt.Sprintf("difficulty=%q", value))
		} else {
			a.Difficulty = n
		}
	}
	if value, exists := values["analysis-timed-out"]; exists {
		n, err := strconv.ParseBool(value)
		if err != nil {
			errors = append(errors, fmt.Sprintf("analysis-timed-out=%q", value))
		} else {
			a.TimedOut = n
		}
	}
	if len(errors) > 0 {
		a.ParseError = fmt.Errorf("invalid field(s): %s", strings.Join(errors, ", "))
	}
	return a
}

func printCollectionTable(out io.Writer, rows []reportRow) {
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "PUZZLE\tVER\tQ\tDIFF\tMOVES\tTIGHT%\tINT-MV\tLAND\tOPT\tDET\tDEAD\tUNR\tSCORP\tFLAGS")
	for _, row := range rows {
		if row.Invalid != nil {
			fmt.Fprintf(w, "%s\t-\t-\t-\t-\t-\t-\t-\t-\t-\t-\t-\t-\t%s\n", row.Name, strings.Join(row.Flags, ","))
			continue
		}
		if !row.HasAnalysis {
			moves := "-"
			if row.Puzzle.HasSolution {
				moves = strconv.Itoa(len(row.Puzzle.Solution))
			}
			fmt.Fprintf(w, "%s\t-\t-\t-\t%s\t-\t-\t-\t-\t-\t-\t-\t-\t%s\n", row.Name, moves, strings.Join(row.Flags, ","))
			continue
		}
		a := row.Analysis
		fmt.Fprintf(w, "%s\t%d\t%d\t%.1f\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n",
			row.Name, a.Version, a.Quality, a.Difficulty, a.Moves, a.Tightness, a.InteractionMoves,
			a.DecisionLandscape, a.Optimal, a.Detours, a.DeadEnds, a.Unresolved, a.ScorpionLoad, strings.Join(row.Flags, ","))
	}
	w.Flush()
}

func printCollectionExtremes(out io.Writer, rows []reportRow) {
	var scored []reportRow
	for _, row := range rows {
		if row.HasAnalysis && row.Analysis.ParseError == nil && row.Analysis.Quality > 0 {
			scored = append(scored, row)
		}
	}
	if len(scored) == 0 {
		return
	}
	minQuality, maxQuality := scored[0], scored[0]
	minDifficulty, maxDifficulty := scored[0], scored[0]
	for _, row := range scored[1:] {
		if row.Analysis.Quality < minQuality.Analysis.Quality {
			minQuality = row
		}
		if row.Analysis.Quality > maxQuality.Analysis.Quality {
			maxQuality = row
		}
		if row.Analysis.Difficulty < minDifficulty.Analysis.Difficulty {
			minDifficulty = row
		}
		if row.Analysis.Difficulty > maxDifficulty.Analysis.Difficulty {
			maxDifficulty = row
		}
	}
	fmt.Fprintf(out, "Ranges: quality %d (%s) to %d (%s); difficulty %.1f (%s) to %.1f (%s)\n",
		minQuality.Analysis.Quality, minQuality.Name, maxQuality.Analysis.Quality, maxQuality.Name,
		minDifficulty.Analysis.Difficulty, minDifficulty.Name, maxDifficulty.Analysis.Difficulty, maxDifficulty.Name)
}

func printSimilarProfiles(out io.Writer, rows []reportRow) {
	type pair struct{ left, right string }
	var pairs []pair
	for i := 0; i < len(rows); i++ {
		a := rows[i]
		if !comparableReportRow(a) {
			continue
		}
		for j := i + 1; j < len(rows); j++ {
			b := rows[j]
			if !comparableReportRow(b) {
				continue
			}
			if abs(a.Analysis.Quality-b.Analysis.Quality) <= 1 &&
				math.Abs(a.Analysis.Difficulty-b.Analysis.Difficulty) <= .1 &&
				abs(a.Analysis.Moves-b.Analysis.Moves) <= 1 &&
				abs(a.Analysis.Tightness-b.Analysis.Tightness) <= 2 &&
				abs(a.Analysis.DecisionLandscape-b.Analysis.DecisionLandscape) <= 3 &&
				abs(a.Analysis.ScorpionLoad-b.Analysis.ScorpionLoad) <= 3 {
				pairs = append(pairs, pair{a.Name, b.Name})
			}
		}
	}
	if len(pairs) == 0 {
		return
	}
	fmt.Fprintf(out, "Potentially similar metric profiles: %d pair(s)\n", len(pairs))
	limit := len(pairs)
	if limit > 10 {
		limit = 10
	}
	for _, pair := range pairs[:limit] {
		fmt.Fprintf(out, "  %s ~ %s\n", pair.left, pair.right)
	}
	if len(pairs) > limit {
		fmt.Fprintf(out, "  ... and %d more\n", len(pairs)-limit)
	}
}

func comparableReportRow(row reportRow) bool {
	return row.HasAnalysis && row.Analysis.ParseError == nil && row.Analysis.Version == analysisVersion && !row.Analysis.TimedOut
}
