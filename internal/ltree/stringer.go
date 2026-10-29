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
	return fmt.Sprintf("(%v,%v)", n.val, n.ind)
}

func (t LoserTree) String() string {
	var sb strings.Builder
	fmt.Fprint(&sb, "{\n")
	for _, nodes := range t.levels {
		sb.WriteByte('[')

		if nodes == nil {
			fmt.Fprint(&sb, "<nil level>")
		} else {
			for _, node := range nodes {
				if node == nil {
					fmt.Fprint(&sb, "<nil node>")
				} else {
					fmt.Fprintf(&sb, "%v", node)
				}
			}
		}

		sb.WriteByte(']')
		sb.WriteByte('\n')
	}
	fmt.Fprintf(&sb, "; Winner node: %v }", t.winner)
	return sb.String()
}
