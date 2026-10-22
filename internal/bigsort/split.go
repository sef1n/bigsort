package bigsort

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"
)

const (
	// Soft limit памяти на чанк
	// Может быть превышен
	LineChunkBytes         = 32 * (1024 * 1024) // 32 мб
	defaultChunkAllocation = 100
)

// splitOnSortedChunks
// Последовательно читает файл чанками по LineChunkBytes строк.
// Пул воркеров (NumCPU*2) параллельно сортирует чанки и пишет их
// в отдельные tmp-файлы (по одному на блок tmpFileFieldsCount строк).
//
// Tmp-файлы пишутся через Seek по offset'ам
// При больших файлах может превысить ulimit
func splitOnSortedChunks(parentCtx context.Context, cmpFieldNum int, inFilePath, tmpFilesDirPath string) ([]*os.File, error) {
	// Parse
	tableFile, err := os.Open(inFilePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tableFile.Close() }()
	tableReader := bufio.NewReader(tableFile)

	var (
		mu       sync.Mutex
		tmpFiles = make([]*os.File, 0, defaultChunkAllocation)
	)

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU() * 2)

	// Отложенная очистка в случае ошибки
	ok := false
	defer func() {
		if ok {
			return
		}
		cancel()
		_ = g.Wait()
		for _, f := range tmpFiles {
			name := f.Name()
			_ = f.Close()
			_ = os.Remove(name)
		}
	}()

	// Параллельно парсим и разбиваем на временные файлы
	for curChunk := 0; ; curChunk++ {
		if gCtx.Err() != nil {
			if gerr := g.Wait(); gerr != nil {
				return nil, gerr
			}
			return nil, gCtx.Err() // fallback
		}
		// Chunk parsing
		var tableParseStep parseStep
		tableParseStep, err = parseCSVTable(tableReader, cmpFieldNum, LineChunkBytes)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("parsing table: %w", err)
		}

		// Захватываем переменные (go < 1.22)
		entries := tableParseStep.entries
		chunk := curChunk

		// Chunk sorting && spliting
		g.Go(func() error {
			tmpChunkFiles, err := processChunk(gCtx, chunk, entries, inFilePath, tmpFilesDirPath)
			if err != nil {
				return err
			}
			mu.Lock()
			tmpFiles = append(tmpFiles, tmpChunkFiles...)
			mu.Unlock()
			return nil
		})

		if errors.Is(err, io.EOF) {
			break
		}
	}
	if err = g.Wait(); err != nil {
		return nil, err
	}
	ok = true
	return tmpFiles, nil
}

// processChunk выполняет сортировку и разбиение чанков на tmp файлы
func processChunk(parentCtx context.Context, chunk int, entries []fieldEntry, inFilePath, tmpFilesDirPath string) ([]*os.File, error) {
	if err := parentCtx.Err(); err != nil {
		return nil, err
	}

	// Chunk sorting
	slices.SortFunc(entries, func(a, b fieldEntry) int {
		return cmp.Compare(a.field, b.field)
	})
	// Проверка контекста в genMiddleFiles

	// Chunk spliting
	tmpChunkFiles, err := genMiddleFiles(parentCtx, entries, inFilePath, tmpFilesDirPath)
	if err != nil {
		return nil, fmt.Errorf("generating middles for chunk %v: %w", chunk, err)
	}

	return tmpChunkFiles, nil
}

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

	for _, v := range processingEntries {
		if err = ctx.Err(); err != nil {
			return
		}

		if _, err = w.Write(v.line); err != nil {
			return
		}
		if err = w.WriteByte('\n'); err != nil {
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
