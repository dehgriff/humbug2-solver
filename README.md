# Humbug2 solver

This command-line program reads a human-editable 10x10 puzzle and searches for
a solution with the fewest moves. It normally uses iterative-deepening A*, with
an optional safe breadth-first fallback for pushing-Beetle puzzles. Repeated
board states are pruned, falling moves are discarded, and no branch is searched
beyond `max-moves`.

## Puzzle format

```text
# Comment lines use # followed by a space; blank lines are ignored.
max-moves 8
board:
..........
..........
..lo*.....
..........
..........
..........
..........
..........
..........
..........
```

Each board row contains at most ten cells:

| Character | Meaning |
|---|---|
| `.` | no platform |
| `o` / `O` | low / high platform |
| `x` / `X` | trampoline on a low / high platform |
| `*` or `1` | one star on a low platform |
| `2` / `3` | two / three stars on a low platform |
| `!` | one star on a high platform |
| `@` / `#` | two / three stars on a high platform |
| `l` / `L` | ladybird on a low / high platform |
| `b` / `B` | bee on a low / high platform |
| `s` / `S` | spider on a low / high platform |
| `t` / `T` | beetle on a low / high platform |
| `n` / `N` | snail on a low / high platform |
| `p` / `P` | Pink Ladybird on a low / high platform |
| `g` / `G` | grasshopper on a low / high platform |
| `f` / `F` | butterfly on a low / high platform |
| `c` / `C` | cockroach on a low / high platform |
| `d` / `D` | Gold Beetle on a low / high platform |
| `u` / `U` | puck on a low / high platform |
| `y` / `Y` | fly on a low / high platform |
| `a` / `A` | ant on a low / high platform |
| `q` / `Q` | scorpion on a low / high platform |
| `k` / `K` | caterpillar on a low / high platform |

The file must contain the same number of goal bugs and stars; pucks and
scorpions are not goal bugs. Coordinates printed in solutions are one-based
`(row,column)` coordinates.

Butterflies fly three squares before landing. Cockroaches walk two squares and
obey normal non-flying height and obstacle rules, but ignore walls during their
own normal movement. Their direct displacement while being pushed is blocked
by walls; like every bug, they can cross walls after entering bounce mode.
Gold Beetles also walk two squares and ignore walls, but permanently remove
each wall crossed during the walking portion of their own movement. After
entering landing/bounce mode they jump over subsequent walls without destroying
them. A bug or puck on the immediately adjacent square prevents all
displacement, but a wall on that same boundary is destroyed and makes the
action a valid go. A void or board edge does not stop the move: the Gold Beetle
falls after attempting its full two-square movement, even if it crossed a wall
and an intermediate star. A direct push cannot carry one through a wall, and a
pushed Gold Beetle never destroys walls. The `d`/`D`
notation uses the final letter of “gold”.
Flies (`y`/`Y`, using the final letter of “fly”) travel until the first platform
in the chosen direction and enter landing mode there. They ignore intervening
voids, walls, and heights, but fall if no platform exists before the board edge.

Ants move like spiders, but ants meeting during normal movement can combine
into groups of at most three. A combined group moves as one unit. If a moving
group would exceed the destination limit, only enough ants join to fill it and
the remainder stop together on the preceding square. Landing or bouncing ants
do not combine; they bounce normally. Stars consume individual ants from a
group, leaving any ants for which no star was available. Puzzle files place
only individual ants (`a`/`A`); combined groups arise during play.

Caterpillars (`k`/`K`) move one square like Snails. When a bug encounters a
hatched Caterpillar as an obstacle during ordinary ground movement, the moving
bug stops and nudges the Caterpillar one square in the same direction. This is
a valid move even if the initiating bug never changes square. Landing or
bouncing on a Caterpillar does not nudge it. If one Caterpillar nudges another,
the effect travels along the contiguous line and only the last movable
Caterpillar moves. A nudged Caterpillar obeys normal Snail rules for walls,
platform height, trampolines, falling, and stars. Caterpillar eggs do not react
to nudges.

Every board cell is exactly one character and board rows must not contain
spaces. For example, this row contains three Snails followed by a low platform
with three stars:

```text
nnn3
```

Each bug that stops on `2`, `3`, `@`, or `#` consumes one star. The other stars
remain available to subsequent bugs. Because `#` is a board character, write
comments as `# ` followed by the comment text; a syntactically valid board row
beginning with `#` is treated as board data.

Trailing void cells may be omitted from every board row, and trailing entirely
void rows may also be omitted. These two boards are therefore equivalent:

```text
*ob.......
o.o*......
..........
..........
..........
..........
..........
..........
..........
..........
```

```text
*ob
o.o*
```

Section headers such as `walls:` or `eggs:` mark the end of a shortened board.

A trampoline cannot hold a bug. When a walking bug enters a trampoline square
(including while crossing it), a flying bug lands on it, or a bug is pushed
onto it, the bug immediately enters bounce mode and tries the next square in
the same direction. Trampolines can be low (`x`) or high (`X`). Normal height
and wall rules apply to the resulting bounce.

### Walls

Walls are listed after the ten board rows using one-based coordinates and a
direction from that square:

