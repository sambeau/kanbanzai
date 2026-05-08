# P61 Design — Handoff Resilience and Binding Hardening

**Status:** draft — design for the strategic remediation following BUG-01KR45KJWB2KY
**Companion:** `P61-report-handoff-investigation.md`
**Author:** architect
**Date:** 2026-05-08

---

## Overview

The Tier 1 stop-gap landed on `main` restored handoff service in 587 ms after a critical outage caused by a silent stage-bindings load failure cascading into a nil-pointer panic. This design covers the strategic remediation to prevent the *class* of failure recurring, and to harden the binding/pipeline subsystem in proportion to its load-bearing role in the orchestration workflow.

The investigation uncovered seven concrete weaknesses (T1–T7 in the report). This design groups them into three coherent tracks plus a documentation reconciliation track, sequenced for safe incremental delivery.

## Goals and Non-Goals

### Goals

1. **Make any future configuration error visible immediately, not silently.** The next operator who breaks `stage-bindings.yaml` should see the failure in the first MCP session, not several hours later via a mysterious "timeout".
2. **Make panics in MCP handlers impossible to misdiagnose as timeouts.** Every handler must convert recovered panics into structured JSON-RPC errors with diagnostic context.
3. **Give the binding schema a forward-compatible upgrade path.** Adding fields to `stage-bindings.yaml` should be a normal operation, not a silent server brick.
4. **Honor MCP request deadlines server-side.** Long-running git or filesystem work must be cancellable; future timeout vectors must be observable, not silent.
5. **Reconcile documentation with implementation.** The current docs describe handoff capabilities that do not exist; either the docs change or the capabilities are added — but the gap must close.

### Non-Goals

- **Re-platform the storage engine.** The investigation found no scaling problem. SQLite/FTS5 migration is a post-2027 concern.
- **Re-introduce the legacy assembly path.** The pipeline-3.0 unification was the right architectural call; the bug was in failure handling, not in the unification itself.
- **Add the missing handoff capabilities (spec sections, conflict annotations, graph traversal) under this plan.** Those are separate features. This plan only flags the gap (T6) so a decision can be made elsewhere.

## Design

### Track A — Configuration safety (T1, T5)

**T1. Schema-versioned `stage-bindings.yaml`.**
Introduce `schema_version: 2` at the top of the file. Loader inspects the version key first and dispatches to the appropriate decoder.

```go
type rawBindingFile struct {
    SchemaVersion int `yaml:"schema_version"`
    // remainder decoded by version-specific decoder
}

func LoadBindingFile(path string) (*BindingFile, []error) {
    data := readFile(path)
    var raw rawBindingFile
    yaml.Unmarshal(data, &raw)
    switch raw.SchemaVersion {
    case 0, 1:
        return decodeV1(data)            // current strict shape
    case 2:
        return decodeV2(data)            // adds Profile/Tier/Modes/Verifying as first-class
    default:
        return nil, []error{fmt.Errorf("unsupported schema_version %d (this binary supports 1, 2)", raw.SchemaVersion)}
    }
}
```

Migration path:
- Add `schema_version: 2` to the live `.kbz/stage-bindings.yaml`.
- Provide `kbz migrate stage-bindings` to add the version key idempotently.
- Older binaries see `schema_version: 2` and refuse with a clear message instead of silently mis-decoding.

**T5. `health()` surfaces `binding_loadable`.**
Add a check that calls `LoadBindingFile` at health-check time and reports any errors as warnings. Operators see binding problems in the regular health dashboard, not just at server start.

```go
type Check struct {
    Name string
    Status string // ok | warning | error
    Detail string
}
// in health.go:
checks = append(checks, runBindingLoadableCheck(stateRoot))
```

### Track B — Failure-mode resilience (T2, T4)

**T2. Propagate `context.Context` through pipeline and git helpers.**
Every MCP handler already receives a `context.Context`. We do not pass it down. Touch points:

- `service.EntityService.Get(ctx, ...)` — currently no ctx; add as first param.
- `git.CommitStateIfDirty(ctx, repoRoot)` — switch to `exec.CommandContext`.
- `kbzctx.Pipeline.Run(ctx, input)` — thread through; surface in `KnowledgeSurfacer.Surface` and `RoleResolver.Resolve` for future I/O.

Per-tool budget enforced at the MCP handler boundary:

```go
ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
defer cancel()
```

This is the bulk of the work — ~200 LOC and signature churn across packages. Land in a feature branch, not directly on main.

**T4. Audit and apply panic-recovery pattern to all MCP handlers.**
The pattern landed in handoff:

```go
defer func() {
    if r := recover(); r != nil {
        log.Printf("[%s] PANIC recovered: %v", toolName, r)
        toolResult = mcp.NewToolResultText(toolErrorJSON("internal_panic", ...))
        retErr = nil
    }
}()
```

Apply identically to every handler in `internal/mcp/*_tool.go`. Extract into a small helper:

```go
// in internal/mcp/handler.go
func wrapWithRecovery(toolName string, h ToolHandler) ToolHandler {
    return func(ctx context.Context, req mcp.CallToolRequest) (result *mcp.CallToolResult, err error) {
        defer recoverToToolError(toolName, &result, &err)
        return h(ctx, req)
    }
}
```

