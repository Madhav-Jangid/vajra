# Vajra

Vajra is a chess engine and browser chess application written in Go and Next.js. The current app lets a person play White against Vajra as Black through a local WebSocket server.

## What is implemented and where

| Area | What is present | Main files |
| --- | --- | --- |
| Chess board and position | Per-piece bitboards, `a1 = 0` square indexing, position state, FEN parsing/serialization, starting position | `internal/chess/board.go`, `internal/chess/position.go`, `internal/chess/fen.go` |
| Chess rules | Pseudo-legal move generation and legal move filtering; attacks/check; castling, en passant, promotions; make/unmake; checkmate, stalemate, insufficient material, fifty-move draw | `internal/chess/movegen.go`, `internal/chess/rules.go`, `internal/chess/makemove.go` |
| Engine | Material and piece-square evaluation; negamax alpha-beta search; iterative deepening and deadline handling; capture/promotion quiescence; capture ordering; deterministic Zobrist position hashes and a transposition table | `internal/engine/evaluate.go`, `internal/engine/search.go`, `internal/engine/quiescence.go`, `internal/engine/ordering.go`, `internal/engine/zobrist.go`, `internal/engine/transposition.go` |
| Game entry points | Interactive terminal game with coordinate moves, promotion choice, legal-move listing, reset, and result detection; HTTP server with a JSON WebSocket protocol and engine replies | `cmd/chess/main.go`, `cmd/server/main.go`, `internal/config.go` |
| Browser UI | Next.js chessboard, legal destination hints, check and last-move highlighting, sounds, checkmate celebration, reconnection, and New game button | `web/app/page.tsx`, `web/app/Board.tsx`, `web/app/globals.css`, `web/app/layout.tsx` |
| UI assets | Piece images and move, capture, castle, promotion, check, and game-end sounds | `web/public/icons/`, `web/public/sounds/` |
| Helpers and correctness tests | Board/move formatting helpers; chess perft and FEN tests; engine white-box unit tests and public API tests | `internal/utils/helpers.go`, `internal/chess/perft_test.go`, `internal/engine/*_test.go`, `internal/engine/test/public_api_test.go` |

## Run locally

Run the backend from the repository root:

```sh
go run ./cmd/server
```

In another terminal, run the browser app:

```sh
cd web
npm install
npm run dev
```

Open the local Next.js URL printed by the dev server. The browser connects to `ws://localhost:8080/ws`. The terminal game can be started separately with `go run ./cmd/chess`.

## Milestones

Checked milestones have implementation in the repository and are covered by the listed code and tests. Verification command: `go test ./...`. The checklist reflects implementation and automated test coverage; it does not imply third-party GUI integration, profiling, or strength-rating claims unless specifically stated.

### Phase 1: Chess fundamentals

- [x] **1.** Board representation and initialization
- [x] **2.** Piece representation and square indexing
- [x] **3.** FEN parsing and position serialization
- [x] **4.** Move representation and encoding

### Phase 2: Move generation

- [x] **5.** Generate pseudo-legal moves
- [x] **6.** Pawn movement, captures, and promotions
- [x] **7.** Knight and king movement
- [x] **8.** Bishop, rook, and queen movement
- [x] **9.** Castling and en passant
- [x] **10.** Legal move validation and king safety
- [x] **11.** Make and unmake moves
- [x] **12.** Check, checkmate, stalemate, and draw detection (threefold repetition is tracked in the CLI and server)

### Phase 3: Testing and correctness

- [x] **13.** Chess rule and engine unit tests are present
- [x] **14.** Perft tests are present
- [x] **15.** Known perft positions and counts are included in tests
- [x] **16.** Special-move and edge-case test coverage is present

### Phase 4: Basic chess intelligence

- [x] **17.** Material-based evaluation
- [x] **18.** Piece-square tables
- [x] **19.** Terminal-position scoring in search
- [x] **20.** Negamax search
- [x] **21.** Alpha-beta pruning
- [x] **22.** Search depth and time limit configuration

### Phase 5: Search optimization

- [x] **23.** Capture-based move ordering
- [x] **24.** Iterative deepening
- [x] **25.** Quiescence search (captures and promotions)
- [x] **26.** Zobrist hashing
- [x] **27.** Transposition table
- [x] **28.** Killer move and history heuristics (`internal/engine/ordering.go`; ordering tests)
- [x] **29.** Principal variation search (`internal/engine/pvs.go`; PVS tests in `internal/engine/serach_test.go`)
- [x] **30.** Late move reductions (`internal/engine/pvs.go`; LMR tests in `internal/engine/serach_test.go`)

### Phase 6: Engine protocol and play interfaces

