package bigsort

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"

	"bigFiles/internal/bigsort/testutils"

	"github.com/spf13/afero"
)

// NOTE: в среднем b.N = от 1 до 2, рекомендуется запускать командой:
// go test -bench='BenchmarkSortFileMemMap' -benchmem -benchtime=10x -count=10 ./internal/bigsort
//
// NOTE: cleanup при Ctrl+C не гарантирован (testing.Cleanup выполняется только при нормальном заврешении)
func initSizes(inputSize, scale int) []int {
	chunksSizes := make([]int, 0)
	for chunk, i := 0, 0; chunk < inputSize/2; i += 2 {
		chunk = scale << i
		chunksSizes = append(chunksSizes, chunk)
	}
	return chunksSizes
}

func BenchmarkSortFileMemMap(b *testing.B) {
	const (
		inputName  = "input"
		outputName = "output"

		inLineDigits  = 10
		inLineLetters = 10
		inLineSymbols = inLineDigits + inLineLetters + testutils.InLinePunctuation

		mib        = 1 << 20
		inputSize  = mib << 7
		totalLines = inputSize / inLineSymbols
	)

	rSeed := rand.New(rand.NewSource(1))
	chunksSizes := initSizes(inputSize, mib)
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
		b.Run(fmt.Sprintf("chunkSize=%dMib", bpc/mib), func(b *testing.B) {
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

func BenchmarkSortFileOs(b *testing.B) {
	const (
		inputName  = "input"
		outputName = "output"

		inLineDigits  = 10
		inLineLetters = 10
		inLineSymbols = inLineDigits + inLineLetters + testutils.InLinePunctuation

		mib        = 1 << 20
		inputSize  = mib << 7
		totalLines = inputSize / inLineSymbols
	)

	rSeed := rand.New(rand.NewSource(1))
	chunksSizes := initSizes(inputSize, mib)
	genLetters := []byte("abcd")

	mockFs := afero.NewOsFs()
	// Использует $TMPDIR
	tmpDir := b.TempDir()
	inputPath := filepath.Join(tmpDir, inputName)
	outputPath := filepath.Join(tmpDir, outputName)
	b.Logf("Process input %v to output %v", inputPath, outputPath)

	inputF, _, err := createInput(
		rSeed, inputPath, totalLines, inLineDigits, inLineLetters, genLetters, mockFs)
	if err != nil {
		b.Fatal(err)
	}
	if err := inputF.Close(); err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() {
		mockFs.Remove(inputPath)
		mockFs.Remove(outputPath)
	})

	for _, bpc := range chunksSizes {
		b.Run(fmt.Sprintf("chunkSize=%dMib", bpc/mib), func(b *testing.B) {
			cfg := SortConfig{
				InPath:        inputPath,
				OutPath:       outputPath,
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
