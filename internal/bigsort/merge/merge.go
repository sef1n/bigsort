package merge

import (
	"bufio"
	"context"
	"io"

	"github.com/sef1n/losertree"
	"github.com/spf13/afero"
)

// MergeWithContext выполняет merge-слияние файлов, пишет результат в out.
// При ошибке не выполняет Flush().
// TODO: принимает []io.Reader?
func MergeWithContext(ctx context.Context, cmpFieldNum int, inS []afero.File, out io.Writer) error {
	readers := make([]*fieldReader, 0, len(inS))
	initial := make([]int, 0, len(inS))

	for _, f := range inS {
		r, fieldV, err := newFieldReader(f, cmpFieldNum)
		if err != nil && err != io.EOF {
			return err
		}
		if len(r.line) == 0 {
			continue
		}
		initial = append(initial, fieldV)
		readers = append(readers, r)
	}

	w := bufio.NewWriterSize(out, 64*1024)
	// defer w.Flush() - нужно возвращать ошибку

	tree, ind, ok := losertree.Init(initial)
	// Не проверяем tree == nil т.к. ok == false => работа не выполняется
	for ok {
		if err := ctx.Err(); err != nil {
			return err
		}

		// line is reset to [:0] in Read(), so appending '\n' here is safe
		// Single Write (vs Write + WriteByte) saves one bufio call per line.
		popped := readers[ind]
		popped.line = append(popped.line, byte('\n'))

		if _, err := w.Write(popped.line); err != nil {
			return err
		}

		if fieldV, err := popped.Read(); err != nil {
			if err != io.EOF {
				return err
			}
			tree.Kill()
		} else {
			// Если нет ошибки, сразу добавляем
			tree.Upd(fieldV)
		}

		ind, ok = tree.Pop()
	}
	return w.Flush()
}
