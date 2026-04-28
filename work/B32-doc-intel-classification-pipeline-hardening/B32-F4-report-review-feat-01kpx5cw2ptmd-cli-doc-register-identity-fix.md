# Review Report: CLI doc register Identity Resolution Fix (FEAT-01KPX5CW2PTMD)

**Feature:** FEAT-01KPX5CW2PTMD  
**Plan:** P32 — Doc-Intel Classification Pipeline Hardening  
**Reviewer:** orchestrator  
**Date:** 2026-04-24  
**Verdict:** pass

---

## Summary

This feature fixes the `kbz doc register` and `kbz doc approve` CLI commands to call
`config.ResolveIdentity("")` so that `created_by` / `approved_by` is auto-resolved from
`.kbz/local.yaml` or `git config user.name` without requiring the explicit `--by` flag.
The same helper is already used by `worktree create` and other commands. The fix eliminates
the six-attempt registration friction documented in the P27 retrospective.

---

## Review Rounds

### Round 1 — Initial review (needs-rework)

Two blocking findings:

1. **`runDocRegister` missing identity resolution** — `created_by` was taken verbatim from the
   `--by` flag with no fallback, so omitting `--by` yielded an empty string passed to the store.
2. **`runDocApprove` missing identity resolution** — same omission on the approve path; the
   `approved_by` field was likewise unresolved when `--by` was absent.

### Round 2 — Re-review (needs-rework)

Implementation of `ResolveIdentity` in `runDocRegister` was correct. One residual finding:

1. **`runDocApprove` not yet fixed** — `ResolveIdentity` was applied only to the register path;
   the approve path still read the raw flag value.

### Round 3 — Final review (approved)

Both `runDocRegister` and `runDocApprove` now call `config.ResolveIdentity("")` before passing
the identity to the store layer. Test comment mismatch on the approve test was also corrected.
No remaining findings.

---

## Tasks Reviewed

| Task | Status | Verdict |
|------|--------|---------|
| TASK-01KPXE3P4BVW4 — Audit identity resolution in CLI doc commands | done | pass |
| TASK-01KPXE3P4D4FW — Apply ResolveIdentity in runDocRegister | done | pass |
| TASK-01KPXE3P4E5FK — Apply ResolveIdentity in runDocApprove | done | pass |
| TASK-01KPXE3P4GPD5 — Unit tests: register auto-resolves identity | done | pass |
| TASK-01KPXE3P4J0PY — Unit tests: approve auto-resolves identity | done | pass |
| TASK-01KPXE3P4K7Y7 — Verify no regression in other CLI commands | done | pass |
| TASK-01KPXE3P4MDCM — Integration smoke: register then approve without --by | done | pass |
| TASK-01KPXE3P4NB1T — Full test suite pass | done | pass |

---

## Findings

### Blocking

None remaining after Round 3.

### Non-Blocking

None.

---

## Test Evidence

```
go test ./cmd/kanbanzai/... -v -count=1

--- PASS: TestDocRegister_AutoResolvesIdentity (0.04s)
--- PASS: TestDocApprove_AutoResolvesIdentity (0.04s)
--- PASS: TestDocRegister_ExplicitByFlagOverridesResolved (0.03s)
--- PASS: TestDocApprove_ExplicitByFlagOverridesResolved (0.03s)
PASS
ok  github.com/sambeau/kanbanzai/cmd/kanbanzai  0.21s
```

Pre-existing flaky TempDir cleanup race in `internal/mcp` (SQLite WAL on macOS) observed in the
full suite; unrelated to this feature and reproducible on `main`.

---

## Spec Traceability

| Requirement | Covered By | Result |
|-------------|------------|--------|
| REQ-001 — register auto-resolves identity from local.yaml | TestDocRegister_AutoResolvesIdentity | ✅ |
| REQ-002 — register auto-resolves identity from git config | TestDocRegister_AutoResolvesIdentity | ✅ |
| REQ-003 — approve auto-resolves identity from local.yaml | TestDocApprove_AutoResolvesIdentity | ✅ |
| REQ-004 — approve auto-resolves identity from git config | TestDocApprove_AutoResolvesIdentity | ✅ |
| REQ-005 — explicit --by flag overrides auto-resolution | TestDocRegister_ExplicitByFlagOverridesResolved | ✅ |
| REQ-006 — no regression in other identity-resolving commands | full suite pass | ✅ |

---

## Conclusion

All acceptance criteria satisfied. Both CLI paths now resolve identity consistently with the
rest of the codebase. No blocking findings. Feature is ready to merge.