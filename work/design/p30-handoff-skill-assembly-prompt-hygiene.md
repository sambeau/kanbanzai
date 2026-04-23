# Design: Handoff Skill Assembly and Prompt Hygiene

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-23                    |
| Status | Draft                         |
| Author | Claude Sonnet 4.6 (architect) |
| Plan   | P30-handoff-skill-assembly-prompt-hygiene |

---

## Overview

The `handoff` tool's 3.0 pipeline always loads the `orchestrate-development` skill when a feature is in `developing` status, regardless of whether the caller passes `role: implementer-go`. This means every implementer sub-agent receives orchestrator-level procedural content (~3,300 words) instead of the `implement-task` skill (~2,800 words). Additionally, `implement-task/SKILL.md` still recommends heredoc as the primary method for writing Go source files — a method that fails on any file containing embedded double-quoted strings, which is virtually all Go files.

This design specifies fixes for both defects: (1) route skill selection through the binding's `sub_agents` block when the caller's role matches a sub-agent role, and (2) replace heredoc with `write_file(entity_id: ...)` as the sole recommended Go file-write method in `implement-task/SKILL.md`.

## Goals and Non-Goals

**Goals:**
- When `handoff(task_id: T, role: implementer-go)` is called for a `developing`-stage feature, the returned prompt contains `implement-task` skill content, not `orchestrate-development`.
- `implement-task/SKILL.md` contains no heredoc recommendation for Go source files.
- The fix requires no changes to the `handoff` tool's public API or to any orchestration skill's dispatch instructions.
- The binding registry remains the single source of truth for role-to-skill mapping (WP-5).

**Non-Goals:**
- Changing the `handoff` tool's parameter set or response format.
- Modifying `stage-bindings.yaml` content or schema.
- Addressing the state persistence O(n) scan (P29).
- Addressing lifecycle gate enforcement (P31).
- Modifying any skill file other than `implement-task/SKILL.md`.

## Dependencies

- No dependency on P29 or P31. This plan can proceed and ship independently.
- `internal/binding/` package must expose the `sub_agents.roles` and `sub_agents.skills` fields as parsed struct members. If they are not already parsed, a minor struct extension is required — this is an internal implementation detail, not an external dependency.

---

## Related Work

### Prior documents consulted

| Document | Type | Relevance |
|----------|------|-----------|
| `work/research/p28-issues-investigation.md` | Research | Identifies prompt inflation cluster; confirms `stepLoadSkill` ignores `role` parameter at `pipeline.go:391`; confirms `implement-task/SKILL.md` still recommends heredoc as primary |
| `work/reports/retro-p28-doc-intel-polish-workflow-reliability.md` | Retrospective | Source observations: Issues 3, 5, 7; four-plan recurrence of heredoc failure |
| `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md` (P17) | Design | WP-5: "The Binding Registry Is the Decision Table" — orchestration layer looks up decisions in the binding, does not make them independently |
| `work/spec/3.0-stage-aware-context-assembly.md` (P16) | Specification | Established the 3.0 pipeline assembly model; introduced `PipelineInput.Role` field and `stepResolveRole`; does not specify sub-agent skill selection behaviour |

### Relevant decisions constraining this design

| Decision | Source | Constraint |
|----------|--------|------------|
| WP-5: Binding Registry Is the Decision Table | P17 design §2.5 | The binding registry (stage-bindings.yaml) must remain the authoritative source for role-to-skill mappings. The pipeline code must look up decisions from the registry, not encode them in Go logic. |
| Role identity ≠ skill content | P16 spec (stage-aware context assembly) | `stepResolveRole` and `stepLoadSkill` are separate pipeline steps by design. This separation must be preserved — the fix must route skill selection through the binding, not collapse role and skill into a single lookup. |
| Skills prevent bad steps | P17 design §2.1 | Skill content is the procedural guard for each stage. Loading the wrong skill removes the guard. This is the design principle that makes the current bug a correctness defect, not a cosmetic one. |

