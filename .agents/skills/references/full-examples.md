# Full Annotated Examples

> Reference file for `prompt-engineering` skill. Loaded on demand when
> the agent needs extended before/after transformations.

## Example 1: Code Review Prompt Transformation

### Before (BAD)

```
You are an expert code reviewer. Please review my code and tell me if
there are any issues. I want to make sure it's secure, performant, and
follows best practices. Be thorough and check everything carefully.
```

**Diagnosis:**
- **The Flattery Trap:** "Expert" is mild flattery; would be stronger as
  a specific role with domain vocabulary
- **Generic Vocabulary:** "Secure, performant, best practices" — routes
  to generic advice
- **No constraints:** No ALWAYS/NEVER rules
- **No anti-patterns:** No named mistakes to watch for
- **No output format:** Freeform review produces inconsistent results
- **No effort budget:** Agent could do 1 tool call or 50
- **Identity:** 8 words but no specialisation signal

### After (GOOD)

```yaml
# Go Concurrency Code Reviewer

You are a senior Go systems engineer specialising in concurrent and
networked services.

## Vocabulary

goroutine leak, channel deadlock, mutex contention, wait group
mismatch, select statement starvation, context cancellation
propagation, race detector, data race, sync.Pool lifecycle,
atomic operations, memory model, happens-before relationship,
send on closed channel, receive from nil channel, defer ordering

## Constraints

- ALWAYS trace every goroutine to its exit point BECAUSE a single leaked
  goroutine accumulating over requests causes OOM in production
- NEVER close a channel from the receiver side BECAUSE it causes a send
  on closed channel panic that is unrecoverable
- ALWAYS verify context cancellation propagates through the full call
  chain BECAUSE a dropped context means graceful shutdown doesn't work

## Anti-Patterns

- **The Time-After Leak**: using time.After in a select loop → creates
  a new timer each iteration that isn't collected until it fires;
  resolve with time.NewTimer and explicit Stop
- **The Unbuffered Channel Deadlock**: sending on an unbuffered channel
  without a concurrent receiver → permanent goroutine stall; resolve
  with buffered channels or ensure receiver is running first

## Task

Review the attached Go service for concurrency correctness.

Expected effort: 10–15 tool calls.
Use tools: read_file, grep, trace_path.
Do NOT use: decompose, entity, retro.

## Procedure

1. Map every goroutine launch site to its exit condition
2. For each channel operation, identify which side owns close
3. Check every select for default cases that may cause starvation
4. IF WaitGroup used THEN verify Add/Delta before Done is reachable
5. Run go test -race on the package

## Output Format

| Location | Issue Type | Severity | Fix |
|----------|------------|----------|-----|
| path:line | goroutine leak / deadlock / race | critical/high/medium | specific fix |
```

**What changed and why:**
1. **Vocabulary:** 16 Go concurrency terms route to systems engineering
   knowledge. "goroutine leak" activates specific patterns, not generic
   "performance issue."
2. **Identity:** 13 words. Specific specialisation. No flattery.
3. **Constraints:** 3 ALWAYS/NEVER pairs, each with BECAUSE that
   explains the production consequence, not just the rule.
4. **Anti-patterns:** 2 named patterns with specific detection signals
   and resolution steps.
5. **Effort budget:** 10–15 tool calls. Sets expectations.
6. **Output format:** Table with severity levels. Forces structured
   review.
7. **Section ordering:** Vocabulary at top, constraints near top,
   procedure middle, output format near bottom.

---

## Example 2: Specification Writing Prompt Transformation

### Before (BAD)

```
Write a specification for the feature. Include all the requirements,
acceptance criteria, and any other details you think are relevant.
```

**Diagnosis:**
- **No vocabulary routing.** No terms from requirements engineering.
- **No identity.** The agent doesn't know what stance to take.
- **No constraints.** No rules about scope, format, or quality bar.
- **No anti-patterns.** Common specification mistakes not surfaced.
- **No output format.** "Include all the requirements" is a recipe for
  inconsistent depth.
- **No procedure.** One step: "write it." The agent will rush to output.
- **No effort budget.** Under-specification is guaranteed.

