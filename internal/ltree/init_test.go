package ltree

import (
	"math"
	"reflect"
	"testing"
)

func TestInit(t *testing.T) {
	inf := func() leaf { return leaf{v: math.MaxInt, alive: false} }
	rn := func(v int) leaf { return leaf{v: v, alive: true} }

	tests := []struct {
		name string
		in   []int
		want *LoserTree
	}{
		{name: "nil input", in: nil, want: nil},
		{name: "empty slice", in: []int{}, want: nil},

		{
			name: "k=1 single leaf",
			in:   []int{42},
			want: &LoserTree{
				leafs:  []leaf{rn(42)},
				nodes:  []leafInd{0},
				winner: 0,
			},
		},
		{
			name: "k=2 sorted",
			in:   []int{1, 2},
			want: &LoserTree{
				leafs:  []leaf{rn(1), rn(2)},
				nodes:  []leafInd{0, 1, 1},
				winner: 0,
			},
		},
		{
			name: "k=2 reversed",
			in:   []int{2, 1},
			want: &LoserTree{
				leafs:  []leaf{rn(2), rn(1)},
				nodes:  []leafInd{0, 1, 0},
				winner: 1,
			},
		},
		{
			name: "k=3 padded to 4",
			in:   []int{3, 2, 1},
			want: &LoserTree{
				leafs:  []leaf{rn(3), rn(2), rn(1), inf()},
				nodes:  []leafInd{0, 1, 2, 3, 0, 3, 1},
				winner: 2,
			},
		},
		{
			name: "k=3 equal vals",
			in:   []int{1, 1, 1},
			want: &LoserTree{
				leafs:  []leaf{rn(1), rn(1), rn(1), inf()},
				nodes:  []leafInd{0, 1, 2, 3, 1, 3, 2},
				winner: 0,
			},
		},
		{
			name: "k=4 descending",
			in:   []int{4, 3, 2, 1},
			want: &LoserTree{
				leafs:  []leaf{rn(4), rn(3), rn(2), rn(1)},
				nodes:  []leafInd{0, 1, 2, 3, 0, 2, 1},
				winner: 3,
			},
		},
		{
			name: "k=4 all equal, stable tie-break",
			in:   []int{5, 5, 5, 5},
			want: &LoserTree{
				leafs:  []leaf{rn(5), rn(5), rn(5), rn(5)},
				nodes:  []leafInd{0, 1, 2, 3, 1, 3, 2},
				winner: 0,
			},
		},
		{
			name: "MaxInt is a valid value",
			in:   []int{math.MaxInt},
			want: &LoserTree{
				leafs:  []leaf{rn(math.MaxInt)},
				nodes:  []leafInd{0},
				winner: 0,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Init(test.in)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Init() mismatch\n got: %+v\nwant: %+v", got, test.want)
			}
		})
	}
}
