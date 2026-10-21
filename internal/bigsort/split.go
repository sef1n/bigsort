package bigsort

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"

	"golang.org/x/sync/errgroup"
)

// genMiddleFiles создает tmp файлы содержащие processingFields. Кол-во строк в новых файлах = tmpFileFieldsCount.
// Одновременно запускает не более NumCPU*2 горутин.
// Возвращает слайс файлов, закрывает и удаляет их при ошибке.
func genMiddleFiles(parentCtx context.Context, processingEntries []fieldEntry, source, creationPath string) ([]*os.File, error) {
	tmpFs := make([]*os.File, (len(processingEntries)+tmpFileFieldsCount-1)/tmpFileFieldsCount)

	g, ctx := errgroup.WithContext(parentCtx)
	g.SetLimit(runtime.NumCPU() * 2)

	// Разбиваем слайс на блоки, в отдельных горутинах пишем в файлы
	for idx, start := 0, 0; start < len(processingEntries); idx, start = idx+1, start+tmpFileFieldsCount {
		end := min(start+tmpFileFieldsCount, len(processingEntries))
		g.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			f, err := createTMP(ctx, processingEntries[start:end], source, creationPath)
			if err != nil {
				return fmt.Errorf("creating block [%d:%d]: %w", start, end, err)
			}
			tmpFs[idx] = f
			return nil
		})
	}

	// При любой ошибке удаляем все файлы
	if err := g.Wait(); err != nil {
		for _, f := range tmpFs {
			if f == nil {
				continue
			}
			_ = f.Close()
			_ = os.Remove(f.Name())
		}
		return nil, err
	}
	return tmpFs, nil
}

// createTMP создает tmp файл, заполняет его строками processingFields.
func createTMP(ctx context.Context, processingEntries []fieldEntry, sourcePath, creationPath string) (tmpF *os.File, err error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = source.Close() }()

	tmpF, err = os.CreateTemp(creationPath, "")
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			_ = os.Remove(tmpF.Name())
			_ = tmpF.Close()
			tmpF = nil
		}
	}()

	w := bufio.NewWriter(tmpF)
	reader := bufio.NewReader(source)

	for _, v := range processingEntries {
		if err = ctx.Err(); err != nil {
			return
		}
		if _, err = source.Seek(v.Offset, io.SeekStart); err != nil {
			return
		}
		reader.Reset(source)

		var line string
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return
		}

		if _, err = w.WriteString(line); err != nil {
			return
		}
	}
	if err = w.Flush(); err != nil {
		return
	}
	if _, err = tmpF.Seek(0, 0); err != nil {
		return
	}
	return tmpF, nil
}
