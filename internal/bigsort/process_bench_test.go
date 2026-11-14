package bigsort

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"bigFiles/internal/bigsort/testutils"

	"github.com/spf13/afero"
)

func initSizes(inputSize, scale int) []int {
	chunksSizes := make([]int, 0)
	for chunk, i := 0, 0; chunk < inputSize/2; i += 2 {
		chunk = scale << i
		chunksSizes = append(chunksSizes, chunk)
	}
	return chunksSizes
}

func BenchmarkSortFileMemMap(b *testing.B) {
	// b.N от 1 до 2, рекомендуется запускать командой:
	// go test -bench='BenchmarkSortFileMemMap' -benchmem -benchtime=10x -count=5 ./internal/bigsort

	const (
		inputName  = "input"
		outputName = "output"

		inLineDigits  = 10
		inLineLetters = 10
		inLineSymbols = inLineDigits + inLineLetters + testutils.InLinePunctuation

		miB        = 1 << 20
		inputSize  = miB << 7
		totalLines = inputSize / inLineSymbols
	)

	rSeed := rand.New(rand.NewSource(1))
	chunksSizes := initSizes(inputSize, miB)
	genLetters := []byte("abcd")

	mockFs := afero.NewMemMapFs()
	inputF, _, err := createInput(
		rSeed, inputName, totalLines, inLineDigits, inLineLetters, genLetters, mockFs)
	if err != nil {
		b.Fatal(err)
	}
	if err := inputF.Close(); err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		mockFs.Remove(inputName)
		mockFs.Remove(outputName)
	})

	for _, bpc := range chunksSizes {
		b.Run(fmt.Sprintf("chunkSize=%dMib", bpc/miB), func(b *testing.B) {
			cfg := SortConfig{
				InPath:        inputName,
				OutPath:       outputName,
				BytesPerChunk: bpc,
			}

			b.SetBytes(int64(inputSize))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := SortFile(context.Background(), cfg, mockFs); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// NOTE: если будет часто вызываться -> сделать структуру для параметров
func createInput(rSeed *rand.Rand, inputName string, totalLines, inLineDigits, inLineLetters int, genLetters []byte, fsys afero.Fs) (afero.File, int, error) {
	inputF, err := fsys.Create(inputName)
	if err != nil {
		return nil, 0, err
	}
	size := 0

	for range totalLines {
		line := testutils.UtilGenLine(rSeed, inLineDigits, inLineLetters, genLetters)
		_, err := inputF.Write(line)
		if err != nil {
			return nil, 0, err
		}
		size += len(line)
	}
	return inputF, size, nil
}
