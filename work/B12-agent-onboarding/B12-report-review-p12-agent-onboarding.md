# Review: P12 — Agent Onboarding and Skill Discovery

| Field    | Value                                              |
|----------|----------------------------------------------------|
| Plan     | P12-agent-onboarding                              |
| Reviewer | Claude Sonnet 4.6                                 |
| Date     | 2026-03-30                                        |
| Verdict  | **Pass**                                          |

---

## Summary

P12 delivered five features making kanbanzai self-orienting for AI agents: `AGENTS.md`
and `.github/copilot-instructions.md` generation on `kbz init`, MCP-tools-only rules
added to the getting-started and workflow skills, a new `kanbanzai-specification` skill,
and an `orientation` breadcrumb field in the `status` and `next` MCP responses. All
implementation is on `main` with 53 passing tests. All 45 acceptance criteria pass. Two
non-blocking SHOULD documentation items (§9.2, §9.3) were left for a future cleanup.

---

## Feature Status

| Feature | Slug | Status | Spec Conformance |
|---------|------|--------|------------------|
| FEAT-01KMZ-A2PPSFK9 | agents-md-generation    | reviewing | ✅ All criteria met (AC-A1–A11) |
| FEAT-01KMZ-A2Z98RAB | copilot-instructions    | reviewing | ✅ All criteria met (AC-B1–B8)  |
| FEAT-01KMZ-A31HYW6A | skill-content-updates   | reviewing | ✅ All criteria met (AC-C1–C5)  |
| FEAT-01KMZ-A33G1FFR | specification-skill     | reviewing | ✅ All criteria met (AC-D1–D9)  |
| FEAT-01KMZ-A34WSPRR | mcp-orientation-breadcrumbs | reviewing | ✅ All criteria met (AC-E1–E7) |

---

## Spec Conformance Detail

### Feature A: AGENTS.md Generation

Implemented in `internal/kbzinit/agents_md.go`. Content is a Go compile-time constant
(`agentsMDContent`) so it is embedded in the binary. The version-aware conflict logic is
shared with Feature B via `writeMarkdownConfig`.

| # | Criterion | Result | Evidence |
|---|-----------|--------|---------|
| AC-A1 | `kbz init` on a new project creates `AGENTS.md` at the project root | ✅ | `TestWriteAgentsMD_NewProject_CreatesFile` |
| AC-A2 | File starts with `<!-- kanbanzai-managed: v1 -->` on line 1 | ✅ | `TestWriteAgentsMD_ManagedMarkerOnLineOne`; const begins with that literal |
| AC-A3 | Contains "Before You Do Anything" with `status`, `next`, and skill path references | ✅ | `TestWriteAgentsMD_ContentRequirements_BeforeYouDoAnything`; verified in const |
| AC-A4 | Contains the three rules (MCP tools, stage gates, human approval) | ✅ | `TestWriteAgentsMD_ContentRequirements_Rules`; all three rules present |
| AC-A5 | Contains a skills reference table listing all installed skills | ✅ | `TestWriteAgentsMD_ContentRequirements_SkillsTable`; 9-row table present |
| AC-A6 | File does not exceed 50 lines | ✅ | `TestWriteAgentsMD_LineCountLimit`; content is 36 lines |
| AC-A7 | Existing managed AGENTS.md at current version → no-op | ✅ | `TestWriteAgentsMD_CurrentVersion_NoOp` |
| AC-A8 | Existing managed AGENTS.md at older version → overwrite | ✅ | `TestWriteAgentsMD_OlderVersion_Overwrites` |
| AC-A9 | Existing non-managed AGENTS.md → skip with warning | ✅ | `TestWriteAgentsMD_NonManaged_SkipsWithWarning` |
| AC-A10 | `kbz init --skip-agents-md` does not create AGENTS.md | ✅ | `TestWriteAgentsMD_SkipFlag_DoesNotCreate`; `Options.SkipAgentsMD` field wired in `init.go` |
| AC-A11 | Generated content is embedded in the binary | ✅ | `TestWriteAgentsMD_ContentIsEmbedded`; content is a package-level Go `const` |

---

### Feature B: Copilot Instructions Generation

Implemented alongside Feature A; shares `writeMarkdownConfig` for conflict logic. The
`.github/` directory is created via `os.MkdirAll` before the file write.

