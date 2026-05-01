# Design Brainstorm: A Human-Facing Planning & Admin Layer for Kanbanzai

| Field  | Value                          |
|--------|--------------------------------|
| Date   | 2026-05-01                     |
| Status | Draft — Brainstorming          |
| Author | architect                      |

## Problem and Motivation

Kanbanzai is an agent-first system. It is excellent at managing AI agents
through structured workflows — decomposition, dispatch, review, merge. But
the path from "we should build something" to "here are your entities" is
entirely conversational and entirely human-led. There are no tools for the
cognitive work that precedes entity creation: capturing raw ideas before
they're shaped, surfacing cross-plan dependencies, synthesising portfolio-
level status, triaging what needs attention across multiple active workstreams.

A previous research report (see Related Work) identified five missing
cognitive functions — idea capture, scope negotiation, dependency surfacing,
prioritisation, and progress synthesis — and mapped them to pre-computer
clerical roles. This brainstorm builds on that foundation, starting from
the human's daily experience rather than from tool gaps.

The core question: **If every human on a small product/design team had a
dedicated, experienced admin assistant, what would that assistant do?**
Answering this question surfaces genuine needs. Building a layer that
addresses those needs frees humans for their core creative work: thinking
about new products and designing them.

### Who is affected?

- **Product managers** who currently hold the entire plan portfolio in their
  head, manually tracking which workstreams are blocked, which decisions are
  pending, and which ideas haven't been shaped yet.
- **Designers** who switch between deep creative flow and administrative
  overhead — finding the latest version of a spec, remembering what changed
  while they were heads-down, checking whether their assumptions still hold.
- **Tech leads** who need to understand the full dependency landscape before
  committing to architecture directions, and who currently discover cross-
  feature conflicts during implementation rather than during planning.

### What happens if we don't address this?

The cognitive ceiling on how many parallel workstreams a human can track
becomes the ceiling on how much the AI agent team can produce. The agents
can parallelise arbitrarily; the humans cannot. The planning/admin layer is
the bottleneck, and the bottleneck is currently unaddressed.

## Related Work

### What already exists in Kanbanzai

**`kanbanzai-planning` skill** (`.agents/skills/kanbanzai-planning/SKILL.md`):
A conversational planning guide. The agent asks clarifying questions, reflects
scope back, flags sizing issues. Produces an informal scope agreement. Entity
creation is a separate, explicit step — the conversation and the creation are
disconnected.

**Plan/Batch entity model** (from P38/B38): Plans (`P{n}`) are strategic
decomposition entities. Batches (`B{n}`) are execution containers that group
features. Features belong to batches; batches optionally belong to plans.
This is the structured entity system the intake layer would feed into.

**`entity(action: create)`**: The only current path from idea to entity.
Form-like — fill in fields, submit. No scoping wizard, no guided decomposition,
no cross-plan awareness.

**`status()` dashboards**: Per-entity synthesis that surfaces progress,
blocked items, and attention items. Powerful within a single entity but has
no portfolio-level (cross-plan, cross-batch) synthesis capability.

**Stage bindings and role system** (`.kbz/stage-bindings.yaml`, `.kbz/roles/`):
Maps workflow stages to roles, skills, and prerequisites. This is the agent-
facing execution machinery. An intake layer would sit upstream of the first
stage in this pipeline (currently Planning → Design).

**`next()` work queue**: Priority-ordered ready tasks. Agent-facing, not
human-facing. Shows what's ready to implement, not what needs attention from
a human.

### Prior research

**`work/research/planning-admin-layer-exploration.md`** (2026-05-01):
Identified five missing cognitive functions (idea capture, scope negotiation,
dependency surfacing, prioritisation, progress synthesis), mapped them to
pre-computer clerical roles (correspondence clerk, committee secretary,
schedule clerk, filing clerk), and recommended a tools-first approach with
the role question deferred. This brainstorm builds on those findings and
adds the daily-burden and empathy dimensions that the research report
identified as a limitation ("no user research").

### External inspiration

- **Linear's command palette** (`Cmd+K`): Keyboard-first, speed-optimised
  quick-add. The intake experience is form-like but fast. Missing: AI agent
  awareness, conversational scoping.