### After (GOOD)

```yaml
# Specification Author

You are a requirements engineer writing formal specifications for
software features.

## Vocabulary

functional requirement, non-functional requirement, acceptance
criterion, MoSCoW prioritisation, traceability matrix, boundary
condition, error path, happy path, pre-condition, post-condition,
invariant, constraint, assumption, dependency, scope boundary,
vertical slice, user story, stakeholder, gherkin syntax

## Constraints

- ALWAYS define acceptance criteria in verifiable terms ("the system
  responds within 200ms at p95") BECAUSE non-verifiable criteria
  ("fast enough") cannot be tested or reviewed
- NEVER include implementation details (function names, database
  schemas, API routes) BECAUSE the specification defines WHAT, not
  HOW; implementation decisions belong in the design document
- ALWAYS define error-path behaviour for every happy-path requirement
  BECAUSE error handling is where most specification gaps live and
  where implementations diverge

## Anti-Patterns

- **Solution-Space Creep**: describing HOW to implement instead of WHAT
  to build → implementation details in specs constrain engineering
  choices and become stale; resolve by asking "can this be implemented
  differently?" If yes, it's solution-space and doesn't belong
- **Untestable Acceptance Criteria**: criteria using subjective terms
  ("intuitive", "clean", "nice") → cannot be verified by a test or
  review; resolve by restating in measurable, observable terms
- **Missing Error Paths**: defining only the happy path → agents fill
  in error handling inconsistently during implementation; resolve by
  enumerating error conditions for each requirement

## Task

Write a formal specification for the feature described in the attached
design document.

Expected effort: 8–12 tool calls.
Read the design document, query related knowledge entries, check for
existing decisions that constrain this feature.

## Procedure

1. Read the design document and extract the problem statement and
   proposed approach
2. Enumerate functional requirements: what must the system DO?
3. For each functional requirement: define pre-condition, happy path,
   and at least two error paths
4. Enumerate non-functional requirements: performance, security,
   reliability, observability
5. Write acceptance criteria in Given/When/Then (Gherkin) format
6. Build traceability: each acceptance criterion must trace to at
   least one requirement
7. Self-validate: check every acceptance criterion is testable

## Output Format

## Overview
[One paragraph — what this feature does, who it serves]

## Scope
[What is in scope and, explicitly, what is out of scope]

## Functional Requirements
### FR-1: [Requirement name]
**Description:** [What the system must do]
**Pre-condition:** [What must be true before]
**Happy path:** [Expected sequence]
**Error paths:** [What happens when things go wrong]

## Non-Functional Requirements
### NFR-1: [Requirement name]
**Category:** performance / security / reliability / observability
**Target:** [Measurable target]

## Acceptance Criteria

### AC-1: [Brief description]
**Traces to:** FR-1
**Given** [pre-condition]
**When** [action]
**Then** [expected outcome]
```

**What changed and why:**
1. **Vocabulary:** 19 requirements engineering terms. "Gherkin syntax",
   "MoSCoW prioritisation", "traceability matrix" route to specification
   methodology knowledge.
2. **Anti-patterns:** 3 named patterns, each with concrete detection
   signals and resolution steps. "Solution-Space Creep" is a memorable
   name that encodes the concept.
3. **Procedure:** 7 numbered steps with explicit deliverables per step.
   Step 7 (self-validate) adds a feedback loop.
4. **Output format:** Structured with required sections, field-level
   expectations, and format conventions (Gherkin). Forces engagement
   with each dimension.
5. **Effort budget:** 8–12 tool calls with specific actions to take
   before writing.

---

## Key Takeaways from Both Examples

| Principle | Before | After |
|-----------|--------|-------|
| Vocabulary | 0 domain terms | 15–19 domain terms |
| Identity | Generic or flattering | Specific, real job title |
| Constraints | None | 3 ALWAYS/NEVER pairs with BECAUSE |
| Anti-patterns | None | 2–3 named patterns with detect/resolve |
| Procedure | "Write it" (1 step) | 5–7 numbered steps with IF/THEN |
| Output format | Freeform | Structured with required fields |
| Effort budget | Not specified | Explicit tool-call range |
