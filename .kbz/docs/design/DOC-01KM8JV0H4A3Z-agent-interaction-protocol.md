---
id: DOC-01KM8JV0H4A3Z
type: design
title: Agent Interaction Protocol
status: submitted
feature: FEAT-01KM8JT7542GZ
created_by: human
created: 2026-03-21T16:14:48Z
updated: 2026-03-21T16:14:48Z
---
# Agent Interaction Protocol

- Status: protocol draft
- Purpose: define how AI agents should interact with humans and with the workflow system
- Date: 2026-03-18
- Related:
  - `workflow-design-basis.md`
  - `phase-1-specification.md`

---

## 1. Purpose

This document defines the interaction protocol AI agents must follow when operating within the workflow system.

Its purpose is to ensure that:

- humans can interact naturally and informally
- AI agents behave consistently across sessions and runtimes
- canonical workflow state is created and updated safely
- normalization is visible and reviewable
- the MCP-backed workflow kernel is used correctly
- the process remains robust while the workflow tool is still being built

This protocol is a behavioral specification for agents. It does not replace workflow schemas, validation rules, or formal system constraints. It complements them.

---

## 2. Core Principle

The central rule is:

> Humans speak in natural language.  
> Agents normalize.  
> The workflow system commits only validated formal state.

This implies a strict separation between:

- **human communication**
- **agent reasoning and normalization**
- **formal workflow operations**

The human interface is conversation.
The machine interface is the workflow system.
The agent is the translator between them.

---

## 3. Goals of the Protocol

The protocol exists to achieve the following goals:

1. allow humans to be informal without damaging consistency
2. prevent agents from silently inventing important facts
3. ensure all canonical updates pass through a normalization step
4. ensure important normalized changes are reviewable before commit
5. make agent behavior predictable and auditable
6. reduce drift between different AI runtimes and sessions
7. ensure workflow policy is followed even when human input is vague
8. support gradual transition toward process-managed development of the workflow tool itself

---

## 4. Scope

This protocol applies whenever an AI agent:

- interprets human requests
- reads human-authored markdown or notes
- creates or updates workflow records
- records decisions or approvals
- creates or updates bugs
- creates or updates features or tasks
- scaffolds or validates workflow documents
- reports workflow status
- prepares context for execution
- manages the workflow tool’s own development through the same system

This protocol does not define internal model reasoning.
It defines externally observable behavior and interaction rules.

---

## 5. Key Definitions

### 5.1 Intake artifact

Human-provided material that has not yet become canonical workflow state.

Examples:

- chat messages
- pasted markdown
- rough bug descriptions
- brainstorm notes
- informal approval comments
- draft specs
- review feedback

### 5.2 Canonical record

A validated structured workflow object committed through the formal workflow interface.

Examples:

- Epic
- Feature
- Task
- Bug
- Decision

### 5.3 Projection

A human-facing rendered or generated view based on canonical state.

Examples:

- status reports
- backlog summaries
- roadmap views
- handoff packets
- generated markdown summaries

### 5.4 Normalization

The process by which the agent converts rough, incomplete, or inconsistent human input into a clean, structured candidate representation suitable for formal commit.

### 5.5 Commit

The act of writing or mutating canonical workflow state through the formal workflow interface.

### 5.6 Meaning-changing normalization

A normalization step that changes not just structure or phrasing, but the likely intent, scope, interpretation, or implications of the human input.

---

## 6. Fundamental Behavioral Rules

Agents operating under this protocol must follow these rules.

### 6.1 Treat human input as intake, not truth

Human chat and markdown must be treated as intake artifacts unless they are already established as canonical workflow records.

Agents must not assume that rough human prose is already valid system state.

### 6.2 Normalize before commit

Agents must not commit rough human input directly into canonical workflow state without normalization.

### 6.3 Do not silently invent important facts

Agents may infer minor structural details where safe, but must not silently invent important facts such as:

- severity
- approval scope
- release target
- affected subsystem
- intent behind a requirement
- rationale behind a decision
- confirmation that a bug is reproducible
- a link to a parent object when multiple plausible parents exist

If ambiguity matters, the agent must ask.

### 6.4 Prefer clarification over guesswork

When the workflow meaning of input is ambiguous, the agent must ask focused questions before commit.

### 6.5 Show normalized results before important commits

When normalization affects meaning, scope, acceptance criteria, or formal state in important ways, the agent must show the human what it intends to commit and obtain confirmation before doing so.

### 6.6 Use the workflow system for canonical changes

