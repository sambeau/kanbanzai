# Conversational Guide: Talking to Kanbanzai

This guide answers the question *"what do I say to move this work forward?"* organised by where you are in the workflow, not by what system function you want to invoke.

It is written for the human side of the collaboration — the design manager who sets direction and approves gates. If you want an exhaustive list of every trigger phrase and its technical effects, see the [Trigger Reference](trigger-reference.md). If you want a conceptual overview of the methodology, see the [Workflow Overview](workflow-overview.md).

---

## Before anything else: check where you are

**The single most valuable habit when working with Kanbanzai is to check status before giving any instruction.** The system maintains precise lifecycle state for every feature and document. If you misread that state — assuming a design is approved when it's still draft, or that tasks exist when they don't — you risk triggering the wrong operation. The agent will either do something unexpected or report a confusing error.

**To check project-wide status:**
> "What's the current status of the project?"

> "What's ready to work on?"

The agent returns a dashboard: active features, their lifecycle states, document approval statuses, blocked items, and what the work queue looks like. Read it before doing anything, especially at the start of a session or after a gap.

**To check a specific feature:**
> "What's the status of [feature name]?"

> "Show me FEAT-xxx."

The agent returns: which lifecycle state the feature is in, which documents exist and whether they are approved, how many tasks exist and how many are done, and any blocking attention items.

**Why this matters in practice:** A feature can be in *specifying* state but have a draft (unapproved) specification. A feature can be in *dev-planning* state but have no tasks yet. The lifecycle state and the document approval status are two independent dimensions. Checking status shows you both at once and makes it clear what the actual next step is.

---

## Understanding the two tracks

Every feature has two things that must move forward together, but which move independently:

**Track 1 — the entity lifecycle state:** `proposed` → `designing` → `specifying` → `dev-planning` → `developing` → `reviewing` → `done`

**Track 2 — document approval status:** Each stage requires a specific document type. Each document starts as a `draft` and must be explicitly approved to open the next stage gate.

| Stage | Required document | Who approves |
|---|---|---|
| designing | Design document | You |
| specifying | Specification | You |
| dev-planning | Dev-plan | Agent (can auto-approve) or you |
| reviewing | Review report | You |

Writing a document does not change the lifecycle state. Approving a document opens the gate but does not automatically walk through it. Both tracks have to move, and you need to drive both.

---

## The lifecycle at a glance

```
proposed → designing → specifying → dev-planning → developing → reviewing → done
```

| Stage | Who leads | What gets produced | Gate |
|---|---|---|---|
| planning | You | Agreed scope | You signal readiness to proceed |
| designing | You + Agent | Approved design document | You approve the design |
| specifying | You + Agent | Approved specification | You approve the spec |
| dev-planning | Agent | Approved dev-plan + task entities | You approve the plan; agent advances feature |
| developing | Agent | Working code | All tasks reach `done` |
| reviewing | Agent + You | Review report | You approve the findings |
| done | — | — | — |

Agents cannot skip stages. If you ask for something that belongs to a later stage while an earlier gate is still closed, the agent will stop and explain what is missing.

---

## Planning: deciding what to build

You are here when you have an idea but nothing formal exists yet in the system.

### What to say

Planning is a conversation, not a command. There are no special trigger phrases — simply describe what you want to build:

> "I want to add a caching layer to the API."

> "Here's my thinking on authentication — can you give me some feedback?"

> "Should this be one feature or should we split it?"

The agent's job in planning is to ask clarifying questions, reflect your scope back to you, suggest structure, and flag when something is too large to treat as a single feature. You make all the scoping decisions.

### What happens at the end of planning

A planning conversation should end with three things agreed:
1. Scope — what is and is not included
2. Structure — is this one Feature, or a Plan containing multiple Features?
3. Your signal to proceed

Then ask the agent to create the appropriate entities:

> "Let's proceed — create a Feature entity for this."

> "Set up a Plan entity with those three features."

### Feature alone vs. Plan with Features

| Use a Feature alone when... | Use a Plan with Features when... |
|---|---|
| The work can be described in a sentence | The work has clearly independent parts |
| It would produce one design document | Different agents could work on parts in parallel |
| It can be implemented in a focused sprint | The work would produce multiple design documents |

Err toward fewer plans. A single feature does not need a Plan. A Plan with only one Feature is usually just a Feature.

### What does not happen automatically

Creating entities does not write any documents. After the entities are created, the Feature is in `designing` state and nothing else has happened yet. Design work begins when you ask for it.

