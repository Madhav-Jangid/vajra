# Web UI and game server

Besides the UCI engine, the repository contains a browser game: you play against Vajra on a chessboard in your browser, as White, as Black or as a random side, in a layout modelled on chess.com.

```
browser (web/)  <--- WebSocket JSON --->  cmd/server  --->  internal/engine
```

## Run it

Start the server from the repository root:

```sh
go run ./cmd/server
```

In a second terminal start the web app:

```sh
cd web
npm install
npm run dev
```

Open the address printed by `npm run dev` (normally `http://localhost:3000`). The page connects to `ws://<host>:8080/ws` and reconnects by itself if the server restarts.

The web app is built with Next.js, React and Tailwind CSS. After changing the server, restart it: the running process keeps the old code.

## What the UI does

- **Choosing a side.** Before a game starts, and after it ends, a **Play as** selector offers White, Random and Black, and a **New game** button starts the next game. While a game is in progress both are hidden. Choosing a side before the first move starts that game at once. Playing Black turns the board around, and Vajra makes the first move.
- **Moving.** Drag a piece to a square or click the piece and then the square. Legal destinations are shown as dots, captures as rings.
- **Premoves.** While Vajra is thinking, drag or click one of your pieces to queue a move. The two squares turn red. When Vajra replies the move is sent at once; if it is no longer legal the server refuses it and the premove is dropped. Right-click cancels. Only one premove is kept, and castling cannot be premoved.
- **Promotion.** A pawn reaching the last rank opens a picker over the board with queen, knight, rook and bishop. Clicking outside it cancels the move. It works for moves and for premoves, from either side of a flipped board.
- **Board.** Rank and file labels, the last move highlighted, and a red blink on the king's square when it is in check. **Flip board** turns the board around.
- **Players.** Each side shows captured pieces and the material difference. Vajra's bar shows the depth of its last search and animated dots while it thinks.
- **Evaluation bar.** A vertical bar beside the board shows Vajra's evaluation from White's point of view, with `M3` style labels for forced mates.
- **Move list.** Moves in standard notation (`e4`, `Nxd5`, `O-O`, `e8=Q+`), with the latest move highlighted.
- **Game over.** A card on the board says who won and why (checkmate, stalemate, insufficient material, the fifty-move rule or repetition) and offers a new game. Winning by checkmate sets off confetti.
- **Sounds** for a move, capture, castle, promotion, check and the end of the game.
- **New game** and **Undo**, which takes back your last move and Vajra's reply.

## Server behaviour

The server runs one game at a time and keeps it in memory. It is meant for local play, not for several people at once: every connection sees and changes the same game. The engine searches up to depth 20 with a limit of 5 seconds per move. Both are set in `internal/config.go`.

The server implements the WebSocket protocol itself, using only the standard library.

## Protocol

Messages are JSON. Squares are numbers from 0 (`a1`) to 63 (`h8`).

### Client to server

| Message | Meaning |
| --- | --- |
| `{"type":"state"}` | Ask for the current position. |
| `{"type":"reset"}` | Start a new game, keeping the side you play. |
| `{"type":"reset","side":"b"}` | Start a new game as `w`, `b` or `random`. When you play Black, the server answers with the empty board and then with Vajra's first move. |
| `{"type":"undo"}` | Take back your last move and Vajra's reply. Ignored when you play Black and only Vajra's opening move has been played. |
| `{"type":"moves","square":12}` | Ask for the legal destinations of the piece on a square. |
| `{"type":"move","from":12,"to":28}` | Play a move. The server answers at once with the new position, then again after the engine has replied. |
| `{"type":"move","from":48,"to":56,"promotion":"n"}` | Play a promotion. `promotion` is `q`, `r`, `b` or `n` and defaults to `q`. |

A move that is not legal is ignored and the server answers with the unchanged position.

### Server to client

`state`:

```json
{
  "type": "state",
  "board": ["R", "N", "...", "k", "r"],
  "turn": "w",
  "side": "w",
  "event": "move-self",
  "hasCheck": false,
  "check": -1,
  "over": false,
  "status": "",
  "history": ["e4", "d6"],
  "lastFrom": 51,
  "lastTo": 43,
  "eval": { "cp": 36, "mate": 0, "depth": 11 }
}
```

`side` is the side you play, `w` or `b`. `board` has 64 entries from `a1` to `h8`. A capital letter is a white piece, a lower case letter is a black piece, and `"."` is an empty square. `event` is one of `move-self`, `capture`, `castle`, `promote`, `move-check` or `game-end`, and tells the UI which sound to play. `check` is the square of the king in check, valid when `hasCheck` is true. When `over` is true, `status` says why.

`history` is every move so far in standard algebraic notation. `lastFrom` and `lastTo` are the squares of the last move, or `-1` before the first move. `eval` is Vajra's evaluation of the position after its last search, always from White's point of view whichever side Vajra plays: `cp` in centipawns, `mate` as the number of moves to a forced mate (positive when White mates, negative when Black does, `0` when there is none) and `depth` the search depth. It is absent until Vajra has moved, and after an undo.

`moves`:

```json
{ "type": "moves", "square": 12, "moves": [20, 28] }
```

The server only answers `moves` for the side to move. The browser works out premove destinations itself, from piece movement alone and ignoring checks, and relies on the server to refuse a premove that turns out to be illegal.

## Tests

```sh
cd web
npm test
npm run lint
npm run build
```

`npm test` runs the unit tests of the helpers in `web/lib`: square and board orientation, promotion detection, captured pieces, premove destinations, evaluation bar maths and result text. The server's game logic (promotion choice, notation in the history, undo, the evaluation sign) is covered by `go test ./cmd/server`.

## Limitations

- One shared game, as described above.
- Premoves cannot castle or capture en passant.
- The interface has been exercised in a real browser with scripted mouse input, but there are no automated browser tests in the repository yet.
