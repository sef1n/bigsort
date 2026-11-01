package ltree

import (
	"math"
)

// Pop обновляет значение победителя
// и путь от победителя до корня.
// Если значение победителя невалидно,
// возвращает winner = nil, ok = false
func Pop(t *LoserTree) (winner *node, ok bool) {
	// По контракту слайс t.nodes != nil, len(t.nodes) != 0
	// t.winner != nil
	if t == nil || t.winner.ind < 0 {
		return
	}

	winner = t.winner
	origInd := t.winner.ind
	t.Next(winner)

	levelPtr := (len(t.nodes) + 1) / 2
	levelSize := levelPtr >> 1
	indShifted := origInd >> 1
	oppInd := levelPtr + indShifted

	for oppInd < len(t.nodes) {
		// battle
		// Проигравший сразу записывается в дерево
		winner, t.nodes[oppInd] = battle(winner, t.nodes[oppInd])

		levelPtr += levelSize
		levelSize >>= 1
		indShifted >>= 1
		oppInd = levelPtr + indShifted
	}
	t.winner = winner

	if t.winner.cur == math.MaxInt {
		return nil, false
	}
	return winner, true
}
