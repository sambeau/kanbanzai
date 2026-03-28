# MCP Discoverability and Reliability Specification

| Document | MCP Discoverability and Reliability Specification |
|----------|---------------------------------------------------|
| Status   | Draft                                             |
| Created  | 2026-03-28                                        |
| Plan     | P9-mcp-discoverability                            |
| Design   | work/design/mcp-discoverability-and-reliability.md |
| Related  | work/reports/kanbanzai-2.0-migration-audit.md     |

---

## 1. Purpose

This specification defines the acceptance criteria for the six features in the MCP Discoverability and Reliability design. It is the implementation contract: each feature section lists what must be true for the feature to be considered done, with enough detail that an implementation task can be written from the corresponding section alone.

The design document (`work/design/mcp-discoverability-and-reliability.md`) contains the rationale, classification model, and full title/description text. This specification references the design rather than repeating it. Implementors should read the design first.

---

## 2. Scope

| Feature | Summary |
|---------|---------|
| **A** | Tool annotations (`readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint`) on all 22 tools |
| **B** | Human-readable `title` annotation on all 22 tools |
| **C** | Revised tool descriptions that guide agent behaviour |
| **D** | Response nudges in `finish` when retrospective signals or knowledge entries are absent |
| **E** | `doc(action: refresh)` — fix stale content hash without re-registering |
| **F** | `doc(action: chain)` — expose existing `SupersessionChain()` service method via MCP |

All features target the `internal/mcp/` package unless otherwise stated.

---

## 3. Feature A: Tool Annotations

### 3.1 Background

MCP tool annotations are the spec-defined mechanism for letting clients make auto-approval decisions. All four boolean annotation fields (`ReadOnlyHint`, `DestructiveHint`, `IdempotentHint`, `OpenWorldHint`) must be explicitly set on every tool. Unset (nil) fields must not exist in a shipped build.

The full classification table is in design §4.3. The three tiers are:

- **Tier 1 — Read-only:** `readOnly:true`, `destructive:false`, `idempotent:true`, `openWorld:false`
- **Tier 2 — Auto-approvable writes:** `readOnly:false`, `destructive:false`, `openWorld:false`, `idempotent:false` (unless noted)
- **Tier 3 — Requires confirmation:** `destructive:true` and/or `openWorld:true`

### 3.2 Tool Classification

Implementors MUST apply annotations exactly as specified in this table. Deviation requires a design amendment.

| Tool | readOnly | destructive | idempotent | openWorld | Tier |
|------|----------|-------------|------------|-----------|------|
| `status` | `true` | `false` | `true` | `false` | 1 |
| `health` | `true` | `false` | `true` | `false` | 1 |
| `handoff` | `true` | `false` | `true` | `false` | 1 |
| `conflict` | `true` | `false` | `true` | `false` | 1 |
| `profile` | `true` | `false` | `true` | `false` | 1 |
| `server_info` | `true` | `false` | `true` | `false` | 1 |
| `entity` | `false` | `false` | `false` | `false` | 2 |
| `doc` | `false` | `false` | `false` | `false` | 2 |
| `doc_intel` | `false` | `false` | `false` | `false` | 2 |
| `next` | `false` | `false` | `false` | `false` | 2 |
| `finish` | `false` | `false` | `false` | `false` | 2 |
| `knowledge` | `false` | `false` | `false` | `false` | 2 |
| `estimate` | `false` | `false` | `false` | `false` | 2 |
| `decompose` | `false` | `false` | `false` | `false` | 2 |
| `checkpoint` | `false` | `false` | `false` | `false` | 2 |
| `incident` | `false` | `false` | `false` | `false` | 2 |
| `retro` | `false` | `false` | `false` | `false` | 2 |
| `branch` | `false` | `false` | `true` | `false` | 2 |
| `cleanup` | `false` | `true` | `false` | `false` | 3 |
| `merge` | `false` | `true` | `false` | `true` | 3 |
| `pr` | `false` | `false` | `false` | `true` | 3 |
| `worktree` | `false` | `true` | `false` | `false` | 3 |

### 3.3 Implementation Pattern

Use the `mcp.WithToolAnnotation(mcp.ToolAnnotation{...})` option on `mcp.NewTool(...)`. All four pointer fields must be set to non-nil values. The helper functions may be used instead if preferred:

```go
mcp.WithReadOnlyHintAnnotation(true)
mcp.WithDestructiveHintAnnotation(false)
mcp.WithIdempotentHintAnnotation(true)
mcp.WithOpenWorldHintAnnotation(false)
```

Both styles are acceptable. The canary test (§3.4) validates the result, not the implementation style.

### 3.4 Safe Defaults

The `mcp.NewTool` call for any tool not yet annotated must default to the most restrictive values if annotations are absent. This is a belt-and-suspenders rule for development; the canary test is the primary enforcement mechanism. The defaults are:

- `readOnlyHint: false`
- `destructiveHint: true`
- `idempotentHint: false`
- `openWorldHint: true`

### 3.5 Canary Test

A new test file `internal/mcp/annotations_test.go` must be created. The test must:

1. Instantiate every tool group in the server (all groups registered by `server.go`).
2. Collect all `server.ServerTool` values.
3. For each tool, assert that `tool.Tool.Annotations.ReadOnlyHint != nil`.
4. For each tool, assert that `tool.Tool.Annotations.DestructiveHint != nil`.
5. For each tool, assert that `tool.Tool.Annotations.IdempotentHint != nil`.
6. For each tool, assert that `tool.Tool.Annotations.OpenWorldHint != nil`.
7. Optionally: assert that every Tier 1 tool (by name) has `*ReadOnlyHint == true`.
8. Optionally: assert that every Tier 3 tool (by name) has `*DestructiveHint == true || *OpenWorldHint == true`.
9. The test must fail if a new tool is added without annotations (the nil check covers this automatically).

The test must be structured as a table-driven test or a loop over all registered tools so that adding a new tool automatically extends the coverage. It must not hardcode which tools exist — it must discover them from the server registration.

### 3.6 Acceptance Criteria

- [ ] All 22 tools have all four annotation fields set to non-nil `*bool` values.
- [ ] The annotation values match the classification table in §3.2 exactly.
- [ ] `annotations_test.go` exists and passes with `go test ./internal/mcp/...`.
- [ ] `go test -race ./...` passes.
- [ ] The test fails if a new tool is added without annotations (verified by temporarily adding an unannotated tool).

---

## 4. Feature B: Tool Titles

### 4.1 Background

The MCP spec `title` annotation field provides a human-readable display name for tools, distinct from the programmatic `name`. Titles appear in client UIs (tool lists, approval dialogs) and help both humans and agents identify tools at a glance.

### 4.2 Title Values

Each tool must have a `title` set via `mcp.WithTitleAnnotation(...)`. The exact title strings are specified below and must be followed verbatim. If a title conflicts with improved UX intuition, document the rationale and amend the design rather than silently deviating.

| Tool | Title |
|------|-------|
| `status` | `Workflow Status Dashboard` |
| `entity` | `Entity Manager` |
| `doc` | `Document Records` |
| `doc_intel` | `Document Intelligence` |
| `next` | `Work Queue & Dispatch` |
| `handoff` | `Sub-Agent Prompt Generator` |
| `finish` | `Task Completion` |
| `knowledge` | `Knowledge Base` |
| `estimate` | `Story Point Estimates` |
| `decompose` | `Feature Decomposition` |
| `conflict` | `Conflict Risk Analysis` |
| `profile` | `Context Role Profiles` |
| `worktree` | `Git Worktree Manager` |
| `merge` | `Merge Gate & Execution` |
| `pr` | `Pull Request Manager` |
| `branch` | `Branch Health Monitor` |
| `cleanup` | `Worktree Cleanup` |
| `incident` | `Incident Tracker` |
| `checkpoint` | `Human Decision Checkpoints` |
| `health` | `System Health Check` |
| `retro` | `Retrospective Synthesis` |
| `server_info` | `Server Build Information` |

### 4.3 Implementation Note

`mcp.WithTitleAnnotation(title)` sets only the `Title` field of `ToolAnnotation`. It can be combined with the four boolean annotation options from Feature A, or the full `mcp.WithToolAnnotation(mcp.ToolAnnotation{Title: ..., ReadOnlyHint: ..., ...})` form may be used. Both are acceptable.

### 4.4 Acceptance Criteria

