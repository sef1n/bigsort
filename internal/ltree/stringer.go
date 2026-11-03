package ltree

import (
	"fmt"
	"strings"
)

// String форматирует дерево для отладки.
// Уровни печатаются от корня (L#0) вниз; nodes[0] не используется.
func (t *LoserTree) String() string {
	if t == nil {
		return "{<nil LoserTree>}"
	}

	var sb strings.Builder
	sb.Grow(64 + 32*len(t.nodes) + 32*len(t.leafs))

	fmt.Fprint(&sb, "{\nLeafs: {")
	for i, l := range t.leafs {
		if i > 0 {
			fmt.Fprint(&sb, " ")
		}
		fmt.Fprintf(&sb, "(i:%d,v:%d,alive:%v)", i, l.v, l.alive)
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
		fmt.Fprintf(&sb, "%d", t.nodes[i])
	}
	if len(t.nodes) > 1 {
		fmt.Fprint(&sb, "\n")
	}

	fmt.Fprintf(&sb, "}\nWinner: {I:%d, valid:%v}}", t.winner.I, t.winner.valid)
	return sb.String()
}
