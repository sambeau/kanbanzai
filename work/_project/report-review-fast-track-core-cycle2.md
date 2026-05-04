# Review: FEAT-01KQSP41PE6JP — Fast-Track Architecture Core (Cycle 2)

| Field        | Value                          |
|--------------|--------------------------------|
| Date         | 2026-05-04                     |
| Feature      | FEAT-01KQSP41PE6JP (B48-F1)    |
| Plan         | P43-fast-track-architecture    |
| Batch        | B48-fast-track-impl            |
| Review Cycle | 2                              |
| Reviewers    | reviewer-conformance, reviewer-quality, reviewer-testing, reviewer-security |
| Spec         | FEAT-01KQSP41PE6JP/spec-p43-spec-fast-track-architecture |

## Aggregate Verdict: **needs_remediation**

The feature has 5 blocking findings and 12 non-blocking findings. Cycle 1 fixes (F1–F4) are confirmed resolved. New issues found in MCP wiring, tier inference design, conditional gate logic, and test quality.

## Cycle 1 Fix Verification

| Finding | Status | Evidence |
|---------|--------|----------|
| F1: Validator dispatch stub | **FIXED** | `SpawnAgentDispatcher.Dispatch` generates handoff prompts with document + parent + rubric, no conversation log |
| F2: Bug.Tier missing | **FIXED** | Bug model has `Tier string` field; `CreateBug` calls `inferTier` |
| F3: Retro signal inference | **FIXED** | `inferTier` checks `"retro"` tag → `TierRetroFix` |
| F4: Conditional gate deferred | **FIXED** | `evaluateConditional` checks `FilesModified` against `isDocOnlyChange` prefix list |

## Per-Dimension Verdicts

| Dimension | Verdict | Reviewer |
|-----------|---------|----------|
| spec_conformance | fail | reviewer-conformance |
| implementation_quality | concern | reviewer-quality |
| test_adequacy | fail | reviewer-testing |
| security | pass_with_notes | reviewer-security |

## Review Unit Dispatch

| Unit | Files | Reviewers |
|------|-------|-----------|
| Full implementation | All 22 files | reviewer-conformance |
| Core Go code | config, validate, service, health, mcp | reviewer-quality |
| Test files | All *_test.go files + rubrics | reviewer-testing |
| Security surface | config, validate, service, health, mcp, entity_tool | reviewer-security |

## Blocking Findings

### B1: `inferTier` cannot distinguish bugs from features — REQ-INFER-002(b) unimplementable
- **Severity:** blocking
- **Spec anchors:** REQ-INFER-002(b), AC-INFER-001
- **Location:** `internal/service/entities.go#L1071-L1084`
- **Description:** The `inferTier` function signature is `func inferTier(explicitTier string, tags []string, cfg *config.Config) string`. It has no knowledge of whether it's being called for a bug or a feature. The spec requires: "(b) if the feature is a bug entity type → `bug_fix`." This rule cannot be implemented with the current function signature. `CreateBug` calls `inferTier(input.Tier, input.Tags, s.cfg)` — a bug with no tags and no explicit tier gets `DefaultTier` (feature), not `bug_fix`.
- **Remediation:** Either add an entity-type parameter to `inferTier`, or have `CreateBug` default to `bug_fix` before calling `inferTier` (e.g., `if input.Tier == "" && no matching tags { tier = "bug_fix" }`).

### B2: MCP entity tool does not pass `Tags` or `Tier` to `CreateBugInput` — inference path unreachable
- **Severity:** blocking
- **Spec anchors:** REQ-INFER-001, REQ-INFER-002, AC-INFER-001
- **Location:** `internal/mcp/entity_tool.go#L163-L169`
- **Description:** `entityCreateOne` for bugs constructs `CreateBugInput` without extracting `tags` or `tier` from MCP arguments. The `CreateFeatureInput` path does pass `Tags` but not `Tier`. Tag-based tier inference works for features but not for bugs through the MCP interface.
- **Remediation:** Extract `tags` and `tier` arguments in both the bug and feature creation cases and pass them to the respective Create*Input structs.

### B3: `evaluateConditional` produces false-positive blocking when `FilesModified` is empty
- **Severity:** blocking
- **Spec anchors:** REQ-TIER-004, AC-TIER-002
- **Location:** `internal/validate/transition_validator.go#L165-L175`
- **Description:** When `FilesModified` is empty (len == 0), `isDocOnly` starts as `false` and the function falls through to the "implementation change" path, returning a blocking `COND_IMPL_CHANGE` result. If the caller doesn't populate `FilesModified`, every retro_fix review gate will block.
- **Remediation:** Either make `FilesModified` required for conditional gates (fail early with a clear error), or provide a fallback mechanism (e.g., inspect the worktree diff when `FilesModified` is empty).

