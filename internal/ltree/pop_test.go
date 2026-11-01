package ltree

import (
	"math"
	"testing"
)

type popResult struct {
	cur int
	ind int
	ok  bool
}

type popTest struct {
	name       string
	tree       *LoserTree
	wantPopped []popResult
}

func TestPop(t *testing.T) {
	initT := func(i [][]int) *LoserTree { return Init(i) }
	popT := func(tr *LoserTree) popResult {
		cur, ind, ok := Pop(tr)
		return popResult{cur: cur, ind: ind, ok: ok}
	}

	tests := []popTest{
		{
			name:       "nil tree",
			tree:       nil,
			wantPopped: []popResult{{ok: false}},
		},
		{
			name:       "nil input",
			tree:       initT(nil),
			wantPopped: []popResult{{ok: false}},
		},
		{
			// Init потребляет первый элемент листа, но Pop теперь
			// возвращает и его тоже. Один элемент → один Pop.
			name: "k=1 single element",
			tree: initT([][]int{{42}}),
			wantPopped: []popResult{
				{cur: 42, ind: 0, ok: true},
				{ok: false},
			},
		},
		{
			name: "k=1 two elements",
			tree: initT([][]int{{1, 2}}),
			wantPopped: []popResult{
				{cur: 1, ind: 0, ok: true},
				{cur: 2, ind: 0, ok: true},
				{ok: false},
			},
		},
		{
			name: "k=2 merge",
			tree: initT([][]int{{1, 2}, {3, 4}}),
			wantPopped: []popResult{
				{cur: 1, ind: 0, ok: true},
				{cur: 2, ind: 0, ok: true},
				{cur: 3, ind: 1, ok: true},
				{cur: 4, ind: 1, ok: true},
				{ok: false},
			},
		},
		{
			name: "k=2 unbalanced",
			tree: initT([][]int{{1}, {2, 3, 4}}),
			wantPopped: []popResult{
				{cur: 1, ind: 0, ok: true},
				{cur: 2, ind: 1, ok: true},
				{cur: 3, ind: 1, ok: true},
				{cur: 4, ind: 1, ok: true},
				{ok: false},
			},
		},
		{
			name: "k=3 padded",
			tree: initT([][]int{{1, 2}, {3, 4}, {5, 6}}),
			wantPopped: []popResult{
				{cur: 1, ind: 0, ok: true},
				{cur: 2, ind: 0, ok: true},
				{cur: 3, ind: 1, ok: true},
				{cur: 4, ind: 1, ok: true},
				{cur: 5, ind: 2, ok: true},
				{cur: 6, ind: 2, ok: true},
				{ok: false},
			},
		},
		{
			// tie-break: на равенстве выигрывает левый (i).
			name: "k=4 all equal",
			tree: initT([][]int{{5}, {5}, {5}, {5}}),
			wantPopped: []popResult{
				{cur: 5, ind: 0, ok: true},
				{cur: 5, ind: 1, ok: true},
				{cur: 5, ind: 2, ok: true},
				{cur: 5, ind: 3, ok: true},
				{ok: false},
			},
		},
		{
			// MaxInt не должен теряться: он не sentinel, sentinel — ind == -1.
			name: "MaxInt is a valid value",
			tree: initT([][]int{{math.MaxInt, math.MaxInt}}),
			wantPopped: []popResult{
				{cur: math.MaxInt, ind: 0, ok: true},
				{cur: math.MaxInt, ind: 0, ok: true},
				{ok: false},
			},
		},
		{
			// MaxInt соседствует с обычными значениями и сортируется правильно.
			name: "MaxInt among regular values",
			tree: initT([][]int{{math.MaxInt}, {1}, {2}}),
			wantPopped: []popResult{
				{cur: 1, ind: 1, ok: true},
				{cur: 2, ind: 2, ok: true},
				{cur: math.MaxInt, ind: 0, ok: true},
				{ok: false},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for i, want := range test.wantPopped {
				got := popT(test.tree)
				if got.ok != want.ok {
					t.Fatalf("pop[%d] ok = %v, want %v", i, got.ok, want.ok)
				}
				if !got.ok {
					continue
				}
				if got.cur != want.cur {
					t.Errorf("pop[%d] cur = %v, want %v", i, got.cur, want.cur)
				}
				if got.ind != want.ind {
					t.Errorf("pop[%d] ind = %v, want %v", i, got.ind, want.ind)
				}
			}

			if got := popT(test.tree); got.ok {
				t.Errorf("returned ok on ended tree")
			}
		})
	}
}