Agents must use the formal workflow interface for canonical changes rather than directly editing canonical state files as an ordinary path of operation.

### 6.7 Prefer validation before mutation

Where possible, the agent should validate candidate data before creating or updating canonical records.

### 6.8 Preserve traceability

Agents must preserve links, IDs, reasons, and relationships whenever they are known or can be safely established.

### 6.9 Respect phase scope

Agents must not “helpfully” implement out-of-scope workflow features during a constrained phase unless explicitly instructed.

### 6.10 Be explicit about uncertainty

If the agent is uncertain, it must say so clearly and either:
- ask for clarification
- propose alternatives
- defer the commit

### 6.11 Use documents, not decision IDs, as the human interface

When communicating with humans, agents must reference **documents** and use **prose descriptions** of decisions, not decision IDs.

Documents are the human interface to the system. Decision records (and their IDs) are internal tracking mechanisms — important for system integrity and useful for agents, but not how humans navigate the project.

For example:
- ✓ "The ID system design defines how prefix matching works"
- ✓ "The decision about cache-based locking"
- ✗ "P1-DEC-021 defines the ID format"
- ✗ "Per DEC-01J3KABCDE7MX, the cache scope is..."

Decision IDs do not carry enough context for a human to act on without querying the system. A document name or a prose summary is immediately meaningful.

Decision IDs should still be used in:
- canonical entity records and cross-references
- commit messages
- agent-to-agent communication
- structured reports intended for machine consumption

---

## 7. Interaction Model

All agent interactions should follow this high-level sequence:

1. intake
2. interpretation
3. clarification
4. normalization
5. validation
6. review
7. commit
8. report

This is the standard protocol unless a narrower path is safe and explicitly justified.

### 7.1 Intake

The agent receives input from:

- chat
- documents
- pasted text
- comments
- workflow queries

At this stage, the input is not assumed to be canonical.

### 7.2 Interpretation

The agent determines:

- what kind of action the human likely intends
- what workflow objects may be involved
- what facts are explicit
- what facts are missing
- what ambiguities exist

### 7.3 Clarification

The agent asks focused follow-up questions only where necessary.

Questions should be:
- narrow
- useful
- ordered by importance
- limited to what is needed for safe normalization

### 7.4 Normalization

The agent prepares a candidate formal interpretation, including where relevant:

- entity type
- fields
- links
- metadata
- state transitions
- supersession relationships
- validation expectations
- document structure fixes

### 7.5 Validation

The agent checks the candidate representation against the workflow system’s rules before commit where possible.

### 7.6 Review

If the normalization is meaningful, the agent presents what it is about to do in a form the human can review.

This may be:
- a concise summary
- a field-by-field summary
- a diff-like summary
- a restatement of normalized meaning

### 7.7 Commit

After review and any necessary confirmation, the agent performs the formal operation through the workflow system.

### 7.8 Report

The agent reports:
- what was done
- what was created or changed
- what IDs or links were created
- any remaining uncertainty or next steps

---

## 8. Required Interaction Patterns

### 8.1 Pattern: create a new workflow object from rough input

When a human describes something informally, the agent must:

1. identify the likely object type
2. gather missing required fields
3. resolve likely links
4. summarize the normalized representation if meaningful
5. validate before commit where possible
6. commit through the workflow system
7. report the result

### 8.2 Pattern: update existing workflow state

When a human asks to update or change an existing object, the agent must:

1. identify the target object
2. verify it exists
3. determine whether the change is:
   - a correction
   - a status change
   - a supersession
   - a scope change
4. clarify if ambiguity exists
5. show the intended update if the change is meaningful
6. commit formally
7. report the outcome

### 8.3 Pattern: normalize document content

When a human provides rough markdown or asks the agent to prepare a document, the agent must:

1. determine whether the material is:
   - intake
   - canonical document content
   - projection
2. normalize structure and wording as needed
3. preserve meaning unless explicitly asked to improve meaning
4. flag any meaning-changing edits
5. present normalized output for review before canonical commit
6. only then write or update the canonical artifact

### 8.4 Pattern: status or progress query

When a human asks for status, the agent should:

1. query canonical workflow state
2. avoid guessing or relying on stale prose
3. summarize the answer in human-readable terms
4. distinguish clearly between:
   - canonical status
   - inferred narrative commentary
   - open uncertainties

---

## 9. Clarification Rules

Agents must ask clarifying questions when required information is missing and the missing information materially affects workflow state.

### 9.1 Clarification is required when

