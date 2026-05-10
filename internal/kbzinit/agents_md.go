package kbzinit

import (
	"fmt"
)

// agentsMDVersion is the current schema version for both AGENTS.md and
// .github/copilot-instructions.md. Increment when the generated content
// changes in a way that warrants overwriting existing managed files.
const agentsMDVersion = 3

// agentsMDContent is the generated content for AGENTS.md.
// Must start with the managed marker comment on line 1.
// Must not exceed 100 lines.
const agentsMDContent = `<!-- kanbanzai-managed: v3 -->

# Agent Instructions

This project uses **Kanbanzai** for workflow management via MCP tools. All
workflow state is managed through the kanbanzai MCP server.

## Before You Do Anything

1. Call ` + "`status`" + ` to see the current project state.
2. Call ` + "`next`" + ` to see the work queue.
3. Read ` + "`.agents/skills/kanbanzai-getting-started/SKILL.md`" + ` for full orientation.

## Rules

- **Use kanbanzai MCP tools** (` + "`status`, `next`, `entity`, `doc`, `finish`" + `) for all
  workflow operations. Do not create or modify entities or documents by writing
  files directly.
- **Follow the stage gates.** Read ` + "`.kbz/stage-bindings.yaml`" + ` to find the
  correct role and skill for each stage. See
  ` + "`.agents/skills/kanbanzai-workflow/SKILL.md`" + ` for lifecycle rules.
- **Human approval is required** at stage gates. When in doubt, stop and ask.

## Skills Reference

### Workflow Skills (.agents/skills/)

| Skill | When to read |
|---|---|
| ` + "`kanbanzai-getting-started`" + ` | Start of every session |
| ` + "`kanbanzai-workflow`" + ` | Before any stage transition |
| ` + "`kanbanzai-documents`" + ` | Creating or registering documents |
| ` + "`kanbanzai-agents`" + ` | Task dispatch, commits, knowledge |
| ` + "`kanbanzai-planning`" + ` | Planning conversations |
| ` + "`kanbanzai-plan-review`" + ` | Reviewing plans |

### Task-Execution Skills (.kbz/skills/)

| Skill | Stage |
|---|---|
| ` + "`write-design`" + ` | designing |
| ` + "`write-spec`" + ` | specifying |
| ` + "`write-dev-plan`" + ` | dev-planning |
| ` + "`decompose-feature`" + ` | dev-planning |
| ` + "`orchestrate-development`" + ` | developing |
| ` + "`implement-task`" + ` | developing (sub-agent) |
| ` + "`orchestrate-review`" + ` | reviewing |
| ` + "`review-code`" + ` | reviewing (sub-agent) |
| ` + "`review-plan`" + ` | batch-reviewing |
| ` + "`write-research`" + ` | researching |
| ` + "`update-docs`" + ` | documenting |
| ` + "`orchestrate-doc-pipeline`" + ` | doc-publishing |
| ` + "`write-docs`" + `, ` + "`edit-docs`" + `, ` + "`check-docs`" + `, ` + "`style-docs`" + `, ` + "`copyedit-docs`" + ` | doc-publishing (sub-stages) |

## Roles (.kbz/roles/)

Read the role file specified in .kbz/stage-bindings.yaml for your stage.

| Role | Stage |
|---|---|
| ` + "`architect`" + ` | designing, dev-planning |
| ` + "`spec-author`" + ` | specifying |
| ` + "`orchestrator`" + ` | developing, reviewing |
| ` + "`implementer`" + ` | developing (sub-agent) |
| ` + "`reviewer-conformance`" + ` | reviewing, batch-reviewing |
| ` + "`reviewer-quality`" + ` | reviewing (sub-agent) |
| ` + "`reviewer-security`" + ` | reviewing (sub-agent) |
| ` + "`reviewer-testing`" + ` | reviewing (sub-agent) |
| ` + "`researcher`" + ` | researching |
| ` + "`documenter`" + ` | documenting |
| ` + "`doc-pipeline-orchestrator`" + ` | doc-publishing |
| ` + "`doc-editor`" + `, ` + "`doc-checker`" + `, ` + "`doc-stylist`" + `, ` + "`doc-copyeditor`" + ` | doc-publishing (sub-agents) |

## Stage Bindings

.kbz/stage-bindings.yaml is the authoritative source of truth. Read it before
entering any workflow stage — it tells you which role and skill to adopt.

## Code Graph Integration (Optional)

If your project uses codebase-memory-mcp, set graph_project in
.kbz/local.yaml once per machine. The worktree tool uses it automatically.

    codebase_memory:
      graph_project: YOUR-GRAPH-PROJECT-NAME

To install graph tool skills, copy .github/skills/ from the kanbanzai repo.
`

// copilotInstructionsContent is the generated content for
// .github/copilot-instructions.md.
// Must start with the managed marker comment on line 1.
// Must not exceed 25 lines.
const copilotInstructionsContent = `<!-- kanbanzai-managed: v3 -->

# Copilot Instructions

This project uses **Kanbanzai** for workflow management. Read ` + "`AGENTS.md`" + ` in
the project root before doing any work.

Read ` + "`.kbz/stage-bindings.yaml`" + ` before entering any workflow stage to find
the correct role and skill to adopt.

## Quick Reference

1. Call ` + "`status`" + ` to see the current project state.
2. Call ` + "`next`" + ` to see the work queue.
3. Read ` + "`.agents/skills/kanbanzai-getting-started/SKILL.md`" + ` at session start.
4. Use kanbanzai MCP tools for all workflow operations — do not write
   ` + "`work/`" + ` documents or ` + "`.kbz/state/`" + ` entities directly.
`

// writeAgentsMD writes AGENTS.md to baseDir using the Manifest's AgentsMd
// artifact and compareManaged decision logic.
func (i *Initializer) writeAgentsMD(baseDir string) error {
	a := manifestByKind(AgentsMd)
	if a == nil {
		return fmt.Errorf("AGENTS.md not found in Manifest")
	}
	return installArtifact(*a, []byte(agentsMDContent), i.stdout, baseDir)
}

// writeCopilotInstructions writes .github/copilot-instructions.md to baseDir,
// creating the .github/ directory if it does not exist.
func (i *Initializer) writeCopilotInstructions(baseDir string) error {
	a := manifestByKind(CopilotInstructions)
	if a == nil {
		return fmt.Errorf("copilot-instructions.md not found in Manifest")
	}
	return installArtifact(*a, []byte(copilotInstructionsContent), i.stdout, baseDir)
}

// manifestByKind returns a pointer to the first Manifest entry of the given
// kind, or nil if none exists.
func manifestByKind(kind ArtifactKind) *Artifact {
	for i := range Manifest {
		if Manifest[i].Kind == kind {
			return &Manifest[i]
		}
	}
	return nil
}
