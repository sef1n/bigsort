package bigsort

import (
	"strings"
	"testing"
)

// Проверять отдельные ошибки слишком хрупко, достаточно сравнить наличие
type parseTest struct {
	name    string
	cmpF    int
	input   string
	want    []fieldEntry
	wantErr bool // true = любая ошибка
}

func TestParseCSVTable(t *testing.T) {
	tests := []parseTest{
		// Пусто.
		{"empty", 0, "", []fieldEntry{}, false},
		{"only newlines", 0, "\n\n", []fieldEntry{}, false},

		// Ошибки парсинга.
		{"only comma", 0, ",", nil, true},
		{"non-numeric", 0, "abc,a\n", nil, true},
		{"cmpF beyond", 2, "1,a\n", nil, true}, // третьего поля нет

		// Валидные.
		{"one line", 0, "1,a\n", []fieldEntry{{0, 1}}, false},
		{"two lines", 0, "1,a\n2,b\n", []fieldEntry{{0, 1}, {4, 2}}, false},
		{"three lines", 0, "1,a\n2,b\n3,c\n", []fieldEntry{{0, 1}, {4, 2}, {8, 3}}, false},
		{"cmpF=1", 1, "a,10\nb,20\n", []fieldEntry{{0, 10}, {5, 20}}, false},
		{"crlf", 0, "1,a\r\n2,b\r\n", []fieldEntry{{0, 1}, {5, 2}}, false},
		{"empty middle line", 0, "1,a\n\n2,b\n", []fieldEntry{{0, 1}, {5, 2}}, false},
		{"no trailing nl", 0, "1,a\n2,b", []fieldEntry{{0, 1}, {4, 2}}, false},
		{"empty line skipped", 0, "1,a\n\n2,b\n", []fieldEntry{{0, 1}, {5, 2}}, false},
		{"whitespace line errors", 0, "   \n", nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseCSVTable(strings.NewReader(tc.input), tc.cmpF)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got nil")
				}
				return // при ошибке entries не проверяем
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
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
