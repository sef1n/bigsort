package ltree

import (
	"math"
)

func (t *LoserTree) Peek() (int, bool) {
	w := t.winner
	// w alive если tree.valid
	return w.I, w.valid
}

// Pop обновляет значение победителя
// и путь от победителя до корня.
// Если значение победителя невалидно,
// возвращает cur = 0, ind = 0, ok = false
func (t *LoserTree) Pop(newval int, alive bool) (ind leafInd, ok bool) {
	// NOTE: без проверки выигрыш на малых k
	// if !t.winner.valid {
	// 	return 0, false
	// }

	leafs := t.leafs
	winnerS := t.winner
	nodes := t.nodes

	winner := winnerS.I

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
		t.winner.valid = false
		return
	}
	t.winner.I = winner
	return winner, true
}
