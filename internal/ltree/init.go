package ltree

// NOTE: листья, чьи значения == nil || len == 0,
// не участвуют в "турнире" - они приравниваются к
// nil (inf value).

func Init(leafs [][]int) (*LoserTree, *node) {
	if leafs == nil || len(leafs) == 0 {
		return nil, nil
	}

	lt := LoserTree{nodes: createTopLevel(leafs)}
	lt.leafsN = len(lt.nodes)

	holdTournament(&lt)

	return &lt, lt.winner
}

func createTopLevel(leafs [][]int) []*node {
	// Вычисляем длину начального уровня
	// вместе с +inf значениями
	extendedLeafs := 1
	for extendedLeafs < len(leafs) {
		extendedLeafs <<= 1
	}

	lvl := make([]*node, 0, extendedLeafs*2)

	// Создаем значащие ноды
	for i, v := range leafs {
		// Валидация листьев
		// Пропускаем несравниваемые,
		// их место займут node{val: nil}
		if v == nil || len(v) == 0 {
			continue
		}
		lvl = append(lvl, &node{val: v[1:], cur: v[0], ind: i})
	}

	// Дополняем уровень +inf значениями
	// WARN: Не указывем индекс.
	for len(lvl) < extendedLeafs {
		lvl = append(lvl, &node{val: nil})
	}

	return lvl
}

// holdTournament зполняет уровни дерева, устанавливает победителя
// Ожидает, что len(ult.nodes) == 1
func holdTournament(lt *LoserTree) {
	rounders := lt.nodes
	winners := make([]*node, 0, lt.leafsN/2)

	// depth - куда пишутся проигравшие
	for len(rounders) > 1 {
		// fmt.Printf("Tournament round start, rounders: ")
		// printPointersSlice(rounders)

		round(rounders, &winners, &lt.nodes)

		// NOTE: можно использовать один слайс для чтения && записи
		// т.к. append значения в winners после
		// прочтения этого значения
		rounders = winners
		winners = winners[:0]

		// fmt.Printf("Tournament round end, winners: ")
		// printPointersSlice(rounders)
		// fmt.Println()
	}
	// fmt.Printf("Tournament end, winner: %v\n", rounders)
	lt.winner = rounders[0]
	return
}

// round обрабатывает один уровень дерева.
// Записывает победителей в winners, проигравших в losers
func round(rouneders []*node, winners *[]*node, losers *[]*node) {
	for i, j := 0, 1; j < len(rouneders); i, j = i+2, j+2 {
		winner, loser := battle(rouneders[i], rouneders[j])

		// fmt.Printf("Winner: %v; Loser: %v\n", *winner, *loser)

		*winners = append(*winners, winner)
		*losers = append(*losers, loser)
	}
}

// battle сравнивает значения. Объявляет победителем меньшее значение
// Принимает nil за +infinity
// WARN: не использует интерфейс
func battle(i, j *node) (winner *node, loser *node) {
	if i.val == nil {
		return j, i
	}
	if j.val == nil {
		return i, j
	}

	if j.cur < i.cur {
		return j, i
	}
	// stable sort
	return i, j
}
