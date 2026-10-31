package ltree

// Pop обновляет значение победителя
// и путь от победителя до корня.
// Если значение победителя невалидно,
// возвращает ok = false
func Pop(t *LoserTree) (winner *node, ok bool) {
	// По контракту слайс t.nodes != nil, len(t.nodes) != 0
	if t == nil {
		return
	}

	winner = t.winner
	winner.Next()

	origInd := winner.ind    // читается один раз
	levelPtr := len(t.arena) // фактическое кол-во нод == кол-ву листьев
	levelSize := len(t.arena) >> 1
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

	if t.winner.val == nil {
		return nil, false
	}
	return winner, true
}
