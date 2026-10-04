package main

import (
	"time"

	"github.com/spf13/pflag"
)

type Cfg struct {
	inFile     string
	outFile    string
	ctxTimeout time.Duration
	cmpField   int
}

func ParseFlags() (Cfg, error) {
	var cfg Cfg
	pflag.StringVarP(&cfg.inFile, "input", "i", default_csv_name, "Path to source (input) file")
	pflag.StringVarP(&cfg.outFile, "output", "o", default_csv_name, "Path to output file")

	pflag.DurationVarP(&cfg.ctxTimeout, "timeout", "t", 0, "Processing timeout")
	pflag.IntVarP(&cfg.cmpField, "field", "f", 0, "Comparing field")

	pflag.Parse()

	return cfg, nil
}
