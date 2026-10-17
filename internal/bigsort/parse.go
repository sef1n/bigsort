package bigsort

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
)

const (
	predictTableLines  = 100
	tmpFileFieldsCount = 1000
)

// getFields читает строки из файла, сохраняет Nth поля, возвращает структуру сравниваемых полей
func parseCSVTable(tablePath string, cmpFieldNum int) ([]fieldEntry, error) {
	tableFile, err := os.Open(tablePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tableFile.Close() }()

	processingFields := make([]fieldEntry, 0, predictTableLines)
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
