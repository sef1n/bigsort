package bigsort

import (
	"bytes"
	"math/rand"
)

func utilGenLine(r *rand.Rand, iN, lN int, letters []byte) []byte {
	var b bytes.Buffer
	// или b := make([]byte, 0, iN+1+lN+1)
	b.WriteString(nDigitString(r, iN))
	b.WriteByte(',')
	for i := range lN {
		l := letters[i%len(letters)]
		b.WriteByte(l)
	}
	b.WriteByte('\n')
	return b.Bytes()
}

// nDigitString возвращает случайное n-значное число (n >= 1) как строку.
func nDigitString(r *rand.Rand, n int) string {
	if n < 1 {
		panic("n >= 1")
	}
	b := make([]byte, n)
	b[0] = byte('1' + r.Intn(9)) // '1'..'9'
	for i := 1; i < n; i++ {
		b[i] = byte('0' + r.Intn(10)) // '0'..'9'
	}
	return string(b)
}
