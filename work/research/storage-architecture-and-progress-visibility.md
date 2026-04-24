| Field  | Value |
|--------|-------|
| Date   | 2026-04-24T12:02:22Z |
| Status | Draft |
| Author | Claude Opus 4.7 (storage architecture researcher role) |
| Supersedes relationship | Extends `work/research/state-backend-comparison.md` (GPT-5.4, 2026-04-22); does not supersede it |

## Research Question

As Kanbanzai scales from solo/small-team use toward distributed collaborative
development, what is the proportional architectural response to two distinct
pressures on the current Git-native YAML state model:

1. **Commit-log noise** — workflow transitions dominating Git history and
   degrading its usefulness for code review, and
2. **Cross-clone progress visibility** — the difficulty of seeing team-wide
   progress when canonical state is scattered across individual clones and
   worktree branches?

This report evaluates Git-native evolution, a fully centralised database
backend, and a family of hybrid approaches, and recommends which to adopt
now, which to defer, and which to rule out.

## Executive Summary

The two pressures above are **separate problems** and should be solved separately.

- **Commit-log noise** is real and empirically confirmed: ~67% of recent commits
  in this repository carry the `workflow(...)` prefix. It is a
  commit-granularity problem, not a storage-model problem, and it is solved
  directly and proportionally by the existing Git-native evolution design
  (JSONL per-entity transition logs + coarser commits), with one addition: a
  custom JSONL merge driver to prevent silent audit-trail loss.

- **Cross-clone progress visibility** is a distinct observability problem that
  does not require centralising canonical authority. It is solved
  proportionally by a push-on-milestone convention plus remote-aware status
  commands (Hybrid C), and — if that proves insufficient — by a read-only
  aggregator service that fetches from `origin` and exposes a team dashboard
  (Hybrid A). Canonical authority remains in Git in both cases.

- A **fully centralised database backend** (canonical authority moved to
  Postgres) is not the proportional response to either pressure. It solves a
  broader class of problems at materially higher operational cost, conflicts
  with the current product identity and viewer contract, and should remain a
  strategic option — not a planned milestone — until a concrete
  multi-writer, sub-push-latency coordination requirement appears.

- A derived SQLite cache **already exists** in `internal/cache/` and already
  accelerates entity reads. This weakens the "rich queries" argument for a
  centralised backend and shifts one of the design document's "future phases"
  into the present.

**Primary recommendation (confidence: high):** Ship Git-native evolution with a
JSONL merge driver, adopt push-on-milestone + `--remote` aware status for team
visibility, and invest in backend-neutral service contracts as optionality
insurance. Do not schedule centralised-backend implementation work.

## Scope and Methodology

**In scope:**

- commit-log noise as a specific, measurable problem
- cross-clone progress visibility as a specific team-observability problem
- Git-native evolution, centralised backend, and hybrid approaches
- alignment with the current product identity (Git-native, viewer-as-clone)
- empirical state of the codebase (existing cache, current concurrency model)
- migration cost and reversibility of each option

**Out of scope:**

- schema design for a centralised database
- implementation task breakdown
- benchmarking of storage engines
- multi-tenant security / hosting model
- product-strategy decisions about target customer segment

**Primary sources reviewed:**

- `work/design/transition-history-storage.md` — Git-native evolution proposal
- `work/design/centralized-state-server.md` — database backend proposal
- `work/design/public-schema-interface.md` — viewer contract
- `work/design/git-commit-policy.md` — commit-history quality constraints
- `work/design/kanbanzai-1.0.md` — Git-native product framing
- `work/research/state-backend-comparison.md` — prior comparison (2026-04-22)
- `internal/cache/cache.go`, `internal/mcp/server.go`,
  `internal/mcp/server_warmup_test.go` — existing derived SQLite cache
- `.kbz/state/` directory layout and `git log` of the working repository

**Secondary / comparative sources:**

- HashiCorp Terraform state backend documentation
- Pulumi state backends (DIY and Pulumi Cloud)
- Jujutsu (jj) operation log design
- Dolt version-controlled SQL database (commit graph, scaling article)
- Steve Yegge's Beads (SQLite → Dolt migration)
- Spec-Kitty issues #569 and #574 (real-world JSONL merge-driver failure)

