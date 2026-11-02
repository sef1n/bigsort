package ltree

import "math"

func (t *LoserTree) Peek() (val int, ind leafInd, ok bool) {
	if t == nil {
		return
	}
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
	// По контракту слайс t.nodes != nil, len(t.nodes) != 0
	// t.winner != nil
	if t == nil || t.done {
		return
	}

	winner := t.winner

	leafs := t.leafs
	nodes := t.nodes

	levelPtr := (len(nodes) + 1) / 2
	levelSize := levelPtr >> 1
	indShifted := winner >> 1
	oppInd := levelPtr + indShifted

	// update после инициализации значений
	if alive {
		leafs[winner].v = newval
	} else {
		leafs[winner].alive = false
		leafs[winner].v = math.MaxInt
	}

	var opp leafInd
	for oppInd < len(t.nodes) {
		// battle
		// Проигравший сразу записывается в дерево
		opp = nodes[oppInd]
		winner, nodes[oppInd] = battle(leafs[winner], leafs[opp], winner, opp)

		levelPtr += levelSize
		levelSize >>= 1
		indShifted >>= 1
		oppInd = levelPtr + indShifted
	}
	if !leafs[winner].alive {
		t.done = true
		return
	}
	t.winner = winner
	return winner, true
}
