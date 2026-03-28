# Design Proposal: `server_info` Tool and Post-Merge Install Automation

- Status: proposal
- Date: 2026-03-28
- Author: orchestrator
- Related: `work/reports/kanbanzai-2.0-workflow-retrospective.md` §3.11
- Retro signal: KE-01KMS0EE97M2P (stale MCP binary, tool-friction, moderate)

---

## 1. Problem

There are two related gaps that together cause the stale-binary problem:

**Gap 1 — No install step in the feature cycle.**
After a feature is merged into main, the `kanbanzai` binary on disk is not
updated automatically. The developer must remember to run `go install` manually.
During P6 Phase 2, this was not common knowledge: the binary was silently stale
while all tests passed and the MCP server appeared healthy.

**Gap 2 — No way to ask the running server what code it is executing.**
Even when an install has been run, there is no way for an agent or developer to
confirm whether the live MCP server has been restarted against the new binary.
The only diagnostic was manual: `ps aux | grep kanbanzai` + `ls -la` on the
binary. This is not something an agent can do reliably or consistently.

Together these gaps make stale-binary issues silent, slow to diagnose, and easy
to mistake for real failures.

---

## 2. Proposed Solution

Three coordinated changes:

1. **`server_info` MCP tool** — the running server can report exactly what code
   it is executing (git SHA, build timestamp, binary path).
2. **Install record** — after each `go install`, a small YAML file is written to
   `.kbz/` recording the SHA that was installed and when.
3. **Post-merge install step** — the `merge` tool (or an explicit `kbz install`
   command) runs `go install` automatically at the end of a feature cycle and
   writes the install record.

With all three in place, the sync check becomes a single tool call:
`server_info` returns the running SHA; the install record holds the expected SHA;
if they match, the server is current.

---

## 3. `server_info` Tool

### 3.1 Definition

**Tool name:** `server_info`

**Group:** `core` — always registered regardless of `mcp.groups` config.
Diagnostics must be available unconditionally.

**Arguments:** none

**Returns:**

```json
{
  "version":          "0.1.0",
  "git_sha":          "9322292a1f...",
  "git_sha_short":    "9322292",
  "build_time":       "2026-03-28T02:15:00Z",
  "go_version":       "go1.22.1",
  "binary_path":      "/Users/sam/go/bin/kanbanzai",
  "dirty":            false,
  "install_record":   {
    "git_sha":        "9322292a1f...",
    "installed_at":   "2026-03-28T02:16:00Z"
  },
  "in_sync":          true
}
```

| Field | Source | Notes |
|-------|--------|-------|
| `version` | `-ldflags -X` at build | Semantic version string; `"dev"` if unset |
| `git_sha` | `-ldflags -X` at build | Full SHA of the commit the binary was built from; `"unknown"` if unset |
| `git_sha_short` | Derived | First 7 characters of `git_sha` |
| `build_time` | `-ldflags -X` at build | RFC 3339 UTC; `"unknown"` if unset |
| `go_version` | `runtime.Version()` | Always available |
| `binary_path` | `os.Executable()` | Resolved symlinks |
| `dirty` | `-ldflags -X` at build | `true` if built from a worktree with uncommitted changes |
| `install_record` | `.kbz/last-install.yaml` | `null` if no install record exists |
| `in_sync` | Derived | `true` if `git_sha` matches `install_record.git_sha`; `null` if either is `"unknown"` or no record exists |

### 3.2 Reading `in_sync`

| `in_sync` value | Meaning |
|-----------------|---------|
| `true` | Server is running the same SHA that was last installed. |
| `false` | Server is running an older binary — a restart is needed. |
| `null` | Cannot determine — either no install record exists, or the binary was built without metadata injection (e.g. plain `go install`). |

A `null` is not an error. It is the expected state for binaries installed via
the plain `go install` path before the install automation is adopted.

### 3.3 Build-time variable injection

Declare variables in `internal/buildinfo/buildinfo.go`:

```go
package buildinfo

var (
    Version   = "dev"
    GitSHA    = "unknown"
    BuildTime = "unknown"
    Dirty     = "false"
)
```

Inject via `Makefile`:

```makefile
LDFLAGS := \
  -X kanbanzai/internal/buildinfo.Version=$(shell git describe --tags --always) \
  -X kanbanzai/internal/buildinfo.GitSHA=$(shell git rev-parse HEAD) \
  -X kanbanzai/internal/buildinfo.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X kanbanzai/internal/buildinfo.Dirty=$(shell git diff --quiet && echo false || echo true)

build:
	go build -ldflags "$(LDFLAGS)" -o $(GOPATH)/bin/kanbanzai ./cmd/kanbanzai

install:
	go build -ldflags "$(LDFLAGS)" -o $(GOPATH)/bin/kanbanzai ./cmd/kanbanzai
	kbz install-record write
```

The `install` target replaces bare `go install` in the development workflow.
It builds with full metadata and then writes the install record (see §4.2).

---

## 4. Install Record

### 4.1 File location and format

