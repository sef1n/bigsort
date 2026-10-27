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

	// Возможно имеет смысл
	// file     io.Reader
}

// newFieldReader выполняет инициализацию FieldReader, в случае ошибки устанавливает её в FieldReader.err.
func newFieldReader(f io.Reader, cmpFieldNum int) (*fieldReader, error) {
	r := bufio.NewReader(f)
	fr := &fieldReader{reader: r, cmpFieldNum: cmpFieldNum, line: make([]byte, 0, 20)}
	return fr, fr.read()
}

// read читает следующую строку в fr.line (ПЕРЕЗАПИСЫВАЯ его).
// Пустые строки (\n или \r\n без данных) НЕ пропускаются —
// caller должен гарантировать, что tmp-файлы их не содержат
// (это делает parseCSVTable). Иначе Atoi вернёт ошибку.
// После успеха fr.line не содержит \n/\r.
// Возвращает nil только с непустым fr.line.
func (fr *fieldReader) read() error {
	fr.line = fr.line[:0]
	readLine, err := fr.reader.ReadSlice('\n')
	fullLineLen := len(readLine)
	if err != nil && err != io.EOF {
		if !errors.Is(err, bufio.ErrBufferFull) {
			return err
		}
		fr.line = append(fr.line, readLine...)

		readLine, err = fr.reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return err
		}
		fullLineLen += len(readLine)
	}

	// Не добавляем лишний раз
	if fullLineLen == 0 {
		return err // EOF без данных
	}

	fr.line = append(fr.line, readLine...)

	fr.line = bytes.TrimRight(fr.line, "\r\n")

	bytesField, err := nthField(fr.line, fr.cmpFieldNum)
	if err != nil {
		return fmt.Errorf("parse %q: %w", bytesField, err)
	}
	v, err := strconv.Atoi(bytesToString(bytesField))
	if err != nil {
		return fmt.Errorf("parse %q: %w", bytesField, err)
	}
	fr.cmpField = v
	return nil
}
