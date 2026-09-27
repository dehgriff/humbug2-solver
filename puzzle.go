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
	Stars    [BoardSize][BoardSize]bool
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
		chars := []rune(row)
		if len(chars) != BoardSize {
			return p, fmt.Errorf("board row %d must have %d characters, got %d", r+1, BoardSize, len(chars))
		}
		for c, ch := range chars {
			pos := Pos{r, c}
			switch ch {
			case '.':
				p.Terrain[r][c] = Void
			case 'o':
				p.Terrain[r][c] = Low
			case 'O':
				p.Terrain[r][c] = High
			case '*':
				p.Terrain[r][c], p.Stars[r][c] = Low, true
				stars++
			case '@':
				p.Terrain[r][c], p.Stars[r][c] = High, true
				stars++
			case 'l', 'b', 's', 't':
				p.Terrain[r][c] = Low
				p.Bugs = append(p.Bugs, Bug{Kind: bugRune(ch), Pos: pos})
			case 'L', 'B', 'S', 'T':
				p.Terrain[r][c] = High
				p.Bugs = append(p.Bugs, Bug{Kind: bugRune(ch), Pos: pos})
			default:
				return p, fmt.Errorf("board row %d column %d: unknown character %q", r+1, c+1, ch)
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

func bugRune(ch rune) BugType {
	switch ch {
	case 'l', 'L':
		return Ladybird
	case 'b', 'B':
		return Bee
	case 's', 'S':
		return Spider
	default:
		return Beetle
	}
}
