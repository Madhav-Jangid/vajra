# UCI reference

Vajra speaks the [Universal Chess Interface](https://www.chessprogramming.org/UCI), so it works in any UCI chess GUI. This page lists what is supported.

## Using Vajra in a GUI

Download a release archive (or build `./cmd/uci`) and add the `vajra` executable as a UCI engine in your GUI. Most GUIs ask for the executable path and nothing else. The binary has no required files and no installer.

To check it by hand, run it and type commands:

```
$ ./vajra
uci
id name Vajra v1.0.0
id author ...
option name Hash type spin default 64 min 1 max 1024
option name Threads type spin default 1 min 1 max 64
option name OwnBook type check default false
option name BookFile type string default <empty>
option name Move Overhead type spin default 50 min 0 max 5000
uciok
position startpos moves e2e4 e7e5
go movetime 1000
info depth 12 score cp 34 nodes 412035 hashfull 18 nps 612000 time 673 pv g1f3 b8c6 f1b5 ...
bestmove g1f3
quit
```

## Command line

| Command | Result |
| --- | --- |
| `vajra` | Starts the UCI loop on standard input and output. |
| `vajra version` | Prints the version, for example `Vajra v1.0.0`. |
| `vajra bench [depth]` | Searches 12 fixed positions to a fixed depth on one thread and prints the node count and speed. The default depth is 10. |

The node count printed by `bench` is deterministic. If it changes, the search behaviour changed. See [benchmarks](benchmarks.md).

## Commands

| Command | Behaviour |
| --- | --- |
| `uci` | Prints the engine name, author and options, then `uciok`. |
| `isready` | Answers `readyok` at any time, including during a search. |
| `ucinewgame` | Clears the transposition table and resets to the starting position. |
| `position startpos \| fen <fen> [moves ...]` | Sets the position. A FEN may omit the move counters. If any move is illegal, the whole command is refused with an `info string` message and the previous position stays. |
| `go` | Starts a search. See below. |
| `stop` | Ends the search and prints `bestmove`. |
| `setoption name <name> [value <value>]` | Sets an option. Names are not case sensitive. |
| `d` | Prints the current position's FEN as an `info string`. |
| `quit` | Stops any search and exits. |

Unknown commands are ignored, as the protocol requires. A malformed known command prints an `info string` error and the engine keeps running.

### go

| Parameter | Behaviour |
| --- | --- |
| `depth <n>` | Searches to depth `n` (at most 64). |
| `movetime <ms>` | Searches for that long, minus the move overhead. |
| `nodes <n>` | Stops after about `n` nodes. |
| `wtime`, `btime`, `winc`, `binc`, `movestogo` | Clock based search. See [time management](#time-management). |
| `infinite` | Searches until `stop`. |
| `ponder` | Accepted, and treated like `infinite`. Pondering itself is not supported. |
| `searchmoves`, `mate` | Parsed and ignored. |

A bare `go` with no parameters searches until `stop`.

A search with a limit finishes before the next `position` or `go` command is handled, which makes scripted use deterministic. A search without a limit is ended by the next state-changing command. `go infinite` never prints `bestmove` until it receives `stop`.

### Search output

Each finished depth prints one line:

```
info depth 14 score cp 31 nodes 1304122 hashfull 42 nps 598000 time 2180 pv e2e4 e7e5 g1f3 b8c6
```

`score` is `cp <centipawns>` or `mate <moves>` (negative when Vajra is being mated), always from the side to move's point of view. `pv` is the principal variation read back from the transposition table. A search stops early once a forced mate has been searched one ply deeper than the mate itself.

If the side to move has no legal move, the reply is `bestmove 0000`.

## Options

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| `Hash` | spin 1 to 1024 | 64 | Transposition table size in megabytes. |
| `Threads` | spin 1 to 64 | 1 | Search threads (Lazy SMP). |
| `OwnBook` | check | false | Use the opening book when one is loaded. |
| `BookFile` | string | empty | Path to an opening book file. |
| `Move Overhead` | spin 0 to 5000 | 50 | Milliseconds kept back per move for GUI and network delay. |

Invalid values are refused with an `info string` message. Changing `Hash` or `Threads` while a search runs first lets a limited search finish, or stops an infinite one.

## Time management

For clock based searches:

```
spare = max(time left - move overhead, 1 ms)
limit = spare / movestogo + increment * 3/4     (movestogo defaults to 30)
limit = min(limit, spare)
```

Iterative deepening also refuses to start a new depth once two thirds of `limit` has been used, because a depth that cannot finish is wasted time.

## Opening book

A book is a text file. Each line is one game line in UCI move notation, from the starting position:

```
e2e4 e7e5 g1f3 b8c6 f1b5 a7a6
e2e4 c7c5 g1f3 d7d6 d2d4 c5d4
d2d4 d7d5 c2c4 e7e6
```

Blank lines are skipped. A line with an illegal move makes the whole load fail with the line number. Moves are weighted by how many lines play them.

```
setoption name BookFile value books/main.txt
setoption name OwnBook value true
```

With `OwnBook` on, a position found in the book is answered at once with `bestmove`, without searching. This applies only to searches with a limit; `go infinite` always searches.

## Not supported

- Pondering (`go ponder` runs as `go infinite`).
- Chess960.
- `searchmoves` and `go mate`.
- Opening books in Polyglot format.
- Tablebases other than the built-in king and pawn against king bitbase.
