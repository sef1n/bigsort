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
	ind int
}

// LoserTree дерево
type LoserTree struct {
	levels [][]*node
	winner node
}

// Next возвращает следующее значение в слайсе.
// Если данные закончились, устанавливает слайс = nil.
func (n *node) Next() {
	if n.val == nil {
		return
	}
	// Слайс хранит cur + next
	if len(n.val) < 2 {
		n.val = nil
		return
	}
	n.val = n.val[1:]
}
