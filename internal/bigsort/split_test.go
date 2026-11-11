package bigsort

import (
	"bytes"
	"context"
	"io"
	"slices"
	"testing"

	"bigFiles/internal/bigsort/merge"
	"bigFiles/internal/bigsort/testutils"

	"github.com/spf13/afero"
)

type test struct {
	name     string
	cmpField int
	input    []byte
	chunk    int
	expErr   bool
}

// WARN: этот тест не поддерживает чанки, начинающиеся с ОДИНАКОВЫХ строк:
// порядок tmp-файлов недетерминирован, восстановление идёт по значению
// первой строки. Для таких данных нужен отдельный тест с доступом к
// внутреннему состоянию splitOnSortedChunks (пока нет).
// Тестовые данные — с уникальными первыми строками.
func TestSplitOnSortedChunks(t *testing.T) {
	tests := []test{
		// ---- без ошибок ----
		{name: "empty", cmpField: 0, input: []byte{}, chunk: 100},
		{name: "single line fits", cmpField: 0, input: []byte("1,a\n"), chunk: 100},
		{name: "multiple lines one chunk", cmpField: 0, input: []byte("1,a\n2,b\n3,c\n"), chunk: 100},
		{name: "exact chunk boundary", cmpField: 0, input: []byte("12,ab\n34,cd\n"), chunk: 10},
		{name: "two chunks overshoot", cmpField: 0, input: []byte("1,a\n2,b\n3,c\n4,d\n"), chunk: 5},
		{name: "last chunk smaller", cmpField: 0, input: []byte("1,aaaa\n2,bb\n"), chunk: 8},
		{name: "cmpField=1", cmpField: 1, input: []byte("1,10\n2,20\n3,30\n"), chunk: 100},
		// ---- с ошибками ----
		{name: "zero chunk", cmpField: 0, input: []byte("1,10\n"), chunk: 0, expErr: true},
		{name: "negative chunk", cmpField: 0, input: []byte("1,10\n"), chunk: -1, expErr: true},
		{name: "cmpField out of range", cmpField: 3, input: []byte("1,10\n"), chunk: 100, expErr: true},
		{name: "non-numeric field", cmpField: 0, input: []byte("x,10\n"), chunk: 100, expErr: true},
		{name: "negative cmpField", cmpField: -1, input: []byte("1,10\n"), chunk: 100, expErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockFs := afero.NewMemMapFs()

			inputF, err := mockFs.Create("input")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { testutils.DelF(mockFs, inputF) })

			if err := testutils.FillFile(inputF, tt.input); err != nil {
				t.Fatal(err)
			}
			if _, err := inputF.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}

			estimatedChunks, err := estimateChunks(inputF, defaultBytesPerChunk)
			if err != nil {
				t.Fatal(err)
			}

			cfg := chunkPlan{
				inFile:          inputF,
				tmpDir:          ".",
				cmpFieldNum:     tt.cmpField,
				bytesPerChunk:   tt.chunk,
				estimatedChunks: estimatedChunks,
			}

			tmps, err := splitOnSortedChunks(context.Background(), cfg, mockFs)
			if !testutils.CmpWithBoolErr(err, tt.expErr) {
				t.Errorf("unexpected err: got %v", err)
			}
			if err != nil {
				return
			}
			for _, tF := range tmps {
				tF := tF
				t.Cleanup(func() { testutils.DelF(mockFs, tF) })
			}

			// ---- Pre-read: newFieldReader уже прочитал первую строку чанка ----
			type chunkState struct {
				fr    *merge.FieldReader
				first []byte // копия: fr.line перезаписывается read()
				cntr  int    // байт в чанке (без \r\n)
				eof   bool   // пустой чанк (создан на EOF-итерации)
			}
			cs := make([]chunkState, 0, len(tmps))
			for _, tF := range tmps {
				fr, ferr := merge.NewFieldReader(tF, tt.cmpField)
				if ferr != nil && ferr != io.EOF {
					t.Fatalf("newFieldReader: %v", ferr)
				}
				if len(fr.Line) == 0 {
					cs = append(cs, chunkState{fr: fr, eof: true})
					continue
				}
				first := append([]byte(nil), fr.Line...)
				cs = append(cs, chunkState{fr: fr, first: first, cntr: len(first)})
			}

			// Порядок чанков недетерминирован (см. WARN) — восстанавливаем по значению.
			slices.SortFunc(cs, func(a, b chunkState) int {
				return bytes.Compare(a.first, b.first)
			})

			// ---- Input reader: init прочитал первую строку ----
			ir, err := merge.NewFieldReader(bytes.NewReader(tt.input), tt.cmpField)
			if err != nil && err != io.EOF {
				t.Fatal(err)
			}

			for i := range cs {
				c := &cs[i]
				if c.eof {
					continue // пустой чанк — не двигает ir
				}

				// Первая строка чанка должна совпасть с текущей позицией ir.
				if !bytes.Equal(c.first, ir.Line) {
					t.Fatalf("chunk %d: first mismatch: fr=%q ir=%q", i, c.first, ir.Line)
				}

				// Дренаж: каждая следующая строка чанка == следующая строка input.
				for {
					ferr := c.fr.Read()
					if ferr == io.EOF {
						break
					}
					if ferr != nil {
						t.Fatalf("chunk %d: %v", i, ferr)
					}
					c.cntr += len(c.fr.Line)

					ierr := ir.Read()
					if ierr != nil && ierr != io.EOF {
						t.Fatalf("chunk %d: input: %v", i, ierr)
					}
					if ierr == io.EOF {
						t.Fatalf("chunk %d: input ended before chunk", i)
					}
					if !bytes.Equal(c.fr.Line, ir.Line) {
						t.Errorf("chunk %d: %q != %q", i, c.fr.Line, ir.Line)
					}
				}

				// Последний физический чанк может быть < bytesPerChunk (хвост input).
				// Эвристика: см. WARN — считаем, что данные уникальны и порядок по
				// значению == порядок по позиции, тогда последний в cs — последний.
				if i < len(cs)-1 && c.cntr < tt.chunk {
					t.Errorf("chunk %d too small: %d < %d", i, c.cntr, tt.chunk)
				}

				// Переход к первой строке следующего чанка.
				// На последнем чанке вернёт io.EOF — норм.
				if err := ir.Read(); err != nil && err != io.EOF {
					t.Fatal(err)
				}
			}

			// ---- Input должен быть полностью исчерпан ----
			if err := ir.Read(); err != io.EOF {
				if err == nil {
					t.Errorf("input not fully consumed: %q", ir.Line)
				} else {
					t.Fatal(err)
				}
			}
		})
	}
}
