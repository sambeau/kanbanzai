# Design: Kanbanzai 2.5 — Infrastructure Hardening

| Field | Value |
|-------|-------|
| Date | 2025-07-18 |
| Status | Draft |
| Author | Design Agent |
| Informed by | `work/reports/v3-feedback-gap-analysis.md`, `work/research/agent-orchestration-research.md`, `work/research/agent-skills-research.md` |
| Related | `work/design/kanbanzai-3.0-workflow-and-tooling-v2.md`, `work/design/skills-system-redesign-v2.md` |

---

## 1. Purpose

This release fixes infrastructure defects, adds missing document management capabilities, and establishes an evaluation baseline — all before the major V3.0 redesign. Every item in this release is either a bug fix, an additive tool enhancement, or measurement infrastructure. Nothing changes existing behaviour in ways that could conflict with V3.0.

### 1.1 Why 2.5, Not 3.0

The V3.0 Feedback and Gap Analysis Report (`work/reports/v3-feedback-gap-analysis.md`) identified three categories of work:

1. **Infrastructure blockers** — bugs and missing capabilities that will cause V3.0 to fail in practice if unfixed.
2. **V3.0 design additions** — skill content and binding registry changes that require the V3.0 skills system to exist.
3. **Workflow doc changes** — implementation sequencing adjustments within V3.0.

Category 1 is independent of the V3.0 design. It can ship now, delivers immediate value, and de-risks the V3.0 implementation. That's 2.5.

### 1.2 Relationship to V3.0

2.5 is a prerequisite for V3.0, not a subset of it. The dependency flows one way:

- The `docint` AC parser fix unblocks V3.0's template → gate → decomposition pipeline.
- The state isolation fix makes V3.0's `orchestrator-workers` topology safe.
- The batch handler fix ensures V3.0's batch operations report real results.
- The evaluation baseline provides the "before" measurement that makes V3.0's impact measurable.

No 2.5 item constrains or overlaps with V3.0's design scope.

### 1.3 Research Basis

The research alignment for each item is documented in the gap analysis report §7. In summary:

- **`docint` fix**: MetaGPT (structured artifacts must be parseable), Masters et al. (decomposition quality is the critical path).
- **State isolation**: Microsoft (mutable shared state anti-pattern), Google (error amplification — 17.2× for independent agents vs 4.4× for centralised).
- **Batch handler**: SWE-agent (poka-yoke — make wrong usage hard, make errors visible).
- **Evaluation baseline**: Skills research Theme 6 ("evaluation must precede documentation"), Anthropic ("start evaluating immediately with small samples").
- **`doc approve` sync**: SWE-agent poka-yoke — make wrong state hard to reach.
- **`doc audit`/`doc import --dry-run`**: No specific research backing, but consistent with the principle that tools should make the right path easy.

---

## 2. Scope

### In Scope

| Category | Items |
|----------|-------|
| Bug fixes (blockers) | `docint` AC-NN pattern recognition, sub-agent state isolation, batch handler false-positive |
| Tool enhancements | `doc audit` command, `doc import --dry-run` mode |
| Measurement infrastructure | Small-sample evaluation suite (15–20 baseline scenarios) |

### Out of Scope

- `doc approve` file-patching (superseded by the consistent-front-matter design — see §6)
- Stage gate enforcement changes (V3.0 Phase A)
- MCP tool description rewrites (V3.0 Phase B)
- Context assembly changes (V3.0 Phase C)
- New skills, roles, or binding registry entries (V3.0)
- Any change to existing lifecycle state machines or transition logic

---

## 3. Bug Fix: `docint` Acceptance Criteria Pattern Recognition

### 3.1 Problem

The document intelligence classifier (`internal/docint/taxonomy.go`) recognises "Acceptance Criteria" as a section heading role, but the context assembly's criteria extractor (`internal/mcp/assembly.go`, `asmExtractCriteria`) cannot parse the `**AC-NN.**` pattern used in Kanbanzai specifications. The extractor only recognises:

- Standard list items (`-`, `*`, `+`, `•`)
- Numbered lists (`1. `, `2. `, etc.)
- Lines containing RFC 2119 keywords (`MUST`, `SHALL`, `MUST NOT`, `SHALL NOT`)

