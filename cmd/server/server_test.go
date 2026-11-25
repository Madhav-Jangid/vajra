package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vajra/internal/chess"
	"vajra/internal/engine"
)

func setPosition(t *testing.T, fen string) {
	t.Helper()

	p, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN(%q): %v", fen, err)
	}

	chess.CurrentPosition = p
	turn = p.SideToMove
	over = false
	status = ""
	moveStack = nil
	engineEval = nil
	humanColor = chess.White
	seen = map[string]int{positionKey(): 1}

	t.Cleanup(func() {
		humanColor = chess.White

		newGame()
	})
}

func sq(name string) int {
	return int(name[0]-'a') + int(name[1]-'1')*8
}

func TestFindMovePromotion(t *testing.T) {
	setPosition(t, "7k/P7/8/8/8/8/8/K7 w - - 0 1")

	tests := []struct {
		promotion string
		want      chess.PieceType
	}{
		{"", chess.WhiteQueen},
		{"q", chess.WhiteQueen},
		{"Q", chess.WhiteQueen},
		{"r", chess.WhiteRook},
		{"b", chess.WhiteBishop},
		{"n", chess.WhiteKnight},
		{"N", chess.WhiteKnight},
	}

	for _, tc := range tests {
		m, ok := findMove(sq("a7"), sq("a8"), tc.promotion)
		if !ok || m.Promotion != tc.want {
			t.Fatalf("promotion %q gave %+v (found=%v), want %v", tc.promotion, m, ok, tc.want)
		}
	}
}

func TestFindMoveRejectsBadPromotionPieces(t *testing.T) {
	setPosition(t, "7k/P7/8/8/8/8/8/K7 w - - 0 1")

	for _, promotion := range []string{"k", "p", "x", "queen"} {
		if m, ok := findMove(sq("a7"), sq("a8"), promotion); ok {
			t.Fatalf("promotion %q must not be accepted, got %+v", promotion, m)
		}
	}
}

func TestFindMoveIgnoresPromotionForOrdinaryMoves(t *testing.T) {
	setPosition(t, chess.StartingFEN)

	for _, promotion := range []string{"", "q", "n"} {
		m, ok := findMove(sq("e2"), sq("e4"), promotion)
		if !ok || m.Flags&chess.Promotion != 0 {
			t.Fatalf("e2e4 with promotion %q gave %+v (found=%v)", promotion, m, ok)
		}
	}

	if _, ok := findMove(sq("e2"), sq("e5"), ""); ok {
		t.Fatal("an illegal move must not be found")
	}
}

func TestFindMovePromotionByCapture(t *testing.T) {
	setPosition(t, "1n5k/P7/8/8/8/8/8/K7 w - - 0 1")

	m, ok := findMove(sq("a7"), sq("b8"), "r")
	if !ok || m.Promotion != chess.WhiteRook || m.Captured != chess.BlackKnight {
		t.Fatalf("got %+v (found=%v), want a rook promotion capturing the knight", m, ok)
	}
}

func TestBlackCanUnderpromote(t *testing.T) {
	setPosition(t, "k7/8/8/8/8/8/p7/7K b - - 0 1")

	m, ok := findMove(sq("a2"), sq("a1"), "n")
	if !ok || m.Promotion != chess.BlackKnight {
		t.Fatalf("got %+v (found=%v), want a black knight", m, ok)
	}
}

func TestApplyMoveRecordsNotationAndLastMove(t *testing.T) {
	newGame()
	t.Cleanup(newGame)

	first, _ := findMove(sq("e2"), sq("e4"), "")
	applyMove(first)

	second, _ := findMove(sq("e7"), sq("e5"), "")
	applyMove(second)

	snap := snapshot("")

	if strings.Join(snap.History, " ") != "e4 e5" {
		t.Fatalf("history = %v", snap.History)
	}

	if snap.LastFrom != sq("e7") || snap.LastTo != sq("e5") {
		t.Fatalf("last move = %d to %d, want e7 to e5", snap.LastFrom, snap.LastTo)
	}
}