- [ ] All 22 tools have a `Title` annotation set to the exact string from the table in §4.2.
- [ ] No tool has an empty or missing title.
- [ ] `go test -race ./...` passes.

---

## 5. Feature C: Improved Tool Descriptions

### 5.1 Background

Tool descriptions are the primary input an agent uses when selecting which tool to call. Current descriptions say *what* a tool does but not *when to use it* or *what not to do instead*. Revised descriptions add behavioural guidance: when to prefer the tool, what bad-habit paths it replaces, and a summary of available actions.

### 5.2 Revised Descriptions

The following seven tools require description updates. All other tools have descriptions already adequate for their purpose; do not modify them unless there is a specific reason.

The exact replacement strings are specified below. Implementors must apply them verbatim. Minor whitespace normalisation (e.g. line wrapping in Go string literals) is acceptable; the rendered description must match.

#### `status`

```
Synthesis dashboard — the primary way to query project, plan, feature, or task state. Returns lifecycle status, attention items, progress metrics, and derived state (what's blocked, what's ready) that raw YAML files do not contain. Use this instead of reading .kbz/state/ files directly. Call with no id for project overview, plan ID for plan dashboard, FEAT-... for feature detail, TASK-... or BUG-... for task detail.
```

#### `entity`

```
Create, read, update, and transition entities (plans, features, tasks, bugs, epics, decisions). Use action: get or action: list to query entities — these return structured data with lifecycle state and cross-references. Do not read .kbz/state/ YAML files directly. Actions: create, get, list, update, transition. Entity type is inferred from ID prefix for get/update/transition. Supports batch creation via the entities array.
```

#### `next`

```
Inspect the work queue or claim a task with full context assembly. Call without id to see all ready tasks (sorted by priority, with optional conflict checking). Call with a task/feature/plan ID to claim the next ready task — returns spec sections, knowledge entries, file paths, and role conventions assembled for the task. This is the primary way to pick up work; prefer it over manually querying entities and assembling context yourself.
```

#### `finish`

```
Record task completion with optional knowledge contribution and retrospective signals. Transitions the task to done (default) or needs-review. Include the retrospective parameter to record observations about workflow friction, tool gaps, spec ambiguity, or things that worked well — these feed the retro tool for future synthesis. Accepts tasks in ready or active status. Supports batch completion via the tasks array.
```

#### `doc`

```
Manage document records: register, approve, query, supersede, refresh, chain, validate, and import. Use action: get or action: list to query document status — do not read .kbz/state/documents/ files directly. Actions: register, approve, get, content, list, gaps, validate, supersede, refresh, chain, import. approve and supersede report entity lifecycle cascade side effects.
```

Note: the `doc` description must include `refresh` and `chain` in the actions list. Update it once after both Feature E and Feature F are implemented, or update it incrementally alongside each feature.

#### `knowledge`

```
Query and manage the project knowledge base. Use action: list to find knowledge entries by topic, tag, or status — this surfaces information that may not be in your context window. Routine contribution happens via finish; use this tool for direct management: confirming entries, resolving conflicts, checking staleness, pruning. Actions: list, get, contribute, confirm, flag, retire, update, promote, compact, prune, resolve, staleness.
```

#### `retro`

```
Synthesise retrospective signals into themed clusters. Before writing any retrospective or review document, call action: synthesise first — it surfaces signals from across the project that may not be in your session context. Actions: synthesise (read signals, cluster, rank), report (generate and register a markdown report document).
```

### 5.3 Acceptance Criteria

- [ ] The seven listed tools have their descriptions updated to match §5.2 verbatim (modulo Go string literal whitespace).
- [ ] No other tool descriptions are modified without explicit justification.
- [ ] `go test -race ./...` passes.

---

## 6. Feature D: Response Nudges

### 6.1 Background

The P7 retrospective showed that all 12 `finish` calls omitted the `retrospective` parameter and none contributed `knowledge` entries. A nudge in the `finish` response — appearing at the moment the agent is already reading the result — is more reliable than any rule stated in advance.

Nudges are informational only. They are a `nudge` string field added to the response JSON. They do not set `isError: true`, do not block the operation, and do not change the schema of any existing response fields.

### 6.2 Nudge 1: Feature Completion With No Retrospective Signals

