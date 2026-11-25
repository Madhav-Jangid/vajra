# Development guide

## Build and test

```sh
go build ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

Use `./cmd/... ./internal/...` instead of `./...`. The `web/` directory contains `node_modules`, which can hold Go files that are not part of this project.

The suite has two layers.

- **Correctness.** Perft counts for the standard positions, rules and move generation edge cases (pins, en passant, castling through check, underpromotion), FEN round trips, make/unmake restoring every position, and a table of forced mates and tactics that the search must find.
- **Search and evaluation properties.** Switching move ordering, history, killer moves and principal variation search on and off must not change a search score. Evaluation must be symmetric between colours. The king and pawn bitbase is checked against the real move generator. Engine self-play checks that every move it chooses is legal. A few guards fail if the node count of a fixed search grows by a large factor.

To run one package or one test:

```sh
go test ./internal/engine
go test ./internal/engine -run TestMatesAreFound -v
```

The race detector needs a C compiler, so it is run in CI on Linux:

```sh
go test -race ./internal/chess ./internal/engine ./internal/uci ./internal/book
```

## Benchmarks and profiling

```sh
go run ./cmd/uci bench
go test ./internal/chess -run '^$' -bench Perft
go test ./internal/engine -run '^$' -bench 'BenchmarkSearch(FixedDepth|TimeBudget)$' -benchmem
```

To profile the search:

```sh
go test ./internal/engine -run '^$' -bench 'BenchmarkSearchFixedDepth/middlegame$' \
  -cpuprofile cpu.out -memprofile mem.out -o engine.test
go tool pprof -top engine.test cpu.out
go tool pprof -sample_index=alloc_space -top engine.test mem.out
```

Profile before optimising. The largest speed-up in the project's history came from a profile that showed most of the time spent in one map lookup.

### The bench signature

`vajra bench` searches 12 fixed positions to a fixed depth on one thread. The node count is the same on every run, on every operating system, so it works like a fingerprint of the search.

- If you change something that should not affect play, such as a refactor or a speed-up, the node count must stay the same.
- If you change the search or evaluation, the node count will change. Report the new number in the pull request.

## Measuring strength

`cmd/match` plays two configurations of the engine against each other from twelve openings, with colours swapped, and prints the score, an Elo difference with a 95% margin and the likelihood of superiority.

```sh
go run ./cmd/match -depth-a 5 -depth-b 4
go run ./cmd/match -depth-a 5 -depth-b 5 -lmr-a=false
```

Use `-h` for all options. `-min-score 0.6` makes the command exit with status 3 when player A scores below 60%, which is how CI uses it. A small number of games gives a wide margin, so judge a change by the margin as well as the score.

## Continuous integration

`.github/workflows/ci.yml` runs on every push to `main` and every pull request:

| Job | What it checks |
| --- | --- |
| `test` | `go vet`, build and the full test suite on Linux and Windows. |
| `race` | The race detector on the chess, engine, UCI and book packages. |
| `strength` | A short self-play match that a deeper search must win. |
| `bench` | Runs `vajra bench` and shows the result in the job summary. |
| `web` | `npm run lint`, `npm test` and `npm run build` for the web app. |

## Releasing

A release is a git tag that looks like `v1.2.3` (or `v1.2.3-rc.1` for a pre-release):

```sh
git tag v1.2.3
git push origin v1.2.3
```

`.github/workflows/release.yml` then does the following, in order. Each stage must pass before the next one starts.

1. **Verify.** Checks the tag is a version, then runs `go vet`, the build and the entire Go test suite on Linux and Windows. This includes every engine, perft and UCI test. It also runs the race detector and a 24-game strength match that the deeper side must win.
2. **Build.** Cross-compiles `vajra` (the UCI engine) and `vajra-server` for Linux (amd64 and arm64), macOS (amd64 and arm64) and Windows (amd64). The version is stamped into the binary. The build then unpacks the Linux binary and checks that `vajra version` prints the right version, that the engine answers `uci` with `uciok`, and that `vajra bench 6` runs.
3. **Publish.** Creates the GitHub release with the archives, a `checksums.txt` file and generated release notes. A tag with a hyphen is marked as a pre-release.

Archives are named `vajra-<version>-<os>-<arch>` and contain the binaries, the README, the license and the docs. Verify a download with:

```sh
sha256sum -c checksums.txt --ignore-missing
```

## Conventions

- Keep the standard library as the only dependency of the Go code.
- Format with `gofmt`.
- Do not use `GenerateMoves` or `LegalMoves` in the search. Use `allMoves`, which is the fast path.
- Any new search feature that changes scores needs a test that shows it keeps forced mates, and a self-play result.