func TestHistoryUsesCheckAndPromotionNotation(t *testing.T) {
	setPosition(t, "7k/P7/8/8/8/8/8/K7 w - - 0 1")

	m, _ := findMove(sq("a7"), sq("a8"), "n")
	applyMove(m)

	if got := snapshot("").History; len(got) != 1 || got[0] != "a8=N" {
		t.Fatalf("history = %v, want [a8=N]", got)
	}

	setPosition(t, "7k/P7/8/8/8/8/8/K7 w - - 0 1")

	m, _ = findMove(sq("a7"), sq("a8"), "q")
	applyMove(m)

	if got := snapshot("").History; len(got) != 1 || got[0] != "a8=Q+" {
		t.Fatalf("history = %v, want [a8=Q+]", got)
	}
}

func TestUndoRemovesBothMovesFromHistory(t *testing.T) {
	newGame()
	t.Cleanup(newGame)

	for _, mv := range [][2]string{{"e2", "e4"}, {"e7", "e5"}, {"g1", "f3"}, {"b8", "c6"}} {
		m, ok := findMove(sq(mv[0]), sq(mv[1]), "")
		if !ok {
			t.Fatalf("%s%s is not legal", mv[0], mv[1])
		}

		applyMove(m)
	}

	undoMoves()

	snap := snapshot("")

	if strings.Join(snap.History, " ") != "e4 e5" {
		t.Fatalf("history after undo = %v, want [e4 e5]", snap.History)
	}

	if snap.LastFrom != sq("e7") || snap.LastTo != sq("e5") {
		t.Fatalf("last move after undo = %d to %d", snap.LastFrom, snap.LastTo)
	}

	if snap.Turn != "w" {
		t.Fatalf("turn after undo = %s, want w", snap.Turn)
	}
}

func TestUndoClearsTheEvaluation(t *testing.T) {
	newGame()
	t.Cleanup(newGame)

	m, _ := findMove(sq("e2"), sq("e4"), "")
	applyMove(m)

	engineEval = &evalMsg{Cp: 20, Depth: 5}

	undoMoves()

	if snapshot("").Eval != nil {
		t.Fatal("an evaluation of a position that was taken back must not be shown")
	}
}

func TestNewGameClearsEverything(t *testing.T) {
	m, _ := findMove(sq("e2"), sq("e4"), "")
	applyMove(m)

	engineEval = &evalMsg{Cp: 10, Depth: 3}

	newGame()

	snap := snapshot("")

	if len(snap.History) != 0 || snap.LastFrom != -1 || snap.LastTo != -1 || snap.Eval != nil || snap.Over {
		t.Fatalf("new game is not clean: %+v", snap)
	}
}

func TestSnapshotJSONShape(t *testing.T) {
	newGame()
	t.Cleanup(newGame)

	data, err := json.Marshal(snapshot(""))
	if err != nil {
		t.Fatal(err)
	}

	text := string(data)

	for _, want := range []string{`"history":[]`, `"lastFrom":-1`, `"lastTo":-1`, `"turn":"w"`, `"over":false`} {
		if !strings.Contains(text, want) {
			t.Fatalf("state JSON lacks %s:\n%s", want, text)
		}
	}

	if strings.Contains(text, `"eval"`) {
		t.Fatalf("no evaluation exists yet, but the JSON has one:\n%s", text)
	}
}

func TestWhiteEvalFlipsTheEngineScore(t *testing.T) {
	e := whiteEval(engine.SearchInfo{Score: 150, Depth: 9}, false)

	if e.Cp != -150 || e.Mate != 0 || e.Depth != 9 {
		t.Fatalf("got %+v: the engine plays Black, so +150 for it is -150 for White", e)
	}

	e = whiteEval(engine.SearchInfo{Score: -80, Depth: 4}, false)

	if e.Cp != 80 {
		t.Fatalf("got %+v, want +80 for White", e)
	}
}

func TestWhiteEvalReportsMateFromWhitesPointOfView(t *testing.T) {
	e := whiteEval(engine.SearchInfo{Score: engine.MateScore - 3, Depth: 8}, false)

	if e.Mate != -2 {
		t.Fatalf("the engine mates in 2, so White sees %d, want -2", e.Mate)
	}

	e = whiteEval(engine.SearchInfo{Score: -engine.MateScore + 4, Depth: 8}, false)

	if e.Mate != 2 {
		t.Fatalf("the engine is mated in 2, so White sees %d, want 2", e.Mate)
	}
}

