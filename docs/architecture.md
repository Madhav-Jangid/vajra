# Architecture

Vajra is a small chess engine written in Go with no dependencies outside the standard library. This page explains how the pieces fit together and what each search and evaluation technique does.

```
                 +------------------+
  chess GUI ---> |  cmd/uci         |  UCI binary: `vajra`
                 +--------+---------+
                          |
  browser ---> cmd/server |   (WebSocket game server for the web UI)
                          v
                 +------------------+
                 |  internal/uci    |  protocol, time management, bench
                 +--------+---------+
                          v
                 +------------------+      +----------------+
                 |  internal/engine |----->| internal/book  |  opening book
                 |  search, eval    |      +----------------+
                 +--------+---------+
                          v
                 +------------------+
                 |  internal/chess  |  board, move generation, rules
                 +------------------+
```

## Packages

| Package | Responsibility |
| --- | --- |
| `internal/chess` | Board representation, FEN, move generation, make/unmake, check, checkmate, stalemate and draw rules, and standard algebraic notation. |
| `internal/engine` | Search, evaluation, transposition table, endgame bitbase, multi-threading. |
| `internal/uci` | The UCI protocol, command parsing, time management, the `bench` command. |
| `internal/book` | Opening book loading and move selection. |
| `internal/elo`, `internal/match` | Self-play matches and Elo estimation, used by `cmd/match`. |
| `internal/utils` | Board and move helpers for the terminal game. |
| `cmd/uci` | The engine executable. |
| `cmd/server` | HTTP and WebSocket server behind the web UI. |
| `cmd/chess` | A terminal game against the engine. |
| `cmd/match` | Self-play strength benchmark. |

Dependencies point one way: `uci` and `book` use `engine`, `engine` uses `chess`. Nothing in `chess` knows about search.

## Board and move generation (`internal/chess`)

The position is twelve bitboards, one per piece type and colour, with `a1 = 0` and `h8 = 63`. A `Position` also carries the side to move, castling rights, the en passant square and the move clocks.

- **Attacks.** Knight, king and pawn attacks come from precomputed tables. Sliding pieces use precomputed rays and a bit scan to find the first blocker. `IsSquareAttacked` combines these and never walks the board square by square.
- **Move generation.** Moves are generated per piece in pseudo-legal form into a reusable buffer, then filtered for legality.
- **Legality without make/unmake.** `AllLegalMoves` finds pinned pieces once. When the side to move is not in check, moves by pieces that are neither pinned nor the king need no test, except en passant, which is always verified. Everything else is checked by making the move and testing the king.
- **Make and unmake.** `MakeMove` returns an `Undo` record with the captured piece, castling rights, en passant square and clocks. `UnmakeMove` restores a position exactly. The colour of a castling rook or an en passant pawn is taken from the moved piece, so unmaking a move for the side that is not to move is also safe.

Correctness is held by perft: the standard positions (start, Kiwipete and the other Chess Programming Wiki positions) are tested to fixed depths against known node counts, with per-move divide tests for the start position and Kiwipete.

## Search (`internal/engine`)

The search is a negamax alpha-beta search driven by iterative deepening. Scores are in centipawns from the side to move's point of view. A mate is `100000` minus the distance to mate in plies.

| Technique | What it does here |
| --- | --- |
| Iterative deepening | Searches depth 1, 2, 3 and so on. Each completed depth feeds the next one its best move. A search can stop at any moment and still return the best move so far. |
| Aspiration windows | From depth 4, searches a narrow window around the previous score (50 centipawns) and widens it on a fail high or low. |
| Principal variation search | The first move gets a full window. Later moves are searched with a zero-width window and re-searched only if they beat alpha. |
| Transposition table | 64 MB by default. Stores score, bound type, depth and best move. Entries carry a generation number, so clearing the table between searches is instant. |
| Null-move pruning | At depth 3 and above, when the side to move has at least two non-pawn pieces and is not in check, passes the move with a reduction of 2 (3 from depth 6). |
| Reverse futility pruning | At depth 3 and below, returns early when the static evaluation is more than 120 centipawns per remaining depth above beta. |
| Late move reductions | The first four moves are searched at full depth. Later quiet moves that do not give check are searched 1 to 2 plies shallower and verified at full depth if they look promising. |
| Quiescence search | Resolves captures and promotions at the horizon, with stand-pat, delta pruning (200 centipawns) and static exchange evaluation to skip losing captures. |
| Draw detection | Repetition (using the game history sent by the GUI), the fifty-move rule and insufficient material. |

