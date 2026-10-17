package bigsort

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type FieldReader struct {
	cmpField int // Поле, по которому выполняется сортировка
	line     string
	reader   *bufio.Scanner
	err      error // Ошибка чтения или io.EOF

	// Возможно имеет смысл
	// file     io.Reader
}

// NewFieldReader выполняет инициализацию FieldReader, в случае ошибки устанавливает её в FieldReader.err.
func NewFieldReader(f io.Reader) *FieldReader {
	scn := bufio.NewScanner(f)
	scn.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	fr := &FieldReader{reader: scn}
	// Scan сам устанавливает ошибки (io.EOF считается ошибкой)
	_ = fr.Scan()
	return fr
}

// Scan выполняет Scanner.Scan, устанавливает новые line и err.
// По окончании файла устанавливает err = io.EOF
func (fr *FieldReader) Scan() bool {
	if !fr.reader.Scan() {
		fr.err = fr.reader.Err()
		if fr.err == nil {
			fr.err = io.EOF
		}
		return false
	}
	fr.line = fr.reader.Text()

	idx := strings.IndexByte(fr.line, ',')
	if idx < 0 {
		fr.err = fmt.Errorf("no comma in line: %q", fr.line)
		return false
	}
	v, err := strconv.Atoi(fr.line[:idx])
	if err != nil {
		fr.err = fmt.Errorf("parse %q: %w", fr.line[:idx], err)
		return false
	}
	fr.cmpField = v
	return true
}
