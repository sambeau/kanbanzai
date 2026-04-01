# Specification: Entity Names

| Document | Entity Names                                |
|----------|---------------------------------------------|
| Status   | Draft                                       |
| Created  | 2026-06-17                                  |
| Updated  | 2026-06-17                                  |
| Design   | `work/design/entity-names.md`               |

---

## 1. Purpose

This specification defines the requirements for consistent entity naming across
the Kanbanzai system. It covers three coordinated changes:

- **Change 1** — Replace the `title` field with `name` on all entity types that
  have it; add `name` to the entity types that currently lack a dedicated display
  name field; retire the `label` field on Feature and Task.
- **Change 2** — Add a `name` field to the project configuration so the project
  itself has a canonical display name.
- **Change 3** — Define and enforce name quality rules in code and in agent
  skill guidance; backfill conforming names onto all existing entities.

---

## 2. Goals

1. Every entity type has a single, consistent `name` field serving as its
   human-readable display identifier.
2. The field is named `name` — not `title`, not `label` — on every entity type
   and in every tool interface.
3. Names are short and identity-oriented. The field name and validation rules
   together discourage verbose, formal, or structured strings.
4. The project itself has a canonical name available to any tool or viewer that
   needs to display "which project am I looking at?"
5. Name quality rules are enforced at the system boundary in code, and
   reinforced by agent skill guidance.
6. All existing entities in the `.kbz` store carry conforming names after a
   backfill pass.
7. No existing entity IDs, slugs, storage keys, or YAML filenames change.

---

## 3. Scope

### 3.1 In scope

- `internal/model/` — Add `Name` field to all entity structs; remove `Title`
  field from Plan, Epic, Bug, Incident; remove `Label` field from Feature and
  Task; add `Name` to Feature, Task, and Decision.
- `internal/storage/` — Persist and read `name` in canonical YAML field order
  for all entity types; handle backward-compatibility reads during the backfill
  window.
- `internal/validate/` — Enforce name validation rules at the service boundary.
- `internal/config/` — Add `Name` field to the project config struct.
- `internal/mcp/` — Rename `title` parameter to `name` and remove `label`
  parameter across all entity tool handlers; surface `name` in all entity
  responses.
- `cmd/kanbanzai/` — Update `kbz init` to prompt for and accept a project name.
- Agent skill files — Add name guidance, rules, and examples to entity creation
  skill content.
- `.kbz/state/` backfill — Write conforming `name` values to all existing entity
  YAML files in this repository.

### 3.2 Out of scope

- Viewer implementation — the viewer will benefit from consistent `name` fields,
  but viewer changes are a separate feature.
- Name-based entity lookup or search — names are display strings; slug and ID
  remain the lookup keys.
- Name uniqueness enforcement — two entities may share a name; uniqueness is not
  a requirement.
- Changes to entity IDs, slugs, storage filenames, or any other identifier.
- Renaming `name` on the prefix entries within `config.yaml` (the `name` field
  on prefix records is a different, pre-existing field and is not affected).

---

## 4. Acceptance Criteria

### 4.1 Model — Name field

**AC-01.** Every entity struct (Plan, Epic, Feature, Task, Bug, Decision,
Incident) has a `Name` field of type string.

**AC-02.** The `Name` field is required — it is not marked optional and has no
default value.

**AC-03.** The `Title` field is removed from Plan, Epic, Bug, and Incident.
No other fields or behaviour of those types change.

**AC-04.** The `Label` field is removed from Feature and Task. No other fields
or behaviour of those types change.

**AC-05.** Feature, Task, and Decision gain a `Name` field. These types
previously had no dedicated display name field.

**AC-06.** In the canonical YAML field order for every entity type, `name`
appears immediately after `slug` and before `status`.

### 4.2 Config — Project name

**AC-07.** The project config struct has a `Name` field of type string.

**AC-08.** The `Name` field serialises as `name` in `config.yaml`, positioned
after `version` and before `prefixes` in the canonical field order.

**AC-09.** `kbz init` prompts the user for a project name during interactive
setup. The prompt is presented before the prefix configuration step.

**AC-10.** The default value offered in the `kbz init` name prompt is derived
from the name of the current working directory.

**AC-11.** `kbz init` accepts a `--name` flag that supplies the project name
non-interactively, skipping the interactive prompt for that field.

**AC-12.** An existing `config.yaml` that does not have a `name` field is read
without error; the field is treated as empty.

### 4.3 Validation

**AC-13.** A name that is empty, or that consists entirely of whitespace after
trimming, fails validation with an error.

**AC-14.** A name exceeding 60 characters fails validation with an error.

**AC-15.** Leading and trailing whitespace in a supplied name is silently
normalised (stripped) before the value is stored or validated against the length
limit.

**AC-16.** A name containing a colon character (`:`) fails validation with an
error.

**AC-17.** A name whose leading characters form a phase or version prefix — a
single uppercase letter immediately followed by one or more digits, then a
space, hyphen, or em-dash — fails validation with an error. Examples that must
be rejected: `"P4 Kanbanzai 2.0"`, `"P8 — decompose"`, `"P11 fresh install"`.

**AC-18.** All five validation rules (AC-13 through AC-17) apply uniformly to
the `name` field on every entity type and to the project `name` in `config.yaml`.

### 4.4 MCP tools — Entity create and update

**AC-19.** The `entity` tool's `create` action accepts a `name` parameter for
all entity types. The parameter is required for all entity types.

