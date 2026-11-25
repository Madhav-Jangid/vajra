# Contributing to Vajra

Thanks for your interest. Bug reports, tests, documentation and engine improvements are all welcome.

## Getting started

You need Go (the version in `go.mod`) and, for the web UI, Node.js 22 or newer.

```sh
git clone <repository-url>
cd vajra
go build ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

The full Go suite takes under a minute. The [development guide](docs/development.md) covers profiling, benchmarks and the release process.

## Before you open a pull request

- Run `gofmt` on the files you changed and `go vet ./cmd/... ./internal/...`.
- Run `go test ./cmd/... ./internal/...`. Every test must pass, none may be skipped.
- Add tests for new behaviour. Bug fixes should come with a test that fails without the fix.
- Keep a change focused. Unrelated clean-ups belong in their own pull request.

## Changes to search or evaluation

Engine changes need evidence, not intuition.

1. Run `go run ./cmd/uci bench` before and after. The node count is deterministic, so a changed number tells you the search behaviour changed. Put both numbers in the pull request.
2. Run the self-play benchmark and report the result: `go run ./cmd/match -depth-a 5 -depth-b 5` with the feature switched on for player A and off for player B (see `go run ./cmd/match -h`).
3. If a change makes the engine slower, say by how much and why it is worth it.

Move generation changes must keep every perft test passing.

## Reporting a bug

Include the Vajra version (`vajra version`), the GUI or command you used, and, if a move is wrong, the FEN of the position. For engine strength problems, a full game in PGN is the most useful thing you can attach.

## License

By contributing you agree that your work is released under the [MIT License](LICENSE).