- **Notion's freeform-to-structured surface**: Pages that can contain
  databases, checklists, and structured blocks. Closest analogue to
  unstructured-to-structured intake. Missing: workflow orchestration,
  agent dispatch.
- **Trello Butler's rule-based automation**: Triggers + actions on cards.
  Good conceptual match for clerical automation (triage, status propagation,
  reminders). Missing: plan generation from unstructured intent.
- **The pre-computer clerical office** (Yates, Gardey): Correspondence clerks,
  committee secretaries, schedule clerks, and filing clerks performed the
  exact cognitive functions Kanbanzai's intake layer needs. They structured
  and surfaced information without making decisions — the decision-makers
  decided. This is the organising metaphor for the layer.

## Daily-Burden Inventory

This inventory is grounded in realistic small-team dynamics: a product
manager, one or two designers, and a tech lead managing a large number of
AI agents through Kanbanzai. Each row traces a specific busy-work activity
to its creative cost.

| # | Burden | Who feels it | Frequency | Creative cost |
|---|--------|-------------|-----------|---------------|
| 1 | **Monday-morning reassembly**: Opening 15+ notifications across threads, PRs, and agent outputs to reconstruct what happened while you were away. Which agents finished? Which are blocked? Which reviews need your eyes? | PM, Tech lead | Weekly (Monday) or after any 2+ day absence | 30–45 min of context recovery before any creative thinking can begin. The first hour of Monday is lost to reassembly. |
| 2 | **"Where are we on X?"** Someone asks for a status update on a plan you last touched three weeks ago. You have to open entities, read agent summaries, check document status, and synthesise an answer from scattered pieces. | PM | 2–3× per week | 10–15 min per query. Breaks creative flow mid-session. |
| 3 | **Idea capture gap**: You have an idea for a feature while in a design session, on a walk, or in a meeting. You jot it in a notes app. It sits there. When you later shape it into a batch, you've lost the original framing and have to reconstruct why it mattered. | PM, Designer | 2–5× per week | Lost ideas (the ones you don't capture at all) plus degraded framing for captured ones. The cost is in *what never gets built* more than in time spent. |
| 4 | **Dependency blindness**: You approve a batch to proceed, only to discover during implementation that it depends on a feature in a different batch that hasn't been designed yet. The implementing agent flags it; you have to pause, re-scope, and re-plan. | PM, Tech lead | 1–2× per month | Hours of rework plus cascading delays. The implementing agents sit idle while you resolve the dependency. |
| 5 | **Backlog grooming as archaeology**: You have 20+ ideas captured as informal notes, conversations, and half-shaped plans. Deciding what to work on next requires reading all of them to remember what they are. | PM | Weekly or biweekly | 1–2 hours of reading old notes before you can even begin prioritising. The backlog itself becomes a source of cognitive load rather than a tool. |
| 6 | **Document version hunting**: You need the latest approved version of a design document to inform a new spec. Is it approved? Was it superseded? Where's the latest spec — is it in the feature directory or the plan directory? | Designer, PM | 3–5× per week | 5–10 min per hunt. Cumulative context-switching cost is significant — each hunt pulls you out of creative flow. |
| 7 | **Review-triage decision fatigue**: Agent reviews produce findings. Some are critical, some are cosmetic. You have to read every finding to decide which ones need your attention and which can be deferred. The system doesn't distinguish signal from noise for human consumers. | PM, Tech lead | 2–4× per week during active batches | 15–30 min per review batch. The mental cost of switching from creative work to evaluative work is as high as the time cost. |
| 8 | **Checklist verification**: Before advancing a feature through a stage gate, you manually verify: is the design approved? Is the spec approved? Are all tasks done? The system knows these answers — but you have to go find them. | PM | 3–5× per week | 5 min per verification. Small individually; cumulatively a persistent low-grade tax on attention. |
| 9 | **Stakeholder reminder-chasing**: You're waiting on a decision from someone. You have to remember to follow up. If you forget, the workstream stalls silently. | PM | 1–3× per week | Stalled workstreams plus the mental burden of tracking outstanding requests in your head. |
| 10 | **"What decisions did we make about Y?"** A design conversation from two months ago produced a decision that's now relevant. Was it written down? Where? You search, ask agents, read old design docs, and hope you find it. | Designer, Tech lead | 1–2× per week | 10–20 min per search. When you don't find it, you either re-decide (wasteful) or proceed with uncertainty (risky). |

