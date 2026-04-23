# Doc-Intel Recurring Issues: Investigation

| Field       | Value                                                                            |
|-------------|----------------------------------------------------------------------------------|
| Date        | 2026-04-23                                                                       |
| Status      | Draft                                                                            |
| Type        | Research                                                                         |
| Input docs  | P27 Retrospective, Layer 3 Pilot, document-retrieval-for-ai-agents.md,           |
|             | skill-authoring-best-practices.md                                                |
| Output for  | Next round of doc-intel designs and feature planning                             |

---

## 1. Summary of Findings

**Q1 — Are these issues small and unconnected?** No. Three of the five most significant
gaps — the empty concept registry, shallow classify-on-register results, and zero
knowledge confirmations — share a single enabling condition: the system places
multi-step enrichment operations as advisory inline instructions at points in the
workflow where agent attention is already committed to a competing primary task.
Registration-time classification competes with the act of committing; concept tagging
competes with role/confidence assignment; knowledge confirmation competes with
task-completion mechanics and lacks sub-agent visibility. The remaining two significant
gaps — the pre-pilot corpus backlog (~200 unindexed documents) and the CLI identity
resolution defect — are genuinely independent. The two bugs uncovered by the pilot
(JSON struct tags, invalid role examples) were point defects, now fixed, that do not
explain the structural pattern. The correct treatment is therefore not five targeted
patches but recognition that three of the five failures are symptoms of one architectural
choice.

**Q2 — Is there a larger architectural issue?** Yes, and it can be named precisely:
the system is built on a **voluntary-step architecture** — critical enrichment operations
(concept tagging, classification, knowledge confirmation) are encoded as advisory
instructions in skill files rather than as enforced requirements in tool logic. This is
not a P27 oversight; the P27 design itself predicted (§11) that "skill changes and
template requirements reduce the probability of agents skipping steps; they do not
eliminate it." P27 nonetheless escalated to stronger advisory wording rather than to
tool-level enforcement. `skill-authoring-best-practices.md` §1.6 is unambiguous: "Where
a procedure must be followed exactly, encode it as a deterministic tool or script — not
as instructions the agent follows on its honour." Every compliance failure in this corpus
is a predictable consequence of treating hard-constraint operations as soft-constraint
guidance. The structural fix is not new instructions; it is a change in enforcement
level across the enrichment pipeline.

**Q3 — Would we be better served by separate plans?** The CLI identity resolution defect
is a one-line fix and belongs in any nearby plan as a single task. The corpus backlog is
an operational campaign with no structural complexity and can run independently once
tooling is stable. But the compliance failure cluster (concept tagging, classification
nudge, knowledge confirmation) and the tool information gap cluster (taxonomy in guide,
section count in pending, content_hash in register nudge, suggested classifications)
must be addressed in a single coherent plan. They are coupled at the classify-on-register
loop: fixing tool affordance without changing enforcement leaves enrichment steps
optional and compliance stays low; enforcing concept tagging without improving the
tool's ability to suggest concepts creates friction that agents route around. Splitting
these two clusters into separate plans leaves the coupling point — the
classify-on-register workflow — unaddressed by either plan, and the same compliance
pattern will reappear in P30.

**Q4 — Should we step back and do more research first?** Two questions cannot be
answered from the four source documents and would, if answered incorrectly, embed the
same class of error into the next sprint's designs. First: whether `concepts_intro`
should be agent-specified, server-suggested via heading analysis and entity references,
or automatically extracted — the pilot recommends server suggestions but does not
evaluate their quality, and this choice directly determines whether the solution is
"better affordance" or "automatic extraction." Second: at which point in the workflow
concept enforcement should trigger — at `doc_intel classify` call time, at `doc approve`
time, or at a dedicated enrichment gate — because the enforcement point determines both
coverage and the friction profile of backlog runs. A targeted experiment (classifying
10–15 high-value specifications with explicit `concepts_intro`, comparing agent-specified
vs. server-suggested concepts) would answer both questions and should precede writing
the plan's specification. These are genuine gaps; all other planning decisions can be
made from evidence already in hand.

