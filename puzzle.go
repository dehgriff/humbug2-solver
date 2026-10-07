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
	Ladybird     BugType = 'L'
	Bee          BugType = 'B'
	Spider       BugType = 'S'
	Beetle       BugType = 'T'
	Snail        BugType = 'N'
	PinkLadybird BugType = 'P'
	Grasshopper  BugType = 'G'
	Butterfly    BugType = 'F'
	Cockroach    BugType = 'C'
	GoldBeetle   BugType = 'D'
	Puck         BugType = 'U'
	Fly          BugType = 'Y'
	Ant          BugType = 'A'
	Scorpion     BugType = 'Q'
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
	case PinkLadybird:
		return "pink ladybird"
	case Grasshopper:
		return "grasshopper"
	case Butterfly:
		return "butterfly"
	case Cockroach:
		return "cockroach"
	case GoldBeetle:
		return "gold beetle"
	case Puck:
		return "puck"
	case Fly:
		return "fly"
	case Ant:
		return "ant"
	case Scorpion:
		return "scorpion"
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
	Kind      BugType
	Pos       Pos
	Egg       bool
	Count     uint8
	Direction Direction
	ID        uint8
	Stunned   bool
}

type Puzzle struct {
	MaxMoves    int
	Terrain     [BoardSize][BoardSize]Terrain
	Stars       [BoardSize][BoardSize]uint8
	Trampolines [BoardSize][BoardSize]bool
	Walls       [BoardSize][BoardSize]uint8
	Bugs        []Bug
}

