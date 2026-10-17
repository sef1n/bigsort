package bigsort

import (
	"bufio"
	"container/heap"
	"context"
	"io"
	"os"
)

type readersHeap []*fieldReader

// sort.Interface
func (h readersHeap) Len() int           { return len(h) }
func (h readersHeap) Less(i, j int) bool { return h[i].cmpField < h[j].cmpField }
func (h readersHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

// heap.Interface
func (h *readersHeap) Push(x any) {
	*h = append(*h, x.(*fieldReader))
}

func (h *readersHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// mergeWithContext выполняет merge-слияние файлов, пишет результат в out.
// При ошибке не выполняет Flush().
// TODO: принимает []io.Reader?
func mergeWithContext(ctx context.Context, inS []*os.File, out io.Writer) error {
	h := make(readersHeap, 0, len(inS))

	for _, f := range inS {
		r := newFieldReader(f)
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

		popped := heap.Pop(&h).(*fieldReader)

		if _, err := w.WriteString(popped.line); err != nil {
			return err
		}
		if err := w.WriteByte('\n'); err != nil {
			return err
		}

		if popped.scan() {
			heap.Push(&h, popped)
		} else if popped.err != io.EOF {
			return popped.err
		}
	}
	return w.Flush()
}