### Open questions raised by prior work

1. The P16 spec introduced `sub_agents.skills` in the binding schema but specified no pipeline step to read it. The field is documentation-only at present. This design must decide whether (a) the pipeline reads `sub_agents.skills` automatically by matching the caller's role, or (b) the caller passes the skill name explicitly as a `handoff` parameter. This question is resolved in the Alternatives section below.

2. The `implement-task/SKILL.md` heredoc recommendation has persisted through four consecutive plans despite knowledge entries documenting the failure. This design must address why documentation has failed as a mitigation and what product change will prevent recurrence.

---

## Problem and Motivation

### The wrong skill is loaded for every implementer sub-agent

The `handoff` tool's 3.0 pipeline assembles a Markdown prompt for sub-agent dispatch. When a feature is in `developing` status and an orchestrator calls `handoff(task_id: T, role: implementer-go)`, the pipeline:

1. Resolves the feature's lifecycle stage as `developing`.
2. Looks up the stage binding — which maps `developing` to `skills: [orchestrate-development]`.
3. Loads `orchestrate-development` as the skill content.
4. Resolves the `implementer-go` role for the identity header only.

The `role: implementer-go` parameter affects only the identity section ("you are implementer-go"). The procedural content — the skill — is always `orchestrate-development`, regardless of the role passed. The result is that every implementer sub-agent receives a prompt containing:

- Full orchestrator dispatch procedure (~3,300 words)
- Context compaction guidance
- Multi-feature scoping instructions
- Parallel dispatch batch logic
- Orchestrator anti-patterns and vocabulary

None of this is relevant to writing a Go function. The `implement-task` skill (~2,800 words), which contains the correct procedure for a sub-agent implementing a single task, is never loaded.

The `sub_agents.skills: [implement-task]` field exists in `stage-bindings.yaml` under the `developing` entry's `sub_agents` block, but no pipeline step reads this field.

**Measured cost:** The orchestrate-development skill is 3,336 words; implement-task is 2,803 words. The inflation is approximately 533 net additional words (~410 tokens) per dispatch from the wrong skill alone — but the qualitative cost is higher, because the orchestrator content actively misleads the sub-agent about its role and scope.

**Compounding cost for sequential chains:** A 4-task sequential chain accumulates 4× the misloaded skill content, plus the spec, code read during investigation, generated Go, and test output. P28 recorded three Sprint 2 orchestrator agents hitting the context ceiling before task 3, each requiring a six-call recovery wave (~30–40 minutes of wall-clock time).

### The heredoc recommendation is still in the skill file

`implement-task/SKILL.md` at line 92 still reads: "Go source files — use heredoc (primary)." The checklist at line 137 still reads: "use `terminal` + heredoc for Go files, `python3 -c` for Markdown/YAML, NOT `edit_file`."

This recommendation is known to be defective: when Go code contains embedded double-quoted strings (a routine occurrence), the heredoc silently truncates or escapes incorrectly. The correct alternative — `write_file(entity_id: ...)` — has been in the knowledge base since P25 and is documented in at least two skill files. Four knowledge entries (KE-01KN5CXMBWSXE and others) document the failure mode.

The root cause of recurrence is structural: the wrong method is documented as "primary" in the skill file that sub-agents actually receive. Knowledge entries and secondary skill files are not read by sub-agents who receive a `handoff` prompt — they only see the content assembled by the pipeline. As long as `implement-task/SKILL.md` says "use heredoc," sub-agents will use heredoc and fail.

**If nothing changes:** With every plan after P28, every implementer sub-agent dispatched via `handoff` will receive orchestrator skill content. Context ceiling events will continue at the current rate (3–4 per plan). The heredoc failure will generate a fifth knowledge entry documenting the same failure. The carrying cost compounds.

---

## Design

### Component overview

The fix spans two independent sub-components with no shared code paths:

