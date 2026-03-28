# server_info Tool Specification

| Document | server_info Tool and Post-Merge Install Automation |
|----------|----------------------------------------------------|
| Status   | Draft                                              |
| Created  | 2026-03-28T11:42:33Z                               |
| Updated  | 2026-03-28T11:42:33Z                               |
| Feature  | FEAT-01KMT-40GZSMHB (server-info-tool)             |
| Plan     | P7-developer-experience                            |
| Design   | `work/design/server-info-tool.md`                  |

---

## 1. Purpose

This specification defines three coordinated changes that together eliminate the
stale-binary problem identified in the P6 retrospective:

1. A `server_info` MCP tool that lets an agent or developer ask the running
   server exactly what code it is executing.
2. A `.kbz/last-install.yaml` install record written after each build, recording
   which SHA was installed and when.
3. A post-merge install step in the `merge` tool that rebuilds the binary
   automatically after a feature lands on main.

With all three in place, confirming server currency becomes a single tool call.

---

## 2. Goals

1. An agent can call `server_info` at any time and receive the running binary's
   version, git SHA, build timestamp, Go version, and binary path.
2. The `server_info` response includes an `install_record` block read from
   `.kbz/last-install.yaml` and a derived `in_sync` boolean.
3. The `server_info` tool is registered in the `core` group and is always
   available regardless of `mcp.groups` configuration.
4. Build-time metadata is injected via `-ldflags` from a `Makefile`; the binary
   works correctly when built without ldflags (all fields default to `"unknown"`
   or `"dev"`).
5. After every `merge execute` on a kanbanzai repository, the binary is rebuilt
   and the install record is written automatically.
6. If the post-merge build fails, the merge still succeeds; the failure is
   reported as a warning in `side_effects`.
7. The `merge` tool response includes an explicit restart notice in `side_effects`
   when an install completes successfully.
8. Post-merge install is on by default when `cmd/kanbanzai/main.go` exists at
   the repository root, and can be disabled via `merge.post_merge_install: false`
   in `.kbz/config.yaml`.
9. A `kbz install-record write` CLI subcommand allows manual record writing after
   a plain `go install`.
10. The total registered MCP tool count increases from 20 to 21.
11. All existing tests continue to pass.

---

## 3. Scope

### 3.1 In scope

- New package `internal/buildinfo/` with build-time variables (`Version`,
  `GitSHA`, `BuildTime`, `Dirty`).
- Updates to `cmd/kanbanzai/main.go` to import `buildinfo` and pass metadata to
  the MCP server.
- New package `internal/install/` with functions to write and read
  `.kbz/last-install.yaml`.
- New `server_info` handler in `internal/mcp/` registered in the `core` group.
- Updates to `internal/merge/` to run the post-merge install step (opt-in via
  config auto-detection; opt-out via `merge.post_merge_install: false`).
- `install-record write` subcommand in `cmd/kanbanzai/`.
- `Makefile` with `build` and `install` targets that inject ldflags.
- Updates to test files that assert on the total tool count (`20` → `21`).

### 3.2 Deferred

- `kbz version` CLI subcommand (shares `internal/buildinfo`; add separately).
- Semantic versioning or release tagging workflow.
- Changelog or release automation.
- Auto-restarting the MCP server (not possible from within the server process).
- Health check warning for `in_sync: false` (can be added as a follow-up).

### 3.3 Explicitly excluded

- Changes to any entity lifecycle, storage, or document record logic.
- Any changes to tools other than `merge` and the new `server_info`.
- Platform-specific binary distribution or CI/CD pipeline changes.

---

## 4. Acceptance Criteria

### 4.1 `internal/buildinfo` package

**AC-01.** A package `internal/buildinfo` exists and exports four `var` declarations:

```go
var (
    Version   = "dev"
    GitSHA    = "unknown"
    BuildTime = "unknown"
    Dirty     = "false"
)
```

