package merge

import (
	"errors"
	"testing"

	"github.com/sambeau/kanbanzai/internal/git"
)

func TestTasksCompleteGate_Interface(t *testing.T) {
	var g Gate = TasksCompleteGate{}

	if g.Name() != "tasks_complete" {
		t.Errorf("Name: got %q, want %q", g.Name(), "tasks_complete")
	}
	if g.Severity() != GateSeverityBlocking {
		t.Errorf("Severity: got %q, want %q", g.Severity(), GateSeverityBlocking)
	}
}

func TestTasksCompleteGate_Check(t *testing.T) {
	tests := []struct {
		name       string
		tasks      []map[string]any
		wantStatus GateStatus
		wantMsg    string
	}{
		{
			name:       "no tasks passes",
			tasks:      nil,
			wantStatus: GateStatusPassed,
		},
		{
			name:       "empty tasks passes",
			tasks:      []map[string]any{},
			wantStatus: GateStatusPassed,
		},
		{
			name: "all done passes",
			tasks: []map[string]any{
				{"id": "TASK-001", "status": "done"},
				{"id": "TASK-002", "status": "done"},
			},
			wantStatus: GateStatusPassed,
		},
		{
			name: "all wont_do passes",
			tasks: []map[string]any{
				{"id": "TASK-001", "status": "wont_do"},
			},
			wantStatus: GateStatusPassed,
		},
		{
			name: "mixed done and wont_do passes",
			tasks: []map[string]any{
				{"id": "TASK-001", "status": "done"},
				{"id": "TASK-002", "status": "wont_do"},
			},
			wantStatus: GateStatusPassed,
		},
		{
			name: "one incomplete fails",
			tasks: []map[string]any{
				{"id": "TASK-001", "status": "done"},
				{"id": "TASK-002", "status": "active"},
			},
			wantStatus: GateStatusFailed,
			wantMsg:    "task not complete: TASK-002",
		},
		{
			name: "multiple incomplete fails",
			tasks: []map[string]any{
				{"id": "TASK-001", "status": "queued"},
				{"id": "TASK-002", "status": "active"},
				{"id": "TASK-003", "status": "done"},
			},
			wantStatus: GateStatusFailed,
			wantMsg:    "2 tasks not complete: TASK-001, TASK-002",
		},
		{
			name: "blocked task fails",
			tasks: []map[string]any{
				{"id": "TASK-001", "status": "blocked"},
			},
			wantStatus: GateStatusFailed,
			wantMsg:    "task not complete: TASK-001",
		},
		{
			name: "needs-review task fails",
			tasks: []map[string]any{
				{"id": "TASK-001", "status": "needs-review"},
			},
			wantStatus: GateStatusFailed,
			wantMsg:    "task not complete: TASK-001",
		},
		{
			name: "task without id shows unknown",
			tasks: []map[string]any{
				{"status": "active"},
			},
			wantStatus: GateStatusFailed,
			wantMsg:    "task not complete: (unknown)",
		},
	}

	gate := TasksCompleteGate{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := GateContext{
				EntityID: "FEAT-001",
				Tasks:    tt.tasks,
			}

			result := gate.Check(ctx)

			if result.Status != tt.wantStatus {
				t.Errorf("Status: got %q, want %q", result.Status, tt.wantStatus)
			}
			if tt.wantMsg != "" && result.Message != tt.wantMsg {
				t.Errorf("Message: got %q, want %q", result.Message, tt.wantMsg)
			}
			if result.Name != "tasks_complete" {
				t.Errorf("Name: got %q, want %q", result.Name, "tasks_complete")
			}
			if result.Severity != GateSeverityBlocking {
				t.Errorf("Severity: got %q, want %q", result.Severity, GateSeverityBlocking)
			}
		})
	}
}

func TestVerificationExistsGate_Interface(t *testing.T) {
	var g Gate = VerificationExistsGate{}

	if g.Name() != "verification_exists" {
		t.Errorf("Name: got %q, want %q", g.Name(), "verification_exists")
	}
	if g.Severity() != GateSeverityBlocking {
		t.Errorf("Severity: got %q, want %q", g.Severity(), GateSeverityBlocking)
	}
}

