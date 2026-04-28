| Field  | Value                                                    |
|--------|----------------------------------------------------------|
| Date   | 2026-04-25                                               |
| Status | Draft                                                    |
| Author | architect (Claude Sonnet 4.6)                            |
| Plan   | TBD                                                      |

---

## Overview

This design addresses a compound failure in the knowledge base's effectiveness as an
agent memory system. Evidence from P27–P32 and a structural audit of the knowledge
delivery pipeline reveals that the system is architecturally sound but operationally
broken: knowledge entries are being delivered to agents who cannot use them, while the
entries that would be genuinely useful are being displaced by noise. The result is that
the core promise of the system — preventing agents from rediscovering known problems from
scratch — is not being fulfilled, despite 67 entries being in the store.

A related strategic question is resolved here: the knowledge base and the doc_intel
concept graph are not competing mechanisms. They serve different memory needs. This
design clarifies the boundary, fixes the knowledge base's delivery problems, and
establishes the doc_intel concept graph as the primary mechanism for cross-plan
architectural knowledge preservation.

---

## Goals and Non-Goals

**Goals**

- Fix the signal-to-noise problem in passive context assembly so agents receive
  relevant knowledge entries, not recent retrospective notes
- Separate implementation knowledge from process/orchestration knowledge so each
  reaches the right agent
- Establish the doc_intel concept graph as the authoritative mechanism for
  cross-plan architectural knowledge, and stop expecting the knowledge base to do
  that job
- Define the correct content format for actionable knowledge entries
- Make the knowledge ranking system meaningful by activating the confirmation quality
  gate
- Ensure the same problems do not reproduce on a greenfield kanbanzai-managed project

**Non-Goals**

- Changes to the knowledge entry storage format or YAML schema
- Automatic knowledge entry generation (LLM-extracted entries without human or agent
  authorship remain out of scope)
- Changes to the doc_intel classification pipeline (covered in P32)
- New MCP tool actions — all changes are to existing tools, skills, and scoring logic
- Backfilling existing entries with the new format (entries will improve over time
  as they are confirmed, promoted, or allowed to expire)

---

## Related Work

### Prior documents consulted

| Document | Type | Relevance |
|---|---|---|
| `PROJECT/design-doc-intel-adoption-design` | Design | Defines the six P27 fixes including the knowledge retrieval mandate and knowledge curation close-out. Established the instrumentation baseline that makes this analysis possible. Directly relevant: Fix 4 (mandatory retrieval) and Fix 6 (plan close-out curation). |
| `PROJECT/report-doc-intel-usage-report` | Report | April 2026 usage audit. First document to identify `use_count = 0` for all knowledge entries and to distinguish active vs passive consumption paths. Source of the "knowledge stored but not consumed" finding. |
| `PROJECT/report-doc-intel-p27-retrospective` | Report | Post-P27/P28 retrospective. Identifies concept registry as empty, knowledge entries unconfirmed, and classification depth as shallow. Provides instrumentation data on `recent_use_count` bulk access patterns. |
| `PROJECT/research-doc-intel-recurring-issues-investigation` | Research | Root cause analysis of doc-intel compliance failures. Names the "voluntary-step architecture" as the structural cause, distinguishes five failure clusters, and establishes that three of five failures share a single enabling condition. Key source for the architectural diagnosis in this design. |
| `PROJECT/report-doc-intel-layer3-classification-pilot` | Report | Layer 3 pilot lessons. Documents the two implementation bugs, batch sizing constraints, and the five enhancement recommendations (§5.1–§5.5). All five have been shipped in P28/P32. |
| `PROJECT/design-p32-doc-intel-classification-pipeline-hardening` | Design | P32 design. Implemented `concepts_suggested` in guide response, concept tagging approval gate, and pending/register nudge enrichment. Establishes the hard enforcement point for concept tagging at `doc approve` time. |
| `FEAT-01KPTHB66Y8TM/specification-doc-intel-instrumentation` | Specification | P27 Fix 5. Defines `LastAccessedAt`, `RecentUseCount`, and `AccessCount` fields added to knowledge entries and document index. Source of the `recent_use_count` / `use_count` distinction and what each measures. |

