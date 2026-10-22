package bigsort

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

var errNoComma = errors.New("no comma in line")

type fieldReader struct {
	cmpField int // Поле, по которому выполняется сортировка
	line     []byte
	reader   *bufio.Reader
	err      error // Ошибка чтения или io.EOF

	// Возможно имеет смысл
	// file     io.Reader
}

// newFieldReader выполняет инициализацию FieldReader, в случае ошибки устанавливает её в FieldReader.err.
func newFieldReader(f io.Reader) *fieldReader {
	scn := bufio.NewReader(f)
	fr := &fieldReader{reader: scn}
	// Scan сам устанавливает ошибки (io.EOF считается ошибкой)
	_ = fr.scan()
	return fr
}

// scan выполняет Scanner.scan, устанавливает новые line и err.
// По окончании файла устанавливает err = io.EOF
func (fr *fieldReader) scan() bool {
	fr.line, fr.err = fr.reader.ReadBytes('\n')
	if fr.err != nil && !errors.Is(fr.err, io.EOF) {
		return false
	}
	if len(fr.line) == 0 {
		return false // EOF без данных
	}
	// Получена валидная строка -> ошибки нет
	fr.err = nil

	fr.line = bytes.TrimRight(fr.line, "\r\n")

	// FIX: неправильный поиск поля
	idx := bytes.IndexByte(fr.line, ',')
	if idx < 0 {
		fr.err = fmt.Errorf("%w: %q", errNoComma, fr.line)
		return false
	}
	v, err := strconv.Atoi(string(fr.line[:idx]))
	if err != nil {
		fr.err = fmt.Errorf("parse %q: %w", fr.line[:idx], err)
		return false
	}
	fr.cmpField = v
	return true
}
