# Benchmarks

All numbers on this page were measured on one machine, and every one can be reproduced with the commands shown.

| | |
| --- | --- |
| CPU | Intel Core i7-12700 (8 performance cores, 4 efficiency cores, 20 threads) |
| Memory | 16 GB |
| OS | Windows 11 |
| Go | 1.27.3 |
| Date | 2026-11-25 |
| Build | pre-release development build (commit `c55ef2b` plus the release preparation) |
| Hash | 64 MB (256 MB for the thread scaling test) |

Speed varies with the machine, the operating system and what else is running. Compare the numbers between your own runs, and treat the ones here as a reference point.

## Move generation

Perft counts every legal move sequence to a fixed depth. It checks that move generation is correct (the node counts must match published values exactly) and measures its raw speed.

| Position | Depth | Nodes | Matches published count | Nodes per second |
| --- | --- | --- | --- | --- |
| Start position | 3 | 8,902 | yes | 6.1 million |
| Start position | 4 | 197,281 | yes | |
| Kiwipete | 3 | 97,862 | yes | 7.8 million |
| Position 3 (rook endgame) | 4 | 43,238 | yes | 6.2 million |
| Position 4 (promotions and castling) | 3 | 9,467 | yes | |
| Position 5 | 3 | 62,379 | yes | |

The speed figures come from the perft test harness, which generates moves piece by piece, without the pin shortcut the search uses.

```sh
go test ./internal/chess -run Perft -v
go test ./internal/chess -run '^$' -bench Perft
```

## Search speed

`vajra bench` searches twelve fixed positions (opening, middlegame, tactical and endgame) to depth 10 on one thread, from an empty transposition table for each.

```sh
vajra bench
```

| Position | Nodes | Time (ms) | Nodes per second |
| --- | --- | --- | --- |
| 1 Start position | 669,264 | 1,072 | 623,762 |
| 2 Kiwipete | 757,463 | 1,465 | 516,920 |
| 3 Rook endgame | 258,043 | 296 | 871,410 |
| 4 Promotions and castling | 194,854 | 465 | 418,352 |
| 5 Promotion tactics | 293,676 | 755 | 388,506 |
| 6 Open middlegame | 338,463 | 1,013 | 333,810 |
| 7 Sicilian | 775,024 | 2,042 | 379,518 |
| 8 Fianchettoed middlegame | 1,086,957 | 2,798 | 388,419 |
| 9 Tactical middlegame | 890,529 | 1,941 | 458,583 |
| 10 Italian game | 455,918 | 1,202 | 379,004 |
| 11 Rook and pawn endgame | 400,105 | 743 | 537,822 |
| 12 Pawn endgame | 66,018 | 83 | 789,208 |
| **Total** | **6,186,314** | **13,882** | **445,604** |

The total node count, 6,186,314, is the bench signature for this version. It is identical on every run and every operating system, because a single-threaded search is fully deterministic. A change to the search or evaluation changes it, and a refactor or speed-up must not. See [development](development.md#the-bench-signature).

### Single thread speed over time

During development a profile showed most of the search time spent in a map lookup inside the board code. Replacing it, together with attack tables and allocation-free move generation, changed the same search of the same positions as follows:

| | Before | After |
| --- | --- | --- |
| Search speed | about 60,000 nodes per second | about 300,000 nodes per second |
| Memory allocations per search | about 1,000,000 | about 47,000 |
| Perft speed | about 2.0 million nodes per second | 6 to 8 million |

The searched node counts were identical before and after that step, which shows the speed-up changed nothing about how the engine plays.

## Multi-threaded search

Each row searches three positions (start, Kiwipete and an Italian game middlegame) for 5 seconds each and reports the reported speed (nodes from all threads) and the depth reached.

| Threads | Nodes per second | Speed-up | Depth reached (start / Kiwipete / middlegame) |
| --- | --- | --- | --- |
| 1 | 541,882 | 1.0x | 11 / 11 / 12 |
| 2 | 1,041,120 | 1.9x | 12 / 12 / 13 |
| 4 | 2,055,053 | 3.8x | 12 / 11 / 11 |
| 8 | 3,794,688 | 7.0x | 13 / 11 / 14 |

Raw speed scales almost linearly up to the 8 performance cores. The depth gained is smaller and varies from run to run. This is expected with Lazy SMP, where threads share only the transposition table and each one repeats some work. The effect of extra threads on playing strength has not been measured.

## Strength

There is no rating for Vajra from a rating list yet, and this page does not claim one. What has been measured is how much each extra ply of search is worth, in games of Vajra against itself.

`cmd/match` played each pair of depths over 12 openings, with both colours (24 games per match):

| Match | Result (win, draw, loss) | Score | Elo difference | Likelihood of superiority |
| --- | --- | --- | --- | --- |
| depth 4 against depth 3 | +17 =6 -1 | 83.3% | +280 ± 162 | 100.0% |
| depth 5 against depth 4 | +12 =9 -3 | 68.8% | +137 ± 119 | 99.0% |
| depth 6 against depth 5 | +17 =6 -1 | 83.3% | +280 ± 162 | 100.0% |
| depth 7 against depth 6 | +16 =3 -5 | 72.9% | +172 ± 161 | 99.2% |

The ± figure is a 95% margin. With only 24 games it is wide, so the four results are consistent with each other and with a gain of roughly 150 to 250 Elo per ply at these depths. Games were limited to 200 plies and drawn at that point.

These are differences between two copies of the same engine. They are not comparable with the ratings of rating lists, which come from games against other engines, and self-play usually overstates real gains. To measure Vajra against other engines, play a gauntlet in a GUI tool such as Cutechess and report the result.

```sh
go run ./cmd/match -depth-a 5 -depth-b 4
go run ./cmd/match -depth-a 5 -depth-b 5 -killers-a=false
```

## Tests and size

| | |
| --- | --- |
| Tests | 305 tests and 687 subtests, none failing and none skipped |
| Run time | about 20 seconds for the whole suite, about 15 seconds of it in the engine package |
| Go source | about 5,600 lines, plus about 10,300 lines of tests |
| Binary | 2.0 MB (stripped, static, no dependencies) |

```sh
go test ./cmd/... ./internal/...
```
