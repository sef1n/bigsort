package bigsort

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"testing"

	"github.com/spf13/afero"
)

func BenchmarkSplitOnSortedChunks(B *testing.B) {
	rSeed := rand.New(rand.NewSource(1))
	inLineDigits, inLineLetters := 10, 10
	genLetters := []byte("abcd")

	mockFs := afero.NewMemMapFs()
	delF := func(f afero.File) {
		name := f.Name()
		_ = f.Close()
		_ = mockFs.Remove(name)
	}

	for _, linesN := range []int{100, 1000, 10000} {
		// NOTE: удаляется после завершения B.Run
		inputF, err := mockFs.Create(strconv.Itoa(linesN))
		if err != nil {
			B.Fatalf("fs: %v", err)
		}
		fSize := 0

		for range linesN {
			line := utilGenLine(rSeed, inLineDigits, inLineLetters, genLetters)
			_, err = inputF.Write(line)
			if err != nil {
				B.Fatalf("fs: %v", err)
			}
			fSize += len(line)
		}

		B.Run(fmt.Sprintf("lines=%d", linesN), func(B *testing.B) {
			B.SetBytes(int64(fSize))
			B.ReportAllocs()
			B.ResetTimer()

			for i := 0; i < B.N; i++ {
				tmps, err := splitOnSortedChunks(context.Background(), 0, inputF.Name(), ".", mockFs)
				if err != nil && !errors.Is(err, io.EOF) {
					B.Fatal(err)
				}

				B.StopTimer()
				for _, f := range tmps {
					delF(f)
				}
				B.StartTimer()
			}
		})
		delF(inputF)
	}
}
