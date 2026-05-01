# Batch Conformance Review: B36-kbz-cli-and-status

## Scope

- **Batch:** B36-kbz-cli-and-status (kbz CLI and Status Command)
- **Features:** 4 total (0 done, 0 cancelled/superseded, **4 incomplete***)
- **Review date:** 2026-04-30 (first pass: lifecycle); 2026-05-01 (second pass: code audit)
- **Reviewer:** reviewer-conformance

> *All 4 features are stuck in non-terminal lifecycle states (`developing`/`dev-planning`)
> despite 19/19 tasks being done. This is a lifecycle advancement gap, not an
> implementation gap. See **Conformance Gap CG-1**.

## Feature Census

| Feature | Status | Tasks | Spec Approved | Dev-Plan Approved | Design Approved | Notes |
|---------|--------|-------|---------------|-------------------|-----------------|-------|
| F1: Binary rename kanbanzai → kbz | **dev-planning** | 5/5 done ✅ | ✅ | ✅ | ✅ | Branch already merged to main. Worktree stale. |
| F2: Status argument resolution | **developing** | 6/6 done ✅ | ✅ | ✅ | ✅ | Branch has merge conflicts with main |
| F3: Status human output | **developing** | 4/4 done ✅ | ✅ | ✅ | ✅ | No merge conflicts |
| F4: Status machine output | **developing** | 4/4 done ✅ | ✅ | ✅ | ✅ | Branch has merge conflicts with main |

## Conformance Gaps

| # | Feature | Type | Description | Severity |
|---|---------|------|-------------|----------|
| CG-1 | All 4 | lifecycle | Features are in non-terminal states (`dev-planning`/`developing`) despite all 19 child tasks being done. The health check confirms `feature_child_consistency` warnings for all 4 features. Features must be advanced through their lifecycle to `done` before the batch can be conformance-passed. | **blocking** |
| CG-2 | All 4 | worktree-cleanup | All 4 worktrees are still `active`. F1's branch is already merged to main (38 commits behind, 0 ahead — unique commits absorbed). F2 and F4 branches have merge conflicts with main. Worktrees should be cleaned up post-merge and conflicts resolved for unmerged branches. | non-blocking |
| CG-3 | B36 | doc-currency | AGENTS.md line 78 still reads `├── cmd/kanbanzai/         ← binary entry point (CLI and MCP server)` — stale reference to the old directory name after the rename. Should be `cmd/kbz/`. | non-blocking |
| CG-4 | B36 | knowledge | No knowledge entries were contributed during the batch. Retrospective synthesis returned zero signals (no `finish(retrospective: [...])` calls were made during task completion). | non-blocking |
| CG-5 | F2 | routing | `ResolveNone` path in `runStatus` immediately errors instead of probing entity-ID lookup then file-path lookup as required by FR-007. Comment in the code reads "try entity first, then path" but the probe is absent. AC-007 cannot pass. | **blocking** |
| CG-6 | F2 | exit-codes | `runStatusEntity` and `runStatusPlanPrefix` return `nil` (exit 0) on not-found. Spec requires exit 1 for unknown entity IDs (FR-016) and unknown plan prefixes (FR-008). AC-006 and AC-013 both fail. | **blocking** |
| CG-7 | F3 | spec-conformance | Unregistered-document human output format violates three requirements: line 1 is `File: {path}` not the bare path (FR-2.1); the phrase `Not registered with Kanbanzai.` (verbatim, FR-2.2) is absent; the suggested command omits `--type` and `--title` placeholders (FR-2.3). | **blocking** |
| CG-8 | F4 | json-schema | Three schema violations in the feature JSON result: (a) `documents.dev-plan` key is emitted as `dev_plan` (underscore) not `dev-plan` (hyphen), breaking AC-5; (b) `feature.display_id` field is absent, violating FR-9.1 and A-2; (c) document slot `id` is populated with the file path instead of the document record ID. | **blocking** |
| CG-9 | F4 | json-schema | Bug `parent_feature_id` is hardcoded to `null` in `JSONRenderer.RenderBug`; plain renderer hardcodes `parent_feature: missing`. The `parentFeature` value extracted in `runStatusBugFormatted` is never forwarded to either renderer. Affects any bug with a parent feature. (FR-9.4, FR-5) | **blocking** |

## Documentation Currency

