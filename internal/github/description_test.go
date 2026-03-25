package github

import (
	"strings"
	"testing"
)

func TestGenerateDescription(t *testing.T) {
	tests := []struct {
		name     string
		data     DescriptionData
		contains []string
		excludes []string
	}{
		{
			name: "Full description with all fields",
			data: DescriptionData{
				EntityID:          "FEAT-01JX",
				EntityTitle:       "Add user authentication",
				EntityDescription: "Implement OAuth2 authentication flow for the application.",
				EntityType:        "feature",
				Tasks: []TaskData{
					{ID: "TASK-01JX.1", Title: "Set up OAuth provider", Status: "done"},
					{ID: "TASK-01JX.2", Title: "Implement login flow", Status: "in_progress"},
					{ID: "TASK-01JX.3", Title: "Add logout endpoint", Status: "ready"},
				},
				Verification:       "All authentication flows work end-to-end",
				VerificationStatus: "pending",
				Created:            "2024-01-15",
				Branch:             "feat/user-auth",
			},
			contains: []string{
				"## Add user authentication",
				"Implement OAuth2 authentication flow",
				"### Tasks",
				"- [x] TASK-01JX.1: Set up OAuth provider (done)",
				"- [ ] TASK-01JX.2: Implement login flow (in progress)",
				"- [ ] TASK-01JX.3: Add logout endpoint (ready)",
				"### Verification",
				"All authentication flows work end-to-end",
				"**Status:** pending",
				"### Workflow",
				"**Entity:** FEAT-01JX",
				"**Created:** 2024-01-15",
				"**Branch:** feat/user-auth",
				"*This description is managed by Kanbanzai",
			},
		},
		{
			name: "Minimal description",
			data: DescriptionData{
				EntityID:    "BUG-01AB",
				EntityTitle: "Fix login crash",
			},
			contains: []string{
				"## Fix login crash",
				"### Workflow",
				"**Entity:** BUG-01AB",
				"*This description is managed by Kanbanzai",
			},
			excludes: []string{
				"### Tasks",
				"### Verification",
				"**Status:**",
				"**Created:**",
				"**Branch:**",
			},
		},
		{
			name: "Description without tasks",
			data: DescriptionData{
				EntityID:          "FEAT-02CD",
				EntityTitle:       "Add export feature",
				EntityDescription: "Allow users to export data as CSV.",
				Created:           "2024-02-01",
				Branch:            "feat/export",
			},
			contains: []string{
				"## Add export feature",
				"Allow users to export data as CSV.",
				"**Created:** 2024-02-01",
				"**Branch:** feat/export",
			},
			excludes: []string{
				"### Tasks",
				"- [",
			},
		},
		{
			name: "Description with only verification",
			data: DescriptionData{
				EntityID:           "BUG-03EF",
				EntityTitle:        "Fix memory leak",
				Verification:       "Memory usage remains stable under load",
				VerificationStatus: "passed",
			},
			contains: []string{
				"## Fix memory leak",
				"### Verification",
				"Memory usage remains stable under load",
				"**Status:** passed",
			},
		},
		{
			name: "All tasks complete",
			data: DescriptionData{
				EntityID:    "FEAT-04GH",
				EntityTitle: "Complete feature",
				Tasks: []TaskData{
					{ID: "TASK-04GH.1", Title: "First task", Status: "done"},
					{ID: "TASK-04GH.2", Title: "Second task", Status: "complete"},
					{ID: "TASK-04GH.3", Title: "Third task", Status: "completed"},
				},
			},
			contains: []string{
				"- [x] TASK-04GH.1: First task (done)",
				"- [x] TASK-04GH.2: Second task (complete)",
				"- [x] TASK-04GH.3: Third task (completed)",
			},
		},
		{
			name: "Mixed task statuses",
			data: DescriptionData{
				EntityID:    "FEAT-05IJ",
				EntityTitle: "Mixed tasks",
				Tasks: []TaskData{
					{ID: "TASK-05IJ.1", Title: "Ready task", Status: "ready"},
					{ID: "TASK-05IJ.2", Title: "Blocked task", Status: "blocked"},
					{ID: "TASK-05IJ.3", Title: "In progress task", Status: "in_progress"},
				},
			},
			contains: []string{
				"- [ ] TASK-05IJ.1: Ready task (ready)",
				"- [ ] TASK-05IJ.2: Blocked task (blocked)",
				"- [ ] TASK-05IJ.3: In progress task (in progress)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateDescription(tt.data)

			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("GenerateDescription() missing expected content: %q\n\nGot:\n%s", want, got)
				}
			}

			for _, notWant := range tt.excludes {
				if strings.Contains(got, notWant) {
					t.Errorf("GenerateDescription() contains unexpected content: %q\n\nGot:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestFormatTaskCheckbox(t *testing.T) {
	tests := []struct {
		name string
		task TaskData
		want string
	}{
		{
			name: "Done task",
			task: TaskData{ID: "TASK-01", Title: "Complete task", Status: "done"},
			want: "- [x] TASK-01: Complete task (done)",
		},
		{
			name: "In progress task",
			task: TaskData{ID: "TASK-02", Title: "Working task", Status: "in_progress"},
			want: "- [ ] TASK-02: Working task (in progress)",
		},
		{
			name: "Ready task",
			task: TaskData{ID: "TASK-03", Title: "Pending task", Status: "ready"},
			want: "- [ ] TASK-03: Pending task (ready)",
		},
		{
			name: "Blocked task",
			task: TaskData{ID: "TASK-04", Title: "Blocked task", Status: "blocked"},
			want: "- [ ] TASK-04: Blocked task (blocked)",
		},
		{
			name: "Complete status variant",
			task: TaskData{ID: "TASK-05", Title: "Finished task", Status: "complete"},
			want: "- [x] TASK-05: Finished task (complete)",
		},
		{
			name: "Completed status variant",
			task: TaskData{ID: "TASK-06", Title: "Finished task", Status: "completed"},
			want: "- [x] TASK-06: Finished task (completed)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTaskCheckbox(tt.task)
			if got != tt.want {
				t.Errorf("formatTaskCheckbox() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsTaskComplete(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"done", true},
		{"Done", true},
		{"DONE", true},
		{"complete", true},
		{"Complete", true},
		{"completed", true},
		{"Completed", true},
		{"  done  ", true},
		{"in_progress", false},
		{"ready", false},
		{"blocked", false},
		{"", false},
		{"doing", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := isTaskComplete(tt.status); got != tt.want {
				t.Errorf("isTaskComplete(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"in_progress", "in progress"},
		{"IN_PROGRESS", "in progress"},
		{"done", "done"},
		{"blocked", "blocked"},
		{"changes_requested", "changes requested"},
		{"  ready  ", "ready"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := formatStatus(tt.status); got != tt.want {
				t.Errorf("formatStatus(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestGenerateDescription_Footer(t *testing.T) {
	data := DescriptionData{
		EntityID:    "FEAT-01",
		EntityTitle: "Test",
	}

	got := GenerateDescription(data)

	// Should end with footer
	if !strings.HasSuffix(got, "*This description is managed by Kanbanzai. Manual edits may be overwritten.*\n") {
		t.Error("GenerateDescription() should end with the standard footer")
	}

	// Should have horizontal rule before footer
	if !strings.Contains(got, "\n---\n") {
		t.Error("GenerateDescription() should have horizontal rule before footer")
	}
}

func TestGenerateDescription_Ordering(t *testing.T) {
	data := DescriptionData{
		EntityID:           "FEAT-01",
		EntityTitle:        "Test Feature",
		EntityDescription:  "Test description",
		Tasks:              []TaskData{{ID: "TASK-01", Title: "Task", Status: "ready"}},
		Verification:       "Verify it works",
		VerificationStatus: "pending",
		Created:            "2024-01-01",
		Branch:             "main",
	}

	got := GenerateDescription(data)

	// Verify section ordering
	titleIdx := strings.Index(got, "## Test Feature")
	descIdx := strings.Index(got, "Test description")
	tasksIdx := strings.Index(got, "### Tasks")
	verifyIdx := strings.Index(got, "### Verification")
	workflowIdx := strings.Index(got, "### Workflow")
	footerIdx := strings.Index(got, "---")

	if titleIdx >= descIdx {
		t.Error("Title should come before description")
	}
	if descIdx >= tasksIdx {
		t.Error("Description should come before tasks")
	}
	if tasksIdx >= verifyIdx {
		t.Error("Tasks should come before verification")
	}
	if verifyIdx >= workflowIdx {
		t.Error("Verification should come before workflow")
	}
	if workflowIdx >= footerIdx {
		t.Error("Workflow should come before footer")
	}
}
