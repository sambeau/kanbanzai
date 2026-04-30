# Retrospective Report 3 — Agent Session Experience

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30T19:25:35Z          |
| Status | Draft                         |
| Author | Claude (DeepSeek V4 Pro)      |

## Research Question

What went well and what didn't go well during this session of using the Kanbanzai
MCP server and workflow tooling to inspect project status and prepare to write
specifications for B36? This report captures first-person agent experience for
iterative system improvement.

## Scope and Methodology

**In scope:** This session (approximately 15 tool calls) covering project
orientation, status inspection, document audit, design document discovery, and
spec-writing prerequisites.

**Out of scope:** Long-running development, task decomposition, review workflows,
sub-agent dispatch. This was a navigation and orientation session.

**Methodology:** Direct experience report. Not synthesised from retro knowledge
or other agents' signals — purely what this agent observed and felt.

## Findings

### Finding 1: Orientation flow is smooth and discoverable

The `status()` → `next()` → claim pattern provides a clear on-ramp. Calling
`status()` with no arguments immediately surfaces the project overview, attention
items, and active batches. The orientation message pointing to the getting-started
skill is helpful as a fallback, even when the agent already knows the pattern.

The `kanbanzai-getting-started` skill checklist (git status → store check →
corpus audit → read AGENTS.md → check work queue → claim task) establishes a
clear session-start ritual. Following it caught orphaned `.kbz/` state files
that needed committing.

Confidence: high (direct observation).

### Finding 2: Stage binding prerequisite enforcement is clear but creates a sequencing puzzle

When asked to write specs for B36, the `specifying` stage binding clearly stated
the prerequisite: an approved design document. Following the chain led to
discovering the design at `work/_project/design-kbz-cli-and-binary-rename.md`.
The design was thorough and directly mapped to all 4 B36 features — but it was
in `draft` status.

This created a natural stopping point: the agent could not proceed to spec
writing without either (a) getting human design approval or (b) overriding the
gate. The system correctly enforced this, and the agent was able to present the
user with clear options (approve design → write specs vs. review first).

What worked: the prerequisite was unambiguous and the design document was
findable via `grep` (since it wasn't linked from the B36 features directly).

What was less smooth: the design document lives under `work/_project/` rather
than in a `work/B36-kbz-cli-and-status/` directory, so the connection between
the batch and its design required a grep search rather than a direct lookup.
A `doc_intel(action: "find", entity_id: "B36-kbz-cli-and-status")` query would
have been more natural, but only works if the design document is classified
with that entity reference.

Confidence: high (direct observation).

### Finding 3: Document audit reveals corpus hygiene gaps

The session-start `doc(action: "audit")` revealed 23 unregistered documents on
disk. The `doc(action: "import")` resolved these cleanly, but many files
couldn't be auto-typed because they fell outside configured document root
patterns (e.g., `work/plan/P10-*.md`, `work/reviews/review-P38-*.md`,
`docs/orchestration-and-knowledge.md`). These files required a `default_type`
to be specified, which the agent didn't provide in the first import attempt.

The audit is valuable but the "no document type available" skips create a
lingering gap — the files are still unregistered after import, and the agent
must notice this and take additional action (or the human must). A follow-up
prompt like "23 files imported, 16 skipped — re-run with default_type to
register remaining files" would help.

Confidence: medium (single session, may vary by project maturity).

### Finding 4: Bug status tracking has a stale-attention problem

When the agent inspected `BUG-01KQB-54280HDP` and `BUG-01KQF-4CEB8Z90`, both
showed `status: "closed"` in the entity detail but the status dashboard still
reported them as attention items with "prioritise resolution" messages. The
agent had to do a second lookup to confirm closure. The attention recalculation
appears to lag behind entity state changes.

Confidence: high (observed with both bugs, reproducible).

### Finding 5: Project overview is dense but scannable

The `status()` project overview returns 38 batches with task counts. For a
project this mature, the list is long but scannable — the `done` vs `active`
distinction is immediately visible. The attention items at the bottom surface
what needs action without requiring the agent to scan all 38 entries.

What helped: the batch list is sorted with active/proposed items mixed in
among the done items, but the status column makes them visually distinct.
It would be slightly easier if active/proposed items were grouped at the top
rather than interleaved chronologically.

Confidence: medium (preference, not a defect).

## Recommendations

- **Recommendation:** Add a `doc_intel` find-by-entity shortcut or have batch
  status dashboards include a `design` document link so agents can go directly
  from "I need to write specs for B36" to "here's the design" without grep.
- **Confidence:** medium
- **Based on:** Finding 2

- **Recommendation:** Improve the doc import UX to surface a count of
  skipped files and suggest `default_type` as a remedy.
- **Confidence:** medium
- **Based on:** Finding 3

- **Recommendation:** Ensure attention items are recalculated when bug
  status changes (or add a TTL so stale warnings expire).
- **Confidence:** high
- **Based on:** Finding 4

## Limitations

- This is a single-session report covering orientation and navigation only.
  No implementation, review, or decomposition workflows were exercised.
- The project is highly mature (35+ completed batches) — the experience of
  navigating a younger or messier project may differ.
- The agent had prior familiarity with Kanbanzai conventions; a first-time
  agent might experience more friction with the orientation checklist.
