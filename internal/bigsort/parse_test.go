package bigsort

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

type parseTest struct {
	name    string
	cmpF    int
	input   string
	want    []fieldEntry
	wantErr func(error) bool
}

func TestParseCSVTable(t *testing.T) {
	atoi := func(err error) bool {
		var ne *strconv.NumError
		return errors.As(err, &ne)
	}
	nilErr := func(err error) bool {
		return err == nil
	}
	fieldNotFound := func(err error) bool {
		return errors.Is(err, errFieldNotFound)
	}
	tests := []parseTest{
		// Пусто
		{"empty", 0, "", []fieldEntry{}, nilErr},
		{"only newlines", 0, "\n\n", []fieldEntry{}, nilErr},
		{"only spaces", 0, "  ", []fieldEntry{}, atoi},

		// Ошибки парсинга
		{"only comma", 0, ",", nil, atoi},
		{"non-numeric", 0, "abc,a\n", nil, atoi},
		{"cmpF beyond", 2, "1,a\n", nil, fieldNotFound},
		{"whitespace line errors", 0, "   \n", nil, atoi},

		// Валидные
		{"one line", 0, "1,a\n", []fieldEntry{{0, 1}}, nilErr},
		{"two lines", 0, "1,a\n2,b\n", []fieldEntry{{0, 1}, {4, 2}}, nilErr},
		{"three lines", 0, "1,a\n2,b\n3,c\n", []fieldEntry{{0, 1}, {4, 2}, {8, 3}}, nilErr},
		{"cmpF=1", 1, "a,10\nb,20\n", []fieldEntry{{0, 10}, {5, 20}}, nilErr},
		{"crlf", 0, "1,a\r\n2,b\r\n", []fieldEntry{{0, 1}, {5, 2}}, nilErr},
		{"empty middle line", 0, "1,a\n\n2,b\n", []fieldEntry{{0, 1}, {5, 2}}, nilErr},
		{"no trailing nl", 0, "1,a\n2,b", []fieldEntry{{0, 1}, {4, 2}}, nilErr},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseCSVTable(strings.NewReader(tc.input), tc.cmpF)

			if !tc.wantErr(err) {
				t.Fatalf("err: got %v, check failed", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("len: got %d, want %d (got=%v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d]: got %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}