---

## 2. Failure Inventory

| # | Failure | Source document | Type | Cluster |
|---|---------|-----------------|------|---------|
| F-01 | Concept registry empty — zero concept nodes, zero TAGGED_WITH edges despite 9,784 classified sections | P27 Retrospective §SC-2, §Concept registry | Architectural gap | C1: Voluntary-step compliance |
| F-02 | Pre-pilot backlog unindexed — ~200 documents from P3–P22 registered but invisible to all doc_intel queries | P27 Retrospective §SC-3 | Design gap | C5: Corpus coverage |
| F-03 | Shallow classify-on-register — P28 specs have exactly 5 sections each, same structural paths, no concept tagging | P27 Retrospective §SC-4, §Why is classification shallow | Architectural gap | C1: Voluntary-step compliance |
| F-04 | Knowledge entries unconfirmed — all 64 entries remain "contributed"; Phase 6 close-out never run | P27 Retrospective §SC-6, §Why are knowledge entries unconfirmed | Design gap | C1: Voluntary-step compliance |
| F-05 | Classification nudge deferred — text string in register response "seen and deferred" at registration time | P27 Retrospective §Why is classification shallow | Architectural gap | C1: Voluntary-step compliance |
| F-06 | `recent_use_count` signal imprecise — counts context-assembly list calls alongside intentional queries, cannot distinguish passive surfacing from active use | P27 Retrospective §SC-7, §Knowledge feedback loop | Implementation bug (minor) | C6: Observability gaps |
| F-07 | CLI `doc register` identity resolution broken — `created_by is required` despite auto-resolution documented; six attempts to register one document; flag naming inconsistency (`--by` vs `--created-by`) | P27 Retrospective §Observed Friction | Implementation bug | C4: CLI identity resolution |
| F-08 | Concept guidance absent from skill and guide — no skill instruction to populate `concepts_intro`; guide response provides no concept affordance or examples | P27 Retrospective §Why is concept registry empty | Design gap | C1: Voluntary-step compliance |
| F-09 | SC-5 conformance enforcement unmeasurable — only one P28 design produced; cannot evaluate rejection rate or determine whether check is running | P27 Retrospective §SC-5 | Design gap (measurement) | C6: Observability gaps |
| F-10 | JSON tags missing on `Classification` struct — `section_path` (snake_case) never decoded from JSON; all classification silently failed before fix | Layer 3 Pilot §3.1 | Implementation bug (**FIXED**) | C3: Fixed bugs |
| F-11 | Invalid role examples in `kanbanzai-documents` skill — `context`, `procedure` cited as valid roles; neither is in `FragmentRole` taxonomy | Layer 3 Pilot §3.2 | Implementation bug (**FIXED**) | C3: Fixed bugs |
| F-12 | Section count absent from `pending` response — agents cannot plan batch sizes without calling `guide` on every document individually | Layer 3 Pilot §5.1 | Design gap | C2: Tool information gap |
| F-13 | Role taxonomy absent from `guide` response — agents must know valid roles from memory or cold skill file; cold-context agents exposed to validation failures (partially addressed P28) | Layer 3 Pilot §5.2 | Design gap (partially fixed) | C2: Tool information gap |
| F-14 | `content_hash` and outline absent from register nudge — extra round-trip (`guide`) required for every classify-on-register operation | Layer 3 Pilot §5.4 | Design gap | C2: Tool information gap |
| F-15 | No suggested classifications in `guide` — agents derive all role assignments from scratch; ~60% of sections could be pre-assigned from heading analysis | Layer 3 Pilot §5.3 | Design gap | C2: Tool information gap |
| F-16 | MCP parameter structs not audited for JSON tag coverage — other structs sharing the yaml/JSON tag pattern may carry same deserialization vulnerability | Layer 3 Pilot §5.5 | Implementation bug (potential) | C3: Fixed bugs |

---

## 3. Cluster Analysis

### Cluster C1: Voluntary-Step Compliance Failures

