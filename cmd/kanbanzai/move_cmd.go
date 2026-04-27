package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/sambeau/kanbanzai/internal/core"
	igit "github.com/sambeau/kanbanzai/internal/git"
	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/service"
)

const moveUsageText = `Usage: kbz move <src-path> <plan-id>
       kbz move <plan-feature-ref>

Move a document file to the canonical location for a plan (Mode 1),
or re-parent a feature to a different plan (Mode 2, not yet implemented).

Mode 1 arguments:
  <src-path>         Path to the source document file (must begin with work/)
  <plan-id>          Target plan ID (e.g. P37 or P37-file-names-and-actions)

Mode 2 arguments:
  <plan-feature-ref> Feature reference in the form P{n}-F{m}
`

func runMove(args []string, deps dependencies) error {
	if len(args) == 0 {
		return fmt.Errorf("expected arguments\n\n%s", moveUsageText)
	}

	// Mode 2: first arg matches P{n}-F{m} pattern
	if isMode2Arg(args[0]) {
		return fmt.Errorf("kbz move Mode 2 (feature re-parent) is not yet implemented")
	}

	// Mode 1: requires exactly <src-path> <plan-id>
	if len(args) != 2 {
		return fmt.Errorf("expected exactly 2 arguments for Mode 1\n\n%s", moveUsageText)
	}

	srcPath := args[0]
	planArg := args[1]

	// REQ-001: Mode 1 detection — first arg must look like a file path
	isFilePath := strings.Contains(srcPath, "/") ||
		strings.HasSuffix(srcPath, ".md") ||
		strings.HasSuffix(srcPath, ".txt")
	if !isFilePath {
		return fmt.Errorf("%q does not look like a file path; expected a path containing '/' or ending with .md/.txt\n\n%s", srcPath, moveUsageText)
	}

	// REQ-002: Source path must be within work/
	if !strings.HasPrefix(srcPath, "work/") {
		return fmt.Errorf("%q is not within work/ — only files under work/ can be moved", srcPath)
	}

	// REQ-003: Source file must exist
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("source file %q not found", srcPath)
	}

	stateRoot := core.StatePath()
	repoRoot := "."

	// REQ-004: Resolve and validate the target plan
	fullPlanID, planShortID, planSlug, err := resolvePlanArg(planArg, stateRoot)
	if err != nil {
		return err
	}

	// REQ-005: Look up document record; infer type if no record
	docSvc := service.NewDocumentService(stateRoot, repoRoot)
	docs, err := docSvc.ListDocuments(service.DocumentFilters{})
	if err != nil {
		return fmt.Errorf("list documents: %w", err)
	}
	var record *service.DocumentResult
	for _, doc := range docs {
		d := doc
		if d.Path == srcPath {
			record = &d
			break
		}
	}

	var docType string
	if record != nil && record.Type != "" {
		docType = record.Type
	} else {
		docType = inferTypeFromFilename(filepath.Base(srcPath))
	}

	// REQ-006: Build target path
	ext := filepath.Ext(srcPath)
	if ext == "" {
		ext = ".md"
	}
	filestem := extractFileStem(srcPath, docType, record)
	targetFolder := filepath.Join("work", planShortID+"-"+planSlug)
	targetFile := planShortID + "-" + docType + "-" + filestem + ext
	dstPath := filepath.Join(targetFolder, targetFile)

	// REQ-007: Reject if target already exists
	if _, err := os.Stat(dstPath); err == nil {
		return fmt.Errorf("target path %q already exists", dstPath)
	}

	// REQ-008: Create target directory
	if err := os.MkdirAll(targetFolder, 0o755); err != nil {
		return fmt.Errorf("create target directory %q: %w", targetFolder, err)
	}

	// REQ-009: Move via git mv
	if err := igit.GitMove(repoRoot, srcPath, dstPath); err != nil {
		return fmt.Errorf("git mv failed: %w", err)
	}

	// REQ-010/011: Update record or print no-record notice
	if record != nil {
		if _, err := docSvc.UpdateDocumentPathAndOwner(record.ID, dstPath, fullPlanID); err != nil {
			return fmt.Errorf("partial state: file moved but document record %q could not be updated: %w", record.ID, err)
		}
	} else {
		fmt.Fprintf(deps.stdout, "No document record found — file moved but no record updated\n")
	}

	// REQ-012: Success output
	fmt.Fprintf(deps.stdout, "Moved %s → %s\n", srcPath, dstPath)
	return nil
}

