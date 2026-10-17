package bigsort

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
)

type fieldReaderTest struct {
	name     string
	input    string
	wantLine string
	checkErr func(error) bool
}

func TestFieldReader(t *testing.T) {
	// noComma проверяет точный текст ошибки "no comma in line: %q".
	noComma := func(s string) func(error) bool {
		want := fmt.Sprintf("no comma in line: %q", s)
		return func(err error) bool { return err != nil && err.Error() == want }
	}
	// eof: io.EOF (sentinel).
	eof := func(err error) bool { return errors.Is(err, io.EOF) }
	// nilErr: ошибки нет.
	nilErr := func(err error) bool { return err == nil }
	// atoi: любая ошибка strconv.Atoi (syntax или range).
	atoi := func(err error) bool {
		var numErr *strconv.NumError
		return errors.As(err, &numErr) && numErr.Func == "Atoi"
	}

	tests := []fieldReaderTest{
		// Пустой ввод и строки без запятой.
		{"empty", "", "", eof},
		{"no comma", "1", "1", noComma("1")},
		{"no comma nl", "1\n", "1", noComma("1")},
		{"no comma crlf", "1\r\n", "1", noComma("1")},
		{"no comma double nl", "1\n\n", "1", noComma("1")},
		{"leading empty line", "\n1\n", "", noComma("")},

		// Atoi-ошибки: fr.line уже установлен (до парсинга), поэтому в wantLine — вся строка.
		{"only comma", ",", ",", atoi},
		{"empty first field", ",a\n", ",a", atoi},
		{"non-numeric", "abc,a\n", "abc,a", atoi},
		{"float", "1.5,a\n", "1.5,a", atoi},
		{"leading space", " 1,a\n", " 1,a", atoi},
		{"trailing space", "1 ,a\n", "1 ,a", atoi},
		{"overflow int", "99999999999999999999,a\n", "99999999999999999999,a", atoi},

		// Валидные.
		{"valid", "1,a\n", "1,a", nilErr},
		{"valid negative", "-1,a\n", "-1,a", nilErr},
		{"valid plus", "+1,a\n", "+1,a", nilErr},
		{"valid zero", "0,a\n", "0,a", nilErr},
		{"valid empty rest", "1,\n", "1,", nilErr},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newFieldReader(strings.NewReader(tc.input))
			if r.line != tc.wantLine {
				t.Fatalf("line: got %q, want %q", r.line, tc.wantLine)
			}
			if !tc.checkErr(r.err) {
				t.Fatalf("err: got %v, check failed", r.err)
			}
		})
	}
}