**AC-02.** All four variables have string type. Default values are `"dev"` for
`Version` and `"unknown"` for `GitSHA`, `BuildTime`. `Dirty` defaults to
`"false"` (string, not bool) because it is set via `-ldflags`.

**AC-03.** The package has no imports from `internal/` or external packages —
it is a pure declaration package with no side effects.

### 4.2 Makefile

**AC-04.** A `Makefile` exists at the repository root with at minimum two
targets: `build` and `install`.

**AC-05.** Both targets build `./cmd/kanbanzai` with `-ldflags` that inject:
- `kanbanzai/internal/buildinfo.Version` from `git describe --tags --always`
- `kanbanzai/internal/buildinfo.GitSHA` from `git rev-parse HEAD`
- `kanbanzai/internal/buildinfo.BuildTime` from `date -u +%Y-%m-%dT%H:%M:%SZ`
- `kanbanzai/internal/buildinfo.Dirty` as `false` if `git diff --quiet`, else `true`

**AC-06.** The `install` target, after building, calls `kbz install-record write`
(or equivalent) to write `.kbz/last-install.yaml`.

**AC-07.** `go build ./...` and `go test ./...` continue to work without the
Makefile; no build step is mandatory for tests.

### 4.3 `internal/install` package

**AC-08.** A package `internal/install` exists and exports at minimum:
- `WriteRecord(root, gitSHA, binaryPath, installedBy string) error` — writes
  `.kbz/last-install.yaml`.
- `ReadRecord(root string) (*InstallRecord, error)` — reads the file; returns
  `nil, nil` if the file does not exist (not an error).

**AC-09.** `InstallRecord` is a struct with exported fields matching the YAML
schema:

```yaml
git_sha: <string>
installed_at: <RFC 3339 UTC>
installed_by: <string>
binary_path: <string>
```

**AC-10.** The YAML serialisation of `InstallRecord` follows the project's
canonical YAML rules: block style, double-quoted strings only where required,
deterministic field order (`git_sha`, `installed_at`, `installed_by`,
`binary_path`), UTF-8, LF endings, trailing newline.

**AC-11.** `WriteRecord` writes atomically (write to temp file, then rename)
so a concurrent read never sees a partial file.

**AC-12.** The file is written to `<root>/.kbz/last-install.yaml` where `root`
is the project root (same directory that contains `.kbz/config.yaml`).

**AC-13.** `.kbz/last-install.yaml` is either already in `.gitignore` (covered
by an existing `.kbz/` exclusion) or an exclusion is explicitly added. The
file must not be committed.

### 4.4 `kbz install-record write` CLI subcommand

**AC-14.** The command `kbz install-record write` runs without error when the
repository has a `.kbz/` directory.

**AC-15.** It accepts an optional `--by <source>` flag; when omitted, the
`installed_by` field defaults to `"manual"`.

**AC-16.** It writes the install record with the current `git rev-parse HEAD`
SHA and the path returned by `os.Executable()`.

**AC-17.** It exits with a non-zero code and prints a diagnostic if run outside
a kanbanzai-initialised directory (no `.kbz/config.yaml`).

### 4.5 `server_info` MCP tool

**AC-18.** A tool named `server_info` is registered in the MCP server.

**AC-19.** `server_info` belongs to the `core` group and is registered
unconditionally — it is present regardless of the `mcp.groups` or `mcp.preset`
configuration.

**AC-20.** `server_info` accepts no arguments.

**AC-21.** `server_info` returns a JSON object with the following fields:

| Field | Type | Source |
|-------|------|--------|
| `version` | string | `buildinfo.Version` |
| `git_sha` | string | `buildinfo.GitSHA` |
| `git_sha_short` | string | First 7 chars of `git_sha`; `"unknown"` if `git_sha` is `"unknown"` |
| `build_time` | string | `buildinfo.BuildTime` |
| `go_version` | string | `runtime.Version()` |
| `binary_path` | string | `os.Executable()` with symlinks resolved |
| `dirty` | bool | `buildinfo.Dirty == "true"` |
| `install_record` | object or null | Content of `.kbz/last-install.yaml`; `null` if file absent |
| `in_sync` | bool or null | Derived (see AC-22) |

