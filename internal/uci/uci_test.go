package uci

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"vajra/internal/chess"
	"vajra/internal/engine"
)

// restoreEngineState puts back the engine globals a UCI session can change.
func restoreEngineState(t *testing.T) {
	t.Helper()

	oldTT, oldHistory, oldThreads := engine.TT, engine.GameHistory, engine.Threads

	t.Cleanup(func() {
		engine.TT, engine.GameHistory, engine.Threads = oldTT, oldHistory, oldThreads

		engine.ClearStop()
	})
}

// session runs a whole script through the engine and returns the output lines.
func session(t *testing.T, script string) []string {
	t.Helper()

	restoreEngineState(t)

	var out bytes.Buffer

	if err := New(strings.NewReader(script), &out).Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var lines []string

	for _, l := range strings.Split(out.String(), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}

	return lines
}

func linesWithPrefix(lines []string, prefix string) []string {
	var out []string

	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			out = append(out, l)
		}
	}

	return out
}

func bestMove(t *testing.T, lines []string) string {
	t.Helper()

	best := linesWithPrefix(lines, "bestmove ")
	if len(best) != 1 {
		t.Fatalf("want exactly one bestmove, got %v in %v", best, lines)
	}

	return strings.TrimPrefix(best[0], "bestmove ")
}

func TestHandshake(t *testing.T) {
	lines := session(t, "uci\nisready\nquit\n")

	want := []string{
		"id name " + Name + " " + Version,
		"id author Madhav Jangid",
		"option name Hash type spin default 64 min 1 max 1024",
		"option name Threads type spin default 1 min 1 max 64",
		"option name OwnBook type check default false",
		"option name BookFile type string default <empty>",
		"option name Move Overhead type spin default 50 min 0 max 5000",
		"uciok",
		"readyok",
	}

	if len(lines) != len(want) {
		t.Fatalf("got %d lines %v, want %d", len(lines), lines, len(want))
	}

	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestEmptyAndUnknownInputIsIgnored(t *testing.T) {
	lines := session(t, "\n   \nfrobnicate now\ndebug on\nregister later\nponderhit\nisready\n")

	if len(lines) != 1 || lines[0] != "readyok" {
		t.Fatalf("got %v, want only readyok", lines)
	}
}

func TestWindowsLineEndings(t *testing.T) {
	lines := session(t, "uci\r\nisready\r\nquit\r\n")

	if len(linesWithPrefix(lines, "uciok")) != 1 || len(linesWithPrefix(lines, "readyok")) != 1 {
		t.Fatalf("CRLF input was not understood: %v", lines)
	}
}

func TestGoDepthAnswersWithALegalMove(t *testing.T) {
	lines := session(t, "position startpos moves e2e4 e7e5\ngo depth 3\n")

	move := bestMove(t, lines)

	p := position(t, chess.StartingFEN)

	for _, m := range []string{"e2e4", "e7e5"} {
		mv, err := ParseMove(&p, m)
		if err != nil {
			t.Fatal(err)
		}

		p.MakeMove(mv)
	}

	if _, err := ParseMove(&p, move); err != nil {
		t.Fatalf("bestmove %s is not legal: %v", move, err)
	}
}

func TestInfoLinesCountDepthsUp(t *testing.T) {
	lines := session(t, "position startpos\ngo depth 4\n")

	info := linesWithPrefix(lines, "info depth ")
	if len(info) != 4 {
		t.Fatalf("want 4 info lines for depth 4, got %d: %v", len(info), info)
	}

	for i, l := range info {
		prefix := "info depth " + string(rune('1'+i)) + " score cp "

		if !strings.HasPrefix(l, prefix) {
			t.Fatalf("line %d = %q, want prefix %q", i, l, prefix)
		}

		for _, field := range []string{" nodes ", " time ", " pv "} {
			if !strings.Contains(l, field) {
				t.Fatalf("line %q lacks %q", l, field)
			}
		}
	}

	// The last line before bestmove is the final iteration, and bestmove is last.
	if !strings.HasPrefix(lines[len(lines)-1], "bestmove ") {
		t.Fatalf("bestmove must be the last line, got %v", lines)
	}
}

func TestPVMatchesBestMove(t *testing.T) {
	lines := session(t, "position startpos\ngo depth 3\n")

	info := linesWithPrefix(lines, "info depth 3 ")
	if len(info) != 1 {
		t.Fatalf("no depth 3 line in %v", lines)
	}

	pv := info[0][strings.Index(info[0], " pv ")+4:]
	fields := strings.Fields(pv)

	if len(fields) == 0 || fields[0] != bestMove(t, lines) {
		t.Fatalf("pv %q differs from the bestmove %q", pv, bestMove(t, lines))
	}
}

func TestMatesAreReportedAndSearchStopsEarly(t *testing.T) {
	tests := []struct {
		name, fen, move, score string
	}{
		{"white mates in 1", "6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1", "a1a8", "mate 1"},
		{"black mates in 1", "r3k3/8/8/8/8/8/5PPP/6K1 b - - 0 1", "a8a1", "mate 1"},
		{"mate in 2", "kbK5/pp6/1P6/8/8/8/8/R7 w - - 0 1", "a1a6", "mate 2"},
		{"knight promotion mate", "1nb5/Pnk1K3/2p5/8/8/8/8/8 w - - 0 1", "a7a8n", "mate 1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lines := session(t, "position fen "+tc.fen+"\ngo depth 20\n")

			if got := bestMove(t, lines); got != tc.move {
				t.Fatalf("bestmove %s, want %s", got, tc.move)
			}

			info := linesWithPrefix(lines, "info depth ")
			last := info[len(info)-1]

			if !strings.Contains(last, "score "+tc.score+" ") {
				t.Fatalf("last info %q should report %s", last, tc.score)
			}

			if len(info) >= 20 {
				t.Fatalf("searched all %d depths even though the mate was proven", len(info))
			}
		})
	}
}

