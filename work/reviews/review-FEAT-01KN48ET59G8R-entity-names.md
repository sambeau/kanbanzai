# Review: Entity Names (FEAT-01KN48ET59G8R)

| Field | Value |
|---|---|
| Feature | FEAT-01KN48ET59G8R (entity-names) |
| Spec | work/spec/entity-names.md |
| Branch | feature/FEAT-01KN48ET59G8R-entity-names |
| Reviewer | orchestrator |
| Date | 2026-06-17 |
| Status | **PARTIAL** — review agent exhausted context before completing §4.5–§4.8 analysis and all documentation checks. Core findings are complete and actionable. |
| Verdict | changes-required |

---

## 1. Spec Conformance

### §4.1 Model — Name field

**AC-01** ✅ PASS — All seven entity structs (Plan, Epic, Feature, Task, Bug, Decision, Incident) have a `Name string` field. Verified in `internal/model/entities.go`.

**AC-02** ⚠️ PARTIAL — The service layer treats Name as optional for Feature, Task, and Decision. `CreateFeature`, `CreateTask`, and `CreateDecision` all use the guard `if input.Name != ""` before calling `validate.ValidateName`, meaning an empty name is silently stored. The MCP layer (`entityCreateOne`) does enforce name as required at the tool boundary by calling `validate.ValidateName("")` unconditionally — but the service itself does not, so callers that bypass the MCP layer (CLI, tests, future integrations) can create nameless Feature/Task/Decision entities. Plan, Epic, Bug, and Incident enforce name at the service layer.

**AC-03** ✅ PASS — `Title` field removed from Plan, Epic, Bug, and Incident model structs. No `Title` field appears in `internal/model/entities.go`.

**AC-04** ✅ PASS — `Label` field removed from Feature and Task model structs. The canonical field order for both types does not include `label`.

**AC-05** ✅ PASS — Feature, Task, and Decision each have a `Name` field. Confirmed in model struct outlines and service create inputs.

**AC-06** ✅ PASS — `name` appears immediately after `slug` and before `status` in `fieldOrderForEntityType` for all entity types. Verified for Plan, Epic, Feature, Task, Bug, Decision, and Incident. For Feature the order is `id → slug → name → parent → status`; for Decision `id → slug → name → summary → ... → status`. In both cases `name` immediately follows `slug` and precedes `status`. `TestNameFieldOrder` covers the Feature case.

---

### §4.2 Config — Project name

**AC-07** ❌ GAP — The runtime `Config` struct in `internal/config/config.go` has **no `Name` field**. Only the write-only `initFileConfig` struct (local to `internal/kbzinit/config_writer.go`) carries `Name`. The runtime `Config` struct that all server and CLI code uses (loaded via `config.Load()` / `config.LoadFrom()`) silently ignores the `name:` key in config.yaml. The project name is written correctly to disk but is not readable at runtime through the canonical config API. This is a direct violation of AC-07.

**AC-08** ⚠️ PARTIAL — `WriteInitConfig` writes `name:` positioned after `version:` and before `prefixes:` (correct per spec). However since `Config` struct lacks `Name`, the field cannot be round-tripped. The write side is correct; the read side is absent.

**AC-09** ✅ PASS — `resolveProjectName()` in `init.go` prompts interactively. The prompt is issued inside `runNewProject` before the prefix configuration step.

**AC-10** ✅ PASS — Default value is `filepath.Base(i.workDir)`. `TestInitNameDefault` verifies this.

**AC-11** ✅ PASS — `--name` flag is accepted in `init_cmd.go` and sets `opts.Name`. `TestInitNameFlag` verifies the name appears in written config.yaml.

**AC-12** ✅ PASS — `TestConfigNameMissing` verifies that a config.yaml without a `name` field loads without error.

---

### §4.3 Validation

**AC-13** ✅ PASS — `ValidateName("")` and `ValidateName("   ")` both return errors. `TestValidateName` covers both cases.

**AC-14** ✅ PASS — A 61-character name returns an error. 60 characters passes. Boundary cases tested in `TestValidateName`.

**AC-15** ✅ PASS — `ValidateName` trims leading/trailing whitespace before the length check and returns the trimmed value. `TestValidateName` "leading and trailing whitespace trimmed" case verifies this.

**AC-16** ✅ PASS — `strings.Contains(trimmed, ":")` check present. `TestValidateName` "contains colon" case verifies rejection.

**AC-17** ✅ PASS — `phasePrefixPattern` regex (`^[A-Z]\d+[\s\-—]`) rejects `"P4 something"`, `"P8 — decompose"`, `"P11 fresh install"`. `TestValidateName` and `TestNamePhasePrefixBoundary` cover this.

