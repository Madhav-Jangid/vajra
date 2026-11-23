package engine

import (
	"testing"

	"vajra/internal/chess"
)

func bit(square int) chess.Bitboard {
	return chess.Bitboard(1) << uint(square)
}

func setPiece(p *chess.Position, piece chess.PieceType, square int) {
	b := bit(square)

	switch piece {
	case chess.WhiteKing:
		p.Board.WhiteKing |= b
	case chess.WhiteQueen:
		p.Board.WhiteQueens |= b
	case chess.WhiteRook:
		p.Board.WhiteRooks |= b
	case chess.WhiteBishop:
		p.Board.WhiteBishops |= b
	case chess.WhiteKnight:
		p.Board.WhiteKnights |= b
	case chess.WhitePawn:
		p.Board.WhitePawns |= b

	case chess.BlackKing:
		p.Board.BlackKing |= b
	case chess.BlackQueen:
		p.Board.BlackQueens |= b
	case chess.BlackRook:
		p.Board.BlackRooks |= b
	case chess.BlackBishop:
		p.Board.BlackBishops |= b
	case chess.BlackKnight:
		p.Board.BlackKnights |= b
	case chess.BlackPawn:
		p.Board.BlackPawns |= b

	default:
		panic("unsupported piece type")
	}
}

func newPosition(side chess.Color, pieces map[chess.PieceType][]int) chess.Position {
	var p chess.Position
	p.SideToMove = side
	p.EnPassant = -1

	for piece, squares := range pieces {
		for _, square := range squares {
			setPiece(&p, piece, square)
		}
	}

	return p
}

func containsMove(moves []chess.Move, target chess.Move) bool {
	for _, move := range moves {
		if move.From == target.From &&
			move.To == target.To &&
			move.Promotion == target.Promotion {
			return true
		}
	}

	return false
}

// ------------------------------------------------------------
// Basic Search Tests
// ------------------------------------------------------------

func TestSearchReturnsLegalMove(t *testing.T) {
	// Normal starting position.
	p := newPosition(chess.White, map[chess.PieceType][]int{
		chess.WhiteKing:   {4}, // e1
		chess.WhiteQueen:  {3}, // d1
		chess.WhiteRook:   {0, 7},
		chess.WhiteBishop: {2, 5},
		chess.WhiteKnight: {1, 6},
		chess.WhitePawn: {
			8, 9, 10, 11, 12, 13, 14, 15,
		},

		chess.BlackKing:   {60}, // e8
		chess.BlackQueen:  {59}, // d8
		chess.BlackRook:   {56, 63},
		chess.BlackBishop: {58, 61},
		chess.BlackKnight: {57, 62},
		chess.BlackPawn: {
			48, 49, 50, 51, 52, 53, 54, 55,
		},
	})

	move, found := Search(&p, 1)

	if !found {
		t.Fatal("Search returned no move from starting position")
	}

	legal := allMoves(&p)

	if !containsMove(legal, move) {
		t.Fatalf(
			"Search returned illegal move: %d -> %d",
			move.From,
			move.To,
		)
	}
}

func TestSearchDepthZeroIsNormalized(t *testing.T) {
	p := newPosition(chess.White, map[chess.PieceType][]int{
		chess.WhiteKing: {4},
		chess.BlackKing: {60},
	})

	move, found := Search(&p, 0)

	if !found {
		t.Fatal("Search(depth=0) should normalize to depth=1 and find a move")
	}

	legal := allMoves(&p)

	if !containsMove(legal, move) {
		t.Fatalf("Search(depth=0) returned illegal move: %d -> %d",
			move.From, move.To)
	}
}

// ------------------------------------------------------------
// Evaluation Perspective
// ------------------------------------------------------------

