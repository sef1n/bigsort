package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"bigFiles/internal/bigsort"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	config, err := parseFlags(args)
	if err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	parentCtx := context.Background()
	if config.ctxTimeout != 0 {
		var cancel context.CancelFunc
		parentCtx, cancel = context.WithTimeout(parentCtx, config.ctxTimeout)
		defer cancel()
	}

	ctx, cancel := signal.NotifyContext(parentCtx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return bigsort.SortFile(ctx, config.cmpField, config.inFile, config.outFile)
}