func TestMoveMessageCarriesThePromotionPiece(t *testing.T) {
	var m inMsg

	if err := json.Unmarshal([]byte(`{"type":"move","from":48,"to":56,"promotion":"n"}`), &m); err != nil {
		t.Fatal(err)
	}

	if m.Type != "move" || m.From != 48 || m.To != 56 || m.Promotion != "n" {
		t.Fatalf("got %+v", m)
	}

	var plain inMsg

	if err := json.Unmarshal([]byte(`{"type":"move","from":12,"to":28}`), &plain); err != nil {
		t.Fatal(err)
	}

	if plain.Promotion != "" {
		t.Fatalf("a move without a promotion field has promotion %q", plain.Promotion)
	}
}

func fastSearch(t *testing.T) {
	t.Helper()

	oldDepth, oldTime := searchDepth, thinkTime

	searchDepth, thinkTime = 3, 200*time.Millisecond

	t.Cleanup(func() { searchDepth, thinkTime = oldDepth, oldTime })
}

func playingAs(t *testing.T, side chess.Color) {
	t.Helper()

	startGame(side)

	t.Cleanup(func() {
		humanColor = chess.White

		newGame()
	})
}

func TestWhiteEvalWhenTheEnginePlaysWhite(t *testing.T) {
	e := whiteEval(engine.SearchInfo{Score: 150, Depth: 9}, true)

	if e.Cp != 150 || e.Mate != 0 {
		t.Fatalf("got %+v: the engine is White, so +150 for it is +150 for White", e)
	}

	e = whiteEval(engine.SearchInfo{Score: engine.MateScore - 3, Depth: 8}, true)

	if e.Mate != 2 {
		t.Fatalf("the engine is White and mates in 2, so White sees %d, want 2", e.Mate)
	}

	e = whiteEval(engine.SearchInfo{Score: -engine.MateScore + 4, Depth: 8}, true)

	if e.Mate != -2 {
		t.Fatalf("the engine is White and is mated in 2, so White sees %d, want -2", e.Mate)
	}
}

func TestResolveSide(t *testing.T) {
	playingAs(t, chess.Black)

	if resolveSide("w") != chess.White || resolveSide("b") != chess.Black {
		t.Fatal("w and b must pick White and Black")
	}

	if resolveSide("") != chess.Black {
		t.Fatal("no choice keeps the side currently played")
	}

	if resolveSide("nonsense") != chess.Black {
		t.Fatal("an unknown choice keeps the side currently played")
	}

	seen := map[chess.Color]bool{}

	for i := 0; i < 200; i++ {
		seen[resolveSide("random")] = true
	}

	if !seen[chess.White] || !seen[chess.Black] {
		t.Fatalf("random never picked both sides: %v", seen)
	}
}

func TestStartGameSetsTheSideInTheState(t *testing.T) {
	for _, side := range []chess.Color{chess.Black, chess.White} {
		playingAs(t, side)

		snap := snapshot("")

		if snap.Side != sideName(side) || snap.Turn != "w" || len(snap.History) != 0 {
			t.Fatalf("side %v: got side=%q turn=%q history=%v", side, snap.Side, snap.Turn, snap.History)
		}

		data, _ := json.Marshal(snap)

		if !strings.Contains(string(data), `"side":"`+sideName(side)+`"`) {
			t.Fatalf("state JSON lacks the side: %s", data)
		}
	}
}

func TestEngineMovesFirstWhenYouPlayBlack(t *testing.T) {
	fastSearch(t)
	playingAs(t, chess.Black)

	var sent []outMsg

	engineReply(func(m outMsg) { sent = append(sent, m) })

	if len(sent) != 1 {
		t.Fatalf("the engine should send one state, sent %d", len(sent))
	}

	snap := sent[0]

	if len(snap.History) != 1 || snap.Turn != "b" || snap.Side != "b" {
		t.Fatalf("got history=%v turn=%s side=%s, want one White move and Black to move", snap.History, snap.Turn, snap.Side)
	}

	if snap.Eval == nil || snap.Eval.Depth == 0 {
		t.Fatal("the engine's move should carry its evaluation")
	}

	if snap.LastFrom < 0 || chess.CurrentPosition.Board.PieceAt(snap.LastTo).Color != chess.White {
		t.Fatalf("the last move should be a White move, got %d to %d", snap.LastFrom, snap.LastTo)
	}
}

