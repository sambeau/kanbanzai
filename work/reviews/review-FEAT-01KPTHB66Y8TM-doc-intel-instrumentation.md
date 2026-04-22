# Review: Doc Intel Access Instrumentation
**Feature:** FEAT-01KPTHB66Y8TM
**Branch:** feature/FEAT-01KPTHB66Y8TM-doc-intel-instrumentation
**Spec:** work/spec/doc-intel-access-instrumentation.md
**Date:** 2026-04-22

---

## Review Unit

| Field | Value |
|-------|-------|
| Feature | FEAT-01KPTHB66Y8TM — doc-intel-instrumentation |
| Overall verdict | **approved_with_followups** |
| Blocking findings | 0 |
| Non-blocking findings | 6 (0 concerns, 6 notes) |
| Remediation | CONCERN-TA-01 resolved: FR-016 Search counter test added (commit 648e939) |

### Files reviewed

- `internal/docint/types.go`
- `internal/mcp/doc_tool.go`
- `internal/mcp/doc_tool_test.go`
- `internal/mcp/knowledge_tool.go`
- `internal/service/doc_audit.go`
- `internal/service/doc_audit_access_test.go`
- `internal/service/intelligence.go`
- `internal/service/intelligence_access_test.go`
- `internal/service/intelligence_test.go`
- `internal/service/knowledge.go`
- `internal/service/knowledge_access_test.go`
- `internal/service/knowledge_test.go`
- `internal/service/retro_synthesis_test.go`
- `internal/context/assemble_test.go`

---

## Dimension: spec_conformance

**Outcome:** pass_with_notes

### Evidence

All 21 functional requirements mapped to implementation and confirmed present. Every new
struct field carries `yaml:",omitempty"` for backward compatibility (FR-018, NFR-003). All
counter call sites verified — `Get`, `List`, `GetOutline`, `GetDocumentIndex`, `GetSection`,
`FindByEntity`, `FindByConcept`, `FindByRole`, `Search`. The `sort:"recent"` path and audit
table are implemented per spec. Counter writes are lazy (background goroutines via
`sync.WaitGroup`) and errors are silently absorbed (NFR-001, NFR-002).

### FR-by-FR mapping

| FR | File | Status |
|----|------|--------|
| FR-001 `recent_use_count` default 0 | `knowledge.go`, `knowledge_tool.go` | ✅ |
| FR-002 `last_accessed_at` absent on new entries | `knowledge.go` | ✅ |
| FR-003 `Get` increments | `knowledge.go:Get` → background `touchKnowledgeEntry` | ✅ |
| FR-004 `List` increments all returned entries | `knowledge.go:List` → background goroutine | ✅ |
| FR-005 30-day rolling window (lazy decay) | `knowledge.go:touchKnowledgeEntry` | ✅ |
| FR-006 `sort:"recent"` on `KnowledgeFilters` | `knowledge.go:List` `sort.Slice` | ✅ |
| FR-007 MCP `knowledge list` includes `recent_use_count` | `knowledge_tool.go:knowledgeListAction` | ✅ |
| FR-008 `sort:"recent"` via MCP parameter | `knowledge_tool.go:knowledgeListAction` | ✅ (code only) |
| FR-009 `DocumentIndex.AccessCount int` | `types.go` | ✅ |
| FR-010 `DocumentIndex.LastAccessedAt *time.Time` | `types.go` | ✅ |
| FR-011 `SectionAccessInfo` + `SectionAccess` map | `types.go` | ✅ |
| FR-012 `GetOutline` increments document counter | `intelligence.go:GetOutline` | ✅ |
| FR-013 guide action increments document counter | `intelligence.go:GetDocumentIndex` | ✅ |
| FR-014 `GetSection` increments doc + section | `intelligence.go:GetSection` → `touchDocumentAccess(docID, sectionPath)` | ✅ |
| FR-015 find actions increment distinct documents | `intelligence.go:FindByEntity/FindByConcept/FindByRole` | ✅ (see NOTE-SC-02) |
| FR-016 `Search` increments document counters | `intelligence.go:Search` + `intelligence_access_test.go` | ✅ |
| FR-017 counter errors do not propagate | `touchDocumentAccess`, `touchKnowledgeEntry` | ✅ |
| FR-018 backward-compatible YAML fields | all new fields tagged `yaml:",omitempty"` | ✅ |
| FR-019 `AuditResult.MostAccessed` field | `doc_audit.go:AuditResult` | ✅ |
| FR-020 top-10 ordered descending, zero excluded | `doc_audit.go:collectMostAccessed` | ✅ |
| FR-021 "Most Accessed Documents" table in audit | `doc_tool.go:docAuditAction`, `doc_audit.go:RenderMostAccessedTable` | ✅ |

