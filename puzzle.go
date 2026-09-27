package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const BoardSize = 10

type BugType byte

const (
	Ladybird BugType = 'L'
	Bee      BugType = 'B'
	Spider   BugType = 'S'
	Beetle   BugType = 'T'
	Snail    BugType = 'N'
)

func (b BugType) String() string {
	switch b {
	case Ladybird:
		return "ladybird"
	case Bee:
		return "bee"
	case Spider:
		return "spider"
	case Beetle:
		return "beetle"
	case Snail:
		return "snail"
	default:
		return "unknown"
	}
}

type Terrain byte

const (
	Void Terrain = iota
	Low
	High
)

type Pos struct{ R, C int }

type Bug struct {
	Kind BugType
	Pos  Pos
}

type Puzzle struct {
	MaxMoves int
	Terrain  [BoardSize][BoardSize]Terrain
	Stars    [BoardSize][BoardSize]uint8
	Bugs     []Bug
}

// ParsePuzzle reads a deliberately small, human-editable format.  Blank lines
// and lines beginning with # are ignored.  The directive "max-moves N" is
// followed by exactly ten board rows, optionally introduced by "board:".
// See README.md for the board characters.
func ParsePuzzle(r io.Reader) (Puzzle, error) {
	var p Puzzle
	hasMaxMoves := false
	s := bufio.NewScanner(r)
	lineNo := 0
	rows := make([]string, 0, BoardSize)
	for s.Scan() {
		lineNo++
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if len(rows) == 0 && strings.HasPrefix(strings.ToLower(line), "max-moves") {
			fields := strings.Fields(line)
			if len(fields) != 2 {
				return p, fmt.Errorf("line %d: expected 'max-moves N'", lineNo)
			}
			n, err := strconv.Atoi(fields[1])
			if err != nil || n < 0 {
				return p, fmt.Errorf("line %d: max-moves must be a non-negative integer", lineNo)
			}
			p.MaxMoves = n
			hasMaxMoves = true
			continue
		}
		if strings.EqualFold(strings.TrimSuffix(line, ":"), "board") && len(rows) == 0 {
			continue
		}
		rows = append(rows, line)
	}
	if err := s.Err(); err != nil {
		return p, err
	}
	if !hasMaxMoves {
		return p, fmt.Errorf("missing max-moves directive")
	}
	if len(rows) != BoardSize {
		return p, fmt.Errorf("board must have %d rows, got %d", BoardSize, len(rows))
	}
	stars := 0
	for r, row := range rows {
		cells := strings.Fields(row)
		if len(cells) == 1 {
			chars := []rune(row)
			cells = make([]string, len(chars))
			for i, ch := range chars {
				cells[i] = string(ch)
			}
		}
		if len(cells) != BoardSize {
			return p, fmt.Errorf("board row %d must have %d cells, got %d", r+1, BoardSize, len(cells))
		}
		for c, cell := range cells {
			pos := Pos{r, c}
			switch cell {
			case ".":
				p.Terrain[r][c] = Void
			case "o":
				p.Terrain[r][c] = Low
			case "O":
				p.Terrain[r][c] = High
			case "*", "*1", "*2", "*3":
				p.Terrain[r][c] = Low
				p.Stars[r][c] = starCount(cell)
				stars += int(p.Stars[r][c])
			case "@", "@1", "@2", "@3":
				p.Terrain[r][c] = High
				p.Stars[r][c] = starCount(cell)
				stars += int(p.Stars[r][c])
			case "l", "b", "s", "t", "n":
				p.Terrain[r][c] = Low
				p.Bugs = append(p.Bugs, Bug{Kind: bugRune(rune(cell[0])), Pos: pos})
			case "L", "B", "S", "T", "N":
				p.Terrain[r][c] = High
				p.Bugs = append(p.Bugs, Bug{Kind: bugRune(rune(cell[0])), Pos: pos})
			default:
				return p, fmt.Errorf("board row %d column %d: unknown cell %q", r+1, c+1, cell)
			}
		}
	}
	if len(p.Bugs) == 0 {
		return p, fmt.Errorf("puzzle must contain at least one bug")
	}
	if stars != len(p.Bugs) {
		return p, fmt.Errorf("puzzle has %d bugs but %d stars", len(p.Bugs), stars)
	}
	return p, nil
}

func starCount(cell string) uint8 {
	if len(cell) == 1 {
		return 1
	}
	return cell[1] - '0'
}

func bugRune(ch rune) BugType {
	switch ch {
	case 'l', 'L':
		return Ladybird
	case 'b', 'B':
		return Bee
	case 's', 'S':
		return Spider
	case 't', 'T':
		return Beetle
	default:
		return Snail
	}
}
