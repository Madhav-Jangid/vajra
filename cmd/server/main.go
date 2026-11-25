package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/bits"
	"net"
	"net/http"
	"sync"

	"vajra/internal"
	"vajra/internal/chess"
	"vajra/internal/engine"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type wsConn struct {
	c net.Conn
}

func upgrade(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "websocket required", http.StatusBadRequest)
		return nil, io.ErrUnexpectedEOF
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack unsupported", http.StatusInternalServerError)
		return nil, io.ErrUnexpectedEOF
	}

	conn, buf, err := hj.Hijack()
	if err != nil {
		return nil, err
	}

	sum := sha1.Sum([]byte(key + wsGUID))
	accept := base64.StdEncoding.EncodeToString(sum[:])

	buf.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	buf.WriteString("Upgrade: websocket\r\n")
	buf.WriteString("Connection: Upgrade\r\n")
	buf.WriteString("Sec-WebSocket-Accept: ")
	buf.WriteString(accept)
	buf.WriteString("\r\n\r\n")

	if err := buf.Flush(); err != nil {
		conn.Close()
		return nil, err
	}

	return &wsConn{c: conn}, nil
}

func (w *wsConn) read() ([]byte, error) {
	for {
		var h [2]byte

		if _, err := io.ReadFull(w.c, h[:]); err != nil {
			return nil, err
		}

		opcode := h[0] & 0x0f
		masked := h[1]&0x80 != 0
		n := uint64(h[1] & 0x7f)

		switch n {
		case 126:
			var b [2]byte

			if _, err := io.ReadFull(w.c, b[:]); err != nil {
				return nil, err
			}

			n = uint64(binary.BigEndian.Uint16(b[:]))

		case 127:
			var b [8]byte

			if _, err := io.ReadFull(w.c, b[:]); err != nil {
				return nil, err
			}

			n = binary.BigEndian.Uint64(b[:])
		}

		if n > 1<<20 {
			return nil, io.ErrShortBuffer
		}

		var mask [4]byte

		if masked {
			if _, err := io.ReadFull(w.c, mask[:]); err != nil {
				return nil, err
			}
		}

		payload := make([]byte, n)

		if _, err := io.ReadFull(w.c, payload); err != nil {
			return nil, err
		}

		if masked {
			for i := range payload {
				payload[i] ^= mask[i%4]
			}
		}

		switch opcode {
		case 0x1, 0x2:
			return payload, nil

		case 0x8:
			return nil, io.EOF

		case 0x9:
			if err := w.write(0xA, payload); err != nil {
				return nil, err
			}
		}
	}
}

func (w *wsConn) write(opcode byte, payload []byte) error {
	hdr := []byte{0x80 | opcode}

	switch n := len(payload); {
	case n < 126:
		hdr = append(hdr, byte(n))

	case n < 1<<16:
		hdr = append(
			hdr,
			126,
			byte(n>>8),
			byte(n),
		)

	default:
		hdr = append(hdr, 127)
		hdr = binary.BigEndian.AppendUint64(hdr, uint64(n))
	}

	_, err := w.c.Write(append(hdr, payload...))
	return err
}

func (w *wsConn) send(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return w.write(0x1, b)
}

var (
	mu sync.Mutex

	turn   = chess.White
	over   bool
	status string

	seen = map[string]int{}
)

func newGame() {
	if err := chess.SetUpInitialPosition(); err != nil {
		log.Printf("failed to setup game: %v", err)
		return
	}

	turn = chess.White
	over = false
	status = ""

	seen = map[string]int{
		positionKey(): 1,
	}

	moveStack = nil
}

// played records a move and the state needed to take it back.
type played struct {
	move chess.Move
	undo chess.Undo
}

var moveStack []played

// undoMoves takes back the last move, plus the engine's reply
// so it is the human (White) to move again.
func undoMoves() {
	for len(moveStack) > 0 {
		key := positionKey()
		if seen[key] > 1 {
			seen[key]--
		} else {
			delete(seen, key)
		}

		last := moveStack[len(moveStack)-1]
		moveStack = moveStack[:len(moveStack)-1]
		chess.CurrentPosition.UnmakeMove(last.move, last.undo)

		if chess.CurrentPosition.SideToMove == chess.White {
			break
		}
	}

	turn = chess.CurrentPosition.SideToMove
	over = false
	status = ""
}

type inMsg struct {
	Type   string `json:"type"`
	Square int    `json:"square"`
	From   int    `json:"from"`
	To     int    `json:"to"`
}

type outMsg struct {
	Type string `json:"type"`

	Board []string `json:"board,omitempty"`

	Turn string `json:"turn,omitempty"`

	Square int   `json:"square"`
	Moves  []int `json:"moves,omitempty"`

	Event string `json:"event,omitempty"`

	HasCheck bool `json:"hasCheck"`
	Check    int  `json:"check"`

	Status string `json:"status,omitempty"`
	Over   bool   `json:"over"`
}

func turnName() string {
	if turn == chess.White {
		return "w"
	}

	return "b"
}

func positionKey() string {
	return chess.CurrentPosition.Key()
}

func snapshot(event string) outMsg {
	board := make([]string, 64)

	for i := range board {
		board[i] = chess.CurrentPosition.Board.PieceAt(i).Face
	}

	check := checkedKing()
	hasCheck := check >= 0

	if hasCheck && event != "" {
		event = "move-check"
	}

	return outMsg{
		Type:     "state",
		Board:    board,
		Turn:     turnName(),
		Event:    event,
		HasCheck: hasCheck,
		Check:    check,
		Over:     over,
		Status:   status,
	}
}