**Trigger condition:** The `finish` call successfully completes the last active or ready task in a feature (i.e., after this completion, the feature has no tasks remaining in `queued`, `ready`, or `active` status), AND no tasks in that feature have any recorded retrospective signals.

**Determining "no retrospective signals":** Query all tasks belonging to the parent feature. A task has retrospective signals if `dispatchSvc.CompleteTask` was called with non-empty `RetroSignals` on a previous call, or if the retrospective section of the task completion record is non-empty in `.kbz/state/`. The simplest implementation is to query task completion metadata and check for the presence of any recorded retro signal entries across the feature's tasks.

**Nudge text:**
```
No retrospective signals were recorded for any task in this feature. If you observed workflow friction, tool gaps, spec ambiguity, or things that worked well, call finish again on any completed task with the retrospective parameter, or use knowledge(action: contribute) with tags: ["retrospective"].
```

**Response field:** `"nudge"` at the top level of the single-item response (not inside `task` or `knowledge`).

### 6.3 Nudge 2: Task Completion With No Knowledge or Retrospective

**Trigger condition:** The `finish` call provides a non-empty `summary`, but both `knowledge` and `retrospective` are absent (nil or empty arrays).

**Nudge text:**
```
Consider including knowledge entries (reusable facts learned during this task) or retrospective signals (process observations) in your finish call. These improve context assembly for future tasks.
```

**Response field:** `"nudge"` at the top level of the single-item response.

**Suppression:** Nudge 2 must NOT fire when:
- `summary` is empty or missing (no substantive completion to reflect on).
- Either `knowledge` or `retrospective` was provided (even if empty after validation — the agent made the attempt).
- The `finish` call is a batch call (batch completions of trivial tasks should not generate noise).

### 6.4 Nudge Priority

When both nudge conditions are met simultaneously (feature completion with no retro signals, AND no knowledge entries in this call), only Nudge 1 must fire. Nudge 1 takes priority. Both nudges must not be present in the same response.

### 6.5 Batch Mode

Nudges must not be included in batch `finish` responses. The batch response schema uses a different wrapper structure; do not add a `nudge` field to individual batch item results.

### 6.6 Acceptance Criteria

- [ ] When `finish` completes the last task in a feature and no tasks in the feature have retro signals, the response includes `"nudge": "<Nudge 1 text>"`.
- [ ] When `finish` completes a task with a non-empty summary and no `knowledge` or `retrospective` provided (not in a batch), the response includes `"nudge": "<Nudge 2 text>"`.
- [ ] When Nudge 1 condition is met, Nudge 1 fires regardless of whether Nudge 2 would also fire.
- [ ] Nudges are absent when `knowledge` or `retrospective` is provided in the call.
- [ ] Nudges are absent in batch mode responses.
- [ ] Nudge text matches §6.2 and §6.3 verbatim (modulo Go string literal whitespace).
- [ ] The `nudge` field is absent (not null, not empty string — absent) when no nudge condition is met.
- [ ] Existing `finish` tests still pass; new tests cover each trigger condition and each suppression condition.
- [ ] `go test -race ./...` passes.

---

## 7. Feature E: `doc(action: refresh)`

### 7.1 Background

When a registered document file is edited after registration, its stored content hash becomes stale. The only current workaround is to delete the document record and re-register, which loses document intelligence data (classifications, entity references, graph edges). The `refresh` action fixes the hash in place, optionally transitioning the record back to `draft` if it was `approved`.

This was previously implemented on the unmerged hardening branch (commit `8ae3b7e`). The reference implementation exists but must not be merged directly; use it as a reference only.

### 7.2 Service Layer

A new method `RefreshContentHash` must be added to `DocumentService` in `internal/service/documents.go`.

**Signature:**
```go
type RefreshInput struct {
    ID   string // document record ID (preferred)
    Path string // file path (alternative lookup if ID is empty)
}

type RefreshResult struct {
    ID               string
    Path             string
    Changed          bool
    OldHash          string
    NewHash          string
    Status           string // final status after refresh
    StatusTransition string // e.g., "approved → draft", or "" if unchanged
}

func (s *DocumentService) RefreshContentHash(input RefreshInput) (RefreshResult, error)
```

**Behaviour:**

