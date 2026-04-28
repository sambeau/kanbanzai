# Entity Names — Review Remediation Handoff

| Document | Entity Names Review Remediation                              |
|----------|--------------------------------------------------------------|
| Status   | Ready                                                        |
| Feature  | FEAT-01KN48ET59G8R (entity-names)                           |
| Branch   | feature/FEAT-01KN48ET59G8R-entity-names                     |
| Worktree | .worktrees/FEAT-01KN48ET59G8R-entity-names/                 |
| Review   | work/reviews/review-FEAT-01KN48ET59G8R-entity-names.md      |
| Spec     | work/spec/entity-names.md                                    |

---

## Context

The Entity Names feature implementation has been reviewed and is **changes-required**.
All 24 test packages pass and the backfill is complete. The issues below are the
remaining gaps before the feature is ready to merge.

---

## Required fixes (defects — must fix before merge)

### Fix 1 — Add `Name` to the runtime `Config` struct (F-01 / AC-07)

**File:** `internal/config/config.go`

The `Config` struct (line ~146) is missing a `Name` field. The project name is
written to `config.yaml` by `kbz init` but is silently ignored on read because
the struct has no corresponding field.

Add `Name string \`yaml:"name,omitempty"\`` as the **second field** in the struct,
after `Version` and before `SchemaVersion`, so the canonical YAML field order
matches the write order (version → name → schema_version → prefixes → …).

No changes to `Load()` or `LoadFrom()` are needed — the yaml library will
populate the new field automatically on unmarshal.

---

### Fix 2 — Validate the project name in `kbz init` (F-02 / AC-18)

**File:** `internal/kbzinit/init.go`

`resolveProjectName()` returns the user-supplied string without calling
`validate.ValidateName`. Any string — including `"P4: Bad Name"` — is
accepted and written to `config.yaml`.

After obtaining the trimmed name value and before returning it, call
`validate.ValidateName(name)`. Return the error to the caller if validation
fails. Use the trimmed value returned by `ValidateName` as the stored name
(it strips leading/trailing whitespace).

The `validate` package is already imported in `init.go`.

---

### Fix 3 — Write `name` unconditionally in Feature, Task, Decision field functions (F-03 / AC-28)

**File:** `internal/service/entities.go`

Three functions conditionally include `name` in their output map, meaning
an entity with an empty name produces YAML without a `name:` key. The
other four entity types (Plan, Epic, Bug, Incident) include `name`
unconditionally. Make these three consistent.

Find and change these three `if` guards to unconditional assignments:

```
// featureFields — around line 1094
if e.Name != "" {
    fields["name"] = e.Name      // ← remove the if, keep the assignment
}

// taskFields — around line 1124
if e.Name != "" {
    fields["name"] = e.Name      // ← same
}

// decisionFields — around line 1298
if e.Name != "" {
    fields["name"] = e.Name      // ← same
}
```

Change each to simply: `fields["name"] = e.Name`

---

### Fix 4 — Enforce name as required in the service for Feature, Task, Decision (F-05 / AC-02)

**File:** `internal/service/entities.go`

`CreateFeature`, `CreateTask`, and `CreateDecision` do not validate that
`input.Name` is non-empty before creating the entity. Plan, Epic, Bug, and
Incident all reject an empty name at the service layer. Make Feature, Task,
and Decision consistent.

In each of these three create functions, add a `validate.ValidateName` call
near the top of the function (after the slug is resolved but before the
entity is written). Pattern to follow — look at how `CreateBug` or `CreatePlan`
handle name validation and replicate the same approach:

```
name, err := validate.ValidateName(input.Name)
if err != nil {
    return EntityResult{}, fmt.Errorf("invalid name: %w", err)
}
```

Use the returned `name` (trimmed) value when setting the struct field and
storing into the fields map.

---

### Fix 5 — Pass `Name` from CLI flags for Feature, Task, Decision (F-04 / AC-19, AC-21)

**File:** `cmd/kanbanzai/entity_cmd.go`

Three entity types in `runEntityCreate` do not pass `Name` from the parsed
flag values to their service input struct. Additionally the usage text still
says `--title`.

**Change 1 — Usage text (line ~16):**
Replace `--title` with `--name` in `entityUsageText`.