- **AGENTS.md:** Needs one minor update — line 78 references `cmd/kanbanzai/` instead of `cmd/kbz/`.
- **Workflow skills:** Not directly affected by B36 — pre-existing stale tool references (not B36-specific).
- **Knowledge entries:** 0 contributed. The implementation team should consider contributing knowledge about the resolution disambiguation logic, TTY detection patterns, and JSON/plain renderer architecture for future reference.
- **Scope Guard:** B36 is not listed in AGENTS.md Scope Guard — this is a pre-existing condition shared by 30+ other completed batches.

## Retrospective Summary

No retrospective signals were recorded during B36 implementation. The `retro(action: "synthesise", scope: "B36-kbz-cli-and-status")` call returned zero themes. Retrospective observations should be contributed during task completion via `finish(retrospective: [...])` in future batches.

## Implementation Verification

### F1: Binary rename
- `cmd/kanbanzai/` removed ✅, `cmd/kbz/` exists ✅
- `Makefile` updated: `BINARY := kbz`, `go build ./cmd/kbz` ✅
- `mcpVersion` bumped to `2` ✅
- Migration detection in `kbz init` ✅
- Protocol name `"kanbanzai"` preserved in MCP config keys ✅
- All 15 acceptance criteria verified by tasks ✅
- Test suite passes (`go test ./cmd/kbz/...` ✅, `go vet` clean ✅)
- **Verdict: pass** — no code-level findings.

### F2: Argument resolution
- `internal/resolution/` package with lexical `Disambiguate()` implementing NFR-001 (no I/O before decision) ✅
- Four resolution kinds: `ResolvePath`, `ResolveEntity`, `ResolvePlanPrefix`, `ResolveNone` ✅
- `--format`/`-f` flag with validation and exit-2 usage errors ✅
- `kbz doc approve` path resolution via `resolveDocApproveTarget` ✅
- Unit test suite (21 tests) covering all disambiguation paths ✅
- **Verdict: fail** — see CG-5 (FR-007 not implemented) and CG-6 (exit codes); see code review §F2 below.

### F3: Human output
- `internal/cli/render/` package with injectable `TTYDetector` interface ✅
- `golang.org/x/term` for real TTY detection; `StaticTTY` for test injection ✅
- Symbol table: Unicode ↔ ASCII fallback mapping implemented ✅
- Colour table: green/yellow/red/default ANSI codes ✅
- Feature, plan, and project views implemented and tested ✅
- `AlignDocuments` helper aligns document rows to a common column width ✅
- **Verdict: fail** — see CG-7 (unregistered doc format); see code review §F3 below.

### F4: Machine output
- `internal/cli/status/` with `PlainRenderer` and `JSONRenderer` types ✅
- `PlainRenderer` covers all 6 scope types (feature, plan, task, bug, document, project) ✅
- `JSONRenderer` wraps entity/doc results in `{"results":[...]}` (D-7) ✅
- Project overview uses distinct top-level shape with `"scope":"project"` (D-8) ✅
- No test files in `internal/cli/status/` ❌
- **Verdict: fail** — see CG-8 (JSON schema violations) and CG-9 (bug parent); see code review §F4 below.

---

## Code Conformance Review

> **Added:** Full code audit against approved specifications (F2 spec `B36-F2-spec-status-argument-resolution.md`, F3 spec `B36-F3-spec-status-human-output.md`, F4 spec `B36-F4-spec-status-machine-output.md`).

### Review Unit: F2 — Status argument resolution