func TestBeingMatedIsReportedAsNegative(t *testing.T) {
	// Black is to move and every reply is met by mate on the next move.
	lines := session(t, "position fen kbK5/pp6/RP6/8/8/8/8/8 b - - 0 1\ngo depth 10\n")

	if got := bestMove(t, lines); got != "b7a6" {
		t.Fatalf("bestmove %s, want b7a6", got)
	}

	info := linesWithPrefix(lines, "info depth ")

	if last := info[len(info)-1]; !strings.Contains(last, "score mate -1 ") {
		t.Fatalf("last info %q should report mate -1", last)
	}

	if len(info) >= 10 {
		t.Fatal("the search should stop once the mate is proven")
	}
}

func TestNoLegalMoveGivesNullMove(t *testing.T) {
	tests := []struct{ name, fen string }{
		{"checkmate", "R5k1/5ppp/8/8/8/8/8/4K3 b - - 0 1"},
		{"stalemate", "7k/5Q2/6K1/8/8/8/8/8 b - - 0 1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lines := session(t, "position fen "+tc.fen+"\ngo depth 3\n")

			if got := bestMove(t, lines); got != nullMove {
				t.Fatalf("bestmove %s, want %s", got, nullMove)
			}
		})
	}
}

func TestPositionVariants(t *testing.T) {
	const kiwipete = "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"

	tests := []struct {
		name    string
		command string
		wantFEN string
	}{
		{"startpos", "position startpos", chess.StartingFEN},
		{"startpos after moves", "position startpos moves e2e4 e7e5", "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2"},
		{"fen", "position fen " + kiwipete, kiwipete},
		{"fen then castle", "position fen " + kiwipete + " moves e1g1", "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R4RK1 b kq - 1 1"},
		{"fen without counters", "position fen 8/8/8/8/8/8/8/k6K w - -", "8/8/8/8/8/8/8/k6K w - - 0 1"},
		{"promotion", "position fen 7k/P7/8/8/8/8/8/K7 w - - 0 1 moves a7a8q", "Q6k/8/8/8/8/8/8/K7 b - - 0 1"},
		{"en passant", "position fen 4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1 moves e5d6", "4k3/8/3P4/8/8/8/8/4K3 b - - 0 1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lines := session(t, tc.command+"\nd\n")

			want := "info string fen " + tc.wantFEN

			if len(lines) != 1 || lines[0] != want {
				t.Fatalf("got %v\nwant %q", lines, want)
			}
		})
	}
}

