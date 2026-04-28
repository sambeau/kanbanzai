# Skills Content Specification

| Document | Skills Content Specification                        |
|----------|-----------------------------------------------------|
| Status   | Draft                                               |
| Feature  | FEAT-01KMKRQSD1TKK                                 |
| Created  | 2026-04-02                                          |
| Related  | `work/design/workflow-design-basis.md`              |
|          | `work/design/agent-interaction-protocol.md`         |
|          | `work/design/document-centric-interface.md`         |

---

## 1. Purpose

This specification defines the acceptance criteria for the six skill files that constitute the primary agent-facing documentation for the Kanbanzai workflow system. Skills are installed by `kanbanzai init` into `.agents/skills/` and are the mechanism by which agents learn how to operate within the workflow.

This spec is written retroactively to formalize delivery requirements after an initial authoring pass and review that identified 8 blockers (fabricated lifecycle states, stale 1.0 tool names).

---

## 2. Scope

### 2.1 In scope

- Six skill files, each at `.agents/skills/{name}/SKILL.md`
- YAML frontmatter metadata for each skill
- Reference files attached to specific skills
- Correct use of the 2.0 MCP tool surface throughout all content
- Correct lifecycle states for all entity types

### 2.2 Out of scope

- The `kanbanzai init` command implementation (separate feature)
- Skill discovery or loading mechanisms in the MCP server
- Automated validation of skill content against the codebase
- Per-project skill customization

---

## 3. The Six Skills

| # | Skill name                   | Purpose                                              |
|---|------------------------------|------------------------------------------------------|
| 1 | `kanbanzai-getting-started`  | Session orientation — minimal bootstrap for any agent |
| 2 | `kanbanzai-workflow`         | Stage gates, lifecycle reference, when to stop and ask |
| 3 | `kanbanzai-documents`        | Document types, registration, approval workflow       |
| 4 | `kanbanzai-agents`           | Context assembly, dispatch/completion, commits, knowledge, sub-agents |
| 5 | `kanbanzai-planning`         | Planning conversations, scope decisions, ambition principle |
| 6 | `kanbanzai-design`           | Design process, draft→approve workflow, quality principles |

---

## 4. 2.0 MCP Tool Surface

All tool references in skill content must use the consolidated 2.0 tool names exclusively. No 1.0 tool names may appear anywhere in skill files.

The complete 2.0 tool set: `branch`, `checkpoint`, `cleanup`, `conflict`, `decompose`, `doc`, `doc_intel`, `entity`, `estimate`, `finish`, `handoff`, `health`, `incident`, `knowledge`, `merge`, `next`, `pr`, `profile`, `retro`, `server_info`, `status`, `worktree`.

Key mappings from retired 1.0 names:

| Retired 1.0 name             | 2.0 equivalent                              |
|------------------------------|---------------------------------------------|
| `work_queue`                 | `next` (without ID)                         |
| `dispatch_task`              | `next` (with task ID)                       |
| `context_assemble`           | `next` (with task ID) or `handoff`          |
| `complete_task`              | `finish`                                    |
| `knowledge_contribute`       | `knowledge` action: `contribute`            |
| `list_entities_filtered`     | `entity` action: `list` or `status`         |
| `update_status`              | `entity` action: `transition`               |
| `doc_record_submit`          | `doc` action: `register`                    |
| `doc_record_approve`         | `doc` action: `approve`                     |
| `doc_record_get`             | `doc` action: `get`                         |
| `doc_record_refresh`         | `doc` action: `refresh`                     |
| `doc_record_supersede`       | `doc` action: `supersede`                   |
| `batch_import_documents`     | `doc` action: `import`                      |
| `context_report`             | Removed — use `knowledge` confirm/flag      |

---

## 5. Entity Lifecycles

The lifecycle reference must match `internal/validate/lifecycle.go` exactly.

**Feature:** proposed → designing → specifying → dev-planning → developing → reviewing → done
- Terminals: superseded, cancelled
- needs-rework: from reviewing back to developing or reviewing

**Task:** queued → ready → active → done
- active → blocked → active
- active → needs-review → done / needs-rework
- active → needs-rework → active
- Terminals: done, not-planned, duplicate

