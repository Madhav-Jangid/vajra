package engine

import (
	"testing"
	"time"

	"vajra/internal/chess"
)

func TestMateIn(t *testing.T) {
	tests := []struct {
		score    int
		wantMove int
		wantMate bool
	}{
		{MateScore - 1, 1, true},
		{MateScore - 2, 1, true},
		{MateScore - 3, 2, true},
		{MateScore - 5, 3, true},
		{-MateScore + 2, -1, true},
		{-MateScore + 4, -2, true},
		{-MateScore, 0, true},
		{0, 0, false},
		{350, 0, false},
		{-350, 0, false},
		{MateScore - 5000, 0, false},
		{-MateScore + 5000, 0, false},
	}

	for _, tc := range tests {
		move, ok := MateIn(tc.score)

		if ok != tc.wantMate || (ok && move != tc.wantMove) {
			t.Fatalf("MateIn(%d) = %d, %v; want %d, %v", tc.score, move, ok, tc.wantMove, tc.wantMate)
		}
	}
}

func TestMateProven(t *testing.T) {
	tests := []struct {
		score, depth int
		want         bool
	}{
		{MateScore - 1, 1, false},
		{MateScore - 1, 2, true},
		{MateScore - 3, 3, false},
		{MateScore - 3, 4, true},
		{-MateScore + 2, 2, false},
		{-MateScore + 2, 3, true},
		{0, 50, false},
		{400, 50, false},
	}

	for _, tc := range tests {
		if got := mateProven(tc.score, tc.depth); got != tc.want {
			t.Fatalf("mateProven(%d, %d) = %v, want %v", tc.score, tc.depth, got, tc.want)
		}
	}
}

func TestIterativeSearchWithInfoReportsEveryDepth(t *testing.T) {
	p := prodPos(t, chess.StartingFEN)

	prodSetFlags(t, prodDefaultFlags)

	var infos []SearchInfo

	move, ok := IterativeSearchWithInfo(&p, 4, 0, func(i SearchInfo) { infos = append(infos, i) })
	if !ok || !prodIsLegal(&p, move) {
		t.Fatalf("got %s (found=%v)", prodMoveName(move), ok)
	}

	if len(infos) != 4 {
		t.Fatalf("want 4 reports, got %d", len(infos))
	}

	for i, info := range infos {
		if info.Depth != i+1 {
			t.Fatalf("report %d has depth %d", i, info.Depth)
		}

		if info.Nodes == 0 || info.Elapsed < 0 || !prodIsLegal(&p, info.Best) {
			t.Fatalf("report %d is incomplete: %+v", i, info)
		}

		if i > 0 && info.Nodes <= infos[i-1].Nodes {
			t.Fatalf("nodes must grow with depth: %d then %d", infos[i-1].Nodes, info.Nodes)
		}
	}

	if infos[3].Best != move {
		t.Fatal("the last report must carry the move that is returned")
	}
}

func TestIterativeSearchWithInfoMatchesIterativeSearch(t *testing.T) {
	for _, tc := range prodFENs {
		t.Run(tc.name, func(t *testing.T) {
			a, b := prodPos(t, tc.fen), prodPos(t, tc.fen)

			prodSetFlags(t, prodDefaultFlags)
			want, wantOK := IterativeSearch(&a, 3, 0)

			prodSetFlags(t, prodDefaultFlags)
			got, gotOK := IterativeSearchWithInfo(&b, 3, 0, nil)

			if want != got || wantOK != gotOK {
				t.Fatalf("IterativeSearch = %s, WithInfo = %s", prodMoveName(want), prodMoveName(got))
			}
		})
	}
}

func TestIterativeSearchWithInfoScoreMatchesNegamax(t *testing.T) {
	for _, tc := range prodFENs {
		t.Run(tc.name, func(t *testing.T) {
			p := prodPos(t, tc.fen)

			prodSetFlags(t, prodFlags{ordering: true, history: true, killers: true, pvs: true})

			var last SearchInfo

			IterativeSearchWithInfo(&p, 3, 0, func(i SearchInfo) { last = i })

			want := prodRootScore(t, prodPos(t, tc.fen), 3, prodFlags{ordering: true, history: true, killers: true, pvs: true})

			if last.Score != want {
				t.Fatalf("reported score %d, root negamax score %d", last.Score, want)
			}
		})
	}
}

