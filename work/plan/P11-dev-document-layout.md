# P11 Feature D: Document Layout — Dev Plan

| Document | P11 Feature D: Document Layout — Dev Plan         |
|----------|---------------------------------------------------|
| Feature  | FEAT-01KMWJ3ZRBYFH (`document-layout`)            |
| Status   | Draft                                             |
| Related  | `work/spec/spec-document-layout.md`               |
|          | `work/design/fresh-install-experience.md` §5.4    |
|          | `work/spec/init-command.md`                       |

---

## 1. Implementation Approach

Three self-contained changes, applied in order:

1. **Model** — add `plan` and `retrospective` constants to `internal/model/entities.go` and
   include them in `AllDocumentTypes()`. Because `ValidDocumentType()` derives from
   `AllDocumentTypes()`, no further service or handler changes are needed for type validation.

2. **Config writer** — update `DefaultDocumentRoots()` to return the eight-entry layout and
   extend `InferDocType()` with the three new basename cases (`plan`, `report`, `retro`).
   Existing cases are preserved unchanged for backwards compatibility.

3. **Init command** — after creating the work directories, write `work/README.md` from a
   static template. Gate the write on the same `isNewProject` condition that guards directory
   creation so existing projects are never touched. Extend the `--skip-work-dirs` guard to
   cover README creation as well.

No migrations, no config changes for existing projects, no document intelligence changes.

---

## 2. Task Breakdown

| # | Task | Files touched | Estimate |
|---|------|---------------|----------|
| 1 | Add `plan` and `retrospective` document type constants | `internal/model/entities.go` | S |
| 2 | Update `DefaultDocumentRoots()` and `InferDocType()` + tests | `internal/kbzinit/config_writer.go`, `internal/kbzinit/init_test.go` | S |
| 3 | Write `work/README.md` on init + tests | `internal/kbzinit/init.go`, `internal/kbzinit/init_test.go` | S |

### Task 1 — Add new document types to model

Add two named constants and include them in the `AllDocumentTypes()` return slice:

- `DocumentTypePlan = "plan"`
- `DocumentTypeRetrospective = "retrospective"`

`AllDocumentTypes()` already drives `ValidDocumentType()`, so AC-1 through AC-5 are satisfied
with no further changes outside `entities.go`.

**Files:** `internal/model/entities.go`
**Estimate:** S

---

### Task 2 — Update `DefaultDocumentRoots()` and `InferDocType()`

**`internal/kbzinit/config_writer.go`**

Replace the five-entry `DefaultDocumentRoots()` slice with the eight-entry layout in spec order:

| Index | Path | Type |
|-------|------|------|
| 0 | `work/design` | `design` |
| 1 | `work/spec` | `specification` |
| 2 | `work/plan` | `plan` |
| 3 | `work/dev` | `dev-plan` |
| 4 | `work/research` | `research` |
| 5 | `work/report` | `report` |
| 6 | `work/review` | `report` |
| 7 | `work/retro` | `retrospective` |

Add three cases to the `InferDocType()` switch, retaining all existing cases:
- `"plan"` → `"plan"`
- `"report"` → `"report"`
- `"retro"` → `"retrospective"`

**`internal/kbzinit/init_test.go`**

- `TestInferDocType` — add rows for `"plan"`, `"report"`, `"retro"` and verify the
  backwards-compatible cases (`"dev"`, `"reports"`, `"spec"`, `"research"`, `"design"`,
  unknown fallback) are still covered.
- `TestWriteInitConfig_CanonicalContent` — update assertion to expect all eight
  `document_roots` entries.
- `TestRun_NewProject_CreatesWorkDirs` — update assertion to expect all eight directories.

**Files:** `internal/kbzinit/config_writer.go`, `internal/kbzinit/init_test.go`
**Estimate:** S

---

### Task 3 — Write `work/README.md` on init

**`internal/kbzinit/init.go`**

In `runNewProject`, immediately after the `createWorkDirs` call, write `work/README.md` with
the exact static content from spec §5. Guard the write with the same condition that gates
`createWorkDirs` (new project path only). Extend the `--skip-work-dirs` branch to skip the
README write as well. Do not write or overwrite the README on an existing project path.

The README content is a compile-time constant (a `const` block or `var` at package level) —
do not construct it dynamically.

**`internal/kbzinit/init_test.go`**

Add the following test cases (table-driven or individual, whichever matches existing style):

| Test | Assertion |
|------|-----------|
| New project creates README | `work/README.md` exists after `kbz init` on a new project |
| README has correct content | File contains the eight-row directory table and the AI agents line |
| `--skip-work-dirs` skips README | `work/README.md` does not exist when flag is set |
| Existing project skips README | `work/README.md` is not created when `.kbz/config.yaml` already exists |
| Second run does not overwrite | README content is unchanged after a second `kbz init` |

**Files:** `internal/kbzinit/init.go`, `internal/kbzinit/init_test.go`
**Estimate:** S

---

## 3. Dependencies

### Intra-feature dependency

Tasks 2 and 3 depend on Task 1. `DefaultDocumentRoots()` references `DocumentTypePlan` and
`DocumentTypeRetrospective` constants; those constants must exist in `internal/model/entities.go`
before `config_writer.go` is updated.

Task 3 has no dependency on Task 2 — `init.go` does not call `DefaultDocumentRoots()` directly.
Tasks 2 and 3 can be developed in parallel once Task 1 is merged.

### Inter-feature dependency (P11)

This feature (Feature D) should be implemented **first** among the four P11 features. The new
document type names `plan` and `retrospective` are referenced in skill content written by
Feature B. Feature B must not be merged before Task 1 of this feature is complete.

Feature C (init command updates) and Feature D share `internal/kbzinit/`. If both are
developed concurrently, coordinate on `init_test.go` to avoid merge conflicts. Prefer
sequencing Feature D before Feature C.

### External dependencies

None. All changes are confined to `internal/model/` and `internal/kbzinit/`. No service layer,
MCP handler, or document intelligence changes are required.