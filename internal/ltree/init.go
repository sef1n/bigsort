package ltree

import (
	"math"
)

// NOTE: листья, чьи значения == nil || len == 0,
// не участвуют в "турнире" - они приравниваются к
// nil (inf value).
//
// NOTE: leafs с (cap == len + 1) ускоряют инициализацию
//
// NOTE: можно удалить слайс t.leaf, но тогда усложнится battle
//
// TODO: функция создания слайса листев (для cap+1)
//
// NOTE: если хранить статус листа через другую структуру ноды,
// то нужен указатель. Текущая реализация быстрее?

func Init(values []int) *LoserTree {
	if values == nil || len(values) == 0 {
		return nil
	}
	lt, ext := initiateStruct(values)

	leafs := lt.leafs
	winners := make([]leafInd, ext*2)
	copy(winners[ext:], lt.nodes[ext:])

	for p := ext - 1; p >= 1; p-- {
		i, j := winners[p*2], winners[p*2+1]
		winner, loser := battle(leafs[i], leafs[j], i, j)

		winners[p] = winner
		lt.nodes[p] = loser
	}
	lt.winner = winners[1]

	return lt
}

func initiateStruct(values []int) (*LoserTree, int) {
	// Вычисляем длину начального уровня
	// вместе с +inf значениями
	ext := 1
	for ext < len(values) {
		ext <<= 1
	}

	lt := LoserTree{
		leafs: make([]leaf, ext),
		nodes: make([]leafInd, ext*2),
	}

	// Переносим значения в lt.leafs
	for i, val := range values {
		lt.leafs[i].v = val
		lt.leafs[i].alive = true
	}

	// Дополняем lt.leafs inf
	for i := len(values); i < ext; i++ {
		lt.leafs[i].v = math.MaxInt
	}

	// Переносим индексы в дерево
	for i := 0; i < ext; i++ {
		lt.nodes[ext+i] = i
	}
	return &lt, ext
}

// battle сравнивает значения. Объявляет победителем меньшее значение
// Значения с ind == -1 считаются бесконечно большими и могут проиграть
// только таким же значениям.
// Функция нестабильна при сортировке двух бесконечно больших значений -
// первый параметр не обязательно будет победителем.
func battle(i, j leaf, ii, ji leafInd) (winner, loser leafInd) {
	if j.v < i.v {
		return ji, ii
	}

	if !i.alive {
		return ji, ii
	}

	return ii, ji
}