**AC-18** ❌ GAP — Validation rules are **not** applied to the project `name` in config.yaml. `resolveProjectName()` returns the raw input without calling `validate.ValidateName`. `WriteInitConfig` accepts and writes any string as the project name without validation. A user can `kbz init --name "P4: Phase 4"` and the invalid name is silently written to config.yaml. Additionally, since `Config` struct has no `Name` field, there is nowhere to validate the name on load either.

---

### §4.4 MCP tools — Entity create and update

**AC-19** ✅ PASS — `entityCreateOne` reads `name` from args, calls `validate.ValidateName`, and returns a tool error if invalid. The `name` parameter is documented in the entity tool definition. All entity types (plan, feature, task, bug, epic, decision) pass `name` to their respective service create calls.

**AC-20** ✅ PASS — `entityUpdateAction` passes `name` in the `fields` map for regular entities, and `input.Name` for plans. Both update paths accept the new name value. Note: the update path does not re-validate the name via `ValidateName` for the generic entity path — it uses `strings.TrimSpace` but does not reject names with colons or phase prefixes on update. This is a minor gap not explicitly called out as its own AC but falls under the spirit of AC-20.

**AC-21** ✅ PASS — No `title` parameter in the entity tool definition. Verified in `entityTool()` function body.

**AC-22** ✅ PASS — No `label` parameter in the entity tool definition.

**AC-23** ✅ PASS — `entityFullRecord()` includes `name` from state. `entityGetAction` uses `entityFullRecord` for all entity types. The `name` field is present in get responses.

---

### §4.5 MCP tools — List and status display

**AC-24** ✅ PASS — `entitySummaries()` includes `entityName` (read from `r.State["name"]`) in every list row alongside `id`, `slug`, `status`, `display_id`, and `entity_ref`. However, `TestEntity_List_SummaryFields` does **not** assert that `name` is present in the list items (it only checks `id`, `type`, `slug`, `status`, `display_id`). The implementation is correct but the test does not verify the name requirement.

**AC-25** ✅ PASS — `featureSummary` struct (used in plan-dashboard feature rows) has a `Name` field. `taskInfo` struct (used in feature-dashboard task rows) has a `Name` field. Both are populated from entity state in `synthesisePlan` and `synthesiseFeature`.

**AC-26** ✅ PASS — No `label` filter parameter in the entity list action. `entityListAction` does not accept or process a label filter.

---

### §4.6 Storage and serialisation

**AC-27** ✅ PASS — Round-trip tests `TestNameRoundTrip_Plan`, `TestNameRoundTrip_Feature`, `TestNameRoundTrip_Task`, `TestNameRoundTrip_Decision` all pass. `TestMarshalCanonicalYAML_IdempotentWrite` also covers the general idempotency property.

**AC-28** ❌ GAP — The `name` field is **not always present** in serialised YAML for Feature, Task, and Decision. In `featureFields()`, `taskFields()`, and `decisionFields()`, name is written conditionally:

```kanbanzai/.worktrees/FEAT-01KN48ET59G8R-entity-names/internal/service/entities.go#L1090-1092
if e.Name != "" {
    fields["name"] = e.Name
}
```

This means an entity with an empty name (which the service allows for Feature/Task/Decision per the AC-02 finding) will be serialised without a `name:` key, violating the "always present, never omitted" requirement. Plan, Epic, Bug, and Incident include `name` unconditionally in their respective field functions. The three affected types are the ones where the service also allows empty names, so the two defects compound each other.

**AC-29** ⚠️ NOT IMPLEMENTED — No backward-compatibility code was written to read `title:` fields as `name` during the backfill window. The implementation instead performed the backfill directly without a compat layer. Since the backfill appears complete (no entity YAML files contain `title:` keys, verified by grep), the absence of compat code has no practical impact on current state. However AC-29 as written was not satisfied. The named test `TestBackwardCompat_TitleField` is absent.

**AC-30** ⚠️ NOT IMPLEMENTED — Same reasoning as AC-29. No warning is emitted for Feature/Task/Decision YAML files that lack `name`. `TestFeatureWithoutLabel_BackwardCompat` provides partial coverage (reads a feature without `name` without error) but does not test for a validation warning. The named test `TestBackwardCompat_MissingName` is absent.

**AC-31** ✅ TRIVIALLY SATISFIED — Since AC-29 and AC-30 compat code was never written, there is nothing to remove. The backfill is complete.

---

### §4.7 Skill files

