# Retrospective: P11 — Fresh Install Experience

- Scope: P11-fresh-install-experience
- Date: 2026-03-29T12:36:39Z
- Author: Claude Sonnet 4.6 (post-implementation review session)

---

## Overview

P11 delivered four improvements to the out-of-box kanbanzai experience: MCP server
connection, embedded review skills, default context roles, and a standard document layout.
All 13 tasks were marked done and passed code review. During subsequent real-world testing,
six bugs were discovered and fixed. This retrospective captures what worked, what caused
friction, and what should change in future plans.

---

## What Worked Well

**The entity and task structure gave instant orientation.**
The task summaries acted as a compressed spec. Reviewing P11 required no archaeology —
`entity list` on the features immediately showed what had been intended and what was
claimed done. This is the system working as designed, and it made the review genuinely
faster than reading commit history would have been.

**Commit discipline made bugs easy to locate.**
Each commit was coherent and well-scoped. When real-world testing revealed bugs, it was
straightforward to identify exactly where each piece had been introduced. The commit
messages were specific enough to be useful as a change log.

**The design document was the right anchor.**
Having `work/design/fresh-install-experience.md` as a single authoritative source meant
systematic spec-vs-implementation checking was possible. The fact that the document
itself had errors is a separate problem (see below) — the principle of having one place
to check against is sound.

**Fixes were small and obvious.**
Once each bug was found, the fix was always local and clear. The codebase structure made
it easy to identify the right place to change. No fix required significant restructuring.

**The review process caught real gaps.**
Three genuine issues were found during the structured review before any code ran: the
AGENTS.md scope guard, the project timeline, and the stale help text. These would have
caused ongoing confusion and health check warnings. The review process justified itself
even before real-world testing began.

---

## Friction Points

### 1. The spec described the happy path only

Every bug found during real-world use fell into the same category: an unspecified edge
case. The "new vs existing project" distinction was defined as "has commits or not" —
which was the wrong signal. The spec never explained what the distinction was *for*,
so a plausible-but-wrong implementation was impossible to detect in review.

The most common real-world scenario — a repo with one README commit being set up with
kanbanzai for the first time — was not covered by any acceptance criterion. It was
treated as an "existing project" and therefore got none of the first-time init behaviour
(no `work/` directories, no `.zed/settings.json`).

**Bugs caught by this gap:** missing `work/` directories on first-time init with commits;
`.zed/settings.json` not written when project had existing commits.

### 2. Install-experience features resist spec-based review

A code review against a spec can verify that the implementation matches what was written.
It cannot verify that what was written is complete. For features whose primary artefact
is something a human runs — an install flow, a CLI interaction, an editor integration —
the only reliable validation is to run the thing in the scenarios that actually occur.

The six bugs found during real use were collectively invisible to unit tests, code review,
and spec-checking because the scenarios they covered were not in the spec. A five-minute
smoke test (run `kanbanzai init` in three different starting states) would have caught
all of them before delivery.

### 3. The design document contained wrong third-party format information

The Zed `context_servers` entry format was documented incorrectly in the design document:
the implementation used a nested `{"path": ..., "args": [...]}` structure that Zed
silently discards. The correct format is a flat string for `command` with `args` as a
sibling key.

The design document was written before the Zed integration was tested. The format was
plausible-looking and internally consistent — it just didn't match Zed's actual schema.
This error propagated faithfully from design doc → implementation → tests → user-facing
docs, so every layer passed review.

**Root cause:** third-party integration formats should be verified against the real tool
before they enter the design document, not after. Five minutes in Zed would have caught
this before a line of code was written.

### 4. The interactive prompt UX was never designed

The prompt `Document root path (e.g. work/docs):` was inherited from an earlier phase
and never revisited when the default layout expanded to eight directories. The hint
`(e.g. work/docs)` looks like a suggested path, so users typed `work` (one directory)
expecting to get the full eight-directory layout. Empty input errored rather than
defaulting.

This was an interaction design gap that no amount of code review would have surfaced. The
UX had to be experienced to be understood as broken.

### 5. `_managed` in `.zed/settings.json` was caught last

The `_managed` block was written to `.zed/settings.json` despite Zed validating that
file against its own JSON schema. This produced a visible linter error in the editor —
not a silent failure, but a clear signal that something was wrong. It was caught during
real use rather than review.

The distinction between files we own (`.mcp.json`, where `_managed` is appropriate) and
files owned by a third-party tool (`.zed/settings.json`, where schema validation applies)
should have been explicit in the design. It was not.

---

## Suggestions

**1. Require edge-case ACs for install-experience features.**
Any feature whose deliverable is a command a human runs should have acceptance criteria
covering the common failure modes: what happens when the project already has commits,
what happens when the user presses Enter at a prompt, what happens when a target file
already exists in a conflicting state. These are not edge cases — they are the scenarios
most users encounter first.

**2. Add a smoke-test gate for init features.**
A short shell script that exercises three or four realistic starting states (empty repo,
repo with one commit, repo already initialised, repo with existing editor config) would
catch integration gaps that unit tests miss. It does not need to be automated in CI — even
running it manually before marking a feature done would have prevented all six bugs here.

**3. Verify third-party formats before writing the design document.**
For any feature that involves writing config for a third-party tool, the format must be
verified against the real tool before it enters the spec. A wrong format in the design
doc propagates to implementation, tests, and user-facing docs simultaneously — every
layer passes review because every layer is faithfully wrong.

**4. Name concepts precisely in the spec.**
The codebase now uses `kbzExisted` as the correct signal for "has kanbanzai been set up
here before". The design document used "new project" which carries the wrong connotation
(implying commit history matters). Precise naming in the spec would have made the wrong
implementation harder to write.

**5. Distinguish owned files from third-party-schema files in design.**
Files the kanbanzai binary writes but does not own (files validated by another tool's
schema) need to be called out explicitly in design. The rule — no `_managed` block in
`.zed/settings.json` — was not obvious from the design, which described all written files
uniformly.

---

## Summary

| Category | Count |
|---|---|
| Bugs found in review (pre-run) | 3 |
| Bugs found in real-world testing (post-review) | 6 |
| Root cause: spec gap (unspecified scenario) | 4 |
| Root cause: wrong third-party format in design doc | 1 |
| Root cause: UX never designed | 1 |

The workflow infrastructure — entities, tasks, design documents, commit discipline,
review procedure — functioned well. The gaps were at the specification level: incomplete
acceptance criteria, unverified third-party formats, and missing interaction design for
the CLI prompt. These are addressable with process changes that do not require changes
to the workflow system itself.