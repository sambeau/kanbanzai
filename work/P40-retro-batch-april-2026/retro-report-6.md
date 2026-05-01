# Retrospective: B38 Batch Review Session (April 2026)

| Field  | Value                                          |
|--------|------------------------------------------------|
| Date   | 2026-04-30                                     |
| Author | AI agent (reviewer-conformance, batch-review)  |
| Scope  | B38-plans-and-batches — batch-reviewing stage  |
| Task   | Running the batch-level conformance review     |

---

## Overview

This session ran the `batch-reviewing` stage for B38 (Plans and Batches): reading the
stage binding and role, gathering evidence across 10 features, verifying documents and
tasks, running health and retro, and producing a conformance report. The work was
evidence-gathering rather than implementation, so the friction profile is different from
an implementation session — it surfaces issues with querying, lookup, and ID resolution.

---

## What Went Well

### The stage-bindings → role → skill chain was clear and fast to orient

Reading `.kbz/stage-bindings.yaml` → `reviewer-conformance.yaml` → `review-plan/SKILL.md`
in sequence took three tool calls and gave a complete picture of what to do, what vocabulary
to use, and what the output format should be. The skill's checklist structure meant I never
had to guess what step came next. This scaffolding is genuinely useful.

### `health()` was the highest-signal single tool call in the session

One call returned: a merge conflict on B38-F1 (error level), confirmation that B38-F9's
branch was already merged to main (warning level), stale worktrees, TTL-expired knowledge
entries, and doc-currency gaps. This alone shaped the majority of the conformance findings.
It is a well-designed tool — dense, structured, and immediately actionable.

### The prior draft report pattern saved significant time

The skill says to re-read an existing draft before re-reviewing. There was already a draft
from an earlier pass in the same day. Reading it let me verify the 4 blocking CGs were still
present rather than reconstructing the full analysis from scratch. The pattern works.

### Parallel tool calls worked cleanly for evidence gathering

Steps 1–4 of the skill (feature census, document lookup, task verification, worktree status)
were naturally parallelisable. Firing 8 tool calls simultaneously and compositing the results
into a single picture was efficient. No ordering issues, no race conditions.

### `doc(action: "list", owner: "B38-plans-and-batches")` found everything

Once I used the correct owner parameter, the document list was complete and gave me the
approval status, path, and type for all B38-scoped documents in one call. The document
record system is clearly well-structured.

---

## What Did Not Go Well

### Short IDs don't resolve — silent empty results, not helpful errors

`entity(action: "list", parent: "B38")` returned `{"entities": [], "total": 0}` with no
error. The same for `status(id: "B38")`, which returned a parse error about unknown prefix
`B`. I had to call `entity(action: "list", type: "batch")` to discover that the correct
ID was `B38-plans-and-batches`. This ID format is not obvious from the display ID `B38`.

The fix is straightforward: either accept `B38` as a short-form alias, or return an error
like "no entity found with parent 'B38' — did you mean 'B38-plans-and-batches'?" Instead of
silent empty results that look like genuine "no features found."

### `doc(action: "get", path: "...")` fails when given a doc record ID

Entity records store a `spec` field with a value like
`FEAT-01KQ7JDT511BZ/specification-p37-f5-spec-work-tree-migration`. This looks like a path.
When I passed it to `doc(action: "get", path: "...")` it returned "no document found."
The parameter that works is `id`, not `path`.

The distinction between `id` (the record's logical key) and `path` (the filesystem path
of the document file) is not obvious. The `doc` tool description doesn't explain that the
value in an entity's `spec` field is the doc ID, not the file path. This required trial
and error to discover.

### `entity(action: "list", parent_feature: "FEAT-...")` returned all 614 tasks

I called this for all three reviewing features expecting to get the 4–5 tasks scoped to
each feature. All three calls returned `total: 614` — the entire project task list.
The `parent_feature` filter appears to be silently ignored or broken.

