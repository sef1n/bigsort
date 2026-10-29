package ltree

// Pop обновляет значение победителя,
// если новое значение больше прошлого,
// обновляет дерево. В ином случае дерево
// остается прежним, новое значение возвращается
func Pop(t *LoserTree) (node, bool) {
	winner := t.levels[0][t.winner.leafInd]
	if winner.val == nil {
		// Бесконечность в корне - значения исчерпаны
		return node{}, false
	}
	// Устанавливаем новое значение
	winner.Next()

	var loser *node
	for i := 1; i < len(t.levels); i++ {
		oppInd := winner.leafInd >> i
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
	return *winner, true
}