**Members:** F-01, F-03, F-04, F-05, F-08

**Shared root / enabling condition:**
The system places enrichment operations as advisory inline steps at points in the
workflow where agent attention is already committed to a competing primary task, and
where performing the enrichment step requires effort beyond the minimum the tool
enforces. Registration-time classification competes with work-commit attention.
Concept tagging competes with role/confidence assignment within the same classify
call. Knowledge confirmation is placed in Phase 4 of implement-task where sub-agents
have no visibility into which entries they actually used. In every case, the step is
optional at the tool level and requires the agent to self-motivate additional work
while doing something else.

**Evidence for the connection:**
- *Retrospective §Why is the concept registry empty:* "No classification session
  instructed agents to populate `concepts_intro`. The skill files do not explicitly
  require concept tagging... With no prompt to populate `concepts_intro` and no
  example of what a well-populated concept entry looks like, agents defaulted to the
  minimum required fields."
- *Retrospective §Why is classification shallow:* "The `classification_nudge` in the
  register response is a text string. It is seen and deferred." And: "An agent
  classifying at registration time reads the document it just wrote — but
  classification from memory... produces section-count-limited results. At
  registration, the agent's attention is on committing and moving to the next task."
- *Retrospective §Why are knowledge entries unconfirmed:* "Sub-agents complete a task
  and stop. They do not have visibility of which knowledge entries from their context
  packet they actually used versus ignored."
- *P27 design prediction (§11, as cited in Retrospective):* "Skill changes and
  template requirements reduce the probability of agents skipping steps; they do not
  eliminate it." P27 acknowledged the mechanism but escalated only to stronger wording.

**Normative reference:**
`skill-authoring-best-practices.md` §1.6 (Enforceable Constraints Beat Advisory
Instructions): "Where a procedure must be followed exactly, encode it as a
deterministic tool or script — not as instructions the agent follows on its honour."
This is grounded in MetaGPT (Hong et al., ICLR 2024), which found structured artifacts
with verification gates reduced cascading errors ~40% vs. free dialogue, and in Masters
et al. (DAI 2025), which formalises the hard-constraint (ℋ) / soft-constraint (𝒮)
distinction: "Hard constraints should become tool-level enforcement (the tool refuses
invalid operations)." The system has consistently treated concept tagging and knowledge
confirmation as 𝒮 when the evidence shows ℋ is required for the compliance rate the
system's functionality depends on.

---

### Cluster C2: Tool Information Gap

**Members:** F-12, F-13, F-14, F-15

**Shared root / enabling condition:**
The tool calls in the classification workflow do not provide sufficient information to
complete the next step correctly without additional round-trips or externally-acquired
knowledge. Every deficiency forces the agent to either make an extra tool call or rely
on knowledge it may not have (role taxonomy, section sizes, content hash). The
classify-on-register loop currently requires three tool calls (`register` → `guide`
→ `classify`) where two would suffice if the register response included the information
the guide call retrieves.

**Evidence for the connection:**
- *Pilot §5.1:* "Agents planning batch sizes have no way to know which documents are
  large (100+ sections) versus small (5–10 sections) without calling `guide` on each
  one individually — which defeats the purpose of planning."
- *Pilot §5.2:* "Every classify attempt that failed on role names (`context`,
  `procedure`, float confidence values) did so because the agent lacked the taxonomy
  at call time. Even after fixing the skill file, agents working from a cold context
  window remain exposed to the same failure."
- *Pilot §5.4:* "The `content_hash` and outline are computed at registration time and
  are already known to the server when the nudge is generated. Requiring an extra
  `guide` call to retrieve them adds a round-trip for every registered document."
- *Pilot §5.3:* "For approximately 60% of sections encountered in this run, the heading
  alone was sufficient for high-confidence classification."

