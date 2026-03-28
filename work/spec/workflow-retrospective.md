# Workflow Retrospective Specification

| Document | Workflow Retrospective Specification                      |
|----------|-----------------------------------------------------------|
| Status   | Draft                                                     |
| Plan     | P5-workflow-retrospective                                 |
| Created  | 2026-03-27T21:05:56Z                                      |
| Related  | `work/design/workflow-retrospective.md`                   |
|          | `work/spec/kanbanzai-2.0-specification.md` §12, §6.4      |
|          | `work/design/kanbanzai-2.0-design-vision.md` §4, §5.2     |
|          | `work/design/machine-context-design.md`                   |

---

## 1. Purpose

This specification defines the requirements for the Workflow Retrospective system
(P5). The system adds a structured feedback loop to the development cycle: agents
capture process observations at task completion, those observations are synthesised
into themes, themes lead to workflow experiments, and experiments are measured in
subsequent cycles.

P5 builds on the existing Kanbanzai infrastructure — the knowledge store, the
`finish` tool, and the planning feature group — without introducing new entity
types, new storage paths, or new lifecycle state machines.

This specification is organised into three phases that can be shipped independently:

- **Phase 1** — Signal collection via an optional `retrospective` parameter on `finish`
- **Phase 2** — Synthesis via a new `retro` tool in the planning feature group
- **Phase 3** — Experiment tracking: measuring whether workflow changes had the intended effect

---

## 2. Goals

1. **Capture process friction at the source.** At the moment an agent completes a task, it has the freshest possible observations about how the work went. The retrospective parameter on `finish` captures these observations before context is discarded.

2. **Make observations aggregatable, not just readable.** Free-text `blockers_encountered` is a write-only field. Structured signals with fixed categories and severity levels are queryable, clusterable, and rankable.

3. **Synthesise into actionable themes.** Raw signals, grouped and ranked by frequency and severity, surface the friction that is worth changing. The synthesis step makes patterns visible that would be invisible signal by signal.

4. **Close the loop on experiments.** Workflow changes should be recorded, tracked, and measured. When the change made things better, that should be visible. When it didn't, the decision should be revisited.

5. **Ride existing infrastructure.** No new entity types, no new storage directories, no new lifecycle state machines. Retrospective signals are knowledge entries with a tag convention.

---

## 3. Scope

### 3.1 In scope

- **Phase 1:** Optional `retrospective` parameter on the `finish` tool and `complete_task`; signal storage as tagged knowledge entries; skill instruction additions for retrospective guidance.
- **Phase 2:** `retro` tool in the planning feature group; clustering, ranking, and structured response; report mode generating a registered markdown document.
- **Phase 3:** `related_decision` field in signal structure; experiment effectiveness section in `retro` synthesis output; context nudge for active `workflow-experiment` decisions.

### 3.2 Deferred beyond P5

- Dedicated retrospective viewer or dashboard UI.
- Cross-project retrospective aggregation.
- Automated threshold-based synthesis triggers (e.g., auto-synthesise when signal count exceeds N).
- `worked-well` signals surfaced proactively in context packets for related tasks.
- Dedicated `kanbanzai-retrospective` skill (separate from `kanbanzai-agents`).
- Experiment outcome tracking outside of retrospective synthesis (e.g., a standalone experiment dashboard).

### 3.3 Explicitly excluded

- New entity types (`RetroSignal` or equivalent). Signals are knowledge entries.
- New storage directories or YAML schemas for signal records.
- Automated workflow modification. The system identifies improvements and measures them; humans decide what to change.
- Real-time alerting on significant signals.
- Blocking work on unacknowledged signals.

---

## 4. Design Principles

The design principles for P5 are defined in `work/design/workflow-retrospective.md`
§2. This section summarises the constraints that are binding for this specification.

**§2.1 — Ride the existing infrastructure.** Retrospective signals are knowledge
entries with a tag convention. The knowledge system's deduplication, confidence
scoring, TTL, compaction, and query mechanisms are reused without modification.

**§2.2 — Capture at the natural moment.** Signals are captured at task completion,
the same moment as completion metadata and knowledge entries. No separate step, no
second tool call.

**§2.3 — Structured signals, not free text.** Categories, severity, and optional
suggestions make signals aggregatable. Free-text observations provide context; the
structured envelope provides the handle for pattern detection.

**§2.4 — Synthesis is explicit, not automatic.** The `retro` tool runs when someone
decides it's time to reflect. There is no background job, no timer, no automatic
threshold trigger.

