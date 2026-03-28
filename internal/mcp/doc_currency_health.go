package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"kanbanzai/internal/service"
	"kanbanzai/internal/validate"
)

// excludedNames is the static set of identifiers that appear in backticks in
// documentation but are not MCP tool names. This filters out CLI tools,
// project terms, format names, and Go keywords/builtins.
var excludedNames = map[string]bool{
	// Common CLI tools
	"go":     true,
	"git":    true,
	"grep":   true,
	"cat":    true,
	"find":   true,
	"sed":    true,
	"make":   true,
	"shasum": true,
	"cd":     true,
	"rm":     true,
	"ls":     true,
	"cp":     true,
	"mv":     true,
	"mkdir":  true,
	"chmod":  true,
	"echo":   true,
	"diff":   true,
	"curl":   true,
	"touch":  true,

	// Project-specific terms
	"kbz":       true,
	"kanbanzai": true,
	"goimports": true,
	"go_fmt":    true,
	"go_vet":    true,
	"go_test":   true,

	// Format identifiers
	"yaml": true,
	"json": true,
	"utf":  true,
	"lf":   true,

	// Go keywords and common identifiers that appear in docs
	"true":      true,
	"false":     true,
	"nil":       true,
	"err":       true,
	"ctx":       true,
	"fmt":       true,
	"string":    true,
	"bool":      true,
	"int":       true,
	"any":       true,
	"func":      true,
	"type":      true,
	"map":       true,
	"var":       true,
	"const":     true,
	"import":    true,
	"package":   true,
	"return":    true,
	"if":        true,
	"for":       true,
	"range":     true,
	"switch":    true,
	"case":      true,
	"default":   true,
	"select":    true,
	"defer":     true,
	"chan":      true,
	"struct":    true,
	"interface": true,
}

// Regex patterns for extracting tool name candidates from markdown.
var (
	// Matches `tool_name(` or `tool_name` in backtick-wrapped text.
	backtickCallRe = regexp.MustCompile("`([a-z][a-z0-9_]+)\\(`")
	backtickNameRe = regexp.MustCompile("`([a-z][a-z0-9_]+)`")
	// Matches tool(action: ...) — the MCP action invocation syntax.
	actionInvokeRe = regexp.MustCompile(`([a-z][a-z0-9_]+)\(action:`)
)

// DocCurrencyHealthChecker returns an AdditionalHealthChecker that detects
// stale references in agent-facing documentation. It has two tiers:
//
//   - Tier 1: tool name validation — scans .skills/*.md and AGENTS.md for
//     references to tools that no longer exist in the MCP registry.
//   - Tier 2: plan completion documentation — checks that done plans are
//     mentioned in AGENTS.md and that associated specs are approved.
func DocCurrencyHealthChecker(
	toolNames map[string]bool,
	repoRoot string,
	entitySvc *service.EntityService,
	docSvc *service.DocumentService,
) AdditionalHealthChecker {
	return func() (*validate.HealthReport, error) {
		report := &validate.HealthReport{
			Summary: validate.HealthSummary{
				EntitiesByType: make(map[string]int),
			},
		}

		// ── Tier 1: Tool Name Validation ────────────────────────────

		tier1Files := collectTier1Files(repoRoot)
		for _, relPath := range tier1Files {
			content, err := os.ReadFile(filepath.Join(repoRoot, relPath))
			if err != nil {
				continue // best-effort
			}
			stale := extractStaleToolNames(string(content), toolNames)
			for _, name := range stale {
				report.Warnings = append(report.Warnings, validate.ValidationWarning{
					EntityType: "doc_currency",
					Message:    fmt.Sprintf("stale tool reference %q in %s", name, relPath),
				})
				report.Summary.WarningCount++
			}
		}

		// ── Tier 2: Plan Completion Documentation ───────────────────

		if entitySvc == nil {
			return report, nil
		}

		plans, err := entitySvc.ListPlans(service.PlanFilters{})
		if err != nil {
			// No plans directory is not an error — just skip.
			return report, nil
		}

		// Read AGENTS.md once for all plan checks.
		agentsPath := filepath.Join(repoRoot, "AGENTS.md")
		agentsContent, agentsErr := os.ReadFile(agentsPath)
		agentsText := ""
		if agentsErr == nil {
			agentsText = string(agentsContent)
		}

		projectStatusSection := extractSection(agentsText, "Project Status")
		scopeGuardSection := extractSection(agentsText, "Scope Guard")

		for _, plan := range plans {
			status, _ := plan.State["status"].(string)
			if status != "done" {
				continue
			}

			planID := plan.ID
			slug := plan.Slug

			// Extract the plan ID prefix (e.g. "P9" from "P9-my-plan").
			prefix := planIDPrefix(planID)

			// Check 1: Project Status mentions the plan slug.
			if agentsErr == nil && slug != "" && !strings.Contains(projectStatusSection, slug) {
				report.Warnings = append(report.Warnings, validate.ValidationWarning{
					EntityType: "doc_currency",
					EntityID:   planID,
					Message:    fmt.Sprintf("plan %q is done but not mentioned in AGENTS.md Project Status", planID),
				})
				report.Summary.WarningCount++
			}

			// Check 2: Scope Guard mentions the plan slug or prefix.
			if agentsErr == nil && slug != "" {
				mentioned := strings.Contains(scopeGuardSection, slug) ||
					(prefix != "" && strings.Contains(scopeGuardSection, prefix))
				if !mentioned {
					report.Warnings = append(report.Warnings, validate.ValidationWarning{
						EntityType: "doc_currency",
						EntityID:   planID,
						Message:    fmt.Sprintf("plan %q is done but not mentioned in AGENTS.md Scope Guard", planID),
					})
					report.Summary.WarningCount++
				}
			}

			// Check 3: Spec documents for done features under this plan.
			features, err := entitySvc.ListEntitiesFiltered(service.ListFilteredInput{
				Type:   "feature",
				Parent: planID,
				Status: "done",
			})
			if err != nil {
				continue
			}

			if docSvc == nil {
				continue
			}
			for _, feat := range features {
				docs, err := docSvc.ListDocumentsByOwner(feat.ID)
				if err != nil {
					continue
				}
				for _, doc := range docs {
					if doc.Type == "specification" && doc.Status == "draft" {
						report.Warnings = append(report.Warnings, validate.ValidationWarning{
							EntityType: "doc_currency",
							EntityID:   doc.ID,
							Message:    fmt.Sprintf("spec document %q is still in draft status but plan %q is done", doc.ID, planID),
						})
						report.Summary.WarningCount++
					}
				}
			}
		}

		return report, nil
	}
}

