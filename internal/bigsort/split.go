package bigsort

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"slices"
	"sync"

	"github.com/calmdocs/ulimit"
	"github.com/spf13/afero"
	"golang.org/x/sync/errgroup"
)

// splitOnSortedChunks
// Последовательно читает файл чанками по bytesPerChunk строк.
// Пул воркеров (NumCPU*2) параллельно сортирует чанки и пишет их
// в отдельные tmp-файлы (по одному на блок tmpFileFieldsCount строк).
//
// Tmp-файлы пишутся через Seek по offset'ам
// При больших файлах может превысить ulimit
func splitOnSortedChunks(parentCtx context.Context, cmpFieldNum int, inFilePath, tmpFilesDirPath string, bytesPerChunk int, fsys afero.Fs) ([]afero.File, error) {
	// TODO: распараллелить чтение таблицы через offset?
	if bytesPerChunk <= 0 {
		return nil, fmt.Errorf("chunk size must be > 0, got %v", bytesPerChunk)
	}

	tableFile, err := fsys.Open(inFilePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tableFile.Close() }()
	tableReader := bufio.NewReader(tableFile)

	estimatedChunks, err := estimateChunks(tableFile, bytesPerChunk)
	if err != nil {
		return nil, fmt.Errorf("chunk estimating: %w", err)
	}

	var (
		mu      sync.Mutex
		middles = make([]afero.File, 0, estimatedChunks)
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
		for _, f := range middles {
			name := f.Name()
			_ = f.Close()
			_ = fsys.Remove(name)
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
		tableParseStep, err = parseCSVTable(tableReader, cmpFieldNum, bytesPerChunk)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("parsing table: %w", err)
		}

		// Захватываем переменные (go < 1.22)
		entries := tableParseStep.entries
		chunk := curChunk

		// Chunk sorting && spliting
		g.Go(func() error {
			middleFile, err := processChunk(gCtx, chunk, entries, tmpFilesDirPath, fsys)
			if err != nil {
				return err
			}
			mu.Lock()
			middles = append(middles, middleFile)
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
	return middles, nil
}

func estimateChunks(toChunksFile afero.File, bytesPerChunk int) (uint64, error) {
	tableStat, err := toChunksFile.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat input: %w", err)
	}
	tableSizeBytes := tableStat.Size()
	estimatedChunks := uint64((tableSizeBytes + int64(bytesPerChunk) - 1) / int64(bytesPerChunk))

	// Проверка лимитов ос на открытые файлы (+ запас)
	lim, err := ulimit.Get()
	if err != nil {
		return 0, fmt.Errorf("os ulimit: %w", err)
	}
	if estimatedChunks > lim-5 {
		return 0, fmt.Errorf("input too large: %d chunks (os max %d). Increase bytesPerChunk", estimatedChunks, lim)
	}
	return estimatedChunks, nil
}

// processChunk выполняет сортировку и разбиение чанков на tmp файлы
func processChunk(parentCtx context.Context, chunk int, entries []fieldEntry, tmpFilesDirPath string, fsys afero.Fs) (afero.File, error) {
	if err := parentCtx.Err(); err != nil {
		return nil, err
	}

	// Chunk sorting
	slices.SortFunc(entries, func(a, b fieldEntry) int {
		return cmp.Compare(a.field, b.field)
	})

	// Проверка контекста в производися в createMiddleFile

	// Chunk spliting
	middleFile, err := createMiddleFile(parentCtx, entries, tmpFilesDirPath, fsys)
	if err != nil {
		return nil, fmt.Errorf("generating middles for chunk %v: %w", chunk, err)
	}

	return middleFile, nil
}

// createMiddleFile создает tmp файл, заполняет его строками processingFields. В случае ошибки закрывает и удаляет файл
func createMiddleFile(ctx context.Context, processingEntries []fieldEntry, creationPath string, fsys afero.Fs) (tmpF afero.File, err error) {
	tmpF, err = afero.TempFile(fsys, creationPath, "")
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			_ = fsys.Remove(tmpF.Name())
			_ = tmpF.Close()
			tmpF = nil
		}
	}()

	w := bufio.NewWriterSize(tmpF, 64*1024)

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
