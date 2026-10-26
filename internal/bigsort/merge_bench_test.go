package bigsort

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

	"github.com/spf13/afero"
)

// setupMergeFiles создаёт filesN файлов по linesPerFile строк.
// Каждый файл отсортирован по первому полю (как после splitOnSortedChunks).
// Возвращает открытые файлы (позиция в начале) и суммарный размер в байтах.
func setupMergeFiles(fsys afero.Fs, filesN, linesPerFile int) ([]afero.File, int64, error) {
	rSeed := rand.New(rand.NewSource(1))
	const inLineDigits, inLineLetters = 10, 10
	genLetters := []byte("abcd")

	files := make([]afero.File, 0, filesN)
	var totalBytes int64

	for i := 0; i < filesN; i++ {
		// Генерим и сортируем строки (эмуляция processChunk)
		lines := make([][]byte, linesPerFile)
		for j := 0; j < linesPerFile; j++ {
			lines[j] = utilGenLine(rSeed, inLineDigits, inLineLetters, genLetters)
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

func BenchmarkMergeWithContext(b *testing.B) {
	cases := []struct {
		filesN       int
		linesPerFile int
	}{
		{10, 1000},   // ~200 KB
		{100, 1000},  // ~2 MB
		{500, 1000},  // ~10 MB — типичный размер для 10 GB с 20 MB чанками
		{100, 10000}, // ~22 MB, но мало файлов
	}

	for _, tc := range cases {
		tc := tc
		name := fmt.Sprintf("files=%d/lines=%d", tc.filesN, tc.linesPerFile)
		b.Run(name, func(b *testing.B) {
			mockFs := afero.NewMemMapFs()
			files, totalBytes, err := setupMergeFiles(mockFs, tc.filesN, tc.linesPerFile)
			if err != nil {
				b.Fatal(err)
			}
			defer func() {
				for _, f := range files {
					delF(mockFs, f)
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
				if err := mergeWithContext(context.Background(), 0, files, io.Discard); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
