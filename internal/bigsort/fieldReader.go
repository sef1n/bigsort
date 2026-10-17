package bigsort

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var errNoComma = errors.New("no comma in line")

type fieldReader struct {
	cmpField int // Поле, по которому выполняется сортировка
	line     string
	reader   *bufio.Scanner
	err      error // Ошибка чтения или io.EOF

	// Возможно имеет смысл
	// file     io.Reader
}

// newFieldReader выполняет инициализацию FieldReader, в случае ошибки устанавливает её в FieldReader.err.
func newFieldReader(f io.Reader) *fieldReader {
	scn := bufio.NewScanner(f)
	scn.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	fr := &fieldReader{reader: scn}
	// Scan сам устанавливает ошибки (io.EOF считается ошибкой)
	_ = fr.scan()
	return fr
}

// scan выполняет Scanner.scan, устанавливает новые line и err.
// По окончании файла устанавливает err = io.EOF
func (fr *fieldReader) scan() bool {
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
		fr.err = fmt.Errorf("%w: %q", errNoComma, fr.line)
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
