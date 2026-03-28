# Release Infrastructure — Dev Plan

| Field        | Value                                              |
|--------------|----------------------------------------------------|
| **Feature**  | FEAT-01KMTSPEE6BXS (release-infrastructure)        |
| **Plan**     | P3-kanbanzai-1.0                                   |
| **Spec**     | `work/spec/release-infrastructure.md`               |
| **Status**   | Active                                             |
| **Created**  | 2026-03-28                                         |

---

## 1. Overview

Three mechanical tasks to prepare the repository for its first public release. The module path change is the critical dependency — it must land before the README update and before the binary-distribution branch can be rebased.

## 2. Task Execution Plan

### Task 1: module-path-change (TASK-01KMTT47PA5P5)

**Goal:** Change Go module path from `kanbanzai` to `github.com/sambeau/kanbanzai`.

**Steps:**
1. `go mod edit -module github.com/sambeau/kanbanzai`
2. `find . -name '*.go' -exec sed -i '' 's|"kanbanzai/|"github.com/sambeau/kanbanzai/|g' {} +`
3. Update `_testexternal/go.mod` to reference the new module path.
4. Run `go build ./...` — must exit 0.
5. Run `go test ./...` — must exit 0.
6. Run `go install ./cmd/kanbanzai` — must exit 0.
7. Verify: `grep -r '"kanbanzai/internal/' --include='*.go'` returns zero results.
8. Single atomic commit.

**ACs covered:** AC-1, AC-2, AC-3, AC-4, AC-5.

**Risk:** The sed replacement must not match inside string literals that aren't import paths. The pattern `"kanbanzai/` (with leading quote) is safe because Go import paths always appear in double-quoted strings.

### Task 2: go-version-evaluation (TASK-01KMTT47PSTBS)

**Goal:** Evaluate whether Go 1.25.0 is actually required and lower if possible.

**Steps:**
1. Check for Go 1.25-specific features: range-over-func (1.23), iterator patterns, new stdlib APIs introduced after 1.24.
2. Attempt build and test with `go 1.24.0` directive (or 1.23.x).
3. If build/test pass at a lower version, update `go.mod`.
4. If Go 1.25.0 features are found, document them and keep 1.25.0.
5. Record evaluation findings in commit message.

**ACs covered:** AC-6, AC-7.

**Decision:** Per DEC-01KMTTMSG2R3H, evaluate purely on technical grounds. Since kanbanzai ships as a binary, the minimum version only matters for source builders.

**Depends on:** None (can run in parallel with Task 1, but should use the updated module path if Task 1 lands first).

### Task 3: update-readme-and-docs (TASK-01KMTT47Q719Y)

**Goal:** Update README.md and docs/getting-started.md with real repository path and installation instructions.

**Steps:**
1. Replace any `your-org` or placeholder org references with `sambeau`.
2. Add remote install instruction: `go install github.com/sambeau/kanbanzai/cmd/kanbanzai@latest`
3. Add local build instructions: clone + `go build ./cmd/kanbanzai`.
4. Update `docs/getting-started.md` if it contains stale module path references.
5. Verify: `grep 'your-org' README.md` returns empty.

**ACs covered:** AC-8, AC-9, AC-10.

**Depends on:** Task 1 (module-path-change) — needs the real path to be in place.

**Note:** AC-11 (Makefile) and AC-12 (goreleaser) will be verified after binary-distribution branch rebase, not as part of this task.

## 3. Dependency Graph

```
module-path-change ──→ update-readme-and-docs
go-version-evaluation  (independent)
```

## 4. Verification

After all three tasks:
- `go build ./...` passes
- `go test ./...` passes
- `grep -r '"kanbanzai/internal/' --include='*.go'` returns zero
- `grep 'your-org' README.md` returns zero
- `go.mod` declares `module github.com/sambeau/kanbanzai`
- Go version in `go.mod` is justified by evaluation record