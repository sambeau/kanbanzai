# Document Layout Specification

| Document | Document Layout Specification                             |
|----------|-----------------------------------------------------------|
| Status   | Draft                                                     |
| Feature  | FEAT-01KMWJ3ZRBYFH                                        |
| Related  | `work/design/fresh-install-experience.md` §5.4            |
|          | `work/spec/init-command.md`                               |

---

## 1. Purpose

This specification defines the acceptance criteria for Feature D of the P11 Fresh Install
Experience: the standard `work/` document layout that `kbz init` creates on new projects.

The scope covers three areas of change:

1. **New document types** — `plan` and `retrospective` are added to the model so that the new
   directories have first-class type support.
2. **`InferDocType()` updates** — new path-basename cases are added, and all existing cases are
   preserved for backwards compatibility.
3. **`DefaultDocumentRoots()` and `kbz init`** — the set of directories created on new projects
   expands from five to eight, `work/README.md` is written as a static directory map, and
   the `--skip-work-dirs` flag suppresses all of this.

Existing projects are not affected. Config files are not migrated. `InferDocType()` retains
every pre-existing case so that batch imports of projects using old layouts continue to work.

---

## 2. Background

### Current state

`DefaultDocumentRoots()` in `internal/kbzinit/config_writer.go` creates five directories:

| Directory | Type |
|---|---|
| `work/design` | design |
| `work/spec` | specification |
| `work/dev` | dev-plan |
| `work/research` | research |
| `work/reports` | report |

`InferDocType()` in the same file handles: `"spec"` → `specification`, `"dev"` → `dev-plan`,
`"research"` → `research`, `"reports"` → `report`; everything else → `design`.

`model.AllDocumentTypes()` lists: `design`, `specification`, `dev-plan`, `research`, `report`,
`policy`, `rca`.

### Problems being fixed

- `work/plan/` has no directory in the default layout, so human project-planning documents
  (roadmaps, scope docs, decision logs) end up mixed with design documents.
- `InferDocType("plan")` silently returns `"design"` because no case exists, causing silent
  type misclassification during batch imports.
- Review artefacts from the reviewing lifecycle gate have no dedicated directory; they
  accumulate in `work/reports/` alongside unrelated content.
- Retrospective synthesis documents (output of the `retro` tool) have no document type of
  their own, making them indistinguishable from general reports at query time.

---

## 3. Design Decisions

This spec implements the following decisions from the design document:

| ID | Summary |
|---|---|
| FI-D-004 | `work/plan/` (new `plan` type, human-facing) and `work/dev/` (existing `dev-plan` type, agent-facing) are separate directories |
| FI-D-005 | `work/report/` (singular) and `work/review/` (separate directory for review gate artefacts) both use document type `report` |
| FI-D-006 | `work/retro/` uses a new `retrospective` document type, distinct from `report` |

---

## 4. Proposed Directory Layout

`kbz init` on a new project creates the following eight directories and `work/README.md`:

| Directory | Document type | Audience | Purpose |
|---|---|---|---|
| `work/design/` | `design` | Human + Agent | Architecture decisions, technical vision, policies |
| `work/spec/` | `specification` | Human + Agent | Acceptance criteria and binding contracts |
| `work/plan/` | `plan` | Human | Project planning: roadmaps, scope, decision logs |
| `work/dev/` | `dev-plan` | Agent | Feature implementation plans and task breakdowns |
| `work/research/` | `research` | Human + Agent | Analysis, exploration, background reading |
| `work/report/` | `report` | Human + Agent | Audit reports, post-mortems, general reports |
| `work/review/` | `report` | Agent | Feature and plan review reports |
| `work/retro/` | `retrospective` | Human + Agent | Retrospective synthesis documents |

---

## 5. `work/README.md` Content

The exact content written to `work/README.md` on new installs:

```text
# work/

Workflow documents for this project. Register all documents with kanbanzai after creation.

| Directory | Type | Contents |
|---|---|---|
| `design/` | design | Architecture decisions, technical vision, policies |
| `spec/` | specification | Acceptance criteria and binding contracts |
| `plan/` | plan | Project planning: roadmaps, scope, decision logs |
| `dev/` | dev-plan | Feature implementation plans and task breakdowns |
| `research/` | research | Analysis, exploration, background reading |
| `report/` | report | Audit reports, post-mortems, general reports |
| `review/` | report | Feature and plan review reports |
| `retro/` | retrospective | Retrospective synthesis documents |

AI agents: see the `kanbanzai-documents` skill for registration instructions.
```

`work/README.md` is static. It is written once at init time and is never modified by
subsequent `kbz init` runs.

---

## 6. Acceptance Criteria

### 6.1 New document types

**AC-1.** `model.AllDocumentTypes()` returns a slice that includes both `"plan"` and
`"retrospective"`.

**AC-2.** `model.ValidDocumentType("plan")` returns `true`.

**AC-3.** `model.ValidDocumentType("retrospective")` returns `true`.

**AC-4.** `doc(action="register")` with `type: "plan"` succeeds without error (type is
accepted as valid by the registration handler).

**AC-5.** `doc(action="register")` with `type: "retrospective"` succeeds without error.

### 6.2 `InferDocType()`

**AC-6.** `InferDocType("plan")` returns `"plan"`.

**AC-7.** `InferDocType("retro")` returns `"retrospective"`.

**AC-8.** `InferDocType("report")` returns `"report"`.