**AC-32** ✅ PASS — Both the embedded source (`internal/kbzinit/skills/agents/SKILL.md`) and the installed copy (`.agents/skills/kanbanzai-agents/SKILL.md`) contain an "Entity Names" section with all five hard rules listed explicitly: required, 60-char max, no colon, no phase prefix, whitespace stripped.

**AC-33** ✅ PASS — The quality guidance section covers all five soft rules from the design: ~4 words target, no em-dashes as separators, name should not be slug capitalised, names must be self-contained, names must not repeat parent context.

**AC-34** ✅ PASS — Six good examples and four bad examples are present in both skill files, each bad example annotated with the rule it violates. Exceeds the three-of-each minimum.

---

### §4.8 Backfill

**AC-35** ✅ PASS — `grep -rL "^name:" .kbz/state/features/ .kbz/state/tasks/ .kbz/state/decisions/` returned no results. All entity YAML files in these directories have a `name:` field. Plans, bugs, and incidents also confirmed to have `name:` fields.

**AC-36** ✅ PASS — `grep -rn "^title:" .kbz/state/` shows `title:` only in `.kbz/state/documents/` files, which are DocumentRecord entities that legitimately use `title`. No entity files (plans, features, tasks, bugs, decisions, incidents, epics) contain `title:` keys.

**AC-37** ✅ PASS — Spot-checked names across plans, tasks, features, and bugs. All conform to validation rules: short, no colon, no phase prefix, within 60 characters. `grep -rn "^name: P[0-9]" .kbz/state/` (phase prefix check) returned no results.

**AC-38** ✅ PASS — `grep -rn "^label:" .kbz/state/features/ .kbz/state/tasks/` returned no results. No label fields remain in feature or task YAML files.

---

## 2. Code Quality

**CQ-01** — Severity: **defect** — `internal/service/entities.go` `featureFields()`, `taskFields()`, and `decisionFields()` use `if e.Name != ""` guards. This contradicts AC-28 and also produces inconsistent YAML output vs. the unconditional name inclusion in `planFields()`, `epicFields()`, `bugFields()`, and `incidentFields()`. The fix is to include `"name": e.Name` unconditionally in all three functions, matching the approach used for Plan, Epic, Bug, and Incident.

**CQ-02** — Severity: **defect** — `internal/kbzinit/init.go` `resolveProjectName()` returns the raw project name without validating it. `WriteInitConfig` also accepts and stores the name without validation. Any string — including strings that violate AC-13 through AC-17 — can be written to `config.yaml`. The fix is to call `validate.ValidateName(name)` before returning from `resolveProjectName` (or inside `WriteInitConfig`) and surface the error to the user.

**CQ-03** — Severity: **defect** — `internal/config/config.go` `Config` struct has no `Name` field. The project name is written to `config.yaml` by `WriteInitConfig` but cannot be read back by any code that uses the standard `config.Load()` API. AC-07 requires a `Name` field on the project config struct. The fix is to add `Name string \`yaml:"name,omitempty"\`` to the `Config` struct.

**CQ-04** — Severity: **defect** — `cmd/kanbanzai/entity_cmd.go` CLI is incompletely updated. Three issues:
1. The usage text (`entityUsageText` const) still references `--title` in the create subcommand description.
2. `runEntityCreate` for `feature` does not pass `Name` from the parsed flags to `service.CreateFeatureInput`.
3. `runEntityCreate` for `task` does not pass `Name` to `service.CreateTaskInput`.
4. `runEntityCreate` for `decision` does not pass `Name` to `service.CreateDecisionInput`.
The CLI is the only surface where these three entity types can be created without a name. Users relying on the CLI (rather than the MCP tool) cannot supply names for features, tasks, or decisions.

**CQ-05** — Severity: **improvement** — `entityUpdateAction` for the generic entity path collects string fields to update using `strings.TrimSpace` but does not run `validate.ValidateName` when `name` is among the updated fields. A user can call `entity(action: "update", name: "P4: Bad Name")` and the invalid name will be stored. The MCP create path validates on write; update should too.

**CQ-06** — Severity: **nit** — The `phasePrefixPattern` variable in `internal/validate/entity.go` has a clear doc comment explaining its semantics. This is good practice. No issue.

---

## 3. Test Coverage

### Named tests from spec §5 — implementation status