```text
walls:
wall 3 4 right
wall 6 7 up
wall 1 1 up
```

`wall 3 4 right` places a wall between `(3,4)` and `(3,5)`; spelling the same
wall as `wall 3 5 left` has identical behavior. At least one of the two squares
must contain a platform. An outward-facing wall is also allowed on a perimeter
platform, such as `wall 1 1 up`. A wall blocks normal non-flying movement.
Flying bugs ignore walls, and every bug can cross a wall once it is already in
landing/bounce mode. Low and high platform levels do not affect walls.

### Eggs

Any bug placed on the board can start as an egg. List eggs after the board with
one-based coordinates that identify an existing bug:

```text
eggs:
egg 3 4
egg 7 2
```

An egg cannot initiate a move, but it can be pushed and otherwise occupies its
platform like a bug. When another bug lands on it in landing mode, the landing
bug bounces onward and the egg hatches into its normal bug type. The hatched
bug can move on subsequent turns. Egg status travels with a pushed bug. An
unhatched egg does not consume a star when pushed onto it. If another bug then
lands on the egg, it hatches and immediately consumes one star and disappears.
A puzzle cannot declare an egg on a star in its initial state.

### Pucks

Use `u` for a puck on a low platform and `U` for one on a high platform. Pucks
occupy platforms but cannot initiate moves. They can be pushed, serve as bounce
obstacles, and bounce from trampolines like bugs. They never consume stars and
do not count toward winning the puzzle. If a pushed puck falls into a square
without a platform or leaves the board, it disappears and play continues.

### Scorpions

Use `q` for a scorpion on a low platform and `Q` for one on a high platform.
Scorpions cannot be selected for a player move, do not consume stars, and do
not need to remain on the board for the puzzle to be solved. They can be pushed,
can start as eggs, and block and bounce other bugs normally. A scorpion that
falls disappears without losing the puzzle.

Scorpions initially point right. When a puzzle is loaded, a hatched scorpion
adjacent to a same-level target bug turns to face it, provided no wall separates
them. Eggs are not targets. If several targets are adjacent, direction priority
is straight, left, right, then reverse.

After every player move has otherwise completed, the program selects the
hatched scorpion with the shortest distance to its preferred target. Each
candidate first looks for a completely clear route and considers routes through
other scorpions only when no clear route exists. A scorpion with a clear route
always acts before one whose route passes through another scorpion, even when
the blocked route is shorter. Distance is compared within that route class.
After the selected scorpion acts, route classes, target choices, and distances
are recalculated on the resulting board for the scorpions still waiting to act.
For equal distances, a scorpion pointing along its intended move wins over one
requiring a 90-degree turn, which wins over one requiring a 180-degree turn.
If still tied, a scorpion whose selected target is directly ahead along a
straight, unobstructed path wins. After that, the scorpion with the higher row
number wins; on the same row, the lower column number wins. Stable numbering is
retained only as a final fallback, and each scorpion acts at most once during
the automatic phase. An
unstunned scorpion adjacent to a hatched, non-scorpion bug on the same platform
level kills it and loses that search branch, unless a wall separates them. Eggs
and other scorpions are neither targets nor victims. A scorpion landed on
during the current player move is stunned: it neither kills nor moves during
its automatic turn for that move.

When at least two hatched scorpions are active, `-play` and `-play-soln` print
the automatic turn order after each move, for example `Scorpion movement order:
2, 1, 3`.

Board displays render ordinary bugs in bold green, scorpions in bold blue, the
next solution bug in bold red, and stars in yellow. Generated solution lines
and solver stdout include an inline note when a scorpion or puck falls off the
board or a bug merges with a star, such as `7 t r # scorpion 2 fell off board`
or `8 n u # snail merged with star at (3,4)`. Comments in the puzzle file are
ignored during replay.

Otherwise, a scorpion moves one step along a shortest same-level path to the
nearest reachable hatched bug. Walls, occupied non-scorpion squares, and
scorpion eggs block paths. It prefers any wholly clear route, but if none exists
it may plan a future route through other hatched scorpions; it still cannot make
its immediate step onto an occupied square. Between equally close targets, one
ahead of the scorpion wins over one to either side, and a target to either side
wins over one behind it. A trampoline is treated as an
ordinary empty platform during route selection—the scorpion does not anticipate
its bounce. If it actually steps onto that trampoline, normal landing/bounce
behavior applies and may carry it over blockers or hatch eggs. If the target
bearing is also equal, competing first steps are preferred in the order
straight ahead, a 90-degree left turn, a 90-degree right turn, then a 180-degree
turn. Its direction becomes the direction of its latest automatic or pushed
movement. If its movement brings it adjacent to its selected target on the same
level without a separating wall, it then turns to face that target. With no
reachable target it stays still.

## Run

Requires Go 1.22 or newer.

```sh
go run . puzzles/example.puzzle
```

The solver uses an exact iterative-deepening A* search when its remaining-bug
lower bound is safe, and prints each lower bound and the number of states
searched. Eggs on stars and Caterpillars contribute zero to the lower bound
because they may score as side effects of another bug's move. By default, the
same search is used for pushing Beetles under the simplifying assumption that
a Beetle move clears at most one goal bug.