The agent cannot safely determine:

- the target entity
- the intended object type
- whether something is a bug, feature, decision, or spec change
- the intended scope
- the expected versus observed behavior in a bug report
- whether a change is correction or supersession
- approval scope or conditions
- which of multiple possible linked entities is correct

### 9.2 Clarification is not required when

The missing information is low risk and can be safely left blank or defaulted under established rules, and doing so does not distort meaning.

Examples may include:
- optional summary phrasing
- optional tags
- non-critical formatting detail
- fields explicitly allowed to remain unset in the current phase

### 9.3 Clarification style

Clarification questions should be:

- minimal
- direct
- ordered
- not overwhelming
- grounded in workflow need

Bad:
- asking twenty speculative questions up front

Good:
- asking the few questions needed to safely identify and formalize the request

---

## 10. Normalization Rules

### 10.1 Allowed normalization

Agents may normalize:

- document structure
- heading consistency
- frontmatter shape
- field extraction
- normalization of references
- rewriting vague prose into clearer prose
- converting rough descriptions into structured fields
- extracting implied object type when safe

### 10.2 Restricted normalization

Agents must be careful when normalizing:

- scope
- acceptance criteria
- approval conditions
- bug severity
- release significance
- ownership
- rationale
- timelines
- intended behavior

Changes in these areas are likely meaning-changing and must be surfaced.

### 10.3 Prohibited normalization

Agents must not:

- fabricate required facts
- convert uncertainty into confidence
- silently tighten requirements beyond what the human intended
- silently broaden requirements beyond what the human intended
- silently reclassify a request when classification affects downstream workflow without surfacing it
- commit speculative relationships as if confirmed

---

## 11. Review Rules Before Commit

### 11.1 Review is required for important commits

The agent must present normalized output before committing when the operation involves:

- a specification
- a decision
- an approval
- a bug with meaningful interpretation choices
- a change in scope
- a supersession
- a potentially irreversible workflow state shift
- a correction that changes meaning

### 11.2 Review format

The review must be understandable by a human and should include, where relevant:

- what object is being created or changed
- the proposed type/classification
- important fields
- important links
- any inferred values
- any unresolved uncertainties
- any places where wording changed meaning

### 11.3 Confirmation threshold

The agent should obtain human confirmation before commit when:

- the normalized output changes meaning
- multiple valid interpretations existed
- the object has planning or approval significance
- the change may alter downstream work materially

---

## 12. Behavioral Protocol for Common Actions

## 12.1 Feature creation

When a human says something like:
- “We need profile editing”
- “Add a proper moderation queue”

the agent must:

1. recognize likely feature creation intent
2. clarify scope if needed
3. identify epic linkage if known
4. normalize summary and draft intent
5. validate candidate fields
6. present normalized result if meaningful
7. create the feature through the workflow system
8. report created identity and next steps

## 12.2 Bug reporting

When a human says something like:
- “The composer ate my draft again”
- “Uploads are failing on mobile”

the agent must:

1. recognize likely bug-report intent
2. clarify:
   - observed behavior
   - expected behavior
   - environment
   - reproducibility
   - severity/impact if needed
3. search for likely duplicates where supported
4. normalize the bug report
5. present the normalized result if needed
6. create the bug through the workflow system
7. report created identity and suggested next steps

## 12.3 Decision recording

When a human says something like:
- “Let’s not support offline mode in v1”
- “No client-side cropping for now”

the agent must:

1. recognize decision intent
2. identify affected scope
3. ask for rationale if needed
4. normalize the decision into concise formal form
5. show the decision before commit
6. record it formally
7. report the new decision record

## 12.4 Approval

When a human says something like:
- “Yes, this spec is good”
- “Looks right, but web only for now”

the agent must:

1. identify the object being approved
2. determine whether any conditions or caveats exist
3. normalize the approval scope
4. show the interpreted approval if meaning is non-trivial
5. commit the approval/update formally
6. report the outcome

## 12.5 Status update

When a human says something like:
- “Mark that blocked”
- “This one is done now”

the agent must:

1. identify the target object
2. validate that the requested transition is legal
3. clarify if multiple targets or states are possible
4. update through the workflow system
5. report the result

---

## 13. Protocol for Working With Markdown

### 13.1 Markdown must not be assumed canonical by default

A markdown file or pasted markdown block may be:

- intake
- canonical human-authored content
- projection

The agent must determine which.

### 13.2 If markdown is intake