func checkedKing() int {
	var king chess.Bitboard
	enemy := chess.White

	if turn == chess.White {
		king = chess.CurrentPosition.Board.WhiteKing
		enemy = chess.Black
	} else {
		king = chess.CurrentPosition.Board.BlackKing
		enemy = chess.White
	}

	if king == 0 {
		return -1
	}

	sq := bits.TrailingZeros64(uint64(king))

	if chess.CurrentPosition.Board.IsSquareAttacked(
		sq,
		enemy,
	) {
		return sq
	}

	return -1
}

// applyMove updates all game-level state after a move.
// This is used for both the human move and the engine move.
func applyMove(move chess.Move) string {
	undo := chess.CurrentPosition.MakeMove(move)
	moveStack = append(moveStack, played{move: move, undo: undo})

	event := moveEvent(move)

	turn = chess.CurrentPosition.SideToMove
	status = chess.CurrentPosition.Status(turn)

	key := positionKey()
	seen[key]++

	if status == "" && seen[key] >= 3 {
		status = "threefold repetition"
	}

	if status != "" {
		over = true
		event = "game-end"
	}

	return event
}

func handle(ws *wsConn, m inMsg) {
	mu.Lock()
	defer mu.Unlock()

	switch m.Type {

	case "state":
		_ = ws.send(snapshot(""))

	case "reset":
		newGame()
		_ = ws.send(snapshot(""))

	case "undo":
		undoMoves()
		_ = ws.send(snapshot(""))

	case "moves":
		p := chess.CurrentPosition.Board.PieceAt(m.Square)

		var moves []chess.Move

		if !over &&
			p.Type != chess.Empty &&
			p.Color == turn {

			moves = chess.CurrentPosition.LegalMoves(m.Square)
		}

		destinations := make([]int, 0, len(moves))

		for _, move := range moves {
			destinations = append(destinations, move.To)
		}

		_ = ws.send(outMsg{
			Type:   "moves",
			Square: m.Square,
			Moves:  destinations,
		})

	case "move":
		// ----------------------------------------
		// Validate human move
		// ----------------------------------------

		p := chess.CurrentPosition.Board.PieceAt(m.From)

		if over ||
			p.Type == chess.Empty ||
			p.Color != turn {

			_ = ws.send(snapshot(""))
			return
		}

		move, ok := findMove(m.From, m.To)

		if !ok {
			_ = ws.send(snapshot(""))
			return
		}

		// ----------------------------------------
		// Human move
		// ----------------------------------------

		event := applyMove(move)

		// Send the human move immediately.
		//
		// This means the frontend does not have to
		// wait for the engine search to finish.
		_ = ws.send(snapshot(event))

		if over {
			return
		}

		if turn != chess.Black {
			return
		}

		fmt.Println("Engine thinking...")
		engineMove, hasMove := engine.IterativeSearch(
			&chess.CurrentPosition,
			internal.Depth,
			internal.MaxTimeLimit,
		)
		if !hasMove {
			log.Println("engine found no move")
			return
		}

		fmt.Println("Human move:", moveString(move))
		fmt.Println("Engine move:", moveString(engineMove))

		engineEvent := applyMove(engineMove)

		_ = ws.send(snapshot(engineEvent))

		chess.CurrentPosition.ValidateFen()

		fmt.Println(
			"Evaluation:",
			engine.Evaluate(&chess.CurrentPosition),
		)
	}
}

// moveString formats a move in coordinate notation, e.g. "e2e4" or "e7e8q".
func moveString(m chess.Move) string {
	s := engine.SquareName(m.From) + engine.SquareName(m.To)

	if m.Flags&chess.Promotion != 0 {
		switch m.Promotion {
		case chess.WhiteQueen, chess.BlackQueen:
			s += "q"
		case chess.WhiteRook, chess.BlackRook:
			s += "r"
		case chess.WhiteBishop, chess.BlackBishop:
			s += "b"
		case chess.WhiteKnight, chess.BlackKnight:
			s += "n"
		}
	}

	return s
}

func findMove(from, to int) (chess.Move, bool) {
	moves := chess.CurrentPosition.LegalMoves(from)

	for _, move := range moves {
		if move.From == from && move.To == to {
			return move, true
		}
	}

	return chess.Move{}, false
}

func moveEvent(move chess.Move) string {
	if move.Flags&chess.Promotion != 0 {
		return "promote"
	}

	if move.Flags&chess.KingCastle != 0 ||
		move.Flags&chess.QueenCastle != 0 {
		return "castle"
	}

	if move.Flags&chess.Capture != 0 {
		return "capture"
	}

	return "move-self"
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrade(w, r)
	if err != nil {
		return
	}

	defer ws.c.Close()

	mu.Lock()
	_ = ws.send(snapshot(""))
	mu.Unlock()

	for {
		data, err := ws.read()
		if err != nil {
			return
		}

		var m inMsg

		if json.Unmarshal(data, &m) != nil {
			continue
		}

		handle(ws, m)
	}
}

func main() {
	newGame()

	http.HandleFunc("/ws", wsHandler)

	log.Println("websocket server on ws://localhost:8080/ws")

	log.Fatal(
		http.ListenAndServe(":8080", nil),
	)
}
