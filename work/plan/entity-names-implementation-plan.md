# Entity Names — Implementation Plan

| Document | Entity Names Implementation Plan                         |
|----------|----------------------------------------------------------|
| Status   | Draft                                                    |
| Created  | 2026-06-17                                               |
| Spec     | `work/spec/entity-names.md`                              |
| Design   | `work/design/entity-names.md`                            |

---

## 1. Implementation Approach

This feature delivers three coordinated changes — renaming `title` → `name`
across all entity types, retiring `label` from Feature and Task, and adding a
project name to `config.yaml` — in a single implementation pass followed by a
backfill of existing state files.

The work splits into seven tasks across five distinct code areas. Four of those
tasks are structurally independent and can be parallelised; the remaining three
follow a clear dependency chain. The recommended execution order is:

**Wave 1 (parallel):** Task 1 (model) and Task 4 (config + init). These touch
entirely different packages and have no shared state.

**Wave 2 (parallel, after Task 1):** Task 2 (storage) and Task 3 (validation).
Both depend on the updated model structs from Task 1, but do not depend on each
other. Task 6 (skill files) can also run in this wave — it has no code
dependencies at all.

**Wave 3 (after Tasks 1, 2, 3):** Task 5 (MCP tools). This task touches all
three preceding concerns — model field names, storage reads, and validation
calls — and must not begin until they are complete.

**Wave 4 (after all code tasks):** Task 7 (backfill). The backward-compat
handling introduced in Task 2 must be in place before the state files are
rewritten, so that reads remain valid throughout.

```
Wave 1:  [Task 1: Model] ─────────────────────────────────────────────┐
         [Task 4: Config + Init] ────────────────────(independent)    │
                                                                       │
Wave 2:  [Task 2: Storage] (after T1) ────────────────────────────────┤
         [Task 3: Validation] (after T1) ──────────────────────────── │
         [Task 6: Skills] (independent) ────────────(independent)    │
                                                                       │
Wave 3:  [Task 5: MCP Tools] (after T1, T2, T3) ─────────────────────┤
                                                                       │
Wave 4:  [Task 7: Backfill] (after all code tasks) ──────────────────┘
```

Each task includes an AC cross-reference at the end of its detail section.
Before starting any task, read the spec (`work/spec/entity-names.md`) in full
and the design document (`work/design/entity-names.md`) for rationale and
examples.

---

## 2. Task Breakdown

| # | Task | Primary Files | Est |
|---|------|---------------|-----|
| 1 | Model — rename `title`→`name`, add `name` to Feature/Task/Decision, remove `label` | `internal/model/entities.go` | S |
| 2 | Storage — update field order, backward-compat reads, round-trip tests | `internal/storage/entity_store.go`, `internal/storage/entity_store_test.go` | M |
| 3 | Validation — `ValidateName` with five rules, remove `ValidateLabel` | `internal/validate/entity.go`, `internal/validate/entity_test.go` | S |
| 4 | Config + init — add `name` to config model, `kbz init` prompt and flag | `internal/kbzinit/config_writer.go`, `internal/kbzinit/init.go`, `internal/kbzinit/init_test.go` | M |
| 5 | MCP tools — rename `title` param to `name`, remove `label` param, update responses | `internal/mcp/entity_tool.go` | M |
| 6 | Skill files — add naming guidance, rules, and examples | `internal/kbzinit/skills/agents/SKILL.md`, `internal/kbzinit/skills/plan-review/SKILL.md` | S |
| 7 | Backfill — write conforming `name` values to all `.kbz` state files | `.kbz/state/**/*.yaml` | M |

---

## 3. Task Details

### Task 1: Model

**Goal:** Update all entity structs so that `Name string` is the single
consistent display-name field. This is the foundation every other task builds
on.

**File:** `internal/model/entities.go`

**Changes required:**

- **Plan struct:** Remove the `Title string \`yaml:"title"\`` field. Add
  `Name string \`yaml:"name"\`` in its place (same position in the struct
  definition — immediately after `Slug`).

- **Epic struct:** Same change as Plan — remove `Title`, add `Name`.

