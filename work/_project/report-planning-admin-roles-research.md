# Planning and Admin Roles for Kanbanzai — Research and Brainstorm

**Date:** 2026-05-01  
**Author:** sambeau (Senior Product Designer)  
**Type:** Research / Brainstorm  
**Status:** Draft

---

## Context

Kanbanzai is currently a workflow, automation, and orchestration system for human-AI collaborative software development. With B38, a planning layer was added. This report explores whether a dedicated set of planning and admin roles — and associated skills — could be added to the system to better serve the *human* side of the pipeline.

The problem being addressed is asymmetric capacity: execution bandwidth has been greatly expanded by AI agents, but the *input* end of the pipeline remains bottlenecked by human cognitive capacity. One human managing ten AI agents has 10× the execution bandwidth but the same 1× organisational bandwidth.

---

## Research: What Existing Tools Do

### Linear

Linear is the closest existing tool to this concept at the input end. Its philosophy centres on *momentum over process theatre*, which is relevant. Key features:

- **Triage / Inbox** — a capture buffer for raw ideas before they are shaped into work
- **Cycles** — time-boxed sprint planning with lightweight AI assistance
- **Projects + Roadmaps** — a top-down hierarchy that maps to the "tunnel in for detail" mental model
- **Linear Ask** — natural language querying: "what's blocked?", "what's due this week?"
- **AI Issue Summaries** — auto-summarises long threads

**Assessment:** Linear's AI is a *retrieval and summarisation* layer bolted onto a human-to-human tool. It does not organise or sequence ideas — that cognitive work remains entirely with the human. It is not designed for a human-to-AI-agent handoff.

### Trello Butler

Butler is rule-based macro automation ("when a card enters Done, move it to Archive"). It is a trigger/action system, not an intelligent assistant. Conceptually a very different thing — not relevant to this problem.

### Other Notable Tools

| Tool | Relevant Feature | Limitation |
|------|-----------------|------------|
| **Basecamp Hill Charts** | Visualises the *uncertainty reduction* phase (unknown → known → done). Captures the idea that the hardest part of planning is reducing uncertainty, not execution. | Human-only; no AI assistance in shaping |
| **Notion AI** | Good at drafting and organising freeform text | No workflow model underneath; great for individuals, weak for pipelines |
| **Height** | Attempted AI-native task decomposition | Stalled; instincts good, execution poor |
| **Asana Portfolios** | Goal-level tracking across projects | Still human-to-human; no AI agent handoff concept |

**Summary:** No existing tool is designed for a human-to-AI-agent handoff. They all assume the output of planning is a human being assigned work.

---

## The Fundamental Insight: Why Traditional Roles Existed

Looking at pre-computer organisations surfaces a useful truth:

> **Traditional organisational roles exist to distribute cognitive load across bounded human minds.**

Each role manages a specific *type* of cognition that doesn't fit in the primary actor's mental workspace:

| Role | Cognitive function outsourced |
|------|-------------------------------|
| Secretary / PA | Working memory — holds context the executive can't keep in their head |
| Clerk | Long-term storage — files things so they are retrievable later |
| Librarian | Taxonomy and discovery — makes accumulated knowledge findable |
| Project Manager | Temporal and dependency reasoning — what order, how long, what blocks what |
| Assistant Product Manager | Goal alignment — are we building the right things in the right sequence? |

The *number* of people in a traditional organisation roughly tracks the cognitive surface area of the problem. As the problem grows, you hire people whose job is essentially to carry extra context.

**For AI teams:** AI agents compress execution dramatically, but the input end of the pipe is still bottlenecked by human cognitive capacity. The gap is not execution — it is the *shaping* of ideas into work the agents can act on.

### Is there a "fundamental context size"?

