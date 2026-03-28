# MCP Discoverability and Reliability — Development Plan

| Document | MCP Discoverability and Reliability Development Plan |
|----------|------------------------------------------------------|
| Status   | Draft                                                |
| Created  | 2026-03-28                                           |
| Plan     | P9-mcp-discoverability                               |
| Feature  | FEAT-01KMTCWXAGX06                                   |
| Spec     | work/spec/mcp-discoverability-and-reliability.md     |
| Design   | work/design/mcp-discoverability-and-reliability.md   |

---

## 1. Overview

Seven implementation tasks across six features. Features A–C (annotations, titles,
descriptions) are pure tool-registration metadata with no runtime logic changes.
Features D–F (nudges, refresh, chain) add new behaviour and require logic + tests.

All work is in `internal/mcp/` except Feature E which also touches
`internal/service/documents.go`.

**Do not merge or cherry-pick from** `feature/FEAT-01KMKRQWF0FCH-hardening`. That
branch targets 1.0 tools. Use it as a reference only.

---

## 2. Task Breakdown

| Task ID | Slug | Feature | Effort | Depends On |
|---------|------|---------|--------|------------|
| TASK-01KMTCYNCX1HC | feat-a-tool-annotations | A | Small | — |
| TASK-01KMTCYNDC99R | feat-a-annotations-canary-test | A | Small | TASK-01KMTCYNCX1HC |
| TASK-01KMTCYNDTJWP | feat-b-tool-titles | B | Small | — |
| TASK-01KMTCYNE8AY9 | feat-c-tool-descriptions | C | Small | — |
| TASK-01KMTCYNEPYPN | feat-d-finish-response-nudges | D | Medium | — |
| TASK-01KMTCYNF4A4K | feat-e-doc-refresh-action | E | Medium | — |
| TASK-01KMTCYNFHFEP | feat-f-doc-chain-action | F | Tiny | — |

Tasks A (annotations), B (titles), and C (descriptions) are independent of each other
and may be executed in parallel or as a single commit. The canary test (Task 2)
depends on annotations (Task 1) being applied first. Tasks D, E, F are each
independent.

**Recommended sequence:**
1. A+B+C together (single commit: metadata sweep across all tool files)
2. Canary test (verifies the sweep)
3. D, E, F in any order (each is a self-contained commit)

---

## 3. Feature A — Tool Annotations

### Files to modify

Every `*_tool.go` file in `internal/mcp/`:

```
branch_tool.go         checkpoint_tool.go     cleanup_tool.go
conflict_tool.go       decompose_tool.go      doc_intel_tool.go
doc_tool.go            entity_tool.go         estimate_tool.go
finish_tool.go         handoff_tool.go        health_tool.go
incident_tool.go       knowledge_tool.go      merge_tool.go
next_tool.go           pr_tool.go             profile_tool.go
retro_tool.go          server_info_tool.go    status_tool.go
worktree_tool.go
```

### Implementation pattern

Add annotation options to the `mcp.NewTool(...)` call in each tool constructor.
The four helper functions may be used individually, or the single
`mcp.WithToolAnnotation(mcp.ToolAnnotation{...})` form may be used. Both are
equivalent; pick whichever keeps the diff cleaner.

Individual helpers (preferred for readability):

```go
mcp.NewTool("status",
    mcp.WithReadOnlyHintAnnotation(true),
    mcp.WithDestructiveHintAnnotation(false),
    mcp.WithIdempotentHintAnnotation(true),
    mcp.WithOpenWorldHintAnnotation(false),
    mcp.WithDescription("..."),
    // parameters...
)
```

Struct form (good when combining with title in one pass):

```go
mcp.NewTool("status",
    mcp.WithToolAnnotation(mcp.ToolAnnotation{
        Title:           mcp.Ptr("Workflow Status Dashboard"),
        ReadOnlyHint:    mcp.Ptr(true),
        DestructiveHint: mcp.Ptr(false),
        IdempotentHint:  mcp.Ptr(true),
        OpenWorldHint:   mcp.Ptr(false),
    }),
    mcp.WithDescription("..."),
    // parameters...
)
```

Check the mcp-go v0.45.0 source to confirm the exact helper function signatures
before committing. The `mcp.Ptr` helper may not exist — use address-of literals
(`ptr := true; &ptr`) or define a local helper if needed.

### Classification reference

See spec §3.2 for the full 22-row table. Tier 1 (read-only):
`status`, `health`, `handoff`, `conflict`, `profile`, `server_info`.
Tier 3 (destructive/open-world):
`cleanup`, `merge`, `pr`, `worktree`.
Everything else is Tier 2.

