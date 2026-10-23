package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	play := flag.Bool("play", false, "play the puzzle interactively instead of solving it")
	playSolution := flag.Bool("play-soln", false, "step through the solution section in the puzzle file")
	testPuzzles := flag.Bool("test", false, "replay embedded solutions in every .puzzle file in a directory")
	safe := flag.Bool("safe", false, "use exact BFS for puzzles containing pushing beetles")
	countSolutions := flag.Bool("count-solutions", false, "count all distinct shortest solution sequences")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [-play | -play-soln] [-safe] [-count-solutions] PUZZLE\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "       %s -test [PUZZLE-DIRECTORY]\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Find a shortest Humbug2 solution up to the puzzle's max-moves limit.")
	}
	flag.Parse()
	if *testPuzzles {
		if *play || *playSolution || *safe || *countSolutions || flag.NArg() > 1 {
			flag.Usage()
			os.Exit(2)
		}
		dir := "puzzles"
		if flag.NArg() == 1 {
			dir = flag.Arg(0)
		}
		if err := testPuzzleDirectory(dir, os.Stdout); err != nil {
			fatal(err)
		}
		return
	}
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if *play && *playSolution || *countSolutions && (*play || *playSolution) {
		fatal(fmt.Errorf("-play, -play-soln, and -count-solutions cannot be combined"))
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
	options := SolveOptions{Safe: *safe, Progress: func(progress SearchProgress) {
		switch progress.Algorithm {
		case "astar":
			fmt.Printf("Lower bound %d: %d state(s) searched (%d total)\n", progress.Depth, progress.States, progress.TotalStates)
		case "count":
			fmt.Printf("Counting distinct solutions of length %d...\n", progress.Depth)
		default:
			fmt.Printf("Depth %d: %d state(s) (%d total)\n", progress.Depth, progress.States, progress.TotalStates)
		}
	}}
	var moves []Move
	var solutionCount string
	var solved bool
	if *countSolutions {
		countedMoves, count, countedSolved := SolveAndCountShortest(p, options)
		moves, solved = countedMoves, countedSolved
		solutionCount = count.String()
	} else {
		moves, solved = SolveWithOptions(p, options)
	}
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
	if *countSolutions {
		fmt.Printf("Distinct shortest solutions: %s\n", solutionCount)
	}
	for i, move := range moves {
		fmt.Printf("%d. %s", i+1, move)
		if len(moveNotes[i]) > 0 {
			fmt.Printf(" # %s", strings.Join(moveNotes[i], "; "))
		}
		fmt.Println()
	}
	fmt.Printf("Solution written to %s\n", flag.Arg(0))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "humbug2:", err)
	os.Exit(1)
}
