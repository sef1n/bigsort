package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

const (
	dataDisk           = "/mnt/FilesHDD"
	csv_name           = "small_csv_file.tmp"
	csv_full_path      = dataDisk + "/" + csv_name
	tmpFilesPath       = dataDisk + "/" + "tmp"
	csvGenerationLines = 1 << 10
	sortingFieldNum    = 0
	tmpFileFieldsCount = 1000
)

var genUsedLetters = []string{"a", "b", "c", "d", "e"}

func input(scn *bufio.Scanner, msg string) (string, error) {
	fmt.Print(msg)
	if !scn.Scan() {
		if err := scn.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return scn.Text(), nil
}

func main() {
	scn := bufio.NewScanner(os.Stdin)

	fmt.Println("g - for file generation")
	fmt.Println("p - for csv parsing")

	action, err := input(scn, "Action (g/p): ")
	if err != nil {
		log.Fatal(err)
	}
	action = strings.ToLower(strings.TrimSpace(action))

	switch action {
	case "g":
		filename, err := input(scn, fmt.Sprintf("output name (default %v): ", csv_name))
		if err != nil {
			log.Fatal(err)
		}
		filename = strings.TrimSpace(filename)
		if filename == "" {
			filename = csv_name
		}
		gen_file(filename)
		fmt.Println("Table generated")

	case "p":
		filename, err := input(scn, fmt.Sprintf("table file (default %v): ", csv_name))
		if err != nil {
			log.Fatal(err)
		}
		filename = strings.TrimSpace(filename)
		if filename == "" {
			filename = csv_name
		}
		if err := parseAndMerge(filename, ""); err != nil {
			log.Fatal(err)
		}
		fmt.Println("CSV parsed")

	default:
		fmt.Printf("Incorrect input: %q\n", action)
		os.Exit(1)
	}
}

func parseAndMerge(inName, outName string) (err error) {
	if outName == "" {
		outName = inName
	}

	tmpFiles, err := genMergers(inName)
	if err != nil {
		return err
	}
	defer func() {
		for _, f := range tmpFiles {
			name := f.Name()
			_ = f.Close()
			_ = os.Remove(name)
		}
	}()

	outF, err := os.CreateTemp(tmpFilesPath, "sorted-*.csv.tmp")
	if err != nil {
		return err
	}
	outNameTmp := outF.Name()
	defer func() {
		_ = outF.Close()
		if err != nil {
			_ = os.Remove(outNameTmp)
		}
	}()

	if err = Merge(tmpFiles, outF); err != nil {
		return err
	}
	if err = outF.Close(); err != nil {
		return err
	}
	if err = os.Rename(outNameTmp, outName); err != nil {
		return err
	}
	return nil
}

func genMergers(tablePath string) ([]*os.File, error) {
	processingFields, err := getFields(tablePath, 0)
	if err != nil {
		return nil, err
	}
	sort.Sort(&processingFields)
	return genMiddleFiles(tablePath, processingFields, context.Background())
}

func genMiddleFiles(source string, processingFields fields, parentCtx context.Context) ([]*os.File, error) {
	var mu sync.Mutex
	tmpFs := make([]*os.File, 0, (len(processingFields)+tmpFileFieldsCount-1)/tmpFileFieldsCount)

	g, ctx := errgroup.WithContext(parentCtx)
	g.SetLimit(runtime.NumCPU() * 2)

	for start := 0; start < len(processingFields); start += tmpFileFieldsCount {
		end := min(start+tmpFileFieldsCount, len(processingFields))
		g.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			f, err := sortedTMP(ctx, processingFields[start:end], source)
			if err != nil {
				return err
			}
			mu.Lock()
			tmpFs = append(tmpFs, f)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		for _, f := range tmpFs {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}
		return nil, err
	}
	return tmpFs, nil
}

func countLines(fPath string) (count int, err error) {
	var out []byte
	if out, err = exec.Command("wc", "-l", fPath).Output(); err != nil {
		return
	}
	count, err = strconv.Atoi(strings.Fields(string(out))[0])
	if err != nil {
		return
	}
	return
}

func getFields(tablePath string, cmpFieldNum int) (fields, error) {
	tableLines, err := countLines(tablePath)
	if err != nil {
		return nil, err
	}
	tableLines++

	table, err := os.Open(tablePath)
	if err != nil {
		return nil, err
	}
	defer table.Close()

	processingFields := make(fields, 0, tableLines)

	var offset int64
	r := bufio.NewReader(table)
	for {
		l, err := r.ReadBytes('\n')
		if len(l) > 0 {
			nthF, err := nthField(l, cmpFieldNum)
			if err != nil {
				return nil, err
			}
			fi, err := newField(offset, string(nthF), sortingFieldNum)
			if err != nil {
				return nil, err
			}
			offset += int64(len(l))
			processingFields = append(processingFields, fi)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return processingFields, nil
}

func nthField(line []byte, n int) ([]byte, error) {
	start := 0
	for i := 0; i < n; i++ {
		idx := bytes.IndexByte(line[start:], ',')
		if idx < 0 {
			return nil, fmt.Errorf("field %d not found in %q", n, line)
		}
		start += idx + 1
		if start > len(line) {
			return nil, fmt.Errorf("field %d beyond end", n)
		}
	}
	end := bytes.IndexByte(line[start:], ',')
	if end < 0 {
		end = len(line) - start
	}
	return line[start : start+end], nil
}

func sortedTMP(ctx context.Context, fs fields, sourcePath string) (tmpF *os.File, err error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return nil, err
	}
	defer source.Close()

	tmpF, err = os.CreateTemp(tmpFilesPath, "")
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			_ = os.Remove(tmpF.Name())
			_ = tmpF.Close()
			tmpF = nil
		}
	}()

	w := bufio.NewWriter(tmpF)
	reader := bufio.NewReader(source)

	for i, v := range fs {
		if i%100 == 0 {
			if err = ctx.Err(); err != nil {
				return
			}
		}
		if _, err = source.Seek(v.offset, io.SeekStart); err != nil {
			return
		}
		reader.Reset(source)

		var line string
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return
		}

		if _, err = w.WriteString(line); err != nil {
			return
		}
	}
	if err = w.Flush(); err != nil {
		return
	}
	if _, err = tmpF.Seek(0, 0); err != nil {
		return
	}
	return tmpF, nil
}

func gen_data(n int) []string {
	line := make([]string, 0, len(genUsedLetters)+1)
	line = append(line, strconv.Itoa(rand.Int()))
	for _, l := range genUsedLetters {
		line = append(line, strings.Repeat(l, n))
	}
	return line
}

func gen_file(outName string) {
	out, err := os.Create(filepath.Join(dataDisk, outName))
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	w := csv.NewWriter(out)
	defer func() {
		fmt.Println("Flushing...")
		w.Flush()
	}()

	for n := 1; n <= csvGenerationLines; n++ {
		fmt.Printf("Writing %v/%v\r", n, csvGenerationLines)
		d := gen_data(n)
		if err := w.Write(d); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println()
	fmt.Println("All generated")
}
