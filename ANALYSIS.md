# Puzzle analysis fields and scoring

The `analysis:` section is produced by replaying the stored solution and, for
version 2, searching the safe alternative moves encountered along it. All
component scores are percentages from 0 to 100 unless stated otherwise.
`clamp(x)` below means limiting a value to 0–100.

Counts of alternative moves are accumulated across every position on the
stored solution. They are not counts of globally distinct move strings.

## Headline quality fields

| Go field | Stored name | Meaning and score effect |
| --- | --- | --- |
| `Quality` | `quality` | Overall quality out of 100: 30% `Tightness`, 25% `Interactions`, 20% `AbilityUse`, 15% `Manageability`, and 10% `SetupBalance`. |
| `Tightness` | `tightness` | `VisitedPlatforms / Platforms` as a percentage. Supplies 30% of quality. |
| `Interactions` | `interactions` in `quality-components` | `InteractionMoves / Moves` as a percentage. Supplies 25% of quality. |
| `AbilityUse` | `ability-use` in `quality-components` | `MechanicsUsed / MechanicOpportunities` as a percentage, or 50 when there is no special-mechanic opportunity. Supplies 20% of quality. |
| `Manageability` | `manageability` in `quality-components` | Mean of move-count and object-count scores. Moves score 100 from 8–25, lose 10 points per move below 8 (starting from 20), and lose 3.3 points per move above 25 to a floor of 50. Objects score 100 from 3–8, score 60/80 for 1/2, and lose 15 points per object above 8. Supplies 15% of quality. |
| `SetupBalance` | `setup-balance` in `quality-components` | `clamp(100 - abs(setup ratio - 0.60) × 100)`, with a floor of 40. It rewards a setup-move proportion near 60% and supplies 10% of quality. |

## Headline difficulty fields

| Go field | Stored name | Meaning and score effect |
| --- | --- | --- |
| `Difficulty` | `difficulty` | Overall difficulty out of 10, rounded to one decimal and clamped to 1–10. Formula: `(0.25 Depth + 0.10 Choices + 0.10 Objects + 0.10 Setup + 0.10 Mechanics + 0.20 Scorpions + 0.35 DecisionLandscape) / 10`. The specialised scorpion and landscape terms are additive, so the theoretical total can exceed 10 before clamping. |
| `DifficultyDepth` | `depth` in `difficulty-components` | `clamp((Moves - 3) / 27 × 100)`. Supplies up to 2.5 difficulty points. |
| `DifficultyChoices` | `choices` in `difficulty-components` | `clamp((AverageLegal - 1) / 7 × 100)`. Supplies up to 1 point. The landscape analysis determines whether those choices are actually good. |
| `DifficultyObjects` | `objects` in `difficulty-components` | `clamp((Objects - 1) / 9 × 100)`. Supplies up to 1 point. |
| `DifficultySetup` | `setup` in `difficulty-components` | `SetupMoves / Moves` as a percentage. Supplies up to 1 point. |
| `DifficultyMechanics` | `mechanics` in `difficulty-components` | `clamp(MechanicsUsed / 5 × 100)`. Measures absolute mechanic variety and supplies up to 1 point. |
| `DifficultyScorpions` | `scorpion-load` and `scorpions` in `difficulty-components` | Cognitive load from autonomous scorpions. Supplies up to 2 additive difficulty points. Detailed below. |
| `DifficultyDecisionLandscape` | `decision-landscape` and `decision-landscape` in `difficulty-components` | Difficulty of choosing moves which preserve a shortest solution. Supplies up to 3.5 additive points. Detailed below. |

## Board and solution measurements

| Go field | Stored name | Meaning and score effect |
| --- | --- | --- |
| `Moves` | `moves` | Number of moves in the stored solution. Drives depth, manageability and per-move ratios. |
| `Objects` | `objects` | Number of starting bugs, scorpions and pucks. Drives object difficulty and manageability. |
| `Platforms` | `platforms` | Number of low or high platform squares, including stars and trampolines. Tightness denominator. |
| `VisitedPlatforms` | `visited-platforms` | Distinct platform squares occupied or traversed by the selected mover, or occupied by any object in a replayed state. Tightness numerator. |
| `OpeningLegal` | `opening-legal-moves` | Safe legal player moves in the initial state. Descriptive; also included in `AverageLegal`. |
| `MaximumLegal` | `maximum-legal-moves` | Largest safe legal move count at one solution position. Descriptive. |
| `AverageLegal` | `average-legal-moves` | Mean safe legal moves across solution positions. A safe move applies successfully, does not immediately lose, and satisfies constraints. Drives `DifficultyChoices`. |
| `ScoringMoves` | `scoring-moves` | Moves which consume at least one star. Multiple stars consumed in one move still count once. Descriptive. |
| `SetupMoves` | `setup-moves` | Moves which consume no star. Drives `SetupBalance` and `DifficultySetup`. |
| `InteractionMoves` | `interaction-moves` | Moves where another object's state changes, an egg hatches, or a wall is destroyed. Automatic scorpion changes count. Drives `Interactions`. |
| `EggsHatched` | `eggs-hatched` | Egg units removed from the egg state. Descriptive; a positive count also demonstrates `egg-hatching`. |
| `WallsDestroyed` | `walls-destroyed` | Wall boundaries destroyed. Descriptive; a positive count also demonstrates `wall-special`. |

## Mechanic fields