```
Files: internal/resolution/resolution.go, cmd/kbz/workflow_cmd.go (runStatus, runStatusEntity,
       runStatusPlanPrefix, runStatusPath), cmd/kbz/doc_cmd.go (resolveDocApproveTarget),
       internal/resolution/resolution_test.go
Spec:  work/spec/B36-F2-spec-status-argument-resolution.md
Reviewer Role: reviewer-conformance

Overall: needs_remediation

Dimensions:
  spec_conformance: fail
    Evidence:
      - FR-003 to FR-004: flag parsing, usage errors, multi-arg rejection all implemented correctly
        (workflow_cmd.go L41-130).
      - FR-005 to FR-006: disambiguation via Disambiguate() correctly classifies paths (slash/
        .md/.txt) and entity IDs (display format, TSID, batch/plan slugs) before any I/O.
      - FR-021 to FR-024: kbz doc approve path resolution implemented and correct
        (doc_cmd.go resolveDocApproveTarget).
    Findings:
      - [blocking] FR-007 not implemented: the ResolveNone branch in runStatus returns an error
        immediately without probing entity-ID lookup then file-path lookup as required.
        The code comment reads "try entity first, then path, then give up" but only the
        give-up step was implemented. AC-007 cannot pass.
        (spec: FR-007, AC-007, location: cmd/kbz/workflow_cmd.go ~L130-134)
      - [blocking] FR-008 exit code 1 on plan-not-found not implemented: runStatusPlanPrefix
        returns nil (exit 0) when GetPlan returns an error, with an inline comment
        "Plan not found — informational, exit 0". Spec requires exit 1.
        AC-006 fails.
        (spec: FR-008, AC-006, location: cmd/kbz/workflow_cmd.go ~L435-439)
      - [blocking] FR-016 exit code 1 on entity-not-found not implemented: runStatusEntity
        returns nil (exit 0) when entitySvc.Get fails. Spec requires exit 1 with a
        descriptive error message containing the ID. AC-013 fails.
        (spec: FR-016, AC-013, location: cmd/kbz/workflow_cmd.go ~L230-238)
      - [non-blocking] ASM-002 partial match: barePlanPrefixRE uses `^[A-Z][0-9]+$` (1 uppercase
        letter) but ASM-002 states the pattern allows 1 or 2 uppercase letters. Two-letter
        batch prefixes (e.g. "BB7") would not be recognised as plan prefixes.
        Recommendation: change regex to `^[A-Z]{1,2}[0-9]+$`.
        (location: internal/resolution/resolution.go L63)

  implementation_quality: pass_with_notes
    Evidence:
      - NFR-001: Disambiguate executes with zero I/O before returning — verified by reading the
        function body (no file/network calls, only regexp and string ops).
      - Disambiguation order follows spec rule sequence: path → entity ID → plan prefix → none.
      - resolveDocApproveTarget handles both path and non-path inputs cleanly.
    Findings:
      - [non-blocking] runStatusEntity and runStatusPlanPrefix both create a new entitySvc
        bound to core.StatePath(); a reference is available from the deps parameter, creating
        a minor inconsistency in how the service is obtained across code paths.
        Recommendation: pass the service in or initialise it once per command invocation.
        (location: cmd/kbz/workflow_cmd.go ~L223, ~L432)

  test_adequacy: pass_with_notes
    Evidence:
      - 21 unit tests in resolution_test.go cover all four ResolutionKind outcomes, edge cases
        (empty string, leading dot-slash, path-overrides-entity, case-insensitive prefixes).
      - String() method coverage included.
    Findings:
      - [non-blocking] No integration tests for the exit-code contract (AC-006, AC-013, AC-007).
        The unit tests cover disambiguation in isolation but not the full runStatus call chain.
        Recommendation: add integration tests in cmd/kbz/workflow_cmd_test.go asserting exit
        codes for the not-found and ResolveNone cases.

Finding Summary:
  Blocking: 3
  Non-blocking: 2
  Total: 5
```

---

### Review Unit: F3 — Status command human output