The pattern of traditional roles may reflect a real constraint in human working memory (~7 items, per Miller's Law) and attention (serial, not parallel). The role structure of pre-computer organisations can be read as a distributed system for managing a problem space that exceeds any single mind's context window. With AI agents, we can replicate this distribution — but the interface to the human must still match human cognition patterns.

---

## The Conceptual Gap Being Addressed

This is not a project management tool. It is something closer to a **cognitive prosthetic** for the human at the top of the pipeline.

The existing Kanbanzai execution pipeline (orchestrate → implement → review) already works. What is needed is a *pre-pipeline* layer that:

1. **Catches ideas when they occur** — not only when the human sits down to work
2. **Holds them without judgement** — raw, incomplete, half-formed
3. **Helps the human shape them** — turns fuzzy intentions into something the PM agent can take over
4. **Sequences and estimates** — what order, roughly how hard, what depends on what
5. **Surfaces what matters** — given everything in the backlog, what should be the focus *now*?

The mental model: **a thinking partner that sits between the human's brain and the Kanbanzai pipeline.**

---

## Proposed Conceptual Model: Three Layers

```
Human Brain
     ↓
┌─────────────────────────────────────────┐
│  CAPTURE LAYER                          │
│  • Frictionless idea inbox              │
│  • Voice, text, quick notes             │
│  • No structure required at input       │
│  • "Park it, I'll deal with it later"   │
└─────────────────────────────────────────┘
     ↓
┌─────────────────────────────────────────┐
│  SHAPING LAYER                          │
│  • AI asks clarifying questions         │
│  • Structures fuzzy ideas into plans    │
│  • Estimates effort, surfaces deps      │
│  • Produces a "brief" not a spec        │
│  • Roadmap view, priority assist        │
└─────────────────────────────────────────┘
     ↓
┌─────────────────────────────────────────┐
│  KANBANZAI PIPELINE (existing)          │
│  • Design → Spec → Dev-plan → Execute   │
└─────────────────────────────────────────┘
```

The **Shaping Layer** is conversational, not form-based. The human does not fill in fields — they write or speak naturally to an AI that asks the right questions and does the structural work for them.

---

## Three Core Cognitive Functions

Rather than mapping directly to job titles (which carry legacy constraints), the fundamental cognitive needs distil to three functions:

### 1. Memory
*"I remember things so you don't have to."*

- Captures ideas the moment they occur, regardless of context
- Persists them faithfully without requiring the human to organise them first
- Reminds the human at the right moment ("you had an idea about X three weeks ago — is it still relevant?")
- Maps to: secretary, clerk

### 2. Structure
*"I help you think more clearly."*

- Takes a vague intention and turns it into a shaped plan
- Asks the questions the human hasn't thought to ask ("what does done look like?", "who benefits from this?")
- Identifies when something is actually two separate ideas
- Maps to: assistant product manager

### 3. Sequence
*"I figure out what order things should happen in."*

- Dependency analysis: "Plan B assumes Plan A is done"
- Effort estimation: "Based on similar work, this is roughly a two-week effort"
- Priority assist: "Given your stated goals, this seems more important than that"
- Roadmap generation: "Here's what the next three months looks like if you approve these plans"
- Maps to: project manager

---

## UI Considerations

A separate UI tool is the right direction. The Kanbanzai MCP interface is designed for agents — structured, precise, and tool-call driven. The capture and shaping interface needs to be designed for *humans*:

| Interface Mode | Rationale |
|---------------|-----------|
| **Conversational first** | Primary interaction is natural language. The AI extracts structure from what the human *says*, not from forms. Mobile-friendly, potentially voice-enabled. |
| **Document-centric output** | Humans trust documents, not databases. Output should be a readable brief or one-pager the human can read, edit, and approve — not a database record. |
| **Spatial roadmap view** | Visual representation where position carries meaning. Timeline + importance grid, not a sortable table. The brain processes spatial layout faster than text. |
| **Checklist output** | Every plan should surface a checklist of "what needs to be true before this starts." Checklists make completion concrete and visible. |
| **Gentle reminders, not alerts** | Surface things at appropriate moments — not interrupts. "You have 3 unreviewed ideas from last week" rather than a badge count. |

### The Basecamp Hill Chart Analogy

Basecamp's Hill Chart is a useful mental model for the Shaping Layer. The hardest part isn't execution — it is the *uncertainty reduction* phase, moving from "I have an idea" to "I know what I'm building." That is where most human cognitive effort goes, and where a thinking partner adds the most value.

---

## The Greenfield Advantage

Why build this rather than extending Linear, Notion, or similar?

### 1. AI-agent native handoff
Existing tools model a *human receiving work*. The data they produce (a card, an issue) is designed to be read by a person. The handoff to a Kanbanzai agent needs different packaging: context, constraints, success criteria, and the reasoning behind the decision. That is a fundamentally different output format.

### 2. Closed-loop feedback
Owning both ends of the pipeline means the planning tool can learn from execution: "Plans of this type tend to take 3× the initial estimate." "This kind of feature has high review rework — perhaps the spec needs more time." No existing tool can do this because they don't have execution data.

### 3. Small team at large scale
Existing tools optimise for teams of 10–100 humans. The target here is 1–5 humans managing N AI agents. The organisational complexity is not *people coordination* — it is *cognitive load management*. That is a different optimisation target entirely.

### 4. No political overhead
Traditional tools have elaborate features for visibility, ownership, and accountability because those are human social problems. When agents do the work, those problems disappear. The tool can be dramatically simpler and faster.

### 5. Radical interface freedom
Starting greenfield means no constraint of "it must look like Jira but better." The primary interaction could be a conversation, voice, a daily digest, or a mobile quick-capture. The question becomes: *what interface does the human brain actually prefer for this cognitive task?*

---

## Roles: Known Titles vs. New Abstractions

**Arguments for known titles:** Clear mental model, easy to explain, known responsibilities.

**Arguments for new abstractions:** AI does not have the same constraints as humans (one AI can be secretary *and* librarian simultaneously). Known titles may be unnecessarily limiting.

**Recommendation:** Use known titles as *metaphors* for communication, but design actual functionality around cognitive needs, not job descriptions. The three core functions (Memory, Structure, Sequence) may be the right abstraction — implementable as one unified assistant or as distinct specialist agents.

---

## System Character

The system being envisaged is more of an **admin assistant to the human** than a project manager. The project manager role in Kanbanzai (an AI agent with that role) would not change. What is needed is a human-friendly AI assistant that helps form raw ideas into something the project manager can take over.

Traditional analogues:
- **Secretary / PA** — organisational skills, filing, to-do lists, reminders
- **Librarian** — cataloguing, filing, making things discoverable  
- **Project Manager** — estimating, dependency analysis, plan ordering, development priorities
- **Assistant Product Manager** — keeping track of product and design priorities, roadmap

---

## Open Questions

These are the most important questions to resolve before moving to design:

1. **Where does capture happen?** If the goal is to catch ideas the moment they occur, the capture interface needs to be ambient — mobile app, voice, browser extension, email forward. What is the right primary channel?

2. **What is the human's relationship with the AI?** Is this a *tool* the human directs, or a *partner* that has its own perspective and pushes back? The "thinking partner" model implies the latter is more valuable, but requires trust.

3. **How much structure does the Shaping Layer impose?** There is tension between frictionless capture and structured output for Kanbanzai. Too much friction at input and ideas don't get captured. Too little structure and the pipeline can't act on them.

4. **Is this one role or several?** A single "Planning Assistant" that does all three functions, or separate agents (Capture Bot, Estimator, Roadmap Agent)? A single agent is simpler and more conversational; multiple agents are more focused and easier to trust.

5. **What does "done" look like at the Shaping Layer?** What is the minimum output needed to hand off to the Kanbanzai PM? This is the API between the new system and the existing one — worth defining early.

6. **Is this a separate product or a Kanbanzai extension?** Given the different interaction paradigm (conversational, human-facing), a separate UI tool with Kanbanzai integration may be more appropriate than extending the MCP interface.

---

## Next Steps (Suggested)

- [ ] Decide whether this becomes a formal plan in Kanbanzai or remains a research thread
- [ ] Explore the "minimum viable brief" — what does the Shaping Layer need to produce for the Kanbanzai PM to take over?
- [ ] Sketch a capture interface concept (mobile/voice/web)
- [ ] Identify whether the three cognitive functions (Memory, Structure, Sequence) map cleanly to distinct Kanbanzai roles and skills, or require new entity types
- [ ] Research voice-first capture interfaces and their adoption patterns
