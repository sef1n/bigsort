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
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/sync/errgroup"
)

const (
	dataDisk           = "/mnt/FilesHDD"
	csv_name           = "table.csv"
	tmpFilesPath       = dataDisk + "/" + "tmp"
	predictTableLines  = 100
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
		// INPUT data
		inFilename, err := input(scn, fmt.Sprintf("input table file (default %v): ", csv_name))
		if err != nil {
			log.Fatal(err)
		}
		inFilename = strings.TrimSpace(inFilename)
		if inFilename == "" {
			inFilename = csv_name
		}
		// OUTPUT data
		outFilename, err := input(scn, fmt.Sprintf("output table file (default %v): ", inFilename))
		if err != nil {
			log.Fatal(err)
		}
		outFilename = strings.TrimSpace(outFilename)
		if outFilename == "" {
			outFilename = inFilename
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		if err := ProcessExistingFile(ctx, inFilename, outFilename); err != nil {
			log.Fatal(err)
		}
		fmt.Println("CSV parsed")

	default:
		// WARN: возвращет lower текст
		fmt.Printf("Incorrect input: %q\n", action)
		os.Exit(1)
	}
}

// ProcessExistingFile обрабатывает уже существующую csv таблицу inFilePath: parse, sort, split, merge.
// Сохраняет результат в outFilePath
func ProcessExistingFile(ctx context.Context, inFilePath, outFilePath string) error {
	// Parse
	sourceTableF, err := os.Open(inFilePath)
	if err != nil {
		return err
	}
	defer sourceTableF.Close()
	tableFields, err := ParseCSVTable(sourceTableF, sortingFieldNum)
	if err != nil {
		return err
	}

	// Sort
	sort.Sort(&tableFields)

	// Split
	tmpFiles, err := genMiddleFiles(ctx, inFilePath, tableFields)
	if err != nil {
		return err
	}
	// Defer close&&remove
	defer func() {
		for _, f := range tmpFiles {
			name := f.Name()
			_ = f.Close()
			_ = os.Remove(name)
		}
	}()

	// Merge
	// TODO: при rename EXDEV — разные ФС, создавать tmp рядом с outFilePath
	outFile, err := os.CreateTemp(tmpFilesPath, "sorted-*.csv.tmp")
	if err != nil {
		return err
	}
	outNameTmp := outFile.Name()
	defer func() {
		_ = outFile.Close()
		if err != nil {
			_ = os.Remove(outNameTmp)
		}
	}()

	if err = MergeWithContext(ctx, tmpFiles, outFile); err != nil {
		return err
	}
	// Закрываем перед переименованием (избегаем проблем на Windows)
	if err = outFile.Close(); err != nil {
		return err
	}
	if err = os.Rename(outNameTmp, outFilePath); err != nil {
		return err
	}
	return nil
}

// genMiddleFiles создает tmp файлы содержащие processingFields. Кол-во строк в новых файлах = tmpFileFieldsCount.
// Возвращает слайс файлов, закрывает и удаляет их при ошибке.
func genMiddleFiles(parentCtx context.Context, source string, processingFields fields) ([]*os.File, error) {
	var mu sync.Mutex
	tmpFs := make([]*os.File, 0, (len(processingFields)+tmpFileFieldsCount-1)/tmpFileFieldsCount)

	g, ctx := errgroup.WithContext(parentCtx)
	g.SetLimit(runtime.NumCPU() * 2)

	// Разбиваем слайс на блоки, в отдельных горутинах пишем в файлы
	for start := 0; start < len(processingFields); start += tmpFileFieldsCount {
		end := min(start+tmpFileFieldsCount, len(processingFields))
		g.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			f, err := createTMP(ctx, processingFields[start:end], source, tmpFilesPath)
			if err != nil {
				return err
			}
			mu.Lock()
			tmpFs = append(tmpFs, f)
			mu.Unlock()
			return nil
		})
	}

	// При любой ошибке удаляем все файлы
	if err := g.Wait(); err != nil {
		for _, f := range tmpFs {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}
		return nil, err
	}
	return tmpFs, nil
}

// getFields читает строки из файла, сохраняет Nth поля, возвращает структуру сравниваемых полей
func ParseCSVTable(tableFile *os.File, cmpFieldNum int) (fields, error) {
	processingFields := make(fields, 0, predictTableLines)
	var offset int64
	r := bufio.NewReader(tableFile)

	for {
		line, err := r.ReadBytes('\n')
		RawLineLen := int64(len(line))
		line = bytes.TrimRight(line, "\r\n")

		if len(line) > 0 {
			nthF, err := nthField(line, cmpFieldNum)
			if err != nil {
				return nil, err
			}

			fmt.Printf("Creating new %s\n", line)

			fi, err := newField(offset, string(nthF), sortingFieldNum)
			if err != nil {
				return nil, err
			}

			offset += RawLineLen
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

// createTMP создает tmp файл, заполняет его строками fs.
func createTMP(ctx context.Context, fs fields, sourcePath, creationPath string) (tmpF *os.File, err error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return nil, err
	}
	defer source.Close()

	tmpF, err = os.CreateTemp(creationPath, "")
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
		// Каждые 100 строк проверяем ctx
		// TODO: вычисляем частоту проверки на основе tmpFileFieldsCount/len(fs)?
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