| # | Criterion | Result | Evidence |
|---|-----------|--------|---------|
| AC-B1 | `kbz init` creates `.github/copilot-instructions.md` | ✅ | `TestWriteCopilotInstructions_NewProject_CreatesFile` |
| AC-B2 | `.github/` directory is created if absent | ✅ | `TestWriteCopilotInstructions_CreatesGithubDir` |
| AC-B3 | File starts with `<!-- kanbanzai-managed: v1 -->` on line 1 | ✅ | `TestWriteCopilotInstructions_ManagedMarkerOnLineOne` |
| AC-B4 | File contains explicit instruction to read `AGENTS.md` | ✅ | `TestWriteCopilotInstructions_ReferencesAgentsMD`; "Read `AGENTS.md`" in content |
| AC-B5 | File does not exceed 25 lines | ✅ | `TestWriteCopilotInstructions_LineCountLimit`; content is 15 lines |
| AC-B6 | Non-managed `.github/copilot-instructions.md` → skip with warning | ✅ | `TestWriteCopilotInstructions_NonManaged_SkipsWithWarning` |
| AC-B7 | `kbz init --skip-agents-md` does not create the file | ✅ | `TestWriteCopilotInstructions_SkipFlag_FieldExists`; same `SkipAgentsMD` flag gates both files |
| AC-B8 | Existing `.github/` files are not modified | ✅ | `TestWriteCopilotInstructions_ExistingGithubDir_OtherFilesUntouched` |

---

### Feature C: Skill Content Updates

Both source files under `internal/kbzinit/skills/` were modified as additions. Existing
content was preserved in both files.

| # | Criterion | Result | Evidence |
|---|-----------|--------|---------|
| AC-C1 | getting-started skill contains MCP-tools-not-direct-write rule | ✅ | "Do not create documents in `work/`...Use `doc`...Use `entity`...report the issue to the human rather than falling back to direct file writes." in SKILL.md |
| AC-C2 | workflow skill Emergency Brake includes direct file writes | ✅ | "You are about to create a document in `work/` or modify entity state in `.kbz/state/` by writing files directly instead of using the corresponding kanbanzai MCP tool" in Emergency Brake list |
| AC-C3 | All existing content preserved — no deletions | ✅ | Both files audited; only additions made; all prior sections intact |
| AC-C4 | Updated skills pass doc-currency health checker | ✅ | `go test ./...` passes (53 tests); health check tests in `internal/health` pass |
| AC-C5 | `kbz init --update-skills` updates both skill files | ✅ | `Options.UpdateSkills` flag wired in `init.go`; `runUpdateSkills` path calls `installSkills` |

---

### Feature D: Specification Skill

New file at `internal/kbzinit/skills/specification/SKILL.md` (136 lines). Registered in
`skillNames` in `internal/kbzinit/skills.go`. The frontmatter uses the same
comment-form metadata (`# kanbanzai-managed: true`, `# kanbanzai-version: dev`) that all
other embedded skills use — consistent with established convention.

| # | Criterion | Result | Evidence |
|---|-----------|--------|---------|
| AC-D1 | File exists at correct path; installed by `kbz init` | ✅ | `internal/kbzinit/skills/specification/SKILL.md` exists; `installSkills` loop covers it |
| AC-D2 | Skill appears in `skillNames` list in `skills.go` | ✅ | `"specification"` present in `skillNames` slice |
| AC-D3 | Frontmatter has correct name, description, managed metadata | ✅ | `name: kanbanzai-specification`; multi-sentence activation description; `# kanbanzai-managed: true` |
| AC-D4 | All required sections from §6.3 present | ✅ | Purpose, When to Use, Roles, The Specification Process, What a Good Specification Contains, Acceptance Criteria Quality Bar, The Approved Specification Invariant, Relationship to Design, Gotchas, Related — all present |
| AC-D5 | Quality bar section explicitly states "it works correctly" is not an AC | ✅ | "«It works correctly» is not an acceptance criterion." in Acceptance Criteria Quality Bar section |
| AC-D6 | File does not exceed 200 lines | ✅ | 136 lines (`wc -l` verified) |
| AC-D7 | Related section references all four required skills | ✅ | `kanbanzai-design`, `kanbanzai-workflow`, `kanbanzai-documents`, `kanbanzai-agents` all listed |
| AC-D8 | Generated AGENTS.md lists `kanbanzai-specification` in skills table | ✅ | `| \`kanbanzai-specification\` | During specification work |` in `agentsMDContent` const |
| AC-D9 | Skill passes doc-currency health checker | ✅ | All health tests pass; no stale tool references introduced |

---

### Feature E: MCP Orientation Breadcrumbs