**Normative reference:**
`document-retrieval-for-ai-agents.md` §3c notes the progressive disclosure pattern as
a strength — the `guide` action is "an excellent entry point" that gives agents "just
enough context to decide what to read next." The gap is that guide does not yet give
agents enough context to *complete* the classify step without further calls. Pilot §5.2
articulates the target state: "An agent that has never read the skill file can correctly
classify any document using only the information in the `guide` response." That goal is
not yet met, and each deficiency in C2 is a separate point on the path to it.

**Coupling with C1:**
C2 compounds C1 without sharing its root. A tool that requires extra round-trips or
external knowledge to use correctly increases the friction of enrichment steps and
makes enforcement more disruptive. Fixing C2 alone does not close C1 — the
advisory-enforcement problem persists regardless of tool quality. But fixing C2 reduces
the cost of complying with C1 enforcement, making the stage-gate approach more feasible
and less friction-laden. The two clusters must be addressed in the same plan because
their coupling point — the classify-on-register loop — requires both tool affordance
and enforcement to be correct simultaneously.

---

### Cluster C3: Fixed Implementation Bugs

**Members:** F-10, F-11, F-16 (potential)

**Shared root:**
Individual code and documentation defects — not failures of system design — discovered
and fixed during the Layer 3 pilot. These are point defects in implementation and
skill-file content.

**Evidence:**
- *Pilot §3.1:* JSON struct tags missing on `Classification` → `section_path`
  (snake_case) never matched `SectionPath` (CamelCase) during `json.Unmarshal`. Fixed
  by adding `json:"..."` tags alongside existing `yaml:"..."` tags.
- *Pilot §3.2:* `kanbanzai-documents` skill cited `context` and `procedure` as valid
  roles; neither is in `FragmentRole`. Fixed by rewriting the section with a full
  role-reference table.
- *Pilot §5.5:* Other MCP parameter structs that have `yaml:` tags but are also decoded
  via `json.Unmarshal` may carry the same silent deserialization vulnerability. One-time
  audit recommended.

**No architectural significance.** These are point defects. F-10 and F-11 blocked the
first pilot run and contributed to agent confusion, but they do not explain the
structural compliance failures in C1. They are catalogued here for completeness and to
support the MCP struct audit recommendation (action item 6).

---

### Cluster C4: CLI Identity Resolution Bug

**Members:** F-07

**Shared root:**
The CLI `doc register` command does not invoke the same identity-resolution code path
as the MCP server. The MCP tool reads `.kbz/local.yaml` and falls back to
`git config user.name` via `config.ResolveIdentity("")`. The CLI does not call this
helper, causing `created_by is required` even when identity is resolvable from context.

**Evidence:**
- *Retrospective §Observed Friction:* Six attempts to register one document. The MCP
  tool succeeded on the first attempt; the CLI failed five times with `unknown flag` or
  `required` errors.
- *Retrospective §Recommendation:* "This is a one-line fix (call the existing
  `config.ResolveIdentity("")` helper that is already used by `worktree create` and
  other commands)."

**Independence:** This failure shares no root with any other cluster. Its relationship
to C1 is indirect: CLI registration friction is one plausible contributing reason agents
defer registration, which prevents the registration-time nudge from being seen at all.
However, F-05 (nudge deferred) is documented even for MCP-based registration, so the
CLI bug is a contributing friction rather than the structural root.

---

### Cluster C5: Corpus Coverage Gap

**Members:** F-02

**Shared root:**
No retroactive indexing campaign was conducted for documents registered before the
April 2026 pilot. Approximately 200 documents from P3–P22 have state records but no
doc_intel index files, making them invisible to all structural, role-based, and
concept-based queries.

**Evidence:**
- *Retrospective §SC-3:* "A large number of registered state records (approximately
  200, from P3–P22) have no corresponding index file at all. These documents were
  registered before the bulk classification pilot, never had `doc_intel guide` run
  against them, and are therefore invisible to all doc_intel queries despite being
  registered."

**Operational, not structural.** There is no structural barrier preventing these
documents from being indexed — the tools exist and the workflow is proven. This is
backlog clearance work. It is lower priority than C1 because the most-referenced prior
work (P15–P27) is already indexed, but a designer working on a problem first addressed
in P8 will find nothing in the corpus until this is addressed.

