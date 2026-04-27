package main

import (
	"bufio"
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
	// Extract --force flag before mode detection.
	force := false
	var filtered []string
	for _, arg := range args {
		if arg == "--force" {
			force = true
		} else {
			filtered = append(filtered, arg)
		}
	}
	args = filtered

	if len(args) == 0 {
		return fmt.Errorf("expected arguments\n\n%s", moveUsageText)
	}

	// Mode 2: first arg matches P{n}-F{m} pattern
	if isMode2Arg(args[0]) {
		if len(args) != 2 {
			return fmt.Errorf("expected exactly 2 arguments for Mode 2: <feature-ref> <plan-id>\n\n%s", moveUsageText)
		}
		stateRoot := core.StatePath()
		repoRoot := "."
		return runMoveFeature(args[0], args[1], force, stateRoot, repoRoot, deps)
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

// runMoveFeature implements Mode 2: re-parents a feature to a different plan.
func runMoveFeature(displayID, planArg string, force bool, stateRoot, repoRoot string, deps dependencies) error {
	entitySvc := service.NewEntityService(stateRoot)
	docSvc := service.NewDocumentService(stateRoot, repoRoot)

	// REQ-014: Resolve display ID to canonical ID and slug.
	canonicalID, slug, err := entitySvc.ResolveFeatureDisplayID(displayID)
	if err != nil {
		return fmt.Errorf("feature %q not found: %w", displayID, err)
	}

	// REQ-004: Validate target plan.
	targetPlanID, targetShortID, targetSlug, err := resolvePlanArg(planArg, stateRoot)
	if err != nil {
		return err
	}

	// Load feature entity to get current parent.
	result, err := entitySvc.Get("feature", canonicalID, slug)
	if err != nil {
		return fmt.Errorf("load feature %s: %w", canonicalID, err)
	}
	var currentParent string
	if v, ok := result.State["parent"].(string); ok {
		currentParent = v
	}

	// REQ-015: Already in target plan — nothing to do.
	if currentParent == targetPlanID {
		return fmt.Errorf("feature %q is already in plan %q — nothing to do", displayID, planArg)
	}

	// Get docs owned by this feature.
	docs, err := docSvc.ListDocuments(service.DocumentFilters{Owner: canonicalID})
	if err != nil {
		return fmt.Errorf("list documents: %w", err)
	}

	// REQ-016: Print planned changes and prompt user unless --force.
	if !force {
		fmt.Fprintf(deps.stdout, "Re-parent feature %s → %s\n", displayID, targetPlanID)
		if len(docs) > 0 {
			fmt.Fprintf(deps.stdout, "Documents to move:\n")
			for _, doc := range docs {
				newPath := buildDocTargetPath(doc.Path, doc.Type, targetShortID, targetSlug)
				fmt.Fprintf(deps.stdout, "  %s → %s\n", doc.Path, newPath)
			}
		}
		fmt.Fprintf(deps.stdout, "Proceed? [y/N]: ")
		reader := bufio.NewReader(deps.stdin)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "y" && line != "yes" {
			return nil
		}
	}

	// REQ-017: Allocate a new display ID in the target plan.
	newDisplayID, err := entitySvc.AllocateFeatureDisplayIDInPlan(targetPlanID)
	if err != nil {
		return fmt.Errorf("allocate display ID in plan %s: %w", targetPlanID, err)
	}

	// REQ-018: Update the feature entity with new parent and display ID.
	if _, err := entitySvc.UpdateEntity(service.UpdateEntityInput{
		Type: "feature",
		ID:   canonicalID,
		Slug: slug,
		Fields: map[string]string{
			"parent":     targetPlanID,
			"display_id": newDisplayID,
		},
	}); err != nil {
		return fmt.Errorf("update feature entity: %w", err)
	}

	// REQ-019: Move each document file and update its record.
	type movedDoc struct{ from, to string }
	var moved []movedDoc
	for _, doc := range docs {
		newPath := buildDocTargetPath(doc.Path, doc.Type, targetShortID, targetSlug)
		if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
			return fmt.Errorf("create directory for %s: %w", newPath, err)
		}
		if err := igit.GitMove(repoRoot, doc.Path, newPath); err != nil {
			return fmt.Errorf("git mv %s → %s: %w", doc.Path, newPath, err)
		}
		if _, err := docSvc.UpdateDocumentPathAndOwner(doc.ID, newPath, targetPlanID); err != nil {
			return fmt.Errorf("update document record %s: %w", doc.ID, err)
		}
		moved = append(moved, movedDoc{doc.Path, newPath})
	}

	// REQ-020: Print results.
	fmt.Fprintf(deps.stdout, "Moved feature %s → %s\n", displayID, newDisplayID)
	for _, m := range moved {
		fmt.Fprintf(deps.stdout, "Moved %s → %s\n", m.from, m.to)
	}

	return nil
}

// buildDocTargetPath builds the canonical destination path for a document
// being moved as part of a feature re-parent operation.
func buildDocTargetPath(oldPath, docType, targetShortID, targetPlanSlug string) string {
	ext := filepath.Ext(oldPath)
	if ext == "" {
		ext = ".md"
	}
	stem := extractFileStem(oldPath, docType, nil)
	targetFolder := filepath.Join("work", targetShortID+"-"+targetPlanSlug)
	targetFile := targetShortID + "-" + docType + "-" + stem + ext
	return filepath.Join(targetFolder, targetFile)
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
