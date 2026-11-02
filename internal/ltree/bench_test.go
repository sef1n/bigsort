package ltree_test

import (
	"fmt"
	"math"
	"testing"

	"bigFiles/internal/ltree"

	"github.com/bboreham/go-loser"
)

// goLoserLoopCount — сколько значений реально отдаёт стандартный
// цикл `for Next { At }` в go-loser. Используется тестом для
// сверки и в комментарии к бенчмарку как «эталонное» число шагов.
func goLoserLoopCount(k, n int) int {
	seqs := makeSeqs(k, n)
	gt := loser.New[int, *seq](seqs, math.MaxInt)
	c := 0
	for gt.Next() {
		_ = gt.At()
		c++
	}
	return c
}

// TestBenchCountsMatch гарантирует, что оба бенчмарка делают
// одинаковое число шагов слияния. go-loser в стандартном цикле
// `for Next {At}` теряет по одному значению на seq (первое),
// поэтому ожидаемое число = k*(n-1). ltree обрезается до того же.
func TestBenchCountsMatch(t *testing.T) {
	for _, tc := range []struct{ k, n int }{
		{4, 10}, {4, 100}, {16, 100}, {256, 100},
	} {
		t.Run(fmt.Sprintf("k=%d,n=%d", tc.k, tc.n), func(t *testing.T) {
			want := tc.k * (tc.n - 1)

			seqs := makeLeafs(tc.k, tc.n)
			initial := make([]int, tc.k)
			rest := make([][]int, tc.k)
			for i := range seqs {
				initial[i] = seqs[i][0]
				rest[i] = seqs[i][1:]
			}
			tree := ltree.Init(initial)
			lc := 0
			for lc < want {
				_, ind, ok := tree.Peek()
				if !ok {
					break
				}
				var newval int
				var alive bool
				if len(rest[ind]) > 0 {
					newval = rest[ind][0]
					rest[ind] = rest[ind][1:]
					alive = true
				}
				tree.Pop(newval, alive)
				lc++
			}

			gc := goLoserLoopCount(tc.k, tc.n)
			t.Logf("go-loser: %d, ltree: %d, want: %d", gc, lc, want)
			if lc != want {
				t.Errorf("ltree did %d pops, want %d", lc, want)
			}
			if gc != want {
				t.Errorf("go-loser did %d iterations, want %d", gc, want)
			}
		})
	}
}

func BenchmarkLtreePop(b *testing.B) {
	for _, k := range []int{4, 16, 64, 256} {
		for _, n := range []int{100, 1000} {
			b.Run(fmt.Sprintf("k=%d,n=%d", k, n), func(b *testing.B) {
				template := makeLeafs(k, n)
				iters := k * (n - 1)

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					leafs := cloneLeafs(template)
					initial := make([]int, k)
					for j := range leafs {
						initial[j] = leafs[j][0]
						leafs[j] = leafs[j][1:]
					}
					tree := ltree.Init(initial)
					b.StartTimer()

					_, ind, ok := tree.Peek()
					for j := 0; j < iters && ok; j++ {
						var newval int
						var alive bool
						if len(leafs[ind]) > 0 {
							newval = leafs[ind][0]
							leafs[ind] = leafs[ind][1:]
							alive = true
						}
						ind, ok = tree.Pop(newval, alive)
					}
				}
			})
		}
	}
}

func BenchmarkGoLoserPop(b *testing.B) {
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
			l[j] = i*n + j
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

// seq адаптирует последовательность под loser.Sequence[int].
// cur — текущее значение (то, что вернёт At), rest — оставшиеся.
// makeSeqs инициализирует cur=vals[0], rest=vals[1:], поэтому
// New видит настоящее первое значение, а не zero-value.
// go-loser в New вызывает Next() на каждом seq — это «теряет»
// первое значение; стандартный цикл `for Next { At }` отдаёт
// k*(n-1) слияний. Бенчмарк ltree обрезан до того же числа.
type seq struct {
	cur  int
	rest []int
}

func (s *seq) At() int { return s.cur }
func (s *seq) Next() bool {
	if len(s.rest) == 0 {
		return false
	}
	s.cur = s.rest[0]
	s.rest = s.rest[1:]
	return true
}

func makeSeqs(k, n int) []*seq {
	seqs := make([]*seq, k)
	for i := range seqs {
		vals := make([]int, n)
		for j := range vals {
			vals[j] = i*n + j
		}
		seqs[i] = &seq{cur: vals[0], rest: vals[1:]}
	}
	return seqs
}

func cloneSeqs(src []*seq) []*seq {
	dst := make([]*seq, len(src))
	for i, s := range src {
		rest := make([]int, len(s.rest))
		copy(rest, s.rest)
		dst[i] = &seq{cur: s.cur, rest: rest}
	}
	return dst
}
