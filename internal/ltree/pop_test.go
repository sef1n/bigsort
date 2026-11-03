package ltree

import (
	"math"
	"testing"
)

type popStep struct {
	cur int
	ind int
	ok  bool
}

func TestPop(t *testing.T) {
	tests := []struct {
		name    string
		streams [][]int
		want    []popStep
	}{
		{name: "k=1 single element", streams: [][]int{{42}}, want: []popStep{
			{42, 0, true}, {0, 0, false},
		}},
		{name: "k=1 two elements", streams: [][]int{{1, 2}}, want: []popStep{
			{1, 0, true}, {2, 0, true}, {0, 0, false},
		}},
		{name: "k=2 merge", streams: [][]int{{1, 2}, {3, 4}}, want: []popStep{
			{1, 0, true}, {2, 0, true}, {3, 1, true}, {4, 1, true}, {0, 0, false},
		}},
		{name: "k=2 unbalanced", streams: [][]int{{1}, {2, 3, 4}}, want: []popStep{
			{1, 0, true}, {2, 1, true}, {3, 1, true}, {4, 1, true}, {0, 0, false},
		}},
		{name: "k=3 padded", streams: [][]int{{1, 2}, {3, 4}, {5, 6}}, want: []popStep{
			{1, 0, true},
			{2, 0, true},
			{3, 1, true},
			{4, 1, true},
			{5, 2, true},
			{6, 2, true},
			{0, 0, false},
		}},
		{name: "k=5 with three levels", streams: [][]int{
			{1, 10}, {2, 20}, {3, 30}, {4, 40}, {5, 50},
		}, want: []popStep{
			{1, 0, true},
			{2, 1, true},
			{3, 2, true},
			{4, 3, true},
			{5, 4, true},
			{10, 0, true},
			{20, 1, true},
			{30, 2, true},
			{40, 3, true},
			{50, 4, true},
			{0, 0, false},
		}},
		{name: "k=4 all equal", streams: [][]int{{5}, {5}, {5}, {5}}, want: []popStep{
			{5, 0, true}, {5, 1, true}, {5, 2, true}, {5, 3, true}, {0, 0, false},
		}},
		{name: "MaxInt is a valid value", streams: [][]int{{math.MaxInt, math.MaxInt}}, want: []popStep{
			{math.MaxInt, 0, true}, {math.MaxInt, 0, true}, {0, 0, false},
		}},
		{name: "MaxInt among regular values", streams: [][]int{{math.MaxInt}, {1}, {2}}, want: []popStep{
			{1, 1, true}, {2, 2, true}, {math.MaxInt, 0, true}, {0, 0, false},
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initial := make([]int, len(test.streams))
			rest := make([][]int, len(test.streams))
			for i, s := range test.streams {
				initial[i] = s[0]
				rest[i] = s[1:]
			}
			tree := Init(initial)

			for i, want := range test.want {
				ind, ok := tree.Peek()
				if ok != want.ok {
					t.Fatalf("step %d: ok=%v want %v", i, ok, want.ok)
				}
				if !ok {
					continue
				}
				if ind != want.ind {
					t.Errorf("step %d: ind=%d want %d", i, ind, want.ind)
				}
				if got := tree.leafs[ind].v; got != want.cur {
					t.Errorf("step %d: val=%d want %d", i, got, want.cur)
				}

				var newval int
				var alive bool
				if len(rest[ind]) > 0 {
					newval = rest[ind][0]
					rest[ind] = rest[ind][1:]
					alive = true
				}
				tree.Pop(newval, alive)
			}

			// Повторный Pop на исчерпанном дереве — no-op, не паника.
			if _, ok := tree.Pop(0, false); ok {
				t.Error("Pop on exhausted tree should return ok=false")
			}
		})
	}
}