---

### Cluster C6: Observability Gaps

**Members:** F-06, F-09

**Shared root:**
Two instrumentation limitations that prevent precise measurement of adoption quality.
Neither is a functional failure.

**Evidence:**
- *Retrospective §SC-7:* "`recent_use_count` counts context-assembly list calls
  alongside explicit query calls. A knowledge entry that appears in a context packet
  but is never read by the agent is indistinguishable from one that is read and used."
- *Retrospective §SC-5:* "Whether the conformance reviewer explicitly ran the blocking
  check... cannot be determined from index data alone." Deferred to P29.

**Not actionable as a cluster.** F-06 has a one-line fix: also track `use_count`
(incremented only on explicit `knowledge get` calls), providing a complement to
`recent_use_count`. F-09 resolves naturally when more design documents are produced in
P29. Neither warrants its own plan.

---

## 4. Architectural Diagnosis

### The Voluntary-Step Architecture

The structural property that explains the persistence of failures across P27 and into
P28 is the system's consistent design choice to place critical enrichment operations as
advisory instructions rather than enforced stage gates.

This is not a feature that was planned and implemented incorrectly. It is an implicit
architectural decision — repeated across multiple workflow stages — that enrichment steps
should be placed where they logically belong in the workflow sequence and trust that
agents will perform them because the skill says to. The registration nudge is placed at
registration. The `concepts_intro` fields are part of the classify call. The confirmation
pass is placed in Phase 4 of implement-task and Phase 6 of orchestrate-development. In
each case, the decision was made to use the advisory mechanism, despite enforcement
mechanisms being available (tool-level checks, stage-gate guards, approval conditions).

The evidence is direct. The retrospective provides a root cause for each compliance
failure: no instruction given (concept tagging), attention committed elsewhere (shallow
classification), sub-agents lack visibility (knowledge confirmation). These are three
different proximate causes, but they share the same enabling condition: if the step were
enforced at the tool level, the proximate cause would be irrelevant. The step would
happen regardless of instruction completeness, attention competition, or visibility
limitations.

`skill-authoring-best-practices.md` §1.6 is unambiguous about this class of problem:
"Every source that compares 'telling agents what to do' with 'preventing them from doing
the wrong thing' finds the latter wins decisively." MetaGPT achieved ~40% error
reduction through verification gates — not better instructions. Masters et al. formalise
this as ℋ vs. 𝒮: hard constraints (tool-level enforcement) are appropriate for
"critical ordering" operations; soft constraints (advisory instructions) are appropriate
for operations where violation merely "incurs a penalty."

The P27 design knew this. Its own §11 prediction acknowledged that skill-based
enforcement would not eliminate step-skipping. The correct response to that prediction
was to design tool-level enforcement for critical enrichment steps. The actual response
was to mark it as a risk to "validate and tighten later" — effectively deferring the
architectural decision into a future sprint that has not yet addressed it.

**What this means for the proposed fixes:**

Every recommendation in the P27 retrospective's Priority 2 ("add concept guidance to
the classify skill") is an advisory escalation. The retrospective is aware of this
(Priority 4 explicitly calls for programmatic enforcement) but treats it as a separate,
lower-priority item. The evidence from C1 is that advisory escalation and programmatic
enforcement are not independently effective — the former without the latter has already
been tried (concept guidance was absent; adding it without enforcement will produce
marginal improvement at best). The latter without the former is friction (enforcement
without affordance produces forced compliance with poor quality output).

The structural fix therefore has two required components that must ship together:
1. **Affordance improvement** (C2 fixes): the tool provides enough information
   inline for agents to perform enrichment correctly without extra round-trips.
2. **Enforcement escalation** (C1 fix): concept tagging transitions from advisory
   to a hard constraint checked at a defined stage gate (recommend: `doc approve`
   time, requiring at least one section with `concepts_intro` populated for
   specification, design, and dev-plan document types).