func TestIllegalMoveLeavesThePositionAlone(t *testing.T) {
	lines := session(t, "position startpos moves e2e4\nposition startpos moves e2e4 e2e4\nd\n")

	if len(lines) != 2 || !strings.HasPrefix(lines[0], "info string ") || !strings.Contains(lines[0], "bad move") {
		t.Fatalf("expected an error then the old position, got %v", lines)
	}

	want := "info string fen rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1"

	if lines[1] != want {
		t.Fatalf("position changed: %q, want %q", lines[1], want)
	}
}

func TestBadFENIsReportedAndIgnored(t *testing.T) {
	lines := session(t, "position fen 9/8/8/8/8/8/8/8 w - - 0 1\nd\n")

	if len(lines) != 2 || !strings.Contains(lines[0], "bad position") {
		t.Fatalf("got %v", lines)
	}

	if lines[1] != "info string fen "+chess.StartingFEN {
		t.Fatalf("the starting position should remain, got %q", lines[1])
	}
}

func TestMalformedCommandsAreReported(t *testing.T) {
	lines := session(t, "go depth x\nposition\nsetoption Hash 5\nisready\n")

	infos := linesWithPrefix(lines, "info string ")
	if len(infos) != 3 {
		t.Fatalf("want 3 error lines, got %v", lines)
	}

	if lines[len(lines)-1] != "readyok" {
		t.Fatalf("the engine must keep working after bad input: %v", lines)
	}
}

func TestNewGameResetsThePosition(t *testing.T) {
	lines := session(t, "position startpos moves e2e4 c7c5\nucinewgame\nd\n")

	if len(lines) != 1 || lines[0] != "info string fen "+chess.StartingFEN {
		t.Fatalf("got %v", lines)
	}
}

func TestSetOption(t *testing.T) {
	restoreEngineState(t)

	before := engine.TT

	lines := session(t, "setoption name Hash value 8\nsetoption name Move Overhead value 20\nisready\n")

	if len(lines) != 1 || lines[0] != "readyok" {
		t.Fatalf("valid options should be silent, got %v", lines)
	}

	if engine.TT == before {
		t.Fatal("setting Hash should replace the transposition table")
	}

	for _, bad := range []string{
		"setoption name Hash value 0",
		"setoption name Hash value 99999",
		"setoption name Hash value big",
		"setoption name Hash",
		"setoption name Move Overhead value -5",
		"setoption name Move Overhead value 999999",
		"setoption name Move Overhead value slow",
		"setoption name Nonsense value 1",
	} {
		lines := session(t, bad+"\nisready\n")

		if len(lines) != 2 || !strings.HasPrefix(lines[0], "info string ") || lines[1] != "readyok" {
			t.Fatalf("%q: got %v, want one error line then readyok", bad, lines)
		}
	}
}

func TestOptionNamesAreCaseInsensitive(t *testing.T) {
	lines := session(t, "setoption name hash value 4\nsetoption name MOVE OVERHEAD value 10\nisready\n")

	if len(lines) != 1 {
		t.Fatalf("got %v", lines)
	}
}

func TestGoMovetimeIsRespected(t *testing.T) {
	start := time.Now()

	lines := session(t, "position startpos\ngo movetime 300\n")

	elapsed := time.Since(start)

	bestMove(t, lines)

	if elapsed > 3*time.Second {
		t.Fatalf("movetime 300 took %v", elapsed)
	}

	if elapsed < 150*time.Millisecond {
		t.Fatalf("returned after %v, the search stopped far too early", elapsed)
	}
}

func TestGoWithClockUsesAFractionOfIt(t *testing.T) {
	start := time.Now()

	lines := session(t, "position startpos\ngo wtime 3000 btime 3000\n")

	bestMove(t, lines)

	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("a 3 second clock should use about 100ms a move, took %v", elapsed)
	}
}

