package split

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

	"bigFiles/internal/bigsort/csv"

	"github.com/calmdocs/ulimit"
	"github.com/spf13/afero"
	"golang.org/x/sync/errgroup"
)

// ChunkPlan — параметры одного прохода split
type ChunkPlan struct {
	In io.Reader
	// Директория для временных файлов
	TmpDir      string
	CmpFieldNum int

	// Soft limit памяти на чанк.
	// Может быть превышен
	BytesPerChunk int

	// Оценочное кол-во чанков в inFile
	EstimatedChunks uint64
}

const DefaultBytesPerChunk = 16 * (1024 * 1024) // 16 мб

// SplitOnSortedChunks
// Последовательно читает файл чанками по bytesPerChunk строк.
// Пул воркеров (NumCPU*2) параллельно сортирует чанки и пишет их
// в отдельные tmp-файлы
//
// При больших файлах может превысить ulimit
func SplitOnSortedChunks(parentCtx context.Context, cfg ChunkPlan, fsys afero.Fs) ([]afero.File, error) {
	// TODO: распараллелить чтение таблицы через offset?
	tableReader := bufio.NewReader(cfg.In)

	var (
		mu      sync.Mutex
		middles = make([]afero.File, 0, cfg.EstimatedChunks)
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
		tableParseStep, err := csv.ParseCSVTable(tableReader, cfg.CmpFieldNum, cfg.BytesPerChunk)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("parsing table: %w", err)
		}
		// Захватываем переменные (go < 1.22)
		entries := tableParseStep.Entries
		chunk := curChunk

		// Chunk sorting && spliting
		g.Go(func() error {
			middleFile, err := processChunk(gCtx, chunk, entries, cfg.TmpDir, fsys)
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
	if err := g.Wait(); err != nil {
		return nil, err
	}
	ok = true
	return middles, nil
}

// EstimateChunks оценивает количество чанков в переданном файле основываясь на ulimit и file size
func EstimateChunks(toChunksFile afero.File, bytesPerChunk int) (uint64, error) {
	if bytesPerChunk <= 0 {
		return 0, fmt.Errorf("bytesPerChunk must be > 0, got %d", bytesPerChunk)
	}
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

// processChunk выполняет сортировку и записывает отсортированный чанк во временный файл
func processChunk(parentCtx context.Context, chunk int, entries []csv.FieldEntry, tmpFilesDirPath string, fsys afero.Fs) (afero.File, error) {
	if err := parentCtx.Err(); err != nil {
		return nil, err
	}

	// Chunk sorting
	slices.SortFunc(entries, func(a, b csv.FieldEntry) int {
		return cmp.Compare(a.Field, b.Field)
	})

	// Проверка контекста в производися в createMiddleFile

	// Chunk spliting
	middleFile, err := createMiddleFile(parentCtx, entries, tmpFilesDirPath, fsys)
	if err != nil {
		return nil, fmt.Errorf("generating middles for chunk %v: %w", chunk, err)
	}

	return middleFile, nil
}

// createMiddleFile создает tmp файл, заполняет его строками processingEntries. В случае ошибки закрывает и удаляет файл
func createMiddleFile(ctx context.Context, processingEntries []csv.FieldEntry, creationPath string, fsys afero.Fs) (tmpF afero.File, err error) {
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

		if _, err = w.Write(v.Line); err != nil {
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