Neither component alone addresses the coupling. This is not a claim that enforcement
is a deep architectural redesign — the retrospective's Priority 4 correctly notes that
the approval-time check is "a small server-side change that requires no new tool
actions." The claim is that this small change has outsized structural impact because it
moves the enrichment step from the 𝒮 class to the ℋ class, which is the single most
effective thing the system design literature recommends for operations that must be
performed reliably.

---

## 5. Planning Recommendation

### The CLI bug is independent — one task, not one plan

F-07 (CLI identity resolution) is a one-line fix. It should be assigned as a single
implementation task in the nearest available plan — it does not warrant its own plan
and should not delay or block any other work.

### The corpus backlog is independent — an operational campaign

F-02 (~200 pre-pilot documents unindexed) is straightforward batch work. It benefits
from C2 improvements (reliable taxonomy in guide, suggested classifications) but has
no structural dependency on C1 enforcement changes. It should be scheduled as an
operational campaign after the tooling improvements in the main plan are stable.
Including it in the main plan would add operational bulk to structural work without
improving either.

### The knowledge close-out is an immediate operational action

Running the Phase 6 P28 close-out is something the orchestrator should do now,
independent of any plan. The retrospective is correct that the fix is to run it, not
to change the skill — for the current P28 cycle. However, the structural question of
what prevents P29's close-out from being skipped the same way is a genuine open
question that the plan should address (see action item 8 below). The operational
close-out should not wait for the plan to start.

### Clusters C1 and C2 must be one plan

The correct scope for the next substantive plan is: tool information gap improvements
(C2) plus enforcement escalation (C1), addressed as a coherent whole. The plan name
would be something like "Classification Pipeline Hardening."

**What this plan includes:**
- Embed concept affordance in `guide` response (extending beyond the P28 role taxonomy
  to include `concepts_suggested` derived from entity references and heading analysis)
- Add `section_count` to `pending` response
- Include `content_hash` and outline in `classification_nudge` payload from `doc
  register`, reducing classify-on-register from 3 tool calls to 2
- Expand `guide` response to include `suggested_classifications` for
  heading-deterministic sections (~60% of sections per pilot data)
- Add approval-time stage gate: `doc approve` rejects specification, design, and
  dev-plan documents that have no classified sections with `concepts_intro` populated
- Audit all MCP parameter structs for JSON tag coverage (F-16)

**What this plan explicitly does not include:**
- P28 knowledge close-out (run immediately, before the plan starts)
- Pre-pilot backlog indexing (operational campaign, scheduled after tooling stabilises)
- Embeddings-based semantic search or the Option B standalone server
  (`document-retrieval-for-ai-agents.md` §5 recommendation: activate the existing
  system before building anything new — this plan is that activation)

**What goes wrong if the boundary is drawn differently:**

*Split into "affordance" plan + "enforcement" plan (C2 first, C1 later):*
The affordance plan ships improvements that lower friction but leave concept tagging
optional. Compliance improves marginally. The concept registry fills slowly if at all.
The enforcement plan, arriving later, adds a gate that agents are unprepared for because
the affordance improvements shipped without explicit enforcement context. The coupling
point — the classify-on-register loop requiring both correct tool output and enforcement
to be reliable — is addressed by neither plan in isolation.

*Include corpus backlog in this plan:*
Plan scope expands to include operational batch work with no structural complexity. The
backlog campaign is time-consuming and drags the plan timeline, delaying the structural
changes that have the most impact. The backlog work proceeds more reliably anyway after
the tool information gaps are fixed — it should come after, not during.

*Treat C1 as "just run the concept backfill" without adding enforcement:*
The retrospective recommends exactly this as Priority 1. It is necessary but not
sufficient. A targeted reclassification pass over 50 high-value documents (as
recommended) will produce a substantive concept registry for P29. But if `doc approve`
does not require concept tagging, P29 classifications will again be produced without
concepts, and the concept registry will degrade back toward empty within two sprints.
The backfill and the enforcement gate must both happen; the gate is what makes the
backfill durable.

---

