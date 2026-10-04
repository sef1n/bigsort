package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"syscall"

	"golang.org/x/sync/errgroup"
)

const (
	predictTableLines  = 100
	tmpFileFieldsCount = 1000
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := ParseFlags()
	if err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	parentCtx := context.Background()
	if cfg.ctxTimeout != 0 {
		var cancel context.CancelFunc
		parentCtx, cancel = context.WithTimeout(parentCtx, cfg.ctxTimeout)
		defer cancel()
	}

	ctx, cancel := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return ProcessExistingFile(ctx, cfg.cmpField, cfg.inFile, cfg.outFile)
}

// ProcessExistingFile обрабатывает уже существующую csv таблицу inFilePath: parse, sort, split, merge.
// Сохраняет результат в outFilePath
func ProcessExistingFile(ctx context.Context, cmpFieldNum int, inFilePath, outFilePath string) error {
	// Parse
	sourceTableF, err := os.Open(inFilePath)
	if err != nil {
		return err
	}
	defer sourceTableF.Close()
	tableFields, err := ParseCSVTable(sourceTableF, cmpFieldNum)
	if err != nil {
		return err
	}

	// Sort
	sort.Sort(&tableFields)

	// Split
	tmpFilesDirPath := filepath.Dir(outFilePath)
	tmpFiles, err := genMiddleFiles(ctx, tableFields, inFilePath, tmpFilesDirPath)
	if err != nil {
		return err
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
		return err
	}
	outNameTmp := outFile.Name()
	defer func() {
		_ = outFile.Close()
		if err != nil {
			_ = os.Remove(outNameTmp)
		}
	}()

	if err = MergeWithContext(ctx, tmpFiles, outFile); err != nil {
		return err
	}
	// Закрываем перед переименованием (избегаем проблем на Windows)
	if err = outFile.Close(); err != nil {
		return err
	}
	if err = os.Rename(outNameTmp, outFilePath); err != nil {
		return err
	}
	return nil
}

// genMiddleFiles создает tmp файлы содержащие processingFields. Кол-во строк в новых файлах = tmpFileFieldsCount.
// Возвращает слайс файлов, закрывает и удаляет их при ошибке.
func genMiddleFiles(parentCtx context.Context, processingFields fields, source, creationPath string) ([]*os.File, error) {
	tmpFs := make([]*os.File, (len(processingFields)+tmpFileFieldsCount-1)/tmpFileFieldsCount)

	g, ctx := errgroup.WithContext(parentCtx)
	g.SetLimit(runtime.NumCPU() * 2)

	// Разбиваем слайс на блоки, в отдельных горутинах пишем в файлы
	for idx, start := 0, 0; start < len(processingFields); idx, start = idx+1, start+tmpFileFieldsCount {
		end := min(start+tmpFileFieldsCount, len(processingFields))
		g.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			f, err := createTMP(ctx, processingFields[start:end], source, creationPath)
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

// getFields читает строки из файла, сохраняет Nth поля, возвращает структуру сравниваемых полей
func ParseCSVTable(tableFile *os.File, cmpFieldNum int) (fields, error) {
	processingFields := make(fields, 0, predictTableLines)
	var offset int64
	r := bufio.NewReader(tableFile)

	for lineNum := 0; ; lineNum++ {
		line, readErr := r.ReadBytes('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, fmt.Errorf("read line %d: %w", lineNum, readErr)
		}

		RawLineLen := int64(len(line))
		line = bytes.TrimRight(line, "\r\n")

		if len(line) > 0 {
			nthF, parseErr := nthField(line, cmpFieldNum)
			if parseErr != nil {
				return nil, fmt.Errorf("parsing line %d: %w", lineNum, parseErr)
			}

			fi, parseErr := newField(offset, string(nthF), cmpFieldNum)
			if parseErr != nil {
				return nil, fmt.Errorf("saving line %d: %w", lineNum, parseErr)
			}

			processingFields = append(processingFields, fi)
		}

		offset += RawLineLen

		if readErr == io.EOF {
			break
		}
	}
	return processingFields, nil
}

// nthField разбивает строку на поля по разделителю ',', возвращает n поле
func nthField(line []byte, n int) ([]byte, error) {
	start := 0
	for i := 0; i < n; i++ {
		idx := bytes.IndexByte(line[start:], ',')
		if idx < 0 {
			return nil, fmt.Errorf("field %d not found in %q", n, line)
		}
		start += idx + 1
		if start > len(line) {
			return nil, fmt.Errorf("field %d beyond end", n)
		}
	}
	end := bytes.IndexByte(line[start:], ',')
	if end < 0 {
		end = len(line) - start
	}
	return line[start : start+end], nil
}

// createTMP создает tmp файл, заполняет его строками processingFields.
func createTMP(ctx context.Context, processingFields fields, sourcePath, creationPath string) (tmpF *os.File, err error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return nil, err
	}
	defer source.Close()

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

	for _, v := range processingFields {
		if err = ctx.Err(); err != nil {
			return
		}
		if _, err = source.Seek(v.offset, io.SeekStart); err != nil {
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
