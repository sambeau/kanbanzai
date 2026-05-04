# Spec Validator Rubrics: S5 and S7

This file defines concrete rubrics for the two spec-validator checks that rely on LLM
classification rather than structural pattern matching. These rubrics MUST be consulted
by the spec-validator agent when evaluating S5 (testable acceptance criteria) and S7
(implementation instruction detection). Verdicts that cannot be clearly classified as
pass or fail against these rubrics MUST be escalated.

---

## S5: Every Acceptance Criterion Is Testable

**Check definition (from spec-validator role):**
Every acceptance criterion is testable — no subjective language without an observable outcome.

**Classification:** Blocking.

### Pass Definition

An acceptance criterion passes S5 when a tester of ordinary skill, given only the
criterion text, can determine whether the described behaviour was observed. This means:

1. **Observable behaviour is stated.** The criterion describes a system output, a state
   change, a returned value, or a logged event — something external to the system that a
   tester can witness.
2. **Conditions are explicit.** The criterion specifies the preconditions (given), the
   action (when), and the expected outcome (then) — or uses equivalent deterministic
   language. The outcome must be verifiable.
3. **No undefined or subjective terms.** "Good," "fast," "intuitive," "nice," "robust" do
   not appear without a measurable anchor (e.g., "responds within 200 ms" is fine;
   "responds quickly" is not). "Should" is acceptable only when paired with an observable
   outcome — "the system should log a warning" is testable because warnings are
   observable.

The AC does NOT need to specify the exact test procedure (unit vs. integration vs.
manual). It only needs to make the pass/fail condition unambiguous.

#### Positive Examples (Pass)

1. **AC-001 from B31-F1 spec** (merge gate review report):
   > Given a feature in `reviewing` status with no registered report documents, when
   > `merge(action: execute)` is called without `override: true`, then the merge is
   > rejected and the error message contains the feature ID, a reference to `reviewing`
   > status, resolution steps, and a statement that the gate cannot be bypassed.

   **Why this passes:** Explicit precondition (feature in reviewing, no reports),
   explicit action (merge called), explicit observable outcome (merge rejected, error
   message contains specific substrings). A tester can set up these conditions and
   verify the result.

2. **AC-101 from B32-F1 spec** (guide concept enrichment):
   > Given any indexed document, when `doc_intel(action: "guide", id: "<doc-id>")` is
   > called, then the response contains a `concepts_suggested` field that is a JSON
   > array (possibly empty) and is never absent.

   **Why this passes:** Observable outcome is a field in a JSON response — trivially
   verifiable. Precondition and action are explicit.

3. **AC-007 from B31-F1 spec** (fail-open on doc service error):
   > Given the document service returns an error while the gate is evaluating, when
   > `merge(action: execute)` is called, then the gate passes, the merge proceeds, and
   > a warning is logged.

   **Why this passes:** Observable outcomes include a logged warning (verifiable via
   log inspection) and merge proceeding (verifiable via entity state). All three
   outcomes are independently testable.

#### Negative Examples (Fail)

1. **"The system should handle errors gracefully."**

   **Why this fails:** "Gracefully" is subjective — it has no observable outcome. What
   does graceful handling look like? A logged error? A user-facing message? A retry?
   Without specifying which, a tester cannot determine pass/fail.

2. **"The UI should be responsive and intuitive."**

   **Why this fails:** "Responsive" is ambiguous without a time bound (200 ms? 2 s?).
   "Intuitive" is entirely subjective — different testers will have different opinions.
   No observable outcome is defined.

3. **"Performance must be good enough for production use."**

   **Why this fails:** "Good enough" and "production use" are undefined thresholds.
   A tester cannot know what metrics to measure or what values constitute pass/fail.

### Borderline → Escalate Pattern

Escalate (flag as `borderline → escalate`) when:

- **Implicit observability.** The criterion describes an internal state change that
  _could_ be observed indirectly (e.g., "the cache is invalidated") but does not state
  _how_ to observe it. A tester would need domain knowledge to infer the observable
  signal. Example: "The feature flag state is updated" — does this mean a config file
  changes? A database row? An API response? If the observable mechanism is not stated,
  escalate.

- **Collective testability.** A criterion references a set of behaviours that are
  individually testable but large enough that "pass" requires enumerating them all.
  Example: "All error paths return an appropriate HTTP status code." If "appropriate" is
  not mapped per error path, escalate — the criterion is testable in principle but not
  in practice without the mapping.

- **Benchmarked but unanchored.** A criterion specifies a number (e.g., "≤ 10 ms
  overhead") but does not specify the measurement conditions (hardware, load, sample
  size). The criterion _looks_ testable but a tester cannot reproduce the measurement.
  Escalate with a note asking for measurement conditions.

---

