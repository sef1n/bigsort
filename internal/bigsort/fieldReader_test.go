package bigsort

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Ошибки (общие для всех тестов)

func isEOF(err error) bool { return errors.Is(err, io.EOF) }
func isNil(err error) bool { return err == nil }
func isAtoiErr(err error) bool {
	var ne *strconv.NumError
	return errors.As(err, &ne) && ne.Func == "Atoi"
}
func isFieldNotFound(err error) bool { return errors.Is(err, errFieldNotFound) }

// TestNewFieldReader — инициализация: pre-read первой строки.
// После newFieldReader:
//   - r.line содержит первую строку (без \r\n)
//   - r.cmpField содержит распарсенное значение поля
//   - err == nil, если первая строка валидна
//   - err == io.EOF, если файл пуст (r.line == "")
//   - err != nil (не io.EOF), если первая строка не парсится
func TestNewFieldReader(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		cmpFieldNum int
		wantLine    []byte
		wantField   int
		checkErr    func(error) bool
	}{
		// Пустой ввод → EOF, line пустой
		{"empty", "", 0, nil, 0, isEOF},

		// Одна строка без \n — read() возвращает nil (данные есть),
		// следующий read() вернёт io.EOF (см. TestFieldReaderRead)
		{"single no newline", "1", 0, []byte("1"), 1, isNil},
		{"single with nl", "1\n", 0, []byte("1"), 1, isNil},
		{"single with crlf", "1\r\n", 0, []byte("1"), 1, isNil},

		// Первая строка пустая после TrimRight → Atoi("") ошибка
		{"leading empty line", "\n1\n", 0, []byte(""), 0, isAtoiErr},

		// Atoi-ошибки
		{"only comma", ",", 0, []byte(","), 0, isAtoiErr},
		{"empty first field", ",a\n", 0, []byte(",a"), 0, isAtoiErr},
		{"non-numeric", "abc,a\n", 0, []byte("abc,a"), 0, isAtoiErr},
		{"float", "1.5,a\n", 0, []byte("1.5,a"), 0, isAtoiErr},
		{"leading space", " 1,a\n", 0, []byte(" 1,a"), 0, isAtoiErr},
		{"trailing space", "1 ,a\n", 0, []byte("1 ,a"), 0, isAtoiErr},
		{"overflow int", "99999999999999999999,a\n", 0, []byte("99999999999999999999,a"), 0, isAtoiErr},

		// Валидные (cmpFieldNum=0)
		{"valid", "1,a\n", 0, []byte("1,a"), 1, isNil},
		{"valid negative", "-1,a\n", 0, []byte("-1,a"), -1, isNil},
		{"valid plus", "+1,a\n", 0, []byte("+1,a"), 1, isNil},
		{"valid zero", "0,a\n", 0, []byte("0,a"), 0, isNil},
		{"valid empty rest", "1,\n", 0, []byte("1,"), 1, isNil},

		// cmpFieldNum=1
		{"cmp=1 valid", "a,10\n", 1, []byte("a,10"), 10, isNil},
		{"cmp=1 negative", "a,-5\n", 1, []byte("a,-5"), -5, isNil},
		{"cmp=1 zero", "a,0\n", 1, []byte("a,0"), 0, isNil},
		{"cmp=1 atoi error", "a,xyz\n", 1, []byte("a,xyz"), 0, isAtoiErr},
		{"cmp=1 no comma", "abc\n", 1, []byte("abc"), 0, isFieldNotFound},

		// cmpFieldNum за пределами
		{"cmp=2 beyond", "1,a\n", 2, []byte("1,a"), 0, isFieldNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := newFieldReader(strings.NewReader(tc.input), tc.cmpFieldNum)
			if r == nil {
				t.Fatal("newFieldReader returned nil")
			}
			if !tc.checkErr(err) {
				t.Fatalf("err: got %v, check failed", err)
			}
			if !slices.Equal(r.line, tc.wantLine) {
				t.Fatalf("line: got %q, want %q", r.line, tc.wantLine)
			}
			// cmpField валиден только при успешном init
			if err == nil && r.cmpField != tc.wantField {
				t.Errorf("cmpField: got %d, want %d", r.cmpField, tc.wantField)
			}
		})
	}
}

