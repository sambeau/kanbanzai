package github

import (
	"strings"
)

// DescriptionData contains data for PR description generation.
type DescriptionData struct {
	EntityID           string     // Entity ID (e.g., "FEAT-01JX")
	EntityTitle        string     // Entity title
	EntityDescription  string     // Entity description text
	EntityType         string     // "feature" or "bug"
	Tasks              []TaskData // List of tasks for this entity
	Verification       string     // Verification criteria text
	VerificationStatus string     // Current verification status
	Created            string     // Creation date (formatted string)
	Branch             string     // Git branch name
}

// TaskData contains data for a single task in the PR description.
type TaskData struct {
	ID     string // Task ID (e.g., "TASK-01JX.1")
	Title  string // Task title
	Status string // Task status: "done", "in_progress", "blocked", "ready", etc.
}

// GenerateDescription generates a PR description from entity state.
// The output follows a standard markdown template that includes
// entity info, task checklist, verification criteria, and workflow metadata.
func GenerateDescription(data DescriptionData) string {
	var sb strings.Builder

	// Title section
	sb.WriteString("## ")
	sb.WriteString(data.EntityTitle)
	sb.WriteString("\n\n")

	// Description section
	if data.EntityDescription != "" {
		sb.WriteString(data.EntityDescription)
		sb.WriteString("\n\n")
	}

	// Tasks section
	if len(data.Tasks) > 0 {
		sb.WriteString("### Tasks\n\n")
		for _, task := range data.Tasks {
			sb.WriteString(formatTaskCheckbox(task))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Verification section
	if data.Verification != "" || data.VerificationStatus != "" {
		sb.WriteString("### Verification\n\n")
		if data.Verification != "" {
			sb.WriteString(data.Verification)
			sb.WriteString("\n\n")
		}
		if data.VerificationStatus != "" {
			sb.WriteString("**Status:** ")
			sb.WriteString(data.VerificationStatus)
			sb.WriteString("\n\n")
		}
	}

	// Workflow metadata section
	sb.WriteString("### Workflow\n\n")
	if data.EntityID != "" {
		sb.WriteString("- **Entity:** ")
		sb.WriteString(data.EntityID)
		sb.WriteString("\n")
	}
	if data.Created != "" {
		sb.WriteString("- **Created:** ")
		sb.WriteString(data.Created)
		sb.WriteString("\n")
	}
	if data.Branch != "" {
		sb.WriteString("- **Branch:** ")
		sb.WriteString(data.Branch)
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("\n---\n")
	sb.WriteString("*This description is managed by Kanbanzai. Manual edits may be overwritten.*\n")

	return sb.String()
}

// formatTaskCheckbox formats a task as a markdown checkbox item.
func formatTaskCheckbox(task TaskData) string {
	var sb strings.Builder

	// Checkbox state based on status
	if isTaskComplete(task.Status) {
		sb.WriteString("- [x] ")
	} else {
		sb.WriteString("- [ ] ")
	}

	// Task ID and title
	sb.WriteString(task.ID)
	sb.WriteString(": ")
	sb.WriteString(task.Title)

	// Status indicator in parentheses
	sb.WriteString(" (")
	sb.WriteString(formatStatus(task.Status))
	sb.WriteString(")")

	return sb.String()
}

// isTaskComplete returns true if the task status indicates completion.
func isTaskComplete(status string) bool {
	normalized := strings.ToLower(strings.TrimSpace(status))
	return normalized == "done" || normalized == "complete" || normalized == "completed"
}

// formatStatus formats a status for display.
func formatStatus(status string) string {
	// Normalize and format for display
	normalized := strings.ToLower(strings.TrimSpace(status))
	return strings.ReplaceAll(normalized, "_", " ")
}