1. **Pipeline skill selection** (`internal/context/pipeline.go`): add sub-agent skill routing to `stepLoadSkill`.
2. **Skill file correction** (`.kbz/skills/implement-task/SKILL.md`): replace heredoc with `write_file` as the primary and only recommended file-write method.

These are independent changes. The skill file fix can ship first; the pipeline fix requires a design decision on binding schema interpretation (resolved below).

---

### Sub-component 1: Sub-agent skill routing in the pipeline

#### Principle

Following WP-5, the binding registry remains the single source of truth. The pipeline does not hard-code which roles are "sub-agent roles" — it reads the information from the binding.

#### Binding schema (existing, no change required)

The `developing` binding already has the required structure:

```yaml
developing:
  roles: [orchestrator]
  skills: [orchestrate-development]
  sub_agents:
    roles: [implementer]
    skills: [implement-task]
```

The `sub_agents.roles` and `sub_agents.skills` fields are parallel arrays: the Nth role in `sub_agents.roles` maps to the Nth skill in `sub_agents.skills`. This mapping is already correct in the YAML. No binding file changes are required.

#### `stepLoadSkill` change

Add a role-to-sub-agent-skill lookup before falling through to the binding's primary skill:

```
stepLoadSkill(state):
  1. If state.Input.Role is non-empty:
       For i, subRole in state.Binding.SubAgents.Roles:
         If subRole matches state.Input.Role (or state.Input.Role has subRole as a prefix):
           If i < len(state.Binding.SubAgents.Skills):
             skillName = state.Binding.SubAgents.Skills[i]
             break
  2. If skillName is still empty:
       skillName = state.Binding.Skills[0]  (existing primary skill path)
  3. Load skill as before.
```

The role match uses prefix logic consistent with how the rest of the system handles role variants (e.g., `implementer-go` matching `implementer`). Specifically: `strings.HasPrefix(callerRole, subAgentRole)` where `subAgentRole` is the binding's entry.

#### Binding struct update

The `Binding` type (in `internal/binding/` or equivalent) must expose the `sub_agents` block as a parsed struct field. If `SubAgents.Roles` and `SubAgents.Skills` are not already parsed (inspection of the binding loader is required during implementation), the struct must be extended. The YAML schema itself does not change.

#### Fallback behaviour

If no sub-agent role match is found, `stepLoadSkill` falls through to the primary skill (`state.Binding.Skills[0]`). This preserves existing behaviour for orchestrator callers and for any stage binding that has no `sub_agents` block.

#### Error handling

If a sub-agent role match is found but the corresponding skill index is out of bounds (misconfigured binding), return a `pipelineError` with step `6` ("skill-loading") and a message pointing to the binding file. This is a configuration error that should surface loudly, not be silently skipped.

---

### Sub-component 2: `implement-task/SKILL.md` correction

#### Current state

Lines 90–110 of `implement-task/SKILL.md` document the heredoc as "primary" with a delimiter-collision warning as a footnote. Line 137 of the checklist repeats the heredoc instruction.

#### Required changes

Replace the "Go source files — use heredoc (primary)" section with a "Go source files — use `write_file`" section. The new section must:

1. Name `write_file(entity_id: "FEAT-xxx", path: "...", content: "...")` as the **only** recommended method for Go source files in a worktree context.
2. Name `edit_file` as the method for files outside a worktree (existing guidance, unchanged).
3. Remove all heredoc recommendations for Go source. The delimiter-collision warning and the heredoc example are removed entirely.
4. Retain the `python3 -c` note only for YAML files where `write_file` is not applicable (if any such cases remain valid).

Update the checklist at line 137 to read: "Confirmed whether this task runs inside a worktree — if yes, use `write_file(entity_id: ...)` for Go files and Markdown. Do NOT use `terminal` + heredoc."

#### Rationale for removal (not deprecation)

The heredoc approach has a fundamental failure mode for any Go file containing embedded double-quoted strings. Since virtually all Go files contain quoted strings (error messages, format strings, log output), the failure mode is not edge-case — it is the common case. Documenting it as "primary (but watch out for this)" trains sub-agents to use a method that will fail on most real Go files. The correct action is removal, not a stronger warning.