func TestEngineDoesNotMoveOnYourTurn(t *testing.T) {
	fastSearch(t)
	playingAs(t, chess.White)

	called := false

	engineReply(func(outMsg) { called = true })

	if called || len(moveStack) != 0 {
		t.Fatal("the engine must wait for White to move first")
	}
}

func TestEngineDoesNotMoveAfterTheGameEnds(t *testing.T) {
	fastSearch(t)
	playingAs(t, chess.Black)

	over = true

	called := false

	engineReply(func(outMsg) { called = true })

	if called {
		t.Fatal("no engine move once the game is over")
	}
}

func TestPlayingBlackFullExchange(t *testing.T) {
	fastSearch(t)
	playingAs(t, chess.Black)

	engineReply(func(outMsg) {})

	if turn != chess.Black {
		t.Fatalf("it should be Black's turn after the engine's first move, got %v", turn)
	}

	var reply *outMsg

	for _, from := range legalFromSquares(chess.Black) {
		for _, move := range chess.CurrentPosition.LegalMoves(from) {
			m, ok := findMove(move.From, move.To, "")
			if !ok {
				t.Fatal("a legal move must be found")
			}

			applyMove(m)

			engineReply(func(o outMsg) { reply = &o })

			goto played
		}
	}

played:
	if reply == nil || len(reply.History) != 3 || reply.Turn != "b" {
		t.Fatalf("after Black moves and the engine answers there should be 3 moves and Black to move, got %+v", reply)
	}
}

func legalFromSquares(color chess.Color) []int {
	var squares []int

	for sq := 0; sq < 64; sq++ {
		piece := chess.CurrentPosition.Board.PieceAt(sq)

		if piece.Type != chess.Empty && piece.Color == color {
			squares = append(squares, sq)
		}
	}

	return squares
}

func TestUndoAsBlackKeepsTheEnginesFirstMove(t *testing.T) {
	fastSearch(t)
	playingAs(t, chess.Black)

	engineReply(func(outMsg) {})

	if canUndo() {
		t.Fatal("with only the engine's first move there is nothing of yours to take back")
	}

	for _, from := range legalFromSquares(chess.Black) {
		moves := chess.CurrentPosition.LegalMoves(from)

		if len(moves) == 0 {
			continue
		}

		applyMove(moves[0])

		break
	}

	engineReply(func(outMsg) {})

	if len(moveStack) != 3 || !canUndo() {
		t.Fatalf("three moves played, undo should be possible, got %d moves", len(moveStack))
	}

	undoMoves()

	if len(moveStack) != 1 || turn != chess.Black {
		t.Fatalf("undo should leave the engine's first move and Black to move, got %d moves, turn %v", len(moveStack), turn)
	}
}

func TestCanUndoAsWhite(t *testing.T) {
	setPosition(t, chess.StartingFEN)

	if canUndo() {
		t.Fatal("nothing to undo at the start")
	}

	m, _ := findMove(sq("e2"), sq("e4"), "")
	applyMove(m)

	if !canUndo() {
		t.Fatal("one move can be taken back")
	}
}

func collect() (func(outMsg), *[]outMsg) {
	var sent []outMsg

	return func(m outMsg) { sent = append(sent, m) }, &sent
}

func TestResetAsBlackLetsTheEngineMoveFirst(t *testing.T) {
	fastSearch(t)
	playingAs(t, chess.White)

	send, sent := collect()

	handle(send, inMsg{Type: "reset", Side: "b"})

	if len(*sent) != 2 {
		t.Fatalf("want the empty board then the engine's move, got %d messages", len(*sent))
	}

	first, second := (*sent)[0], (*sent)[1]

	if first.Side != "b" || len(first.History) != 0 || first.Turn != "w" {
		t.Fatalf("first message: side=%s history=%v turn=%s", first.Side, first.History, first.Turn)
	}

	if len(second.History) != 1 || second.Turn != "b" || second.Eval == nil {
		t.Fatalf("second message: history=%v turn=%s eval=%v", second.History, second.Turn, second.Eval)
	}
}

