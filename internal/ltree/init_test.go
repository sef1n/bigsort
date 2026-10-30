package ltree

import (
	"reflect"
	"testing"
)

type initTest struct {
	name  string
	inSeq [][]int
	want  *LoserTree
}

func TestInit(t *testing.T) {
	tests := []initTest{
		// Пустые входные данные.
		{name: "nil input", inSeq: nil, want: nil},
		{name: "empty slice", inSeq: [][]int{}, want: nil},

		// Незначащие значения на входе.
		{
			// k=1: паддинга нет, единственный участник - победитель
			name: "k=1 nil leaf", inSeq: [][]int{nil},
			want: &LoserTree{
				levels: [][]*node{
					{
						{val: nil, ind: 0},
					},
				},
				winner: node{val: nil, ind: 0},
			},
		},
		{
			// k=3: паддинг до 4.
			name:  "k=3 all nil",
			inSeq: [][]int{nil, nil, nil},
			want: &LoserTree{
				levels: [][]*node{
					{{val: nil, ind: 0}, {val: nil, ind: 0}, {val: nil, ind: 0}, {val: nil, ind: 0}},
					{{val: nil, ind: 0}, {val: nil, ind: 0}},
					{{val: nil, ind: 0}},
				},
				winner: node{val: nil, ind: 0},
			},
		},

		{
			// k=1: паддинга нет, единственный участник - победитель
			name: "k=1 len of leaf == 0", inSeq: [][]int{{}},
			want: &LoserTree{
				levels: [][]*node{
					{
						{val: nil, ind: 0},
					},
				},
				winner: node{val: nil, ind: 0},
			},
		},
		{
			// k=3: паддинг до 4.
			name:  "k=3 len of all leafs == 0",
			inSeq: [][]int{{}, {}, {}},
			want: &LoserTree{
				levels: [][]*node{
					{{val: nil, ind: 0}, {val: nil, ind: 0}, {val: nil, ind: 0}, {val: nil, ind: 0}},
					{{val: nil, ind: 0}, {val: nil, ind: 0}},
					{{val: nil, ind: 0}},
				},
				winner: node{val: nil, ind: 0},
			},
		},

		// Валидные данные на входе.
		{
			// k=1: паддинга нет, holdTournament не делает ни одного раунда.
			name:  "k=1 single leaf",
			inSeq: [][]int{{42}},
			want: &LoserTree{
				levels: [][]*node{
					{{val: []int{42}, ind: 0}},
				},
				winner: node{val: []int{42}, ind: 0},
			},
		},

		{
			// k=2 sorted: [1] < [2]. Loser — [2].
			name:  "k=2 sorted",
			inSeq: [][]int{{1}, {2}},
			want: &LoserTree{
				levels: [][]*node{
					{{val: []int{1}, ind: 0}, {val: []int{2}, ind: 1}},
					{{val: []int{2}, ind: 1}},
				},
				winner: node{val: []int{1}, ind: 0},
			},
		},

		{
			// k=2 reversed: [2] > [1]. Loser — [2]@0, winner — [1]@1.
			name:  "k=2 reversed",
			inSeq: [][]int{{2}, {1}},
			want: &LoserTree{
				levels: [][]*node{
					{{val: []int{2}, ind: 0}, {val: []int{1}, ind: 1}},
					{{val: []int{2}, ind: 0}},
				},
				winner: node{val: []int{1}, ind: 1},
			},
		},

		{
			// k=3 → паддинг до 4.
			// R1: (0,1)→win[2]@1 lose[3]@0; (2,3)→win[1]@2 lose nil@0
			// R2: ([2]@1,[1]@2)→win[1]@2 lose[2]@1
			name:  "k=3 padded to 4",
			inSeq: [][]int{{3}, {2}, {1}},
			want: &LoserTree{
				levels: [][]*node{
					{
						{val: []int{3}, ind: 0},
						{val: []int{2}, ind: 1},
						{val: []int{1}, ind: 2},
						{val: nil, ind: 0},
					},
					{
						{val: []int{3}, ind: 0},
						{val: nil, ind: 0},
					},
					{
						{val: []int{2}, ind: 1},
					},
				},
				winner: node{val: []int{1}, ind: 2},
			},
		},

		{
			// k=3: паддинг до 4.
			// Сортировка стабильна: победитель - первый элемент
			// с этим значением
			name:  "k=3 equal vals",
			inSeq: [][]int{{1}, {1}, {1}},
			want: &LoserTree{
				levels: [][]*node{
					{
						{val: []int{1}, ind: 0},
						{val: []int{1}, ind: 1},
						{val: []int{1}, ind: 2},
						{val: nil, ind: 0},
					},
					{
						{val: []int{1}, ind: 1},
						{val: nil, ind: 0},
					},
					{
						{val: []int{1}, ind: 2},
					},
				},
				winner: node{val: []int{1}, ind: 0},
			},
		},

		{
			// k=4: R1 losers [4]@0,[2]@2; R2 loser [3]@1; winner [1]@3.
			name:  "k=4",
			inSeq: [][]int{{4}, {3}, {2}, {1}},
			want: &LoserTree{
				levels: [][]*node{
					{
						{val: []int{4}, ind: 0},
						{val: []int{3}, ind: 1},
						{val: []int{2}, ind: 2},
						{val: []int{1}, ind: 3},
					},
					{
						{val: []int{4}, ind: 0},
						{val: []int{2}, ind: 2},
					},
					{
						{val: []int{3}, ind: 1},
					},
				},
				winner: node{val: []int{1}, ind: 3},
			},
		},

		{
			// k=7 → паддинг до 8. Совпадает с выводом cmd/test
			// ([1 1]@0, [2 2]@1, [3 3]@2, [4 4]@3, [5 5]@4, [6 6]@5, [7]@6, nil@0).
			name: "k=7 padded to 8",
			inSeq: [][]int{
				{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}, {6, 6}, {7},
			},
			want: &LoserTree{
				levels: [][]*node{
					{
						{val: []int{1, 1}, ind: 0},
						{val: []int{2, 2}, ind: 1},
						{val: []int{3, 3}, ind: 2},
						{val: []int{4, 4}, ind: 3},
						{val: []int{5, 5}, ind: 4},
						{val: []int{6, 6}, ind: 5},
						{val: []int{7}, ind: 6},
						{val: nil, ind: 0},
					},
					{
						{val: []int{2, 2}, ind: 1},
						{val: []int{4, 4}, ind: 3},
						{val: []int{6, 6}, ind: 5},
						{val: nil, ind: 0},
					},
					{
						{val: []int{3, 3}, ind: 2},
						{val: []int{7}, ind: 6},
					},
					{
						{val: []int{5, 5}, ind: 4},
					},
				},
				winner: node{val: []int{1, 1}, ind: 0},
			},
		},

		{
			// battle при равенстве возвращает i как winner,
			// то есть стабильный tie-break по меньшему leafInd.
			name:  "k=4 all equal, stable tie-break",
			inSeq: [][]int{{5}, {5}, {5}, {5}},
			want: &LoserTree{
				levels: [][]*node{
					{
						{val: []int{5}, ind: 0},
						{val: []int{5}, ind: 1},
						{val: []int{5}, ind: 2},
						{val: []int{5}, ind: 3},
					},
					{
						{val: []int{5}, ind: 1},
						{val: []int{5}, ind: 3},
					},
					{
						{val: []int{5}, ind: 2},
					},
				},
				winner: node{val: []int{5}, ind: 0},
			},
		},

		{
			// Less смотрит только на [0]; хвосты не влияют на порядок.
			name:  "same head, stable tie-break by leaf order",
			inSeq: [][]int{{5, 10}, {5, 20}, {5, 30}},
			want: &LoserTree{
				levels: [][]*node{
					{
						{val: []int{5, 10}, ind: 0},
						{val: []int{5, 20}, ind: 1},
						{val: []int{5, 30}, ind: 2},
						{val: nil, ind: 0},
					},
					{
						{val: []int{5, 20}, ind: 1},
						{val: nil, ind: 0},
					},
					{
						{val: []int{5, 30}, ind: 2},
					},
				},
				winner: node{val: []int{5, 10}, ind: 0},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _ := Init(test.inSeq)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Init() tree mismatch\n got: %+v\nwant: %+v", got, test.want)
			}
		})
	}
}