func TestRepetitionHistoryReachesTheEngine(t *testing.T) {
	restoreEngineState(t)

	var out bytes.Buffer

	script := "position startpos moves g1f3 g8f6 f3g1 f6g8 g1f3\ngo depth 1\n"

	if err := New(strings.NewReader(script), &out).Run(); err != nil {
		t.Fatal(err)
	}

	// One hash for the start and for each move except the last position.
	if got := len(engine.GameHistory); got != 5 {
		t.Fatalf("GameHistory has %d entries, want 5", got)
	}

	p := position(t, chess.StartingFEN)

	if engine.GameHistory[0] != engine.HashPosition(&p) {
		t.Fatal("the first history entry should be the starting position")
	}
}

func TestSequentialSearchesEachAnswer(t *testing.T) {
	lines := session(t, "position startpos\ngo depth 2\nposition startpos moves e2e4\ngo depth 2\ngo depth 1\n")

	if got := len(linesWithPrefix(lines, "bestmove ")); got != 3 {
		t.Fatalf("want 3 answers, got %d in %v", got, lines)
	}
}

// lineSink is an io.Writer a test can wait on while the engine is running.
type lineSink struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	lines []string
}

func (s *lineSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf.Write(p)

	for {
		text := s.buf.String()

		i := strings.IndexByte(text, '\n')
		if i < 0 {
			break
		}

		s.lines = append(s.lines, strings.TrimSpace(text[:i]))
		s.buf.Next(i + 1)
	}

	return len(p), nil
}

func (s *lineSink) snapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.lines...)
}

func (s *lineSink) waitFor(prefix string, timeout time.Duration) (string, bool) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if found := linesWithPrefix(s.snapshot(), prefix); len(found) > 0 {
			return found[0], true
		}

		time.Sleep(5 * time.Millisecond)
	}

	return "", false
}

// live runs an engine on a pipe so a test can send commands one at a time.
type live struct {
	in   *io.PipeWriter
	out  *lineSink
	done chan struct{}
	err  error
}

func startLive(t *testing.T) *live {
	t.Helper()

	restoreEngineState(t)

	r, w := io.Pipe()
	sink := &lineSink{}
	l := &live{in: w, out: sink, done: make(chan struct{})}

	go func() {
		l.err = New(r, sink).Run()

		close(l.done)
	}()

	t.Cleanup(func() {
		w.Close()

		select {
		case <-l.done:
		case <-time.After(5 * time.Second):
			t.Error("engine did not shut down")
		}
	})

	return l
}

func (l *live) send(t *testing.T, line string) {
	t.Helper()

	if _, err := io.WriteString(l.in, line+"\n"); err != nil {
		t.Fatalf("send %q: %v", line, err)
	}
}

func TestStopEndsAnInfiniteSearch(t *testing.T) {
	l := startLive(t)

	l.send(t, "position startpos")
	l.send(t, "go infinite")

	if _, ok := l.out.waitFor("info depth 3 ", 5*time.Second); !ok {
		t.Fatalf("the search never got going: %v", l.out.snapshot())
	}

	if got := linesWithPrefix(l.out.snapshot(), "bestmove"); len(got) != 0 {
		t.Fatalf("an infinite search answered before stop: %v", got)
	}

	l.send(t, "stop")

	line, ok := l.out.waitFor("bestmove ", 5*time.Second)
	if !ok {
		t.Fatalf("no bestmove after stop: %v", l.out.snapshot())
	}

	p := position(t, chess.StartingFEN)

	if _, err := ParseMove(&p, strings.TrimPrefix(line, "bestmove ")); err != nil {
		t.Fatalf("%s is not a legal move: %v", line, err)
	}
}

func TestInfiniteSearchWaitsForStopEvenAfterAMate(t *testing.T) {
	l := startLive(t)

	l.send(t, "position fen 6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1")
	l.send(t, "go infinite")

	if _, ok := l.out.waitFor("info depth 2 ", 5*time.Second); !ok {
		t.Fatalf("no search output: %v", l.out.snapshot())
	}

	time.Sleep(200 * time.Millisecond)

	if got := linesWithPrefix(l.out.snapshot(), "bestmove"); len(got) != 0 {
		t.Fatalf("answered before stop: %v", got)
	}

	l.send(t, "stop")

	line, ok := l.out.waitFor("bestmove ", 5*time.Second)
	if !ok || line != "bestmove a1a8" {
		t.Fatalf("got %q (found=%v), want bestmove a1a8", line, ok)
	}
}