// collectTier1Files returns relative paths of files to scan for tool names:
// all .md files under .skills/ and AGENTS.md at the repo root.
func collectTier1Files(repoRoot string) []string {
	var files []string

	// .skills/*.md
	skillsDir := filepath.Join(repoRoot, ".skills")
	entries, err := os.ReadDir(skillsDir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				files = append(files, filepath.Join(".skills", e.Name()))
			}
		}
	}

	// AGENTS.md
	if _, err := os.Stat(filepath.Join(repoRoot, "AGENTS.md")); err == nil {
		files = append(files, "AGENTS.md")
	}

	return files
}

// extractStaleToolNames scans content for candidate tool names and returns
// those that are not in the known tool set and not excluded.
func extractStaleToolNames(content string, toolNames map[string]bool) []string {
	seen := make(map[string]bool)
	var stale []string

	// Helper to check and collect a candidate.
	check := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		if len(name) < 2 {
			return
		}
		if toolNames[name] {
			return
		}
		if excludedNames[name] {
			return
		}
		stale = append(stale, name)
	}

	// Pattern 1: `tool_name(` — backtick-wrapped call syntax.
	for _, m := range backtickCallRe.FindAllStringSubmatch(content, -1) {
		check(m[1])
	}

	// Pattern 2: `tool_name` — backtick-wrapped bare name.
	for _, m := range backtickNameRe.FindAllStringSubmatch(content, -1) {
		check(m[1])
	}

	// Pattern 3: tool(action: ...) — MCP action invocation syntax.
	for _, m := range actionInvokeRe.FindAllStringSubmatch(content, -1) {
		check(m[1])
	}

	return stale
}

// extractSection extracts the content of a markdown section by heading name.
// It finds the first heading containing the given name and returns all content
// up to the next heading of equal or lesser depth.
func extractSection(content, heading string) string {
	lines := strings.Split(content, "\n")
	var sectionLines []string
	inSection := false
	sectionDepth := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			depth := 0
			for _, ch := range line {
				if ch == '#' {
					depth++
				} else {
					break
				}
			}
			if inSection {
				// A heading of equal or lesser depth ends the section.
				if depth <= sectionDepth {
					break
				}
			} else if strings.Contains(line, heading) {
				inSection = true
				sectionDepth = depth
				continue
			}
		}
		if inSection {
			sectionLines = append(sectionLines, line)
		}
	}

	return strings.Join(sectionLines, "\n")
}

// planIDPrefix extracts the prefix portion of a plan ID (e.g. "P9" from
// "P9-my-plan").
func planIDPrefix(planID string) string {
	idx := strings.Index(planID, "-")
	if idx > 0 {
		return planID[:idx]
	}
	return ""
}