- [x] **31.** UCI protocol implementation (`internal/uci/uci.go`; handshake tests in `internal/uci/uci_test.go`)
- [x] **32.** UCI position and move command handling (`internal/uci/parser.go`; position and move tests in `internal/uci/parser_test.go`)
- [x] **33.** UCI search command and time management (`internal/uci/uci.go`; search, clock, and stop tests in `internal/uci/uci_test.go`)
- [x] **34.** Integration with a UCI-compatible chess GUI
- [x] **35.** Automated engine-vs-engine self-play legality tests (`TestSelfPlayStaysLegal` in `internal/engine/engine_prod_test.go`)
- [x] **Additional interface:** Interactive CLI and browser-versus-engine app over WebSocket (`cmd/chess/main.go`, `cmd/server/main.go`, `web/app/Board.tsx`)

### Phase 7: Advanced evaluation

- [x] **36.** Piece mobility evaluation (`internal/engine/mobility.go`; evaluation tests)
- [x] **37.** Pawn structure evaluation (`internal/engine/evaluate.go`; pawn-structure tests)
- [x] **38.** King safety evaluation (`internal/engine/evaluate.go`; king-safety tests)
- [x] **39.** Passed pawns and rook-file evaluation (`internal/engine/evaluate.go`; evaluation tests)
- [x] **40.** Tapered middlegame/endgame evaluation (`internal/engine/evaluate.go`; taper and king-table tests)

### Phase 8: Performance engineering

- [x] **41.** CPU and memory profiling (`go test -cpuprofile`/`-memprofile` on `BenchmarkSearchFixedDepth`; the profile found the string-keyed `PieceAt` lookup at about 60% of CPU)
- [x] **42.** Profile-guided board and move-generation optimization (`internal/chess/attacks.go`, `internal/chess/board.go`, `internal/chess/movegen.go`; search went from about 60k to about 300k nodes per second with identical node counts)
- [x] **43.** Search nodes-per-second benchmark (`BenchmarkSearchFixedDepth` and `BenchmarkSearchTimeBudget` in `internal/engine/bench_test.go`)
- [x] **44.** Comparative evaluation of bitboards (square-walking attack detection replaced by attack tables and ray lookups: perft about 2.0M to 6-8M nodes per second, `LegalMoves` 43.6us to 7.2us)
- [x] **45.** Allocation and garbage-collection optimization (append-style move generation: about 1,000,000 to about 47,000 allocations per search)

### Phase 9: Advanced engine features and delivery

- [x] **46.** Opening book support (`internal/book`; UCI options `OwnBook` and `BookFile`; text format with one line of UCI moves per game line)
- [x] **47.** Endgame tablebases: KPK bitbase only (`internal/engine/kpk.go`, built by retrograde analysis and used by `Evaluate`; no other tablebases)
- [x] **48.** Multi-threaded search (Lazy SMP in `internal/engine/analysis.go`; UCI option `Threads`; shared table with striped locks)
- [x] **49.** Engine strength benchmarking and rating estimation (`internal/elo`, `internal/match`, `cmd/match`: Elo difference between two configurations, with an optional anchor rating)
- [x] **50.** Automated regression pipeline and release builds (`.github/workflows/ci.yml`; written and cross-compiled locally, not yet run on GitHub)

## Verification and remaining work

Run the Go checks with:

```sh
go test ./...
```

The suite includes rule and perft tests, engine evaluation/search tests, UCI parser/session tests, and automated engine self-play that checks every chosen move for legality. Benchmarks are available with `go test ./internal/engine -run '^$' -bench 'BenchmarkSearch(FixedDepth|TimeBudget)$'`; the benchmark reports nodes per second, nodes per search, depth, and transposition-table occupancy.

Open items and limits: the CI workflow has not run on GitHub yet, and the data-race check (`go test -race`) needs a C compiler, so it runs only in CI. Tablebase support covers king and pawn against king only. The opening book is a plain text format, not Polyglot. `cmd/match` measures Elo differences between engine configurations, not an absolute rating, unless you pass `-anchor`. Threaded search is Lazy SMP and its node counts vary from run to run. The browser app connects to the local server at `ws://localhost:8080/ws`; promotion selection in the browser UI is not exposed.

## Project layout

```text
cmd/chess/          Terminal chess game
cmd/server/         HTTP and WebSocket game server
cmd/uci/            UCI engine binary
cmd/match/          Self-play strength benchmark
internal/chess/     Position, rules, move generation, make/unmake
internal/engine/    Evaluation and search
internal/uci/       UCI protocol
internal/book/      Opening book
internal/elo/       Elo estimation
internal/match/     Self-play match runner
internal/utils/     CLI board and move helpers
web/app/            Next.js browser application
web/public/         Chess piece and sound assets
```
