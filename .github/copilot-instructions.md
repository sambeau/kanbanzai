# Copilot Instructions for Kanbanzai

This project is a Git-native workflow system for human-AI collaborative software development.
It has an MCP server (`kanbanzai serve`) that provides structured workflow tools — use those
tools instead of reading `.kbz/` state files directly.

## Start here

Read `AGENTS.md` in the repository root. It contains project-specific conventions, repository
structure, build commands, Git discipline rules, and a required pre-task checklist.

## Skills

This project has agent skills in two locations. Read the relevant skill when working in its domain.

### Kanbanzai workflow skills (`.agents/skills/`)

| Skill | Path | When to use |
|-------|------|-------------|
| **Getting started** | `.agents/skills/kanbanzai-getting-started/SKILL.md` | Start of every session — orientation and work queue |
| **Workflow** | `.agents/skills/kanbanzai-workflow/SKILL.md` | Stage gates, lifecycle transitions, when to stop and ask |
| **Agents** | `.agents/skills/kanbanzai-agents/SKILL.md` | Task dispatch, commits, knowledge, sub-agent spawning |
| **Documents** | `.agents/skills/kanbanzai-documents/SKILL.md` | Creating, registering, approving documents |
| **Design** | `.agents/skills/kanbanzai-design/SKILL.md` | Design documents, alternatives, quality assessment |
| **Planning** | `.agents/skills/kanbanzai-planning/SKILL.md` | Scoping work, feature vs plan decisions |
| **Code review** | `.agents/skills/kanbanzai-code-review/SKILL.md` | Review procedure, finding classification, verdicts |
| **Plan review** | `.agents/skills/kanbanzai-plan-review/SKILL.md` | Plan completion verification, delivery review |

### Codebase knowledge graph skills (`.github/skills/`)

| Skill | Path | When to use |
|-------|------|-------------|
| **Exploring** | `.github/skills/codebase-memory-exploring/SKILL.md` | Codebase orientation, architecture understanding |
| **Tracing** | `.github/skills/codebase-memory-tracing/SKILL.md` | Call chain analysis, impact assessment |
| **Quality** | `.github/skills/codebase-memory-quality/SKILL.md` | Code quality analysis patterns |
| **Reference** | `.github/skills/codebase-memory-reference/SKILL.md` | Full tool reference for graph tools |
| **Planning** | `.github/skills/planning-excellence/SKILL.md` | Planning philosophy and design quality |

## Critical rules

- **Check `git status` before every task.** Commit or stash previous work first.
- **Do not read `.kbz/` files directly.** Use MCP tools (`entity`, `doc`, `status`, `knowledge`, etc.).
- **Use graph tools over grep** for structural code questions. Project: `Users-samphillips-Dev-kanbanzai`.
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