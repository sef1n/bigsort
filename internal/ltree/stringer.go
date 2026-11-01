package ltree

import (
	"fmt"
	"strings"
)

// Debug вывод

func (n node) String() string {
	return fmt.Sprintf("(cur:%v,ind%v)", n.cur, n.ind)
}

func (t LoserTree) String() string {
	var sb strings.Builder

	fmt.Fprint(&sb, "{Arena:{")
	for i, node := range t.arena {
		if i > 0 {
			fmt.Fprint(&sb, ",")
		}
		fmt.Fprintf(&sb, "%v", node)
	}
	fmt.Fprint(&sb, "},\n{Nodes:{\nL#0 ")

	lvlLen := (len(t.nodes) + 1) / 2
	lvlEnd := lvlLen
	lvlN := 0
	for i, node := range t.nodes {
		if i > 0 {
			fmt.Fprint(&sb, ",")
			if lvlEnd == i {
				lvlN++
				fmt.Fprintf(&sb, "\nL#%d ", lvlN)
				lvlLen >>= 1
				lvlEnd += lvlLen
			}
		}
		if node == nil {
			fmt.Fprint(&sb, "(<nil node>)")
		} else {
			fmt.Fprintf(&sb, "%v", node)
		}
	}

	fmt.Fprintf(&sb, "\n}\nWinner: %v}", t.winner)
	return sb.String()
}
