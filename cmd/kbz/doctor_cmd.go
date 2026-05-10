package main

import (
	"fmt"
	"os"

	"github.com/sambeau/kanbanzai/internal/kbzdoctor"
)

func runDoctor(args []string, deps dependencies) error {
	doctor := kbzdoctor.New(deps.stdout, os.Stderr)
	results, err := doctor.Run(".")
	if err != nil {
		return fmt.Errorf("doctor: %w", err)
	}
	doctor.PrintResults(results)
	code := kbzdoctor.ExitCode(results)
	if code != 0 {
		os.Exit(code)
	}
	return nil
}
