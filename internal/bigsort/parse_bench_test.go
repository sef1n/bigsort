package bigsort

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"testing"
)

func BenchmarkParseCSVTable(b *testing.B) {
	rSeed := rand.New(rand.NewSource(1))
	inLineDigits, inLineLetters := 10, 10
	genLetters := []byte("abcd")

	for _, linesN := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("lines=%d", linesN), func(b *testing.B) {
			input := make([]byte, 0, linesN*(inLineDigits+inLineLetters+2))

			lineSize := 0
			for range linesN {
				line := utilGenLine(rSeed, inLineDigits, inLineLetters, genLetters)
				input = append(input, line...)
				lineSize += len(line)
			}
			br := bufio.NewReader(bytes.NewReader(input))

			maxParsed := lineSize + 1
			b.SetBytes(int64(lineSize))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				br.Reset(bytes.NewReader(input))
				if _, err := parseCSVTable(br, 0, maxParsed); err != nil && !errors.Is(err, io.EOF) {
					b.Fatal(err)
				}
			}
		})
	}
}