### Constraining decisions

| Decision | Source | Constraint on this design |
|---|---|---|
| Knowledge surfacing cap of 10 entries per context assembly | `internal/context/surfacer.go` (`defaultMaxSurfacedEntries`) | The cap exists to protect context budget. Any fix to relevance filtering must work within 10 entries, not increase the cap. |
| Scoring formula: `confidence × recencyMultiplier × confirmedBoost` | `internal/knowledge/score.go` | The scoring mechanism is already well-designed for a world where confirmation is used. This design activates it rather than replacing it. |
| `scope: "project"` entries match all tasks | `internal/knowledge/surface.go` (`matchesAlways`) | The current matching logic makes project-scope a catch-all. This design constrains which entry types may use project scope without changing the matching logic itself. |
| Passive delivery (context assembly) is the primary path | `internal/mcp/handoff_tool.go` (`renderHandoffPrompt`) | Knowledge is rendered inline into the handoff prompt. Agents do not need to call `knowledge get` to receive it. The `use_count = 0` finding is expected behaviour, not a failure. |
| `finish(retrospective: [...])` is the primary contribution path for tier 3 entries | `implement-task/SKILL.md`, `finish_tool.go` | Retrospective signals will continue to be the dominant tier 3 contribution mechanism. This design manages the delivery impact of that signal, not the contribution mechanism itself. |

### Open questions from prior work

