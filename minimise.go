package main

import (
	"fmt"
	"io"
	"strings"
)

type removedWall struct {
	Pos Pos
	Dir Direction
}

// MinimisePuzzle greedily removes platforms and walls while requiring the
// embedded solution to replay completely and remain shortest after every
// removal. It never writes the source puzzle; the resulting complete puzzle is
// written to out.
func MinimisePuzzle(p Puzzle, out io.Writer) error {
	replay, err := readSolution(p)
	if err != nil {
		return err
	}

	minimised := p
	var removedPlatforms []Pos
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			pos := Pos{r, c}
			if minimised.Terrain[r][c] == Void || bugAt(minimised.Bugs, pos) >= 0 {
				continue
			}
			candidate := minimised
			candidate.Terrain[r][c] = Void
			candidate.Stars[r][c] = 0
			candidate.Trampolines[r][c] = false
			if removalPreservesShortestSolution(candidate, len(replay.moves)) {
				minimised = candidate
				removedPlatforms = append(removedPlatforms, pos)
			}
		}
	}

	var removedWalls []removedWall
	for _, wall := range distinctWalls(minimised.Walls) {
		candidate := minimised
		knockDownWall(&candidate.Walls, wall.Pos, wall.Dir)
		if removalPreservesShortestSolution(candidate, len(replay.moves)) {
			minimised = candidate
			removedWalls = append(removedWalls, wall)
		}
	}

	// The stored text is parsed again against the modified starting state when
	// checking candidates. Format from the replayed moves to retain useful notes.
	solution, err := formatSolution(minimised, replay.moves)
	if err != nil {
		return fmt.Errorf("format retained solution: %w", err)
	}

	if err := writeMinimisedPuzzle(out, minimised, solution, removedPlatforms, removedWalls); err != nil {
		return err
	}
	return nil
}

func removalPreservesShortestSolution(p Puzzle, solutionLength int) bool {
	if _, err := readSolution(p); err != nil {
		return false
	}
	if solutionLength == 0 {
		return true
	}
	// We already know the retained solution works at solutionLength. Searching
	// only below that depth is sufficient to prove the removal did not create a
	// shortcut. Safe mode gives pushing-Beetle puzzles an exact BFS check.
	shorter := p
	shorter.MaxMoves = solutionLength - 1
	_, solved := SolveWithOptions(shorter, SolveOptions{Safe: true})
	return !solved
}

func distinctWalls(walls [BoardSize][BoardSize]uint8) []removedWall {
	var result []removedWall
	for r := 0; r < BoardSize; r++ {
		for c := 0; c < BoardSize; c++ {
			for _, dir := range directions {
				if walls[r][c]&(1<<dir) == 0 {
					continue
				}
				next := add(Pos{r, c}, deltas[dir])
				if inside(next) && dir != Right && dir != Down {
					continue
				}
				result = append(result, removedWall{Pos: Pos{r, c}, Dir: dir})
			}
		}
	}
	return result
}

func writeMinimisedPuzzle(out io.Writer, p Puzzle, solution string, platforms []Pos, walls []removedWall) error {
	var text strings.Builder
	fmt.Fprintf(&text, "# Minimise summary: removed %d platform(s) and %d wall(s).\n", len(platforms), len(walls))
	if len(platforms) > 0 {
		text.WriteString("# Removed platforms:")
		for _, pos := range platforms {
			fmt.Fprintf(&text, " (%d,%d)", pos.R+1, pos.C+1)
		}
		text.WriteByte('\n')
	}
	if len(walls) > 0 {
		text.WriteString("# Removed walls:")
		for _, wall := range walls {
			fmt.Fprintf(&text, " (%d,%d) %s", wall.Pos.R+1, wall.Pos.C+1, directionName(wall.Dir))
		}
		text.WriteByte('\n')
	}
	fmt.Fprintf(&text, "max-moves %d\nboard:\n", p.MaxMoves)
	writeCompactBoard(&text, p)
	writePuzzleDirectives(&text, p)
	text.WriteString("solution:\n")
	text.WriteString(solution)
	_, err := io.WriteString(out, text.String())
	return err
}

func writeCompactBoard(out *strings.Builder, p Puzzle) {
	lastRow := 0
	rows := make([]string, BoardSize)
	for r := 0; r < BoardSize; r++ {
		cells := make([]byte, BoardSize)
		lastColumn := -1
		for c := 0; c < BoardSize; c++ {
			cells[c] = puzzleCell(p, Pos{r, c})
			if cells[c] != '.' {
				lastColumn = c
			}
		}
		if lastColumn < 0 {
			rows[r] = "."
		} else {
			rows[r] = string(cells[:lastColumn+1])
			lastRow = r
		}
	}
	for r := 0; r <= lastRow; r++ {
		out.WriteString(rows[r])
		out.WriteByte('\n')
	}
}

func puzzleCell(p Puzzle, pos Pos) byte {
	terrain := p.Terrain[pos.R][pos.C]
	if terrain == Void {
		return '.'
	}
	if index := bugAt(p.Bugs, pos); index >= 0 {
		code := byte(p.Bugs[index].Kind)
		if terrain == Low {
			code += 'a' - 'A'
		}
		return code
	}
	if p.Trampolines[pos.R][pos.C] {
		if terrain == High {
			return 'X'
		}
		return 'x'
	}
	stars := p.Stars[pos.R][pos.C]
	if stars > 0 {
		if terrain == High {
			return []byte{'?', '!', '@', '#'}[stars]
		}
		return []byte{'?', '*', '2', '3'}[stars]
	}
	if terrain == High {
		return 'O'
	}
	return 'o'
}

func writePuzzleDirectives(out *strings.Builder, p Puzzle) {
	walls := distinctWalls(p.Walls)
	if len(walls) > 0 {
		out.WriteString("walls:\n")
		for _, wall := range walls {
			fmt.Fprintf(out, "wall %d %d %s\n", wall.Pos.R+1, wall.Pos.C+1, directionName(wall.Dir))
		}
	}
	var eggs []Bug
	for _, bug := range p.Bugs {
		if bug.Egg {
			eggs = append(eggs, bug)
		}
	}
	if len(eggs) > 0 {
		out.WriteString("eggs:\n")
		for _, bug := range eggs {
			fmt.Fprintf(out, "egg %d %d\n", bug.Pos.R+1, bug.Pos.C+1)
		}
	}
	constraints := false
	for _, bug := range p.Bugs {
		constraints = constraints || bug.OneMove || bug.NoLow || bug.NoHigh
	}
	if constraints {
		out.WriteString("constraints:\n")
		for _, bug := range p.Bugs {
			if bug.OneMove {
				fmt.Fprintf(out, "onemove %d %d\n", bug.Pos.R+1, bug.Pos.C+1)
			}
			if bug.NoLow {
				fmt.Fprintf(out, "nolow %d %d\n", bug.Pos.R+1, bug.Pos.C+1)
			}
			if bug.NoHigh {
				fmt.Fprintf(out, "nohigh %d %d\n", bug.Pos.R+1, bug.Pos.C+1)
			}
		}
	}
}

func directionName(dir Direction) string {
	return []string{"up", "right", "down", "left"}[dir]
}