func TestVerificationExistsGate_Check(t *testing.T) {
	tests := []struct {
		name       string
		entity     map[string]any
		wantStatus GateStatus
		wantMsg    string
	}{
		{
			name:       "with verification passes",
			entity:     map[string]any{"verification": "All tests pass, manual QA complete"},
			wantStatus: GateStatusPassed,
		},
		{
			name:       "empty verification fails",
			entity:     map[string]any{"verification": ""},
			wantStatus: GateStatusFailed,
			wantMsg:    "verification field is empty",
		},
		{
			name:       "whitespace-only verification fails",
			entity:     map[string]any{"verification": "   "},
			wantStatus: GateStatusFailed,
			wantMsg:    "verification field is empty",
		},
		{
			name:       "missing verification fails",
			entity:     map[string]any{},
			wantStatus: GateStatusFailed,
			wantMsg:    "verification field is empty",
		},
		{
			name:       "nil entity fails",
			entity:     nil,
			wantStatus: GateStatusFailed,
			wantMsg:    "verification field is empty",
		},
	}

	gate := VerificationExistsGate{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := GateContext{
				EntityID: "FEAT-001",
				Entity:   tt.entity,
			}

			result := gate.Check(ctx)

			if result.Status != tt.wantStatus {
				t.Errorf("Status: got %q, want %q", result.Status, tt.wantStatus)
			}
			if tt.wantMsg != "" && result.Message != tt.wantMsg {
				t.Errorf("Message: got %q, want %q", result.Message, tt.wantMsg)
			}
		})
	}
}

func TestVerificationPassedGate_Interface(t *testing.T) {
	var g Gate = VerificationPassedGate{}

	if g.Name() != "verification_passed" {
		t.Errorf("Name: got %q, want %q", g.Name(), "verification_passed")
	}
	if g.Severity() != GateSeverityBlocking {
		t.Errorf("Severity: got %q, want %q", g.Severity(), GateSeverityBlocking)
	}
}

func TestVerificationPassedGate_Check(t *testing.T) {
	tests := []struct {
		name       string
		entity     map[string]any
		wantStatus GateStatus
		wantMsg    string
	}{
		{
			name:       "status passed passes",
			entity:     map[string]any{"verification_status": "passed"},
			wantStatus: GateStatusPassed,
		},
		{
			name:       "status failed fails",
			entity:     map[string]any{"verification_status": "failed"},
			wantStatus: GateStatusFailed,
			wantMsg:    `verification_status is "failed", expected "passed"`,
		},
		{
			name:       "status pending fails",
			entity:     map[string]any{"verification_status": "pending"},
			wantStatus: GateStatusFailed,
			wantMsg:    `verification_status is "pending", expected "passed"`,
		},
		{
			name:       "empty status fails",
			entity:     map[string]any{"verification_status": ""},
			wantStatus: GateStatusFailed,
			wantMsg:    "verification_status not set",
		},
		{
			name:       "missing status fails",
			entity:     map[string]any{},
			wantStatus: GateStatusFailed,
			wantMsg:    "verification_status not set",
		},
	}

	gate := VerificationPassedGate{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := GateContext{
				EntityID: "FEAT-001",
				Entity:   tt.entity,
			}

			result := gate.Check(ctx)

			if result.Status != tt.wantStatus {
				t.Errorf("Status: got %q, want %q", result.Status, tt.wantStatus)
			}
			if tt.wantMsg != "" && result.Message != tt.wantMsg {
				t.Errorf("Message: got %q, want %q", result.Message, tt.wantMsg)
			}
		})
	}
}

func TestBranchNotStaleGate_Interface(t *testing.T) {
	var g Gate = BranchNotStaleGate{}

	if g.Name() != "branch_not_stale" {
		t.Errorf("Name: got %q, want %q", g.Name(), "branch_not_stale")
	}
	if g.Severity() != GateSeverityWarning {
		t.Errorf("Severity: got %q, want %q", g.Severity(), GateSeverityWarning)
	}
}

