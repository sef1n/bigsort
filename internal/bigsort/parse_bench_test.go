package bigsort

import (
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

func BenchmarkParseCSVTable(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("lines=%d", n), func(b *testing.B) {
			input := utilGenLines(n)      // готовим ОДИН раз
			b.SetBytes(int64(len(input))) // для MB/s
			b.ReportAllocs()
			b.ResetTimer() // не считаем генерацию
			for i := 0; i < b.N; i++ {
				r := strings.NewReader(input) // свежий reader на итерацию
				if _, err := parseCSVTable(r, 0); err != nil && err != io.EOF {
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
		for _, l := range letters {
			sb.WriteByte(',')
			sb.WriteString(strings.Repeat(l, i))
		}
		sb.WriteString(",x\n")
	}
	return sb.String()
}