### B4: Integration tests are assertion-free logging stubs
- **Severity:** blocking
- **Spec anchors:** AC-TRANS-001, AC-TRANS-003, AC-RVW-002, AC-TRANS-004, AC-TRANS-005, AC-INFER-002
- **Location:** `internal/mcp/fast_track_integration_test.go#L155-L477`
- **Description:** Of 17 integration test functions, at least 6 are assertion-free: they set up rich fixtures and then only `t.Logf` the results without a single `t.Error`/`t.Fatalf` assertion. These tests will pass regardless of system behavior.
- **Remediation:** Replace `t.Logf` with `t.Errorf`/`t.Fatalf` assertions on the expected outcomes. Mark tests that can't assert yet as `t.Skip()`.

### B5: 17 acceptance criteria lack test coverage or have only assertion-free coverage
- **Severity:** blocking
- **Spec anchors:** AC-SPEC-001 through AC-SPEC-004, AC-PLAN-001 through AC-PLAN-004, AC-RVW-001, AC-RVW-003, AC-TIER-002 through AC-TIER-004, AC-INFER-001, AC-INFER-003, AC-PIPE-003, AC-SESS-001 through AC-SESS-003, AC-NF-001, AC-NF-002
- **Location:** Across all test files
- **Description:** The spec has 29 acceptance criteria. Only ~12 have tests with meaningful assertions. The entire spec-validator behavioral verification, plan-validator behavioral verification, and session management are unverified.
- **Remediation:** For each AC, write a test with assertions or file a task explicitly deferring the test with justification.

## Non-Blocking Findings

### N1: Prompt injection risk via document content in `buildPrompt`
- **Location:** `internal/validate/validator_dispatch.go#L154-L197`
- **Description:** Document content is interpolated directly into LLM prompts. A document containing `## Instructions` or `Ignore all previous instructions...` could inject conflicting directives. Wrap in `<document>` XML tags.

### N2: `PersistFeatureBlockedReason` error swallowed in cycle cap path
- **Location:** `internal/mcp/doc_tool.go#L396-L402`
- **Description:** Error logged but function continues, feature's `blocked_reason` may not persist.

### N3: Doc comment mismatch in `SpawnAgentDispatcher.Dispatch`
- **Location:** `internal/validate/validator_dispatch.go#L122-L128`

### N4: `inferTier` does not validate explicit tier value
- **Location:** `internal/service/entities.go#L1072-L1074`

### N5: Missing dedicated `quality_review_test.go`
- **Location:** `internal/health/quality_review.go`

### N6: Root Bug struct lacks `Tier` field (merge consistency)
- **Location:** `internal/model/entities.go` (main tree)

### N7: No documented evidence of rubric testing against 15–20 documents
- **Location:** `work/P43-fast-track-architecture/validator-rubrics/`

### N8: Document path traversal risk in `readDoc` (CWE-22)

### N9: Override records missing agent identity

### N10: Missing content integrity check before validator dispatch

### N11: Checkpoint deduplication could prevent spam on cycle cap

### N12: Performance tests use wall-clock timing instead of tool-call counting
- **Location:** `internal/mcp/fast_track_integration_test.go#L604-L665`

## Finding Summary

| Classification | Count |
|----------------|-------|
| Blocking | 5 |
| Non-blocking | 12 |
| **Total** | **17** |

## What Went Well

- All 4 Cycle 1 blocking findings confirmed resolved with correct implementations
- Architecture design solid: `ValidatorDispatcher` interface, `FastTrackConfig`, `TransitionValidatorDispatcher`
- Stage bindings integration clean with `transition_validator` hooks in YAML
- Fresh-session isolation correct: `buildPrompt` includes only document + parent + rubric
- `ValidatorDispatcher` interface properly abstracts dispatch mechanism (REQ-SESS-004)
- Tier inference for features (tag-based) works correctly for security/critical/retro signals
- Config schema comprehensive with strong validation and test coverage
- Rubric files thorough with real Kanbanzai document examples
- Override mechanism complete with recording
- Cycle tracking and escalation work correctly
- Security posture appropriate for a local-first tool — no critical findings

## Remediation Plan

1. **(B1)** Add entity-type awareness to `inferTier` or have `CreateBug` default to `bug_fix`
2. **(B2)** Pass `tags` and `tier` args through `entityCreateOne` to `CreateBugInput` and `CreateFeatureInput`
3. **(B3)** Fix `evaluateConditional` to handle empty `FilesModified` gracefully
4. **(B4)** Add assertions to the 6 logging-only integration tests
5. **(B5)** Write tests with assertions for the 17 uncovered acceptance criteria, or explicitly defer with justification
