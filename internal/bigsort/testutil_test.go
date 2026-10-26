package bigsort

import (
	"bufio"
	"bytes"
	"io"
	"math/rand"

	"github.com/spf13/afero"
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

func fillFile(f afero.File, input []byte) error {
	w := bufio.NewWriter(f)
	if _, err := w.Write(input); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	_, err := f.Seek(0, io.SeekStart)
	return err
}

func cmpWithBoolErr(err error, isNeeded bool) bool {
	var bErr bool
	if err != nil {
		bErr = true
	}
	return bErr == isNeeded
}

func delF(fs afero.Fs, f afero.File) {
	name := f.Name()
	_ = f.Close()
	_ = fs.Remove(name)
}