1. Resolve the document record. If `ID` is non-empty, load by ID. If `ID` is empty and `Path` is non-empty, look up by path. If neither is provided, return an error: `"id or path is required"`.
2. If the record is not found, return an error: `"document not found: <id-or-path>"`.
3. Resolve the full file path: `filepath.Join(repoRoot, doc.Path)`.
4. Compute the current content hash using `storage.ComputeContentHash(fullPath)`.
5. If the file does not exist, return an error: `"document file not found at path <fullPath>; verify the file exists before calling refresh"`.
6. Compare the current hash with the stored `ContentHash`.
7. **If unchanged:** return `RefreshResult{Changed: false, OldHash: storedHash, NewHash: storedHash, Status: doc.Status}`. Do not write anything.
8. **If changed:**
   a. Update `doc.ContentHash` to the new hash.
   b. Update `doc.Updated` to `now`.
   c. If `doc.Status == "approved"`, set `doc.Status = "draft"` and set `statusTransition = "approved → draft"`.
   d. Write the updated record to storage using the existing `fileHash` for optimistic locking.
   e. Return `RefreshResult{Changed: true, OldHash: storedHash, NewHash: currentHash, Status: finalStatus, StatusTransition: statusTransition}`.

### 7.3 MCP Layer

The `doc` tool handler in `internal/mcp/doc_tool.go` must add a case for `action == "refresh"`.

**Input:** `id` (string, preferred) or `path` (string, alternative). Both are already declared as parameters on the `doc` tool.

**Response shape:**
```json
{
  "id": "DOC-01JX...",
  "path": "work/spec/foo.md",
  "changed": true,
  "old_hash": "abc123...",
  "new_hash": "def456...",
  "status": "draft",
  "status_transition": "approved → draft"
}
```

When `changed: false`:
```json
{
  "id": "DOC-01JX...",
  "path": "work/spec/foo.md",
  "changed": false,
  "old_hash": "abc123...",
  "new_hash": "abc123...",
  "status": "approved",
  "status_transition": ""
}
```

**Error responses:** Returned as MCP errors (non-nil `error` return). Include the document ID or path in the error message. Provide actionable guidance for the file-not-found case.

### 7.4 Description Update

The `doc` tool description must include `refresh` in the actions list (see Feature C §5.2 — the `doc` description already includes it; ensure this is applied together with the description update).

### 7.5 Acceptance Criteria

- [ ] `DocumentService.RefreshContentHash` exists in `internal/service/documents.go`.
- [ ] `doc(action: refresh, id: "...")` updates the stored content hash when the file has changed.
- [ ] When the document was `approved` and the file has changed, the refresh transitions the status to `draft` and reports `status_transition: "approved → draft"` in the response.
- [ ] When the file is unchanged, the response returns `changed: false` and no record is written.
- [ ] When the file does not exist, the tool returns an actionable error message identifying the expected path.
- [ ] When neither `id` nor `path` is provided, the tool returns an error: `"id or path is required"`.
- [ ] `doc(action: refresh, path: "...")` works as an alternative to ID-based lookup.
- [ ] Unit tests for `RefreshContentHash` cover: hash changed (draft doc), hash changed (approved doc → transitions to draft), hash unchanged, file not found, record not found, empty input.
- [ ] MCP-layer test covers the tool action routing for `refresh`.
- [ ] `go test -race ./...` passes.

---

## 8. Feature F: `doc(action: chain)`

### 8.1 Background

The `DocumentService.SupersessionChain()` method is implemented and tested in `internal/service/documents.go` and `internal/service/supersession_test.go`. It was a 1.0 tool (`doc_supersession_chain`) removed in the 2.0 migration without being wired to the consolidated `doc` tool. This feature wires it in.

### 8.2 Service Layer

No service-layer changes are required. `SupersessionChain(docID string) ([]DocumentResult, error)` is already implemented and tested.

### 8.3 MCP Layer

The `doc` tool handler in `internal/mcp/doc_tool.go` must add a case for `action == "chain"`.

**Input:** `id` (string, required). The `id` parameter is already declared on the `doc` tool.

**Behaviour:** Call `docSvc.SupersessionChain(id)`. Return the resulting chain as a JSON array ordered from oldest to newest.