---

## Designing: how it should work

You are here when a Feature entity exists and you want to establish the architectural direction.

**Check your state first:**
> "What's the status of [feature]?"

You want to see: Feature in `designing` status, no approved design document yet.

### Starting design work

> "Write a design document for [feature]."

> "Draft a design for [feature]."

**What the agent does before writing a single word:** It searches the document corpus for related prior decisions and constraints. This step is non-optional — designing without consulting prior work risks contradicting existing decisions or rebuilding something that already exists. Expect a few tool calls before the draft appears.

**What the agent produces:**
- A design document with four required sections: Problem and Motivation, Design, Alternatives Considered, and Decisions
- At least two candidate approaches analysed with trade-offs
- Every architectural decision recorded with its rationale
- The document registered as a **draft** — not yet approved

**What the agent does not do:** It does not approve the document. It does not create any other entities. It does not advance the feature's lifecycle state. It does not write a specification.

### Iterating on the design

After the agent presents the draft, you are in a design conversation. Revise freely:

> "Can you add more detail to the alternatives section?"

> "I want to use approach B instead — please revise."

> "There's an open question in section 3 — my answer is [X]."

> "The failure modes section feels thin."

The document stays in draft status throughout. Iterate until you are satisfied.

### Approving the design

When you are ready to formally close the design and move to specification:

> "Approved."

> "LGTM."

> "Looks good, let's proceed to spec."

**What happens:**
1. The agent records your approval — the document status changes from `draft` to `approved`
2. The specifying stage gate opens
3. The approval is permanently recorded

**What does not happen:** The feature does not automatically advance to `specifying` status. No specification is written. You need to ask for those things explicitly.

### Phrases to avoid in the design stage

**"Write a draft document"** — the agent doesn't know which document type. Say the type explicitly.

**"Write a design and spec"** — requests both in one go and will skip the approval gate between them. If you later want to change the design, the spec has to be rewritten. Always approve the design before asking for the spec.

**"This looks fine"** — said in passing, this is not an approval. Nothing is recorded. Say "Approved" when you mean it.

---

## Specifying: what exactly to build

You are here when a design document is approved and you want precise, testable requirements.

**Check your state first:**
> "What's the status of [feature]?"

You want to see: Feature in `specifying` status (or advancing to it). Design document approved.

### Starting specification

> "Write a specification for [feature]."

> "Draft a spec for [feature]."

> "Produce acceptance criteria for [feature]."

**Prerequisite guard:** The agent will check that an approved design document exists before writing anything. If no approved design exists, it will stop and tell you. Do not try to work around this — a specification based on an unapproved design is built on shifting ground, and every agent that implements against it inherits the instability.

**What the agent does:**
1. Reads the approved design document fully
2. Checks related specifications for consistency — shared data shapes, error contracts, or behavioural invariants with adjacent features
3. Derives numbered requirements from the design (functional and non-functional)
4. Writes testable acceptance criteria for each requirement
5. Builds a verification plan: for each acceptance criterion, a verification method
6. Registers as a draft and presents to you

**What the agent does not do:** It does not create tasks. It does not write a dev-plan. It does not advance the lifecycle automatically.

### Why the specification matters

The specification is the binding contract for implementation. Every agent that writes code does so against the spec. Every reviewer checks the result against the spec. Ambiguities or missing edge cases in the spec become bugs in the implementation — discovered later, at higher cost.

Read it carefully. When you spot vagueness, fix it now:

> "REQ-003 is too vague — can you tighten the acceptance criterion?"

> "Add a non-functional requirement for response time under load."

> "The scope section doesn't mention [X] — add it explicitly as out of scope."

### Approving the specification

> "Approved."

> "The spec looks good — approved."

**What happens:** Spec status → `approved`. Dev-planning gate opens.

### Specs for multi-feature plans

If you have a Plan with multiple Features, each Feature needs its own specification — there is no "write specs for all features in this plan" command. Do each Feature individually:

> "Write a specification for [Feature A]." → review → "Approved."

> "Write a specification for [Feature B]." → review → "Approved."

---

## Dev-planning: how to implement it

You are here when a specification is approved and work needs to be broken into implementable tasks.

**Check your state first:**
> "What's the status of [feature]?"

You want to see: Feature in `dev-planning` status. Specification approved.

### The two-part nature of dev-planning

This stage involves two operations that are easy to confuse:

| Operation | What it produces | Creates tasks? |
|---|---|---|
| Write a dev-plan | A plan *document* describing the approach | No |
| Decompose into tasks | Task *entities* in the work queue | Yes |

**The recommended phrase does both at once:**

> "Write a dev-plan and decompose [feature] into tasks."

This is the canonical trigger for the dev-planning stage. If you say only "write a dev-plan", you get the document but no tasks. If development doesn't start after dev-planning, missing tasks are the likely reason.

### If you want to do them separately

**For the document only:**
> "Write a dev-plan for [feature]."

> "Write an implementation plan for [feature]."

**For the tasks only (after the document exists):**
> "Decompose [feature] into tasks."

> "Create the task breakdown for [feature]."

### The approval gap

After you review and approve the dev-plan, **nothing happens automatically**. This is the most common point of confusion in the dev-planning stage. Approving the document opens the gate, but the feature does not advance and tasks do not become active until you say:

> "Advance [feature] to developing."

> "Start development on [feature]."

If you approve the dev-plan and nothing proceeds, this is why.

---

## Developing: building it

You are here when the feature is in `developing` status and tasks exist.

**Check your state first:**
> "What's the status of [feature]?"

You want to see: Feature in `developing` status. Tasks visible in the work queue.

### Triggering full orchestrated development

> "Orchestrate development for [feature]."

> "Run the dev-plan for [feature]."

> "Coordinate implementation of [feature]."

**What the agent does:**
1. Identifies all tasks whose dependencies are complete (the ready frontier)
2. Checks for file-scope conflicts between tasks that could run in parallel
3. Dispatches sub-agents to implement tasks simultaneously where possible
4. Waits for outcomes; compresses each completed sub-agent's output to a short summary to keep context manageable across long features
5. Verifies each task is `done` before allowing dependent tasks to proceed
6. Dispatches the next wave of now-unblocked tasks
7. On failure: one retry with updated guidance; if the second attempt also fails, pauses and escalates to you
8. Continues until all tasks are `done`

**What the agent does not do:** It does not review the code — that is a separate stage. It does not create a pull request automatically.

### When you will be interrupted

The agent will pause work and ask you a question (a checkpoint) when:

- A sub-agent encounters a genuine decision not covered by the specification — an architecture choice, an ambiguous requirement, a conflict with an existing system
- A task fails twice and automated recovery is not viable
- A spec conflict emerges during implementation that needs your direction

Answer the checkpoint directly. Work resumes once you respond.

### Checking progress

> "What's the progress on [feature]?"

> "How many tasks are done for [feature]?"

---

## Reviewing: checking it

You are here when all tasks are complete. The agent may tell you proactively: *"All N tasks for [feature] are done — ready for review."*

### Triggering code review

> "Orchestrate a code review for [feature]."

> "Run the review for [feature]."

**What the agent does:**
1. Groups changed files into coherent review units
2. Dispatches up to four specialist sub-agents in parallel:
   - **Conformance** — does the code implement every requirement in the specification?
   - **Quality** — is the code well-structured, clear, and maintainable?
   - **Security** — are there vulnerabilities, unsafe data handling, or missing validation?
   - **Testing** — is test coverage adequate? Are edge cases covered?
3. Validates that each reviewer's output includes evidence, not just verdicts
4. Deduplicates findings raised by multiple reviewers at the same location
5. Collates everything into a single report, categorised as blocking or advisory
6. Presents the report to you

### Reading the review report

**Blocking findings** must be resolved before the feature can be marked done. The agent creates rework tasks for each one.

**Advisory findings** are your call — whether to address them now, defer them to a follow-up, or accept them as known trade-offs.

When all blocking findings are resolved:

> "Approved."

> "The review looks good — close it out."

This advances the feature to `done`.

### Plan-level review

Once all features in a plan are done:

> "Review plan [X] for completion."

> "Check plan delivery status for [plan]."

This runs a conformance check across all features: are they all in terminal states? Are all specs approved? Is the documentation current with what was delivered?

---

## Approvals in depth

Approvals are the most consequential conversational action in the system. They deserve a precise understanding.

### What counts as an approval

| ✅ These are approvals | ❌ These are not |
|---|---|
| "Approved" | "This looks fine" |
| "LGTM" | "I like it" |
| "Looks good, let's proceed" | "Not bad" |
| "Let's proceed" *(when a document is under review)* | "OK" *(in passing)* |
| "Go ahead" *(when clearly about the document)* | "Sure" |

When in doubt, be explicit: **"I approve this document"** leaves no ambiguity.

### What happens when you approve