I had to fall back to grepping the `.kbz/state/tasks/` directory to identify tasks by
their slug naming convention. That worked, but it's a workaround for a filtering tool that
should just work. A broken filter that returns everything instead of nothing is particularly
confusing because results look plausible.

### `retro(scope: "B38-plans-and-batches")` returned 0 signals

Scoping retrospective synthesis to the batch ID returned nothing. Switching to
`scope: "project"` with a `since` date filter surfaced 1 relevant signal. It's unclear
whether the batch-scoped retro is working as intended or whether signals aren't being
tagged to batch scope. The skill says to call `retro` as part of the review procedure,
but if batch scope silently returns nothing it's easy to conclude there's no data when
there is.

### `status(id: "B38-plans-and-batches")` is not supported

The `status` tool returned a resolution error: it only recognises `P` prefix (for legacy
plan IDs). Batches use `B` prefix but aren't understood by `status`. For a project that
has formally migrated from plans to batches, this is a significant gap — the primary
dashboard tool doesn't support the current entity type.

---

## Observations on the MCP Tool UX Generally

**The tool count is high.** To run a batch review I used: `entity`, `doc`, `doc_intel`,
`worktree`, `health`, `retro`, `read_file`, `find_path`, `terminal`, `edit_file`, `now`.
That's 11 distinct tools for a read-heavy workflow. Most of the friction came not from
workflow complexity but from discovering which tool accepted which parameter format.

**The tools are internally consistent but not mutually consistent.** Each tool's schema
is internally logical. But `entity` uses `parent: "B38-plans-and-batches"` while `doc`
uses `owner: "B38-plans-and-batches"` — same value, different parameter names. The spec
field in an entity record looks like a path but is used as an ID. These cross-tool
inconsistencies accumulate into real friction during a session.

**Error messages are sparse when they matter most.** The two empty-result cases
(`entity list parent: "B38"` and `doc get path: "<doc-id>"`) both needed better errors.
An empty result is only safe when you're confident the filter is working. When the filter
might be broken or the ID format wrong, an empty result is indistinguishable from a
genuine "nothing there" answer.

---

## What to Improve

### 1. Short-form batch ID resolution

`B38` should resolve to `B38-plans-and-batches` across all tools — or return a typed error
suggesting the full ID. A "did you mean?" error is far better than a silent empty result.

### 2. Fix `entity list parent_feature` filter

The filter is silently broken or ignored. It should return only tasks for the specified
feature. This is the most functionally impactful bug found in this session — task-per-feature
queries are fundamental to review and orchestration workflows.

### 3. Clarify `doc` tool: `id` vs `path`

Either the `doc` tool description should explicitly say "when entity records reference a
spec or dev-plan, that value is the `id`, not the `path`" — or the tool should accept
doc IDs via the `path` parameter as well (by trying ID lookup when path lookup fails).

### 4. `status` tool support for batch IDs

`status(id: "B38-plans-and-batches")` should return a batch dashboard. Batches are the
primary execution unit in the current workflow. Not supporting them in the main dashboard
tool is a gap that will recur every time a reviewer or orchestrator tries to orient.

### 5. `retro` scope for batch IDs

`retro(scope: "B38-plans-and-batches")` should return signals associated with the batch.
If no signals are tagged to that scope, the response should say so explicitly (e.g., "0
signals found for scope 'B38-plans-and-batches' — try `scope: project` with a date range").

---

## Summary

The batch review workflow itself is sound. The stage-bindings → role → skill chain, the
`health()` tool, and the prior-draft pattern all worked well. The friction was concentrated
in four specific places: short-form ID resolution, the broken `parent_feature` task filter,
the `id` vs `path` ambiguity in the `doc` tool, and the absence of batch support in `status`.
None of these blocked delivery — I found workarounds in every case — but they each cost
multiple tool calls and required me to reason about implementation details rather than workflow
intent. Fixing these four issues would make the review workflow noticeably cleaner.
