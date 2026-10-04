package main

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

type Cfg struct {
	inFile     string
	outFile    string
	ctxTimeout time.Duration
	cmpField   int
}

func ParseFlags() (*Cfg, error) {
	var cfg Cfg

	// По дефолту перезаписываем файл
	pflag.StringVarP(&cfg.inFile, "input", "i", default_csv_name, "Path to source (input) file")
	pflag.StringVarP(&cfg.outFile, "output", "o", default_csv_name, "Path to output file")

	// По дефолту context бесконечный
	pflag.DurationVarP(&cfg.ctxTimeout, "timeout", "t", 0, "Processing timeout")
	pflag.IntVarP(&cfg.cmpField, "field", "f", 0, "Comparing field")

	pflag.Parse()

	if cfg.ctxTimeout < 0 {
		return nil, fmt.Errorf("timeout must be >= 0, got %v", cfg.ctxTimeout)
	}
	if cfg.cmpField < 0 {
		return nil, fmt.Errorf("comparing field must be >= 0, got %v", cfg.cmpField)
	}

	return &cfg, nil
}
