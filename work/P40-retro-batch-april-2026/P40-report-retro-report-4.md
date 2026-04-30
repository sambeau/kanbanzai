# Personal Retrospective — April 2026 Batch

**Author:** Claude Sonnet (via sambeau)
**Date:** 2026-04-30
**Scope:** Bug fixes and performance investigation across BUG-01KQB-54280HDP,
BUG-01KQB-6TKSABJJ, and the performance follow-up design/spec/plan cycle.

## What went well

### Cache performance payoff is real
Wiring the SQLite cache into the CLI dropped `kanbanzai status` from 4.1s to
0.7s — a single-digit-line change with immediate user-visible impact. The P29
design was correct and the implementation was sound; the gap was purely a
wiring omission in the CLI dependency injection. Finding and fixing this felt
like the system working as intended: design → implement → discover gap → fix.

### The cache infrastructure itself is well-built
`Cache.EntityExists()`, `Cache.IsWarm()`, `Cache.ListByType()` — all existed
and were tested before I needed them. The write-through consistency model
(`cacheUpsertFromResult` on every mutation) meant I never had to think about
staleness windows. The fallback discipline (cache miss → filesystem) means
correctness is never gated on cache availability. This is good architecture.

### Document pipeline produced consistent output
Writing design → spec → dev-plan for the performance follow-up was
straightforward once I understood the validation script requirements. The
`validate-*-structure.sh` scripts caught missing sections immediately. The
doc tool's register/refresh workflow is smooth.

### Health check caught real drift
The `kanbanzai status` health check was the canary that surfaced the original
`entityDirectory("batch")` → `"batchs"` bug. Without it, the false positive
would have gone unnoticed indefinitely. The health infrastructure is
valuable.

## What didn't go well

### I jumped to code before analysis
The user asked "why does status take so long" — a diagnostic question — and I
immediately started editing files. This violated the intent of the interaction
and skipped the design/spec pipeline. A "Diagnostic vs. Directive" rule now
exists, but the bias toward action-completion is deep in the model. Rules help
but don't fully solve this; the agent needs to pause and classify the request
type before acting.

### Skill/validation script mismatch
The write-spec skill describes headings like `## Problem Statement` and
`## Requirements > ### Functional Requirements`, but the validation script
(`validate-spec-structure.sh`) requires `## Overview`, `## Scope`,
`## Functional Requirements` (top-level), `## Non-Functional Requirements`
(top-level). I had to rewrite the spec's heading structure after validation
failed. The skill and the script should agree — one of them is wrong, and the
agent shouldn't have to discover which by trial and error.

### Bug lifecycle is friction-heavy
Closing a bug required 7 sequential `entity(action: "transition")` calls:
reported → triaged → planned → in-progress → needs-review → verified →
closed. Each is a round-trip. The `advance` parameter exists for features but
not for bugs. For a bug that was already fixed (committed to main), this felt
like ceremony without value.

### Bugs fixed on main don't auto-advance
BUG-01KQB-54280HDP was fixed in `fc749adf` (on main, not a bug branch)
and sat in `reported` status for days. The commit was the real closure signal;
the entity state was an afterthought. The system doesn't connect "commit
referencing this bug" to "maybe advance the bug's lifecycle."

### Worktree auto-creation on bug transition is noisy
Transitioning a bug to `in-progress` auto-created a worktree
(WT-01KQF7Y0YQS36) that was never used (the fix was already on main). The
auto-worktree hook is appropriate for features that need isolation, but for
bugs being administratively advanced through lifecycle states, it creates
waste.

### No graph tools available
The codebase knowledge graph (codebase-memory-mcp) was not indexed, so
`search_graph`, `trace_path`, and `query_graph` all returned "project not
found." I fell back to `grep` for all structural navigation. The instructions
say to use graph tools over grep — but they weren't available. The fallback
worked but was slower and less precise.

## What to improve

### Add cache fast-path to entityExists()
This is the highest-ROI remaining gap. `Cache.EntityExists()` exists but
`EntityService.entityExists()` never calls it. Every cross-reference
validation hits the filesystem. The design, spec, and dev-plan for this
already exist (PROJECT/design-design-performance-follow-up and siblings).

### Add bug lifecycle fast-forward
The `entity` tool's `advance` parameter should work for bugs, or there should
be a `close` action that chains through the required intermediate states.
Reducing 7 calls to 1 would make bug hygiene feel lightweight instead of
burdensome.

### Align write-spec skill with validation script
Either update the skill to use the validation script's heading names, or
update the script to match the skill. The current mismatch creates rework for
every spec author.

### Consider a "bug resolved by commit" heuristic
If a commit message references a bug ID and the bug is in reported/triaged/
planned/in-progress, the system could suggest advancing it — or at least
surface it as a health warning ("bug referenced in commit X but still open").

### Ensure the knowledge graph is indexed
The codebase-memory-mcp index was missing for this project. The `AGENTS.md`
instructions assume it's available. Either the index should be rebuilt
automatically or the instructions should acknowledge the fallback path.
