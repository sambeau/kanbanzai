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

| Role | File | Stage |
|------|------|-------|
| `architect` | `.kbz/roles/architect.yaml` | designing, dev-planning |
| `spec-author` | `.kbz/roles/spec-author.yaml` | specifying |
| `implementer-go` | `.kbz/roles/implementer-go.yaml` | developing (sub-agent) |
| `orchestrator` | `.kbz/roles/orchestrator.yaml` | developing, reviewing |
| `reviewer` | `.kbz/roles/reviewer.yaml` | reviewing (base) |
| `reviewer-conformance` | `.kbz/roles/reviewer-conformance.yaml` | reviewing, batch-reviewing |
| `reviewer-quality` | `.kbz/roles/reviewer-quality.yaml` | reviewing |
| `reviewer-security` | `.kbz/roles/reviewer-security.yaml` | reviewing |
| `reviewer-testing` | `.kbz/roles/reviewer-testing.yaml` | reviewing |
| `researcher` | `.kbz/roles/researcher.yaml` | researching |
| `documenter` | `.kbz/roles/documenter.yaml` | documenting (writing) |
| `doc-pipeline-orchestrator` | `.kbz/roles/doc-pipeline-orchestrator.yaml` | doc-publishing |
| `doc-editor` | `.kbz/roles/doc-editor.yaml` | doc-publishing (sub-agent) |
| `doc-checker` | `.kbz/roles/doc-checker.yaml` | doc-publishing (sub-agent) |
| `doc-stylist` | `.kbz/roles/doc-stylist.yaml` | doc-publishing (sub-agent) |
| `doc-copyeditor` | `.kbz/roles/doc-copyeditor.yaml` | doc-publishing (sub-agent) |

### Skills (`.kbz/skills/`)

Skills define *what you're doing right now* — the procedure, vocabulary, anti-patterns, and
checklist for a specific task type. Each skill is a `SKILL.md` file in its own directory.
Always read the skill specified by the stage binding before starting work.

| Skill | Path | Stage |
|-------|------|-------|
| **write-design** | `.kbz/skills/write-design/SKILL.md` | designing |
| **write-spec** | `.kbz/skills/write-spec/SKILL.md` | specifying |
| **write-dev-plan** | `.kbz/skills/write-dev-plan/SKILL.md` | dev-planning |
| **decompose-feature** | `.kbz/skills/decompose-feature/SKILL.md` | dev-planning |
| **orchestrate-development** | `.kbz/skills/orchestrate-development/SKILL.md` | developing |
| **implement-task** | `.kbz/skills/implement-task/SKILL.md` | developing (sub-agent) |
| **review-code** | `.kbz/skills/review-code/SKILL.md` | reviewing |
| **orchestrate-review** | `.kbz/skills/orchestrate-review/SKILL.md` | reviewing |
| **review-plan** | `.kbz/skills/review-plan/SKILL.md` | batch-reviewing |
| **write-research** | `.kbz/skills/write-research/SKILL.md` | researching |
| **update-docs** | `.kbz/skills/update-docs/SKILL.md` | documenting |
| **orchestrate-doc-pipeline** | `.kbz/skills/orchestrate-doc-pipeline/SKILL.md` | doc-publishing |
| **write-docs** | `.kbz/skills/write-docs/SKILL.md` | doc-publishing (write stage) |
| **edit-docs** | `.kbz/skills/edit-docs/SKILL.md` | doc-publishing (edit stage) |
| **check-docs** | `.kbz/skills/check-docs/SKILL.md` | doc-publishing (check stage) |
| **style-docs** | `.kbz/skills/style-docs/SKILL.md` | doc-publishing (style stage) |
| **copyedit-docs** | `.kbz/skills/copyedit-docs/SKILL.md` | doc-publishing (copyedit stage) |
| **audit-codebase** | `.kbz/skills/audit-codebase/SKILL.md` | auditing (on-demand) |

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

- **Check `git status` before every task.** Commit or stash previous work first.
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