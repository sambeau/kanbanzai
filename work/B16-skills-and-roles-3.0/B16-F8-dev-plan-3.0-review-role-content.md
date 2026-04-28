# Implementation Plan: Review Role Content

| Document | Review Role Content — Implementation Plan |
|----------|------------------------------------------|
| Feature  | FEAT-01KN588PFG6GY (review-role-content) |
| Status   | Draft |
| Spec     | `work/spec/3.0-review-role-content.md` |
| Design   | `work/design/skills-system-redesign-v2.md` §3.1, §4.3 |

---

## 1. Overview

This plan decomposes the authoring of 5 review role YAML files into 3 assignable tasks.
The roles form an inheritance hierarchy where `reviewer` provides the shared review
identity, vocabulary, and anti-patterns, and 4 specialist roles inherit from it and add
domain-specific content:

- **Base review role:** `reviewer.yaml` — foundational review vocabulary, shared anti-patterns
- **Specialist reviewers:** `reviewer-conformance`, `reviewer-quality`, `reviewer-security`, `reviewer-testing`

The inheritance chain is: specialist → `reviewer` → `base`. The `reviewer` role must be
authored first because all specialists inherit from it and must verify no content
duplication against it. Once `reviewer` is complete, the 4 specialists are grouped into
2 parallel tasks for efficient dispatch.

The composition model is central to this plan: the same `review-code` skill (defined in
a separate plan) combined with different reviewer roles produces different expertise lenses
on the same code. Role files carry identity, vocabulary, and anti-patterns — never
procedures, checklists, or output formats. Those belong in skills.

**Scope boundaries (from spec):**
- This plan covers CONTENT authoring only — review role YAML files
- Schema definition, parsing, inheritance resolution mechanics, and context assembly are out of scope
- Base and authoring roles (`base`, `architect`, etc.) are covered by a separate plan
- Skill files (`review-code`, `orchestrate-review`, `review-plan`) are covered by a separate plan
- All 5 roles are associated with the `reviewing` stage only

**Key constraint:** Specialist roles carry ADDITIONAL vocabulary and anti-patterns that
MERGE with the parent's during inheritance resolution. No duplication of parent content
in child files. Each child file is a delta — the assembled context is the union.

---

## 2. Task Breakdown

### Task 1: Author `reviewer.yaml`

**Objective:** Create the base review role that all specialist reviewers inherit. This role
carries the shared review identity, foundational review vocabulary, and the cross-cutting
review anti-patterns that apply to every review dimension. Content must be intentionally
minimal — foundational terms only — to leave room for specialist vocabulary without
exceeding lean content constraints.

**Specification references:** FR-001, FR-002, FR-003, FR-012, FR-014, NFR-001, NFR-002,
NFR-003, NFR-004, NFR-005, NFR-006

**Input context:**
- Spec §FR-001 — common schema requirements (all 6 fields required, `inherits: base`)
- Spec §FR-002 — reviewer identity and vocabulary (6 foundational terms)
- Spec §FR-003 — reviewer anti-patterns (4 named patterns with research citations)
- Spec §FR-012 — tool subset for review roles (read-oriented, no `decompose`/`estimate`)
- Spec §FR-014 — stage association (`reviewing` only)
- Design §4.3 — `reviewer` base role definition with full vocabulary and anti-pattern text
- Design §3.1 — role YAML schema example (reviewer-security as reference)
- The completed `base.yaml` from the Base and Authoring Role Content plan — to verify
  inheritance target exists and no content duplication
- Role YAML schema: `id`, `inherits`, `identity`, `vocabulary`, `anti_patterns`, `tools`

**Output artifacts:**
- `.kbz/roles/reviewer.yaml`

**Dependencies:** The `base.yaml` role from the Base and Authoring Role Content plan must
exist (it is the inheritance target for `reviewer`).

**Content guidance:**
- `id: reviewer`, `inherits: base`, `identity: "Senior code reviewer"`
- `vocabulary`: 6 foundational review terms (5–15 range per NFR-004):
  - finding classification
  - evidence-backed verdict
  - review dimension
  - blocking vs non-blocking
  - severity assessment
  - remediation recommendation
- `anti_patterns`: At least 4 entries, each with name/detect/because/resolve:
  - "Rubber-stamp approval" — because: MAST FM-3.1 identifies LLM sycophancy making
    approval the path of least resistance; this is the #1 quality failure in multi-agent
    systems. Resolve: require per-dimension evidence or at least one finding for clearance
  - "Dimension bleed" — detect: one dimension's result influencing another's verdict.
    Resolve: evaluate each dimension in isolation
  - "Prose commentary" — detect: qualitative prose replacing structured findings.
    Resolve: replace every qualitative statement with a structured finding entry
  - "Severity inflation" — detect: disproportionate blocking/critical classifications.
    Resolve: re-check each blocking finding against classification criteria
