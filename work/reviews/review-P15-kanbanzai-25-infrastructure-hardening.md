# Review: Kanbanzai 2.5 — Infrastructure Hardening (P15)

| Field        | Value                                                                        |
|--------------|------------------------------------------------------------------------------|
| Plan         | P15-kanbanzai-25-infrastructure-hardening                                    |
| Specification| `work/design/kanbanzai-2.5-infrastructure-hardening.md` + per-feature specs |
| Reviewer     | Review Agent                                                                 |
| Date         | 2026-04-01                                                                   |
| Status       | Draft                                                                        |

---

## 1. Scope

This review covers all six features of P15:

| Feature ID          | Slug                         | Spec                                           |
|---------------------|------------------------------|------------------------------------------------|
| FEAT-01KN4ZPCMJ1FP  | docint-ac-pattern-recognition | `work/spec/docint-ac-pattern-recognition.md`  |
| FEAT-01KN4ZPG44T5B  | sub-agent-state-isolation     | `work/spec/sub-agent-state-isolation.md`      |
| FEAT-01KN4ZPKCJWPF  | batch-handler-false-positive  | `work/spec/batch-handler-false-positive.md`   |
| FEAT-01KN4ZPQFQN1C  | doc-audit                     | `work/spec/doc-audit.md`                      |
| FEAT-01KN4ZPTQSZT5  | doc-import-dry-run            | `work/spec/doc-import-dry-run.md`             |
| FEAT-01KN4ZPXXEG1F  | evaluation-baseline           | `work/spec/evaluation-baseline.md`            |

All tests pass (`go test ./...`) with no failures or regressions.

---

## 2. Summary

The implementation is solid. Five of the six features are complete and correct against
their specifications. The one code-level defect is a gap in the `doc audit` path
handler — a non-existent explicit `path` argument returns empty results instead of an
error as specified. There are several documentation and test coverage gaps. No design
decisions require human consultation.

**Finding counts:** 1 defect · 5 gaps · 2 improvements · 2 nits

---

## 3. Findings

### F-01 · DEFECT — `doc audit`: explicit `path` non-existence not an error

**Specification:** `work/spec/doc-audit.md` REQ-02

> When `doc(action: "audit", path: "<dir>")` is called, the tool walks only the
> specified directory. **If the directory does not exist, the tool returns an error.**

**Implementation:** `internal/service/doc_audit.go` `AuditDocuments`

When `dirs` is a single-element slice from an explicit `path` argument, the walk
loop reaches this block:

```kanbanzai/internal/service/doc_audit.go#L148-151
if _, statErr := os.Stat(absDir); os.IsNotExist(statErr) {
    continue
}
```

The nonexistent directory is silently skipped and the function returns a valid
`AuditResult` with all-zero counts and empty arrays. The caller never learns that
the path was invalid. The `continue` is correct for the default-directory case
(where some dirs may not exist in a fresh project), but when the caller supplies an
explicit `path` the spec requires an error.

**Fix (implementation decision):** Before the walk loop, check whether an explicit
path was provided and, if so, stat it first and return an error if it is absent. The
default-directory case (where `dirs` came from `defaultAuditDirs`) should retain the
existing silent-skip behaviour.

---

### F-02 · GAP — AC-06: no integration test for `decompose propose` + bold-AC spec

**Specification:** `work/spec/docint-ac-pattern-recognition.md` AC-06, REQ-13–REQ-14

AC-06 requires:

> `decompose propose` on a specification file that uses `**AC-NN.**` format produces
> task summaries derived from the criteria text, not from section headers.

There are thorough unit tests on `asmExtractCriteria` in `assembly_test.go`, and the
baseline measurement `baseline-eval-001-20260401.yaml` documents the pre-fix failure
mode. However, there is no automated test that exercises the full pipeline:
`decompose propose` → `asmExtractCriteria` → task summary generation. The
verification plan in the design doc (`§11`) calls for "an integration test with
`decompose propose` on a fixture spec", which does not exist.

The unit tests provide high confidence in the extractor itself. The gap is that a
regression in the plumbing between the extractor and the task summary generator would
not be caught automatically.

---

### F-03 · GAP — AGENTS.md scope guard and project timeline not updated for P15

**Quality criterion:** Workflow document currency

The `AGENTS.md` scope guard currently ends at:

> … and P12 (agent onboarding and skill discovery — …) are all complete.

P15 is functionally complete — all 15 tasks are in `done` status — but neither the
scope guard nor `docs/project-timeline.md` mention it. P13 and P14 are also absent
from both documents.

Agents reading AGENTS.md to understand what has been built will not discover P15,
its features, or the evaluation baseline infrastructure. The scope guard exists
precisely to prevent agents from re-building completed work.

**Note:** The plan status is still `proposed` in the entity store. If P15 is
considered complete, the plan should be transitioned to `done` and the scope guard
updated accordingly.

---

### F-04 · GAP — Eval scenario schema extensions undocumented in README

**Specification:** `work/spec/evaluation-baseline.md` REQ-10

