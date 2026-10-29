package ltree

// Pop обновляет значение победителя
// и путь от победителя до корня.
// Если значение победителя невалидно,
// возвращает ok = false
func Pop(t *LoserTree) (winner *node, ok bool) {
	winner = t.levels[0][t.winner.ind]
	// Устанавливаем новое значение
	winner.Next()

	var loser *node
	for i := 1; i < len(t.levels); i++ {
		oppInd := winner.ind >> i
		// fmt.Printf("Opp ind [%v][%v]\n", i, oppInd)
		opponent := t.levels[i][oppInd]
		// fmt.Printf("Participant %v; opponent %v\n", winner, opponent)

		// battle
		// Participant хранит значение победителя
		// Проигравший сразу записывается в дерево
		winner, loser = battle(winner, opponent)
		// fmt.Printf("Battle res: winner %v, loser %v\n", winner, loser)

		t.levels[i][oppInd] = loser

		// fmt.Printf("Afterround tree %+v\n", t)
		// fmt.Println()
	}
	t.winner = *winner

	if t.winner.val == nil {
		return nil, false
	}
	return winner, true
}
