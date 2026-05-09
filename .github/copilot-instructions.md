<!-- kanbanzai-project: this file is hand-maintained for kanbanzai's own development.
It contains project-local references not present in generated consumer installs.
See internal/kbzinit/agents_md.go for the canonical consumer version. -->

# Copilot Instructions for Kanbanzai

This project is a Git-native workflow system for human-AI collaborative software development.
It has an MCP server (`kanbanzai serve`) that provides structured workflow tools — use those
tools instead of reading `.kbz/` state files directly.

## Start here

Read `AGENTS.md` in the repository root. It contains project-specific conventions, repository
structure, build commands, Git discipline rules, and a required pre-task checklist.

## Plan/Batch Vocabulary Summary

This project uses two distinct entity types where previously there was one:

| Term | Meaning |
|------|---------|
| **plan** | Strategic layer — scope decomposition, long-term direction. Lifecycle: `idea → shaping → ready → active → done`. ID prefix: `P{n}`. |
| **batch** | Execution layer — groups features for implementation. Replaces what was previously called "plan". Lifecycle: `proposed → designing → active → reviewing → done`. ID prefix: `B{n}`. |
| **dev-plan** | Document type — the implementation plan document attached to a feature. Unchanged by the plan/batch distinction. |

Features belong to batches. Batches optionally belong to plans.

## Roles, skills, and stage bindings

This project uses an evidence-based roles and skills system. **Before starting any task, check
which role and skill apply to the current workflow stage.** The stage bindings file is the
single source of truth for this mapping.

### Stage bindings — read this first

`.kbz/stage-bindings.yaml` maps each workflow stage to the roles, skills, orchestration
pattern, and prerequisites that apply. When you enter a stage (designing, specifying,
dev-planning, developing, reviewing, batch-reviewing, researching, documenting), read the
corresponding binding to know what role to adopt and which skill to follow.

### Roles (`.kbz/roles/`)

Roles define *who you are* — identity, vocabulary, anti-patterns, and tool constraints.
Each role is a YAML file. Roles use inheritance (e.g. `reviewer-security` inherits from
`reviewer`). Always read the role specified by the stage binding before starting work.

