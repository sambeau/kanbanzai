package binding

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func intPtr(n int) *int       { return &n }
func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

func validMinimalBinding() *StageBinding {
	return &StageBinding{
		Description:   "A test stage",
		Orchestration: "single-agent",
		Roles:         []string{"designer"},
		Skills:        []string{"design-skill"},
	}
}

func validFullBinding() *StageBinding {
	return &StageBinding{
		Description:   "Full stage with all fields",
		Orchestration: "orchestrator-workers",
		Roles:         []string{"orchestrator", "lead"},
		Skills:        []string{"planning", "code-review"},
		HumanGate:     true,
		DocumentType:  strPtr("specification"),
		Prerequisites: &Prerequisites{
			Documents: []DocumentPrereq{
				{Type: "design", Status: "approved"},
			},
			Tasks: &TaskPrereq{
				AllTerminal: boolPtr(true),
			},
		},
		Notes:           "Some notes",
		EffortBudget:    "medium",
		MaxReviewCycles: intPtr(3),
		SubAgents: &SubAgents{
			Roles:     []string{"backend", "frontend"},
			Skills:    []string{"implementation"},
			Topology:  "parallel",
			MaxAgents: intPtr(4),
		},
		DocumentTemplate: &DocumentTemplate{
			RequiredSections:         []string{"overview", "details"},
			CrossReferences:          []string{"design-doc"},
			AcceptanceCriteriaFormat: "checklist",
		},
	}
}

