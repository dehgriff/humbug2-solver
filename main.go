package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	play := flag.Bool("play", false, "play the puzzle interactively instead of solving it")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [-play] PUZZLE\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "Find a shortest Humbug2 solution up to the puzzle's max-moves limit.")
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	f, err := os.Open(flag.Arg(0))
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	p, err := ParsePuzzle(f)
	if err != nil {
		fatal(fmt.Errorf("parse puzzle: %w", err))
	}
	if *play {
		Play(p, os.Stdin, os.Stdout)
		return
	}
	moves, solved := SolveWithProgress(p, func(progress SearchProgress) {
		if progress.Algorithm == "astar" {
			fmt.Printf("Lower bound %d: %d state(s) searched (%d total)\n", progress.Depth, progress.States, progress.TotalStates)
		} else {
			fmt.Printf("Depth %d: %d state(s) (%d total)\n", progress.Depth, progress.States, progress.TotalStates)
		}
	})
	if !solved {
		fmt.Printf("No solution within %d moves.\n", p.MaxMoves)
		return
	}
	fmt.Printf("Shortest solution: %d move(s)\n", len(moves))
	for i, move := range moves {
		fmt.Printf("%d. %s\n", i+1, move)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "humbug2:", err)
	os.Exit(1)
}