```
Files: internal/cli/render/renderer.go, tty.go, tty_term.go, symbols.go, colour.go,
       types.go, alignment.go, feature.go, plan.go, project.go,
       alignment_test.go, feature_test.go, plan_test.go, project_test.go,
       cmd/kbz/workflow_cmd.go (runStatusPath human branch)
Spec:  work/spec/B36-F3-spec-status-human-output.md
Reviewer Role: reviewer-conformance

Overall: needs_remediation

Dimensions:
  spec_conformance: fail
    Evidence:
      - FR-1 (TTY detection): IsTTY() uses golang.org/x/term via termTTY; StaticTTY for
        injection; NFR-4 injectable interface satisfied.
      - FR-1.2/1.3: Symbol() returns Unicode or ASCII fallbacks from ttySymbols/asciiSymbols
        maps. All six required symbols present and correctly mapped.
      - FR-4 (direct feature lookup): header, plan line, documents block, tasks summary,
        attention block all implemented. Edge cases: no-plan omits Plan line (FR-4.8); all-
        missing docs render all three rows (FR-4.6); zero tasks renders 0 counts (FR-4.7).
      - FR-5 (plan lookup): header, Features block, tasks aggregate, attention block present.
      - FR-6 (project overview): Plans block, Health line, Attention block (omitted when empty,
        FR-6.7), Work queue line all present.
      - FR-7.1: AlignDocuments() computes column width from longest label within the documents
        block.
    Findings:
      - [blocking] FR-2.1 violated: when a file is unregistered, line 1 of output is
        `File: {normalised-path}` not the file path as supplied by the user. The `./` prefix
        is stripped (normalised) before rendering (FR-2.1 says "as supplied by the user, not
        normalised"). Additionally the `File:` prefix is not in the spec.
        (spec: FR-2.1, AC-2, location: cmd/kbz/workflow_cmd.go ~L571)
      - [blocking] FR-2.2 violated: the required verbatim line `Not registered with Kanbanzai.`
        is absent; the implementation outputs `Status: not registered` instead.
        (spec: FR-2.2, AC-2, location: cmd/kbz/workflow_cmd.go ~L571-574)
      - [blocking] FR-2.3 violated: the suggested `kbz doc register` command omits `--type`
        and `--title` placeholders. Output is `kbz doc register <path>` with no flag
        arguments, violating the MUST requirement.
        (spec: FR-2.3, AC-2, location: cmd/kbz/workflow_cmd.go ~L574)
      - [non-blocking] FR-3.1 violated: the registered-document human block starts with
        `Document: <id>` followed by `ID:`, `Type:`, `Title:`, `Status:`, `Path:` keys. Spec
        says line 1 must be the document file path (not the ID), then indented `Title:`,
        `Type:`, `Status:` in that order. The `ID:` field is also not in the spec layout.
        Recommendation: restructure to output path on line 1, then aligned Title/Type/Status.
        (spec: FR-3.1, AC-3, location: cmd/kbz/workflow_cmd.go ~L592-597)
      - [non-blocking] FR-2/F2 spec conflict — exit code for file-not-found:
        F3 FR-2.5 says output a message and exit 0; F2 FR-009 says exit 1; F4 A-5 sides with
        exit non-zero. The implementation returns an error (exit 1), following F2/F4. The
        conflict should be resolved by updating F3 FR-2.5 to align with F2 FR-009.
        (spec: F3 FR-2.5 vs F2 FR-009, location: cmd/kbz/workflow_cmd.go ~L554-556)
      - [non-blocking] FR-5.3 not fully satisfied: plan feature rows use fixed double-space
        separators (`%s  %s  %s`) rather than computed column widths. Rows with varying
        display-ID lengths will not align at the slug or status columns.
        Recommendation: pass feature rows through AlignDocuments or a similar helper before
        writing them.
        (spec: FR-5.3, location: internal/cli/render/plan.go ~L23)
      - [non-blocking] FR-6.2 not fully satisfied: project overview plan rows omit the plan
        slug. Output is `{displayID}  {statusIcon}  {featDesc}`; spec requires display ID,
        slug, status symbol, status word, and activity summary.
        Recommendation: add `p.Slug` or `p.Name` to the plan row format string.
        (spec: FR-6.2, location: internal/cli/render/project.go ~L26)

  implementation_quality: pass
    Evidence:
      - Renderer is stateless and read-only (NFR-2 satisfied).
      - NFR-3 (no panics on nil/empty): feature with nil plan, nil documents, nil attention
        all handled gracefully (verified by feature_test.go edge-case tests).
      - colour.go Red/Yellow/Green functions short-circuit to identity when tty=false.

  test_adequacy: pass_with_notes
    Evidence:
      - feature_test.go: 9 subtests covering TTY/non-TTY, missing docs, zero tasks,
        attention items, no-plan edge case.
      - plan_test.go and project_test.go: coverage exists (files present).
      - alignment_test.go present.
    Findings:
      - [non-blocking] No test for the unregistered-document or registered-document human
        rendering path (the rendering lives in runStatusPath in workflow_cmd.go, not in the
        render package, and is therefore not covered by the render package unit tests).
        Recommendation: extract the unregistered-doc and registered-doc rendering into
        render.Renderer methods so they can be unit-tested like the feature/plan/project views.

Finding Summary:
  Blocking: 3
  Non-blocking: 4
  Total: 7
```

---

