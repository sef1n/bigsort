package bigsort

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// ProcessExistingFile обрабатывает уже существующую csv таблицу inFilePath: parse, sort, split, merge.
// Сохраняет результат в outFilePath
func SortFile(ctx context.Context, cmpFieldNum int, inFilePath, outFilePath string) error {
	// Parse
	tableEntries, err := parseCSVTable(inFilePath, cmpFieldNum)
	if err != nil {
		return fmt.Errorf("parsing table: %w", err)
	}

	// Sort
	slices.SortFunc(tableEntries, func(a, b fieldEntry) int {
		return cmp.Compare(a.Entry, b.Entry)
	})

	// Split
	tmpFilesDirPath := filepath.Dir(outFilePath)
	tmpFiles, err := genMiddleFiles(ctx, tableEntries, inFilePath, tmpFilesDirPath)
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
