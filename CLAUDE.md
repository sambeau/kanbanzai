# Claude Code Instructions for Kanbanzai

Read `AGENTS.md` first — it has project-specific conventions, structure, and build commands.

## How the Skills and Roles System Works

1. Check `.kbz/stage-bindings.yaml` — it maps each workflow stage to a role and skill.
2. Read the role (`.kbz/roles/*.yaml`) for identity, vocabulary, and anti-patterns.
3. Read the skill (`SKILL.md`) for the procedure, checklist, and evaluation criteria.

## Roles

| Role | Stage |
|------|-------|
| `architect` | designing, dev-planning |
| `spec-author` | specifying |
| `implementer-go` | developing (sub-agent) |
| `orchestrator` | developing, reviewing |
| `reviewer-conformance` | reviewing, plan-reviewing |
| `reviewer-quality` | reviewing |
| `reviewer-security` | reviewing |
| `reviewer-testing` | reviewing |
| `researcher` | researching |
| `documenter` | documenting |

## Task-Execution Skills (`.kbz/skills/`)

| Skill | Stage |
|-------|-------|
| `write-design` | designing |
| `write-spec` | specifying |
| `write-dev-plan` | dev-planning |
| `decompose-feature` | dev-planning |
| `orchestrate-development` | developing |
| `implement-task` | developing (sub-agent) |
| `review-code` | reviewing |
| `orchestrate-review` | reviewing |
| `review-plan` | plan-reviewing |
| `write-research` | researching |
| `update-docs` | documenting |

## System Skills (`.agents/skills/`)

| Skill | When to use |
|-------|-------------|
| `kanbanzai-getting-started` | Start of every session |
| `kanbanzai-workflow` | Stage gates, lifecycle, when to stop and ask |
| `kanbanzai-agents` | Task dispatch, commits, knowledge, sub-agents |
| `kanbanzai-documents` | Creating, registering, approving documents |
| `kanbanzai-planning` | Scoping work, feature vs plan decisions |

## Critical Rules

- **Check `git status` before every task.** Commit or stash previous work first.
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
