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

	origInd := winner.ind // читается один раз
	levelPtr := t.leafsN
	levelSize := t.leafsN >> 1
	indShifted := origInd >> 1
	oppInd := levelPtr + indShifted

	for oppInd < len(t.nodes) {
		// fmt.Printf("Opp ind [%v]\n", oppInd)
		// fmt.Printf("Participant %v; opponent %v\n", winner, opponent)

		// battle
		// Participant хранит значение победителя
		// Проигравший сразу записывается в дерево
		winner, t.nodes[oppInd] = battle(winner, t.nodes[oppInd])
		// fmt.Printf("Battle res: winner %v, loser %v\n", winner, loser)

		levelPtr += levelSize
		levelSize >>= 1
		indShifted >>= 1
		oppInd = levelPtr + indShifted

		// fmt.Printf("Afterround tree %+v\n", t)
		// fmt.Println()
	}
	t.winner = winner

	if t.winner.val == nil {
		return nil, false
	}
	return winner, true
}
