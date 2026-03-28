# Workflow Retrospective Design

- Status: draft design
- Purpose: define a feedback loop that captures process observations during work and synthesises them into actionable workflow improvements
- Date: 2026-03-27T13:21:27Z
- Related:
  - `work/design/kanbanzai-2.0-design-vision.md` §4 (P4: invisible infrastructure), §5.2 (`finish`), §6.2 (knowledge system)
  - `work/design/machine-context-design.md`
  - `work/design/workflow-design-basis.md`
  - `work/design/quality-gates-and-review-policy.md`

---

## 1. Problem

Kanbanzai has feedback loops for factual knowledge. The knowledge system captures what agents learn about the codebase ("the API returns 404 not 410"), confidence-scores it, and retires entries that turn out to be wrong. This works.

What Kanbanzai does not have is a feedback loop for the workflow itself. When a task is difficult because the spec was ambiguous, or because the context packet missed a critical convention, or because the decomposition produced tasks that were too large — that observation is lost. The `blockers_encountered` field on `complete_task` captures it as a free-text string on the task YAML, but nothing aggregates it, nothing surfaces patterns, and nothing closes the loop by turning recurring friction into a concrete change.

Human teams solve this with retrospectives: periodic structured reflection on what worked, what didn't, and what to try next. The insight behind this proposal is that the same mechanism works for agent-driven workflows — potentially better, because agents can generate structured observations at volume and without the social friction that makes human retrospectives difficult.

The goal is to make the workflow self-improving: each cycle of work produces observations about the process; periodically those observations are synthesised into improvement candidates; improvements are tried and measured; the ones that work are kept.

---

## 2. Design Principles

### 2.1 Ride the existing infrastructure

The knowledge system already has tagged entries, confidence scoring, TTL-based expiry, compaction, and querying by scope and tag. Retrospective signals should be knowledge entries with a specific tag convention, not a new entity type. This avoids schema changes, new storage paths, new query mechanisms, and new tools.

### 2.2 Capture at the natural moment

The moment an agent completes a task is the moment it has the freshest, most specific observations about what went well and what didn't. Retrospective signals are captured at task completion — the same moment that already captures knowledge entries, blockers, and verification results. No separate step, no second tool call.

### 2.3 Structured signals, not free text

Categories, severity, and optional suggestions make signals aggregatable. Free-text observations are useful for context but not for pattern detection. The structure is what turns individual grumbles into visible themes.

### 2.4 Synthesis is explicit, not automatic

Retrospectives happen when someone decides it's time to reflect — not after every task, and not on a fixed schedule. The synthesis step is a tool call (or a conversation), triggered by a human or orchestrator when a meaningful body of work is complete.

### 2.5 Try, measure, decide

Improvements are experiments. They are recorded as decisions, tracked through subsequent cycles, and evaluated based on whether the signals they were meant to address actually decrease. If they don't, the experiment is reverted or revised.

---

## 3. What Exists Today

Three existing mechanisms partially address retrospective concerns:

| Mechanism | What it captures | Limitation |
|---|---|---|
| `blockers_encountered` on `complete_task` | Free-text obstacles during the task | Written to a field on the task YAML; never aggregated, never surfaced again. Write-only memory. |
| `knowledge_entries` on `complete_task` | Factual knowledge learned during work | Scoped to codebase facts, not process observations. "The API uses UTC timestamps" — not "the spec didn't define the timezone convention." |
| `context_report` after task completion | Which knowledge entries were useful vs. wrong | Feedback on the knowledge *store*, not on the workflow, the tooling, or the documents. |

The gap: nothing captures "the process itself was the problem," and nothing aggregates per-task observations into cross-task patterns.

---

## 4. Retrospective Signals

### 4.1 Definition

A retrospective signal is a structured observation about the workflow, tooling, or process — as distinct from factual knowledge about the codebase. It records what an agent (or human) noticed about *how the work was done*, not *what the work produced*.

### 4.2 Structure

Each signal has:

| Field | Required | Description |
|---|---|---|
| `category` | Yes | What kind of process issue this is (see §4.3) |
| `observation` | Yes | What happened, in one or two sentences |
| `suggestion` | No | What could be done differently |
| `severity` | Yes | How much friction this caused: `minor`, `moderate`, `significant` |
| `source` | Auto | `agent` or `human` — set automatically based on who contributed it |

### 4.3 Categories

Categories are fixed strings that enable aggregation. They are chosen to cover the main areas where workflow friction occurs:

