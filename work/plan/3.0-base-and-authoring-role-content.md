# Implementation Plan: Base and Authoring Role Content

| Document | Base and Authoring Role Content — Implementation Plan |
|----------|------------------------------------------------------|
| Feature  | FEAT-01KN588PF5P5Y (base-and-authoring-role-content) |
| Status   | Draft |
| Spec     | `work/spec/3.0-base-and-authoring-role-content.md` |
| Design   | `work/design/skills-system-redesign-v2.md` §4.1, §4.2, §4.4 |

---

## 1. Overview

This plan decomposes the authoring of 8 role YAML files into 5 assignable tasks. The
roles form the base and authoring layers of the Kanbanzai 3.0 role taxonomy:

- **Base layer:** `base.yaml` — project identity, hard constraints, orientation, project-wide anti-patterns
- **Authoring layer:** `architect`, `spec-author`, `implementer`, `implementer-go`, `researcher`, `documenter`
- **Coordination layer:** `orchestrator` — dispatch mechanics, workflow governance, hard constraints

The `base` role is the root of the inheritance hierarchy and must be authored first.
All other roles inherit from `base` (directly or transitively) and carry only ADDITIONAL
content — no duplication of base content. Once `base` is complete, the remaining 4 tasks
are fully independent and can execute in parallel.

**Scope boundaries (from spec):**
- This plan covers CONTENT authoring only — role YAML files with vocabulary, anti-patterns, and tool declarations
- Schema definition, parsing logic, inheritance resolution mechanics, and context assembly are out of scope
- Review roles are covered by a separate plan
- Skill files are covered by separate plans

---

## 2. Task Breakdown

### Task 1: Author `base.yaml`

**Objective:** Create the foundation role that every other role inherits. The base role
carries project identity, hard constraints, commit conventions, orientation convention,
and project-wide anti-patterns — all within a 200–300 token budget.

**Specification references:** FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007,
FR-008, NFR-001, NFR-002, NFR-003, NFR-004, NFR-005

**Input context:**
- Spec §FR-002 through §FR-005 — required content for each base role section
- Design §4.1 (`base` role definition) — project identity statement, hard constraints,
  orientation convention, project-wide anti-patterns with full detect/because/resolve text
- Design §2 (DP-3, DP-4, DP-6, DP-10) — identity constraints, anti-pattern structure,
  lean content, novelty test
- Role YAML schema: fields are `id`, `identity`, `vocabulary`, `anti_patterns`, `tools`
  (no `inherits` for base)

**Output artifacts:**
- `.kbz/roles/base.yaml`

**Dependencies:** None — this is the first task.

**Content guidance:**
- `id: base` — no `inherits` field
- `identity`: Project identity statement referencing "Kanbanzai — Git-native workflow
  system for human-AI development"
