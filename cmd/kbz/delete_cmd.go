package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sambeau/kanbanzai/internal/core"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/service"
)

const deleteUsageText = `Usage: kbz delete <file-path> [--force]

Delete a document file and its associated record.

Arguments:
  <file-path>    Path to the document file, relative to repo root (must begin with work/)

Flags:
  --force    Bypass confirmation prompt and approved-document guard
`

func runDelete(args []string, deps dependencies) error {
	force := false
	var positional []string
	for _, arg := range args {
		if arg == "--force" {
			force = true
		} else {
			positional = append(positional, arg)
		}
	}

	if len(positional) != 1 {
		return fmt.Errorf("expected exactly one file path\n\n%s", deleteUsageText)
	}
	path := positional[0]

	if !strings.HasPrefix(path, "work/") {
		return fmt.Errorf("%q is not within work/ — only files under work/ can be deleted", path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file %q not found", path)
	}

	stateRoot := core.StatePath()
	repoRoot := "."
	docSvc := service.NewDocumentService(stateRoot, repoRoot)
	docs, err := docSvc.ListDocuments(service.DocumentFilters{})
	if err != nil {
		return fmt.Errorf("list documents: %w", err)
	}
	var matches []service.DocumentResult
	for _, doc := range docs {
		if doc.Path == path {
			matches = append(matches, doc)
		}
	}
	if len(matches) > 1 {
		return fmt.Errorf("multiple document records found for %q — cannot safely delete", path)
	}
	var record *service.DocumentResult
	if len(matches) == 1 {
		r := matches[0]
		record = &r
	}
	if record != nil && record.Status == string(model.DocumentStatusApproved) && !force {
		return fmt.Errorf("%q is an approved document — re-run with --force to delete it", path)
	}
	if !force {
		fmt.Fprintf(deps.stdout, "Delete %s and its document record? [y/N] ", path)
		reader := bufio.NewReader(deps.stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		if answer != "y" && answer != "Y" {
			fmt.Fprintf(deps.stdout, "Aborted.\n")
			return nil
		}
	}
	cmd := exec.Command("git", "rm", path)
	cmd.Dir = repoRoot
	var gitStderr bytes.Buffer
	cmd.Stderr = &gitStderr
	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(gitStderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return fmt.Errorf("git rm failed: %s", errMsg)
	}
	if record != nil {
		if _, err := docSvc.DeleteDocument(service.DeleteDocumentInput{
			ID:    record.ID,
			Force: true,
		}); err != nil {
			return fmt.Errorf("partial state: file deleted but document record %q could not be removed: %w", record.ID, err)
		}
		fmt.Fprintf(deps.stdout, "Deleted %s (document record %s removed)\n", path, record.ID)
	} else {
		fmt.Fprintf(deps.stdout, "No document record found — file deleted but no record updated\n")
	}
	return nil
}