### Review Unit: F4 — Status command machine output

```
Files: internal/cli/status/plain.go, internal/cli/status/json.go,
       internal/cli/render/types.go (DocInput — root cause for one finding),
       cmd/kbz/workflow_cmd.go (runStatusBugFormatted)
Spec:  work/spec/B36-F4-spec-status-machine-output.md
Reviewer Role: reviewer-conformance

Overall: needs_remediation

Dimensions:
  spec_conformance: fail
    Evidence:
      - FR-2 (plain general rules): key:value pairs, lowercase keys, no whitespace, `scope`
        first — all satisfied across all six PlainRenderer methods.
      - FR-3 (plain feature): all 16 required keys present in RenderFeature, including
        `plan: missing` when PlanID is empty (FR-3.1) and `attention: none` when empty (FR-3.4).
      - FR-8 (JSON general rules): RFC 8259 validity (json.Marshal), single top-level object,
        snake_case keys — satisfied.
      - FR-9.2 (plan JSON): results array, scope, plan object, features counts, attention array
        — all correct.
      - FR-9.3 (task JSON): results array, scope, task object with four fields — correct.
      - FR-9.5/9.6 (document JSON): registered boolean, null id/type/status for unregistered,
        attention array with warning for unregistered — correct.
      - FR-10 (project JSON): distinct top-level shape, scope:project, plans array, health
        object, attention array — correct.
    Findings:
      - [blocking] FR-9.1 schema violation — `documents.dev-plan` key: jsonDocs struct uses
        Go field `DevPlan` with JSON tag `json:"dev_plan"` (underscore). The spec schema and
        AC-5 require the key `"dev-plan"` (hyphen). The serialised key name is wrong, breaking
        any consumer that follows the spec schema.
        Fix: change the tag to `json:"dev-plan"`.
        (spec: FR-9.1, AC-5, NFR-1.1, location: internal/cli/status/json.go L43)
      - [blocking] FR-9.1 schema violation — `feature.display_id` missing: jsonFeature has
        fields id, slug, status, plan_id but no display_id. Spec A-2 states it must be
        present and non-null (e.g. "F-042"). FeatureInput carries DisplayID which could
        populate it.
        Fix: add `DisplayID string` to jsonFeature with tag `json:"display_id"` and populate
        it from in.DisplayID in RenderFeature.
        (spec: FR-9.1 schema, A-2, location: internal/cli/status/json.go L33)
      - [blocking] FR-9.1 data integrity — document slot ID populated with file path: in
        RenderFeature, `byType[d.Type] = &documentSlot{ID: d.Path, Path: d.Path, Status: d.Status}`
        sets documentSlot.ID to the file path instead of the document record ID.
        Root cause: DocInput in render/types.go has no ID field, so the document record ID is
        lost before it reaches the renderer. The spec example shows
        `"design": {"id": "DOC-0019", "path": "...", "status": "..."}` — the id and path are
        distinct. Callers using the id for document lookup receive the path instead.
        Fix: add ID field to DocInput; populate it from the document record ID when building
        FeatureInput in workflow_cmd.go; propagate it through to documentSlot.ID.
        (spec: FR-9.1, location: internal/cli/status/json.go ~L135,
         internal/cli/render/types.go DocInput)
      - [blocking] FR-9.4 / FR-5 data integrity — bug parent_feature_id always null/missing:
        JSONRenderer.RenderBug hardcodes `ParentFeatureID: nil`; PlainRenderer.RenderBug
        hardcodes `{"parent_feature", "missing"}`. Neither renderer accepts a parentFeature
        parameter, so the value is never forwarded. runStatusBugFormatted extracts the parent
        feature from entity state but does not pass it to the renderer. For bugs with a parent
        feature, both outputs are unconditionally wrong.
        Fix: add a parentFeature string parameter to both RenderBug signatures and pass the
        extracted value from runStatusBugFormatted.
        (spec: FR-9.4, FR-5, location: internal/cli/status/json.go L200,
         internal/cli/status/plain.go L82, cmd/kbz/workflow_cmd.go ~L253)
      - [non-blocking] FR-7 approximation — plain project `features.done` is computed as
        `p.FeaturesTotal - p.FeaturesActive` (noted as "approximate" in a code comment).
        Features in non-done, non-active states (ready, specifying, designing) inflate the
        done count. ProjectPlanInput has no FeaturessDone field.
        Recommendation: add FeaturessDone to ProjectPlanInput and populate it accurately in
        the project overview builder.
        (spec: FR-7, location: internal/cli/status/plain.go ~L104)
      - [non-blocking] NFR-1.5 not satisfied — no contract test exists: `internal/cli/status/`
        contains no test files. The schema stability contract test required by NFR-1.5 and
        AC-11 ("MUST run in CI") is absent. This was noted in the first review pass and
        remains unaddressed.
        Recommendation: add plain_test.go and json_test.go; include a contract test that
        asserts the presence of every required key/field for each scope type.
        (spec: NFR-1.5, AC-11)

  implementation_quality: pass_with_notes
    Evidence:
      - attnToJSON returns [] not null when items is nil — correct per FR-8.5.
      - marshalResults correctly wraps single results in a {"results":[...]} array.
      - Null handling via `any` type for optional/nullable fields (plan_id, parent_feature_id,
        document id/type/status when unregistered) is idiomatic and correct.
      - severityRank sort in PlainRenderer.attentionFirst correctly surfaces highest-severity
        attention item (FR-3.3).
    Findings:
      - [non-blocking] RenderProject in json.go sets jsonProjectPlan.Slug to DisplayID (the
        plan ID, not the slug): `Slug: p.DisplayID`. ProjectPlanInput has no separate Slug
        field; DisplayID carries the full ID including slug in the batch system (e.g.
        "B36-kbz-cli-and-status"). The spec says `slug` should be the slug portion only.
        Recommendation: add a Slug field to ProjectPlanInput and populate it from plan data.
        (spec: FR-10, location: internal/cli/status/json.go ~L233)

  test_adequacy: fail
    Evidence:
      - No files exist in internal/cli/status/ other than plain.go and json.go.
    Findings:
      - [non-blocking] No unit tests for PlainRenderer or JSONRenderer. AC-11 (schema
        contract test, CI-gated) cannot be verified. NFR-1.5 is unmet.
        (spec: NFR-1.5, AC-11)

Finding Summary:
  Blocking: 4
  Non-blocking: 3
  Total: 7
```