Wire via the existing `wrapAllTools` hook in `actionlog/hook.go`.

### Track C — Performance hygiene (T3, T7)

**T3. Knowledge entry cache with generation token.**
`KnowledgeService` exposes `Generation() (token, error)` derived from directory mtime + file count (cheap, no full scan):

```go
func (s *KnowledgeService) Generation() (string, error) {
    info, err := os.Stat(s.knowledgeDir)
    if err != nil { return "", err }
    return fmt.Sprintf("%d:%d", info.ModTime().UnixNano(), s.fileCountCached()), nil
}
```

`Surfacer` caches the loaded slice keyed by generation; reloads on miss. Negligible memory cost at current scale; meaningful at 5 000 entries.

**T7. De-duplicate `entitySvc.Get(parent_feature)` in handoff handler.**
Single Get; reuse for both re-review guidance and `input.FeatureState`. Trivial cleanup; bundled with the Track B context-propagation work.

### Track D — Documentation reconciliation (T6)

The current state of documentation vs. code:

| Source | Claims handoff does | Code actually does |
|---|---|---|
| `internal/mcp/handoff_tool.go` header | "uses pipeline unconditionally; pipeline validates" | true post-fix; was nil-deref pre-fix |
| User brief / project docs | spec sections, conflict annotations, graph traversal | none of these |
| `internal/mcp/assembly.go` header | "shared by next and handoff" | now used only by `next`; handoff uses `pipeline.go` |

T6 deliverables:
- Update `handoff_tool.go` header comment to describe actual behaviour.
- Update `internal/mcp/assembly.go` header to clarify it's `next`-only after the legacy-removal commits.
- Update `AGENTS.md` and `.github/copilot-instructions.md` handoff sections to match reality.
- Open a separate planning discussion: should the design-intent capabilities (spec sections etc.) be re-introduced as new features? If so, file as a separate FEAT under a different plan; this plan is about resilience, not capability addition.

## Alternatives Considered

1. **Loosen `KnownFields(true)` to silent-ignore unknown fields.** Rejected. Strictness is what catches typo'd role/skill names early. The bug here was not strictness but silent failure paired with no fallback. Fix the silent failure; keep the strictness. Schema versioning (T1) gives the same forward-compatibility benefit without losing typo protection.
2. **Move stage-bindings to SQLite or another structured store.** Rejected for scope. The current YAML file is appropriately human-editable; storage migration would be tail-wagging-the-dog given the bug was in error handling, not in the storage choice.
3. **Add a global `recover()` in the `mcp-go` worker.** Out of our control (third-party library). The handler-level wrapping (T4) is the appropriate layer.
4. **Async pre-computation / cache warming for handoff prompts.** Rejected. Handoff measured at 587 ms for a fully cold run including server start. There is no latency problem to solve.

## Dependencies

- **Track A T1** depends on no in-flight changes to `internal/binding`. Coordinate with B58 (constraint card stage binding hydration) which also touches the binding subsystem.
- **Track B T2** signature changes to `EntityService.Get` ripple through ~30 call sites. Land behind a feature flag or in a single atomic commit.
- **Track D T6** docs change can land independently of code work — recommend doing it first to prevent further misdiagnosis.

## Verification

For each track:

- **Track A.** Negative test: introduce `nonsense_field: 42` into `.kbz/stage-bindings.yaml`. `kbz serve` must fail at startup with `unsupported schema_version` or a structured strict-decode error, never start in pipeline-disabled mode. `health()` returns a `binding_loadable` warning when the file is malformed.
- **Track B.** Negative test: install a deliberate `time.Sleep(60s)` in a handler step and confirm the MCP request returns a structured timeout error within the 5 s budget, not a hang. Panic test: deliberate `panic("test")` in any handler returns `internal_panic` JSON within 1 s.
- **Track C.** Benchmark `Surface()` against the live 140-entry knowledge store: cached path < 1 ms, cold path < 100 ms. Generation token correctly invalidates after `kbz knowledge contribute`.
- **Track D.** `grep` for handoff capability claims across `AGENTS.md`, `.github/copilot-instructions.md`, `handoff_tool.go`, `assembly.go`; all references match implementation.

## Implementation Effort

| Track | Effort | Risk |
|---|---|---|
| A (T1+T5) | 1.5 days | Medium — touches foundational config |
| B (T2+T4) | 1.5–2 days | Medium — wide call-site changes |
| C (T3+T7) | 4 hours | Low |
| D (T6) | 4 hours | Low |
| **Total** | **~4 days** | — |

Recommend decomposition into one feature per track, with Track D landing first.

## Open Questions

1. Should `health()` emit a `binding_loadable` warning or escalate to error? Erroring would prevent the server from starting at all, which is safer but also less convenient if a stage binding is broken in a non-critical way.
2. Should the schema-versioned loader silently upgrade in-place (`v1 → v2` rewrite on disk) or require an explicit `kbz migrate` command? Lean toward explicit; in-place rewrites surprise users.
3. Track D — capability gap question: re-introduce spec sections / conflict annotations / graph traversal in handoff, or accept that these belong in `next` and adjust the docs? Out of scope for this design; flag for separate decision.
