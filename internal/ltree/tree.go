package ltree

import "fmt"

// Leaf лист дерева
type leaf []int

// node узел с победителем и его индексом на нижнем уровне
type node struct {
	val     *leaf
	leafInd int
}

// LoserTree дерево
type LoserTree struct {
	levels [][]*node
	winner node
}

func Init(leafs []*leaf) *LoserTree {
	// leafs completion to len%2=0
	if leafs == nil || len(leafs) == 0 {
		return nil
	}
	lt := LoserTree{levels: createTopLevel(leafs)}
	holdTournament(&lt)
	return &lt
}

func createTopLevel(leafs []*leaf) [][]*node {
	lev := make([][]*node, 1, len(leafs))
	for i, v := range leafs {
		lev[0] = append(lev[0], &node{val: v, leafInd: i})
	}
	complete(&lev[0])
	return lev
}

func complete(lev *[]*node) {
	n := 1
	for n < len(*lev) {
		n <<= 1
	}
	for len(*lev) < n {
		*lev = append(*lev, &node{val: nil, leafInd: len(*lev)})
	}
}

// // TODO: interface
// type Competitor interface {
// 	// i может быть nil.
// 	CompareWith(i Competitor) (less Competitor, bigger Competitor)
// }

// holdTournament зполняет уровни дерева, устанавливает победителя
// Ожидает, что len(ult.levels) == 1
func holdTournament(lt *LoserTree) {
	rounders := lt.levels[0]
	winners := make([]*node, 0, len(lt.levels)/2)

	// depth - куда пишутся проигравшие
	for depth := 1; len(rounders) != 1; depth++ {
		fmt.Printf("Tournament round start, rounders: ")
		printPointersSlice(rounders)

		lt.levels = append(lt.levels, make([]*node, 0, len(winners)/2))
		round(rounders, &winners, &lt.levels[depth])

		// NOTE: можно использовать один слайс для чтения && записи
		// т.к. append значения в winners после
		// прочтения этого значения
		rounders = winners
		winners = winners[:0]

		fmt.Printf("Tournament round end, winners: ")
		printPointersSlice(rounders)
		fmt.Println()
	}
	fmt.Printf("Tournament end, winner: %v\n", rounders)
	lt.winner = *rounders[0]
	return
}

// round обрабатывает один уровень дерева.
// Записывает победителей в winners, проигравших в losers
func round(rouneders []*node, winners *[]*node, losers *[]*node) {
	for i, j := 0, 1; j < len(rouneders); i, j = i+2, j+2 {
		winner, loser := battle(rouneders[i], rouneders[j])
		fmt.Printf("Winner: %v; Loser: %v\n", *winner, *loser)

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

	if Less(*j.val, *i.val) {
		return j, i
	}
	// stable sort
	return i, j
}

func Less(j, i leaf) bool {
	return j[0] < i[0]
}
