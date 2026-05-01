# Dev Plan: Status command machine output formats

| Field   | Value                                                    |
|---------|----------------------------------------------------------|
| Date    | 2026-04-30                                               |
| Status  | approved (amended: 2026-04-30)                           |
| Author  | architect                                                |
| Feature | FEAT-01KQ2VHKJB5V8                                       |
| Spec    | work/spec/B36-F4-spec-status-machine-output.md           |
| Design  | work/_project/design-kbz-cli-and-binary-rename.md §5.4, §6.2–6.4, D-7, D-8 |

---

## Overview

This plan implements the machine-readable output formats (`--format plain` and `--format json`)
for the `kbz status` command as specified in
`work/spec/B36-F4-spec-status-machine-output.md` (FEAT-01KQ2VHKJB5V8/spec-b36-f4-spec-status-machine-output).

> **Amended 2026-04-30:** Added Tasks 5–8 to address blocking findings from the B36 batch
> conformance review (see `work/reviews/batch-review-b36-kbz-cli-and-status.md`). These fix
> four JSON schema and data-integrity violations: incorrect `dev-plan` key (Task 5), missing
> `display_id` field (Task 6), document slot ID populated with file path instead of record ID
> (Task 7), and bug `parent_feature_id` always null/missing (Task 8).

### Goals

- Deliver `--format plain` key:value output for all six scope types per FR-3 through FR-7
- Deliver `--format json` structured output with results-array wrapping (D-7) and distinct
  project shape (D-8) per FR-9–FR-10
- Guarantee schema stability with an automated CI contract test (NFR-1.5)
- Integrate into `kbz status` with full AC coverage and perf verification (NFR-2.1)
- Fix four JSON schema/data-integrity violations identified in batch conformance review

### Non-Goals

- `--format human` output (B36-F3)
- Argument resolution and routing (B36-F2 — dependency)
- MCP server tool changes; the CLI renderers consume existing service-layer data
- Multi-target queries (Q-1 — deferred)

---

## Task Breakdown

### Task 1: Plain format renderer

- **Description:** Implement a plain-format renderer that consumes the existing service-layer
  synthesis structs (`projectOverview`, `featureDetail`, `planDashboard`, `taskDetail`,
  `bugDetail`, and document data from `service.DocumentResult`) and emits `key: value` lines
  per the schemas in FR-3 through FR-7. Handle edge cases: `missing` for null values,
  `registered: true/false` on documents, single `attention:` line (highest severity), and
  `none` for empty attention. The renderer must be a pure function (no I/O, no side effects)
  taking a typed input per scope and returning `io.Writer`-writable output.
- **Deliverable:** `internal/cli/status/plain.go` — contains a `PlainRenderer` struct with
  per-scope methods: `RenderFeature(*mcp.FeatureDetail)`, `RenderPlan(*mcp.PlanDashboard)`,
  `RenderTask(*mcp.TaskDetail)`, `RenderBug(*mcp.BugDetail)`,
  `RenderDocument(*service.DocumentResult)`, `RenderProject(*mcp.ProjectOverview)`.
- **Depends on:** None (consumes existing types from `internal/mcp` and `internal/service`)
- **Effort:** Medium
- **Spec requirements:** FR-1.2, FR-1.3, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7

**Key design decisions:**

- The renderer accepts typed structs directly, not a `map[string]any` or `any` — each scope
  has its own method with the exact struct from the service layer. This avoids reflection and
  makes the contract explicit.
- `missing` is the sentinel for null/absent values. For documents specifically, `registered`
  is always present and is `true` or `false`.
- The `attention` field emits only the highest-severity item's message (error > warning), or
  `none` when empty.
- Keys are emitted in a fixed order matching the spec schemas; a struct field tag convention
  (e.g. `plain:"doc.design"`) drives ordering without hard-coded switch statements.

---

### Task 2: JSON format renderer

- **Description:** Implement a JSON-format renderer that consumes the same service-layer
  structs and emits RFC 8259 JSON. Entity/document queries are wrapped in `{"results": [...]}`
  per D-7. The project overview uses `{"scope": "project", ...}` per D-8. Each scope type
  has a distinct JSON schema matching FR-9 and FR-10 exactly. Edge cases: `null` for missing
  references (not absent), `[]` for empty attention (never `null`), `registered: false` with
  `id: null` for unregistered documents. Single-line compact JSON via `json.Marshal`, not
  `MarshalIndent`, matching the requirement that output be parseable by `jq` without flags.
