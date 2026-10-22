package bigsort

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

type fieldReader struct {
	cmpFieldNum int // Номер поля (начиная с 0)
	cmpField    int // Поле, по которому выполняется сортировка
	line        []byte
	reader      *bufio.Reader
	err         error // Ошибка чтения или io.EOF

	// Возможно имеет смысл
	// file     io.Reader
}

// newFieldReader выполняет инициализацию FieldReader, в случае ошибки устанавливает её в FieldReader.err.
func newFieldReader(f io.Reader, cmpFieldNum int) *fieldReader {
	scn := bufio.NewReader(f)
	fr := &fieldReader{reader: scn, cmpFieldNum: cmpFieldNum}
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

	bytesField, err := nthField(fr.line, fr.cmpFieldNum)
	if err != nil {
		fr.err = fmt.Errorf("parse %q: %w", bytesField, err)
		return false
	}
	v, err := strconv.Atoi(string(bytesField))
	if err != nil {
		fr.err = fmt.Errorf("parse %q: %w", bytesField, err)
		return false
	}
	fr.cmpField = v
	return true
}