Kanbanzai specs use a different format:

```
**AC-01.** The system must do X when Y.

**AC-02.** The system must not do Z unless W.
```

This is not a list item and is not prefixed with a number followed by `. `. The extractor silently returns zero criteria. The `decompose propose` tool then falls back to section headers and produces dangerously plausible garbage like "Implement 1. Purpose" and "Implement 3. Scope".

### 3.2 Impact

The V3.0 pipeline depends on this working:

1. **Document templates** (skills doc §5.1) define acceptance criteria as a required section.
2. **Structural checks** (workflow doc §10.4) verify acceptance criteria are present at stage gates.
3. **Decomposition** (workflow doc §11) uses extracted criteria to generate meaningful tasks.

If the extractor can't parse the criteria, the entire pipeline produces garbage from step 3 onward.

### 3.3 Design

Add recognition of the bold-identifier pattern to `asmExtractCriteria` in `internal/mcp/assembly.go`:

**Pattern**: `**AC-NN.**` followed by text on the same line, where `NN` is one or more digits. The bold marker and identifier are stripped; the trailing text becomes the criterion.

Additionally, recognise the generalised bold-identifier pattern `**XX-NN.**` (e.g., `**REQ-01.**`, `**C-03.**`) so that requirement and constraint identifiers in the same format are also extracted. This covers the formats recommended by the V3.0 specification template.

**Extraction rules:**

1. In sections identified as acceptance/criteria/requirement sections: extract all bold-identifier lines as criteria.
2. In other sections: extract bold-identifier lines only if the text also contains RFC 2119 keywords.
3. Preserve the identifier prefix (e.g., `AC-01: The system must...`) in the extracted criterion for traceability.

### 3.4 Affected Files

| File | Change |
|------|--------|
| `internal/mcp/assembly.go` | Add bold-identifier pattern to `asmExtractCriteria` |
| `internal/mcp/assembly_test.go` | Add test cases for `**AC-NN.**` and `**REQ-NN.**` patterns |

### 3.5 Acceptance Criteria

- **AC-01.** `asmExtractCriteria` extracts criteria from lines matching `**AC-NN.** text` in acceptance criteria sections.
- **AC-02.** `asmExtractCriteria` extracts criteria from lines matching `**REQ-NN.** text` and `**C-NN.** text` in requirement/constraint sections.
- **AC-03.** The extracted criterion text preserves the identifier (e.g., `AC-01: The system must...`).
- **AC-04.** Bold-identifier lines outside acceptance/requirement sections are only extracted if they contain RFC 2119 keywords.
- **AC-05.** Existing list-item and numbered-list extraction is unaffected (regression test).
- **AC-06.** `decompose propose` on a spec using `**AC-NN.**` format produces task summaries derived from the criteria, not from section headers.

---

## 4. Bug Fix: Sub-Agent State Isolation

### 4.1 Problem

MCP tool calls write to `.kbz/state/` without committing. When a sub-agent runs `git stash`, `git checkout`, or `git reset`, these uncommitted state changes are captured or reverted. In one observed incident, feature statuses reverted from `developing` to `specifying`, specs went from `approved` to `draft`, and tasks from `ready` to `queued`. The state was permanently lost.

This is a known anti-pattern identified by Microsoft's orchestration patterns reference: "sharing mutable state between concurrent agents" is listed as a common failure mode.

### 4.2 Impact

The V3.0 design relies heavily on `orchestrator-workers` topology. The orchestrator approves documents, transitions features, and creates tasks — all of which write to `.kbz/state/`. It then dispatches sub-agents for implementation. If any sub-agent runs a git operation that affects the working tree, the orchestrator's state changes are destroyed.

### 4.3 Design

The fix has two parts: a **tool-level safeguard** (enforceable) and a **skill-level anti-pattern** (advisory, deferred to V3.0).

**Tool-level safeguard (2.5):**

Before the `handoff` tool assembles a sub-agent prompt, commit any pending `.kbz/state/` changes. This is a defensive commit — it ensures that state written by the orchestrator's MCP tool calls is persisted before a sub-agent gets a chance to disrupt the working tree.