- **Deliverable:** `internal/cli/status/json.go` — contains a `JSONRenderer` struct with
  per-scope methods: `RenderFeature`, `RenderPlan`, `RenderTask`, `RenderBug`,
  `RenderDocument`, `RenderProject`.
- **Depends on:** None (consumes existing types)
- **Effort:** Medium
- **Spec requirements:** FR-8, FR-9 (FR-9.1 through FR-9.6), FR-10

**Key design decisions:**

- Each `Render*` method constructs an intermediate struct that maps 1:1 to the spec's JSON
  schema, then calls `json.Marshal`. The intermediate structs use `json:` struct tags
  matching the spec's `snake_case` field names exactly.
- `feature.plan_id` is `null` (JSON `null`) when the feature has no parent plan, not absent.
- `documents` in the feature result always contains keys for all three document types
  (`design`, `spec`, `dev-plan`), mapped to `null` when not registered.
- Document `id` is `null` when `registered` is `false` (unregistered docs have no ID).
- Project overview is NOT wrapped in `results` — it uses `{"scope": "project", ...}` as
  the top-level shape.
- `health.errors` and `health.warnings` are computed from attention items at query time,
  not cached.

---

### Task 3: Schema stability contract test

- **Description:** Implement a contract test that enumerates every required key from the
  plain schemas (FR-3 through FR-7) and every required field from the JSON schemas (FR-9
  through FR-10), then asserts their presence in the output of each command variant. The
  test seeds a minimal project with one of each entity type, runs `kbz status` with `--format
  plain` and `--format json` for each scope type, and verifies that all expected keys/fields
  are present. This test must run in CI and fail on key/field removal.
- **Deliverable:** `internal/cli/status/contract_test.go`
- **Depends on:** Task 1, Task 2, Task 5, Task 6, Task 7, Task 8 (needs all renderers and schema fixes to exist)
- **Effort:** Small
- **Spec requirements:** NFR-1.1, NFR-1.2, NFR-1.5, AC-11

**Key design decisions:**

- The test maintains a `requiredPlainKeys` map keyed by scope type (e.g.
  `"feature" → []string{"scope", "id", "slug", "status", ...}`) and a `requiredJSONFields`
  map similarly keyed. These maps are the authoritative schema reference — they are checked
  against the spec, and the test asserts the renderers produce them.
- Seeding uses the test helpers already available in `cmd/kanbanzai/main_test.go`
  (`newFakeEntityService`, `setupTestGitRepo`).
- CI integration via `go test ./internal/cli/status/...` in the existing CI workflow.

---

### Task 4: Integration and verification

- **Description:** Wire the plain and JSON renderers into the `kbz status` command path,
  behind the `--format` flag (which B36-F2 routes). Implement the document-by-path lookup
  path (call `DocumentService.ListDocuments` with a path filter, produce a
  `DocumentResult` for the renderers). Add integration tests covering all 11 ACs
  (AC-1 through AC-11), plus performance verification for NFR-2.1 (<2s for 200-feature
  project). Wire into the existing `cmd/kanbanzai/main.go` switch by extending the
  `runStatus` function to accept and route `--format`.
- **Deliverable:** Modified `cmd/kanbanzai/workflow_cmd.go` (format flag routing),
  `internal/cli/status/status.go` (dispatcher that calls service-layer synthesis + renderer),
  and `cmd/kanbanzai/status_format_test.go` (integration tests).
- **Depends on:** Task 1, Task 2, Task 5, Task 6, Task 7, Task 8, B36-F2 (argument resolution)
- **Effort:** Large
- **Spec requirements:** FR-1.1, FR-11, NFR-2.1, NFR-3.1, NFR-3.2, AC-1 through AC-11

### Task 5: Fix dev-plan JSON Key

- **Description:** The `jsonDocs` struct in `internal/cli/status/json.go` uses Go field
  `DevPlan` with JSON tag `json:"dev_plan"` (underscore). The spec schema and AC-5
  require `"dev-plan"` (hyphen). Change the tag to `json:"dev-plan"`. Also update
  any test expectations or contract test maps (Task 3) that reference the old key.
  This is a one-line fix with a cascading schema impact — the contract test must
  be updated simultaneously to prevent CI breakage.
