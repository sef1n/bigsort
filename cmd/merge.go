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
	cmpField int
	line     string
	reader   *bufio.Scanner
	file     io.Reader
	err      error
}

func NewFieldReader(f io.Reader) (*FieldReader, error) {
	scn := bufio.NewScanner(f)
	scn.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	fr := &FieldReader{reader: scn, file: f}
	if !fr.Scan() {
		if fr.err == nil {
			fr.err = io.EOF
		}
	}
	return fr, nil
}

func (fr *FieldReader) Scan() bool {
	if !fr.reader.Scan() {
		fr.err = fr.reader.Err()
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

func (h ReadersHeap) Len() int           { return len(h) }
func (h ReadersHeap) Less(i, j int) bool { return h[i].cmpField < h[j].cmpField }
func (h ReadersHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

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

func MergeWithContext(ctx context.Context, inS []*os.File, out io.Writer) error {
	h := make(ReadersHeap, 0, len(inS))

	for _, f := range inS {
		r, err := NewFieldReader(f)
		if err != nil {
			return err
		}
		if r.err == nil {
			h = append(h, r)
		} else if r.err != io.EOF {
			return r.err
		}
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
		} else if popped.err != nil && popped.err != io.EOF {
			return popped.err
		}
	}
	return w.Flush()
}
