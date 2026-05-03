# Hill Charts as a Shaping Metaphor for Kanbanzai: Research Report

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-05-01T12:14:47Z          |
| Status | Draft                         |
| Author | researcher                    |

## Research Question

Can Basecamp's Hill Chart concept (from the Shape Up methodology) serve as a useful metaphor for the shaping layer of a system that feeds into Kanbanzai? Specifically: does the hill metaphor hold up when applied to the uncertainty-reduction phase of moving from "I have an idea" to "I know what I'm building," and does it scale to large, multi-cycle projects without collapsing under its own weight?

This research was requested as a brainstorming exercise. No design or architecture decisions are being taken at this stage.

## Scope and Methodology

**In scope:**
- The Hill Chart concept as described in Shape Up (Chapters 11–13) and related Basecamp/37signals writing
- How the hill metaphor maps to Kanbanzai's existing entity hierarchy (plan, batch, feature) and lifecycle stages (idea → shaping → ready → active → done)
- Whether the concept scales from single-cycle projects to large multi-cycle initiatives with nested plans
- Where an AI thinking partner could add the most value within the hill framework
- Relationship to prior Kanbanzai research on the planning and administration layer

**Out of scope:**
- Detailed tool specification or MCP tool signatures
- Role design (the Admin Assistant vs. decomposed roles question — addressed in the prior research)
- Implementation planning or timeline
- Comparison with alternative uncertainty-tracking frameworks beyond the hill metaphor

**Methodology:** Primary source analysis (Shape Up chapters 3, 11, 12, 13), competitive analysis (Basecamp's implementation), internal system analysis (Kanbanzai entity model, stage bindings, plan lifecycle, and prior research report on planning-admin layer). Evidence graded by source type and recency.

## Findings

### Finding 1: The Hill Chart Is Built on One Central Insight — Work Has Two Fundamentally Different Phases

The Hill Chart's core contribution is the recognition that progress tracking conflates two distinct kinds of work:

| Phase | Position | What's Happening | How It Feels |
|-------|----------|-----------------|--------------|
| **Uphill** | Bottom → Top | Figuring out *what* to do and *how* to approach it | Uncertainty, unknowns, problem-solving, discovery |
| **Downhill** | Top → Bottom | Executing known work where all steps are visible | Certainty, confidence, knowing what to do |

During the uphill phase, the question "what percent complete is this?" is nonsensical. You cannot estimate work you haven't discovered yet. In fact, to-do lists grow during the uphill phase as tasks are discovered through engagement with the problem. This is the distinction between *imagined* tasks (planned up front) and *discovered* tasks (found through doing).

The downhill phase is where traditional project management tools work: tasks are known, estimates are meaningful, and completion can be tracked linearly. The uphill phase is where most cognitive effort goes, and where most tools provide no support.

Source: Shape Up, Chapter 13 ("Show Progress"). Primary, 2019.

### Finding 2: The Hill Chart Provides Five Distinct Capabilities Beyond Progress Tracking

Analysis of Shape Up Chapter 13 reveals five capabilities the hill chart unlocks, each of which has a potential Kanbanzai analogue:

**a) Status without asking.** Managers don't need to interrupt teams — they can see dots moving (or not moving) on the hill. This replaces the awkward status question with self-serve visibility.

**b) Stuckness detection without blame.** A dot that doesn't move is a raised hand: "Something might be wrong here." Nobody has to say "I don't know." The language of uphill/downhill makes the conversation about the work, not the person.

**c) Scope refactoring prompts.** When a dot is stuck too long, the question becomes: is the work really one thing, or should it be split into smaller scopes that can move independently? The hill chart reveals bad scope boundaries — a scope that contains work at different hill positions needs to be decomposed.

**d) Risk sequencing via the inverted pyramid.** Teams push the scariest work (most unknowns) uphill *first*, leaving routine work for later. The critical insight: some scopes present novel problems that could take weeks, while others (like email templates) could be whipped together in a day. The hill chart makes this sequencing decision visible and explicit.

