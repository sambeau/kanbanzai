## Instruction: Create an Implementation Plan

**Specification:** {{SPECIFICATION_DOCUMENT}}
**Output:** {{PLAN_OUTPUT_PATH}}

**What an implementation plan is:**

- An orchestration document that decomposes a specification into assignable
  units of work for AI agents
- The purpose is to enable efficient parallel execution — not to implement
  the solution itself
- Each unit of work must be self-contained enough that an agent can begin
  without reading the entire plan

**What to include:**

- A task breakdown where each task maps to one or more specification
  requirements
- For each task:
  - **Objective** — what the task must achieve, stated precisely
  - **Specification references** — which requirements this task satisfies
  - **Input context** — files to read, documents to consult, relevant
    decisions or conventions
  - **Output artifacts** — files to create or modify, tests to write
  - **Dependencies** — which tasks must complete before this one can start
- **Interface contracts** — where two or more tasks produce code that must
  interoperate, specify the shared interface explicitly (function signatures,
  data structures, protocols) so agents can work independently against
  the same contract
- A dependency graph or execution order that identifies which tasks can
  run in parallel and which must be serialised
- Scope boundaries carried forward from the specification

**What to exclude:**

- Code or implementation logic — agents will write the code; the plan
  provides direction, not solutions
- Design rationale — this belongs in the design document
- Specification requirements verbatim — reference them, don't duplicate them

**Key properties:**

- **Traceability** — every specification requirement must appear in at least
  one task; every task must trace back to at least one requirement
- **Sufficiency** — an agent receiving a task should have enough context to
  begin work without reading unrelated tasks or reverse-engineering intent
- **Parallelism** — maximise independent tasks that can execute concurrently;
  minimise serial bottlenecks

**Skills:** Apply any relevant project skills ({{SKILLS_REFERENCE}})