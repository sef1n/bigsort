package ltree

import (
	"fmt"
	"strings"
)

func (t LoserTree) String() string {
	var sb strings.Builder
	fmt.Fprint(&sb, "{\nLeafs: {")

	for i, l := range t.leafs {
		fmt.Fprintf(&sb, "(%d,{%d,%v})", i, l.v, l.alive)
	}

	fmt.Fprint(&sb, "};\nNodes: {\nL#0: ")

	lvlLen := (len(t.nodes) + 1) / 2
	lvlEnd := lvlLen
	lvlN := 0
	for i, node := range t.nodes {
		if i > 0 {
			fmt.Fprint(&sb, ",")
			if lvlEnd == i {
				lvlN++
				fmt.Fprintf(&sb, "\nL#%d: ", lvlN)
				lvlLen >>= 1
				lvlEnd += lvlLen
			}
		}
		fmt.Fprintf(&sb, "%v", node)
	}

	fmt.Fprintf(&sb, "\n}\nWinner: %v}", t.winner)
	return sb.String()
}