**e) The "build your way uphill" rule.** You don't get to the top by thinking alone — you must use your hands. The first third of the uphill is "I've thought about this," the second third is "I've validated my approach," and the final third is "I'm far enough with what I've built that I don't believe there are other unknowns." This prevents teams from declaring a scope "solved" based on theory alone, only to slide back down when reality intervenes.

Source: Shape Up, Chapter 13. Primary, 2019.

### Finding 3: The Hill Maps Cleanly to Kanbanzai's Existing Entity Hierarchy, but at Two Distinct Levels

Kanbanzai's entity hierarchy (plan → batch → feature → task) and its plan lifecycle (idea → shaping → ready → active → done) already contain the conceptual space for hill charts. However, conflating the two levels at which hills operate causes confusion:

| Level | Shape Up Concept | Kanbanzai Equivalent | What the Hill Tracks |
|-------|-----------------|---------------------|---------------------|
| **Execution hill** | One project's scopes in a 6-week cycle | One batch's features | Uncertainty reduction on *specific scoped implementation work* |
| **Portfolio hill** | Not addressed in Shape Up (deliberately omitted) | One plan's batches (or sub-plans) | Uncertainty reduction on *strategic direction* |

**The execution hill** maps to batches. A batch is a shippable unit of work (analogous to a Shape Up cycle). Its features are scopes on the hill. The hill tracks uncertainty reduction within that batch: which features have known scope and clear implementation paths, and which are still being discovered. This is the level Shape Up describes in detail — scopes within a cycle share a common time box and a common level of granularity, making all dots comparable.

**The portfolio hill** maps to plans. Plans are recursive strategic containers that span multiple batches. Shape Up deliberately does not address this level — the methodology trusts that shipping complete cycles sequentially will cause the big thing to emerge. However, Kanbanzai's Plan entity provides the structural container that Shape Up lacks, making a portfolio-level hill conceptually coherent: a plan at the bottom of the hill means "we have a vague strategic intent but no clear shape"; a plan at the top means "we know exactly what batches belong here and what each one needs to achieve."

**Critically, these are different kinds of uncertainty** and should not be plotted on the same hill. Execution-hill uncertainty is about implementation: "how do we build this?" Portfolio-hill uncertainty is about direction: "what should we build?" Conflating them would produce dots that aren't comparable — a batch at the bottom of the portfolio hill (strategically unclear) is in a fundamentally different state from a feature at the bottom of the execution hill (implementation unclear).

Source: Internal system analysis — `.kbz/stage-bindings.yaml` (plan lifecycle), `kanbanzai-planning` skill (entity hierarchy and scope decisions). Primary, current.

### Finding 4: The "Many Hills" Question Has a Specific Answer in Shape Up — The Appetite Mechanism

Shape Up handles large initiatives not by providing a multi-hill visualization, but by enforcing a structural constraint: **the appetite**. Before any work reaches the hill, it goes through shaping where a raw idea gets bounded by an explicit time budget:

- Small Batch: 1–2 weeks
- Big Batch: 6 weeks

If an idea is too big to fit within the appetite, the options are:
1. **Narrow the problem definition** — find a more specific version that fits
2. **Break off a meaningful slice** — carve out a piece that can be shaped
3. **Walk away** — "Interesting. Maybe some day."

This means a large initiative never reaches a cycle as one entity. It's decomposed during shaping into pieces that each fit within an appetite. Each piece gets its own cycle, its own scopes, and its own hill chart.

The "foothills" are not planned in advance as a decomposition of the big hill. They are **discovered sequentially**. Each completed foothill tells you something about the terrain that shapes the next one. This is fundamentally different from waterfall decomposition where everything is broken down upfront.

Shape Up's scoping mechanism (Chapter 12, "Map the Scopes") reinforces this: scopes are "discovered, not planned." They emerge from walking the territory. You don't know what the real interdependencies are until you start doing real work. The scope map is drawn retrospectively from engagement, not prospectively from imagination.