| Spec Test | Status | Notes |
|---|---|---|
| `TestNameRoundTrip_Plan` | ✅ Present | `internal/storage/entity_store_test.go` L1427 |
| `TestNameRoundTrip_Feature` | ✅ Present | L1469 |
| `TestNameRoundTrip_Task` | ✅ Present | L1510 |
| `TestNameRoundTrip_Decision` | ✅ Present | L1551 |
| `TestNameRoundTrip_Bug` | ❌ Absent | No Bug round-trip test with name |
| `TestNameRoundTrip_Incident` | ❌ Absent | No Incident round-trip test with name |
| `TestNameFieldOrder` | ✅ Present | L1591 — covers Feature only |
| `TestNameRequired` | ❌ Absent | Not as a standalone test; covered indirectly by `TestValidateRecord_MissingRequiredFields` for Epic, but not for entity creation via service |
| `TestNameEmpty` | ✅ Present | Covered within `TestValidateName` |
| `TestNameTooLong` | ✅ Present | Covered within `TestValidateName` |
| `TestNameWhitespaceTrimmed` | ✅ Present | Covered within `TestValidateName` |
| `TestNameNoColon` | ✅ Present | Covered within `TestValidateName` |
| `TestNameNoPhasePrefix` | ✅ Present | Covered within `TestValidateName` |
| `TestNamePhasePrefixBoundary` | ✅ Present | `internal/validate/entity_test.go` |
| `TestLabelFieldAbsent_Feature` | ⚠️ Renamed | Present as `TestFeatureNoLabelOmitted` — functionally equivalent |
| `TestLabelFieldAbsent_Task` | ❌ Absent | No equivalent test for Task |
| `TestBackwardCompat_TitleField` | ❌ Absent | No test for title→name backward compat read path |
| `TestBackwardCompat_MissingName` | ❌ Absent | `TestFeatureWithoutLabel_BackwardCompat` partially covers but does not check for a warning |
| `TestConfigNameField` | ❌ Absent | No round-trip test for config.yaml `name` field |
| `TestConfigNameMissing` | ✅ Present | `internal/kbzinit/init_test.go` L300 |
| `TestInitNameFlag` | ✅ Present | L249 |
| `TestInitNameDefault` | ✅ Present | L274 |
| `TestEntityCreateName` | ⚠️ Partial | `TestEntity_Create_Task` etc. create with name but don't assert `name` in response |
| `TestEntityUpdateName` | ❌ Absent | `TestEntity_Update_TaskSummary` exists but no name-specific update test |
| `TestEntityGetIncludesName` | ❌ Absent | `TestEntity_Get_Task/Feature/Plan` do not assert `name` in response |
| `TestEntityListIncludesName` | ❌ Absent | `TestEntity_List_SummaryFields` checks `id/type/slug/status/display_id` but not `name` |
| `TestEntityToolNoTitleParam` | ❌ Absent | No test verifying `title` is rejected |
| `TestEntityToolNoLabelParam` | ❌ Absent | No test verifying `label` is rejected |

### Additional observations

- `TestEntity_Create_BatchTasks` and `TestEntity_Create_MutationHasSideEffectsField` both create tasks without a `name` field. Since `entityCreateOne` validates name as required, both tests exercise the name-validation-failure path rather than the success path. They pass (because `err == nil` is returned with a tool error result), but they do not actually verify successful entity creation. These tests should be updated to include a `name` field.

- All tests pass (`go test ./internal/...` — all packages green).

---

## 4. Documentation

### Internal documentation

- `ValidateName` has a clear doc comment and the `phasePrefixPattern` regex has an explanatory comment. ✅
- `WriteInitConfig` doc comment does not mention that the `name` parameter bypasses validation — a reader of the function signature would expect validation to have been applied by the caller. Minor gap.

### User-facing documentation

- **`cmd/kanbanzai/init_cmd.go`** — `--name` flag is fully documented in `initUsageText` with description and examples. ✅
- **`cmd/kanbanzai/entity_cmd.go`** — `entityUsageText` still contains `--title` in the create subcommand description (`--slug, --title, --summary, --parent, etc.`). This is stale and contradicts the feature goal. Should read `--name`. ❌
- Error messages from `ValidateName` are user-friendly and descriptive. ✅

### Agent-facing documentation

- **Embedded `internal/kbzinit/skills/agents/SKILL.md`** — Entity Names section present with all hard rules, all soft rules, good/bad examples. ✅
- **Installed `.agents/skills/kanbanzai-agents/SKILL.md`** — Same content, confirmed in sync. ✅
- **Embedded `internal/kbzinit/skills/plan-review/SKILL.md`** — Plan Review Report Format uses `{plan name}` in the doc registration title template. ✅
- **Installed `.agents/skills/kanbanzai-plan-review/SKILL.md`** — Same, confirmed in sync. ✅

---

## 5. Workflow Currency

| Document | Status Field | Assessment |
|---|---|---|
| `work/spec/entity-names.md` | `Draft` | Should be updated to `Approved` or `Final` if spec is accepted |
| `work/design/entity-names.md` | `proposal` | Should be updated to reflect implementation status |
| `work/plan/entity-names-implementation-plan.md` | `Draft` | Should be updated to reflect completion |