func TestBranchNotStaleGate_Check(t *testing.T) {
	tests := []struct {
		name        string
		branch      string
		repoPath    string
		mockStatus  git.BranchStatus
		mockErr     error
		wantStatus  GateStatus
		wantMsgPart string
	}{
		{
			name:       "healthy branch passes",
			branch:     "feature/FEAT-001",
			repoPath:   "/repo",
			mockStatus: git.BranchStatus{},
			wantStatus: GateStatusPassed,
		},
		{
			name:     "branch with warnings warns",
			branch:   "feature/FEAT-001",
			repoPath: "/repo",
			mockStatus: git.BranchStatus{
				Warnings: []string{"branch is stale: no commits in 20 days"},
			},
			wantStatus:  GateStatusWarning,
			wantMsgPart: "stale",
		},
		{
			name:     "branch with errors warns",
			branch:   "feature/FEAT-001",
			repoPath: "/repo",
			mockStatus: git.BranchStatus{
				Errors: []string{"branch has critical drift: 150 commits behind main"},
			},
			wantStatus:  GateStatusWarning,
			wantMsgPart: "drift",
		},
		{
			name:     "branch with warnings and errors",
			branch:   "feature/FEAT-001",
			repoPath: "/repo",
			mockStatus: git.BranchStatus{
				Warnings: []string{"branch is stale"},
				Errors:   []string{"critical drift"},
			},
			wantStatus:  GateStatusWarning,
			wantMsgPart: "stale",
		},
		{
			name:        "no branch warns",
			branch:      "",
			repoPath:    "/repo",
			wantStatus:  GateStatusWarning,
			wantMsgPart: "no branch specified",
		},
		{
			name:        "no repo path warns",
			branch:      "feature/FEAT-001",
			repoPath:    "",
			wantStatus:  GateStatusWarning,
			wantMsgPart: "no repository path specified",
		},
		{
			name:        "evaluator error warns",
			branch:      "feature/FEAT-001",
			repoPath:    "/repo",
			mockErr:     errors.New("git failed"),
			wantStatus:  GateStatusWarning,
			wantMsgPart: "cannot evaluate branch status",
		},
	}

	gate := BranchNotStaleGate{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := GateContext{
				EntityID: "FEAT-001",
				Branch:   tt.branch,
				RepoPath: tt.repoPath,
				BranchStatusChecker: func(repoPath, branch string, thresholds git.BranchThresholds) (git.BranchStatus, error) {
					return tt.mockStatus, tt.mockErr
				},
			}

			result := gate.Check(ctx)

			if result.Status != tt.wantStatus {
				t.Errorf("Status: got %q, want %q", result.Status, tt.wantStatus)
			}
			if tt.wantMsgPart != "" && result.Message == "" {
				t.Errorf("Message: expected to contain %q, got empty", tt.wantMsgPart)
			}
			if result.Severity != GateSeverityWarning {
				t.Errorf("Severity: got %q, want %q", result.Severity, GateSeverityWarning)
			}
		})
	}
}

func TestNoConflictsGate_Interface(t *testing.T) {
	var g Gate = NoConflictsGate{}

	if g.Name() != "no_conflicts" {
		t.Errorf("Name: got %q, want %q", g.Name(), "no_conflicts")
	}
	if g.Severity() != GateSeverityBlocking {
		t.Errorf("Severity: got %q, want %q", g.Severity(), GateSeverityBlocking)
	}
}

func TestNoConflictsGate_Check(t *testing.T) {
	tests := []struct {
		name          string
		branch        string
		repoPath      string
		mockConflicts bool
		mockErr       error
		wantStatus    GateStatus
		wantMsgPart   string
	}{
		{
			name:          "no conflicts passes",
			branch:        "feature/FEAT-001",
			repoPath:      "/repo",
			mockConflicts: false,
			wantStatus:    GateStatusPassed,
		},
		{
			name:          "conflicts fails",
			branch:        "feature/FEAT-001",
			repoPath:      "/repo",
			mockConflicts: true,
			wantStatus:    GateStatusFailed,
			wantMsgPart:   "merge conflicts",
		},
		{
			name:        "no branch fails",
			branch:      "",
			repoPath:    "/repo",
			wantStatus:  GateStatusFailed,
			wantMsgPart: "no branch specified",
		},
		{
			name:        "no repo path fails",
			branch:      "feature/FEAT-001",
			repoPath:    "",
			wantStatus:  GateStatusFailed,
			wantMsgPart: "no repository path specified",
		},
		{
			name:        "checker error fails",
			branch:      "feature/FEAT-001",
			repoPath:    "/repo",
			mockErr:     errors.New("git failed"),
			wantStatus:  GateStatusFailed,
			wantMsgPart: "cannot check for conflicts",
		},
	}

	gate := NoConflictsGate{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := GateContext{
				EntityID: "FEAT-001",
				Branch:   tt.branch,
				RepoPath: tt.repoPath,
				ConflictChecker: func(repoPath, branch, base string) (bool, error) {
					return tt.mockConflicts, tt.mockErr
				},
				DefaultBranchDetector: func(repoPath string) (string, error) {
					return "main", nil
				},
			}

			result := gate.Check(ctx)

			if result.Status != tt.wantStatus {
				t.Errorf("Status: got %q, want %q", result.Status, tt.wantStatus)
			}
			if tt.wantMsgPart != "" && result.Message == "" {
				t.Errorf("Message: expected to contain %q, got empty", tt.wantMsgPart)
			}
			if result.Severity != GateSeverityBlocking {
				t.Errorf("Severity: got %q, want %q", result.Severity, GateSeverityBlocking)
			}
		})
	}
}

