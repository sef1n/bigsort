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

var cfg Cfg

func ParseFlags() (Cfg, error) {
	inFile := pflag.StringP("input", "i", default_csv_name, "Path to source (input) file")
	outFile := pflag.StringP("output", "o", default_csv_name, "Path to output file")

	ctxTimeout := pflag.DurationP("timeout", "t", 0, "Processing timeout")
	cmpField := pflag.IntP("field", "f", 0, "Comparing field")

	pflag.Parse()

	cfg = Cfg{
		*inFile,
		*outFile,
		*ctxTimeout,
		*cmpField,
	}
	return cfg, nil
}