---

### Code Review Summary

| Review Unit | Overall | Blocking | Non-blocking | Total |
|-------------|---------|----------|--------------|-------|
| F1: Binary rename | pass | 0 | 0 | 0 |
| F2: Argument resolution | needs_remediation | 3 | 2 | 5 |
| F3: Human output | needs_remediation | 3 | 4 | 7 |
| F4: Machine output | needs_remediation | 4 | 3 | 7 |
| **Batch total** | **needs_remediation** | **10** | **9** | **19** |

> **Note on blocking rate:** 10 of 19 findings are blocking (53%). This exceeds the
> typical review calibration threshold (40%). Each blocking finding was verified to have
> a specific violated acceptance criterion or MUST-level functional requirement before
> classification was retained. The elevated rate reflects genuine implementation gaps
> in the F2 routing layer and F4 JSON schema, not severity inflation.

## Batch Verdict

**FAIL** — two independent sets of blocking findings:

1. **CG-1 (lifecycle):** Four features in non-terminal lifecycle states despite all tasks done.
2. **CG-5 through CG-9 (code conformance):** Ten blocking code-level findings across F2, F3, and F4. Specifically: FR-007 routing gap, two exit-code violations (F2), unregistered-doc output format violations (F3), three JSON schema violations and one data-integrity bug in F4.

## Recommended Actions

### Required — lifecycle
1. Advance all 4 features through their lifecycle to `done`:
   - F1: `dev-planning` → `done` (branch already merged)
   - F2: `developing` → resolve merge conflicts → merge → `done`
   - F3: `developing` → merge → `done`
   - F4: `developing` → resolve merge conflicts → merge → `done`
2. Clean up all 4 worktrees after merges.