func TestNoConflictsGate_UsesDefaultBranchDetector(t *testing.T) {
	var checkedBase string
	ctx := GateContext{
		EntityID: "FEAT-001",
		Branch:   "feature/FEAT-001",
		RepoPath: "/repo",
		DefaultBranchDetector: func(repoPath string) (string, error) {
			return "develop", nil
		},
		ConflictChecker: func(repoPath, branch, base string) (bool, error) {
			checkedBase = base
			return false, nil
		},
	}

	gate := NoConflictsGate{}
	result := gate.Check(ctx)

	if result.Status != GateStatusPassed {
		t.Errorf("Status: got %q, want %q", result.Status, GateStatusPassed)
	}
	if checkedBase != "develop" {
		t.Errorf("Expected conflict check against %q, got %q", "develop", checkedBase)
	}
}

func TestNoConflictsGate_DefaultBranchDetectorError(t *testing.T) {
	ctx := GateContext{
		EntityID: "FEAT-001",
		Branch:   "feature/FEAT-001",
		RepoPath: "/repo",
		DefaultBranchDetector: func(repoPath string) (string, error) {
			return "", errors.New("no default branch found")
		},
		ConflictChecker: func(repoPath, branch, base string) (bool, error) {
			t.Fatal("ConflictChecker should not be called when DefaultBranchDetector fails")
			return false, nil
		},
	}

	gate := NoConflictsGate{}
	result := gate.Check(ctx)

	if result.Status != GateStatusFailed {
		t.Errorf("Status: got %q, want %q", result.Status, GateStatusFailed)
	}
	if result.Message == "" {
		t.Error("Expected error message about default branch detection")
	}
}

func TestHealthCheckCleanGate_Interface(t *testing.T) {
	var g Gate = HealthCheckCleanGate{}

	if g.Name() != "health_check_clean" {
		t.Errorf("Name: got %q, want %q", g.Name(), "health_check_clean")
	}
	if g.Severity() != GateSeverityBlocking {
		t.Errorf("Severity: got %q, want %q", g.Severity(), GateSeverityBlocking)
	}
}

func TestHealthCheckCleanGate_Check(t *testing.T) {
	gate := HealthCheckCleanGate{}
	ctx := GateContext{
		EntityID: "FEAT-001",
	}

	result := gate.Check(ctx)

	// Placeholder always passes
	if result.Status != GateStatusPassed {
		t.Errorf("Status: got %q, want %q (placeholder should pass)", result.Status, GateStatusPassed)
	}
	if result.Message != "" {
		t.Errorf("Message: got %q, want empty", result.Message)
	}
}

func TestEntityDoneGate_Interface(t *testing.T) {
	var g Gate = EntityDoneGate{}

	if g.Name() != "entity_done" {
		t.Errorf("Name: got %q, want %q", g.Name(), "entity_done")
	}
	if g.Severity() != GateSeverityBlocking {
		t.Errorf("Severity: got %q, want %q", g.Severity(), GateSeverityBlocking)
	}
}

