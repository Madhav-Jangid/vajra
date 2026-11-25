package engine

import (
	"runtime"
	"testing"
	"time"

	"vajra/internal/chess"
)

func withThreads(t *testing.T, n int) {
	t.Helper()

	old := Threads
	Threads = n

	t.Cleanup(func() {
		Threads = old

		ClearStop()
	})
}

func TestThreadedSearchFindsMates(t *testing.T) {
	tests := []struct {
		name, fen, move string
		depth           int
	}{
		{"back rank", "6k1/5ppp/8/8/8/8/8/R3K3 w - - 0 1", "a1a8", 6},
		{"smothered", "6rk/6pp/8/6N1/8/8/8/K7 w - - 0 1", "g5f7", 6},
		{"mate in 2", "kbK5/pp6/1P6/8/8/8/8/R7 w - - 0 1", "a1a6", 8},
		{"knight promotion", "1nb5/Pnk1K3/2p5/8/8/8/8/8 w - - 0 1", "a7a8n", 6},
	}

	for _, threads := range []int{2, 4} {
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				withThreads(t, threads)
				prodSetFlags(t, prodDefaultFlags)

				p := prodPos(t, tc.fen)

				move, ok := IterativeSearchWithInfo(&p, tc.depth, 0, nil)
				if !ok || prodMoveName(move) != tc.move {
					t.Fatalf("threads=%d: got %s (found=%v), want %s", threads, prodMoveName(move), ok, tc.move)
				}
			})
		}
	}
}

func TestThreadedSearchReturnsLegalMoves(t *testing.T) {
	for _, tc := range prodFENs {
		t.Run(tc.name, func(t *testing.T) {
			withThreads(t, 4)
			prodSetFlags(t, prodDefaultFlags)

			p := prodPos(t, tc.fen)

			move, ok := IterativeSearchWithInfo(&p, 4, 0, nil)
			if !ok || !prodIsLegal(&p, move) {
				t.Fatalf("got %s (found=%v)", prodMoveName(move), ok)
			}
		})
	}
}

func TestThreadedSearchLeavesThePositionUntouched(t *testing.T) {
	withThreads(t, 4)
	prodSetFlags(t, prodDefaultFlags)

	for _, tc := range prodFENs {
		p := prodPos(t, tc.fen)
		before := p

		IterativeSearchWithInfo(&p, 3, 0, nil)

		if p != before {
			t.Fatalf("%s: the search changed the position", tc.name)
		}
	}
}

func TestThreadedSearchStartsHelpers(t *testing.T) {
	withThreads(t, 4)
	prodSetFlags(t, prodDefaultFlags)

	p := prodPos(t, "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	base := runtime.NumGoroutine()
	peak := 0

	IterativeSearchWithInfo(&p, 12, 400*time.Millisecond, func(i SearchInfo) {
		if i.Depth == 3 {
			peak = runtime.NumGoroutine()
		}
	})

	if peak < base+3 {
		t.Fatalf("expected 3 helper goroutines while searching, saw %d on top of %d", peak-base, base)
	}
}

func TestThreadedSearchLeavesNoGoroutinesBehind(t *testing.T) {
	withThreads(t, 4)
	prodSetFlags(t, prodDefaultFlags)

	p := prodPos(t, "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	base := runtime.NumGoroutine()

	for i := 0; i < 3; i++ {
		IterativeSearchWithInfo(&p, 12, 150*time.Millisecond, nil)
	}

	if now := runtime.NumGoroutine(); now > base {
		t.Fatalf("%d goroutines before, %d after", base, now)
	}
}

func TestThreadedSearchStopsPromptly(t *testing.T) {
	withThreads(t, 4)
	prodSetFlags(t, prodDefaultFlags)
	ClearStop()

	p := prodPos(t, "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	base := runtime.NumGoroutine()
	reached := make(chan struct{}, 1)
	done := make(chan chess.Move, 1)

	go func() {
		move, _ := IterativeSearchWithInfo(&p, 64, 0, func(i SearchInfo) {
			if i.Depth == 2 {
				reached <- struct{}{}
			}
		})

		done <- move
	}()

	select {
	case <-reached:
	case <-time.After(5 * time.Second):
		t.Fatal("the search never started")
	}

	RequestStop()

	select {
	case move := <-done:
		check := prodPos(t, "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

		if !prodIsLegal(&check, move) {
			t.Fatalf("illegal move %s after a stop", prodMoveName(move))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the threaded search ignored the stop request")
	}

	time.Sleep(50 * time.Millisecond)

	if now := runtime.NumGoroutine(); now > base {
		t.Fatalf("%d goroutines before the search, %d after the stop", base, now)
	}
}

func TestThreadedSearchRespectsTheTimeLimit(t *testing.T) {
	withThreads(t, 4)
	prodSetFlags(t, prodDefaultFlags)

	p := prodPos(t, "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")

	start := time.Now()

	move, ok := IterativeSearchWithInfo(&p, 64, 250*time.Millisecond, nil)

	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("a 250ms search took %v", elapsed)
	}

	if !ok || !prodIsLegal(&p, move) {
		t.Fatalf("got %s (found=%v)", prodMoveName(move), ok)
	}
}

func TestSharedTableSurvivesConcurrentUse(t *testing.T) {
	tt := NewTranspositionTable(1)
	done := make(chan struct{})

	for g := 0; g < 8; g++ {
		go func(g int) {
			defer func() { done <- struct{}{} }()

			for i := 0; i < 20000; i++ {
				key := uint64(i*8 + g%3 + 1)

				tt.Store(key, i%20, i, TTExact, chess.Move{From: g, To: i % 64})

				if entry, ok := tt.Probe(key); ok && entry.Key != key {
					t.Errorf("probe returned the wrong entry: %d for %d", entry.Key, key)

					return
				}
			}
		}(g)
	}

	for g := 0; g < 8; g++ {
		<-done
	}
}

func TestSingleThreadStaysTheDefault(t *testing.T) {
	if Threads != 1 {
		t.Fatalf("Threads defaults to %d, want 1", Threads)
	}
}

func TestHelpersDoNotLeaveTheEngineStopped(t *testing.T) {
	withThreads(t, 3)
	prodSetFlags(t, prodDefaultFlags)

	p := prodPos(t, chess.StartingFEN)

	IterativeSearchWithInfo(&p, 3, 0, nil)

	if expired(time.Time{}) {
		t.Fatal("a finished threaded search left the stop flag set")
	}

	Threads = 1

	q := prodPos(t, chess.StartingFEN)

	move, ok := IterativeSearch(&q, 3, 0)
	if !ok || !prodIsLegal(&q, move) {
		t.Fatalf("a search after a threaded one failed: %s (found=%v)", prodMoveName(move), ok)
	}
}
