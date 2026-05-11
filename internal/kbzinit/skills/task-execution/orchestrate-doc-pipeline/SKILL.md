---
# kanbanzai-managed: true
# kanbanzai-version: dev
name: orchestrate-doc-pipeline
description:
  expert: "Sequential pipeline coordination dispatching five editorial
    stages with changelog collation, advisory human checkpoints,
    re-entry handling, and completion reporting for documentation
    refinement"
  natural: "Run a document through the editorial pipeline — coordinate
    the five stages, track changes, offer checkpoints, and produce
    a summary"
triggers:
  - run the documentation pipeline
  - publish a document through the editorial pipeline
  - editorial pipeline
  - refine documentation
  - run document through all editorial stages
  - coordinate document editing
roles: [doc-pipeline-orchestrator]
stage: documenting
constraint_level: medium
---

## Vocabulary

**Pipeline structure:**
- **pipeline stage** — one of five sequential editorial passes: Write, Edit, Check, Style, Copyedit; each operates at a smaller scale than the previous
- **stage boundary** — what each stage owns and must not touch; the most important property of the pipeline
- **large-to-small principle** — structural decisions before content verification, content before prose cleanup, prose before sentence polish; the ordering rationale for the pipeline
- **sequential dispatch** — each stage runs after the previous completes; the document and previous changelog are passed forward together
- **idempotency** — every stage is safe to run twice; a second pass should produce minimal or no changes

**Change tracking:**
- **changelog** — per-stage output summarising what was changed and why; enables delta review rather than full re-read
- **stage flag** — an issue found by one stage that belongs to a different stage's scope; recorded in the changelog and surfaced in the completion summary
- **completion summary** — all five changelogs collated into a single report showing the document's journey through the pipeline

**Human interaction:**
- **advisory checkpoint** — a recommended human review point that does not block the pipeline; offered after Edit (structural decisions are expensive to undo) and after Copyedit (final read-through)
- **re-entry** — sending a document back to an earlier stage when a later stage finds a problem outside its scope; should be rare

**System integration:**
- **doc registration** — recording the document with `doc(action: register)` at pipeline start; refreshing with `doc(action: refresh)` at pipeline end
- **feedback signal** — a recurring finding pattern from downstream stages that should feed back into upstream SKILL anti-patterns via retrospective synthesis

## Anti-Patterns

### Stage Boundary Violation
- **Detect:** A stage produces changes outside its scope — the fact-checker rewrote sentences, the copy editor restructured sections, the style editor changed factual content
- **BECAUSE:** Boundary discipline is the primary quality property of the pipeline; when stages drift into each other's territory, work is duplicated, undone, or conflicted — and the later stage's changes haven't been reviewed by the stage that owns that scope
- **Resolve:** Review each stage's output against its SKILL's boundaries before passing to the next stage; if out-of-scope changes are detected, discard them and re-run with explicit boundary reminders

### Missing Changelog
- **Detect:** A stage produces revised text with no summary of what was changed and why
- **BECAUSE:** Without a changelog the human reviewer must re-read the entire document to understand what happened at each stage; changelogs are what make the pipeline reviewable and what enable the coordinator to detect boundary violations
- **Resolve:** Require every stage to output a changelog alongside the revised document; reject stage output that lacks one and re-run the stage

### Unnecessary Re-entry
- **Detect:** A document is sent back to an earlier stage for a minor issue that could be noted and continued
- **BECAUSE:** Re-entry restarts the pipeline from an earlier point, re-running stages that already completed successfully; the cost is proportional to the number of stages re-run and the wasted previous work
- **Resolve:** Reserve re-entry for severe issues only: structural problems that make later stages pointless (misplaced sections), hallucinated content that invalidates surrounding text, or a document so problematic it needs rewriting. Flag minor issues in the completion summary for human review

### Checkpoint Avoidance
- **Detect:** The coordinator skips the advisory checkpoint after Edit or after Copyedit without offering the human a chance to review
- **BECAUSE:** Advisory checkpoints exist because structural decisions (post-Edit) are expensive to undo and the final product (post-Copyedit) deserves a human eye; skipping them saves minutes but risks publishing a document the human would have caught problems in
- **Resolve:** Always offer the checkpoint. If the human declines or doesn't respond within a reasonable time, continue — the checkpoint is advisory, not blocking

### Assessment-Only Dispatch
- **Detect:** Stages produce reports describing what *should* change, but nobody edits the file — the document arrives at the end of the pipeline unchanged
- **BECAUSE:** The pipeline exists to improve a document, not to describe how it could be improved. A report without applied changes is a review, not an editorial stage. The two patterns are: Edit and Check produce reports → the orchestrator applies changes to the file before dispatching the next stage; Style and Copyedit edit the file directly → the orchestrator reviews the changelog for boundary violations. If neither the sub-agent nor the orchestrator edits the file, the stage did nothing.
- **Resolve:** After receiving each stage's output, confirm the document file has been modified (or that the orchestrator has applied the stage's findings). If the document is unchanged and the stage reported non-zero findings, stop — something went wrong. Re-dispatch the stage with explicit instructions to edit the file, or apply the findings yourself before continuing.

