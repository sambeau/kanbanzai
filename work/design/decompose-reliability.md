# Design: `decompose propose` Reliability Fixes

- Status: proposal
- Date: 2026-03-28
- Author: orchestrator
- Retro signals: KE-01KMT4T26W1VZ (stale-index fallback), KE-01KMT4T59DB2N (misleading fallback output)

---

## 1. Problem

`decompose propose` failed silently during P7 setup. Three specs were written
and registered in the same session, then `decompose propose` was called
immediately. The document intelligence index had not yet processed the new
files, so the tool found the document records but read empty parsed content.

Rather than stopping, the tool fell back to generating one task per markdown
section header: "Implement 1. Purpose", "Implement 2. Goals", "Implement 3.
Scope", and so on — structurally plausible output that was entirely wrong as
a task breakdown.

The root cause has two parts:

**Gap 1 — No precondition check on spec approval status.**
`decompose propose` does not verify that the spec document is in `approved`
status before attempting to parse it. A draft spec may be incomplete,
unreviewed, or — as in this case — not yet indexed.

**Gap 2 — Fallback produces misleading output.**
When the index returns no parsed content (empty or absent), the tool generates
a section-header-derived task list rather than stopping. The only signal is a
single top-level warning: "No acceptance criteria found in spec". That warning
is easy to miss because the surrounding output — task slugs, summaries,
dependency hints, estimates — looks structurally correct. A reader must know
to look for the warning and understand that it indicates an infrastructure
problem, not a content problem.

Together: the agent receives a complete-looking proposal, evaluates it,
discovers it is wrong, discards it, and creates tasks manually — wasting a
full round trip and leaving no trace of the failure for future improvement.

---

## 2. Proposed Fixes

### Fix 1 — Spec approval gate

Before parsing, `decompose propose` checks that the resolved spec document
record is in `approved` status. If not:

```
error: spec "FEAT-.../specification-..." is in "draft" status
       Approve the spec before decomposing.
```

No proposal is generated. The agent is told exactly what to fix.

This gate is meaningful independently of the index staleness problem: a draft
spec may still be changing, so decomposing it prematurely produces tasks that
will need revising anyway.

### Fix 2 — Hard stop when index returns no AC content

After locating the spec, `decompose propose` checks whether the document
intelligence index has parsed content for the file (non-empty fragment list).
If the index has no content for a registered, approved spec:

```
error: spec content not yet indexed
       The spec file exists and is approved, but the document intelligence
       index has not processed it yet. Run index_repository, then retry.
```

No proposal is generated. The section-header fallback path is removed.

This is a stronger guarantee than Fix 1 alone: even if the spec is approved,
a freshly approved file may not yet be in the index.

### Fix 3 — AGENTS.md precondition rule (stopgap)

Add a rule to `AGENTS.md` under the decomposition stage gate:

> **Before calling `decompose propose`:**
> 1. Confirm the spec document record is in `approved` status.
> 2. If the spec was registered in the current session, call
>    `index_repository` first to ensure the document intelligence index
>    has processed it.

This is a documentation-level safeguard that works immediately, before code
changes land, and remains useful even after Fixes 1 and 2 are in place.

---

## 3. What is not changing

- The section-header fallback is being **removed**, not improved. There is no
  valid use case for generating tasks from headings — if the spec is approved
  and indexed, parse it properly; if either precondition fails, stop and say
  so. A partial proposal is worse than no proposal.
- Fix 3 (disk read fallback when index is empty) from the original analysis is
  **deferred**. Fixes 1 and 2 together fully prevent silent failure. A disk
  fallback adds implementation complexity and blurs the boundary between the
  document intelligence layer and raw file I/O. It can be revisited if there
  is a demonstrated need.
- No changes to the spec document format, AC naming conventions, or document
  registration workflow.

---

## 4. Scope

| Component | Change |
|-----------|--------|
| `internal/mcp/` decompose handler | Add spec approval status check (Fix 1) |
| `internal/mcp/` decompose handler | Add index content check; remove section-header fallback (Fix 2) |
| `AGENTS.md` | Add decompose precondition rule (Fix 3) |
| Tests | Unit tests for both error paths |

---

## 5. Open Questions

None. Approach agreed in design review.