- **Deliverable:** Modified `internal/cli/status/json.go` (jsonDocs struct, ~L43).
- **Depends on:** Task 2 (needs the JSON renderer struct).
- **Effort:** Small.
- **Spec requirement:** FR-9.1, AC-5, NFR-1.1.

### Task 6: Add display_id to Feature JSON

- **Description:** The `jsonFeature` struct in `internal/cli/status/json.go` has fields
  `id`, `slug`, `status`, `plan_id` but no `display_id`. Spec A-2 and FR-9.1 require
  `display_id` to be present and non-null (e.g. `"F-042"`). Add `DisplayID string` field
  with tag `json:"display_id"` to `jsonFeature`; populate it from `in.DisplayID` in
  `JSONRenderer.RenderFeature`. FeatureInput in `render/types.go` already carries
  DisplayID — use it.
- **Deliverable:** Modified `internal/cli/status/json.go` (jsonFeature struct, ~L33;
  RenderFeature population).
- **Depends on:** Task 2 (needs the JSON renderer and FeatureInput type).
- **Effort:** Small.
- **Spec requirement:** FR-9.1, A-2.

### Task 7: Fix Document Slot ID (Add ID to DocInput)

- **Description:** In `RenderFeature` (json.go ~L135), `byType[d.Type]` sets
  `documentSlot.ID` to `d.Path` — the file path instead of the document record ID.
  Root cause: `DocInput` in `internal/cli/render/types.go` has no `ID` field, so
  the document record ID is lost before reaching the renderer. Fix:
  (1) Add `ID string` field to `DocInput` in `render/types.go`.
  (2) Populate it from the document record ID when building FeatureInput in
  `cmd/kbz/workflow_cmd.go` (`runStatusFeatureFormatted` and
  `runStatusEntityHuman` builders).
  (3) Use `d.ID` (not `d.Path`) for `documentSlot.ID` in both `json.go` and
  `plain.go` document rendering.
  The contract test (Task 3) must assert that doc `id` ≠ doc `path` for registered
  documents.
- **Deliverable:** Modified `internal/cli/render/types.go` (DocInput struct),
  `internal/cli/status/json.go` (RenderFeature population),
  `internal/cli/status/plain.go` (RenderDocument),
  `cmd/kbz/workflow_cmd.go` (feature/doc input builders).
- **Depends on:** Task 1, Task 2 (needs both renderers and DocInput type).
- **Effort:** Small.
- **Spec requirement:** FR-9.1.

### Task 8: Fix Bug parent_feature_id Forwarding

- **Description:** `JSONRenderer.RenderBug` hardcodes `ParentFeatureID: nil`;
  `PlainRenderer.RenderBug` hardcodes `{"parent_feature", "missing"}`. Neither
  accepts a parent feature parameter. Meanwhile, `runStatusBugFormatted` in
  `cmd/kbz/workflow_cmd.go` (~L253) extracts `parentFeature` from entity
  state but never forwards it to either renderer. Fix:
  (1) Add `parentFeature string` parameter to `PlainRenderer.RenderBug`.
  When non-empty, emit `parent_feature: {value}`; when empty, emit
  `parent_feature: missing`.
  (2) Add `parentFeature string` parameter to `JSONRenderer.RenderBug`.
  When non-empty, set `ParentFeatureID` to a `*string` pointing to the value;
  when empty, set `ParentFeatureID: nil` (JSON `null`).
  (3) Forward `parentFeature` from `runStatusBugFormatted` to the selected
  renderer.
- **Deliverable:** Modified `internal/cli/status/json.go` (RenderBug signature
  and body, ~L200), `internal/cli/status/plain.go` (RenderBug signature and
  body, ~L82), `cmd/kbz/workflow_cmd.go` (runStatusBugFormatted call site,
  ~L253).
- **Depends on:** Task 1, Task 2 (needs both renderers).
- **Effort:** Small.
- **Spec requirement:** FR-9.4, FR-5.

**Key design decisions:**

- The dispatcher function in `internal/cli/status/status.go` follows this flow:
  1. Call the appropriate `synthesise*` function from `internal/mcp` (same as the MCP tool).
  2. Dispatch to the correct renderer method based on `--format` and scope type.
  3. Write to stdout; errors go to stderr.
