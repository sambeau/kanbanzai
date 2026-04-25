# Implementation Plan: Plan ID Prefix Resolution

| Field   | Value                                                                    |
|---------|--------------------------------------------------------------------------|
| Date    | 2026-04-25                                                               |
| Status  | Draft                                                                    |
| Author  | Architect                                                                |
| Feature | FEAT-01KQ2E0RB4P8A (plan-id-prefix-resolution)                          |
| Spec    | `work/spec/feat-01kq2e0rb4p8a-plan-id-prefix-resolution.md`             |
| Plan    | P34-agent-workflow-ergonomics                                            |

---

## 1. Scope

This plan implements the requirements defined in
`work/spec/feat-01kq2e0rb4p8a-plan-id-prefix-resolution.md`
(FEAT-01KQ2E0RB4P8A/specification-feat-01kq2e0rb4p8a-plan-id-prefix-resolution).
It covers three tasks: adding the `ParseShortPlanRef` model predicate (Task 1),
adding the `ResolvePlanByNumber` service method (Task 2), and integrating
short-ref pre-resolution into the `entity` and `status` MCP tool handlers (Task 3).

**Out of scope:** Changes to `IsPlanID`, `ParsePlanID`, or any existing model
function; short-ref handling for FEAT/TASK/BUG IDs; any tool other than `entity`
and `status`.

**Execution order:**

```
[Task 1: ParseShortPlanRef] ─┐
                              ├──→ [Task 3: Tool integration]
[Task 2: ResolvePlanByNumber]─┘
```

Tasks 1 and 2 are independent and can run in parallel. Task 3 depends on both.

---

## 2. Interface Contracts

### `model.ParseShortPlanRef`

```go
// ParseShortPlanRef reports whether s is a short plan reference of the form
// <single-non-digit-rune><one-or-more-digits> with nothing else.
// On success it returns the prefix rune as a string and the digit string.
func ParseShortPlanRef(s string) (prefix, number string, ok bool)
```

Rules:
- `"P30"` → `("P", "30", true)`
- `"ñ5"` → `("ñ", "5", true)` — non-ASCII prefix is valid
- `"P30-slug"` → `("", "", false)` — hyphen present
- `"30"` → `("", "", false)` — no leading non-digit rune
- `""` → `("", "", false)`
- `"P"` → `("", "", false)` — no digits

No I/O. Pure function. Lives in `internal/model/entities.go` alongside
`IsPlanID` and `ParsePlanID`.

### `EntityService.ResolvePlanByNumber`

```go
func (s *EntityService) ResolvePlanByNumber(cfg config.Config, prefix, number string) (id, slug string, err error)
```

Contract:
- Calls `cfg.IsActivePrefix(prefix)` first; if false, returns error:
  `"unknown plan prefix %q — valid prefixes are: [P, M, ...]"` (list from
  `cfg.Prefixes` filtered to non-retired entries).
- Scans plans via `s.ListPlanIDs()` (cache-backed after P29).
- Returns `(fullCanonicalID, slug, nil)` for the matched plan.
- Returns a non-nil error when no plan matches prefix+number.
- Method lives in `internal/service/entities.go` or a new
  `internal/service/plans_resolve.go`.

---

## 3. Task Breakdown

| # | Task | Files | Spec refs |
|---|------|-------|-----------|
| 1 | `ParseShortPlanRef` model predicate | `internal/model/entities.go`, `internal/model/entities_test.go` | FR-001–FR-004, AC-006–AC-010 |
| 2 | `ResolvePlanByNumber` service method | `internal/service/entities.go`, `internal/service/entities_test.go` | FR-005–FR-008, AC-011 |
| 3 | Tool integration (`entity` + `status`) | `internal/mcp/entity_tool.go`, `internal/mcp/status_tool.go`, `internal/mcp/entity_tool_test.go`, `internal/mcp/status_tool_test.go` | FR-009–FR-012, AC-001–AC-005 |

---

## 4. Task Details

### Task 1: `ParseShortPlanRef` model predicate