**Change 2 — Feature create (around line 88):**
```
result, err := svc.CreateFeature(service.CreateFeatureInput{
    Slug:      values["slug"],
    Parent:    values["parent"],
    Name:      values["name"],      // ← add this line
    Summary:   values["summary"],
    CreatedBy: values["created_by"],
})
```

**Change 3 — Task create (around line 104):**
```
result, err := svc.CreateTask(service.CreateTaskInput{
    ParentFeature: values["parent_feature"],
    Slug:          values["slug"],
    Name:          values["name"],   // ← add this line
    Summary:       values["summary"],
})
```

**Change 4 — Decision create (around line 130):**
```
result, err := svc.CreateDecision(service.CreateDecisionInput{
    Slug:      values["slug"],
    Name:      values["name"],       // ← add this line
    Summary:   values["summary"],
    Rationale: values["rationale"],
    DecidedBy: values["decided_by"],
})
```

---

## Secondary fixes (gaps — address in same pass)

### Fix 6 — Validate name on update in the MCP tool (F-14 / AC-20)

**File:** `internal/mcp/entity_tool.go`

The update path collects field changes into a map and passes them through
`strings.TrimSpace` but does not call `validate.ValidateName` when `name`
is one of the updated fields. A caller can update an entity with
`name: "P4: Bad"` and it will be stored.

Find the update handler section that processes the `name` field. After
trimming, call `validate.ValidateName(trimmed)` and return an error result
if it fails.

---

## Test gaps (secondary — address in same pass)

These named tests from the spec §5 test plan are absent and should be added.

| Test to add | File | What to test |
|---|---|---|
| `TestNameRoundTrip_Bug` | `internal/storage/entity_store_test.go` | Bug with name round-trips; no `title` key in output |
| `TestNameRoundTrip_Incident` | `internal/storage/entity_store_test.go` | Same for Incident |
| `TestLabelFieldAbsent_Task` | `internal/storage/entity_store_test.go` | Serialised Task YAML never contains `label:` |
| `TestConfigNameField` | `internal/kbzinit/init_test.go` | `config.yaml` with `name:` round-trips via `Config` struct after Fix 1 |
| `TestEntityGetIncludesName` | `internal/mcp/entity_tool_test.go` | `entity get` response includes `name` field |
| `TestEntityListIncludesName` | `internal/mcp/entity_tool_test.go` | `entity list` rows include `name` field |
| `TestEntityToolNoTitleParam` | `internal/mcp/entity_tool_test.go` | Passing `title` to entity tool returns an unknown-parameter error or is ignored |
| `TestEntityToolNoLabelParam` | `internal/mcp/entity_tool_test.go` | Same for `label` |
| Fix batch create tests | `internal/mcp/entity_tool_test.go` | `TestEntity_Create_BatchTasks` and `TestEntity_Create_MutationHasSideEffectsField` pass `name` so they exercise the success path, not the validation-failure path |

---

## Verification

After all fixes, run:

```
go test -race ./...
```

All 24 packages must pass. Specifically confirm:
- `internal/config` — `Config.Name` round-trips in config load tests
- `internal/validate` — existing `TestValidateName` still passes
- `internal/service` — Feature/Task/Decision create with empty name returns error
- `internal/mcp` — new entity get/list tests assert `name` in responses
- `cmd/kanbanzai` — CLI create for Feature/Task/Decision with `--name` flag works

---

## Commit guidance

Group the fixes into two commits:

```
fix(FEAT-01KN48ET59G8R): add Config.Name field, validate project name in init, enforce name in Feature/Task/Decision service and storage
fix(FEAT-01KN48ET59G8R): pass name from CLI for feature/task/decision, add missing test coverage
```

---

## Workflow currency (after fixes land)

Once the remediation is committed and tests pass, update document statuses:

- `work/spec/entity-names.md` — change `Status: Draft` to `Status: Approved`
- `work/design/entity-names.md` — change `Status: proposal` to `Status: implemented`
- `work/plan/entity-names-implementation-plan.md` — change `Status: Draft` to `Status: Done`
- Transition `FEAT-01KN48ET59G8R` to `reviewing` (already done) then to `done` after remediation