### Feedback Loop Neglect
- **Detect:** Recurring downstream findings (same classification appearing across multiple documents) are never synthesised or proposed as upstream SKILL improvements
- **BECAUSE:** Without feedback, the Write stage makes the same mistakes repeatedly and the Check/Style stages catch them every time — a pipeline that never improves its upstream stages is doing unnecessary work on every document
- **Resolve:** After every 10 documents (or quarterly), synthesise recurring findings using the retro tool; propose anti-pattern additions to the Write SKILL for human review

## Checklist

```
Copy this checklist and track your progress:

Before starting:
- [ ] The document to be processed is identified and accessible
- [ ] The document is registered with `doc(action: register)` or already exists in the doc system
- [ ] The document's purpose, type, and audience are understood (needed for the Write stage; may already exist)

Per stage (who edits the file?):
- Write, Style, Copyedit → the sub-agent edits the file directly
- Edit, Check → the sub-agent produces a report; the orchestrator applies changes to the file
- [ ] The stage receives the document and the previous stage's changelog (if any)
- [ ] The document file has been modified (by the sub-agent or the orchestrator) — if the stage reported findings but the file is unchanged, something went wrong
- [ ] The stage's output includes a changelog of what was changed and why
- [ ] The changelog is reviewed for boundary violations before passing to the next stage
- [ ] Any stage flags (issues outside scope) are recorded

Checkpoints:
- [ ] Advisory checkpoint offered after Edit (structural review)
- [ ] Advisory checkpoint offered after Copyedit (final review)

Completion:
- [ ] All five changelogs are collated into a completion summary
- [ ] Stage flags are surfaced in the summary
- [ ] The document record is refreshed with `doc(action: refresh)`
- [ ] Any feedback signals (recurring patterns) are noted for future retrospective
```

## Procedure

### Step 1: Register the Document

If the document is not already registered, call `doc(action: register)` with the document path, type, and title. This creates the tracking record. IF the document already exists in the doc system, confirm its record is current and note its type and purpose.

### Step 2: Dispatch Write Stage

Pass the document purpose, type, audience, and source material to the `write-docs` skill via `handoff`. If the document already exists (human-written or previously drafted), pass the existing document as the input — the Write stage will use it as the base rather than drafting from scratch. The Write stage **edits the file directly** — it produces a revised draft (or a first draft) and verification notes. Confirm the output includes a changelog or initial notes describing what was produced.

### Step 3: Dispatch Edit Stage

Pass the draft document and the Write stage's output to the `edit-docs` skill. Receive: structural edit report with findings classified as structural-blocking or structural-suggestion. The Edit stage produces a **report**, not a revised file. Review the changelog for boundary violations — the editor should not have rewritten sentences or fixed facts. IF structural-blocking findings exist, **the orchestrator applies them to the file** before continuing. This is the orchestrator's responsibility — do not pass an unmodified document to the Check stage when there are blocking findings.

### Step 4: Advisory Checkpoint — Post-Edit

Offer the human a chance to review the structural edit. Present the heading skeleton assessment, any structural-blocking findings, and the Edit changelog. IF the human provides feedback → incorporate it and note the feedback in the checkpoint record. IF the human declines or does not respond → continue. Record the checkpoint outcome (reviewed / skipped / feedback incorporated).

### Step 5: Dispatch Check Stage

Pass the structurally-edited document to the `check-docs` skill. Receive: QA report with classified findings (hallucination, unverified, stale, vague, inflated, promotional). The Check stage produces a **report**, not a revised file. **The orchestrator applies factual corrections** (hallucinations, stale references) to the file. Flag substance issues (vague, inflated) for the Style stage — do not correct those here. Review the changelog for boundary violations — the checker should not have restructured sections or rewritten prose.

### Step 6: Re-entry Check

Evaluate the Check stage's flags. IF the Check stage flagged severe structural problems (e.g. an entire section based on a hallucinated feature, or sections that need to be removed entirely) → send the document back to the Edit stage (Step 3) and re-run from there. IF issues are minor (a few vague claims, a stale version number) → note them and continue. Re-entry should be rare — reserve it for issues that make later stages pointless.

### Step 7: Dispatch Style Stage

Pass the fact-checked document and any substance flags from the Check stage to the `style-docs` skill. The Style stage **edits the file directly** — it applies all vocabulary and pattern changes, then produces a changelog of what it changed. Review the changelog for boundary violations — the style editor should not have restructured sections or changed factual content. If the file is unchanged but the changelog lists findings, the stage failed to apply its changes — re-dispatch with explicit editing instructions.