// resolvePlanArg resolves a plan argument (short ID like "P37" or full ID like
// "P37-my-plan") to the full plan ID, short ID (prefix+number), and slug.
// Returns an error if the plan cannot be found.
func resolvePlanArg(arg, stateRoot string) (fullID, shortID, slug string, err error) {
	plansDir := filepath.Join(stateRoot, "plans")

	if model.IsPlanID(arg) {
		// Full plan ID — verify the file exists on disk
		planFile := filepath.Join(plansDir, arg+".yaml")
		if _, statErr := os.Stat(planFile); os.IsNotExist(statErr) {
			return "", "", "", fmt.Errorf("plan %q not found", arg)
		}
		prefix, num, planSlug := model.ParsePlanID(arg)
		return arg, prefix + num, planSlug, nil
	}

	// Short ID like "P37" — scan the plans directory for a matching file
	entries, readErr := os.ReadDir(plansDir)
	if readErr != nil {
		return "", "", "", fmt.Errorf("read plans directory: %w", readErr)
	}

	argLower := strings.ToLower(arg)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		if !model.IsPlanID(name) {
			continue
		}
		prefix, num, planSlug := model.ParsePlanID(name)
		candidate := prefix + num
		if strings.ToLower(candidate) == argLower {
			return name, candidate, planSlug, nil
		}
	}

	return "", "", "", fmt.Errorf("plan %q not found", arg)
}

// inferTypeFromFilename infers a document type from the base filename.
// Returns "design" as the default.
func inferTypeFromFilename(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.Contains(lower, "dev-plan"), strings.Contains(lower, "devplan"):
		return "dev-plan"
	case strings.Contains(lower, "spec"):
		return "spec"
	case strings.Contains(lower, "research"):
		return "research"
	case strings.Contains(lower, "report"):
		return "report"
	case strings.Contains(lower, "retro"):
		return "retro"
	case strings.Contains(lower, "review"):
		return "review"
	case strings.Contains(lower, "plan"):
		return "plan"
	case strings.Contains(lower, "design"):
		return "design"
	default:
		return "design"
	}
}

// extractFileStem derives the filestem for the target filename.
// It strips any "{shortPlanID}-{type}-" prefix from the source filename stem,
// then falls back to the title slug from the record, then the raw stem.
func extractFileStem(srcPath, docType string, record *service.DocumentResult) string {
	base := filepath.Base(srcPath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	if stripped := stripTypePrefix(stem); stripped != stem {
		return stripped
	}

	if record != nil && record.Title != "" {
		return toSlug(record.Title)
	}

	return stem
}

// stripTypePrefix removes a "{shortPlanID}-{type}-" prefix from a filename stem
// where shortPlanID is a letter followed by digits (e.g. "P37") and type is one
// of the known document type keywords. Returns the original stem if no match.
func stripTypePrefix(stem string) string {
	knownTypes := []string{
		"dev-plan", "design", "spec", "research", "report", "retro", "plan", "review",
	}
	parts := strings.Split(stem, "-")
	if len(parts) < 3 {
		return stem
	}

	// parts[0] must look like a short plan ID: non-digit followed only by digits
	first := parts[0]
	if len(first) < 2 {
		return stem
	}
	runes := []rune(first)
	if unicode.IsDigit(runes[0]) {
		return stem
	}
	for _, r := range runes[1:] {
		if !unicode.IsDigit(r) {
			return stem
		}
	}

	// Try to match a known type starting at parts[1]
	for _, t := range knownTypes {
		typeParts := strings.Split(t, "-")
		switch len(typeParts) {
		case 1:
			// Single-word type: parts[1] == type, need at least one more part
			if len(parts) > 2 && strings.EqualFold(parts[1], t) {
				return strings.Join(parts[2:], "-")
			}
		case 2:
			// Two-word type (e.g. "dev-plan"): parts[1]+parts[2] == type words
			if len(parts) > 3 &&
				strings.EqualFold(parts[1], typeParts[0]) &&
				strings.EqualFold(parts[2], typeParts[1]) {
				return strings.Join(parts[3:], "-")
			}
		}
	}

	return stem
}

// toSlug converts a title to a lowercase hyphen-separated slug.
func toSlug(title string) string {
	lower := strings.ToLower(title)
	var b strings.Builder
	prevHyphen := false
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen {
			b.WriteRune('-')
			prevHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// isMode2Arg returns true if arg matches the P{n}-F{m} pattern used by Mode 2.
func isMode2Arg(arg string) bool {
	if strings.Contains(arg, "/") || strings.Contains(arg, ".") {
		return false
	}
	if !model.IsPlanID(arg) {
		return false
	}
	_, _, slug := model.ParsePlanID(arg)
	if len(slug) < 2 || slug[0] != 'F' {
		return false
	}
	for _, c := range slug[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