- `tools`: Read-oriented review subset — must include `entity`, `doc_intel`, `knowledge`,
  `read_file`, `grep`, `search_graph`. Must NOT include `decompose` or `estimate`.
  May also include `status`, `get_code_snippet`, `trace_call_path`, `query_graph`
- Stage: `reviewing` only (not designing, specifying, developing, researching, documenting)
- Content must not contain procedures, checklists, or output format definitions — those
  belong in the `review-code` skill
- Every `because` clause must explain WHY, not restate the `detect` signal

---

### Task 2: Author `reviewer-conformance.yaml` and `reviewer-quality.yaml`

**Objective:** Create two specialist reviewer roles that inherit from `reviewer`. The
conformance reviewer adds requirements verification vocabulary and anti-patterns. The
quality reviewer adds code quality analysis vocabulary and anti-patterns. Each role carries
only ADDITIONAL content — no duplication of terms or anti-patterns from the parent
`reviewer` role.

**Specification references:** FR-001, FR-004, FR-005, FR-006, FR-007, FR-012, FR-013,
FR-014, NFR-001, NFR-002, NFR-003, NFR-004, NFR-005, NFR-006

**Input context:**
- Spec §FR-004 — reviewer-conformance identity and additional vocabulary (6 terms)
- Spec §FR-005 — reviewer-conformance additional anti-patterns (3 named patterns)
- Spec §FR-006 — reviewer-quality identity and additional vocabulary (8+ terms)
- Spec §FR-007 — reviewer-quality additional anti-patterns (3 named patterns)
- Spec §FR-012 — tool subset (same read-oriented subset as parent)
- Spec §FR-013 — no content duplication across inheritance
- Design §4.3 — `reviewer-conformance` and `reviewer-quality` definitions
- The completed `reviewer.yaml` from Task 1 — to verify no content duplication against
  the parent's vocabulary and anti-pattern lists

**Output artifacts:**
- `.kbz/roles/reviewer-conformance.yaml`
- `.kbz/roles/reviewer-quality.yaml`

**Dependencies:** Task 1 (`reviewer.yaml` must exist to verify no content duplication)

**Content guidance for `reviewer-conformance.yaml`:**
- `id: reviewer-conformance`, `inherits: reviewer`
- `identity: "Senior requirements verification engineer"`
- `vocabulary`: At least 6 ADDITIONAL terms (not duplicating parent):
  - acceptance criteria traceability
  - spec requirement mapping
  - gap analysis
  - criterion-by-criterion verification
  - deviation classification
  - conformance matrix
- `anti_patterns`: At least 3 ADDITIONAL entries (not duplicating parent):
  - "Assumed conformance" — detect: marking pass without tracing to a requirement
  - "Partial verification" — detect: checking some acceptance criteria but not all
  - "Phantom requirement" — detect: finding issues against requirements not stated in spec
- `tools`: Same read-oriented subset as `reviewer`
- Stage: `reviewing`
- Verify: no vocabulary term appears in parent `reviewer` vocabulary list
- Verify: no anti-pattern name appears in parent `reviewer` anti-patterns list

**Content guidance for `reviewer-quality.yaml`:**
- `id: reviewer-quality`, `inherits: reviewer`
- `identity: "Senior software quality engineer"`
- `vocabulary`: At least 8 ADDITIONAL terms (not duplicating parent):
  - cyclomatic complexity
  - error handling chain
  - defensive copying
  - invariant assertion
  - contract violation
  - resource lifecycle (open/close pairing)
  - naming consistency
  - package cohesion
  - dead code detection
- `anti_patterns`: At least 3 ADDITIONAL entries (not duplicating parent):
  - "Style-as-defect" — detect: flagging style preferences as quality issues
  - "Nitpick escalation" — detect: trivial issues marked as blocking
  - "Improvement suggestion disguised as defect" — detect: recommending enhancements
    in the guise of defect findings
- `tools`: Same read-oriented subset as `reviewer`
- Stage: `reviewing`
- Verify: no vocabulary term appears in parent `reviewer` vocabulary list
- Verify: no anti-pattern name appears in parent `reviewer` anti-patterns list

---

### Task 3: Author `reviewer-security.yaml` and `reviewer-testing.yaml`

**Objective:** Create two specialist reviewer roles that inherit from `reviewer`. The
security reviewer adds application security vocabulary (OWASP, STRIDE, CWE, CVSS) and
security-specific anti-patterns. The testing reviewer adds test evaluation vocabulary and
testing-specific anti-patterns. Each role carries only ADDITIONAL content — no duplication
of terms or anti-patterns from the parent `reviewer` role.