**§2.5 — Try, measure, decide.** Improvements are experiments. They are recorded as
decisions, tracked in subsequent cycles, and evaluated against signal data. Humans
make the keep/revise/revert decision; the synthesis provides the evidence.

---

## 5. Retrospective Signals

### 5.1 Structure

A retrospective signal has the following fields:

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `category` | Yes | string (enum) | Category of process observation (see §5.2) |
| `observation` | Yes | string | One or two sentences describing what happened |
| `severity` | Yes | string (enum) | How much friction this caused: `minor`, `moderate`, or `significant` |
| `suggestion` | No | string | What could be done differently |
| `related_decision` | No | string | Decision ID (e.g., `DEC-042`) of an active workflow experiment this signal relates to. Phase 3 only. |

The `source` field (`agent` or `human`) is set automatically based on the identity
recorded in the contributing session; it is not an agent-supplied field.

### 5.2 Categories

Categories are fixed strings. Implementations MUST reject signals with an unrecognised
category (see §10.1.6). The set of valid categories is:

| Category | Meaning |
|----------|---------|
| `workflow-friction` | A workflow step was unnecessarily difficult, slow, or confusing |
| `tool-gap` | A tool was missing, or an existing tool didn't support the needed operation |
| `tool-friction` | A tool exists but was awkward, required too many parameters, or returned unhelpful output |
| `spec-ambiguity` | The specification was unclear, incomplete, or contradictory on a point that mattered during implementation |
| `context-gap` | Information needed during the task was not in the context packet |
| `decomposition-issue` | Tasks were too large, too small, poorly ordered, or had undeclared dependencies |
| `design-gap` | The design document did not address something that turned out to matter during implementation |
| `worked-well` | Something about the process worked well and should be preserved |

New categories may be added in future revisions. The `retro` synthesis tool (§7)
treats unrecognised categories as valid but unclustered, so older clients synthesising
newer signals degrade gracefully rather than failing.

### 5.3 Severity

| Value | Weight | Meaning |
|-------|--------|---------|
| `minor` | 1 | Caused slight friction; work proceeded without significant disruption |
| `moderate` | 3 | Required non-trivial workaround or extra investigation |
| `significant` | 5 | Materially slowed or endangered the work |

Severity weights are used in the synthesis ranking formula (§7.2).

### 5.4 Storage convention

Each signal is stored as a knowledge entry with the following properties:

| Property | Value |
|----------|-------|
| Tier | 3 |
| Scope | `project` |
| Tags | `["retrospective", "<category>"]` |
| Topic | `retro-{task-id}` for the first signal from a task; `retro-{task-id}-{N}` for subsequent signals from the same task (N=2, 3, …) |
| Content | Formatted string per §5.5 |
| `learned_from` | Automatically set to the completing task ID |

For signals contributed outside a task context (§6.3), the topic is
`retro-{slug}` where `slug` is a caller-supplied identifier such as a date or
session descriptor.

### 5.5 Content format

Signal content is a formatted string rather than embedded structured data.
This preserves compatibility with the knowledge system's Jaccard-similarity
deduplication and compaction, which operate on content strings.

Format when `suggestion` is absent:

```
[{severity}] {category}: {observation}
```

Format when `suggestion` is present:

```
[{severity}] {category}: {observation} Suggestion: {suggestion}
```

Format when `related_decision` is present (Phase 3), appended after any suggestion:

```
[{severity}] {category}: {observation} Suggestion: {suggestion} Related: {related_decision}
```

Example:

```
[moderate] spec-ambiguity: Spec says "handle errors appropriately" without defining error format or retry policy. Suggestion: Add error format and retry policy sections to the spec template. Related: DEC-042
```

---

## 6. Phase 1: Signal Collection

### 6.1 `retrospective` parameter on `finish`

The `finish` tool gains an optional `retrospective` parameter: an array of signal
objects matching the structure in §5.1.

Extended input parameters for `finish` (single mode):

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `retrospective` | object[] | no | Array of retrospective signal objects (see §5.1) |

Extended input for batch mode: each entry in the `tasks` array may also include a
`retrospective` key with the same semantics.

Behaviour:

1. If `retrospective` is absent or empty, no signals are stored; the tool behaves identically to the pre-P5 implementation.
2. Each signal is validated before the task is completed (see §10.1.3–§10.1.6). Per-signal validation failures are rejected individually and reported in the response without blocking completion of the task or other signals.
3. Accepted signals are stored as knowledge entries per §5.4 after the task status is successfully transitioned. If the task transition itself fails, no signals are stored.
4. Accepted signal entry IDs are reported in the `side_effects` array of the `finish` response using a new side-effect type `retrospective_signal_contributed`.

### 6.2 Extended `finish` response

A new side-effect type is added to the existing side-effect vocabulary
(`work/spec/kanbanzai-2.0-specification.md` §8.3):

| Type | Trigger | Description |
|------|---------|-------------|
| `retrospective_signal_contributed` | `finish` with accepted retrospective signal | A retrospective signal was stored as a knowledge entry |

The `retrospective` section in the `finish` response reports signal outcomes alongside
the existing `knowledge` section:

```yaml
retrospective:
  accepted:
    - entry_id: "KE-01JZ..."
      topic: "retro-TASK-01JX..."
      category: "spec-ambiguity"
  rejected:
    - category: "unknown-category"
      observation: "..."
      reason: "Unknown category: 'unclear-thing'. Valid categories: workflow-friction, tool-gap, ..."
  total_attempted: 2
  total_accepted: 1
```

### 6.3 Standalone contribution

Retrospective observations that arise outside a task context — during planning
sessions, design reviews, or general usage — can be contributed directly using the
`knowledge` tool (or `knowledge_contribute` in the 1.0 surface) following the
conventions in §5.4. This requires no new tool and works today.

The system makes no distinction between signals contributed via `finish` and those
contributed directly. The `retro` synthesis tool (§7) queries by tag and treats all
`retrospective`-tagged entries equally.

### 6.4 Skill instructions

The `kanbanzai-agents` skill file is updated to include retrospective guidance. The
addition instructs agents on:

- When to include retrospective signals (spec ambiguity, context gaps, tool friction, decomposition problems, things that worked well).
- What makes a signal useful (specific: name the section, tool, or step; not just "things were confusing").
- That signals are optional: agents are not required to produce them, and should not force observations when they have nothing meaningful to note.

The guidance uses `finish`'s `retrospective` parameter as the primary collection
mechanism and also explains the standalone `knowledge_contribute` path for
observations that arise outside task completion.

---

## 7. Phase 2: Synthesis

### 7.1 `retro` tool

The `retro` tool is added to the **planning** feature group (`work/spec/kanbanzai-2.0-specification.md` §6.4). It reads accumulated retrospective signals, clusters them into themes, and returns a ranked synthesis.

Input parameters:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `action` | string | no | `"synthesise"` (default) or `"report"` (see §7.4) |
| `scope` | string | no | Plan ID, Feature ID, or `"project"` (default: `"project"`) |
| `since` | string | no | ISO 8601 timestamp; only include signals created after this time |
| `until` | string | no | ISO 8601 timestamp; only include signals created before this time |
| `min_severity` | string | no | Minimum severity to include: `"minor"` (default), `"moderate"`, or `"significant"` |

The `scope` parameter filters by `learned_from` task ancestry: signals contributed
at `finish` have `learned_from` set to the completing task ID, which belongs to a
feature, which belongs to a plan. Scope filtering uses this ancestry. Signals
contributed standalone (§6.3) are always included when `scope` is `"project"`.

### 7.2 Clustering and ranking

The synthesis algorithm:

1. Query all knowledge entries with `tags: ["retrospective"]` matching scope and date filters.
2. Group signals by `category`.
3. Within each category, cluster signals by textual similarity using Jaccard similarity on content tokens, with a threshold of approximately 0.5. Signals below the threshold form singleton clusters.
4. Rank clusters by `cluster_score = signal_count × max_severity_weight_in_cluster`. If two clusters have the same score, the one with the higher `signal_count` ranks higher.
5. Separate `worked-well` signals from negative categories; report them in a distinct section.
6. If Phase 3 is active: cross-reference signals whose content contains `Related: {decision_id}` with known `workflow-experiment` decisions and produce an `experiments` section (§8.3).

The specific clustering algorithm is an implementation detail. Implementations may
substitute a different similarity measure (e.g., trigram overlap, LLM-assisted
clustering) provided the observable outputs — grouping, ranking, representative
observation — remain consistent with this definition.

### 7.3 Response structure