### Verification

```
go test ./internal/mcp/... -run TestAnnotations
```

The canary test (Task 2) is the definitive gate. Build passes with all existing
tests before writing the canary.

---

## 4. Feature A — Annotations Canary Test

### File to create

`internal/mcp/annotations_test.go`

### Design

The test must discover tools dynamically — do not hardcode the list of tool names.
Use the same server/group instantiation path as the existing `server_test.go` and
`server_groups_test.go` to collect all registered `server.ServerTool` values.

```go
func TestAllToolsHaveAnnotations(t *testing.T) {
    t.Parallel()

    // Instantiate all tools via the group constructors (see groups.go).
    tools := collectAllTools(t) // helper that calls each group and flattens

    for _, st := range tools {
        name := st.Tool.Name
        ann := st.Tool.Annotations

        if ann.ReadOnlyHint == nil {
            t.Errorf("tool %q: ReadOnlyHint is nil", name)
        }
        if ann.DestructiveHint == nil {
            t.Errorf("tool %q: DestructiveHint is nil", name)
        }
        if ann.IdempotentHint == nil {
            t.Errorf("tool %q: IdempotentHint is nil", name)
        }
        if ann.OpenWorldHint == nil {
            t.Errorf("tool %q: OpenWorldHint is nil", name)
        }
    }
}
```

Look at `server_groups_test.go` for the existing pattern of how groups are
instantiated for test purposes. Follow the same pattern.

### Optional stronger assertions

Add a second test that checks Tier 1 tools have `*ReadOnlyHint == true` and
Tier 3 tools have `*DestructiveHint == true || *OpenWorldHint == true`. Use
a map literal for the expected values — easier to maintain than a switch:

```go
var tier1 = map[string]bool{
    "status": true, "health": true, "handoff": true,
    "conflict": true, "profile": true, "server_info": true,
}
```

---

## 5. Feature B — Tool Titles

### Files to modify

Same 22 `*_tool.go` files as Feature A. If doing A and B in one pass, apply titles
alongside annotations in the same `mcp.NewTool` call.

### Title strings

See spec §4.2 for the exact strings. Examples:

| Tool | Title |
|------|-------|
| `status` | `Workflow Status Dashboard` |
| `finish` | `Task Completion` |
| `cleanup` | `Worktree Cleanup` |
| `server_info` | `Server Build Information` |

Apply verbatim — no truncation, no paraphrasing.

### Implementation

```go
mcp.WithTitleAnnotation("Workflow Status Dashboard")
```

or set the `Title` field in `mcp.WithToolAnnotation(...)`.

---

## 6. Feature C — Tool Descriptions

### Files to modify

Only 7 files need description changes:

| Tool | File |
|------|------|
| `status` | `status_tool.go` |
| `entity` | `entity_tool.go` |
| `next` | `next_tool.go` |
| `finish` | `finish_tool.go` |
| `doc` | `doc_tool.go` |
| `knowledge` | `knowledge_tool.go` |
| `retro` | `retro_tool.go` |

### Description text

Full replacement strings are in spec §5.2. Replace the existing `mcp.WithDescription(...)`
argument for each tool. Do not modify any other aspect of the tool.

The `doc` description in spec §5.2 already includes `refresh` and `chain` in the
actions list. If implementing Features E and F in the same PR, apply the final
description then. If implementing C before E/F, use an interim description that
excludes `refresh` and `chain`, then update once E and F are merged.

### Go string literal notes

Long descriptions span multiple lines in Go source. Use string concatenation or
raw string literals consistently with the existing style in each file. The rendered
description (what the MCP client receives) must match spec §5.2 content; Go source
formatting does not matter.

---

## 7. Feature D — `finish` Response Nudges

### File to modify

`internal/mcp/finish_tool.go`

### Implementation approach

The nudge logic sits in `finishOne`, after the task is completed and `resp` is
assembled. Two conditions are evaluated in priority order:

**Nudge 1 — Feature completion with no retro signals**

After the task is completed (status is now `done`), check:
1. Does the task have a `parent_feature`?
2. Load all tasks in that feature.
3. Are all tasks now in terminal status (`done`, `needs-review`, `cancelled`)?
4. Do any tasks in the feature have recorded retrospective signals?

For step 4, `dispatchSvc.CompleteTask` returns `result.RetroContributions`. Track
whether any task in the feature has retro data by loading completion records from
the entity service. The simplest approach: call `entitySvc.List("task", parent_feature)`
and check each task's state for the presence of `retro_signals` metadata. Alternatively,
query the knowledge base for signals tagged with the feature's task IDs. Choose the
approach that requires the fewest new service dependencies.