**AC-9.** Backwards-compatible cases are preserved:
- `InferDocType("dev")` returns `"dev-plan"`.
- `InferDocType("reports")` returns `"report"`.
- `InferDocType("spec")` returns `"specification"`.
- `InferDocType("research")` returns `"research"`.

**AC-10.** `InferDocType("design")` returns `"design"`. Any basename that does not match a
known case also returns `"design"` (existing fallback behaviour preserved).

### 6.3 `DefaultDocumentRoots()`

**AC-11.** `DefaultDocumentRoots()` returns exactly eight entries in the following order:

| Index | Path | Type |
|---|---|---|
| 0 | `work/design` | `design` |
| 1 | `work/spec` | `specification` |
| 2 | `work/plan` | `plan` |
| 3 | `work/dev` | `dev-plan` |
| 4 | `work/research` | `research` |
| 5 | `work/report` | `report` |
| 6 | `work/review` | `report` |
| 7 | `work/retro` | `retrospective` |

The count is exactly eight. No additional entries. The order matches the table above.

### 6.4 Directory creation by `kbz init`

**AC-12.** `kbz init` on a new project creates all eight `work/` subdirectories, each
containing a `.gitkeep` placeholder file. The directories are:
`work/design`, `work/spec`, `work/plan`, `work/dev`, `work/research`, `work/report`,
`work/review`, `work/retro`.

**AC-13.** `kbz init` on an existing project (one with a pre-existing `.kbz/config.yaml`)
does **not** create any `work/` subdirectories. Existing behaviour is preserved.

### 6.5 `work/README.md`

**AC-14.** `kbz init` on a new project creates `work/README.md`.

**AC-15.** `work/README.md` contains a Markdown table listing all eight directories, their
document type, and a brief description of their contents (as specified in §5 above).

**AC-16.** `work/README.md` contains the exact line:

```
AI agents: see the `kanbanzai-documents` skill for registration instructions.
```

**AC-17.** `kbz init` on an existing project does **not** create `work/README.md`.

**AC-18.** A second `kbz init` run on a project that already has `work/README.md` does
**not** overwrite or modify it.

### 6.6 `--skip-work-dirs` flag

**AC-19.** `kbz init --skip-work-dirs` on a new project skips creation of all eight `work/`
subdirectories **and** skips creation of `work/README.md`. Neither the directories nor the
README are present after the command completes (unless they already existed before the run).

### 6.7 Backwards compatibility

**AC-20.** A project whose `config.yaml` references `work/dev/` with type `dev-plan`
continues to operate correctly without any modification. The directory name `work/dev` is
valid both before and after this change.

**AC-21.** A project whose `config.yaml` references `work/reports/` with type `report`
continues to operate correctly without any modification. The `InferDocType("reports")` →
`"report"` case is preserved.

---

## 7. Verification Methods

| AC | Method |
|---|---|
| AC-1 | Unit test: call `model.AllDocumentTypes()`, assert slice contains `"plan"` and `"retrospective"` |
| AC-2 | Unit test: `model.ValidDocumentType("plan")` == `true` |
| AC-3 | Unit test: `model.ValidDocumentType("retrospective")` == `true` |
| AC-4 | Integration test: register a document with `type: "plan"`, assert no error returned |
| AC-5 | Integration test: register a document with `type: "retrospective"`, assert no error returned |
| AC-6 | Unit test: `InferDocType("plan")` == `"plan"` |
| AC-7 | Unit test: `InferDocType("retro")` == `"retrospective"` |
| AC-8 | Unit test: `InferDocType("report")` == `"report"` |
| AC-9 | Unit tests: one assertion per backwards-compatible case |
| AC-10 | Unit tests: `InferDocType("design")` == `"design"`; `InferDocType("unknown")` == `"design"` |
| AC-11 | Unit test: call `DefaultDocumentRoots()`, assert len == 8, assert each entry's path and type in order |
| AC-12 | Integration test: run `kbz init` on a new project, assert all eight directories exist with `.gitkeep` |
| AC-13 | Integration test: run `kbz init` on an existing project, assert no new `work/` directories are created |
| AC-14 | Integration test: run `kbz init` on a new project, assert `work/README.md` exists |
| AC-15 | Integration test: read `work/README.md`, assert table rows for all eight directories are present |
| AC-16 | Integration test: read `work/README.md`, assert the exact agent instruction line is present |
| AC-17 | Integration test: run `kbz init` on an existing project, assert `work/README.md` is not created |
| AC-18 | Integration test: run `kbz init` twice on a new project, assert `work/README.md` content is unchanged |
| AC-19 | Integration test: run `kbz init --skip-work-dirs` on a new project, assert no `work/` directories and no `work/README.md` are created |
| AC-20 | Integration test: load a config with `work/dev` → `dev-plan`, assert all document operations succeed |
| AC-21 | Integration test: load a config with `work/reports` → `report`, assert all document operations succeed |

---

## 8. Out of Scope

- Migration of existing project configs — `DefaultDocumentRoots()` changes affect new installs
  only. No `kbz migrate` or automatic upgrade of `config.yaml` is performed.
- Renaming or moving existing documents between the old and new directories — this is a manual
  operation for projects that choose to adopt the new layout.
- Changes to the document intelligence layer (`doc`, `doc_intel`) — it reads from whatever
  roots are registered in `config.yaml` and is unaffected.
- The `kanbanzai-documents` skill update — that is covered by the skills feature (Feature B).
- The `--update-managed` flag — that is covered by Feature B.