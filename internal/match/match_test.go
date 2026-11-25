package match

import (
	"strings"
	"testing"

	"vajra/internal/chess"
	"vajra/internal/engine"
	"vajra/internal/uci"
)

func TestOpeningsAreLegalAndDistinct(t *testing.T) {
	seen := map[string]bool{}

	for _, opening := range Openings {
		pos, err := chess.ParseFEN(chess.StartingFEN)
		if err != nil {
			t.Fatal(err)
		}

		for _, text := range opening {
			m, err := uci.ParseMove(&pos, text)
			if err != nil {
				t.Fatalf("%v: %v", opening, err)
			}

			pos.MakeMove(m)
		}

		if key := strings.Join(opening, " "); seen[key] {
			t.Fatalf("%s is listed twice", key)
		} else {
			seen[key] = true
		}
	}

	if len(Openings) < 10 {
		t.Fatalf("only %d openings", len(Openings))
	}
}

var foolsMate = strings.Fields("f2f3 e7e5 g2g4 d8h4")

func TestPlayRecognisesCheckmate(t *testing.T) {
	p := Player{Name: "p", Depth: 1}

	outcome, err := Play(p, p, foolsMate, 100)
	if err != nil || outcome != BlackWins {
		t.Fatalf("got %v, %v, want BlackWins", outcome, err)
	}
}

func TestPlayAdjudicatesADrawAtTheMoveLimit(t *testing.T) {
	p := Player{Name: "p", Depth: 1}

	outcome, err := Play(p, p, strings.Fields("e2e4 e7e5"), 2)
	if err != nil || outcome != Draw {
		t.Fatalf("got %v, %v, want Draw", outcome, err)
	}
}

func TestPlayRejectsAnIllegalOpening(t *testing.T) {
	p := Player{Name: "p", Depth: 1}

	if _, err := Play(p, p, strings.Fields("e2e4 e2e4"), 20); err == nil {
		t.Fatal("an illegal opening move must be an error")
	}
}

func TestPlayCallsEachPlayersSetupForItsOwnMoves(t *testing.T) {
	var white, black int

	w := Player{Depth: 1, Setup: func() { white++ }}
	b := Player{Depth: 1, Setup: func() { black++ }}

	if _, err := Play(w, b, strings.Fields("e2e4 e7e5"), 8); err != nil {
		t.Fatal(err)
	}

	if white != 3 || black != 3 {
		t.Fatalf("white moved %d times and black %d times, want 3 and 3 (plies 2 to 7)", white, black)
	}
}

func TestPlayEndsInAnOutcomeForARealGame(t *testing.T) {
	p := Player{Name: "p", Depth: 2}

	outcome, err := Play(p, p, Openings[0], 60)
	if err != nil {
		t.Fatal(err)
	}

	if outcome != Draw && outcome != WhiteWins && outcome != BlackWins {
		t.Fatalf("invalid outcome %v", outcome)
	}
}

func TestRunTalliesFromThePlayersPointOfView(t *testing.T) {
	a := Player{Name: "a", Depth: 1}
	b := Player{Name: "b", Depth: 1}

	tally, err := Run(a, b, [][]string{foolsMate}, 100)
	if err != nil {
		t.Fatal(err)
	}

	if tally.Wins != 1 || tally.Losses != 1 || tally.Draws != 0 || tally.Games() != 2 {
		t.Fatalf("got %+v, want one win and one loss", tally)
	}
}

func TestRunPlaysEachOpeningWithBothColours(t *testing.T) {
	a := Player{Name: "a", Depth: 1}
	b := Player{Name: "b", Depth: 1}

	tally, err := Run(a, b, Openings[:3], 12)
	if err != nil {
		t.Fatal(err)
	}

	if tally.Games() != 6 {
		t.Fatalf("got %d games, want 6", tally.Games())
	}
}

func TestRunStopsOnABadOpening(t *testing.T) {
	a := Player{Name: "a", Depth: 1}

	if _, err := Run(a, a, [][]string{strings.Fields("e2e5")}, 10); err == nil {
		t.Fatal("a bad opening must stop the match")
	}
}

func TestSetupChangesEngineFeatures(t *testing.T) {
	old := engine.UseLMR

	t.Cleanup(func() { engine.UseLMR = old })

	off := Player{Depth: 1, Setup: func() { engine.UseLMR = false }}
	on := Player{Depth: 1, Setup: func() { engine.UseLMR = true }}

	engine.UseLMR = true

	if _, err := Play(off, off, strings.Fields("e2e4 e7e5"), 4); err != nil {
		t.Fatal(err)
	}

	if engine.UseLMR {
		t.Fatal("the player Setup should have switched LMR off")
	}

	if _, err := Play(on, on, strings.Fields("e2e4 e7e5"), 4); err != nil {
		t.Fatal(err)
	}

	if !engine.UseLMR {
		t.Fatal("the player Setup should have switched LMR on")
	}
}

func TestTallyGames(t *testing.T) {
	if (Tally{Wins: 3, Draws: 2, Losses: 4}).Games() != 9 {
		t.Fatal("Games must add up all results")
	}
}
