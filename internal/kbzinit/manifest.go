package kbzinit

import "strconv"

// Manifest is the single canonical list of every artifact that kbz init
// installs. It is the sole authoritative source for artifact paths and
// version-comparison metadata (REQ-002). No other file in this package
// may redeclare these artifact names as string literals.
//
// CurrentValue in each MarkerSpec:
//   - AGENTS.md / CopilotInstructions: set to the integer version string
//     (same value as agentsMDVersion) — known at compile time.
//   - Skills, roles, stage-bindings: empty — T4's installArtifact will
//     populate it from the Initializer's runtime binary version.
//
// Marker.Comment is the line prefix that compareManaged uses to find and
// extract the version from an installed file:
//   - Skills: "# kanbanzai-version:" (the version-bearing comment line)
//   - Roles (non-base): `  version: "` (YAML field with leading indent and
//     opening quote; compareManaged strips the trailing quote via Trim)
//   - StageBindings: "# kanbanzai-version:" (same two-comment-line format)
//   - AgentsMd / CopilotInstructions: "<!-- kanbanzai-managed: v" (HTML
//     comment with version inline)
var Manifest = []Artifact{
	// ----------------------------------------------------------------
	// Workflow skills — installed to .agents/skills/kanbanzai-<name>/SKILL.md
	// ----------------------------------------------------------------
	{
		Name:        "kanbanzai-agents",
		Kind:        WorkflowSkill,
		EmbedPath:   "skills/agents/SKILL.md",
		InstallPath: ".agents/skills/kanbanzai-agents/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "kanbanzai-design",
		Kind:        WorkflowSkill,
		EmbedPath:   "skills/design/SKILL.md",
		InstallPath: ".agents/skills/kanbanzai-design/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "kanbanzai-documents",
		Kind:        WorkflowSkill,
		EmbedPath:   "skills/documents/SKILL.md",
		InstallPath: ".agents/skills/kanbanzai-documents/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "kanbanzai-getting-started",
		Kind:        WorkflowSkill,
		EmbedPath:   "skills/getting-started/SKILL.md",
		InstallPath: ".agents/skills/kanbanzai-getting-started/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "kanbanzai-plan-review",
		Kind:        WorkflowSkill,
		EmbedPath:   "skills/plan-review/SKILL.md",
		InstallPath: ".agents/skills/kanbanzai-plan-review/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "kanbanzai-planning",
		Kind:        WorkflowSkill,
		EmbedPath:   "skills/planning/SKILL.md",
		InstallPath: ".agents/skills/kanbanzai-planning/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "kanbanzai-review",
		Kind:        WorkflowSkill,
		EmbedPath:   "skills/review/SKILL.md",
		InstallPath: ".agents/skills/kanbanzai-review/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "kanbanzai-specification",
		Kind:        WorkflowSkill,
		EmbedPath:   "skills/specification/SKILL.md",
		InstallPath: ".agents/skills/kanbanzai-specification/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "kanbanzai-workflow",
		Kind:        WorkflowSkill,
		EmbedPath:   "skills/workflow/SKILL.md",
		InstallPath: ".agents/skills/kanbanzai-workflow/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},

	// ----------------------------------------------------------------
	// Task-execution skills — installed to .kbz/skills/<name>/SKILL.md
	// ----------------------------------------------------------------
	{
		Name:        "audit-codebase",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/audit-codebase/SKILL.md",
		InstallPath: ".kbz/skills/audit-codebase/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "check-docs",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/check-docs/SKILL.md",
		InstallPath: ".kbz/skills/check-docs/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "copyedit-docs",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/copyedit-docs/SKILL.md",
		InstallPath: ".kbz/skills/copyedit-docs/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "decompose-feature",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/decompose-feature/SKILL.md",
		InstallPath: ".kbz/skills/decompose-feature/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "edit-docs",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/edit-docs/SKILL.md",
		InstallPath: ".kbz/skills/edit-docs/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "implement-task",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/implement-task/SKILL.md",
		InstallPath: ".kbz/skills/implement-task/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "orchestrate-development",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/orchestrate-development/SKILL.md",
		InstallPath: ".kbz/skills/orchestrate-development/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "orchestrate-doc-pipeline",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/orchestrate-doc-pipeline/SKILL.md",
		InstallPath: ".kbz/skills/orchestrate-doc-pipeline/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "orchestrate-review",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/orchestrate-review/SKILL.md",
		InstallPath: ".kbz/skills/orchestrate-review/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "prompt-engineering",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/prompt-engineering/SKILL.md",
		InstallPath: ".kbz/skills/prompt-engineering/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "review-code",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/review-code/SKILL.md",
		InstallPath: ".kbz/skills/review-code/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "review-plan",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/review-plan/SKILL.md",
		InstallPath: ".kbz/skills/review-plan/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "style-docs",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/style-docs/SKILL.md",
		InstallPath: ".kbz/skills/style-docs/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "update-docs",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/update-docs/SKILL.md",
		InstallPath: ".kbz/skills/update-docs/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "write-design",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/write-design/SKILL.md",
		InstallPath: ".kbz/skills/write-design/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "write-dev-plan",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/write-dev-plan/SKILL.md",
		InstallPath: ".kbz/skills/write-dev-plan/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "write-docs",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/write-docs/SKILL.md",
		InstallPath: ".kbz/skills/write-docs/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "write-research",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/write-research/SKILL.md",
		InstallPath: ".kbz/skills/write-research/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "write-skill",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/write-skill/SKILL.md",
		InstallPath: ".kbz/skills/write-skill/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
	{
		Name:        "write-spec",
		Kind:        TaskSkill,
		EmbedPath:   "skills/task-execution/write-spec/SKILL.md",
		InstallPath: ".kbz/skills/write-spec/SKILL.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},

	// ----------------------------------------------------------------
	// Role files — installed to .kbz/roles/<name>.yaml
	//
	// base.yaml is a scaffold role: written once on first init and never
	// overwritten. It carries no managed marker. All other roles are
	// managed (version-aware update via Marker).
	//
	// The Comment for managed roles is `  version: "` — this matches the
	// indented YAML field `  version: "x.y.z"`. compareManaged strips the
	// trailing quote via strings.Trim(raw, `"`).
	// ----------------------------------------------------------------
	{
		Name:        "base.yaml",
		Kind:        Role,
		EmbedPath:   "roles/base.yaml",
		InstallPath: ".kbz/roles/base.yaml",
		Required:    true,
		// Scaffold — no managed marker; compareManaged is not called for this entry.
	},
	{
		Name:        "architect.yaml",
		Kind:        Role,
		EmbedPath:   "roles/architect.yaml",
		InstallPath: ".kbz/roles/architect.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "doc-checker.yaml",
		Kind:        Role,
		EmbedPath:   "roles/doc-checker.yaml",
		InstallPath: ".kbz/roles/doc-checker.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "doc-copyeditor.yaml",
		Kind:        Role,
		EmbedPath:   "roles/doc-copyeditor.yaml",
		InstallPath: ".kbz/roles/doc-copyeditor.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "doc-editor.yaml",
		Kind:        Role,
		EmbedPath:   "roles/doc-editor.yaml",
		InstallPath: ".kbz/roles/doc-editor.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "doc-pipeline-orchestrator.yaml",
		Kind:        Role,
		EmbedPath:   "roles/doc-pipeline-orchestrator.yaml",
		InstallPath: ".kbz/roles/doc-pipeline-orchestrator.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "doc-stylist.yaml",
		Kind:        Role,
		EmbedPath:   "roles/doc-stylist.yaml",
		InstallPath: ".kbz/roles/doc-stylist.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "documenter.yaml",
		Kind:        Role,
		EmbedPath:   "roles/documenter.yaml",
		InstallPath: ".kbz/roles/documenter.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "implementer-go.yaml",
		Kind:        Role,
		EmbedPath:   "roles/implementer-go.yaml",
		InstallPath: ".kbz/roles/implementer-go.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "implementer.yaml",
		Kind:        Role,
		EmbedPath:   "roles/implementer.yaml",
		InstallPath: ".kbz/roles/implementer.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "orchestrator.yaml",
		Kind:        Role,
		EmbedPath:   "roles/orchestrator.yaml",
		InstallPath: ".kbz/roles/orchestrator.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "researcher.yaml",
		Kind:        Role,
		EmbedPath:   "roles/researcher.yaml",
		InstallPath: ".kbz/roles/researcher.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "reviewer-conformance.yaml",
		Kind:        Role,
		EmbedPath:   "roles/reviewer-conformance.yaml",
		InstallPath: ".kbz/roles/reviewer-conformance.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "reviewer-quality.yaml",
		Kind:        Role,
		EmbedPath:   "roles/reviewer-quality.yaml",
		InstallPath: ".kbz/roles/reviewer-quality.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "reviewer-security.yaml",
		Kind:        Role,
		EmbedPath:   "roles/reviewer-security.yaml",
		InstallPath: ".kbz/roles/reviewer-security.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "reviewer-testing.yaml",
		Kind:        Role,
		EmbedPath:   "roles/reviewer-testing.yaml",
		InstallPath: ".kbz/roles/reviewer-testing.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "reviewer.yaml",
		Kind:        Role,
		EmbedPath:   "roles/reviewer.yaml",
		InstallPath: ".kbz/roles/reviewer.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},
	{
		Name:        "spec-author.yaml",
		Kind:        Role,
		EmbedPath:   "roles/spec-author.yaml",
		InstallPath: ".kbz/roles/spec-author.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     `  version: "`,
			VersionKind: Semver,
		},
	},

	// ----------------------------------------------------------------
	// Top-level configuration files
	// ----------------------------------------------------------------

	// AGENTS.md — generated content (not embedded from file).
	// Uses an integer version counter; CurrentValue is set at compile time.
	{
		Name:        "AGENTS.md",
		Kind:        AgentsMd,
		EmbedPath:   "",
		InstallPath: "AGENTS.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:      "<!-- kanbanzai-managed: v",
			VersionKind:  IntCounter,
			CurrentValue: strconv.Itoa(agentsMDVersion),
		},
	},

	// .github/copilot-instructions.md — generated content.
	// Shares the same integer version counter as AGENTS.md.
	{
		Name:        "copilot-instructions.md",
		Kind:        CopilotInstructions,
		EmbedPath:   "",
		InstallPath: ".github/copilot-instructions.md",
		Required:    true,
		Marker: MarkerSpec{
			Comment:      "<!-- kanbanzai-managed: v",
			VersionKind:  IntCounter,
			CurrentValue: strconv.Itoa(agentsMDVersion),
		},
	},

	// .kbz/stage-bindings.yaml — embedded file.
	// Must use Semver so compareManaged can parse the binary semver
	// correctly (fixes the always-rewrite defect from AC-005).
	{
		Name:        "stage-bindings.yaml",
		Kind:        StageBindings,
		EmbedPath:   "stage-bindings.yaml",
		InstallPath: ".kbz/stage-bindings.yaml",
		Required:    true,
		Marker: MarkerSpec{
			Comment:     "# kanbanzai-version:",
			VersionKind: Semver,
		},
	},
}