**AC-22.** `in_sync` is computed as follows:
- `true` — `git_sha` equals `install_record.git_sha` and neither is `"unknown"`.
- `false` — both are known strings and they differ.
- `null` — `git_sha` is `"unknown"`, or `install_record` is `null`, or
  `install_record.git_sha` is empty.

**AC-23.** The install record is read from disk at each `server_info` call (not
cached at startup), so a newly written record is immediately reflected.

**AC-24.** If `.kbz/last-install.yaml` cannot be read due to a filesystem error
other than "file not found", `server_info` returns the error as a tool error
rather than silently returning `null`.

**AC-25.** The total number of registered MCP tools across all groups is 21.
Any test that previously asserted a count of 20 must be updated to 21.

### 4.6 Post-merge install step

**AC-26.** When `merge execute` is called and `cmd/kanbanzai/main.go` exists at
the repository root, the merge tool attempts to build the binary with ldflags
and write the install record after a successful merge.

**AC-27.** When `merge.post_merge_install: false` is set in `.kbz/config.yaml`,
the post-merge install step is skipped entirely; no build is attempted.

**AC-28.** When `cmd/kanbanzai/main.go` does not exist at the repository root,
the post-merge install step is skipped silently; no error or warning is emitted.

**AC-29.** If the post-merge build succeeds, `side_effects` includes an entry of
type `install_complete` containing at minimum: `git_sha` (short form), `binary_path`,
and a human-readable `message` that instructs the user to restart the MCP server.

**AC-30.** If the post-merge build fails, the merge operation is not rolled back.
The merge result is still a success; `side_effects` includes a warning entry of
type `install_failed` with the build error message.

**AC-31.** The restart notice in `side_effects` is present and non-empty whenever
`install_complete` is reported, so the required human action is always explicit.

### 4.7 Build correctness without ldflags

**AC-32.** A binary built with plain `go build ./cmd/kanbanzai` (no ldflags)
starts without error. All `buildinfo` fields return their default values
(`"dev"`, `"unknown"`, `"unknown"`, `"false"`).

**AC-33.** `server_info` called against such a binary returns `in_sync: null`
(not an error), because `git_sha` is `"unknown"`.

### 4.8 Testing

**AC-34.** `internal/buildinfo` has at least one test confirming the default
values of all four variables.

**AC-35.** `internal/install` has round-trip tests: `WriteRecord` followed by
`ReadRecord` returns the same values that were written.

**AC-36.** `internal/install` has a test confirming that `ReadRecord` returns
`nil, nil` when the file does not exist.

**AC-37.** The `server_info` handler has a unit test confirming `in_sync` is:
- `true` when both SHAs match and are known.
- `false` when both are known and differ.
- `null` when either is unknown or the record is absent.

**AC-38.** All tests pass under `go test -race ./...`.

---

## 5. File and Package Summary

| Path | Action |
|------|--------|
| `internal/buildinfo/buildinfo.go` | Create |
| `internal/buildinfo/buildinfo_test.go` | Create |
| `internal/install/install.go` | Create |
| `internal/install/install_test.go` | Create |
| `internal/mcp/server_info.go` (or inline) | Create |
| `internal/mcp/tools.go` (or equivalent) | Update — register `server_info` in core group |
| `internal/merge/merge.go` (or equivalent) | Update — post-merge install step |
| `cmd/kanbanzai/main.go` | Update — import buildinfo; pass to server |
| `cmd/kanbanzai/install_record.go` (or inline) | Create — `install-record write` subcommand |
| `Makefile` | Create |
| Test files asserting tool count | Update — 20 → 21 |