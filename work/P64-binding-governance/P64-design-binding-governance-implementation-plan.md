# Design: Binding Governance Implementation Plan

| Field  | Value           |
|--------|-----------------|
| Date   | 2026-05-11      |
| Status | approved |
| Author | architect       |

---

## Related Work

Corpus discovery was performed per the `write-design` Phase 0 procedure. Concepts
searched: *stage binding*, *FastTrack tier routing*, *orchestration pipeline*, *tier*,
*routing*, *binding loader*. Entities searched: `P64-binding-governance`,
`B69-skills-discoverability-quick-patches`, `B57-retro-pipeline-tightening-impl`,
`B48-fast-track-impl`, `P44-model-routing-agent-launcher`. Pending classification
queue is large (411 documents) but the corpus already returns the directly relevant
documents below.

The following prior work constrains this design:

| Document | Relationship |
|----------|--------------|
| `P64-binding-governance/research-p64-research-binding-governance-investigation` | The originating diagnosis. Findings 1–16 set the problem statement; recommendations §6.1–6.3 set the phasing; Option C (§5) is the recommended architectural direction. |
| `B69-skills-discoverability-quick-patches/spec-p64-spec-b69-skills-discoverability-quick-patches` | Already merged. Three stop-the-bleeding patches (sub-agent prompt directives, status orientation skills index, doc register canonical-path validation). This design must not re-litigate those changes; it builds on their assumption that agents can now see skills. |
| `P44-model-routing-agent-launcher/design-p44-design-feature-execution-pipeline` | The layer that will sit *above* this one. P44 promotes dispatch from an orchestrator action to a side effect of lifecycle transitions; its `StageController` and `Provider` interfaces consume whatever routing this design exposes. The interface boundary at the end of Phase 3 must not require P44 to rework. |
| `P52-fast-track-orchestration/design-p52-design-fast-track-orchestration` and `P43-fast-track-architecture/design-p43-design-fast-track-architecture` | Establish the FastTrack subsystem (Subsystem A in P64 research) as the working tier-aware routing layer. This design treats FastTrack as the system of record for tier behaviour rather than re-inventing it. |
| `P51-handoff-pipeline-unification/design-p51-design-handoff-pipeline-unification` | The 3.0 context-assembly pipeline this design shares with. Pipeline steps 1–3 (resolve stage, look up binding, load skill) are the immediate consumers of the routing decisions designed here. |
| `P59-roles-skills-remediation/design-p59-design-roles-skills-remediation` | Earlier roles/skills audit and unification. Established the `RoleStore` and `SkillStore` abstractions this design uses for reachability checks. |
| `P61-handoff-resilience-binding-hardening/design-p61-design-handoff-resilience` and the `binding_loadable` health check (commit `3e98c6f2`) | Established the schema-versioned binding loader. This design tightens its production wiring rather than introducing new infrastructure. |
| `P60-whole-project-remediation-cycle/report-p60-report-whole-project-formal-review` §M4 | Independently identified the canonical/embedded `stage-bindings.yaml` drift. Phase 1 of this design makes that drift detectable. |

Decisions extracted from prior documents that constrain this design:

- **P52/P43 (FastTrack):** Tier inference is performed at entity-creation time
  (`inferTier`); the tier is **never re-inferred** afterwards. This design must
  honour that immutability — routing reads `Feature.Tier`, it never recomputes.
- **P51 (Handoff Pipeline Unification):** The 3.0 pipeline is the only assembly
  path. This design must not introduce a parallel routing path; the routing
  decision must feed into the existing pipeline's `stepLookupBinding`.
- **P44 (Feature Execution Pipeline) Decision 1:** Dispatch is a side effect of
  lifecycle transitions. The routing surface this design exposes must be
  callable from a transition hook (synchronously, no I/O) so that P44's
  `PipelineTransitionHook.AfterTransition` can use it without rework.
