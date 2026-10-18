package bigsort

import (
	"bufio"
	"errors"
	"io"
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
	fieldNotFound := func(err error) bool { return errors.Is(err, errFieldNotFound) }
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
		{"empty", 0, 1, "", parseStep{[]fieldEntry{}, 1, 0}, eof},
		{"only newlines", 0, 1, "\n\n", parseStep{[]fieldEntry{}, 3, 2}, eof},

		// --- Невалидный maxP → *parseError (не io.EOF), step zero, offset сохранён ---
		{"maxP=0", 0, 0, "1,a\n", parseStep{}, isParseError(0)},
		{"maxP=-1", 0, -1, "1,a\n", parseStep{}, isParseError(0)},

		// --- Ошибки парсинга → parseStep{} (zero) ---
		{"only spaces", 0, 1, "  ", parseStep{}, atoi},
		{"only comma", 0, 1, ",", parseStep{}, atoi},
		{"non-numeric", 0, 1, "abc,a\n", parseStep{}, atoi},
		{"cmpF beyond", 2, 1, "1,a\n", parseStep{}, fieldNotFound},
		{"whitespace line errors", 0, 1, "   \n", parseStep{}, atoi},

		// --- Ровно maxP → nil (без EOF) ---
		{"one line exactly max", 0, 1, "1,a\n", parseStep{[]fieldEntry{{0, 1}}, 1, 4}, nilErr},
		{"two lines exactly max", 0, 2, "1,a\n2,b\n", parseStep{[]fieldEntry{{0, 1}, {4, 2}}, 2, 8}, nilErr},

		// --- Файл кончился → io.EOF с данными (+ пустая EOF-итерация) ---
		{"one line with EOF", 0, 5, "1,a\n", parseStep{[]fieldEntry{{0, 1}}, 2, 4}, eof},
		{"two lines with EOF", 0, 5, "1,a\n2,b\n", parseStep{[]fieldEntry{{0, 1}, {4, 2}}, 3, 8}, eof},
		{"three lines with EOF", 0, 5, "1,a\n2,b\n3,c\n", parseStep{[]fieldEntry{{0, 1}, {4, 2}, {8, 3}}, 4, 12}, eof},
		{"cmpF=1", 1, 5, "a,10\nb,20\n", parseStep{[]fieldEntry{{0, 10}, {5, 20}}, 3, 10}, eof},
		{"crlf", 0, 5, "1,a\r\n2,b\r\n", parseStep{[]fieldEntry{{0, 1}, {5, 2}}, 3, 10}, eof},
		{"empty middle line", 0, 5, "1,a\n\n2,b\n", parseStep{[]fieldEntry{{0, 1}, {5, 2}}, 4, 9}, eof},
		{"no trailing nl", 0, 5, "1,a\n2,b", parseStep{[]fieldEntry{{0, 1}, {4, 2}}, 2, 7}, eof},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseCSVTable(bufio.NewReader(strings.NewReader(tc.input)), tc.cmpF, tc.maxP, 0)

			if !tc.wantErr(err) {
				t.Fatalf("err: got %v, check failed", err)
			}
			// При ошибке (не EOF) step должен быть zero value.
			if err != nil && !errors.Is(err, io.EOF) {
				if got.entries != nil || got.readlines != 0 || got.offset != 0 {
					t.Fatalf("step on error: got %+v, want zero", got)
				}
				return
			}
			// entries
			if len(got.entries) != len(tc.want.entries) {
				t.Fatalf("entries len: got %d, want %d (got=%v)", len(got.entries), len(tc.want.entries), got.entries)
			}
			for i := range got.entries {
				if got.entries[i] != tc.want.entries[i] {
					t.Errorf("entries[%d]: got %+v, want %+v", i, got.entries[i], tc.want.entries[i])
				}
			}
			// readlines / offset
			if got.readlines != tc.want.readlines {
				t.Errorf("readlines: got %d, want %d", got.readlines, tc.want.readlines)
			}
			if got.offset != tc.want.offset {
				t.Errorf("offset: got %d, want %d", got.offset, tc.want.offset)
			}
		})
	}
}
