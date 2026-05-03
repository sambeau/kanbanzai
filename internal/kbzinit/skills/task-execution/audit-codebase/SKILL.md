---
# kanbanzai-managed: true
# kanbanzai-version: dev
name: audit-codebase
description:
  expert: "Performs a spec-free, whole-codebase quality audit by running static
    analysis tools, knowledge-graph structural queries, and Go style conformance
    checks. Produces a triage-ready audit report saved to work/reviews/. Used
    before public releases, after large refactors, or on demand."
  natural: "Do a quality audit of the whole codebase. Check for dead code,
    linting errors, style issues, and performance problems."
triggers:
  - "audit the codebase"
  - "run a quality sweep"
  - "check for dead code"
  - "pre-release quality check"
  - "find linting errors across the whole project"
  - "code quality audit"
  - "sweep for quality issues"
  - "360 review of the codebase"
roles: [reviewer-quality]
stage: auditing
constraint_level: low
---

## Vocabulary

- **audit finding** — a quality observation not anchored to a specification;
  classified by severity (critical, major, minor) rather than blocking/non-blocking
- **static analysis** — automated tool-based inspection using `go vet` and
  `staticcheck` to surface undefined behaviour, deprecated APIs, and suspicious constructs
- **structural analysis** — knowledge-graph-based inspection for dead code,
  coupling, and complexity using degree queries
- **dead code candidate** — a function with zero inbound CALLS edges that is not
  an entry point; must be verified with `trace_call_path` before classification
- **entry point** — a function registered as a route handler, `main()`, or
  framework callback that legitimately has zero graph callers and must be
  excluded from dead code results
- **fan-out** — the count of other functions a function calls directly; fan-out
  ≥ 10 indicates a function doing too much (Single Responsibility violation)
- **change coupling** — two files that co-change frequently in git history;
  high coupling between semantically unrelated files indicates hidden dependencies
- **triage pass** — the mandatory step of grouping, deduplicating, and
  severity-ranking raw tool output before writing the report
- **severity: critical** — a finding that poses correctness, data loss, race
  condition, or test-failure risk; blocks confidence in a release
- **severity: major** — a finding that degrades maintainability, test
  reliability, or operational confidence at a structural level
- **severity: minor** — a style deviation, naming inconsistency, or low-impact
  observation worth recording but not requiring immediate action
- **audit scope** — the directory path(s) to audit; defaults to `./...` unless
  narrowed by the caller
- **false positive** — a dead-code or linter finding that appears to be an
  issue but is valid code (interface implementation, reflection-used function)
- **SRP violation** — a function or package with more than one clear
  responsibility, identified by high fan-out or unrelated concerns in one file

## Anti-Patterns

### Spec-Anchoring Reflex

- **Detect:** Refusing to classify a finding as actionable because no spec
  requirement is violated, or marking all findings non-blocking by default.
- **BECAUSE:** This is not a conformance review. There is no spec. Quality
  findings stand on their own as violations of Go idiom, project style rules
  in `refs/go-style.md`, or the structural thresholds defined in this skill.
  Anchoring to a missing spec produces an empty report.
- **Resolve:** Anchor findings to `refs/go-style.md` rules, Go idiom standards,
  or the structural thresholds defined here. Do not look for a spec.

### Linter Literalism

- **Detect:** Pasting raw linter output verbatim without triage — reporting
  dozens of findings at uniform severity with no grouping.
- **BECAUSE:** Raw tool output contains noise, duplicates, and low-signal nits
  that overwhelm the report reader and obscure genuinely critical findings.
  A report where everything is equally important is a report where nothing is.
- **Resolve:** Run a triage pass before reporting: deduplicate, group by
  dimension, and severity-rank. Report triaged findings, not tool output lines.

### Dead Code False Positive

- **Detect:** Classifying a function as dead code based solely on a
  zero-inbound-degree graph result, without verification.
- **BECAUSE:** The graph does not model reflection-based calls, interface
  implementations registered at runtime, or test helpers called only from
  test files. Premature deletion of valid code causes regressions.
- **Resolve:** For every dead code candidate, call
  `trace_call_path(direction: "inbound")` and check for USAGE edges before
  classifying as dead. If uncertain, mark as "verify before deletion".

### Scope Creep into Design