1. The document record's status changes from `draft` to `approved`
2. Any stage gate requiring that document type now passes
3. The approval is permanently recorded — it persists across all sessions

### What does not happen when you approve

- The next document is not written automatically
- The feature lifecycle state does not advance automatically (you or the agent must do this separately)
- No code is merged
- The approval cannot be reversed with a casual "actually, ignore that" in conversation — it requires formal supersession

### Being explicit about which document you're approving

In long sessions where multiple documents are being discussed, name the document:

> "I approve the design document for [feature]."

> "The specification for [feature] looks good — approved."

---

## The "plan" naming problem

The word *plan* means three different things in Kanbanzai, and these meanings collide in conversation consistently.

| What you might mean | What to say instead |
|---|---|
| The top-level coordinating workflow entity | "Create a **Plan entity** for X" |
| The implementation plan document | "Write a **dev-plan** for X" |
| An informal discussion of approach | "Let's discuss the approach for X" |

**Never say just "create a plan."** The agent will have to guess which of the three meanings applies, and it may guess wrong. The most damaging case is when "create a plan" is interpreted as "write a dev-plan document" — which skips the entity creation step and leaves you with a document but no workflow state anchoring it.

Using the word **entity** removes the ambiguity: "create a Plan entity" is always unambiguous.

---

## Common mistakes and how to fix them

### Skipping the status check

**Situation:** You return to a session and immediately say "write a spec for [feature]" without checking status first.

**What can go wrong:** The feature might not be in `specifying` state. The design might be unapproved. A spec might already exist and need revision rather than replacement.

**Fix:** Always check status first. Ten seconds of checking saves potentially significant rework.

---

### Saying "looks good" instead of "Approved"

**Situation:** You review a design in conversation and say something like "yeah that looks fine, let's move on."

**What happens:** Nothing is recorded. The stage gate is still closed. The next time you or an agent checks prerequisites, the design appears unapproved — because it is.

**Fix:** When you want to formally approve, say "Approved" explicitly. If you suspect a previous approval wasn't recorded, ask: "Did you record my approval of the [design/spec] document?" and re-approve if needed.

---

### Expecting dev-plan approval to trigger development

**Situation:** You approve the dev-plan and nothing happens.

**Why:** Approving the dev-plan document and advancing the feature to `developing` are separate operations. The approval opens the gate; you have to walk through it.

**Fix:** After approving the dev-plan, say "Advance [feature] to developing" or "Start development on [feature]."

---

### Asking for a design and spec in one message

**Situation:** "Write a design and spec for [feature]."

**What happens:** The agent may write both documents in sequence without stopping for your approval between them. The spec is based on an unapproved design. If you later revise the design, the spec has to be rewritten.

**Fix:** Always approve the design before asking for the spec. The gate exists to prevent wasted work.

---

### Using "write an implementation plan" without requesting task creation

**Situation:** "Write an implementation plan for [feature]." → You approve it → Development doesn't start.

**Why:** "Write an implementation plan" triggers the dev-plan *document* only. Task entities are a separate operation. Without tasks, there is nothing in the work queue to develop.

**Fix:** Use "Write a dev-plan **and decompose** [feature] into tasks" as your standard dev-planning phrase.

---

### Using "create a plan" ambiguously

**Situation:** "Create an implementation plan for [feature]."

**What might happen:** The agent may write a dev-plan document (possibly correct), or interpret "plan" as a Plan entity (a completely different thing), or open a planning conversation.

**Fix:** Be specific. "Write a dev-plan for [feature]" for the document. "Create a Plan entity for [scope]" for the coordinating entity.

---

## When things don't behave as expected

**The agent refuses to proceed:** Read the refusal message — it almost always names the missing prerequisite ("no approved design exists", "feature is not in specifying status"). Address the prerequisite rather than trying to reword the request.

**Nothing happens after an approval:** The approval may not have been formally recorded. Ask: "Did you record my approval of the [document]?" If not, say "Please formally approve it now."

**Development doesn't start after dev-plan approval:** The feature needs to be advanced to `developing` status. Say "Advance [feature] to developing."

**The agent did something unexpected:** Check status immediately to see the current state of all entities and documents. Address any incorrect state before proceeding — incorrect state compounds.

**You need to skip a stage for a known reason:** Be explicit about the intent and reason:

> "I want to skip the design stage for [feature] because [reason] — please override the gate and advance to specifying."

The agent will log the override reason permanently. Stage skips are tracked, not silently allowed.