Reverse futility, null-move pruning, late move reductions and the quiescence pruning are controlled by one switch (`UseLMR`). The tests switch it off wherever they need an exact, unpruned score to compare against.

### Move ordering

Moves are tried in this order:

1. The best move from the transposition table.
2. Captures and promotions, by victim value minus attacker value (most valuable victim, least valuable attacker).
3. Killer moves: two quiet moves per ply that recently caused a beta cutoff.
4. Quiet moves by history score, which counts how often a move caused a cutoff (capped so history never outranks a strong capture).

### Multi-threaded search

Threads use Lazy SMP. Each helper thread runs its own iterative deepening on a private copy of the position, with its own killer and history tables and its own repetition path. The helpers share only the transposition table, which is protected by 1,024 striped locks. They start at different depths so they explore different parts of the tree. The main thread alone decides the move and prints the `info` lines. Node counts with more than one thread vary from run to run, which is expected with Lazy SMP. Single-threaded search is fully deterministic.

### Stopping a search

`RequestStop` sets an atomic flag that every node checks, so a search ends within microseconds and returns the best move found so far. The UCI `stop` command, a time limit and a node limit all end the search this way.

## Evaluation (`internal/engine`)

`Evaluate` returns a score in centipawns from White's point of view. The game phase (from the minor pieces, rooks and queens left) blends a middlegame and an endgame value for several terms.

| Term | Notes |
| --- | --- |
| Material | Pawn 100, knight 320, bishop 330, rook 500, queen 900. |
| Piece-square tables | One table per piece. The king has separate middlegame and endgame tables. |
| Pawn structure | Penalties for doubled and isolated pawns. |
| Passed pawns | Bonus by rank, larger in the endgame. |
| Pieces | Bishop pair, rooks on open and semi-open files. |
| Mobility | Squares a knight, bishop, rook or queen can reach, minus squares guarded by enemy pawns and minus a typical value for the piece. |
| King safety | Pawn shield in front of a castled king, a penalty for a king that has left its back ranks, and a penalty that grows with the number of enemy pieces attacking the squares around the king. Fades with the game phase. |
| Threats | A penalty for a piece standing on a square attacked by an enemy pawn, larger when the piece has nowhere safe to go. |
| Mop-up | In won endgames, rewards driving the losing king to a corner and bringing the winning king closer. |

### King and pawn against king

When the board holds only two kings and one pawn, `Evaluate` does not use the terms above. It looks the position up in a bitbase that says exactly whether the pawn side wins or the game is drawn. The bitbase is built the first time it is needed, by retrograde analysis over all 196,608 positions, using the same method as other open chess engines. The tests check the table against the real move generator.

## Opening book (`internal/book`)

A plain text file with one game line per row, each a sequence of moves in UCI notation. Positions reached by different move orders share an entry. Each book move is weighted by the number of lines that play it, and one is picked at random in proportion. The UCI layer plays a book move instantly, and only for searches that have a limit. See [UCI](uci.md#opening-book) for the options.

## UCI layer (`internal/uci`)

The command loop reads lines, parses them into commands and runs them. A search runs on its own goroutine so `stop`, `isready` and `quit` are answered while it works. A `go` with a limit finishes before the next `position` or `go` is handled. An infinite search waits for `stop` before it reports `bestmove`, as the protocol requires.

Time management gives each move about one thirtieth of the time left (or the time left divided by `movestogo`) plus three quarters of the increment, minus a configurable move overhead, and never more than the time left. Iterative deepening does not start a new depth once two thirds of the move's time is gone.