| Category | Meaning | Example |
|---|---|---|
| `workflow-friction` | A workflow step was unnecessarily difficult, slow, or confusing | "Had to manually transition through three statuses to reach the one I needed" |
| `tool-gap` | A tool was missing, or an existing tool didn't support what was needed | "No way to query all tasks across features in a plan without multiple calls" |
| `tool-friction` | A tool exists but was awkward to use, required too many parameters, or returned unhelpful output | "complete_task requires task_id but the agent only has the slug at this point" |
| `spec-ambiguity` | The specification was unclear, incomplete, or contradictory on a point that mattered during implementation | "Spec says 'handle errors appropriately' without defining what appropriate means here" |
| `context-gap` | Information that was needed during the task was not in the context packet and had to be found manually | "Error format convention exists as a knowledge entry but wasn't surfaced for this task" |
| `decomposition-issue` | Tasks were too large, too small, poorly ordered, or had missing dependencies | "This task depended on a database migration that was buried inside a different task" |
| `design-gap` | The design document didn't address something that turned out to matter during implementation | "Design didn't consider the case where the user has no plan entities yet" |
| `worked-well` | Something about the process worked particularly well and should be preserved | "The vertical slice decomposition meant each task was independently testable" |

The `worked-well` category is important. Retrospectives that only capture problems produce a distorted view. Knowing what to preserve is as valuable as knowing what to change.

New categories can be added as minor schema changes. The synthesis step treats unrecognised categories as valid but unclustered.

### 4.4 Storage

Signals are stored as knowledge entries with the following conventions:

- **Tier**: 3 (session-level; 30-day TTL — signals that aren't synthesised within 30 days are stale by definition)
- **Scope**: `project`
- **Tags**: `retrospective`, plus the category as a second tag (e.g., `retrospective`, `tool-friction`)
- **Topic**: `retro-{task-id}` (or `retro-{slug}` for human-contributed signals outside a task context)
- **Content**: A structured string combining the signal fields:

```
[severity] category: observation. Suggestion: suggestion.
```

For example:

```
[moderate] spec-ambiguity: Spec says "handle errors appropriately" without defining error format. Suggestion: Add error format convention to the project-level spec template.
```

This is deliberately a formatted string rather than structured YAML within the content field. Knowledge entries have a single `content` string. Putting structure inside it (as JSON or YAML) would complicate the knowledge system's deduplication and compaction, which operate on string similarity. A formatted string with a consistent prefix pattern gives synthesis enough structure to parse while remaining compatible with existing knowledge infrastructure.

### 4.5 Why Not a New Entity Type?

A new entity type (e.g., `RetroSignal`) would require:

- A new storage directory and store implementation
- New model types, lifecycle states, and validation
- New MCP tools for CRUD operations
- Schema version bump
- Viewer support

Knowledge entries require none of this. The tags and conventions in §4.4 are sufficient to distinguish retrospective signals from factual knowledge. The `knowledge_list(tags: ["retrospective"])` query returns exactly the signals. Compaction, TTL, and pruning work as designed.

If volume or query complexity eventually outgrows knowledge entries, a dedicated entity type can be introduced in a future version. Starting with knowledge entries is the lower-risk choice.

---

## 5. Signal Collection

### 5.1 At Task Completion (Agent)

The `finish` tool (2.0) or `complete_task` tool (current) gains an optional `retrospective` parameter. This is an array of signal objects:

```
finish(
  task_id: "TASK-...",
  summary: "Implemented the billing webhook handler",
  retrospective: [
    {
      category: "spec-ambiguity",
      observation: "Spec did not define retry behaviour for failed webhook deliveries",
      suggestion: "Add retry policy section to webhook specs",
      severity: "moderate"
    },
    {
      category: "worked-well",
      observation: "Context packet included the billing API idempotency knowledge entry, which saved a round of debugging",
      severity: "minor"
    }
  ]
)
```

Each signal in the array is stored as a knowledge entry per §4.4. The response includes the created entry IDs alongside the other completion metadata.

The `retrospective` parameter is optional. Tasks with no process observations simply omit it. There is no enforcement — agents are not required to produce signals, and the system does not prompt for them. The skill instructions (§8) are sufficient to encourage contribution when there is something worth noting.

### 5.2 Standalone (Human or Agent)

Observations that arise outside the context of a specific task — during planning conversations, design reviews, or general usage — can be contributed directly through the knowledge system:

```
knowledge_contribute(
  topic: "retro-planning-session-2025-03-27",
  content: "[moderate] workflow-friction: Creating features requires an approved design document, but during early exploration we want to sketch features before committing to a design. Suggestion: Allow a 'draft' plan that can hold provisional features.",
  scope: "project",
  tier: 3,
  tags: ["retrospective", "workflow-friction"]
)
```

This works today with no tool changes. The convention is documented; the infrastructure exists.

### 5.3 Signal Quality

Agents will vary in the quality of their retrospective signals. Some will be precise and actionable ("spec section 3.2 contradicts section 4.1 on error codes"); others will be vague ("things were confusing"). This is expected.

The synthesis step (§6) handles quality variance by clustering signals — a vague signal that clusters with three precise ones still contributes to the theme's weight. Isolated vague signals are deprioritised naturally.

Over time, the skill instructions can be refined based on what makes signals most useful for synthesis. This is itself a retrospective feedback loop on the retrospective system.

---

## 6. Synthesis

### 6.1 Purpose

Synthesis is the retrospective meeting. It takes the accumulated raw signals and produces a structured report: themes ranked by frequency and severity, with actionable improvement candidates.

### 6.2 When It Happens

Synthesis is triggered explicitly — not on a timer, not automatically. The natural trigger points are:

- **Plan completion**: when a Plan transitions to `done`, the orchestrator or human runs synthesis on all signals accumulated during that Plan's lifetime
- **Feature completion**: for longer features, a feature-level synthesis can identify issues specific to that feature's workflow
- **On demand**: a human asks "what's the retrospective looking like?" at any point

### 6.3 The `retro` Tool

In the 2.0 tool surface, synthesis is a tool in a feature group (likely Planning, since retrospectives feed into planning for the next cycle):

```
retro(
  scope: "P1-basic-ui",         # Plan ID, Feature ID, or "project"
  since: "2025-03-01T00:00:00Z" # optional: only signals after this date
)
```

The tool:

1. Queries knowledge entries with `tags: ["retrospective"]`, filtered by scope and date
2. Groups signals by category
3. Within each category, clusters signals by textual similarity (reusing the knowledge system's existing Jaccard similarity, threshold ~0.5)
4. Ranks clusters by `count × severity_weight` where severity weights are: minor=1, moderate=3, significant=5
5. For each cluster, extracts the most common suggestion (if any)
6. Returns a structured response

The response structure:

```
{
  "scope": "P1-basic-ui",
  "signal_count": 23,
  "period": { "from": "2025-03-01", "to": "2025-03-27" },
  "themes": [
    {
      "rank": 1,
      "category": "spec-ambiguity",
      "title": "Error handling underspecified",       // generated summary
      "signal_count": 7,
      "severity_score": 17,                           // sum of weights
      "signals": ["KE-...", "KE-...", ...],           // entry IDs
      "top_suggestion": "Add error format and retry policy sections to spec template",
      "representative_observation": "Spec says 'handle errors appropriately' without defining..."
    },
    ...
  ],
  "worked_well": [
    {
      "title": "Vertical slice decomposition",
      "signal_count": 4,
      "representative_observation": "Each task was independently testable and deployable"
    }
  ]
}
```

### 6.4 The Retrospective Report

The `retro` tool returns structured data. Optionally, a `retro_report` tool (or mode) generates a markdown document suitable for committing to the repository:

```
retro_report(
  scope: "P1-basic-ui",
  since: "2025-03-01T00:00:00Z",
  output_path: "work/reports/retro-p1-basic-ui.md"
)
```

The generated document is registered as a document record with type `report` and can be reviewed, discussed, and approved like any other document. This makes the retrospective output part of the project's permanent record.

### 6.5 Signal Lifecycle After Synthesis

After synthesis, the signals that contributed to the report are not immediately retired. They remain in the knowledge store with their normal TTL (30 days for tier 3). This allows:

- Multiple synthesis runs over the same period (e.g., a feature-level retro followed by a plan-level retro)
- Re-running synthesis after implementing improvements to see if the signal count changes

Signals expire naturally through TTL. No manual cleanup is needed.

---

## 7. Closing the Loop: Experiments

### 7.1 The Problem with Improvements

Identifying friction is only half the retrospective cycle. The other half is: did the change we made actually help? Without measurement, retrospectives generate an ever-growing list of changes with no way to know which ones mattered.

### 7.2 Workflow Experiments

When a retrospective identifies an improvement candidate and the team decides to try it, the change is recorded as a decision entity with specific tags:

```
record_decision(
  slug: "retro-add-error-format-to-spec-template",
  summary: "Add error format and retry policy sections to the specification template, based on recurring spec-ambiguity signals",
  rationale: "7 signals across P1-basic-ui flagged underspecified error handling. Agents had to guess or ask for clarification, slowing implementation.",
  tags: ["workflow-experiment", "retrospective"]
)
```

The `workflow-experiment` tag marks this as an active experiment. The decision entity provides the rationale, the provenance (which retrospective themes drove it), and a stable ID that subsequent signals can reference.

### 7.3 Measuring Impact

In subsequent work cycles, retrospective signals can optionally reference a decision:

```
{
  category: "worked-well",
  observation: "The spec template now has an error format section and I didn't have to guess — used it directly",
  severity: "minor",
  related_decision: "DEC-042"
}
```

Or, conversely:

```
{
  category: "spec-ambiguity",
  observation: "The new error format section in the spec template doesn't cover async error callbacks",
  severity: "moderate",
  related_decision: "DEC-042"
}
```

The `related_decision` field is optional and recorded in the signal's content (as `Related: DEC-042`). During synthesis, the tool cross-references signals that mention decision IDs and produces an experiment effectiveness section:

```
"experiments": [
  {
    "decision_id": "DEC-042",
    "title": "Add error format to spec template",
    "positive_signals": 3,
    "negative_signals": 1,
    "net_assessment": "improvement — original theme (7 signals) reduced to 1 residual signal",
    "recommendation": "keep"
  }
]
```

This closes the loop. The retrospective → improvement → measurement cycle becomes visible and auditable.

### 7.4 Experiment Outcomes

Based on synthesis results, experiments have three possible outcomes:

| Outcome | Meaning | Action |
|---|---|---|
| **Keep** | Positive signals outweigh negative; original theme reduced | Decision status stays `accepted` |
| **Revise** | Mixed results; the change helped but has its own problems | Record a new decision superseding the original; try a revision |
| **Revert** | Negative signals dominate or original theme unchanged | Decision transitions to `rejected`; revert the change |

Humans make the keep/revise/revert decision. The synthesis provides the data. This is consistent with Kanbanzai's ownership model: humans own decisions; agents own execution and measurement.

---

## 8. Skill Instructions

The retrospective system is only useful if agents actually produce signals. The primary mechanism for this is skill content — specifically, additions to the `kanbanzai-agents` skill that instruct agents on when and how to contribute retrospective observations.

The skill addition:

```
## Retrospective Observations

When completing a task, reflect briefly on the process — not just the output.
If you noticed any of the following, include a retrospective signal:

- The spec was ambiguous or contradictory on a point that mattered
- Information you needed wasn't in the context packet
- A tool was missing, awkward, or returned unhelpful output
- The task was too large, too small, or had undeclared dependencies
- A workflow step felt unnecessary or was confusing
- Something worked particularly well and should be preserved

Not every task will have observations. That's fine — don't force it.
When you do have something to note, be specific: name the section, the tool,
or the step that caused friction.
```

This is guidance, not enforcement. Agents that produce low-quality signals will have those signals naturally deprioritised during synthesis (they won't cluster with precise signals and will expire via TTL).

---

## 9. Integration with 2.0 Tool Surface

### 9.1 Where It Fits

In the 2.0 tool surface design, the retrospective system touches three areas:

**`finish` (core tool)**: Gains the optional `retrospective` parameter for signal collection at task completion. This aligns with P2 (batch by default — signals are an array) and P4 (invisible infrastructure — signals are stored as knowledge entries behind the scenes).

**`retro` (planning feature group)**: A new tool for synthesis. This lives in the Planning group because retrospectives feed directly into planning the next cycle. It is not a core tool — most work sessions don't need it.

**`knowledge` (knowledge feature group)**: Existing. No changes needed. Signals contributed through `knowledge_contribute` already work with the tagging convention.

### 9.2 Tool Count Impact

One new tool (`retro`) in the Planning feature group, one new optional parameter on `finish`. Consistent with the 2.0 goal of minimal tool surface.

### 9.3 Context Budget

Retrospective signals are tier-3 knowledge entries. They are not surfaced in context packets by default — an agent implementing a task does not need to see process observations from other tasks. They are only queried during synthesis.

The exception: `worked-well` signals tagged with a specific feature or category could, in a future version, be surfaced to agents working on related features. This is deferred.

---

## 10. Implementation Phases

### Phase 1: Signal Collection (Minimal)

- Add `retrospective` parameter to `complete_task` / `finish`
- Store signals as tagged knowledge entries per §4.4
- Update `kanbanzai-agents` skill with retrospective guidance (§8)
- No new tools; synthesis is done manually in conversation

This phase is sufficient to start collecting data. Manual synthesis ("list all knowledge entries tagged retrospective and let's discuss them") validates the signal quality before investing in automated synthesis.

### Phase 2: Synthesis Tool

- Implement `retro` tool with clustering and ranking (§6.3)
- Implement `retro_report` for markdown output (§6.4)
- Register retrospective reports as document records

### Phase 3: Experiment Tracking

- Add `related_decision` support to signal structure
- Add experiment effectiveness section to synthesis output
- Update skill instructions to reference active experiments

Phase 1 can ship with 2.0. Phases 2 and 3 can follow in point releases, informed by the data collected in Phase 1.

---

## 11. What This Does Not Cover

- **Automated workflow changes.** The system identifies improvements and measures them. It does not automatically modify workflows, tools, or templates. Humans decide what to change.
- **Real-time alerting.** A significant signal does not trigger an immediate notification or block work. Signals are collected passively and reviewed during synthesis.
- **Cross-project retrospectives.** Signals are scoped to a single project (repository). Cross-project pattern detection is a future concern.
- **Human retrospective facilitation.** The system does not schedule meetings, send reminders, or guide a human retrospective conversation. It provides data for whatever retrospective format the team prefers.
- **Modification to existing entity lifecycles.** No new statuses, no new transitions, no new entity types. The system works entirely within the existing knowledge entry infrastructure plus one new tool parameter and one new tool.

---

## 12. Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Agents produce low-quality, generic signals | Medium | Low — bad signals don't cluster and expire via TTL | Refine skill instructions iteratively based on actual signal quality |
| Signal volume overwhelms synthesis | Low | Medium — synthesis becomes noisy | Category filtering, date-range scoping, severity threshold in `retro` tool |
| Agents game the system by always producing signals | Low | Low — volume without substance is easily detected in synthesis | No incentive structure rewards signal production; skill instructions say "don't force it" |
| `worked-well` signals are underproduced relative to complaints | High | Medium — distorted view of what to preserve | Explicit mention in skill instructions; monitor ratio during Phase 1 |
| Experiment tracking adds bureaucratic overhead | Medium | Medium — teams stop recording experiments | Phase 3 is optional; the core value is in signal collection and synthesis |

---

## 13. Success Criteria

The retrospective system is working if:

1. **Agents produce signals voluntarily** — at least 20% of completed tasks include one or more retrospective observations after skill instructions are deployed
2. **Signals cluster into recognisable themes** — synthesis produces 3–7 distinct themes per plan, not 50 unclustered observations
3. **Themes lead to changes** — at least one workflow experiment is recorded per completed plan
4. **Changes are measurable** — the signal count for the addressed theme decreases in the next cycle
5. **The system uses itself** — retrospective signals about the retrospective system itself are captured and acted on

Criterion 5 is the most important. If the retrospective system can improve itself, the design is sound.

---

## 14. Resolved Questions

1. **Clustering approach.** The choice between Jaccard similarity and LLM-based semantic clustering is an implementation decision, not a design decision. §6.3 describes the synthesis operation; the clustering algorithm is an internal detail. Start with the simplest approach that works (which may be an agent reading the list and clustering it in conversation) and automate if scale demands it.

2. **Scope granularity.** Signals are `project`-scoped knowledge entries. They do not carry Plan or Feature IDs as additional tags. Keep it simple for the first iteration. Date-range filtering in the `retro` tool is sufficient for scoping. If this proves inadequate, tags can be added in a future version without breaking existing signals.

3. **Experiment tracking nudge.** During Phase 3, context assembly will include active `workflow-experiment` decisions in the context packet — a brief note listing each experiment and its decision ID, so that agents can reference the decision in their retrospective signals. This closes the measurement loop without requiring agents to independently discover active experiments. The mechanism is described in §7.3. The implementation is deferred to Phase 3; by then, Phase 1 and 2 data will confirm whether the nudge is needed or whether theme-matching across cycles is sufficient on its own.

---

## 15. Open Questions

1. **Should there be a `retro` skill?** The retrospective guidance is currently proposed as an addition to `kanbanzai-agents`. A dedicated `kanbanzai-retrospective` skill would make it independently activatable but adds to the skill count. The right answer probably depends on how much the guidance grows.

2. **Integration with the viewer.** Retrospective reports are documents and will appear in the viewer. Should the viewer have a dedicated retrospective view that shows signal trends over time? This is a viewer concern, not a Kanbanzai concern, but the data model should support it.