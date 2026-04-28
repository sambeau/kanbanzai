# Specification: Review Role Content

| Field | Value |
|-------|-------|
| **Feature** | FEAT-01KN588PFG6GY (review-role-content) |
| **Design** | `work/design/skills-system-redesign-v2.md` §3.1, §4.3 |
| **Status** | Draft |

## Overview

This specification defines the required content for five role YAML files that form the review layer of the Kanbanzai 3.0 role taxonomy: `reviewer` (base review role), `reviewer-conformance`, `reviewer-quality`, `reviewer-security`, and `reviewer-testing`. These roles form an inheritance hierarchy where `reviewer` provides the shared review identity, vocabulary, and anti-patterns, and each specialist role inherits from `reviewer` and adds domain-specific vocabulary and anti-patterns. Specialist roles carry ADDITIONAL content that is MERGED with the parent's during inheritance resolution — they do not duplicate the parent's content. The composition model is: same review skill (`review-code`) combined with different reviewer roles produces different expertise lenses on the same code.

## Scope

### In Scope

- Content requirements for 5 role YAML files: `reviewer`, `reviewer-conformance`, `reviewer-quality`, `reviewer-security`, `reviewer-testing`
- Required fields, field constraints, and content guidelines for each role
- The inheritance hierarchy: all specialist reviewers inherit from `reviewer`, which inherits from `base`
- Vocabulary payload content requirements per role (parent vocabulary vs additional specialist vocabulary)
- Anti-pattern content requirements per role (parent anti-patterns vs additional specialist anti-patterns)
- Tool subset declarations per role
- The additive merge constraint: specialist roles carry ADDITIONAL content, not duplicated content
- Stage binding associations (all review roles are associated with the `reviewing` stage)

### Explicitly Excluded

- The role YAML schema definition and parsing logic (covered by the Role System feature FEAT-01KN588PCVN4Y)
- Inheritance resolution mechanics (covered by the Role System feature)
- Context assembly pipeline (covered by FEAT-01KN588PE43M6)
- Base and authoring roles — covered by the Base and Authoring Role Content specification
- Skill file content (`review-code`, `orchestrate-review`, `review-plan`) — covered by the Review Skill Content specification
- Binding registry structure and enforcement — covered by FEAT-01KN588PDPE8V
- Review orchestration logic, dispatch patterns, or collation — covered by the review skill content specification
- Implementation details: file paths, YAML serialisation, parsing code, validation code

## Functional Requirements

### FR-001: Common Role Schema for Review Roles

Every review role file MUST contain an `id` field, an `inherits` field, an `identity` field, a `vocabulary` field, an `anti_patterns` field, and a `tools` field. The `reviewer` base role MUST have `inherits: base`. All specialist reviewer roles MUST have `inherits: reviewer`.

**Acceptance criteria:**
- Each of the 5 role files contains all six required fields
- The `reviewer` role has `inherits: base`
- `reviewer-conformance`, `reviewer-quality`, `reviewer-security`, and `reviewer-testing` each have `inherits: reviewer`
- The three-level inheritance chain is valid: specialist → `reviewer` → `base`

### FR-002: Reviewer Base Role — Identity and Vocabulary

The `reviewer` role MUST have `id: reviewer` and `identity: "Senior code reviewer"`. Its vocabulary MUST include the following foundational review terms: finding classification, evidence-backed verdict, review dimension, blocking vs non-blocking, severity assessment, and remediation recommendation. These terms form the shared review vocabulary that all specialist reviewers inherit.

**Acceptance criteria:**
- The `reviewer` role contains `id: reviewer` and `identity: "Senior code reviewer"`
- The `identity` field is under 50 tokens and contains no superlatives
- The vocabulary list contains at least the 6 foundational review terms listed above
- The vocabulary terms pass the 15-year practitioner test (a senior reviewer would use these exact terms with a peer)

### FR-003: Reviewer Base Role — Anti-Patterns

The `reviewer` role MUST carry at least 4 anti-patterns: "Rubber-stamp approval" (referencing MAST FM-3.1), "Dimension bleed," "Prose commentary," and "Severity inflation." Each anti-pattern MUST have `name`, `detect`, `because`, and `resolve` fields. The "Rubber-stamp approval" anti-pattern MUST reference the MAST finding that LLM sycophancy makes approval the path of least resistance and that FM-3.1 is the primary quality failure mode in multi-agent systems.

