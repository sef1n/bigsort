package ltree

import (
	"math"
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
	inf := &node{val: nil, cur: math.MaxInt, ind: -1}
	tests := []initTest{
		// Пустые входные данные.
		{name: "nil input", inSeq: nil, want: nil},
		{name: "empty slice", inSeq: [][]int{}, want: nil},

		// Все листья незначащие — единственный участник inf.
		{
			name: "k=1 nil leaf", inSeq: [][]int{nil},
			want: &LoserTree{
				nodes:  []*node{inf},
				winner: inf,
			},
		},
		{
			name: "k=3 all nil", inSeq: [][]int{nil, nil, nil},
			want: &LoserTree{
				nodes:  []*node{inf, inf, inf, inf, inf, inf, inf},
				winner: inf,
			},
		},
		{
			name: "k=1 len of leaf == 0", inSeq: [][]int{{}},
			want: &LoserTree{
				nodes:  []*node{inf},
				winner: inf,
			},
		},
		{
			name: "k=3 len of all leafs == 0", inSeq: [][]int{{}, {}, {}},
			want: &LoserTree{
				nodes:  []*node{inf, inf, inf, inf, inf, inf, inf},
				winner: inf,
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
			// 3 листа + 1 pad.
			// R1: battle(l0,l1)=(l1,l0), battle(l2,inf)=(l2,inf)
			// R2: battle(l1,l2)=(l2,l1)
			name: "k=3 padded to 4", inSeq: [][]int{{3}, {2}, {1}},
			want: &LoserTree{
				nodes: []*node{
					{val: []int{}, cur: 3, ind: 0}, // l0
					{val: []int{}, cur: 2, ind: 1}, // l1
					{val: []int{}, cur: 1, ind: 2}, // l2
					inf,                            // pad
					{val: []int{}, cur: 3, ind: 0}, // loser l0
					inf,                            // loser pad
					{val: []int{}, cur: 2, ind: 1}, // loser l1
				},
				winner: &node{val: []int{}, cur: 1, ind: 2},
			},
		},
		{
			// tie-break: на равенстве выигрывает левый (i).
			name: "k=3 equal vals", inSeq: [][]int{{1}, {1}, {1}},
			want: &LoserTree{
				nodes: []*node{
					{val: []int{}, cur: 1, ind: 0},
					{val: []int{}, cur: 1, ind: 1},
					{val: []int{}, cur: 1, ind: 2},
					inf,
					{val: []int{}, cur: 1, ind: 1}, // loser l1
					inf,
					{val: []int{}, cur: 1, ind: 2}, // loser l2
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
			// 7 листьев + 1 pad.
			// R1 losers: l1, l3, l5, inf
			// R2 losers: l2, l6
			// R3 loser:  l4
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
					inf,
					{val: []int{2}, cur: 2, ind: 1},
					{val: []int{4}, cur: 4, ind: 3},
					{val: []int{6}, cur: 6, ind: 5},
					inf,
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
			// 3 листа с равными cur, разными хвостами + 1 pad.
			// Less смотрит только на cur; tie-break по меньшему ind.
			name:  "same head, stable tie-break by leaf order",
			inSeq: [][]int{{5, 10}, {5, 20}, {5, 30}},
			want: &LoserTree{
				nodes: []*node{
					{val: []int{10}, cur: 5, ind: 0},
					{val: []int{20}, cur: 5, ind: 1},
					{val: []int{30}, cur: 5, ind: 2},
					inf,
					{val: []int{20}, cur: 5, ind: 1},
					inf,
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

//
// type testnode struct {
// 	cur int
// 	val []int
// 	ind int
// }
// type cmpRes struct {
// 	w, l *testnode
// }
// type battleTest struct {
// 	i, j *testnode
// 	res  cmpRes
// }
//
// func TestBattle(t *testing.T) {
// 	maxDead := &testnode{cur: math.MaxInt, ind: -1}
// 	maxAlive := &testnode{cur: math.MaxInt}
// 	normAlive := &testnode{cur: 1}
// 	tests := []battleTest{
// 		{i: maxDead, j: maxAlive, res: cmpRes{maxAlive, maxDead}},
// 		{i: maxAlive, j: maxDead, res: cmpRes{maxAlive, maxDead}},
// 		{i: maxDead, j: maxDead, res: cmpRes{maxDead, maxDead}},
//
// 		{i: maxDead, j: normAlive, res: cmpRes{normAlive, maxDead}},
// 		{i: normAlive, j: maxDead, res: cmpRes{normAlive, maxDead}},
// 		{i: normAlive, j: maxAlive, res: cmpRes{normAlive, maxAlive}},
// 		{i: maxAlive, j: normAlive, res: cmpRes{normAlive, maxAlive}},
// 	}
//
// 	for _, test := range tests {
// 		w, l := battle2(test.i, test.j)
// 		if !resEqual(w, test.res.w) {
// 			t.Errorf("got w %v != want w %v", w, test.res.w)
// 		}
//
// 		if !resEqual(l, test.res.l) {
// 			t.Errorf("got l %v != want l %v", l, test.res.l)
// 		}
// 	}
// }
//
// func battle2(i, j *testnode) (winner, loser *testnode) {
// 	if j.cur < i.cur {
// 		return j, i
// 	}
//
// 	if i.ind < 0 {
// 		return j, i
// 	}
//
// 	return i, j
// }
//
// func resEqual(i, j *testnode) bool {
// 	if (i == nil && j != nil) || (i != nil && j == nil) {
// 		return false
// 	}
//
// 	if (i.cur == j.cur) && slices.Equal(i.val, j.val) && (i.ind == j.ind) {
// 		return true
// 	}
//
// 	return false
// }
