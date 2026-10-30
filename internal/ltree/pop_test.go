package ltree

import (
	"slices"
	"testing"
)

type popResult struct {
	node *node
	ok   bool
}

type popTest struct {
	name       string
	tree       *LoserTree
	wantPopped []popResult
}

func TestPopOnGeneratedTree(t *testing.T) {
	initT := func(i [][]int) *LoserTree {
		t, _ := Init(i)
		return t
	}

	popT := func(t *LoserTree) popResult {
		v, ok := Pop(t)
		return popResult{node: v, ok: ok}
	}

	tests := []popTest{
		{
			name:       "nil input tree",
			tree:       initT(nil),
			wantPopped: []popResult{{ok: false}},
		},
		{
			name:       "k=1 single element lost",
			tree:       initT([][]int{{42}}),
			wantPopped: []popResult{{ok: false}},
		},
		{
			name: "k=1 two elements",
			tree: initT([][]int{{1, 2}}),
			wantPopped: []popResult{
				{node: &node{val: []int{2}, ind: 0}, ok: true},
				{ok: false},
			},
		},
		{
			name: "k=2 merge",
			tree: initT([][]int{{1, 2}, {3, 4}}),
			wantPopped: []popResult{
				{node: &node{val: []int{2}, ind: 0}, ok: true},
				{node: &node{val: []int{3, 4}, ind: 1}, ok: true},
				{node: &node{val: []int{4}, ind: 1}, ok: true},
				{ok: false},
			},
		},
		{
			name: "k=2 unbalanced",
			tree: initT([][]int{{1}, {2, 3, 4}}),
			wantPopped: []popResult{
				{node: &node{val: []int{2, 3, 4}, ind: 1}, ok: true},
				{node: &node{val: []int{3, 4}, ind: 1}, ok: true},
				{node: &node{val: []int{4}, ind: 1}, ok: true},
				{ok: false},
			},
		},
		{
			name: "k=3 padded",
			tree: initT([][]int{{1, 2}, {3, 4}, {5, 6}}),
			wantPopped: []popResult{
				{node: &node{val: []int{2}, ind: 0}, ok: true},
				{node: &node{val: []int{3, 4}, ind: 1}, ok: true},
				{node: &node{val: []int{4}, ind: 1}, ok: true},
				{node: &node{val: []int{5, 6}, ind: 2}, ok: true},
				{node: &node{val: []int{6}, ind: 2}, ok: true},
				{ok: false},
			},
		},
		{
			name: "k=4 all equal",
			tree: initT([][]int{{5}, {5}, {5}, {5}}),
			wantPopped: []popResult{
				// tie-break: winner at each battle is the left i.
				// After first Pop, leaf 0 advances to nil, leaf 1 becomes champ.
				{node: &node{val: []int{5}, ind: 1}, ok: true},
				{node: &node{val: []int{5}, ind: 2}, ok: true},
				{node: &node{val: []int{5}, ind: 3}, ok: true},
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

				if got.node == nil {
					t.Fatalf("pop[%d] node = nil, want non-nil", i)
				}
				if want.node == nil {
					t.Fatalf("pop[%d] want.node = nil but ok=true", i)
				}

				if !slices.Equal(got.node.val, want.node.val) {
					t.Errorf("pop[%d] val = %v, want %v", i, got.node.val, want.node.val)
				}
				if got.node.ind != want.node.ind {
					t.Errorf("pop[%d] ind = %v, want %v", i, got.node.ind, want.node.ind)
				}
			}

			if got := popT(test.tree); got.ok {
				t.Errorf("returned ok on ended tree")
			}
		})
	}
}
