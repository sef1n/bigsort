package bigsort

import (
	"bytes"
	"context"
	"io"
	"slices"
	"testing"

	"github.com/spf13/afero"
)

type test struct {
	name     string
	cmpField int
	input    []byte
	chunk    int
	// Тест проверяет корректность логики парсинга
	// У входящих в SplitOnSortedChunks есть свои тесты, поэтому
	// Достаточно факта наличия ошибки
	expErr bool
}

// WARN: этот тест НЕ поддерживает тестирование сплита таблиц,
// у которых чанки начинаются на ОДИНАКОВЫЕ строки:
// порядок создания временных файлов недетерминирован,
// чтобы протестировать разбиение на чанки, необходимо реконструировать
// порядок файлов - данный тест для упрорядочивания ориентируется на
// совпадение первых строк на предположительном месте начала чанка.
func TestSplitOnSortedChunks(T *testing.T) {
	tests := []test{
		// ---- без ошибок ----
		{
			name:     "empty",
			cmpField: 0,
			input:    []byte{},
			chunk:    100,
			expErr:   false,
		},
		{
			name:     "single line fits",
			cmpField: 0,
			input:    []byte("1,a\n"),
			chunk:    100,
			expErr:   false,
		},
		{
			name:     "multiple lines one chunk",
			cmpField: 0,
			input:    []byte("1,a\n2,b\n3,c\n"),
			chunk:    100,
			expErr:   false,
		},
		{
			name:     "exact chunk boundary",
			cmpField: 0,
			input:    []byte("12,ab\n34,cd\n"), // 5 + 5 = 10
			chunk:    10,
			expErr:   false,
		},
		{
			name:     "two chunks overshoot",
			cmpField: 0,
			input:    []byte("1,a\n2,b\n3,c\n4,d\n"), // каждый 3 байта
			chunk:    5,                              // чанки по 6 байт (2 строки)
			expErr:   false,
		},
		{
			name:     "last chunk smaller",
			cmpField: 0,
			input:    []byte("1,aaaa\n2,bb\n"), // 6 + 4
			chunk:    8,
			expErr:   false,
		},
		{
			name:     "cmpField=1",
			cmpField: 1,
			input:    []byte("1,10\n2,20\n3,30\n"),
			chunk:    100,
			expErr:   false,
		},
		// ---- с ошибками ----
		{
			name:     "zero chunk",
			cmpField: 0,
			input:    []byte("1,10\n"),
			chunk:    0,
			expErr:   true, // parseError: maxParsedBytes must be > 0
		},
		{
			name:     "negative chunk",
			cmpField: 0,
			input:    []byte("1,10\n"),
			chunk:    -1,
			expErr:   true,
		},
		{
			name:     "cmpField out of range",
			cmpField: 3,
			input:    []byte("1,10\n"),
			chunk:    100,
			expErr:   true, // errFieldNotFound
		},
		{
			name:     "non-numeric field",
			cmpField: 0,
			input:    []byte("x,10\n"),
			chunk:    100,
			expErr:   true, // strconv.Atoi
		},
		{
			name:     "negative cmpField",
			cmpField: -1,
			input:    []byte("1,10\n"),
			chunk:    100,
			expErr:   true,
		},
	}

	mockFs := afero.NewMemMapFs()
	for _, test := range tests {
		T.Run(test.name, func(T *testing.T) {
			inF, err := mockFs.Create("input")
			if err != nil {
				T.Fatal(err)
			}
			if err := fillFile(inF, test.input); err != nil {
				T.Fatal(err)
			}

			tmps, err := splitOnSortedChunks(context.Background(), test.cmpField, inF.Name(), ".", test.chunk, mockFs)
			if !cmpWithBoolErr(err, test.expErr) {
				T.Errorf("unexpected err: got %v", err)
			}
			// При ошибке данные файлов невалидны
			if err != nil {
				return
			}

			T.Cleanup(func() {
				for _, tF := range tmps {
					delF(mockFs, tF)
				}
				delF(mockFs, inF)
			})

			type fieldReaderCounted struct {
				cntr int
				*fieldReader
			}
			// TODO: scanWithCntr()

			readers := make([]*fieldReaderCounted, 0, len(tmps))
			for _, tF := range tmps {
				r := newFieldReader(tF, test.cmpField)
				if err := r.err; err != nil && err != io.EOF {
					T.Fatalf("reader creation: %v", err)
				}
				readers = append(readers, &fieldReaderCounted{fieldReader: r, cntr: len(r.line)})
			}

			ir := fieldReaderCounted{
				fieldReader: newFieldReader(bytes.NewReader(test.input), test.cmpField),
			}
			ir.cntr = len(ir.line)

			for range readers {
				found := false

				for _, fr := range readers {
					if !bytes.Equal(fr.line, ir.line) {
						continue
					}
					found = true

					for {
						if !fr.scan() {
							if ferr := fr.err; ferr != nil && ferr != io.EOF {
								T.Fatalf("tmp reader: %v", ferr)
							}
							break
						}

						fr.line = bytes.TrimSuffix(fr.line, []byte("\n"))
						fr.line = bytes.TrimSuffix(fr.line, []byte("\r"))

						fr.cntr += len(fr.line)

						_ = ir.scan()
						if ierr := ir.err; ierr != nil && ierr != io.EOF {
							T.Fatalf("input reader: %v", ierr)
						}
						ir.line = bytes.TrimSuffix(ir.line, []byte("\n"))
						ir.line = bytes.TrimSuffix(ir.line, []byte("\r"))
						ir.cntr += len(ir.line)

						// Сравниваем строки до обработки EOF,
						// иначе последняя строка пропускается.
						if !slices.Equal(fr.line, ir.line) {
							T.Errorf("non equal lines: fr.line=%q, ir.line=%q", fr.line, ir.line)
						}

						// ferr == nil, но input уже кончился — рассинхронизация.
						if fr.err != ir.err {
							T.Errorf("diff errs: ferr=%v, ierr=%v", fr.err, ir.err)
							break
						}
					}
					// Обновляем ir.line
					_ = ir.scan()

					// Размер проверяем ПОСЛЕ цикла. Последний чанк — тот,
					// после которого ir оказался на EOF. Его размер не ограничен снизу.
					if fr.cntr < test.chunk && ir.err != io.EOF {
						T.Errorf("chunk too small: got %d, want >= %d", fr.cntr, test.chunk)
					}

					// Обрабатываем один чанк за итерацию
					break
				}
				if !found {
					tmpsStarts := make([]string, 0, len(readers))
					for _, fr := range readers {
						tmpsStarts = append(tmpsStarts, string(fr.line))
					}
					T.Errorf("no file with identical start found (input line=\"%s\", tmps starts=\"%+v\")",
						ir.line, tmpsStarts)
				}
			}
			if ir.err != io.EOF {
				T.Errorf("input not fully consumed: remaining=%q", ir.line)
			}
		})
	}
}
