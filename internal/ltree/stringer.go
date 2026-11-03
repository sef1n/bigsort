package ltree

import (
	"fmt"
	"strings"
)

// String форматирует дерево. Уровни печатаются от корня (L#0) вниз.
// nodes[0] не используется.
func (t LoserTree) String() string {
	var sb strings.Builder
	fmt.Fprint(&sb, "{\nLeafs: {")
	for i, l := range t.leafs {
		fmt.Fprintf(&sb, "(i:%d,{v:%d,valid:%v})", i, l.v, l.alive)
	}
	fmt.Fprint(&sb, "};\nNodes: {\n")

	level := 0
	levelStart := 1 // nodes[1] — начало уровня 0
	for i := 1; i < len(t.nodes); i++ {
		if i == levelStart {
			if i > 1 {
				fmt.Fprint(&sb, "\n")
			}
			fmt.Fprintf(&sb, "L#%d: ", level)
			level++
			levelStart <<= 1
		} else {
			fmt.Fprint(&sb, ",")
		}
		fmt.Fprintf(&sb, "%v", t.nodes[i])
	}
	if len(t.nodes) > 1 {
		fmt.Fprint(&sb, "\n")
	}

	fmt.Fprintf(&sb, "}\nWinner: %v}", t.winner)
	return sb.String()
}