## 6. Research Gaps

### RG-1: Quality of server-suggested concept extraction

**Question:** If the `guide` response returns `concepts_suggested` derived from entity
references and heading analysis, what is the signal quality of those suggestions? What
acceptance rate would agents achieve? What fraction of valuable concepts would be
missed entirely and require agent initiative to capture?

**Why it cannot be answered now:** The pilot report recommends this feature (§5.3)
based on the observation that heading-based role assignment works for ~60% of sections.
It does not evaluate whether entity-reference-based concept extraction produces useful
suggestions for the remaining work. The retrieval research (`document-retrieval-for-ai-agents.md`
§5, open question 3) explicitly lists concept synonymy handling as unresolved. The
retrospective describes the `concepts_intro` schema but provides no data on what
well-populated concept entries look like in practice (only one document in the entire
corpus contains `concepts_intro` data).

**Cost of a wrong assumption:** If server-suggested concepts are high-quality (70%+
acceptance rate), the right design is server-side extraction with agent review and
extension — the tool does most of the work. If suggestions are low-quality, the right
design is agent-specified with taxonomy and examples in the guide response, and server
suggestions become noise. A wrong assumption here produces a plan that ships the wrong
mechanism: either agents are burdened with overriding bad suggestions on every
classification call, or agents get no useful starting point and the enrichment step
remains high-friction even after the plan ships.

**Minimum evidence needed:** Classify 10–15 high-value specifications (P20–P27
approved specs) with explicit `concepts_intro` populated, using both agent-specified
concepts and a simulated server-suggestion pass (extract entity refs and heading
keywords, present as candidates). Compare coverage, acceptance rate, and concept
quality. This experiment requires one agent session and provides direct answers to both
the quality question and the design question.

### RG-2: Optimal enforcement point for concept tagging

**Question:** Should concept tagging be enforced at `doc_intel classify` call time
(reject calls with no `concepts_intro` for eligible document types), at `doc approve`
time (check whether the document's index has at least one concept-tagged section before
allowing approval), or at a new dedicated enrichment stage gate?

**Why it cannot be answered now:** The retrospective recommends `doc approve` time
enforcement (Priority 4), but this is stated as a recommendation without a comparative
analysis of the three alternatives. The pilot does not address enforcement mechanisms.
The skill-authoring research establishes that enforcement must happen at the tool level
but does not specify which tool or trigger point. The three options have meaningfully
different properties: classify-time enforcement catches every classification call but
creates friction for backlog runs (forcing concept population on every section of every
pre-pilot document); approve-time enforcement gates the final approval but allows
unenforced classification throughout development; a dedicated enrichment gate is
cleanest but adds lifecycle complexity.

**Cost of a wrong assumption:** Enforce at classify time: backlog runs require concept
tagging for every section of every pre-pilot document — this converts a batch
reclassification campaign into a full concept-tagging campaign, dramatically increasing
the effort of C5 corpus backlog work. Enforce only at approve time without careful
design: agents defer concept tagging to the last moment and produce minimal
token-filling entries under time pressure, yielding a concept registry that exists but
is low-quality. A new enrichment gate: adds lifecycle state complexity that may not
be justified if approve-time enforcement would achieve the same outcome.

**Minimum evidence needed:** Design review of the three alternatives against the
specific workflows they affect: single-document classify-on-register, backlog batch
runs, and post-approval enrichment passes. This can be done analytically from the
existing codebase without experimentation.

### RG-3: Knowledge confirmation structural recurrence

**Question:** What structural change, if any, would cause the orchestrate-development
Phase 6 close-out to occur reliably rather than requiring a manual action that can be
deferred indefinitely?

**Why it cannot be answered now:** The retrospective concludes that "the fix is to run
the close-out — not to change the skill." This is operationally correct for clearing
P28 debt. But it does not address what prevents P29's close-out from being deferred the
same way. The skill-authoring research's finding (§1.6) that advisory steps are skipped
reliably applies equally to Phase 6 as it does to concept tagging. The available
documents do not contain an analysis of how to make the close-out automatic, semi-
automatic, or gate-enforced. If this question is left unanswered, the plan will address
one cycle of knowledge confirmation debt without eliminating the recurrence pattern.

