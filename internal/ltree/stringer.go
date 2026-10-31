package ltree

import (
	"fmt"
	"strings"
)

// Debug вывод

func (n node) String() string {
	if n.val == nil {
		return fmt.Sprintf("(<nil>,%v)", n.ind)
	}
	return fmt.Sprintf("(%v,%v,%v)", n.val, n.cur, n.ind)
}

func (t LoserTree) String() string {
	var sb strings.Builder
	fmt.Fprint(&sb, "{\nNodes: ")
	fmt.Fprint(&sb, "{")
	for _, node := range t.nodes {
		if node == nil {
			fmt.Fprint(&sb, "<nil node>")
		} else {
			fmt.Fprintf(&sb, "%v", node)
		}
	}
	fmt.Fprint(&sb, "},\nArena: ")

	fmt.Fprint(&sb, "{")
	for _, node := range t.arena {
		fmt.Fprintf(&sb, "%v", node)
	}
	fmt.Fprint(&sb, "},\n")
	fmt.Fprintf(&sb, "Winner: %v\n}", t.winner)
	return sb.String()
}