- **Bug struct:** Same change — remove `Title`, add `Name`.

- **Incident struct:** Same change — remove `Title`, add `Name`.

- **Feature struct:** Remove the `Label string \`yaml:"label,omitempty"\``
  field entirely. Add `Name string \`yaml:"name"\`` after `Slug` and before
  the `Parent` field.

- **Task struct:** Remove the `Label string \`yaml:"label,omitempty"\`` field
  entirely. Add `Name string \`yaml:"name"\`` after `Slug` and before
  `Summary`.

- **Decision struct:** Add `Name string \`yaml:"name"\`` after `Slug` and
  before `Summary`. Decision has no `Title` or `Label` to remove — this is a
  net addition only.

**Interface contract for downstream tasks:**

Tasks 2, 3, and 5 all reference struct fields directly. After this task, the
canonical field names are:

| Struct field | YAML key | Present on |
|---|---|---|
| `Name` | `name` | All entity types |
| ~~`Title`~~ | ~~`title`~~ | Removed from Plan, Epic, Bug, Incident |
| ~~`Label`~~ | ~~`label`~~ | Removed from Feature, Task |

Tasks 2, 3, and 5 must use `Name` (not `Title` or `Label`) when accessing the
display-name field on any entity struct after this change.

**Spec ACs covered:** AC-01 through AC-06.

---

### Task 2: Storage and serialisation

**Goal:** Reflect the model changes in the storage layer — canonical YAML field
order for all entity types, backward-compat reads during the backfill window,
and verified round-trip stability.

**Depends on:** Task 1 (struct field names must be finalised first).

**Files:** `internal/storage/entity_store.go`, `internal/storage/entity_store_test.go`

**Background:** The function `fieldOrderForEntityType` in `entity_store.go`
defines the canonical YAML field order for each entity type as a `[]string`.
This is the single place to update field ordering. The storage layer works with
raw `map[string]any` state rather than typed structs, so field names are string
keys — `"name"`, `"title"`, `"label"` — not struct field references.

**Changes to `fieldOrderForEntityType`:**

For each entity type, update the returned slice as follows. The surrounding
fields (those not mentioned) do not change.

- **plan:** Replace `"title"` with `"name"` in the same position (after `"slug"`,
  before `"status"`).

- **epic:** Replace `"title"` with `"name"` in the same position (after `"slug"`,
  before `"status"`).

- **feature:** Remove `"label"` from the slice. Add `"name"` after `"slug"` and
  before `"parent"` (i.e., the new order at the top is: `"id"`, `"slug"`,
  `"name"`, `"parent"`, `"status"`, ...).

- **task:** Remove `"label"` from the slice. Add `"name"` after `"slug"` and
  before `"summary"` (i.e.: `"id"`, `"parent_feature"`, `"slug"`, `"name"`,
  `"summary"`, `"status"`, ...).

- **bug:** Replace `"title"` with `"name"` in the same position (after `"slug"`,
  before `"status"`).

- **decision:** Add `"name"` after `"slug"` and before `"summary"` (decision
  currently has no `title` or `label` — this is a net addition).

- **incident:** Replace `"title"` with `"name"` in the same position (after
  `"slug"`, before `"status"`).

**Backward-compat reads (backfill window):**

The storage layer reads entities as raw maps. Add logic that runs when an entity
is read from disk, before it is returned to callers:

1. If the map contains `"title"` but not `"name"`: copy the `"title"` value to
   `"name"` and remove `"title"` from the map. This handles Plan, Epic, Bug, and
   Incident files that have not yet been backfilled.

2. If the map contains neither `"title"` nor `"name"`, and the entity type is
   `feature`, `task`, or `decision`: log a validation warning (do not return an
   error) indicating that the name field is absent. Return the entity as-is so
   callers are not blocked.

This logic must be removed once the backfill (Task 7) is complete and verified —
add a `// TODO: remove after backfill verified` comment to make the removal
obvious.

**Round-trip tests:**