If conditions 1–4 are all met, set `resp["nudge"] = nudgeNoRetroSignals`.

**Nudge 2 — No knowledge or retrospective in this call**

If Nudge 1 was NOT triggered, and `input.Summary != ""` and `len(input.Knowledge) == 0`
and `len(input.RetroSignals) == 0` and not in batch mode, set
`resp["nudge"] = nudgeNoKnowledge`.

**Constants**

Define nudge strings as package-level constants (not inline) for easier testing:

```go
const nudgeNoRetroSignals = "No retrospective signals were recorded for any task in this feature. " +
    "If you observed workflow friction, tool gaps, spec ambiguity, or things that worked well, " +
    "call finish again on any completed task with the retrospective parameter, " +
    `or use knowledge(action: contribute) with tags: ["retrospective"].`

const nudgeNoKnowledge = "Consider including knowledge entries (reusable facts learned during " +
    "this task) or retrospective signals (process observations) in your finish call. " +
    "These improve context assembly for future tasks."
```

**Absence rule**

The `nudge` key must be absent from `resp` (not `nil`, not `""`) when no condition
is met. Use a conditional assignment rather than `resp["nudge"] = nil`.

**Batch mode**

The batch path in `finishTool` goes through `ExecuteBatch`, which calls `finishOne`
per item. Add a `isBatch bool` parameter to `finishOne`, or check at the call site.
The simplest approach: add a `batch bool` field to `finishInput` and set it before
calling `finishOne` in batch mode.

### Tests

Add to `finish_tool_test.go` (or a new `finish_nudge_test.go` if the existing file
is already large):

- `TestNudge1_FiredOnFeatureCompletionWithNoRetro` — all tasks done, no retro signals
- `TestNudge1_SuppressedWhenRetroSignalsExist` — at least one task has retro
- `TestNudge1_SuppressedWhenFeatureNotComplete` — sibling tasks still active
- `TestNudge2_FiredWhenSummaryPresentNoKnowledgeNoRetro` — single item, no extras
- `TestNudge2_SuppressedWhenKnowledgeProvided`
- `TestNudge2_SuppressedWhenRetroProvided`
- `TestNudge2_SuppressedWhenSummaryEmpty`
- `TestNudge_AbsentInBatchMode`
- `TestNudge_Nudge1TakesPriorityOverNudge2`

---

## 8. Feature E — `doc(action: refresh)`

### Files to modify

1. `internal/service/documents.go` — new `RefreshContentHash` method
2. `internal/mcp/doc_tool.go` — new action case

### Service layer

Add after `SupersessionChain` in `documents.go`:

```go
type RefreshInput struct {
    ID   string
    Path string
}

type RefreshResult struct {
    ID               string
    Path             string
    Changed          bool
    OldHash          string
    NewHash          string
    Status           string
    StatusTransition string
}

func (s *DocumentService) RefreshContentHash(input RefreshInput) (RefreshResult, error)
```

Behaviour (see spec §7.2 for full detail):
1. Validate: at least one of ID or Path is non-empty.
2. Resolve the record: prefer ID lookup; fall back to path-based lookup.
3. `filepath.Join(s.repoRoot, doc.Path)` → full file path.
4. `storage.ComputeContentHash(fullPath)` → current hash.
5. If file missing: return actionable error with the expected path.
6. Compare `currentHash` with `doc.ContentHash`.
7. If unchanged: return `RefreshResult{Changed: false, ...}` — no write.
8. If changed: update hash + `doc.Updated`. If status was `"approved"`,
   set to `"draft"` and record transition string. Write with optimistic locking.

Path-based lookup: scan the document store for a record matching `doc.Path == input.Path`.
Use the existing `ListDocuments` or equivalent store method — check `store.go` in the
storage package for a suitable traversal. Do not implement a new index; a linear scan
is acceptable.

### MCP layer

In `doc_tool.go`, add to the action switch after the existing cases:

```go
case "refresh":
    id := stringArg(args, "id")
    path := stringArg(args, "path")
    result, err := docSvc.RefreshContentHash(service.RefreshInput{ID: id, Path: path})
    if err != nil {
        return nil, err
    }
    return map[string]any{
        "id":                result.ID,
        "path":              result.Path,
        "changed":           result.Changed,
        "old_hash":          result.OldHash,
        "new_hash":          result.NewHash,
        "status":            result.Status,
        "status_transition": result.StatusTransition,
    }, nil
```

