package bigsort

import (
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"
	"testing"
)

type fieldReaderTest struct {
	name        string
	input       string
	cmpFieldNum int
	wantLine    []byte
	wantField   int // значение cmpField при err == nil
	checkErr    func(error) bool
}

func TestFieldReader(t *testing.T) {
	incorSeps := func(err error) bool { return errors.Is(err, errFieldNotFound) }
	eof := func(err error) bool { return errors.Is(err, io.EOF) }
	nilErr := func(err error) bool { return err == nil }
	atoi := func(err error) bool {
		var numErr *strconv.NumError
		return errors.As(err, &numErr) && numErr.Func == "Atoi"
	}

	tests := []fieldReaderTest{
		// Пустой ввод → EOF, line пустой
		{"empty", "", 0, nil, 0, eof},

		// Строки без запятой при cmpFieldNum=0 — это валидная строка,
		// т.к. nthField("1", 0) возвращает "1" без ошибки
		{"no comma", "1", 0, []byte("1"), 1, nilErr},
		{"no comma nl", "1\n", 0, []byte("1"), 1, nilErr},
		{"no comma crlf", "1\r\n", 0, []byte("1"), 1, nilErr},
		{"no comma double nl", "1\n\n", 0, []byte("1"), 1, nilErr},

		// Первая строка пустая после TrimRight → Atoi("") ошибка
		{"leading empty line", "\n1\n", 0, nil, 0, atoi},

		// Atoi-ошибки
		{"only comma", ",", 0, []byte(","), 0, atoi},
		{"empty first field", ",a\n", 0, []byte(",a"), 0, atoi},
		{"non-numeric", "abc,a\n", 0, []byte("abc,a"), 0, atoi},
		{"float", "1.5,a\n", 0, []byte("1.5,a"), 0, atoi},
		{"leading space", " 1,a\n", 0, []byte(" 1,a"), 0, atoi},
		{"trailing space", "1 ,a\n", 0, []byte("1 ,a"), 0, atoi},
		{"overflow int", "99999999999999999999,a\n", 0, []byte("99999999999999999999,a"), 0, atoi},

		// Валидные (cmpFieldNum=0)
		{"valid", "1,a\n", 0, []byte("1,a"), 1, nilErr},
		{"valid negative", "-1,a\n", 0, []byte("-1,a"), -1, nilErr},
		{"valid plus", "+1,a\n", 0, []byte("+1,a"), 1, nilErr},
		{"valid zero", "0,a\n", 0, []byte("0,a"), 0, nilErr},
		{"valid empty rest", "1,\n", 0, []byte("1,"), 1, nilErr},

		// cmpFieldNum=1 (поле 1) — вот главное: значение должно быть из поля 1
		{"cmp=1 valid", "a,10\n", 1, []byte("a,10"), 10, nilErr},
		{"cmp=1 negative", "a,-5\n", 1, []byte("a,-5"), -5, nilErr},
		{"cmp=1 zero", "a,0\n", 1, []byte("a,0"), 0, nilErr},
		{"cmp=1 atoi error", "a,xyz\n", 1, []byte("a,xyz"), 0, atoi},
		{"cmp=1 no comma", "abc\n", 1, []byte("abc"), 0, incorSeps},

		// cmpFieldNum за пределами строки
		{"cmp=2 beyond", "1,a\n", 2, []byte("1,a"), 0, incorSeps},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newFieldReader(strings.NewReader(tc.input), tc.cmpFieldNum)
			if !slices.Equal(r.line, tc.wantLine) {
				t.Fatalf("line: got %q, want %q", r.line, tc.wantLine)
			}
			if !tc.checkErr(r.err) {
				t.Fatalf("err: got %v, check failed", r.err)
			}
			if r.err == nil && r.cmpField != tc.wantField {
				t.Errorf("cmpField: got %d, want %d", r.cmpField, tc.wantField)
			}
		})
	}
}
