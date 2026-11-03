package ltree

import (
	"math"
)

func (t *LoserTree) Peek() (val int, ind leafInd, ok bool) {
	w := t.winner
	if !t.leafs[w].alive {
		return
	}
	return t.leafs[w].v, w, true
}

// Pop обновляет значение победителя
// и путь от победителя до корня.
// Если значение победителя невалидно,
// возвращает cur = 0, ind = 0, ok = false
func (t *LoserTree) Pop(newval int, alive bool) (ind leafInd, ok bool) {
	leafs := t.leafs
	winner := t.winner
	nodes := t.nodes

	if alive {
		leafs[winner].v = newval
	} else {
		leafs[winner].alive = false
		leafs[winner].v = math.MaxInt

	}
	for i := (winner + len(leafs)) >> 1; i > 0; i >>= 1 {
		opp := nodes[i]
		winner, nodes[i] = battle(leafs[winner], leafs[opp], winner, opp)
	}

	if !leafs[winner].alive {
		return
	}
	t.winner = winner
	return winner, true
}