### Themes that emerge from the inventory

- **Reassembly dominates.** Burdens 1, 2, and 5 are all variants of "I need to reconstruct state that the system already knows." The system has the data; the human doesn't have a view of it.
- **The gap between capture and structure is where ideas die.** Burden 3 describes a pipeline break — raw ideas have nowhere to live in Kanbanzai, so they live outside it, and the transfer cost means many never make it.
- **Dependency discovery is reactive, not proactive.** Burden 4 is discovered by agents during implementation — the worst possible time. The system could surface it during planning.
- **The system knows things it doesn't tell humans.** Burdens 6, 7, and 8 all describe information the system has (document status, review verdict severity, gate prerequisites) that requires human effort to extract. The data exists; the surfacing doesn't.

## Experienced-Helper Empathy Exercise

If each team member had a dedicated, experienced human assistant — someone
who knows the project, knows the workflow, knows what matters — what would
that assistant do? The table below maps each burden to a specific helper
behaviour, distinguishing proactive help (the assistant notices and does it)
from reactive help (the human explicitly delegates).

| Burden | Helper behaviour | P or R? | Who it serves |
|--------|-----------------|---------|---------------|
| 1. Monday-morning reassembly | Prepares a morning briefing: "Here's what changed since Friday — 3 agents completed tasks, 1 is blocked waiting on spec clarification, 2 reviews need your eyes. The most urgent is the blocked agent because it's holding up B12." | **Proactive** — delivered before the human asks, first thing in the morning | PM, Tech lead |
| 2. "Where are we on X?" | Maintains a live portfolio summary: every plan, batch, and feature with current status, blockers, and last meaningful change. Answers "where are we on X?" in seconds rather than minutes. | **Reactive** — human asks, assistant answers | PM |
| 3. Idea capture gap | Provides a frictionless capture point: "Tell me the idea, I'll file it with context, tag it, and surface it when you're ready to shape it." The assistant files it, preserves the original framing, and reminds the human it exists during backlog grooming. | **Both** — capture is reactive (human offers idea); surfacing is proactive (assistant surfaces during grooming) | PM, Designer |
| 4. Dependency blindness | Scans new scope proposals against existing work: "This feature depends on the event system in B8, which is still in design. Flagging now so you can decide whether to wait or parallelise with an interface contract." | **Proactive** — assistant checks during scoping without being asked | PM, Tech lead |
| 5. Backlog grooming as archaeology | Organises the backlog: categorises, tags, estimates rough size, notes which ideas reference which existing work, surfaces the top candidates for next batch. The human walks into grooming with a curated menu, not an archaeological dig site. | **Proactive** — assistant maintains the backlog continuously; human arrives to decide, not to discover | PM |
| 6. Document version hunting | Tracks document status across the project: "The latest approved design for the auth system is here. It was superseded once — here's the superseding document and why. The spec that depends on it is here, currently in draft." | **Reactive** — human asks "where's the design for X?", assistant answers with the full context | Designer, PM |
| 7. Review-triage decision fatigue | Pre-triages agent review findings: "3 findings need your decision. The rest are cosmetic or informational — I've noted them but they don't block merge." Surfaces a decision menu, not a firehose. | **Proactive** — assistant triages before presenting | PM, Tech lead |
| 8. Checklist verification | Verifies stage gate prerequisites automatically: "B12 is ready to advance to review. All tasks are done, spec is approved, design is approved. Here's the summary — say 'advance' and I'll handle it." | **Proactive** — assistant checks before the human needs to ask | PM |
| 9. Stakeholder reminder-chasing | Tracks outstanding decisions: "You're waiting on a decision from Alex about the API surface. It's been 4 days. Would you like me to draft a follow-up, or should I flag it in the next checkpoint?" | **Proactive** — assistant notices the stall and surfaces it | PM |
| 10. "What decisions did we make about Y?" | Is the librarian: every decision, rationale, and outcome is findable. "In the design doc for the auth system (approved March 15), you decided to use OAuth2 with PKCE rather than API keys. The rationale was..." | **Reactive** — human asks, assistant retrieves with context | Designer, Tech lead |

