package uci

import "time"

// Kind identifies a UCI command.
type Kind int

const (
	Unknown Kind = iota
	UCI
	IsReady
	NewGame
	Position
	Go
	Stop
	Quit
	SetOption
	Display
	Ignored
)

func (k Kind) String() string {
	switch k {
	case UCI:
		return "uci"
	case IsReady:
		return "isready"
	case NewGame:
		return "ucinewgame"
	case Position:
		return "position"
	case Go:
		return "go"
	case Stop:
		return "stop"
	case Quit:
		return "quit"
	case SetOption:
		return "setoption"
	case Display:
		return "d"
	case Ignored:
		return "ignored"
	}

	return "unknown"
}

// Command is one parsed line from the GUI.
type Command struct {
	Kind     Kind
	Raw      string
	Position PositionArgs
	Go       GoArgs
	Option   OptionArgs
}

// PositionArgs describes "position startpos|fen <fen> [moves ...]".
type PositionArgs struct {
	// FEN is empty for startpos.
	FEN   string
	Moves []string
}

// GoArgs holds the parameters of a "go" command. Zero values mean "not given".
type GoArgs struct {
	Depth     int
	Nodes     uint64
	MoveTime  time.Duration
	WTime     time.Duration
	BTime     time.Duration
	WInc      time.Duration
	BInc      time.Duration
	MovesToGo int
	Mate      int
	Infinite  bool
	Ponder    bool
}

// HasClock reports whether the GUI sent clock information for either side.
func (g GoArgs) HasClock() bool {
	return g.WTime > 0 || g.BTime > 0 || g.WInc > 0 || g.BInc > 0
}

// OptionArgs describes "setoption name <name> [value <value>]".
type OptionArgs struct {
	Name  string
	Value string
}
