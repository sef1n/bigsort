package main

import (
	"bufio"
	"container/heap"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type FieldReader struct {
	cmpField int // Поле, по которому выполняется сортировка
	line     string
	reader   *bufio.Scanner
	err      error // Ошибка чтения или io.EOF

	// Возможно имеет смысл
	// file     io.Reader
}

// NewFieldReader выполняет инициализацию FieldReader, в случае ошибки устанавливает её в FieldReader.err.
func NewFieldReader(f io.Reader) *FieldReader {
	scn := bufio.NewScanner(f)
	scn.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	fr := &FieldReader{reader: scn}
	// Scan сам устанавливает ошибки (io.EOF считается ошибкой)
	_ = fr.Scan()
	return fr
}

// Scan выполняет Scanner.Scan, устанавливает новые line и err.
// По окончании файла устанавливает err = io.EOF
func (fr *FieldReader) Scan() bool {
	if !fr.reader.Scan() {
		fr.err = fr.reader.Err()
		if fr.err == nil {
			fr.err = io.EOF
		}
		return false
	}
	fr.line = fr.reader.Text()

	idx := strings.IndexByte(fr.line, ',')
	if idx < 0 {
		fr.err = fmt.Errorf("no comma in line: %q", fr.line)
		return false
	}
	v, err := strconv.Atoi(fr.line[:idx])
	if err != nil {
		fr.err = fmt.Errorf("parse %q: %w", fr.line[:idx], err)
		return false
	}
	fr.cmpField = v
	return true
}

type ReadersHeap []*FieldReader

// sort.Interface
func (h ReadersHeap) Len() int           { return len(h) }
func (h ReadersHeap) Less(i, j int) bool { return h[i].cmpField < h[j].cmpField }
func (h ReadersHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

// heap.Interface
func (h *ReadersHeap) Push(x any) {
	*h = append(*h, x.(*FieldReader))
}

func (h *ReadersHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// MergeWithContext выполняет merge-слияние файлов, пишет результат в out.
// При ошибке не выполняет Flush().
// TODO: принимает []io.Reader?
func MergeWithContext(ctx context.Context, inS []*os.File, out io.Writer) error {
	h := make(ReadersHeap, 0, len(inS))

	for _, f := range inS {
		r := NewFieldReader(f)
		if r.err != nil {
			if r.err == io.EOF {
				continue
			}
			return r.err
		}
		h = append(h, r)
	}

	heap.Init(&h)

	w := bufio.NewWriter(out)
	// defer w.Flush() - нужно возвращать ошибку

	for counter := 0; h.Len() > 0; counter++ {
		if counter%100 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}

		popped := heap.Pop(&h).(*FieldReader)

		if _, err := w.WriteString(popped.line); err != nil {
			return err
		}
		if err := w.WriteByte('\n'); err != nil {
			return err
		}

		if popped.Scan() {
			heap.Push(&h, popped)
		} else if popped.err != io.EOF {
			return popped.err
		}
	}
	return w.Flush()
}