## S7: No Requirement Is a Disguised Implementation Instruction

**Check definition (from spec-validator role):**
No requirement is a disguised implementation instruction — prescribing internal data
structures, algorithms, or APIs rather than observable behaviour.

**Classification:** Non-blocking.

### Pass Definition

A requirement (or acceptance criterion) passes S7 when it describes WHAT the system must
do, not HOW to build it. Specifically:

1. **No internal data structures.** The requirement does not name Go structs, database
   schemas, file formats, or in-memory data layouts. ("The system must store a list of
   pending approvals" is fine; "The system must use a `PendingApprovalQueue` struct with
   a `sync.Mutex`" is not.)
2. **No algorithms.** The requirement does not prescribe a specific algorithm, loop
   structure, or computational approach. ("Results must be sorted by creation date" is
   fine; "Results must be sorted using quicksort on the `CreatedAt` field" is not.)
3. **No API design.** The requirement does not prescribe specific function signatures,
   method names, or parameter types. References to existing Kanbanzai tool signatures
   (like `merge(action: "execute")`) are acceptable when they describe user-visible
   behaviour. Describing a new internal helper function signature is not.

A requirement MAY reference existing system interfaces (MCP tool names, CLI commands)
when describing observable behaviour — those are part of the system's external contract.
It MUST NOT prescribe internal implementation details of those interfaces.

#### Positive Examples (Pass)

1. **FR-001 from B31-F1 spec** (gate activation):
   > The `ReviewReportExistsGate` MUST be evaluated when `merge(action: execute)` is
   > called and the target feature's current lifecycle status is `reviewing`.

   **Why this passes:** Describes WHEN the system must perform a behaviour.
   `ReviewReportExistsGate` is a named concern from the design doc — this requirement
   says it must be evaluated under specific conditions. It does not say how to
   implement the gate (no struct names, no algorithm).

2. **FR-003 from B31-F1 spec** (gate pass condition):
   > The gate MUST return `Pass` when at least one document of type `report` is
   > registered with an `owner` matching the target feature ID, regardless of that
   > document's status.

   **Why this passes:** Describes the observable outcome (gate returns Pass) and the
   conditions (document exists, type=report, owner=feature). Does not prescribe how
   to query documents or what data structure holds the result.

3. **REQ-101 from B32-F1 spec** (concepts_suggested field):
   > `doc_intel(action: "guide")` must include a `concepts_suggested` array in its
   > response. The field must always be present (never absent, never null). It may be
   > an empty array when no concepts can be derived.

   **Why this passes:** Describes the external contract — a field in an API response.
   "Array" is a JSON type, not a Go type. The requirement does not say
   `[]ConceptSuggestion` or prescribe serialization.

#### Negative Examples (Fail)

1. **"The system must use a `sync.RWMutex` to protect concurrent access to the entity
   cache."**

   **Why this fails:** Prescribes a specific Go synchronization primitive. This is an
   implementation decision — the spec should state "concurrent access to the entity
   cache must be safe" and leave the mechanism to the dev-plan.

2. **"The `merge.Execute` function must accept a `context.Context` as its first
   parameter and return `(*MergeResult, error)`."**

   **Why this fails:** Prescribes a specific function signature. This is Go API design,
   not observable behaviour. The spec should state what merge does, not its function
   signature.

3. **"Validation results must be stored in a `ValidatorResult` table with columns
   `feature_id VARCHAR(64)`, `check_id VARCHAR(16)`, and `passed BOOLEAN`."**

   **Why this fails:** Prescribes a database schema (table name, column names, column
   types). The spec should state what validation results are persisted and what they
   contain — not how the database is structured.

### Borderline → Escalate Pattern

Escalate (flag as `borderline → escalate`) when:

- **Naming is design-authorised.** A requirement references a name that appears in the
  parent design document. Example: "The `NonBypassableBlockingFailures` helper must..."
  — if this name was coined in the design, using it in the spec is traceability, not
  implementation prescription. But if the name was coined in the spec itself, it IS
  implementation prescription. Escalate when you cannot determine the name's origin.

- **Structural necessity.** A requirement describes an internal interface between two
  subsystems that the design explicitly calls out as an architectural boundary.
  Example: "The merge tool must call `DocService.ListDocuments`" — this prescribes an
  internal API but may be the design's explicit architectural seam. Escalate with
  reference to the design section that authorises the boundary.

- **Data shape vs. data structure.** A requirement describes the shape of data (e.g.,
  "each entry must have `section_path`, `section_title`, and `suggested_concepts`
  fields") — this is a data contract, not an implementation instruction, because it
  describes the external API response. But if the requirement says "must be a Go struct
  with json tags," that crosses the line. When the distinction is blurry (e.g.,
  describing a JSON field that happens to match an internal type), escalate.
