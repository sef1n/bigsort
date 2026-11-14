package merge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"bigFiles/internal/bigsort/testutils"

	"github.com/spf13/afero"
)

// makeFiles creates one afero.File per content entry, seeks each to start.
// Registers cleanup: closes and removes all files.
func makeFiles(t *testing.T, fs afero.Fs, contents ...[]byte) []afero.File {
	t.Helper()
	files := make([]afero.File, 0, len(contents))
	for i, c := range contents {
		name := fmt.Sprintf("f%d", i)
		f, err := fs.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := f.Write(c); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			t.Fatalf("seek %s: %v", name, err)
		}
		files = append(files, f)
	}
	t.Cleanup(func() {
		for _, f := range files {
			testutils.DelF(fs, f)
		}
	})
	return files
}

// b is a helper for []byte test data.
func b(s string) []byte { return []byte(s) }

// failWriter always returns err on Write.
type failWriter struct{ err error }

func (w *failWriter) Write(p []byte) (int, error) { return 0, w.err }

func TestMergeWithContext(t *testing.T) {
	tests := []struct {
		name        string
		cmpFieldNum int
		files       [][]byte
		want        string
	}{
		{
			name:        "two interleaved files",
			cmpFieldNum: 0,
			files:       [][]byte{b("1,a\n3,c\n5,e\n"), b("2,b\n4,d\n6,f\n")},
			want:        "1,a\n2,b\n3,c\n4,d\n5,e\n6,f\n",
		},
		{
			name:        "single file",
			cmpFieldNum: 0,
			files:       [][]byte{b("1,a\n2,b\n")},
			want:        "1,a\n2,b\n",
		},
		{
			name:        "no files",
			cmpFieldNum: 0,
			files:       nil,
			want:        "",
		},
		{
			name:        "all empty files",
			cmpFieldNum: 0,
			files:       [][]byte{b(""), b(""), b("")},
			want:        "",
		},
		{
			name:        "empty files skipped",
			cmpFieldNum: 0,
			files:       [][]byte{b(""), b("2,b\n"), b(""), b("1,a\n")},
			want:        "1,a\n2,b\n",
		},
		{
			name:        "cmpFieldNum=1",
			cmpFieldNum: 1,
			files:       [][]byte{b("a,1\nc,3\n"), b("b,2\nd,4\n")},
			want:        "a,1\nb,2\nc,3\nd,4\n",
		},
		{
			name:        "crlf lines",
			cmpFieldNum: 0,
			files:       [][]byte{b("1,a\r\n3,c\r\n"), b("2,b\r\n")},
			want:        "1,a\n2,b\n3,c\n", // \r stripped by reader, \n added by merge
		},
		{
			name:        "last line without trailing newline",
			cmpFieldNum: 0,
			files:       [][]byte{b("1,a\n3,c"), b("2,b\n")},
			want:        "1,a\n2,b\n3,c\n", // merge always appends '\n'
		},
		{
			name:        "single-line files",
			cmpFieldNum: 0,
			files:       [][]byte{b("3,c\n"), b("1,a\n"), b("2,b\n")},
			want:        "1,a\n2,b\n3,c\n",
		},
		{
			name:        "negative keys across files",
			cmpFieldNum: 0,
			files:       [][]byte{b("-2,b\n1,a\n"), b("-1,c\n0,d\n")},
			want:        "-2,b\n-1,c\n0,d\n1,a\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			files := makeFiles(t, fs, tc.files...)

			var buf bytes.Buffer
			if err := MergeWithContext(context.Background(), tc.cmpFieldNum, files, &buf); err != nil {
				t.Fatalf("err: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Errorf("got:\n%q\nwant:\n%q", got, tc.want)
			}
		})
	}
}

// TestMergeWithContext_Duplicates checks that equal keys don't lose lines.
// Order between lines with equal keys is undefined (k-way merge is not stable).
func TestMergeWithContext_Duplicates(t *testing.T) {
	fs := afero.NewMemMapFs()
	files := makeFiles(t, fs, b("1,a\n1,c\n"), b("1,b\n1,d\n"))

	var buf bytes.Buffer
	if err := MergeWithContext(context.Background(), 0, files, &buf); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("want 4 lines, got %d: %v", len(lines), lines)
	}
	wantSet := map[string]bool{"1,a": true, "1,b": true, "1,c": true, "1,d": true}
	for _, l := range lines {
		if !wantSet[l] {
			t.Errorf("unexpected line: %q", l)
		}
		delete(wantSet, l)
	}
	if len(wantSet) != 0 {
		t.Errorf("missing lines: %v", wantSet)
	}
}

func TestMergeWithContext_ContextCancelled(t *testing.T) {
	fs := afero.NewMemMapFs()
	files := makeFiles(t, fs, b("1,a\n"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: first loop iteration returns ctx.Err()

	var buf bytes.Buffer
	err := MergeWithContext(ctx, 0, files, &buf)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
}

func TestMergeWithContext_WriteError(t *testing.T) {
	fs := afero.NewMemMapFs()
	files := makeFiles(t, fs, b("1,a\n"))

	wantErr := errors.New("disk full")
	w := &failWriter{err: wantErr}

	// bufio.Writer buffers; the error surfaces on w.Flush() at the end.
	err := MergeWithContext(context.Background(), 0, files, w)
	if !errors.Is(err, wantErr) {
		t.Errorf("want writeErr, got %v", err)
	}
}

func TestMergeWithContext_ReadError(t *testing.T) {
	// First line valid, second fails on Atoi.
	fs := afero.NewMemMapFs()
	files := makeFiles(t, fs, b("1,a\nxyz,b\n"))

	var buf bytes.Buffer
	err := MergeWithContext(context.Background(), 0, files, &buf)
	if err == nil {
		t.Fatal("want read error, got nil")
	}
	var ne *strconv.NumError
	if !errors.As(err, &ne) {
		t.Errorf("want *strconv.NumError, got %T: %v", err, err)
	}
}

func TestMergeWithContext_NegativeCmpFieldNum(t *testing.T) {
	fs := afero.NewMemMapFs()
	files := makeFiles(t, fs, b("1,a\n"))

	var buf bytes.Buffer
	err := MergeWithContext(context.Background(), -1, files, &buf)
	if err == nil {
		t.Fatal("want error for negative cmpFieldNum")
	}
}