```yaml
scope: "project"
signal_count: 23
period:
  from: "2026-03-01T00:00:00Z"
  to: "2026-03-27T21:05:56Z"
themes:
  - rank: 1
    category: "spec-ambiguity"
    title: "Error handling underspecified across multiple features"
    signal_count: 7
    severity_score: 17
    signals:
      - "KE-01JA..."
      - "KE-01JB..."
    top_suggestion: "Add error format and retry policy sections to the spec template"
    representative_observation: "Spec says 'handle errors appropriately' without defining error format or retry policy"
  - rank: 2
    category: "context-gap"
    title: "Error format convention not surfaced in context"
    signal_count: 3
    severity_score: 9
    signals:
      - "KE-01JC..."
    top_suggestion: "Tag error-format knowledge entries with 'always-include' for backend tasks"
    representative_observation: "The error format KE existed but was not in the context packet; found it manually via knowledge_list"
worked_well:
  - title: "Vertical slice decomposition"
    signal_count: 4
    representative_observation: "Each task was independently testable and deployable; no integration surprises"
```

The `title` of each theme is a short, generated summary of the cluster. Implementations
may generate this from the representative observation, from the cluster centroid, or
via any other means; the title is informational and not schema-validated.

### 7.4 Report mode

When called with `action: "report"`, the `retro` tool additionally generates a
markdown document and registers it as a document record:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `output_path` | string | yes (report mode) | Repository-relative path for the generated report file |
| `title` | string | no | Title for the document record (defaults to `"Retrospective: {scope} {date}"`) |

Behaviour:

1. Run synthesis as in §7.2.
2. Render the synthesis as a structured markdown document (themes, worked-well, and if Phase 3 is active, experiments).
3. Write the document to `output_path`.
4. Register the document as a document record with `type: "report"` using `doc_record_submit`.
5. Return the synthesis response (§7.3) extended with:

```yaml
report:
  path: "work/reports/retro-p1-basic-ui-2026-03-27.md"
  document_id: "PROJECT/report-retro-p1-basic-ui-2026-03-27"
```

The generated document is in draft status after creation. It can be reviewed,
discussed, and approved like any other document record. Retrospective reports become
part of the project's permanent record.

### 7.5 Feature group placement

The `retro` tool is a member of the **planning** feature group. The updated group
membership:

| Group | Tools |
|-------|-------|
| **planning** | `decompose`, `estimate`, `conflict`, `retro` |

The planning group is enabled/disabled as a unit via `mcp.groups` in
`.kbz/config.yaml`. Projects that have not enabled the planning group do not see the
`retro` tool.

---

## 8. Phase 3: Experiment Tracking

### 8.1 Workflow experiment decisions

When a retrospective theme leads to a workflow change, the change is recorded as a
decision entity tagged `workflow-experiment`. The tag marks the decision as an active
experiment that should be tracked in subsequent synthesis runs.

Example:

```yaml
slug: retro-add-error-format-to-spec-template
summary: Add error format and retry policy sections to the specification template
rationale: >
  7 signals across P1-basic-ui flagged underspecified error handling. Agents had
  to guess or ask for clarification, slowing implementation.
tags:
  - workflow-experiment
  - retrospective
```

No new decision lifecycle states are introduced. Experiments in `accepted` status
are active. An experiment that is judged ineffective has its decision status updated
to `rejected` (or superseded by a revised decision).

### 8.2 `related_decision` in signals

The signal structure (§5.1) gains a `related_decision` optional field in Phase 3.
When present, its value is appended to the content string per §5.5 as `Related: {decision_id}`.

The `related_decision` field is accepted in `finish`'s `retrospective` parameter
from Phase 3 onward. Earlier versions that do not recognise this field must ignore
it gracefully (the content format ensures the related decision is still stored even
if the field is stripped during validation on a Phase 1–2 system).

### 8.3 Experiment effectiveness section in synthesis

When `retro` synthesises signals and `workflow-experiment` decisions exist in the
project, the response includes an `experiments` section:

```yaml
experiments:
  - decision_id: "DEC-042"
    title: "Add error format to spec template"
    positive_signals: 3
    negative_signals: 1
    net_assessment: "improvement — original theme reduced from 7 signals to 1 residual"
    recommendation: "keep"
  - decision_id: "DEC-043"
    title: "Require context profile for all new features"
    positive_signals: 0
    negative_signals: 2
    net_assessment: "no improvement — original theme unchanged"
    recommendation: "revise"
```

Cross-referencing: a signal is attributed to a decision if its content contains
`Related: {decision_id}`. Signals are classified as `positive` if their category is
`worked-well`, and `negative` otherwise.

