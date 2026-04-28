# Review Report: Doc-Intel Register Workflow Friction (FEAT-01KPVDDYSEK8P)

**Feature:** FEAT-01KPVDDYSEK8P  
**Plan:** P28 — Doc-Intel Polish & Workflow Reliability  
**Reviewer:** orchestrator  
**Date:** 2026-04-23  
**Verdict:** pass

---

## Summary

This feature delivers two improvements to the classify-on-register workflow:

1. **Structured `classification_nudge`** (§5.4) — promotes the nudge from a plain instructional string to a JSON object containing `message`, `content_hash`, and `outline`. Callers can now proceed directly to `doc_intel classify` after `doc register` without a separate `doc_intel guide` call, reducing the workflow from 3 tool calls to 2.

2. **JSON struct tag audit** (§5.5) — audits MCP parameter structs decoded via `json.Unmarshal` and adds explicit `json:` tags where absent. A reflect-based regression test prevents future drift.

---

## Tasks Reviewed

| Task | Name | Status | Verdict |
|------|------|--------|---------|
| TASK-01KPVFXTHVYG3 | Promote `classification_nudge` to structured object | done | pass |
| TASK-01KPVG83MJ0H2 | Audit and fix json struct tags on MCP parameter structs | done | pass |
| TASK-01KPVG85SXVXZ | Unit tests for enhanced classification nudge | done | pass |
| TASK-01KPVG9AQNE3J | Add JSON struct tag regression test | done | pass |

---

## Findings

### Blocking

None.

### Non-Blocking

None.

---

## Test Evidence

```
go test ./internal/mcp/... -run "TestDocTool_ClassificationNudge|TestJSONStructTags" -v -count=1

--- PASS: TestJSONStructTags (0.00s)
--- PASS: TestDocTool_ClassificationNudge_MessageExactFormat (0.06s)
--- PASS: TestDocTool_ClassificationNudge_IsJSONObject (0.06s)
--- PASS: TestDocTool_ClassificationNudge_OutlineMatchesGuide (0.06s)
--- PASS: TestDocTool_ClassificationNudge_RegisterThenClassifyNoGuide (0.06s)
--- PASS: TestDocTool_ClassificationNudge_ContentHashPassesClassify (0.06s)
--- PASS: TestDocTool_ClassificationNudge_Batch3Docs (0.09s)
PASS
ok  github.com/sambeau/kanbanzai/internal/mcp  0.385s
```

Benchmark: `BenchmarkDocTool_Register_50Sections` — delta-p99-ms=0 over 100 invocations (REQ-NF-001: ≤ 50 ms). Pass.

---

## Spec Traceability

| Requirement | AC | Covered By | Result |
|-------------|-----|------------|--------|
| REQ-001 (structured nudge object) | AC-001 | TestDocTool_ClassificationNudge_IsJSONObject | ✅ |
| REQ-002 (message preserved) | AC-002 | TestDocTool_ClassificationNudge_MessageExactFormat | ✅ |
| REQ-003 (content_hash in nudge) | AC-003 | TestDocTool_ClassificationNudge_ContentHashPassesClassify | ✅ |
| REQ-004 (outline in nudge) | AC-004 | TestDocTool_ClassificationNudge_OutlineMatchesGuide | ✅ |
| REQ-005 (batch register nudge) | AC-005 | TestDocTool_ClassificationNudge_Batch3Docs | ✅ |
| REQ-006 (2-call workflow) | AC-006 | TestDocTool_ClassificationNudge_RegisterThenClassifyNoGuide | ✅ |
| REQ-007/008 (json tags on structs) | AC-007 | TASK-01KPVG83MJ0H2 audit + build pass | ✅ |
| REQ-009 (tag regression test) | AC-008 | TestJSONStructTags | ✅ |
| REQ-NF-001 (latency ≤ 50 ms p99) | AC-009 | BenchmarkDocTool_Register_50Sections | ✅ |
| REQ-NF-002 (test completes < 2 s) | AC-008 | TestJSONStructTags (0.32 s) | ✅ |

---

## Conclusion

All acceptance criteria satisfied. No blocking findings. Feature is ready to merge.