**Method:** direct reading of the primary design documents; empirical
inspection of the current repository (commit patterns, file layout, code);
comparison against how comparable developer tools solve analogous problems;
synthesis into findings with graded confidence and proportional
recommendations.

## Findings

### Finding 1: Commit-log noise is empirically confirmed and proportional to workflow activity, not entity volume

Direct measurement of the last 30 days of `git log` in this repository:

- total commits: **1611**
- commits prefixed `workflow(...)`: **1085** (~67%)
- commits prefixed `docs`: 195; `feat`: 151; `fix`: 55; `chore`: 52; `test`: 13

Individual entities accumulate 7–9 workflow commits each across their
lifecycle (e.g. `BUG-01KPQY347TABX`, `FEAT-01KPX5CVVY357`). This matches
exactly the "reported → triaged → reproduced → planned → in-progress → …"
noise pattern identified in `work/design/transition-history-storage.md`.

The repository contains ~133 features, ~537 tasks, and ~9 bugs. The problem
is therefore **not** entity volume or user count — it is **commit-per-transition
density** in a system where each entity naturally passes through many
lifecycle states. Reducing commit granularity eliminates the pain without
changing storage shape.

- Evidence grade: **primary / current / empirical**
- Confidence: **high**

### Finding 2: The current concurrency model does not exercise file-based scaling limits

The worktree discipline and the agent-interaction protocol already enforce
single-writer-per-worktree. The routinely concurrent path is **one
orchestrator on `main` plus one agent per worktree branch** — parallelism
bounded by active worktrees, not by users. The failure modes Postgres
solves (row-level locking, multi-user optimistic concurrency, cross-session
transactions) are not failure modes Kanbanzai currently experiences.

The centralised-state design acknowledges this indirectly: its motivation
is framed around *future* team deployments, not current pain.

- Source: `work/design/centralized-state-server.md` §Problem and Motivation;
  `work/design/agent-interaction-protocol.md`; `.kbz/state/worktrees/` layout.
- Evidence grade: **primary / current**
- Confidence: **high**

### Finding 3: A derived SQLite cache already exists and is actively doing query-acceleration work

`internal/cache/cache.go` opens a WAL-mode SQLite database at
`.kbz/cache/kbz.db`, rebuilt from canonical YAML at MCP server startup
(`RebuildCache()` in `internal/mcp/server.go`). `.kbz/cache/` is explicitly
excluded from the public schema (`work/design/public-schema-interface.md`
§2.3: "Derived SQLite cache — rebuilt on demand, not canonical").
The cache already powers `EntityService.Get()` and `List()` fast paths
(see `server_warmup_test.go`, cache-first read tests).

Implications:

- The "rich queries and analytics" argument for a centralised backend is
  significantly weaker than it appears; query acceleration is already
  achievable locally.
- The transition-history design's "Phase 4: optional indexing" is behind
  the codebase. Derived SQLite indexing is an existing pattern, not a
  future option.
- Extending the cache to cover transition events is a natural next move;
  introducing a second SQLite file would be redundant.

- Evidence grade: **primary / current / empirical**
- Confidence: **high**

### Finding 4: JSONL append-only logs have a known merge-conflict failure mode that requires explicit mitigation

The transition-history design assumes per-entity sharding is sufficient to
contain merge conflicts. Spec-Kitty issues #569 and #574 ("P0 release
blocker") document a concrete real-world failure with effectively the same
architecture: default Git 3-way merge on an append-only JSONL produces
content conflicts when both branches append after the fork point, and
resolving with either `--ours` or `--theirs` silently drops events. Their
fix was a custom `.gitattributes` merge driver that performs union-merge
with deduplication by a stable per-event ID.

Kanbanzai's proposed per-entity sharding reduces but does not eliminate
this risk: a single entity can legitimately receive transitions on `main`
(e.g., orchestrator marks a feature `reviewing`) and on a worktree branch
(e.g., agent marks a task `done` which cascades up) inside the same fork
window. Without a union-merge driver, the same data-loss class appears.

- Source: Spec-Kitty GitHub issues #569, #574 (secondary, current);
  `work/design/transition-history-storage.md` §Failure modes (primary).