Recommendations follow the three outcomes defined in
`work/design/workflow-retrospective.md` §7.4: `keep`, `revise`, or `revert`. The
recommendation is heuristic; humans make the actual keep/revise/revert decision. The
synthesis provides the data.

### 8.4 Context nudge for active experiments

During Phase 3, context assembly (inside `next` and `handoff`) includes a brief
section listing active `workflow-experiment` decisions when the project has any. This
section lists each experiment's decision ID and summary, allowing agents to reference
the relevant decision ID when they encounter friction or success that relates to it.

The nudge is a short, appended section in the assembled context packet. It is not a
knowledge entry and does not count against the knowledge tier-3 budget. It is
assembled from decision entities tagged `workflow-experiment` with status `accepted`.

---

## 9. What This Does Not Cover

- **Automated workflow changes.** The system identifies and measures improvements. Humans decide what to change.
- **Real-time alerting.** Significant signals do not trigger notifications or block work. They are collected passively and reviewed during synthesis.
- **Cross-project retrospectives.** Signals are scoped to a single project instance.
- **Human retrospective facilitation.** The system provides data; it does not schedule retrospective meetings or guide conversation.
- **Modification to existing entity lifecycles.** No new statuses, no new transitions. The system works within the existing knowledge entry infrastructure plus one new tool parameter and one new tool.
- **Signal enforcement.** Agents are not required to produce retrospective signals. The system never blocks work on missing or insufficient signals.

---

## 10. Acceptance Criteria

Criteria are numbered `P5-1.x` (Phase 1), `P5-2.x` (Phase 2), and `P5-3.x`
(Phase 3). Phase 1 must be complete before Phase 2 begins. Phase 2 must be complete
before Phase 3 begins.

### 10.1 Phase 1: Signal Collection

**`finish` parameter:**

- [ ] **P5-1.1** `finish(task_id, summary)` without a `retrospective` parameter behaves identically to the pre-P5 implementation; no regression.
- [ ] **P5-1.2** `finish(task_id, summary, retrospective: [...])` with one or more valid signals completes the task and stores each accepted signal as a knowledge entry.
- [ ] **P5-1.3** Each stored signal is a tier-3, project-scoped knowledge entry with tags `["retrospective", "<category>"]`.
- [ ] **P5-1.4** Topic for the first signal from a task is `retro-{task-id}`. Topic for the second signal from the same task is `retro-{task-id}-2`, and so on.
- [ ] **P5-1.5** Content is formatted per §5.5: `[{severity}] {category}: {observation}` when no suggestion is given; `[{severity}] {category}: {observation} Suggestion: {suggestion}` when a suggestion is given.
- [ ] **P5-1.6** `learned_from` is set to the completing task's ID on each stored signal entry.
- [ ] **P5-1.7** Signals are only stored after the task status transition succeeds. If the task transition fails, no signals are stored.

**Validation:**

- [ ] **P5-1.8** A signal with an unrecognised `category` is rejected with an error message listing the valid categories. The rejection does not block task completion or other valid signals in the same array.
- [ ] **P5-1.9** A signal with an unrecognised `severity` (i.e., not `minor`, `moderate`, or `significant`) is rejected. The rejection does not block task completion or other valid signals.
- [ ] **P5-1.10** A signal missing the required `observation` field is rejected. The rejection does not block task completion or other valid signals.
- [ ] **P5-1.11** A signal missing the required `category` field is rejected. The rejection does not block task completion or other valid signals.
- [ ] **P5-1.12** A signal missing the required `severity` field is rejected. The rejection does not block task completion or other valid signals.
- [ ] **P5-1.13** `suggestion` is optional; a signal without it is accepted and stored with no `Suggestion:` clause in the content string.

**Response:**

- [ ] **P5-1.14** The `finish` response includes a `retrospective` section with `accepted` (entry IDs and topics), `rejected` (with reasons), `total_attempted`, and `total_accepted`.
- [ ] **P5-1.15** Each accepted signal produces a `retrospective_signal_contributed` entry in the `side_effects` array of the `finish` response.

**Batch:**

- [ ] **P5-1.16** Batch `finish` (via `tasks` array) supports a `retrospective` key per task entry, with the same semantics as single mode.
- [ ] **P5-1.17** Signal validation failures in one task entry of a batch do not affect signal processing in other task entries.

**Skill instructions:**