- Document-path lookup: `DocumentService.ListDocuments(DocumentFilters{})` is filtered
  client-side by `Path` match. Multiple matches is an error. No match produces an
  unregistered-document result (`registered: false`).
- Exit code `0` for all successful queries including those with `health.errors > 0` or
  `registered: false`.
- Performance: The existing service-layer synthesis is already fast — the renderers add
  negligible overhead. If the 200-feature threshold is breached, the fix is in the
  service layer, not the renderers.

---

## Dependency Graph

```
Task 1 (plain renderer) ──┐
                           ├── Task 3 (contract test)
Task 2 (JSON renderer) ───┘       │
       │                           │
       ├── Task 5 (dev-plan key) ──┤
       ├── Task 6 (display_id) ────┤
       ├── Task 7 (doc slot ID) ───┤
       └── Task 8 (bug parent) ────┘
                                   │
B36-F2 (argument resolution) ─────┤
                                   │
                           Task 4 (integration + verification)
```

**Parallel groups:** [Task 1, Task 2] for initial renderer implementation;
[Task 5, Task 6, Task 7, Task 8] for review fixes — all depend on Task 2 and
are independent of each other.

**Critical path:** Task 2 → Task 7 → Task 4 (Task 7 touches types.go which
Task 4's integration tests depend on; all four fix tasks must complete before
Task 4).

---

## Interface Contracts

### Task 1 → Task 4: PlainRenderer

```go
// PlainRenderer writes plain key:value status output to w.
type PlainRenderer struct{}

func (r *PlainRenderer) RenderFeature(w io.Writer, d *mcp.FeatureDetail) error
func (r *PlainRenderer) RenderPlan(w io.Writer, d *mcp.PlanDashboard) error
func (r *PlainRenderer) RenderTask(w io.Writer, d *mcp.TaskDetail) error
func (r *PlainRenderer) RenderBug(w io.Writer, d *mcp.BugDetail, parentFeature string) error
func (r *PlainRenderer) RenderDocument(w io.Writer, d *service.DocumentResult) error
func (r *PlainRenderer) RenderProject(w io.Writer, p *mcp.ProjectOverview) error
```

### Task 2 → Task 4: JSONRenderer

```go
type JSONRenderer struct{}

func (r *JSONRenderer) RenderFeature(w io.Writer, d *mcp.FeatureDetail) error
func (r *JSONRenderer) RenderPlan(w io.Writer, d *mcp.PlanDashboard) error
func (r *JSONRenderer) RenderTask(w io.Writer, d *mcp.TaskDetail) error
func (r *JSONRenderer) RenderBug(w io.Writer, d *mcp.BugDetail, parentFeature string) error
func (r *JSONRenderer) RenderDocument(w io.Writer, d *service.DocumentResult) error
func (r *JSONRenderer) RenderProject(w io.Writer, p *mcp.ProjectOverview) error
```

### Task 1, 2 → Task 4: Dispatcher (in `internal/cli/status/status.go`)

```go
// Dispatch synthesises a status result using the existing service-layer functions
// and renders it using the appropriate renderer for the requested format.
func Dispatch(w io.Writer, format string, target string, entitySvc *service.EntityService, docSvc *service.DocumentService) error
```

The dispatcher encapsulates the `synthesise*` → renderer pipeline. `format` is one of
`"plain"` or `"json"` (human is handled by B36-F3).

### Shared types (imported from existing packages)

| Type | Package | Used by |
|------|---------|---------|
| `projectOverview` | `internal/mcp` | Plain, JSON, dispatcher |
| `featureDetail` | `internal/mcp` | Plain, JSON, dispatcher |
| `planDashboard` | `internal/mcp` | Plain, JSON, dispatcher |
| `taskDetail` | `internal/mcp` | Plain, JSON, dispatcher |
| `bugDetail` | `internal/mcp` | Plain, JSON, dispatcher |
| `DocumentResult` | `internal/service` | Plain, JSON, dispatcher |
| `DocumentService` | `internal/service` | Dispatcher |
| `EntityService` | `internal/service` | Dispatcher |

---

## Traceability Matrix

