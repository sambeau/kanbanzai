# Release Infrastructure Specification

| Field             | Value                                                          |
|-------------------|----------------------------------------------------------------|
| **Status**        | Draft                                                          |
| **Created**       | 2026-03-28                                                     |
| **Feature**       | FEAT-01KMTSPEE6BXS (release-infrastructure)                   |
| **Plan**          | P3-kanbanzai-1.0                                               |
| **Related docs**  | `work/design/kanbanzai-1.0.md` §5 (Distribution & Installation) |

---

## 1. Purpose

This specification defines the acceptance criteria for resolving infrastructure issues that block the first public release of Kanbanzai. These issues are mechanical but foundational — without them, external users cannot install, build, or contribute to the project.

The four areas are:

1. **Module path** — `go.mod` declares a bare `module kanbanzai` path that is incompatible with `go install` from GitHub.
2. **Go version** — `go.mod` requires Go 1.25.0, which external users may not have.
3. **README and documentation** — installation instructions contain placeholder values.
4. **Build system** — `Makefile` and `.goreleaser.yml` must be compatible with the corrected module path.

---

## 2. Scope

### In scope

- Changing the Go module path to match the canonical GitHub repository URL.
- Updating all internal import paths across every `.go` file.
- Evaluating and potentially lowering the minimum Go version.
- Updating `README.md` and `docs/getting-started.md` with real installation instructions.
- Ensuring `Makefile` and `.goreleaser.yml` compatibility.

### Out of scope

- Choosing the GitHub owner/org name (human decision — see §5).
- Setting up CI/CD pipelines.
- Creating release binaries or distribution packages.
- Homebrew formula, package manager integration, or other distribution channels.
- Changes to the MCP server protocol or tool surface.

---

## 3. Acceptance Criteria

### Module path

**AC-1: Module path matches GitHub repository URL.**
`go.mod` must declare a module path in the form `github.com/<owner>/kanbanzai`, where `<owner>` is the canonical GitHub owner or organisation. The specific value of `<owner>` is a prerequisite decision (see §5).

**AC-2: All import paths updated.**
Every `.go` file in the repository must use the new module path in its import statements. No references to the bare `kanbanzai/internal/...` import path may remain anywhere in the codebase. Verified by: `grep -r '"kanbanzai/internal/' --include='*.go'` returns zero results.

**AC-3: Build succeeds.**
`go build ./...` must complete without errors using the new module path.

**AC-4: Tests pass.**
`go test ./...` must pass with the new module path. No test may be skipped or disabled to satisfy this criterion.

**AC-5: Local install succeeds.**
`go install ./cmd/kanbanzai` must succeed when run from a local checkout of the repository.

### Go version

**AC-6: Go version evaluated and documented.**
The Go version declared in `go.mod` must be evaluated against actual usage. If no language features or standard library APIs specific to Go 1.25.0 are used, the version must be lowered to the latest stable release (1.24.x or 1.23.x as appropriate). The evaluation must produce a written record (in the implementation PR or commit message) documenting:
- Which Go 1.25.0-specific features were searched for.
- Whether any were found.
- The chosen minimum version and rationale.

**AC-7: Build and tests pass at the chosen version.**
If the Go version is lowered, both `go build ./...` and `go test ./...` must pass with that version. This must be verified by building with the target Go version, not merely by changing the directive.

### README and documentation

**AC-8: Real repository path in README.**
`README.md` installation instructions must reference the real repository path (`github.com/<owner>/kanbanzai`), not the `your-org` placeholder. No occurrence of `your-org` may remain in `README.md`.

**AC-9: Complete installation instructions.**
`README.md` must include instructions for both:
- Remote install: `go install github.com/<owner>/kanbanzai/cmd/kanbanzai@latest`
- Local build: cloning the repository and running `go build` / `go install` from the checkout.

**AC-10: Getting-started guide updated.**
`docs/getting-started.md` must be updated to use the real module path and repository URL. If the file does not reference the old module path or placeholder org, no change is required (this criterion is satisfied trivially).

### Build system

**AC-11: Makefile compatible.**
The `Makefile` must work with the new module path. All targets that reference the module path (build, install, test) must succeed.

**AC-12: goreleaser compatible.**
`.goreleaser.yml` (on the `binary-distribution` branch) must be compatible with the new module path. This may be verified after the module path change is merged to `main` and the branch is rebased or updated. The criterion is satisfied when `goreleaser check` passes with the new configuration.

---

## 4. Verification

All acceptance criteria are verified by running the specified commands or checks against the repository after implementation. The verification is mechanical:

| Criterion | Verification method                                              |
|-----------|------------------------------------------------------------------|
| AC-1      | Inspect `go.mod` first line                                      |
| AC-2      | `grep -r '"kanbanzai/internal/' --include='*.go'` returns empty  |
| AC-3      | `go build ./...` exits 0                                         |
| AC-4      | `go test ./...` exits 0                                          |
| AC-5      | `go install ./cmd/kanbanzai` exits 0                             |
| AC-6      | Written evaluation exists in PR or commit                        |
| AC-7      | Build and test with target Go version                            |
| AC-8      | `grep 'your-org' README.md` returns empty                        |
| AC-9      | README contains both remote and local install instructions        |
| AC-10     | Inspect `docs/getting-started.md` for stale references           |
| AC-11     | `make build` and `make test` succeed                             |
| AC-12     | `goreleaser check` passes on binary-distribution branch          |

---

## 5. Prerequisites

**GitHub owner/org name is a human decision.** This specification intentionally does not prescribe the value of `<owner>` in the module path `github.com/<owner>/kanbanzai`. This decision must be made and communicated by the project owner before implementation begins. The implementation task must not guess or assume a value.

**Go version evaluation requires the target Go toolchain.** AC-7 requires building with the chosen minimum Go version. The implementer must have access to that version (e.g., via `go install golang.org/dl/goX.Y.Z@latest` or a version manager).

---

## 6. Implementation Notes

These are non-normative observations to assist implementation:

- The import path change is mechanical and can be performed with `sed`, `goimports`, or `go mod edit -module` followed by a find-and-replace. The key risk is missing a reference — AC-2's grep check catches this.
- The `go.mod` module directive and all import paths must be changed atomically in a single commit to keep the repository buildable at every commit.
- The Go version evaluation (AC-6) should check for usage of: range-over-func, iterator patterns, new standard library functions, and any other features introduced after the candidate minimum version.