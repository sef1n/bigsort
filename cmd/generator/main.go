package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	dataDisk           = "/mnt/FilesHDD"
	default_csv_name   = "table.csv"
	csvGenerationLines = 1 << 15
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

	filename, err := input(scn, fmt.Sprintf("output name (default %v): ", default_csv_name))
	if err != nil {
		log.Fatal(err)
	}
	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = default_csv_name
	}
	gen_file(filename)
	fmt.Println("Table generated")
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
