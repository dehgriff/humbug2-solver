# Humbug2 solver

This command-line program reads a human-editable 10x10 puzzle and uses a
breadth-first search to find a solution with the fewest moves. Repeated board
states are searched only once, falling moves are discarded, and no branch is
searched beyond `max-moves`.

## Puzzle format

```text
# Lines beginning with # and blank lines are ignored.
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
| `*` / `@` | one star on a low / high platform |
| `*2`, `*3` | two or three stars on a low platform (spaced rows only) |
| `@2`, `@3` | two or three stars on a high platform (spaced rows only) |
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

The file must contain the same number of bugs and stars. Coordinates printed
in solutions are one-based `(row,column)` coordinates.

Butterflies fly three squares before landing. Cockroaches walk two squares and
obey normal non-flying height and obstacle rules, but ignore walls during their
own normal movement. Their direct displacement while being pushed is blocked
by walls; like every bug, they can cross walls after entering bounce mode.
Gold Beetles also walk two squares and ignore walls, but permanently remove
each wall they cross during their own movement. A direct push cannot carry one
through a wall, and a pushed Gold Beetle never destroys walls—even if a bounce
carries it across one. The `d`/`D` notation uses the final letter of “gold”.
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

Rows containing only single-character cells can use the compact form shown
above. To place multiple stars on one platform, write all ten cells separated
by spaces. For example, this row has three stars on its fourth low platform:

```text
n n n *3 . . . . . .
```

Each bug that stops on `*2`, `*3`, `@2`, or `@3` consumes one star. The other
stars remain available to subsequent bugs.

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

## Run

Requires Go 1.22 or newer.

```sh
go run . puzzles/example.puzzle
```

Or build a reusable binary:

```sh
go build -o humbug2 .
./humbug2 puzzles/example.puzzle
```

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