Source: Shape Up, Chapters 3 ("Set Boundaries") and 12 ("Map the Scopes"). Primary, 2019.

### Finding 5: Shape Up's Portfolio-Level Gap Is Kanbanzai's Opportunity

Shape Up deliberately omits portfolio-level tracking across cycles. This is a philosophical choice — the methodology is designed around complete, shippable cycles with a "cool-down" period between them. If a project can't be shaped to fit a single cycle, the methodology says: narrow the problem further, or accept that the initiative will span multiple cycles with each being independently valuable.

However, this omission is exactly where Kanbanzai's Plan entity adds value. The plan provides:
- A container for tracking strategic direction across multiple batches
- A lifecycle (idea → shaping → ready → active → done) that mirrors the uncertainty-reduction arc
- Recursive structure (plans can contain sub-plans) that maps to hierarchical decomposition of large initiatives

The prior research on the planning-admin layer (Finding 2) identified five missing cognitive functions, two of which are portfolio-hill concerns:
- **Cross-plan dependency detection:** "Plan A's batch B2 depends on Plan C's feature F3"
- **Portfolio-level progress synthesis:** "across all active plans, what's blocked, what's close to done, and what needs attention?"

A portfolio hill chart — tracking plan and batch positions on the uncertainty-reduction arc — would directly address these gaps while staying within the hill metaphor's conceptual framework.

Source: Internal system analysis — prior research report `planning-admin-layer-exploration.md` (Findings 1 and 2), `kanbanzai-planning` skill (plan lifecycle and scope decisions). Primary, current.

### Finding 6: The Hill Metaphor Is Strongest as a Conversational Framework, Not Just a Visualization

The Shape Up book describes hill charts primarily as a *visualization* tool for managers. But for Kanbanzai's purposes — an MCP-first, text-mediated system where AI agents and humans collaborate through conversation — the more interesting application is as a **conversational framework**.

The hill metaphor provides a shared language that replaces broken questions with productive ones:

| Broken Question | Hill-Based Replacement | Why It's Better |
|---|---|---|
| "What percent complete is this?" | "Where is this on the hill?" | Doesn't assume tasks are known |
| "How long will this take?" | "What's keeping this at the bottom?" | Prompts articulation of the unknown |
| "Are you stuck?" | "This has been uphill for a while — what's the unknown holding it back?" | About the work, not the person |
| "What should we work on first?" | "Which scope has the most unknowns and the highest criticality?" | Sequences by uncertainty, not just priority |
| "Is this ready to implement?" | "Can you see all the steps from here?" | Tests for the top-of-hill condition |

This maps directly to the `kanbanzai-planning` skill's conversational pattern — the agent asks clarifying questions, reflects scope back, and flags sizing issues. The hill metaphor makes this conversation more precise and less awkward, providing a vocabulary for uncertainty that both human and AI can use.

The AI partner's value is not in rendering a visual hill chart (though that may be useful later). It's in using the hill language to structure the shaping conversation: asking the right questions at the right phase, surfacing stuck items, prompting scope decomposition when a dot won't move, and suggesting sequencing based on the inverted pyramid principle.

Source: Internal system analysis — `kanbanzai-planning` skill (conversational planning pattern). Primary, current. Shape Up, Chapter 13 (hill chart as status conversation tool). Primary, 2019.

### Finding 7: Where an AI Thinking Partner Adds the Most Value

The hardest cognitive work isn't in the downhill phase — Kanbanzai already handles that well with orchestrators, implementers, and reviewers. The hard part is the uphill phase: moving from "I have an idea" to "I know what I'm building." This is where an AI thinking partner could add the most value, functioning as a **shaping clerk** that helps the human reduce uncertainty at each stage of the climb:

| Hill Position | Human Cognitive Load | AI Partner Contribution |
|---|---|---|
| **Bottom of hill** — raw idea | "I have a vague sense something should exist" | Capture and reflect: "Here's what I understand you want. Did I get it right? What problem does this solve?" |
| **Lower uphill** — exploring | "What are the options? What's been done before?" | Research and surface: search existing designs, flag overlaps, retrieve relevant decisions and constraints |
| **Mid-uphill** — narrowing | "Which approach? What are the tradeoffs?" | Structure the comparison: "Option A gives you X but costs Y. Option B gives you Z. Here's how each maps to existing architecture." |
| **Upper uphill** — validating | "I think this works, but am I missing something?" | Stress-test: "If we build this, here's what breaks. These three assumptions need validation. This dependency is risky." |
| **Top of hill** — ready | "I know what to build and can describe it" | Transition to execution: decompose into batches/features, flag which have remaining unknowns, propose a design document structure |

The "build your way uphill" rule from Shape Up has a natural analog in Kanbanzai: you move uphill by producing documents. An idea becomes a research report (lower uphill), which becomes a design draft (mid-uphill), which becomes an approved design (top of hill). The AI partner facilitates this progression by drafting research, surfacing design questions, identifying decision points, and flagging when a design is detailed enough to transition to specification.

This aligns with the prior research's Finding 4: pre-computer clerical roles (correspondence clerk, committee secretary, schedule clerk) map well to the cognitive functions needed. The AI partner doesn't decide what to build — it structures, organizes, and surfaces information so the human can decide effectively.

Source: Internal system analysis — prior research report `planning-admin-layer-exploration.md` (Finding 4 on clerical roles), stage bindings (design → specification → implementation pipeline). Primary, current. Shape Up, Chapter 13. Primary, 2019.

## Trade-Off Analysis

### The Hill Metaphor vs. Alternative Uncertainty-Tracking Approaches

| Criterion | Hill Chart | Linear Progress Bar | Gantt Chart | Kanban Board |
|-----------|-----------|-------------------|-------------|--------------|
| **Captures uncertainty phase** | Yes — uphill is explicitly about uncertainty | No — assumes work is known | No — assumes tasks are estimated | Partial — "blocked" is binary, not graduated |
| **Distinguishes discovery from execution** | Yes — fundamental to the model | No — all progress is linear | No — dependencies, not discovery | No — columns track status, not certainty |
| **Prompts scope decomposition** | Yes — stuck dots reveal bad boundaries | No | No | No |
| **Supports risk sequencing** | Yes — inverted pyramid is integral | No | No | No |
| **Requires visual rendering** | Yes — spatial metaphor | Minimal — bar is simple | Yes — dependency lines | Yes — columns and cards |
| **Works in text-mediated conversation** | Yes — the language works without visuals | Yes | No — loses meaning without lines | Partial — columns translate, swimlanes don't |
| **Scales to portfolio level** | Yes — with two-level distinction | No — single-bar model | Partial — becomes unwieldy | No — designed for single-board view |

The hill chart's key advantage is that it's the only approach that explicitly models uncertainty reduction as a distinct phase with its own vocabulary and logic. Its key disadvantage is that it's inherently spatial and loses some meaning when reduced to text alone — though the *language* of the hill (uphill/downhill, top of the hill, build your way uphill) works well conversationally.

### Single Hill vs. Two-Level Hill Architecture

| Criterion | Single Portfolio Hill | Two-Level (Portfolio + Execution Hills) |
|-----------|----------------------|----------------------------------------|
| **Simplicity** | High — one view | Lower — two views to understand |
| **Dot comparability** | Low — plans and features aren't at the same granularity | High — each hill has comparable dots |
| **Matches Kanbanzai entity model** | Low — conflates plan and batch/feature levels | High — maps to plan vs. batch distinction |
| **Matches Shape Up philosophy** | Low — Shape Up doesn't do portfolio tracking | Medium — extends Shape Up while respecting its boundaries |
| **Implementation complexity** | Lower | Higher — two hill tracking systems |
| **Conversational clarity** | Lower — "where is this on the hill?" is ambiguous | Higher — "where is this plan on the portfolio hill?" vs. "where is this feature on the batch hill?" |