---

### What this design does NOT change

- The `handoff` tool's public API (parameters, response format) is unchanged.
- The `stage-bindings.yaml` schema and file content are unchanged.
- The legacy 2.0 assembly path in `handoff_tool.go` (the `buildLegacyResponse` branch) is unchanged — it already uses the `role` parameter differently and is not the source of this bug.
- No other skill files are modified in this plan. The `orchestrate-development` skill is not modified.
- The `next` tool and state persistence layer are out of scope (addressed by P29).

---

## Alternatives Considered

### Alternative A: Caller passes skill name explicitly as a `handoff` parameter

**Description:** Add a `skill` parameter to the `handoff` tool. The orchestrator explicitly passes `skill: implement-task` alongside `role: implementer-go`. The pipeline's `stepLoadSkill` reads this parameter directly.

**Trade-offs:**
- Simple to implement: one new parameter, one changed step.
- Eliminates any ambiguity about which skill to load.
- Requires all existing orchestration skills (and the `orchestrate-development` SKILL.md) to document the correct skill name for each sub-agent role they dispatch.
- Places skill selection responsibility on the caller (the orchestrator), which means every `handoff` call site in every orchestrator skill must be updated to include the `skill` parameter.
- **Fragile:** if a caller omits the `skill` parameter (the fallback case), the bug silently recurs.
- Contradicts WP-5: the binding is the decision table; callers should not need to know which skill to load for which role.

**Verdict:** Rejected. Callers should not need to know implementation details of sub-agent skill selection. The binding already encodes this mapping.

### Alternative B: Stage binding drives skill selection automatically, using sub_agents block (recommended)

**Description:** `stepLoadSkill` checks whether the caller's role matches a sub-agent role in the binding's `sub_agents.roles` list. If so, it loads the corresponding skill from `sub_agents.skills`. This is the design described in the Design section above.

**Trade-offs:**
- Consistent with WP-5: the binding remains the single source of truth.
- Zero changes to the `handoff` tool's public API.
- Zero changes to orchestration skills or dispatch instructions — orchestrators continue to pass `role: implementer-go` as they already do.
- Requires correct maintenance of the `sub_agents` block in stage-bindings.yaml for any stage that uses sub-agents.
- Role matching must handle variants (e.g., `implementer-go` matching `implementer`) — prefix logic is simple and consistent with existing conventions.

**Verdict:** Chosen. Correct binding-registry-first design, no API change, no caller update required.

### Alternative C: Introduce a separate `sub_agent_handoff` tool

**Description:** Create a new MCP tool specifically for sub-agent dispatch that always loads the correct sub-agent skill. The existing `handoff` tool is for orchestrator context only.

**Trade-offs:**
- Cleanly separates orchestrator and sub-agent assembly paths.
- Breaking change: all existing orchestration skills that use `handoff` with `role: implementer-go` must be updated to use the new tool name.
- Doubles the surface area for prompt assembly logic.
- Increases MCP tool count, which has direct cost on context consumption when tool lists are included in prompts.

**Verdict:** Rejected. The `handoff` tool already has a `role` parameter whose stated purpose is "context shaping." Making the `role` parameter actually shape the skill content is the right fix, not introducing a second tool.

### Alternative D: Strong warning + knowledge entry (status quo improvement)

**Description:** Add a fifth knowledge entry about the skill mismatch and update the `orchestrate-development` SKILL.md to include a note: "When dispatching implementers, pass `role: implementer-go` — note that the sub-agent will receive orchestrator skill context."

**Trade-offs:**
- Zero code change.
- Does not fix the problem — agents dispatched via `handoff` receive skill content, not knowledge entries. The warning would only be seen by the orchestrator, who cannot change what skill content the pipeline loads.
- The heredoc pattern has already demonstrated that documentation-only mitigations do not prevent recurrence.