**Specification references:** FR-001, FR-008, FR-009, FR-010, FR-011, FR-012, FR-013,
FR-014, NFR-001, NFR-002, NFR-003, NFR-004, NFR-005, NFR-006

**Input context:**
- Spec §FR-008 — reviewer-security identity and additional vocabulary (12+ terms)
- Spec §FR-009 — reviewer-security additional anti-patterns (5 named patterns with
  specific detection criteria and resolution mechanisms)
- Spec §FR-010 — reviewer-testing identity and additional vocabulary (8+ terms)
- Spec §FR-011 — reviewer-testing additional anti-patterns (5 named patterns)
- Spec §FR-012 — tool subset (same read-oriented subset as parent)
- Spec §FR-013 — no content duplication across inheritance
- Design §4.3 — `reviewer-security` and `reviewer-testing` definitions
- Design §3.1 — full `reviewer-security` YAML example (the canonical example in the
  design document)
- The completed `reviewer.yaml` from Task 1 — to verify no content duplication
- Spec assumption 4 — the security-specific "Severity Inflation" anti-pattern is distinct
  from the parent's general "Severity inflation" because it carries CVSS-specific detection
  signals and resolution

**Output artifacts:**
- `.kbz/roles/reviewer-security.yaml`
- `.kbz/roles/reviewer-testing.yaml`

**Dependencies:** Task 1 (`reviewer.yaml` must exist to verify no content duplication)

**Content guidance for `reviewer-security.yaml`:**
- `id: reviewer-security`, `inherits: reviewer`
- `identity: "Senior application security engineer"`
- `vocabulary`: At least 12 ADDITIONAL terms (not duplicating parent):
  - OWASP Top 10 (2021 edition)
  - STRIDE threat model
  - CWE weakness classification
  - CVSS v3.1 scoring
  - input validation boundary
  - authentication flow analysis
  - authorization bypass pattern
  - secrets detection
  - SQL injection via string concatenation
  - insecure direct object reference (IDOR)
  - cross-site request forgery (CSRF)
  - security header configuration (CSP, HSTS)
  - dependency vulnerability scanning
  - least privilege principle
  - defense-in-depth layering
