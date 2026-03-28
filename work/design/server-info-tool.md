# Design Proposal: `server_info` MCP Tool

- Status: proposal
- Date: 2026-03-28
- Author: orchestrator
- Related: `work/reports/kanbanzai-2.0-workflow-retrospective.md` §3.11
- Retro signal: KE-01KMS0EE97M2P (stale MCP binary, tool-friction, moderate)

---

## 1. Problem

When the `kanbanzai` MCP binary is stale — built before recent code changes — tool
calls silently return wrong results. During P6 Phase 2, this caused 3 of 9
acceptance criteria to appear failing when the code was correct. The only
diagnostic was manual: `ps aux | grep kanbanzai` to find the binary path, then
`ls -la` to check its modification time.

There is no way to ask the running server "what code are you actually running?"

## 2. Proposed Solution

Add a single read-only MCP tool, `server_info`, that returns build metadata
embedded in the binary at compile time.

### 2.1 Tool definition

**Tool name:** `server_info`

**Group:** `core`

**Arguments:** none

**Returns:**

```json
{
  "version":         "0.1.0",
  "git_sha":         "1f36007a3c...",
  "git_sha_short":   "1f36007",
  "build_time":      "2026-03-28T01:42:00Z",
  "go_version":      "go1.22.1",
  "binary_path":     "/Users/sam/go/bin/kanbanzai",
  "dirty":           false
}
```

| Field | Source | Notes |
|-------|--------|-------|
| `version` | `-ldflags -X` at build | Semantic version string; `"dev"` if unset |
| `git_sha` | `-ldflags -X` at build | Full 40-char SHA; `"unknown"` if unset |
| `git_sha_short` | Derived from `git_sha` | First 7 characters |
| `build_time` | `-ldflags -X` at build | RFC 3339 UTC; `"unknown"` if unset |
| `go_version` | `runtime.Version()` | Always available |
| `binary_path` | `os.Executable()` | Resolved symlinks |
| `dirty` | `-ldflags -X` at build | `true` if built from a worktree with uncommitted changes |

### 2.2 Diagnosing a stale binary

With this tool an agent can immediately determine whether the running server
matches the HEAD commit:

```
server_info() → { git_sha_short: "1f36007", build_time: "2026-03-27T..." }
git log -1 --format="%h %ci" → "9322292 2026-03-28..."
```

SHA mismatch → binary is stale → run `go install` and restart the MCP server.

## 3. Implementation

### 3.1 Build-time variable injection

Declare package-level variables in `cmd/kanbanzai/main.go` (or a dedicated
`internal/buildinfo/buildinfo.go`):

```go
var (
    Version   = "dev"
    GitSHA    = "unknown"
    BuildTime = "unknown"
    Dirty     = "false"
)
```

Inject values via the `Makefile` or `go build` invocation:

```makefile
LDFLAGS := \
  -X main.Version=$(shell git describe --tags --always --dirty) \
  -X main.GitSHA=$(shell git rev-parse HEAD) \
  -X main.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X main.Dirty=$(shell git diff --quiet && echo false || echo true)

build:
    go build -ldflags "$(LDFLAGS)" ./cmd/kanbanzai
```

`go install` does not honour a project `Makefile`. For the common development
path of `go install ./cmd/kanbanzai`, the variables will remain at their default
`"unknown"` / `"dev"` values — which is itself a signal: if `server_info` returns
`build_time: "unknown"`, the binary was installed without metadata injection and
the SHA cannot be trusted.

### 3.2 MCP handler

The handler is trivial — no I/O, no state, no error path:

```go
func handleServerInfo(req map[string]any) (map[string]any, error) {
    binaryPath, _ := os.Executable()
    shaShort := GitSHA
    if len(GitSHA) >= 7 {
        shaShort = GitSHA[:7]
    }
    dirty := Dirty == "true"
    return map[string]any{
        "version":       Version,
        "git_sha":       GitSHA,
        "git_sha_short": shaShort,
        "build_time":    BuildTime,
        "go_version":    runtime.Version(),
        "binary_path":   binaryPath,
        "dirty":         dirty,
    }, nil
}
```

### 3.3 Tool count

The current server registers 20 tools. Adding `server_info` makes 21. Three test
files assert the count as the literal `20` (retro item D3). Those assertions must
be updated. This is the only test change required.

## 4. Scope

**In scope:**
- `cmd/kanbanzai/main.go` — declare build variables
- `internal/mcp/` — register `server_info` tool in the `core` group
- `Makefile` (or equivalent) — inject ldflags at build time
- Test files asserting tool count — update `20` → `21`

**Out of scope:**
- Structured version file (e.g. `VERSION`) — not needed; git describe is sufficient
- A `kbz version` CLI subcommand — useful but separate; share the same buildinfo
  package when added
- Changelog or release tooling

## 5. Open Questions

| # | Question | Suggested answer |
|---|----------|-----------------|
| Q1 | Should `server_info` appear in `mcp.groups` filtering? | No — always registered regardless of group config. Diagnostics should always be available. |
| Q2 | Should `go install` path also inject metadata? | A wrapper script or `tools.go` approach could help, but is optional. Returning `"unknown"` for dev installs is an acceptable and informative default. |
| Q3 | Same info via `kbz version` CLI? | Yes, eventually — share the same `internal/buildinfo` package. Out of scope for this proposal. |