- Evidence grade: **mixed primary/secondary / current**
- Confidence: **high** on the failure mode;
  **medium** on how often it would occur in Kanbanzai's specific pattern.

### Finding 5: The comparable-system pattern is "local-canonical + optional service backend," not "file replacement"

Four data points:

- **Terraform:** default backend is local filesystem; remote backends (S3,
  Postgres, Terraform Cloud) are opt-in per configuration; one backend per
  workspace.
- **Pulumi:** DIY backends (filesystem, S3, Postgres) vs. Pulumi Cloud;
  each stack chooses one backend at a time.
- **Jujutsu:** operation log is file-based and designed for lock-free
  concurrency on shared filesystems; no central server.
- **Dolt / Beads:** "embedded mode" (single writer, local, no server) vs.
  "server mode" (multi-writer, needs a running Dolt server); chosen per
  project. Beads migrated JSONL → SQLite → Dolt as its concurrency
  requirements grew, but preserved an embedded mode for solo users.

The convergent lesson: **systems that serve both solo and team workflows
expose backend selection as configuration, keep canonical authority in one
backend at a time, and ship with a solid local-first default**. None of
these systems operate with canonical authority split between local and
remote simultaneously.

- Evidence grade: **secondary / current**
- Confidence: **high**

### Finding 6: Retiring Git as the canonical transport would break the viewer contract

`work/design/public-schema-interface.md` §2.1 defines the public schema as
**committed state** — the viewer reads from a Git clone and sees only what
has been committed. A centralised backend where the database is canonical
(Model B in the centralised-state design) means the repository no longer
contains workflow state at all; the viewer contract must be redesigned
around a service API rather than a Git clone.

Model C (dual backend, per-project selection) preserves the viewer
contract for Git-native projects but creates an ambiguity for centralised
projects: what does the viewer show when canonical data lives in a
database? The centralised-state design acknowledges this indirectly but
does not resolve it.

- Source: `work/design/public-schema-interface.md` §2.1–2.3;
  `work/design/centralized-state-server.md` §Repository relationship in
  centralized mode, §Operational model in centralized mode.
- Evidence grade: **primary / current**
- Confidence: **high**

### Finding 7: Operational burden of a Postgres backend is strictly additive and redefines the adoption surface

A centralised canonical backend introduces: authentication, authorisation,
schema migrations, backup/restore, service availability, multi-tenant
isolation, audit retention, and incident response for state corruption.
None exist in the current product. For a tool whose identity is "Git-native
workflow system," these are not incidental costs — they redefine who can
adopt it. Solo developers and small teams lose "clone the repo and run the
MCP server" as the onboarding path.

Treating operational complexity as a first-class evaluation criterion (not
an implementation detail) is essential to avoid the **Operational Amnesia**
anti-pattern.

- Source: `work/design/centralized-state-server.md` §Operational model in
  centralized mode, §Failure modes and handling.
- Evidence grade: **primary / current**
- Confidence: **high**

### Finding 8: The prior comparison reaches the same conclusion but predates the cache and does not cite the JSONL merge-conflict failure

`work/research/state-backend-comparison.md` (2026-04-22) concludes
Git-native evolution is the lower-risk near-term direction and that
centralised state is a strategic expansion path. This report agrees with
both. However, the prior research:

- does not account for `internal/cache/` (implemented later) — which
  shifts Finding 3's weight in the trade-off matrix, and
- does not cite the concrete JSONL merge-driver failure mode from
  Finding 4 — which suggests the Git-native evolution path needs a
  specific deliverable (a custom merge driver) that is not currently in
  the design document.

Both adjustments tighten the case for Git-native evolution rather than
weaken it; they do not change the direction of the recommendation.

- Evidence grade: **primary / mixed recency**
- Confidence: **medium-high**

### Finding 9: Cross-clone progress visibility is a distinct problem from concurrency, commit noise, or canonical authority

In a distributed team, each contributor works in their own clone with
their own worktrees. Canonical `.kbz/state/` mutations are visible to
others only after commit **and** push **and** fetch/pull by the reader.
Consequences:

- A team lead viewing `main` sees only merged work, not in-flight agent
  activity on worktree branches.
- A contributor cannot see whether someone else has already claimed
  `FEAT-X` until the claim is pushed.
