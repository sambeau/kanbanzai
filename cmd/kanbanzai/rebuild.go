package main

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/sambeau/kanbanzai/internal/core"
	"github.com/sambeau/kanbanzai/internal/service"
)

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
