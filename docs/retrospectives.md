# Retrospectives

Kanbanzai's retrospective system turns process observations into actionable patterns. You **record** signals during task completion, **synthesise** them into ranked themes, and **generate reports** that highlight what caused friction and what worked well.

> **Prerequisites.** This document assumes familiarity with the [User Guide](user-guide.md) — particularly task completion and the knowledge system. Signals are stored as knowledge entries, so that foundation helps.

---

## What signals are

A retrospective **signal** is a short, structured observation about the development process. It answers the question: *what happened during this task that the team should know about?*

Each signal records:

- A **category** — what kind of observation it is
- An **observation** — a one- or two-sentence description of what happened
- A **severity** — how much friction it caused (or how notable the positive outcome was)
- An optional **suggestion** — what could be done differently
- An optional **related decision** — a Decision entity ID linking the signal to an active workflow experiment

Signals are not bug reports or feature requests. They capture *process-level* observations — things about how the work happened, not what the work produced. A signal might note that a specification was ambiguous, that a tool behaved unexpectedly, or that parallel worktrees eliminated merge conflicts.

The system stores each signal as a tagged knowledge entry, so signals participate in the same lifecycle as other knowledge — they have confidence scores, use counts, and can be promoted, compacted, or retired. The `retrospective` tag distinguishes them from regular knowledge entries.

---

## Signal categories

Eight categories classify the type of observation. Each category has a distinct purpose:

| Category | What it captures | Example |
|----------|-----------------|---------|
| **workflow-friction** | Steps that slowed work down or caused unnecessary effort | "Had to manually transition three tasks because auto-promotion didn't fire" |
| **tool-gap** | Missing tool capability that would have helped | "No way to query which tasks are blocked by a specific dependency" |
| **tool-friction** | Existing tool that behaved unexpectedly or was hard to use | "edit_file writes to the main repo root, not the active worktree" |
| **spec-ambiguity** | Specification that was unclear, incomplete, or contradictory | "Error format undefined — had to guess the JSON structure" |
| **context-gap** | Missing context that forced guesswork or extra research | "No knowledge entry for the canonical YAML field ordering convention" |
| **decomposition-issue** | Task breakdown that was too coarse, too fine, or had dependency problems | "Tasks 4 and 6 had a circular dependency requiring manual intervention" |
| **design-gap** | Design decision that was missing, outdated, or caused problems | "No guidance on how merge gates interact with draft PRs" |
| **worked-well** | Something that went notably well and should be repeated | "Sub-agent parallelism with disjoint file scopes produced zero merge conflicts" |

Seven categories capture friction; one — **worked-well** — captures positive outcomes. The synthesis engine treats **worked-well** signals separately, grouping them into a dedicated "What Worked Well" section in reports.

---

## How to record signals

Record signals by including them in the `finish` call when completing a task. The `retrospective` parameter accepts an array of signal objects.

Each signal object has three required fields and two optional fields:

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| **category** | yes | string | One of the eight categories listed above |
| **observation** | yes | string | One- or two-sentence description of what happened |
| **severity** | yes | string | `minor`, `moderate`, or `significant` |
| **suggestion** | no | string | What could be done differently |
| **related_decision** | no | string | Decision entity ID (e.g. `DEC-042`) linking to a workflow experiment |

### Severity levels

Choose the severity that matches the impact of the observation:

- **minor** — a small annoyance or a nice-to-have improvement; did not meaningfully slow the work
- **moderate** — caused noticeable friction or delay; worth addressing if a pattern emerges
- **significant** — caused substantial rework, wasted effort, or blocked progress; should be addressed soon

The synthesis engine uses severity to rank themes. Each level carries a numeric weight — significant (5), moderate (3), minor (1) — and the engine multiplies signal count by the highest severity weight in a cluster to produce a **severity score**. Higher scores surface first in reports.

### Example

A `finish` call with two retrospective signals:

```json
{
  "task_id": "TASK-01KN5AK342WDJ",
  "summary": "Implemented merge gate validation",
  "retrospective": [
    {
      "category": "spec-ambiguity",
      "observation": "The spec did not define the error format for failed merge gates.",
      "severity": "moderate",
      "suggestion": "Add error format examples to the spec template."
    },
    {
      "category": "worked-well",
      "observation": "Parallel worktrees for gate implementation and gate testing eliminated merge conflicts.",
      "severity": "minor"
    }
  ]
}
```

### Validation and error handling

The system validates each signal independently. An invalid signal — missing a required field or using an unrecognised category or severity — is rejected with an error message, but **does not block task completion**. Valid signals in the same call are still accepted and stored.

This non-blocking design means you never have to choose between completing a task and recording observations. If a signal is malformed, the response includes the rejection reason so you can correct and resubmit it.

### How signals are stored

Each accepted signal becomes a knowledge entry with:

- **Topic:** `retro-{task_id}` for the first signal from a task, `retro-{task_id}-2`, `retro-{task_id}-3`, etc. for subsequent signals
- **Tags:** `["retrospective", "{category}"]`
- **Content:** `[{severity}] {category}: {observation}` — with `Suggestion: {suggestion}` and `Related: {decision_id}` appended when present
- **Scope:** `project` (hardcoded for all retrospective signals)
- **Learned from:** the task ID

### Nudges

The system encourages signal recording through two nudges. At most one nudge fires per completion — the feature-level nudge takes priority:

1. **Feature completion nudge** — when the last task in a feature completes and no task in that feature recorded any retrospective signals, the response includes a reminder to consider adding signals.
2. **Task completion nudge** — when a task completes with a summary but no knowledge entries and no retrospective signals (and the feature-level nudge did not fire), the response suggests including them.

Nudges are informational — they do not block completion or require action. Batch-mode `finish` calls suppress nudges entirely.

---

## How synthesis works

The `retro` tool's **synthesise** action reads accumulated signals, clusters them by category and textual similarity, and returns a ranked synthesis.

Call `retro` with no arguments (or `action: "synthesise"`) to synthesise all project signals. The tool also accepts the US spelling `"synthesize"`.

```json
{"tool": "retro", "arguments": {}}
```

### Parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| **action** | no | `synthesise` | `synthesise` or `report` |
| **scope** | no | `project` | Plan ID, Feature ID, or `project` |
| **since** | no | — | ISO 8601 timestamp; include only signals created after this time |
| **until** | no | — | ISO 8601 timestamp; include only signals created before this time |
| **min_severity** | no | `minor` | Minimum severity to include: `minor`, `moderate`, or `significant` |

### Scoping

The **scope** parameter controls which signals the engine considers:

- **project** — all retrospective signals in the knowledge base (no task-level filtering)
- **Plan ID** (e.g. `P22-documentation-for-public-release`) — signals from tasks belonging to features in that plan
- **Feature ID** (e.g. `FEAT-01KP8T4HXPEAY`) — signals from tasks belonging to that feature

Scoping uses the `learned_from` field on each knowledge entry to trace signals back to their originating task, then matches tasks to features and features to plans.

### Clustering

The engine groups signals in two stages:

1. **Category grouping** — signals are first separated by category.
2. **Jaccard similarity clustering** — within each category, signals are clustered using greedy Jaccard similarity with a threshold of 0.5. Each signal either joins the first existing cluster whose centroid word set is sufficiently similar, or starts a new singleton cluster.

This means two signals about "spec ambiguity in error formats" will likely cluster together, while a signal about "spec ambiguity in lifecycle states" will form its own cluster.

### Ranking

Each cluster becomes a **theme**. Themes are ranked by **severity score** — the product of the cluster's signal count and the highest severity weight in the cluster. When two themes have the same severity score, the one with more signals ranks higher.

### Response structure

The synthesise action returns a JSON object. The top-level fields are:

| Field | Description |
|-------|-------------|
| **scope** | The scope that was applied |
| **signal_count** | Total number of signals that matched the filters |
| **period** | `{from, to}` — the time range of the matching signals |
| **themes** | Array of ranked theme objects (friction categories only) |
| **worked_well** | Array of positive-outcome clusters (from **worked-well** signals) |
| **experiments** | Array of experiment tracking entries (when signals reference Decision entities tagged `workflow-experiment`) |

