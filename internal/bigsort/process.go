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
	"slices"
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
// Tmp-файлы пишутся через Seek по offset'ам
//
// Ограничения:
//   - Число tmp-файлов = (всего строк) / (linesChunk * tmpFileFieldsCount).
//     При больших файлах может превысить ulimit — увеличь linesChunk или
//     используй SortFile.
func SortFileParallel(parentCtx context.Context, cmpFieldNum int, inFilePath, outFilePath string) error {
	tmpFilesDirPath := filepath.Dir(outFilePath)
	tmpFiles, err := splitOnSortedChunks(parentCtx, cmpFieldNum, inFilePath, tmpFilesDirPath)
	if err != nil {
		return fmt.Errorf("spliting on chunks: %w", err)
	}
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