- Unpushed state mutations (e.g. batched for milestone commits) are
  invisible to anyone else, even though they exist on disk locally.
- Aggregate progress ("what is the state of Plan P29 across all
  contributors right now") requires fetching every active worktree
  branch from every active clone.

This is a **visibility / observability** problem, not an authority or
concurrency problem. The centralised-state design addresses it as a side
effect, which conflates two different reasons to adopt a server.
Visibility can be solved without moving canonical authority. Conflating
the two leads directly to **Premature Database Syndrome**: recommending a
full Postgres canonical backend when a thin read-replica would suffice.

GitHub itself is the canonical hybrid example: canonical authority lives
in Git; GitHub's database is a derived index that powers dashboards,
search, PR status, and the UI. Developers can work fully offline; the
aggregate view updates on push.

- Source: `work/design/centralized-state-server.md` §Problem and Motivation
  (lists "commit-bound visibility" and "limited real-time coordination" as
  separate pressures but then recommends a single unified response);
  `work/design/public-schema-interface.md` §2.1 (viewer reads from its
  own clone — inherits the visibility gap).
- Evidence grade: **primary / current**
- Confidence: **high**

### Finding 10: Hybrid patterns separate visibility from authority and cover the relevant trade-off space

Four hybrids worth distinguishing:

- **Hybrid A — Read-only aggregator service (pull-based).** A lightweight
  service fetches from `origin` (all branches, including worktree
  branches) on a schedule, parses `.kbz/state/` from each ref, and
  exposes an aggregate query API and dashboard. Canonical authority stays
  in Git. Precedents: GitHub UI, Sourcegraph, Terraform Cloud workspace
  view, Pulumi Cloud reading DIY backends.

- **Hybrid B — Push-based event mirror (non-authoritative).** In
  addition to writing canonical files locally, agents emit lightweight
  events to a service on every transition. The service stores events for
  dashboards and metrics. Events are derived and replayable from Git.
  Catches pre-push activity but requires disciplined reconciliation.

- **Hybrid C — Push-on-transition convention + remote-aware status.** No
  new service. Agents push worktree branches to `origin` at milestone
  boundaries (the same boundaries the JSONL design already identifies as
  commit points). A `status --remote` flag and a `dashboard` command
  fetch from `origin` and aggregate across all branches found there.
  Team-level visibility is only as stale as the latest push.

- **Hybrid D — Shared derived cache (write-through).** A shared version
  of the existing `internal/cache/`, written to by every agent. Most of
  the operational cost of a canonical DB with fewer of the benefits, and
  introduces cache-invalidation problems that (A) and (B) avoid by being
  read-only or explicitly reconciled. **Not recommended.**

Hybrid C is the proportional first step; Hybrid A is the proportional
next step if C is insufficient; Hybrid B is reserved for a specific
pre-push-visibility requirement; Hybrid D should be ruled out.

- Evidence grade: **primary (design intent) + secondary (precedent)**
- Confidence: **high** on the taxonomy; **medium** on the exact
  thresholds that would motivate moving from C to A to B.

## Trade-Off Matrix

Options evaluated against the two pressures and the main constraint criteria.

| Criterion | Status quo (commit-per-transition) | Git-native evolution (JSONL + coarse commits) | Hybrid C (push-on-milestone + `--remote`) | Hybrid A (read-only aggregator) | Hybrid B (event mirror) | Model C (dual backend) | Model B (DB canonical, Git retired) |
|---|---|---|---|---|---|---|---|
| Solves commit noise | ❌ | ✅ directly | ✅ (via coarser commits) | ✅ (via coarser commits) | ✅ (via coarser commits) | ✅ for Git mode | ✅ (as side effect) |
| Solves cross-clone visibility | ❌ | ❌ | ✅ at push granularity | ✅ at fetch granularity | ✅ near real-time | ✅ in DB mode | ✅ real-time |
| Visibility of unpushed work | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ in DB mode | N/A (no local authority) |
| Preserves Git-native identity | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠ partial | ❌ |
| Preserves viewer contract | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠ mode-dependent | ❌ requires redesign |
| Real-time multi-writer coordination | ❌ | ❌ | ❌ | ❌ | ⚠ partial | ✅ in DB mode | ✅ |
| Row-level concurrency control | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ in DB mode | ✅ |
| Query richness | ⚠ via existing SQLite cache | ⚠ via existing SQLite cache | ⚠ cache + remote refs | ✅ aggregate queries | ✅ | ✅ in DB mode | ✅ |
| Operational burden | None | None | None | Low (one read-only service) | Medium | High in DB mode | High |
| Migration cost | 0 | Low | Near 0 | Low | Medium | High | Very high |
| Reversibility | N/A | High (just commit differently) | Trivial | High (stop running it) | High | Low | Very low |
| JSONL merge-conflict risk | N/A | Medium (needs union-merge driver) | Medium | Medium | Medium | Medium for Git mode | N/A |
| Handles large distributed teams | ❌ | ⚠ only if team accepts Git flow | ⚠ push cadence dependent | ✅ | ✅ | ✅ | ✅ |

## Recommendations

Recommendations are ordered: immediate work (1–3), visibility work (4–5),
strategic positioning (6–7), and governance (8–10).

### Recommendation 1: Ship Git-native evolution as the immediate response to commit noise

- **Action:** Implement `work/design/transition-history-storage.md`
  Phases 1–3 (JSONL dual-write, switch history consumers to the log,
  milestone-based commit boundaries).
- **Confidence:** high
- **Based on:** Findings 1, 2, 5, 7
- **Conditions:** Applies if the near-term goal is to eliminate commit
  noise while preserving the current product identity. Does not apply if
  the near-term roadmap requires sub-push-latency real-time coordination
  across machines without worktree separation.

Proportional to the measured pain. No new infrastructure. Preserves the
viewer contract. Reversible at low cost.

### Recommendation 2: Add a custom JSONL merge driver as part of the evolution work

- **Action:** Ship a `.gitattributes` entry and a
  `kanbanzai merge-driver transitions` command that union-merges
  transition-log JSONL files, deduplicated by a per-event ULID, sorted by
  `at`. Register the driver in `kanbanzai init` so new projects inherit it.
- **Confidence:** high
- **Based on:** Finding 4
- **Conditions:** Applies alongside Recommendation 1. Per-entity sharding
  alone is not sufficient protection.

The Spec-Kitty P0 bug is an existence proof that this class of data loss
is not theoretical. Without a union-merge driver, the same failure is
reachable in Kanbanzai's worktree-vs-main concurrency pattern.

### Recommendation 3: Update the transition-history design to build on `internal/cache/` rather than proposing a new SQLite layer

- **Action:** Revise `work/design/transition-history-storage.md` §Migration
  strategy to replace "Phase 4: optional indexing" with an explicit
  extension of `internal/cache/` to cover transition events. Do not
  introduce a second SQLite file.
- **Confidence:** high
- **Based on:** Findings 3, 8
- **Conditions:** Applies whenever the design document is next updated.

Keeps the architectural story coherent and avoids duplicating
infrastructure that already exists.

### Recommendation 4: Adopt push-on-milestone + `--remote` aware status as the first visibility improvement (Hybrid C)

- **Action:** As part of the coarser-commit work in Recommendation 1, also
  enforce push-on-milestone in the default orchestration loop, add
  `status --remote` and a `dashboard` command that fetch from `origin`
  and aggregate state across all known refs (including worktree branches).
- **Confidence:** high
- **Based on:** Findings 5, 9, 10
- **Conditions:** Applies always. Should ship as part of the same release
  as Recommendation 1.

Delivers the majority of the team-visibility benefit at near-zero
operational cost. Also functions as **prerequisite evidence**: whether
teams still need a shared service becomes empirically observable rather
than speculative.

### Recommendation 5: If Hybrid C proves insufficient, build Hybrid A (read-only aggregator) as a separate product

- **Action:** Build a standalone read-only aggregator service that
  fetches from `origin` on a schedule (and on webhook when available),
  parses `.kbz/state/` across refs, and exposes an aggregate API plus a
  web dashboard. Canonical authority stays in Git. Framing it as a
  separate product keeps the MCP server Git-native.
- **Confidence:** medium-high
- **Based on:** Findings 5, 7, 9, 10
- **Conditions:** Applies only if Hybrid C has shipped and team feedback
  shows push-latency visibility is still a pain point. Do not begin this
  work before Hybrid C is in production — without the evidence, the
  aggregator's requirements cannot be scoped correctly.

This is the same relationship GitHub's web UI has to Git, and the same
relationship Pulumi Cloud has to DIY backends: a derived read-replica,
not a canonical authority.

### Recommendation 6: Invest in backend-neutral service contracts as optionality insurance

- **Action:** When refactoring the service layer for transition logs,
  ensure entity, document, knowledge, checkpoint, worktree, and
  transition services do not leak file-layout assumptions to their
  callers. No `filepath.Join(".kbz/state", ...)` at callsites; use a
  persistence interface.
- **Confidence:** medium-high
- **Based on:** `work/design/centralized-state-server.md` §Stage 1;
  Finding 5; Pulumi/Terraform precedent.
- **Conditions:** Applies if the team wants to preserve the option of a
  future centralised backend at minimal marginal cost. Pays off for
  Git-native evolution today regardless (improves testability, keeps
  the service layer honest, simplifies Hybrid A's implementation later).

This is the only part of the centralised-state proposal worth doing now.
It is worth doing anyway.

### Recommendation 7: Treat a fully centralised backend as a strategic option, not a planned milestone

- **Action:** Keep `work/design/centralized-state-server.md` as a
  reference design. Do not schedule implementation work against it. Do
  not commit the product to dual-mode support in public documentation.
- **Confidence:** high
- **Based on:** Findings 2, 5, 6, 7
- **Conditions:** Applies until a concrete product requirement appears
  that cannot be solved within Git-native mode plus Hybrids C and A —
  for example, a named team customer requiring real-time multi-writer
  coordination or pre-push visibility across an organisation.

### Recommendation 8: Reframe the centralised-state design around two decoupled sub-problems

- **Action:** When `work/design/centralized-state-server.md` is next
  updated, split its motivation into two sub-problems:
  (a) progress visibility across distributed clones — solvable by
  hybrid aggregation;
  (b) centralised canonical authority — required only for real-time
  multi-writer coordination or sub-push-latency team coordination.
  Recommend (a) as the proportional first response and keep (b) as a
  longer-term strategic option.
- **Confidence:** high
- **Based on:** Findings 6, 7, 9, 10
- **Conditions:** Applies whenever the design document is next updated.

This separates two decisions that have been conflated and makes each
independently evaluable. It also aligns the document with the
comparable-system pattern from Finding 5.

### Recommendation 9: If centralised mode is ever pursued, enforce one canonical backend per project

- **Action:** Any future dual-backend work must surface backend choice at
  project init, refuse writes to the non-canonical backend, and provide
  explicit migration tooling (as Beads does with
  `bd migrate --to-dolt` / `--to-sqlite`). Dual-write exists only as a
  transient migration state, never as steady-state architecture.
- **Confidence:** high
- **Based on:** Finding 5;
  `work/design/centralized-state-server.md` §Decisions.
- **Conditions:** Applies to any future centralised-mode implementation.

### Recommendation 10: Defer Hybrid B (event mirror) until a specific pre-push-visibility requirement appears

- **Action:** Do not pre-emptively design an event-mirror service. If a
  future requirement demands real-time visibility into unpushed work,
  design the event mirror as an explicit sidecar with documented
  reconciliation rules — never as canonical state.
- **Confidence:** high
- **Based on:** Findings 7, 9, 10
- **Conditions:** Applies until a named use case requires pre-push
  visibility (for example, a live "who is doing what right now" wall for
  a tightly coordinated team).

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| JSONL merge conflicts silently drop audit-trail events | Medium | High (data loss, dashboard regressions) | Recommendation 2 — ship union-merge driver with the initial evolution; test it explicitly in a two-worktree scenario |
| Coarser commit boundaries create large windows of unpushed state invisible to teammates | Medium | Medium (visibility regression while fixing commit noise) | Recommendation 4 — ship push-on-milestone alongside coarser commits, not after |
| Aggregator service (Hybrid A) drifts from Git due to stale fetches | Medium | Low (read-only; recovery is re-fetch) | Design the aggregator to be stateless over its DB — any row must be reconstructable from a ref in `origin` |
| Backend-neutral service contracts are deferred, locking in file-layout assumptions | Medium | Medium (future migration becomes hard) | Recommendation 6 — do this as part of the transition-log refactor, not after |
| Centralised-mode work starts speculatively and consumes roadmap without a named requirement | Low | High (sunk cost, identity drift) | Recommendation 7 — explicit governance: centralised work does not start without a documented trigger |
| Existing `internal/cache/` is over-extended and becomes a de-facto canonical authority | Low | High (architectural drift) | Keep `.kbz/cache/` gitignored; enforce that every cache row is rebuildable from YAML/JSONL; reject designs that write to the cache without also writing canonical state |

## Limitations

- Empirical measurements (commit counts, entity volumes) reflect this
  repository only. Other projects using Kanbanzai may show different
  distributions.
- `internal/cache/` effectiveness was inferred from code and tests; no
  load benchmark was performed.
- The claim that a JSONL union-merge driver is sufficient is based on
  Spec-Kitty's reported experience; it has not been tested in Kanbanzai's
  specific multi-worktree pattern.
- Hybrid C's effectiveness depends on how faithfully agents push on
  milestone boundaries. Without enforcement (orchestrator-level push
  hooks), the "visibility at push granularity" claim degrades.
- Hybrid A assumes teams are willing to push worktree branches to
  `origin` for visibility. Teams that prefer to publish only merged work
  will need Hybrid B or will accept reduced visibility.
- No formal analysis of what counts as a "milestone" for coarser commits.
  Candidates are listed in the transition-history design but should be
  finalised before implementation.
- This research is architectural. Product-strategy factors (named
  customer demand, competitive landscape, hosting revenue model) are out
  of scope and could shift the decision.

## Next Steps

If Recommendations 1–4 and 6 are accepted — the immediate path:

1. **Update `work/design/transition-history-storage.md`**
   - reference `internal/cache/` as the existing index, not a future phase
   - add an explicit section on the JSONL merge driver (Recommendation 2)
   - finalise the list of milestone commit boundaries
   - add a push-on-milestone requirement (Recommendation 4)
2. **Update `work/design/centralized-state-server.md`**
   - split motivation into visibility vs. authority (Recommendation 8)
   - cross-link to this research document
3. **Draft a persistence-interface ADR** (Recommendation 6) enumerating
   the boundary between services and storage and flagging callsites
   currently coupled to `.kbz/state/` paths.
4. **Decompose the transition-history feature** into implementation tasks:
   JSONL writer, service helpers, merge driver, milestone-boundary commit
   scheduling, push-on-milestone, `status --remote`, `dashboard` command,
   health-check rules, cache extension for transition events.
5. **Define the commit-noise measurement** so post-implementation effect
   is observable. Target: share of `workflow(...)` commits in the last
   30 days drops from ~67% to under ~25%.

If Recommendation 5 becomes relevant later (Hybrid A aggregator):

6. **Define the trigger:** concrete team feedback that push-granularity
   visibility (Hybrid C) is insufficient, with at least one named
   scenario that (A) would demonstrably improve.
7. **Scope the aggregator as a separate product**, not part of the MCP
   server. Treat it as a read-replica consuming the public schema.

If Recommendation 7 is ever reopened (centralised canonical authority):

8. **Define the trigger conditions in advance:** e.g., "when N teams of
   more than M concurrent users request shared authority," or "when the
   viewer needs sub-push-latency updates." Without a named trigger, this
   conversation tends to recur without new information.

## Related Documents

- `work/design/transition-history-storage.md` — Git-native evolution
  (primary design for Recommendations 1–3)
- `work/design/centralized-state-server.md` — centralised backend
  reference design (strategic option under Recommendation 7, to be
  reframed under Recommendation 8)
- `work/research/state-backend-comparison.md` — prior comparison; this
  report extends rather than supersedes it
- `work/design/public-schema-interface.md` — viewer contract that
  constrains canonical authority choices
- `work/design/git-commit-policy.md` — commit-history quality constraints
  motivating Recommendation 1
- `work/design/kanbanzai-1.0.md` — Git-native product framing
- `internal/cache/cache.go`, `internal/mcp/server.go` — existing derived
  SQLite cache referenced in Finding 3 and Recommendation 3
