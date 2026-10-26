package bigsort

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
)

// SortConfig - параметры external sort
type SortConfig struct {
	InPath string

	// Путь, по которому создастся отсортированный файл.
	// Не используется io.Writer, чтобы не потерять данные в случае ошибки
	// (если пишется в исходный)
	OutPath     string
	CmpFieldNum int

	// Soft limit памяти на чанк.
	// Может быть превышен
	BytesPerChunk int

	// SkipFDLimit   bool
}

const defaultBytesPerChunk = 16 * (1024 * 1024) // 16 мб

func DefaultSortConfig() SortConfig {
	return SortConfig{BytesPerChunk: defaultBytesPerChunk}
}

func (cfg SortConfig) validate() error {
	if cfg.CmpFieldNum < 0 {
		return fmt.Errorf("field num must be >= 0, got %v", cfg.CmpFieldNum)
	}
	if cfg.BytesPerChunk <= 0 {
		return fmt.Errorf("chunk size must be > 0, got %v", cfg.BytesPerChunk)
	}
	return nil
}

// SortFileParallel сортирует CSV файл, распараллелив parse+sort+split по чанкам.
// В случае ошибки переименовывания файл-результат не удаляется.
//
// Пайплайн:
//  1. Reader последовательно читает файл чанками по linesChunk строк.
//  2. Пул воркеров (NumCPU*2) параллельно сортирует чанки и пишет их
//     в отдельные tmp-файлы (по одному на блок tmpFileFieldsCount строк).
//  3. Merge-фаза сливает все tmp-файлы в outFilePath (атомарно через rename).
//
// Ограничения:
//   - Число tmp-файлов = (всего строк) / (linesChunk * tmpFileFieldsCount).
//     При больших файлах может превысить ulimit — увеличь linesChunk
func SortFile(parentCtx context.Context, cfg SortConfig, fsys afero.Fs) error {
	// TODO: счетчик открытых файлов, их слияние при приближению к лимиту.
	if err := cfg.validate(); err != nil {
		return fmt.Errorf("params validating: %w", err)
	}

	inFile, err := fsys.Open(cfg.InPath)
	if err != nil {
		return err
	}
	defer func() { _ = inFile.Close() }()
	// Вычисление ожидаемого числа чанков в inFile
	estimatedChunks, err := estimateChunks(inFile, cfg.BytesPerChunk)
	if err != nil {
		return fmt.Errorf("chunk estimating: %w", err)
	}

	// Split
	tmpFilesDirPath := filepath.Dir(cfg.OutPath)
	tmpFiles, err := splitOnSortedChunks(parentCtx, chunkPlan{
		inFile:          inFile,
		tmpDir:          tmpFilesDirPath,
		cmpFieldNum:     cfg.CmpFieldNum,
		bytesPerChunk:   cfg.BytesPerChunk,
		estimatedChunks: estimatedChunks,
	}, fsys)
	if err != nil {
		return fmt.Errorf("spliting on chunks: %w", err)
	}
	defer func() {
		for _, f := range tmpFiles {
			name := f.Name()
			_ = f.Close()
			_ = fsys.Remove(name)
		}
	}()

	// Merge
	outFile, err := afero.TempFile(fsys, tmpFilesDirPath, "sorted-*.csv.tmp")
	if err != nil {
		return fmt.Errorf("creating tmp result: %w", err)
	}
	// Сохраняем имя до Close - гарантированно валидно
	outNameTmp := outFile.Name()

	keepTmpFile := false
	defer func() {
		_ = outFile.Close() // если вышли до явного Close
		if !keepTmpFile {
			_ = fsys.Remove(outNameTmp)
		}
	}()

	if err = mergeWithContext(parentCtx, cfg.CmpFieldNum, tmpFiles, outFile); err != nil {
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
	if err = fsys.Rename(outNameTmp, cfg.OutPath); err != nil {
		keepTmpFile = true
		return fmt.Errorf("rename %s -> %s: %w; result saved to %s", outNameTmp, cfg.OutPath, err, outNameTmp)
	}

	keepTmpFile = true
	return nil
}
