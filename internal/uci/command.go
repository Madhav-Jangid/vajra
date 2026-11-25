package uci

import "time"

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

type Command struct {
	Kind     Kind
	Raw      string
	Position PositionArgs
	Go       GoArgs
	Option   OptionArgs
}

type PositionArgs struct {
	FEN   string
	Moves []string
}

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

func (g GoArgs) HasClock() bool {
	return g.WTime > 0 || g.BTime > 0 || g.WInc > 0 || g.BInc > 0
}

type OptionArgs struct {
	Name  string
	Value string
}