### What themes emerge from the empathy exercise?

**The assistant spends most of its time on three activities:**
1. **Synthesising state** — turning raw entity/document/review data into
   human-consumable summaries (burdens 1, 2, 5, 7, 8)
2. **Surfacing connections** — noticing dependencies, conflicts, and stalled
   work that the human hasn't seen (burdens 4, 9)
3. **Filing and retrieving** — capturing ideas with context, organising the
   backlog, making past decisions findable (burdens 3, 6, 10)

**Proactive vs. reactive split:** 7 of 10 helper behaviours are at least
partially proactive. The assistant doesn't just answer questions — it
notices things and surfaces them. This is a significant design input: a
purely on-demand tool would miss more than half the value.

**The assistant is a single person with a single relationship to each human.**
In every scenario, the human delegates to "my assistant" — not to a cast of
specialised clerks. The assistant may *use* different skills internally, but
presents a unified identity. This suggests the named-role approach (Option A
from the research report) has strong empathy support.

## Role-Specific Needs

Different human roles have different core creative work to protect, different
busy-work that disrupts it, and potentially different preferences for how
they interact with an assistant.

### Product Manager

**Core creative work to protect:** Product vision, user need discovery,
feature prioritisation, scope negotiation, stakeholder alignment. The PM
thinks in terms of problems, outcomes, and roadmaps.

**Top 3 burdens an assistant would most impactfully absorb:**
1. **Status reassembly and reporting** (burdens 1, 2) — the PM spends
   significant time reconstructing state to answer questions from stakeholders
   and to decide what needs attention.
2. **Backlog grooming as archaeology** (burden 5) — the PM's backlog is their
   primary planning tool, and maintaining it currently requires reading old
   notes to remember what they contain.
3. **Dependency blindness** (burden 4) — the PM is accountable for delivery
   coordination, and discovering cross-batch dependencies during
   implementation is the most expensive kind of surprise.

