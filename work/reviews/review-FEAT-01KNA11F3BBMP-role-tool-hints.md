# Review: Role-Scoped Tool Hints

> Feature: FEAT-01KNA-11F3BBMP — role-tool-hints
> Review cycle: 1
> Reviewers dispatched: reviewer-conformance, reviewer-quality, reviewer-testing
> Review units: 1 (tool-hints-all)
> Branch: feature/FEAT-01KNA11F3BBMP-role-tool-hints (3 commits ahead of main)

---

## Per-Reviewer Summary

### Reviewer: reviewer-conformance

- **Review unit:** tool-hints-all
- **Verdict:** approved_with_followups
- **Dimensions:**
  - spec_conformance: pass_with_notes
- **Findings:** 0 blocking, 3 non-blocking

All 19 acceptance criteria are satisfied at the production code level. Every
functional requirement has correct, working implementation. 12 of 19 ACs have
direct test coverage; the remaining 7 ACs are structurally guaranteed by guard
clauses and omitempty tags but lack the dedicated tests named in the
verification plan.

### Reviewer: reviewer-quality

- **Review unit:** tool-hints-all
- **Verdict:** approved
- **Dimensions:**
  - implementation_quality: pass_with_notes
- **Findings:** 0 blocking, 3 non-blocking

Clean, well-scoped feature. ~100 lines of new production code with correct
nil/empty handling, no race conditions, no dead code, no unreachable paths.
Naming is consistent with the codebase. Parameter threading is mechanical but
correct. The localConfig load was moved earlier in server.go to serve the hints
merge — a clean refactor that removes a duplicate load.

### Reviewer: reviewer-testing

- **Review unit:** tool-hints-all
- **Verdict:** needs_remediation
- **Dimensions:**
  - test_adequacy: concern
- **Findings:** 2 blocking, 4 non-blocking

Tests that exist are well-crafted: table-driven, properly isolated with
`t.TempDir()`, meaningful assertions, and exceed spec requirements in several
areas (7 merge cases vs 3 required, 7 resolve cases vs 4 required, truncation
edge case). Two code paths have zero test coverage and are classified as
blocking: the legacy 2.0 handoff injection and the next context map injection.

---

## Collated Findings (deduplicated)

### Blocking

**[B-1] Legacy 2.0 handoff path untested (AC-012, AC-013)**

- **Dimension:** test_adequacy
- **Location:** internal/mcp/handoff_tool.go:~L480–486, internal/mcp/handoff_tool_test.go
- **Spec ref:** AC-012 (FR-013), AC-013 (FR-014)
- **Description:** `renderHandoffPrompt` conditionally injects `## Available Tools`
  when `actx.toolHint` is non-empty (before "Additional Instructions"). No test
  exercises this branch in either direction. The 3.0 pipeline path is covered
  (AC-010/011) but the legacy path is entirely untested. The spec verification
  plan requires `TestHandoffLegacy_HintInjected` and
  `TestHandoffLegacy_NoHintNoSection` — neither exists.
- **Reported by:** reviewer-conformance (as non-blocking), reviewer-testing (as blocking)
- **Remediation:** Add two tests following the existing
  `TestRenderHandoffPrompt_WithExperiments` pattern in assembly_test.go.

**[B-2] Next context map tool_hint field untested (AC-014, AC-015)**

- **Dimension:** test_adequacy
- **Location:** internal/mcp/next_tool.go:~L412–415, internal/mcp/next_tool_test.go
- **Spec ref:** AC-014 (FR-015), AC-015 (FR-016)
- **Description:** `nextContextToMap` conditionally adds `"tool_hint"` to the
  output map when the resolved hint is non-empty. No test verifies this key is
  present or absent. The spec verification plan requires
  `TestNextContext_HintIncluded` and `TestNextContext_NoHintOmitted` — neither
  exists. Existing `TestNextContextToMap_With/WithoutExperiments` tests
  demonstrate the exact pattern needed.
- **Reported by:** reviewer-conformance (as non-blocking), reviewer-testing (as blocking)
- **Remediation:** Add two tests following the existing experiments test pattern.