- **B69:** The `status()` orientation block now surfaces skill names inline.
  Reachability gaps for unreferenced skills remain — but agents can at least
  see what exists. This reduces the urgency of a runtime "skill discovery"
  facility and lets the design focus on routing correctness.

No prior design covers the full `(status, tier) → binding` routing question end
to end. P64 research is the first document to enumerate it.

---

## Problem and Motivation

The Kanbanzai MCP server cannot reliably manage its own implementation pipeline.
Three failures compound:

1. **Tier-aware routing is declared but unwired.** Every feature carries a `tier`
   field (`retro_fix`, `bug_fix`, `feature`, `critical`). Every `StageBinding`
   carries matching `Profile`/`Tier`/`Modes`/`Verifying` fields. The pipeline
   reads neither. A retro-fix feature with `status: developing` resolves to the
   generic `developing` binding, and the purpose-built `implement-retro-fix`
   skill is unreachable from any code path. (P64 Findings 1, 2, 9.)
2. **Bug-fix has no pipeline binding at all.** Twelve `BugStatus` constants map
   to zero stage-binding keys. The system creates a worktree when a bug
   transitions to `in-progress`, then strands the agent inside it — the
   pipeline's lifecycle validator rejects every `BugStatus`. The 19 active
   health warnings ("bug … reached 'closed' without passing through
   needs-review") are the downstream symptom. (P64 Finding 7.)
3. **Validation is loaded but not run.** `ValidateBindingFile` exists, performs
   the cross-cutting checks (stage-name allowlist, role/skill existence,
   orchestration enum) and is never invoked in production. Two production
   bindings (`retro-fixing`, `doc-publishing`) would fail it if it were. The
   embedded consumer copy (`internal/kbzinit/stage-bindings.yaml`) drops three
   stages and the schema marker, and there is no test that detects the drift.
   Five of twelve declared stages are missing from `validStages`. (P64
   Findings 4–8.)

Underlying these is a **two-subsystem confusion**. `FastTrackConfig` (code,
working) and the `stage-bindings.yaml retro-fixing:` block (YAML, dead) share
the vocabulary of "tiers" — `retro_fix`, `bug_fix`, `human`, `auto`,
`conditional` — but are not connected at runtime. The YAML cannot route on
tier without code support, and the code never consults the YAML's tier-aware
fields. (P64 Finding 3.)

**Who is affected.**

- *Sub-agents implementing retro-fix work* receive the wrong skill (generic
  `implement-task` instead of `implement-retro-fix`), so retro-fix features
  are over-engineered, over-reviewed, and decomposed at the wrong granularity.
- *Bug fixers* receive `lifecycle-validation` errors when calling `next` or
  `handoff` against a bug, despite the system having created a worktree for
  them. They either give up, work outside the workflow, or skip review
  entirely (hence the 19 health warnings).
- *Consumer projects* receive a binding file missing three stages and the
  schema marker. Downstream consumer behaviour drifts silently from the
  project's own dogfood.
- *Future contributors* must hand-edit four parallel allowlists
  (`validStages`, `workableStatuses`, `FeatureStatus*`, `BugStatus*`) when
  adding or renaming a stage. The 2026-04-28 `plan-reviewing → batch-reviewing`
  rename is the exhibit: YAML updated, three of four follow-up edits missed.

**What happens if nothing changes.** The pattern of "additions to schema
surface without commensurate additions to the consumer code" (P64 Finding 13)
continues. Every new tier, stage, or status adds to the ten-surface
synchronisation contract (P64 Finding 10), which is already past the
contributor-error inflection point identified by the cited literature. The
P44 model-routing layer cannot safely build on this foundation: it would
inherit silent failure modes and a routing question with no single answer.

---

## Design

The design organises work into three phases. Each phase brings the MCP server
to a state where it can correctly orchestrate the next phase's implementation
using its own pipeline. Phases are structural, not cosmetic — Phase 2's
implementation depends on Phase 1's startup validation; Phase 3's code
generator depends on Phase 2's resolved tier semantics.

### Component map

Three subsystems exist today. The design preserves their separation but
clarifies responsibilities and the contract between them.

| Subsystem | Responsibility | Source of truth |
|-----------|----------------|-----------------|
| **A. Tier router (`FastTrackConfig` and friends)** | Decides *which agent profile and gate sequence applies* for a given (status, tier). Owns lifecycle decisions: gate auto/human/conditional, max review cycles, DoD variant. | Code (`internal/config/config.go`, plus a new thin `internal/binding/router.go` introduced in Phase 3). |
| **B. Stage-content registry (`stage-bindings.yaml`)** | Describes *how each agent acts* per stage: role, skill, sub-agent topology, document template, effort budget. No routing decisions, no tier awareness. | YAML (`.kbz/stage-bindings.yaml`). |
| **C. Loader / validator / lookup (`internal/binding/`)** | Parses YAML, runs cross-checks, exposes a typed lookup. | Code, with strict load-time enforcement after Phase 1. |

The contract: **A decides which binding key to load, B describes what the
binding contains, C enforces both halves are valid before the server accepts
traffic.** Today the contract is implicit, allowing A to bypass B (FastTrack
ignores the YAML's tier fields) and B to declare routing intentions C cannot
enforce (the dead `retro-fixing` block).

### Phase 1 — Stop the lying

**Goal.** Convert the silent failures enumerated in P64 Finding 11 into loud
failures at server startup. After Phase 1, the MCP server refuses to start
when its own binding file is broken. This is the precondition for any
further work, because every Phase 2 and Phase 3 change is itself a binding
change that must pass validation.

**Components changed.**

- **Loader wiring (Subsystem C).** Replace the four production
  `LoadBindingFile` call sites with `BindingRegistry.Load`, which runs
  `ValidateBindingFile`. Server startup fails on validation errors for
  any binding file — project-owned or consumer-owned. The asymmetric
  warn-window approach was considered and rejected (see Decision 3).
- **Allowlists (Subsystem C).** `validStages` and `workableStatuses` are
  brought into agreement with the canonical YAML — `merging`, `verifying`,
  `batch-reviewing`, `doc-publishing`, `retro-fixing` added; stale
  `plan-reviewing` removed. This is a no-op for runtime routing today, but
  it is the precondition for `ValidateBindingFile` to pass.
- **Embedded copy (Subsystem C).** A CI test asserts structural equality
  between `.kbz/stage-bindings.yaml` and `internal/kbzinit/stage-bindings.yaml`.
  The embedded copy is brought in line. This catches the drift that
  P60 §M4 and P64 Finding 8 both independently identified.
- **Reachability tests (Subsystem C).** A test suite asserts: every
  `roles`/`skills` reference resolves on disk; every `FeatureStatus*` and
  `BugStatus*` constant is either bound to a binding or explicitly listed
  as "not handled by binding pipeline"; `ValidateBindingFile` succeeds
  against the canonical file.
- **Two existing failing bindings.** `retro-fixing` is removed from YAML
  in Phase 2 (see Decisions); for Phase 1 it is given the minimum fields
  needed to pass validation (a passthrough `orchestration: single-agent`,
  `roles: [orchestrator]`, `skills: [orchestrate-development]`) so Phase 1
  can land independently. `doc-publishing` requires either adding
  `pipeline-coordinator` to `validOrchestrations` or moving the binding's
  routing-relevant fields to a separate validated section. The design
  recommends the former — a one-line code change that admits the existing
  semantics without expanding the schema.

**What Phase 1 leaves operational.** All current routing behaviour is
preserved. No feature is rerouted. No skill is added or removed. The
server's behaviour is identical except that a bad binding file now
prevents startup instead of corrupting routing silently.

**Litmus test.** After Phase 1, the MCP server can correctly identify
that the `retro-fixing` and bug-fix bindings are broken; the question of
*how* to fix them (Phase 2) is a separate, validatable question.

### Phase 2 — Resolve the two failures and reduce drift

**Goal.** Make retro-fix and bug-fix features route to the right skills.
After Phase 2, every feature and bug status routes to a binding whose
skill is fit for purpose. This is the litmus test that the MCP server
can now manage Phase 3's own multi-tier work — Phase 3 itself contains
both bug-fix work (the validation hardening) and feature work (the
router extraction).

**Components changed.**

- **Retro-fixing block (Subsystems A + B).** The `retro-fixing` block in
  YAML is removed entirely. `FastTrackConfig` is documented in code and
  in the binding YAML header as the system of record for tier-aware
  behaviour. The `implement-retro-fix` skill is wired into the FastTrack
  `developing` path through a tier-conditional skill include: when the
  tier router resolves a `retro_fix` feature, the binding's `skills` list
  is post-processed to substitute `implement-retro-fix` for
  `implement-task`. The pipeline's `stepLoadSkill` is unchanged; only the
  resolved skill list differs.
- **Bug bindings (Subsystem B).** Two new top-level binding keys are
  added: `bug-developing` and `bug-reviewing`. `workableStatuses` is
  extended to include the `BugStatus` constants `in-progress` and
  `needs-review`. The pipeline's `stepResolveStage` is taught to map
  `BugStatus` values to `bug-*` binding keys via a small
  `bugStatusToBindingKey` function (a pure mapping table). All other
  `BugStatus` values (`reported`, `triaged`, `closed`, etc.) remain
  out-of-pipeline by explicit declaration. The `WorktreeTransitionHook`
  is unchanged; it now creates a worktree the pipeline can actually use.
- **Tier-aware fields (Subsystem B).** The orphaned `Profile`, `Tier`,
  `Modes`, `Verifying` fields on `StageBinding` are removed from the
  model. Any field that survives must have a runtime consumer. This
  reverses the silent schema expansion in commit `f75b47cb`.
- **Synchronisation surface count (Subsystem C).** Drops from 10 to 7:
  the embedded-vs-canonical drift is now CI-enforced; the
  validStages-vs-YAML drift is now CI-enforced; the orphaned schema
  fields are gone.

**Migration impact on consumers.** Removing `retro-fixing` from the
embedded YAML is safe — no consumer can have been routing through it
because it is unroutable. Adding `bug-developing` and `bug-reviewing` to
the embedded YAML is purely additive. The schema version stays at `2`.

**Litmus test.** After Phase 2, calling `next` or `handoff` against a
bug or a retro-fix feature returns an assembled prompt with the correct
skill, role, and tool hints. The 19 health warnings about bugs reaching
`closed` without review are eliminable: a real review pipeline now
exists for them.

### Phase 3 — Establish the contract for P44

**Goal.** Codify Option C (hybrid: code routes, YAML describes) as the
explicit architectural direction. Reduce the synchronisation surface
count further. Expose a clean seam that P44's stage controllers and
provider integration can consume without rework.

**Components changed.**

- **Router extraction (new component, Subsystem A).** A new
  `internal/binding/router.go` file owns the
  `(status, tier) → BindingResolution` mapping. Located alongside the
  binding loader and validator because routing is a binding concern;
  the pipeline package (`internal/context/`) consumes routing
  decisions but does not own them. `BindingResolution` is a
  small struct: `BindingKey string; SkillOverrides []string; ModeProfile
  *FastTrackTier`. `Resolve` is a pure function: no I/O, no logging,
  callable from a transition hook. FastTrackConfig is one of its inputs;
  the bug-status mapping and tier-conditional skill substitution from
  Phase 2 are absorbed into it. The pipeline's `stepLookupBinding` calls
  `Resolve` instead of looking up the bare `state.Stage`.
- **Generated registry (Subsystem C).** A single source-of-truth YAML
  companion file (`internal/binding/routing.yaml`) declares which
  feature/bug statuses are routable and which binding key each maps
  to. Located inside `internal/` because routing is a code concern
  under Option C; placing it in `.kbz/` would invite consumer edits
  the design explicitly does not support and would contradict the
  contract documented in `stage-bindings.yaml`'s header. `validStages`,
  `workableStatuses`, the `FeatureStatus*` and `BugStatus*` enum
  constants, and the `bugStatusToBindingKey` table from Phase 2 are
  generated from it via `go generate`. After Phase 3, four hand-edited
  surfaces collapse to one. (The design names this approach but does
  not implement the generator — that is implementation detail for the
  spec.)
- **Header documentation (Subsystem B).** The canonical
  `stage-bindings.yaml` gains a header comment that names the contract:
  *"This file describes how agents act per stage. It does not decide
  which agent runs — that is owned by `internal/binding/router.go` and
  `internal/config` (FastTrack)."* The boundary is documented at the
  point of confusion.
- **P44 interface (Subsystem A).** The exported surface for P44 is
  `Resolve(featureID) → BindingResolution`. A stage controller calls
  it, receives the binding key, and the existing 3.0 pipeline does the
  rest. P44's `Provider` interface (token tracking, fallback chains)
  sits *above* this seam and never sees routing concerns.

**Synchronisation surface count.** Drops from 7 (post-Phase 2) to 4: the
canonical YAML, the routing companion file, the role files, and the
skill files. Each pair has either a CI test or a generator enforcing
consistency.

**Litmus test.** After Phase 3, P44 can build its `StageController` and
`PipelineTransitionHook` against the `Resolve` interface without
requiring further changes to the binding subsystem. The "two parallel
tier-routing subsystems" finding from P64 §3 no longer applies — there
is one router, with FastTrack as one of its inputs.

### Failure mode handling

The P64 research finds that failure modes are predominantly silent.
Each Phase 1 change converts a silent mode to an explicit one. The
table below names the policy for each failure surface.

| Failure | Phase | Behaviour |
|---------|-------|-----------|
| Any `stage-bindings.yaml` (project or consumer) fails `ValidateBindingFile` at startup | 1 | **Hard fail.** Server refuses to start. Validation errors are surfaced with a fix hint and a pointer to `kbz init --upgrade` for unmodified consumer files. |
| Embedded YAML drifts from canonical | 1 | **CI fail.** Structural equality test in the test suite. |
| Skill referenced in YAML missing on disk | 1 | **Startup fail** (via `ValidateBindingFile` already covers this; just needs to be invoked). |
| Skill on disk unreferenced by any binding | 1 | **CI warn**, not fail. Some skills (`audit-codebase`, `prompt-engineering`) are direct-trigger and intentionally unbound; the test asserts they appear on a known-allowlist. |
| Role referenced but missing | 1 | **Startup fail.** |
| Pipeline encounters a status with no binding | 2 | **Per-task fail with an actionable message** ("status `X` has no binding; either add one to `stage-bindings.yaml` or list it as out-of-pipeline in `stage-routing.yaml`"). Same loudness as today, better message. |
| Tier router resolves an unknown tier | 2 | **Hard fail at `Resolve` call.** Tiers are an enumerated set; unknown tiers indicate corrupt entity state. |
| `bug_fix` feature in a status that has no `bug-*` binding | 2 | **Per-task fail with explicit out-of-pipeline message** (same template as above). |

All binding files are validated identically. The brief's concern
that "a startup validation gate that hard-fails on schema drift would
break consumer projects" is addressed by Decision 3 — the cost of an
upgrade-time break is judged lower than the cost of a permanent
two-class validation policy and a warn-window mechanism that may not
be noticed by consumers in time.

### Migration strategy for consumer projects

Two scenarios:

1. **Consumer has not customised `stage-bindings.yaml`.** `kbz init
   --upgrade` overwrites the embedded copy in place (existing
   behaviour, gated by the `kanbanzai-managed: true` marker). After
   Phase 1, the embedded copy passes validation strictly. No action
   required by the consumer.
2. **Consumer has customised `stage-bindings.yaml`.** The marker is
   absent, so `kbz init --upgrade` does not overwrite. On first start
   after upgrade, the server hard-fails if the customised file does
   not pass validation. The error message names each violation and
   the canonical fix.

To make the upgrade-time break recoverable, Phase 1 also adds a
`kbz binding doctor` command that runs the validator against the
current file and reports issues *without* starting the server.
Consumers can run it before upgrading (or before restarting after
upgrade) to surface and fix issues offline.

The schema version stays at `2` throughout. A schema bump would be
warranted only if Subsystem B's content shape changed; under Option C
it does not. (A future consumer-facing extension to declare new
stages would be a schema-v3 question, deliberately out of scope per
the brief.)

### Interface contract for P44

P44 will dispatch sub-agents from transition hooks. It needs three
things from this design:

1. **A pure routing function:** `Resolve(featureID) → BindingResolution`.
   No I/O, no logging in the hot path, callable synchronously from
   `AfterTransition`. Phase 3 delivers this as
   `internal/binding/router.go`.
2. **A typed binding payload:** `BindingResolution` exposes the binding
   key, the sub-agent profile (role + skill + topology), the
   tier-derived gate mode (`auto`/`human`/`conditional`), and the
   max-cycles cap. P44's `StageController` reads these to wire its
   bounded loops without re-deriving them from FastTrackConfig.
3. **A stable failure surface:** failures from `Resolve` are typed
   (`ErrNoBinding`, `ErrUnknownTier`, `ErrSkillMissing`) so P44 can
   raise checkpoints with specific messages. P44's `Provider` interface
   is unaffected; routing failures never reach the provider.

P44 does not need (and is not given): the YAML's role/skill content
(it consumes the existing 3.0 pipeline output), the FastTrackConfig
struct directly (it consumes `BindingResolution`), or any view of the
binding loader.

---

## Alternatives Considered

### Alternative A — Harden instruction-only (P64 Option A)

Wire `ValidateBindingFile` at startup, fix the allowlists, sync the
embedded copy — and stop there. Leave the two tier subsystems
unconnected.

**Trade-offs.** Low cost; small migration risk. Does not address that
the YAML cannot route on tier without code support — the architectural
confusion remains, and `implement-retro-fix` stays unreachable. Bug
worktrees remain stranded.

**Why rejected.** Equivalent to doing only Phase 1 of the recommended
plan. The brief explicitly requires §6.2 (Option D resolution) and
§6.3 (Option C direction) be addressed.

### Alternative B — Full code-managed pipeline (P64 Option B)

Move all routing into Go. Reduce `stage-bindings.yaml` to documentation.
Consumers would patch Go and rebuild to customise.

**Trade-offs.** Highest correctness guarantee. Lowest configurability.
Forces every consumer with a non-default binding to rewrite, then
recompile. Large migration cost (3–4 weeks per P64 §5). Existing
FastTrack subsystem is essentially this design point in miniature —
extending it to all routing duplicates that work without adding value.

**Why rejected.** Brief explicitly excludes this option. Subsystem A
already provides the working code-managed routing this option proposes
to build from scratch.

### Alternative C — Hybrid (P64 Option C — recommended and adopted)

Code owns routing invariants (`(status, tier) → binding`); YAML owns
agent behaviour content (role, skill, topology, document template).
This is the design above.

**Trade-offs.** Best balance of correctness and configurability per the
P64 §5 trade-off table. Aligned with industry practice (LangGraph,
CrewAI, AutoGen — see P64 Finding 14). Migration is additive: existing
YAML keeps working. Two-week implementation cost is concentrated in
Phase 2.

**Why chosen.** Resolves the architectural confusion without removing
the consumer-editable surface. Honours the FastTrack subsystem that
already exists. Provides the seam P44 needs without designing P44.

### Alternative D — Retire dead bindings, promote FastTrack (P64 Option D)

Delete the `retro-fixing` block; document FastTrack as the system of
record; add bug-fix bindings or document bugs as out-of-pipeline. Stop
short of structural change.

**Trade-offs.** Smallest cost. Resolves the two specific failures
(retro-fix unreachable, bugs unrouted). Leaves the underlying
two-subsystem confusion unresolved and the ten synchronisation
surfaces in place. P44 would inherit the same confusion this design
exists to remove.

**Why partly adopted, not adopted whole.** Phase 2 of the recommended
design *is* Option D, executed as a unit. The recommended design
extends past Option D into Phase 3 (Option C structural direction),
because stopping at Option D leaves P44 with an unstable foundation.

### Alternative E — Do nothing / status quo

Leave the system as it is.

**Trade-offs.** Zero cost. The 19 active health warnings persist; new
ones accumulate. The next contributor who renames a stage repeats the
2026-04-28 mistake. P44 cannot proceed.

**Why rejected.** The brief identifies P44 as blocked on this work.
The status quo is not a trajectory the project can accept.

### Alternative F — Schema-v3 YAML to support tier routing declaratively

Extend the YAML schema to declare `(status, tier) → binding` mappings
in YAML. Validate strictly. No code-side router required.

**Trade-offs.** Most expressive. Pushes the YAML schema past the
contributor-comprehensibility ceiling identified in P64 Finding 16.
Consumers must learn a richer schema. The system gains more
synchronisation surfaces, not fewer.

**Why rejected.** Brief explicitly excludes this. P64 Finding 16
identifies adding YAML surface as the wrong direction.

---

## Decisions

### Decision 1 — Resolve `retro-fixing` by deleting the YAML block and wiring `implement-retro-fix` through FastTrack

**Context.** The `retro-fixing` YAML block is unroutable: no feature ever
has `retro-fixing` as a status. The block was added in commit `b4b2de39`
without a tracked design. The `implement-retro-fix` skill exists on disk
but is referenced by no binding.

**Rationale.** FastTrack is already the working tier-aware routing
system. The YAML block is the second of two parallel mechanisms,
silently broken. Removing it eliminates the duplication; routing
`implement-retro-fix` through a FastTrack tier-conditional include in
the `developing` path makes the skill reachable. This is P64 §6.2
recommendation 5, executed in full.

**Consequences.** Positive: the two-subsystem confusion in this corner
is resolved; the orphan skill becomes reachable. Negative: the
gated-mode profile schema (`Profile`/`Tier`/`Modes`/`Verifying`) is
removed from `StageBinding`. Any consumer who copied this block from
the canonical YAML loses it; mitigation is Phase 1's warn-window for
consumer files plus a release note.

### Decision 2 — Add `bug-developing` and `bug-reviewing` bindings rather than declare bugs out-of-pipeline

**Context.** Twelve `BugStatus` values currently route to zero
bindings. The system creates a worktree for `BugStatus.in-progress`
and then strands the agent. 19 active health warnings show bugs
closing without review.

**Rationale.** The two options the brief offers are: (a) add
`bug-developing` / `bug-reviewing` bindings, or (b) document bugs as
FastTrack-only and explicitly out-of-pipeline. Option (a) is chosen
because the system already does the expensive part — creates the
worktree — and only fails at the cheap step (skill load). Adding two
bindings closes the gap with the smallest amount of new schema. Option
(b) would require unwinding the worktree-on-`in-progress` behaviour or
permanently leaving agents to work outside the workflow, which is the
worst-of-both-worlds state P64 Finding 7 identifies.

**Consequences.** Positive: bugs gain a real review pipeline; the 19
health warnings become eliminable. Negative: two more binding keys to
maintain; `workableStatuses` grows. Mitigation: Phase 3's generated
registry collapses these surfaces into one.

### Decision 3 — Validation failure mode is hard-fail at startup for all binding files

**Context.** The brief flags that "a startup validation gate that
hard-fails on schema drift would break those projects." Three
policies were considered: (a) hard-fail for everyone, (b)
warn-and-continue for consumers during a defined release window then
flip to hard-fail, (c) permanent warn-mode for consumers, hard-fail
for the project file only.

**Rationale.** The warn-window option (b) adds significant
implementation complexity (a `strictValidation` flag, a release
checklist item to flip it, log-surfacing through `status`/`health`
because MCP startup logs are easy to miss) for moderate benefit. Its
value depends entirely on consumers reading log messages they
historically have not needed to read. The permanent warn option (c)
preserves the architectural invariant for the project but leaves
consumer files quietly broken indefinitely — the same silent-failure
class this entire design exists to eliminate.

Hard-failing for everyone (a) is the simplest, most honest, and most
consistent policy. Kanbanzai's user base is small and technical;
upgrade breakage is recoverable through the new `kbz binding doctor`
command. If this causes meaningful consumer pain in practice, the
policy can be revisited and a warn-window introduced as a follow-up
feature — but the cost of starting strict and relaxing is far lower
than the cost of starting lenient and tightening.

**Consequences.** Positive: a single, mechanical validation policy;
no release-cadence dependency; no policy state to manage; no
two-class code path. Consumer files are held to the same standard as
the project's own. Negative: a consumer with a customised binding
file that drifts from the validator's expectations will see a
startup failure on upgrade. Mitigation: `kbz binding doctor` (added
in Phase 1) can be run before restart to surface issues offline; the
release note for the Phase 1 release prominently documents the
policy change.

### Decision 4 — Reduce synchronisation surfaces via a single routing companion file generated into code

**Context.** Ten synchronisation surfaces exist today (P64 Finding 10).
The four hand-edited tables — `validStages`, `workableStatuses`,
`FeatureStatus*` constants, `BugStatus*` constants — are the highest
drift risk because every stage rename touches all four.

**Rationale.** A single small YAML companion file
(`internal/binding/routing.yaml`) declares which statuses route to
which binding keys. A `go generate` step produces the four tables
from it. After Phase 3, renaming a stage is a one-line edit. This is
P64 §6.3 recommendation 9. The companion file lives inside
`internal/` because routing is a code concern under Option C;
placing it in `.kbz/` would invite consumer edits the design
explicitly does not support.

**Consequences.** Positive: the highest-drift surfaces become
generated; the surface count drops from 7 to 4. Negative: a
`go generate` step and a regenerated-files-checked-in convention must
be established. Phase 3 designs the approach and locates the
generator; the spec specifies the generator's input/output and the
build wiring.

### Decision 5 — `doc-publishing` retains `pipeline-coordinator` orchestration; the validator admits it

**Context.** `ValidateBinding` accepts `single-agent` and
`orchestrator-workers`. `doc-publishing` declares `pipeline-coordinator`,
which the validator would reject. The doc-publishing routing is
consumed via a separate code path, not the main pipeline.

**Rationale.** Adding `pipeline-coordinator` to `validOrchestrations`
is a one-line code change that admits the existing semantics. The
alternative — moving `doc-publishing` out of the binding file or
reshaping it — is a much larger change with no behavioural benefit.

**Consequences.** Positive: Phase 1 unblocks; existing `doc-publishing`
behaviour is preserved. Negative: a third orchestration mode now exists
in the validator. Mitigation: the third mode must remain a closed set
(no further additions) and is documented in the validator's enum
comment.

---

## Open Questions

None. The five questions present in the initial draft were resolved
before approval:

- **`pipeline-coordinator` under Option C** — resolved: stays outside
  `Resolve`. The router answers "what binding does this feature/bug
  task route to?" `doc-publishing` is not a feature lifecycle stage
  and is triggered separately.
- **Tier-conditional skill substitution mechanism** — resolved: data
  table inside the router. Composes with the Phase 3 generated
  registry; one fewer code path to read.
- **Router package location** — resolved: `internal/binding/router.go`.
  Routing is a binding concern.
- **Validation failure mode for consumers** — resolved: hard-fail for
  all binding files, with `kbz binding doctor` as the recovery path.
  See Decision 3.
- **Companion file location** — resolved: `internal/binding/routing.yaml`.
  Code-internal, not consumer-customisable. See Decision 4.
