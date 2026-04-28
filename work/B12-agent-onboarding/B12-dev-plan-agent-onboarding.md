# Agent Onboarding and Skill Discovery — Dev Plan

| Document | Agent Onboarding Dev Plan                          |
|----------|----------------------------------------------------|
| Status   | Draft                                              |
| Created  | 2026-03-30                                         |
| Plan     | P12-agent-onboarding                               |
| Design   | `work/design/agent-onboarding.md`                  |
| Spec     | `work/spec/agent-onboarding.md`                    |

---

## 1. Overview

Five features, decomposed into five tasks with clear file boundaries. Tasks 1–4
have no mutual dependencies and can run in parallel. Task 5 depends on all of them.

---

## 2. Task Summary

| Task | Feature | Summary | Files touched | Depends on |
|------|---------|---------|---------------|------------|
| T1 | A + B | AGENTS.md and copilot-instructions generation | `internal/kbzinit/agents_md.go` (new), `internal/kbzinit/init.go`, `cmd/kanbanzai/init_cmd.go`, `internal/kbzinit/agents_md_test.go` (new) | — |
| T2 | C | Skill content updates (MCP-tools-only rule) | `internal/kbzinit/skills/getting-started/SKILL.md`, `internal/kbzinit/skills/workflow/SKILL.md` | — |
| T3 | D | New kanbanzai-specification skill | `internal/kbzinit/skills/specification/SKILL.md` (new), `internal/kbzinit/skills.go` | — |
| T4 | E | MCP orientation breadcrumbs | `internal/mcp/status_tool.go`, `internal/mcp/next_tool.go`, `internal/mcp/status_tool_test.go`, `internal/mcp/next_tool_test.go` | — |
| T5 | All | Documentation updates and integration test | `docs/getting-started.md`, `internal/kbzinit/init_test.go` | T1, T2, T3, T4 |

---

## 3. Task Details

### T1: AGENTS.md and Copilot Instructions Generation

**Features:** A (AGENTS.md) + B (Copilot instructions)

**What to build:**

1. **New file `internal/kbzinit/agents_md.go`** containing:
   - A `const agentsMDVersion = 1` matching the marker version.
   - An embedded string (or `//go:embed` file) for the AGENTS.md template content.
     The template MUST include: managed marker comment, title, "Before You Do Anything"
     section, rules section, and skills reference table. All skills including the new
     `kanbanzai-specification` skill MUST appear in the table. ≤50 lines.
   - An embedded string for the `.github/copilot-instructions.md` template content.
     MUST include: managed marker, title, pointer to AGENTS.md, quick reference. ≤25 lines.
   - A `writeAgentsMD(baseDir string) error` method on `*Initializer` implementing
     the version-aware conflict logic from spec §3.4:
     - No file → create.
     - File exists, `<!-- kanbanzai-managed: vN -->` on line 1, N < current → overwrite.
     - File exists, marker present, N >= current → no-op.
     - File exists, no marker → skip + print warning.
   - A `writeCopilotInstructions(baseDir string) error` method with the same conflict
     logic, plus `.github/` directory creation if absent (spec §4.2).
   - A shared `readMarkdownManagedVersion(filePath string) (int, bool, error)` helper
     that parses the first line for `<!-- kanbanzai-managed: vN -->` and returns
     (version, managed, error).

2. **Modify `internal/kbzinit/init.go`:**
   - Add `SkipAgentsMD bool` to the `Options` struct.
   - In `runNewProject`: call `writeAgentsMD` and `writeCopilotInstructions` after
     the MCP config block, gated on `!opts.SkipAgentsMD`.
   - In `runExistingProject`: same insertion point and gate.

3. **Modify `cmd/kanbanzai/init_cmd.go`:**
   - Add `--skip-agents-md` to the flag switch and usage text.