`synthesiseProject` in `status_tool.go` unconditionally sets `Orientation` on the
`projectOverview` struct. `nextQueueMode` in `next_tool.go` sets `orientation` only when
`len(queueItems) == 0`. Claim mode returns a plain `task`+`context` map with no
orientation key.

| # | Criterion | Result | Evidence |
|---|-----------|--------|---------|
| AC-E1 | `status` with no `id` returns `orientation` field | ✅ | `TestStatusTool_ProjectOverview_HasOrientation` |
| AC-E2 | `orientation.message` references getting-started skill path | ✅ | Message: `"...read .agents/skills/kanbanzai-getting-started/SKILL.md"` |
| AC-E3 | `orientation.skills_path` is `".agents/skills/"` | ✅ | `TestStatusTool_ProjectOverview_HasOrientation`; value hardcoded |
| AC-E4 | `next` with empty queue returns `orientation` | ✅ | `TestNext_EmptyQueue_HasOrientation` |
| AC-E5 | `next` with non-empty queue does NOT return `orientation` | ✅ | `TestNext_NonEmptyQueue_NoOrientation` |
| AC-E6 | `next` in claim mode does NOT return `orientation` | ✅ | `TestNext_ClaimMode_NoOrientation` |
| AC-E7 | Existing `status` and `next` fields are unchanged | ✅ | `TestStatusTool_ProjectOverview_OrientationDoesNotBreakExistingFields`; `scope`, `plans`, `total`, `health`, `attention`, `generated_at` all present |

---

### Integration Acceptance Criteria

| # | Criterion | Result | Notes |
|---|-----------|--------|-------|
| AC-INT-1 | After `kbz init`, agent can determine MCP tools, stage gates, and skill paths from AGENTS.md alone | ✅ | Rules section addresses all three; skills table provides paths |
| AC-INT-2 | GitHub Copilot agent is directed from copilot-instructions.md → AGENTS.md → skills | ✅ | copilot-instructions.md says "Read `AGENTS.md`"; AGENTS.md lists skill paths |
| AC-INT-3 | Agent calling `status` without reading any files receives orientation breadcrumb | ✅ | Orientation always present in project overview; no precondition |
| AC-INT-4 | `kanbanzai-specification` appears in AGENTS.md skills table and in workflow stage gates progression | ✅ | Skills table: `kanbanzai-specification` listed; workflow skill: Specification row in stage gates table |
| AC-INT-5 | `kbz init --skip-agents-md --skip-skills` → no AGENTS.md, no copilot instructions, no skills | ✅ | `TestP12_Integration_SkipAgentsMDAndSkipSkills`; orientation breadcrumb in `status` is sole discovery path |

---

## Test Coverage

| Scope | Tests | Result |
|-------|-------|--------|
| `readMarkdownManagedVersion` unit tests | 6 | ✅ Pass |
| Feature A (`writeAgentsMD`) | 11 | ✅ Pass |
| Feature B (`writeCopilotInstructions`) | 8 | ✅ Pass |
| Integration (both files together, idempotency) | 2 + 2 skip-flag tests | ✅ Pass |
| Feature E status orientation | 3 | ✅ Pass |
| Feature E next orientation | 3 | ✅ Pass |
| Full suite (`go test ./...`) | 53 (all packages) | ✅ Pass |

---

## Findings

### Blocking

None.

### Non-blocking

**NB-1 — `work/spec/init-command.md` §3.3 not updated (§9.2 SHOULD)**
Section 3.3 still states "AGENTS.md — never created or modified by `init`." This is now
factually incorrect. §9.2 of the agent-onboarding spec requires (SHOULD) a note pointing
to the new spec, but the note was not added. Impact: a reader of the init-command spec
will see contradictory information. Recommended fix: add a one-line note such as:
> _Note: superseded by `work/spec/agent-onboarding.md` §3–4 regarding AGENTS.md and
> `.github/copilot-instructions.md`. The exclusion below no longer applies._

**NB-2 — `docs/mcp-tool-reference.md` not updated (§9.3 SHOULD)**
The `orientation` field added to `status` and `next` responses is not documented in the
MCP tool reference. §9.3 calls for a SHOULD update. This is a docs-only gap with no
functional impact.

---

## Verdict

**Pass — approved for merge.**

All 45 acceptance criteria (AC-A1–A11, AC-B1–B8, AC-C1–C5, AC-D1–D9, AC-E1–E7,
AC-INT-1–INT-5) are satisfied. All tests pass. The two non-blocking documentation items
(NB-1, NB-2) are SHOULD-level from the spec and can be addressed in a future cleanup
pass without blocking delivery.