### Findings

~~**NOTE-SC-01** — FR-016 has no acceptance test~~ **RESOLVED**  
Two tests added in commit 648e939: `TestIntelligenceService_Search_IncrementsAccessCount`
and `TestIntelligenceService_Search_NoIncrementWhenNoResults` in `intelligence_access_test.go`.
Both pass. AC-16 is now fully verified.

**NOTE-SC-02** — `FindByEntity` spawns goroutine unconditionally when results are empty  
Both the SQLite path and YAML fallback path in `FindByEntity` call `wg.Add(1)` and spawn a
goroutine before checking whether `matches` is non-empty. `touchDistinctDocuments([])` is a
no-op, so no spurious increment occurs, but the behaviour is inconsistent with FR-015's
"after returning a non-empty result set" qualifier and with `FindByConcept`/`FindByRole`,
which correctly guard with `if len(matches) > 0` before goroutine creation.

---

## Dimension: implementation_quality

**Outcome:** pass_with_notes

### Evidence

The background-goroutine pattern with `sync.WaitGroup` is correctly and consistently applied
in both `KnowledgeService` and `IntelligenceService`. Error absorption is uniform
(`nolint:errcheck` on all best-effort writes, silent returns on load failures). The
injectable clock (`s.now func() time.Time`) in `KnowledgeService` makes the 30-day decay
logic fully deterministic in tests. All new YAML fields carry `omitempty`. No new module
dependencies introduced (NFR-005). The dual `most_accessed` (structured) +
`most_accessed_table` (markdown) in the audit response is beyond spec but improves API
ergonomics without harm.

### Findings

**NOTE-IQ-01** — Unnecessary goroutine spawn in `FindByEntity` on empty results  
`intelligence.go:FindByEntity` calls `wg.Add(1)` and spawns a goroutine regardless of
whether the result slice is empty. The goroutine is a no-op when empty, but it incurs
goroutine allocation and WaitGroup overhead on every call with no matches. The fix is to
wrap both `wg.Add(1)` / goroutine-spawn sites in `FindByEntity` with `if len(matches) > 0`
guards, mirroring `FindByConcept` and `FindByRole`.

```kanbanzai/.worktrees/FEAT-01KPTHB66Y8TM-doc-intel-instrumentation/internal/service/intelligence.go#L241-244
// SQLite path (current — spawns goroutine unconditionally):
s.wg.Add(1)
go func(m []EntityDocMatch) { defer s.wg.Done(); s.touchDistinctDocuments(m) }(matches)
return matches, nil
```

Suggested fix:
```/dev/null/suggestion.go#L1-5
if len(matches) > 0 {
    s.wg.Add(1)
    go func(m []EntityDocMatch) { defer s.wg.Done(); s.touchDistinctDocuments(m) }(matches)
}
return matches, nil
```

Apply the same guard to the YAML fallback path at the end of `FindByEntity`.

