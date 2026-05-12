# Research Report: Adopting the Hardening Principle for Kanbanzai

**Prepared for:** Lead Software Architect, Kanbanzai
**Date:** 2026-05-12
**Classification:** Architectural recommendation, evidence-based
**Scope:** Evaluate adoption of the Hardening Principle — every fuzzy LLM step that must behave identically every time must eventually be replaced by a deterministic tool — across the Kanbanzai workflow system and its consumers.

**Source:** [The Hardening Principle](https://jdforsythe.github.io/10-principles/principles/hardening/) (J.D. Forsythe, 10 Principles of Working with AI Agents)

---

## 1. Executive Summary

**Key finding.** Kanbanzai's MCP server architecture already embodies the Hardening Principle's core thesis — the workflow infrastructure (entity lifecycle, document records, worktree management, decomposition, conflict detection, merge gating) is implemented as deterministic Go tools. The system's fuzzy LLM steps are appropriately concentrated in genuinely creative tasks: spec writing, code review, development orchestration, knowledge synthesis, and document drafting.

The gap is not in the *infrastructure layer* (which is well-hardened) but in the *orchestration glue* — the intermediate processing steps inside orchestrator sessions. Post-completion summarization, review finding collation, verdict aggregation, and standalone validation checks are currently LLM-driven despite having deterministic inputs and outputs. These are the highest-impact targets for further hardening.

**Three-tier assessment:**

| Layer | Hardening Status | Assessment |
|-------|----------------|------------|
| **Infrastructure** (MCP tools) | ✅ Hardened | Entity CRUD, doc records, worktree, merge, branch, conflict, decompose — all deterministic Go code |
| **Orchestration** (skill procedures) | ⚠️ Mixed | Dispatch decisions and sub-agent prompt assembly are good; post-task summarization, review collation, and verdict aggregation are still LLM-driven |
| **Creative** (document authoring, review findings) | 🟡 LLM-appropriate | Spec writing, code review, knowledge synthesis — these genuinely benefit from fuzzy reasoning and should remain LLM-driven |

**Top recommendations (full list in §8):**

1. **Formalise a "Hardened vs. Orchestration" tool taxonomy** — add a metadata annotation to every MCP tool so the API surface self-documents which steps are deterministic. (High impact, low effort.)
2. **Build `task(action: "summarize")`** — a deterministic tool that extracts a structured post-completion summary from a completed task's state record, replacing the manual orchestration summarization step. (Highest impact on orchestrator context budget.)
3. **Build `review(action: "collate")`** — a structured tool that accepts multiple structured review findings and produces a deduplicated, aggregated finding list with aggregate verdict. (High impact on review stage reliability.)
4. **Extend `doc validate` with additional deterministic checks** — spec AC-ID uniqueness, dev-plan task-to-AC traceability, dependency DAG acyclicity. (Medium impact, reduces LLM burn on validation.)
5. **Standardise the `Finding` type** across all structured tool outputs — review findings, validation findings, and decomposition warnings should share a common schema. (Foundational enabler for items 2–4.)
6. **Add a `harden` workflow action** — an MCP tool or lifecycle step that identifies fuzzy-but-determinizable sub-steps in skills, generates tool skeletons, and creates implementation tasks. (Turns the principle from philosophy into workflow.)

---

## 2. Current State: What's Already Hardened

Kanbanzai's MCP server provides ~40+ deterministic tools that replace what an LLM would otherwise do with improvised shell commands or state-file reads. The following table maps the Hardening Principle's "After" architecture (deterministic orchestration -> LLM reasoning) to Kanbanzai's existing surface:

| MCP tool | What it replaces (LLM doing it ad-hoc) | Deterministic since |
|----------|---------------------------------------|---------------------|
| `entity(action: "create\|get\|list\|update\|transition")` | LLM reading/writing YAML state files, tracking lifecycle transitions | Phase 1 |
| `doc(action: "register\|approve\|get\|list\|validate")` | LLM managing document records, checking section completeness | Phase 2 |
| `decompose(action: "propose")` | LLM parsing a spec and generating task lists — already rule-based | Phase 5 |
| `worktree(action: "create\|get\|list\|remove")` | LLM constructing `git worktree add` commands, tracking branches | Phase 4a |
| `merge(action: "check\|execute")` | LLM checking CI status, branch health, review approvals via ad-hoc API calls | Phase 4b |
| `branch(action: "status")` | LLM running `git log` and `git diff` to assess staleness and drift | Phase 4b |
| `conflict(action: "check")` | LLM guessing file overlap across features | Phase 4b |
| `knowledge(action: "list\|get\|contribute")` | LLM reading/writing `.kbz/state/knowledge/` files | Phase 2 |
| `status(id: "...")` | LLM aggregating entity states manually from multiple YAML files | Phase 2 |
| `health` | LLM running multiple independent checks across subsystems | Phase 4a |
| `pr(action: "create\|status\|update")` | LLM looking up branch names, composing PR bodies, checking CI | Phase 6 |
| `handoff(task_id: "...")` | LLM composing sub-agent prompts from scratch (INV-001) | Phase 7 |

### 2.1 The decompose tool: a case study in successful hardening

The `decompose(action: "propose")` tool at `internal/service/decompose.go:587` is a textbook application of the Hardening Principle:

1. **Step 1 — Map:** The original workflow had an LLM reading a specification, understanding acceptance criteria, and generating tasks. (Done)
2. **Step 2 — Categorize:** Task generation is deterministic — a spec's ACs are structural, task slugs follow rules, grouping decisions depend on AC count. (Done)
3. **Step 3 — Prototype:** The rule-based AC parser (`parseSpecStructure`) was built. (Done)
4. **Step 4 — Harden:** The `generateProposal` function applies rule-based grouping by section, size limits, paired test tasks, and dependency ordering. (Done)
5. **Step 5 — Pull LLM back to caller:** The LLM now calls `decompose(action: "propose")` instead of implementing proposal generation itself. (Done)
6. **Step 6 — Test independently:** The tool has Go unit tests. (Done)
7. **Step 7 — Log and monitor:** Every invocation produces action-pattern log entries. (Done)

**Result:** Task decomposition went from "LLM guesses task breakdowns from spec text" (variable quality) to "deterministic rule engine" (identical output for identical input).

---

## 3. Gap Analysis: Fuzzy Steps Remaining in Orchestrator Sessions

Despite the well-hardened infrastructure, several orchestration-level steps remain LLM-driven. These are the "orchestration glue" — intermediate processing between tool calls that the orchestrator handles manually.

### 3.1 Post-completion summarization

**The fuzzy step (from `orchestrate-development/SKILL.md`, Phase 4):**
> When a sub-agent completes a task, reduce its outcome to a post-completion summary: the task ID, 2–3 sentences describing what was built and any notable decisions, and whether it passed or failed. Discard everything else.

**Why it's fuzzy:** The orchestrator reads the sub-agent's full transcript (tool calls, reasoning, diffs, test output) and manually condenses it. Two orchestrators will produce different-quality summaries for the same task. The summary quality degrades as context fills.

**Why it's deterministic:** The data already exists in structured form — the task entity has `status`, `verification` fields; the completion records file paths. The sub-agent's output is available. Summary extraction from structured state is a mechanical transformation.

**Hardening proposal (see §7.1):** A `task(action: "summarize")` tool that reads the task record, extracts the completion data, and produces a structured 3-field summary (task_id, outcome, files_changed, short_description).

### 3.2 Review finding collation and deduplication

**The fuzzy step (from `orchestrate-review/SKILL.md`):**
> Merge structured findings from multiple sub-agent review outputs into a single consolidated list. Identify findings from different reviewers that describe the same issue at the same location and collapse them.

**Why it's fuzzy:** The orchestrator holds all review outputs in context simultaneously and manually merges. This is the canonical failure mode the Hardening Principle describes — a step that demands identical behavior (deduplication is a mechanical comparison of location + description similarity) is being done probabilistically.

**Why it's deterministic:** Review findings follow a structured format (location, severity, category, description, evidence_ref). Merging and deduplication are set operations on structured data — file X line Y:Z with description containing keyword W → merge.

**Hardening proposal (see §7.2):** A `review(action: "collate")` tool that accepts an array of structured finding lists and produces a deduplicated, verdict-aggregated result.

### 3.3 Standalone spec and dev-plan validation

**The fuzzy step:** The `spec-validator` and `plan-validator` roles use LLMs to check that specs and dev-plans are complete. The `doc validate` tool already performs some deterministic checks, but the primary validation path is still LLM-driven.

**Why it's fuzzy:** Validation checks like "are all ACs numbered uniquely?" or "does each task cover at least one AC?" are structural queries over structured text. The LLM reads the entire document to answer these, burning context on a mechanical search.

**Why it's deterministic:** These are regex/parse-tree operations. The `parseSpecStructure` function already extracts ACs and sections. Adding deterministic checkers for AC-ID uniqueness, task-to-AC traceability, and dependency acyclicity is a mechanical extension.

**Hardening proposal (see §7.3):** Extend the `doc validate` tool with additional check types: `ac-unique-ids`, `ac-to-task-traceability`, `dag-acyclic`.

### 3.4 Knowledge signal extraction from task completion

**The fuzzy step:** The `finish` tool accepts `retrospective` and `knowledge` parameters, and the `retro(action: "synthesise")` tool clusters and ranks retrospective signals. The *signal extraction* step — turning a task outcome into structured retrospective entries — is currently done by the LLM caller.

**Why it's fuzzy:** The caller decides what constitutes a "workflow friction" vs. "tool gap" signal. This categorisation is not entirely arbitrary but has no deterministic enforcement.

**Why it's partially deterministic:** The signal categories (workflow-friction, tool-gap, tool-friction, spec-ambiguity, context-gap, decomposition-issue, design-gap, worked-well) are predetermined. The mapping from completion data to category is a classification problem that could be assisted by structured prompts with constrained output.

**Assessment:** This is the line where hardening would be "hardening the wrong step." Signal extraction from completion context is genuinely fuzzy — the same outcome might be a "tool gap" in one context and a "spec ambiguity" in another. Leave this step in the LLM's domain. The structured categories and schema already constrain the output well enough.

---

## 4. Hardening Opportunities by Impact and Effort

| Candidate | Impact | Effort | Current Stage | Recommendation |
|-----------|--------|--------|--------------|----------------|
| `task(action: "summarize")` tool | High | Low | Every orchestrator session does N of these | **Harden now** |
| `review(action: "collate")` tool | High | Medium | Every review does this | **Harden now** |
| Extended `doc validate` checks | Medium | Low | Every spec/plan validation | **Harden now** |
| Standardised `Finding` type | High | Medium | Foundational | **Pre-requisite for collation** |
| Hardened tool metadata annotation | Medium | Low | One-time | **Harden now** |
| `harden` workflow action | Medium | High | Enables the rest | **Design first** |
| Knowledge signal extraction | Low | Medium | Genuinely fuzzy | **Leave as LLM** |
| Dev-plan task ordering | Low | High | Planning with trade-offs | **Leave as LLM** |
| Code review bug detection | Low | Very High | Requires genuine judgment | **Leave as LLM** |

**Assessment:** The "now" candidates share a pattern — they are mechanical transformations of already-structured data. None require introducing a new tool from scratch; they extend existing tools or add new actions to them.

---

## 5. Risks and Pitfalls

### 5.1 Premature hardening (the article's pitfall 1)

**Risk.** Building `task(action: "summarize")` before the post-completion summary format is stable. The summary schema might change as the orchestration skill evolves.

**Mitigation.** The summary schema is already well-defined in the `orchestrate-development` skill: task ID, 2–3 sentence description, files changed, pass/fail outcome. Any future schema change would be additive (new fields), not a restructuring. This is not premature.

### 5.2 Hardening the wrong step (the article's pitfall 3)

**Risk.** Attempting to harden knowledge signal extraction (section 3.4) or dev-plan task ordering (which depends on design trade-offs).

**Mitigation.** The gap analysis in §3 explicitly classifies each candidate. Only steps that take *structured* input and produce *structured* output with a *known* transformation are hardening targets. Steps whose output varies legitimately are left to the LLM.

### 5.3 Never hardening (the article's pitfall 2)

**Risk.** The orchestration glue steps (§3.1, §3.2) have been LLM-driven since Phase 7. The system has adapted around their unreliability — orchestrators develop workarounds, check outputs, re-run "just to be sure."

**Mitigation.** This very report is the mitigation. The recommendation timeline (§9) sets concrete milestones.

### 5.4 Tool proliferation without schema standardisation

**Risk.** Adding `task(action: "summarize")` and `review(action: "collate")` independently without first standardising the `Finding` type (§6.1) creates two tools with incompatible output schemas, reproducing the collation problem one layer up.

**Mitigation.** Standardise the `Finding` type before or concurrently with building either new tool.

---

## 6. Foundational Changes Required

These are changes that enable all other hardening. They are separate from the candidate builds in §7.

### 6.1 Standardise the structured output types

Currently, the system has multiple structurally analogous but type-distinct output shapes:

| Output | Location | Fields |
|--------|----------|--------|
| Review finding | `orchestrate-review` skill. Findings are structured objects with location, severity, description. |
| Decomposition warning | `decompose.go` `Proposal.Warnings` | Free-text string array |
| Validation finding | `doc validate` output | Structured finding objects |
| Health check issue | `health` tool | Structured check results with severity and recommendation |

**Proposal:** Define a shared `Finding` Go type with the following fields:

```go
type Finding struct {
    Category    string   `json:"category"`    // e.g. "gap", "oversized", "cycle", "ambiguity"
    Severity    string   `json:"severity"`    // "blocking", "warning", "info"
    Location    string   `json:"location,omitempty"`    // file path or section reference
    Description string   `json:"description"` // human-readable
    EvidenceRef string   `json:"evidence_ref,omitempty"` // link to supporting data
    Source      string   `json:"source,omitempty"`      // tool or role that produced it
}
```

All tools that produce structured findings reference this common type. This is a refactoring of existing code, not new functionality.

### 6.2 Add "hardened" annotation to tool metadata

The MCP tool definitions already support annotations (`mcp.WithReadOnlyHintAnnotation`, `mcp.WithIdempotentHintAnnotation`, etc.). Add a new hint annotation:

```go
mcp.WithHardenedAnnotation(true) // true = deterministic Go code, no LLM execution
```

This would let consumers of the tool list (downstream tools, orchestrators, documentation generators) distinguish infrastructure tools from orchestration-only tools at a glance.

### 6.3 Formally separate "orchestration glue" from "creative" in skill procedures

The `orchestrate-development` and `orchestrate-review` skills currently mix procedural instructions for the orchestrator with creative tasks for sub-agents. The procedural instructions for the orchestrator (summarize, collate, verify) should be annotated or separated so it is clear which steps are candidates for tool hardening.

**Proposal:** Mark each checklist item in skill procedures with a hardening status label:

- `🔧 Hardened` — a deterministic tool exists for this step
- `⚙️ Hardenable` — the step has deterministic I/O but no tool yet
- `🧠 LLM-appropriate` — the step genuinely benefits from fuzzy reasoning

This label can be updated as new tools are built, providing a running hardening audit.

---

## 7. Specific Tool Proposals

### 7.1 `task(action: "summarize")` — Post-Completion Summary Tool

**Input:** Task ID (TASK-xxx).

**Behaviour:**
1. Load the task entity to get `status`, `verification`, `files_modified`.
2. If status != "done" and status != "needs-review", return error: "task is not yet complete".
3. Extract from the task record: task ID, slug, summary, any sub-agent outcome stored in the record.
4. Determine the files changed from the `files_modified` field or from git diff at the last status transition time.
5. Determine pass/fail from `verification_status` (if set) or default to "pass" for done tasks.
6. Produce structured output:

```json
{
  "task_id": "TASK-042",
  "slug": "implement-spec-router",
  "outcome": "pass",
  "files_changed": ["internal/service/router.go", "internal/service/router_test.go"],
  "summary": "Implemented the spec router with section-based routing and fallback chains. Tests pass. 3 files modified, 0 existing files changed.",
  "verification_performed": true
}
```

**Use case:** Replaces Phase 4.2 of `orchestrate-development` — the orchestrator calls `task(action: "summarize", id: "TASK-xxx")` instead of reading the sub-agent transcript and manually summarising.

**Implementation notes:**
- Add to the existing `FinishTools` or create a new `TaskGroup` in the MCP server.
- The summary text is deterministic because it's assembled from structured fields in the task record plus a git diff summary. The template is fixed (2–3 sentences, always the same structure).
- Does not read the sub-agent's conversation transcript — it reads only the entity state.

### 7.2 `review(action: "collate")` — Review Finding Collation Tool

**Input:**
- `feature_id` (FEAT-xxx) — feature under review
- `review_units` — array of structured review outputs, each containing:
  - `reviewer_role` — the specialist role that produced this (e.g. "reviewer-quality")
  - `findings` — array of Finding objects with location, severity, category, description, evidence_ref
  - `per_dimension_verdicts` — verdict per review dimension

**Behaviour:**
1. Merge all findings into a single list.
2. Deduplicate: two findings are duplicates if they reference the same file path and line range AND their descriptions have high token overlap.
3. Aggregate verdicts by dimension: if any review_unit found a blocking issue in dimension X, the aggregate verdict for dimension X is "blocking".
4. Compute overall aggregate verdict: "pass" if no blocking findings, "fail" if any blocking findings, "warn" otherwise.

**Output:**

```json
{
  "feature_id": "FEAT-001",
  "aggregate_verdict": "fail",
  "blocking_count": 2,
  "warning_count": 5,
  "deduplicated_findings": [...],
  "dimension_summary": {
    "quality": "fail",
    "conformance": "pass",
    "testing": "warn",
    "security": "pass"
  },
  "duplicates_removed": 3
}
```

**Use case:** Replaces the finding collation and verdict aggregation steps in `orchestrate-review`. The orchestrator calls `review(action: "collate", ...)` instead of holding all outputs in context and manually deduplicating.

**Implementation notes:**
- This tool is a pure function: same inputs → same outputs, every time. Perfect hardening candidate.
- Deduplication logic: match on `Location` field first, then use a simple Jaccard similarity on `Description` tokens. Threshold tuneable.
- Does not write to state — it is a read-only computation that feeds the review report writing step.

### 7.3 Extended `doc validate` checks

**Proposal:** Add the following check types to the existing `doc validate` tool:

| Check ID | Applies to | Deterministic logic |
|----------|-----------|---------------------|
| `ac-unique-ids` | specification | Checks that all bold-identifier ACs (**AC-01.**, etc.) have unique IDs |
| `ac-to-task-traceability` | dev-plan | Checks that every AC in the parent spec maps to exactly one task in the plan's Task Breakdown |
| `dag-acyclic` | dev-plan | Checks that the depends_on graph forms a DAG (already exists in `decompose review`) |
| `required-sections` | all | Already exists — verifies sections from `stage-bindings.yaml` document_template.required_sections are present |
| `section-hierarchy` | all | Checks that heading levels increase by exactly 1 (no ## → #### jumps) |

**Use case:** The `spec-validator` and `plan-validator` roles can call `doc(action: "validate", id: "...", checks: ["ac-unique-ids", ...])` and get structured results, reserving the LLM for genuinely fuzzy validation (e.g. "are these ACs testable?").

### 7.4 `harden` workflow action (design sketch)

**Concept:** An MCP tool (or lifecycle action) that helps identify and build deterministic replacements for fuzzy LLM steps — turning the Hardening Principle from a philosophy into a practical workflow.

**Input:** A stage name (e.g. "reviewing") and optionally a skill section reference.

**Behaviour:**
1. Loads the skill SKILL.md for the given stage.
2. Parses the procedure and checklist for substeps marked `⚙️ Hardenable` (see §6.3).
3. For each hardenable substep, generates:
   - A Go function skeleton with well-defined input/output types
   - An MCP tool registration skeleton
   - A test skeleton
4. Writes these to a work/P{plan}/ subdirectory.
5. Creates a feature or task to implement the hardened tool.

**This is a design sketch, not a build proposal.** The Hardening Principle says "prototype with AI, harden into code." A `harden` action would be the system's way of formalising that loop — using the LLM to propose a tool, then generating the deterministic implementation task.

---

## 8. Recommendations (Priority-Ordered)

Priority is based on (impact / effort) ratio.

### Must do (high impact, low effort, time to market: 1–2 sprints)

1. **Add `mcp.WithHardenedAnnotation` to all existing tools.** One-time metadata change. Consumer tools can then distinguish hardened tools from orchestration tools programmatically.

2. **Build `task(action: "summarize")`.** The single highest-impact improvement for orchestrator context budget. Every feature with 4+ tasks spends 4+ summarization cycles manually. This tool eliminates the variability and frees context.

3. **Build `review(action: "collate")`.** The second highest-impact improvement. Review orchestration is the most context-intensive stage. Automating collation removes the step where findings are most commonly dropped.

### Should do (medium impact, medium effort, time to market: 2–4 sprints)

4. **Standardise the `Finding` type across all tools.** Before building review collation, refactor existing outputs (health check results, decomposition warnings, validation findings) to share a common schema. This is a prerequisite for clean collation and enables cross-tool analysis.

5. **Extend `doc validate` with AC-unique-IDs and AC-to-task-traceability checks.** These are straightforward parser extensions (the spec parser already exists in `decompose.go`). Reduces LLM burn on every spec validation.

### Could do (design-phase, time to market: 4+ sprints)

6. **Design the `harden` workflow action.** Scope: which stages have hardenable substeps, how to generate tool skeletons, how to integrate with the lifecycle. Do not build until items 2–4 are delivered and the pattern is proven.

7. **Add hardening status labels (`🔧`, `⚙️`, `🧠`) to all skill checklist items.** This is documentation-driven — update the SKILL.md files to annotate each step. Low effort and immediately useful for identifying the next candidate.

### Should not do (leave as LLM-appropriate)

8. **Harden knowledge signal extraction.** The mapping from completion context to retrospective category is genuinely fuzzy. The structured schema already constrains the output.

9. **Harden dev-plan task ordering.** Task ordering depends on design trade-offs and architectural constraints that a deterministic tool cannot know.

---

## 9. Implementation Timeline (Sketch)

| Sprint | Deliverable |
|--------|-------------|
| Sprint 1 | `mcp.WithHardenedAnnotation` on all tools + hardening status labels in SKILL.md files |
| Sprint 2 | Standardise `Finding` type across codebase (refactor) |
| Sprint 2–3 | `task(action: "summarize")` — prototype, test, register |
| Sprint 3–4 | `review(action: "collate")` — depends on Finding standardisation |
| Sprint 4 | Extended `doc validate` checks |
| Sprint 5+ | `harden` action design and prototype |

---

## 10. Conclusion

Kanbanzai is not starting from scratch. The MCP server is already a textbook example of the Hardening Principle's "After" architecture — a deterministic orchestration layer calling LLMs for genuinely fuzzy work. The remaining fuzzy steps are in the orchestration glue: post-completion summarization, review finding collation, and standalone validation checks.

These are the mechanical steps hiding inside the agentic workflow. Each has structured inputs, structured outputs, and a known transformation between them. Harden them, and the orchestrator's context budget improves, review reliability increases, and the system's trust foundation — the same trust foundation the article describes — moves from "it worked yesterday maybe" to "it works every time."

The metric to watch: **workflow completion rate without manual intervention** (the article's own recommended metric). Currently, orchestrator sessions that lose context mid-flow or produce variable-quality collations are the primary source of incomplete runs. Harden the glue, and the completion rate converges on 100%.
