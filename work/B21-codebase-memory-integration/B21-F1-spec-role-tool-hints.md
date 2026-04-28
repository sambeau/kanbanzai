# Role-Scoped Tool Hints — Specification

> Specification for FEAT-01KNA-11F3BBMP (role-tool-hints)
> Plan: P21-codebase-memory-integration
> Design: work/design/role-tool-hints.md

---

## Overview

This specification defines the observable behaviour of role-scoped tool hints —
a configuration mechanism that injects machine-specific tool availability
guidance into sub-agent prompts. Hints are keyed by role ID, stored in
`local.yaml` (per-machine) and/or `config.yaml` (per-project), merged per-key
with local overriding project, resolved through role inheritance, and injected
automatically into `handoff` prompts and `next` context output. The `health`
tool surfaces merged hints for verification. The feature is additive — existing
configurations and prompts are unchanged when no hints are defined.

---

## Scope

**In scope:**
- `tool_hints` field on `Config` (`config.yaml`) and `LocalConfig` (`local.yaml`)
- Per-key merge of project and local hints (local wins on conflict)
- Role inheritance resolution when looking up a hint for a given role
- Injection into `handoff` output as an `## Available Tools` section (both
  3.0 pipeline and legacy 2.0 paths)
- Injection into `next` context output as a `tool_hint` string field
- Surfacing of merged hints in `health` tool output
- Coexistence with the Phase 2 `## Code Graph` section

**Explicitly out of scope:**
- Validation or schema enforcement of hint values
- Multiple hints per role (arrays or composition)
- Hint templating or variable substitution
- Auto-detection of available MCP tools
- Global per-machine config (`~/.config/kanbanzai/`)
- Schema version bump for `config.yaml`

---

## Functional Requirements

### Config Parsing

**FR-001:** Both `config.yaml` and `local.yaml` MUST support a `tool_hints`
field containing a map of string keys to string values. Keys are role IDs.
Values are opaque strings.

**FR-002:** The `tool_hints` field MUST be optional in both files. A file that
omits `tool_hints` entirely MUST parse identically to one that existed before
this feature was introduced.

**FR-003:** Hint values MUST be treated as opaque strings. The system MUST NOT
parse, validate, or enforce any schema on hint content.

### Merge Strategy

**FR-004:** When both `config.yaml` and `local.yaml` define `tool_hints`, the
effective hint map MUST be the per-key merge of both maps: keys present only in
`config.yaml` are used as-is, keys present only in `local.yaml` are used as-is,
and keys present in both MUST use the `local.yaml` value.

**FR-005:** When only one file defines `tool_hints`, the effective hint map
MUST be that file's map unchanged.

**FR-006:** When neither file defines `tool_hints`, the effective hint map MUST
be empty (nil or zero-length).

### Role Inheritance Resolution

**FR-007:** When resolving a hint for a given role ID, the system MUST first
check for an exact match in the merged hint map. If found, that hint MUST be
used.

**FR-008:** If no exact match exists for the role ID, the system MUST walk the
role's `inherits` chain (as defined in the role's YAML definition) and return
the hint for the first ancestor that has an entry in the merged hint map.

**FR-009:** If no hint is found after walking the full inheritance chain, the
system MUST treat the result as "no hint" and MUST NOT inject any tool hint
section into the output.

### Handoff Injection (3.0 Pipeline)

**FR-010:** When `handoff` assembles a prompt via the 3.0 pipeline and a hint
resolves for the active role, the prompt MUST contain an `## Available Tools`
section whose body is the resolved hint string.

**FR-011:** The `## Available Tools` section MUST be placed after the Identity
and Role sections but before the Procedure and Knowledge sections.

**FR-012:** When no hint resolves for the active role, the `## Available Tools`
section MUST be omitted entirely. The prompt MUST NOT contain an empty
`## Available Tools` section.

### Handoff Injection (Legacy 2.0)

**FR-013:** When `handoff` assembles a prompt via the legacy 2.0 path
(`renderHandoffPrompt`) and a hint resolves for the active role, the prompt
MUST contain an `## Available Tools` section whose body is the resolved hint
string, placed before the "Additional Instructions" section.

