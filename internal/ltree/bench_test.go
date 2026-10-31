package ltree_test

import (
	"fmt"
	"math"
	"testing"

	"bigFiles/internal/ltree"

	"github.com/bboreham/go-loser"
)

func BenchmarkLtreePop(b *testing.B) {
	for _, k := range []int{4, 16, 64, 256} {
		for _, n := range []int{100, 1000} {
			b.Run(fmt.Sprintf("k=%d,n=%d", k, n), func(b *testing.B) {
				template := makeLeafs(k, n)

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					leafs := cloneLeafs(template)
					tree, _ := ltree.Init(leafs)
					b.StartTimer()

					for {
						if _, ok := ltree.Pop(tree); !ok {
							break
						}
					}
				}
			})
		}
	}
}

func BenchmarkGoLoserPop(b *testing.B) {
	// for _, k := range []int{4, 16, 64, 256} {
	for _, k := range []int{4, 16, 64, 256} {
		for _, n := range []int{100, 1000} {
			b.Run(fmt.Sprintf("k=%d,n=%d", k, n), func(b *testing.B) {
				template := makeSeqs(k, n)

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					seqs := cloneSeqs(template)
					tree := loser.New[int, *seq](seqs, math.MaxInt)
					b.StartTimer()

					for tree.Next() {
						_ = tree.At()
					}
				}
			})
		}
	}
}

func makeLeafs(k, n int) [][]int {
	leafs := make([][]int, k)
	for i := range leafs {
		l := make([]int, n)
		for j := range l {
			l[j] = i*n + j // Каждый лист отсортирован
		}
		leafs[i] = l
	}
	return leafs
}

func cloneLeafs(src [][]int) [][]int {
	dst := make([][]int, len(src))
	for i, l := range src {
		c := make([]int, len(l))
		copy(c, l)
		dst[i] = c
	}
	return dst
}

// seq адаптирует []int под loser.Sequence[int].
// New вызывает Next() один раз для инициализации,
// поэтому первый элемент «теряется» — ровно как и в ltree.Pop.
type seq []int

func (s *seq) At() int { return (*s)[0] }
func (s *seq) Next() bool {
	if len(*s) <= 1 {
		*s = nil
		return false
	}
	*s = (*s)[1:]
	return true
}

func makeSeqs(k, n int) []*seq {
	seqs := make([]*seq, k)
	for i := range seqs {
		s := make(seq, n)
		for j := range s {
			s[j] = i*n + j
		}
		seqs[i] = &s
	}
	return seqs
}

func cloneSeqs(src []*seq) []*seq {
	dst := make([]*seq, len(src))
	for i, s := range src {
		c := make(seq, len(*s))
		copy(c, *s)
		dst[i] = &c
	}
	return dst
}