func TestEvalSidePerspective(t *testing.T) {
	// White has a queen. From White's perspective this is positive.
	whitePosition := newPosition(chess.White, map[chess.PieceType][]int{
		chess.WhiteKing:  {4},  // e1
		chess.WhiteQueen: {3},  // d1
		chess.BlackKing:  {60}, // e8
	})

	whiteScore := evalSide(&whitePosition)

	if whiteScore <= 0 {
		t.Fatalf("expected positive score for White, got %d", whiteScore)
	}

	// Same material, but Black to move.
	blackPosition := whitePosition
	blackPosition.SideToMove = chess.Black

	blackScore := evalSide(&blackPosition)

	if blackScore >= 0 {
		t.Fatalf("expected negative score for Black, got %d", blackScore)
	}

	if whiteScore != -blackScore {
		t.Fatalf(
			"side perspective mismatch: white=%d black=%d",
			whiteScore,
			blackScore,
		)
	}
}

// ------------------------------------------------------------
// Material Choice
// ------------------------------------------------------------

func TestSearchCapturesQueen(t *testing.T) {
	// White bishop on b1 can capture Black queen on h7.
	//
	// b1 -> c2 -> d3 -> e4 -> f5 -> g6 -> h7
	//
	// Black king is far enough away that h7 is not protected.
	p := newPosition(chess.White, map[chess.PieceType][]int{
		chess.WhiteKing:   {4}, // e1
		chess.WhiteBishop: {1}, // b1

		chess.BlackKing:  {60}, // e8
		chess.BlackQueen: {55}, // h7
	})

	move, found := Search(&p, 1)

	if !found {
		t.Fatal("expected a move")
	}

	if move.From != 1 || move.To != 55 {
		t.Fatalf(
			"expected Bxh7 (1 -> 55), got %d -> %d",
			move.From,
			move.To,
		)
	}
}

// ------------------------------------------------------------
// Checkmate
// ------------------------------------------------------------

func mateInOnePosition() chess.Position {
	// White:
	// King  f6
	// Queen f7
	//
	// Black:
	// King  h8
	//
	// White can play Qg7#.
	//
	// h8 -> g8 is controlled by Qg7
	// h8 -> g7 is occupied/protected by Qg7 + Kf6
	// h8 -> h7 is controlled by Qg7
	return newPosition(chess.White, map[chess.PieceType][]int{
		chess.WhiteKing:  {45}, // f6
		chess.WhiteQueen: {53}, // f7
		chess.BlackKing:  {63}, // h8
	})
}

func TestNegamaxDetectsCheckmate(t *testing.T) {
	p := mateInOnePosition()

	// After Qg7#, Black is checkmated.
	undo := p.MakeMove(chess.Move{
		From: 53, // f7
		To:   54, // g7
		// Piece/flags aren't needed by MakeMove if your implementation
		// determines them from the board. If yours requires Piece,
		// set Piece: chess.WhiteQueen.
	})

	score := negamax(&p, 0, 3, -Infinity, Infinity)

	p.UnmakeMove(
		chess.Move{
			From: 53,
			To:   54,
		},
		undo,
	)

	expected := -MateScore + 3

	if score != expected {
		t.Fatalf(
			"expected checkmate score %d, got %d",
			expected,
			score,
		)
	}
}

func TestSearchFindsMateInOne(t *testing.T) {
	p := mateInOnePosition()

	move, found := Search(&p, 1)

	if !found {
		t.Fatal("expected Search to find a move")
	}

	if move.From != 53 || move.To != 54 {
		t.Fatalf(
			"expected Qg7# (53 -> 54), got %d -> %d",
			move.From,
			move.To,
		)
	}
}

// ------------------------------------------------------------
// Stalemate
// ------------------------------------------------------------

func stalematePosition() chess.Position {
	// Black king h8
	// White king f7
	// White queen g6
	//
	// Black is NOT in check.
	//
	// g8 -> attacked by Kf7
	// g7 -> attacked by Kf7
	// h7 -> attacked by Qg6
	//
	// Therefore Black has no legal moves and is not in check.
	return newPosition(chess.Black, map[chess.PieceType][]int{
		chess.BlackKing:  {63}, // h8
		chess.WhiteKing:  {53}, // f7
		chess.WhiteQueen: {46}, // g6
	})
}