**Bug:** reported → triaged → reproduced → planned → in-progress → needs-review → verified → closed
- triaged → cannot-reproduce → triaged
- needs-review → needs-rework → in-progress
- Terminals: closed, duplicate, not-planned

**Plan:** proposed → designing → active → reviewing → done
- reviewing → active (rework)
- Terminals: superseded, cancelled (from any non-terminal, including done)

---

## 6. Acceptance Criteria

### File Structure

**AC-01** — All six skill files exist at `.agents/skills/{name}/SKILL.md` where `{name}` is one of: `kanbanzai-getting-started`, `kanbanzai-workflow`, `kanbanzai-documents`, `kanbanzai-agents`, `kanbanzai-planning`, `kanbanzai-design`.

**AC-02** — Each skill file begins with valid YAML frontmatter (delimited by `---`) containing at minimum a `name` and `description` field.

**AC-03** — Each skill file contains a "When to Use" section that clearly states when an agent should read that skill.

**AC-04** — Each skill file contains a "Related" section that links to other relevant skills by their correct skill names.

### Content Accuracy

**AC-05** — `kanbanzai-getting-started` is under 70 lines total (including frontmatter). It provides minimal session bootstrap only.

**AC-06** — Each skill's content matches the intent defined in the design document for that skill's topic area. No skill fabricates workflow concepts, entity types, or behaviors that do not exist in the system.

**AC-07** — `kanbanzai-workflow` covers stage gates, lifecycle states, and when to stop and ask a human. It does not duplicate the full lifecycle tables inline but references the lifecycle reference file.

**AC-08** — `kanbanzai-agents` covers context assembly via `next`/`handoff`, task dispatch and completion via `next`/`finish`, commit policy, knowledge contribution, and sub-agent delegation.

**AC-09** — `kanbanzai-planning` covers planning conversations, scope decisions, and the ambition principle.

**AC-10** — `kanbanzai-design` covers the design process, draft-to-approve document workflow, and quality principles.

**AC-11** — `kanbanzai-documents` covers document types, registration via `doc`, and the approval workflow.

### Tool References

**AC-12** — No skill file contains any retired 1.0 tool name (including but not limited to: `work_queue`, `dispatch_task`, `context_assemble`, `complete_task`, `knowledge_contribute`, `list_entities_filtered`, `update_status`, `doc_record_submit`, `doc_record_approve`, `doc_record_get`, `doc_record_refresh`, `doc_record_supersede`, `batch_import_documents`, `context_report`).

**AC-13** — Every MCP tool referenced in any skill file is a valid 2.0 tool name from the set defined in §4.

### Lifecycle Reference

**AC-14** — The file `.agents/skills/kanbanzai-workflow/references/lifecycle.md` exists and contains the complete lifecycle state machine for all four entity types: feature, task, bug, and plan.

**AC-15** — Every lifecycle state, transition, and terminal state in the lifecycle reference file matches `internal/validate/lifecycle.go` exactly. No states are fabricated; no valid states are omitted.

### Design Quality Reference

**AC-16** — The file `.agents/skills/kanbanzai-design/references/design-quality.md` exists and contains design quality principles and criteria.

### Cross-References

**AC-17** — Skills that reference other skills use the correct skill names from AC-01. No skill references a non-existent skill.

**AC-18** — Skills that reference entity types, lifecycle states, or document types use the actual names from the codebase — not paraphrases or approximations.

### Quality

**AC-19** — No skill contains placeholder text, TODO markers, or incomplete sections.

**AC-20** — Skills are written for an AI agent audience. They are direct, procedural, and unambiguous. They tell agents what to do, not what the system philosophy is.

---

## 7. Verification

Verification is manual review against each acceptance criterion. The reviewer must:

1. Confirm all 6 files and 2 reference files exist at the specified paths
2. Validate YAML frontmatter parses correctly
3. Grep all skill files for any retired 1.0 tool name from the mapping table
4. Compare the lifecycle reference file against `internal/validate/lifecycle.go`
5. Confirm `kanbanzai-getting-started` line count is under 70
6. Verify cross-skill references resolve to actual skill names