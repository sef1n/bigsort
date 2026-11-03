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
				nodes:  []leafInd{0, 0},
				winner: Winner{I: 0, valid: true},
			},
		},
		{
			name: "k=2 sorted",
			in:   []int{1, 2},
			want: &LoserTree{
				leafs:  []leaf{rn(1), rn(2)},
				nodes:  []leafInd{0, 1, 0, 1},
				winner: Winner{I: 0, valid: true},
			},
		},
		{
			name: "k=2 reversed",
			in:   []int{2, 1},
			want: &LoserTree{
				leafs:  []leaf{rn(2), rn(1)},
				nodes:  []leafInd{0, 0, 0, 1},
				winner: Winner{I: 1, valid: true},
			},
		},
		{
			name: "k=3 padded to 4",
			in:   []int{3, 2, 1},
			want: &LoserTree{
				leafs:  []leaf{rn(3), rn(2), rn(1), inf()},
				nodes:  []leafInd{0, 1, 0, 3, 0, 1, 2, 3},
				winner: Winner{I: 2, valid: true},
			},
		},
		{
			name: "k=3 equal vals",
			in:   []int{1, 1, 1},
			want: &LoserTree{
				leafs:  []leaf{rn(1), rn(1), rn(1), inf()},
				nodes:  []leafInd{0, 2, 1, 3, 0, 1, 2, 3},
				winner: Winner{I: 0, valid: true},
			},
		},
		{
			name: "k=4 descending",
			in:   []int{4, 3, 2, 1},
			want: &LoserTree{
				leafs:  []leaf{rn(4), rn(3), rn(2), rn(1)},
				nodes:  []leafInd{0, 1, 0, 2, 0, 1, 2, 3},
				winner: Winner{I: 3, valid: true},
			},
		},
		{
			name: "k=4 all equal, stable tie-break",
			in:   []int{5, 5, 5, 5},
			want: &LoserTree{
				leafs:  []leaf{rn(5), rn(5), rn(5), rn(5)},
				nodes:  []leafInd{0, 2, 1, 3, 0, 1, 2, 3},
				winner: Winner{I: 0, valid: true},
			},
		},
		{
			name: "k=5 padded to 8",
			in:   []int{1, 2, 3, 4, 5},
			want: &LoserTree{
				leafs: []leaf{
					rn(1), rn(2), rn(3), rn(4), rn(5), inf(), inf(), inf(),
				},
				nodes: []leafInd{
					0,
					4, 2, 7,
					1, 3, 5, 6,
					0, 1, 2, 3, 4, 5, 6, 7,
				},
				winner: Winner{I: 0, valid: true},
			},
		},
		{
			// Регрессия на баг: winner.I = 3, nodes[3] = 7 (padding, dead).
			// Старая формула lt.winner.valid = leafs[nodes[winner.I]].alive
			// давала здесь false, хотя победитель — живой лист 3.
			name: "k=5 winner at leaf 3, nodes[3] is padding loser",
			in:   []int{5, 3, 2, 1, 4},
			want: &LoserTree{
				leafs: []leaf{
					rn(5), rn(3), rn(2), rn(1), rn(4), inf(), inf(), inf(),
				},
				nodes: []leafInd{
					0,
					4, 1, 7,
					0, 2, 5, 6,
					0, 1, 2, 3, 4, 5, 6, 7,
				},
				winner: Winner{I: 3, valid: true},
			},
		},
		{
			name: "MaxInt is a valid value",
			in:   []int{math.MaxInt},
			want: &LoserTree{
				leafs:  []leaf{rn(math.MaxInt)},
				nodes:  []leafInd{0, 0},
				winner: Winner{I: 0, valid: true},
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
