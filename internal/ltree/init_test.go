package ltree

import (
	"fmt"
	"reflect"
	"testing"
)

// snapshot возвращает копию дерева, у которой обнулено поле arena.
// Сравниваем только логическое состояние (nodes + winner),
// а не внутреннее представление (арена, размеры и т.п.).
func snapshot(t *LoserTree) *LoserTree {
	if t == nil {
		return nil
	}
	cp := *t
	cp.arena = nil
	return &cp
}

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

		// Незначащие на входе.
		{
			name: "k=1 nil leaf", inSeq: [][]int{nil},
			want: &LoserTree{
				nodes:  []*node{{}},
				winner: &node{},
			},
		},
		{
			name: "k=3 all nil", inSeq: [][]int{nil, nil, nil},
			want: &LoserTree{
				nodes: []*node{
					{val: nil, ind: 0},
					{val: nil, ind: 0},
					{val: nil, ind: 0},
					{val: nil, ind: 0},
					{val: nil, ind: 0},
					{val: nil, ind: 0},
					{val: nil, ind: 0},
				},
				winner: &node{val: nil, ind: 0},
			},
		},
		{
			name: "k=1 len of leaf == 0", inSeq: [][]int{{}},
			want: &LoserTree{
				nodes:  []*node{{val: nil, ind: 0}},
				winner: &node{val: nil, ind: 0},
			},
		},
		{
			name: "k=3 len of all leafs == 0", inSeq: [][]int{{}, {}, {}},
			want: &LoserTree{
				nodes: []*node{
					{val: nil, ind: 0},
					{val: nil, ind: 0},
					{val: nil, ind: 0},
					{val: nil, ind: 0},
					{val: nil, ind: 0},
					{val: nil, ind: 0},
					{val: nil, ind: 0},
				},
				winner: &node{val: nil, ind: 0},
			},
		},

		// Валидные данные.
		{
			name: "k=1 single leaf", inSeq: [][]int{{42}},
			want: &LoserTree{
				nodes:  []*node{{val: []int{}, cur: 42, ind: 0}},
				winner: &node{val: []int{}, cur: 42, ind: 0},
			},
		},
		{
			name: "k=2 sorted", inSeq: [][]int{{1}, {2}},
			want: &LoserTree{
				nodes: []*node{
					{val: []int{}, cur: 1, ind: 0},
					{val: []int{}, cur: 2, ind: 1},
					{val: []int{}, cur: 2, ind: 1},
				},
				winner: &node{val: []int{}, cur: 1, ind: 0},
			},
		},
		{
			name: "k=2 reversed", inSeq: [][]int{{2}, {1}},
			want: &LoserTree{
				nodes: []*node{
					{val: []int{}, cur: 2, ind: 0},
					{val: []int{}, cur: 1, ind: 1},
					{val: []int{}, cur: 2, ind: 0},
				},
				winner: &node{val: []int{}, cur: 1, ind: 1},
			},
		},
		{
			name: "k=3 padded to 4", inSeq: [][]int{{3}, {2}, {1}},
			want: &LoserTree{
				nodes: []*node{
					{val: []int{}, cur: 3, ind: 0},
					{val: []int{}, cur: 2, ind: 1},
					{val: []int{}, cur: 1, ind: 2},
					{val: nil, ind: 0},
					{val: []int{}, cur: 3, ind: 0},
					{val: nil, ind: 0},
					{val: []int{}, cur: 2, ind: 1},
				},
				winner: &node{val: []int{}, cur: 1, ind: 2},
			},
		},
		{
			name: "k=3 equal vals", inSeq: [][]int{{1}, {1}, {1}},
			want: &LoserTree{
				nodes: []*node{
					{val: []int{}, cur: 1, ind: 0},
					{val: []int{}, cur: 1, ind: 1},
					{val: []int{}, cur: 1, ind: 2},
					{val: nil, ind: 0},
					{val: []int{}, cur: 1, ind: 1},
					{val: nil, ind: 0},
					{val: []int{}, cur: 1, ind: 2},
				},
				winner: &node{val: []int{}, cur: 1, ind: 0},
			},
		},
		{
			name: "k=4", inSeq: [][]int{{4}, {3}, {2}, {1}},
			want: &LoserTree{
				nodes: []*node{
					{val: []int{}, cur: 4, ind: 0},
					{val: []int{}, cur: 3, ind: 1},
					{val: []int{}, cur: 2, ind: 2},
					{val: []int{}, cur: 1, ind: 3},
					{val: []int{}, cur: 4, ind: 0},
					{val: []int{}, cur: 2, ind: 2},
					{val: []int{}, cur: 3, ind: 1},
				},
				winner: &node{val: []int{}, cur: 1, ind: 3},
			},
		},
		{
			name: "k=7 padded to 8",
			inSeq: [][]int{
				{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}, {6, 6}, {7},
			},
			want: &LoserTree{
				nodes: []*node{
					{val: []int{1}, cur: 1, ind: 0},
					{val: []int{2}, cur: 2, ind: 1},
					{val: []int{3}, cur: 3, ind: 2},
					{val: []int{4}, cur: 4, ind: 3},
					{val: []int{5}, cur: 5, ind: 4},
					{val: []int{6}, cur: 6, ind: 5},
					{val: []int{}, cur: 7, ind: 6},
					{val: nil, ind: 0},
					{val: []int{2}, cur: 2, ind: 1},
					{val: []int{4}, cur: 4, ind: 3},
					{val: []int{6}, cur: 6, ind: 5},
					{val: nil, ind: 0},
					{val: []int{3}, cur: 3, ind: 2},
					{val: []int{}, cur: 7, ind: 6},
					{val: []int{5}, cur: 5, ind: 4},
				},
				winner: &node{val: []int{1}, cur: 1, ind: 0},
			},
		},
		{
			name: "k=4 all equal, stable tie-break", inSeq: [][]int{{5}, {5}, {5}, {5}},
			want: &LoserTree{
				nodes: []*node{
					{val: []int{}, cur: 5, ind: 0},
					{val: []int{}, cur: 5, ind: 1},
					{val: []int{}, cur: 5, ind: 2},
					{val: []int{}, cur: 5, ind: 3},
					{val: []int{}, cur: 5, ind: 1},
					{val: []int{}, cur: 5, ind: 3},
					{val: []int{}, cur: 5, ind: 2},
				},
				winner: &node{val: []int{}, cur: 5, ind: 0},
			},
		},
		{
			name:  "same head, stable tie-break by leaf order",
			inSeq: [][]int{{5, 10}, {5, 20}, {5, 30}},
			want: &LoserTree{
				nodes: []*node{
					{val: []int{10}, cur: 5, ind: 0},
					{val: []int{20}, cur: 5, ind: 1},
					{val: []int{30}, cur: 5, ind: 2},
					{val: nil, ind: 0},
					{val: []int{20}, cur: 5, ind: 1},
					{val: nil, ind: 0},
					{val: []int{30}, cur: 5, ind: 2},
				},
				winner: &node{val: []int{10}, cur: 5, ind: 0},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, w := Init(test.inSeq)
			if !reflect.DeepEqual(snapshot(got), snapshot(test.want)) {
				t.Errorf("Init() tree mismatch\n got: %+v\nwant: %+v",
					snapshot(got), snapshot(test.want))
			}

			var wantWinner *node
			if test.want != nil {
				wantWinner = test.want.winner
			}
			if !reflect.DeepEqual(w, wantWinner) {
				t.Errorf("Init() winner mismatch\n got: %+v\nwant: %+v", w, wantWinner)
			}
		})
	}
}

