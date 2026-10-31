package ltree

// NOTE: листья, чьи значения == nil || len == 0,
// не участвуют в "турнире" - они приравниваются к
// nil (inf value).

func Init(leafs [][]int) (*LoserTree, *node) {
	if leafs == nil || len(leafs) == 0 {
		return nil, nil
	}

	lt := initStruct(leafs)

	holdTournament(lt)

	return lt, lt.winner
}

func initStruct(leafs [][]int) *LoserTree {
	// Вычисляем длину начального уровня
	// вместе с +inf значениями
	extendedLeafs := 1
	for extendedLeafs < len(leafs) {
		extendedLeafs <<= 1
	}

	lt := LoserTree{
		nodes: make([]*node, 0, extendedLeafs*2),
		// Все inf ссылаются на одну ноду
		arena: make([]node, 0, len(leafs)+1),
	}

	// Создаем значащие ноды
	// Для создания не более одной nil-node в арене
	for i, v := range leafs {
		// Валидация листьев
		// Пропускаем несравниваемые,
		// их место займут node{val: nil}
		if v == nil || len(v) == 0 {
			continue
		}
		lt.arena = append(lt.arena, node{})
		n := &lt.arena[len(lt.arena)-1]
		n.val, n.cur, n.ind = v[1:], v[0], i
		lt.nodes = append(lt.nodes, n)
	}

	// Дополняем уровень +inf значениями
	// Создаем ноду, на которую сслыаются
	// все заведомо проигравшие
	if len(lt.nodes) < extendedLeafs {
		lt.arena = append(lt.arena, node{})
	}

	// NOTE: Не указывем индекс.
	for len(lt.nodes) < extendedLeafs {
		lt.nodes = append(lt.nodes, &lt.arena[len(lt.arena)-1])
	}
	return &lt
}

// holdTournament зполняет уровни дерева, устанавливает победителя
// Ожидает, что len(ult.nodes) == 1
func holdTournament(lt *LoserTree) {
	rounders := lt.nodes
	winners := make([]*node, 0, len(lt.nodes)/2)

	for len(rounders) > 1 {
		round(rounders, &winners, &lt.nodes)

		// NOTE: можно использовать один слайс для чтения && записи
		// т.к. append значения в winners после
		// прочтения этого значения
		rounders = winners
		winners = winners[:0]
	}
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
