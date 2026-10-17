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

func parseFlags(args []string) (*cfg, error) {
	fs := pflag.NewFlagSet("sorter", pflag.ContinueOnError)

	var c cfg
	fs.StringVarP(&c.inFile, "input", "i", defaultCsvName, "Path to source (input) file")
	fs.StringVarP(&c.outFile, "output", "o", defaultCsvName, "Path to output file")
	fs.DurationVarP(&c.ctxTimeout, "timeout", "t", 0, "Processing timeout")
	fs.IntVarP(&c.cmpField, "field", "f", 0, "Comparing field")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if c.ctxTimeout < 0 {
		return nil, fmt.Errorf("timeout must be >= 0, got %v", c.ctxTimeout)
	}
	if c.cmpField < 0 {
		return nil, fmt.Errorf("comparing field must be >= 0, got %d", c.cmpField)
	}

	return &c, nil
}
