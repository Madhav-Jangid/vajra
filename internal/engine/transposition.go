package engine

import "vajra/internal/chess"

type TTFlag uint8

const (
	TTExact TTFlag = iota
	TTLowerBound
	TTUpperBound
)

type TTEntry struct {
	Key      uint64
	Valid    bool
	Depth    int
	Score    int
	Flag     TTFlag
	BestMove chess.Move
}

type TranspositionTable struct {
	entries []TTEntry
}

func NewTranspositionTable(sizeMB int) *TranspositionTable {
	if sizeMB < 1 {
		sizeMB = 1
	}

	entrySize := 64

	count := (sizeMB * 1024 * 1024) / entrySize

	if count < 1 {
		count = 1
	}

	return &TranspositionTable{
		entries: make([]TTEntry, count),
	}
}

func (tt *TranspositionTable) Clear() {
	for i := range tt.entries {
		tt.entries[i] = TTEntry{}
	}
}

func (tt *TranspositionTable) index(key uint64) int {
	return int(key % uint64(len(tt.entries)))
}

func (tt *TranspositionTable) Probe(key uint64) (TTEntry, bool) {
	if tt == nil || len(tt.entries) == 0 {
		return TTEntry{}, false
	}

	entry := tt.entries[tt.index(key)]

	if !entry.Valid || entry.Key != key {
		return TTEntry{}, false
	}

	return entry, true
}

func (tt *TranspositionTable) Store(
	key uint64,
	depth int,
	score int,
	flag TTFlag,
	bestMove chess.Move,
) {
	if tt == nil || len(tt.entries) == 0 {
		return
	}

	index := tt.index(key)
	old := tt.entries[index]

	if old.Valid && old.Key == key && old.Depth > depth {
		return
	}

	tt.entries[index] = TTEntry{
		Key:      key,
		Valid:    true,
		Depth:    depth,
		Score:    score,
		Flag:     flag,
		BestMove: bestMove,
	}
}