**Behaviour:**

1. When `handoff` is called, check for uncommitted changes under `.kbz/state/`.
2. If changes exist, create a commit with message: `chore(kbz): persist workflow state before sub-agent dispatch`.
3. Proceed with context assembly as normal.
4. If the commit fails (e.g., git lock contention), log a warning but do not block the handoff — this is a best-effort safeguard.

**What this does NOT do:**

- It does not prevent sub-agents from running git operations. That would require sandboxing beyond the current architecture.
- It does not commit non-state changes (code, documents). Only `.kbz/state/` files are included.
- It does not apply to `next` (which claims work for the current agent, not a sub-agent).

**Skill-level anti-pattern (V3.0):**

The `orchestrate-development` and `orchestrate-review` skills should include a "State Destruction via Git Operations" anti-pattern. This is deferred to V3.0 because the skills system doesn't exist yet.

### 4.4 Affected Files

| File | Change |
|------|--------|
| `internal/mcp/handoff_tool.go` | Add pre-dispatch state commit |
| `internal/git/commit.go` (or new file) | Add function to commit `.kbz/state/` if dirty |
| `internal/mcp/handoff_tool_test.go` | Test that state is committed before assembly |

### 4.5 Acceptance Criteria

- **AC-07.** When `handoff` is called and `.kbz/state/` has uncommitted changes, a commit is created before context assembly.
- **AC-08.** The commit includes only files under `.kbz/state/`, not other working tree changes.
- **AC-09.** The commit message follows the project's commit policy format.
- **AC-10.** If no `.kbz/state/` changes exist, no commit is created (no empty commits).
- **AC-11.** If the commit fails, `handoff` logs a warning and proceeds (best-effort, non-blocking).

---

## 5. Bug Fix: Batch Handler False-Positive

### 5.1 Problem

`ExecuteBatch` in `internal/mcp/batch.go` counts items as `succeeded` when the handler returns `(data, nil)`. But individual MCP tool handlers return tool-result errors as `(errorMap, nil)` — the Go error is `nil`, but the data payload signals failure. `ExecuteBatch` cannot distinguish these from genuine successes because it only checks the Go error return.

This means validation failures in batch operations (e.g., creating a task with a missing required field) are counted as successes in the batch summary.

### 5.2 Impact

Any batch operation that triggers validation errors will report inflated success counts. This masks real problems and makes batch operations untrustworthy for automation.

### 5.3 Design

The handler function signature is:

```
type BatchItemHandler func(ctx context.Context, item any) (itemID string, data any, err error)
```

The fix is to inspect the `data` return value for the tool-error pattern. In the Kanbanzai MCP convention, a tool-result error is a `map[string]any` with an `"error"` key (or a `*ToolResultError` type — check which pattern is used).

**Option A — Convention-based detection:**

After the handler returns `(itemID, data, nil)`, check if `data` is a `map[string]any` with an `"error"` key or an `"is_error"` field. If so, count it as `failed` and populate the `Error` field on the `ItemResult`.

**Option B — Explicit error type:**

Add a sentinel type (e.g., `BatchItemError`) that handlers can return as `data` to signal a non-Go-error failure. `ExecuteBatch` checks for this type.

**Recommendation:** Option A is simpler and doesn't require changing all callers. The MCP tool convention already uses `map[string]any{"error": ...}` for inline errors. Check for this pattern.

### 5.4 Affected Files

| File | Change |
|------|--------|
| `internal/mcp/batch.go` | Add tool-error detection in `ExecuteBatch` |
| `internal/mcp/batch_test.go` | Add test: handler returns `(errorMap, nil)` → counted as failed |

### 5.5 Acceptance Criteria

- **AC-12.** When a batch item handler returns `(data, nil)` where `data` is a tool-result error, `ExecuteBatch` counts it as `failed`, not `succeeded`.
- **AC-13.** The `ItemResult` for such items has `Status: "error"` and the error message from the tool-result error.
- **AC-14.** Handlers returning genuine success data (`(data, nil)` where `data` is not a tool-result error) are still counted as `succeeded` (regression test).
- **AC-15.** Handlers returning Go errors (`(_, _, err)`) are still counted as `failed` (regression test).