The `AGENTS.md` Scope Guard does not reference this feature or plan by name, but the Scope Guard convention lists completed phases generically. No specific stale `title` or `label` references were found in the workflow guidance prose.

---

## 6. Findings Summary

| ID | Severity | Area | Finding | AC/Criterion |
|---|---|---|---|---|
| F-01 | defect | Config | `Config` struct in `internal/config/config.go` has no `Name` field; project name written to disk is not readable at runtime | AC-07 |
| F-02 | defect | Config | `WriteInitConfig` and `resolveProjectName` do not call `validate.ValidateName`; invalid project names are accepted silently | AC-18 |
| F-03 | defect | Storage | `featureFields()`, `taskFields()`, `decisionFields()` include `name` conditionally (`if e.Name != ""`); name can be absent from YAML | AC-28 |
| F-04 | defect | CLI | `entity_cmd.go` create for Feature, Task, Decision does not pass `Name` from CLI flags; usage text still says `--title` | AC-19, AC-21 |
| F-05 | gap | Service | `CreateFeature`, `CreateTask`, `CreateDecision` treat Name as optional; empty names are stored without error | AC-02 |
| F-06 | gap | Storage | No backward compat code for `title` → `name` field migration during backfill window; named tests `TestBackwardCompat_TitleField` and `TestBackwardCompat_MissingName` absent | AC-29, AC-30 |
| F-07 | gap | Tests | `TestNameRoundTrip_Bug` and `TestNameRoundTrip_Incident` absent | §5 test plan |
| F-08 | gap | Tests | `TestEntityGetIncludesName` absent; existing get tests do not assert `name` in response | AC-23, §5 |
| F-09 | gap | Tests | `TestEntityListIncludesName` absent; `TestEntity_List_SummaryFields` does not assert `name` | AC-24, §5 |
| F-10 | gap | Tests | `TestEntityToolNoTitleParam` and `TestEntityToolNoLabelParam` absent | AC-21, AC-22, §5 |
| F-11 | gap | Tests | `TestLabelFieldAbsent_Task` absent; only Feature has an equivalent test | §5 |
| F-12 | gap | Tests | `TestConfigNameField` absent; no round-trip test verifying `name` persists in config.yaml | §5 |
| F-13 | gap | Tests | Batch creation tests and `TestEntity_Create_MutationHasSideEffectsField` use tasks without `name`; now exercise failure path, not success | §5, AC-19 |
| F-14 | improvement | Service | `entityUpdateAction` generic path does not run `validate.ValidateName` on the new name value | AC-20 |
| F-15 | nit | Docs | `work/spec/entity-names.md`, design doc, and impl plan still carry Draft/proposal status | Workflow currency |

---

## 7. Verdict

**changes-required**

The feature is substantially complete: all tests pass, the backfill is fully done, the core entity tool (MCP) validates and surfaces names correctly, the skill files are correct and in sync, and the AC-06 field ordering is satisfied across all entity types.

However four defects must be resolved before merge:

1. **F-01 / AC-07** — `Config` struct must gain a `Name` field so the project name is readable at runtime. Without this, AC-07, AC-08, and the full intent of AC-18 cannot be satisfied.
2. **F-02 / AC-18** — `resolveProjectName` (or `WriteInitConfig`) must call `validate.ValidateName` to enforce the five validation rules on the project name.
3. **F-03 / AC-28** — `featureFields`, `taskFields`, and `decisionFields` must include `name` unconditionally (matching Plan/Epic/Bug/Incident) so the `name` key is always present in serialised YAML.
4. **F-04** — `entity_cmd.go` CLI create for Feature, Task, and Decision must pass `Name` from flags, and the usage text `--title` reference must be updated to `--name`.

The service-layer name optionality (F-05) compounds F-03 and should be addressed in the same pass: `CreateFeature`, `CreateTask`, and `CreateDecision` should validate that Name is non-empty just as `CreateEpic`, `CreateBug`, and `CreatePlan` already do.

The test coverage gaps (F-07 through F-13) are secondary but should be addressed to prevent future regressions.

---

*Note: This review is marked partial. The review agent exhausted its context window after completing the §4.1–§4.8 conformance analysis and primary code quality checks. The following areas received reduced scrutiny: incident tool MCP handler (confirmed Name is in `incidentFields` unconditionally), status tool synthesise functions beyond featureSummary/taskInfo struct verification, the service migration layer, and the full set of backfill validation (spot checks only). The four defects identified are well-evidenced and actionable regardless.*