func TestValidateBinding(t *testing.T) {
	tests := []struct {
		name       string
		binding    *StageBinding
		stage      string
		wantErrors []string // substrings expected in error messages; nil means no errors
	}{
		{
			name:       "valid minimal binding",
			binding:    validMinimalBinding(),
			stage:      "designing",
			wantErrors: nil,
		},
		{
			name:       "valid full binding",
			binding:    validFullBinding(),
			stage:      "developing",
			wantErrors: nil,
		},
		{
			name: "missing description",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Description = ""
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"description must not be empty"},
		},
		{
			name: "invalid orchestration",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Orchestration = "multi-agent"
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"invalid orchestration"},
		},
		{
			name: "empty roles",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Roles = nil
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"roles must not be empty"},
		},
		{
			name: "invalid role ID format",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Roles = []string{"Invalid-Role!"}
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"invalid role ID"},
		},
		{
			name: "empty skills",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Skills = nil
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"skills must not be empty"},
		},
		{
			name: "invalid skill name format",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Skills = []string{"Bad Skill Name"}
				return b
			}(),
			stage:      "designing",
			wantErrors: []string{"invalid skill name"},
		},
		{
			name: "sub_agents with single-agent orchestration is valid",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Orchestration = "single-agent"
				b.SubAgents = &SubAgents{
					Roles:    []string{"worker"},
					Skills:   []string{"impl"},
					Topology: "parallel",
				}
				return b
			}(),
			stage:      "designing",
			wantErrors: nil,
		},
		{
			name: "orchestrator-workers without sub_agents",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Orchestration = "orchestrator-workers"
				b.SubAgents = nil
				return b
			}(),
			stage:      "developing",
			wantErrors: []string{"orchestration \"orchestrator-workers\" requires sub_agents"},
		},
		{
			name: "sub_agents missing required fields",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Orchestration = "orchestrator-workers"
				b.SubAgents = &SubAgents{
					Roles:    nil,
					Skills:   nil,
					Topology: "bogus",
				}
				return b
			}(),
			stage: "developing",
			wantErrors: []string{
				"sub_agents.roles must not be empty",
				"sub_agents.skills must not be empty",
				"sub_agents.topology must be one of",
			},
		},
		{
			name: "sub_agents max_agents less than 1",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Orchestration = "orchestrator-workers"
				b.SubAgents = &SubAgents{
					Roles:     []string{"worker"},
					Skills:    []string{"impl"},
					Topology:  "parallel",
					MaxAgents: intPtr(0),
				}
				return b
			}(),
			stage:      "developing",
			wantErrors: []string{"sub_agents.max_agents must be >= 1"},
		},
		{
			name: "prerequisites tasks with both min_count and all_terminal",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Prerequisites = &Prerequisites{
					Tasks: &TaskPrereq{
						MinCount:    intPtr(2),
						AllTerminal: boolPtr(true),
					},
				}
				return b
			}(),
			stage:      "reviewing",
			wantErrors: []string{"exactly one of min_count or all_terminal, not both"},
		},
		{
			name: "prerequisites tasks with neither min_count nor all_terminal",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Prerequisites = &Prerequisites{
					Tasks: &TaskPrereq{},
				}
				return b
			}(),
			stage:      "reviewing",
			wantErrors: []string{"exactly one of min_count or all_terminal"},
		},
		{
			name: "prerequisites tasks min_count less than 1",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.Prerequisites = &Prerequisites{
					Tasks: &TaskPrereq{
						MinCount: intPtr(0),
					},
				}
				return b
			}(),
			stage:      "reviewing",
			wantErrors: []string{"min_count must be >= 1"},
		},
		{
			name: "document_template with empty required_sections",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.DocumentTemplate = &DocumentTemplate{
					RequiredSections: nil,
				}
				return b
			}(),
			stage:      "specifying",
			wantErrors: []string{"document_template.required_sections must not be empty"},
		},
		{
			name: "max_review_cycles less than 1",
			binding: func() *StageBinding {
				b := validMinimalBinding()
				b.MaxReviewCycles = intPtr(0)
				return b
			}(),
			stage:      "reviewing",
			wantErrors: []string{"max_review_cycles must be >= 1"},
		},
		{
			name: "multiple errors accumulated",
			binding: &StageBinding{
				Description:     "",
				Orchestration:   "bogus",
				Roles:           nil,
				Skills:          nil,
				MaxReviewCycles: intPtr(0),
			},
			stage: "designing",
			wantErrors: []string{
				"description must not be empty",
				"invalid orchestration",
				"roles must not be empty",
				"skills must not be empty",
				"max_review_cycles must be >= 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateBinding(tt.binding, tt.stage)

			if tt.wantErrors == nil {
				if len(errs) != 0 {
					t.Errorf("expected no errors, got %d:", len(errs))
					for _, e := range errs {
						t.Errorf("  %s", e)
					}
				}
				return
			}

			if len(errs) < len(tt.wantErrors) {
				t.Errorf("expected at least %d errors, got %d:", len(tt.wantErrors), len(errs))
				for _, e := range errs {
					t.Errorf("  %s", e)
				}
				return
			}

			errStrings := make([]string, len(errs))
			for i, e := range errs {
				errStrings[i] = e.Error()
			}
			joined := strings.Join(errStrings, "\n")

			for _, want := range tt.wantErrors {
				if !strings.Contains(joined, want) {
					t.Errorf("expected error containing %q, got errors:\n%s", want, joined)
				}
			}

			// Verify stage name is included in every error message.
			for _, e := range errs {
				if !strings.Contains(e.Error(), tt.stage) {
					t.Errorf("error %q does not contain stage name %q", e, tt.stage)
				}
			}
		})
	}
}

