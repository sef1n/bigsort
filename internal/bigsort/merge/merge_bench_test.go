package merge

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"math/rand"
	"slices"
	"strconv"
	"testing"

	"bigFiles/internal/bigsort/testutils"

	"github.com/spf13/afero"
)

const totalLines = 1 << 20

var filesCounts = []int{8, 16, 64, 256, 1024}

func BenchmarkMergeWithContext(b *testing.B) {
	for _, filesN := range filesCounts {
		linesPerFile := totalLines / filesN
		name := fmt.Sprintf("files=%d/lines=%d", filesN, linesPerFile)
		b.Run(name, func(b *testing.B) {
			mockFs := afero.NewMemMapFs()
			files, totalBytes, err := setupMergeFiles(mockFs, filesN, linesPerFile)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				for _, f := range files {
					testutils.DelF(mockFs, f)
				}
			}()

			b.SetBytes(totalBytes)
			b.ReportAllocs()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Сброс позиций — вне таймера
				b.StopTimer()
				for _, f := range files {
					if _, err := f.Seek(0, io.SeekStart); err != nil {
						b.Fatal(err)
					}
				}
				b.StartTimer()

				// io.Discard: не меряем стоимость записи на диск,
				// только CPU merge (heap + Atoi + WriteString в bufio)
				if err := MergeWithContext(context.Background(), 0, files, io.Discard); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// setupMergeFiles создаёт filesN файлов по linesPerFile строк.
// Каждый файл отсортирован по первому полю (как после splitOnSortedChunks).
// Возвращает открытые файлы (позиция в начале) и суммарный размер в байтах.
func setupMergeFiles(fsys afero.Fs, filesN, linesPerFile int) ([]afero.File, int64, error) {
	rSeed := rand.New(rand.NewSource(testutils.RandSeed))
	const inLineDigits, inLineLetters = 10, 10
	genLetters := []byte("abcd")

	files := make([]afero.File, 0, filesN)
	var totalBytes int64

	for i := 0; i < filesN; i++ {
		// Генерим и сортируем строки (эмуляция processChunk)
		lines := make([][]byte, linesPerFile)
		for j := 0; j < linesPerFile; j++ {
			lines[j] = testutils.UtilGenLine(rSeed, inLineDigits, inLineLetters, genLetters)
		}
		slices.SortFunc(lines, func(a, b []byte) int {
			return cmp.Compare(firstField(a), firstField(b))
		})

		f, err := fsys.Create(fmt.Sprintf("input_%d", i))
		if err != nil {
			return nil, 0, err
		}
		w := bufio.NewWriterSize(f, 64*1024)
		for _, line := range lines {
			if _, err := w.Write(line); err != nil {
				return nil, 0, err
			}
			totalBytes += int64(len(line))
		}
		if err := w.Flush(); err != nil {
			return nil, 0, err
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, 0, err
		}
		files = append(files, f)
	}
	return files, totalBytes, nil
}

func firstField(line []byte) int {
	idx := bytes.IndexByte(line, ',')
	if idx < 0 {
		return 0
	}
	v, _ := strconv.Atoi(string(line[:idx]))
	return v
}
