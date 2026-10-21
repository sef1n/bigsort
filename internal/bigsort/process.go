package bigsort

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"
)

const (
	linesChunk             = 1000
	defaultChunkAllocation = 100
)

// ProcessExistingFile обрабатывает уже существующую csv таблицу inFilePath: parse, sort, split, merge.
// Сохраняет результат в outFilePath
func SortFile(ctx context.Context, cmpFieldNum int, inFilePath, outFilePath string) error {
	// Parse
	tableFile, err := os.Open(inFilePath)
	if err != nil {
		return err
	}
	// WARN: заглушка
	tableParseStep, err := parseCSVTable(bufio.NewReader(tableFile), cmpFieldNum, 1<<30, int64(0))
	_ = tableFile.Close()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return fmt.Errorf("parsing table: %w", err)
		}
	}

	// Sort
	slices.SortFunc(tableParseStep.entries, func(a, b fieldEntry) int {
		return cmp.Compare(a.Entry, b.Entry)
	})

	// Split
	tmpFilesDirPath := filepath.Dir(outFilePath)
	tmpFiles, err := genMiddleFiles(ctx, tableParseStep.entries, inFilePath, tmpFilesDirPath)
	if err != nil {
		return fmt.Errorf("generating middles: %w", err)
	}
	// Defer close&&remove
	defer func() {
		for _, f := range tmpFiles {
			name := f.Name()
			_ = f.Close()
			_ = os.Remove(name)
		}
	}()

	// Merge
	outFile, err := os.CreateTemp(tmpFilesDirPath, "sorted-*.csv.tmp")
	if err != nil {
		return fmt.Errorf("creating tmp result: %w", err)
	}
	// Сохранияем имя до Close - гарантированно валидно
	outNameTmp := outFile.Name()

	keepTmpFile := false
	// TODO: обрабатывать ошибки?
	defer func() {
		_ = outFile.Close() // если вышли до явного Close
		if !keepTmpFile {
			_ = os.Remove(outNameTmp)
		}
	}()

	if err = mergeWithContext(ctx, tmpFiles, outFile); err != nil {
		return fmt.Errorf("merging middles: %w", err)
	}
	// Закрываем перед переименованием (избегаем проблем на Windows)
	// Ошибка Close означает, что данные могли не записаться.
	// Файл считается невалидным и удаляется в defer.
	if err = outFile.Close(); err != nil {
		return fmt.Errorf("closing output file: %w", err)
	}
	// В случае ошибки переименовывания файл не удаляется,
	// возвращаемая ошибка содержит имя файла с результатом
	if err = os.Rename(outNameTmp, outFilePath); err != nil {
		keepTmpFile = true
		return fmt.Errorf("rename %s -> %s: %w; result saved to %s", outNameTmp, outFilePath, err, outNameTmp)
	}

	keepTmpFile = true
	return nil
}

// SortFileParallel сортирует CSV файл, распараллелив parse+sort+split по чанкам.
//
// Пайплайн:
//  1. Reader последовательно читает файл чанками по linesChunk строк.
//  2. Пул воркеров (NumCPU*2) параллельно сортирует чанки и пишет их
//     в отдельные tmp-файлы (по одному на блок tmpFileFieldsCount строк).
//  3. Merge-фаза сливает все tmp-файлы в outFilePath (атомарно через rename).
//
// Входной файл читается один раз, но tmp-файлы пишутся через Seek по offset'ам
//
// Ограничения:
//   - Число tmp-файлов = (всего строк) / (linesChunk * tmpFileFieldsCount).
//     При больших файлах может превысить ulimit — увеличь linesChunk или
//     используй SortFile.
func SortFileParallel(parentCtx context.Context, cmpFieldNum int, inFilePath, outFilePath string) error {
	var mu sync.Mutex
	tmpFilesDirPath := filepath.Dir(outFilePath)
	tmpFiles := make([]*os.File, 0, defaultChunkAllocation)
	defer func() {
		for _, f := range tmpFiles {
			name := f.Name()
			_ = f.Close()
			_ = os.Remove(name)
		}
	}()

	// NOTE: анониная функция - ограничиваем видимость переменных
	err := func() error {
		// Parse
		tableFile, err := os.Open(inFilePath)
		if err != nil {
			return err
		}
		defer func() { _ = tableFile.Close() }()
		tableReader := bufio.NewReader(tableFile)

		ctx, cancel := context.WithCancel(parentCtx)
		defer cancel()
		g, gCtx := errgroup.WithContext(ctx)
		g.SetLimit(runtime.NumCPU() * 2)

		// Параллельно парсим и разбиваем на временные файлы
		var tableOffset int64
		for curChunk := 0; ; curChunk++ {
			if err := gCtx.Err(); err != nil {
				cancel()
				_ = g.Wait()
				return err
			}
			// Chunk parsing
			tableParseStep, err := parseCSVTable(tableReader, cmpFieldNum, linesChunk, tableOffset)
			if err != nil && !errors.Is(err, io.EOF) {
				cancel()
				_ = g.Wait()
				return fmt.Errorf("parsing table: %w", err)
			}
			tableOffset = tableParseStep.offset

			// Захватываем переменные (go < 1.22)
			entries := tableParseStep.entries
			chunk := curChunk

			// Chunk sorting && spliting
			g.Go(func() error {
				if err := gCtx.Err(); err != nil {
					return err
				}
				// Chunk sorting
				slices.SortFunc(entries, func(a, b fieldEntry) int {
					return cmp.Compare(a.Entry, b.Entry)
				})
				// Проверка контекста в genMiddleFiles

				// Chunk spliting
				tmpChunkFiles, err := genMiddleFiles(gCtx, entries, inFilePath, tmpFilesDirPath)
				if err != nil {
					return fmt.Errorf("generating middles for chunk %v: %w", chunk, err)
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
		return g.Wait()
	}()
	if err != nil {
		return err
	}

	// Merge
	outFile, err := os.CreateTemp(tmpFilesDirPath, "sorted-*.csv.tmp")
	if err != nil {
		return fmt.Errorf("creating tmp result: %w", err)
	}
	// Сохранияем имя до Close - гарантированно валидно
	outNameTmp := outFile.Name()

	keepTmpFile := false
	defer func() {
		_ = outFile.Close() // если вышли до явного Close
		if !keepTmpFile {
			_ = os.Remove(outNameTmp)
		}
	}()

	if err = mergeWithContext(parentCtx, tmpFiles, outFile); err != nil {
		return fmt.Errorf("merging middles: %w", err)
	}
	// Закрываем перед переименованием (избегаем проблем на Windows)
	// Ошибка Close означает, что данные могли не записаться.
	// Файл считается невалидным и удаляется в defer.
	if err = outFile.Close(); err != nil {
		return fmt.Errorf("closing output file: %w", err)
	}
	// В случае ошибки переименовывания файл не удаляется,
	// возвращаемая ошибка содержит имя файла с результатом
	if err = os.Rename(outNameTmp, outFilePath); err != nil {
		keepTmpFile = true
		return fmt.Errorf("rename %s -> %s: %w; result saved to %s", outNameTmp, outFilePath, err, outNameTmp)
	}

	keepTmpFile = true
	return nil
}
