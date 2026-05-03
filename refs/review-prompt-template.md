## Instruction: Review Implementation

**Subject:** {{SUBJECT_DESCRIPTION}}
**Specification:** {{SPECIFICATION_DOCUMENT}}
**Scope:** {{FILE_PATHS_OR_PACKAGES}}

**What a review is:**

- A systematic verification of an implementation against its specification
  and quality standards
- The purpose is to identify gaps, defects, and improvements — not to fix them
- Findings must be recorded in a structured review document **before** any
  fixes are attempted

**Review scope — check each of the following:**

1. **Specification completeness** — every requirement in the specification has
   a corresponding implementation
2. **Specification conformance** — every implementation element is traceable to
   a specification requirement and satisfies it
3. **Code quality** — the implementation is correct, secure, performant, and
   idiomatic for the language and project conventions
4. **Test coverage and quality** — tests exist at an appropriate level of
   coverage, are correct, and test behaviour rather than implementation details
5. **Internal documentation** — non-obvious logic, architectural decisions, and
   complex code paths are documented for future implementors
6. **User-facing documentation** — all user-facing interfaces are documented,
   including text-based interfaces (CLI help text, error messages, usage examples)
7. **Agent-facing documentation** — interfaces intended for AI agents are
   documented in locations where agents will discover them
   (e.g. AGENTS.md, skill files, tool descriptions)
8. **Workflow document currency** — project planning, progress, and status
   documents accurately reflect the current state of the implementation

**Output requirements:**

- Produce a structured review document at {{REVIEW_OUTPUT_PATH}} capturing all
  findings before making or proposing any changes
- Categorise each finding by severity: defect, gap, improvement, or nit
- Reference the specific specification requirement or quality criterion each
  finding relates to

**Example finding format:**

> **[defect] Specification requirement FR-003 not implemented**
> Location: `internal/service/task_service.go`, lines 145-160
> The `finish` function does not check the parent feature's lifecycle
> status before completing a task. FR-003 requires this validation.
> Severity: defect — functional requirement is unmet.
>
> **[improvement] Test coverage for edge case**
> Location: `internal/service/task_service_test.go`
> No test covers the case where `finish` is called on a task with
> no parent feature. While the spec doesn't require this, defensive
> handling would improve robustness.
> Severity: improvement — not a spec violation but strengthens the code.

**Note:** For comprehensive multi-agent code review with orchestration,
sub-agent dispatch, and structured review dimensions, see the
`kanbanzai-code-review` skill in `.agents/skills/`. This template is
for simpler, single-pass reviews.

**Decision authority:**

- **Implementation decisions** (how to fix, refactor, or improve code) — you may
  make these independently
- **Design decisions** (changes to requirements, interfaces, scope, or behaviour
  not covered by the specification) — raise for human consultation; do not
  resolve unilaterally