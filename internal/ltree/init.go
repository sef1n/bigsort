package ltree

// NOTE: листья, чьи значения == nil || len == 0,
// не участвуют в "турнире" - они приравниваются к
// nil (inf value).

func Init(leafs [][]int) (*LoserTree, node) {
	if leafs == nil || len(leafs) == 0 {
		return nil, node{}
	}

	lt := LoserTree{levels: createTopLevel(leafs)}
	holdTournament(&lt)

	return &lt, lt.winner
}

func createTopLevel(leafs [][]int) [][]*node {
	// Вычисляем длину начального уровня
	// вместе с +inf значениями
	extendedLeafs := 1
	for extendedLeafs < len(leafs) {
		extendedLeafs <<= 1
	}

	lvls := make([][]*node, 0, extendedLeafs/2)
	lvls = append(lvls, make([]*node, 0, extendedLeafs))

	// Создаем значащие ноды
	for i, v := range leafs {
		// Валидация листьев
		// Пропускаем несравниваемые,
		// их место займут node{val: nil}
		if v == nil || len(v) == 0 {
			continue
		}
		lvls[0] = append(lvls[0], &node{val: v, ind: i})
	}

	// Дополняем уровень +inf значениями
	// WARN: Не указывем индекс.
	for len(lvls[0]) < extendedLeafs {
		lvls[0] = append(lvls[0], &node{val: nil})
	}

	return lvls
}

func complete(lev *[]*node) {
	n := 1
	for n < len(*lev) {
		n <<= 1
	}
	for len(*lev) < n {
		*lev = append(*lev, &node{val: nil, ind: len(*lev)})
	}
}

// holdTournament зполняет уровни дерева, устанавливает победителя
// Ожидает, что len(ult.levels) == 1
func holdTournament(lt *LoserTree) {
	rounders := lt.levels[0]
	winners := make([]*node, 0, len(lt.levels[0])/2)

	// depth - куда пишутся проигравшие
	for depth := 1; len(rounders) > 1; depth++ {
		// fmt.Printf("Tournament round start, rounders: ")
		// printPointersSlice(rounders)

		lt.levels = append(lt.levels, make([]*node, 0, len(rounders)/2))
		round(rounders, &winners, &lt.levels[depth])

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
	lt.winner = *rounders[0]
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
		winner = j
		loser = i
		return
	}
	if j.val == nil {
		winner = i
		loser = j
		return
	}

	if j.val[0] < i.val[0] {
		return j, i
	}
	// stable sort
	return i, j
}
