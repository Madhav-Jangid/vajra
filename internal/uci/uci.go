package uci

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"vajra/internal/chess"
	"vajra/internal/engine"
)

const (
	Name   = "Vajra"
	Author = "Madhav Jangid"

	MaxSearchDepth = 64

	defaultOverhead = 50 * time.Millisecond
	defaultHashMB   = 64
	maxHashMB       = 1024
	maxOverheadMS   = 5000
	maxThreads      = 64

	movesToPlan = 30
)

func Budget(g GoArgs, side chess.Color, overhead time.Duration) (limit time.Duration, depth int) {
	depth = MaxSearchDepth

	if g.Depth > 0 {
		depth = min(g.Depth, MaxSearchDepth)
	}

	if g.Infinite || g.Ponder {
		return 0, depth
	}

	if g.MoveTime > 0 {
		return max(g.MoveTime-overhead, time.Millisecond), depth
	}

	remaining, inc := g.BTime, g.BInc
	if side == chess.White {
		remaining, inc = g.WTime, g.WInc
	}

	if !g.HasClock() {
		return 0, depth
	}

	spare := max(remaining-overhead, time.Millisecond)

	moves := movesToPlan
	if g.MovesToGo > 0 {
		moves = g.MovesToGo
	}

	limit = spare/time.Duration(moves) + inc*3/4

	return max(min(limit, spare), time.Millisecond), depth
}

type Book interface {
	Probe(p *chess.Position) (chess.Move, bool)
}

type Engine struct {
	in  io.Reader
	out io.Writer

	LoadBook func(path string) (Book, error)

	book    Book
	useBook bool

	outMu sync.Mutex

	pos      chess.Position
	history  []uint64
	overhead time.Duration

	search *running
}

type running struct {
	done     chan struct{}
	stop     chan struct{}
	infinite bool
	once     sync.Once
}

func (r *running) halt() {
	r.once.Do(func() {
		close(r.stop)
		engine.RequestStop()
	})
}

func New(in io.Reader, out io.Writer) *Engine {
	e := &Engine{in: in, out: out, overhead: defaultOverhead}
	engine.PreserveTT = true

	e.newGame()

	return e
}

func (e *Engine) send(format string, args ...any) {
	e.outMu.Lock()
	defer e.outMu.Unlock()

	fmt.Fprintf(e.out, format+"\n", args...)
}

func (e *Engine) newGame() {
	e.pos, _ = chess.ParseFEN(chess.StartingFEN)
	e.history = nil

	engine.TT.Clear()
}

func (e *Engine) Run() error {
	scanner := bufio.NewScanner(e.in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		cmd, err := ParseCommand(scanner.Text())
		if err != nil {
			if err != ErrEmptyCommand {
				e.send("info string %v", err)
			}

			continue
		}

		if quit := e.handle(cmd); quit {
			e.stopSearch()

			return nil
		}
	}

	if e.search != nil && e.search.infinite {
		e.stopSearch()
	}

	e.waitSearch()

	return scanner.Err()
}

func (e *Engine) handle(cmd Command) (quit bool) {
	switch cmd.Kind {
	case UCI:
		e.send("id name %s", Name)
		e.send("id author %s", Author)
		e.send("option name Hash type spin default %d min 1 max %d", defaultHashMB, maxHashMB)
		e.send("option name Threads type spin default 1 min 1 max %d", maxThreads)
		e.send("option name OwnBook type check default false")
		e.send("option name BookFile type string default <empty>")
		e.send("option name Move Overhead type spin default %d min 0 max %d", defaultOverhead.Milliseconds(), maxOverheadMS)
		e.send("uciok")
	case IsReady:
		e.send("readyok")
	case NewGame:
		e.settle()
		e.newGame()
	case Position:
		e.settle()

		if err := e.setPosition(cmd.Position); err != nil {
			e.send("info string %v", err)
		}
	case Go:
		e.startSearch(cmd.Go)
	case Stop:
		e.stopSearch()
	case SetOption:
		e.setOption(cmd.Option)
	case Display:
		e.send("info string fen %s", e.pos.FEN())
	case Quit:
		return true
	}

	return false
}

