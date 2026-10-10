package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func writePuzzleSolution(path string, p Puzzle, moves []Move) error {
	solution, err := formatSolution(p, moves)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	prefix := data
	if start := solutionSectionOffset(data); start >= 0 {
		prefix = data[:start]
	}
	var text strings.Builder
	text.Write(prefix)
	if text.Len() > 0 && !strings.HasSuffix(text.String(), "\n") {
		text.WriteByte('\n')
	}
	text.WriteString("solution:\n")
	text.WriteString(solution)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".humbug2-solution-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(info.Mode().Perm()); err != nil {
		temp.Close()
		return err
	}
	if _, err := io.WriteString(temp, text.String()); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}

func solutionSectionOffset(data []byte) int {
	start := 0
	for start < len(data) {
		end := start
		for end < len(data) && data[end] != '\n' {
			end++
		}
		line := strings.TrimSpace(string(data[start:end]))
		if strings.EqualFold(line, "solution:") {
			return start
		}
		start = end + 1
	}
	return -1
}

func formatSolution(p Puzzle, moves []Move) (string, error) {
	state := initialState(p)
	var output strings.Builder
	for number, move := range moves {
		index := bugAt(state.Bugs, move.From)
		if index < 0 || state.Bugs[index].Kind != move.Kind {
			return "", fmt.Errorf("move %d does not match the replayed state", number+1)
		}
		code := bugCode(move.Kind)
		if countBugType(state.Bugs, move.Kind) == 1 {
			fmt.Fprintf(&output, "%d %s %s\n", number+1, code, directionCode(move.Dir))
		} else {
			fmt.Fprintf(&output, "%d %s %d %d %s\n", number+1, code, move.From.R+1, move.From.C+1, directionCode(move.Dir))
		}
		outcome := applyMoveDetailed(p, state, index, move.Dir)
		if outcome.status != moveApplied {
			return "", fmt.Errorf("move %d cannot be replayed: %s", number+1, outcome.reason)
		}
		state = outcome.state
	}
	return output.String(), nil
}

func initialState(p Puzzle) State {
	state := State{Bugs: append([]Bug(nil), p.Bugs...), Stars: p.Stars, Walls: p.Walls}
	canonicalize(&state)
	return state
}

func bugCode(kind BugType) string {
	code := byte(kind)
	if code >= 'A' && code <= 'Z' {
		code += 'a' - 'A'
	}
	return string(code)
}

func parseBugCode(value string) (BugType, bool) {
	if len(value) != 1 {
		return 0, false
	}
	switch strings.ToLower(value) {
	case "l":
		return Ladybird, true
	case "b":
		return Bee, true
	case "s":
		return Spider, true
	case "t":
		return Beetle, true
	case "n":
		return Snail, true
	case "p":
		return PinkLadybird, true
	case "g":
		return Grasshopper, true
	case "f":
		return Butterfly, true
	case "c":
		return Cockroach, true
	case "d":
		return GoldBeetle, true
	case "u":
		return Puck, true
	case "y":
		return Fly, true
	case "a":
		return Ant, true
	case "q":
		return Scorpion, true
	case "k":
		return Caterpillar, true
	default:
		return 0, false
	}
}

func directionCode(dir Direction) string {
	return []string{"u", "r", "d", "l"}[dir]
}

func countBugType(bugs []Bug, kind BugType) int {
	count := 0
	for _, bug := range bugs {
		if bug.Kind == kind {
			count++
		}
	}
	return count
}

func parseStateMove(line string, state State) (int, Direction, error) {
	fields := strings.Fields(line)
	if len(fields) != 2 && len(fields) != 4 {
		return 0, Up, fmt.Errorf("expected TYPE DIRECTION or TYPE ROW COLUMN DIRECTION")
	}
	kind, ok := parseBugCode(fields[0])
	if !ok {
		return 0, Up, fmt.Errorf("unknown bug type %q", fields[0])
	}
	directionField := fields[len(fields)-1]
	dir, ok := parsePlayDirection(directionField)
	if !ok {
		return 0, Up, fmt.Errorf("direction must be u, r, d, l or its full name")
	}
	if len(fields) == 2 {
		index := -1
		for i, bug := range state.Bugs {
			if bug.Kind != kind {
				continue
			}
			if index >= 0 {
				return 0, Up, fmt.Errorf("more than one %s exists; coordinates are required", kind)
			}
			index = i
		}
		if index < 0 {
			return 0, Up, fmt.Errorf("no %s exists", kind)
		}
		return index, dir, nil
	}

	row, errRow := strconv.Atoi(fields[1])
	col, errCol := strconv.Atoi(fields[2])
	if errRow != nil || errCol != nil || row < 1 || row > BoardSize || col < 1 || col > BoardSize {
		return 0, Up, fmt.Errorf("row and column must be between 1 and %d", BoardSize)
	}
	index := bugAt(state.Bugs, Pos{row - 1, col - 1})
	if index < 0 {
		return 0, Up, fmt.Errorf("square (%d,%d) does not contain an object", row, col)
	}
	if state.Bugs[index].Kind != kind {
		return 0, Up, fmt.Errorf("square (%d,%d) contains a %s, not a %s", row, col, state.Bugs[index].Kind, kind)
	}
	return index, dir, nil
}

type solutionReplay struct {
	lines  []string
	moves  []Move
	states []State
}

