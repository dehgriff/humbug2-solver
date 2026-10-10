package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	play := flag.Bool("play", false, "play the puzzle interactively instead of solving it")
	playSolution := flag.Bool("play-soln", false, "step through the solution section in the puzzle file")
	safe := flag.Bool("safe", false, "use exact BFS for puzzles containing pushing beetles")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [-play | -play-soln] [-safe] PUZZLE\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Find a shortest Humbug2 solution up to the puzzle's max-moves limit.")
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	if *play && *playSolution {
		fatal(fmt.Errorf("-play and -play-soln cannot be used together"))
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
	moves, solved := SolveWithOptions(p, SolveOptions{Safe: *safe, Progress: func(progress SearchProgress) {
		if progress.Algorithm == "astar" {
			fmt.Printf("Lower bound %d: %d state(s) searched (%d total)\n", progress.Depth, progress.States, progress.TotalStates)
		} else {
			fmt.Printf("Depth %d: %d state(s) (%d total)\n", progress.Depth, progress.States, progress.TotalStates)
		}
	}})
	if !solved {
		fmt.Printf("No solution within %d moves.\n", p.MaxMoves)
		return
	}
	if err := writePuzzleSolution(flag.Arg(0), p, moves); err != nil {
		fatal(fmt.Errorf("write solution to %s: %w", flag.Arg(0), err))
	}
	fmt.Printf("Shortest solution: %d move(s)\n", len(moves))
	for i, move := range moves {
		fmt.Printf("%d. %s\n", i+1, move)
	}
	fmt.Printf("Solution written to %s\n", flag.Arg(0))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "humbug2:", err)
	os.Exit(1)
}