| Requirement | Task(s) | Verification |
|-------------|---------|-------------|
| FR-1: Format flag | Task 4 | AC-1 through AC-11 |
| FR-2: Plain general rules | Task 1 | AC-1 through AC-4 |
| FR-3: Plain feature | Task 1 | AC-1, AC-2 |
| FR-4: Plain plan | Task 1 | Integration test (Task 4) |
| FR-5: Plain task/bug | Task 1, Task 8 | Integration test (Task 4) |
| FR-6: Plain document | Task 1 | AC-3 |
| FR-7: Plain project | Task 1 | AC-4 |
| FR-8: JSON general rules | Task 2 | AC-5 through AC-9 |
| FR-9: JSON entity/doc queries | Task 2, Task 5, Task 6, Task 7, Task 8 | AC-5, AC-6, AC-7, AC-9 |
| FR-10: JSON project overview | Task 2 | AC-8 |
| FR-11: Exit codes | Task 4 | AC-10 |
| NFR-1: Schema stability | Task 3, Task 5, Task 6, Task 7 | AC-11 |
| NFR-2: Performance | Task 4 | Performance benchmark |
| NFR-3: Parsability | Task 4 | jq/grep integration tests |

---

## Risk Assessment

### Risk: Service-layer struct mismatch with spec schemas

- **Probability:** Medium
- **Impact:** Medium — the existing MCP status structs have more fields than the CLI output
  schemas require (e.g. `taskInfo` has `Estimate`, `Name`; feature has `ReviewCycle`,
  `BlockedReason`). The renderers must subset these structs to match the spec exactly.
- **Mitigation:** Each renderer method constructs an intermediate output-specific type rather
  than serializing the service struct directly. The contract test (Task 3) catches any
  drift between the intermediate type and the spec schema.
- **Affected tasks:** Task 1, Task 2, Task 3

### Risk: B36-F2 delay blocks integration

- **Probability:** Low
- **Impact:** High — Task 4 depends on B36-F2 for argument resolution and routing. Without
  it, the `--format` flag cannot be parsed and dispatched.
- **Mitigation:** The renderers (Task 1, Task 2) are independent of B36-F2 and can be built
  and tested in isolation. The contract test (Task 3) can also run against the renderers
  directly without the CLI routing layer. Only Task 4 is blocked.
- **Affected tasks:** Task 4

### Risk: DocInput ID field addition breaks human renderer

- **Probability:** Low
- **Impact:** Medium — DocInput is shared across F3 (human) and F4 (plain/JSON). Adding
  a new ID field to the struct is backward-compatible (zero value is empty string), but
  the human renderer's doc block must not regress.
- **Mitigation:** The human renderer (F3) does not iterate struct fields — it accesses
  known fields by name. Adding a new field has no effect on existing rendering. Run the
  full render test suite after the change.
- **Affected tasks:** Task 7

### Risk: Document path lookup returns multiple records

- **Probability:** Low
- **Impact:** Low — the current document store enforces path uniqueness per ID convention.
  If it occurs, it's a data integrity issue, not a renderer bug.
- **Mitigation:** The dispatcher returns an error if multiple records match the path. The
  integration test (Task 4) includes a multi-match error case.
- **Affected tasks:** Task 4

---

## Verification

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|--------------------|----------------|
| AC-1: Plain format — feature with full documents | Integration test | Task 4 |
| AC-2: Plain format — feature with no plan | Integration test | Task 4 |
| AC-3: Plain format — unregistered document | Integration test | Task 4 |
| AC-4: Plain format — project overview health gate | Integration test | Task 4 |
| AC-5: JSON format — feature results array | Integration test | Task 4, Task 5 |
| AC-6: JSON format — feature with null plan_id | Integration test | Task 4 |
| AC-7: JSON format — unregistered document | Integration test | Task 4 |
| AC-8: JSON format — project overview shape | Integration test | Task 4 |
| AC-9: JSON format — empty attention | Integration test | Task 4 |
| AC-10: Exit codes | Integration test | Task 4 |
| AC-11: Schema contract test | Contract test (automated) | Task 3 |
| NFR-1.5: Contract test runs in CI | CI workflow check | Task 3 |
| NFR-2.1: <2s for 200-feature project | Performance benchmark | Task 4 |
| NFR-3.1: JSON parseable by jq/Python | Integration test (pipe through jq) | Task 4 |
| NFR-3.2: Plain grep-able | Integration test (pipe through grep) | Task 4 |
