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
type leafInd = int

// LoserTree дерево
type LoserTree struct {
	// Дерево из индексов листьев
	nodes []leafInd

	// Листья
	leafs []leaf

	// TODO: можно не хранить winner?
	winner leafInd
	// Флаг исчерпания значений
	// устанавливается после получения inf
	// в Pop
	// WARN: документация поведения при Update done дерева
	// или функция RECOVER?
	done bool
}
