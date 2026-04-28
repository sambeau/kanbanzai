# Review: FEAT-01KNA-11F5FAFW — Worktree Graph Context

> Feature: FEAT-01KNA-11F5FAFW (worktree-graph-context)
> Plan: P21-codebase-memory-integration
> Spec: work/spec/worktree-graph-context.md
> Review Cycle: 1
> Verdict: **approved_with_followups**

---

## Summary

Phase 2 of the codebase memory integration adds a `GraphProject` field to
`worktree.Record` and wires it through `handoff`, `next`, `status`, and
`cleanup` so that agents automatically receive the correct project name,
tool call examples, and re-indexing instructions.

All 18 acceptance criteria (AC-001 through AC-018) and all 3 non-functional
requirements (NFR-001 through NFR-003) are satisfied. Zero blocking findings.
Four non-blocking follow-up observations are noted below.

---

## Review Panel

| Reviewer | Dimension | Verdict |
|----------|-----------|---------|
| reviewer-conformance | spec_conformance | pass_with_notes |
| reviewer-quality | implementation_quality | pass |
| reviewer-testing | test_adequacy | pass_with_notes |

---

## Files Changed

### Production (9 files)

- `internal/worktree/worktree.go` — `GraphProject` field on `Record`, `Fields()`, `FieldOrder()`
- `internal/worktree/store.go` — `recordFromFields()` backward-compatible parsing
- `internal/mcp/worktree_tool.go` — create (graph_project param), new update action, remove cleanup note, `worktreeRecordToMap()`
- `internal/mcp/assembly.go` — `graphProject`, `worktreePath`, `hasWorktree` on `assembledContext`; worktree lookup in `assembleContext()`
- `internal/mcp/handoff_tool.go` — `## Code Graph` section rendering (3 states), `worktreeStore` param
- `internal/mcp/next_tool.go` — `graph_project` in structured output, `worktreeStore` param
- `internal/mcp/status_tool.go` — `missing_graph_index` info attention item in `generateFeatureAttention()`
- `internal/mcp/cleanup_tool.go` — graph project note in `cleanupExecuteAction()` with pre-capture pattern
- `internal/mcp/server.go` — wiring `worktreeStore` to `HandoffTools()` and `NextTools()`

### Tests (8 files, ~24 new test cases)

- `internal/worktree/worktree_test.go` — 4 new tests (Fields, FieldOrder, legacy deserialisation, round-trip)
- `internal/mcp/worktree_tool_test.go` — 8 new tests (create, update, remove with/without GraphProject)
- `internal/mcp/assembly_test.go` — 7 new tests (Code Graph section: project set, empty, no worktree, ordering, byte limit, next output)
- `internal/mcp/status_tool_test.go` — 6 new tests (unit + integration for missing_graph_index: present, suppressed by project, suppressed by no worktree)
- `internal/mcp/cleanup_tool_test.go` — 2 new tests (graph project note present, absent)
- `internal/mcp/handoff_tool_test.go` — signature fix (trailing nil)
- `internal/mcp/integration_test.go` — signature fixes for HandoffTools/NextTools
- `internal/mcp/next_tool_test.go` — signature fix (trailing nil)

---

## Acceptance Criteria Traceability

| AC | FR | Verdict | Evidence |
|----|-----|---------|----------|
| AC-001 | FR-001 | ✅ pass | `recordFromFields()` type assertion yields zero value for missing key. Test: `TestRecordFromFields_LegacyNoGraphProject` |
| AC-002 | FR-002 | ✅ pass | `worktreeCreateAction()` sets `GraphProject: req.GetString("graph_project", "")`. Test: `TestWorktreeCreate_GraphProject_SetInRecord` |
| AC-003 | FR-002 | ✅ pass | Same path; default empty string when param omitted. Test: `TestWorktreeCreate_GraphProject_DefaultEmpty` |
| AC-004 | FR-003 | ✅ pass | `worktreeUpdateAction()` checks raw args map presence. Test: `TestWorktreeUpdate_GraphProject_SetsValue` |
| AC-005 | FR-003 | ✅ pass | Omitted param preserves existing value. Test: `TestWorktreeUpdate_GraphProject_PreservedWhenOmitted` |
| AC-006 | FR-004 | ✅ pass | `renderHandoffPrompt()` emits `## Code Graph` with project name, 4 tool examples, preference instruction, re-index instruction. Test: `TestRenderHandoffPrompt_CodeGraphSection_ProjectSet` |
| AC-007 | FR-005 | ✅ pass | Empty GraphProject + worktree → `## Code Graph` with `index_repository` instruction. Test: `TestRenderHandoffPrompt_CodeGraphSection_ProjectEmpty` |
| AC-008 | FR-006 | ✅ pass | No worktree → no `## Code Graph` section. Test: `TestRenderHandoffPrompt_CodeGraphSection_NoWorktree` |
| AC-009 | FR-007 | ✅ pass | `## Code Graph` rendered after `## Available Tools` by code ordering. Test: `TestRenderHandoffPrompt_CodeGraphSection_AfterAvailableTools` |
| AC-010 | FR-008 | ✅ pass | `nextContextToMap()` unconditionally sets `out["graph_project"]`. Test: `TestNextContextToMap_GraphProject` |
| AC-011 | FR-008 | ✅ pass | Empty string when no worktree. Test: `TestNextContextToMap_GraphProjectEmpty` |
| AC-012 | FR-009 | ✅ pass | `generateFeatureAttention()` emits `missing_graph_index` info item. Tests: `TestFeatureAttention_MissingGraphIndex`, `TestSynthesiseFeature_MissingGraphIndex_Integration` |
| AC-013 | FR-010 | ✅ pass | Non-empty GraphProject → no item. Tests: `TestFeatureAttention_NoMissingGraphIndex_ProjectSet`, integration variant |
| AC-014 | FR-010 | ✅ pass | No worktree → no item. Tests: `TestFeatureAttention_NoMissingGraphIndex_NoWorktree`, integration variant |
| AC-015 | FR-011 | ✅ pass | `worktreeRemoveAction()` adds `graph_project_note`. Test: `TestWorktreeRemove_GraphProjectNote` |
| AC-016 | FR-011 | ✅ pass | Empty GraphProject → no note. Test: `TestWorktreeRemove_NoGraphProjectNote` |
| AC-017 | FR-012 | ✅ pass | `cleanupExecuteAction()` pre-captures GraphProject, adds note. Test: `TestCleanup_GraphProjectNote` |
| AC-018 | FR-013 | ✅ pass | No Go code calls `codebase_memory_mcp`. All graph output is gated on field values. Structural argument documented in test comments. |

