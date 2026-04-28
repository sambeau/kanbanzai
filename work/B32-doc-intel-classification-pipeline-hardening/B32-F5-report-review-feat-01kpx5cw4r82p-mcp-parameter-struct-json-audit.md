# Review Report: MCP Parameter Struct JSON Tag Audit (FEAT-01KPX5CW4R82P)

**Feature:** FEAT-01KPX5CW4R82P  
**Plan:** P32 — Doc-Intel Classification Pipeline Hardening  
**Reviewer:** orchestrator  
**Date:** 2026-04-24  
**Verdict:** pass

---

## Summary

This feature audits all MCP parameter structs in `internal/mcp/` that are decoded from JSON tool parameters via `json.Unmarshal` and adds explicit `json:` tags to any exported fields that were missing them. It also adds a reflect-based round-trip regression test to prevent future drift.

The scope clarification — that `Classification` and `ConceptIntroEntry` were already fixed in a prior cycle and only the remaining structs required audit — was confirmed correct. No `yaml:` struct tags were found in `internal/mcp/`; all tag additions are `json:`-only.

---

## Review History

| Round | Findings | Outcome |
|-------|----------|---------|
| 1 | 2 majors: (1) audit scope not documented — unclear which structs were in scope vs already fixed; (2) regression test only covered happy path, failure path missing | needs-rework |
| 2 | Both addressed: scope documented in test file comment, failure-path self-test added | pass |
| 3 | No findings | approved |

---

## Tasks Reviewed

| Task | Name | Status | Verdict |
|------|------|--------|---------|
| TASK-01KPXE335QXD7 | Audit existing structs for missing json tags | done | pass |
| TASK-01KPXE3361ANK | Add json tags to structs missing them | done | pass |
| TASK-01KPXE3VRGBQF | Confirm no yaml: tags in internal/mcp/ | done | pass |
| TASK-01KPXE45XV5A5 | Add round-trip regression test (happy path) | done | pass |
| TASK-01KPXE4CR77A7 | Add regression test failure path + document scope | done | pass |

---

## Findings

### Round 1 — Blocking

**F-001 (major):** Audit scope was undocumented. It was not clear which structs had been audited and found clean vs which required fixes, making the regression test impossible to verify without re-reading the full diff.

**F-002 (major):** `TestJSONTagRoundTrip` only tested the happy path (all tags present). No self-test verified that the checker would actually catch a missing tag.

### Round 2 — Residual

None. Both F-001 and F-002 were resolved:
- Scope documented in test file header comment listing audited structs and prior-fixed structs.
- Failure-path self-test added: a synthetic struct missing a `json:` tag is injected and the checker is asserted to detect it.

### Round 3

No findings. Feature approved.

---

## Test Evidence

```
go test ./internal/mcp/... -run TestJSONTagRoundTrip -v -count=1

--- PASS: TestJSONTagRoundTrip (0.00s)
    --- PASS: TestJSONTagRoundTrip/happy_path (0.00s)
    --- PASS: TestJSONTagRoundTrip/failure_path_self_test (0.00s)
PASS
ok  github.com/sambeau/kanbanzai/internal/mcp  0.53s
```

Full suite (pre-existing flaky test excluded):
```
go test ./... — all packages pass except pre-existing SQLite WAL TempDir cleanup race on macOS (intermittent, present on main, not introduced by this feature)
```

---

## Spec Traceability

| Requirement | AC | Covered By | Result |
|-------------|-----|------------|--------|
| REQ-001 (audit all structs with yaml: tags decoded via json.Unmarshal) | AC-001 | TASK-01KPXE335QXD7 audit + grep confirmation | ✅ |
| REQ-002 (add json: tags to fields missing them) | AC-002 | TASK-01KPXE3361ANK + build pass | ✅ |
| REQ-003 (no yaml: tags in scope) | AC-003 | TASK-01KPXE3VRGBQF grep | ✅ |
| REQ-004 (regression test — happy path) | AC-004 | TestJSONTagRoundTrip/happy_path | ✅ |
| REQ-005 (regression test — failure path self-test) | AC-005 | TestJSONTagRoundTrip/failure_path_self_test | ✅ |

---

## Conclusion

All acceptance criteria satisfied. The audit is complete and scope is documented. The regression guard covers both happy-path and failure-path. No blocking findings remain. Feature is ready to merge.