- `vocabulary`: Hard constraints ("Spec is law," "No scope creep," "Deterministic YAML
  serialisation"), commit conventions, orientation convention (`status` then `next`)
- `anti_patterns`: At least 2 entries:
  - "Flattery Prompting" — detect: superlatives in prompts; because: PRISM research on
    motivational/marketing text pattern activation; resolve: remove superlatives, use
    role identity and vocabulary
  - "Silent Scope Expansion" — detect: adding features not in spec; because: undocumented
    design decisions are expensive to discover in review; resolve: implement only what spec
    requires, stop and ask if something seems missing
- `tools`: Core tools available to all roles — `entity`, `doc`, `knowledge`, `status`,
  `next`, `finish`, `read_file`, `grep`, `search_graph`
- Total content must fit 200–300 tokens (±10% tolerance)
- Every element must pass the novelty test — no general-knowledge explanations

---

### Task 2: Author `architect.yaml` and `spec-author.yaml`

**Objective:** Create two authoring roles for the design and specification stages.
The architect role provides system decomposition expertise. The spec-author role provides
requirements engineering expertise. Both inherit from `base` and carry only additional
domain-specific content.

**Specification references:** FR-001, FR-006, FR-007, FR-008, FR-009, FR-010, NFR-001,
NFR-002, NFR-003, NFR-004, NFR-005

**Input context:**
- Spec §FR-009 — architect role requirements (identity, vocabulary, anti-patterns, stages)
- Spec §FR-010 — spec-author role requirements
- Design §4.2 — `architect` and `spec-author` definitions with full vocabulary and
  anti-pattern lists
- Design §3.3 — stage bindings for `designing`, `dev-planning`, `specifying`
- The completed `base.yaml` from Task 1 — to verify no content duplication
- MCP tool names available: `entity`, `doc`, `doc_intel`, `knowledge`, `status`, `next`,
  `handoff`, `finish`, `decompose`, `estimate`, `read_file`, `grep`, `search_graph`,
  `get_code_snippet`, `trace_call_path`, `query_graph`

**Output artifacts:**
- `.kbz/roles/architect.yaml`
- `.kbz/roles/spec-author.yaml`

**Dependencies:** Task 1 (`base.yaml` must exist to verify no content duplication and to
confirm the inheritance target)

**Content guidance for `architect.yaml`:**
- `id: architect`, `inherits: base`, `identity: "Senior software architect"`
- `vocabulary`: At least 6 terms — system decomposition, vertical slice, dependency graph,
  coupling analysis, blast radius assessment, interface boundary, separation of concerns,
  inversion of control, contract-first design, failure mode enumeration, capacity planning,
  migration strategy
- `anti_patterns`: At least 3 entries — Gold plating, Premature abstraction, Accidental
  coupling (each with detect/because/resolve)
- `tools`: Design-oriented subset — include `decompose`, `doc`, `doc_intel`, `entity`,
  `knowledge`, `status`, `next`, `finish`, `read_file`, `grep`, `search_graph`
- Stages: `designing`, `dev-planning`

**Content guidance for `spec-author.yaml`:**
- `id: spec-author`, `inherits: base`, `identity: "Senior requirements engineer"`
- `vocabulary`: At least 5 terms — acceptance criteria (Given/When/Then), requirement
  traceability, testable assertion, boundary condition, equivalence partition, specification
  completeness, ambiguity resolution, INVEST criteria, definition of done
- `anti_patterns`: At least 5 entries — Untestable requirement, Implicit assumption,
  Scope ambiguity, Over-specification, Under-specification
- `tools`: Specification-oriented subset — include `doc`, `doc_intel`, `entity`,
  `knowledge`, `status`, `next`, `finish`, `read_file`, `grep`, `search_graph`
- Stage: `specifying`

---

### Task 3: Author `implementer.yaml` and `implementer-go.yaml`

**Objective:** Create the abstract implementer base role and the Go-specific concrete
implementer role. The `implementer` role is an abstract parent that `implementer-go`
inherits from, forming a two-level inheritance chain: `implementer-go` → `implementer` →
`base`. The Go role carries language-specific vocabulary and anti-patterns.

**Specification references:** FR-001, FR-006, FR-007, FR-008, FR-011, NFR-001, NFR-002,
NFR-003, NFR-004, NFR-005

**Input context:**
- Spec §FR-011 — implementer and implementer-go requirements (identity, vocabulary,
  anti-patterns, stages, inheritance chain)
- Design §4.2 — `implementer` and `implementer-go` definitions with full vocabulary and
  anti-pattern lists
- Design §3.3 — stage bindings for `developing`
- The completed `base.yaml` from Task 1 — to verify no content duplication
- MCP tool names available for implementation work

**Output artifacts:**
- `.kbz/roles/implementer.yaml`
- `.kbz/roles/implementer-go.yaml`

**Dependencies:** Task 1 (`base.yaml` must exist)

**Content guidance for `implementer.yaml`:**
- `id: implementer`, `inherits: base`
- `identity`: A general implementer identity (e.g., "Senior software engineer")
- Abstract parent — carries shared implementation conventions that language-specific
  subtypes inherit. Vocabulary and anti-patterns should be language-agnostic implementation
  terms (if any). Can be minimal — the concrete subtypes carry the real payload.
- `tools`: Implementation-oriented subset — include `entity`, `knowledge`, `status`,
  `next`, `finish`, `read_file`, `grep`, `search_graph`, `get_code_snippet`,
  `trace_call_path`, `query_graph`

**Content guidance for `implementer-go.yaml`:**
- `id: implementer-go`, `inherits: implementer`, `identity: "Senior Go engineer"`
- `vocabulary`: At least 8 Go-specific terms — goroutine leak, interface segregation,
  error wrapping (%w), table-driven test, struct embedding, functional option pattern,
  context propagation, channel direction, sync.Mutex contention, io.Reader/io.Writer
  composition, zero-value usability, package-level encapsulation
- `anti_patterns`: At least 5 entries — God struct, Interface pollution, init() coupling,
  Naked goroutine, Error swallowing (each with Go-specific detect/because/resolve)
- `tools`: Same as `implementer` parent
- Stage: `developing`

---

### Task 4: Author `researcher.yaml` and `documenter.yaml`

**Objective:** Create two authoring roles for the research and documentation stages.
The researcher role provides technical analysis expertise. The documenter role provides
technical writing expertise. Both inherit from `base`.

**Specification references:** FR-001, FR-006, FR-007, FR-008, FR-012, FR-013, NFR-001,
NFR-002, NFR-003, NFR-004, NFR-005

**Input context:**
- Spec §FR-012 — researcher role requirements (identity, vocabulary, anti-patterns, stages)
- Spec §FR-013 — documenter role requirements
- Design §4.2 — `researcher` and `documenter` definitions with full vocabulary and
  anti-pattern lists
- Design §3.3 — stage bindings for `researching`, `documenting`
- The completed `base.yaml` from Task 1 — to verify no content duplication

**Output artifacts:**
- `.kbz/roles/researcher.yaml`
- `.kbz/roles/documenter.yaml`

**Dependencies:** Task 1 (`base.yaml` must exist)

**Content guidance for `researcher.yaml`:**
- `id: researcher`, `inherits: base`, `identity: "Senior technical analyst"`
- `vocabulary`: At least 5 terms — literature review, evidence synthesis, citation
  traceability, finding classification, confidence assessment, applicability analysis,
  counter-evidence, research gap identification
- `anti_patterns`: At least 3 entries — Cherry-picking, False equivalence, Unsupported
  generalisation (each with detect/because/resolve)
- `tools`: Research-oriented subset — include `doc`, `doc_intel`, `entity`, `knowledge`,
  `status`, `next`, `finish`, `read_file`, `grep`, `search_graph`
- Stage: `researching`

**Content guidance for `documenter.yaml`:**
- `id: documenter`, `inherits: base`, `identity: "Senior technical writer"`
- `vocabulary`: At least 5 terms — progressive disclosure, information architecture,
  cross-reference integrity, terminology consistency, audience-appropriate register,
  structural parallelism, reading order optimisation
- `anti_patterns`: At least 4 entries — Documentation-code divergence, Outdated example,
  Assumed knowledge, Documentation duplication
- `tools`: Documentation-oriented subset — include `doc`, `doc_intel`, `entity`,
  `knowledge`, `status`, `next`, `finish`, `read_file`, `grep`, `search_graph`
- Stage: `documenting`

---

### Task 5: Author `orchestrator.yaml`

**Objective:** Create the coordination role for agents that dispatch and manage other
agents. This is the most complex role — it carries categorised vocabulary across 4 domains,
at least 7 anti-patterns with research-backed BECAUSE clauses, and 3 hard constraints that
function as non-negotiable decision boundaries.

**Specification references:** FR-001, FR-006, FR-007, FR-008, FR-014, FR-015, FR-016,
FR-017, NFR-001, NFR-002, NFR-003, NFR-004, NFR-005

**Input context:**
- Spec §FR-014 — orchestrator vocabulary requirements (4 categories: dispatch mechanics,
  workflow governance, quality assessment, pattern matching)
- Spec §FR-015 — orchestrator anti-pattern requirements (7 named anti-patterns with
  specific research citations)
- Spec §FR-016 — orchestrator hard constraints (45% threshold, 4-agent saturation,
  cascade pattern)
- Spec §FR-017 — orchestrator stage associations (developing, reviewing only)
- Design §4.4 — `orchestrator` definition with full vocabulary, anti-patterns, and
  special constraints
- Design §5.4 — skill composition during review (context for how the orchestrator
  dispatches specialist sub-agents)
- The completed `base.yaml` from Task 1 — to verify no content duplication

**Output artifacts:**
- `.kbz/roles/orchestrator.yaml`

**Dependencies:** Task 1 (`base.yaml` must exist)

**Content guidance:**
- `id: orchestrator`, `inherits: base`
- `identity: "Senior engineering manager coordinating an agent team"`
- `vocabulary` organised in 4 categories (at least 10 terms total):
  - *Dispatch mechanics:* task decomposition, handoff protocol, parallel dispatch,
    conflict detection, dependency ordering, remediation routing
  - *Workflow governance:* lifecycle gate, stage prerequisite, hard constraint (ℋ),
    soft constraint (𝒮)
  - *Quality assessment:* decomposition quality, vertical slice completeness, review
    verdict
  - *Pattern matching:* sequential reasoning penalty, parallelisable task,
    orchestrator-workers parallel
- `anti_patterns`: At least 7 entries, each with detect/because/resolve:
  - Over-decomposition — splitting below useful granularity
  - Under-decomposition — monolithic tasks exceeding context budget
  - Context forwarding — dumping full context instead of scoped packets
  - Result-without-evidence — accepting output without checking for evidence
  - Reactive communication — because: Masters et al. found proactive orchestrators
    decompose 14.5× more and track dependencies 26× more
  - Premature delegation — because: Google Research found multi-agent coordination
    degrades sequential reasoning by 39–70%
  - Infinite refinement loop — because: after `max_review_cycles` the remaining issues
    are likely spec ambiguity; resolve: escalate to human checkpoint
- Hard constraints (positioned distinctly from general vocabulary):
  - 45% context utilisation threshold
  - Agent saturation at 4 for specialist panels
  - Cascade pattern
- `tools`: Orchestration-oriented subset — include `entity`, `doc`, `doc_intel`,
  `knowledge`, `status`, `next`, `handoff`, `finish`, `decompose`, `estimate`, `conflict`,
  `checkpoint`, `health`, `branch`, `worktree`, `pr`, `merge`, `read_file`, `grep`,
  `search_graph`
- Stages: `developing`, `reviewing` (not single-agent stages)

---

## 3. Dependency Graph

```
Task 1: base.yaml
  │
  ├──→ Task 2: architect.yaml + spec-author.yaml
  │
  ├──→ Task 3: implementer.yaml + implementer-go.yaml
  │
  ├──→ Task 4: researcher.yaml + documenter.yaml
  │
  └──→ Task 5: orchestrator.yaml
```

**Execution order:**
1. Task 1 executes first (serial — all others depend on it)
2. Tasks 2, 3, 4, 5 execute in parallel (no dependencies between them)

**Maximum parallelism:** 4 concurrent tasks after Task 1 completes.

---

## 4. Interface Contracts

These tasks produce content files, not code. The shared interfaces are conventions and
terminology that must be consistent across all 8 role files.

### 4.1 Role YAML Schema Contract

Every role file must use this field structure:

```yaml
id: <role-id>
inherits: <parent-role-id>    # omitted for base only
identity: "<job title under 50 tokens>"
vocabulary:
  - term 1
  - term 2
anti_patterns:
  - name: "Pattern Name"
    detect: "Observable signal"
    because: "Why this is harmful"
    resolve: "Concrete corrective action"
tools:
  - tool_name_1
  - tool_name_2
```

### 4.2 Anti-Pattern Structure Contract

Every anti-pattern across all role files must have exactly 4 fields:
- `name` — human-readable label (title case)
- `detect` — observable signal (present tense, describes what to look for)
- `because` — explanation of harm (enables generalisation to adjacent cases)
- `resolve` — concrete corrective action (imperative, specific)

The `because` field must never be a restatement of `detect`. The `resolve` field must
never be vague (no "be careful" or "consider alternatives").

### 4.3 Vocabulary Term Format Contract

Vocabulary terms across all role files must be:
- Lowercase except for proper nouns and acronyms (e.g., "OWASP," "Go")
- Domain-specific — must pass the 15-year practitioner test
- Not general-knowledge terms a junior developer would use
- Parenthetical clarifications are permitted (e.g., "error wrapping (%w)")

### 4.4 Identity Field Contract

Every `identity` field across all role files must be:
- A real job title, recognisable in the industry
- Under 50 tokens
- Free of superlatives: no "expert," "world-class," "the best," "excels"

### 4.5 No Content Duplication Contract

Child roles carry only ADDITIONAL content. No child role's `vocabulary` or `anti_patterns`
may contain entries present in its parent. The inheritance resolver merges parent and child
content additively — the file itself is the delta.

### 4.6 Tool Name Contract

The `tools` field uses exact MCP tool names. Valid tool names:

- Workflow: `entity`, `doc`, `doc_intel`, `knowledge`, `status`, `next`, `handoff`,
  `finish`, `decompose`, `estimate`, `profile`, `health`
- Git/branch: `branch`, `worktree`, `pr`, `merge`, `cleanup`
- Coordination: `conflict`, `checkpoint`, `retro`, `incident`
- Read: `read_file`, `grep`, `search_graph`, `get_code_snippet`, `trace_call_path`,
  `query_graph`
- Meta: `server_info`

---

## 5. Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| FR-001 (Common role schema) | 1, 2, 3, 4, 5 |
| FR-002 (Base — project identity) | 1 |
| FR-003 (Base — orientation convention) | 1 |
| FR-004 (Base — project-wide anti-patterns) | 1 |
| FR-005 (Base — token budget) | 1 |
| FR-006 (Identity field constraints) | 1, 2, 3, 4, 5 |
| FR-007 (Vocabulary field constraints) | 1, 2, 3, 4, 5 |
| FR-008 (Anti-pattern field structure) | 1, 2, 3, 4, 5 |
| FR-009 (Architect role content) | 2 |
| FR-010 (Spec-author role content) | 2 |
| FR-011 (Implementer and implementer-go) | 3 |
| FR-012 (Researcher role content) | 4 |
| FR-013 (Documenter role content) | 4 |
| FR-014 (Orchestrator role content) | 5 |
| FR-015 (Orchestrator anti-patterns) | 5 |
| FR-016 (Orchestrator hard constraints) | 5 |
| FR-017 (Orchestrator stage associations) | 5 |
| NFR-001 (Novelty test compliance) | 1, 2, 3, 4, 5 |
| NFR-002 (Tone and explanatory style) | 1, 2, 3, 4, 5 |
| NFR-003 (Terminology consistency) | 1, 2, 3, 4, 5 |
| NFR-004 (Lean content) | 1, 2, 3, 4, 5 |
| NFR-005 (No implementation leakage) | 1, 2, 3, 4, 5 |