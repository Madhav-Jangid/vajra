package engine

import (
	"sync"

	"vajra/internal/chess"
)

type TTFlag uint8

const (
	TTExact TTFlag = iota
	TTLowerBound
	TTUpperBound
)

type TTEntry struct {
	Key      uint64
	Valid    bool
	Gen      uint32
	Depth    int
	Score    int
	Flag     TTFlag
	BestMove chess.Move
}

const ttStripes = 1024

type TranspositionTable struct {
	entries []TTEntry
	gen     uint32
	locks   [ttStripes]sync.Mutex
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
		gen:     1,
	}
}

func (tt *TranspositionTable) Clear() {
	if tt == nil {
		return
	}
	tt.gen++
	if tt.gen == 0 {
		tt.gen = 1
		for i := range tt.entries {
			tt.entries[i] = TTEntry{}
		}
	}
}

func (tt *TranspositionTable) index(key uint64) int {
	return int(key % uint64(len(tt.entries)))
}

func (tt *TranspositionTable) Probe(key uint64) (TTEntry, bool) {
	if tt == nil || len(tt.entries) == 0 {
		return TTEntry{}, false
	}

	index := tt.index(key)

	tt.locks[index%ttStripes].Lock()
	entry := tt.entries[index]
	tt.locks[index%ttStripes].Unlock()

	if !entry.Valid || entry.Gen != tt.gen || entry.Key != key {
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

	tt.locks[index%ttStripes].Lock()
	defer tt.locks[index%ttStripes].Unlock()

	old := tt.entries[index]

	if old.Valid && old.Gen == tt.gen && old.Key == key && old.Depth > depth {
		return
	}

	tt.entries[index] = TTEntry{
		Key:      key,
		Valid:    true,
		Gen:      tt.gen,
		Depth:    depth,
		Score:    score,
		Flag:     flag,
		BestMove: bestMove,
	}
}

func (tt *TranspositionTable) HashFull() int {
	if tt == nil || len(tt.entries) == 0 {
		return 0
	}
	sample := len(tt.entries)
	if sample > 1000 {
		sample = 1000
	}
	used := 0
	for i := 0; i < sample; i++ {
		tt.locks[i%ttStripes].Lock()
		e := tt.entries[i]
		tt.locks[i%ttStripes].Unlock()

		if e.Valid && e.Gen == tt.gen {
			used++
		}
	}
	return (used * 1000) / sample
}
