package engine

import (
	"math/bits"
	"strings"
	"testing"

	"vajra/internal/chess"
)

func TestPieceValue(t *testing.T) {
	tests := []struct {
		name  string
		piece chess.PieceType
		want  int
	}{
		{"white pawn", chess.WhitePawn, PawnValue},
		{"black pawn", chess.BlackPawn, PawnValue},

		{"white knight", chess.WhiteKnight, KnightValue},
		{"black knight", chess.BlackKnight, KnightValue},

		{"white bishop", chess.WhiteBishop, BishopValue},
		{"black bishop", chess.BlackBishop, BishopValue},

		{"white rook", chess.WhiteRook, RookValue},
		{"black rook", chess.BlackRook, RookValue},

		{"white queen", chess.WhiteQueen, QueenValue},
		{"black queen", chess.BlackQueen, QueenValue},

		{"white king", chess.WhiteKing, KingValue},
		{"black king", chess.BlackKing, KingValue},

		{"empty", chess.Empty, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pieceValue(tc.piece)

			if got != tc.want {
				t.Errorf(
					"pieceValue(%v) = %d, want %d",
					tc.piece,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestFlipSquare(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"a1 to a8", 0, 56},
		{"b1 to b8", 1, 57},
		{"e1 to e8", 4, 60},
		{"h1 to h8", 7, 63},

		{"a2 to a7", 8, 48},
		{"e2 to e7", 12, 52},

		{"a4 to a5", 24, 32},
		{"e4 to e5", 28, 36},

		{"h8 to h1", 63, 7},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flipSquare(tc.in)

			if got != tc.want {
				t.Errorf(
					"flipSquare(%d) = %d, want %d",
					tc.in,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestPieceSquareBonusUsesCorrectPST(t *testing.T) {
	tests := []struct {
		name   string
		piece  chess.PieceType
		color  chess.Color
		square int
		want   int
	}{
		{
			name:   "white pawn",
			piece:  chess.WhitePawn,
			color:  chess.White,
			square: 28,
			want:   pawnPST[28],
		},
		{
			name:   "black pawn mirrored",
			piece:  chess.BlackPawn,
			color:  chess.Black,
			square: 36,
			want:   pawnPST[flipSquare(36)],
		},
		{
			name:   "white knight",
			piece:  chess.WhiteKnight,
			color:  chess.White,
			square: 18,
			want:   knightPST[18],
		},
		{
			name:   "black knight mirrored",
			piece:  chess.BlackKnight,
			color:  chess.Black,
			square: 42,
			want:   knightPST[flipSquare(42)],
		},
		{
			name:   "white bishop",
			piece:  chess.WhiteBishop,
			color:  chess.White,
			square: 26,
			want:   bishopPST[26],
		},
		{
			name:   "black bishop mirrored",
			piece:  chess.BlackBishop,
			color:  chess.Black,
			square: 34,
			want:   bishopPST[flipSquare(34)],
		},
		{
			name:   "white rook",
			piece:  chess.WhiteRook,
			color:  chess.White,
			square: 0,
			want:   rookPST[0],
		},
		{
			name:   "black rook mirrored",
			piece:  chess.BlackRook,
			color:  chess.Black,
			square: 56,
			want:   rookPST[flipSquare(56)],
		},
		{
			name:   "white queen",
			piece:  chess.WhiteQueen,
			color:  chess.White,
			square: 27,
			want:   queenPST[27],
		},
		{
			name:   "black queen mirrored",
			piece:  chess.BlackQueen,
			color:  chess.Black,
			square: 35,
			want:   queenPST[flipSquare(35)],
		},
		{
			name:   "white king",
			piece:  chess.WhiteKing,
			color:  chess.White,
			square: 4,
			want:   kingMGPST[4],
		},
		{
			name:   "black king mirrored",
			piece:  chess.BlackKing,
			color:  chess.Black,
			square: 60,
			want:   kingMGPST[flipSquare(60)],
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pieceSquareBonus(
				tc.piece,
				tc.color,
				tc.square,
			)

			if got != tc.want {
				t.Errorf(
					"pieceSquareBonus(%v, %v, %d) = %d, want %d",
					tc.piece,
					tc.color,
					tc.square,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestPieceSquareBonusBlackMirrorsWhite(t *testing.T) {
	tests := []struct {
		name        string
		whitePiece  chess.PieceType
		blackPiece  chess.PieceType
		whiteSquare int
		blackSquare int
	}{
		{
			name:        "pawn",
			whitePiece:  chess.WhitePawn,
			blackPiece:  chess.BlackPawn,
			whiteSquare: 28,
			blackSquare: 36,
		},
		{
			name:        "knight",
			whitePiece:  chess.WhiteKnight,
			blackPiece:  chess.BlackKnight,
			whiteSquare: 18,
			blackSquare: 42,
		},
		{
			name:        "bishop",
			whitePiece:  chess.WhiteBishop,
			blackPiece:  chess.BlackBishop,
			whiteSquare: 26,
			blackSquare: 34,
		},
		{
			name:        "rook",
			whitePiece:  chess.WhiteRook,
			blackPiece:  chess.BlackRook,
			whiteSquare: 0,
			blackSquare: 56,
		},
		{
			name:        "queen",
			whitePiece:  chess.WhiteQueen,
			blackPiece:  chess.BlackQueen,
			whiteSquare: 27,
			blackSquare: 35,
		},
		{
			name:        "king",
			whitePiece:  chess.WhiteKing,
			blackPiece:  chess.BlackKing,
			whiteSquare: 4,
			blackSquare: 60,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			white := pieceSquareBonus(
				tc.whitePiece,
				chess.White,
				tc.whiteSquare,
			)

			black := pieceSquareBonus(
				tc.blackPiece,
				chess.Black,
				tc.blackSquare,
			)

			if white != black {
				t.Errorf(
					"white bonus = %d, black mirrored bonus = %d",
					white,
					black,
				)
			}
		})
	}
}

func TestEvaluateStartingPosition(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	position, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN() error = %v", err)
	}

	got := Evaluate(&position)

	// Starting position must be materially equal.
	//
	// With mirrored PSTs, the positional contribution of the
	// starting position is also symmetric.
	want := 0

	if got != want {
		t.Errorf(
			"Evaluate(start position) = %d, want %d",
			got,
			want,
		)
	}
}

func TestEvaluateIsSymmetricForMirroredMaterial(t *testing.T) {
	tests := []struct {
		name string
		fen  string
	}{
		{
			name: "white pawn vs black pawn",
			fen:  "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 1",
		},
		{
			name: "black pawn vs white pawn",
			fen:  "rnbqkbnr/pppp1ppp/8/4p3/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			position, err := chess.ParseFEN(tc.fen)
			if err != nil {
				t.Fatalf("ParseFEN() error = %v", err)
			}

			got := Evaluate(&position)

			if got == 0 {
				t.Errorf(
					"position with material imbalance should not evaluate to 0",
				)
			}
		})
	}
}

func TestEvaluateWhiteAdvantage(t *testing.T) {
	baseFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	whiteFEN := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR w KQkq - 0 1"

	base, err := chess.ParseFEN(baseFEN)
	if err != nil {
		t.Fatalf("ParseFEN(base) error = %v", err)
	}

	position, err := chess.ParseFEN(whiteFEN)
	if err != nil {
		t.Fatalf("ParseFEN(position) error = %v", err)
	}

	baseScore := Evaluate(&base)
	score := Evaluate(&position)

	if score <= baseScore {
		t.Errorf(
			"adding a white pawn should improve evaluation: base=%d score=%d",
			baseScore,
			score,
		)
	}
}

func TestEvaluateBlackAdvantage(t *testing.T) {
	baseFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	blackFEN := "rnbqkbnr/pppp1ppp/8/4p3/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	base, err := chess.ParseFEN(baseFEN)
	if err != nil {
		t.Fatalf("ParseFEN(base) error = %v", err)
	}

	position, err := chess.ParseFEN(blackFEN)
	if err != nil {
		t.Fatalf("ParseFEN(position) error = %v", err)
	}

	baseScore := Evaluate(&base)
	score := Evaluate(&position)

	if score >= baseScore {
		t.Errorf(
			"adding a black pawn should worsen evaluation: base=%d score=%d",
			baseScore,
			score,
		)
	}
}

func evalFEN(t *testing.T, fen string) int {
	t.Helper()

	p, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN(%q): %v", fen, err)
	}

	return Evaluate(&p)
}

func breakdownFEN(t *testing.T, fen string) EvalBreakdown {
	t.Helper()

	p, err := chess.ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN(%q): %v", fen, err)
	}

	return EvaluateBreakdown(&p)
}

// mirrorPlacement flips the board vertically and swaps colours, so
// the evaluation must come out exactly negated.
func mirrorPlacement(placement string) string {
	ranks := strings.Split(placement, "/")
	out := make([]string, 8)

	for i, r := range ranks {
		swapped := strings.Map(func(c rune) rune {
			switch {
			case c >= 'a' && c <= 'z':
				return c - 32
			case c >= 'A' && c <= 'Z':
				return c + 32
			}

			return c
		}, r)

		out[7-i] = swapped
	}

	return strings.Join(out, "/")
}

func TestEvaluateColourSymmetry(t *testing.T) {
	placements := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR",
		"r1bq1rk1/ppp2ppp/2np1n2/2b1p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1",
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R",
		"8/5pk1/6p1/2P5/8/1P4P1/5PK1/8",
		"4k3/8/8/3P4/8/8/PP3P2/4K3",
	}

	for _, pl := range placements {
		white := evalFEN(t, pl+" w - - 0 1")
		black := evalFEN(t, mirrorPlacement(pl)+" w - - 0 1")

		if white != -black {
			t.Fatalf("%s: eval %d but mirrored eval %d, want %d", pl, white, black, -white)
		}
	}
}

func TestEvaluateBreakdownSumsToEvaluate(t *testing.T) {
	fen := "r1bq1rk1/ppp2ppp/2np1n2/2b1p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 7"

	if got, want := breakdownFEN(t, fen).Total(), evalFEN(t, fen); got != want {
		t.Fatalf("breakdown total %d != Evaluate %d", got, want)
	}
}

func TestEvaluatePassedPawn(t *testing.T) {
	// Same pawn on d5; only difference is whether Black has a c/d/e pawn ahead of it.
	passed := breakdownFEN(t, "4k3/8/8/3P4/8/8/8/4K3 w - - 0 1")
	blocked := breakdownFEN(t, "4k3/3p4/8/3P4/8/8/8/4K3 w - - 0 1")

	if passed.PassedPawns <= 0 {
		t.Fatalf("lone advanced pawn should be passed, got %d", passed.PassedPawns)
	}

	if blocked.PassedPawns != 0 {
		t.Fatalf("pawn facing an enemy pawn is not passed, got %d", blocked.PassedPawns)
	}

	// Black's pawn on c7 guards the d-pawn's path diagonally: still not passed.
	if got := breakdownFEN(t, "4k3/2p5/8/3P4/8/8/8/4K3 w - - 0 1").PassedPawns; got != 0 {
		t.Fatalf("pawn with enemy pawn on adjacent file ahead is not passed, got %d", got)
	}

	// Further advanced is worth more, and more in the endgame.
	far := breakdownFEN(t, "4k3/3P4/8/8/8/8/8/4K3 w - - 0 1").PassedPawns
	near := breakdownFEN(t, "4k3/8/8/8/3P4/8/8/4K3 w - - 0 1").PassedPawns

	if far <= near {
		t.Fatalf("rank 7 passer (%d) must outrank rank 4 passer (%d)", far, near)
	}

	mid := breakdownFEN(t, "r3k2r/8/8/3P4/8/8/8/R3K2R w - - 0 1").PassedPawns
	end := breakdownFEN(t, "4k3/8/8/3P4/8/8/8/4K3 w - - 0 1").PassedPawns

	if end <= mid {
		t.Fatalf("passed pawn should be worth more in the endgame: end=%d middlegame=%d", end, mid)
	}

	// Black passers count for Black.
	if got := breakdownFEN(t, "4k3/8/8/8/3p4/8/8/4K3 w - - 0 1").PassedPawns; got >= 0 {
		t.Fatalf("black passed pawn should score negative for White, got %d", got)
	}
}

func TestEvaluatePawnStructure(t *testing.T) {
	healthy := breakdownFEN(t, "4k3/8/8/8/8/8/PPP5/4K3 w - - 0 1").PawnStructure
	doubled := breakdownFEN(t, "4k3/8/8/8/8/P7/PP6/4K3 w - - 0 1").PawnStructure
	isolated := breakdownFEN(t, "4k3/8/8/8/8/8/P1P5/4K3 w - - 0 1").PawnStructure

	if healthy != 0 {
		t.Fatalf("connected pawns should have no penalty, got %d", healthy)
	}

	if doubled >= healthy {
		t.Fatalf("doubled pawns must be penalised: %d vs %d", doubled, healthy)
	}

	if isolated >= healthy {
		t.Fatalf("isolated pawns must be penalised: %d vs %d", isolated, healthy)
	}
}

func TestEvaluateBishopPairAndRookFiles(t *testing.T) {
	pair := breakdownFEN(t, "4k3/8/8/8/8/8/8/2B1KB2 w - - 0 1").Pieces
	single := breakdownFEN(t, "4k3/8/8/8/8/8/8/4KB2 w - - 0 1").Pieces

	if pair <= single {
		t.Fatalf("bishop pair should score more: pair=%d single=%d", pair, single)
	}

	open := breakdownFEN(t, "4k3/8/8/8/8/8/8/R3K3 w - - 0 1").Pieces
	semi := breakdownFEN(t, "4k3/p7/8/8/8/8/8/R3K3 w - - 0 1").Pieces
	closed := breakdownFEN(t, "4k3/p7/8/8/8/8/P7/R3K3 w - - 0 1").Pieces

	if !(open > semi && semi > closed) {
		t.Fatalf("want open > semi-open > closed, got %d, %d, %d", open, semi, closed)
	}
}

func TestEvaluateKingShield(t *testing.T) {
	shielded := breakdownFEN(t, "r1bq1rk1/ppp2ppp/2np1n2/2b1p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 7").KingSafety
	stripped := breakdownFEN(t, "r1bq1rk1/ppp2ppp/2np1n2/2b1p3/2B1P3/2NP1N2/PPP5/R1BQ1RK1 w - - 0 7").KingSafety

	if stripped >= shielded {
		t.Fatalf("losing the f2/g2/h2 pawns must hurt king safety: shielded=%d stripped=%d", shielded, stripped)
	}

	// Shield is a middlegame idea: it must fade with the pieces.
	endgame := breakdownFEN(t, "4k3/8/8/8/8/8/8/6K1 w - - 0 1").KingSafety
	if endgame != 0 {
		t.Fatalf("king safety should vanish in a bare endgame, got %d", endgame)
	}
}

func TestEvaluateKingEndgamePST(t *testing.T) {
	// Bare kings and a pawn: the king belongs in the centre.
	centre := breakdownFEN(t, "4k3/8/8/8/3K4/8/7P/8 w - - 0 1").PieceSquare
	corner := breakdownFEN(t, "4k3/8/8/8/8/8/7P/K7 w - - 0 1").PieceSquare

	if centre <= corner {
		t.Fatalf("endgame king should prefer the centre: centre=%d corner=%d", centre, corner)
	}

	// With all pieces on, a castled king beats a centralised one.
	castled := breakdownFEN(t, "r1bq1rk1/pppp1ppp/2n2n2/2b1p3/2B1P3/2N2N2/PPPP1PPP/R1BQ1RK1 w - - 0 1").PieceSquare
	walked := breakdownFEN(t, "r1bq1rk1/pppp1ppp/2n2n2/2b1p3/2B1K3/2N2N2/PPPP1PPP/R1BQ3R w - - 0 1").PieceSquare

	if walked >= castled {
		t.Fatalf("middlegame king in the centre should score worse: castled=%d walked=%d", castled, walked)
	}
}

func TestTaper(t *testing.T) {
	if taper(100, 40, maxPhase) != 100 || taper(100, 40, 0) != 40 {
		t.Fatal("taper must return the middlegame value at full phase and endgame value at zero")
	}

	if got := taper(100, 40, maxPhase/2); got != 70 {
		t.Fatalf("taper at half phase = %d, want 70", got)
	}
}

func TestMobilityTables(t *testing.T) {
	// Knight: corner has 2 targets, centre has 8.
	if got := bits.OnesCount64(knightAttacks[0]); got != 2 {
		t.Fatalf("knight a1 attacks %d squares, want 2", got)
	}

	if got := bits.OnesCount64(knightAttacks[27]); got != 8 {
		t.Fatalf("knight d4 attacks %d squares, want 8", got)
	}

	// Rook on an empty board always sees 14 squares.
	if got := bits.OnesCount64(slideAttacks(27, 0, rookDirs)); got != 14 {
		t.Fatalf("empty-board rook sees %d squares, want 14", got)
	}

	// Bishop on d4 of an empty board sees 13.
	if got := bits.OnesCount64(slideAttacks(27, 0, bishopDirs)); got != 13 {
		t.Fatalf("empty-board bishop sees %d squares, want 13", got)
	}

	// A blocker on d6 is included but nothing beyond it.
	blocked := slideAttacks(27, uint64(1)<<43, rookDirs)
	if blocked&(uint64(1)<<43) == 0 || blocked&(uint64(1)<<51) != 0 {
		t.Fatal("slider must include the first blocker and stop there")
	}
}

func TestPawnAttacks(t *testing.T) {
	// White pawn e4 (28) attacks d5 (35) and f5 (37).
	got := pawnAttacks(uint64(1)<<28, chess.White)
	if got != uint64(1)<<35|uint64(1)<<37 {
		t.Fatalf("white e4 attacks %#x", got)
	}

	// Black pawn e5 (36) attacks d4 (27) and f4 (29).
	got = pawnAttacks(uint64(1)<<36, chess.Black)
	if got != uint64(1)<<27|uint64(1)<<29 {
		t.Fatalf("black e5 attacks %#x", got)
	}

	// No wrap-around across the board edge.
	if got := pawnAttacks(uint64(1)<<24, chess.White); got != uint64(1)<<33 {
		t.Fatalf("white a4 attacks %#x, want only b5", got)
	}

	if got := pawnAttacks(uint64(1)<<31, chess.White); got != uint64(1)<<38 {
		t.Fatalf("white h4 attacks %#x, want only g5", got)
	}
}

func TestEvaluateMobility(t *testing.T) {
	centre := breakdownFEN(t, "4k3/8/8/8/3N4/8/8/4K3 w - - 0 1").Mobility
	corner := breakdownFEN(t, "4k3/8/8/8/8/8/8/N3K3 w - - 0 1").Mobility

	if centre <= corner {
		t.Fatalf("centralised knight should be more mobile: centre=%d corner=%d", centre, corner)
	}

	open := breakdownFEN(t, "4k3/8/8/8/3B4/8/8/4K3 w - - 0 1").Mobility
	boxed := breakdownFEN(t, "4k3/8/8/8/8/2P1P3/3B4/4K3 w - - 0 1").Mobility

	if open <= boxed {
		t.Fatalf("open bishop should beat a boxed-in one: open=%d boxed=%d", open, boxed)
	}

	// Rook behind its own pawn is worse than a free rook.
	freeRook := breakdownFEN(t, "4k3/8/8/8/3R4/8/8/4K3 w - - 0 1").Mobility
	blockedRook := breakdownFEN(t, "4k3/8/8/3P4/3R4/3P4/8/4K3 w - - 0 1").Mobility

	if freeRook <= blockedRook {
		t.Fatalf("free rook should beat a blocked one: free=%d blocked=%d", freeRook, blockedRook)
	}
}

// Squares guarded by an enemy pawn are not real mobility.
func TestMobilityIgnoresPawnGuardedSquares(t *testing.T) {
	free := breakdownFEN(t, "4k3/8/8/8/3N4/8/8/4K3 w - - 0 1").Mobility
	// Black pawns on a6 and e6 guard b5, d5 and f5: three of the knight's targets.
	guarded := breakdownFEN(t, "4k3/8/p3p3/8/3N4/8/8/4K3 w - - 0 1").Mobility

	if guarded >= free {
		t.Fatalf("pawn-guarded squares must not count: guarded=%d free=%d", guarded, free)
	}
}

func TestEvaluateExposedKingIsPunished(t *testing.T) {
	home := breakdownFEN(t, "r1bq1rk1/ppp2ppp/2np1n2/2b1p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 7").KingSafety
	// Same position with Black's king dragged to c5.
	walked := breakdownFEN(t, "r1bq1r2/ppp2ppp/2np1n2/2k1p3/2B1P3/2NP1N2/PPP2PPP/R1BQ1RK1 w - - 0 7").KingSafety

	if walked <= home {
		t.Fatalf("Black king on c5 should be worse for Black (higher for White): home=%d walked=%d", home, walked)
	}

	// It must fade in the endgame, where an active king is good.
	endHome := breakdownFEN(t, "4k3/8/8/8/8/8/8/6K1 w - - 0 1").KingSafety
	endWalked := breakdownFEN(t, "8/8/8/3k4/8/8/8/6K1 w - - 0 1").KingSafety

	if endHome != endWalked {
		t.Fatalf("king exposure must vanish in an endgame: %d vs %d", endHome, endWalked)
	}
}

func TestKingDangerGrowsWithAttackers(t *testing.T) {
	const g8 = 62

	queen := chess.Bitboard(1) << 35  // d5, long diagonal toward g8
	knight := chess.Bitboard(1) << 45 // f6, hits g8 and h7
	occ := uint64(queen | knight)

	if got := kingDanger(0, 0, 0, 0, occ, g8); got != 0 {
		t.Fatalf("no attackers should mean no danger, got %d", got)
	}

	if got := kingDanger(0, 0, 0, 0, 0, -1); got != 0 {
		t.Fatalf("no king should mean no danger, got %d", got)
	}

	one := kingDanger(0, 0, 0, queen, occ, g8)
	two := kingDanger(knight, 0, 0, queen, occ, g8)

	if one <= 0 {
		t.Fatalf("queen aimed at the king ring should be dangerous, got %d", one)
	}

	if two <= 2*one {
		t.Fatalf("a second attacker should raise danger by more than its own weight: one=%d two=%d", one, two)
	}

	if two > maxKingDanger {
		t.Fatalf("danger %d exceeds cap %d", two, maxKingDanger)
	}
}

func TestThreatPenalty(t *testing.T) {
	if got := threatPenalty(BishopValue, 5); got != (BishopValue-PawnValue)/pawnThreatDivisor {
		t.Fatalf("attacked bishop with escapes: penalty %d", got)
	}

	trapped := threatPenalty(BishopValue, 0)
	free := threatPenalty(BishopValue, 5)

	if trapped <= free {
		t.Fatalf("a trapped piece must cost more: trapped=%d free=%d", trapped, free)
	}

	if threatPenalty(QueenValue, 3) <= threatPenalty(KnightValue, 3) {
		t.Fatal("a more valuable piece must cost more")
	}
}

func TestEvaluateThreatsFromPawns(t *testing.T) {
	// Black bishop on g6, nothing attacks it.
	calm := breakdownFEN(t, "4k3/8/6b1/8/8/8/8/4K3 w - - 0 1").Threats
	if calm != 0 {
		t.Fatalf("unattacked piece should cost nothing, got %d", calm)
	}

	// A white pawn on h5 attacks it, but it still has squares: White gains.
	attacked := breakdownFEN(t, "4k3/8/6b1/7P/8/8/8/4K3 w - - 0 1").Threats
	if attacked <= 0 {
		t.Fatalf("bishop attacked by a pawn should favour White, got %d", attacked)
	}

	// The real game position: bishop on g6 hit by h5 with h7 and f5 unavailable.
	trapped := breakdownFEN(t, "r2qk2r/pppn1ppp/6b1/3Pp2P/4P3/2N2P2/PP3KP1/R2Q1B1R b kq - 0 14").Threats
	if trapped <= attacked {
		t.Fatalf("trapped bishop must be worse than one with escapes: trapped=%d attacked=%d", trapped, attacked)
	}

	// Threats work for both colours.
	black := breakdownFEN(t, "4k3/8/8/8/8/1p6/B7/4K3 w - - 0 1").Threats
	if black >= 0 {
		t.Fatalf("white bishop attacked by a black pawn should favour Black, got %d", black)
	}
}