### Non-Blocking

**[NB-1] Config YAML parsing round-trip not tested (AC-001, AC-002)**

- **Dimension:** test_adequacy
- **Location:** internal/config/tool_hints_test.go
- **Spec ref:** AC-001 (FR-001, FR-002), AC-002 (FR-001, FR-003)
- **Description:** The verification plan requires `TestConfigParse_NoToolHints`
  and `TestConfigParse_WithToolHints` to verify ToolHints deserializes correctly
  from YAML. No test writes a YAML blob with/without `tool_hints` and asserts
  the parsed struct. Risk is low — the field is a simple `map[string]string`
  with standard yaml struct tags — but the spec explicitly names these tests.
- **Reported by:** reviewer-testing
- **Recommendation:** Add two subtests to an existing config parse test or a new
  `TestConfig_ToolHintsParsing` function.

**[NB-2] Backward-compatibility regression test missing (AC-019)**

- **Dimension:** test_adequacy
- **Location:** internal/mcp/handoff_tool_test.go
- **Spec ref:** AC-019 (FR-020, FR-021)
- **Description:** No test compares pre-feature and post-feature output when no
  hints are configured to verify byte-identical prompts. The structural guarantee
  (omitempty + nil merge + guard clauses) is strong, but the spec verification
  plan names `TestHandoff_NoHintsIdenticalOutput` and it does not exist.
- **Reported by:** reviewer-conformance, reviewer-testing
- **Recommendation:** Add a test that captures `renderHandoffPrompt` output with
  a zero-value `assembledContext` and compares to a known baseline.

**[NB-3] Asymmetric nil-RoleStore fallback between pipeline and legacy paths**

- **Dimension:** implementation_quality
- **Location:** internal/context/pipeline.go:~L648–662, internal/mcp/assembly.go:~L219–221
- **Spec ref:** N/A (quality observation)
- **Description:** Pipeline's `stepResolveToolHint` falls back to exact-match
  when `ToolHintRoleStore` is nil. The assembly path silently skips resolution
  entirely when `roleStore` is nil. Both receive the same store from server.go
  so this divergence is harmless in practice, but the inconsistency could
  surprise future maintainers.
- **Reported by:** reviewer-quality
- **Recommendation:** Consider extracting the "exact-match fallback" into
  `ResolveToolHint` itself (treat nil store as "no inheritance available").

**[NB-4] Configured hints reported as warnings in health output**

- **Dimension:** implementation_quality
- **Location:** internal/mcp/health_tool.go:~L259–267
- **Spec ref:** N/A (quality observation)
- **Description:** Each configured role→hint pair is emitted as a
  `ValidationWarning`, inflating `WarningCount` in health output. This is
  informational, not a condition needing attention. The codebase only offers
  Errors and Warnings channels (no Info tier), so this is an infrastructure
  constraint rather than a defect.
- **Reported by:** reviewer-quality
- **Recommendation:** If an Info severity is added later, move these to it.

**[NB-5] Missing blank line between stepResolveToolHint and stepTokenBudget**

- **Dimension:** implementation_quality
- **Location:** internal/context/pipeline.go:~L664–665
- **Spec ref:** N/A (style observation)
- **Description:** Style-only. The spacing convention between step methods in
  this file uses a blank line separator; the new `stepResolveToolHint` is
  missing one before `stepTokenBudget`.
- **Reported by:** reviewer-quality
- **Recommendation:** Add a blank line to match the file's convention.

**[NB-6] AC-016 test is const-level, not behavioral**

- **Dimension:** test_adequacy
- **Location:** internal/context/pipeline_tool_hints_test.go
- **Spec ref:** AC-016 (FR-017)
- **Description:** `TestStepAssembleSections_ToolHintsBeforeProcedure` asserts
  `PositionAvailableTools < PositionProcedure` at the constant level. This
  validates the ordering invariant but does not verify rendered output ordering.
  The Phase 2 Code Graph section does not yet exist so a full behavioral test
  is not possible, but the constant-level test is fragile if the rendering
  strategy changes.