REQ-10 requires the README to document "the complete YAML schema for scenario files,
with field descriptions and allowed values." However, at least two fields appear in
scenario files but are absent from the README schema:

1. `starting_state.notes` — present in `eval-001.yaml` as a prose annotation.
2. `expected_pattern[].tool_args_must_include` — present in every stage of
   `eval-001.yaml` with structured argument assertions (e.g.,
   `{action: transition, status: designing}`).

`tool_args_must_include` in particular is semantically significant: it expresses
testable assertions about how tools must be called, beyond the plain-text `output`
description. V3.0 evaluation runners that parse scenario files will encounter these
fields without any schema documentation to guide them.

---

### F-05 · GAP — No service-level unit tests for `ImportDryRun`

**Quality criterion:** Test coverage and quality

`internal/service/import.go` exports `ImportDryRun`, but `internal/service/import_test.go`
contains no tests for it. All dry-run coverage lives in `internal/mcp/doc_tool_test.go`
at the MCP handler layer. The service-layer logic — in particular the consistency
guarantee between `Import` and `ImportDryRun` (REQ-13) — is exercised only through
the full tool handler stack.

A logic regression in `ImportDryRun` that does not affect the outer handler response
shape would not be caught by the current test suite.

---

### F-06 · GAP — No service-level unit tests for `AuditDocuments`

**Quality criterion:** Test coverage and quality

`internal/service/doc_audit.go` has no corresponding `doc_audit_test.go`. All audit
tests are integration-style in `internal/mcp/doc_tool_test.go`. The service-layer
invariant (`summary.registered + summary.unregistered == summary.total_on_disk`) and
the missing-record scoping logic are exercised only indirectly.

This is lower priority than F-05 because the MCP-layer tests are thorough, but a
direct service test would improve isolation and catch regressions in the `AuditDocuments`
function independently of the handler.

---

### F-07 · IMPROVEMENT — `hasRFC2119Keyword` is case-insensitive despite ASM-02

**Specification:** `work/spec/docint-ac-pattern-recognition.md` ASM-02

ASM-02 states:

> RFC 2119 keywords for the purposes of this specification are `MUST`, `MUST NOT`,
> `SHALL`, `SHALL NOT`, `SHOULD`, `SHOULD NOT`, `MAY`, `REQUIRED`, `RECOMMENDED`,
> `OPTIONAL` (case-sensitive, per RFC 2119).

The implementation in `assembly.go` converts text to uppercase before comparing:

```kanbanzai/internal/mcp/assembly.go#L33-45
func hasRFC2119Keyword(text string) bool {
	upper := strings.ToUpper(text)
	padded := " " + upper + " "
	for _, kw := range []string{...} {
		if strings.Contains(padded, kw) {
			return true
		}
	}
	return false
}
```

This means a bold-identifier line containing `must`, `should`, or `may` (lowercase)
outside an acceptance section will be extracted. The spec's assumption says those
lowercase forms should not match.

In practice this is more permissive than harmful — real specification documents tend
to use uppercase RFC 2119 keywords — but it contradicts ASM-02. The behaviour should
either be made case-sensitive (matching ASM-02) or the assumption should be relaxed
in the spec.

---

### F-08 · IMPROVEMENT — RFC 2119 threshold asymmetry between extraction paths

**Quality criterion:** Internal documentation / code quality

The `hasRFC2119Keyword` function (used for bold-identifier lines outside AC sections)
checks all eight RFC 2119 keywords including `SHOULD`, `MAY`, `REQUIRED`,
`RECOMMENDED`, and `OPTIONAL`. The legacy list-item extraction path (lines 354–361
of `assembly.go`) only gates on `MUST`, `SHALL`, `MUST NOT`, and `SHALL NOT`.

This means:

- A **list item** containing "The handler SHOULD validate input" outside an AC section
  is **not** extracted.
- A **bold-identifier line** `**AC-01.** The handler SHOULD validate input.` outside
  an AC section **is** extracted.

The asymmetry is not documented and could produce surprising extraction differences
between spec documents that use list-item format versus bold-identifier format. If the
broader keyword set is the intended threshold, the list-item path should be updated to
match. If the narrower set is correct, `hasRFC2119Keyword` should be trimmed.

---

### F-09 · NIT — Spec ASM-01 imprecise about `ItemResult.Error` type

**Specification:** `work/spec/batch-handler-false-positive.md` ASM-01

ASM-01 states:

> The `ItemResult` struct has a Status field that accepts at least `"success"` and
> `"error"` values, and an **Error field that accepts a string message**.

The actual `ItemResult.Error` field is `*ErrorDetail` (a struct with `Code` and
`Message` fields), not a plain string. The error message is correctly conveyed via
`ErrorDetail.Message`, and the tests confirm this (`result.Error.Message == errMsg`).
The spec assumption is imprecise about the concrete type. No functional impact.

Additionally, the assumption refers to `"success"` as a Status value, but the
implementation uses `"ok"` for successful items. This was a pre-existing convention;
the assumption simply inherited the wrong string.

---

### F-10 · NIT — Skip-reason terminology inconsistency between live import and dry-run

