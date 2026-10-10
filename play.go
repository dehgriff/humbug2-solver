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

	fmt.Fprintln(out, "Enter TYPE DIRECTION when that type is unique (for example: s r).")
	fmt.Fprintln(out, "Otherwise enter TYPE ROW COLUMN DIRECTION (for example: p 5 2 r).")
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
			fmt.Fprintln(out, "Move: TYPE u|r|d|l, or TYPE ROW COLUMN u|r|d|l when that type is not unique.")
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

		index, dir, err := parseStateMove(line, state)
		if err != nil {
			fmt.Fprintf(out, "Invalid input: %v\n", err)
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
	printBoardStateWithMove(out, p, state, fmt.Sprintf("Move %d/%d — %s", moveNumber, p.MaxMoves, status), nil)
}

func printBoardState(out io.Writer, p Puzzle, state State, heading string) {
	printBoardStateDecorated(out, p, state, heading, nil, false)
}

func printBoardStateWithMove(out io.Writer, p Puzzle, state State, heading string, nextMove *Move) {
	printBoardStateDecorated(out, p, state, heading, nextMove, true)
}

func printBoardStateDecorated(out io.Writer, p Puzzle, state State, heading string, nextMove *Move, solutionColours bool) {
	fmt.Fprintf(out, "\n%s\n", heading)
	rows, cols := renderedBoardSize(p)
	fmt.Fprint(out, "   ")
	for c := 1; c <= cols; c++ {
		fmt.Fprintf(out, "%-5d ", c)
	}
	fmt.Fprintln(out)
	for r := 0; r < rows; r++ {
		fmt.Fprintf(out, "%2d ", r+1)
		for c := 0; c < cols; c++ {
			pos := Pos{r, c}
			token := playCellToken(p, state, pos)
			if solutionColours {
				writeSolutionCell(out, p, state, pos, token, nextMove)
				continue
			}
			fmt.Fprintf(out, "%-5s ", token)
		}
		fmt.Fprintln(out)
	}
	printWalls(out, state.Walls)
	fmt.Fprintln(out, "Tokens: e=egg, a2/a3=ant group, q1>/q2^=scorpion number/direction")
}

func writeSolutionCell(out io.Writer, p Puzzle, state State, pos Pos, token string, nextMove *Move) {
	const (
		boldRed   = "\x1b[1;31m"
		boldGreen = "\x1b[1;32m"
		yellow    = "\x1b[33m"
		reset     = "\x1b[0m"
	)

	bugIndex := bugAt(state.Bugs, pos)
	stars := state.Stars[pos.R][pos.C]
	star := ""
	bugToken := token
	if stars > 0 {
		star = starToken(p.Terrain[pos.R][pos.C], stars)
		if bugIndex >= 0 {
			bugToken = strings.TrimSuffix(token, star)
		}
	}

	visibleLength := len(token)
	if bugIndex >= 0 {
		colour := boldGreen
		if nextMove != nil && pos == nextMove.From {
			colour = boldRed
			bugToken += directionArrow(nextMove.Dir)
			visibleLength++
		}
		fmt.Fprintf(out, "%s%s%s", colour, bugToken, reset)
		if star != "" {
			fmt.Fprintf(out, "%s%s%s", yellow, star, reset)
		}
	} else if star != "" {
		fmt.Fprintf(out, "%s%s%s", yellow, star, reset)
	} else {
		fmt.Fprint(out, token)
	}
	writeCellPadding(out, visibleLength)
}

func writeCellPadding(out io.Writer, tokenLength int) {
	padding := 6 - tokenLength
	if padding < 1 {
		padding = 1
	}
	fmt.Fprint(out, strings.Repeat(" ", padding))
}

func renderedBoardSize(p Puzzle) (int, int) {
	rows, cols := 1, 1
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			if p.Terrain[r][c] == Void {
				continue
			}
			if r+1 > rows {
				rows = r + 1
			}
			if c+1 > cols {
				cols = c + 1
			}
		}
	}
	return rows, cols
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