The two-level architecture is the stronger choice for Kanbanzai because it respects the entity hierarchy already in place. The risk of a single portfolio hill is that it would produce meaningless comparisons — a plan at "mid-uphill" means something fundamentally different from a feature at "mid-uphill," and plotting them together obscures rather than reveals.

## Recommendations

### Recommendation 1: Adopt the Hill Metaphor as a Conversational Framework for the Shaping Layer

**Recommendation:** Use the hill chart's vocabulary and concepts (uphill/downhill, top of the hill, inverted pyramid sequencing, "build your way uphill") as the conversational framework for the planning and shaping stages of Kanbanzai's workflow. Do not attempt to replicate Basecamp's visual drag-and-drop hill chart in the initial implementation — the language itself provides most of the value in a text-mediated, AI-facilitated conversation.

**Confidence:** High
**Based on:** Findings 1, 2, and 6
**Conditions:** This recommendation assumes the shaping conversation remains text-mediated (MCP/CLI). If a visual dashboard is later added (per the prior research's Recommendation 3), a visual hill chart would be a natural addition at that point.

### Recommendation 2: Model Two Distinct Hill Levels — Portfolio and Execution

**Recommendation:** Implement hill tracking at two levels:
- **Execution hill:** Per-batch, tracking feature-level uncertainty reduction. This is the direct analogue of Shape Up's hill chart — scopes (features) within a bounded work unit (batch).
- **Portfolio hill:** Per-plan, tracking batch-level or sub-plan-level uncertainty reduction. This is the extension Shape Up lacks — strategic direction tracking across multiple execution units.

The two levels use the same vocabulary (uphill/downhill, top of the hill) but track different entities at different granularities. They should not be merged into a single view — dots at different levels are not comparable.

**Confidence:** Medium
**Based on:** Findings 3, 4, and 5
**Conditions:** The portfolio hill is the more speculative of the two. The execution hill has a proven model (Shape Up). The portfolio hill's value depends on whether Kanbanzai users actually manage multiple concurrent plans with strategic uncertainty — if most plans are single-batch, the portfolio hill adds complexity without benefit.

### Recommendation 3: Map Hill Position to Document Production, Not Subjective Assessment

**Recommendation:** Tie hill positions to concrete artifacts rather than subjective confidence ratings. The "build your way uphill" rule from Shape Up translates naturally to Kanbanzai's document pipeline:

| Hill Position | Kanbanzai Artifact | What It Means |
|---|---|---|
| Bottom | Raw idea (no document) | Intent exists only in conversation |
| Lower uphill | Research report registered | Options explored, constraints documented |
| Mid-uphill | Design document (draft) | Approach selected, tradeoffs articulated |
| Upper uphill | Design document (approved) | Approach validated, open questions resolved |
| Top of hill | Specification (approved) | All requirements defined, ready for dev-plan |
| Downhill | Dev-plan → Tasks → Implementation | Kanbanzai's existing execution pipeline |

This makes hill position objective and auditable — you can't claim to be at the top of the hill without an approved specification. It also integrates naturally with Kanbanzai's existing document approval gates.

**Confidence:** High
**Based on:** Findings 7, internal system analysis of document pipeline
**Conditions:** This mapping assumes the current document types (research, design, specification) cover the full uncertainty-reduction arc. If new document types are needed for earlier-stage work (e.g., a lightweight "concept note"), the mapping should be extended.

### Recommendation 4: Integrate with the Existing Planning Conversation, Don't Replace It

**Recommendation:** The hill framework should augment the `kanbanzai-planning` skill's conversational pattern — not replace it with a new tool or workflow. The planning skill already describes the agent's role (ask clarifying questions, reflect scope, flag sizing issues). The hill metaphor provides better vocabulary for that conversation and adds the concepts of sequencing (inverted pyramid) and stuckness detection.

**Confidence:** High
**Based on:** Finding 6, prior research Recommendation 1 (tools-first approach)
**Conditions:** If the hill language proves confusing or unnatural in practice, fall back to the existing planning conversation vocabulary. The metaphor is a tool, not a requirement.

### Recommendation 5: Defer the "Admin Assistant" Role Decision

**Recommendation:** Consistent with the prior research's Recommendation 4, do not create a new role (Admin Assistant, Shaping Clerk, etc.) until the tool shape is validated through use. The hill-based shaping conversation can be facilitated by the existing orchestrator or architect roles in the planning and design stages. If the absence of a named persona causes friction in practice, introduce one later.

**Confidence:** Medium
**Based on:** Prior research Finding 5 and Recommendation 4
**Conditions:** This is a UX question best answered with evidence. The prior research's Option A (Admin Assistant role) remains viable if conversational friction emerges.

## Relationship to Prior Research

This report extends and concretizes the prior research report `planning-admin-layer-exploration.md` (2026-05-01) in the following ways:

- **Finding 2 of the prior report** identified five missing cognitive functions (idea capture, scope negotiation, dependency surfacing, prioritisation, progress synthesis). The hill chart framework directly addresses scope negotiation (through uphill/downhill language), prioritisation (through inverted pyramid sequencing), and progress synthesis (through portfolio-hill and execution-hill tracking). Idea capture and dependency surfacing are complementary functions not directly addressed by the hill metaphor.

- **Finding 4 of the prior report** mapped pre-computer clerical roles to Kanbanzai functions. The "shaping clerk" concept in this report (Finding 7) is a direct extension — the clerk's job is to maintain the hill positions, surface stuck items, and prompt scope decomposition, which maps to the Correspondence Clerk and Schedule Clerk roles identified in the prior research.

- **Recommendations 1–4 of the prior report** (tools-first, model on clerical functions, keep in CLI/MCP initially, defer the role question) remain applicable and are reinforced by this report's findings. The hill chart framework provides a concrete shape for the "tools" that the prior research recommended building first.

- **The "named human jobs vs. decomposed functional roles" tension** (prior research Finding 5) is not resolved by this report. The hill metaphor works equally well with a single Admin Assistant role, decomposed roles, or the existing orchestrator/architect roles. The recommendation to defer the role decision stands.

## Limitations

- **No user research.** The analysis of how humans would interact with a hill-based shaping conversation is based on system analysis and the Shape Up methodology, not on direct observation of users attempting to shape work in Kanbanzai. The hill metaphor's conversational value is theoretical until tested.

- **Shape Up was designed for human-only teams.** The methodology assumes a designer and one or two programmers working in a six-week cycle with a shaping phase led by a senior person. Kanbanzai's human-AI hybrid model — where AI agents perform decomposition, implementation, and review — is a different operating environment. The hill metaphor may need adaptation for AI-mediated workflows where the "team" includes agents that don't experience uncertainty the way humans do.

- **The portfolio-level hill is speculative.** Shape Up deliberately omits cross-cycle tracking. The proposal to extend the hill metaphor to the portfolio level (plans containing batches) is a logical extension but has no proven model to reference. It may turn out that portfolio-level uncertainty is better tracked through other means (e.g., decision logs, roadmap documents).

- **No appetite mechanism in Kanbanzai.** Shape Up's hill chart depends on the appetite (fixed time, variable scope) to create the pressure that forces scope decisions. Kanbanzai currently has no appetite or time-boxing concept. Without it, hills have no right edge — scopes can drift uphill indefinitely with no pressure to reach the top. Whether the hill metaphor works without the appetite constraint is an open question.

- **The visual-to-text translation may lose meaning.** Basecamp's hill chart is a drag-and-drop visual widget. Translating this to text-mediated MCP tools may lose the spatial intuition that makes the metaphor work. The recommendation to use the hill *language* rather than attempt a visual replica mitigates this but does not eliminate the risk.

- **Scope of "shaping" in Kanbanzai is still evolving.** The plan lifecycle includes a `shaping` stage, but what happens during shaping is not yet fully specified beyond the `kanbanzai-planning` skill's conversational pattern. The hill framework proposed here may need revision as the shaping stage's responsibilities are further defined.
