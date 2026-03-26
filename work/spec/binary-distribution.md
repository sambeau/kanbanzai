# Binary Distribution: Pre-compiled Releases via GitHub

| Document | Binary Distribution Specification |
|----------|----------------------------------|
| Status   | Draft                            |
| Created  | 2026-05-31                       |
| Updated  | 2026-05-31                       |
| Related  | `work/design/kanbanzai-1.0.md` §5 |

---

## 1. Purpose

This specification defines how Kanbanzai binaries are built, packaged, and distributed to end users. It covers the release pipeline, target platforms, archive formats, checksum verification, and the installation experience for new users.

---

## 2. Goals

- Users can download a pre-compiled, ready-to-run binary for their platform without needing a Go toolchain.
- Every tagged release produces binaries automatically, with no manual build steps.
- Users can verify binary integrity using a published checksum file.
- A new user can complete installation with a single command.
- The release pipeline is reproducible and auditable via GitHub Actions.

---

## 3. Scope

### 3.1 In scope

- GoReleaser configuration for cross-compilation and archive creation.
- GitHub Actions workflow triggered on version tags.
- Binary targets for the platform rollout schedule (alpha → beta → 1.0).
- Archive formats: `.tar.gz` for macOS and Linux, `.zip` for Windows.
- Checksum file (SHA-256) published alongside each release.
- GitHub Releases as the distribution channel.
- Single-command installation experience (via direct download script or equivalent).
- Release naming and asset naming conventions.

### 3.2 Deferred

- Homebrew tap (deferred past 1.0 alpha).
- Package manager integrations (APT, RPM, Scoop, Winget, etc.).
- Signed/notarised binaries (macOS Gatekeeper notarisation, Windows Authenticode).
- Automatic update checks within the binary.
- Docker image distribution.

### 3.3 Explicitly excluded

- Source-only releases (users who want to build from source use `go install` independently).
- Release of the MCP server as a separate standalone binary — it is the same binary as the CLI, invoked differently.
- Per-commit or rolling release channels (only tagged versions produce releases).

---

## 4. Design Principles

**Automate everything.** No human should need to run a build command or upload assets to produce a release. The entire pipeline runs on `git push --tags`.

**Verify by default.** Every release publishes a checksum file. Installation tooling must validate checksums before executing a downloaded binary.

**Staged rollout.** Platform support is added incrementally — alpha, beta, then 1.0 — rather than attempting all platforms at once. This reduces risk and lets early adopters surface platform-specific issues before the full audience is affected.

**Single binary, dual mode.** The same binary serves as both the CLI (`kbz`) and the MCP server (`kanbanzai serve`). Distribution produces one archive per platform, not separate CLI and server packages.

**Minimal installation friction.** A new user on a supported platform should reach a working binary in one command without reading multi-step instructions.

---

## 5. Platform Support and Rollout Schedule

### 5.1 Target platforms

| Platform | Architecture | Milestone |
|----------|-------------|-----------|
| macOS    | ARM64 (Apple Silicon) | 1.0 alpha |
| macOS    | AMD64 (Intel)         | 1.0 beta  |
| Linux    | AMD64                 | 1.0 beta  |
| Linux    | ARM64                 | 1.0 beta  |
| Windows  | AMD64                 | 1.0        |

GoReleaser's `builds` matrix must reflect this schedule. The alpha configuration builds only macOS ARM64. The beta configuration adds macOS AMD64, Linux AMD64, and Linux ARM64. The 1.0 configuration adds Windows AMD64. Version tag naming (e.g., `v1.0.0-alpha.1`, `v1.0.0-beta.1`, `v1.0.0`) determines which GoReleaser configuration is active, or a single configuration with build constraints.

### 5.2 GOARCH and GOOS values

| Platform | GOOS    | GOARCH |
|----------|---------|--------|
| macOS ARM64   | darwin  | arm64  |
| macOS AMD64   | darwin  | amd64  |
| Linux AMD64   | linux   | amd64  |
| Linux ARM64   | linux   | arm64  |
| Windows AMD64 | windows | amd64  |

---

## 6. Archive Format and Naming

### 6.1 Archive format by OS

| OS      | Format   |
|---------|----------|
| macOS   | `.tar.gz` |
| Linux   | `.tar.gz` |
| Windows | `.zip`   |

### 6.2 Archive naming convention

Archives must follow the pattern:

```
kanbanzai_<version>_<os>_<arch>.<ext>
```

Examples:
- `kanbanzai_1.0.0-alpha.1_darwin_arm64.tar.gz`
- `kanbanzai_1.0.0_linux_amd64.tar.gz`
- `kanbanzai_1.0.0_windows_amd64.zip`

Version strings must not include a leading `v` in the archive filename (GoReleaser default behaviour with `{{ .Version }}` strips the `v`).

### 6.3 Archive contents

Each archive must contain:
- The `kanbanzai` binary (or `kanbanzai.exe` on Windows).
- `README.md` from the repository root.
- `LICENSE` from the repository root.

