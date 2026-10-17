package bigsort

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

const (
	predictTableLines  = 100
	tmpFileFieldsCount = 1000
)

// parseCSVTable читает csv из r, возвращает записи с байтовыми offset'ами.
//
// Пустые строки (после отсечения \r\n) пропускаются и в результат
// не попадают; offset при этом корректно сдвигается. Строки из
// одних пробелов НЕ пропускаются и приводят к ошибке парсинга
func parseCSVTable(r io.Reader, cmpFieldNum int) ([]fieldEntry, error) {
	processingFields := make([]fieldEntry, 0, predictTableLines)
	var offset int64
	br := bufio.NewReader(r)

	for lineNum := 0; ; lineNum++ {
		line, readErr := br.ReadBytes('\n')
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

			fi, parseErr := newFieldEntry(offset, string(nthF), cmpFieldNum)
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

// TODO: Любой Entry?
type fieldEntry struct {
	Offset int64
	Entry  int
}

func newFieldEntry(off int64, l string, i int) (fieldEntry, error) {
	v, err := strconv.Atoi(l)
	if err != nil {
		return fieldEntry{}, err
	}
	return fieldEntry{off, v}, nil
}