func (e *Engine) setPosition(args PositionArgs) error {
	fen := chess.StartingFEN
	if args.FEN != "" {
		fen = args.FEN
	}

	pos, err := chess.ParseFEN(fen)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBadPosition, err)
	}

	var history []uint64

	for _, text := range args.Moves {
		move, err := ParseMove(&pos, text)
		if err != nil {
			return err
		}

		history = append(history, engine.HashPosition(&pos))

		pos.MakeMove(move)
	}

	e.pos, e.history = pos, history

	return nil
}

func (e *Engine) setOption(o OptionArgs) {
	switch strings.ToLower(o.Name) {
	case "hash":
		mb, err := strconv.Atoi(o.Value)
		if err != nil || mb < 1 || mb > maxHashMB {
			e.send("info string Hash must be a number from 1 to %d", maxHashMB)

			return
		}

		e.settle()

		engine.TT = engine.NewTranspositionTable(mb)
	case "threads":
		n, err := strconv.Atoi(o.Value)
		if err != nil || n < 1 || n > maxThreads {
			e.send("info string Threads must be a number from 1 to %d", maxThreads)

			return
		}

		e.settle()

		engine.Threads = n
	case "ownbook":
		switch strings.ToLower(o.Value) {
		case "true":
			e.useBook = true
		case "false":
			e.useBook = false
		default:
			e.send("info string OwnBook must be true or false")
		}
	case "bookfile":
		if e.LoadBook == nil {
			e.send("info string opening books are not available")

			return
		}

		book, err := e.LoadBook(o.Value)
		if err != nil {
			e.send("info string cannot load book: %v", err)

			return
		}

		e.book = book
	case "move overhead":
		ms, err := strconv.Atoi(o.Value)
		if err != nil || ms < 0 || ms > maxOverheadMS {
			e.send("info string Move Overhead must be a number from 0 to %d", maxOverheadMS)

			return
		}

		e.overhead = time.Duration(ms) * time.Millisecond
	default:
		e.send("info string unknown option %q", o.Name)
	}
}

func (e *Engine) startSearch(g GoArgs) {
	e.settle()

	if e.useBook && e.book != nil && !g.Infinite && !g.Ponder {
		if move, ok := e.book.Probe(&e.pos); ok {
			e.send("bestmove %s", FormatMove(move))

			return
		}
	}

	pos := e.pos
	limit, depth := Budget(g, pos.SideToMove, e.overhead)

	r := &running{
		done:     make(chan struct{}),
		stop:     make(chan struct{}),
		infinite: limit == 0 && g.Depth == 0 && g.Nodes == 0,
	}

	e.search = r

	engine.ClearStop()
	engine.NodeLimit = g.Nodes

	engine.GameHistory = append([]uint64(nil), e.history...)

	go func() {
		defer close(r.done)

		best, ok := engine.IterativeSearchWithInfo(&pos, depth, limit, e.reportInfo())

		if r.infinite {
			<-r.stop
		}

		if !ok {
			e.send("bestmove %s", nullMove)

			return
		}

		e.send("bestmove %s", FormatMove(best))
	}()
}

func (e *Engine) reportInfo() func(engine.SearchInfo) {
	return func(info engine.SearchInfo) {
		score := fmt.Sprintf("cp %d", info.Score)

		if mate, ok := engine.MateIn(info.Score); ok {
			score = fmt.Sprintf("mate %d", mate)
		}

		ms := info.Elapsed.Milliseconds()

		line := fmt.Sprintf("info depth %d score %s nodes %d hashfull %d", info.Depth, score, info.Nodes, info.HashFull)

		if ms > 0 {
			line += fmt.Sprintf(" nps %d", int64(info.Nodes)*1000/ms)
		}

		pvText := FormatMove(info.Best)
		if len(info.PV) > 0 {
			parts := make([]string, len(info.PV))
			for i, m := range info.PV {
				parts[i] = FormatMove(m)
			}
			pvText = strings.Join(parts, " ")
		}

		e.send("%s time %d pv %s", line, ms, pvText)
	}
}

func (e *Engine) stopSearch() {
	if e.search == nil {
		return
	}

	e.search.halt()

	<-e.search.done

	e.search = nil
}

func (e *Engine) settle() {
	if e.search != nil && e.search.infinite {
		e.stopSearch()

		return
	}

	e.waitSearch()
}

func (e *Engine) waitSearch() {
	if e.search == nil {
		return
	}

	<-e.search.done

	e.search = nil
}
