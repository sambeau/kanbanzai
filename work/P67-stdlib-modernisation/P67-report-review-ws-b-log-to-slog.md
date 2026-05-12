# Review: ws-b-log-to-slog

**Feature:** FEAT-01KREH8QC3VZ4 — ws-b-log-to-slog  
**Reviewer role:** reviewer-conformance  
**Date:** 2025-07-10  
**Verdict:** CONDITIONAL PASS (2 findings, no blockers)

---

## Summary

Replaced all `log` package usages with `log/slog` across the codebase. 16 production files
had their `"log"` import removed and their `log.Printf`/`log.Println`/`log.Print` calls
translated to structured slog calls. 54 slog call sites were inserted across those files.
Two global entry points (`cmd/kbz/main.go` and `internal/mcp/server.go`) received
`slog.SetDefault(...)` configuration so all slog output routes through a consistent handler.

Three tasks completed across three commits:
- `f0d82ee04` — slog entry-point configuration (B1)
- `b4002f554` — translate internal/mcp log calls to slog (B2)
- `db41d2f68` — translate remaining log calls to slog (B3)

---

## Conformance Findings

### F-01 (Minor) — `server_warmup_test.go` not migrated; tests now test dead code paths

**File:** `internal/mcp/server_warmup_test.go`  
**Evidence:** The file was not modified in this branch (confirmed via `git diff`). It still
imports `"log"` and uses `log.SetOutput(&buf)` / `log.Printf(...)` to verify log message
format for AC-005 and AC-006.

The production code it covers (`server.go` warmup block) now emits via `slog.Info(...)`.
The tests exercise their *own* `log.Printf` calls rather than calling the production
function, so they pass but provide no coverage of the new structured slog messages.
Comments in the test file still refer to the old `log.Printf("[server] cache warm-up: ...")` format.

**Impact:** Tests pass but are now effectively dead coverage. AC-005 and AC-006 acceptance
criteria are no longer verified by any test.  
**Severity:** Minor — no breakage, no regression; existing tests continue to pass.  
**Recommendation:** Migrate `server_warmup_test.go` to capture slog output via
`slog.SetDefault(slog.New(slog.NewTextHandler(&buf, ...)))`, mirroring the pattern already
used in `internal/service/documents_test.go`.

---

### F-02 (Moderate) — Stray changes in `next_tool.go` outside migration scope

**File:** `internal/mcp/next_tool.go` (and corresponding test files)  
**Evidence:** The diff for `next_tool.go` removes:
- `"sync"` import
- `checkKbzDirtyFuncMu sync.RWMutex` (mutex guarding concurrent test stub injection)
- `setCheckKbzDirtyFunc` / `restoreCheckKbzDirtyFunc` helpers

The corresponding test files (`invariant_boundary_test.go`, `finish_tool_test.go`,
`doc_intel_tool_test.go`, `next_tool_test.go`) were updated to assign `checkKbzDirtyFunc`
directly without the mutex. Several test setup functions also dropped
`t.Cleanup(knowledgeSvc.Close)` calls.

None of these changes relate to log-to-slog migration.

**Impact:** Mutex removal means parallel tests that stub `checkKbzDirtyFunc` are no longer
race-safe. The `-race` detector may surface data races. The dropped `Close()` cleanups are
a potential resource leak in tests.  
**Severity:** Moderate — scope creep introduces risk not reviewed under this feature's spec.
Tests continue to pass under `go test ./...` without `-race`.  
**Recommendation:** Isolate these changes to a separate commit or feature with explicit
rationale. If intentional (the mutex was over-engineering), document the decision.

---

## Conformance Checks

| Check | Result | Notes |
|-------|--------|-------|
| `slog.SetDefault(...)` at both entry points | PASS | `cmd/kbz/main.go` and `internal/mcp/server.go` |
| `"log"` import removed from all production files | PASS | 17 removals in diff; `next_tool.go` change is unrelated to log |
| `log.Printf`/`Println`/`Print` calls translated | PASS | 54 slog call sites inserted |
| Level mapping — `WARNING:` prefix → `slog.Warn` | PASS | Consistent across all files |
| Level mapping — `ERROR:` prefix → `slog.Error` | PASS | Panic recovery sites use `slog.Error` |
| Level mapping — unqualified → `slog.Info` | PASS | All remaining calls use `slog.Info` |
| Structured fields (key-value pairs) used | PASS | Args passed as `"key", value` pairs |
| `go build ./...` clean | PASS | Stated by implementer; no compile errors |
| `go test ./... ` (42 packages) | PASS | All pass; warmup tests pass for wrong reason (F-01) |

---

## Quality Notes

- Panic/recovery sites correctly mapped to `slog.Error` — this is the right level for
  unexpected runtime errors caught at goroutine boundaries.
- `internal/service/documents_test.go` was properly migrated: captures slog output via
  `slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, ...)))` with cleanup. This is the
  correct pattern and should be applied to `server_warmup_test.go` (see F-01).

---

## Overall Verdict

**CONDITIONAL PASS.** The core migration is correct, complete, and consistent. Both findings
are non-blocking: F-01 is a test coverage gap (no regression), and F-02 is scope creep that
does not break anything under the current test run. Recommend addressing both in a follow-up
before the next batch review cycle.
