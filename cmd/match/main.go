package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"vajra/internal/elo"
	"vajra/internal/engine"
	"vajra/internal/match"
)

func player(name string, depth int, lmr, pvs, history, killers bool) match.Player {
	return match.Player{
		Name:  name,
		Depth: depth,
		Setup: func() {
			engine.UseLMR, engine.UsePVS, engine.UseHistory, engine.UseKillers = lmr, pvs, history, killers
		},
	}
}

func main() {
	pairs := flag.Int("openings", len(match.Openings), "number of openings, each played with both colours")
	maxPlies := flag.Int("max-plies", 160, "games longer than this are drawn")
	anchor := flag.Float64("anchor", 0, "known rating of player B, to print an estimated rating for player A")
	minScore := flag.Float64("min-score", 0, "exit with status 3 when player A scores below this fraction")

	depthA := flag.Int("depth-a", 4, "search depth of player A")
	depthB := flag.Int("depth-b", 3, "search depth of player B")
	lmrA := flag.Bool("lmr-a", true, "late move reductions for player A")
	lmrB := flag.Bool("lmr-b", true, "late move reductions for player B")
	pvsA := flag.Bool("pvs-a", true, "principal variation search for player A")
	pvsB := flag.Bool("pvs-b", true, "principal variation search for player B")
	historyA := flag.Bool("history-a", true, "history heuristic for player A")
	historyB := flag.Bool("history-b", true, "history heuristic for player B")
	killersA := flag.Bool("killers-a", true, "killer moves for player A")
	killersB := flag.Bool("killers-b", true, "killer moves for player B")

	flag.Parse()

	if *pairs < 1 || *pairs > len(match.Openings) {
		fmt.Fprintf(os.Stderr, "openings must be from 1 to %d\n", len(match.Openings))
		os.Exit(2)
	}

	a := player("A", *depthA, *lmrA, *pvsA, *historyA, *killersA)
	b := player("B", *depthB, *lmrB, *pvsB, *historyB, *killersB)

	tally, err := match.Run(a, b, match.Openings[:*pairs], *maxPlies)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	diff, margin := elo.Estimate(tally.Wins, tally.Draws, tally.Losses)

	fmt.Printf("games %d: +%d =%d -%d\n", tally.Games(), tally.Wins, tally.Draws, tally.Losses)
	fmt.Printf("score %.1f%%\n", 100*elo.Score(tally.Wins, tally.Draws, tally.Losses))
	fmt.Printf("elo difference %+.0f +/- %.0f\n", diff, margin)
	fmt.Printf("likelihood of superiority %.1f%%\n", 100*elo.LikelihoodOfSuperiority(tally.Wins, tally.Losses))

	if *anchor != 0 && !math.IsInf(diff, 0) {
		fmt.Printf("estimated rating %.0f\n", elo.Rating(*anchor, diff))
	}

	if elo.Score(tally.Wins, tally.Draws, tally.Losses) < *minScore {
		os.Exit(3)
	}
}
