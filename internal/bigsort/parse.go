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
	predictTableLines  = 100
	tmpFileFieldsCount = 1000
)

type parseStep struct {
	entries   []fieldEntry
	readlines int   // число итераций ReadBytes, включая пустую EOF-итерацию — для нумерации ошибок
	offset    int64 // offset начала следующей строки
}

type parseError struct {
	Line   int
	Offset int64 // байтовый offset начала проблемной строки
	Err    error // исходная ошибка
}

func (e *parseError) Error() string {
	return fmt.Sprintf("line %d (offset %d): %v", e.Line, e.Offset, e.Err)
}

func (e *parseError) Unwrap() error { return e.Err }

// parseCSVTable читает не более maxParsed непустых строк из br.
// offset — байтовая позиция начала чтения; возвращаемый offset —
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
func parseCSVTable(br *bufio.Reader, cmpFieldNum, maxParsed int, offset int64) (parseStep, error) {
	if maxParsed <= 0 {
		return parseStep{}, &parseError{0, offset, fmt.Errorf("maxParsed must be > 0, got %d", maxParsed)}
	}
	processingFields := make([]fieldEntry, 0, maxParsed)

	readLines, parsedLines := 0, 0
	for parsedLines < maxParsed {
		line, readErr := br.ReadBytes('\n')
		if readErr != nil && readErr != io.EOF {
			return parseStep{}, &parseError{readLines, offset, fmt.Errorf("reading: %w", readErr)}
		}
		readLines++

		rawLineLen := int64(len(line))
		line = bytes.TrimRight(line, "\r\n")

		if len(line) > 0 {
			nthF, parseErr := nthField(line, cmpFieldNum)
			if parseErr != nil {
				return parseStep{}, &parseError{readLines, offset, fmt.Errorf("parsing: %w", parseErr)}
			}

			fi, parseErr := newFieldEntry(offset, string(nthF))
			if parseErr != nil {
				return parseStep{}, &parseError{readLines, offset, fmt.Errorf("saving: %w", parseErr)}
			}

			processingFields = append(processingFields, fi)
			parsedLines++
		}

		offset += rawLineLen

		if readErr == io.EOF {
			return parseStep{processingFields, readLines, offset}, io.EOF
		}
	}
	return parseStep{processingFields, readLines, offset}, nil
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
	Offset int64
	Entry  int
}

func newFieldEntry(off int64, e string) (fieldEntry, error) {
	v, err := strconv.Atoi(e)
	if err != nil {
		return fieldEntry{}, err
	}
	return fieldEntry{off, v}, nil
}
