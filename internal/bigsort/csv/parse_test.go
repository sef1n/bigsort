package csv

import (
	"bufio"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"
	"testing"
)

type parseTest struct {
	name    string
	cmpF    int
	maxP    int
	input   string
	want    parseStep
	wantErr func(error) bool
}

func TestParseCSVTable(t *testing.T) {
	atoi := func(err error) bool {
		var ne *strconv.NumError
		return errors.As(err, &ne)
	}
	nilErr := func(err error) bool { return err == nil }
	eof := func(err error) bool { return errors.Is(err, io.EOF) }
	fieldNotFound := func(err error) bool { return errors.Is(err, ErrFieldNotFound) }
	isParseError := func(wantLine int) func(error) bool {
		return func(err error) bool {
			var pe *parseError
			if !errors.As(err, &pe) {
				return false
			}
			return pe.Line == wantLine
		}
	}

	tests := []parseTest{
		// --- Пусто / только переводы строк → io.EOF ---
		{"empty", 0, 1, "", parseStep{[]FieldEntry{}, 1}, eof},
		{"only newlines", 0, 1, "\n\n", parseStep{[]FieldEntry{}, 3}, eof},

		// --- Невалидный maxP → *parseError (не io.EOF) ---
		{"maxP=0", 0, 0, "1,a\n", parseStep{}, isParseError(0)},
		{"maxP=-1", 0, -1, "1,a\n", parseStep{}, isParseError(0)},

		// --- Ошибки парсинга → parseStep{} (zero) ---
		{"only spaces", 0, 1, "  ", parseStep{}, atoi},
		{"only comma", 0, 1, ",", parseStep{}, atoi},
		{"non-numeric", 0, 1, "abc,a\n", parseStep{}, atoi},
		{"cmpF beyond", 2, 1, "1,a\n", parseStep{}, fieldNotFound},
		{"whitespace line errors", 0, 1, "   \n", parseStep{}, atoi},

		// --- Ровно maxP БАЙТ → nil (без EOF) ---
		// "1,a"=3 байта, maxP=3: после первой строки parsedBytes=3>=3 → выход
		{
			"one line exactly max", 0, 3, "1,a\n",
			parseStep{[]FieldEntry{{[]byte("1,a"), 1}}, 1},
			nilErr,
		},
		// "1,a"+"2,b"=6 байт, maxP=6
		{
			"two lines exactly max", 0, 6, "1,a\n2,b\n",
			parseStep{[]FieldEntry{{[]byte("1,a"), 1}, {[]byte("2,b"), 2}}, 2},
			nilErr,
		},

		// --- Мягкий лимит: одна строка перескакивает границу ---
		// maxP=2, первая строка 3 байта → 1 entry, nilErr
		{
			"soft limit", 0, 2, "1,a\n2,b\n",
			parseStep{[]FieldEntry{{[]byte("1,a"), 1}}, 1},
			nilErr,
		},

		// --- Файл кончился → io.EOF с данными (+ пустая EOF-итерация) ---
		{
			"one line with EOF", 0, 100, "1,a\n",
			parseStep{[]FieldEntry{{[]byte("1,a"), 1}}, 2},
			eof,
		},
		{
			"two lines with EOF", 0, 100, "1,a\n2,b\n",
			parseStep{[]FieldEntry{{[]byte("1,a"), 1}, {[]byte("2,b"), 2}}, 3},
			eof,
		},
		{
			"three lines with EOF", 0, 100, "1,a\n2,b\n3,c\n",
			parseStep{[]FieldEntry{{[]byte("1,a"), 1}, {[]byte("2,b"), 2}, {[]byte("3,c"), 3}}, 4},
			eof,
		},
		{
			"cmpF=1", 1, 100, "a,10\nb,20\n",
			parseStep{[]FieldEntry{{[]byte("a,10"), 10}, {[]byte("b,20"), 20}}, 3},
			eof,
		},
		{
			"crlf", 0, 100, "1,a\r\n2,b\r\n",
			parseStep{[]FieldEntry{{[]byte("1,a"), 1}, {[]byte("2,b"), 2}}, 3},
			eof,
		},
		{
			"empty middle line", 0, 100, "1,a\n\n2,b\n",
			parseStep{[]FieldEntry{{[]byte("1,a"), 1}, {[]byte("2,b"), 2}}, 4},
			eof,
		},
		{
			"no trailing nl", 0, 100, "1,a\n2,b",
			parseStep{[]FieldEntry{{[]byte("1,a"), 1}, {[]byte("2,b"), 2}}, 2},
			eof,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseCSVTable(bufio.NewReader(strings.NewReader(tc.input)), tc.cmpF, tc.maxP)

			if !tc.wantErr(err) {
				t.Fatalf("err: got %v, check failed", err)
			}
			// При ошибке (не EOF) step должен быть zero value.
			if err != nil && !errors.Is(err, io.EOF) {
				if got.Entries != nil || got.readlines != 0 {
					t.Fatalf("step on error: got %+v, want zero", got)
				}
				return
			}

			// entries: сначала длина
			if len(got.Entries) != len(tc.want.Entries) {
				t.Fatalf("entries len: got %d, want %d", len(got.Entries), len(tc.want.Entries))
			}

			// entries: поэлементно, с превью для line
			for i := range got.Entries {
				g, w := got.Entries[i], tc.want.Entries[i]

				if g.Field != w.Field {
					t.Errorf("entries[%d].field: got %d, want %d", i, g.Field, w.Field)
				}
				if !slices.Equal(g.Line, w.Line) {
					t.Errorf("entries[%d].line: got %d bytes %q, want %d bytes %q",
						i, len(g.Line), truncate(g.Line, 30),
						len(w.Line), truncate(w.Line, 30))
				}
			}

			// readlines / offset
			if got.readlines != tc.want.readlines {
				t.Errorf("readlines: got %d, want %d", got.readlines, tc.want.readlines)
			}
		})
	}
}

func TestParseCSVTable_FallbackLongLine(t *testing.T) {
	input := strings.Repeat("a", 5000) + ",1\n" + "b,2\n" // > 4096 buffer

	got, err := ParseCSVTable(bufio.NewReader(strings.NewReader(input)), 1, 100_000)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("err: %v", err)
	}
	if len(got.Entries) != 2 {
		t.Fatalf("entries: got %d, want 2", len(got.Entries))
	}
	if got.Entries[0].Field != 1 {
		t.Errorf("entries[0].field: got %d, want 1", got.Entries[0].Field)
	}
	if got.Entries[1].Field != 2 {
		t.Errorf("entries[1].field: got %d, want 2", got.Entries[1].Field)
	}
}

// csv/parse_test.go
func TestNthField_Negative(t *testing.T) {
	_, err := NthField([]byte("1,a"), -1)
	if err == nil {
		t.Fatal("want error for negative n")
	}
}

// truncate возвращает превью []byte для лога: не более n байт.
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