---

## 6. Withdrawn: `doc approve` Header Sync

This section originally proposed patching the document file's `Status:` front matter field when `doc approve` is called. **This item has been withdrawn** — the consistent-front-matter design (`work/design/consistent-front-matter.md`) solves the same problem with a better approach.

The consistent-front-matter design establishes that `Status` is a store-managed field that must not appear in managed documents' front matter at all. Rather than syncing a duplicated field, the design removes the duplication:

1. **Strip `Status` from managed documents** during a one-time migration (consistent-front-matter §9).
2. **Health-check-warn** if anyone reintroduces store-managed fields into front matter (consistent-front-matter §7).
3. **MCP tools remain read-only** with respect to document file content (consistent-front-matter §11) — automated file rewriting was explicitly considered and rejected.

The consistent-front-matter design, specification, and implementation plan exist (`work/design/consistent-front-matter.md`, `work/spec/consistent-front-matter.md`, `work/plan/consistent-front-matter-plan.md`) but the implementation has not yet been executed. The three tasks (T1: pipe-table parser, T2: health checks, T3: document migration) are independent of 2.5 scope and can be implemented separately or in parallel.

The original feedback that motivated this section — "agents see 'Draft' when the system says 'Approved'" — is fully addressed by the consistent-front-matter migration. Once `Status` is stripped from files, there is nothing to drift.

---

## 7. Tool Enhancement: `doc audit`

### 7.1 Problem

22 documents accumulated unregistered during normal work. There is no way to discover which files on disk are missing from the document store without manually comparing `doc list` output against a directory listing.

### 7.2 Design

Add a new action `audit` to the `doc` MCP tool. It scans known document directories and compares against the store.

**Behaviour:**

1. Walk the configured document directories (by default: `work/design/`, `work/spec/`, `work/plan/`, `work/research/`, `work/reports/`, `work/reviews/`, `docs/`).
2. For each `.md` file found, check whether a document record exists with that path.
3. Return three lists:
   - **Unregistered**: files on disk with no store record.
   - **Missing**: store records whose files no longer exist on disk.
   - **Registered**: files with matching store records (count only, not listed individually).

**Parameters:**

- `path` (optional): Scan only a specific directory instead of all defaults.
- `include_registered` (optional, default `false`): Include the full list of registered files, not just the count.

**Output format:**

```json
{
  "unregistered": [
    {"path": "work/design/foo.md", "inferred_type": "design"},
    {"path": "work/spec/bar.md", "inferred_type": "specification"}
  ],
  "missing": [
    {"path": "work/plan/deleted.md", "doc_id": "PROJECT/plan-deleted"}
  ],
  "summary": {
    "total_on_disk": 142,
    "registered": 133,
    "unregistered": 7,
    "missing": 2
  }
}
```

The `inferred_type` is derived from the directory path using the same conventions as `doc import` (`work/design/` → design, `work/spec/` → specification, etc.).

### 7.3 Affected Files

| File | Change |
|------|--------|
| `internal/mcp/doc_tool.go` | Add `audit` action to the doc tool dispatch |
| `internal/service/documents.go` (or new `doc_audit.go`) | Audit logic: walk directories, compare against store |
| `internal/mcp/doc_tool_test.go` | Integration tests for the audit action |

### 7.4 Acceptance Criteria

- **AC-16.** `doc(action: "audit")` returns unregistered files found under default document directories.
- **AC-17.** `doc(action: "audit")` returns missing records whose files no longer exist on disk.
- **AC-18.** Each unregistered file includes an `inferred_type` based on its directory path.
- **AC-19.** The `path` parameter scopes the scan to a specific directory.
- **AC-20.** Files that are already registered are counted but not listed by default.

---

## 8. Tool Enhancement: `doc import --dry-run`

### 8.1 Problem

`doc import` exists but agents don't trust it — unclear what types it would infer, what titles it would assign, and whether it would produce correct results. The feedback report notes: "I opted for explicit batch registration to stay in control."

