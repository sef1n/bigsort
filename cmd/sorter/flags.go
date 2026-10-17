package main

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

const defaultCsvName = "table.csv"

type cfg struct {
	inFile     string
	outFile    string
	ctxTimeout time.Duration
	cmpField   int
}

func parseFlags() (*cfg, error) {
	var config cfg
	// По дефолту перезаписываем файл
	pflag.StringVarP(&config.inFile, "input", "i", defaultCsvName, "Path to source (input) file")
	pflag.StringVarP(&config.outFile, "output", "o", defaultCsvName, "Path to output file")

	// По дефолту context бесконечный
	pflag.DurationVarP(&config.ctxTimeout, "timeout", "t", 0, "Processing timeout")
	pflag.IntVarP(&config.cmpField, "field", "f", 0, "Comparing field")

	pflag.Parse()

	if config.ctxTimeout < 0 {
		return nil, fmt.Errorf("timeout must be >= 0, got %v", config.ctxTimeout)
	}
	if config.cmpField < 0 {
		return nil, fmt.Errorf("comparing field must be >= 0, got %v", config.cmpField)
	}

	return &config, nil
}
