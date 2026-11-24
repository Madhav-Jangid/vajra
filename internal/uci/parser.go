package uci

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"vajra/internal/chess"
)

var (
	ErrEmptyCommand = errors.New("empty command")
	ErrBadPosition  = errors.New("bad position command")
	ErrBadGo        = errors.New("bad go command")
	ErrBadOption    = errors.New("bad setoption command")
	ErrBadMove      = errors.New("bad move")
)

// ParseCommand parses one line from the GUI. Unknown commands are returned with
// Kind Unknown and no error, because UCI says unknown input must be ignored.
func ParseCommand(line string) (Command, error) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return Command{}, ErrEmptyCommand
	}

	cmd := Command{Raw: strings.TrimSpace(line)}

	switch fields[0] {
	case "uci":
		cmd.Kind = UCI
	case "isready":
		cmd.Kind = IsReady
	case "ucinewgame":
		cmd.Kind = NewGame
	case "stop":
		cmd.Kind = Stop
	case "quit":
		cmd.Kind = Quit
	case "d", "display":
		cmd.Kind = Display
	case "debug", "register", "ponderhit":
		cmd.Kind = Ignored
	case "position":
		args, err := parsePosition(fields[1:])
		if err != nil {
			return Command{}, err
		}

		cmd.Kind, cmd.Position = Position, args
	case "go":
		args, err := parseGo(fields[1:])
		if err != nil {
			return Command{}, err
		}

		cmd.Kind, cmd.Go = Go, args
	case "setoption":
		args, err := parseOption(fields[1:])
		if err != nil {
			return Command{}, err
		}

		cmd.Kind, cmd.Option = SetOption, args
	default:
		cmd.Kind = Unknown
	}

	return cmd, nil
}

func parsePosition(tokens []string) (PositionArgs, error) {
	if len(tokens) == 0 {
		return PositionArgs{}, fmt.Errorf("%w: missing startpos or fen", ErrBadPosition)
	}

	var args PositionArgs

	rest := tokens[1:]

	switch tokens[0] {
	case "startpos":
	case "fen":
		end := len(rest)

		for i, tok := range rest {
			if tok == "moves" {
				end = i

				break
			}
		}

		fen := rest[:end]

		if len(fen) < 4 || len(fen) > 6 {
			return PositionArgs{}, fmt.Errorf("%w: a FEN needs 4 to 6 fields, got %d", ErrBadPosition, len(fen))
		}

		// Some GUIs leave out the move counters.
		for len(fen) < 6 {
			if len(fen) == 4 {
				fen = append(fen[:4:4], "0")
			} else {
				fen = append(fen[:5:5], "1")
			}
		}

		args.FEN = strings.Join(fen, " ")
		rest = rest[end:]
	default:
		return PositionArgs{}, fmt.Errorf("%w: expected startpos or fen, got %q", ErrBadPosition, tokens[0])
	}

	if len(rest) > 0 {
		if rest[0] != "moves" {
			return PositionArgs{}, fmt.Errorf("%w: unexpected %q", ErrBadPosition, rest[0])
		}

		args.Moves = append([]string(nil), rest[1:]...)
	}

	return args, nil
}