The recurring-issues investigation (§6.3, RG-3) asked whether the knowledge confirmation
failure is structural (sub-agents cannot confirm what they don't know they used) or
cultural (the close-out step exists but isn't being run). This design takes a position:
it is both, and the fix addresses both.

The investigation also flagged (§4) that the "voluntary-step architecture" is the root
cause of enrichment failures across classification, concept tagging, and knowledge
confirmation. This design focuses on the knowledge base specifically. The broader
voluntary-step question as it applies to doc_intel classification is governed by P32's
concept tagging approval gate.

---

## Problem and Motivation

### 1. The `use_count = 0` finding is real but misdiagnosed

The April 2026 usage report identified `use_count = 0` across all 67 knowledge entries
and called it a significant gap. This alarmed. However, reading the code clarifies what
it actually means.

The `handoff` tool's context assembler calls `Surfacer.Surface()` and injects the
returned entries directly into the sub-agent's prompt under a "Known Constraints" section.
Agents receive knowledge passively — it is in their context when they start work. The
`use_count` counter increments only on explicit `knowledge(action: "get")` calls. Since
the passive path already delivers the knowledge, there is no reason for an agent to make
an explicit retrieval call. `use_count = 0` is expected behaviour given the architecture.

The real question is not "why aren't agents calling `knowledge get`?" It is: **is the
passive delivery path delivering anything worth acting on?**

### 2. The surfacer delivers the 10 most recently created entries, not the 10 most relevant

The scoring formula is `confidence × recencyMultiplier × confirmedBoost`. In the current
corpus, every entry has `confidence: 0.5` (the default for contributed entries) and
`confirmedBoost: 0.8` (contributed, not confirmed). With base confidence and boost
identical for all 67 entries, the ranking is determined entirely by `recencyMultiplier`,
which is `1 / (1 + days/90)`. The surfacer therefore selects the 10 most recently
*created* entries.

This inversion means:

- `mcp-thin-adapter-pattern`, `error-handling-conventions`, `canonical-yaml-field-ordering`
  — tier 2 entries contributed in March 2026 that describe permanent architectural
  conventions every implementer needs — score near the bottom. They are consistently
  excluded from the 10-entry cap.
- `[minor] worked-well: T1-T4 parallelism plan held exactly`, `[moderate] worked-well:
  Sub-agent delegation per editorial pipeline stage was effective` — tier 3 retrospective
  notes from April 2026 — score near the top and are surfaced to every implementation task.

The scoring system has the right design. The confirmation quality gate (`confirmedBoost:
1.0` vs `0.8`) would differentiate proven entries from speculative ones. But the gate has
never been activated. The Phase 6 knowledge curation close-out, mandated in
`orchestrate-development/SKILL.md` since P27, has never been run for any plan. Without
confirmed entries, recency dominates by default.

### 3. ~50% of the knowledge pool is wrong content type for implementation context

Of 67 entries, approximately 35 are `finish(retrospective: [...])` signals:
`[minor] worked-well: ...`, `[moderate] workflow-friction: ...`,
`[moderate] tool-friction: ...`. These are planning-level observations. They are useful
input for sprint retrospectives, design decisions, and tool improvement proposals. They
are not useful to an implementer sub-agent working on a Go function right now.

An agent implementing a new MCP tool handler receives a "Known Constraints" section
containing notes about parallelism patterns, heredoc failures in terminals, and editorial
pipeline success stories. The two or three entries that actually pertain to MCP tool
implementation are either capped out by recency (if they are older tier 2 entries) or
absent entirely.

The signal-to-noise ratio in the context packet is poor enough that agents plausibly
ignore the section. The knowledge entries from the implement-task skill examples — the
rate-limiter initialisation pitfall that would have saved two hours of debugging — only
work if the "Known Constraints" section has a credible signal-to-noise ratio. It does not.

### 4. `scope: "project"` is used universally, eliminating all relevance filtering

Every knowledge entry in the store has `scope: "project"`. The `MatchEntries` function
treats `scope == "project"` as a match for every task regardless of context. This means
the surfacer's matching stage performs no filtering at all — it passes all 67 entries to
`RankAndCap`, which selects 10 by recency. The file-path scoping and tag-matching logic
that was designed to provide relevance filtering is bypassed entirely because it is never
used.

An entry about `mcp-thin-adapter-pattern` and an entry about terminal heredoc failures
and an entry about parallelism plan quality all look identical to the surfacer. All
three are project-scoped. All three have confidence 0.5. All three are contributed.
Only creation date differentiates them.

### 5. The knowledge base is attempting the wrong job for architectural knowledge

The system was designed partly to address generational knowledge loss as the project
grows beyond 200 features. A new agent working on feature 201 should know about
patterns and constraints established in features 1–50.

The knowledge base cannot do this job effectively. A knowledge entry that says "stage
gates work as described in the lifecycle-integrity design" is a pointer to a document,
not the knowledge itself. The document intelligence concept graph — when `find(concept:
"stage-gate")` returns the five design documents that introduced and refined the concept
— provides far richer and more trustworthy cross-plan memory than any knowledge entry.

The knowledge base and doc_intel are not the same tool. Conflating their roles produces
knowledge entries that try to summarise architectural decisions in 3 sentences, which is
always worse than reading the source design document.

### 6. These problems will reproduce on a greenfield project

If a new kanbanzai-managed project starts without changes to the knowledge contribution
mechanism:

- `finish(retrospective: [...])` will produce tier 3 retrospective signals as the
  dominant entry type from sprint 1
- All entries will default to `scope: "project"`, bypassing relevance filtering
- The confirmation close-out will not run until the culture is established
- The 10-most-recent entries will dominate, regardless of relevance

The TTL mechanism (30-day expiry for tier 3) does provide natural pool improvement over
time. But in the first 30 days of any plan, the pool will be dominated by that plan's
retrospective signals unless the taxonomy and scoping conventions are enforced from the
start.

---

## Design

The design has three components, addressable independently but most effective together.

### Component 1: Delivery channel taxonomy

Introduce a `channel` field on knowledge entries with three values:

- `implementer` — surfaced in implementation task context packets (sub-agent handoffs
  for `developing`-stage tasks). Examples: Go patterns, file-editing conventions,
  MCP tool gotchas, anti-patterns specific to a package.
- `orchestrator` — surfaced in orchestration context packets (handoff for
  orchestrator-role agents). Examples: dispatch patterns, context management
  strategies, worktree handling, plan close-out procedures.
- `policy` — surfaced in all context packets regardless of role, capped at 3 per
  assembly to prevent displacement of role-specific entries. Examples: flaky test
  policy, review bug-filing policy, commit message format.

The `Surfacer` filters by channel against the agent role present in the handoff call.
An implementer sub-agent receives up to 10 entries: up to 7 `implementer` entries +
up to 3 `policy` entries. An orchestrator receives up to 10 entries: up to 7
`orchestrator` entries + up to 3 `policy` entries.

Retrospective signals (`[minor] worked-well: ...`, `[moderate] workflow-friction: ...`)
have no channel. They are excluded from context assembly entirely. They serve their
purpose as inputs to `retro synthesise` and sprint planning; they are not actionable
instructions. The `finish(retrospective: [...])` mechanism continues to contribute them
to the knowledge store for retrospective analysis; they are simply never surfaced in
implementation or orchestration context packets.

**Migration:** Existing entries without a `channel` field default to `policy` (safe
fallback — they appear in all contexts at low priority). The orchestrator's Phase 6
curation pass assigns channels when confirming entries. New entries from contribution
workflows are required to specify a channel.

### Component 2: Scope specificity requirement for technical entries

The `implementer` channel requires a non-project scope. A knowledge entry with
`channel: implementer` must have a `scope` value that is a file path prefix:
`internal/mcp/`, `internal/knowledge/`, `internal/storage/`, etc. This is validated at
contribution time (the `knowledge contribute` action rejects `implementer` entries with
`scope: "project"`).

Why: the surfacer's `matchesFilePath` logic already handles this correctly. An entry
scoped to `internal/mcp/` only appears in handoffs for tasks whose file scope includes
MCP tool files. This is exactly the right behaviour — an agent implementing a new entity
service function does not need to know about MCP struct tag gotchas.

`orchestrator` and `policy` entries may retain `scope: "project"` because they apply
across all contexts by design.

The effect: instead of 67 project-scoped entries all competing for 10 slots, the
surfacer assembles a set drawn from genuinely relevant scoped entries plus universal
policy entries. The cap of 10 remains; the content within it becomes meaningful.

### Component 3: Actionable entry format mandate

Knowledge entries in the `implementer` and `orchestrator` channels must follow an
imperative instruction format. The content must answer the question "what should the
agent do (or not do)?" rather than "what was observed."

**Rejected format (retrospective/narrative):**
> `[moderate] worked-well: Python-based file editing via terminal was reliable for
> worktree modifications across 8 files.`

**Required format (actionable instruction):**
> When writing files in worktrees, always use `write_file(entity_id: ...)`. Do not
> use `edit_file`, terminal heredocs, or python3 string embedding. The `edit_file`
> tool writes to the main project root, not the worktree. Heredocs fail with embedded
> double quotes in Go source code.

The difference: the first format is a historical observation. An agent reading it must
infer an instruction. The second format IS the instruction. It states the rule, names
the alternatives to avoid, and gives the reason in one compact entry.

This is a skill-level convention enforced by the write-design and implement-task skills,
not by code validation. The `knowledge contribute` MCP action surfaces a reminder:
"For implementer and orchestrator entries, write as an actionable instruction: 'When
X, do Y. Do not do Z because W.'"

### Component 4: Activating the confirmation quality gate

The scoring formula already rewards confirmed entries (`confirmedBoost: 1.0` vs `0.8`).
What it needs is confirmed entries to reward.

The fix is not a code change. It is establishing the Phase 6 close-out as a required
deliverable. The orchestrate-development skill already mandates the knowledge curation
pass at Phase 6. What is missing is accountability: the plan close-out is not complete
until the curation pass has run. A plan review finding of "Phase 6 close-out not
executed" should block plan completion with the same severity as an unmerged feature.

Additionally, sub-agents in the implement-task skill are given a cleaner instruction:
rather than "confirm/flag entries you used," the instruction becomes "confirm entries
whose content you applied; flag entries that gave you incorrect or misleading guidance."
The distinction matters: agents cannot confirm what they only passively received; they
can confirm what they actively applied.

For the backlog: the orchestrator runs the Phase 6 curation pass for all plans from P25
onwards. Tier 2 entries confirmed during that pass will score meaningfully higher than
unconfirmed entries and begin displacing retrospective noise in the rankings naturally.

### Component 5: Strategic boundary clarification — knowledge base vs doc_intel

This component requires no code changes. It requires a design decision that is made
explicit here and propagated to the relevant skill files.

**The knowledge base handles operational, task-level facts:**
- This tool has this gotcha when called this way
- This file must be modified in tandem with that file
- This test pattern fails under parallel execution
- This Go convention applies throughout the codebase

These are sharp, specific facts that do not require context to apply. An implementer
can act on them directly.

**The doc_intel concept graph handles architectural and cross-plan knowledge:**
- Why does the stage gate work the way it does? → find the lifecycle-integrity design
- What decisions were made about the entity model? → find the entity-structure design
- Which plans have touched the knowledge system? → find all documents tagged with
  concept "knowledge-base"
- What did we learn about agent orchestration at scale? → find research and retrospective
  documents classified with concept "sub-agent-dispatch"

These are questions about the history, rationale, and evolution of the system. A
knowledge entry cannot answer them; a classified design document can. The concept graph's
`find(concept: "X")` query is the right tool for generational knowledge preservation at
200+ features.

**The implication for skill files:** The `implement-task` skill's Phase 1 instruction
("Call `knowledge(action: "list")` with domain-relevant tags") should be accompanied
by: "For questions about *why* something is designed the way it is, or what prior plans
have said about this area, use `doc_intel(action: "find", concept: "X")` rather than
searching the knowledge base."

The knowledge base and doc_intel are complementary. They answer different questions.
Agents should be directed to the right tool for the right question.

---

## Alternatives Considered

### A. Increase the surfacing cap from 10 to 20 entries

Surfacing more entries would reduce the chance that relevant ones are excluded by the
cap. The immediate implementation is trivial — one constant change in `surfacer.go`.

**Rejected because:** it treats the symptom rather than the cause. The problem is not
that the cap is too small; it is that the wrong 10 entries are being selected. Doubling
the cap to 20 doubles the amount of irrelevant retrospective noise in every context
packet, worsening the signal-to-noise ratio and increasing context budget consumption.
The cap exists for a reason: context is finite and expensive.

### B. Replace passive delivery with explicit retrieval mandates

Remove knowledge from context assembly entirely and require agents to call
`knowledge(action: "list")` explicitly in their Phase 1 checklist.

**Rejected because:** this is precisely the mechanism that the usage data shows does not
work. `use_count = 0` despite the explicit checklist mandate in implement-task. Agents
are not reliably calling the tool. Passive delivery is the correct model for AI agents —
information in context is read; information requiring a tool call is skipped. The problem
with the current passive delivery is content quality, not the delivery mechanism.

### C. Automatic LLM-based knowledge extraction from documents

Have a pipeline agent read completed feature docs and automatically generate knowledge
entries, bypassing the manual contribution requirement.

**Rejected because:** automatic extraction produces the wrong content type. An LLM
reading a specification will extract summaries and descriptions — narration, not
actionable instructions. The most valuable knowledge entries are about things that
*went wrong* or are *unexpectedly constrained* — information that is usually not in
specifications, but in agent retrospectives and debugging sessions. Manual contribution
with better format guidance produces higher-quality entries than automatic extraction.

### D. Replace the knowledge base with a structured ruleset

Define a static ruleset of conventions (Go patterns, file handling, tool usage) that is
embedded in the relevant skill files, removing the dynamic knowledge base entirely.

**Rejected because:** static rules cannot capture project-specific discoveries. The
value of the knowledge base is precisely that it accumulates discoveries made during
development of *this project* — things that are not in general Go documentation or the
kanbanzai skill files. The rate-limiter initialisation pitfall example from the
implement-task skill is real: it is something discovered in this codebase, not a general
Go rule. Static skill files would need to be updated manually for every such discovery.
The knowledge base automates that accumulation with lower friction.

### E. Keep the current system and rely on TTL to improve the pool over time

Tier 3 entries expire after 30 days. The retrospective noise will naturally cycle out.
As the project matures and entries are confirmed, the ranking will improve on its own.
Take no action.

**Rejected because:** the pool does not improve fast enough without intervention. The
backlog of existing tier 2 unconfirmed entries (35 entries, none confirmed) will not
expire — they have 90-day TTL and will remain in the pool indefinitely until confirmed
or retired. Without the Phase 6 confirmation close-out running, the tier 2 entries are
permanently unconfirmed and the confirmation quality gate is permanently inactive.
Additionally, the content format and scope discipline problems will reproduce on every
new plan regardless of TTL improvement.

### F. Status quo — do nothing

Accept that the knowledge base is a write-only append log and use it only as input to
retrospective reports via `retro synthesise`.

**Partially accepted as a realistic baseline for retrospective signals.** The
`finish(retrospective: [...])` tier 3 entries are genuinely useful as retrospective
inputs and not much else. This design accepts that and routes them accordingly
(no channel = excluded from implementation context). For the tier 2 operational and
convention entries, status quo means permanent underperformance that will worsen as
the project grows. Rejected for those entry types.

---

## Decisions

**Decision 1: `use_count = 0` is not a failure indicator for the knowledge system**
- **Context:** The April 2026 usage report identified `use_count = 0` as a significant
  gap, prompting concern that agents are not using the knowledge base.
- **Decision:** `use_count` measures explicit `knowledge get` calls, which are not how
  the system delivers knowledge. The passive delivery path (context assembly via
  `handoff`) is the primary mechanism and is working. `use_count = 0` is expected
  behaviour given the architecture, not evidence that knowledge is being ignored.
- **Rationale:** The code confirms this: knowledge is injected into the prompt at
  `handoff` time. There is no reason for an agent to call `knowledge get` if the
  content is already in their context. Fixing `use_count` would require restructuring
  the delivery mechanism in a way that conflicts with how LLMs process context.
- **Consequences:** `recent_use_count` remains the meaningful passive-path metric.
  The `use_count` field should be retained but reframed in documentation as
  "explicit retrieval count" with the expectation that it will typically be low.
  Monitoring focus should be on whether confirmed entries are displacing noise
  in the ranked cap, not on `use_count`.

**Decision 2: Retrospective signals are excluded from implementation and orchestration context packets**
- **Context:** ~50% of knowledge entries are `finish(retrospective: [...])` process
  observations. They are useful for sprint planning and retrospective analysis but
  noise in implementation context packets.
- **Decision:** Entries without a `channel` field, and entries contributed via the
  retrospective path, are never surfaced in implementation or orchestration handoffs.
  They remain accessible via `knowledge list` for retrospective and planning purposes.
- **Rationale:** The signal-to-noise ratio in context packets is the primary driver
  of whether agents act on knowledge entries. An agent that receives 10 entries of
  which 7 are irrelevant process observations will, rationally, stop reading the
  "Known Constraints" section. This behaviour is correct from the agent's perspective
  and is precisely the failure mode we need to prevent.
- **Consequences:** Tier 3 retrospective entries effectively become planning-tool-only
  artifacts rather than implementation-context artifacts. The `retro synthesise` and
  `knowledge list` tools remain the right consumers of these entries.

**Decision 3: The knowledge base handles operational facts; doc_intel handles architectural knowledge**
- **Context:** The system was partly designed to address generational knowledge loss at
  200+ features. The knowledge base and doc_intel concept graph have been conflated as
  alternative solutions to the same problem.
- **Decision:** They are not alternatives. The knowledge base handles sharp, task-level
  operational facts. The doc_intel concept graph handles the why — cross-plan design
  decisions, architectural rationale, and concept evolution. Agents should be directed
  to `knowledge list` for the first type and `doc_intel find(concept: "X")` for the
  second type.
- **Rationale:** A knowledge entry cannot convey the reasoning behind a multi-plan
  architectural decision. A classified design document can. The concept graph's query
  surface (`find(concept: "stage-gate")`) provides richer, more trustworthy,
  more contextualised access to architectural history than any knowledge entry.
  The empty concept registry is therefore a more serious gap for generational knowledge
  preservation than any knowledge base quality problem.
- **Consequences:** The concept backfill campaign (classifying top 50 approved
  specifications and designs from P15–P27 with `concepts_intro` populated) becomes a
  higher priority than knowledge base quality improvements. Skill files should direct
  agents to the right tool for the right question.

**Decision 4: Scope specificity is enforced for `implementer` channel entries**
- **Context:** All existing entries have `scope: "project"`, which bypasses relevance
  filtering entirely. The surfacer's file-path matching logic is unused.
- **Decision:** Entries with `channel: implementer` are required to have a file-path
  scope prefix (e.g., `internal/mcp/`, `internal/storage/`). Entries with
  `channel: orchestrator` or `channel: policy` may retain `scope: "project"`.
- **Rationale:** File-path scoping is the most direct way to ensure that an MCP
  implementation task receives MCP-relevant knowledge and an entity service task
  receives entity-service-relevant knowledge. The matching logic already handles this
  correctly; the convention simply needs to be enforced at contribution time.
- **Consequences:** Writing a good `implementer` entry requires knowing which files it
  applies to. This is a small additional friction at contribution time that yields
  substantial relevance improvement at delivery time. Entries that genuinely apply
  across all implementation work (e.g., `write_file` vs `edit_file` for worktrees) are
  valid candidates for `channel: policy` with `scope: "project"`.

**Decision 5: Phase 6 confirmation close-out is a plan completion gate**
- **Context:** The orchestrate-development Phase 6 close-out procedure includes a
  knowledge curation pass that has never been run for any plan.
- **Decision:** A plan review finding of "Phase 6 close-out not executed (knowledge
  curation pass skipped)" is treated as a blocking finding by the reviewer-conformance
  role. Plans are not closed until the curation pass has run.
- **Rationale:** The scoring system's confirmation quality gate (`confirmedBoost: 1.0`
  vs `0.8`) is the primary mechanism by which proven entries outrank speculative ones.
  It is inert without confirmed entries. One close-out pass per plan is sufficient to
  build a confirmation corpus over time. Without this gate, the close-out will continue
  to be deferred indefinitely.