- [ ] **P5-1.18** The `kanbanzai-agents` skill file includes retrospective guidance covering: when to include signals, what makes a signal specific and useful, the `finish` `retrospective` parameter as primary mechanism, standalone `knowledge_contribute` as the secondary path, and the instruction not to force observations when there is nothing meaningful to note.

### 10.2 Phase 2: Synthesis

**`retro` tool registration:**

- [ ] **P5-2.1** The `retro` tool is registered as a member of the `planning` feature group.
- [ ] **P5-2.2** `retro` is accessible when the `planning` group is enabled in `.kbz/config.yaml`, and absent when the group is not enabled.

**`retro` synthesis (default action):**

- [ ] **P5-2.3** `retro()` with no parameters queries all `retrospective`-tagged knowledge entries in the project and returns a synthesis response per §7.3.
- [ ] **P5-2.4** `retro(since: "...")` filters signals to those created after the given timestamp.
- [ ] **P5-2.5** `retro(until: "...")` filters signals to those created before the given timestamp.
- [ ] **P5-2.6** `retro(min_severity: "moderate")` excludes `minor` signals from the synthesis.
- [ ] **P5-2.7** `retro(scope: "P1-basic-ui")` filters signals by ancestry: only signals whose `learned_from` task belongs to a feature in that plan are included, plus project-scoped standalone signals.
- [ ] **P5-2.8** `worked-well` signals are reported in a separate `worked_well` section and are not ranked alongside negative-category themes.
- [ ] **P5-2.9** Themes are ranked in descending order by `signal_count × max_severity_weight`. When counts and scores are equal, the cluster with the higher raw `signal_count` ranks higher.
- [ ] **P5-2.10** The response includes `signal_count` (total signals considered), `period.from` (earliest signal timestamp), and `period.to` (latest signal timestamp or synthesis time if no signals exist).
- [ ] **P5-2.11** If no signals match the filter criteria, `retro` returns an empty `themes` array and a `signal_count` of 0 without error.
- [ ] **P5-2.12** Each theme includes: `rank`, `category`, `title`, `signal_count`, `severity_score`, `signals` (array of knowledge entry IDs), `representative_observation`, and optionally `top_suggestion` (present only if at least one signal in the cluster has a suggestion).

**Report mode:**

- [ ] **P5-2.13** `retro(action: "report", output_path: "...")` runs synthesis, generates a markdown document at the given path, and registers it as a document record with `type: "report"`.
- [ ] **P5-2.14** The registered document record is in `draft` status after creation.
- [ ] **P5-2.15** The `retro(action: "report")` response includes the `report.path` and `report.document_id` fields in addition to the full synthesis response.
- [ ] **P5-2.16** Calling `retro(action: "report")` twice with the same `output_path` either updates the existing document or returns an error; it does not create a duplicate document record.

### 10.3 Phase 3: Experiment Tracking

**Signal `related_decision` field:**

- [ ] **P5-3.1** `finish` with `retrospective: [{..., related_decision: "DEC-042"}]` stores the signal with `Related: DEC-042` appended to the content string per §5.5.
- [ ] **P5-3.2** `related_decision` is optional; signals without it are accepted and stored without a `Related:` clause.
- [ ] **P5-3.3** `related_decision` with an ID that does not correspond to a known decision entity in the project is accepted and stored (no lookup required at collection time).

**Experiment section in synthesis:**

- [ ] **P5-3.4** When at least one signal in the synthesis result contains `Related: {decision_id}`, the `retro` response includes an `experiments` section.
- [ ] **P5-3.5** Each entry in `experiments` includes: `decision_id`, `title` (decision summary), `positive_signals` (count of `worked-well` signals referencing this decision), `negative_signals` (count of non-`worked-well` signals referencing this decision), `net_assessment` (informational string), and `recommendation` (one of `keep`, `revise`, `revert`).
- [ ] **P5-3.6** Signals not referencing any decision ID are not attributed to any experiment entry.
- [ ] **P5-3.7** The `experiments` section is absent (not an empty array) when no signals reference any decision ID.

**Context nudge:**

- [ ] **P5-3.8** When the project has one or more decision entities tagged `workflow-experiment` with status `accepted`, context assembly (inside `next` and `handoff`) appends a section listing those experiments by decision ID and summary.
- [ ] **P5-3.9** When no `workflow-experiment` decisions are in `accepted` status, no experiment section is appended to assembled context (no change to context assembly output).
- [ ] **P5-3.10** The experiment nudge section does not count against the knowledge entry budget in context assembly; it is sourced from decision entities, not knowledge entries.