### 8.2 Design

Add a `dry_run` boolean parameter to the `doc import` action. When `true`, the import runs the full inference pipeline (directory walking, type inference, title extraction) but does not create any store records. It returns the list of documents that *would* be registered, with their inferred metadata.

**Behaviour:**

1. Walk the directory as normal.
2. For each file that would be registered, compute: path, inferred type, inferred title, inferred owner.
3. Return the results without writing to the store.

**Parameters:**

- `dry_run` (boolean, default `false`): When `true`, return what would be imported without committing.

**Output format (dry_run=true):**

```json
{
  "would_import": [
    {
      "path": "work/design/foo.md",
      "type": "design",
      "title": "Design: Foo Feature",
      "owner": ""
    }
  ],
  "would_skip": [
    {"path": "work/design/bar.md", "reason": "already registered"}
  ],
  "summary": {
    "would_import": 5,
    "would_skip": 12
  }
}
```

### 8.3 Affected Files

| File | Change |
|------|--------|
| `internal/mcp/doc_tool.go` | Pass `dry_run` parameter to import service |
| `internal/service/batch_import.go` | Add dry-run mode that skips store writes |
| `internal/mcp/doc_tool_test.go` | Test: dry_run returns results without creating records |

### 8.4 Acceptance Criteria

- **AC-21.** `doc(action: "import", path: "work/", dry_run: true)` returns the list of files that would be imported.
- **AC-22.** In dry-run mode, no document records are created in the store.
- **AC-23.** Each entry includes the inferred type, title, and owner.
- **AC-24.** Files that would be skipped (already registered) are listed with reasons.
- **AC-25.** When `dry_run` is `false` or omitted, behaviour is unchanged from current implementation.

---

## 9. Measurement Infrastructure: Evaluation Baseline

### 9.1 Problem

The V3.0 design adds mandatory gates, rewrites tool descriptions, adds stage-aware assembly, and introduces review loops. Without baseline measurements against the current system, there is no way to know whether these changes improve agent behaviour.

Both research documents are emphatic on this point:

- Skills research Theme 6: "Evaluation must precede documentation."
- Anthropic: "Start evaluating immediately with small samples — a set of about 20 test cases was enough to spot dramatic changes in early development."

The V3.0 implementation plan (workflow doc §15) currently puts the evaluation suite in Phase E — the last phase. The gap analysis report recommends moving it to Phase A. Building the baseline in 2.5 makes this possible.

### 9.2 Design

Create a set of 15–20 representative workflow scenarios as structured YAML files. Each scenario defines a starting state, an expected interaction pattern, and success criteria. The scenarios are stored in `work/eval/` and are not committed to `.kbz/`.

**Scenario structure:**

```yaml
id: eval-001
name: "Happy path: proposed → done"
description: >
  A feature starts in proposed status with no documents.
  The agent should advance it through all lifecycle stages,
  producing documents at each gate.
starting_state:
  feature_status: proposed
  documents: []
  tasks: []
expected_pattern:
  - stage: designing
    tools: [entity, doc, doc_intel, knowledge]
    output: design document registered
  - stage: specifying
    tools: [entity, doc, doc_intel, knowledge]
    output: specification document registered
  - stage: dev-planning
    tools: [entity, doc, decompose]
    output: dev-plan registered, tasks created
  - stage: developing
    tools: [handoff, next, finish]
    output: all tasks completed
  - stage: reviewing
    tools: [entity, doc, doc_intel, finish]
    output: review report registered
success_criteria:
  - feature reaches done status
  - all required documents registered and approved
  - no gate overrides used
```

**Scenario categories:**

| Category | Count | Examples |
|----------|-------|---------|
| Happy path | 3–4 | Full lifecycle, spec-only feature, plan-level spec |
| Gate failure + recovery | 3–4 | Missing spec, missing tasks, unapproved design |
| Review-rework loop | 2–3 | Single rework cycle, iteration cap reached |
| Multi-feature plan | 2–3 | Parallel features, cross-feature dependencies |
| Edge cases | 3–4 | Feature with no design stage, decompose failure, doc import |
| Tool selection | 2–3 | Agent picks correct tool for stage, avoids wrong-stage tools |

