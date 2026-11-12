package merge

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"

	"bigFiles/internal/bigsort/csv"
)

// WARN: obsolete doc

type fieldReader struct {
	cmpFieldNum int // Номер поля (начиная с 0)
	line        []byte
	reader      *bufio.Reader

	// Возможно имеет смысл
	// file     io.Reader
}

// TODO: type fieldExtractor interface
// сейчас просто импорты csv

// newFieldReader выполняет инициализацию FieldReader, в случае ошибки устанавливает её в FieldReader.err.
func newFieldReader(f io.Reader, cmpFieldNum int) (*fieldReader, int, error) {
	r := bufio.NewReader(f)
	fr := &fieldReader{reader: r, cmpFieldNum: cmpFieldNum, line: make([]byte, 0, 20)}
	v, err := fr.Read()
	return fr, v, err
}

// Read читает следующую строку в fr.line (ПЕРЕЗАПИСЫВАЯ его).
// Пустые строки (\n или \r\n без данных) НЕ пропускаются —
// caller должен гарантировать, что tmp-файлы их не содержат
// (это делает parseCSVTable). Иначе Atoi вернёт ошибку.
// После успеха fr.line не содержит \n/\r.
// Возвращает nil только с непустым fr.line.
func (fr *fieldReader) Read() (int, error) {
	fr.line = fr.line[:0]
	readLine, err := fr.reader.ReadSlice('\n')
	fullLineLen := len(readLine)
	if err != nil && err != io.EOF {
		if !errors.Is(err, bufio.ErrBufferFull) {
			return 0, err
		}
		fr.line = append(fr.line, readLine...)

		readLine, err = fr.reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return 0, err
		}
		fullLineLen += len(readLine)
	}

	// Не добавляем лишний раз
	if fullLineLen == 0 {
		return 0, err // EOF без данных
	}

	fr.line = append(fr.line, readLine...)
	fr.line = bytes.TrimRight(fr.line, "\r\n")

	bytesField, err := csv.NthField(fr.line, fr.cmpFieldNum)
	if err != nil {
		return 0, fmt.Errorf("parse %q: %w", bytesField, err)
	}
	v, err := strconv.Atoi(csv.BytesToString(bytesField))
	if err != nil {
		return 0, fmt.Errorf("parse %q: %w", bytesField, err)
	}
	return v, nil
}