- **Detect:** Findings that recommend architectural redesigns, package
  restructuring, or API contract changes rather than describing a concrete
  quality problem.
- **BECAUSE:** An audit identifies what is wrong today; it does not propose
  new architectures. Design recommendations belong in a design document
  produced through the designing stage, not in an audit report.
- **Resolve:** Describe the concrete quality problem and the minimum fix
  (rename, extract, delete, add error check). If a finding genuinely requires
  a design decision, note it as "design decision required" and stop there.

### Graph-Only Shortcut

- **Detect:** Skipping static analysis tools and relying solely on graph
  queries to produce findings.
- **BECAUSE:** Graph queries detect structural patterns but cannot find linter
  violations, incorrect error handling, race conditions, or test failures.
  Each method has distinct, non-overlapping coverage.
- **Resolve:** Always run both static analysis tools AND structural graph
  queries. Neither substitutes for the other.

### Flattened Severity

- **Detect:** All findings reported at the same severity level — everything
  minor, or everything major.
- **BECAUSE:** Uniform severity forces the reader to manually triage every
  finding to determine what needs attention first, defeating the purpose of
  a ranked report.
- **Resolve:** Apply the severity definitions strictly. At least two severity
  levels should appear in any non-trivial audit. If everything truly is the
  same severity, state that explicitly with justification.

## Checklist

- [ ] Confirm audit scope (default: `./...`)
- [ ] Check index currency: `index_status()`
- [ ] Run `go vet ./...` — capture all output
- [ ] Run `staticcheck ./...` — capture all output
- [ ] Run `go test -race ./...` — record pass/fail and failure details
- [ ] Run dead code query — verify each candidate before classifying
- [ ] Run high fan-out query — review each function ≥ 10 outbound calls
- [ ] Run change coupling query — note file pairs with coupling_score ≥ 0.5
- [ ] Spot-check style conformance against `refs/go-style.md`
- [ ] Run triage pass — deduplicate, group, severity-rank
- [ ] Write audit report to `work/reviews/audit-{slug}.md`
- [ ] Register the report document

## Procedure

### Step 1: Establish scope

1. Record the audit scope: use the caller-specified path filter, or `./...`
   if none was given.
2. Check index currency: call `index_status()`. IF the index is stale (last
   indexed > 7 days ago) or not indexed → call
   `index_repository(repo_path: ".")` before running any graph queries.
3. Record the scope and index state in the report header.

### Step 2: Static analysis

Run each tool and capture its full output:

1. `go vet ./...` — catches undefined behaviour, unreachable code, printf
   format mismatches, and suspicious constructs.
2. `staticcheck ./...` — catches deprecated API usage, redundant code,
   incorrect format strings, and a wider set of correctness issues.

IF a tool is not installed → note the gap explicitly in the report under
that dimension. Do not skip the step silently.

Do NOT run `golangci-lint` unless the project has a `.golangci.yml` or
`.golangci.toml` config file. Running it without project-specific
configuration produces excessive noise.

### Step 3: Run the test suite

Run `go test -race ./...`. Record:
- Overall pass or fail
- The name and failure message of any failing test
- Any race conditions detected by the race detector

IF tests fail → record these as critical findings. A failing test suite
invalidates confidence in all other quality claims made by this audit.

### Step 4: Structural analysis via knowledge graph

Read `.github/skills/codebase-memory-quality/SKILL.md` for the exact tool
calls before running queries. Then execute:

1. **Dead code candidates** — `search_graph` with `label: "Function"`,
   `relationship: "CALLS"`, `direction: "inbound"`, `max_degree: 0`,
   `exclude_entry_points: true`.
   For each candidate: verify with `trace_call_path(direction: "inbound",
   depth: 1)` and check for USAGE edges. Classify confirmed dead code as
   major. Mark uncertain cases as "verify before deletion".

2. **High fan-out functions** — `search_graph` with `relationship: "CALLS"`,
   `direction: "outbound"`, `min_degree: 10`.
   For each result: is the fan-out incidental (a large dispatch table or
   switch) or a genuine SRP violation? Classify genuine SRP violations as
   major; incidental fan-out as minor.

3. **Change coupling** — `query_graph` for `FILE_CHANGES_WITH` edges with
   `coupling_score >= 0.5`. Identify semantically unrelated file pairs with
   high coupling. Classify hidden coupling between unrelated packages as major.

