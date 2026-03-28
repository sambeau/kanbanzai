// Command schemagen generates the Kanbanzai JSON Schema file from the public
// Go types defined in the kbzschema package. Run it from the repository root:
//
//	go run ./cmd/schemagen -o schema/kanbanzai.schema.json
//
// The generated schema is committed to the repository and verified by a CI
// check in the kbzschema package tests.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sambeau/kanbanzai/kbzschema"
)

func main() {
	outFlag := flag.String("o", "schema/kanbanzai.schema.json", "output file path")
	flag.Parse()

	data, err := kbzschema.GenerateSchema()
	if err != nil {
		fmt.Fprintf(os.Stderr, "schemagen: %v\n", err)
		os.Exit(1)
	}

	// Append a trailing newline for clean diffs.
	data = append(data, '\n')

	outPath := *outFlag
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "schemagen: create output directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "schemagen: write %s: %v\n", outPath, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "wrote %s\n", outPath)
}
