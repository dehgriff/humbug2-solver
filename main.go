package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	edit := flag.Bool("edit", false, "open a browser-based isometric puzzle editor")
	editAddr := flag.String("edit-addr", "127.0.0.1:8080", "address for the browser editor")
	play := flag.Bool("play", false, "play the puzzle interactively instead of solving it")
	playSolution := flag.Bool("play-soln", false, "step through the solution section in the puzzle file")
	testPuzzles := flag.Bool("test", false, "replay embedded solutions in every .puzzle file in a directory")
	reportPuzzles := flag.Bool("report", false, "report stored quality and difficulty analysis for a puzzle directory")
	reportCSV := flag.String("report-csv", "", "write the collection report rows to this CSV file")
	safe := flag.Bool("safe", false, "use exact BFS for puzzles containing pushing beetles")
	countSolutions := flag.Bool("count-solutions", false, "count all distinct shortest solution sequences")
	score := flag.Bool("score", false, "score the quality and difficulty of an embedded solution without solving")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [-play | -play-soln | -score | -edit] [-safe] [-count-solutions] PUZZLE\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "       %s -test [PUZZLE-DIRECTORY]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "       %s -report [-report-csv FILE] [PUZZLE-DIRECTORY]\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Find a shortest Humbug2 solution up to the puzzle's max-moves limit.")
	}
	flag.Parse()
	if *edit {
		if flag.NArg() != 1 || *play || *playSolution || *score || *testPuzzles || *reportPuzzles || *reportCSV != "" || *safe || *countSolutions {
			flag.Usage()
			os.Exit(2)
		}
		if err := runPuzzleEditor(flag.Arg(0), *editAddr); err != nil {
			fatal(err)
		}
		return
	}
	reportMode := *reportPuzzles || *reportCSV != ""
	if *testPuzzles || reportMode {
		if *testPuzzles && reportMode || *play || *playSolution || *score || *safe || *countSolutions || flag.NArg() > 1 {
			flag.Usage()
			os.Exit(2)
		}
		dir := "puzzles"
		if flag.NArg() == 1 {
			dir = flag.Arg(0)
		}
		var err error
		if reportMode {
			err = reportPuzzleDirectoryWithCSV(dir, os.Stdout, *reportCSV)
		} else {
			err = testPuzzleDirectory(dir, os.Stdout)
		}
		if err != nil {
			fatal(err)
		}
		return
	}
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if *play && *playSolution || *score && (*play || *playSolution || *countSolutions) || *countSolutions && (*play || *playSolution) {
		fatal(fmt.Errorf("-play, -play-soln, -score, and -count-solutions cannot be combined"))
	}
	f, err := os.Open(flag.Arg(0))
	if err != nil {
		fatal(err)
	}
	p, err := ParsePuzzle(f)
	closeErr := f.Close()
	if err != nil {
		fatal(fmt.Errorf("parse puzzle: %w", err))
	}
	if closeErr != nil {
		fatal(fmt.Errorf("close puzzle: %w", closeErr))
	}
	if *play {
		Play(p, os.Stdin, os.Stdout)
		return
	}
	if *playSolution {
		if err := PlaySolution(p, os.Stdin, os.Stdout); err != nil {
			fatal(fmt.Errorf("play solution in %s: %w", flag.Arg(0), err))
		}
		return
	}
	if *score {
		replay, err := readSolution(p)
		if err != nil {
			fatal(fmt.Errorf("score %s: %w", flag.Arg(0), err))
		}
		fmt.Println("Analyzing solution and alternative moves...")
		analysis, err := AnalyzePuzzle(p, replay.moves)
		if err != nil {
			fatal(fmt.Errorf("score %s: %w", flag.Arg(0), err))
		}
		if err := writePuzzleAnalysis(flag.Arg(0), analysis); err != nil {
			fatal(fmt.Errorf("write analysis to %s: %w", flag.Arg(0), err))
		}
		printPuzzleAnalysis(os.Stdout, analysis)
		fmt.Printf("Analysis written to %s\n", flag.Arg(0))
		return
	}
	options := SolveOptions{Safe: *safe, Progress: func(progress SearchProgress) {
		switch progress.Algorithm {
		case "astar":
			fmt.Printf("Lower bound %d: %d state(s) searched (%d total)\n", progress.Depth, progress.States, progress.TotalStates)
		default:
			fmt.Printf("Depth %d: %d state(s) (%d total)\n", progress.Depth, progress.States, progress.TotalStates)
		}
	}}
	moves, solved := SolveWithOptions(p, options)
	if !solved {
		fmt.Printf("No solution within %d moves.\n", p.MaxMoves)
		return
	}
	moveNotes, err := solutionMoveNotes(p, moves)
	if err != nil {
		fatal(fmt.Errorf("format solution output: %w", err))
	}
	if err := writePuzzleSolution(flag.Arg(0), p, moves); err != nil {
		fatal(fmt.Errorf("write solution to %s: %w", flag.Arg(0), err))
	}
	fmt.Printf("Shortest solution: %d move(s)\n", len(moves))
	for i, move := range moves {
		fmt.Printf("%d. %s", i+1, move)
		if len(moveNotes[i]) > 0 {
			fmt.Printf(" # %s", strings.Join(moveNotes[i], "; "))
		}
		fmt.Println()
	}
	fmt.Printf("Solution written to %s\n", flag.Arg(0))
	fmt.Println("Analyzing solution and alternative moves...")
	analysis, err := AnalyzePuzzle(p, moves)
	if err != nil {
		fatal(fmt.Errorf("score solution: %w", err))
	}
	if err := writePuzzleAnalysis(flag.Arg(0), analysis); err != nil {
		fatal(fmt.Errorf("write analysis to %s: %w", flag.Arg(0), err))
	}
	printPuzzleAnalysis(os.Stdout, analysis)
	fmt.Printf("Analysis written to %s\n", flag.Arg(0))
	if *countSolutions {
		fmt.Printf("Counting distinct solutions of length %d...\n", len(moves))
		count := CountSolutionsAtDepth(p, len(moves), *safe)
		fmt.Printf("Distinct shortest solutions: %s\n", count)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "humbug2:", err)
	os.Exit(1)
}