- **Reported by:** reviewer-testing
- **Recommendation:** Extend `TestStepAssembleSections_AvailableTools` to also
  verify relative ordering in the assembled Sections slice.

**[NB-7] Cycle protection in ResolveToolHint not exercised by tests**

- **Dimension:** test_adequacy
- **Location:** internal/context/tool_hints.go:~L17, internal/context/tool_hints_test.go
- **Spec ref:** N/A (boundary coverage observation)
- **Description:** `ResolveToolHint` has a `visited` map for cycle protection
  in the inheritance walk, but no test creates a cyclic `inherits` chain to
  exercise this code path.
- **Reported by:** reviewer-testing
- **Recommendation:** Add a test with circular inheritance
  (e.g., A inherits B, B inherits A) to verify the cycle breaker.

---

## Aggregate Verdict: approved_with_followups

### Rationale

- **spec_conformance: pass_with_notes** — All 19 acceptance criteria are
  satisfied at the implementation level. Production code is correct.
- **implementation_quality: pass_with_notes** — Clean, minimal, well-integrated.
  Three minor observations, all non-blocking.
- **test_adequacy: concern** — 12 of 19 ACs have direct test coverage. Two
  blocking findings identify untested conditional branches in the legacy handoff
  and next context map paths. These are simple guard-clause injections (4–6
  lines each) backed by thoroughly tested underlying functions, and the primary
  3.0 pipeline path IS tested — but the branches themselves have zero coverage.

### Blocking Finding Assessment

The 2 blocking findings (B-1 and B-2) are test gaps, not implementation
defects. The production code for both paths is verified correct by the
conformance reviewer. The underlying `ResolveToolHint` and `MergeToolHints`
functions are thoroughly tested. The missing tests are ~10-line additions
following established patterns already in the codebase
(`TestRenderHandoffPrompt_WithExperiments`, `TestNextContextToMap_WithExperiments`).

**Recommendation:** Add the 4 missing tests (2 for B-1, 2 for B-2) before
merging. This is a small remediation (~40 lines of test code) that can be done
in a single pass. The 5 non-blocking test gaps (NB-1, NB-2, NB-6, NB-7, and
the config parsing tests) can be follow-up items.

---

## Review Unit Breakdown

### Unit: tool-hints-all

**Scope:** All files changed on the feature branch across 3 packages.

**Production files (10):**
- `internal/config/config.go` — ToolHints field on Config struct
- `internal/config/tool_hints.go` — MergeToolHints (17 lines)
- `internal/config/user.go` — ToolHints field on LocalConfig struct
- `internal/context/pipeline.go` — Position constants, stepResolveToolHint, section assembly
- `internal/context/tool_hints.go` — ResolveToolHint (33 lines)
- `internal/mcp/assembly.go` — assembledContext.toolHint, asmInput fields, resolution call
- `internal/mcp/handoff_tool.go` — Parameter threading, renderHandoffPrompt injection
- `internal/mcp/health_tool.go` — ToolHintsHealthChecker
- `internal/mcp/next_tool.go` — Parameter threading, nextContextToMap injection
- `internal/mcp/server.go` — Startup wiring (merge at init, pass to tools)

**Test files (8):**
- `internal/config/tool_hints_test.go` — 7 subtests (new)
- `internal/context/tool_hints_test.go` — 7 subtests (new)
- `internal/context/pipeline_tool_hints_test.go` — 6 tests (new)
- `internal/mcp/health_tool_hints_test.go` — 3 tests (new)
- `internal/mcp/handoff_tool_test.go` — Signature updates only
- `internal/mcp/next_tool_test.go` — Signature updates only
- `internal/mcp/integration_test.go` — Signature updates only
- `internal/mcp/status_tool_test.go` — Signature updates only

**Spec sections reviewed:** All (FR-001 through FR-021, AC-001 through AC-019)

**Dispatched reviewers:**
- reviewer-conformance → spec_conformance
- reviewer-quality → implementation_quality
- reviewer-testing → test_adequacy