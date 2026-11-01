package ltree

import (
	"math"
	"reflect"
	"testing"
)

// snapshot возвращает копию дерева, у которой обнулены поля,
// не относящиеся к логическому состоянию.
func snapshot(t *LoserTree) *LoserTree {
	if t == nil {
		return nil
	}
	cp := *t
	cp.arena = nil
	cp.leafs = nil // мутируется Init, в Init-тесте не сравниваем
	return &cp
}

// inf — sentinel-узел: значение = +∞, ind = -1.
func inf() *node { return &node{cur: math.MaxInt, ind: -1} }

// rn — обычный (значащий) узел.
func rn(cur, ind int) *node { return &node{cur: cur, ind: ind} }

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

		// Все листья незначащие → всё дерево из sentinel-ов.
		{
			name:  "k=1 nil leaf",
			inSeq: [][]int{nil},
			want: &LoserTree{
				nodes:  []*node{inf()},
				winner: inf(),
			},
		},
		{
			name:  "k=3 all nil",
			inSeq: [][]int{nil, nil, nil},
			want: &LoserTree{
				nodes:  []*node{inf(), inf(), inf(), inf(), inf(), inf(), inf()},
				winner: inf(),
			},
		},
		{
			name:  "k=1 empty leaf",
			inSeq: [][]int{{}},
			want: &LoserTree{
				nodes:  []*node{inf()},
				winner: inf(),
			},
		},
		{
			name:  "k=3 all empty leafs",
			inSeq: [][]int{{}, {}, {}},
			want: &LoserTree{
				nodes:  []*node{inf(), inf(), inf(), inf(), inf(), inf(), inf()},
				winner: inf(),
			},
		},

		// Валидные данные.
		{
			name:  "k=1 single leaf",
			inSeq: [][]int{{42}},
			want: &LoserTree{
				nodes:  []*node{rn(42, 0)},
				winner: rn(42, 0),
			},
		},
		{
			name:  "k=2 sorted",
			inSeq: [][]int{{1}, {2}},
			want: &LoserTree{
				nodes:  []*node{rn(1, 0), rn(2, 1), rn(2, 1)},
				winner: rn(1, 0),
			},
		},
		{
			name:  "k=2 reversed",
			inSeq: [][]int{{2}, {1}},
			want: &LoserTree{
				nodes:  []*node{rn(2, 0), rn(1, 1), rn(2, 0)},
				winner: rn(1, 1),
			},
		},
		{
			// 3 листа + 1 pad.
			// R1: battle(l0,l1)=(l1,l0), battle(l2,inf)=(l2,inf)
			// R2: battle(l1,l2)=(l2,l1)
			name:  "k=3 padded to 4",
			inSeq: [][]int{{3}, {2}, {1}},
			want: &LoserTree{
				nodes: []*node{
					rn(3, 0), rn(2, 1), rn(1, 2), inf(), // level 0
					rn(3, 0), inf(), // level 1 losers
					rn(2, 1), // level 2 loser
				},
				winner: rn(1, 2),
			},
		},
		{
			// tie-break: на равенстве выигрывает левый (i).
			name:  "k=3 equal vals",
			inSeq: [][]int{{1}, {1}, {1}},
			want: &LoserTree{
				nodes: []*node{
					rn(1, 0), rn(1, 1), rn(1, 2), inf(),
					rn(1, 1), inf(),
					rn(1, 2),
				},
				winner: rn(1, 0),
			},
		},
		{
			// R1 losers: l0, l2; R2 loser: l1; winner: l3.
			name:  "k=4",
			inSeq: [][]int{{4}, {3}, {2}, {1}},
			want: &LoserTree{
				nodes: []*node{
					rn(4, 0), rn(3, 1), rn(2, 2), rn(1, 3),
					rn(4, 0), rn(2, 2),
					rn(3, 1),
				},
				winner: rn(1, 3),
			},
		},
		{
			// 7 листьев + 1 pad.
			name: "k=7 padded to 8",
			inSeq: [][]int{
				{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}, {6, 6}, {7},
			},
			want: &LoserTree{
				nodes: []*node{
					rn(1, 0), rn(2, 1), rn(3, 2), rn(4, 3),
					rn(5, 4), rn(6, 5), rn(7, 6), inf(),
					rn(2, 1), rn(4, 3), rn(6, 5), inf(),
					rn(3, 2), rn(7, 6),
					rn(5, 4),
				},
				winner: rn(1, 0),
			},
		},
		{
			name:  "k=4 all equal, stable tie-break",
			inSeq: [][]int{{5}, {5}, {5}, {5}},
			want: &LoserTree{
				nodes: []*node{
					rn(5, 0), rn(5, 1), rn(5, 2), rn(5, 3),
					rn(5, 1), rn(5, 3),
					rn(5, 2),
				},
				winner: rn(5, 0),
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
