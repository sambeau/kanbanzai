# Design: Standalone bugs visible in status health

| Field | Value |
|-------|-------|
| Date | 2025-07-17 |
| Author | Architect |
| Feature | FEAT-01KPPG3MSRRCE — Standalone bugs visible in status health |
| Plan | P24-retro-recommendations |

---

## 1. Overview

P19 added attention items to `status` responses and wired in open high/critical bugs — but only
for bugs linked to a specific in-flight feature via the `origin_feature` field. A bug filed
against general code with no feature attachment (`origin_feature == ""`) is invisible in every
`status` call. It does not appear at project scope, plan scope, or anywhere else.

This design closes that gap. The fix is a targeted addition to `synthesiseProject` in
`internal/mcp/status_tool.go`: after the existing `generateProjectAttention` call, load all bugs
and append attention items for open standalone bugs whose severity is `high` or `critical`.

---

## 2. Goals and Non-Goals

### Goals

- Open bugs with severity `high` or `critical` and no `origin_feature` appear in project-level
  `status` attention items.
- Double-surfacing is impossible by construction: feature-linked bugs surface at feature scope;
  standalone bugs surface at project scope; the two filters are disjoint.
- The change is additive — no existing attention items are removed or reordered.

### Non-Goals

- Surfacing standalone bugs at plan scope or feature scope (project scope is the natural home for
  unattached bugs).
- Surfacing standalone bugs with severity `medium` or `low` (matches the existing threshold used
  for feature-linked bugs).
- Changing how feature-linked bugs are surfaced in `synthesiseFeature`.
- Adding a new `origin_feature` field validation or enforcement mechanism.

---

## 3. Design

### 3.1 Where the filter lives today

`synthesiseFeature` (line ~786 in `internal/mcp/status_tool.go`) loads all bugs and filters them
down to those where `origin_feature == featID`:

```kanbanzai/internal/mcp/status_tool.go#L786-800
var openBugs []bugItem
if allBugs, bugErr := entitySvc.List("bug"); bugErr == nil {
    for _, b := range allBugs {
        originFeature, _ := b.State["origin_feature"].(string)
        if originFeature != featID {
            continue
        }
        // ... terminal-status skip, severity check, append
    }
}
```

`synthesiseProject` (line ~282) calls `generateProjectAttention` and then injects health-check
items. It performs no bug query at all. `generateProjectAttention` itself takes `plans, allTasks,
worktreeBranches, repoPath` — no bug data.

### 3.2 What to change

Add a standalone-bug pass directly in `synthesiseProject`, after `generateProjectAttention` returns
and before the health-item injection. This mirrors the structural pattern already used for health
items (a separate, ordered append block):

```kanbanzai/internal/mcp/status_tool.go#L393-415
attention := generateProjectAttention(summaries, allTasks, worktreeBranches, repoPath)

// Surface standalone high/critical bugs (no feature linkage) — spec gap P19 C4.
if allBugs, bugErr := entitySvc.List("bug"); bugErr == nil {
    for _, b := range allBugs {
        originFeature, _ := b.State["origin_feature"].(string)
        if originFeature != "" {
            continue // feature-linked bugs are surfaced at feature scope
        }
        bStatus, _ := b.State["status"].(string)
        switch bStatus {
        case "done", "closed", "not-planned", "duplicate", "wont-fix":
            continue
        }
        bSeverity, _ := b.State["severity"].(string)
        if bSeverity != "high" && bSeverity != "critical" {
            continue
        }
        bID, _ := b.State["id"].(string)
        bName, _ := b.State["name"].(string)
        msg := fmt.Sprintf("Standalone %s bug: %s", bSeverity, bName)
        if bName == "" {
            msg = fmt.Sprintf("Standalone %s bug: %s", bSeverity, bID)
        }
        attention = append(attention, AttentionItem{
            Type:      "open_critical_bug",
            Severity:  "warning",
            EntityID:  bID,
            DisplayID: id.FormatFullDisplay(bID),
            Message:   msg,
        })
    }
}

// Pillar C — T7: inject health findings …
```

### 3.3 Filter conditions

