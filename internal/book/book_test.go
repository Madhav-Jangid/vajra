package book

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vajra/internal/chess"
	"vajra/internal/uci"
)

func start(t *testing.T) chess.Position {
	t.Helper()

	p, err := chess.ParseFEN(chess.StartingFEN)
	if err != nil {
		t.Fatal(err)
	}

	return p
}

func parse(t *testing.T, text string) *Book {
	t.Helper()

	b, err := Parse(strings.NewReader(text), 1)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	return b
}

func play(t *testing.T, p *chess.Position, moves ...string) {
	t.Helper()

	for _, text := range moves {
		m, err := uci.ParseMove(p, text)
		if err != nil {
			t.Fatal(err)
		}

		p.MakeMove(m)
	}
}

func TestProbeReturnsABookMove(t *testing.T) {
	b := parse(t, "e2e4 e7e5 g1f3 b8c6\n")
	p := start(t)

	m, ok := b.Probe(&p)
	if !ok || uci.FormatMove(m) != "e2e4" {
		t.Fatalf("got %s (found=%v), want e2e4", uci.FormatMove(m), ok)
	}

	play(t, &p, "e2e4", "e7e5")

	if m, ok := b.Probe(&p); !ok || uci.FormatMove(m) != "g1f3" {
		t.Fatalf("got %s (found=%v), want g1f3", uci.FormatMove(m), ok)
	}
}

func TestProbeMissesOutsideTheBook(t *testing.T) {
	b := parse(t, "e2e4 e7e5\n")
	p := start(t)

	play(t, &p, "d2d4")

	if _, ok := b.Probe(&p); ok {
		t.Fatal("a position that is not in the book must miss")
	}

	play(t, &p, "d7d5", "c2c4")

	if _, ok := b.Probe(&p); ok {
		t.Fatal("a deeper unknown position must miss")
	}
}

func TestLastMoveOfALineHasNoReply(t *testing.T) {
	b := parse(t, "e2e4 e7e5\n")
	p := start(t)

	play(t, &p, "e2e4", "e7e5")

	if _, ok := b.Probe(&p); ok {
		t.Fatal("the book ends after e7e5")
	}
}

func TestTranspositionsShareAnEntry(t *testing.T) {
	b := parse(t, "g1f3 g8f6 b1c3 b8c6\nb1c3 b8c6 g1f3 g8f6 e2e4\n")
	p := start(t)

	play(t, &p, "g1f3", "g8f6", "b1c3", "b8c6")

	m, ok := b.Probe(&p)
	if !ok || uci.FormatMove(m) != "e2e4" {
		t.Fatalf("the transposed line should give e2e4, got %s (found=%v)", uci.FormatMove(m), ok)
	}
}

func TestWeightsFollowHowOftenAMoveIsPlayed(t *testing.T) {
	b := parse(t, "e2e4\ne2e4\ne2e4\ne2e4\nd2d4\n")
	p := start(t)

	counts := map[string]int{}

	for i := 0; i < 2000; i++ {
		m, ok := b.Probe(&p)
		if !ok {
			t.Fatal("start position must be in the book")
		}

		counts[uci.FormatMove(m)]++
	}

	if counts["e2e4"] < 1400 || counts["e2e4"] > 1800 || counts["d2d4"] < 200 {
		t.Fatalf("weights are off: %v (want about 4:1)", counts)
	}

	if len(counts) != 2 {
		t.Fatalf("only the two book moves may appear, got %v", counts)
	}
}

func TestProbeIsRepeatableWithTheSameSeed(t *testing.T) {
	text := "e2e4\nd2d4\ng1f3\nc2c4\n"
	a, _ := Parse(strings.NewReader(text), 7)
	c, _ := Parse(strings.NewReader(text), 7)
	p := start(t)

	for i := 0; i < 50; i++ {
		m1, _ := a.Probe(&p)
		m2, _ := c.Probe(&p)

		if m1 != m2 {
			t.Fatalf("draw %d differs for the same seed", i)
		}
	}
}

func TestBookMovesAreAlwaysLegal(t *testing.T) {
	b := parse(t, "e2e4 c7c5 g1f3 d7d6 d2d4 c5d4 f3d4 g8f6 b1c3 a7a6\ne2e4 e7e5 g1f3 b8c6 f1b5 a7a6 b5a4 g8f6 e1g1 f8e7\n")
	p := start(t)

	for ply := 0; ply < 12; ply++ {
		m, ok := b.Probe(&p)
		if !ok {
			break
		}

		if _, err := uci.ParseMove(&p, uci.FormatMove(m)); err != nil {
			t.Fatalf("ply %d: book move %s is illegal: %v", ply, uci.FormatMove(m), err)
		}

		p.MakeMove(m)
	}
}

func TestBlankLinesAreSkipped(t *testing.T) {
	b := parse(t, "\n\n  \ne2e4 e7e5\n\n")

	if b.Len() != 2 {
		t.Fatalf("got %d positions, want 2", b.Len())
	}
}

func TestEmptyBook(t *testing.T) {
	b := parse(t, "")
	p := start(t)

	if b.Len() != 0 {
		t.Fatalf("got %d positions", b.Len())
	}

	if _, ok := b.Probe(&p); ok {
		t.Fatal("an empty book never has a move")
	}
}

func TestBadLinesAreReportedWithTheirNumber(t *testing.T) {
	tests := []struct{ name, text, want string }{
		{"illegal move", "e2e4 e7e5\ne2e5\n", "line 2"},
		{"move for the wrong side", "e2e4 e2e4\n", "line 1"},
		{"garbage", "e2e4\nhello\n", "line 2"},
		{"bad promotion", "e2e4 e7e5 a2a4 a7a5 a4a5 h7h6 a5a6 h6h5 a6b7 h5h4 b7a8k\n", "line 1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(strings.NewReader(tc.text), 1); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want one mentioning %q", err, tc.want)
			}
		})
	}
}

func TestPromotionsAndCastlingInALine(t *testing.T) {
	b := parse(t, "e2e4 e7e5 g1f3 b8c6 f1c4 g8f6 e1g1\n")
	p := start(t)

	play(t, &p, "e2e4", "e7e5", "g1f3", "b8c6", "f1c4", "g8f6")

	m, ok := b.Probe(&p)
	if !ok || uci.FormatMove(m) != "e1g1" || m.Flags&chess.KingCastle == 0 {
		t.Fatalf("got %s %+v (found=%v), want castling", uci.FormatMove(m), m, ok)
	}
}

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.txt")

	if err := os.WriteFile(path, []byte("e2e4 e7e5\nd2d4 d7d5\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b, err := Load(path, 1)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if b.Len() != 3 {
		t.Fatalf("got %d positions, want 3", b.Len())
	}

	if _, err := Load(filepath.Join(t.TempDir(), "missing.txt"), 1); err == nil {
		t.Fatal("a missing file must be an error")
	}
}