4. **New file `internal/kbzinit/agents_md_test.go`** with tests for:
   - New project creates both files (AC-A1, AC-B1).
   - Managed marker is on line 1 (AC-A2, AC-B3).
   - Content requirements present (AC-A3, AC-A4, AC-A5, AC-B4).
   - Line count limits (AC-A6, AC-B5).
   - Idempotency: second run does not modify files (AC-A7).
   - Older version gets overwritten (AC-A8).
   - Non-managed file skipped with warning (AC-A9, AC-B6).
   - `--skip-agents-md` prevents creation of both files (AC-A10, AC-B7).
   - `.github/` directory created when absent (AC-B2).
   - Existing `.github/` contents not disturbed (AC-B8).

**Acceptance criteria covered:** AC-A1 through AC-A11, AC-B1 through AC-B8.

---

### T2: Skill Content Updates

**Feature:** C (Skill content updates)

**What to build:**

1. **Modify `internal/kbzinit/skills/getting-started/SKILL.md`:**
   - Add a new numbered step to the "Session Start" section (after step 3, "Check the
     work queue", before step 4):
   - Title: something like "Use kanbanzai tools for workflow operations"
   - Content per spec §5.1: do not create documents by writing files directly, use
     `doc` and `entity` tools, if MCP tools are unavailable report the issue rather
     than falling back to direct writes.

2. **Modify `internal/kbzinit/skills/workflow/SKILL.md`:**
   - Add a new bullet to the "Emergency Brake" list per spec §5.2: stop and ask when
     about to create a document in `work/` or entity in `.kbz/state/` without using
     the corresponding MCP tool.

3. **Verify** both modified skills pass the doc-currency health checker (no stale tool
   name references introduced).

**Acceptance criteria covered:** AC-C1 through AC-C5.

**Constraints:**
- Additions only — no existing content deleted or reorganised.
- Tone and formatting must match the surrounding text.

---

### T3: Specification Skill

**Feature:** D (Specification skill)

**What to build:**

1. **New file `internal/kbzinit/skills/specification/SKILL.md`:**
   - YAML frontmatter with `name: kanbanzai-specification`, description following
     activation-trigger pattern, `metadata.kanbanzai-managed: "true"`,
     `metadata.version: "0.2.0"`.
   - All sections per spec §6.3: Purpose, When to Use, Roles, Process, What a Good
     Specification Contains, Acceptance Criteria Quality Bar, Approved Specification
     Invariant, Relationship to Design, Gotchas, Related.
   - Explicitly states "it works correctly" is not an acceptance criterion (AC-D5).
   - References `kanbanzai-design`, `kanbanzai-workflow`, `kanbanzai-documents`,
     `kanbanzai-agents` in Related section.
   - ≤200 lines.
   - Follows the structure and tone of `kanbanzai-design/SKILL.md`.

2. **Modify `internal/kbzinit/skills.go`:**
   - Add `"specification"` to the `skillNames` slice (in alphabetical order among the
     existing entries).

3. **Verify** the new skill passes the doc-currency health checker.

**Acceptance criteria covered:** AC-D1 through AC-D9.

---

### T4: MCP Orientation Breadcrumbs

**Feature:** E (MCP orientation breadcrumbs)

**What to build:**

1. **Modify `internal/mcp/status_tool.go`:**
   - Add an `Orientation` field to the `projectOverview` struct:
     ```
     Orientation *orientationInfo `json:"orientation,omitempty"`
     ```
   - Define `orientationInfo`:
     ```
     type orientationInfo struct {
         Message    string `json:"message"`
         SkillsPath string `json:"skills_path"`
     }
     ```
   - In `synthesiseProject`, always populate the `Orientation` field with the message
     and path from spec §7.1.

2. **Modify `internal/mcp/next_tool.go`:**
   - In `nextQueueMode`: when the queue is empty (zero ready tasks), add the
     `orientation` key to the returned map. When the queue is non-empty, do not
     include it.
   - In `nextClaimMode`: do not include `orientation` in the returned map.

3. **Add tests in `internal/mcp/status_tool_test.go`:**
   - Project overview includes `orientation` field (AC-E1).
   - Message references getting-started skill path (AC-E2).
   - `skills_path` value is `.agents/skills/` (AC-E3).
   - Existing fields unchanged (AC-E7).

4. **Add tests in `internal/mcp/next_tool_test.go`:**
   - Empty queue includes `orientation` (AC-E4).
   - Non-empty queue excludes `orientation` (AC-E5).
   - Claim mode excludes `orientation` (AC-E6).

**Acceptance criteria covered:** AC-E1 through AC-E7.

---

### T5: Documentation Updates and Integration Test

**Features:** All (cross-cutting)

**Depends on:** T1, T2, T3, T4.

**What to build:**

1. **Modify `docs/getting-started.md`:**
   - Add `AGENTS.md` and `.github/copilot-instructions.md` to the directory structure
     diagram in "Initialising a project".
   - Add `.agents/skills/kanbanzai-specification/` to the skills listing.
   - Mention `--skip-agents-md` in the flags context.
   - Add a brief note explaining that AGENTS.md orients AI agents to the kanbanzai
     workflow.

2. **Add integration test in `internal/kbzinit/init_test.go`:**
   - Run `kbz init` on a temp directory and verify (spec §11.2):
     - `AGENTS.md` exists with managed marker.
     - `.github/copilot-instructions.md` exists and references AGENTS.md.
     - `.agents/skills/kanbanzai-specification/SKILL.md` exists.
     - Getting-started and workflow skills contain the MCP-tools rule text.
     - Second `kbz init` run does not modify any of these files (idempotency).
   - Covers AC-INT-1 through AC-INT-5.

3. **Optionally update `docs/mcp-tool-reference.md`:**
   - Document the `orientation` field in `status` and `next` response schemas.

**Acceptance criteria covered:** AC-INT-1 through AC-INT-5, spec §9.

---

## 4. Parallelism

```
T1 (AGENTS.md + copilot) ──┐
T2 (skill updates)         ──┤
T3 (specification skill)   ──┼── T5 (docs + integration)
T4 (MCP breadcrumbs)       ──┘
```

T1–T4 have no file overlap and no data dependencies. They can be implemented in
parallel by separate agents or sequentially in any order. T5 must run after all
four are complete because the integration test verifies the combined result.

---

## 5. File Ownership Summary

This table ensures no two tasks touch the same file:

| File | Task |
|------|------|
| `internal/kbzinit/agents_md.go` (new) | T1 |
| `internal/kbzinit/agents_md_test.go` (new) | T1 |
| `internal/kbzinit/init.go` | T1 |
| `cmd/kanbanzai/init_cmd.go` | T1 |
| `internal/kbzinit/skills/getting-started/SKILL.md` | T2 |
| `internal/kbzinit/skills/workflow/SKILL.md` | T2 |
| `internal/kbzinit/skills/specification/SKILL.md` (new) | T3 |
| `internal/kbzinit/skills.go` | T3 |
| `internal/mcp/status_tool.go` | T4 |
| `internal/mcp/next_tool.go` | T4 |
| `internal/mcp/status_tool_test.go` | T4 |
| `internal/mcp/next_tool_test.go` | T4 |
| `docs/getting-started.md` | T5 |
| `docs/mcp-tool-reference.md` | T5 |
| `internal/kbzinit/init_test.go` | T5 |

---

## 6. Estimation

| Task | Estimate | Rationale |
|------|----------|-----------|
| T1 | 5 | New file with template + writer + conflict logic + flag plumbing + 11 test cases |
| T2 | 1 | Two targeted additions to existing skill files |
| T3 | 3 | New skill file (~150 lines of carefully structured content) + skillNames update |
| T4 | 3 | Two MCP tool modifications + 7 test cases across two test files |
| T5 | 2 | Doc updates + integration test wiring existing pieces together |
| **Total** | **14** | |

---

## 7. Risk Notes

- **T1 is the critical path.** It has the most new code and the most test cases. If
  anything slips, T1 is where it will happen. The markdown marker parsing (HTML comment
  on line 1) is simple but must handle edge cases: BOM, trailing whitespace, Windows
  line endings.
- **T3 content quality matters.** The specification skill will be read by every agent
  writing a spec in every kanbanzai project. Poor content propagates widely. This task
  benefits from human review before merge.
- **T4 is low risk.** Adding a field to an existing response struct is a well-understood
  pattern in this codebase.