### Step 8: Dispatch Copyedit Stage

Pass the style-edited document to the `copyedit-docs` skill. The Copyedit stage **edits the file directly** — it applies all sentence-level changes, then produces a changelog. Review the changelog for boundary violations — the copy editor should not have restructured sections, changed content, or hunted for AI artifacts (that was the Style stage's job). If the file is unchanged but the changelog lists findings, the stage failed to apply its changes — re-dispatch with explicit editing instructions.

### Step 9: Advisory Checkpoint — Post-Copyedit

Offer the human a final review of the finished document. Present the completion summary with all five changelogs collated, plus any unresolved flags. IF the human provides feedback → apply it and note the feedback in the checkpoint record. IF the human declines or does not respond → continue. Record the checkpoint outcome.

### Step 10: Complete

1. Refresh the document record with `doc(action: refresh)`.
2. Output the completion summary (see Output Format below).
3. Note any recurring finding patterns as feedback signals for future retrospective synthesis. IF this is the 10th+ document through the pipeline since the last retro synthesis, consider calling the `retro` tool to surface patterns.

## Output Format

At pipeline completion, produce a summary:

```
## Pipeline Completion Summary

**Document:** {document path}
**Type:** {README | getting-started | manual | reference | design}
**Stages completed:** {5/5 or fewer if re-entry occurred}

### Stage Results

| Stage | Changes | Flags | Key action |
|-------|---------|-------|------------|
| Write | — | {count} | {one-line summary} |
| Edit | {count} | {count} | {one-line summary} |
| Check | {count} findings | {count} | {one-line summary} |
| Style | {count} | {count} | {one-line summary} |
| Copyedit | {count} | {count} | {one-line summary} |

### Human Checkpoint Results
- **Post-Edit:** {reviewed / skipped / feedback incorporated}
- **Post-Copyedit:** {reviewed / skipped / feedback incorporated}

### Re-entry Events
{none, or description of re-entry and reason}

### Unresolved Flags
{issues flagged by stages that were not resolved during the pipeline run}

### Feedback Signals
{recurring patterns to consider for upstream SKILL improvement, if any}
```

## Examples

### BAD: Pipeline run with no change tracking

```
Ran the document through all five stages. The document looks good now.
No issues found.
```

WHY BAD: No changelogs, no stage summaries, no evidence that any stage did meaningful work. The human cannot tell what changed at each stage or whether boundary violations occurred. This is a rubber stamp of the pipeline, not coordination.

### GOOD: Pipeline run with full tracking

```
## Pipeline Completion Summary

**Document:** refs/getting-started.md
**Type:** getting-started
**Stages completed:** 5/5

### Stage Results

| Stage | Changes | Flags | Key action |
|-------|---------|-------|------------|
| Write | — | 2 unverified claims | Drafted 8-section guide following getting-started template |
| Edit | 3 structural | 0 | Moved Quick Start before Background; split §4 into two sections |
| Check | 5 findings | 1 for Style | Caught hallucinated --verbose flag; flagged 2 vague claims |
| Style | 12 words, 2 rewrites | 0 | Replaced 8 banned words; rewrote 2 fingerprint-cluster passages |
| Copyedit | 7 sentences | 0 | Fixed 4 passive→active; split 2 long sentences; standardised contractions |

### Human Checkpoint Results
- **Post-Edit:** Reviewed. Approved section reorder.
- **Post-Copyedit:** Skipped (human did not respond within timeout).

### Re-entry Events
None.

### Unresolved Flags
- Check flagged "performance improvement of up to 10x" as unverified — no benchmark source found. Left for author to confirm or remove.

### Feedback Signals
- Third document in a row where Style caught "leverage" and "utilize" — consider adding to Write SKILL anti-patterns.
```

WHY GOOD: Every stage's contribution is visible. The human can see what changed and why at each stage. Boundary violations are implicitly absent (each stage's changes match its scope). Checkpoints were offered. The completion summary collates all five changelogs into a reviewable record. A feedback signal was identified for upstream improvement.

## Evaluation Criteria

1. (weight: 0.25) Does every stage produce both a revised document and a changelog?
2. (weight: 0.20) Are advisory checkpoints offered after Edit and after Copyedit?
3. (weight: 0.15) Are stage boundary violations detected and addressed?
4. (weight: 0.15) Is the completion summary comprehensive — all stages, flags, checkpoints, re-entry events?
5. (weight: 0.15) Are re-entry decisions appropriate — reserved for severe issues, not minor findings?
6. (weight: 0.10) Are feedback signals identified for recurring downstream patterns?

## Questions This Skill Answers

- How do I run a document through the full editorial pipeline?
- What happens at each stage of the pipeline?
- When should I send a document back to an earlier stage?
- How do I review what the pipeline changed?
- When does the human get to review?
- How do recurring problems feed back into the writing stage?
- What does the completion summary look like?