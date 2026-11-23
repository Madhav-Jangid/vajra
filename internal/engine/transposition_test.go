package engine

import (
	"testing"

	"vajra/internal/chess"
)

func TestZobristHashChangesAfterMove(t *testing.T) {
	p, err := chess.ParseFEN(chess.StartingFEN)
	if err != nil {
		t.Fatalf("failed to parse starting FEN: %v", err)
	}

	original := HashPosition(&p)

	// e2 -> e3/e4
	moves := p.LegalMoves(12)

	if len(moves) == 0 {
		t.Fatal("expected legal pawn moves from e2")
	}

	move := moves[0]

	undo := p.MakeMove(move)

	after := HashPosition(&p)

	if original == after {
		t.Fatal("hash did not change after move")
	}

	p.UnmakeMove(move, undo)

	restored := HashPosition(&p)

	if original != restored {
		t.Fatalf(
			"hash not restored after unmake: original=%x restored=%x",
			original,
			restored,
		)
	}
}

func TestZobristSamePositionSameHash(t *testing.T) {
	p1, err := chess.ParseFEN(chess.StartingFEN)
	if err != nil {
		t.Fatalf("failed to parse starting FEN for p1: %v", err)
	}

	p2, err := chess.ParseFEN(chess.StartingFEN)
	if err != nil {
		t.Fatalf("failed to parse starting FEN for p2: %v", err)
	}

	h1 := HashPosition(&p1)
	h2 := HashPosition(&p2)

	if h1 != h2 {
		t.Fatalf(
			"same position produced different hashes: %x != %x",
			h1,
			h2,
		)
	}
}

func TestZobristSideToMoveChangesHash(t *testing.T) {
	whitePosition, err := chess.ParseFEN(
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
	)
	if err != nil {
		t.Fatalf("failed to parse white-to-move position: %v", err)
	}

	blackPosition, err := chess.ParseFEN(
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 1",
	)
	if err != nil {
		t.Fatalf("failed to parse black-to-move position: %v", err)
	}

	whiteHash := HashPosition(&whitePosition)
	blackHash := HashPosition(&blackPosition)

	if whiteHash == blackHash {
		t.Fatal("side to move does not affect Zobrist hash")
	}
}

func TestZobristCastlingRightsChangeHash(t *testing.T) {
	withCastling, err := chess.ParseFEN(
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
	)
	if err != nil {
		t.Fatalf("failed to parse position with castling rights: %v", err)
	}

	withoutCastling, err := chess.ParseFEN(
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w - - 0 1",
	)
	if err != nil {
		t.Fatalf("failed to parse position without castling rights: %v", err)
	}

	h1 := HashPosition(&withCastling)
	h2 := HashPosition(&withoutCastling)

	if h1 == h2 {
		t.Fatal("castling rights do not affect Zobrist hash")
	}
}

func TestZobristEnPassantChangesHash(t *testing.T) {
	withEP, err := chess.ParseFEN(
		"rnbqkbnr/pppppppp/8/8/8/4P3/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
	)
	if err != nil {
		t.Fatalf("failed to parse position with en-passant square: %v", err)
	}

	withoutEP, err := chess.ParseFEN(
		"rnbqkbnr/pppppppp/8/8/8/4P3/PPPP1PPP/RNBQKBNR b KQkq - 0 1",
	)
	if err != nil {
		t.Fatalf("failed to parse position without en-passant square: %v", err)
	}

	h1 := HashPosition(&withEP)
	h2 := HashPosition(&withoutEP)

	if h1 == h2 {
		t.Fatal("en-passant square does not affect Zobrist hash")
	}
}

func TestTranspositionTableStoreProbe(t *testing.T) {
	tt := NewTranspositionTable(1)

	var move chess.Move
	move.From = 12
	move.To = 28

	const key uint64 = 123456789

	tt.Store(
		key,
		5,
		150,
		TTExact,
		move,
	)

	entry, ok := tt.Probe(key)

	if !ok {
		t.Fatal("expected TT entry")
	}

	if entry.Key != key {
		t.Fatalf(
			"wrong key: got %d want %d",
			entry.Key,
			key,
		)
	}

	if entry.Depth != 5 {
		t.Fatalf(
			"wrong depth: got %d want 5",
			entry.Depth,
		)
	}

	if entry.Score != 150 {
		t.Fatalf(
			"wrong score: got %d want 150",
			entry.Score,
		)
	}

	if entry.Flag != TTExact {
		t.Fatalf(
			"wrong flag: got %d want %d",
			entry.Flag,
			TTExact,
		)
	}

	if !sameMove(entry.BestMove, move) {
		t.Fatal("wrong best move")
	}
}

func TestTranspositionTableClear(t *testing.T) {
	tt := NewTranspositionTable(1)

	const key uint64 = 123

	tt.Store(
		key,
		5,
		100,
		TTExact,
		chess.Move{},
	)

	tt.Clear()

	_, ok := tt.Probe(key)

	if ok {
		t.Fatal("expected TT to be empty after Clear")
	}
}

func TestTranspositionTableRejectsShallowerReplacement(t *testing.T) {
	tt := NewTranspositionTable(1)

	const key uint64 = 123

	var firstMove chess.Move
	firstMove.From = 12
	firstMove.To = 28

	var secondMove chess.Move
	secondMove.From = 12
	secondMove.To = 20

	tt.Store(
		key,
		8,
		200,
		TTExact,
		firstMove,
	)

	// Shallower result should not replace the deeper entry.
	tt.Store(
		key,
		4,
		50,
		TTExact,
		secondMove,
	)

	entry, ok := tt.Probe(key)

	if !ok {
		t.Fatal("expected TT entry")
	}

	if entry.Depth != 8 {
		t.Fatalf(
			"shallower entry replaced deeper entry: got depth %d want 8",
			entry.Depth,
		)
	}

	if entry.Score != 200 {
		t.Fatalf(
			"shallower entry replaced score: got %d want 200",
			entry.Score,
		)
	}

	if !sameMove(entry.BestMove, firstMove) {
		t.Fatal("shallower entry replaced best move")
	}
}
