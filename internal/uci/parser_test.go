package uci

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"vajra/internal/chess"
)

func mustParse(t *testing.T, line string) Command {
	t.Helper()

	cmd, err := ParseCommand(line)
	if err != nil {
		t.Fatalf("ParseCommand(%q): %v", line, err)
	}

	return cmd
}

func TestParseSimpleCommands(t *testing.T) {
	tests := []struct {
		line string
		want Kind
	}{
		{"uci", UCI},
		{"isready", IsReady},
		{"ucinewgame", NewGame},
		{"stop", Stop},
		{"quit", Quit},
		{"d", Display},
		{"display", Display},
		{"debug on", Ignored},
		{"register later", Ignored},
		{"ponderhit", Ignored},
		{"  uci  ", UCI},
		{"isready extra words are ignored", IsReady},
		{"flibbertigibbet", Unknown},
		{"UCI", Unknown},
	}

	for _, tc := range tests {
		t.Run(tc.line, func(t *testing.T) {
			if got := mustParse(t, tc.line).Kind; got != tc.want {
				t.Fatalf("Kind = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseEmptyLines(t *testing.T) {
	for _, line := range []string{"", "   ", "\t", "\r"} {
		if _, err := ParseCommand(line); !errors.Is(err, ErrEmptyCommand) {
			t.Fatalf("ParseCommand(%q) error = %v, want ErrEmptyCommand", line, err)
		}
	}
}

func TestKindString(t *testing.T) {
	for k := Unknown; k <= Ignored; k++ {
		if k.String() == "" {
			t.Fatalf("Kind %d has no name", k)
		}
	}

	if Kind(99).String() != "unknown" {
		t.Fatal("an out of range kind should print as unknown")
	}
}

func TestParsePosition(t *testing.T) {
	const fen = "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"

	tests := []struct {
		name string
		line string
		want PositionArgs
	}{
		{"startpos", "position startpos", PositionArgs{}},
		{"startpos with moves", "position startpos moves e2e4 e7e5 g1f3", PositionArgs{Moves: []string{"e2e4", "e7e5", "g1f3"}}},
		{"startpos with empty moves", "position startpos moves", PositionArgs{}},
		{"fen", "position fen " + fen, PositionArgs{FEN: fen}},
		{"fen with moves", "position fen " + fen + " moves e1g1 e8c8", PositionArgs{FEN: fen, Moves: []string{"e1g1", "e8c8"}}},
		{"fen without move counters", "position fen 8/8/8/8/8/8/8/k6K w - -", PositionArgs{FEN: "8/8/8/8/8/8/8/k6K w - - 0 1"}},
		{"fen with only the halfmove clock", "position fen 8/8/8/8/8/8/8/k6K w - - 7", PositionArgs{FEN: "8/8/8/8/8/8/8/k6K w - - 7 1"}},
		{"fen without counters then moves", "position fen 8/8/8/8/8/8/8/k6K w - - moves h1g1", PositionArgs{FEN: "8/8/8/8/8/8/8/k6K w - - 0 1", Moves: []string{"h1g1"}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := mustParse(t, tc.line)

			if cmd.Kind != Position {
				t.Fatalf("Kind = %v, want Position", cmd.Kind)
			}

			if cmd.Position.FEN != tc.want.FEN {
				t.Fatalf("FEN = %q, want %q", cmd.Position.FEN, tc.want.FEN)
			}

			if len(cmd.Position.Moves) != len(tc.want.Moves) ||
				(len(tc.want.Moves) > 0 && !reflect.DeepEqual(cmd.Position.Moves, tc.want.Moves)) {
				t.Fatalf("Moves = %v, want %v", cmd.Position.Moves, tc.want.Moves)
			}
		})
	}
}

func TestParsePositionErrors(t *testing.T) {
	for _, line := range []string{
		"position",
		"position nonsense",
		"position fen",
		"position fen 8/8/8/8/8/8/8/k6K",
		"position fen 8/8/8/8/8/8/8/k6K w",
		"position fen 8/8/8/8/8/8/8/k6K w - - 0 1 extra",
		"position startpos e2e4",
		"position startpos moves-e2e4",
	} {
		if _, err := ParseCommand(line); !errors.Is(err, ErrBadPosition) {
			t.Fatalf("ParseCommand(%q) error = %v, want ErrBadPosition", line, err)
		}
	}
}

func TestParseGo(t *testing.T) {
	ms := time.Millisecond

	tests := []struct {
		name string
		line string
		want GoArgs
	}{
		{"bare go", "go", GoArgs{}},
		{"depth", "go depth 7", GoArgs{Depth: 7}},
		{"movetime", "go movetime 1500", GoArgs{MoveTime: 1500 * ms}},
		{"infinite", "go infinite", GoArgs{Infinite: true}},
		{"ponder", "go ponder wtime 1000 btime 1000", GoArgs{Ponder: true, WTime: 1000 * ms, BTime: 1000 * ms}},
		{"clocks", "go wtime 60000 btime 59000 winc 1000 binc 2000", GoArgs{WTime: 60000 * ms, BTime: 59000 * ms, WInc: 1000 * ms, BInc: 2000 * ms}},
		{"clocks with movestogo", "go wtime 30000 btime 30000 movestogo 20", GoArgs{WTime: 30000 * ms, BTime: 30000 * ms, MovesToGo: 20}},
		{"nodes", "go nodes 100000", GoArgs{Nodes: 100000}},
		{"mate", "go mate 3", GoArgs{Mate: 3}},
		{"negative clock is clamped", "go wtime -50 btime 1000", GoArgs{BTime: 1000 * ms}},
		{"searchmoves is skipped", "go searchmoves e2e4 d2d4 depth 5", GoArgs{Depth: 5}},
		{"searchmoves at the end", "go depth 4 searchmoves e2e4 d2d4", GoArgs{Depth: 4}},
		{"unknown tokens are ignored", "go depth 3 flibber", GoArgs{Depth: 3}},
		{"everything at once", "go wtime 1000 btime 2000 winc 10 binc 20 movestogo 5 depth 9 movetime 300 nodes 77 mate 2 infinite", GoArgs{
			WTime: 1000 * ms, BTime: 2000 * ms, WInc: 10 * ms, BInc: 20 * ms, MovesToGo: 5, Depth: 9, MoveTime: 300 * ms, Nodes: 77, Mate: 2, Infinite: true,
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := mustParse(t, tc.line)

			if cmd.Kind != Go {
				t.Fatalf("Kind = %v, want Go", cmd.Kind)
			}

			if cmd.Go != tc.want {
				t.Fatalf("got %+v\nwant %+v", cmd.Go, tc.want)
			}
		})
	}
}

func TestParseGoErrors(t *testing.T) {
	for _, line := range []string{
		"go depth",
		"go depth x",
		"go depth -3",
		"go movetime",
		"go movetime soon",
		"go wtime",
		"go btime abc",
		"go winc",
		"go binc 1.5",
		"go movestogo",
		"go nodes many",
		"go mate",
	} {
		if _, err := ParseCommand(line); !errors.Is(err, ErrBadGo) {
			t.Fatalf("ParseCommand(%q) error = %v, want ErrBadGo", line, err)
		}
	}
}

func TestGoArgsHasClock(t *testing.T) {
	if (GoArgs{}).HasClock() || (GoArgs{Depth: 5}).HasClock() || (GoArgs{MoveTime: time.Second}).HasClock() {
		t.Fatal("no clock was given")
	}

	for _, g := range []GoArgs{{WTime: 1}, {BTime: 1}, {WInc: 1}, {BInc: 1}} {
		if !g.HasClock() {
			t.Fatalf("%+v has a clock", g)
		}
	}
}

func TestParseSetOption(t *testing.T) {
	tests := []struct {
		line string
		want OptionArgs
	}{
		{"setoption name Hash value 128", OptionArgs{Name: "Hash", Value: "128"}},
		{"setoption name Move Overhead value 30", OptionArgs{Name: "Move Overhead", Value: "30"}},
		{"setoption name Clear Hash", OptionArgs{Name: "Clear Hash"}},
		{"setoption name Style value Very Aggressive", OptionArgs{Name: "Style", Value: "Very Aggressive"}},
		{"setoption name Hash value", OptionArgs{Name: "Hash"}},
	}

	for _, tc := range tests {
		t.Run(tc.line, func(t *testing.T) {
			cmd := mustParse(t, tc.line)

			if cmd.Kind != SetOption || cmd.Option != tc.want {
				t.Fatalf("got %v %+v, want %+v", cmd.Kind, cmd.Option, tc.want)
			}
		})
	}

	for _, line := range []string{"setoption", "setoption Hash value 1", "setoption name", "setoption name value 5"} {
		if _, err := ParseCommand(line); !errors.Is(err, ErrBadOption) {
			t.Fatalf("ParseCommand(%q) error = %v, want ErrBadOption", line, err)
		}
	}
}

func position(t *testing.T, fen string) chess.Position {
	t.Helper()

	p, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}

	return p
}

func TestParseMove(t *testing.T) {
	const castling = "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1"

	tests := []struct {
		name string
		fen  string
		move string
		from string
		to   string
		flag chess.MoveFlags
	}{
		{"pawn push", chess.StartingFEN, "e2e4", "e2", "e4", chess.DoublePawnPush},
		{"knight", chess.StartingFEN, "g1f3", "g1", "f3", chess.Quiet},
		{"capture", "4k3/8/8/3p4/8/2N5/8/4K3 w - - 0 1", "c3d5", "c3", "d5", chess.Capture},
		{"white king-side castle", castling, "e1g1", "e1", "g1", chess.KingCastle},
		{"white queen-side castle", castling, "e1c1", "e1", "c1", chess.QueenCastle},
		{"black king-side castle", "r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1", "e8g8", "e8", "g8", chess.KingCastle},
		{"black queen-side castle", "r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1", "e8c8", "e8", "c8", chess.QueenCastle},
		{"en passant", "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1", "e5d6", "e5", "d6", chess.Capture | chess.EnPassant},
		{"promotion to queen", "7k/P7/8/8/8/8/8/K7 w - - 0 1", "a7a8q", "a7", "a8", chess.Promotion},
		{"promotion to rook", "7k/P7/8/8/8/8/8/K7 w - - 0 1", "a7a8r", "a7", "a8", chess.Promotion},
		{"promotion to bishop", "7k/P7/8/8/8/8/8/K7 w - - 0 1", "a7a8b", "a7", "a8", chess.Promotion},
		{"promotion to knight", "7k/P7/8/8/8/8/8/K7 w - - 0 1", "a7a8n", "a7", "a8", chess.Promotion},
		{"upper case promotion letter", "7k/P7/8/8/8/8/8/K7 w - - 0 1", "a7a8Q", "a7", "a8", chess.Promotion},
		{"black promotion", "k7/8/8/8/8/8/p7/7K b - - 0 1", "a2a1n", "a2", "a1", chess.Promotion},
		{"capture promotion", "1n5k/P7/8/8/8/8/8/K7 w - - 0 1", "a7b8q", "a7", "b8", chess.Capture | chess.Promotion},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := position(t, tc.fen)

			m, err := ParseMove(&p, tc.move)
			if err != nil {
				t.Fatalf("ParseMove(%q): %v", tc.move, err)
			}

			from, _ := parseSquare(tc.from)
			to, _ := parseSquare(tc.to)

			if m.From != from || m.To != to || m.Flags != tc.flag {
				t.Fatalf("got %+v, want %s-%s with flags %v", m, tc.from, tc.to, tc.flag)
			}
		})
	}
}

func TestParseMoveChoosesTheRequestedPromotion(t *testing.T) {
	want := map[string]chess.PieceType{
		"a7a8q": chess.WhiteQueen,
		"a7a8r": chess.WhiteRook,
		"a7a8b": chess.WhiteBishop,
		"a7a8n": chess.WhiteKnight,
	}

	for text, piece := range want {
		p := position(t, "7k/P7/8/8/8/8/8/K7 w - - 0 1")

		m, err := ParseMove(&p, text)
		if err != nil || m.Promotion != piece {
			t.Fatalf("%s: got %+v, %v, want promotion to %v", text, m, err, piece)
		}
	}
}

func TestParseMoveErrors(t *testing.T) {
	tests := []struct {
		name string
		fen  string
		move string
	}{
		{"empty", chess.StartingFEN, ""},
		{"too short", chess.StartingFEN, "e2e"},
		{"too long", chess.StartingFEN, "e2e4qq"},
		{"bad source file", chess.StartingFEN, "z2e4"},
		{"bad source rank", chess.StartingFEN, "e9e4"},
		{"bad target file", chess.StartingFEN, "e2i4"},
		{"bad target rank", chess.StartingFEN, "e2e0"},
		{"empty source square", chess.StartingFEN, "e4e5"},
		{"opponent's piece", chess.StartingFEN, "e7e5"},
		{"illegal pawn move", chess.StartingFEN, "e2e5"},
		{"illegal knight move", chess.StartingFEN, "g1g3"},
		{"king into attack", "4k3/8/8/8/8/8/r7/4K3 w - - 0 1", "e1e2"},
		{"promotion without a piece", "7k/P7/8/8/8/8/8/K7 w - - 0 1", "a7a8"},
		{"bad promotion piece", "7k/P7/8/8/8/8/8/K7 w - - 0 1", "a7a8k"},
		{"promotion piece on an ordinary move", chess.StartingFEN, "e2e4q"},
		{"null move", chess.StartingFEN, "0000"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := position(t, tc.fen)
			before := p

			if _, err := ParseMove(&p, tc.move); !errors.Is(err, ErrBadMove) {
				t.Fatalf("ParseMove(%q) error = %v, want ErrBadMove", tc.move, err)
			}

			if p != before {
				t.Fatal("a failed ParseMove must not change the position")
			}
		})
	}
}

func TestMoveTextRoundTripsForEveryLegalMove(t *testing.T) {
	fens := []string{
		chess.StartingFEN,
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
		"r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1",
		"r2q1rk1/pP1p2pp/Q4n2/bbp1p3/Np6/1B3NBn/pPPP1PPP/R3K2R b KQ - 0 1",
		"rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
		"4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1",
	}

	for _, fen := range fens {
		p := position(t, fen)
		seen := map[string]bool{}
		count := 0

		for sq := 0; sq < 64; sq++ {
			piece := p.Board.PieceAt(sq)
			if piece.Type == chess.Empty || piece.Color != p.SideToMove {
				continue
			}

			for _, m := range p.LegalMoves(sq) {
				text := FormatMove(m)

				if seen[text] {
					t.Fatalf("%s: two moves are both written %q", fen, text)
				}

				seen[text] = true
				count++

				back, err := ParseMove(&p, text)
				if err != nil || back != m {
					t.Fatalf("%s: %q does not round trip: got %+v, %v want %+v", fen, text, back, err, m)
				}
			}
		}

		if count == 0 {
			t.Fatalf("%s: no legal moves, the test is broken", fen)
		}
	}
}

func TestFormatMove(t *testing.T) {
	tests := []struct {
		move chess.Move
		want string
	}{
		{chess.Move{From: 12, To: 28, Piece: chess.WhitePawn}, "e2e4"},
		{chess.Move{From: 0, To: 63, Piece: chess.WhiteQueen}, "a1h8"},
		{chess.Move{From: 48, To: 56, Piece: chess.WhitePawn, Promotion: chess.WhiteQueen, Flags: chess.Promotion}, "a7a8q"},
		{chess.Move{From: 48, To: 56, Piece: chess.WhitePawn, Promotion: chess.WhiteRook, Flags: chess.Promotion}, "a7a8r"},
		{chess.Move{From: 48, To: 56, Piece: chess.WhitePawn, Promotion: chess.WhiteBishop, Flags: chess.Promotion}, "a7a8b"},
		{chess.Move{From: 48, To: 56, Piece: chess.WhitePawn, Promotion: chess.WhiteKnight, Flags: chess.Promotion}, "a7a8n"},
		{chess.Move{From: 8, To: 0, Piece: chess.BlackPawn, Promotion: chess.BlackKnight, Flags: chess.Promotion}, "a2a1n"},
		{chess.Move{}, "0000"},
	}

	for _, tc := range tests {
		if got := FormatMove(tc.move); got != tc.want {
			t.Fatalf("FormatMove(%+v) = %q, want %q", tc.move, got, tc.want)
		}
	}
}

func TestBudget(t *testing.T) {
	ms := time.Millisecond
	overhead := 50 * ms

	tests := []struct {
		name      string
		g         GoArgs
		side      chess.Color
		wantLimit time.Duration
		wantDepth int
	}{
		{"depth only", GoArgs{Depth: 6}, chess.White, 0, 6},
		{"depth is capped", GoArgs{Depth: 500}, chess.White, 0, MaxSearchDepth},
		{"bare go searches until stopped", GoArgs{}, chess.White, 0, MaxSearchDepth},
		{"infinite", GoArgs{Infinite: true, WTime: 1000}, chess.White, 0, MaxSearchDepth},
		{"ponder", GoArgs{Ponder: true, WTime: 1000}, chess.White, 0, MaxSearchDepth},
		{"movetime keeps the overhead back", GoArgs{MoveTime: 1000 * ms}, chess.White, 950 * ms, MaxSearchDepth},
		{"movetime shorter than the overhead", GoArgs{MoveTime: 10 * ms}, chess.White, ms, MaxSearchDepth},
		{"movetime with depth", GoArgs{MoveTime: 500 * ms, Depth: 8}, chess.Black, 450 * ms, 8},
		{"white uses its own clock", GoArgs{WTime: 31000 * ms, BTime: 3000 * ms}, chess.White, (31000*ms - overhead) / 30, MaxSearchDepth},
		{"black uses its own clock", GoArgs{WTime: 31000 * ms, BTime: 3000 * ms}, chess.Black, (3000*ms - overhead) / 30, MaxSearchDepth},
		{"increment adds three quarters", GoArgs{WTime: 31000 * ms, WInc: 2000 * ms}, chess.White, (31000*ms-overhead)/30 + 1500*ms, MaxSearchDepth},
		{"movestogo spreads the clock", GoArgs{WTime: 10050 * ms, MovesToGo: 10}, chess.White, 1000 * ms, MaxSearchDepth},
		{"never more than the time left", GoArgs{WTime: 1000 * ms, WInc: 30000 * ms}, chess.White, 1000*ms - overhead, MaxSearchDepth},
		{"nearly out of time", GoArgs{WTime: 20 * ms}, chess.White, ms, MaxSearchDepth},
		{"no time at all", GoArgs{BTime: 1, WTime: 0}, chess.White, ms, MaxSearchDepth},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			limit, depth := Budget(tc.g, tc.side, overhead)

			if limit != tc.wantLimit || depth != tc.wantDepth {
				t.Fatalf("Budget = %v, %d; want %v, %d", limit, depth, tc.wantLimit, tc.wantDepth)
			}
		})
	}
}