- **Consequences:** The first plan to enforce this gate will require retroactive
  close-outs for P25–P32. This is approximately 30–35 tier 2 entries across those
  plans. A one-time catch-up session is appropriate; the ongoing per-plan cost is low.

---

## Dependencies

| Dependency | Type | Notes |
|---|---|---|
| `internal/context/surfacer.go` | Code | Add channel filtering to `Surface()`. Requires `channel` field to be readable from entry maps. Small change — one filter pass before `RankAndCap`. |
| `internal/knowledge/surface.go` (`MatchEntries`) | Code | Add channel parameter to `MatchInput`. Pass the agent role's expected channel from the handoff tool. |
| `internal/mcp/handoff_tool.go` | Code | Pass agent role / channel to the surfacer call. The role is already known at handoff time from the stage binding lookup. |
| `internal/validate/knowledge.go` | Code | Add validation: `channel: implementer` entries must have a non-project scope. |
| `internal/model/knowledge.go` | Code | Add `channel` field to the knowledge entry model (no schema migration needed — YAML is additive). |
| `implement-task/SKILL.md` | Skill | Update Phase 1 knowledge retrieval instruction to clarify "active application" vs "passive receipt" for confirm/flag. Add guidance to use `doc_intel find(concept:)` for architectural questions. |
| `orchestrate-development/SKILL.md` | Skill | Strengthen Phase 6 curation pass from advisory to plan-completion gate. |
| `kanbanzai-documents/SKILL.md` | Skill | Add `channel` field guidance to the knowledge contribution section. |
| `reviewer-conformance/SKILL.md` | Skill | Add "Phase 6 close-out not run" as a blocking finding class. |
```

Now let me save that file: