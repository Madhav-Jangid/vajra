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
	// Name and Author are sent in response to "uci".
	Name   = "Vajra"
	Author = "Madhav Jangid"

	// MaxSearchDepth bounds a search that has no depth limit of its own.
	MaxSearchDepth = 64

	defaultOverhead = 50 * time.Millisecond
	defaultHashMB   = 64
	maxHashMB       = 1024
	maxOverheadMS   = 5000

	// movesToPlan is how many more moves the clock is spread over when the
	// GUI does not say.
	movesToPlan = 30
)

// Budget turns the parameters of a "go" command into a time limit and a depth
// limit for the side to move. A zero limit means "no time limit".
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

// Engine runs the UCI protocol over a reader and a writer.
type Engine struct {
	in  io.Reader
	out io.Writer

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

// New returns an engine that reads commands from in and writes replies to out.
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

// Run reads commands until "quit" or the end of the input. A running search is
// finished (or stopped, if it has no end) before Run returns.
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

// setPosition replaces the position only if every move in the command is legal.
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

		// In infinite mode the GUI must stop the search before the engine answers.
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

// stopSearch ends the running search, if any, and waits for its bestmove.
func (e *Engine) stopSearch() {
	if e.search == nil {
		return
	}

	e.search.halt()

	<-e.search.done

	e.search = nil
}

// settle gets the engine idle before a command that changes its state. A search
// with a limit is allowed to finish and send its bestmove. A search without one
// would never end, so it is stopped.
func (e *Engine) settle() {
	if e.search != nil && e.search.infinite {
		e.stopSearch()

		return
	}

	e.waitSearch()
}

// waitSearch lets the running search finish on its own.
func (e *Engine) waitSearch() {
	if e.search == nil {
		return
	}

	<-e.search.done

	e.search = nil
}
