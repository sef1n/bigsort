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

func Init(leafs []int) *LoserTree {
	if leafs == nil || len(leafs) == 0 {
		return nil
	}

	lt := initStruct(leafs)
	holdTournament(lt)

	return lt
}

func initStruct(leafs []int) *LoserTree {
	// Вычисляем длину начального уровня
	// вместе с +inf значениями
	extendedLeafs := 1
	for extendedLeafs < len(leafs) {
		extendedLeafs <<= 1
	}

	lt := LoserTree{
		leafs: make([]leaf, extendedLeafs),
		nodes: make([]leafInd, 0, extendedLeafs*2),
	}

	// Создаем значащие ноды
	for i, val := range leafs {
		lt.leafs[i].v = val
		lt.leafs[i].alive = true

		lt.nodes = append(lt.nodes, i)
	}

	for i := len(lt.nodes); len(lt.nodes) < extendedLeafs; i++ {
		lt.leafs[i].v = math.MaxInt
		lt.nodes = append(lt.nodes, i)
	}
	return &lt
}

// holdTournament зполняет уровни дерева, устанавливает победителя
// Ожидает, что len(ult.nodes) == 1
func holdTournament(lt *LoserTree) {
	leafs := lt.leafs
	rounders := lt.nodes
	winners := make([]leafInd, 0, len(lt.nodes)/2)

	for len(rounders) > 1 {
		var p1, p2 leafInd
		for i, j := 0, 1; j < len(rounders); i, j = i+2, j+2 {
			p1, p2 = rounders[i], rounders[j]
			winner, loser := battle(leafs[p1], leafs[p2], p1, p2)

			winners = append(winners, winner)
			lt.nodes = append(lt.nodes, loser)
		}

		rounders = winners
		winners = winners[:0]
	}
	lt.winner = rounders[0]
	return
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