| Go field | Stored name | Meaning and score effect |
| --- | --- | --- |
| `Mechanics` | `mechanics` | Sorted names of mechanic categories demonstrated by the solution. Determines `MechanicsUsed`. |
| `MechanicsUsed` | numerator of `mechanics-used` | Available categories actually demonstrated. Drives `AbilityUse` and `DifficultyMechanics`. |
| `MechanicOpportunities` | denominator of `mechanics-used` | Categories offered by the starting puzzle. It is the `AbilityUse` denominator but does not directly increase difficulty. |

The categories are:

- `egg-hatching`: an egg hatches.
- `ant-merging`: a new multi-ant group forms.
- `beetle-pushing`: a Beetle move also changes another object's state.
- `caterpillar-nudging`: a Caterpillar changes when another object was selected.
- `puck-use`: a puck changes state or lies on the selected mover's path.
- `scorpion-action`: a scorpion's position, direction or existence changes.
- `wall-special`: a Gold Beetle destroys a wall or a Cockroach crosses one.
- `flight`: a flying bug meaningfully crosses a gap, object or wall.
- `distance-adjustment`: a Ladybird or Pink Ladybird finishes at a distance
  other than its normal two or three squares.
- `trampoline-use`: the selected mover's path contains a trampoline.

## Scorpion fields

| Go field | Stored name | Meaning and score effect |
| --- | --- | --- |
| `PeakActiveScorpions` | `peak-active-scorpions` | Greatest number of hatched, on-board scorpions before a player move. Its subscore reaches 100 at four. |
| `AverageActiveScorpions` | `average-active-scorpions` | Mean active count across solution positions. Its subscore reaches 100 at two. |
| `ScorpionSteps` | `scorpion-steps` | Scorpions whose position changes or which fall off following player moves. Denominator for direction-change load. |
| `ScorpionTurns` | `scorpion-turns` | Player moves followed by at least one scorpion position change. `ScorpionTurns / Moves` contributes 20% of scorpion load. |
| `MultiScorpionTurns` | `multi-scorpion-turns` | Player moves followed by more than one scorpion position change. `MultiScorpionTurns / Moves` contributes 20%. |
| `ScorpionDirectionChanges` | `scorpion-direction-changes` | Scorpions whose direction changes. `ScorpionDirectionChanges / ScorpionSteps` contributes 10%. |

`DifficultyScorpions` is the clamped sum of 30% peak load, 30% average load,
20% scorpion-turn coverage, 20% multi-scorpion-turn coverage, and 10%
direction-change load. The inputs deliberately total 110%, allowing an
especially busy scorpion puzzle to reach the component cap in different ways.

## Decision-landscape fields

| Go field | Stored name | Meaning and score effect |
| --- | --- | --- |
| `OptimalMoves` | `optimal-moves` | Safe alternatives which still permit completion in exactly the shortest remaining moves. A position can contribute several. |
| `DetourMoves` | `detour-moves` | Safe alternatives which remain solvable within the remaining `max-moves` allowance but need extra moves. They are known bad choices. |
| `DeadEndMoves` | `dead-end-moves` | Safe alternatives proved unable to solve within the remaining allowance. They are known bad choices. This does not claim no solution exists with unlimited moves. |
| `UnresolvedMoves` | `unresolved-moves` | Alternatives not classified before a resource limit. They are never treated as dead ends, but their ratio contributes a smaller uncertainty cost. |
| `BranchingSteps` | `branching-steps` | Solution positions with more than one safe legal move. Denominator for forced and misleading ratios. |
| `ForcedChoiceSteps` | `forced-choice-steps` | Branching positions with exactly one proved optimal move. Ratio to branching steps contributes 20% of landscape difficulty. |
| `MultipleOptimalSteps` | `multiple-optimal-steps` | Positions with several shortest-preserving moves. Descriptive; their effect appears through lower bad and forced ratios. |
| `MisleadingSteps` | `misleading-steps` | Branching positions with at least one proved detour or dead end. Ratio to branching steps contributes 15%. |
| `DetourPenalty` | `total-detour-penalty` | Sum of extra moves required by all proved detours. Descriptive. |
| `MaximumDetourPenalty` | `maximum-detour-penalty` | Largest extra cost of one detour. `clamp(maximum / 6 × 100)` contributes 10%. |
| `LandscapeStates` | `landscape-states-searched` | States expanded by alternative searches. A logarithmic percentage of the two-million-state limit contributes 5%. |
| `AverageOptimalMoves` | `average-optimal-moves` | `OptimalMoves / Moves`. Descriptive summary of shortest-preserving choices per solution position. |
| `AnalysisTimedOut` | `analysis-timed-out` | True when the 30-second limit is reached. It adds 10 landscape points; unclassified alternatives become unresolved. |

A **known bad move** is a detour or dead end. A **classified move** is optimal,
a detour or a dead end; unresolved moves are excluded from ratios requiring a
proved result. `DifficultyDecisionLandscape` is:

```text
30% × known-bad / classified
+ 20% × forced-choice steps / branching steps
+ 15% × misleading steps / branching steps
+ 10% × dead ends / known-bad
+ 10% × clamped maximum-detour severity
+ 30% × unresolved / all alternatives
+  5% × logarithmic search effort
+ 10 points if timed out
```

The result is clamped to 0–100. The inputs intentionally can total more than
100% when unresolved work exists: uncertainty or a resource limit must not
make a computationally difficult puzzle appear artificially easy.

The stored `version 2` line is the analysis format and formula version. It is
not a field in `PuzzleAnalysis`.