**Preferred interaction mode:** The PM is likely to want both conversational
and dashboard interactions. Conversational for scoping and shaping ("should
this be a batch or a plan?", "what depends on this if we build it now?").
Dashboard/visual for portfolio-level awareness ("show me everything that's
blocked across all active plans"). The PM is the role most likely to want
proactive surfacing — they need to know about problems before being asked.

**PM-specific assistant behaviours that might not apply to designers:**
- Portfolio-level status synthesis across all active plans
- Backlog curation and "what should we work on next?" recommendations
- Stakeholder reminder tracking
- Stage-gate readiness verification

### Designer

**Core creative work to protect:** Deep creative flow — exploring design
directions, prototyping, iterating on interaction patterns, developing
visual systems. The designer thinks in terms of experiences, flows, and
systems. Flow state is sacred; interruptions are disproportionately
expensive.

**Top 3 burdens an assistant would most impactfully absorb:**
1. **Document version hunting** (burden 6) — the designer needs to work from
   the latest approved specs and designs, and finding them is a context switch
   that breaks flow.
2. **"What decisions did we make about Y?"** (burden 10) — design decisions
   from past work inform current work, and not finding them means either
   re-deciding or designing with uncertainty.
3. **Idea capture gap** (burden 3) — designers have ideas during creative
   flow that they don't want to stop to document properly; losing those
   ideas is a creative cost.

**Preferred interaction mode:** The designer is likely to prefer on-demand,
non-interruptive interaction. A conversational interface they can query when
they need something — but that doesn't proactively push notifications during
flow. Alternatively, an ambient dashboard they can glance at without
switching contexts. The designer may also prefer visual, spatial organisation
of ideas (mood boards, canvases, clustering) over structured lists.

**Designer-specific assistant behaviours that might not apply to PMs:**
- Design artifact tracking (which version of the mockup goes with which spec?)
- Creative brief filing and retrieval
- Non-interruptive, glanceable status (ambient rather than push)

### Where needs converge and diverge

**Convergence (one assistant shape serves both):**
- Both need document retrieval and decision lookup
- Both benefit from idea capture with context preservation
- Both want stage-gate verification handled automatically
- Both need to find "where are we on X?" quickly

**Divergence (role-specific behaviours or separate assistants):**
- **Proactivity tolerance:** PMs describe wanting proactive help ("tell me
  before I ask"); designers describe wanting on-demand help ("don't interrupt
  my flow"). This is a significant interaction-model tension.
- **Portfolio vs. focused scope:** PMs need cross-plan, portfolio-level
  awareness. Designers typically need awareness of the batch or feature they're
  currently designing, plus the decisions that inform it.
- **Visual vs. structured organisation:** Designers may want spatial, visual
  organisation of ideas. PMs may prefer structured lists and dashboards.
- **Stakeholder management:** PMs need reminder-chasing and decision-tracking.
  This is not a designer burden.

**The fundamental tension: Is proactivity a feature or an interruption?**
For the PM, proactive surfacing of problems is essential — that's what they
pay an assistant for. For the designer, proactive surfacing is a flow-breaker.
A single assistant might need to adapt its proactivity per role, or per
context (proactive during planning sessions, quiet during design work). Or
there might be separate PM and designer assistants with different defaults.

## Candidate Design Directions

From the burden inventory, empathy exercise, and role analysis, three
contrasting conceptual directions emerge. They differ along two key axes:
proactive vs. on-demand, and conversational vs. visual.

### Direction A: The Personal Assistant ("Mise en Place")

**Core metaphor:** A skilled executive assistant who knows your work, your
priorities, and your preferences. They prepare your workspace before you
arrive, handle routine administrative work without being asked, and are
always available for questions. The name evokes a chef's *mise en place* —
everything in its place before the work begins.

**Primary interaction mode:** Conversational, with proactive briefings pushed
to where you already work (your chat interface, your terminal, your morning
routine). You talk to your assistant the way you'd talk to a human one: "What
do I need to know today?", "File this idea for later", "Is B12 ready to
advance?", "What's blocking the auth workstream?"

**Signature behaviours:**
1. **Morning briefing** (proactive): "Good morning. Since Friday: 3 agents
   completed work, 1 is blocked on spec clarification (B12, holding up 2
   dependent tasks), and 2 reviews need your attention. The urgent item is
   the blocked agent — would you like me to draft a checkpoint for the
   human gate?"
2. **Frictionless idea capture** (reactive): "I have an idea for a notification
   system." Assistant: "Filed under ideas/notification-system with today's
   context. I've noted it relates to the event system in B8. I'll surface it
   during your next backlog grooming."
3. **Pre-gate checklist** (proactive): "B12 is ready to advance to review
   — all tasks done, spec approved, design approved, review docs registered.
   Say 'advance' and I'll handle the transition."
4. **Stall detection** (proactive): "You've been waiting on a decision from
   Alex about the API surface for 4 days. B12's next phase depends on it.
   Want me to surface this in the next status update?"
5. **Review triage** (proactive): "3 findings need your decision. 8 are
   cosmetic — I've noted them in the log but they don't block anything."

**Handoff to PM/orchestrator:** The assistant *prepares* work for the pipeline
but does not *direct* it. It captures ideas, helps shape scope, verifies
gates — then hands off: "Scope agreed for feature F. Creating entities now."
At that point, Kanbanzai's existing orchestrator takes over (decompose into
tasks, dispatch, review). The assistant continues to monitor and surface, but
the orchestrator drives execution.

**Who it serves best:** Product managers (whose work is interrupt-driven and
benefits from proactive surfacing). Designers would need the assistant to
adapt to a quieter, on-demand mode.

**Biggest risk or open question:** The assistant's scope could grow unboundedly
— "my assistant handles it" is a powerful mental model that invites scope
creep. Where exactly is the boundary between the assistant's responsibilities
(organise, remind, surface, capture, structure) and the PM/orchestrator's
(decompose, dispatch, review, track)? This boundary needs to be crisp and
enforceable.

### Direction B: The Planning Workbench

**Core metaphor:** A structured visual surface for capturing, organising,
and grooming work before it enters the execution pipeline. Think of a
designer's workspace — a large surface where ideas are cards, batches are
clusters, and dependencies are visible connections. Less like a dashboard
and more like a canvas.

**Primary interaction mode:** Visual/spatial, with conversational elements
for scoping and shaping. The human sees their portfolio as a spatial layout:
plans, batches, and features as cards or nodes; dependencies as visible
connections; status as colour or position. Interaction is direct — drag to
reprioritise, click to expand, draw a line to create a dependency.

**Signature behaviours:**
1. **Visual portfolio canvas**: Every active plan, batch, and feature is
   visible on a single surface. Zoom out for portfolio view; zoom in for
   batch detail. Blocked items are visually distinct. Dependencies are
   visible as connections between cards.
2. **Backlog as spatial organisation**: Ideas aren't a list — they're cards
   you can cluster, rank, and annotate. Grooming is a spatial activity:
   group related ideas, move high-priority ones to the "next batch" zone,
   archive stale ones.
3. **Dependency visualisation**: When scoping a new batch, the workbench
   overlays potential dependencies on the existing portfolio: "This proposed
   feature would depend on B8 (auth system) and B12 (event system). B12 is
   still in design." The human sees the dependency graph, not a text warning.
4. **Scope-shaping conversation sidebar**: A conversational panel alongside
   the visual surface. You describe the idea conversationally; the assistant
   structures it into draft entities that appear on the canvas for you to
   arrange, connect, and refine.
5. **Portfolio-level status synthesis**: The canvas itself is the status
   report. Glance at it and see: what's green, what's red, what's stalled,
   what's moving. No need to run a status command or read a report.

**Handoff to PM/orchestrator:** When a batch or feature is shaped and scoped
on the workbench, the human "commits" it — which creates the entities in
Kanbanzai and hands off to the orchestrator. The workbench continues to
display status (fed by the orchestrator's progress data) but doesn't direct
execution. The handoff is visual and explicit: drag from "shaping" zone to
"active" zone.

**Who it serves best:** Designers (who think spatially and visually) and PMs
during portfolio-level planning sessions. The visual canvas matches how both
roles think about relationships between work items.

**Biggest risk or open question:** A visual workbench is a separate application
from Kanbanzai's CLI/MCP interface. This creates a context-switching cost and
a development cost. Is the spatial/visual benefit worth splitting the tool
surface? Alternatively: could a text-based canvas (a structured markdown
document with dependency graphs rendered as ASCII/Mermaid) achieve 80% of the
value without the development cost of a visual UI?

### Direction C: The Team Librarian

**Core metaphor:** A research librarian who catalogues, cross-references, and
retrieves. The librarian doesn't shape work or surface priorities — they
ensure that every decision, document, and idea is findable, correctly linked,
and properly contextualised. They are the answer to "where is X?" and "what
decisions did we make about Y?"

**Primary interaction mode:** Conversational, on-demand. The human asks
questions; the librarian retrieves answers with full context. There is no
proactive surfacing and no visual dashboard — the librarian is a knowledge
retrieval expert, not a status monitor. (Could be combined with Direction A
as a "mode" — the same assistant doing retrieval on demand vs. surfacing
proactively.)

**Signature behaviours:**
1. **Decision retrieval**: "Why did we choose OAuth2 with PKCE over API keys?"
   The librarian returns the exact decision, its rationale, the document it's
   recorded in, when it was made, and what other decisions reference it.
2. **Document provenance tracking**: "Is this the latest version of the auth
   design?" The librarian answers with the full document chain: original,
   superseding document, current approved version, and any drafts in progress.
3. **Cross-reference discovery**: "What specs depend on the event system
   design?" The librarian traces the document dependency graph and returns
   all documents that reference or depend on the specified document.
4. **Idea filing with rich metadata**: When the human captures an idea, the
   librarian files it with tags, related documents, related entities, and
   the original capture context. Retrieval is precise because filing was
   thorough.
5. **Gap detection**: "You have a spec that references a design that was
   never approved. Would you like me to flag this?" The librarian notices
   inconsistencies in the document corpus that humans would miss.

**Handoff to PM/orchestrator:** The librarian is purely an information
retrieval layer. It doesn't feed into the pipeline — it supports the humans
who do. The handoff is indirect: humans use the librarian to find the
information they need to make planning, design, and review decisions.

**Who it serves best:** Everyone, but especially designers and tech leads who
need precise answers to specific questions to inform their work. Less useful
for PMs who need proactive surfacing and portfolio synthesis.

**Biggest risk or open question:** A pure retrieval layer may not feel like
enough — it addresses burdens 6 and 10 (document hunting and decision lookup)
but not burdens 1, 2, 4, or 7 (status reassembly, dependency blindness, review
triage). The librarian is an important function, but is it a standalone
direction or a capability that should be part of a broader assistant?

## Trade-Offs and Tensions

### Proactive vs. On-Demand

The empathy exercise revealed that 7 of 10 helper behaviours are at least
partially proactive. But the role analysis revealed that PMs want proactivity
and designers want on-demand interaction to protect flow. This is the central
design tension. Candidate resolutions:

- **Adaptive proactivity**: A single assistant that adjusts its proactivity
  level based on the human's role and current context (proactive during
  planning sessions, quiet during design work).
- **Role-specific defaults**: PM assistant is proactive by default; designer
  assistant is on-demand by default. Both can be configured.
- **Proactivity as opt-in per behaviour**: The human configures which
  behaviours are proactive (morning briefing? yes. Stall detection? yes.
  Review triage? only if critical) and which are on-demand.

### Single Assistant vs. Role-Specific Assistants

The empathy exercise strongly supports a single-assistant model — every
scenario reads as "my assistant." But the role analysis reveals genuinely
different needs between PMs and designers along the proactivity axis and
the portfolio-vs-focused axis. Candidate resolutions:

- **One assistant, role-adaptive**: A single assistant identity that presents
  differently to PMs (proactive, portfolio-level) and designers (on-demand,
  focused scope).
- **One assistant, multiple interfaces**: The same assistant backend with
  different front-ends — a proactive briefing for PMs, a query interface for
  designers.
- **Separate assistants for separate roles**: A "PM Assistant" and a "Design
  Assistant" with different behaviours, different default proactivity levels,
  and different tool surfaces. This matches Kanbanzai's existing pattern of
  role-specific profiles.

### Conversational vs. Visual

The PM's preferred interaction mode spans both conversational and dashboard.
The designer's spans conversational and visual/spatial. The existing Kanbanzai
system is entirely conversational (CLI/MCP). A visual surface is a
significant architectural departure. Candidate resolutions:

- **Conversational-first with rendered output**: All interaction is
  conversational, but the assistant can render visual output (Mermaid
  dependency graphs, structured tables, status dashboards as markdown).
- **Conversational + visual workbench**: The conversational assistant handles
  capture, scoping, and Q&A. A separate visual workbench handles portfolio
  viewing, dependency visualisation, and backlog organisation.
- **Conversational-only**: Accept that some spatial thinking (portfolio canvas,
  backlog clustering) doesn't translate to conversation, and the value of a
  single interaction surface outweighs the spatial benefits.

### Inside Kanbanzai vs. Separate Tool

Direction B (Planning Workbench) implies a separate application. Directions
A and C (Personal Assistant and Team Librarian) could live inside the existing
CLI/MCP interface. Candidate resolutions:

- **Everything in MCP**: Build all intake/admin functions as MCP tools within
  the existing server. The "visual" components are markdown renderings
  (Mermaid graphs, tables). This is consistent with the research report's
  Recommendation 3 and avoids tool-surface fragmentation.
- **MCP for logic, separate UI for visual**: Build the logic as MCP tools
  and add a thin visual front-end that consumes the same structured data.
  This is the research report's "visual UI can be added later as a rendering
  layer."
- **Separate visual tool**: Commit to a visual workbench as a first-class
  product alongside the CLI/MCP server.

### Greenfield vs. Retrofit

Kanbanzai already has an entity system, a planning skill, status dashboards,
and a conversational planning pattern. An intake layer could be:

- **Retrofit**: Extend existing tools (`entity`, `status`, `doc_intel`) with
  intake/admin capabilities. Add morning briefing, dependency surfacing,
  backlog management as new parameters or new tools within the existing MCP
  server.
- **Greenfield**: Build the planning/admin layer as a separate product
  (possibly a separate MCP server, possibly a visual app) that sits in front
  of Kanbanzai and feeds structured entities into it.

## Questions for the Team

These are the questions the human team should answer before moving to a
formal design stage. They are organised by how foundational they are —
answer the early ones first, as later answers depend on them.

### Direction and scope

1. **Which direction excites you most, and why?** Direction A (Personal
   Assistant), Direction B (Planning Workbench), Direction C (Team
   Librarian), or a hybrid? This is the primary question — everything
   else flows from it.

2. **Is the Librarian (Direction C) a standalone product direction or
   a capability that all directions need?** The librarian function
   (decision retrieval, document provenance, cross-reference discovery)
   seems universally valuable regardless of which direction we choose.
   Should it be part of every direction, or is it valuable enough to be
   its own thing?

3. **How ambitious should we be?** Direction A (Personal Assistant) is the
   most architecturally conservative — it extends the existing conversational
   pattern. Direction B (Planning Workbench) is the most ambitious — it
   introduces a visual application. Direction C (Team Librarian) is in
   between. Which level of ambition matches our appetite?

### Interaction model

4. **How do you want to interact with this layer?** Chat/conversational?
   Dashboard/visual? CLI? A combination? If combination, which is primary
   and which is secondary?

5. **Would you read a morning briefing, or would you prefer to ask for
   one on demand?** This is the proxy question for the proactivity debate.
   If at least one team member would read a briefing, proactivity has a
   foothold. If everyone prefers on-demand, the proactive behaviours may
   be over-engineered.

6. **Is role-specific proactivity a feature or a configuration nightmare?**
   PMs want proactive help; designers want on-demand help to protect flow.
   Should the assistant adapt per role, per context, or per configuration?
   Or should we pick one default and live with the consequences?

### Role model

7. **Would you rather have one assistant that serves everyone, or
   role-specific assistants?** The empathy exercise strongly suggests a
   single assistant ("my assistant"), but the role analysis suggests
   meaningfully different needs. Which feels more natural to you?

8. **Should the assistant have a named identity?** (e.g., a "Clerk" role
   in the Kanbanzai role system, or an "Admin Assistant" conversational
   persona.) Or should it be a set of tools you use through your existing
   agent interactions? The research report recommended deferring this
   question — is now the right time to answer it?

### Specifics

9. **What did I miss in the burden inventory?** Are there daily busy-work
   activities you experience that aren't captured? Are any of the listed
   burdens overestimated or not actually a problem?

10. **What's the one thing that, if this layer did it well, would most
    improve your daily experience?** This is the "if we only build one
    thing" question. It helps prioritise within whatever direction we
    choose.

11. **Should the idea-capture surface be inside or outside your primary
    workflow?** The empathy exercise describes "frictionless capture" —
    is that a chat message to your assistant, a quick-add command, a
    mobile-friendly capture point, or something else?

12. **How much backlog structure do you actually want?** The backlog
    grooming burden (5) assumes you want a curated, categorised backlog.
    Would you actually maintain one, or would it become another thing
    to manage? Is a lightweight "idea inbox" sufficient?

## What This Document Is and Isn't

This is a brainstorming artefact — it explores the design space, surfaces
tensions, and poses questions. It is not a resolved design, not a
specification, and not an implementation plan. The team should react to it,
not approve it.

If a direction is chosen, the next stage would be a formal design document
produced by the architect role following the `write-design` skill, with
the design informed by the team's answers to the questions above.

The research report that preceded this brainstorm
(`work/research/planning-admin-layer-exploration.md`) provides additional
context: competitive analysis of existing tools, historical analogy to
pre-computer clerical roles, and a trade-off analysis of the role-model
question. It's recommended reading alongside this document.