Add or update tests in `entity_store_test.go` to cover the cases in the test
plan. Each test should: construct a minimal valid entity map with a `"name"`
value, write it to a temp directory, read it back, and assert that the
`"name"` key is present with the correct value and that no `"title"` or `"label"`
key is present. Existing tests that use `"title"` in their fixture maps must
be updated to use `"name"`.

**Spec ACs covered:** AC-06, AC-27, AC-28, AC-29, AC-30, AC-31.

---

### Task 3: Validation

**Goal:** Add a `ValidateName` function that enforces the five hard rules from
the spec. Remove `ValidateLabel`, which is no longer needed.

**Depends on:** Task 1 (the model change confirms `Name` is the field being
validated; no direct struct dependency, but Task 1 should be committed first for
clarity).

**Files:** `internal/validate/entity.go`, `internal/validate/entity_test.go`

**Changes to `entity.go`:**

Add a `ValidateName(name string) (string, error)` function. The return type
includes the trimmed name so callers can use the normalised value directly
without a separate trim call. The function must enforce these rules in order:

1. Trim leading and trailing whitespace from the input before any other check
   (the trimmed value is what gets stored and returned).
2. If the trimmed value is empty, return an error.
3. If `len(trimmed) > 60`, return an error.
4. If `trimmed` contains a colon (`:`), return an error.
5. If `trimmed` matches the phase-prefix pattern — a single uppercase ASCII
   letter immediately followed by one or more ASCII digits, then a space,
   hyphen (`-`), or em-dash (`—`) — return an error.

Remove `ValidateLabel` and its constant `maxLabelLength`. If `ValidateLabel`
is called from any other location in the codebase, those call sites must be
removed as part of this task (not left as dead code).

Also update the entity-level validation entry point (the function that
dispatches per-field validation based on entity type) to call `ValidateName`
for all entity types instead of `ValidateTitle` (if such a dispatch function
exists) and to stop dispatching `ValidateLabel` for feature and task.

**Tests in `entity_test.go`:**

Cover the tests listed in the spec test plan under `TestNameRequired`,
`TestNameEmpty`, `TestNameTooLong`, `TestNameWhitespaceTrimmed`,
`TestNameNoColon`, `TestNameNoPhasePrefix`, and `TestNamePhasePrefixBoundary`.
The boundary test must confirm that two-letter prefixes (`"AC-01 something"`),
labels with no digit (`"G policy-docs"`), and ordinary names beginning with
an uppercase letter (`"Public API endpoints"`) are all accepted.

**Interface contract for Task 5:**

Task 5 (MCP tools) will call `validate.ValidateName(name)` and use the returned
trimmed string as the stored value. The function signature is:

```
func ValidateName(name string) (trimmed string, err error)
```

**Spec ACs covered:** AC-13 through AC-18.

---

### Task 4: Config model and `kbz init`

**Goal:** Give the project itself a canonical name by adding a `name` field to
`config.yaml` and updating `kbz init` to prompt for it.

**Independent of Tasks 1–3.** This task touches a separate package
(`internal/kbzinit/`) and has no dependency on the entity model changes.

**Files:** `internal/kbzinit/config_writer.go`, `internal/kbzinit/init.go`,
`internal/kbzinit/init_test.go`

**Changes to `config_writer.go`:**

The `initFileConfig` struct currently has `Version`, `Prefixes`, and `Documents`
fields. Add a `Name string \`yaml:"name"\`` field. The canonical field order in
the written YAML must be: `version`, `name`, `prefixes`, `documents` — i.e.,
`Name` is the second field in the struct, positioned before `Prefixes`.

Update the `WriteInitConfig` function signature to accept the project name:

```
func WriteInitConfig(kbzDir string, name string, roots []DocumentRoot) error
```

The `name` argument is written into `cfg.Name`. If `name` is empty, the field
is still written (as an empty string) so the prompt result is always persisted.

**Changes to `init.go`:**

Add a `Name string` field to the `Options` struct, alongside the existing flag
fields. This holds the value supplied via `--name` or entered at the interactive
prompt.

During the `Run` flow, before the prefix configuration step, add a name prompt:

- If `opts.Name` is non-empty (supplied via `--name`), use it directly — no
  prompt.
- If `opts.NonInteractive` is true and `opts.Name` is empty, return an error:
  `--name is required in non-interactive mode`.
- Otherwise, prompt the user: `"Project name [<default>]: "` where `<default>`
  is the base name of the working directory (`filepath.Base(i.workDir)`). If
  the user presses enter without typing, use the default.

Pass the resolved name to `WriteInitConfig`.

An existing `config.yaml` that pre-dates this change will not have a `name`
field; loading it must not error (AC-12). Verify that the config-loading path
(wherever `config.yaml` is read back) treats a missing `name` as an empty
string, not a parse error. The `yaml` package does this by default for string
fields with no `omitempty` tag — confirm this is the case and add a test.

**Tests in `init_test.go`:**

Add tests covering: `--name` flag sets project name in written config
(`TestInitNameFlag`); default name is derived from working directory
(`TestInitNameDefault`); existing `config.yaml` without `name` loads without
error (`TestConfigNameMissing`).

**Spec ACs covered:** AC-07 through AC-12.

---

### Task 5: MCP tools

**Goal:** Update the `entity` tool to use `name` consistently — renaming the
`title` parameter, removing the `label` parameter, and surfacing `name` in all
entity responses, lists, and the status dashboard.

**Depends on:** Tasks 1, 2, and 3. Do not begin until all three are complete and
their changes are committed to the feature branch.

**File:** `internal/mcp/entity_tool.go`

**Background:** This is the most change-dense task. Read the existing file
thoroughly before starting. Key areas:

- The tool parameter definitions (the `mcp.WithString(...)` calls near the top
  of the tool constructor) — these are the MCP schema declarations.
- The create handler section — branched by entity type, each branch sets struct
  fields from `args`.
- The update handler section — applies partial updates from `args` to existing
  state maps.
- The get response formatter — assembles the output map returned to callers.
- The list response formatter — builds the row for each entity in the list.
- The status dashboard formatter — builds feature and task rows.
- The `entityArgStr` and `entityStateStr` helper functions — used to read args
  and state map values respectively.

**Parameter schema changes:**

- Remove the `mcp.WithString("title", ...)` declaration.
- Remove the `mcp.WithString("label", ...)` declaration.
- Add `mcp.WithString("name", mcp.Description("Human-readable display name for the entity (required on create). Short, ~4 words, no colon, no phase prefix."))`.

**Create handler changes:**

For every entity type that previously set `Title:` from `entityArgStr(args, "title")`,
change to set `Name:` from `entityArgStr(args, "name")`. This affects Plan, Epic,
Bug, and Incident creation paths.

For Feature, Task, and Decision creation paths (which previously had no `title`
or only `label`), add `Name: entityArgStr(args, "name")` to the struct
initialiser.

Remove any line that sets `Label:` from `entityArgStr(args, "label")`.