func TestEntityDoneGate_Check(t *testing.T) {
	tests := []struct {
		name       string
		entityID   string
		entity     map[string]any
		wantStatus GateStatus
		wantMsg    string
	}{
		{
			name:       "feature done passes",
			entityID:   "FEAT-001",
			entity:     map[string]any{"status": "done"},
			wantStatus: GateStatusPassed,
		},
		{
			name:       "feature reviewing fails",
			entityID:   "FEAT-001",
			entity:     map[string]any{"status": "reviewing"},
			wantStatus: GateStatusFailed,
			wantMsg:    `feature status is "reviewing", expected "done"`,
		},
		{
			name:       "feature needs-rework fails",
			entityID:   "FEAT-001",
			entity:     map[string]any{"status": "needs-rework"},
			wantStatus: GateStatusFailed,
			wantMsg:    `feature status is "needs-rework", expected "done"`,
		},
		{
			name:       "feature developing fails",
			entityID:   "FEAT-001",
			entity:     map[string]any{"status": "developing"},
			wantStatus: GateStatusFailed,
			wantMsg:    `feature status is "developing", expected "done"`,
		},
		{
			name:       "feature proposed fails",
			entityID:   "FEAT-001",
			entity:     map[string]any{"status": "proposed"},
			wantStatus: GateStatusFailed,
			wantMsg:    `feature status is "proposed", expected "done"`,
		},
		{
			name:       "feature empty status fails",
			entityID:   "FEAT-001",
			entity:     map[string]any{},
			wantStatus: GateStatusFailed,
			wantMsg:    "feature status not set",
		},
		{
			name:       "feature nil entity fails",
			entityID:   "FEAT-001",
			entity:     nil,
			wantStatus: GateStatusFailed,
			wantMsg:    "feature status not set",
		},
		{
			name:       "bug closed passes",
			entityID:   "BUG-001",
			entity:     map[string]any{"status": "closed"},
			wantStatus: GateStatusPassed,
		},
		{
			name:       "bug in-progress fails",
			entityID:   "BUG-001",
			entity:     map[string]any{"status": "in-progress"},
			wantStatus: GateStatusFailed,
			wantMsg:    `bug status is "in-progress", expected "closed"`,
		},
		{
			name:       "bug needs-rework fails",
			entityID:   "BUG-001",
			entity:     map[string]any{"status": "needs-rework"},
			wantStatus: GateStatusFailed,
			wantMsg:    `bug status is "needs-rework", expected "closed"`,
		},
		{
			name:       "bug empty status fails",
			entityID:   "BUG-001",
			entity:     map[string]any{},
			wantStatus: GateStatusFailed,
			wantMsg:    "bug status not set",
		},
		{
			name:       "unknown entity type passes unconditionally",
			entityID:   "TASK-001",
			entity:     map[string]any{"status": "active"},
			wantStatus: GateStatusPassed,
		},
	}

	gate := EntityDoneGate{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := GateContext{
				EntityID: tt.entityID,
				Entity:   tt.entity,
			}

			result := gate.Check(ctx)

			if result.Status != tt.wantStatus {
				t.Errorf("Status: got %q, want %q", result.Status, tt.wantStatus)
			}
			if tt.wantMsg != "" && result.Message != tt.wantMsg {
				t.Errorf("Message: got %q, want %q", result.Message, tt.wantMsg)
			}
			if result.Name != "entity_done" {
				t.Errorf("Name: got %q, want %q", result.Name, "entity_done")
			}
			if result.Severity != GateSeverityBlocking {
				t.Errorf("Severity: got %q, want %q", result.Severity, GateSeverityBlocking)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"empty string", "", ""},
		{"int", 42, "42"},
		{"bool", true, "true"},
		{"float", 3.14, "3.14"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toString(tt.input)
			if got != tt.want {
				t.Errorf("toString(%v): got %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestVerificationPassedGate_Partial verifies that VerificationPassedGate returns
// GateStatusWarning (non-blocking) when verification_status is "partial" (FR-009).
func TestVerificationPassedGate_Partial(t *testing.T) {
	gate := VerificationPassedGate{}
	ctx := GateContext{
		EntityID: "FEAT-001",
		Entity:   map[string]any{"verification_status": "partial"},
	}

	result := gate.Check(ctx)

	if result.Status != GateStatusWarning {
		t.Errorf("Status: got %q, want GateStatusWarning (partial must not block merge)", result.Status)
	}
	wantMsg := "verification_status is \"partial\", expected \"passed\""
	if result.Message != wantMsg {
		t.Errorf("Message: got %q, want %q", result.Message, wantMsg)
	}
}

// TestVerificationPassedGate_None verifies that VerificationPassedGate returns
// GateStatusFailed when verification_status is "none" (FR-009).
func TestVerificationPassedGate_None(t *testing.T) {
	gate := VerificationPassedGate{}
	ctx := GateContext{
		EntityID: "FEAT-001",
		Entity:   map[string]any{"verification_status": "none"},
	}

	result := gate.Check(ctx)

	if result.Status != GateStatusFailed {
		t.Errorf("Status: got %q, want GateStatusFailed", result.Status)
	}
	wantMsg := "verification_status is \"none\", expected \"passed\""
	if result.Message != wantMsg {
		t.Errorf("Message: got %q, want %q", result.Message, wantMsg)
	}
}
