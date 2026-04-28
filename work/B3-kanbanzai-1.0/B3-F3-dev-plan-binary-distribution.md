# Binary Distribution: Feature Dev-Plan

| Document  | Binary Distribution Dev-Plan              |
|-----------|-------------------------------------------|
| Status    | Draft                                     |
| Feature   | FEAT-01KMKRQT9QCPR (binary-distribution)  |
| Spec      | work/spec/binary-distribution.md          |
| Created   | 2026-03-26                                |

---

## 1. Overview

This feature delivers pre-compiled binary distribution via GitHub Releases. It requires
GoReleaser configuration, a GitHub Actions release workflow, a `--version` flag, and an
install script. It has no hard dependency on other features — the pipeline can be set up
and validated as soon as the binary builds cleanly.

**Soft dependency:** the alpha release should include `kanbanzai init`, so binary-distribution
is best shipped after `init-command` (FEAT-01KMKRQRRX3CC) is complete. However, pipeline
infrastructure tasks (goreleaser config, release workflow, version flag) can proceed in
parallel with init-command implementation.

---

## 2. Tasks

### T1 — `version-flag`
Implement `kanbanzai --version` (and `kbz --version`). The version string is injected at
link time via `go build -ldflags "-X main.version=..."`. GoReleaser sets this automatically
via `{{ .Version }}`. Exits with status 0 and prints `kanbanzai <version>`.

**Acceptance criteria covered:** AC-12

---

### T2 — `goreleaser-config`
Create `.goreleaser.yml` at the repository root. Must include:
- `builds` matrix with alpha platform (macOS ARM64) expanding to beta and 1.0 platforms
- Archive configuration: `.tar.gz` for macOS/Linux, `.zip` for Windows
- Archive naming: `kanbanzai_{{ .Version }}_{{ .Os }}_{{ .Arch }}.{{ .Ext }}`
- Archive contents: binary + `README.md` + `LICENSE`
- Checksum: `checksums.txt`, algorithm SHA-256
- GitHub Releases integration

Commit the file. Test locally with `goreleaser check` (dry-run validation).

**Acceptance criteria covered:** AC-1, AC-2, AC-3, AC-4, AC-8, AC-11

---

### T3 — `release-workflow`
Create `.github/workflows/release.yml`. Must:
- Trigger only on `push` events with tags matching `v*.*.*`
- Check out at tagged commit
- Set up Go using the version in `go.mod`
- Run `goreleaser release --clean`
- Require no secrets beyond the automatically injected `GITHUB_TOKEN`

**Acceptance criteria covered:** AC-6, AC-7

---

### T4 — `install-script`
Write `install.sh` at the repository root. The script must:
- Detect host OS (`uname -s`) and architecture (`uname -m`)
- Map to the GoReleaser archive naming convention
- Download the correct archive from GitHub Releases (latest by default)
- Download and validate `checksums.txt` with `sha256sum -c` before extracting
- Abort with a clear error message and non-zero exit if checksum fails
- Install the binary to `/usr/local/bin` (falling back to `~/.local/bin` if not writable)
- Be idempotent: re-running upgrades in place

**Acceptance criteria covered:** AC-9, AC-10

---

### T5 — `release-validation`
Push a `v1.0.0-alpha.1` tag to trigger the workflow and verify end-to-end:
- Workflow runs to completion without manual steps
- Correct platform artefacts produced for alpha tier (macOS ARM64 only)
- `checksums.txt` present and passes `sha256sum -c`
- `kanbanzai --version` prints `1.0.0-alpha.1` on the downloaded binary
- Install script installs and runs the downloaded binary successfully

This task is not purely code — it is a validation step that should be performed by a human
or in a CI environment against a real tag. It is done when all AC pass against a real release.

**Acceptance criteria covered:** AC-1 through AC-12 (end-to-end)

---

## 3. Implementation Order

```
T1 (version-flag)  ──┐
T2 (goreleaser)    ──┤──► T5 (release-validation)
T3 (workflow)      ──┤
T4 (install-script)──┘
```

T1–T4 are independent and can be implemented in any order or in parallel. T5 requires
all four to be complete.

---

## 4. Notes

- GoReleaser must be added as a dev dependency (used only in CI; not in `go.mod`).
  The workflow installs it via `goreleaser/goreleaser-action` or direct download.
- The `install.sh` script should live at the root so users can do:
  `curl -sSfL https://raw.githubusercontent.com/.../main/install.sh | sh`
- `README.md` and `LICENSE` must exist at the repository root before T2 can be tested.
- Do not include any personal access tokens or third-party secrets in the workflow.
  `GITHUB_TOKEN` (auto-injected by GitHub Actions) is sufficient for `goreleaser`.