**Response shape:**
```json
{
  "chain": [
    {
      "id": "DOC-01JX...",
      "path": "work/spec/foo-v1.md",
      "type": "specification",
      "title": "Foo Spec v1",
      "status": "superseded",
      "superseded_by": "DOC-02JX..."
    },
    {
      "id": "DOC-02JX...",
      "path": "work/spec/foo-v2.md",
      "type": "specification",
      "title": "Foo Spec v2",
      "status": "approved",
      "superseded_by": ""
    }
  ],
  "length": 2
}
```

Each chain item includes: `id`, `path`, `type`, `title`, `status`, `superseded_by` (empty string if none).

**Error responses:** If `id` is empty, return `"id is required for action: chain"`. If the document is not found, the service method returns an error; surface it as an MCP error.

### 8.4 Description Update

The `doc` tool description must include `chain` in the actions list (see Feature C §5.2 — the `doc` description already includes it; ensure this is applied together with the description update).

### 8.5 Acceptance Criteria

- [ ] `doc(action: chain, id: "...")` returns the supersession chain for a document.
- [ ] The response includes `chain` (array) and `length` (integer).
- [ ] Each chain item includes `id`, `path`, `type`, `title`, `status`, `superseded_by`.
- [ ] The chain is ordered from oldest to newest version.
- [ ] When `id` is empty, the tool returns an error: `"id is required for action: chain"`.
- [ ] When the document does not exist, the tool returns an appropriate error.
- [ ] A test covers the tool action routing for `chain` (may use the existing `supersession_test.go` fixtures for setup).
- [ ] No changes are made to the service layer.
- [ ] `go test -race ./...` passes.

---

## 9. Cross-Cutting Requirements

### 9.1 All Features

- All code changes must be in the `internal/mcp/` package (plus `internal/service/documents.go` for Feature E).
- No changes to the CLI layer, storage layer, or model layer are required for any feature.
- `go test -race ./...` must pass after each feature is implemented.
- No existing test may be deleted or weakened to make a new test pass.

### 9.2 Tool Count

At time of writing, 22 tools are registered. Features A and B must cover all of them. The canary test (Feature A §3.5) will catch any discrepancy at test time. The exact tool count is not hardcoded in the spec; the canary drives compliance.

### 9.3 Schema Compatibility

Features A, B, and C make no changes to request or response schemas — they modify only tool registration metadata. Features D, E, and F introduce new response fields and a new action value. No existing fields are removed or renamed. Backwards compatibility is preserved for all existing callers.

### 9.4 Reference Implementation

The unmerged hardening branch (`feature/FEAT-01KMKRQWF0FCH-hardening`) contains reference implementations for some of these features. The branch must NOT be merged; it targets the 1.0 tool set. Implementors may read the branch for patterns and ideas. Any code taken from the branch must be adapted to the 2.0 tool structure.

---

## 10. Out of Scope

The following items were explicitly deferred in the design and are out of scope for this plan:

- MCP Prompts (slash commands for portable instruction delivery)
- MCP Resources (knowledge base entries as `kbz://` URIs)
- Improved service-layer error messages (separate follow-up)
- `suggest_links` automatic behaviour
- AGENTS.md updates (instructions belong in the server, not project files)

---

## 11. Open Questions Resolved

The following open questions from the design have been resolved for the purposes of this specification:

**§11.1.1 — `branch` tier:** Classified as Tier 2 (conservative). The `branch` tool may gain write actions in the future; Tier 2 avoids a reclassification ceremony at that point. See design §4.2 Tier 2 for the `branch` row.

**§11.1.2 — `retro report` composing `synthesise`:** Deferred to implementation. The implementor may choose to have `retro(action: report)` call `synthesise` internally, but this is not required by this specification. If implemented, the composed call must not change the observable response schema or require additional parameters from the caller.

**§11.2.3 — `handoff` retro reminder:** Not included in this specification. Adding boilerplate to the `handoff` prompt output is low-confidence (agents may filter it as noise) and adds token cost. Revisit after Feature D nudges have been deployed for a full implementation cycle.

**§11.2.4 — Description phrasing effectiveness:** The descriptions in §5.2 are the spec-mandated text. They have been reviewed against the P7 failure modes. Minor wording refinements are acceptable after deployment based on observed agent behaviour; such refinements do not require a design amendment.
```

Now let me register this spec, approve it, create the feature, and decompose: