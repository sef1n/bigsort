package ltree

import "math"

func (t *LoserTree) Peek() (int, bool) {
	w := t.winner
	return w.I, w.valid
}

func (t *LoserTree) Upd(ind, new int) {
	t.leafs[ind].v = new
}

func (t *LoserTree) Kill(ind int) {
	t.leafs[ind] = leaf{v: math.MaxInt}
}

// Pop обновляет значение победителя
// и путь от победителя до корня.
// Если значение победителя невалидно,
// возвращает cur = 0, ind = 0, ok = false
func (t *LoserTree) Pop() (ind leafInd, ok bool) {
	// NOTE: без проверки выигрыш на малых k
	// if !t.winner.valid {
	// 	return 0, false
	// }

	leafs := t.leafs
	winnerS := t.winner
	nodes := t.nodes

	winner := winnerS.I

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
