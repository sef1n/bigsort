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

type FieldReader struct {
	CmpFieldNum int // Номер поля (начиная с 0)
	CmpField    int // Поле, по которому выполняется сортировка
	Line        []byte
	Reader      *bufio.Reader

	// Возможно имеет смысл
	// file     io.Reader
}

// TODO: type fieldExtractor interface
// сейчас просто импорты csv

// NewFieldReader выполняет инициализацию FieldReader, в случае ошибки устанавливает её в FieldReader.err.
func NewFieldReader(f io.Reader, cmpFieldNum int) (*FieldReader, error) {
	r := bufio.NewReader(f)
	fr := &FieldReader{Reader: r, CmpFieldNum: cmpFieldNum, Line: make([]byte, 0, 20)}
	return fr, fr.Read()
}

// Read читает следующую строку в fr.line (ПЕРЕЗАПИСЫВАЯ его).
// Пустые строки (\n или \r\n без данных) НЕ пропускаются —
// caller должен гарантировать, что tmp-файлы их не содержат
// (это делает parseCSVTable). Иначе Atoi вернёт ошибку.
// После успеха fr.line не содержит \n/\r.
// Возвращает nil только с непустым fr.line.
func (fr *FieldReader) Read() error {
	fr.Line = fr.Line[:0]
	readLine, err := fr.Reader.ReadSlice('\n')
	fullLineLen := len(readLine)
	if err != nil && err != io.EOF {
		if !errors.Is(err, bufio.ErrBufferFull) {
			return err
		}
		fr.Line = append(fr.Line, readLine...)

		readLine, err = fr.Reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return err
		}
		fullLineLen += len(readLine)
	}

	// Не добавляем лишний раз
	if fullLineLen == 0 {
		return err // EOF без данных
	}

	fr.Line = append(fr.Line, readLine...)

	fr.Line = bytes.TrimRight(fr.Line, "\r\n")

	bytesField, err := csv.NthField(fr.Line, fr.CmpFieldNum)
	if err != nil {
		return fmt.Errorf("parse %q: %w", bytesField, err)
	}
	v, err := strconv.Atoi(csv.BytesToString(bytesField))
	if err != nil {
		return fmt.Errorf("parse %q: %w", bytesField, err)
	}
	fr.CmpField = v
	return nil
}