**Acceptance criteria:**
- The `reviewer` role contains an `anti_patterns` list with at least 4 entries
- Every entry has all four fields: `name`, `detect`, `because`, `resolve`
- One entry is named "Rubber-stamp approval" (or equivalent) and its `because` references MAST FM-3.1 and sycophancy
- One entry is named "Dimension bleed" with a `detect` signal about cross-dimension influence
- One entry is named "Prose commentary" with a `detect` signal about qualitative prose instead of structured findings
- One entry is named "Severity inflation" with a `detect` signal about disproportionate blocking/critical classifications

### FR-004: Reviewer-Conformance — Identity and Additional Vocabulary

The `reviewer-conformance` role MUST have `id: reviewer-conformance`, `inherits: reviewer`, and `identity: "Senior requirements verification engineer"`. Its vocabulary MUST contain ADDITIONAL terms (not duplicating the parent's) specific to requirements verification: acceptance criteria traceability, spec requirement mapping, gap analysis, criterion-by-criterion verification, deviation classification, and conformance matrix.

**Acceptance criteria:**
- The role contains the specified `id`, `inherits`, and `identity` values
- The `identity` field is under 50 tokens and contains no superlatives
- The vocabulary list contains at least 6 requirements-verification-specific terms
- No vocabulary entry duplicates a term already present in the parent `reviewer` vocabulary
- The vocabulary terms are specific to conformance verification, not general review terminology

### FR-005: Reviewer-Conformance — Additional Anti-Patterns

The `reviewer-conformance` role MUST carry ADDITIONAL anti-patterns (not duplicating the parent's) specific to conformance verification: "Assumed conformance" (marking pass without tracing to a requirement), "Partial verification" (checking some acceptance criteria but not all), and "Phantom requirement" (finding issues against requirements not stated in the spec).

**Acceptance criteria:**
- The role contains at least 3 anti-pattern entries in its `anti_patterns` field
- Every entry has all four fields: `name`, `detect`, `because`, `resolve`
- One entry detects marking conformance without evidence of tracing to specific requirements
- One entry detects incomplete coverage of acceptance criteria
- One entry detects evaluating against unstated or invented requirements
- No entry duplicates an anti-pattern already present in the parent `reviewer` role

### FR-006: Reviewer-Quality — Identity and Additional Vocabulary

The `reviewer-quality` role MUST have `id: reviewer-quality`, `inherits: reviewer`, and `identity: "Senior software quality engineer"`. Its vocabulary MUST contain ADDITIONAL terms specific to code quality analysis: cyclomatic complexity, error handling chain, defensive copying, invariant assertion, contract violation, resource lifecycle (open/close pairing), naming consistency, package cohesion, and dead code detection.

**Acceptance criteria:**
- The role contains the specified `id`, `inherits`, and `identity` values
- The `identity` field is under 50 tokens and contains no superlatives
- The vocabulary list contains at least 8 code-quality-specific terms
- No vocabulary entry duplicates a term already present in the parent `reviewer` vocabulary

### FR-007: Reviewer-Quality — Additional Anti-Patterns

The `reviewer-quality` role MUST carry ADDITIONAL anti-patterns specific to quality review: "Style-as-defect" (flagging style preferences as quality issues), "Nitpick escalation" (trivial issues marked as blocking), and "Improvement suggestion disguised as defect" (recommending enhancements in the guise of defect findings).

**Acceptance criteria:**
- The role contains at least 3 anti-pattern entries in its `anti_patterns` field
- Every entry has all four fields: `name`, `detect`, `because`, `resolve`
- One entry detects conflating style preferences with genuine quality issues
- One entry detects trivial issues classified at blocking severity
- One entry detects improvement suggestions reported as defects
- No entry duplicates an anti-pattern already present in the parent `reviewer` role

### FR-008: Reviewer-Security — Identity and Additional Vocabulary

The `reviewer-security` role MUST have `id: reviewer-security`, `inherits: reviewer`, and `identity: "Senior application security engineer"`. Its vocabulary MUST contain ADDITIONAL terms specific to application security: OWASP Top 10 (2021 edition), STRIDE threat model, CWE weakness classification, CVSS v3.1 scoring, input validation boundary, authentication flow analysis, authorization bypass pattern, secrets detection, SQL injection via string concatenation, insecure direct object reference (IDOR), cross-site request forgery (CSRF), security header configuration (CSP, HSTS), dependency vulnerability scanning, least privilege principle, and defense-in-depth layering.

**Acceptance criteria:**
- The role contains the specified `id`, `inherits`, and `identity` values
- The `identity` field is under 50 tokens and contains no superlatives
- The vocabulary list contains at least 12 security-specific terms from the design's canonical example
- The vocabulary includes recognised security frameworks and standards (OWASP, STRIDE, CWE, CVSS)
- No vocabulary entry duplicates a term already present in the parent `reviewer` vocabulary

### FR-009: Reviewer-Security — Additional Anti-Patterns

The `reviewer-security` role MUST carry at least 5 ADDITIONAL anti-patterns specific to security review: "Checkbox Compliance," "Scope Creep into Exploitation," "Severity Inflation," "Framework Trust," and "Boundary Blindness." The "Checkbox Compliance" anti-pattern MUST detect evaluating against a checklist without understanding the threat model. The "Scope Creep into Exploitation" anti-pattern MUST detect attempting to exploit vulnerabilities rather than identify and classify them. The "Framework Trust" anti-pattern MUST detect assuming framework defaults are secure without verification. The "Boundary Blindness" anti-pattern MUST detect reviewing only changed code without examining trust boundaries.

**Acceptance criteria:**
- The role contains at least 5 anti-pattern entries in its `anti_patterns` field
- Every entry has all four fields: `name`, `detect`, `because`, `resolve`
- "Checkbox Compliance" references the gap between checklist compliance and actual threat model understanding
- "Scope Creep into Exploitation" distinguishes the reviewer's role (assess and report) from a pentester's role (prove exploitability)
- "Severity Inflation" references CVSS scoring as the resolution mechanism, with a detection threshold of >30% critical findings
- "Framework Trust" references custom configurations and escape hatches as the gap in framework defaults
- "Boundary Blindness" references trust boundaries (user→server, service→service, internal→external) as the focus area
- No entry duplicates an anti-pattern already present in the parent `reviewer` role (note: the security-specific "Severity Inflation" is distinct from the parent's — it is scoped to security severity with CVSS, not general review severity)

### FR-010: Reviewer-Testing — Identity and Additional Vocabulary

The `reviewer-testing` role MUST have `id: reviewer-testing`, `inherits: reviewer`, and `identity: "Senior test engineer"`. Its vocabulary MUST contain ADDITIONAL terms specific to test evaluation: boundary value analysis, equivalence partitioning, test isolation, fixture management, assertion specificity, coverage metric (statement, branch, path), mutation testing signal, test pyramid (unit, integration, e2e), flaky test detection, and test-as-documentation.

**Acceptance criteria:**
- The role contains the specified `id`, `inherits`, and `identity` values
- The `identity` field is under 50 tokens and contains no superlatives
- The vocabulary list contains at least 8 testing-specific terms
- The vocabulary covers both test design techniques (boundary value analysis, equivalence partitioning) and test infrastructure concerns (fixture management, flaky test detection)
- No vocabulary entry duplicates a term already present in the parent `reviewer` vocabulary

### FR-011: Reviewer-Testing — Additional Anti-Patterns

The `reviewer-testing` role MUST carry ADDITIONAL anti-patterns specific to test review: "Coverage theater" (high coverage numbers with weak assertions), "Mock overuse" (testing the mock, not the code), "Happy-path-only testing," "Test coupling" (tests that break when unrelated code changes), and "Assertion-free tests."

**Acceptance criteria:**
- The role contains at least 5 anti-pattern entries in its `anti_patterns` field
- Every entry has all four fields: `name`, `detect`, `because`, `resolve`
- One entry detects inflated coverage metrics masking weak test quality
- One entry detects excessive mocking that tests mock behaviour rather than real behaviour
- One entry detects test suites that only exercise success paths
- One entry detects tests with coupling to unrelated implementation details
- One entry detects test functions that execute code but make no assertions
- No entry duplicates an anti-pattern already present in the parent `reviewer` role

### FR-012: Tool Subset for Review Roles

Every review role MUST declare a `tools` field listing the MCP tool subset appropriate for review work. The tool list MUST include tools for entity access (`entity`), document intelligence (`doc_intel`), knowledge queries (`knowledge`), file reading (`read_file`), and code search (`grep`, `search_graph`). Review roles MUST NOT include tools for entity creation, task decomposition (`decompose`), or estimation (`estimate`) — reviewers evaluate existing work, they do not create new work items.

**Acceptance criteria:**
- Each review role's `tools` field is a non-empty list
- Every review role includes at least: `entity`, `doc_intel`, `knowledge`, `read_file`, `grep`, `search_graph`
- No review role includes `decompose` or `estimate`
- The tool list is appropriate for read-oriented review work

### FR-013: No Content Duplication Across Inheritance

Specialist reviewer roles MUST NOT duplicate vocabulary terms or anti-patterns that are already present in the parent `reviewer` role. Each specialist role carries only ADDITIONAL content. The assembled context for any specialist reviewer is produced by the inheritance resolution mechanism merging parent and child content — the role file itself contains only the delta.

**Acceptance criteria:**
- No specialist role's `vocabulary` list contains any term that appears in the `reviewer` role's `vocabulary` list
- No specialist role's `anti_patterns` list contains an entry with the same `name` as an entry in the `reviewer` role's `anti_patterns` list
- The total assembled vocabulary for any specialist (after inheritance merge) is the union of the parent's and child's vocabulary
- The total assembled anti-patterns for any specialist (after inheritance merge) is the union of the parent's and child's anti-patterns

### FR-014: Stage Association

All five review roles MUST be associated with the `reviewing` stage. No review role MUST be associated with any other workflow stage. The composition model is: the `review-code` skill provides the procedure, the reviewer role provides the expertise lens — different roles produce different review perspectives on the same code using the same skill.

**Acceptance criteria:**
- Every review role is declared for use in stage `reviewing`
- No review role is declared for stages `designing`, `specifying`, `dev-planning`, `developing`, `researching`, or `documenting`

## Non-Functional Requirements

### NFR-001: Novelty Test Compliance

Every content element in every review role file MUST pass the novelty test (design §8.1). General explanations of what code review is, how security vulnerabilities work, or what test coverage means MUST NOT appear. Content MUST be limited to domain vocabulary that routes model attention and anti-patterns with project-specific detection signals and resolution steps.

**Acceptance criteria:**
- No review role file contains explanations of general review concepts, security fundamentals, or testing methodology
- Every content element is either a vocabulary routing term or a structured anti-pattern with project-specific guidance

### NFR-002: Tone and Explanatory Style

Anti-pattern `because` clauses MUST use an explanatory tone. They MUST explain WHY the pattern is harmful in a way that enables the model to generalise to adjacent cases. Bare imperatives without rationale ("NEVER do X") MUST NOT appear.

**Acceptance criteria:**
- Every `because` clause provides a substantive explanation, not just a restatement of the `detect` signal
- Every `resolve` clause provides a concrete corrective action
- No instructional content consists of unexplained imperatives

### NFR-003: Terminology Consistency

Each role's vocabulary payload defines the canonical terms for its domain. Within the role file's own content — anti-patterns, any prose — those canonical terms MUST be used exclusively. If the vocabulary says "finding classification," the anti-patterns MUST NOT use "issue categorisation" or "defect typing" as synonyms.

**Acceptance criteria:**
- Within each role file, terms used in anti-pattern text match the vocabulary entries
- No synonyms are used for terms that appear in the vocabulary list

### NFR-004: Lean Content

Each role file MUST follow design principle DP-6 (lean by default). The `reviewer` base role vocabulary MUST contain 5–15 foundational terms. Each specialist role vocabulary MUST contain 5–15 additional terms. Anti-pattern lists MUST contain 2–10 entries per role. Total content per specialist role (the delta, excluding inherited content) MUST NOT exceed approximately 400 tokens.

**Acceptance criteria:**
- All vocabulary lists are within the 5–15 term range per role
- All anti-pattern lists are within the 2–10 entry range per role
- No specialist role's own content (delta) is inflated with redundant or overly verbose entries

### NFR-005: Identity Constraints

Every review role's `identity` field MUST be a real job title under 50 tokens. Identities MUST NOT contain superlatives, flattery, or elaborate backstories. The identity is a job title; competence is defined by the vocabulary and anti-patterns.

**Acceptance criteria:**
- Each role's `identity` field is under 50 tokens
- No `identity` field contains "expert," "world-class," "the best," "excels," or equivalent superlatives
- Each `identity` field is recognisable as a real job title in the relevant domain

### NFR-006: Composition Model Integrity

The review role hierarchy MUST support the composition model described in the design: the same `review-code` skill combined with different reviewer roles produces different expertise lenses on the same code. Role content MUST NOT contain procedural instructions that belong in skills. Role content MUST be limited to identity, vocabulary, anti-patterns, and tool declarations — the "who you are" layer.

**Acceptance criteria:**
- No review role file contains step-by-step procedures, output format definitions, or checklists
- Role files contain only identity, vocabulary, anti-patterns, and tools
- Any two specialist roles can be paired with the same `review-code` skill to produce meaningfully different review outputs

## Acceptance Criteria

The acceptance criteria for each requirement are listed inline with each FR and NFR above. The following are aggregate acceptance criteria for the specification as a whole:

1. **Completeness:** All 5 review role files are authored with all required fields populated.
2. **Inheritance integrity:** The inheritance chain is valid: each specialist → `reviewer` → `base`. No circular or broken references.
3. **No duplication:** The `reviewer` role's vocabulary and anti-patterns do not appear in any specialist role's own lists. Specialist roles carry only ADDITIONAL (delta) content.
4. **Additive merge correctness:** After inheritance resolution, each specialist role's assembled context contains the union of `base` + `reviewer` + specialist vocabulary and anti-patterns — verified by inspecting the merged output from the role system's inheritance resolver.
5. **Composition verification:** Running the `review-code` skill with `reviewer-security` produces security-focused review output, while running it with `reviewer-testing` produces testing-focused review output — the difference comes entirely from the role, not the skill. Verified by running representative review tasks with different roles and comparing the vocabulary used in the output.
6. **Anti-pattern coverage:** Each specialist role's anti-patterns address the most common failure modes for that review dimension, as identified in the design document.

## Dependencies and Assumptions

### Dependencies

- **Role System feature (FEAT-01KN588PCVN4Y):** Defines the YAML schema that these role files must conform to, including the inheritance resolution mechanism that merges parent and child vocabulary and anti-patterns.
- **Base and Authoring Role Content (FEAT-01KN588PF5P5Y):** Defines the `base` role that `reviewer` inherits from. The `base` role must be authored before the `reviewer` role's inheritance can be resolved.
- **Binding Registry feature (FEAT-01KN588PDPE8V):** Defines the stage-to-role mappings. This specification declares that all review roles are associated with the `reviewing` stage, but the binding registry enforces that association.
- **Context Assembly Pipeline (FEAT-01KN588PE43M6):** Defines how role vocabulary and anti-patterns from parent and child roles are merged during inheritance resolution and how the assembled context is ordered by the attention curve.
- **Review Skill Content (FEAT-01KN588PGJNM0):** Defines the `review-code` skill that all review roles are composed with. The composition model (same skill + different roles = different expertise lenses) depends on both the roles defined here and the skill defined there.

### Assumptions

1. The role YAML schema supports all field types referenced in this specification (string identity, list vocabulary, structured anti-pattern entries with four sub-fields, list tools).
2. Inheritance resolution merges `vocabulary` and `anti_patterns` lists additively — a child role's entries are appended to the parent's, not substituted. This applies at each level: specialist inherits from `reviewer`, which inherits from `base`.
3. The `tools` field accepts a list of MCP tool names and is used to scope tool availability during context assembly.
4. The `reviewer-security` role's "Severity Inflation" anti-pattern is considered distinct from the parent `reviewer` role's "Severity inflation" anti-pattern because it carries security-specific detection signals (CVSS scoring, >30% critical threshold) and security-specific resolution (apply CVSS). The inheritance resolver treats these as two separate entries in the merged anti-pattern list based on differing content, not just differing names.
5. The `reviewer` role's vocabulary is intentionally minimal (foundational review terms only) to leave room for specialist vocabulary without exceeding the lean content constraints.
6. Token counts referenced in this specification use the approximate tokenisation of modern LLMs (GPT-4/Claude class). Exact counts may vary by model.