func readSolution(p Puzzle) (solutionReplay, error) {
	if !p.HasSolution {
		return solutionReplay{}, fmt.Errorf("puzzle has no solution section")
	}
	replay := solutionReplay{states: []State{initialState(p)}}
	state := replay.states[0]
	for lineNumber, line := range p.Solution {
		index, dir, err := parseStateMove(line, state)
		if err != nil {
			return solutionReplay{}, fmt.Errorf("solution move %d: %w", lineNumber+1, err)
		}
		outcome := applyMoveDetailed(p, state, index, dir)
		if outcome.status != moveApplied {
			return solutionReplay{}, fmt.Errorf("solution move %d: %s", lineNumber+1, outcome.reason)
		}
		replay.lines = append(replay.lines, line)
		replay.moves = append(replay.moves, outcome.move)
		replay.states = append(replay.states, outcome.state)
		state = outcome.state
	}
	if remainingBugCount(state) != 0 {
		return solutionReplay{}, fmt.Errorf("solution ends after %d move(s) but the puzzle is not solved", len(replay.moves))
	}
	return replay, nil
}

func PlaySolution(p Puzzle, in io.Reader, out io.Writer) (err error) {
	replay, err := readSolution(p)
	if err != nil {
		return err
	}
	if file, ok := in.(*os.File); ok {
		restore, raw, rawErr := makeTerminalRaw(file)
		if rawErr != nil {
			return fmt.Errorf("enable terminal navigation: %w", rawErr)
		}
		if raw {
			defer func() {
				if restoreErr := restore(); err == nil && restoreErr != nil {
					err = fmt.Errorf("restore terminal: %w", restoreErr)
				}
			}()
			return playSolutionRaw(p, replay, in, out)
		}
	}
	return playSolutionLines(p, replay, in, out)
}

func playSolutionLines(p Puzzle, replay solutionReplay, in io.Reader, out io.Writer) error {
	step := 0
	fmt.Fprintln(out, "Solution navigation: right/r advances, left/l goes back, quit exits.")
	printSolutionState(out, p, replay, step)
	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprint(out, "> ")
		if !scanner.Scan() {
			fmt.Fprintln(out)
			return scanner.Err()
		}
		command := strings.ToLower(strings.TrimSpace(scanner.Text()))
		switch command {
		case "right", "r", ">", "\x1b[c":
			if step == len(replay.moves) {
				fmt.Fprintln(out, "Already at the end of the solution.")
				continue
			}
			fmt.Fprintf(out, "Applied: %s\n", replay.lines[step])
			step++
			printSolutionState(out, p, replay, step)
		case "left", "l", "<", "\x1b[d":
			if step == 0 {
				fmt.Fprintln(out, "Already at the start of the solution.")
				continue
			}
			step--
			fmt.Fprintf(out, "Rewound: %s\n", replay.lines[step])
			printSolutionState(out, p, replay, step)
		case "quit", "q", "exit":
			return nil
		case "help", "h", "?":
			fmt.Fprintln(out, "Commands: right/r, left/l, quit")
		default:
			fmt.Fprintln(out, "Unknown command. Use right, left, or quit.")
		}
	}
}

type navigationKey uint8

const (
	navigationNone navigationKey = iota
	navigationRight
	navigationLeft
	navigationQuit
)

func playSolutionRaw(p Puzzle, replay solutionReplay, in io.Reader, out io.Writer) error {
	step := 0
	fmt.Fprintln(out, "Solution navigation: → advances, ← goes back, q quits.")
	printSolutionState(out, p, replay, step)
	for {
		key, err := readNavigationKey(in)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		switch key {
		case navigationRight:
			if step == len(replay.moves) {
				fmt.Fprintln(out, "Already at the end of the solution.")
				continue
			}
			fmt.Fprintf(out, "Applied: %s\n", replay.lines[step])
			step++
			printSolutionState(out, p, replay, step)
		case navigationLeft:
			if step == 0 {
				fmt.Fprintln(out, "Already at the start of the solution.")
				continue
			}
			step--
			fmt.Fprintf(out, "Rewound: %s\n", replay.lines[step])
			printSolutionState(out, p, replay, step)
		case navigationQuit:
			return nil
		}
	}
}

func readNavigationKey(in io.Reader) (navigationKey, error) {
	var value [1]byte
	if _, err := io.ReadFull(in, value[:]); err != nil {
		return navigationNone, err
	}
	switch value[0] {
	case 'q', 'Q', 3, 4:
		return navigationQuit, nil
	case 0x1b:
		// Arrow keys normally arrive as CSI sequences (ESC [ C/D), while
		// some terminals use application mode sequences (ESC O C/D).
		if _, err := io.ReadFull(in, value[:]); err != nil {
			return navigationNone, err
		}
		if value[0] != '[' && value[0] != 'O' {
			return navigationNone, nil
		}
		for sequenceLength := 0; sequenceLength < 16; sequenceLength++ {
			if _, err := io.ReadFull(in, value[:]); err != nil {
				return navigationNone, err
			}
			if value[0] < 0x40 || value[0] > 0x7e {
				continue
			}
			switch value[0] {
			case 'C':
				return navigationRight, nil
			case 'D':
				return navigationLeft, nil
			default:
				return navigationNone, nil
			}
		}
	}
	return navigationNone, nil
}

func printSolutionState(out io.Writer, p Puzzle, replay solutionReplay, step int) {
	heading := fmt.Sprintf("Solution step %d/%d. Puzzle solved.", step, len(replay.moves))
	if step < len(replay.moves) {
		heading = fmt.Sprintf("Solution step %d/%d. Next move: %s", step, len(replay.moves), replay.moves[step])
	}
	printBoardState(out, p, replay.states[step], heading)
}
