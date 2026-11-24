package engine

import (
	"sort"
	"vajra/internal/chess"
)

var UseHistory = true

const maxHistory = 1000

type HistoryTable struct {
	scores [2][64][64]int
}

func (h *HistoryTable) get(color chess.Color, m chess.Move) int {
	return h.scores[color][m.From][m.To]
}

func (h *HistoryTable) update(color chess.Color, m chess.Move, depth int) {
	s := &h.scores[color][m.From][m.To]

	*s += depth * depth

	if *s > maxHistory {
		*s = maxHistory
	}
}

func isQuiet(m chess.Move) bool {
	return m.Flags&(chess.Capture|chess.Promotion) == 0
}

func orderMoves(moves []chess.Move) {
	sort.Slice(moves, func(i, j int) bool {
		return moveScore(moves[i]) > moveScore(moves[j])
	})
}

var UseKillers = true

const killerBonus = 2000

func (k *KillerTable) isKiller(ply int, m chess.Move) int {
	if ply < 0 || ply >= MaxPly {
		return -1
	}

	for slot := 0; slot < 2; slot++ {
		if sameMove(k.moves[ply][slot], m) {
			return slot
		}
	}

	return -1
}

func (k *KillerTable) store(ply int, m chess.Move) {
	if ply < 0 || ply >= MaxPly {
		return
	}

	if sameMove(k.moves[ply][0], m) {
		return
	}

	k.moves[ply][1] = k.moves[ply][0]
	k.moves[ply][0] = m
}

func orderMovesHeuristic(
	moves []chess.Move,
	color chess.Color,
	ply int,
	state *SearchState,
) {
	if state == nil || (!UseHistory && !UseKillers) {
		orderMoves(moves)
		return
	}

	score := func(m chess.Move) int {
		if !isQuiet(m) {
			return moveScore(m)
		}

		s := 0

		if UseKillers {
			if slot := state.killers.isKiller(ply, m); slot >= 0 {
				s += killerBonus - slot
			}
		}

		if UseHistory {
			s += state.history.get(color, m)
		}

		return s
	}

	sort.SliceStable(moves, func(i, j int) bool {
		return score(moves[i]) > score(moves[j])
	})
}
