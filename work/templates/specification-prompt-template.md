## Instruction: Produce a Design Specification

Read and understand the design document: {{DESIGN_DOCUMENT}}

**What a specification is:**

- A formal, precise distillation of a design document into verifiable requirements
- The authoritative reference for what must be built — not how to build it
- The basis for two downstream activities:
  - **Implementation planning** — decomposing the spec into concrete work
  - **Verification** — confirming that every requirement was implemented and that
    nothing was implemented that wasn't specified

**What to include:**

- Every functional requirement, constraint, and invariant from the design —
  stated as testable assertions
- Acceptance criteria for each requirement where applicable
- Scope boundaries — what is explicitly in scope and what is explicitly excluded
- Dependencies and assumptions

**Required sections in the specification:**

1. **Overview** — one-paragraph summary of what is being specified
2. **Scope** — what is in scope and what is explicitly excluded
3. **Functional Requirements** — numbered, testable requirements (FR-001, FR-002, ...)
4. **Non-Functional Requirements** — performance, compatibility, constraints
5. **Acceptance Criteria** — how each requirement will be verified
6. **Dependencies and Assumptions** — what must be true for this spec to be valid

**Example of a well-formed requirement:**

> **FR-003:** The `finish` tool MUST reject task completion when the parent
> feature is in `draft` status. The tool MUST return an error message that
> identifies the blocking condition and the required feature status.
>
> **Acceptance criteria:**
> - Calling `finish` on a task whose parent feature is in `draft` returns
>   an error containing "feature must be in" and the required status
> - The task status remains unchanged after the rejected call
> - A task whose parent feature is in `implementing` completes successfully

**What to exclude:**

- Implementation details, code, or technology choices
  (these belong in the implementation plan)
- Design rationale or alternatives analysis
  (these belong in the design document)
- Task breakdowns or sequencing
  (these belong in the implementation plan)

**Key property:** The specification must be complete with respect to the design —
every decision in the design document must be traceable to a requirement in the
specification, and the specification must introduce no decisions that aren't
grounded in the design.

**Output:** Write the specification to {{OUTPUT_PATH}}