**Minimum evidence needed:** Examine what a plan-closure gate would look like: what
tool action closes a plan, and could a check be added to that action that verifies a
knowledge confirmation pass has been run (e.g., by checking whether any tier 2 entries
contributed during the plan period remain in `contributed` status)? This is an
architectural question about the plan lifecycle that can be assessed from the codebase
and workflow documentation without external research.

---

## 7. Recommended Next Actions

Items are ordered by priority. Items 1–2 are immediate and do not require a plan.
Items 3–7 are inputs to or components of the "Classification Pipeline Hardening" plan.
Item 8 is a structural follow-on.

1. **Run the P28 knowledge close-out immediately** (not a plan item). Call
   `knowledge(action: "list", status: "contributed", tier: 2)` and confirm, retire,
   or flag each of the 35 tier 2 entries. This is the first execution of the
   orchestrate-development Phase 6 close-out procedure. Running it once establishes the
   pattern, clears accumulated P28 debt, and begins converting the knowledge base from
   an append-only log to a confidence-weighted reference. Do not wait for a plan.

2. **Fix CLI `doc register` identity resolution** (single implementation task, one-line
   fix). Call `config.ResolveIdentity("")` in the `doc register` CLI handler, matching
   the MCP tool's behaviour. The `--by` flag remains available as an override. This
   eliminates the six-attempt registration friction documented in the retrospective.

3. **Run the concept-tagging experiment before writing the plan specification** (answers
   RG-1 and provides input for RG-2). Classify 10–15 approved specifications from
   P20–P27 with explicit `concepts_intro` populated. Compare agent-specified concepts
   vs. entity-reference-extracted candidates. Record acceptance rate, missed concepts,
   and time cost. Use results to decide whether the C2 fix is "add concept extraction
   to guide" or "add concept taxonomy and examples to guide." The plan specification
   should not be written without this data.

4. **Write and execute the "Classification Pipeline Hardening" plan** (addresses C1+C2
   together). Scope per §5 above: tool information gap improvements (C2) plus
   enforcement escalation at `doc approve` time (C1), in a single plan. The plan should
   produce a system where `doc approve` for specification, design, and dev-plan
   documents requires at least one classified section with `concepts_intro` populated,
   and where the classify-on-register workflow requires no more than 2 tool calls.

5. **Run the concept backfill sprint** (Priority 1 from retrospective, to be executed
   after enforcement gate ships). Classify the top 50 approved specifications and
   designs from P15–P27 with explicit `concepts_intro` populated. The enforcement gate
   from action 4 makes this backfill durable — without the gate, the registry degrades
   again within two sprints. Schedule this immediately after the plan in action 4 ships.

6. **Audit MCP parameter structs for missing JSON tags** (small task within the plan
   from action 4, or as standalone). Run the one-time audit from Pilot §5.5. Any struct
   decoded via `json.Unmarshal` from a tool parameter must have explicit `json:` tags
   on all exported fields. Add a Go test or `go vet` check to prevent regression.

7. **Schedule the pre-pilot corpus backlog campaign** (Cluster C5, after tooling
   stabilises). Once the guide response reliably provides taxonomy and suggested
   classifications (from action 4), run batch `guide` + classify for P3–P22 documents.
   Prioritise approved specifications and designs over reports and dev-plans. This
   extends the queryable corpus back to project inception.

8. **Design knowledge close-out enforcement as a plan-lifecycle gate** (addresses
   structural recurrence of F-04). After the P28 close-out is run manually, analyse
   whether a check can be added to the plan-closure tool action that verifies knowledge
   entries contributed during the plan period are not all in `contributed` status. This
   is the structural analogue of the concept tagging enforcement gate — it moves the
   close-out from a Phase 6 advisory step to a hard constraint on plan closure. Design
   this as part of the follow-on plan, not as an afterthought.