No other files should be included.

### 6.4 Checksum file

Every release must include a single `checksums.txt` file listing the SHA-256 digest of every archive in the release, one entry per line in the format:

```
<sha256hex>  <filename>
```

This format is compatible with `sha256sum -c`. GoReleaser generates this automatically when `checksum.name_template` and `checksum.algorithm: sha256` are configured.

---

## 7. Release Pipeline

### 7.1 Trigger

The GitHub Actions release workflow is triggered exclusively by a push of a version tag matching `v*.*.*` (e.g., `v1.0.0-alpha.1`, `v1.0.0-beta.2`, `v1.0.0`). Pushes to branches do not trigger a release.

### 7.2 GoReleaser

The pipeline uses GoReleaser to:
1. Cross-compile binaries for all target platforms for the given release tier.
2. Package each binary into the appropriate archive format.
3. Generate `checksums.txt`.
4. Create or update the GitHub Release with all assets attached.

The GoReleaser configuration lives at `.goreleaser.yml` (or `.goreleaser.yaml`) in the repository root and is committed to version control. It is the authoritative source of truth for what each release produces.

### 7.3 GitHub Actions workflow

The workflow file lives at `.github/workflows/release.yml`. It must:
1. Check out the repository at the tagged commit.
2. Set up the Go toolchain at the version specified in `go.mod`.
3. Run `goreleaser release --clean`.
4. Not require any secrets beyond `GITHUB_TOKEN` (which GitHub Actions provides automatically).

The workflow must not perform any steps that modify source files or commit back to the repository.

### 7.4 Build reproducibility

All builds must be performed on the CI runner — local builds are for development only and are never uploaded as release assets. This ensures that release binaries are always produced from the exact committed source at the tagged SHA.

---

## 8. Installation Experience

### 8.1 Single-command install

A new user on a supported platform must be able to install Kanbanzai with a single shell command. The exact mechanism — a shell script hosted in the repository, a direct `curl | sh` download script, or a `go install` equivalent — is an implementation detail resolved during development. The requirement is that the user does not need to:
- Manually find the correct asset URL.
- Manually extract the archive.
- Manually move the binary to a `PATH` location.

### 8.2 Install script requirements

If an install script is provided, it must:
- Detect the host OS and architecture automatically.
- Download the correct archive for the detected platform.
- Download and validate `checksums.txt` before extracting.
- Abort with a clear error message if checksum validation fails.
- Install the binary to a sensible default location (e.g., `/usr/local/bin` on macOS/Linux, or a user-local `~/.local/bin`).
- Be idempotent — re-running it upgrades to the latest version without side effects.

### 8.3 Manual installation fallback

For users who cannot or prefer not to use the install script, the GitHub Releases page must clearly document the manual steps: download the archive, verify the checksum, extract, and place the binary on `PATH`. This documentation lives in the release notes template, not in this specification.

### 8.4 Post-install

After installation, the user runs `kanbanzai init` (or `kbz init`) to set up a project. That workflow is covered by the `project-initialisation` feature (FEAT-01KMKRQRRX3CC) and is out of scope here. This specification ends at the point where the binary is on the user's `PATH` and executes successfully.

---

## 9. Acceptance Criteria

1. Running the release GitHub Actions workflow on a version tag produces pre-compiled binaries for all target platforms defined for that release tier (alpha: macOS ARM64; beta: adds macOS AMD64, Linux AMD64, Linux ARM64; 1.0: adds Windows AMD64).

2. Each release produces archives in the correct format: `.tar.gz` for macOS and Linux targets, `.zip` for Windows targets.

3. Each release asset archive follows the naming convention `kanbanzai_<version>_<os>_<arch>.<ext>` with no leading `v` on the version component.

4. Every release includes a `checksums.txt` file containing a SHA-256 digest entry for every archive asset, in a format accepted by `sha256sum -c`.

5. Running `sha256sum -c checksums.txt` against a freshly downloaded release archive exits with status 0 (checksum passes).

6. The GitHub Actions release workflow requires no secrets beyond the automatically provided `GITHUB_TOKEN` — no personal access tokens or third-party credentials are needed.

7. The release workflow is triggered solely by a version tag push matching `v*.*.*` and is not triggered by branch pushes or pull requests.

8. Each archive contains exactly: the `kanbanzai` binary (or `kanbanzai.exe` on Windows), `README.md`, and `LICENSE`. No other files are present.

9. A user on macOS ARM64 can install Kanbanzai using a single command without manually locating, downloading, or extracting any archive.

10. The install script (or equivalent mechanism) validates the checksum of the downloaded archive and aborts with a non-zero exit code and an explanatory message if validation fails.

11. The GoReleaser configuration (`.goreleaser.yml`) is committed to the repository and is the sole source of truth for release asset generation — no undocumented manual steps are required to reproduce a release build.

12. After installation, running `kanbanzai --version` prints the version string matching the release tag and exits with status 0 on all supported platforms.