**FR-014:** When no hint resolves for the active role in the legacy path, the
`## Available Tools` section MUST be omitted entirely.

### Next Injection

**FR-015:** When `next` returns structured context output and a hint resolves
for the active role, the context map MUST include a `tool_hint` field whose
value is the resolved hint string.

**FR-016:** When no hint resolves for the active role, the `tool_hint` field
MUST be omitted from the context map.

### Coexistence with Phase 2

**FR-017:** When both `## Available Tools` (from tool hints) and `## Code Graph`
(from Phase 2 worktree-aware graph context) are present in a handoff prompt,
`## Available Tools` MUST appear before `## Code Graph`.

### Health Surfacing

**FR-018:** The `health` tool output MUST include a `tool_hints` section that
displays the merged hint map (after local-overrides-project merge, before role
inheritance resolution).

**FR-019:** When the merged hint map is empty, the `tool_hints` section in
`health` output MUST indicate that no hints are configured.

### Backward Compatibility

**FR-020:** Prompts generated for users who have not configured `tool_hints`
MUST be identical to prompts generated before this feature was introduced. No
new sections, no empty sections, no additional whitespace.

**FR-021:** Existing `config.yaml` and `local.yaml` files that do not contain a
`tool_hints` field MUST continue to parse without error and without any change
in behaviour.

---

## Non-Functional Requirements

**NFR-001:** The merged hint map MUST be computed once at server startup (or
config reload), not on every `handoff` or `next` call. Per-call work is limited
to role inheritance resolution.

**NFR-002:** Role inheritance resolution for a single hint lookup MUST complete
in O(d) time where d is the depth of the inheritance chain (typically ≤ 3).

**NFR-003:** The total implementation MUST NOT exceed approximately 100 lines
of Go (excluding tests), consistent with the design estimate.

**NFR-004:** No external dependencies MUST be introduced by this feature.

---

## Acceptance Criteria

**AC-001 (FR-001, FR-002):** Given a `config.yaml` with no `tool_hints` field,
when the config is parsed, then parsing MUST succeed and the effective hint map
MUST be empty.

**AC-002 (FR-001, FR-003):** Given a `local.yaml` with `tool_hints:
{implementer-go: "Use search_graph..."}`, when the config is parsed, then the
hint for key `implementer-go` MUST be the exact string `"Use search_graph..."`.

**AC-003 (FR-004):** Given `config.yaml` defines `tool_hints: {implementer:
"Use grep"}` and `local.yaml` defines `tool_hints: {implementer: "Use
search_graph"}`, when the hints are merged, then the effective hint for
`implementer` MUST be `"Use search_graph"` (local wins).

**AC-004 (FR-004, FR-005):** Given `config.yaml` defines `tool_hints:
{reviewer: "Read tracing skill"}` and `local.yaml` defines `tool_hints:
{implementer-go: "Use search_graph"}`, when the hints are merged, then the
effective map MUST contain both keys with their respective values.

**AC-005 (FR-006):** Given neither `config.yaml` nor `local.yaml` defines
`tool_hints`, when the hints are merged, then the effective map MUST be empty.

**AC-006 (FR-007):** Given the merged hints contain `{implementer-go: "Use
search_graph"}` and the active role is `implementer-go`, when a hint is
resolved, then the result MUST be `"Use search_graph"` (exact match).

**AC-007 (FR-008):** Given the merged hints contain `{implementer: "Use
search_graph"}` and the active role is `implementer-go` which inherits from
`implementer`, when a hint is resolved, then the result MUST be `"Use
search_graph"` (inherited match).

**AC-008 (FR-007, FR-008):** Given the merged hints contain both
`{implementer: "Use grep", implementer-go: "Use search_graph"}` and the active
role is `implementer-go`, when a hint is resolved, then the result MUST be
`"Use search_graph"` (exact match takes priority over inherited).

**AC-009 (FR-009):** Given the merged hints contain `{researcher: "Use
get_architecture"}` and the active role is `implementer-go` (which does not
inherit from `researcher`), when a hint is resolved, then no hint MUST be
returned.

**AC-010 (FR-010, FR-011):** Given a resolved hint exists for the active role,
when `handoff` assembles a 3.0 pipeline prompt, then the prompt MUST contain an
`## Available Tools` section with the hint text, positioned after Identity/Role
and before Procedure/Knowledge.