**AC-20.** The `entity` tool's `update` action accepts a `name` parameter for
all entity types. When supplied, the new value replaces the existing name and
is validated before being persisted.

**AC-21.** The `title` parameter is removed from the `entity` tool. `name`
replaces it for all entity types that previously used `title` (Plan, Epic, Bug,
Incident).

**AC-22.** The `label` parameter is removed from the `entity` tool. No
replacement parameter is added.

**AC-23.** `entity get` responses include the `name` field for all entity types.

### 4.5 MCP tools — List and status display

**AC-24.** `entity list` responses include `name` alongside the ID and slug in
every entity row.

**AC-25.** The `status` dashboard includes `name` in feature rows and in task
rows.

**AC-26.** The `label` filter parameter is removed from `entity list`. Filtering
by label is no longer supported.

### 4.6 Storage and serialisation

**AC-27.** An entity written with a `name` value, read back, and re-written
produces byte-identical YAML output (round-trip stable).

**AC-28.** The `name` field is always present in serialised entity YAML. It is
never omitted, unlike optional fields.

**AC-29.** During the backfill window, an existing entity YAML file that carries
a `title` field instead of `name` is read without error; the `title` value is
used as the name value.

**AC-30.** During the backfill window, Feature, Task, and Decision YAML files
that lack a `name` field entirely are read without a hard error; the absence
produces a validation warning, not a parse failure.

**AC-31.** After the backfill is complete and verified, the backward-
compatibility handling in AC-29 and AC-30 is removed.

### 4.7 Skill files

**AC-32.** The agent skill content for entity creation includes guidance on
writing entity names. The guidance covers all five hard validation rules
(AC-13 through AC-17) so that agents produce conforming names without relying
solely on error feedback.

**AC-33.** The skill guidance includes the soft rules from the design: target
approximately four words; no em-dashes or separator punctuation; name should not
be merely the slug capitalised; name should be self-contained and readable
without knowing the parent entity; name should not repeat hierarchy context
already encoded in the entity's parent field.

**AC-34.** The skill guidance includes at least three good examples and three
bad examples, each bad example annotated with the rule it violates.

### 4.8 Backfill

**AC-35.** After the backfill pass, every Plan, Epic, Feature, Task, Bug,
Decision, and Incident entity file in the `.kbz` store has a `name` field with
a non-empty value.

**AC-36.** Every `title` field in existing entity YAML files has been renamed
to `name`.

**AC-37.** All backfilled and renamed names conform to the validation rules in
§4.3 — short, no colon, no phase prefix, no more than 60 characters. Names that
previously violated these rules (e.g. Plan titles carrying phase prefixes or
colons) are rewritten to conform, not merely renamed.

**AC-38.** No Feature or Task entity YAML file in the `.kbz` store contains a
`label` field after the backfill pass. Since no entities carry a label value,
this requires no data rewriting — only verification.

---

## 5. Test Plan

| Test | What it covers |
|------|----------------|
| `TestNameRoundTrip_Plan` | Write Plan with name, read back, compare YAML |
| `TestNameRoundTrip_Feature` | Write Feature with name, read back, compare YAML |
| `TestNameRoundTrip_Task` | Write Task with name, read back, compare YAML |
| `TestNameRoundTrip_Decision` | Write Decision with name, read back, compare YAML |
| `TestNameRoundTrip_Bug` | Write Bug with name, read back, compare YAML |
| `TestNameRoundTrip_Incident` | Write Incident with name, read back, compare YAML |
| `TestNameFieldOrder` | `name` appears after `slug` and before `status` in serialised YAML for all entity types |
| `TestNameRequired` | Entity creation without a name returns a validation error |
| `TestNameEmpty` | Empty string and whitespace-only names fail validation |
| `TestNameTooLong` | Name exceeding 60 characters fails validation |
| `TestNameWhitespaceTrimmed` | Leading and trailing whitespace is stripped before storage |
| `TestNameNoColon` | Name containing `:` fails validation |
| `TestNameNoPhasePrefix` | Names matching the phase prefix pattern (`P4 ...`, `P8 — ...`) fail validation |
| `TestNamePhasePrefixBoundary` | Names that resemble but do not match the prefix pattern are accepted (e.g. two-letter prefix, no digit) |
| `TestLabelFieldAbsent_Feature` | Serialised Feature YAML never contains a `label` field |
| `TestLabelFieldAbsent_Task` | Serialised Task YAML never contains a `label` field |
| `TestBackwardCompat_TitleField` | Entity YAML with `title` instead of `name` is read without error; value is used as name |
| `TestBackwardCompat_MissingName` | Feature, Task, and Decision YAML without `name` produces a warning, not a parse error |
| `TestConfigNameField` | `config.yaml` with `name` field round-trips correctly |
| `TestConfigNameMissing` | `config.yaml` without `name` field loads without error |
| `TestInitNameFlag` | `kbz init --name "My Project"` sets project name in config without prompting |
| `TestInitNameDefault` | Default name offered during `kbz init` is derived from the working directory name |
| `TestEntityCreateName` | `entity create` with `name` parameter persists the name |
| `TestEntityUpdateName` | `entity update` with `name` parameter replaces the existing name |
| `TestEntityGetIncludesName` | `entity get` response includes `name` field for all entity types |
| `TestEntityListIncludesName` | `entity list` response rows include `name` for all entity types |
| `TestEntityToolNoTitleParam` | `entity` tool no longer accepts a `title` parameter |
| `TestEntityToolNoLabelParam` | `entity` tool no longer accepts a `label` parameter |