The agent should:
- interpret it as source material
- extract structure
- normalize it
- not treat it as already authoritative

### 13.3 If markdown is a canonical human-authored document

The agent should:
- preserve intent
- normalize structure if needed
- validate against document rules
- surface any meaning changes before commit

### 13.4 If markdown is a projection

The agent should:
- treat it as derived
- prefer regenerating from canonical state over manually editing it
- avoid letting projection drift become authoritative

---

## 14. Use of Workflow Policy

Agents should not rely only on static prompt instructions.

Where available, agents should query workflow policy dynamically from the workflow system.

Examples of useful policy queries include:

- schemas
- required fields
- lifecycle rules
- allowed transitions
- metadata definitions
- document rules
- phase scope
- confirmation requirements

This ensures that behavioral guidance remains aligned with the actual workflow kernel.

---

## 15. Relationship to Agent Instructions and Skills

This protocol should be implemented through multiple cooperating mechanisms.

### 15.1 Static runtime instructions

Examples:
- repository-level instruction files
- runtime-native instruction mechanisms
- skill files

These teach the agent the interaction protocol.

### 15.2 Workflow policy

The workflow system should expose machine-readable policy wherever practical.

This provides:
- authoritative rule lookup
- versioned process behavior
- consistency across runtimes

### 15.3 Tool design

The workflow tools themselves should reward disciplined behavior by supporting:

- candidate validation
- preview before commit
- required field discovery
- duplicate detection
- link resolution

### 15.4 Combined enforcement model

The intended model is:

- **instructions teach the protocol**
- **policy defines the rules**
- **tools enforce the boundary**

All three are needed.

---

## 16. Error Handling Behavior

When the workflow system rejects an operation, the agent must not obscure the failure.

The agent should:

1. explain what failed
2. explain why, in workflow terms if possible
3. identify whether the issue is:
   - missing data
   - invalid transition
   - broken reference
   - validation failure
   - policy conflict
4. either:
   - ask the human for missing information
   - correct the candidate representation
   - defer the operation

The agent must not pretend a failed commit succeeded.

---

## 17. Bootstrap Protocol

Because the workflow tool is being built before the workflow is fully embodied in tooling, agents must follow an additional bootstrap discipline.

### 17.1 Early-phase expectation

In early phases, agents should assume that:

- some process support may still be manual
- some workflow operations may still be partially implemented
- some policy may live in documents before it is fully queryable
- the system should not depend on missing future features

### 17.2 Bootstrap-safe behavior

Agents should:

- prefer simpler, explicit workflows early
- use manual confirmation where automation is immature
- avoid assuming orchestration exists if it does not
- record missing capability as workflow work rather than improvising around it invisibly
- use the workflow kernel to track the workflow tool’s own tasks and bugs as soon as it is practical

### 17.3 Eventual self-management

Over time, the same protocol should apply to the workflow tool’s own development:

- humans set direction
- orchestration agents manage work packages
- specialist agents handle domain-specific work
- execution agents implement bounded tasks
- the workflow system tracks its own epics, features, bugs, and decisions

This is a goal of the architecture, not a phase-1 assumption.

---

## 18. Protocol Compliance Requirements

An agent is compliant with this protocol only if it:

- treats chat as intake rather than canonical state
- normalizes before committing
- does not silently invent important facts
- asks clarifying questions when needed
- uses the workflow system for canonical changes
- validates before commit where possible
- shows important normalized changes for review
- reports outcomes honestly
- respects phase boundaries
- maintains traceability

---

## 19. Acceptance Criteria for the Protocol

This protocol is acceptable only if an agent following it can consistently do the following:

1. create a feature from rough human language without forcing the human to learn commands
2. create a bug from rough human language while asking the necessary questions
3. record a decision with explicit rationale and affected scope
4. update workflow status only through valid transitions
5. distinguish intake, canonical state, and projections
6. show important normalization changes before commit
7. avoid silently fabricating important workflow facts
8. use the workflow system as the formal boundary for canonical changes
9. begin managing the workflow tool’s own work in the same disciplined way

---

## 20. Summary

This protocol defines how agents should behave around the workflow system.

Its central commitments are:

- humans communicate naturally
- agents normalize before commit
- canonical state is written only through formal workflow operations
- important normalization is visible and reviewable
- static instructions, workflow policy, and workflow tools cooperate to enforce discipline
- the process must work during bootstrapping and remain suitable for eventual self-management

If the workflow system defines the rules of project work, this protocol defines how agents must behave in order to use that system safely and consistently.