**AC-011 (FR-012):** Given no hint resolves for the active role, when `handoff`
assembles a 3.0 pipeline prompt, then the prompt MUST NOT contain an
`## Available Tools` section.

**AC-012 (FR-013):** Given a resolved hint exists for the active role, when
`handoff` assembles a legacy 2.0 prompt, then the prompt MUST contain an
`## Available Tools` section with the hint text, positioned before "Additional
Instructions".

**AC-013 (FR-014):** Given no hint resolves for the active role, when `handoff`
assembles a legacy 2.0 prompt, then the prompt MUST NOT contain an
`## Available Tools` section.

**AC-014 (FR-015):** Given a resolved hint exists for the active role, when
`next` returns structured context, then the context map MUST include
`tool_hint` with the resolved hint string.

**AC-015 (FR-016):** Given no hint resolves for the active role, when `next`
returns structured context, then the context map MUST NOT include a `tool_hint`
field.

**AC-016 (FR-017):** Given both a tool hint and a Phase 2 code graph context
are present, when `handoff` assembles a prompt, then `## Available Tools` MUST
appear before `## Code Graph`.

**AC-017 (FR-018):** Given `config.yaml` defines `tool_hints: {reviewer: "Read
tracing skill"}` and `local.yaml` defines `tool_hints: {implementer-go: "Use
search_graph"}`, when `health` is called, then the output MUST include a
`tool_hints` section showing both entries with their merged values.

**AC-018 (FR-019):** Given no tool hints are configured in either file, when
`health` is called, then the `tool_hints` section MUST indicate that no hints
are configured.

**AC-019 (FR-020, FR-021):** Given a project with no `tool_hints` in either
config file, when `handoff` assembles a prompt, then the prompt MUST be
byte-identical to the prompt that would have been generated before this feature
was introduced.

---

## Verification Plan

| Criterion | Method | Description |
|-----------|--------|-------------|
| AC-001 | Automated test | `TestConfigParse_NoToolHints` — parse config without `tool_hints`, assert empty map |
| AC-002 | Automated test | `TestConfigParse_WithToolHints` — parse local config with hints, assert exact string values |
| AC-003 | Automated test | `TestMergeToolHints_LocalOverridesProject` — same key in both, assert local wins |
| AC-004 | Automated test | `TestMergeToolHints_DisjointKeys` — different keys in each file, assert both present |
| AC-005 | Automated test | `TestMergeToolHints_BothEmpty` — no hints in either, assert empty result |
| AC-006 | Automated test | `TestResolveToolHint_ExactMatch` — role ID matches key directly |
| AC-007 | Automated test | `TestResolveToolHint_InheritedMatch` — child role resolves parent's hint |
| AC-008 | Automated test | `TestResolveToolHint_ExactOverInherited` — exact match takes priority over ancestor |
| AC-009 | Automated test | `TestResolveToolHint_NoMatch` — unrelated role, assert empty result |
| AC-010 | Automated test | `TestHandoffPipeline_HintInjected` — verify `## Available Tools` section present and correctly positioned |
| AC-011 | Automated test | `TestHandoffPipeline_NoHintNoSection` — verify section absent when no hint resolves |
| AC-012 | Automated test | `TestHandoffLegacy_HintInjected` — verify section present before "Additional Instructions" |
| AC-013 | Automated test | `TestHandoffLegacy_NoHintNoSection` — verify section absent in legacy path |
| AC-014 | Automated test | `TestNextContext_HintIncluded` — verify `tool_hint` field present in context map |
| AC-015 | Automated test | `TestNextContext_NoHintOmitted` — verify `tool_hint` field absent when no hint resolves |
| AC-016 | Automated test | `TestHandoff_ToolHintsBeforeCodeGraph` — verify section ordering with both present |
| AC-017 | Automated test | `TestHealth_MergedHintsDisplayed` — verify health output shows merged hint entries |
| AC-018 | Automated test | `TestHealth_NoHintsConfigured` — verify health output indicates no hints |
| AC-019 | Automated test | `TestHandoff_NoHintsIdenticalOutput` — compare prompt output with and without feature, assert identical |