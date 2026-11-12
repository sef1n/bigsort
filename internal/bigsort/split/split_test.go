package split

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"slices"
	"testing"

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

			estimatedChunks, err := EstimateChunks(inputF, tt.chunk)
			if err != nil {
				if tt.expErr {
					return
				}
				t.Fatal(err)
			}

			cfg := ChunkPlan{
				In:              inputF,
				TmpDir:          ".",
				CmpFieldNum:     tt.cmpField,
				BytesPerChunk:   DefaultBytesPerChunk,
				EstimatedChunks: estimatedChunks,
			}

			tmps, err := SplitOnSortedChunks(context.Background(), cfg, mockFs)
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

			type chunkState struct {
				r     *bufio.Scanner
				first []byte // Первая строка для поиска чанка
				cur   []byte // Текущая строка
				cntr  int    // байт в чанке (без \r\n)
				eof   bool   // пустой чанк (создан на EOF-итерации)
			}
			css := make([]chunkState, 0, len(tmps))
			for _, tF := range tmps {
				s := bufio.NewScanner(tF)
				if !s.Scan() {
					if err := s.Err(); err != nil {
						t.Fatalf("scanner: %v", err)
					}
				}

				first := append([]byte(nil), s.Bytes()...)
				if len(first) == 0 {
					css = append(css, chunkState{r: s, eof: true})
					continue
				}
				css = append(css, chunkState{r: s, first: first, cur: first, cntr: len(first)})
			}

			// Порядок чанков недетерминирован (см. WARN) — восстанавливаем по значению.
			slices.SortFunc(css, func(a, b chunkState) int {
				return bytes.Compare(a.first, b.first)
			})

			is := func() chunkState {
				s := bufio.NewScanner(bytes.NewReader(tt.input))
				if !s.Scan() {
					if err := s.Err(); err != nil {
						t.Fatalf("I reader err: %v", err)
					}
				}
				first := append([]byte(nil), s.Bytes()...)
				return chunkState{r: s, first: first, cur: first, cntr: len(first)}
			}()

			for _, c := range css {
				t.Log(string(c.first))
			}
			for n := range css {
				cs := &css[n]
				if cs.eof {
					continue // пустой чанк — не двигает ir
				}
				t.Log(string(cs.first))

				// Первая строка чанка должна совпасть с текущей позицией is.
				t.Log(string(cs.first), string(is.cur))
				if !bytes.Equal(cs.first, is.cur) {
					t.Fatalf("chunk %d: first mismatch: cs=%q ir=%q", n, cs.first, is.cur)
				}

				// Каждая следующая строка чанка == следующая строка input.
				for {
					if !cs.r.Scan() {
						if err := cs.r.Err(); err != nil {
							t.Fatalf("chunk %d: %v", n, err)
						}
						break // EOF
					}
					cs.cur = cs.r.Bytes()
					cs.cntr += len(cs.cur)

					if !is.r.Scan() {
						if err := is.r.Err(); err != nil {
							t.Fatalf("chunk %d: input: %v", n, is.r.Err())
						}
						t.Fatalf("chunk %d: input ended before chunk", n)
					}
					is.cur = is.r.Bytes()
					is.cntr += len(is.cur)

					if !bytes.Equal(cs.cur, is.cur) {
						t.Errorf("chunk %d: %q != %q", n, cs.cur, is.cur)
					}
				}

				// Последний физический чанк может быть < bytesPerChunk (хвост input).
				// Эвристика: см. WARN — считаем, что данные уникальны и порядок по
				// значению == порядок по позиции, тогда последний в cs — последний.
				if n < len(css)-1 && cs.cntr < tt.chunk {
					t.Errorf("chunk %d too small: %d < %d", n, cs.cntr, tt.chunk)
				}

				// Переход к первой строке следующего чанка.
				// На последнем чанке вернёт io.EOF — норм.
				if !is.r.Scan() {
					if err := is.r.Err(); err != nil {
						t.Fatalf("chunk %d: input: %v", n, err)
					}
					// EOF на последнем чанке
				}
				is.cur = is.r.Bytes()
			}

			// Input должен быть полностью исчерпан
			if is.r.Scan() {
				t.Errorf("input not fully consumed: %q", is.r.Bytes())
			}
			if err := is.r.Err(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
