package merge

// readersHeap — старая heap-реализация, оставлена ТОЛЬКО для
// сравнительных бенчмарков (BenchmarkMergeHeap vs BenchmarkMergeLoser).

import (
	"bufio"
	"bytes"
	"container/heap"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"testing"

	"bigFiles/internal/bigsort/csv"
	"bigFiles/internal/bigsort/testutils"

	"github.com/spf13/afero"
)

func BenchmarkMergeWithContextHeap(b *testing.B) {
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

				// io.Discard: не измеряем стоимость записи на диск,
				// только CPU merge (heap + Atoi + WriteString в bufio)
				if err := MergeWithContextHeap(context.Background(), 0, files, io.Discard); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

type readersHeap []*fieldReaderHeap

// sort.Interface
func (h readersHeap) Len() int           { return len(h) }
func (h readersHeap) Less(i, j int) bool { return h[i].cmpField < h[j].cmpField }
func (h readersHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

// heap.Interface
func (h *readersHeap) Push(x any) {
	*h = append(*h, x.(*fieldReaderHeap))
}

func (h *readersHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// MergeWithContext выполняет merge-слияние файлов, пишет результат в out.
// При ошибке не выполняет Flush().
func MergeWithContextHeap(ctx context.Context, cmpFieldNum int, inS []afero.File, out io.Writer) error {
	h := make(readersHeap, 0, len(inS))

	for _, f := range inS {
		r, err := NewFieldReaderHeap(f, cmpFieldNum)
		if err != nil && err != io.EOF {
			return err
		}
		if len(r.line) == 0 {
			continue
		}
		h = append(h, r)
	}
	heap.Init(&h)

	w := bufio.NewWriterSize(out, 64*1024)
	// defer w.Flush() - нужно возвращать ошибку

	for h.Len() > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}

		popped := heap.Pop(&h).(*fieldReaderHeap)
		popped.line = append(popped.line, byte('\n'))

		// Пишем сразу с \n
		if _, err := w.Write(popped.line); err != nil {
			return err
		}

		if err := popped.Read(); err != nil {
			if err != io.EOF {
				return err
			}
		} else {
			// Если нет ошибки, сразу добавляем
			heap.Push(&h, popped)
		}
	}
	return w.Flush()
}

type fieldReaderHeap struct {
	cmpFieldNum int // Номер поля (начиная с 0)
	cmpField    int // Поле, по которому выполняется сортировка
	line        []byte
	reader      *bufio.Reader

	// Возможно имеет смысл
	// file     io.Reader
}

// TODO: type fieldExtractor interface
// сейчас просто импорты csv

// NewFieldReader выполняет инициализацию FieldReader, в случае ошибки устанавливает её в FieldReader.err.
func NewFieldReaderHeap(f io.Reader, cmpFieldNum int) (*fieldReaderHeap, error) {
	r := bufio.NewReader(f)
	fr := &fieldReaderHeap{reader: r, cmpFieldNum: cmpFieldNum, line: make([]byte, 0, 20)}
	return fr, fr.Read()
}

// Read читает следующую строку в fr.line (ПЕРЕЗАПИСЫВАЯ его).
// Пустые строки (\n или \r\n без данных) НЕ пропускаются —
// caller должен гарантировать, что tmp-файлы их не содержат
// (это делает parseCSVTable). Иначе Atoi вернёт ошибку.
// После успеха fr.line не содержит \n/\r.
// Возвращает nil только с непустым fr.line.
func (fr *fieldReaderHeap) Read() error {
	fr.line = fr.line[:0]
	readLine, err := fr.reader.ReadSlice('\n')
	fullLineLen := len(readLine)
	if err != nil && err != io.EOF {
		if !errors.Is(err, bufio.ErrBufferFull) {
			return err
		}
		fr.line = append(fr.line, readLine...)

		readLine, err = fr.reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return err
		}
		fullLineLen += len(readLine)
	}

	// Не добавляем лишний раз
	if fullLineLen == 0 {
		return err // EOF без данных
	}

	fr.line = append(fr.line, readLine...)

	fr.line = bytes.TrimRight(fr.line, "\r\n")

	bytesField, err := csv.NthField(fr.line, fr.cmpFieldNum)
	if err != nil {
		return fmt.Errorf("parse %q: %w", bytesField, err)
	}
	v, err := strconv.Atoi(csv.BytesToString(bytesField))
	if err != nil {
		return fmt.Errorf("parse %q: %w", bytesField, err)
	}
	fr.cmpField = v
	return nil
}