**Baseline measurement:**

For each scenario, record:

- Which MCP tools the agent calls, in what order.
- Where the agent gets stuck or picks the wrong tool.
- Whether the feature reaches the expected final state.
- Total tool calls and elapsed time.
- Any gate overrides or error recoveries.

The baseline is captured by running each scenario against the current (pre-V3.0) system. V3.0 Phase A then re-runs the same scenarios after enabling mandatory gates and compares results.

### 9.3 Affected Files

| File | Change |
|------|--------|
| `work/eval/` (new directory) | Scenario YAML files |
| `work/eval/README.md` (new) | Scenario format documentation and run instructions |

### 9.4 Acceptance Criteria

- **AC-26.** 15–20 scenario files exist in `work/eval/` covering all categories above.
- **AC-27.** Each scenario has a defined starting state, expected interaction pattern, and success criteria.
- **AC-28.** A baseline measurement is captured for at least 5 representative scenarios against the current system.
- **AC-29.** The baseline data is stored alongside the scenarios in a format that V3.0 can compare against.

---

## 10. Implementation Priority

All items are independent of each other and can be implemented in any order or in parallel.

**Recommended ordering by risk reduction:**

| Order | Item | Rationale |
|-------|------|-----------|
| 1 | `docint` AC-NN fix (§3) | Unblocks the highest-value V3.0 pipeline. Self-contained in one file. |
| 2 | Batch handler fix (§5) | Small, self-contained. Ensures batch operations are trustworthy for all subsequent work. |
| 3 | `doc audit` (§7) | Additive. No existing code changes. |
| 4 | `doc import --dry-run` (§8) | Small addition to existing import logic. |
| 5 | State isolation (§4) | Requires git integration work. Most valuable when orchestrator-workers is used heavily (V3.0). |
| 6 | Evaluation baseline (§9) | Can be built incrementally alongside the other items. |

Items 1–4 are each 1–3 hours of implementation. Item 5 is half a day. Item 6 is 1–2 days of scenario authoring and baseline capture.

---

## 11. Verification Plan

Each item has numbered acceptance criteria (AC-01 through AC-29). Verification approach per category:

| Category | Method |
|----------|--------|
| AC-01 through AC-06 (docint) | Unit tests on `asmExtractCriteria`; integration test with `decompose propose` on a fixture spec |
| AC-07 through AC-11 (state isolation) | Unit test on handoff pre-commit; integration test with dirty `.kbz/state/` |
| AC-12 through AC-15 (batch handler) | Unit tests on `ExecuteBatch` with tool-error payloads |
| AC-16 through AC-20 (doc audit) | Integration tests with fixture directories and store |
| AC-21 through AC-25 (doc import dry-run) | Integration tests: dry-run produces results but no store records |
| AC-26 through AC-29 (evaluation baseline) | Manual review: scenario files exist, baseline data captured |

All code changes must pass `go test ./...` and `go vet ./...` with no regressions.

---

## 12. Open Questions

1. **State commit scope.** Should the pre-dispatch commit include `.kbz/` files beyond `state/` (e.g., `index/`, `cache/`)? Current design limits to `state/` to minimise commit size. If other `.kbz/` directories prove fragile to stash/checkout, the scope could be widened.

2. **Evaluation scenario format.** The YAML structure in §9.2 is a starting point. Should scenarios be executable (a tool or script runs them automatically), or are they documentation for manual agent-driven evaluation runs?

3. **`doc audit` default directories.** The current design hardcodes `work/` and `docs/` subdirectories. Should this be configurable in `.kbz/config.yaml`, or is the hardcoded list sufficient for now?

4. **Consistent-front-matter implementation.** The design, specification, and implementation plan for consistent front matter exist but are unexecuted (`work/design/consistent-front-matter.md`, `work/spec/consistent-front-matter.md`, `work/plan/consistent-front-matter-plan.md`). The three tasks (T1: pipe-table parser, T2: health checks, T3: document migration) are independent of 2.5 and could be implemented in parallel. Should they be folded into 2.5, or remain a separate workstream?