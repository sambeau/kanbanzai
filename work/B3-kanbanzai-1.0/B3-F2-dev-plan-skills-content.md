# Skills Content: Feature Dev-Plan

| Document | Skills Content Dev-Plan                      |
|----------|----------------------------------------------|
| Feature  | FEAT-01KMKRQSD1TKK                           |
| Status   | Draft                                        |
| Created  | 2026-03-26                                   |
| Spec     | `work/design/skills-content.md` (design serves as spec) |

---

## 1. Overview

Author the six skill files installed by `kanbanzai init`. Each file is a standalone
Markdown document with YAML frontmatter that an AI agent reads to orient itself in a
Kanbanzai-managed project. The files live under `.agents/skills/` in the target project
after `kanbanzai init` runs.

This feature is **Wave 1** and has no dependencies. It must be completed before
`init-command` (FEAT-01KMKRQRRX3CC) can embed the skill files into the binary.

---

## 2. Deliverables

Six skill files authored and reviewed, to be embedded in the binary by the `init-command`
feature. Canonical source locations during development:

| Skill name                  | Source path                              |
|-----------------------------|------------------------------------------|
| `kanbanzai-getting-started` | `internal/init/skills/getting-started/SKILL.md` |
| `kanbanzai-workflow`        | `internal/init/skills/workflow/SKILL.md` |
| `kanbanzai-documents`       | `internal/init/skills/documents/SKILL.md` |
| `kanbanzai-agents`          | `internal/init/skills/agents/SKILL.md`   |
| `kanbanzai-planning`        | `internal/init/skills/planning/SKILL.md` |
| `kanbanzai-design`          | `internal/init/skills/design/SKILL.md`   |

Each file must include the YAML frontmatter block specified in `work/design/skills-content.md §2`,
including the `kanbanzai-managed` marker and `version` placeholder.

---

## 3. Tasks

### T1 · `author-getting-started-skill`

Author `kanbanzai-getting-started/SKILL.md`.

Content (per design §3.1):
- What Kanbanzai is and how the MCP server works
- How to verify the MCP connection is active
- The `kbz status` command (or equivalent health check)
- Signpost to the `kanbanzai-workflow` skill for next steps

### T2 · `author-workflow-skill`

Author `kanbanzai-workflow/SKILL.md`.

Content (per design §3.2):
- The six stage-gate model (planning → design → features → spec → dev-plan → tasks)
- Human-AI responsibility split: humans own intent, agents own execution
- When to stop and ask the human vs. when to proceed autonomously
- Links to `kanbanzai-planning` and `kanbanzai-design` for opinionated guidance

### T3 · `author-documents-skill`

Author `kanbanzai-documents/SKILL.md`.

Content (per design §3.3):
- Document types and where they live (`work/design/`, `work/spec/`, `work/dev/`, etc.)
- How to register a document with `doc_record_submit`
- When to use `batch_import_documents`
- Drift detection and `doc_record_refresh`
- The document approval lifecycle (draft → approved)

### T4 · `author-agents-skill`

Author `kanbanzai-agents/SKILL.md`.

Content (per design §3.4):
- Agent roles and context profiles
- How to assemble context with `context_assemble`
- How to dispatch and complete tasks
- Sub-agent delegation conventions (propagating codebase-memory context)
- The agent interaction protocol summary (normalize before commit, ask on ambiguity)

### T5 · `author-planning-skill`

Author `kanbanzai-planning/SKILL.md` (opinionated skill).

Content (per design §4.1):
- How to create a Plan (`create_plan`, prefix registry)
- How to create Features under a Plan
- How to check the work queue (`work_queue`)
- Task lifecycle: ready → active → done
- Decision recording (`record_decision`)

### T6 · `author-design-skill`

Author `kanbanzai-design/SKILL.md` (opinionated skill).

Content (per design §4.2):
- How to create and register a design document
- The design → spec → dev-plan progression
- When to use `decompose_feature`
- How to approve a document (`doc_record_approve`)
- Emergency brake: stop and ask before making architecture decisions

---

## 4. Acceptance

All six tasks complete when:
- Each skill file exists at its source path with correct frontmatter (name, description,
  metadata block with `kanbanzai-managed` marker and `version` placeholder).
- Content covers all topics listed in the design doc for that skill.
- Progressive disclosure is respected: essential information first, advanced guidance
  later in the file.
- Files have been reviewed and are ready to embed.