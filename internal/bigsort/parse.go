package bigsort

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

const (
	// Ожидаемая средняя длинна строки в таблице
	predictAvgLineLen = 200 // байт
	// Размер временного файла
	// TODO: считать в байтах?
	tmpFileFieldsCount = 1000 // строк
)

type parseStep struct {
	entries   []fieldEntry
	readlines int // число итераций ReadBytes, включая пустую EOF-итерацию — для нумерации ошибок
}

type parseError struct {
	Line int
	Err  error // исходная ошибка
}

func (e *parseError) Error() string {
	return fmt.Sprintf("line %d: %v", e.Line, e.Err)
}

func (e *parseError) Unwrap() error { return e.Err }

// parseCSVTable читает не более maxParsed непустых строк из br.
// начало следующей строки (для продолжения).
//
// Возвращает:
//   - (parseStep, nil) — чанк прочитан, данные есть, продолжаем.
//   - (parseStep, io.EOF)  — последний чанк; step валиден.
//   - (step{}, *parseError) — ошибка; step невалиден.
//
// Пустые строки (после отсечения \r\n) пропускаются; строки из
// одних пробелов — ошибка парсинга.
// Требует maxParsed > 0.
func parseCSVTable(br *bufio.Reader, cmpFieldNum, maxParsedBytes int) (parseStep, error) {
	if maxParsedBytes <= 0 {
		return parseStep{}, &parseError{0, fmt.Errorf("maxParsedBytes must be > 0, got %d", maxParsedBytes)}
	}
	processingFields := make([]fieldEntry, 0, min(maxParsedBytes/predictAvgLineLen, 100_000)) // Защита

	// Размер успешно распаршенных строк в байтах
	parsedBytes := 0
	// Количество прочтанных строк br.ReadBytes (включая строку io.EOF)
	readLines := 0
	for parsedBytes < maxParsedBytes {
		line, readErr := br.ReadBytes('\n')
		readLines++
		if readErr != nil && readErr != io.EOF {
			return parseStep{}, &parseError{readLines, fmt.Errorf("reading: %w", readErr)}
		}

		// Обрезаем только крайние символы
		line = bytes.TrimSuffix(line, []byte("\n"))
		line = bytes.TrimSuffix(line, []byte("\r"))

		if len(line) > 0 {
			nthF, parseErr := nthField(line, cmpFieldNum)
			if parseErr != nil {
				return parseStep{}, &parseError{readLines, fmt.Errorf("parsing: %w", parseErr)}
			}

			fi, parseErr := newFieldEntry(line, string(nthF))
			if parseErr != nil {
				return parseStep{}, &parseError{readLines, fmt.Errorf("saving: %w", parseErr)}
			}

			processingFields = append(processingFields, fi)
			parsedBytes += len(line)
		}

		if readErr == io.EOF {
			return parseStep{processingFields, readLines}, io.EOF
		}
	}
	return parseStep{processingFields, readLines}, nil
}

var errFieldNotFound error = errors.New("field not found")

// nthField разбивает строку на поля по разделителю ',', возвращает n поле
func nthField(line []byte, n int) ([]byte, error) {
	start := 0
	for i := 0; i < n; i++ {
		idx := bytes.IndexByte(line[start:], ',')
		if idx < 0 {
			return nil, fmt.Errorf("%w: no %d in %q", errFieldNotFound, n, line)
		}
		start += idx + 1
	}
	end := bytes.IndexByte(line[start:], ',')
	if end < 0 {
		end = len(line) - start
	}
	return line[start : start+end], nil
}

// TODO: Любой Entry?
type fieldEntry struct {
	line  []byte
	field int
}

func newFieldEntry(line []byte, field string) (fieldEntry, error) {
	v, err := strconv.Atoi(field)
	if err != nil {
		return fieldEntry{}, err
	}
	return fieldEntry{line, v}, nil
}