- `anti_patterns`: At least 5 ADDITIONAL entries (not duplicating parent):
  - "Checkbox Compliance" — detect: evaluating against a checklist without understanding
    the threat model. Resolve: start from the threat model, not the checklist
  - "Scope Creep into Exploitation" — detect: attempting to exploit vulnerabilities rather
    than identify and classify them. Because: the reviewer's role is assess and report,
    not prove exploitability (that is a pentester's role)
  - "Severity Inflation" (security-specific, distinct from parent's general version) —
    detect: >30% critical findings. Resolve: apply CVSS v3.1 scoring
  - "Framework Trust" — detect: assuming framework defaults are secure without
    verification. Because: custom configurations and escape hatches create gaps in defaults
  - "Boundary Blindness" — detect: reviewing only changed code without examining trust
    boundaries (user→server, service→service, internal→external)
- `tools`: Same read-oriented subset as `reviewer`
- Stage: `reviewing`

**Content guidance for `reviewer-testing.yaml`:**
- `id: reviewer-testing`, `inherits: reviewer`
- `identity: "Senior test engineer"`
- `vocabulary`: At least 8 ADDITIONAL terms (not duplicating parent):
  - boundary value analysis
  - equivalence partitioning
  - test isolation
  - fixture management
  - assertion specificity
  - coverage metric (statement, branch, path)
  - mutation testing signal
  - test pyramid (unit, integration, e2e)
  - flaky test detection
  - test-as-documentation
- `anti_patterns`: At least 5 ADDITIONAL entries (not duplicating parent):
  - "Coverage theater" — detect: high coverage numbers with weak assertions
  - "Mock overuse" — detect: testing the mock, not the code
  - "Happy-path-only testing" — detect: test suites that only exercise success paths
  - "Test coupling" — detect: tests that break when unrelated code changes
  - "Assertion-free tests" — detect: test functions that execute code but make no assertions
- `tools`: Same read-oriented subset as `reviewer`
- Stage: `reviewing`

---

## 3. Dependency Graph

```
[External] base.yaml (from Base and Authoring Role Content plan)
  │
  └──→ Task 1: reviewer.yaml
         │
         ├──→ Task 2: reviewer-conformance.yaml + reviewer-quality.yaml
         │
         └──→ Task 3: reviewer-security.yaml + reviewer-testing.yaml
```

**Execution order:**
1. Task 1 executes first (serial — Tasks 2 and 3 depend on it)
2. Tasks 2 and 3 execute in parallel (no dependencies between them)

**External dependency:** `base.yaml` must exist before Task 1 begins. This is produced
by Task 1 of the Base and Authoring Role Content plan.

**Maximum parallelism:** 2 concurrent tasks after Task 1 completes.

---

## 4. Interface Contracts

These tasks produce content files, not code. The shared interfaces are conventions and
terminology that must be consistent across all 5 review role files.

### 4.1 Role YAML Schema Contract

Every review role file must use this field structure:

```yaml
id: <role-id>
inherits: <parent-role-id>
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

### 4.2 Inheritance Hierarchy Contract

The inheritance chain must be:

```
base
  └── reviewer
        ├── reviewer-conformance
        ├── reviewer-quality
        ├── reviewer-security
        └── reviewer-testing
```

Every specialist role has `inherits: reviewer`. The `reviewer` role has `inherits: base`.
No other inheritance paths are permitted.

### 4.3 No-Duplication Contract

This is the critical contract for this plan. Before finalising any specialist role file,
the author must verify:

1. **Vocabulary check:** No term in the specialist's `vocabulary` list appears in the
   `reviewer` role's `vocabulary` list (case-insensitive comparison).
2. **Anti-pattern check:** No entry in the specialist's `anti_patterns` list has the same
   `name` as an entry in the `reviewer` role's `anti_patterns` list.
3. **Exception:** The security-specific "Severity Inflation" is considered distinct from
   the parent's "Severity inflation" because it carries CVSS-specific detection signals
   (>30% critical threshold) and security-specific resolution (apply CVSS scoring). The
   inheritance resolver treats these as separate entries based on differing content.

### 4.4 Tool Subset Contract

Every review role must declare the same base tool subset:

**Required tools:** `entity`, `doc_intel`, `knowledge`, `read_file`, `grep`, `search_graph`

**Permitted additional tools:** `status`, `get_code_snippet`, `trace_call_path`,
`query_graph`, `doc`

**Prohibited tools:** `decompose`, `estimate`, `handoff`, `finish`, `worktree`, `pr`,
`merge`, `branch` — reviewers evaluate existing work, they do not create new work items
or manage branches.

All 5 review roles should declare the same tool list for consistency, since the tool
needs are shared across all review dimensions.

### 4.5 Anti-Pattern Structure Contract

Every anti-pattern across all review role files must have exactly 4 fields:
- `name` — human-readable label (title case)
- `detect` — observable signal (present tense, describes what to look for)
- `because` — explanation of harm (enables generalisation, not a restatement of detect)
- `resolve` — concrete corrective action (imperative, specific, actionable)

### 4.6 Composition Model Contract

No review role file may contain:
- Step-by-step procedures (belongs in `review-code` skill)
- Output format definitions (belongs in `review-code` skill)
- Checklists (belongs in `review-code` skill)
- Evaluation criteria (belongs in `review-code` skill)

Role files contain only: identity, vocabulary, anti-patterns, and tool declarations.
The role defines "who you are." The skill defines "what you're doing right now."

---

## 5. Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| FR-001 (Common role schema for review roles) | 1, 2, 3 |
| FR-002 (Reviewer base — identity and vocabulary) | 1 |
| FR-003 (Reviewer base — anti-patterns) | 1 |
| FR-004 (Reviewer-conformance — identity and vocabulary) | 2 |
| FR-005 (Reviewer-conformance — additional anti-patterns) | 2 |
| FR-006 (Reviewer-quality — identity and vocabulary) | 2 |
| FR-007 (Reviewer-quality — additional anti-patterns) | 2 |
| FR-008 (Reviewer-security — identity and vocabulary) | 3 |
| FR-009 (Reviewer-security — additional anti-patterns) | 3 |
| FR-010 (Reviewer-testing — identity and vocabulary) | 3 |
| FR-011 (Reviewer-testing — additional anti-patterns) | 3 |
| FR-012 (Tool subset for review roles) | 1, 2, 3 |
| FR-013 (No content duplication across inheritance) | 2, 3 |
| FR-014 (Stage association — reviewing only) | 1, 2, 3 |
| NFR-001 (Novelty test compliance) | 1, 2, 3 |
| NFR-002 (Tone and explanatory style) | 1, 2, 3 |
| NFR-003 (Terminology consistency) | 1, 2, 3 |
| NFR-004 (Lean content) | 1, 2, 3 |
| NFR-005 (Identity constraints) | 1, 2, 3 |
| NFR-006 (Composition model integrity) | 1, 2, 3 |