package bigsort

import (
	"bufio"
	"container/heap"
	"context"
	"io"
	"os"
)

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

	w := bufio.NewWriterSize(out, 64*1024)
	// defer w.Flush() - нужно возвращать ошибку

	for h.Len() > 0 {
		if err := ctx.Err(); err != nil {
			return err
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