**Quality criterion:** User-facing documentation (tool output)

The live `Import` path records skips with reason `"already imported"`. The
`ImportDryRun` path records the same condition as `"already registered"` (per REQ-12).

Both are descriptively accurate in their respective contexts, but an agent comparing
live import output with a prior dry-run output will see different reason strings for
the same semantic condition. This could cause string-matching logic in agent prompts
or scripts to miss the equivalence.

---

## 4. Specification Completeness Checklist

| Feature                        | All ACs implemented | Tests cover all ACs | Notes                        |
|--------------------------------|---------------------|---------------------|------------------------------|
| docint AC pattern recognition  | ✅ AC-01–06         | ⚠️ AC-06 unit only  | See F-02                     |
| Sub-agent state isolation      | ✅ AC-07–11         | ✅                  |                              |
| Batch handler false-positive   | ✅ AC-12–15         | ✅                  |                              |
| doc audit                      | ⚠️ REQ-02 gap       | ✅ AC-16–20         | See F-01, F-06               |
| doc import dry-run             | ✅ AC-21–25         | ✅ AC-21–25         | See F-05                     |
| Evaluation baseline            | ✅ AC-26–30         | N/A (docs)          | See F-03, F-04               |

---

## 5. Code Quality Observations

**assembly.go:** The `boldIdentifierRe` regex and `hasRFC2119Keyword` function are
clean and well-commented. The `continue` after the bold-identifier match guard
correctly prevents double-processing against the list-item path. The deduplication
via the `seen` map is correct and handles the same criterion appearing in multiple
sections.

**batch.go:** `extractToolResultError` is cleanly separated from `ExecuteBatch` and
correctly handles all edge cases (non-map data, missing key, non-string value, empty
string). The function is well-commented with spec references. The `nonEmptyEffects`
helper keeps JSON output clean.

**commit.go:** `CommitStateIfDirty` is a minimal, focused implementation. Using
`git status --porcelain` before `git add` correctly avoids empty commits. The
`runGitCmd` helper's stderr-in-error behaviour makes failures debuggable.

**handoff_tool.go:** The `commitStateFunc` package-level variable for test injection
is idiomatic Go and correctly uses `defer` to restore the original after each test.
The non-blocking failure handling (`log.Printf` + proceed) is consistent with the
spec's best-effort intent.

**doc_audit.go:** The trailing-slash normalisation for the missing-record check
(`relDir += "/"`) correctly prevents `"work/design"` from matching
`"work/design-old"`. The invariant comment is accurate. The absolute-path handling
for test injection is well-considered.

**import.go (`ImportDryRun`):** The implementation faithfully mirrors the live
`Import` path, sharing `inferDocType` and `deriveTitle`. The "already registered"
skip reason (vs "already imported" in `Import`) is worth aligning; see F-10.

---

## 6. Evaluation Baseline Assessment

The 20 scenario files cover all six required categories with counts meeting or
exceeding the minimums from REQ-09. The `work/eval/README.md` is comprehensive —
it documents the schema, allowed category values, known MCP tool names, measurement
record schema, run instructions, and V3.0 comparison methodology.

Five baseline measurements have been captured (eval-001, 004, 008, 015, 018),
covering happy-path, gate-failure-and-recovery, review-rework-loop, edge-case, and
tool-selection categories. The multi-feature-plan category was not measured in the
baseline. This is within the implementer's discretion per the spec; the five measured
scenarios are otherwise representative.

The baseline measurement for eval-001 usefully documents two of the exact defects
that P15 fixes — the bold-AC extraction failure causing section-heading task
summaries, and the absent pre-dispatch state commit. This provides clear "before"
evidence for the docint and state-isolation fixes.

The undocumented schema extensions (F-04) are the primary quality gap in this
deliverable.

---

## 7. Recommended Actions

| Priority | Finding | Action                                                                          |
|----------|---------|---------------------------------------------------------------------------------|
| High     | F-01    | Fix `AuditDocuments` to return an error when an explicit `path` does not exist  |
| High     | F-03    | Transition P15 plan to `done`; update AGENTS.md scope guard and project timeline |
| Medium   | F-02    | Add an integration test: `decompose propose` on a fixture spec with `**AC-NN.**` |
| Medium   | F-04    | Document `tool_args_must_include` and `starting_state.notes` in the README schema |
| Low      | F-05    | Add service-level unit tests for `ImportDryRun` in `import_test.go`             |
| Low      | F-06    | Add service-level unit tests for `AuditDocuments` in `doc_audit_test.go`        |
| Low      | F-07    | Decide: make `hasRFC2119Keyword` case-sensitive (per ASM-02) or update ASM-02   |
| Low      | F-08    | Align RFC 2119 keyword threshold between list-item and bold-identifier paths     |
| Nit      | F-09    | Correct ASM-01 in the batch spec to reflect `*ErrorDetail` and `"ok"` status    |
| Nit      | F-10    | Align skip reason: use `"already registered"` in both live import and dry-run   |
```

Now I'll save this file and then write the review document to disk.