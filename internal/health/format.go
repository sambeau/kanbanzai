package health

import (
	"sort"
	"strconv"
)

// FormatHealthResult returns YAML-friendly structured output.
func FormatHealthResult(result HealthResult) map[string]any {
	output := map[string]any{
		"status": string(result.Status),
	}

	if len(result.Categories) == 0 {
		output["categories"] = nil
		return output
	}

	categories := make(map[string]any)

	// Get sorted category names for deterministic output
	names := make([]string, 0, len(result.Categories))
	for name := range result.Categories {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		cat := result.Categories[name]
		categories[name] = FormatCategoryResult(cat)
	}

	output["categories"] = categories
	return output
}

// FormatCategoryResult returns YAML-friendly structured output for a single category.
func FormatCategoryResult(result CategoryResult) map[string]any {
	output := map[string]any{
		"status": string(result.Status),
	}

	if len(result.Issues) == 0 {
		output["issues"] = nil
		return output
	}

	issues := make([]map[string]any, len(result.Issues))
	for i, issue := range result.Issues {
		issues[i] = FormatIssue(issue)
	}

	output["issues"] = issues
	return output
}

// FormatIssue returns YAML-friendly structured output for a single issue.
func FormatIssue(issue Issue) map[string]any {
	output := map[string]any{
		"severity": string(issue.Severity),
		"message":  issue.Message,
	}

	if issue.EntityID != "" {
		output["entity_id"] = issue.EntityID
	}

	if issue.EntryID != "" {
		output["entry_id"] = issue.EntryID
	}

	if len(issue.Entries) > 0 {
		output["entries"] = issue.Entries
	}

	return output
}

// CountIssues returns the total number of issues across all categories.
func CountIssues(result HealthResult) int {
	total := 0
	for _, cat := range result.Categories {
		total += len(cat.Issues)
	}
	return total
}

// CountBySeverity returns counts of issues by severity.
func CountBySeverity(result HealthResult) map[Severity]int {
	counts := map[Severity]int{
		SeverityOK:      0,
		SeverityWarning: 0,
		SeverityError:   0,
	}

	for _, cat := range result.Categories {
		for _, issue := range cat.Issues {
			counts[issue.Severity]++
		}
	}

	return counts
}

// Summary returns a human-readable summary of the health check result.
func Summary(result HealthResult) string {
	counts := CountBySeverity(result)
	errors := counts[SeverityError]
	warnings := counts[SeverityWarning]

	if errors == 0 && warnings == 0 {
		return "All health checks passed"
	}

	if errors == 0 {
		if warnings == 1 {
			return "1 warning found"
		}
		return formatCount(warnings) + " warnings found"
	}

	if warnings == 0 {
		if errors == 1 {
			return "1 error found"
		}
		return formatCount(errors) + " errors found"
	}

	return formatCount(errors) + " errors, " + formatCount(warnings) + " warnings found"
}

// formatCount returns a string representation of a count.
func formatCount(n int) string {
	return strconv.Itoa(n)
}
