package main

import (
	"fmt"
	"time"

	"bigFiles/internal/bigsort"

	"github.com/spf13/pflag"
)

const defaultCsvName = "table.csv"

type cfg struct {
	bigsort.SortConfig
	ctxTimeout time.Duration
}

func parseFlags(args []string) (*cfg, error) {
	fs := pflag.NewFlagSet("sorter", pflag.ContinueOnError)

	c := cfg{SortConfig: bigsort.DefaultSortConfig()}

	fs.StringVarP(&c.SortConfig.InPath, "input", "i", defaultCsvName, "Path to source (input) file")
	fs.StringVarP(&c.SortConfig.OutPath, "output", "o", defaultCsvName, "Path to output file")
	fs.IntVarP(&c.SortConfig.CmpFieldNum, "field", "f", 0, "Comparing field")

	fs.DurationVarP(&c.ctxTimeout, "timeout", "t", 0, "Processing timeout")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if c.ctxTimeout < 0 {
		return nil, fmt.Errorf("timeout must be >= 0, got %v", c.ctxTimeout)
	}
	return &c, nil
}