| Condition | Value | Rationale |
|-----------|-------|-----------|
| `origin_feature` | `""` (empty or absent) | Standalone bugs only; feature-linked bugs handled elsewhere |
| `status` | NOT in `done`, `closed`, `not-planned`, `duplicate`, `wont-fix` | Matches the resolved-bug skip list in `synthesiseFeature` |
| `severity` | `high` or `critical` | Matches the existing threshold used for feature-linked bug warnings (REQ-025) |

### 3.4 Double-surfacing avoidance

Double-surfacing cannot occur by construction:

- `synthesiseFeature` filter: `origin_feature == featID` (non-empty, specific feature)
- `synthesiseProject` filter (new): `origin_feature == ""` (empty)

A bug satisfies exactly one condition. A bug shown in a feature-detail `status(id: FEAT-...)` call
will never appear in the project-level `status()` attention list, and vice versa.

`maxAttentionItems` applies inside `generateProjectAttention`. The standalone-bug block appends
after this function returns. To avoid unbounded growth, the implementation should guard each append
with `len(attention) < maxAttentionItems` (or a reasonable per-category cap), consistent with how
health items are appended without an explicit cap but are naturally bounded by the number of health
findings.

### 3.5 AttentionItem shape

```kanbanzai/internal/mcp/status_tool.go#L179-186
AttentionItem{
    Type:      "open_critical_bug",   // reuses existing type constant
    Severity:  "warning",
    EntityID:  bugID,
    DisplayID: id.FormatFullDisplay(bugID),
    Message:   "Standalone <severity> bug: <name>",
}
```

Reusing `"open_critical_bug"` as the `Type` value keeps the consumer interface stable — clients
that already handle this type for feature-linked bugs will handle standalone bugs without any
schema change.

### 3.6 Verify-then-fix test approach

1. Create a bug with `severity: high` and no `origin_feature` field (standalone).
2. Call `synthesiseProject` (or the `status` MCP tool with no ID).
3. Assert the returned `attention` slice contains at least one item with:
   - `type == "open_critical_bug"`
   - `entity_id == <the bug's ID>`
4. Create a second bug with `severity: high` and `origin_feature` set to a real feature ID.
5. Assert the project-level attention does **not** contain the second bug (it should only appear
   in `status(id: FEAT-...)` for that feature).
6. Transition the standalone bug to `closed`. Assert it no longer appears in project attention.

This covers the positive case, the double-surfacing guard, and the resolved-bug skip.

---

## 4. Alternatives Considered

### Surface standalone bugs at all scopes (project, plan, feature)

Rejected. Plan and feature scopes already have focused attention items. Injecting unrelated
standalone bugs into a plan or feature view would be noisy and confusing. Project scope is the
correct home for bugs with no organisational attachment.

### Add a new `AttentionItem.Type` value (e.g., `"standalone_bug"`)

Rejected for now. Reusing `"open_critical_bug"` keeps the consumer interface stable and avoids a
schema change. The `Message` field already conveys the standalone distinction
(`"Standalone high bug: …"` vs `"Open high bug: …"` in feature context). A dedicated type can be
introduced later if filtering by type becomes important.

### Surface all open standalone bugs regardless of severity

Rejected. Low/medium severity standalone bugs would flood the attention list for projects with
accumulated technical debt. The `high`/`critical` threshold matches the existing policy for
feature-linked bugs and keeps attention items actionable.

### Move standalone-bug logic into `generateProjectAttention`

Possible but would require adding an `openBugs []bugItem` parameter to that function's signature,
mirroring the approach used in `generateFeatureAttention`. That is a valid refactor, but it adds
indirection for a concern that is straightforward to handle inline in `synthesiseProject`, where
`entitySvc` is already available. The inline approach is simpler and easier to read.

---

## 5. Dependencies

| Dependency | Notes |
|------------|-------|
| `entitySvc.List("bug")` | Already called in `synthesiseFeature`; same call pattern |
| `AttentionItem` type | Already defined in `status_tool.go` |
| `id.FormatFullDisplay` | Already used throughout `synthesiseProject` |
| `maxAttentionItems` | Package-level constant in `status_tool.go`; use as cap guard |