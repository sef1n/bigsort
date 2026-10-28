package ltree

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

// Leaf лист дерева
// dev тип
type Leaf []int

// node узел с победителем и его индексом на нижнем уровне
type node struct {
	val     *Leaf
	leafInd int
}

// LoserTree дерево
type LoserTree struct {
	levels [][]*node
	winner node
}

func (l *Leaf) Next() (int, bool) {
	// cur + next
	if len(*l) < 2 {
		return 0, false
	}
	v := (*l)[1]
	*l = (*l)[1:]
	return v, true
}