`.kbz/last-install.yaml` — tracked in `.kbz/`, ignored by git (add to
`.gitignore` or the existing `.kbz/` exclusion):

```yaml
git_sha: 9322292a1f36007b8c4d5e6f7a8b9c0d1e2f3a4b
installed_at: "2026-03-28T02:16:00Z"
installed_by: "make install"
binary_path: /Users/sam/go/bin/kanbanzai
```

| Field | Notes |
|-------|-------|
| `git_sha` | The HEAD SHA at the time `go install` was run |
| `installed_at` | RFC 3339 UTC timestamp of the install |
| `installed_by` | Free-form provenance: `"make install"`, `"kbz merge"`, `"manual"` |
| `binary_path` | Path the binary was written to |

The file is machine-written and human-readable. It is not committed — it is
per-machine state, analogous to `.kbz/local.yaml`.

### 4.2 Writing the record

A new CLI subcommand or internal helper writes the record:

```
kbz install-record write [--by <source>]
```

This is called automatically by `make install` and by the `merge` tool (§5).
It can also be called manually after a bare `go install` to bring the record
up to date:

```
go install ./cmd/kanbanzai && kbz install-record write --by manual
```

### 4.3 `server_info` reads the record

The `server_info` handler reads `.kbz/last-install.yaml` at call time (not at
startup) so that the record written after a restart is always current. Reading
at call time also means the handler never caches a stale value.

---

## 5. Post-Merge Install Step

### 5.1 Where in the feature cycle

The install should happen after the feature branch is merged into main. The
`merge` tool already handles the merge operation and branch cleanup. It is the
natural place to add the install step.

**Proposed `merge` tool behaviour (execute action):**

1. Verify merge gates (existing)
2. Merge branch into main (existing)
3. Delete branch / schedule cleanup (existing)
4. Run `go build -ldflags "..." -o $(GOPATH)/bin/kanbanzai ./cmd/kanbanzai`
5. Write `.kbz/last-install.yaml`
6. Return merge result with a new `install` field in `side_effects`

Step 4 is conditional: only runs if the repository root contains
`cmd/kanbanzai/main.go` (i.e. the repo is the kanbanzai project itself, not a
user project managed by kanbanzai). This keeps the behaviour contained and
avoids running arbitrary build steps in user repos.

Alternatively, the install step can be opt-in via `.kbz/config.yaml`:

```yaml
merge:
  post_merge_install: true
```

This is the safer default for a tool intended to manage other projects — only
the kanbanzai project itself needs auto-install on merge.

### 5.2 Restart notice

The MCP server cannot restart itself. After install, the merge tool should
include a prominent notice in its response:

```json
{
  "side_effects": [
    {
      "type": "install_complete",
      "git_sha": "9322292",
      "binary_path": "/Users/sam/go/bin/kanbanzai",
      "message": "Binary updated. Restart the MCP server in your IDE to load the new version."
    }
  ]
}
```

This makes the required human action explicit rather than relying on the
developer remembering.

### 5.3 Sync check workflow

After a feature cycle completes, an agent or developer can confirm the server
is current in a single call:

```
server_info()
→ in_sync: true   ✓ server is running the merged code
→ in_sync: false  ✗ server restart needed
→ in_sync: null   ? install record not present or binary built without metadata
```

The `health` check can also surface this as an attention item:

```
health() → warnings: ["MCP server is running an older binary (9bcaf9c). 
           Last install: 9322292. Restart the server to load the latest version."]
```

---

## 6. Scope

### In scope

| Component | Change |
|-----------|--------|
| `internal/buildinfo/` | New package — build-time variables |
| `cmd/kanbanzai/main.go` | Import buildinfo; pass to server |
| `internal/mcp/` | Register `server_info` in `core` group; read install record |
| `internal/merge/` | Add post-merge install step (opt-in via config) |
| `internal/install/` | New package — write/read `.kbz/last-install.yaml` |
| `cmd/kanbanzai/` | Add `install-record write` CLI subcommand |
| `Makefile` | `build` and `install` targets with ldflags |
| Test files | Update tool count assertions `20` → `21` |

### Out of scope

- `kbz version` CLI subcommand — shares `internal/buildinfo`; add separately
- Semantic versioning / tagging workflow
- Changelog or release automation
- Auto-restarting the MCP server (not possible from within the server process)

---

## 7. Open Questions

| # | Question | Suggested answer |
|---|----------|-----------------|
| Q1 | Should post-merge install be on by default? | No — opt-in via `merge.post_merge_install: true` in `.kbz/config.yaml`. Default off for user projects; the kanbanzai repo sets it to true. |
| Q2 | What if the build fails during merge? | Merge succeeds regardless; install failure is reported as a warning in side_effects, not a merge rollback. Code correctness and install are independent concerns. |
| Q3 | Should the health check warn when no install record exists? | No — only warn on a confirmed mismatch (`in_sync: false`). Missing record is a neutral `null`, not a problem. |
| Q4 | Plain `go install` without metadata — should it be blocked? | No. Plain `go install` remains valid; it just produces `in_sync: null` rather than `true`. The Makefile `install` target is the recommended path, not the only one. |