// ParsePuzzle reads a deliberately small, human-editable format.  Blank lines
// and lines beginning with # are ignored.  The directive "max-moves N" is
// followed by up to ten board rows, optionally introduced by "board:".
// Missing trailing cells and rows are treated as void squares.
// See README.md for the board characters.
func ParsePuzzle(r io.Reader) (Puzzle, error) {
	var p Puzzle
	hasMaxMoves := false
	s := bufio.NewScanner(r)
	lineNo := 0
	rows := make([]string, 0, BoardSize)
	type sourceLine struct {
		number int
		text   string
	}
	var directiveLines []sourceLine
	inDirectives := false
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
		section := strings.ToLower(strings.TrimSuffix(line, ":"))
		if section == "walls" || section == "eggs" {
			inDirectives = true
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 && (strings.EqualFold(fields[0], "wall") || strings.EqualFold(fields[0], "egg")) {
			inDirectives = true
		}
		if !inDirectives && len(rows) < BoardSize {
			rows = append(rows, line)
			continue
		}
		directiveLines = append(directiveLines, sourceLine{number: lineNo, text: line})
	}
	if err := s.Err(); err != nil {
		return p, err
	}
	if !hasMaxMoves {
		return p, fmt.Errorf("missing max-moves directive")
	}
	if len(rows) == 0 {
		return p, fmt.Errorf("board must contain at least one row")
	}
	for len(rows) < BoardSize {
		rows = append(rows, ".")
	}
	stars := 0
	for r, row := range rows {
		cells := strings.Fields(row)
		spaced := strings.ContainsAny(row, " \t") || row == "*2" || row == "*3" || row == "@2" || row == "@3"
		if !spaced {
			chars := []rune(row)
			cells = make([]string, len(chars))
			for i, ch := range chars {
				cells[i] = string(ch)
			}
		}
		if len(cells) > BoardSize {
			return p, fmt.Errorf("board row %d has more than %d cells", r+1, BoardSize)
		}
		for len(cells) < BoardSize {
			cells = append(cells, ".")
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
			case "x":
				p.Terrain[r][c], p.Trampolines[r][c] = Low, true
			case "X":
				p.Terrain[r][c], p.Trampolines[r][c] = High, true
			case "*", "*1", "*2", "*3":
				p.Terrain[r][c] = Low
				p.Stars[r][c] = starCount(cell)
				stars += int(p.Stars[r][c])
			case "@", "@1", "@2", "@3":
				p.Terrain[r][c] = High
				p.Stars[r][c] = starCount(cell)
				stars += int(p.Stars[r][c])
			case "l", "b", "s", "t", "n", "p", "g", "f", "c", "d", "u", "y", "a", "q":
				p.Terrain[r][c] = Low
				kind := bugRune(rune(cell[0]))
				p.Bugs = append(p.Bugs, newBug(kind, pos, p.Bugs))
			case "L", "B", "S", "T", "N", "P", "G", "F", "C", "D", "U", "Y", "A", "Q":
				p.Terrain[r][c] = High
				kind := bugRune(rune(cell[0]))
				p.Bugs = append(p.Bugs, newBug(kind, pos, p.Bugs))
			default:
				return p, fmt.Errorf("board row %d column %d: unknown cell %q", r+1, c+1, cell)
			}
		}
	}
	bugCount := 0
	for _, bug := range p.Bugs {
		if bug.Kind != Puck && bug.Kind != Scorpion {
			bugCount++
		}
	}
	if bugCount == 0 {
		return p, fmt.Errorf("puzzle must contain at least one bug")
	}
	if stars != bugCount {
		return p, fmt.Errorf("puzzle has %d bugs but %d stars", bugCount, stars)
	}
	for _, line := range directiveLines {
		fields := strings.Fields(line.text)
		if len(fields) == 3 && strings.EqualFold(fields[0], "egg") {
			r, errR := strconv.Atoi(fields[1])
			c, errC := strconv.Atoi(fields[2])
			if errR != nil || errC != nil || r < 1 || r > BoardSize || c < 1 || c > BoardSize {
				return p, fmt.Errorf("line %d: invalid egg position", line.number)
			}
			found := false
			for i := range p.Bugs {
				if p.Bugs[i].Pos == (Pos{r - 1, c - 1}) {
					if p.Bugs[i].Kind == Puck {
						return p, fmt.Errorf("line %d: a puck cannot be an egg", line.number)
					}
					if p.Stars[r-1][c-1] > 0 {
						return p, fmt.Errorf("line %d: an egg cannot start on a star", line.number)
					}
					if p.Bugs[i].Egg {
						return p, fmt.Errorf("line %d: bug is already marked as an egg", line.number)
					}
					p.Bugs[i].Egg = true
					found = true
					break
				}
			}
			if !found {
				return p, fmt.Errorf("line %d: egg position does not contain a bug", line.number)
			}
			continue
		}
		if len(fields) != 4 || !strings.EqualFold(fields[0], "wall") {
			return p, fmt.Errorf("line %d: expected a wall or egg directive", line.number)
		}
		r, errR := strconv.Atoi(fields[1])
		c, errC := strconv.Atoi(fields[2])
		dir, ok := parseDirection(fields[3])
		if errR != nil || errC != nil || !ok || r < 1 || r > BoardSize || c < 1 || c > BoardSize {
			return p, fmt.Errorf("line %d: invalid wall position or direction", line.number)
		}
		from := Pos{r - 1, c - 1}
		to := Pos{from.R + wallDR[dir], from.C + wallDC[dir]}
		if to.R < 0 || to.R >= BoardSize || to.C < 0 || to.C >= BoardSize {
			if p.Terrain[from.R][from.C] == Void {
				return p, fmt.Errorf("line %d: an edge wall must border a platform", line.number)
			}
			p.Walls[from.R][from.C] |= 1 << dir
			continue
		}
		if p.Terrain[from.R][from.C] == Void && p.Terrain[to.R][to.C] == Void {
			return p, fmt.Errorf("line %d: at least one side of a wall must be a platform", line.number)
		}
		p.Walls[from.R][from.C] |= 1 << dir
		p.Walls[to.R][to.C] |= 1 << oppositeWallDirection(dir)
	}
	return p, nil
}

var wallDR = [4]int{-1, 0, 1, 0}
var wallDC = [4]int{0, 1, 0, -1}

func parseDirection(value string) (int, bool) {
	switch strings.ToLower(value) {
	case "up":
		return 0, true
	case "right":
		return 1, true
	case "down":
		return 2, true
	case "left":
		return 3, true
	default:
		return 0, false
	}
}

func oppositeWallDirection(dir int) int { return (dir + 2) % 4 }

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
	case 'n', 'N':
		return Snail
	case 'p', 'P':
		return PinkLadybird
	case 'g', 'G':
		return Grasshopper
	case 'f', 'F':
		return Butterfly
	case 'c', 'C':
		return Cockroach
	case 'd', 'D':
		return GoldBeetle
	case 'u', 'U':
		return Puck
	case 'y', 'Y':
		return Fly
	case 'q', 'Q':
		return Scorpion
	default:
		return Ant
	}
}

func newBug(kind BugType, pos Pos, existing []Bug) Bug {
	bug := Bug{Kind: kind, Pos: pos, Count: initialCount(kind)}
	if kind == Scorpion {
		bug.Direction = Right
		for _, other := range existing {
			if other.Kind == Scorpion {
				bug.ID++
			}
		}
	}
	return bug
}

func initialCount(kind BugType) uint8 {
	if kind == Ant {
		return 1
	}
	return 0
}