func TestStopWithoutASearchIsHarmless(t *testing.T) {
	lines := session(t, "stop\nstop\nisready\n")

	if len(lines) != 1 || lines[0] != "readyok" {
		t.Fatalf("got %v", lines)
	}
}

func TestIsReadyAnswersDuringASearch(t *testing.T) {
	l := startLive(t)

	l.send(t, "position startpos")
	l.send(t, "go infinite")
	l.send(t, "isready")

	if _, ok := l.out.waitFor("readyok", 5*time.Second); !ok {
		t.Fatalf("isready was not answered while searching: %v", l.out.snapshot())
	}

	l.send(t, "stop")
	l.out.waitFor("bestmove ", 5*time.Second)
}

func TestQuitStopsASearchAndExits(t *testing.T) {
	l := startLive(t)

	l.send(t, "position startpos")
	l.send(t, "go infinite")
	l.out.waitFor("info depth 2 ", 5*time.Second)
	l.send(t, "quit")

	select {
	case <-l.done:
		if l.err != nil {
			t.Fatalf("Run returned %v", l.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("quit did not end the engine")
	}
}

func TestEndOfInputStopsAnInfiniteSearch(t *testing.T) {
	l := startLive(t)

	l.send(t, "position startpos")
	l.send(t, "go infinite")
	l.out.waitFor("info depth 2 ", 5*time.Second)

	l.in.Close()

	select {
	case <-l.done:
	case <-time.After(5 * time.Second):
		t.Fatal("closing the input did not end an infinite search")
	}
}

func TestPositionDuringAnInfiniteSearchStopsIt(t *testing.T) {
	l := startLive(t)

	l.send(t, "position startpos")
	l.send(t, "go infinite")
	l.out.waitFor("info depth 2 ", 5*time.Second)
	l.send(t, "position startpos moves e2e4")
	l.send(t, "d")

	if _, ok := l.out.waitFor("bestmove ", 5*time.Second); !ok {
		t.Fatalf("the old search was never ended: %v", l.out.snapshot())
	}

	if _, ok := l.out.waitFor("info string fen rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1", 5*time.Second); !ok {
		t.Fatalf("the new position was not applied: %v", l.out.snapshot())
	}
}

func TestNewGameAfterASearchStartsClean(t *testing.T) {
	lines := session(t, "position startpos moves e2e4\ngo depth 2\nucinewgame\nposition startpos\ngo depth 2\nd\n")

	if got := len(linesWithPrefix(lines, "bestmove ")); got != 2 {
		t.Fatalf("want 2 answers, got %d in %v", got, lines)
	}
}

func TestThreadsOption(t *testing.T) {
	lines := session(t, "setoption name Threads value 3\nposition startpos\ngo depth 4\nd\n")

	if engine.Threads != 3 {
		t.Fatalf("Threads = %d, want 3", engine.Threads)
	}

	p := position(t, chess.StartingFEN)

	if _, err := ParseMove(&p, bestMove(t, lines)); err != nil {
		t.Fatalf("the threaded search returned an illegal move: %v", err)
	}
}

func TestThreadsOptionRejectsBadValues(t *testing.T) {
	for _, bad := range []string{
		"setoption name Threads value 0",
		"setoption name Threads value -2",
		"setoption name Threads value 1000",
		"setoption name Threads value many",
		"setoption name Threads",
	} {
		lines := session(t, bad+"\nisready\n")

		if len(lines) != 2 || !strings.HasPrefix(lines[0], "info string Threads must be") || lines[1] != "readyok" {
			t.Fatalf("%q: got %v", bad, lines)
		}

		if engine.Threads != 1 {
			t.Fatalf("%q changed Threads to %d", bad, engine.Threads)
		}
	}
}

func TestThreadedInfiniteSearchStops(t *testing.T) {
	l := startLive(t)

	l.send(t, "setoption name Threads value 4")
	l.send(t, "position startpos")
	l.send(t, "go infinite")

	if _, ok := l.out.waitFor("info depth 3 ", 5*time.Second); !ok {
		t.Fatalf("no output from the threaded search: %v", l.out.snapshot())
	}

	l.send(t, "stop")

	if _, ok := l.out.waitFor("bestmove ", 5*time.Second); !ok {
		t.Fatalf("no bestmove after stop: %v", l.out.snapshot())
	}
}

type fakeBook map[string]string

func (f fakeBook) Probe(p *chess.Position) (chess.Move, bool) {
	text, ok := f[p.FEN()]
	if !ok {
		return chess.Move{}, false
	}

	m, err := ParseMove(p, text)

	return m, err == nil
}

func bookSession(t *testing.T, book Book, loadErr error, script string) []string {
	t.Helper()

	restoreEngineState(t)

	var out bytes.Buffer

	e := New(strings.NewReader(script), &out)

	e.LoadBook = func(path string) (Book, error) {
		if loadErr != nil {
			return nil, loadErr
		}

		return book, nil
	}

	if err := e.Run(); err != nil {
		t.Fatal(err)
	}

	var lines []string

	for _, l := range strings.Split(out.String(), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}

	return lines
}

var startBook = fakeBook{chess.StartingFEN: "d2d4"}

func TestBookMoveIsPlayedWithoutSearching(t *testing.T) {
	lines := bookSession(t, startBook, nil, "setoption name BookFile value any.txt\nsetoption name OwnBook value true\nposition startpos\ngo depth 20\n")

	if len(lines) != 1 || lines[0] != "bestmove d2d4" {
		t.Fatalf("want only the book move, got %v", lines)
	}
}

func TestBookIsOffByDefault(t *testing.T) {
	lines := bookSession(t, startBook, nil, "setoption name BookFile value any.txt\nposition startpos\ngo depth 2\n")

	if len(linesWithPrefix(lines, "info depth ")) == 0 {
		t.Fatalf("OwnBook is off, the engine should search: %v", lines)
	}
}

func TestBookMissFallsBackToSearch(t *testing.T) {
	lines := bookSession(t, startBook, nil, "setoption name BookFile value any.txt\nsetoption name OwnBook value true\nposition startpos moves e2e4\ngo depth 2\n")

	if len(linesWithPrefix(lines, "info depth ")) == 0 {
		t.Fatalf("a position outside the book should be searched: %v", lines)
	}

	bestMove(t, lines)
}

func TestBookCanBeSwitchedOff(t *testing.T) {
	script := "setoption name BookFile value any.txt\nsetoption name OwnBook value true\nsetoption name OwnBook value false\nposition startpos\ngo depth 2\n"

	if lines := bookSession(t, startBook, nil, script); len(linesWithPrefix(lines, "info depth ")) == 0 {
		t.Fatalf("OwnBook was switched off: %v", lines)
	}
}

func TestBookIsIgnoredForInfiniteSearches(t *testing.T) {
	l := startLive(t)

	l.send(t, "setoption name OwnBook value true")
	l.send(t, "position startpos")
	l.send(t, "go infinite")

	if _, ok := l.out.waitFor("info depth 2 ", 5*time.Second); !ok {
		t.Fatalf("an infinite search must search: %v", l.out.snapshot())
	}

	l.send(t, "stop")
	l.out.waitFor("bestmove ", 5*time.Second)
}

func TestBookOptionErrors(t *testing.T) {
	lines := session(t, "setoption name OwnBook value maybe\nsetoption name BookFile value x.txt\nisready\n")

	infos := linesWithPrefix(lines, "info string ")
	if len(infos) != 2 || lines[len(lines)-1] != "readyok" {
		t.Fatalf("want two errors then readyok, got %v", lines)
	}

	if !strings.Contains(infos[1], "not available") {
		t.Fatalf("without a loader the engine should say so, got %q", infos[1])
	}

	lines = bookSession(t, nil, errors.New("no such file"), "setoption name BookFile value x.txt\nisready\n")

	if len(lines) != 2 || !strings.Contains(lines[0], "cannot load book: no such file") {
		t.Fatalf("got %v", lines)
	}
}

func TestOwnBookIsCaseInsensitive(t *testing.T) {
	script := "setoption name BookFile value a.txt\nsetoption name ownbook value TRUE\nposition startpos\ngo depth 9\n"

	if lines := bookSession(t, startBook, nil, script); len(lines) != 1 || lines[0] != "bestmove d2d4" {
		t.Fatalf("got %v", lines)
	}
}