func parseGo(tokens []string) (GoArgs, error) {
	var g GoArgs

	number := func(i int) (int64, error) {
		if i+1 >= len(tokens) {
			return 0, fmt.Errorf("%w: %q needs a value", ErrBadGo, tokens[i])
		}

		n, err := strconv.ParseInt(tokens[i+1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %q is not a number", ErrBadGo, tokens[i+1])
		}

		return n, nil
	}

	millis := func(i int) (time.Duration, error) {
		n, err := number(i)
		if err != nil {
			return 0, err
		}

		if n < 0 {
			n = 0
		}

		return time.Duration(n) * time.Millisecond, nil
	}

	keywords := map[string]bool{
		"searchmoves": true, "ponder": true, "wtime": true, "btime": true, "winc": true,
		"binc": true, "movestogo": true, "depth": true, "nodes": true, "mate": true,
		"movetime": true, "infinite": true,
	}

	for i := 0; i < len(tokens); i++ {
		var err error

		switch tokens[i] {
		case "infinite":
			g.Infinite = true
		case "ponder":
			g.Ponder = true
		case "searchmoves":
			for i+1 < len(tokens) && !keywords[tokens[i+1]] {
				i++
			}
		case "depth":
			var n int64

			n, err = number(i)
			g.Depth = int(n)
			i++
		case "nodes":
			var n int64

			n, err = number(i)
			g.Nodes = uint64(max(n, 0))
			i++
		case "movestogo":
			var n int64

			n, err = number(i)
			g.MovesToGo = int(n)
			i++
		case "mate":
			var n int64

			n, err = number(i)
			g.Mate = int(n)
			i++
		case "movetime":
			g.MoveTime, err = millis(i)
			i++
		case "wtime":
			g.WTime, err = millis(i)
			i++
		case "btime":
			g.BTime, err = millis(i)
			i++
		case "winc":
			g.WInc, err = millis(i)
			i++
		case "binc":
			g.BInc, err = millis(i)
			i++
		}

		if err != nil {
			return GoArgs{}, err
		}
	}

	if g.Depth < 0 {
		return GoArgs{}, fmt.Errorf("%w: depth must not be negative", ErrBadGo)
	}

	return g, nil
}

func parseOption(tokens []string) (OptionArgs, error) {
	if len(tokens) == 0 || tokens[0] != "name" {
		return OptionArgs{}, fmt.Errorf("%w: expected \"name\"", ErrBadOption)
	}

	var name, value []string

	target := &name

	for _, tok := range tokens[1:] {
		if tok == "value" && target == &name {
			target = &value

			continue
		}

		*target = append(*target, tok)
	}

	if len(name) == 0 {
		return OptionArgs{}, fmt.Errorf("%w: missing option name", ErrBadOption)
	}

	return OptionArgs{Name: strings.Join(name, " "), Value: strings.Join(value, " ")}, nil
}

const nullMove = "0000"

// FormatMove writes a move in UCI coordinate notation: "e2e4", "e7e8q".
func FormatMove(m chess.Move) string {
	if m == (chess.Move{}) {
		return nullMove
	}

	s := squareName(m.From) + squareName(m.To)

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

func squareName(sq int) string {
	return string(rune('a'+sq%8)) + string(rune('1'+sq/8))
}

func parseSquare(s string) (int, bool) {
	if len(s) != 2 || s[0] < 'a' || s[0] > 'h' || s[1] < '1' || s[1] > '8' {
		return 0, false
	}

	return int(s[0]-'a') + int(s[1]-'1')*8, true
}

// ParseMove finds the legal move in p that matches a UCI coordinate string.
func ParseMove(p *chess.Position, s string) (chess.Move, error) {
	if len(s) != 4 && len(s) != 5 {
		return chess.Move{}, fmt.Errorf("%w: %q", ErrBadMove, s)
	}

	from, ok := parseSquare(s[0:2])
	if !ok {
		return chess.Move{}, fmt.Errorf("%w: %q has a bad source square", ErrBadMove, s)
	}

	to, ok := parseSquare(s[2:4])
	if !ok {
		return chess.Move{}, fmt.Errorf("%w: %q has a bad target square", ErrBadMove, s)
	}

	promo := byte(0)

	if len(s) == 5 {
		promo = s[4]

		if promo >= 'A' && promo <= 'Z' {
			promo += 'a' - 'A'
		}

		if !strings.ContainsRune("qrbn", rune(promo)) {
			return chess.Move{}, fmt.Errorf("%w: %q has a bad promotion piece", ErrBadMove, s)
		}
	}

	piece := p.Board.PieceAt(from)
	if piece.Type == chess.Empty || piece.Color != p.SideToMove {
		return chess.Move{}, fmt.Errorf("%w: %q moves a piece that is not on turn", ErrBadMove, s)
	}

	for _, m := range p.LegalMoves(from) {
		if m.To != to {
			continue
		}

		isPromotion := m.Flags&chess.Promotion != 0

		if isPromotion != (promo != 0) {
			continue
		}

		if isPromotion && !strings.HasSuffix(FormatMove(m), string(rune(promo))) {
			continue
		}

		return m, nil
	}

	return chess.Move{}, fmt.Errorf("%w: %q is not legal here", ErrBadMove, s)
}
