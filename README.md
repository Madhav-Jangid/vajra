# Vajra

Vajra is a UCI chess engine written in Go. It is a single static binary of about 2 MB with no dependencies, it plays in any UCI chess GUI, and the whole engine is small enough to read in an afternoon.

It also comes with a browser game, so you can play against it without installing a GUI.

- **Fast.** About 450,000 nodes per second on one thread and 3.8 million with 8 threads, measured on an Intel Core i7-12700. Move generation runs perft at 6 to 8 million nodes per second.
- **Correct.** Perft counts match the published values for the standard positions, and the test suite has 325 tests and 687 subtests.
- **Reproducible.** `vajra bench` prints a deterministic node count, so any change to how the engine searches shows up as a different number.
- **Readable.** Standard library only. The packages are small and each one does a single job.

## Features

**Search**
- Negamax alpha-beta with iterative deepening and aspiration windows
- Principal variation search, late move reductions, null-move pruning, reverse futility pruning
- Quiescence search with static exchange evaluation
- Transposition table, killer moves, history heuristic
- Multi-threaded search (Lazy SMP)
- Draw detection by repetition, the fifty-move rule and insufficient material

**Evaluation**
- Material and piece-square tables, with a tapered middlegame and endgame blend
- Pawn structure, passed pawns, bishop pair, rooks on open files
- Piece mobility, king safety, threats against pieces
- Exact king and pawn against king endgames from a built-in bitbase
- Mop-up evaluation for won endgames

**Engine**
- Browser game: play White, Black or a random side, with drag and drop, premoves, a promotion picker, an evaluation bar and a move list
- UCI protocol with `Hash`, `Threads`, `Move Overhead`, `OwnBook` and `BookFile` options
- Opening book from a plain text file of game lines
- Principal variation and hash usage in every `info` line
- `vajra bench` and `vajra version`

## Quick start

### Download

Take the archive for your system from the [releases page](../../releases). It holds the `vajra` engine, the `vajra-server` game server, the license and the documentation. Verify it against `checksums.txt` if you like.

### Build from source

You need Go (the version in `go.mod`).

```sh
git clone <repository-url>
cd vajra
go build -o vajra ./cmd/uci
```

### Play it in a chess GUI

Add the `vajra` executable to your GUI as a UCI engine. No other files are needed.

You can also talk to it directly:

```
$ ./vajra
uci
...
uciok
position startpos moves e2e4 e7e5
go movetime 1000
info depth 12 score cp 34 nodes 412035 hashfull 18 nps 612000 time 673 pv g1f3 b8c6 f1b5 ...
bestmove g1f3
```

### Play it in the browser

```sh
go run ./cmd/server
```

and in a second terminal

```sh
cd web
npm install
npm run dev
```

Then open the address `npm run dev` prints. See [web UI and server](docs/web.md).

## Benchmarks

Measured on an Intel Core i7-12700 with 16 GB of memory. Full tables, the method and the commands to reproduce them are in [docs/benchmarks.md](docs/benchmarks.md).

| | Result |
| --- | --- |
| Bench, depth 10, 12 positions, 1 thread | 6,186,314 nodes in 13.9 s, 445,604 nodes per second |
| Perft (start position, Kiwipete, position 3) | 6.1, 7.8 and 6.2 million nodes per second |
| Search speed with 1, 2, 4 and 8 threads | 0.54, 1.04, 2.06 and 3.79 million nodes per second |
| Self-play, each extra ply of depth | +137 to +280 Elo over four matches of 24 games |
| Binary size | 2.0 MB |

### How strong is it?

Not yet measured against other engines, so there is no rating to quote. Self-play shows that each extra ply of search is worth roughly 150 to 250 Elo at the depths tested, with wide error margins because each match was only 24 games. Self-play gains overstate gains against other engines. If you play Vajra in a gauntlet against other engines, please share the result.

You can measure the effect of any change yourself:

```sh
go run ./cmd/match -depth-a 5 -depth-b 4
```

### Known limits

- Vajra has been tested through its protocol test suite and scripted sessions. It has not been checked in specific GUIs yet. Reports are welcome.
- No pondering, no Chess960, and `searchmoves` and `go mate` are ignored.
- The opening book uses a simple text format, not Polyglot.
- The only endgame tablebase is the built-in king and pawn against king bitbase.
- In the browser UI premoves cannot castle, and the server runs one shared game.

## Documentation

| | |
| --- | --- |
| [Architecture](docs/architecture.md) | How the engine works: board, search, evaluation, threads |
| [UCI reference](docs/uci.md) | Commands, options, time management, opening books |
| [Benchmarks](docs/benchmarks.md) | Measurements, method and how to reproduce them |
| [Web UI and server](docs/web.md) | The browser game and its WebSocket protocol |
| [Development guide](docs/development.md) | Building, testing, profiling, CI and releasing |

## Project layout

```text
cmd/
  uci/         the engine executable (vajra)
  server/      HTTP and WebSocket server for the web UI
  chess/       terminal game
  match/       self-play strength benchmark
internal/
  chess/       board, move generation, rules
  engine/      search, evaluation, transposition table, bitbase, threads
  uci/         UCI protocol, time management, bench
  book/        opening book
  elo/         Elo estimation
  match/       self-play match runner
  utils/       helpers for the terminal game
web/           Next.js browser UI
docs/          documentation
.github/       CI and release workflows
```

## Testing

```sh
go test ./cmd/... ./internal/...
```

Perft counts for the standard positions, rules edge cases, search properties (for example that move ordering never changes a score), forced mates, the king and pawn bitbase checked against the move generator, and engine self-play are all covered. CI also runs the race detector, a short strength match and the web app's lint and build. A release only goes out after the full suite, the race detector and a strength match pass on Linux and Windows. See the [development guide](docs/development.md).

## Contributing

Bug reports and pull requests are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) first. For engine changes, please include the bench node count and a self-play result.

## Acknowledgements

The [Chess Programming Wiki](https://www.chessprogramming.org) documents nearly every technique used here, and its perft positions are the backbone of the move generator tests. The king and pawn bitbase uses the retrograde analysis method published by Stockfish.

## License

[MIT](LICENSE)
