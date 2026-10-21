package bigsort

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

// parse_bench_test.go
func BenchmarkParseCSVTable(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("lines=%d", n), func(b *testing.B) {
			input := utilGenLines(n)
			maxParsed := n + 1
			b.SetBytes(int64(len(input)))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				br := bufio.NewReader(strings.NewReader(input))
				if _, err := parseCSVTable(br, 0, maxParsed); err != nil && !errors.Is(err, io.EOF) {
					b.Fatal(err)
				}
			}
		})
	}
}

func utilGenLines(n int) string {
	var sb strings.Builder
	letters := []string{"a", "b", "c", "d", "e"}
	for i := 0; i < n; i++ {
		sb.WriteString(strconv.Itoa(rand.Int()))
		sb.WriteByte(',')
		for _, l := range letters {
			sb.WriteString(strings.Repeat(l, i))
			sb.WriteByte(',')
		}
		sb.WriteString("x\n")
	}
	return sb.String()
}