func TestRoleIDRegexp(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"ab", true},
		{"designer", true},
		{"backend-dev", true},
		{"a1-b2-c3", true},
		{"a", false},
		{"-bad", false},
		{"bad-", false},
		{"BAD", false},
		{"has space", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := roleIDRegexp.MatchString(tt.input); got != tt.valid {
				t.Errorf("roleIDRegexp.MatchString(%q) = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}

func TestSkillNameRegexp(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"ab", true},
		{"design-skill", true},
		{"code-review-and-analysis", true},
		{"a", false},
		{"-bad", false},
		{"bad-", false},
		{"BAD", false},
		{"has space", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := skillNameRegexp.MatchString(tt.input); got != tt.valid {
				t.Errorf("skillNameRegexp.MatchString(%q) = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}

// TestValidStagesSync verifies that validStages contains exactly the stage keys
// defined in the canonical .kbz/stage-bindings.yaml (AC-003, AC-004, REQ-002).
func TestValidStagesSync(t *testing.T) {
	t.Parallel()

	canonicalPath := findCanonicalBindings(t)
	canonical := loadBindingFile(t, canonicalPath)

	// Verify every canonical stage is in validStages.
	for stageName := range canonical.StageBindings {
		if !validStages[stageName] {
			t.Errorf("canonical stage %q not in validStages allowlist", stageName)
		}
	}

	// Verify every validStages entry appears in the canonical file.
	for stageName := range validStages {
		if _, ok := canonical.StageBindings[stageName]; !ok {
			t.Errorf("validStages entry %q not found in canonical stage-bindings.yaml", stageName)
		}
	}

	// Explicitly verify plan-reviewing is NOT in validStages (AC-004).
	if validStages["plan-reviewing"] {
		t.Error("plan-reviewing must not be in validStages — it has been replaced by batch-reviewing")
	}
}

// TestValidOrchestrationsSync verifies that validOrchestrations includes every
// orchestration value used in the canonical stage-bindings.yaml (AC-007, REQ-004).
func TestValidOrchestrationsSync(t *testing.T) {
	t.Parallel()

	canonicalPath := findCanonicalBindings(t)
	canonical := loadBindingFile(t, canonicalPath)

	seen := map[string]bool{}
	for stageName, b := range canonical.StageBindings {
		if b.Orchestration == "" {
			continue
		}
		seen[b.Orchestration] = true
		if !validOrchestrations[b.Orchestration] {
			t.Errorf("orchestration %q (used by stage %q) not in validOrchestrations allowlist", b.Orchestration, stageName)
		}
	}

	// Verify pipeline-coordinator is explicitly present (AC-007).
	if !validOrchestrations["pipeline-coordinator"] {
		t.Error("pipeline-coordinator must be in validOrchestrations")
	}

	// Verify no unused orchestration entries (defensive).
	for orch := range validOrchestrations {
		if !seen[orch] {
			t.Logf("orchestration %q in allowlist but not used by any canonical stage", orch)
		}
	}
}

// findCanonicalBindings resolves the path to .kbz/stage-bindings.yaml.
// Tries relative to the repo root by walking up from the working directory.
func findCanonicalBindings(t *testing.T) string {
	t.Helper()
	// .kbz/stage-bindings.yaml relative to the repository root.
	const rel = ".kbz/stage-bindings.yaml"
	if _, err := os.Stat(rel); err == nil {
		return rel
	}
	// Try relative to GIT_COMMON_DIR (bare repo) or via go env.
	// Fall back to walking up directories.
	for _, prefix := range []string{"../../", "../../../", "../../../../"} {
		p := prefix + rel
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Fatalf("cannot find %s relative to working directory", rel)
	return ""
}

// loadBindingFile reads and unmarshals a stage-bindings.yaml file.
func loadBindingFile(t *testing.T, path string) *BindingFile {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	var bf BindingFile
	if err := yaml.Unmarshal(data, &bf); err != nil {
		t.Fatalf("failed to parse %s: %v", path, err)
	}
	return &bf
}

// findRepoRoot walks up from the test file to find the repository root.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find repository root (no .git found)")
		}
		dir = parent
	}
}

// =============================================================================
// Stage-bindings structural equality test (AC-009, REQ-006)
// =============================================================================

// TestStageBindingsStructuralEquality verifies that .kbz/stage-bindings.yaml
// and internal/kbzinit/stage-bindings.yaml have identical stage keys and
// schema_version. This test must fail CI if the files drift apart (AC-009).
func TestStageBindingsStructuralEquality(t *testing.T) {
	t.Parallel()

	repoRoot := findRepoRoot(t)

	canonicalPath := filepath.Join(repoRoot, ".kbz", "stage-bindings.yaml")
	embeddedPath := filepath.Join(repoRoot, "internal", "kbzinit", "stage-bindings.yaml")

	canonical := loadBindingFile(t, canonicalPath)
	embedded := loadBindingFile(t, embeddedPath)

	// Compare schema_version (AC-010).
	if canonical.SchemaVersion != embedded.SchemaVersion {
		t.Errorf("schema_version mismatch: canonical=%d, embedded=%d",
			canonical.SchemaVersion, embedded.SchemaVersion)
	}

	// Compare stage keys — every canonical stage must be in embedded.
	for stageName := range canonical.StageBindings {
		if _, ok := embedded.StageBindings[stageName]; !ok {
			t.Errorf("stage %q exists in canonical .kbz/stage-bindings.yaml but missing from embedded internal/kbzinit/stage-bindings.yaml", stageName)
		}
	}

	// Compare stage keys — every embedded stage must be in canonical.
	for stageName := range embedded.StageBindings {
		if _, ok := canonical.StageBindings[stageName]; !ok {
			t.Errorf("stage %q exists in embedded internal/kbzinit/stage-bindings.yaml but missing from canonical .kbz/stage-bindings.yaml", stageName)
		}
	}
}

// =============================================================================
// Role and skill reachability tests (AC-011, REQ-008)
// =============================================================================

// TestBindingRolesReachable verifies that every role value referenced in the
// canonical stage-bindings.yaml resolves to a file in .kbz/roles/ (AC-011).
func TestBindingRolesReachable(t *testing.T) {
	t.Parallel()

	repoRoot := findRepoRoot(t)
	canonicalPath := filepath.Join(repoRoot, ".kbz", "stage-bindings.yaml")
	canonical := loadBindingFile(t, canonicalPath)

	rolesDir := filepath.Join(repoRoot, ".kbz", "roles")

	// Collect all role values from all stages (including sub_agents.roles).
	allRoles := collectBindingRoles(canonical)

	for _, role := range allRoles {
		roleFile := filepath.Join(rolesDir, role+".yaml")
		if _, err := os.Stat(roleFile); os.IsNotExist(err) {
			t.Errorf("role %q referenced in stage-bindings.yaml but no file exists at %s", role, roleFile)
		}
	}
}

// TestBindingSkillsReachable verifies that every skill value referenced in the
// canonical stage-bindings.yaml resolves to a directory in .kbz/skills/ (AC-011).
// Skills on disk that are unreferenced by any binding produce a CI warning
// (not a failure), with a known-allowlist for direct-trigger skills.
func TestBindingSkillsReachable(t *testing.T) {
	t.Parallel()

	repoRoot := findRepoRoot(t)
	canonicalPath := filepath.Join(repoRoot, ".kbz", "stage-bindings.yaml")
	canonical := loadBindingFile(t, canonicalPath)

	skillsDir := filepath.Join(repoRoot, ".kbz", "skills")

	// Collect all skill values from all stages (including sub_agents.skills).
	allSkills := collectBindingSkills(canonical)

	for _, skill := range allSkills {
		skillDir := filepath.Join(skillsDir, skill)
		if fi, err := os.Stat(skillDir); err != nil || !fi.IsDir() {
			t.Errorf("skill %q referenced in stage-bindings.yaml but no directory exists at %s", skill, skillDir)
		}
	}

	// Check for unreferenced skills on disk (CI warning, not failure).
	// Known-allowlist: direct-trigger skills that are never bound to a stage.
	knownUnreferenced := map[string]bool{
		"audit-codebase":         true,
		"implement-retro-fix":    true,
		"implement-task":         true,
		"prompt-engineering":     true,
		"validate-plan":          true,
		"validate-review":        true,
		"validate-spec":          true,
		"write-skill":            true,
		"write-docs":             true,
		"edit-docs":              true,
		"check-docs":             true,
		"style-docs":             true,
		"copyedit-docs":          true,
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("cannot read skills directory: %v", err)
	}

	referenced := make(map[string]bool, len(allSkills))
	for _, s := range allSkills {
		referenced[s] = true
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip non-skill directories like "references".
		if name == "references" {
			continue
		}
		if !referenced[name] && !knownUnreferenced[name] {
			t.Logf("WARNING: skill directory %q exists on disk but is not referenced by any stage binding", name)
		}
	}
}

// collectBindingRoles extracts all role values from a BindingFile, including
// those in sub_agents.roles.
func collectBindingRoles(bf *BindingFile) []string {
	seen := make(map[string]bool)
	var roles []string
	for _, b := range bf.StageBindings {
		for _, r := range b.Roles {
			if !seen[r] {
				seen[r] = true
				roles = append(roles, r)
			}
		}
		if b.SubAgents != nil {
			for _, r := range b.SubAgents.Roles {
				if !seen[r] {
					seen[r] = true
					roles = append(roles, r)
				}
			}
		}
	}
	return roles
}

// collectBindingSkills extracts all skill values from a BindingFile, including
// those in sub_agents.skills.
func collectBindingSkills(bf *BindingFile) []string {
	seen := make(map[string]bool)
	var skills []string
	for _, b := range bf.StageBindings {
		for _, s := range b.Skills {
			if !seen[s] {
				seen[s] = true
				skills = append(skills, s)
			}
		}
		if b.SubAgents != nil {
			for _, s := range b.SubAgents.Skills {
				if !seen[s] {
					seen[s] = true
					skills = append(skills, s)
				}
			}
		}
	}
	return skills
}

// =============================================================================
// FeatureStatus / BugStatus coverage tests (AC-012, REQ-009)
// =============================================================================

// featureStatusConstants is the hardcoded reference list of all FeatureStatus
// constants defined in internal/model/entities.go. Must be kept in sync.
var featureStatusConstants = []string{
	// Phase 2 Feature statuses (document-driven lifecycle).
	"proposed",
	"designing",
	"specifying",
	"dev-planning",
	"developing",
	"reviewing",
	"merging",
	"verifying",
	"needs-rework",
	"done",
	"superseded",
	"cancelled",
	// Phase 1 Feature statuses (deprecated, for migration compatibility).
	"draft",
	"in-review",
	"approved",
	"in-progress",
	"review",
}

// featureStatusOutOfPipeline lists FeatureStatus values that are legitimate
// lifecycle states but not stage-binding keys (terminal states, interstitial
// states, rework states, deprecated states).
var featureStatusOutOfPipeline = map[string]string{
	"proposed":     "entry state — feature has not yet entered a pipeline stage",
	"needs-rework": "interstitial — feature returns to a prior stage for rework",
	"done":         "terminal state — feature has completed all pipeline stages",
	"superseded":   "terminal state — feature was replaced by a newer feature",
	"cancelled":    "terminal state — feature was cancelled before completion",
	"draft":        "deprecated Phase 1 status",
	"in-review":    "deprecated Phase 1 status",
	"approved":     "deprecated Phase 1 status",
	"in-progress":  "deprecated Phase 1 status",
	"review":       "deprecated Phase 1 status",
}

// TestFeatureStatusCoverage verifies that every FeatureStatus constant is either
// bound to a stage-binding key or explicitly listed in the out-of-pipeline
// declaration (AC-012, REQ-009).
func TestFeatureStatusCoverage(t *testing.T) {
	t.Parallel()

	repoRoot := findRepoRoot(t)
	canonicalPath := filepath.Join(repoRoot, ".kbz", "stage-bindings.yaml")
	canonical := loadBindingFile(t, canonicalPath)

	for _, status := range featureStatusConstants {
		_, inBinding := canonical.StageBindings[status]
		_, outOfPipeline := featureStatusOutOfPipeline[status]

		if !inBinding && !outOfPipeline {
			t.Errorf("FeatureStatus %q is neither a stage-binding key nor in the out-of-pipeline declaration", status)
		}

		if inBinding && outOfPipeline {
			t.Errorf("FeatureStatus %q is both a stage-binding key AND in the out-of-pipeline declaration — remove one", status)
		}
	}
}

// bugStatusConstants is the hardcoded reference list of all BugStatus constants
// defined in internal/model/entities.go. Must be kept in sync.
var bugStatusConstants = []string{
	"reported",
	"triaged",
	"reproduced",
	"planned",
	"in-progress",
	"needs-review",
	"needs-rework",
	"verifying",
	"closed",
	"duplicate",
	"not-planned",
	"cannot-reproduce",
}

// bugStatusOutOfPipeline lists BugStatus values that are legitimate
// bug-lifecycle states but not stage-binding keys.
var bugStatusOutOfPipeline = map[string]string{
	"reported":          "entry state — bug has not yet entered a pipeline stage",
	"triaged":           "interstitial — bug is being triaged",
	"reproduced":        "interstitial — bug has been reproduced",
	"planned":           "interstitial — bug has been planned for a fix",
	"in-progress":       "interstitial — bug fix is in progress",
	"needs-review":      "interstitial — bug fix is awaiting review",
	"needs-rework":      "interstitial — bug fix needs rework",
	"closed":            "terminal state — bug is closed",
	"duplicate":         "terminal state — bug is a duplicate",
	"not-planned":       "terminal state — bug will not be fixed",
	"cannot-reproduce":  "terminal state — bug cannot be reproduced",
}

// TestBugStatusCoverage verifies that every BugStatus constant is either bound
// to a stage-binding key or explicitly listed in the out-of-pipeline declaration
// (AC-012, REQ-009).
func TestBugStatusCoverage(t *testing.T) {
	t.Parallel()

	repoRoot := findRepoRoot(t)
	canonicalPath := filepath.Join(repoRoot, ".kbz", "stage-bindings.yaml")
	canonical := loadBindingFile(t, canonicalPath)

	for _, status := range bugStatusConstants {
		_, inBinding := canonical.StageBindings[status]
		_, outOfPipeline := bugStatusOutOfPipeline[status]

		if !inBinding && !outOfPipeline {
			t.Errorf("BugStatus %q is neither a stage-binding key nor in the out-of-pipeline declaration", status)
		}

		if inBinding && outOfPipeline {
			t.Errorf("BugStatus %q is both a stage-binding key AND in the out-of-pipeline declaration — remove one", status)
		}
	}
}

// =============================================================================
// ValidateBindingFile canonical success test (AC-013, REQ-010)
// =============================================================================

// TestValidateBindingFile_Canonical verifies that ValidateBindingFile succeeds
// against the canonical .kbz/stage-bindings.yaml with a RoleChecker that
// resolves roles against the .kbz/roles/ directory (AC-013, REQ-010).
func TestValidateBindingFile_Canonical(t *testing.T) {
	t.Parallel()

	repoRoot := findRepoRoot(t)
	canonicalPath := filepath.Join(repoRoot, ".kbz", "stage-bindings.yaml")
	rolesDir := filepath.Join(repoRoot, ".kbz", "roles")

	bf, errs := LoadBindingFile(canonicalPath)
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		t.Fatalf("LoadBindingFile failed on canonical YAML: %s", strings.Join(msgs, "; "))
	}

	// Real RoleChecker: checks whether a .yaml file exists in .kbz/roles/.
	realRoleChecker := func(id string) bool {
		roleFile := filepath.Join(rolesDir, id+".yaml")
		_, err := os.Stat(roleFile)
		return err == nil
	}

	result := ValidateBindingFile(bf, realRoleChecker)

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("ValidateBindingFile error on canonical YAML: %s", e)
		}
	}

	// Warnings are allowed (e.g. role fallback warnings), but log them.
	for _, w := range result.Warnings {
		t.Logf("ValidateBindingFile warning on canonical YAML: %s", w)
	}
}