**Objective:** Expose a new exported function in `internal/model/entities.go`
that detects and extracts the prefix and number from a short plan reference.

**Specification references:** FR-001, FR-002, FR-003, FR-004.

**Input context:**
- Read `internal/model/entities.go` — specifically the `IsPlanID` and
  `ParsePlanID` functions and how they handle Unicode prefix characters
  (`unicode.IsDigit`). `ParseShortPlanRef` must use the same Unicode semantics.
- Read `internal/model/entities_test.go` — follow existing test patterns
  (table-driven, `t.Parallel()`).

**Output artifacts:**
- `internal/model/entities.go` — add `ParseShortPlanRef` near `ParsePlanID`.
- `internal/model/entities_test.go` — add `TestParseShortPlanRef` with at
  minimum: `"P30"` (ok), `"ñ5"` (ok, non-ASCII), `"30"` (no prefix, not ok),
  `"P30-slug"` (hyphen, not ok), `""` (empty, not ok), `"P"` (no digits, not ok),
  `"P30X"` (trailing non-digit, not ok).

**Constraints:**
- No I/O. The function must be a pure lexical check.
- Accept any single non-digit Unicode rune as prefix (do not restrict to ASCII).
- Do not modify `IsPlanID`, `ParsePlanID`, or any other existing function.

---

### Task 2: `ResolvePlanByNumber` service method

**Objective:** Add a method on `EntityService` that resolves a `(prefix, number)`
pair — as returned by `ParseShortPlanRef` — to the full canonical plan ID.

**Specification references:** FR-005, FR-006, FR-007, FR-008.

**Input context:**
- Read `internal/service/entities.go` — find `ListPlanIDs` (or the cache-backed
  listing equivalent) and `ResolvePrefix` for structural patterns to follow.
- Read `internal/config/config.go` — `IsActivePrefix`, `Prefixes []PrefixEntry`,
  and the `Retired` field.
- Read `internal/model/entities.go` — `ParsePlanID` decomposes a full plan ID
  into `(prefix, number, slug)`. Use it inside the scan loop.
- Read `internal/service/plans_test.go` — follow test patterns.

**Output artifacts:**
- `internal/service/entities.go` (or `internal/service/plans_resolve.go` if
  the method is cleanest in a new file) — add `ResolvePlanByNumber`.
- `internal/service/entities_test.go` — add `TestEntityService_ResolvePlanByNumber`
  covering: unknown prefix returns descriptive error (AC-011 style), valid prefix
  + matching plan returns full ID, valid prefix + no match returns non-nil error.

**Constraints:**
- Use `ListPlanIDs()` (cache-backed list); do not introduce a new file-scan loop.
- The error for an unknown prefix MUST name the prefix and list valid active
  prefixes (FR-006).
- `cfg config.Config` is passed as a parameter — do not load config internally.

---

### Task 3: Tool integration (`entity` and `status`)

**Objective:** Apply short-ref pre-resolution in `entityGetAction`,
`entityUpdateAction`, and `entityTransitionAction` in `entity_tool.go`, and in
the plan ID dispatch path of `status_tool.go`.

**Specification references:** FR-009, FR-010, FR-011, FR-012.

**Input context:**
- Read `internal/mcp/entity_tool.go` — focus on `entityGetAction` (line ~337),
  `entityUpdateAction` (line ~487), `entityTransitionAction` (line ~580), and
  `entityInferType` (line ~1290). The pre-resolution step must occur after
  `entityID := id.NormalizeID(entityArgStr(args, "id"))` and before
  `entityInferType` is called.
- Read `internal/mcp/status_tool.go` — focus on `inferIDType` (line ~150) and
  the `idTypePlan` dispatch path. Short refs must resolve before `inferIDType`.
- Config loading: use `config.Load()` (or `config.LoadFrom(root)` if the
  service exposes its root) to obtain the `config.Config` value needed by
  `ResolvePlanByNumber`. Check how other handlers obtain the config (e.g.
  `entityTransitionAction` calls `entitySvc.AdvanceFeatureStatus` which loads
  it internally — determine the cleanest pattern to follow).