<!-- registry-gen:begin:role-index source=.kbz/roles/*.yaml -->
> **Generated** — canonical source: `.kbz/roles/*.yaml`. Do not hand-edit this section; run `make registry-sync` to update.

| Role | Identity | Inherits | Source |
|------|----------|----------|--------|
| `architect` | Senior software architect | `base` | `.kbz/roles/architect.yaml` |
| `base` | Software development agent | — | `.kbz/roles/base.yaml` |
| `doc-checker` | Technical fact-checker | `base` | `.kbz/roles/doc-checker.yaml` |
| `doc-copyeditor` | Senior copy editor | `base` | `.kbz/roles/doc-copyeditor.yaml` |
| `doc-editor` | Developmental editor | `base` | `.kbz/roles/doc-editor.yaml` |
| `doc-pipeline-orchestrator` | AI content editor | `base` | `.kbz/roles/doc-pipeline-orchestrator.yaml` |
| `doc-stylist` | AI prose editor | `base` | `.kbz/roles/doc-stylist.yaml` |
| `documenter` | Senior technical writer | `base` | `.kbz/roles/documenter.yaml` |
| `implementer` | Senior software engineer | `base` | `.kbz/roles/implementer.yaml` |
| `implementer-go` | Senior Go engineer | `implementer` | `.kbz/roles/implementer-go.yaml` |
| `orchestrator` | Senior engineering manager coordinating an agent team | `base` | `.kbz/roles/orchestrator.yaml` |
| `plan-validator` | Senior implementation plan auditor. Verify that dev-plans are complete, well-decomposed, and fully traceable to their parent specification. Do not evaluate whether task ordering is *optimal* — only whether it is *valid* and *complete*. | `base` | `.kbz/roles/plan-validator.yaml` |
| `researcher` | Senior technical analyst | `base` | `.kbz/roles/researcher.yaml` |
| `review-gate-validator` | Senior review quality auditor. Verify that a completed review is thorough, evidence-backed, and suitable for auto-approval. Do not re-review the code — audit the review process itself. | `reviewer` | `.kbz/roles/review-gate-validator.yaml` |
| `reviewer` | Senior code reviewer | `base` | `.kbz/roles/reviewer.yaml` |
| `reviewer-conformance` | Senior requirements verification engineer | `reviewer` | `.kbz/roles/reviewer-conformance.yaml` |
| `reviewer-quality` | Senior software quality engineer | `reviewer` | `.kbz/roles/reviewer-quality.yaml` |
| `reviewer-security` | Senior application security engineer | `reviewer` | `.kbz/roles/reviewer-security.yaml` |
| `reviewer-testing` | Senior test engineer | `reviewer` | `.kbz/roles/reviewer-testing.yaml` |
| `spec-author` | Senior requirements engineer | `base` | `.kbz/roles/spec-author.yaml` |
| `spec-validator` | Senior requirements quality auditor. Verify that specifications are complete, testable, and traceable to their parent design. Do not evaluate whether requirements are *correct* — only whether they are *well-formed* and *complete*. | `base` | `.kbz/roles/spec-validator.yaml` |
| `verifier` | Methodical close-out auditor | `base` | `.kbz/roles/verifier.yaml` |
<!-- registry-gen:end:role-index -->

### Skills (`.kbz/skills/`)

Skills define *what you're doing right now* — the procedure, vocabulary, anti-patterns, and
checklist for a specific task type. Each skill is a `SKILL.md` file in its own directory.
Always read the skill specified by the stage binding before starting work.

<!-- registry-gen:begin:roles-and-skills source=.kbz/stage-bindings.yaml -->
> **Generated** — canonical source: `.kbz/stage-bindings.yaml`. Do not hand-edit this section; run `make registry-sync` to update.

| Stage | Description | Roles | Skills | Gate | Doc Type |
|-------|-------------|-------|--------|------|----------|
| designing | Creating or revising a design document | `architect` | [write-design](.kbz/skills/write-design/SKILL.md) | auto | design |
| specifying | Writing a formal specification with acceptance criteria | `spec-author` | [write-spec](.kbz/skills/write-spec/SKILL.md) | human | specification |
| dev-planning | Breaking a spec into an implementation plan and tasks | `architect` | [write-dev-plan](.kbz/skills/write-dev-plan/SKILL.md), [decompose-feature](.kbz/skills/decompose-feature/SKILL.md) | human | dev-plan |
| developing | Implementing tasks from the dev plan | `orchestrator` | [orchestrate-development](.kbz/skills/orchestrate-development/SKILL.md) | auto | — |
| reviewing | Evaluating implementation against the specification | `orchestrator` | [orchestrate-review](.kbz/skills/orchestrate-review/SKILL.md) | human | report |
| merging | Merging the feature branch into main after review approval | `orchestrator` | [orchestrate-review](.kbz/skills/orchestrate-review/SKILL.md) | auto | — |
| verifying | Delegating close-out verification of the Definition of Done to a clean-context verifier sub-agent | `orchestrator` | [orchestrate-review](.kbz/skills/orchestrate-review/SKILL.md) | auto | — |
| batch-reviewing | Reviewing a completed batch for aggregate delivery | `reviewer-conformance` | [review-plan](.kbz/skills/review-plan/SKILL.md) | human | report |
| researching | Producing a research report or analysis | `researcher` | [write-research](.kbz/skills/write-research/SKILL.md) | auto | research |
| documenting | Updating project documentation for currency | `documenter` | [update-docs](.kbz/skills/update-docs/SKILL.md) | auto | — |
| doc-publishing | Running a document through the five-stage editorial pipeline | `doc-pipeline-orchestrator` | [orchestrate-doc-pipeline](.kbz/skills/orchestrate-doc-pipeline/SKILL.md) | auto | — |
| retro-fixing | Implementing a fix for a retrospective theme | — | — | auto | — |
<!-- registry-gen:end:roles-and-skills -->

### How to use the system

1. Determine your current workflow stage (from the feature's lifecycle state).
2. Read `.kbz/stage-bindings.yaml` for that stage — it tells you the role, skill, and prerequisites.
3. Read the role YAML file to adopt its identity, vocabulary, and anti-patterns.
4. Read the skill `SKILL.md` to follow its procedure and checklist.
5. Use the vocabulary from both role and skill consistently. Do not substitute synonyms.

## Kanbanzai workflow guides (`.agents/skills/`)

These skills describe how to use the Kanbanzai system itself — lifecycle rules, commit
conventions, and tool usage. They complement the task-execution skills above.

| Skill | Path | When to use |
|-------|------|-------------|
| **Getting started** | `.agents/skills/kanbanzai-getting-started/SKILL.md` | Start of every session — orientation and work queue |
| **Workflow** | `.agents/skills/kanbanzai-workflow/SKILL.md` | Stage gates, lifecycle transitions, when to stop and ask |
| **Agents** | `.agents/skills/kanbanzai-agents/SKILL.md` | Task dispatch, commits, knowledge, sub-agent spawning |
| **Documents** | `.agents/skills/kanbanzai-documents/SKILL.md` | Creating, registering, approving documents |
| **Planning** | `.agents/skills/kanbanzai-planning/SKILL.md` | Scoping work, feature vs batch decisions |

## Codebase knowledge graph skills (`.github/skills/`)

**Before using any graph tool (`search_graph`, `codebase_memory_mcp_search_code`, `query_graph`,
`trace_call_path`, `get_code_snippet`, etc.), read the relevant SKILL.md file below.** These
skills define correct tool sequences, query patterns, and anti-patterns. Do not rely on intuition
or prior session context — read the skill file first.

| Skill | Path | When to use |
|-------|------|-------------|
| **Exploring** | `.github/skills/codebase-memory-exploring/SKILL.md` | Codebase orientation, architecture understanding |
| **Tracing** | `.github/skills/codebase-memory-tracing/SKILL.md` | Call chain analysis, impact assessment |
| **Quality** | `.github/skills/codebase-memory-quality/SKILL.md` | Code quality analysis patterns |
| **Reference** | `.github/skills/codebase-memory-reference/SKILL.md` | Full tool reference for graph tools |
| **Planning** | `.github/skills/planning-excellence/SKILL.md` | Planning philosophy and design quality |

## Critical rules

- **Use `kanbanzai_edit_file` and `write_file` with `entity_id` for worktree file operations.** Both tools accept `entity_id` to scope writes to a feature's worktree. Prefer these over terminal-based workarounds (heredoc, python3 -c) when working inside a worktree.
- **Check `git status` before every task.** Commit previous work first; do not use `git stash` (stashing hides state from parallel agents and is silently lost across worktree switches).
- **Verify entity existence before working on it.** When asked to work on a plan, batch, or feature by name, call `status(id: "...")` or `entity(action: "get", id: "...")` first. If the entity does not exist, STOP and ask the human whether to create it. Never proceed with work under an unregistered entity name — unregistered entities have no lifecycle state, no worktree isolation, and no task tracking. P47/B46 is the canonical example: documents and a branch used the name, but no entity was ever created, so implementation happened on main with no guardrails.
- **Read the stage binding, role, and skill before starting work.** They define your vocabulary, procedure, and constraints.
- **Do not read `.kbz/state/` files directly.** Use MCP tools (`entity`, `doc`, `status`, `knowledge`, etc.). Role, skill, and stage-binding files in `.kbz/` are meant to be read directly.
- **Use graph tools over grep** for structural code questions. Project: `Users-samphillips-Dev-kanbanzai`. Read the relevant `.github/skills/codebase-memory-*/SKILL.md` before using any graph tool.
- **Follow commit message format** from the `kanbanzai-agents` skill: `type(scope): description`.
- **Do not skip workflow stages.** Check stage gate prerequisites before advancing features.

## Reference files

| Topic | Path |
|-------|------|
| Go code style | `refs/go-style.md` |
| Test conventions | `refs/testing.md` |
| Knowledge graph usage | `refs/knowledge-graph.md` |
| Sub-agent delegation | `refs/sub-agents.md` |
| Document-to-topic map | `refs/document-map.md` |

## Templates

Use these when producing specifications, plans, or reviews:

- `work/templates/specification-prompt-template.md` — required sections and example requirements
- `work/templates/implementation-plan-prompt-template.md` — task breakdown structure and examples
- `work/templates/review-prompt-template.md` — finding format and severity levels