func TestNegamaxDetectsStalemate(t *testing.T) {
	p := stalematePosition()

	score := negamax(&p, 0, 1, -Infinity, Infinity)

	if score != 0 {
		t.Fatalf("expected stalemate score 0, got %d", score)
	}
}

func TestSearchReturnsNoMoveOnStalemate(t *testing.T) {
	p := stalematePosition()

	_, found := Search(&p, 3)

	if found {
		t.Fatal("Search should return no move in stalemate")
	}
}

// ------------------------------------------------------------
// Checkmate Position
// ------------------------------------------------------------

func checkmatePosition() chess.Position {
	// Black is already checkmated:
	//
	// Black king h8
	// White king f6
	// White queen g7
	return newPosition(chess.Black, map[chess.PieceType][]int{
		chess.BlackKing:  {63},
		chess.WhiteKing:  {45},
		chess.WhiteQueen: {54},
	})
}

func TestSearchReturnsNoMoveOnCheckmate(t *testing.T) {
	p := checkmatePosition()

	_, found := Search(&p, 3)

	if found {
		t.Fatal("Search should return no move in checkmate")
	}
}

func TestCheckmateScore(t *testing.T) {
	p := checkmatePosition()

	score := negamax(&p, 3, 7, -Infinity, Infinity)

	expected := -MateScore + 7

	if score != expected {
		t.Fatalf(
			"expected mate score %d, got %d",
			expected,
			score,
		)
	}
}

// ------------------------------------------------------------
// Mate Preference
// ------------------------------------------------------------

func TestSearchPrefersMate(t *testing.T) {
	p := mateInOnePosition()

	// Depth 2 is enough to verify that the immediate mate
	// receives a much larger score than ordinary moves.
	move, found := Search(&p, 2)

	if !found {
		t.Fatal("expected Search to find a move")
	}

	if move.From != 53 || move.To != 54 {
		t.Fatalf(
			"expected immediate mate Qg7#, got %d -> %d",
			move.From,
			move.To,
		)
	}
}

// ------------------------------------------------------------
// Make / Unmake Integrity
// ------------------------------------------------------------

func TestSearchDoesNotModifyPosition(t *testing.T) {
	p := newPosition(chess.White, map[chess.PieceType][]int{
		chess.WhiteKing:   {4},
		chess.WhiteQueen:  {3},
		chess.WhiteRook:   {0, 7},
		chess.WhiteBishop: {2, 5},
		chess.WhiteKnight: {1, 6},
		chess.WhitePawn: {
			8, 9, 10, 11, 12, 13, 14, 15,
		},

		chess.BlackKing:   {60},
		chess.BlackQueen:  {59},
		chess.BlackRook:   {56, 63},
		chess.BlackBishop: {58, 61},
		chess.BlackKnight: {57, 62},
		chess.BlackPawn: {
			48, 49, 50, 51, 52, 53, 54, 55,
		},
	})

	before := p

	Search(&p, 3)

	if p != before {
		t.Fatal("Search modified the original position")
	}
}

// ------------------------------------------------------------
// Negamax Sanity
// ------------------------------------------------------------

func TestNegamaxDoesNotReturnInfinityForNormalPosition(t *testing.T) {
	p := newPosition(chess.White, map[chess.PieceType][]int{
		chess.WhiteKing: {4},
		chess.BlackKing: {60},
	})

	score := negamax(&p, 2, 1, -Infinity, Infinity)

	if score == Infinity || score == -Infinity {
		t.Fatalf("normal position returned infinity: %d", score)
	}
}

func TestAllMovesMatchesSearchMove(t *testing.T) {
	p := mateInOnePosition()

	legalMoves := allMoves(&p)

	if len(legalMoves) == 0 {
		t.Fatal("mate-in-one position unexpectedly has no legal moves")
	}

	move, found := Search(&p, 1)

	if !found {
		t.Fatal("Search returned no move")
	}

	if !containsMove(legalMoves, move) {
		t.Fatalf(
			"Search returned move %d -> %d that is not in legal move list",
			move.From,
			move.To,
		)
	}
}
