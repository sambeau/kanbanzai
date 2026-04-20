package main

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/sambeau/kanbanzai/internal/core"
	"github.com/sambeau/kanbanzai/internal/service"
)

const rebuildIndexUsageText = `Usage: kbz rebuild-index

Rebuilds the SQLite document intelligence index from all per-document YAML index files.
Deletes the existing database and recreates it from scratch.

This command is useful after upgrading Kanbanzai or when the SQLite database is missing or corrupted.
`

// runRebuildIndex rebuilds the SQLite document intelligence index.
func runRebuildIndex(_ []string, deps dependencies) error {
	out := deps.stdout
	if out == nil {
		out = io.Discard
	}

	fmt.Fprintln(out, "Rebuilding document index...")

	repoRoot := "."
	indexRoot := filepath.Join(core.InstanceRootDir, "index")
	intelSvc := service.NewIntelligenceService(indexRoot, repoRoot)
	defer intelSvc.Close()

	stats, err := intelSvc.RebuildIndex()
	if err != nil {
		return fmt.Errorf("rebuild-index: %w", err)
	}

	fmt.Fprintf(out, "  Documents:          %d\n", stats.Documents)
	fmt.Fprintf(out, "  Edges:              %d\n", stats.Edges)
	fmt.Fprintf(out, "  Entity references:  %d\n", stats.EntityRefs)
	fmt.Fprintf(out, "  FTS sections:       %d\n", stats.FTSSections)
	fmt.Fprintln(out, "Done.")
	return nil
}
