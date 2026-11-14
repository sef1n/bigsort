package csv

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unsafe"
)

const minLineLen = 20

type parseStep struct {
	Entries   []FieldEntry
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

// ParseCSVTable читает непустые строки из br с soft limit = MaxParsed.
// Использует ReadSlice с fallback на ReadBytes при любых ошибках чтения.
// Возвращает:
//   - (parseStep, nil) — чанк прочитан, данные есть, продолжаем.
//   - (parseStep, io.EOF)  — последний чанк; step валиден.
//   - (step{}, *parseError) — ошибка; step невалиден.
//
// Пустые строки (после отсечения \r\n) пропускаются;
// строки из одних пробелов — ошибка парсинга.
// Требует maxParsed > 0.
func ParseCSVTable(br *bufio.Reader, cmpFieldNum, maxParsedBytes int) (parseStep, error) {
	if maxParsedBytes <= 0 {
		return parseStep{}, &parseError{0, fmt.Errorf("maxParsedBytes must be > 0, got %d", maxParsedBytes)}
	}
	// NOTE: read+fallback duplicated intentionally - extraction (closure/helper)
	// adds ~2-5ns per line via indirect call + escape of arena/readLines,
	// which is 2-3% on CPU-bound benchmark. Keep inline.

	// Размер успешно распаршенных строк в байтах
	parsedBytes := 0
	// Количество прочтанных строк br.ReadSlice/br.ReadBytes (включая строку io.EOF)
	readLines := 0

	var processingFields []FieldEntry
	var linesArena []byte
	arenaOffset := 0

	// NOTE: we read the first line separately to allocate the correct size
	// processingFields and linesArena
	// to reduce the number of allocations when reading
	// We use ReadSlice + fallback on readBytes for very large lines
	// We take a cap with a margin for an extra page so as not to reallocate
	line, readErr := br.ReadSlice('\n')
	readLines++
	fullLineLen := len(line)

	if readErr != nil && readErr != io.EOF {
		// fallback только для длинных строк
		if !errors.Is(readErr, bufio.ErrBufferFull) {
			return parseStep{}, &parseError{readLines, fmt.Errorf("reading: %w", readErr)}
		}

		linesArena = make([]byte, 0, maxParsedBytes+fullLineLen)
		linesArena = append(linesArena, line...)

		// новый указатель - остаток строки
		line, readErr = br.ReadBytes('\n')
		if readErr != nil && readErr != io.EOF {
			return parseStep{}, &parseError{readLines, fmt.Errorf("reading (fell back to ReadBytes): %w", readErr)}
		}
		fullLineLen += len(line)
	} else {
		linesArena = make([]byte, 0, maxParsedBytes+fullLineLen)
	}

	processingFields = make([]FieldEntry, 0, maxParsedBytes/max(minLineLen, fullLineLen))

	for {
		linesArena = append(linesArena, line...)
		line = linesArena[arenaOffset : arenaOffset+fullLineLen]
		// arenaOffset меняем после trim

		// Обрезаем только крайние символы
		// Указатель на line меняется, в арене продолжает храниться обрезанное
		line = bytes.TrimSuffix(line, []byte("\n"))
		line = bytes.TrimSuffix(line, []byte("\r"))

		// Обновляем гранцы, исключаем обрезанное
		linesArena = linesArena[:arenaOffset+len(line)]
		arenaOffset += len(line)

		if len(line) > 0 {
			nthF, parseErr := NthField(line, cmpFieldNum)
			if parseErr != nil {
				return parseStep{}, &parseError{readLines, fmt.Errorf("parsing: %w", parseErr)}
			}

			fi, parseErr := newFieldEntry(line, nthF)
			if parseErr != nil {
				return parseStep{}, &parseError{readLines, fmt.Errorf("saving: %w", parseErr)}
			}

			processingFields = append(processingFields, fi)
			parsedBytes += len(line)
		}

		if readErr == io.EOF {
			return parseStep{processingFields, readLines}, io.EOF
		}

		if parsedBytes >= maxParsedBytes {
			break
		}

		line, readErr = br.ReadSlice('\n')
		readLines++
		// Переменная для подсчета позиции на арене после fallback
		fullLineLen = len(line)

		if readErr != nil && readErr != io.EOF {
			// NOTE: fallback только для длинных строк
			if !errors.Is(readErr, bufio.ErrBufferFull) {
				return parseStep{}, &parseError{readLines, fmt.Errorf("reading: %w", readErr)}
			}
			// Получена незавершенная строка,
			// чтобы не выделять память, сразу добавляем в арену
			linesArena = append(linesArena, line...)

			// Если дочитали до EOF, ошибка видна циклу
			// новый указатель - остаток строки
			line, readErr = br.ReadBytes('\n')
			if readErr != nil && readErr != io.EOF {
				return parseStep{}, &parseError{readLines, fmt.Errorf("reading (fell back to ReadBytes): %w", readErr)}
			}
			fullLineLen += len(line)
		}

	}
	return parseStep{processingFields, readLines}, nil
}

var ErrFieldNotFound = errors.New("field not found")

// NthField разбивает строку на поля по разделителю ',', возвращает n поле
func NthField(line []byte, n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("field num must be >= 0, got %v", n)
	}
	start := 0
	for i := 0; i < n; i++ {
		idx := bytes.IndexByte(line[start:], ',')
		if idx < 0 {
			return nil, fmt.Errorf("%w: no %d in %q", ErrFieldNotFound, n, line)
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
type FieldEntry struct {
	Line  []byte
	Field int
	// NOTE: если использовать поля для хранения arena offset (см. parseCSVTable),
	// то снизится нагрузка на gc, но нужно контролировать время жизни arena
}

// BytesToString — zero-copy []byte → string для read-only операций.
// Результат НЕ должен переживать b; используется только для
// strconv.Atoi, который читает и не сохраняет строку
func BytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func newFieldEntry(line []byte, field []byte) (FieldEntry, error) {
	v, err := strconv.Atoi(BytesToString(field))
	if err != nil {
		return FieldEntry{}, err
	}
	return FieldEntry{line, v}, nil
}