// TestInitArenaLayout проверяет инвариант арены:
// каждая нода дерева и победитель должны указывать внутрь arena.
// Конкретные значения нод и точный размер arena не важны —
// это внутреннее представление, а не контракт.
func TestInitArenaLayout(t *testing.T) {
	for _, k := range []int{1, 2, 3, 4, 5, 7, 8, 16, 100, 256} {
		t.Run(fmt.Sprintf("k=%d", k), func(t *testing.T) {
			in := make([][]int, k)
			for i := range in {
				in[i] = []int{i, i + 1000} // у каждого листа хотя бы одно значение
			}
			lt, _ := Init(in)
			if lt == nil {
				t.Fatal("Init returned nil for non-empty input")
			}

			inArena := func(n *node) bool {
				for j := range lt.arena {
					if n == &lt.arena[j] {
						return true
					}
				}
				return false
			}

			for i, n := range lt.nodes {
				if n == nil {
					t.Errorf("nodes[%d] = nil", i)
					continue
				}
				if !inArena(n) {
					t.Errorf("nodes[%d] = %p not from arena", i, n)
				}
			}

			if lt.winner == nil {
				t.Fatal("winner = nil")
			}
			if !inArena(lt.winner) {
				t.Errorf("winner = %p not from arena", lt.winner)
			}
		})
	}
}
