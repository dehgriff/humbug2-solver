package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Play runs the interactive puzzle player.
func Play(p Puzzle, in io.Reader, out io.Writer) {
	state := State{Bugs: append([]Bug(nil), p.Bugs...), Stars: p.Stars, Walls: p.Walls}
	canonicalize(&state)
	history := make([]State, 0, p.MaxMoves)
	moveNumber := 0
	over := false

	fmt.Fprintln(out, "Enter a move as ROW COLUMN DIRECTION (for example: 4 6 u).")
	fmt.Fprintln(out, "Commands: undo, help, quit")
	printPlayState(out, p, state, moveNumber, "Puzzle in progress.")

	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprint(out, "> ")
		if !scanner.Scan() {
			fmt.Fprintln(out)
			return
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		switch strings.ToLower(line) {
		case "quit", "q", "exit":
			return
		case "help", "h", "?":
			fmt.Fprintln(out, "Move: ROW COLUMN u|r|d|l (coordinates are one-based).")
			fmt.Fprintln(out, "Other commands: undo, quit")
			continue
		case "undo":
			if len(history) == 0 {
				fmt.Fprintln(out, "Nothing to undo.")
				continue
			}
			state = history[len(history)-1]
			history = history[:len(history)-1]
			moveNumber--
			over = false
			printPlayState(out, p, state, moveNumber, "Move undone. Puzzle in progress.")
			continue
		}
		if over {
			fmt.Fprintln(out, "The puzzle is over. Enter undo or quit.")
			continue
		}

		row, col, dir, err := parsePlayMove(line)
		if err != nil {
			fmt.Fprintf(out, "Invalid input: %v\n", err)
			continue
		}
		index := bugAt(state.Bugs, Pos{row, col})
		if index < 0 {
			fmt.Fprintf(out, "Invalid move: square (%d,%d) does not contain an object.\n", row+1, col+1)
			continue
		}
		outcome := applyMoveDetailed(p, state, index, dir)
		if outcome.status == moveInvalid {
			fmt.Fprintf(out, "Invalid move: %s.\n", outcome.reason)
			continue
		}
		history = append(history, state)
		state = outcome.state
		moveNumber++
		if outcome.status == moveApplied {
			fmt.Fprintf(out, "Played: %s\n", outcome.move)
		}
		if outcome.status == moveLost {
			over = true
			printPlayState(out, p, state, moveNumber, "Puzzle over: "+outcome.reason+".")
			continue
		}
		if remainingBugCount(state) == 0 {
			over = true
			printPlayState(out, p, state, moveNumber, "Puzzle solved.")
			continue
		}
		if moveNumber >= p.MaxMoves {
			over = true
			printPlayState(out, p, state, moveNumber, "Puzzle over: maximum move count reached.")
			continue
		}
		printPlayState(out, p, state, moveNumber, "Puzzle in progress.")
	}
}

func parsePlayMove(line string) (int, int, Direction, error) {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return 0, 0, Up, fmt.Errorf("expected ROW COLUMN DIRECTION")
	}
	row, errRow := strconv.Atoi(fields[0])
	col, errCol := strconv.Atoi(fields[1])
	if errRow != nil || errCol != nil || row < 1 || row > BoardSize || col < 1 || col > BoardSize {
		return 0, 0, Up, fmt.Errorf("row and column must be between 1 and %d", BoardSize)
	}
	dir, ok := parsePlayDirection(fields[2])
	if !ok {
		return 0, 0, Up, fmt.Errorf("direction must be u, r, d, l or its full name")
	}
	return row - 1, col - 1, dir, nil
}

func parsePlayDirection(value string) (Direction, bool) {
	switch strings.ToLower(value) {
	case "u", "up":
		return Up, true
	case "r", "right":
		return Right, true
	case "d", "down":
		return Down, true
	case "l", "left":
		return Left, true
	default:
		return Up, false
	}
}

func bugAt(bugs []Bug, pos Pos) int {
	for i := range bugs {
		if bugs[i].Pos == pos {
			return i
		}
	}
	return -1
}

func printPlayState(out io.Writer, p Puzzle, state State, moveNumber int, status string) {
	fmt.Fprintf(out, "\nMove %d/%d — %s\n", moveNumber, p.MaxMoves, status)
	fmt.Fprint(out, "       1     2     3     4     5     6     7     8     9    10\n")
	for r := 0; r < BoardSize; r++ {
		fmt.Fprintf(out, "%2d ", r+1)
		for c := 0; c < BoardSize; c++ {
			fmt.Fprintf(out, "%-5s ", playCellToken(p, state, Pos{r, c}))
		}
		fmt.Fprintln(out)
	}
	printWalls(out, state.Walls)
	fmt.Fprintln(out, "Tokens: e=egg, a2/a3=ant group, q1>/q2^=scorpion number/direction")
}

func playCellToken(p Puzzle, state State, pos Pos) string {
	index := bugAt(state.Bugs, pos)
	if index >= 0 {
		bug := state.Bugs[index]
		code := byte(bug.Kind)
		if p.Terrain[pos.R][pos.C] == Low && code >= 'A' && code <= 'Z' {
			code += 'a' - 'A'
		}
		token := string(code)
		if bug.Kind == Ant && antCount(bug) > 1 {
			token += strconv.Itoa(int(antCount(bug)))
		}
		if bug.Kind == Scorpion {
			token += strconv.Itoa(int(bug.ID) + 1)
		}
		if bug.Egg {
			token += "e"
		} else if bug.Kind == Scorpion {
			token += directionArrow(bug.Direction)
		}
		if stars := state.Stars[pos.R][pos.C]; stars > 0 {
			token += starToken(p.Terrain[pos.R][pos.C], stars)
		}
		return token
	}
	if stars := state.Stars[pos.R][pos.C]; stars > 0 {
		return starToken(p.Terrain[pos.R][pos.C], stars)
	}
	switch p.Terrain[pos.R][pos.C] {
	case Low:
		if p.Trampolines[pos.R][pos.C] {
			return "x"
		}
		return "o"
	case High:
		if p.Trampolines[pos.R][pos.C] {
			return "X"
		}
		return "O"
	default:
		return "."
	}
}

func starToken(level Terrain, count uint8) string {
	token := "*"
	if level == High {
		token = "@"
	}
	if count > 1 {
		token += strconv.Itoa(int(count))
	}
	return token
}

func directionArrow(dir Direction) string {
	return []string{"^", ">", "v", "<"}[dir]
}

func printWalls(out io.Writer, walls [BoardSize][BoardSize]uint8) {
	var entries []string
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			pos := Pos{r, c}
			for _, dir := range directions {
				if walls[r][c]&(1<<dir) == 0 {
					continue
				}
				next := add(pos, deltas[dir])
				if inside(next) && (dir == Up || dir == Left) {
					continue // the other square prints this shared wall
				}
				entries = append(entries, fmt.Sprintf("(%d,%d) %s", r+1, c+1, dir))
			}
		}
	}
	if len(entries) == 0 {
		fmt.Fprintln(out, "Walls: none")
		return
	}
	fmt.Fprintln(out, "Walls:", strings.Join(entries, ", "))
}