Each **theme** in the `themes` array contains:

| Field | Description |
|-------|-------------|
| **rank** | Position in the ranked list (1 = most important) |
| **category** | The signal category |
| **title** | Short descriptive title derived from the representative observation |
| **signal_count** | Number of signals in this cluster |
| **severity_score** | signal_count × highest severity weight |
| **signals** | Array of knowledge entry IDs |
| **representative_observation** | The observation from the highest-severity signal in the cluster |
| **top_suggestion** | First non-empty suggestion in the cluster (if any) |

Each entry in the `worked_well` array contains:

| Field | Description |
|-------|-------------|
| **title** | Short descriptive title |
| **signal_count** | Number of signals in this cluster |
| **representative_observation** | The observation from the highest-severity signal |

Each entry in the `experiments` array contains:

| Field | Description |
|-------|-------------|
| **decision_id** | The Decision entity ID |
| **title** | The decision's summary |
| **positive_signals** | Count of **worked-well** signals referencing this decision |
| **negative_signals** | Count of friction signals referencing this decision |
| **net_assessment** | Summary string (e.g. "3 positive, 1 negative") |
| **recommendation** | `keep` (positive > negative), `revert` (zero positive, at least one negative), or `revise` (everything else — including cases where negatives outnumber positives) |

The experiment tracking feature connects retrospective signals to workflow experiments. When you try a new process and record signals with `related_decision` pointing to the experiment's Decision entity, the synthesis engine automatically tracks whether the experiment is producing more positive or negative outcomes.

---

## How to generate reports

The `retro` tool's **report** action runs the same synthesis, then generates a Markdown document and registers it as a document record.

```json
{
  "tool": "retro",
  "arguments": {
    "action": "report",
    "scope": "project",
    "output_path": "work/retro/sprint-12.md",
    "title": "Sprint 12 Retrospective"
  }
}
```

### Additional parameters

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| **output_path** | yes | — | Repository-relative path for the generated Markdown file |
| **title** | no | `Retrospective: {scope} {date}` | Title for the document and the heading |

The report action:

1. Runs synthesis with the same filtering parameters (scope, since, until, min_severity)
2. Renders a Markdown document with a metadata table, ranked themes, experiment results, and a "What Worked Well" section
3. Writes the file to **output_path**, creating parent directories if needed
4. Registers the file as a document record (type: `report`, status: `draft`)
5. Returns the synthesis result plus a `report` object with the file path and document record ID

### Report structure

The generated Markdown document contains:

- A **title heading** and metadata table (scope, total signals, period)
- A **Themes** section with ranked subsections — each showing the category, signal count, severity score, representative observation, and top suggestion
- An **Experiments** section (when applicable) showing each workflow experiment's signal tally and recommendation
- A **What Worked Well** section listing positive-outcome clusters

---

## When to run a retrospective

There is no enforced cadence — run retrospectives when they will be most useful:

- **After completing a feature** — scope to the feature ID to see what friction and successes that feature produced. This is the most common cadence.
- **After completing a plan** — scope to the plan ID for a broader view across all features. Good for identifying systemic patterns.
- **Periodically on long-running projects** — use time-range filtering (`since`, `until`) to review a specific period regardless of entity boundaries.
- **Before planning new work** — synthesise recent signals to inform process improvements in the next plan's design.

The `min_severity` filter helps focus attention. Set it to `moderate` or `significant` to skip minor observations and concentrate on the patterns that caused real friction.

> **Tip.** Always call `retro(action: "synthesise")` before writing any retrospective or review document. The synthesis surfaces signals from across the project that may not be in your immediate context. Do not write retrospective documents from memory alone.

---

## What to read next

- **Return to the basics** — the [User Guide](user-guide.md) covers task completion, knowledge management, and the broader workflow.
- **Understand orchestration** — the [Orchestration and Knowledge](orchestration-and-knowledge.md) document explains how signals fit into the knowledge system lifecycle.
- **Look up tool parameters** — the [MCP Tool Reference](mcp-tool-reference.md) covers every parameter for `finish` and `retro`.