### Non-Functional Requirements

| NFR | Verdict | Evidence |
|-----|---------|----------|
| NFR-001 | ✅ pass | No new I/O introduced. GraphProject read from already-loaded Record. |
| NFR-002 | ✅ pass | Go type assertion zero value handles missing YAML key. Test: `TestRecordFromFields_LegacyNoGraphProject` |
| NFR-003 | ✅ pass | Test: `TestRenderHandoffPrompt_CodeGraphSection_Under500Bytes` — section measures ~335 bytes with realistic inputs. |

---

## Findings

### Non-Blocking Findings

**NB-1: Missing dedicated test for AC-018** (conformance + testing)

The spec's verification plan names `TestWorkflow_NoGraphToolsAvailable` as the
verification method for AC-018. No such test exists. The implementation is
structurally sound — `codebase_memory_mcp` is never imported or called from Go
code, and all graph output is gated on field presence — but a lightweight
composite test would formally close the verification plan entry.

- Spec: AC-018
- Location: verification plan table
- Recommendation: Add a single test that exercises `renderHandoffPrompt`,
  `nextContextToMap`, `generateFeatureAttention`, and cleanup output with
  empty GraphProject, asserting no errors and identical non-graph behaviour.

**NB-2: assembleContext sets hasWorktree for any worktree status** (quality)

`assembleContext()` in `assembly.go:267-272` sets `hasWorktree = true` for
any worktree record regardless of status (active, merged, abandoned), while
`synthesiseFeature()` in `status_tool.go:789-795` correctly gates
`hasActiveWorktree` on `StatusActive` only. This means handoff's
`index_repository` fallback (FR-005) could fire for a feature with a
merged/abandoned worktree.

In practice this is harmless — handoff is only called for features with
active tasks, implying active development — and the spec does not qualify
FR-005/FR-006 by worktree status.

- Location: `assembly.go:267-272` vs `status_tool.go:789-795`
- Recommendation: Consider adding `wt.Status == worktree.StatusActive`
  check in assembleContext. Low priority.

**NB-3: Fields() serialization strategy inconsistency** (quality)

`Fields()` always serializes `graph_project` (even when empty string), while
`MergedAt` and `CleanupAfter` use conditional serialization (only when
non-nil). Both patterns produce correct round-trip behaviour.

- Location: `worktree.go:55-69`
- Recommendation: No change needed. Being explicit about `graph_project`
  avoids ambiguity between "field absent" and "field empty". The current
  approach is more robust for this field.

**NB-4: AC-002 tests persistence but not full MCP create handler** (testing)

`TestWorktreeCreate_GraphProject_SetInRecord` tests the store-level round-trip
but does not exercise the full `worktreeCreateAction` handler's extraction of
`graph_project` from request arguments. The parameter extraction is a single
`req.GetString("graph_project", "")` call — low risk.

- Location: `worktree_tool_test.go:99-126`
- Recommendation: The update action tests do exercise the full handler path
  and validate parameter extraction. Consider noting this cross-coverage in
  a test comment if no handler-level test is added.

---

## Review Unit Breakdown

| Unit | Files | Reviewers |
|------|-------|-----------|
| worktree-graph-context | All 17 changed files | reviewer-conformance, reviewer-quality, reviewer-testing |

Single review unit was appropriate given ≤17 files across two packages
(worktree + mcp) implementing a single cohesive feature.

---

## Verdict

**approved_with_followups**

All 18 acceptance criteria pass with traceable evidence. All 3 NFRs pass.
Zero blocking findings. Four non-blocking observations are recorded for
follow-up but do not block approval.

The implementation is clean, well-structured, and follows existing codebase
patterns. Notable quality highlights:
- The `worktreeUpdateAction` argument detection pattern correctly
  distinguishes "omitted" from "set to empty" for FR-003.
- The pre-capture pattern in `cleanupExecuteAction` prevents data loss
  when cleanup deletes records before graph project notes are emitted.
- The `## Code Graph` section stays well under the 500-byte NFR-003 limit.