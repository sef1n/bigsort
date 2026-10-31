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

// node узел с значением и его индексом на нижнем уровне
type node struct {
	val []int
	cur int
	ind int
}

// LoserTree дерево
type LoserTree struct {
	// Само дерево
	nodes []*node
	// arena - арена для нод
	// все аллоцируются в одно место
	arena []node

	winner *node
}

// Next возвращает следующее значение в слайсе.
// Если данные закончились, устанавливает слайс = nil.
func (n *node) Next() {
	if len(n.val) == 0 {
		n.val = nil
		return
	}
	n.cur = n.val[0]
	n.val = n.val[1:]
}