Use `-safe` when that assumption may not hold:

```sh
go run . -safe puzzles/example.puzzle
```

## Puzzle regression tests

Use `-test` to parse every `.puzzle` file in a directory and replay every
embedded solution from beginning to end:

```sh
go run . -test puzzles
```

The report marks files as `PASS`, `MISSING`, `INVALID`, or `FAIL`. A missing
solution is reported but does not make the command fail; an invalid puzzle or a
solution that no longer completes the puzzle produces a non-zero exit status.
If the directory is omitted, `puzzles` is used.

Solving a puzzle writes the discovered shortest solution back into that puzzle
file. The following resumable shell loop solves every file that still lacks a
solution section:

```sh
for puzzle in puzzles/*.puzzle; do
    if ! grep -qi '^solution:' "$puzzle"; then
        go run . "$puzzle"
    fi
done
```

Stopping the loop is safe: completed files retain their solution sections, and
running the loop again skips them.

Safe mode falls back to breadth-first search for puzzles containing a pushing
Beetle and therefore retains the unconditional shortest-solution guarantee.
For puzzles without one, `-safe` still uses exact iterative-deepening A*.

Or build a reusable binary:

```sh
go build -o humbug2 .
./humbug2 puzzles/example.puzzle
```

When a solution is found, the solver appends or replaces a `solution:` section
at the end of the puzzle file. It preserves everything before an existing
section. The section contains one consecutively numbered move per line:

```text
solution:
1 s r
2 t d
3 p 5 2 r
```

The short `TYPE DIRECTION` form is used when exactly one object of that bug
type exists at that point in the solution. If the type is ambiguous, its
one-based row and column are required. An unsuccessful search does not alter
an existing solution section. Because `solution:` consumes the remainder of
the file, it must be the final section.

### Search constraints

An optional `constraints:` section can restrict the solver's search. The
`onemove ROW COLUMN` directive identifies a player-movable bug by its starting
coordinate:

```text
constraints:
onemove 1 3
nolow 4 6
nohigh 3 4
```

The coordinate is one-based. Other bugs may move first, and the marked bug may
be pushed in the meantime. Once the marked bug is selected for its own first
move, that move must remove it by reaching a star; otherwise that search branch
is discarded. Multiple bugs may each have an `onemove` constraint.

The `nolow ROW COLUMN` and `nohigh ROW COLUMN` directives discard a branch if
the marked bug finishes one of its own moves on a low or high platform,
respectively. A bug that reaches a star and disappears has not finished on the
platform, so that move is allowed. Constraint types may be combined on the same
bug, and their coordinates always refer to the starting board.

### Interactive play

Use `-play` to inspect and play a puzzle manually:

```sh
go run . -play puzzles/example.puzzle
```

Manual moves use the same notation as solution files. If a bug type is unique,
coordinates may be omitted; otherwise they are required. Both short and full
directions are accepted:

```text
s r
p 5 2 r
p 5 2 right
```

Coordinate-only input such as `5 2 r` is not accepted.

After every move the program prints a coordinate-labelled board, remaining
walls, move number, and whether the puzzle is in progress, solved, or lost.
The display extends from `(1,1)` to the puzzle's lowest and rightmost platform,
omitting unused trailing rows and columns while retaining leading dots for an
offset board.
Scorpions are displayed with their stable number and direction, such as `q1>`;
eggs use an `e` suffix and grouped ants use their count. Enter `undo` to restore
the previous state, including after the puzzle is over. Bugs and other objects
are shown in bold green, while star markers are yellow. Enter `help` or `quit`
for the other interactive commands.

### Solution playback

Use `-play-soln` to validate and step through the puzzle's embedded solution:

```sh
go run . -play-soln puzzles/d12.puzzle
```

Use the right arrow key to advance one move, the left arrow key to go back, and
`q` to quit; no Enter key is needed. When input is redirected rather than read
from a terminal, the line commands `right`/`r`, `left`/`l`, and `quit` remain
available. The complete board is displayed at every step. The next bug to move
is bold red and followed by an arrow indicating its direction; other bugs are
bold green, and star markers (`*`, `2`, `!`, `@`, `#`, and so on) are yellow. Playback
rejects malformed moves, ambiguous shorthand, illegal moves, and files that do
not finish the puzzle.

The solver implements the rules in `game-rules.md`. Where those rules are
implicit, it uses these interpretations:

- walking bugs fall as soon as they enter a square without a platform;
- a bee ignores terrain while flying, then bounces over occupied landing
  platforms; a missing landing platform causes it to fall;
- a bouncing bug can land on either a low or a high platform;
- a blocked move that travels zero squares is not a move;
- every bug that comes to rest on a star disappears, including a pushed bug;
- pushed bugs must obey the low-to-high restriction, while a push into a void
  is a losing branch;
- a beetle on a high platform can push only bugs on contiguous high platforms;
  the final pushed bug enters landing mode if it drops to a low platform;
- after a non-flying bug drops from high to low, it is in landing mode and
  bounces forward over occupied platforms, as specified by the rules.
