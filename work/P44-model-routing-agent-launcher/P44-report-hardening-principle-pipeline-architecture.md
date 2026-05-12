# Report: Adopting the Hardening Principle via Pipeline Architecture

**Plan:** P44 — Model routing and agent launcher
**Type:** Report
**Status:** Draft
**Date:** 2026-05-12
**Source:** Conversational analysis between user and Claude Opus 4.7

## Background

This report captures an exploratory conversation about adopting the
[Hardening Principle](https://jdforsythe.github.io/10-principles/principles/hardening/)
in Kanbanzai:

> Every fuzzy LLM step that must behave identically every time must eventually
> be replaced by a deterministic tool.

The conversation explored three questions in sequence:

1. How would Kanbanzai adopt the Hardening Principle today (chat-orchestrated)?
2. How would the answer change under a code-based orchestration architecture?
3. What would a concrete pipeline look like, and is this an upgrade or a
   greenfield rebuild?

The analysis is intended as input to P44's design work, not as a specification
or implementation plan.

---

## Part 1 — Adopting the Hardening Principle in Today's Kanbanzai

### Where Kanbanzai already follows the principle

A lot of the system is already on the right side of the line. The MCP server
is essentially a hardening layer:

| Mechanical concern | Deterministic tool today |
|---|---|
| Entity lifecycle transitions | `entity(action: transition)` — explicit state machine |
| Branch/worktree isolation | `worktree`, `branch`, `merge` |
| Stage gate prerequisites | `entity` transition guards + `.kbz/stage-bindings.yaml` |
| PR creation, CI status | `pr(action: create/status)` |
| Document registration & approval | `doc(action: register/approve/audit)` |
| Knowledge dedup at write time | `finish(knowledge: [...])` rejects duplicates per-entry |
| Conflict risk between tasks | `conflict(action: check)` |
| Health and consistency sweeps | `health` |
| Context assembly for handoff | `handoff(task_id)` (already deterministic — `next` returns structured JSON) |
| Cleanup of merged worktrees | `cleanup` |

This is exactly the pattern Forsythe describes: the LLM calls a tool; the tool
does the mechanical thing identically every time.

### Where the system is still leaning on LLM judgement for mechanical work

Candidates for hardening, ordered by current pain:

#### 1. Validators (highest impact, lowest risk)

`spec-validator`, `plan-validator`, and `review-gate-validator` are LLM roles
whose job is *"is this artifact well-formed and complete?"* That's the textbook
definition of a mechanical step:

- "Every requirement has an ID" — regex.
- "Every requirement traces to a parent design section" — graph lookup against
  `doc_intel`.
- "Every task in the dev-plan references at least one requirement ID" — set
  intersection.
- "Every acceptance criterion is testable" — fuzzy, keep with LLM.
- "Section X.Y is present and non-empty" — markdown parser.
- "All referenced FEAT/TASK IDs exist" — `entity(action: get)`.

Today these checks vary run-to-run. A `kanbanzai lint spec <id>` /
`lint plan <id>` / `lint review <id>` command emitting structured findings
would (a) eliminate the "did the validator miss it this time?" failure mode
and (b) let the human-gated stage gates trust their own preflight.

The LLM validator role doesn't go away — it shrinks to *"do the requirements
actually capture the user's intent?"*, which is genuinely fuzzy.

#### 2. Document classification (`doc_intel classify`)

Currently an LLM tags fragments as `requirement | decision | rationale |
constraint | assumption | risk | question | definition | example | alternative
| narrative` with a confidence score. This is the meeting-pipeline-wikilinks
problem: the model confidently mis-tags, and downstream queries silently return
wrong sets.

Hardening path: adopt **structured markdown conventions** in spec/design/plan
templates — explicit `### Requirement REQ-001` headings, `**Decision:**` /
`**Rationale:**` prefixes, fenced metadata blocks. A parser then produces
classifications byte-identically. The LLM only classifies *narrative* sections
that don't fit the template.

#### 3. Decomposition (`decompose(action: propose)`)

A spec → task list is mostly mechanical with a fuzzy core:

- **Mechanical**: ID generation, dependency edges from "task B reads file X
  that task A writes", file-glob conflict avoidance, traceability to
  requirement IDs, completeness check.
- **Fuzzy**: choosing the slice boundaries, naming, sizing.

Today the entire output is one LLM artifact and `decompose(action: review)` is
another LLM pass. Splitting it would make `decompose(action: apply)`
reproducible from the same proposal.

#### 4. Review findings

`orchestrate-review` produces findings with severity, category, and evidence.
The fuzzy part is *spotting* the issue. The mechanical parts that vary today
and shouldn't:

- Severity/category taxonomy enforcement (schema validation).
- Evidence link resolution (file:line must exist; commit SHA must exist).
- Mapping findings to acceptance criteria (set membership, not judgement).
- Auto-finding categories: lint failures, test coverage drops, missing tests
  for new exported functions, dependency additions.

A `kanbanzai review preflight <feature>` would let human reviewers focus on
genuinely fuzzy "is this the right design?" questions.

#### 5. Retrospective synthesis (`retro(action: synthesise)`)

Clustering signals into themes is currently model-driven and varies per run,
which undermines `retro(action: create_fix, mode: auto)` (it picks different
themes each time). Embedding-based clustering with a fixed model + threshold
is deterministic. The LLM can write the *theme description* but the *theme
membership and ranking* should be reproducible.

#### 6. Commit messages and PR descriptions

Today enforced by reviewer eyeballs. A pre-commit hook + `pr(action: create)`
template assembled from entity metadata closes this.

#### 7. Knowledge tier and forward decisions

`finish(knowledge: [...])` uses LLM judgement for `tier` and `forward`
defaults. Tier-2 vs tier-3 has measurable signals. A classifier removes the
per-session drift in what gets promoted.

### Fundamental changes proposed (chat-orchestration era)

1. **Make "deterministic boundaries" a first-class concept in every SKILL.md.**
   Add a required section: *"What this skill must hand off to a tool, not do
   itself."*
2. **Introduce schemas for spec/dev-plan/review documents.** Just enough
   structural anchoring (heading patterns, ID conventions, required sections)
   that a parser can extract the machine-relevant parts losslessly.
3. **Replace LLM validators with linters that emit machine-readable findings.**
   `kanbanzai lint <doc-id>` → JSON. Stage gates consume the JSON.
4. **Split every two-phase LLM operation (`propose` → `apply`) so the second
   phase is a pure function of the first.**
5. **Add a "hardening backlog" surface.** Extend `retro` signal categories
   with `hardening-candidate` so non-deterministic LLM steps get logged,
   ranked, and surfaced.
6. **Document the determinism status of every MCP tool and skill.** A column:
   `deterministic | hybrid | fuzzy`.
7. **For consumers**, ship hardening hooks not policies. `kanbanzai lint`,
   `kanbanzai validate`, `kanbanzai preflight` as first-class CLI surfaces.

### Where the impact concentrates

Ranked by trust-deficit eliminated per unit of work:

1. Validator linters (spec / plan / review)
2. Document classification via templates
3. Decomposition split (proposal → deterministic apply)
4. Review preflight
5. Retro clustering

This ordering also reflects enabling sequence: deterministic validators make
schemas worth adopting; schemas make classification trivial; classification
makes review preflight precise; precise reviews make retro signals trustworthy.

---

## Part 2 — How the Advice Changes Under Code-Based Orchestration

### The shift in what "hardening" means

In chat orchestration, "harden" means *replace an LLM step with a tool the LLM
can call*. The LLM still decides when to call it, with what arguments, and
whether to trust the result. The probability of correct behaviour is bounded
above by the orchestrator's reliability.

In code orchestration, "harden" means something stronger: *the pipeline
decides when the LLM runs, validates its output against a schema before
continuing, and retries or fails loudly on violation*. The LLM is no longer a
decision-maker about workflow — it's a typed function in a deterministic
graph. The principle compounds: every previously-fuzzy edge between steps
becomes deterministic for free.

The mental model shifts from *"the LLM uses tools"* to *"the pipeline uses
LLMs as tools."* That inversion is the entire game.

### How earlier advice changes

| Earlier recommendation | Under code orchestration |
|---|---|
| Add lint validators for spec/plan/review | Becomes a **pipeline precondition** — no advisory step. Cannot advance from `spec_drafted` to `spec_approved` without lint passing. |
| Adopt document schemas | Becomes the **input contract for LLM steps**. A spec generation step rejects its own output if it doesn't parse, and retries with the violation in the prompt. |
| Split `decompose` propose/apply | Trivial — `propose` is a typed LLM function returning `Proposal`, `apply` is a pure function `Proposal → [Task]`. |
| Retro clustering | Scheduled deterministic step using embeddings + fixed threshold. The LLM only writes theme prose. |
| Stage bindings as YAML | The YAML *becomes the pipeline DSL*, or is replaced by it. Stage transitions are graph edges, not advisory mappings. |
| Skills as prompts agents read | Skills become **prompt templates + input schemas + output schemas + validators + retry policies**. The markdown narrative is documentation; the executable artifact is structured. |

### New advantages only available with code orchestration

These aren't extensions of the Hardening Principle — they're things chat
orchestration structurally cannot offer.

1. **Schema-enforced LLM I/O with bounded retry.** Malformed output is
   detected by code, not by the next agent down the chain noticing something
   looks off.
2. **Idempotent replay.** Each pipeline step emits an event; rerunning from
   any checkpoint produces the same artifact (cache hit) or re-runs only the
   affected subgraph.
3. **Caching of LLM calls.** Identical (prompt, model, temperature=0, schema)
   → cached result. Both cost reduction and determinism boost.
4. **Workflow-level testing.** Integration-test pipelines by mocking LLM
   steps. Cannot test *"does the chat orchestrator correctly follow the
   skill?"*
5. **Versioned, diffable workflows.** Pipeline is code in a repo. Workflow
   changes go through PRs.
6. **A/B testing of LLM steps.** Same input, two prompt variants, measure
   output quality.
7. **Cost and latency accounting per artifact.** Structured pipeline property,
   not a guess.
8. **Parallelism that's safe by construction.** Orchestrator owns the task
   dependency graph. `conflict(action: check)` becomes a graph precondition.
9. **Backpressure, rate limiting, retries with jitter.** Standard
   distributed-systems concerns.
10. **Human-in-the-loop becomes a first-class typed step.** Pipeline pauses,
    surfaces structured prompt + artifacts, resumes with structured response.
11. **Structural enforcement of invariants.** INV-001, INV-004, INV-005
    become impossible to violate, not advisory.
12. **Crash recovery.** Restart from last completed step.
13. **Multi-tenant fairness.** One orchestrator drives many features without
    cognitive collisions.

### What becomes obsolete

Much of the current MCP surface and rule set was designed to compensate for
chat-orchestrator unreliability:

- `handoff`, `next` (queue-claiming), `spawn_agent` cease being agent-facing
  tools — they become orchestrator internals.
- INV-001, INV-004, INV-005 stop being written rules and become structural
  impossibilities.
- "Check `git status` before every task" — pipeline owns the working tree.
- The `health` tool's purpose narrows.
- Stage gate prerequisite docs — pipeline edge enforces.
- Skills/role/stage-bindings registry becomes a *pipeline registry*.
- Most of `AGENTS.md` and `copilot-instructions.md` becomes shorter.

### What stays fuzzy (correctly)

- Drafting prose: specs, designs, dev-plans, research reports, retros, reviews.
- Authoring code.
- Synthesising signals into theme prose.
- Interpreting ambiguous user intent at the entry point.
- Judging whether a spec captures user intent (vs. whether it's well-formed).
- Reading code for design/quality/security review (the *judgement*).

Each becomes a typed function with input schema, prompt template, output
schema, validator, retry policy.

### New risks introduced

1. **Pipeline rigidity vs. exploratory work.** Need an explicit "exploratory
   mode" — chat orchestration over the same MCP toolkit — with a convention
   for retroactively pulling artifacts into a pipeline once the pattern
   stabilises.
2. **Pipeline definitions become a new place for bugs.** Integration tests
   essential.
3. **Onboarding cost.** Contributors read pipeline code + schemas + prompt
   templates instead of markdown skills.
4. **Schema brittleness.** Over-tight constrains useful LLM output;
   under-tight defeats the determinism win.
5. **Cache-invalidation footguns.** Stale cached results that look correct.
6. **Loss of "one chat session, one feature" mental model.** A pipeline-driven
   system is more like CI/CD for thought, less immediately legible.

---

## Part 3 — Concrete Pipeline Sketch and Migration Strategy

### The feature pipeline, end to end

Notation: **`[T]`** tool step (deterministic), **`[L]`** LLM step (typed
function), **`[H]`** human gate, **`[A]`** auto gate (code-evaluated).

#### Stage 1 — Design

```
[T] gather_design_context(feature_id)              → ContextBundle
[L] draft_design(ContextBundle)                    → DesignDraft
[T] lint_design(DesignDraft)                       → LintReport
[T] register_design_doc(DesignDraft)               → DocRecord
[A] advance_if(LintReport.passed)
    on fail: retry draft_design with LintReport as feedback (max 3)
```

`ContextBundle` includes the parent batch's design, sibling feature designs,
and knowledge entries scoped to the batch. `DesignDraft` schema requires
sections (Goals, Decisions, Alternatives, Risks).

#### Stage 2 — Specification

```
[T] gather_spec_context(design_doc)                → DesignContext
[L] draft_spec(DesignContext)                      → SpecDraft
[T] lint_spec(SpecDraft)                           → StructuralLint
    (every req has REQ-ID; every req traces to a design section;
     every AC matches a testable-shape pattern; no orphan sections)
[L] validate_spec_intent(SpecDraft, DesignContext) → IntentReport
[T] register_spec_doc(SpecDraft)                   → DocRecord
[H] human_approval(SpecDraft, StructuralLint, IntentReport)
    on reject: re-enter draft_spec with reviewer comments as input
```

The split between `lint_spec` (structural, deterministic) and
`validate_spec_intent` (judgement, fuzzy) is the textbook Hardening
application.

#### Stage 3 — Dev-plan

```
[T] gather_spec(spec_doc)                          → SpecContext
[L] propose_decomposition(SpecContext)             → Proposal
    (slices with rationale, declared file globs, dep hints)
[T] derive_task_graph(Proposal)                    → TaskGraph
    (ID generation, deps from file-glob analysis, conflict matrix)
[T] lint_dev_plan(TaskGraph, SpecContext)          → LintReport
    (every req covered, no orphan tasks, DAG acyclic, conflict matrix clean)
[L] review_decomposition(Proposal, TaskGraph, LintReport) → ReviewReport
[T] register_dev_plan_doc(Proposal, TaskGraph)     → DocRecord
[H] human_approval(...)
[T] apply_decomposition(TaskGraph)                 → [TaskEntity]
```

`derive_task_graph` is the single biggest determinism win — task IDs,
dependencies, and conflict edges become a pure function of the proposal,
not an LLM re-reasoning step.

#### Stage 4 — Develop (parallel)

```
[T] build_ready_queue(feature)                     → [ReadyTask]
    (topological sort respecting deps and conflict matrix)

# fan out, bounded by max_concurrent_workers
for_each_ready_task in parallel:
    [T] create_task_worktree(task)                 → Worktree
    [T] gather_task_context(task, worktree)        → TaskBundle
    [L] implement_task(TaskBundle)                 → Implementation
    [T] validate_implementation(Implementation)    → ValidationReport
        (lint, type-check, unit tests for changed files,
         scope guard: files_modified ⊆ declared write-set)
        on fail: retry implement_task with feedback (max 2)
    [T] commit_changes(Implementation, worktree)
    [T] finish_task(task, Implementation)

# fan in
[T] integrate_worktrees(feature)                   → IntegratedBranch
    (rebase task worktrees onto feature branch in dep order)
[A] advance_if(all_tasks_done)
```

Three things only code orchestration unlocks: bounded-parallel fan-out
without coordination drift, scope-guard enforcement as a hard precondition
(not a reviewer-spotted issue), and retry-with-feedback structurally
identical to a CI loop.

#### Stage 5 — Review

```
[T] gather_review_context(feature)                 → ReviewBundle
[T] deterministic_preflight(ReviewBundle)          → PreflightFindings
    (coverage delta, lint failures, dependency additions, missing
     tests for new exports, traceability: every req → ≥1 diff hunk)

# fan out: one LLM step per reviewer role
[L] review_security(ReviewBundle)                  → SecurityFindings
[L] review_quality(ReviewBundle)                   → QualityFindings
[L] review_testing(ReviewBundle)                   → TestingFindings
[L] review_conformance(ReviewBundle)               → ConformanceFindings

[T] merge_findings([...all...])                    → ReviewReport
[L] validate_review(ReviewReport, ReviewBundle)    → ReviewQualityReport
[T] register_review_report(ReviewReport)           → DocRecord
[H] human_approval(ReviewReport)
    on reject: auto-create rework tasks from findings flagged 'actionable',
    re-enter Stage 4
```

The four reviewer roles run in parallel. The deterministic preflight catches
*checklist* findings before any LLM reviewer runs — so LLM reviewers spend
tokens on judgement, not table-stakes.

#### Stage 6 — Merge

```
[T] merge_check(feature)                           → MergeGateReport
    (CI green, review approved, all tasks done, branch clean vs main)
[T] open_pr_if_missing(feature)                    → PR
[H] merge_approval (optional, repo-configurable)
[T] merge_execute(feature, strategy=squash)        → MergeResult
[T] cleanup_worktrees(feature)
```

#### Stage 7 — Verify

```
[T] gather_verification_context(feature)           → VerificationBundle
    (DoD items extracted from spec, merged commit, CI artifacts)
[L] verify_definition_of_done(VerificationBundle)  → VerificationReport
[T] lint_verification(VerificationReport)          → LintReport
    (every DoD item addressed, every claim has evidence link)
[A] advance_to(done) if pass
    else transition_to(needs-rework) and create rework tasks
```

### Patterns that fall out of the sketch

- **Every LLM step is sandwiched between a deterministic context-gather before
  it and a deterministic validator after it.** The LLM never reads from the
  store directly and never writes to the store directly. This is the real
  generalisation of the Hardening Principle to pipelines: not just "harden
  mechanical steps" but "deterministically bracket every fuzzy step."
- **Human gates are typed.** Structured payload in (artifact + lint +
  judgement reports), structured response out (approve/reject + comments +
  per-finding disposition).
- **Retries are local.** A failing `implement_task` retries that one task; it
  doesn't restart the stage. A rejected human approval re-enters the *prior
  LLM step* with the rejection as input.
- **Fan-out and fan-in are explicit.** Stages 4 and 5 have parallel sections
  with deterministic join points.
- **The MCP tools you already have map almost 1:1 to the `[T]` steps.**

### Greenfield vs upgrade — recommendation: **hybrid**

**Greenfield for the orchestration layer, salvage the data and tool layer.**

#### What's worth keeping

- The entity, document, knowledge, and worktree stores.
- The MCP tool surface for those stores — `entity`, `doc`, `knowledge`,
  `worktree`, `branch`, `merge`, `pr`, `conflict`, `health`, `cleanup`,
  `estimate`, `incident`.
- The Go packages underneath the MCP server (in-process performance for the
  pipeline).
- Existing user data (entities, docs, knowledge in real installs).

#### What's vestigial under code orchestration

- Skill markdown as the executable workflow definition.
- Stage bindings YAML as the workflow router.
- Role inheritance system as a runtime concern.
- `handoff`, `next`, `spawn_agent` as MCP tools.
- INV-001, INV-004, INV-005 as written rules.
- Most of `AGENTS.md` and the consumer `copilot-instructions.md`.

#### Why pure upgrade is wrong

The current architecture has accreted significant compensating machinery for
chat-orchestrator unreliability — the skill/role/stage-binding system, INV-*
rules, the "always check git status" discipline, the handoff-vs-spawn
distinction, the doc audit tool, half of `health`. These aren't bugs in the
design; they're the right design *given chat orchestration*. Carrying them
forward means perpetually translating between two paradigms, and contributors
will never know which one is canonical.

#### Why pure greenfield is wrong

You'd rebuild the entity store, doc store, knowledge graph, worktree manager,
and MCP server almost identically. That's the bulk of the existing codebase
and it's not the part with the architectural problem. You'd also lose the
ability to run chat orchestration as an *exploratory mode* alongside
pipelines.

#### The hybrid path

**Phase 1 — pipeline engine alongside existing system.** Build the workflow
engine as a new component (`kanbanzai pipeline ...` CLI or a new
`kanbanzai-orchestrator` binary). It consumes the existing Go packages
directly. The MCP server keeps running unchanged.

**Phase 2 — first pipeline end to end.** Implement the doc-publishing
pipeline (lowest stakes, well-defined stages). Run it in parallel with the
chat-orchestrated version for two to four weeks on real work. Measure:
completion rate, output consistency on identical input, cost,
time-to-completion, human intervention count.

**Phase 3 — feature/bug pipelines.** With the doc pipeline proving the
engine, port the feature pipeline (the sketch above) and the bug pipeline.
New installs default to pipeline mode; existing installs opt in per workflow.

**Phase 4 — deprecate vestigial surfaces.** Once pipelines are the default
for >80% of work, mark `handoff`, `next`, `spawn_agent`, the skill markdown
system, and stage bindings as legacy. Remove in a major version bump.

**Phase 5 — exploratory mode formalised.** What remains of chat orchestration
is rebranded as the "exploratory mode" — a thin layer over the same MCP
toolkit, used for novel work that doesn't fit a pipeline.

#### Estimated split of current codebase

- **~60% kept as-is**: stores, MCP tools for stores, Go packages, CLI
  infrastructure, lifecycle definitions, document graph, indexing.
- **~25% repackaged**: skills become prompt+schema bundles; roles become
  prompt-shaping templates loaded by LLM steps; stage bindings become
  pipeline definitions.
- **~15% deprecated**: handoff/next/spawn_agent/orchestration-specific tools,
  INV-* rules, chat-orchestrator-specific guidance.

That ratio is *why hybrid is right*. Greenfield throws away 60% of the work
for no benefit. Upgrade keeps the 15% that actively fights the new model.

#### Migration story for users

- **No data migration required.** New orchestrator reads/writes the same
  stores.
- **No flag day.** Both modes coexist; users opt in per workflow.
- **Pipelines ship as code in the Kanbanzai repo.** Consumers don't author
  pipelines initially — they consume the standard set.
- **The MCP server stays.** Both as the pipeline's internal API and as the
  exploratory-mode entry point.

---

## Summary

Under chat orchestration, the Hardening Principle reads as *"replace LLM
steps with tools one at a time."* The most valuable targets are validators,
document classification, decomposition splits, review preflights, and retro
clustering — in roughly that order.

Under code orchestration, the principle reads as *"the workflow itself is a
deterministic program that calls LLMs as typed functions."* Every LLM step
is bracketed by deterministic context-gathering and deterministic validation;
schemas enforce I/O; retries are local and use validator feedback; human
gates are typed; parallelism is safe by construction.

The recommended path is hybrid: greenfield the orchestrator, salvage the
existing toolkit, run a new pipeline alongside the old chat orchestration for
one workflow (doc-publishing) to validate the architecture, then migrate the
rest. Existing user data and MCP tools stay; skill/role/stage-binding
machinery is repackaged as pipeline definitions and prompt-template bundles;
chat-orchestration-compensating machinery (INV-*, handoff dispatch rules,
git-status discipline) is deprecated as it becomes structurally unnecessary.

## Related documents

- `P44-research-hardening-principle.md`
- `P44-research-orchestrator-architecture.md`
- `P44-design-deterministic-workflow-controller.md`
- `P44-design-feature-execution-pipeline.md`
- `P44-design-model-routing-agent-launcher.md`
- `P44-report-hardening-principle-independent-opinion.md`