**Verdict:** Rejected. Documentation has been tried for four consecutive plans and has not fixed the heredoc failure. The same mitigation will not fix the skill-mismatch failure.

---

## Decisions

### Decision 1: Sub-agent skill selection is driven by the binding's `sub_agents` block, not by a new caller parameter

**Decision:** `stepLoadSkill` reads `state.Binding.SubAgents.Roles` and `state.Binding.SubAgents.Skills` to select the skill when the caller's role matches a sub-agent role. No new `handoff` parameter is introduced.

**Context:** The research document identified that `sub_agents.skills: [implement-task]` already exists in the binding YAML but is never read. The WP-5 principle establishes the binding as the authoritative decision table. Two alternatives — explicit caller parameter and a new tool — were considered.

**Rationale:** Keeping the binding as the decision source is consistent with the existing architecture. Adding a caller parameter would distribute knowledge about skill selection into every orchestration skill file, creating a maintenance surface that the binding already eliminates. The existing `sub_agents` block in the binding schema encodes exactly the right information; reading it requires only a change to `stepLoadSkill`.

**Consequences:**
- Positive: orchestrators need no changes; they continue to call `handoff(task_id: T, role: implementer-go)` as today.
- Positive: if additional sub-agent roles are added in future, updating the binding automatically routes them to the correct skill.
- Negative: the `Binding` struct must expose `SubAgents.Roles` and `SubAgents.Skills` as parsed fields — a small struct extension may be required if these fields are not already parsed.

---

### Decision 2: Heredoc is removed from `implement-task/SKILL.md`, not deprecated with a warning

**Decision:** The heredoc section is removed and replaced with `write_file(entity_id: ...)` as the sole recommended method. The delimiter-collision warning is also removed. The checklist is updated accordingly.

**Context:** The heredoc approach fails on any Go file containing embedded double-quoted strings, which is the common case for all Go files (error messages, format strings). Four consecutive plans have documented this failure. Three previous knowledge entries and two skill file notes have not prevented recurrence, because sub-agents receive skill content via the `handoff` prompt and do not consult knowledge entries or secondary files at dispatch time.

**Rationale:** A "use heredoc (primary) but watch for this edge case" instruction trains agents to use a method that fails on routine Go code. Since the failure mode is not edge-case but common-case, the correct action is removal. `write_file(entity_id: ...)` is the correct tool for this purpose: it handles embedded quotes natively, does not require shell escaping, and is already used correctly by Sprint 2 sub-agents in P28. Deprecating rather than removing would preserve the defective pattern in the instruction set.

**Consequences:**
- Positive: heredoc failure cannot recur from skill content — the instruction no longer exists.
- Positive: sub-agents have a single, unambiguous instruction for Go file writes.
- Negative: any sub-agent prompt assembled from a cached or stale skill file would retain the old instruction until the server is restarted and the skill is reloaded. (Skill files are read from disk at assembly time — this is not a persistent risk.)
- Neutral: the `python3 -c` note for YAML files is retained only if a valid use case remains; it is removed if `write_file` covers all remaining worktree write scenarios.

---

### Decision 3: The `stage-bindings.yaml` schema and file content are not modified

**Decision:** No changes to `.kbz/stage-bindings.yaml`. The existing `sub_agents.skills: [implement-task]` field is read by the updated pipeline without any YAML change.

**Context:** The `sub_agents` block already contains the correct role-to-skill mapping. Modifying the schema would risk breaking binding loading elsewhere or introducing a migration path.

**Rationale:** The field is already present and correctly specified. The bug is in the code that reads the binding, not in the binding itself. Keeping the YAML unchanged also means this design does not affect any other component that parses stage-bindings.yaml.

**Consequences:**
- Positive: no migration risk, no schema versioning required.
- Positive: the fix is purely additive to the Go code reading the binding.
- Neutral: if a future stage binding needs a different sub-agent skill mapping structure, the current schema may need extension at that time.