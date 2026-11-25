package book

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"

	"vajra/internal/chess"
	"vajra/internal/uci"
)

type entry struct {
	move   chess.Move
	weight int
}

type Book struct {
	positions map[string][]entry
	rng       *rand.Rand
}

func Parse(r io.Reader, seed int64) (*Book, error) {
	b := &Book{positions: map[string][]entry{}, rng: rand.New(rand.NewSource(seed))}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for number := 1; scanner.Scan(); number++ {
		moves := strings.Fields(scanner.Text())
		if len(moves) == 0 {
			continue
		}

		if err := b.addLine(moves); err != nil {
			return nil, fmt.Errorf("line %d: %w", number, err)
		}
	}

	return b, scanner.Err()
}

func Load(path string, seed int64) (*Book, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Parse(f, seed)
}

func (b *Book) addLine(moves []string) error {
	pos, err := chess.ParseFEN(chess.StartingFEN)
	if err != nil {
		return err
	}

	type step struct {
		key  string
		move chess.Move
	}

	steps := make([]step, 0, len(moves))

	for _, text := range moves {
		move, err := uci.ParseMove(&pos, text)
		if err != nil {
			return err
		}

		steps = append(steps, step{pos.Key(), move})

		pos.MakeMove(move)
	}

	for _, s := range steps {
		b.add(s.key, s.move)
	}

	return nil
}

func (b *Book) add(key string, move chess.Move) {
	entries := b.positions[key]

	for i := range entries {
		if entries[i].move == move {
			entries[i].weight++

			return
		}
	}

	b.positions[key] = append(entries, entry{move: move, weight: 1})
}

func (b *Book) Len() int {
	return len(b.positions)
}

func (b *Book) Probe(p *chess.Position) (chess.Move, bool) {
	entries := b.positions[p.Key()]
	if len(entries) == 0 {
		return chess.Move{}, false
	}

	total := 0

	for _, e := range entries {
		total += e.weight
	}

	pick := b.rng.Intn(total)

	for _, e := range entries {
		if pick < e.weight {
			return e.move, true
		}

		pick -= e.weight
	}

	return entries[len(entries)-1].move, true
}
