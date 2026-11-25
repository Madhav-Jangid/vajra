package elo

import (
	"math"
	"testing"
)

func near(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

func TestScore(t *testing.T) {
	tests := []struct {
		w, d, l int
		want    float64
	}{
		{0, 0, 0, 0.5},
		{10, 0, 0, 1},
		{0, 0, 10, 0},
		{0, 10, 0, 0.5},
		{5, 0, 5, 0.5},
		{3, 2, 1, 4.0 / 6},
		{1, 1, 0, 0.75},
	}

	for _, tc := range tests {
		if got := Score(tc.w, tc.d, tc.l); !near(got, tc.want, 1e-12) {
			t.Fatalf("Score(%d,%d,%d) = %v, want %v", tc.w, tc.d, tc.l, got, tc.want)
		}
	}
}

func TestDifferenceKnownValues(t *testing.T) {
	tests := []struct {
		score, want float64
	}{
		{0.5, 0},
		{0.75, 190.85},
		{0.25, -190.85},
		{0.64, 100.03},
		{0.36, -100.03},
		{0.9, 381.28},
		{0.99, 798.3},
	}

	for _, tc := range tests {
		if got := Difference(tc.score); !near(got, tc.want, 1) {
			t.Fatalf("Difference(%v) = %.2f, want about %.2f", tc.score, got, tc.want)
		}
	}
}

func TestDifferenceEdgesAndShape(t *testing.T) {
	if !math.IsInf(Difference(0), -1) || !math.IsInf(Difference(-0.2), -1) {
		t.Fatal("a score of 0 means an unbounded deficit")
	}

	if !math.IsInf(Difference(1), 1) || !math.IsInf(Difference(1.5), 1) {
		t.Fatal("a score of 1 means an unbounded lead")
	}

	prev := math.Inf(-1)

	for s := 0.01; s < 1; s += 0.01 {
		d := Difference(s)

		if d <= prev {
			t.Fatalf("Difference must grow with the score, %v gave %v after %v", s, d, prev)
		}

		if !near(d, -Difference(1-s), 1e-9) {
			t.Fatalf("Difference must be antisymmetric around 0.5 at %v", s)
		}

		prev = d
	}
}

func TestEstimate(t *testing.T) {
	diff, margin := Estimate(50, 0, 50)

	if !near(diff, 0, 1e-9) || margin <= 0 || math.IsInf(margin, 0) {
		t.Fatalf("an even match gives %v ± %v", diff, margin)
	}

	diff, _ = Estimate(75, 0, 25)

	if !near(diff, 190.85, 1) {
		t.Fatalf("75%% should be about +191, got %v", diff)
	}

	diff, _ = Estimate(25, 0, 75)

	if diff >= 0 {
		t.Fatalf("losing 25-75 must be negative, got %v", diff)
	}

	_, small := Estimate(10, 10, 10)
	_, large := Estimate(1000, 1000, 1000)

	if large >= small {
		t.Fatalf("more games must shrink the margin: %v then %v", small, large)
	}

	diff, margin = Estimate(0, 0, 0)

	if diff != 0 || !math.IsInf(margin, 1) {
		t.Fatalf("no games should give 0 ± inf, got %v ± %v", diff, margin)
	}

	if diff, _ := Estimate(10, 0, 0); !math.IsInf(diff, 1) {
		t.Fatalf("a clean sweep is an unbounded lead, got %v", diff)
	}

	if diff, _ := Estimate(0, 0, 10); !math.IsInf(diff, -1) {
		t.Fatalf("a whitewash is an unbounded deficit, got %v", diff)
	}
}

func TestDrawsShrinkTheMargin(t *testing.T) {
	_, decisive := Estimate(60, 0, 40)
	_, drawish := Estimate(30, 60, 10)

	if drawish >= decisive {
		t.Fatalf("a drawish match of 100 games is more certain: %v vs %v", drawish, decisive)
	}
}

func TestLikelihoodOfSuperiority(t *testing.T) {
	if got := LikelihoodOfSuperiority(0, 0); got != 0.5 {
		t.Fatalf("no decisive games gives %v, want 0.5", got)
	}

	if got := LikelihoodOfSuperiority(30, 30); !near(got, 0.5, 1e-9) {
		t.Fatalf("equal wins and losses give %v, want 0.5", got)
	}

	if got := LikelihoodOfSuperiority(80, 20); got < 0.999 {
		t.Fatalf("80-20 gives %v, want nearly 1", got)
	}

	if got := LikelihoodOfSuperiority(20, 80); got > 0.001 {
		t.Fatalf("20-80 gives %v, want nearly 0", got)
	}

	if a, b := LikelihoodOfSuperiority(12, 8), LikelihoodOfSuperiority(8, 12); !near(a+b, 1, 1e-9) {
		t.Fatalf("the two sides must sum to 1, got %v", a+b)
	}
}

func TestRating(t *testing.T) {
	if got := Rating(1800, 50); got != 1850 {
		t.Fatalf("got %v", got)
	}

	if got := Rating(1800, -120.5); got != 1679.5 {
		t.Fatalf("got %v", got)
	}
}
