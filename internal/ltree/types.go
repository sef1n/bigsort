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

type leaf struct {
	v     int
	alive bool
}

type Winner struct {
	I     leafInd
	valid bool
}

type leafInd = int

// LoserTree дерево
type LoserTree struct {
	// Дерево из индексов листьев
	nodes []leafInd

	// Листья
	leafs []leaf

	winner Winner
}