Name is required on create: if `entityArgStr(args, "name")` is empty and the
entity type requires a name (all types), return a validation error before
creating the entity. Call `validate.ValidateName(name)` (Task 3's function) to
both validate and obtain the trimmed value.

**Update handler changes:**

The update path applies changes to an existing state map. The current code has a
branch that checks `if _, has := args["title"]` and applies the update. Change
this to check `args["name"]` and apply it to the `"name"` key in the state map,
calling `validate.ValidateName` before storing.

Remove the `label` update branch (the block that currently checks
`if v, exists := args["label"]`).

**Get response changes:**

In the response map assembled by `entity get`, include `"name"` from
`entityStateStr(result.State, "name")`. This field should appear early in the
response — alongside the ID and slug — so callers see it prominently. Remove
any code that reads `"title"` or `"label"` from the state map for inclusion in
the response.

**List response changes:**

In the row map built for each entity in `entity list`, include `"name"` from
the entity's state. Remove the `"label"` field from the row. Remove the
`labelFilter` logic that currently filters by `args["label"]`.

**Status dashboard changes:**

Feature rows and task rows in the status dashboard output include `"name"`.
Remove the conditional `has_labels` logic and the Label column. The Name column
is always present.

**Spec ACs covered:** AC-19 through AC-26.

---

### Task 6: Skill files

**Goal:** Add entity naming guidance to the skill content installed by
`kbz init`, and update any existing skill content that references `title` in
the context of entity creation.

**Independent of all code tasks.** This task can be worked at any point.

**Primary file:** `internal/kbzinit/skills/agents/SKILL.md`

This is the source for the skill installed to `.agents/skills/kanbanzai-agents/SKILL.md`
by `kbz init`. It contains the dispatch-and-completion protocol, commit format
guidance, and knowledge contribution instructions. It is where agents learn how
to create entities.

Add a new section titled **"Entity Names"** (or within an existing entity
creation section if one exists). The section must include:

**Hard rules** (present these as a checklist agents must follow):
- Name is required on all entity types — do not omit it.
- Maximum 60 characters.
- No colon (`:`) — the system will reject it.
- No phase or version prefix: do not begin a name with a pattern like `P4`, `P8`,
  `P11` followed by a space or dash. The phase is already encoded in the entity's
  parent; repeating it in the name adds noise.
- Leading and trailing whitespace is stripped automatically.

**Soft rules** (present as guidance for quality):
- Target approximately four words. If you need more than five or six, the scope
  is probably too broad or the name is doing the summary's job.
- No em-dashes used as separators (a hyphen in a compound term like
  "Human-friendly ID display" is fine; "P8 — decompose" is not).
- A name should not be merely the slug capitalised. `init-command` → "Init command"
  adds nothing. Prefer "Project init command" or "Init and skill install" —
  something that identifies the entity without a lookup.
- Names must be self-contained: readable without knowing the parent entity.
  "Update agents" is ambiguous; "Update AGENTS.md layout" is not.
- Do not include the parent plan or feature name in the entity name. The
  hierarchy is visible from the parent field.

**Examples** (include all of these, formatted as a table or labelled list):

Good examples:
- "Kanbanzai 2.0" (Plan)
- "Human-friendly ID display" (Feature)
- "Init and skill install" (Feature)
- "Server info tool" (Task)
- "Label model and storage" (Task)
- "Use TSID for entity IDs" (Decision)

Bad examples (annotate each with the violated rule):
- "P4 Kanbanzai 2.0: MCP Tool Surface Redesign" — phase prefix and colon
- "P8 — decompose propose Reliability Fixes" — phase prefix and separator dash
- "The kanbanzai init command: creates .kbz/config.yaml, installs skill files" — colon and far too long; this is a summary, not a name
- "Update" — too vague, not self-contained

**Also update:** `internal/kbzinit/skills/plan-review/SKILL.md`

This file contains two references to `{plan title}` in template strings (lines
210 and 277 of the current file). Change both to `{plan name}` to match the
renamed field.

After updating the embedded sources, the live installed files at
`.agents/skills/kanbanzai-agents/SKILL.md` and
`.agents/skills/kanbanzai-plan-review/SKILL.md` must also be updated in the
same commit so the running project benefits immediately without requiring a
`kbz init --update-skills` pass.

**Spec ACs covered:** AC-32 through AC-34.

---

### Task 7: Backfill

**Goal:** Write conforming `name` values to every entity YAML file in the
`.kbz` store, rename `title` → `name` on existing files, and verify that no
`label` fields remain.

**Depends on:** All code tasks (Tasks 1–6) must be committed to the feature
branch before the backfill begins. The backward-compat handling introduced in
Task 2 must be in place so that reads continue to work as files are updated
progressively.

**Process:**

The backfill is performed by an agent working through the state directories in
order. For each entity file:

1. Read the current YAML.
2. If it has a `title` field: derive a conforming `name` from it (see naming
   rules below), write the `name` field, and remove `title`.
3. If it has neither `title` nor `name` (Feature, Task, Decision): derive a
   conforming `name` from the `slug` and `summary` fields.
4. If it has a `label` field: remove it (no data loss — no entities carry a
   label value).
5. Write the file back.

**Naming rules for derived names** — follow the guidance in the spec (§4.3) and
the design (§3.2 and §3.3):

- The name should be approximately four words.
- Use the slug as the starting point: convert hyphens to spaces and capitalise
  the first word. Then improve it — add a word of context if the slug is too
  terse, remove redundant words if it is too long.
- Use the `summary` field for additional context when the slug alone is
  ambiguous.
- For Plans: the current `title` values contain phase prefixes and colons.
  Strip the phase prefix (e.g. `"P4 Kanbanzai 2.0: MCP Tool Surface Redesign"`
  becomes `"Kanbanzai 2.0"` or `"MCP tool surface"`).
- For Decisions: the slug is often already a good name (`"use-tsid-for-ids"` →
  `"Use TSID for IDs"`).
- Never produce a name longer than 60 characters.
- Never produce a name containing a colon.

**State directories to process** (in order):

1. `.kbz/state/plans/` — rename `title` → `name`, rewrite to remove prefixes/colons
2. `.kbz/state/features/` — add `name` derived from slug/summary
3. `.kbz/state/tasks/` — add `name` derived from slug/summary
4. `.kbz/state/bugs/` — rename `title` → `name`
5. `.kbz/state/decisions/` — add `name` derived from slug/summary
6. `.kbz/state/incidents/` — rename `title` → `name`

**Verification:**

After all files are updated, run the following checks:

```
grep -rl "^title:" .kbz/state/     # should produce no output
grep -rl "^label:" .kbz/state/     # should produce no output
grep -rL "^name:" .kbz/state/      # should produce no output (every file has name)
```

Run `go test ./...` to confirm all tests pass with the backfilled state.

Once verification passes, remove the backward-compat handling in
`internal/storage/entity_store.go` (the `// TODO: remove after backfill
verified` blocks introduced in Task 2) and commit the removal.

**Spec ACs covered:** AC-35 through AC-38, and the removal of AC-29/AC-30
compatibility handling (AC-31).

---

## 4. Dependencies

| Task | Depends on | Notes |
|------|-----------|-------|
| Task 1 | — | Independent; start immediately |
| Task 2 | Task 1 | Needs finalised struct field names and YAML keys |
| Task 3 | Task 1 | Needs confirmation of `Name` as the validated field |
| Task 4 | — | Independent; can run in parallel with Task 1 |
| Task 5 | Tasks 1, 2, 3 | All three must be committed before Task 5 begins |
| Task 6 | — | Independent; can run at any point |
| Task 7 | Tasks 1–6 | All code changes must be on the branch; backward-compat reads must be in place |

### Parallelism guide

Tasks 1 and 4 can be assigned to separate agents simultaneously — they touch
different packages (`internal/model/` vs `internal/kbzinit/`) with no shared
files.

Tasks 2, 3, and 6 can be assigned to separate agents simultaneously after
Task 1 is merged to the feature branch. Tasks 2 and 3 both touch
`internal/storage/` and `internal/validate/` respectively — different packages,
no conflict.

Task 5 is the integration point and should be a single agent working alone.

Task 7 is a sequential pass through state files; no parallelism is beneficial.

---

## 5. Acceptance Criteria Coverage

| Spec section | AC range | Covered by |
|---|---|---|
| §4.1 Model — Name field | AC-01 – AC-06 | Task 1 |
| §4.2 Config — Project name | AC-07 – AC-12 | Task 4 |
| §4.3 Validation | AC-13 – AC-18 | Task 3 |
| §4.4 MCP tools — create and update | AC-19 – AC-23 | Task 5 |
| §4.5 MCP tools — list and status | AC-24 – AC-26 | Task 5 |
| §4.6 Storage and serialisation | AC-27 – AC-31 | Task 2 (AC-27 – AC-30); Task 7 (AC-31 removal) |
| §4.7 Skill files | AC-32 – AC-34 | Task 6 |
| §4.8 Backfill | AC-35 – AC-38 | Task 7 |

All 38 acceptance criteria from `work/spec/entity-names.md` are covered.