# Report: Evaluation of Writ as an Approach to Agent Rule-Following

**Plan:** P44 — Model routing and agent launcher
**Type:** Report
**Status:** Draft
**Date:** 2026-05-12
**Subject:** [infinri/Writ](https://github.com/infinri/Writ) — "Claude Code harness for AI coding agents"
**Compared against:** `P44-design-deterministic-workflow-controller.md`,
  `P44-research-hardening-principle.md`,
  `P44-report-hardening-principle-pipeline-architecture.md`

---

## TL;DR

Writ is a thoughtful project with one architectural insight that **independently
corroborates the central decision in the new P44 design** (tool affordance is
the only reliable lever for rule enforcement — prose rules get ignored). It is
*not*, however, a system Kanbanzai should adopt:

1. Writ's "process keeper" only works inside Claude Code, because it relies on
   Claude Code's `PreToolUse` hooks. Kanbanzai is client-agnostic by design.
   P44's controller approach achieves the same enforcement guarantee at the
   MCP server boundary — which is *stricter* (the forbidden tool isn't even
   advertised to the model) and works for every MCP client.
2. Writ's "librarian" (hybrid-RAG over a rule corpus) solves a problem
   Kanbanzai doesn't have. Kanbanzai's role + skill + stage-binding system
   targets context by stage, not by RAG retrieval over a flat rulebook. The
   advertised "726× token reduction" is computed against a strawman baseline
   that nobody actually runs.
3. Writ's quality numbers (Hit rate 0.76; MRR@5 0.49 on ambiguous queries) are
   *mediocre for an authoritative system* — roughly one in four queries fails
   to surface the relevant rule at all. Kanbanzai's hardening direction
   (P44 research) is the opposite: replace fuzzy LLM/RAG steps with
   deterministic ones, not add another retrieval pass.
4. Operational cost is high: mandatory Neo4j Docker container, Python 3.11+
   service, Tantivy + hnswlib + sentence-transformers + ONNX runtime. This is
   a serious step backwards from Kanbanzai's "single Go binary, install per
   repo" footprint.
5. Two narrow ideas are worth borrowing as small techniques (not a system
   adoption): the **approval-token-via-typed-phrase** pattern, and an
   "always-on/mandatory" tier of context that bypasses ranking. Both can be
   absorbed inside the existing P44 design without a new plan.

**Recommendation: do not create a new plan; do not adopt Writ. Continue with
P44 as designed.** Note the two borrow-able techniques in P44's open
questions for consideration during specification.

---

## 1. What Writ actually is

Writ ships two layers that share a Neo4j-backed knowledge graph.

### 1.1 The librarian (the knowledge layer)

A FastAPI service on `localhost:8765` running a five-stage hybrid-RAG
pipeline against a corpus of 276 rules:

1. Domain filter (<1 ms).
2. BM25 keyword search via Tantivy (top 50, <2 ms).
3. ANN vector search via hnswlib over ONNX-served embeddings (top 10, <3 ms).
4. Graph traversal in Neo4j over `DEPENDS_ON`, `SUPPLEMENTS`, `CONFLICTS_WITH`
   edges (<3 ms).
5. Two-pass ranker fusing reciprocal-rank with severity, confidence, and
   graph-proximity weights, against a token budget (<1 ms).

Total p95 budget: 10 ms; measured 0.59 ms p95 on the live corpus.

Mandatory rules (`mandatory: true`, 30 of 276) are **excluded from the
retrieval pipeline at index build time** and loaded out-of-band by hooks
with their own 5,000-token budget cap. This is the architectural invariant
that no ranking change can drop a critical rule.

### 1.2 The process keeper (the enforcement layer)

30 hook scripts under `.claude/hooks/`, wired into Claude Code's hook system
via `templates/settings.json`, plus a session state machine
(`bin/lib/writ-session.py`), slash commands, and 6 sub-agent role files.

The notable mechanism: when Claude tries to write code in **Work mode**
without an approved `plan.md` and test skeletons, the `PreToolUse` hook for
`Write`/`Edit`/`Bash` denies the call with a structured reason. Approval is
gated by a one-time token written to `/tmp/writ-gate-token-${SESSION_ID}`
that only materialises when the *user* types an approval phrase consumed by
the `/writ-approve` slash command. Self-approval attempts by the agent are
denied and logged as `agent_self_approval_blocked`.

### 1.3 Mode/gate state machine

Four modes (Conversation / Debug / Review / Work) with two gates in Work
mode (`phase-a` requires `plan.md` with four named sections and validated
rule citations; `test-skeletons` requires at least one test file with real
assertions before production code is allowed).

---

## 2. Are the claims realistic?

| Claim | Verdict | Notes |
|---|---|---|
| 0.59 ms p95 retrieval on 276-rule corpus | **Plausible** | Tantivy + hnswlib in-memory genuinely run at that latency. ONNX inference on cached embeddings is sub-ms. Neo4j hop traversal at this scale is also realistic. |
| 0.557 ms p95 at 10,000-rule synthetic scale | **Plausible** | Index data structures are sub-linear; the headroom is consistent with the architecture. |
| 726× token reduction at 10k rules | **Technically true, rhetorically misleading** | The denominator is "stuff every rule into every prompt" — a baseline no working system uses. Against a stage-targeted baseline (which is what Kanbanzai already does), the reduction would be far smaller and probably negative once retrieval overhead is counted. |
| Cold start 1.72 s | **Plausible** | This is a separate process the user has to start; it's fine for a long-running daemon, less fine if it crashes. |
| Hit rate 0.76 across 165 ground-truth queries | **Mediocre** | One in four queries does not return the correct rule at all. The README notes that the hit-rate floor was *retuned downward* during corpus expansion — a candid disclosure that scaling worsened quality. |
| MRR@5 of 0.4886 on ambiguous queries | **Mediocre** | When the right rule is in the top-5 it sits around position 2 on average. For an "authoritative rules" system this is a worrying signal. |
| Methodology MRR 0.86 / hit rate 1.0 | **Strong, but separate corpus** | Calculated against a curated, signed-off corpus (40 queries) — i.e. the cases where the rules and queries were tuned together. Not a generalisation guarantee. |
| Tool affordance enforces what prose cannot | **Correct, well-argued** | This matches Anthropic's published guidance and matches P44's Decision 1 exactly. |
| Self-approval blocked by token-via-typed-phrase | **Genuinely clever** | Concrete mechanism for "approval cannot be self-served." Worth noting. |

The claims that hold up are the latency claims and the architectural one
about tool affordance. The token-reduction headline number is marketing
framing. The retrieval-quality numbers are honest but not impressive.

---

## 3. Is the Neo4j dependency mandatory?

**Yes.** The `docker-compose.yml` in Writ runs Neo4j 5 as the single
required service. The `writ` CLI's `import-markdown`, `export`, `add`,
`edit`, `validate`, `compress`, `propose`, and `review` commands all read
and write the graph directly. The `/conflicts`, `/always-on`,
`/subagent-role/{name}` endpoints all traverse Neo4j. Removing it would
require rebuilding the storage layer and the ranking pipeline.

For Kanbanzai this is a non-starter. Kanbanzai is a single statically-linked
Go binary, installed per-repo via `kanbanzai init`, with state in `.kbz/`
(YAML + JSONL + SQLite indexes). Adding a Neo4j Docker container plus a
Python service plus an ONNX runtime as a hard dependency would:

- Break per-repo isolation (one Neo4j volume across repos, or N volumes).
- Break the "open Zed/Claude Code/Cursor and it just works" story.
- Multiply the install footprint by ~50× (Neo4j alone is ~600MB).
- Introduce a network-port collision surface (7687, 7474, 8765).
- Move state out of `.kbz/` and out of git's reach.

The user's framing was correct: speed is not Kanbanzai's bottleneck;
accuracy and token cost are. Neo4j buys speed. The accuracy story (76% hit
rate) is *worse* than what Kanbanzai already achieves with stage-targeted
deterministic context assembly.

---

## 4. Comparison against the new P44 design

The two systems are addressing the **same root problem** — "agents won't
follow prose rules" — and they converge on the **same architectural
insight** but apply it in **different scopes** with **different
enforcement loci**.

### 4.1 Where they agree (the architectural insight)

Both systems independently arrive at this conclusion:

> Tool affordance enforces what prose cannot. If you don't want the agent to
> do X, don't give it the tool for X.

- **Writ**: blocks `Write`/`Edit`/`Bash` via `PreToolUse` hooks until gates
  pass.
- **P44**: removes `spawn_agent` from the orchestrator's tool list entirely
  and replaces it with `dispatch_task`, which runs the assembly gate, attaches
  context, and writes an audit row (Decision 1, §5.1).

This is corroborating evidence — same conclusion, different team, different
codebase, same reasoning. It strengthens the case for P44 Phase 1.

### 4.2 Where they diverge (the enforcement locus)

| Dimension | Writ | P44 |
|---|---|---|
| Enforcement boundary | Claude Code hook (`PreToolUse`) | MCP server tool registration |
| Strictness | Tool is *callable* but blocked at runtime | Tool is *not advertised* to the model |
| Client portability | Claude Code only | Any MCP client |
| Bypass surface | Hook can be disabled, settings.json edited, raw bash tried | None — model never sees the tool |
| Failure mode | Hook crashes → silent unenforced state | Tool absence is structural |
| Audit | Hook logs to filesystem | `internal/audit` JSONL with correlation IDs |

P44's enforcement is **strictly stronger**. A blocked tool call still
consumes the model's planning context and a tool turn. A non-existent tool
cannot be reasoned about. The difference matters most under exactly the
failure mode that motivated P44: the four-plan record (P50 → P58) shows
that *every* prose rule against composing prompts manually has been
bypassed, including the rule that explicitly forbade the bypass. Writ's
hook-based enforcement is a stronger version of "policy" — it is not the
"affordance removal" P44 is reaching for.

### 4.3 Where they don't address the same thing

Writ has **no equivalent** of:

- Stage controllers (Writ's session state machine is per-chat, not
  per-feature, and doesn't survive a Claude Code restart).
- A two-layered Definition of Done (deterministic Go checks + LLM
  adjudication). Writ's gates are essentially the deterministic layer for
  *plan and tests exist*, but there is no analog of "git ancestry +
  tests pass + doc records exist + worktree clean" verification.
- A durable, replayable workflow substrate.
- Per-entity feature-flagged migration.
- Multi-feature parallelism with conflict checking.
- Audit logging beyond per-hook trace files.
- A policy engine consolidating tool permissions / transition prerequisites
  / dispatch rules.

These are exactly the components P44 §4.2 inventories as the new packages.
Writ does not compete with this layer — it operates entirely below it,
inside a single Claude Code session.

P44 has **no equivalent** of:

- Hybrid-RAG retrieval over a rule corpus.
- An always-on "mandatory rules" tier with its own token budget cap.
- An interactive `add`/`edit` rule wizard with redundancy checking.
- An AI-rule-proposal pipeline with a 5-check structural gate.

Of these, only the "mandatory rules tier" idea has any traction for
Kanbanzai; the rest presuppose that "rules in a corpus" is the unit of
configuration, which Kanbanzai has explicitly rejected in favour of
"role + skill + stage binding + invariant" being purpose-built artefacts
loaded by the 3.0 context-assembly pipeline.

### 4.4 Where Writ goes against the P44 hardening direction

The `P44-research-hardening-principle.md` argument is that Kanbanzai should
move *fuzzy LLM steps* (validators, classifiers, retrieval ranking) toward
*deterministic code*. Writ's librarian moves the other way: it injects an
additional fuzzy step (hybrid-RAG retrieval with mediocre hit rate) into
every turn. This is the opposite of hardening.

Under the P44 model, the rule loaded for a task is determined by
`role + skill + stage + entity-spec-references` — a deterministic lookup
that returns the same set every time. Under Writ, the rule loaded is the
output of a probabilistic ranker. For a system where, per the user, "speed
is not the first issue, accuracy and token use are," this is an
anti-feature.

---

## 5. Are there ideas worth stealing?

Two narrow techniques, neither requiring a new plan.

### 5.1 The approval-token-via-typed-phrase pattern

Writ's `/writ-approve` slash command requires a one-time token at
`/tmp/writ-gate-token-${SESSION_ID}` that only materialises when the *user*
types an approval phrase. The agent cannot self-approve by calling the
slash command via raw bash; the token is missing.

This is a concrete, simple mechanism for "human attestation cannot be
self-served," which is exactly the failure mode P56 (bug-lifecycle gates)
and P44 Decision 3 (verifier) are trying to address structurally.

**Proposal:** note in P44 Open Question 4 (per-role tool-permission
enforcement) that for transitions requiring human attestation
(e.g. `reviewing → merging` after human review, `verifying → done`), the
controller could require a session-bound token written by a side channel
(e.g. a CLI command the human runs, or a click in a web dashboard) before
the policy engine permits the transition. This is consistent with P44's
two-layered verifier philosophy: the deterministic layer can include "did
the human attest?" as a check.

This is a small specification-time decision, not a new plan.

### 5.2 The "always-on / mandatory" tier with its own token budget

Writ separates 30 mandatory rules from the rest of its corpus and loads
them out-of-band, ensuring no ranking change can drop them. Kanbanzai
already has an analogous distinction (INV-* invariants in role files +
stage-binding prerequisites), but it is not formally separated from the
rest of the assembled context, and therefore has no token budget guarantee.

**Proposal:** consider, during Phase 1 assembly-gate specification, whether
the prompt assembly pipeline should emit two clearly-separated context
blocks — `MANDATORY` (invariants, the relevant role's hard constraints,
the stage's gate prerequisites) and `ASSEMBLED` (everything else) — with
the mandatory block always loaded regardless of token pressure. This is
already implicit in the role/skill design; making it explicit is one
section of code in `internal/context/pipeline.go`.

This too is a specification-time refinement, not a new plan.

---

## 6. Concrete recommendations

1. **Do not adopt Writ.** Its enforcement layer is client-coupled to
   Claude Code; Kanbanzai is client-agnostic. Its retrieval layer solves
   a problem Kanbanzai doesn't have, with quality numbers that are below
   what deterministic stage-targeted assembly already achieves. Its
   operational footprint (Neo4j, Python service, ONNX, Docker) breaks
   Kanbanzai's per-repo single-binary install model.

2. **Do not pause or revise P44.** Writ provides independent corroboration
   of P44's load-bearing decision (remove `spawn_agent` from the
   orchestrator's tool list). It does not provide an alternative to the
   stage controllers, the two-layered verifier, the policy engine, the
   audit log, or the eval harness — which are what actually solves the
   four-plan failure pattern.

3. **Do not create a new plan.** Nothing in Writ justifies a new strategic
   plan ahead of P44. The two ideas worth borrowing (5.1 and 5.2) are
   small enough to absorb into P44's specification stage.

4. **Note the two borrow-able techniques in P44's open questions** so they
   are considered during specification:
   - Add an open question about session-bound human-attestation tokens
     for transitions requiring human approval (extends Open Question 4).
   - Add an open question about explicit `MANDATORY` vs `ASSEMBLED`
     context blocks with separate token budgets in the assembly pipeline
     (refines the Phase 1 assembly-gate specification).

5. **Capture the corroboration in the design.** Add a one-paragraph note
   to `P44-design-deterministic-workflow-controller.md` §6 Decision 1
   pointing out that Writ (an independent project addressing the same
   failure pattern) reaches the same conclusion via a different mechanism
   — strengthening the empirical case that tool affordance is the only
   load-bearing lever.

6. **Consider Writ's quality numbers as a cautionary data point** for
   *any* future proposal to add an LLM-ranked retrieval step into
   Kanbanzai's context assembly. A 76% hit rate on a tuned corpus is not
   an upgrade over a deterministic lookup that returns the same set every
   time. The hardening principle wins on this one.

---

## 7. Why not a "wait and watch" stance?

Writ is at v1.0.1 with 24 stars, three contributors, and a tightly
Claude-Code-coupled architecture. Even if the project succeeds in its own
niche (single-developer Claude Code users with a large internal rulebook),
its evolution will continue down a path that diverges from Kanbanzai's
client-agnostic, multi-agent, parallel-feature, MCP-server-native model.

The architectural insight Writ corroborates is one P44 already encodes.
The implementation choices Writ makes (hooks over MCP, RAG over targeted
assembly, Neo4j over embedded stores, Python over Go, Claude Code only
over MCP-portable) are systematically incompatible with Kanbanzai's
direction. There is no convergent path where Writ becomes adoptable later;
it would require Kanbanzai to abandon its install model and its client
portability.

---

## 8. Summary

Writ is a serious project with one architectural insight worth
underlining and two small techniques worth considering. It is not an
alternative to P44, not a complement to P44, and not a candidate for
adoption ahead of P44. P44 should proceed as designed, with the two
borrow-able ideas noted in its open questions for handling at
specification time.

## Related documents

- `P44-design-deterministic-workflow-controller.md`
- `P44-research-orchestrator-architecture.md`
- `P44-research-hardening-principle.md`
- `P44-report-hardening-principle-pipeline-architecture.md`
- `P44-report-hardening-principle-independent-opinion.md`
- External: <https://github.com/infinri/Writ>
