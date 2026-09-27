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

Each board row is exactly ten characters:

| Character | Meaning |
|---|---|
| `.` | no platform |
| `o` / `O` | low / high platform |
| `*` / `@` | star on a low / high platform |
| `l` / `L` | ladybird on a low / high platform |
| `b` / `B` | bee on a low / high platform |
| `s` / `S` | spider on a low / high platform |
| `t` / `T` | beetle on a low / high platform |

The file must contain the same number of bugs and stars. Coordinates printed
in solutions are one-based `(row,column)` coordinates.

## Run

Requires Go 1.22 or newer.

```sh
go run . example.puzzle
```

Or build a reusable binary:

```sh
go build -o humbug2 .
./humbug2 example.puzzle
```

The solver implements the rules in `game-rules.md`. Where those rules are
implicit, it uses these interpretations:

- walking bugs fall as soon as they enter a square without a platform;
- a bee ignores terrain while flying, then bounces over occupied landing
  platforms; a missing landing platform causes it to fall;
- a blocked move that travels zero squares is not a move;
- every bug that comes to rest on a star disappears, including a pushed bug;
- pushed bugs must obey the low-to-high restriction, while a push into a void
  is a losing branch;
- after a non-flying bug drops from high to low, it is in landing mode and
  bounces forward over occupied platforms, as specified by the rules.