func TestIterativeSearchWithInfoStopsOnceAMateIsProven(t *testing.T) {
	p := prodPos(t, "6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1")

	prodSetFlags(t, prodDefaultFlags)

	var depths []int

	move, ok := IterativeSearchWithInfo(&p, 30, 0, func(i SearchInfo) { depths = append(depths, i.Depth) })

	if !ok || prodMoveName(move) != "a1a8" {
		t.Fatalf("got %s (found=%v), want a1a8", prodMoveName(move), ok)
	}

	if len(depths) != 2 {
		t.Fatalf("a mate in 1 is proven at depth 2, but depths %v were searched", depths)
	}
}

func TestIterativeSearchWithInfoWithoutALegalMove(t *testing.T) {
	p := prodPos(t, "R5k1/5ppp/8/8/8/8/8/4K3 b - - 0 1")

	prodSetFlags(t, prodDefaultFlags)

	called := false

	if _, ok := IterativeSearchWithInfo(&p, 4, 0, func(SearchInfo) { called = true }); ok {
		t.Fatal("a checkmated side has no move")
	}

	if called {
		t.Fatal("nothing should be reported when there is no move")
	}
}

func TestIterativeSearchWithInfoFallsBackWhenOutOfTime(t *testing.T) {
	p := prodPos(t, "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	prodSetFlags(t, prodDefaultFlags)

	move, ok := IterativeSearchWithInfo(&p, 10, time.Nanosecond, nil)
	if !ok || !prodIsLegal(&p, move) {
		t.Fatalf("an expired deadline must still give a legal move, got %s (found=%v)", prodMoveName(move), ok)
	}
}

func TestRequestStopEndsASearchPromptly(t *testing.T) {
	p := prodPos(t, "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	prodSetFlags(t, prodDefaultFlags)

	ClearStop()

	t.Cleanup(ClearStop)

	type result struct {
		move chess.Move
		ok   bool
	}

	done := make(chan result, 1)
	reached := make(chan struct{}, 1)

	go func() {
		move, ok := IterativeSearchWithInfo(&p, 64, 0, func(i SearchInfo) {
			if i.Depth == 2 {
				reached <- struct{}{}
			}
		})

		done <- result{move, ok}
	}()

	select {
	case <-reached:
	case <-time.After(5 * time.Second):
		t.Fatal("the search never reached depth 2")
	}

	RequestStop()

	select {
	case r := <-done:
		if !r.ok {
			t.Fatal("a stopped search must still return the best move so far")
		}

		check := prodPos(t, "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

		if !prodIsLegal(&check, r.move) {
			t.Fatalf("illegal move %s after a stop", prodMoveName(r.move))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the search did not end after RequestStop")
	}
}

func TestStopFlagLifecycle(t *testing.T) {
	t.Cleanup(ClearStop)

	ClearStop()

	if expired(time.Time{}) {
		t.Fatal("without a deadline or a stop request nothing is expired")
	}

	RequestStop()

	if !expired(time.Time{}) || !expired(time.Now().Add(time.Hour)) {
		t.Fatal("a stop request expires every search, with or without a deadline")
	}

	ClearStop()

	if expired(time.Time{}) || expired(time.Now().Add(time.Hour)) {
		t.Fatal("ClearStop must re-arm the engine")
	}
}

func TestSearchRootScoredAgreesWithSearchRoot(t *testing.T) {
	for _, tc := range prodFENs {
		t.Run(tc.name, func(t *testing.T) {
			a, b := prodPos(t, tc.fen), prodPos(t, tc.fen)

			prodSetFlags(t, prodDefaultFlags)
			startSearch(0)
			m1, f1, c1 := searchRoot(&a, 3, time.Time{}, chess.Move{}, &SearchState{})

			prodSetFlags(t, prodDefaultFlags)
			startSearch(0)
			m2, f2, c2, _ := searchRootScored(&b, 3, time.Time{}, chess.Move{}, &SearchState{})

			if m1 != m2 || f1 != f2 || c1 != c2 {
				t.Fatalf("searchRoot = %s/%v/%v, searchRootScored = %s/%v/%v", prodMoveName(m1), f1, c1, prodMoveName(m2), f2, c2)
			}
		})
	}
}
