# Claude Code Instructions for Kanbanzai

Read `AGENTS.md` first — it has project-specific conventions, structure, and build commands.

## How the Skills and Roles System Works

1. Check `.kbz/stage-bindings.yaml` — it maps each workflow stage to a role and skill.
2. Read the role (`.kbz/roles/*.yaml`) for identity, vocabulary, and anti-patterns.
3. Read the skill (`SKILL.md`) for the procedure, checklist, and evaluation criteria.

## Roles

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

## Stages, Skills, and Gates

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

## System Skills (`.agents/skills/`)

| Skill | When to use |
|-------|-------------|
| `kanbanzai-getting-started` | Start of every session |
| `kanbanzai-workflow` | Stage gates, lifecycle, when to stop and ask |
| `kanbanzai-agents` | Task dispatch, commits, knowledge, sub-agents |
| `kanbanzai-documents` | Creating, registering, approving documents |
| `kanbanzai-planning` | Scoping work, feature vs plan decisions |

## Critical Rules

- **Check `git status` before every task.** Commit previous work first; do not use `git stash` (stashing hides state from parallel agents and is silently lost across worktree switches).
- **Use MCP tools, not raw file reads** for `.kbz/state/` queries — use `entity`, `doc`, `status`, `knowledge`.
- **`.kbz/state/` files are versioned project state** — commit them alongside code. Never stash, discard, or `.gitignore` them.
- **Follow commit format:** `type(scope): description` (see `kanbanzai-agents` skill).
- **Do not skip workflow stages.** Check stage gate prerequisites before advancing features.
- **Use graph tools** (`search_graph`, `query_graph`, `trace_call_path`) over `grep` for structural code questions. Project: `Users-samphillips-Dev-kanbanzai`.

## Build Commands

```
go build ./...          # build
go test ./...           # test
go test -race ./...     # test with race detector
go vet ./...            # static analysis
```
