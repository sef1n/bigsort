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
func MergeWithContext(ctx context.Context, cmpFieldNum int, inputs []afero.File, out io.Writer) error {
	// NOTE: There is no point in using []io.Reader instead of []afero.File now:
	// it leads to copying a slice of files or using generics,
	// which adds premature complexity.
	readers := make([]*fieldReader, 0, len(inputs))
	initial := make([]int, 0, len(inputs))

	for _, f := range inputs {
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
