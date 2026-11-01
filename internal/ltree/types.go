package ltree

import "math"

// type Luser interface {
// 	// Следующее значение
// 	// Обновляет возвращаемое Val значение
// 	Next() (val int, ok bool)
//
//	// Сравнивает Val с другим Luser или nil.
//	// Возвращает меньшее и большее значение.
//	// Если значения равны, первым возвращает сове значение,
//	// вторым переданное.
// 	CompareWith(i Luser) (less Luser, bigger Luser)
// }

// node узел с значением и его индексом на нижнем уровне
type node struct {
	// Значение для сравнения
	cur int
	// Индекс листа в t.leafs
	ind int
}

// LoserTree дерево
type LoserTree struct {
	// Само дерево
	nodes []*node

	// Листья
	leafs [][]int

	// arena - арена для нод
	// все аллоцируются в одно место
	arena []node

	winner *node
}

// Next обновляет значения переданной ноды.
// Если данные в листе закончились, устанавливает
// {val: nil, cur: MaxInt, ind: -1}
func (t *LoserTree) Next(n *node) {
	valPtr := &t.leafs[n.ind]
	val := *valPtr
	if len(val) == 0 {
		n.cur = math.MaxInt
		n.ind = -1
		return
	}
	n.cur = val[0]
	*valPtr = (*valPtr)[1:]
}