func TestResetAsWhiteWaitsForYou(t *testing.T) {
	fastSearch(t)
	playingAs(t, chess.Black)

	send, sent := collect()

	handle(send, inMsg{Type: "reset", Side: "w"})

	if len(*sent) != 1 || (*sent)[0].Side != "w" || (*sent)[0].Turn != "w" || len((*sent)[0].History) != 0 {
		t.Fatalf("got %+v", *sent)
	}
}

func TestResetWithoutASideKeepsTheCurrentOne(t *testing.T) {
	fastSearch(t)
	playingAs(t, chess.Black)

	send, sent := collect()

	handle(send, inMsg{Type: "reset"})

	if (*sent)[0].Side != "b" {
		t.Fatalf("a plain reset must keep the side, got %q", (*sent)[0].Side)
	}
}

func TestAMoveOutOfTurnIsRefused(t *testing.T) {
	fastSearch(t)
	playingAs(t, chess.Black)

	send, sent := collect()

	handle(send, inMsg{Type: "move", From: sq("e7"), To: sq("e5")})

	if len(*sent) != 1 || len((*sent)[0].History) != 0 || len(moveStack) != 0 {
		t.Fatalf("Black cannot move before White has: %+v", *sent)
	}
}

func TestYouCannotMoveTheEnginesPieces(t *testing.T) {
	fastSearch(t)
	playingAs(t, chess.White)

	send, sent := collect()

	handle(send, inMsg{Type: "move", From: sq("e7"), To: sq("e5")})

	if len(moveStack) != 0 || len((*sent)[len(*sent)-1].History) != 0 {
		t.Fatal("a black pawn must not move for the White player")
	}
}

func TestMovesAreOnlyOfferedForYourPiecesOnYourTurn(t *testing.T) {
	playingAs(t, chess.White)

	send, sent := collect()

	handle(send, inMsg{Type: "moves", Square: sq("e2")})
	handle(send, inMsg{Type: "moves", Square: sq("e7")})

	if len((*sent)[0].Moves) != 2 {
		t.Fatalf("e2 should offer e3 and e4, got %v", (*sent)[0].Moves)
	}

	if len((*sent)[1].Moves) != 0 {
		t.Fatalf("an engine piece must offer nothing, got %v", (*sent)[1].Moves)
	}

	playingAs(t, chess.Black)

	send, sent = collect()

	handle(send, inMsg{Type: "moves", Square: sq("e7")})

	if len((*sent)[0].Moves) != 0 {
		t.Fatalf("Black has no moves to offer while it is White's turn, got %v", (*sent)[0].Moves)
	}
}

func TestAWhiteGameStillWorksEndToEnd(t *testing.T) {
	fastSearch(t)
	playingAs(t, chess.White)

	send, sent := collect()

	handle(send, inMsg{Type: "move", From: sq("e2"), To: sq("e4")})

	if len(*sent) != 2 {
		t.Fatalf("want your move then the engine's reply, got %d messages", len(*sent))
	}

	if (*sent)[0].Turn != "b" || len((*sent)[0].History) != 1 {
		t.Fatalf("after your move: %+v", (*sent)[0])
	}

	if (*sent)[1].Turn != "w" || len((*sent)[1].History) != 2 || (*sent)[1].Side != "w" {
		t.Fatalf("after the reply: %+v", (*sent)[1])
	}

	handle(send, inMsg{Type: "undo"})

	last := (*sent)[len(*sent)-1]

	if len(last.History) != 0 || last.Turn != "w" {
		t.Fatalf("undo should take back both moves, got %+v", last)
	}
}

func TestUndoIsIgnoredWhenThereIsNothingOfYoursToTakeBack(t *testing.T) {
	fastSearch(t)
	playingAs(t, chess.Black)

	send, sent := collect()

	handle(send, inMsg{Type: "reset", Side: "b"})
	handle(send, inMsg{Type: "undo"})

	last := (*sent)[len(*sent)-1]

	if len(last.History) != 1 || last.Turn != "b" {
		t.Fatalf("undo must keep the engine's opening move, got %+v", last)
	}
}