Also update the `action` parameter description in `doc_tool.go` to include `refresh`.

### Tests

Service-layer unit tests (new file `internal/service/doc_refresh_test.go` or added
to `documents_test.go`):

- Hash changed, draft document — hash updated, status stays draft
- Hash changed, approved document — hash updated, status transitions to draft,
  `StatusTransition == "approved → draft"`
- Hash unchanged — `Changed == false`, no write occurs (verify by checking
  modification time or by reading back and comparing)
- File not found — error message includes the expected file path
- Record not found (by ID) — error
- Record not found (by path) — error
- Both ID and Path empty — error `"id or path is required"`

MCP-layer test: add to `doc_tool_test.go` — verify `action: refresh` routes correctly
and returns the expected JSON shape.

### Reference

Commit `8ae3b7e` on `feature/FEAT-01KMKRQWF0FCH-hardening` has the 1.0 implementation
of `doc_record_refresh`. The service logic is similar; the MCP wiring is different
(1.0 used a standalone tool, 2.0 is an action on the consolidated `doc` tool).

---

## 9. Feature F — `doc(action: chain)`

### File to modify

`internal/mcp/doc_tool.go` only. No service changes.

### Implementation

The service method is already implemented and tested:

```go
// internal/service/documents.go
func (s *DocumentService) SupersessionChain(docID string) ([]DocumentResult, error)
```

Add to the action switch in `doc_tool.go`:

```go
case "chain":
    id := stringArg(args, "id")
    if strings.TrimSpace(id) == "" {
        return nil, fmt.Errorf("id is required for action: chain")
    }
    chain, err := docSvc.SupersessionChain(id)
    if err != nil {
        return nil, err
    }
    items := make([]map[string]any, len(chain))
    for i, doc := range chain {
        items[i] = map[string]any{
            "id":            doc.ID,
            "path":          doc.Path,
            "type":          doc.Type,
            "title":         doc.Title,
            "status":        doc.Status,
            "superseded_by": doc.SupersededBy,
        }
    }
    return map[string]any{
        "chain":  items,
        "length": len(chain),
    }, nil
```

Also update the `action` parameter description in `doc_tool.go` to include `chain`.

Check the `DocumentResult` struct in `internal/service/documents.go` to confirm
the field name for the superseded-by reference (`SupersededBy` or similar). The
struct is defined around L59–L80 of that file.

### Tests

Add to `doc_tool_test.go`:

- `TestDocChain_ReturnsChain` — set up two linked documents (v1 superseded by v2),
  call `action: chain`, verify the response shape and ordering.
- `TestDocChain_EmptyID` — verify error `"id is required for action: chain"`.
- `TestDocChain_NotFound` — verify error is returned when document does not exist.

The existing `internal/service/supersession_test.go` has fixture helpers
(`writeTestDocument`, `newTestDocService`) that can be reused for the MCP-layer
test setup.

---

## 10. Integration Checklist

Before marking the feature done:

- [ ] `go build ./...` — clean build
- [ ] `go test -race ./...` — all tests pass
- [ ] `go test ./internal/mcp/... -run TestAnnotations` — canary test passes
- [ ] Manual verification (optional): connect an MCP client (Claude Desktop or
      similar), confirm Tier 1 tools no longer trigger approval prompts, Tier 3
      tools still do.
- [ ] `doc` action parameter description lists all actions including `refresh`
      and `chain`.
- [ ] No 1.0 tool files or hardening branch code have been merged or copied
      without adaptation.

---

## 11. Key File Index

| File | Purpose |
|------|---------|
| `internal/mcp/*_tool.go` (22 files) | Tool registration — Features A, B, C |
| `internal/mcp/annotations_test.go` | New canary test — Feature A |
| `internal/mcp/finish_tool.go` | Nudge logic — Feature D |
| `internal/mcp/finish_tool_test.go` | Nudge tests — Feature D |
| `internal/mcp/doc_tool.go` | refresh + chain actions — Features E, F |
| `internal/mcp/doc_tool_test.go` | MCP-layer tests — Features E, F |
| `internal/service/documents.go` | `RefreshContentHash` method — Feature E |
| `internal/service/doc_refresh_test.go` | Service-layer tests — Feature E |
| `internal/service/supersession_test.go` | Existing fixtures for Feature F tests |
| `work/spec/mcp-discoverability-and-reliability.md` | Acceptance criteria |
| `work/design/mcp-discoverability-and-reliability.md` | Rationale + full tables |
| `work/reports/kanbanzai-2.0-migration-audit.md` | Context: what was lost in 2.0 |