### Required — code fixes (blocking findings)
3. **F2 — implement FR-007 fallback probe** (`runStatus` `ResolveNone` branch): attempt entity lookup, then path lookup, before returning an error. Both `runStatusEntity` and `runStatusPlanPrefix` must also return proper errors (not nil) on not-found so the caller can exit with code 1.
4. **F2 — fix exit codes**: `runStatusEntity` and `runStatusPlanPrefix` must return errors (not nil) on not-found. The caller `runStatus` must translate these errors to exit 1 with a descriptive message.
5. **F3 — fix unregistered-doc output format**: line 1 must be the target path as supplied (no `File:` prefix, no normalisation applied to the displayed value); second line blank + indented `Not registered with Kanbanzai.`; register suggestion must include `--type <type> --title <title>` placeholders.
6. **F4 — fix `dev-plan` JSON key**: change `json:"dev_plan"` to `json:"dev-plan"` in `jsonDocs`.
7. **F4 — add `display_id` to feature JSON**: add `DisplayID string` with tag `json:"display_id"` to `jsonFeature`; populate from `in.DisplayID` in `RenderFeature`.
8. **F4 — fix document slot ID**: add `ID string` field to `DocInput` in `render/types.go`; populate it from the document record ID in the `runStatusFeatureFormatted` and `runStatusEntityHuman` builders; use it (not Path) for `documentSlot.ID`.
9. **F4 — fix bug parent_feature_id**: add `parentFeature string` parameter to `PlainRenderer.RenderBug` and `JSONRenderer.RenderBug`; forward the extracted value from `runStatusBugFormatted`.

### Recommended — non-blocking improvements
10. **F4 — add tests**: add `internal/cli/status/plain_test.go` and `json_test.go`; include a contract test enumerating required keys/fields for each scope type (NFR-1.5, AC-11).
11. **F3 — fix registered-doc human layout**: restructure `runStatusPath` registered branch to output file path as line 1, then aligned `Title:` / `Type:` / `Status:` indented fields (FR-3.1).
12. **F3 — align plan feature rows**: pass feature rows through a column-alignment helper (FR-5.3).
13. **F3 — add slug to project plan rows** (FR-6.2).
14. **F4 — fix features.done approximation**: add `FeaturesDone int` to `ProjectPlanInput`; populate accurately.
15. **F2 — expand bare plan prefix regex** to `^[A-Z]{1,2}[0-9]+$` for two-letter prefix support (ASM-002).
16. **B36 — fix AGENTS.md line 78** stale `cmd/kanbanzai/` reference.
17. **B36 — contribute knowledge entries** for disambiguation logic, TTY detection, and JSON renderer architecture decisions.

## Evidence

### First pass (lifecycle)
- Batch entity: `entity(action: "get", id: "B36-kbz-cli-and-status")` → 4 features, status: active
- Feature list: `entity(action: "list", type: "feature", parent: "B36-kbz-cli-and-status")` → 4 features
- Spec approvals: All 4 specs approved ✅
- Health check: `health()` → 4 `feature_child_consistency` warnings for B36 features
- Retro synthesis: `retro(action: "synthesise", scope: "B36-kbz-cli-and-status")` → 0 signals
- Build verification: `go build ./cmd/kbz/` → success; `go vet ./internal/cli/... ./internal/resolution/...` → clean
- Test verification: `go test ./internal/cli/render/... ./internal/resolution/...` → pass

### Second pass (code audit)
- Specs read: all 4 approved specs (`B36-F1` through `B36-F4`)
- Source files read: `internal/resolution/resolution.go`, `internal/cli/render/{types,symbols,tty,renderer,feature,plan,project}.go`, `internal/cli/status/{plain,json}.go`, `cmd/kbz/{workflow_cmd,doc_cmd,main}.go`
- Test files read: `internal/resolution/resolution_test.go`, `internal/cli/render/feature_test.go`
- CG-5: confirmed at `cmd/kbz/workflow_cmd.go` `runStatus` default branch (~L130)
- CG-6: confirmed at `runStatusEntity` (~L233-238) and `runStatusPlanPrefix` (~L435-439)
- CG-7: confirmed at `runStatusPath` unregistered branch (~L571-574)
- CG-8a (`dev_plan` key): confirmed at `internal/cli/status/json.go` L43
- CG-8b (`display_id` absent): confirmed at `json.go` L33 (`jsonFeature` struct)
- CG-8c (doc slot ID = path): confirmed at `json.go` `RenderFeature` body and `render/types.go` `DocInput` struct (no ID field)
- CG-9 (bug parent always null): confirmed at `json.go` `RenderBug` L200 and `plain.go` `RenderBug` L82; `workflow_cmd.go` `runStatusBugFormatted` extracts `parentFeature` but never passes it