- Read `internal/mcp/entity_tool_test.go` and `internal/mcp/status_tool_test.go`
  for test patterns.

**Output artifacts:**
- `internal/mcp/entity_tool.go` — add a helper (e.g. `resolveShortPlanRef`) that
  wraps the `ParseShortPlanRef` + `ResolvePlanByNumber` pattern and call it in the
  three action functions above.
- `internal/mcp/status_tool.go` — apply the same pre-resolution before
  `inferIDType`.
- Test additions:
  - `entity_tool_test.go`: test that `entity(action: "get", id: "P<n>")` returns
    plan details when the plan exists; test that an unknown prefix returns an error
    containing "unknown plan prefix"; test that a full canonical ID still works.
  - `status_tool_test.go`: test that `status(id: "P<n>")` resolves correctly.

**Constraints:**
- Pre-resolution fires only when `ParseShortPlanRef` returns `ok = true`.
  All existing full-ID and FEAT/TASK/BUG paths are unchanged (FR-011).
- When `ResolvePlanByNumber` returns an error, surface it to the caller and
  do not fall through (FR-012).
- Do not add a new config-load round-trip on the hot path for non-plan IDs.

---

## 5. Dependency Graph

```
Task 1 (ParseShortPlanRef)   ──────────────────────┐
                                                    ├──→ Task 3 (tool integration)
Task 2 (ResolvePlanByNumber) ──────────────────────┘

Parallel groups: [Task 1, Task 2]
Critical path:   Task 1 → Task 3  (or Task 2 → Task 3, same length)
```

---

## 6. Risk Assessment

### Risk: Config loading in tool handlers
- **Probability:** medium
- **Impact:** low
- **Mitigation:** `entityGetAction` does not currently load config; the implementer
  should check how the entity service exposes its root (`s.root`) and whether
  `config.Load()` without arguments uses the working directory (which is the repo
  root in production). If unclear, add a `Root() string` accessor to `EntityService`
  and load config via `config.LoadFrom(entitySvc.Root())`.
- **Affected tasks:** Task 3.

### Risk: `ListPlanIDs` availability
- **Probability:** low
- **Impact:** low
- **Mitigation:** Confirm that `ListPlanIDs` (or the equivalent cache-backed
  listing method used by `ResolvePrefix`) is available on `EntityService` before
  writing `ResolvePlanByNumber`. If only a file-glob approach is available, use
  it — the degraded O(n) path is acceptable per the spec (NFR-002 says "must
  not introduce a new O(n) path"; it must delegate to the existing list operation
  whatever that operation currently is).
- **Affected tasks:** Task 2.

---

## 7. Verification Approach

Run after all tasks are complete:

```
go build ./...
go test ./internal/model/...
go test ./internal/service/...
go test ./internal/mcp/...
go test -race ./...
go vet ./...
```

| Acceptance Criterion | Verification Method | Producing Task |
|----------------------|--------------------|--------------------|
| AC-001 (entity + short ref resolves) | Integration test in entity_tool_test.go | Task 3 |
| AC-002 (status + short ref resolves) | Integration test in status_tool_test.go | Task 3 |
| AC-003 (unknown prefix error message) | Unit test in entity_tool_test.go | Task 3 |
| AC-004 (full canonical ID unchanged) | Regression test in entity_tool_test.go | Task 3 |
| AC-005 (FEAT ID unchanged) | Existing tests pass | Task 3 |
| AC-006 (`ParseShortPlanRef("P30")`) | Unit test in model/entities_test.go | Task 1 |
| AC-007 (`ParseShortPlanRef("P30-foo")` = false) | Unit test | Task 1 |
| AC-008 (`ParseShortPlanRef("30")` = false) | Unit test | Task 1 |
| AC-009 (`ParseShortPlanRef("")` = false) | Unit test | Task 1 |
| AC-010 (`ParseShortPlanRef("ñ5")` = ok) | Unit test | Task 1 |
| AC-011 (no-match plan returns error) | Unit test in service/entities_test.go | Task 2 |