**NOTE-IQ-02** — Misleading comment in `knowledge_access_test.go`  
`TestKnowledgeService_Get_IncrementsMultipleTimes` contains the comment: *"Re-create service
to reset the wait group state (Close drains the WaitGroup but doesn't destroy it)"* — but no
re-creation occurs; the same `svc` is used throughout. The code is correct (`sync.WaitGroup`
is safe to reuse after `Wait()` returns with counter at 0), but the comment incorrectly
implies a re-creation step that doesn't happen. The comment should be removed or corrected to
avoid confusing future maintainers.

---

## Dimension: test_adequacy

**Outcome:** pass_with_notes

### Evidence

`knowledge_access_test.go` provides comprehensive coverage of FR-001 through FR-006 including
the 30-day decay edge cases. `doc_audit_access_test.go` covers FR-019 and FR-020 with 10
dedicated tests (population, zero-exclusion, nil-exclusion, cap-at-10, empty cases, field
correctness, descending order) plus three `RenderMostAccessedTable` tests. The intelligence
access tests cover FR-009 through FR-015 and FR-017 with well-isolated helpers and a
`loadIndexDirect` pattern that correctly bypasses the service to avoid counter contamination.

CONCERN-TA-01 (missing FR-016 test) was resolved by adding two tests in commit 648e939.
All named acceptance criteria now have test coverage. Remaining gaps are MCP-layer only,
with full service-layer coverage in place.

### Findings

~~**CONCERN-TA-01** — No test for FR-016: `Search` counter increment~~ **RESOLVED**  
`TestIntelligenceService_Search_IncrementsAccessCount` and
`TestIntelligenceService_Search_NoIncrementWhenNoResults` added to
`intelligence_access_test.go` in commit 648e939. Both pass. AC-16 verified.

**NOTE-TA-02** — No MCP-level test for FR-008 (`sort:"recent"` parameter passthrough)  
`knowledge_tool_test.go` is not in the changed files. The `sort` parameter passthrough in
`knowledgeListAction` is a one-liner (`Sort: req.GetString("sort", "")`) and the service-layer
sort is fully tested in `knowledge_access_test.go`, so the gap is low-risk. A follow-up test
exercising the MCP handler with `sort:"recent"` would complete the coverage chain.

**NOTE-TA-03** — No MCP-level test for FR-021 (audit `most_accessed_table` in response)  
`doc_tool_test.go` audit tests (`TestDocAudit_*`) do not pass an `IntelligenceService` to the
audit action, so the `most_accessed_table` key in the MCP response is never asserted at the
tool layer. The service-level rendering logic is comprehensively tested in
`doc_audit_access_test.go`. A follow-up test exercising `docAuditAction` end-to-end with an
`IntelligenceService` containing seeded indexes would close this gap.

---

## Finding Summary

| ID | Dimension | Severity | Description |
|----|-----------|----------|-------------|
| ~~NOTE-SC-01~~ | spec_conformance | ✅ resolved | FR-016 Search test added (commit 648e939) |
| NOTE-SC-02 | spec_conformance | note | FindByEntity spawns goroutine unconditionally on empty results |
| NOTE-IQ-01 | implementation_quality | note | FindByEntity unconditional goroutine spawn (NFR overhead) |
| NOTE-IQ-02 | implementation_quality | note | Misleading comment in Get_IncrementsMultipleTimes test |
| ~~CONCERN-TA-01~~ | test_adequacy | ✅ resolved | FR-016 Search counter test added (commit 648e939) |
| NOTE-TA-02 | test_adequacy | note | No MCP-level test for sort:"recent" parameter (FR-008) |
| NOTE-TA-03 | test_adequacy | note | No MCP-level test for most_accessed_table in audit response (FR-021) |

**Blocking: 0 · Non-blocking: 6 (0 concerns, 6 notes)**

---

## Verdict

**approved_with_followups**

The implementation is complete, correct, and spec-conformant across all 21 functional
requirements. All counter call sites are wired, error isolation is uniform, the 30-day
decay and `sort:"recent"` paths are well-tested, and all named acceptance criteria now
have service-layer test coverage (CONCERN-TA-01 resolved in commit 648e939).
The feature is safe to merge.

The remaining notes are cosmetic or low-risk and may be addressed opportunistically.

### Recommended follow-up tasks

1. Fix `FindByEntity` goroutine guard inconsistency to match `FindByConcept`/`FindByRole`
   (addresses NOTE-SC-02 and NOTE-IQ-01).
2. Correct misleading comment in `TestKnowledgeService_Get_IncrementsMultipleTimes`
   (addresses NOTE-IQ-02).
3. Add MCP-level test for `sort:"recent"` in `knowledge_tool_test.go` (addresses NOTE-TA-02).
4. Add MCP-level audit test asserting `most_accessed_table` in `doc_tool_test.go`
   (addresses NOTE-TA-03).

---

## Re-review note

CONCERN-TA-01 resolved by commit 648e939 on 2026-04-22. No other findings changed status.
Verdict remains `approved_with_followups` (0 blocking, 6 non-blocking notes).