4. **Unused imports** — `search_graph` with `relationship: "IMPORTS"`,
   `direction: "outbound"`, `max_degree: 0`, `label: "Module"`. These should
   already be caught by `go vet`; note any that were missed by tooling.

### Step 5: Style conformance spot-check

Read `refs/go-style.md`. For each rule category, scan for violations using
`grep` and `read_file`:

- **Error handling:** look for `_ =` on error-returning calls; unwrapped
  error returns that discard context.
- **Naming:** acronym casing (`URL`, `HTTP`, `ID` not `Url`, `Http`, `Id`);
  package names with underscores or mixed case.
- **Interfaces:** defined at the provider package rather than the consumer.
- **Comments:** exported functions missing a doc comment starting with the
  function name.
- **Concurrency:** goroutines spawned without `context.Context` for
  cancellation.

### Step 6: Triage pass

Before writing the report:

1. Deduplicate: collapse findings reported by multiple tools at the same
   location into one finding at the highest severity seen.
2. Group by dimension: `test_health`, `static_analysis`, `structural_quality`,
   `style_conformance`.
3. Severity-rank within each dimension: critical → major → minor.
4. Calibration check: if > 80% of findings are minor → re-examine whether
   any were under-classified.
5. Calibration check: if findings include critical but tests all pass → verify
   the critical classification is genuinely correctness-affecting.

### Step 7: Write and register the audit report

1. Write to `work/reviews/audit-{slug}.md` where `{slug}` is a short
   kebab-case descriptor (e.g., `pre-v1-release`, `post-storage-refactor`).
2. Use the Output Format below.
3. Register: `doc(action: "register", path: "work/reviews/audit-{slug}.md",
   type: "report", title: "Quality Audit: {description}", auto_approve: true)`

## Output Format

```
Quality Audit: <description>
Date: <ISO 8601 date>
Scope: <path filter, e.g. ./...>
Index state: current | refreshed | stale (noted)
Tools run: go vet, staticcheck, go test -race, knowledge graph

Overall: clean | clean_with_findings | has_major_findings | has_critical_findings

Dimensions:

  test_health: clean | clean_with_notes | has_findings
    Summary: <N tests, Y passed, Z failed; race: none | detected>
    Findings:
      - [critical] <test name>: <failure message> (location: <file:line>)
        Recommendation: <fix>

  static_analysis: clean | clean_with_notes | has_findings
    Summary: <N findings from go vet, M findings from staticcheck>
    Findings:
      - [<severity>] <tool>: <message> (location: <file:line>)
        Recommendation: <fix>

  structural_quality: clean | clean_with_notes | has_findings
    Summary: <N dead code candidates confirmed, M high fan-out, K coupled pairs>
    Findings:
      - [<severity>] dead_code: <FunctionName> — zero callers confirmed
          (location: <file:line>)
          Recommendation: delete or add intent comment if deliberate
      - [<severity>] high_fan_out: <FunctionName> calls <N> functions
          (location: <file:line>)
          Recommendation: <extract / decompose>
      - [<severity>] change_coupling: <file-a> <-> <file-b>
          coupling_score: <X>, co-changes: <N>
          Recommendation: investigate / extract shared dependency

  style_conformance: clean | clean_with_notes | has_findings
    Summary: <N violations found>
    Findings:
      - [<severity>] <rule>: <description> (location: <file:line>)
        Recommendation: <fix>

Finding Summary:
  Critical: <count>
  Major:    <count>
  Minor:    <count>
  Total:    <count>

Next Actions:
  1. <highest-severity finding — one-line description and location>
  2. <next finding>
  ...
```

**Overall verdict rules:**
- `clean` — zero findings across all dimensions.
- `clean_with_findings` — zero critical or major; one or more minor findings.
- `has_major_findings` — one or more major findings; zero critical.
- `has_critical_findings` — one or more critical findings (test failures,
  race conditions, correctness bugs).

## Examples

### BAD: Dead code reported without verification

> structural_quality: has_findings
>   Findings:
>     - [major] dead_code: formatEntity — zero callers (location: internal/format.go:42)
>       Recommendation: delete

**WHY BAD:** `formatEntity` was identified via a graph degree query but was
never verified with `trace_call_path` or a USAGE edge check. It may be called
via an interface, a test helper, or registered through reflection. Deleting it
without verification risks a regression.