// TestFieldReaderRead — многократные read() до EOF.
// После init первая строка уже прочитана; проверяем, что
// каждая последующая read() отдаёт следующую строку, а в конце — io.EOF.
func TestFieldReaderRead(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		cmpFieldNum int
		wantLines   [][]byte
		wantFields  []int
	}{
		{
			name:        "two lines",
			input:       "1,a\n2,b\n",
			cmpFieldNum: 0,
			wantLines:   [][]byte{[]byte("1,a"), []byte("2,b")},
			wantFields:  []int{1, 2},
		},
		{
			name:        "three lines",
			input:       "1,a\n2,b\n3,c\n",
			cmpFieldNum: 0,
			wantLines:   [][]byte{[]byte("1,a"), []byte("2,b"), []byte("3,c")},
			wantFields:  []int{1, 2, 3},
		},
		{
			name:        "last line no newline",
			input:       "1,a\n2,b",
			cmpFieldNum: 0,
			wantLines:   [][]byte{[]byte("1,a"), []byte("2,b")},
			wantFields:  []int{1, 2},
		},
		{
			name:        "single line no newline",
			input:       "1,a",
			cmpFieldNum: 0,
			wantLines:   [][]byte{[]byte("1,a")},
			wantFields:  []int{1},
		},
		{
			name:        "cmpField=1",
			input:       "a,10\nb,20\nc,30\n",
			cmpFieldNum: 1,
			wantLines:   [][]byte{[]byte("a,10"), []byte("b,20"), []byte("c,30")},
			wantFields:  []int{10, 20, 30},
		},
		{
			name:        "crlf lines",
			input:       "1,a\r\n2,b\r\n",
			cmpFieldNum: 0,
			wantLines:   [][]byte{[]byte("1,a"), []byte("2,b")},
			wantFields:  []int{1, 2},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := newFieldReader(strings.NewReader(tc.input), tc.cmpFieldNum)
			if err != nil && err != io.EOF {
				t.Fatalf("init: %v", err)
			}
			if err == io.EOF {
				t.Fatal("init returned io.EOF for non-empty input")
			}

			// Первая строка уже прочитана при init — копируем (r.line перезаписывается)
			var gotLines [][]byte
			var gotFields []int
			gotLines = append(gotLines, append([]byte(nil), r.line...))
			gotFields = append(gotFields, r.cmpField)

			for {
				err := r.read()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("read %d: %v", len(gotLines), err)
				}
				gotLines = append(gotLines, append([]byte(nil), r.line...))
				gotFields = append(gotFields, r.cmpField)
			}

			if len(gotLines) != len(tc.wantLines) {
				t.Fatalf("lines count: got %d, want %d (got=%q)", len(gotLines), len(tc.wantLines), gotLines)
			}
			for i := range tc.wantLines {
				if !slices.Equal(gotLines[i], tc.wantLines[i]) {
					t.Errorf("line[%d]: got %q, want %q", i, gotLines[i], tc.wantLines[i])
				}
				if gotFields[i] != tc.wantFields[i] {
					t.Errorf("field[%d]: got %d, want %d", i, gotFields[i], tc.wantFields[i])
				}
			}
		})
	}
}

// TestFieldReaderRead_Errors — ошибки при read() (2-я+ строка).
// Проверяем, что ошибка возвращается на конкретной итерации,
// и r.line содержит проблемную строку.
func TestFieldReaderRead_Errors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		cmpFieldNum int
		// wantErrAt — на какой итерации ошибка (0 = init; 1 = первый read)
		wantErrAt int
		wantLine  []byte
		checkErr  func(error) bool
	}{
		{
			name:        "atoi error on second",
			input:       "1,a\nxyz,c\n3,e\n",
			cmpFieldNum: 0,
			wantErrAt:   1,
			wantLine:    []byte("xyz,c"),
			checkErr:    isAtoiErr,
		},
		{
			name:        "atoi error on third",
			input:       "1,a\n2,b\nabc\n",
			cmpFieldNum: 0,
			wantErrAt:   2,
			wantLine:    []byte("abc"),
			checkErr:    isAtoiErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := newFieldReader(strings.NewReader(tc.input), tc.cmpFieldNum)
			if err != nil && err != io.EOF {
				t.Fatalf("init: %v", err)
			}

			// Итерация 0 — init, итерация N — N-й read()
			for i := 0; ; i++ {
				if i == tc.wantErrAt {
					if !tc.checkErr(err) {
						t.Fatalf("iter %d: err got %v, check failed", i, err)
					}
					if !slices.Equal(r.line, tc.wantLine) {
						t.Errorf("iter %d: line got %q, want %q", i, r.line, tc.wantLine)
					}
					return
				}
				if err != nil {
					t.Fatalf("iter %d: unexpected err before wantErrAt: %v", i, err)
				}
				if err = r.read(); err == io.EOF {
					t.Fatalf("iter %d: got EOF before wantErrAt", i)
				}
			}
		})
	}
}

// Guard against unused import if test shrinks
var _ = bytes.Equal
