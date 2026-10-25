package bigsort

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unsafe"
)

const (
	// Ожидаемая средняя длинна строки в таблице
	// TODO: оценивать по первой строке файла?
	predictAvgLineLen = 100 // байт
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

// parseCSVTable читает непустые строки из br с soft limit = MaxParsed.
// Использует ReadSlice с fallback на ReadBytes при любых ошибках чтения.
// Возвращает:
//   - (parseStep, nil) — чанк прочитан, данные есть, продолжаем.
//   - (parseStep, io.EOF)  — последний чанк; step валиден.
//   - (step{}, *parseError) — ошибка; step невалиден.
//
// Пустые строки (после отсечения \r\n) пропускаются;
// строки из одних пробелов — ошибка парсинга.
// Требует maxParsed > 0.
func parseCSVTable(br *bufio.Reader, cmpFieldNum, maxParsedBytes int) (parseStep, error) {
	if maxParsedBytes <= 0 {
		return parseStep{}, &parseError{0, fmt.Errorf("maxParsedBytes must be > 0, got %d", maxParsedBytes)}
	}
	// Нет смысла ограничивать cap - если возникают ошибки при выделении памяти,
	// нужно менять значение размера чанка
	processingFields := make([]fieldEntry, 0, maxParsedBytes/predictAvgLineLen)

	// Арена для уменьшения кол-ва аллокаций при чтении
	// Использует ReadSlice + fallback на ReadBytes для очень больших строк
	// Берем cap с запасом на лишнюю стрку, чтобы не реаллоцировать
	linesArena := make([]byte, 0, maxParsedBytes+predictAvgLineLen)
	arenaOffset := 0

	// Размер успешно распаршенных строк в байтах
	parsedBytes := 0
	// Количество прочтанных строк br.ReadSlice/br.ReadBytes (включая строку io.EOF)
	readLines := 0
	for parsedBytes < maxParsedBytes {
		line, readErr := br.ReadSlice('\n')
		readLines++
		if readErr != nil && readErr != io.EOF {
			buf := make([]byte, len(line), len(line)+256)
			copy(buf, line)
			line = buf

			rest, readErr := br.ReadBytes('\n')
			if readErr != nil && readErr != io.EOF {
				fmt.Println("FALLBACK ERR")
				return parseStep{}, &parseError{readLines, fmt.Errorf("reading (fell back to ReadBytes): %w", readErr)}
			}

			line = append(line, rest...)
		}

		// Обрезаем только крайние символы
		line = bytes.TrimSuffix(line, []byte("\n"))
		line = bytes.TrimSuffix(line, []byte("\r"))

		if len(line) > 0 {
			linesArena = append(linesArena, line...)
			// Меняем указатель
			line = linesArena[arenaOffset : arenaOffset+len(line)]
			arenaOffset += len(line)

			nthF, parseErr := nthField(line, cmpFieldNum)
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
	}
	return parseStep{processingFields, readLines}, nil
}

var errFieldNotFound error = errors.New("field not found")

// nthField разбивает строку на поля по разделителю ',', возвращает n поле
func nthField(line []byte, n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("field num must be >= 0, got %v", n)
	}
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
	// NOTE: если использовать поля для хранения arena offset (см. parseCSVTable),
	// то снизится нагрузка на gc, но нужно контролировать время жизни arena
}

// bytesToString — zero-copy []byte → string для read-only операций.
// Результат НЕ должен переживать b; используется только для
// strconv.Atoi, который читает и не сохраняет строку
func bytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func newFieldEntry(line []byte, field []byte) (fieldEntry, error) {
	v, err := strconv.Atoi(bytesToString(field))
	if err != nil {
		fmt.Println(field)
		return fieldEntry{}, err
	}
	return fieldEntry{line, v}, nil
}