### BAD: Linter output pasted without triage

> static_analysis: has_findings
>   Findings:
>     - [major] staticcheck: SA1006: printf with dynamic first arg (location: cmd/root.go:12)
>     - [major] staticcheck: SA1006: printf with dynamic first arg (location: cmd/root.go:12)
>     - [minor] go vet: unreachable code (location: internal/store/store.go:88)
>     - [major] staticcheck: SA4016: certain bitwise ops have no effect (location: internal/id.go:31)
>     - [major] staticcheck: SA1006: printf with dynamic first arg (location: cmd/util.go:7)
>     - [major] staticcheck: SA1006: printf with dynamic first arg (location: cmd/util.go:7)

**WHY BAD:** The same finding at `cmd/root.go:12` appears twice (not
deduplicated). All findings are classified major regardless of actual severity.
The reader cannot tell what to fix first.

### GOOD: Triaged audit with calibrated severity

> Quality Audit: pre-v1-release
> Scope: ./...
> Tools run: go vet, staticcheck, go test -race, knowledge graph
>
> Overall: has_major_findings
>
> Dimensions:
>
>   test_health: clean
>     Summary: 142 tests, 142 passed, 0 failed; race: none detected
>
>   static_analysis: clean_with_notes
>     Summary: 0 go vet findings, 2 staticcheck findings
>     Findings:
>       - [minor] staticcheck SA1006: printf with dynamic first arg
>           (location: internal/cli/root.go:14)
>           Recommendation: use fmt.Println or wrap in a format string
>       - [minor] staticcheck SA4016: bitwise AND always returns zero
>           (location: internal/id/id.go:31)
>           Recommendation: review flag mask; likely off-by-one in constant
>
>   structural_quality: has_findings
>     Summary: 1 dead code confirmed, 1 high fan-out, 0 coupled pairs
>     Findings:
>       - [major] dead_code: legacyMigrateV1 — zero callers confirmed,
>           no USAGE edges, not an entry point
>           (location: internal/store/migrate.go:88)
>           Recommendation: delete; migration path was superseded
>       - [minor] high_fan_out: applyTransition calls 11 functions
>           (location: internal/lifecycle/transition.go:55)
>           Recommendation: incidental — large switch dispatch; acceptable
>
>   style_conformance: clean_with_notes
>     Summary: 1 violation found
>     Findings:
>       - [minor] naming: exported type EntityId uses lowercase acronym
>           (location: internal/entity/types.go:12)
>           Recommendation: rename to EntityID per refs/go-style.md acronym rule
>
> Finding Summary:
>   Critical: 0
>   Major:    1
>   Minor:    3
>   Total:    4
>
> Next Actions:
>   1. Delete legacyMigrateV1 (internal/store/migrate.go:88) — confirmed dead code
>   2. Fix printf dynamic arg (internal/cli/root.go:14)
>   3. Rename EntityId -> EntityID (internal/entity/types.go:12)
>   4. Verify bitwise mask in id.go:31

**WHY GOOD:** Dead code was confirmed with `trace_call_path` before
classification. Findings are deduplicated and severity-ranked. The high-fan-out
function was examined and correctly classified as incidental rather than an SRP
violation. The Next Actions list gives the reader an unambiguous work queue.

## Evaluation Criteria

1. Does the report run and record output from all four analysis methods
   (go vet, staticcheck, go test -race, graph queries)?
   Weight: required.
2. Is every dead code finding confirmed with `trace_call_path` before
   classification? Weight: required.
3. Are findings deduplicated, grouped by dimension, and severity-ranked
   before the report is written? Weight: high.
4. Does the severity distribution use at least two levels when total
   findings > 3? Weight: high.
5. Does the Next Actions list reflect finding severity order?
   Weight: medium.
6. Is the report registered as a document record after writing?
   Weight: medium.

## Questions This Skill Answers

- How do I audit the whole codebase for quality issues without a spec?
- What tools should I run for a pre-release quality sweep?
- How do I find dead code in the knowledge graph safely?
- What counts as a critical vs. major vs. minor audit finding?
- How do I identify hidden coupling between files?
- When should I run `golangci-lint` vs. just `staticcheck`?
- How do I produce a structured quality report that is not tied to a feature?
- What is the correct output format for a codebase audit report?