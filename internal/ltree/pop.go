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

	levelPtr := t.leafsN
	oppInd := levelPtr + (winner.ind >> 1)
	levelSize := t.leafsN >> 1 // размер 1-го уровня

	for i := 2; oppInd < len(t.nodes); i++ {
		// fmt.Printf("Opp ind [%v]\n", oppInd)
		// fmt.Printf("Participant %v; opponent %v\n", winner, opponent)

		// battle
		// Participant хранит значение победителя
		// Проигравший сразу записывается в дерево
		winner, t.nodes[oppInd] = battle(winner, t.nodes[oppInd])
		// fmt.Printf("Battle res: winner %v, loser %v\n", winner, loser)

		levelPtr += levelSize // переход к следующему уровню
		levelSize >>= 1
		oppInd = levelPtr + (winner.ind >> i)

		// fmt.Printf("Afterround tree %+v\n", t)
		// fmt.Println()
	}
	t.winner = winner

	if t.winner.val == nil {
		return nil, false
	}
	return winner, true
}
