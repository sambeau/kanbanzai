# Design: Evidence-Based Skills and Roles System for Kanbanzai 3.0

| Field | Value |
|-------|-------|
| Date | 2025-07-13 |
| Status | Proposal |
| Author | Design Agent |
| Informed by | `work/research/ai-agent-best-practices-research.md` |
| Supersedes | Current `.skills/` system and `.kbz/context/roles/` profiles |

---

## 1. Problem Statement

The current Kanbanzai system has two separate subsystems for shaping agent behaviour:

1. **Context profiles** (`.kbz/context/roles/*.yaml`) — define identity and conventions per role. Currently: `base`, `developer`, `reviewer`.
2. **SKILLs** (`.skills/*.md`) — define step-by-step procedures for repeatable tasks. Currently: `code-review`, `plan-review`, `document-creation`.

Research from 17 peer-reviewed papers (distilled in the "10 Claude Code Principles" series) identifies specific, measurable problems with this separation:

**Problem 1: No vocabulary routing.** The research identifies domain-specific vocabulary as the *primary quality lever* for LLM output (Ranjan et al., 2024 — "One Word Is Not Enough"). Vocabulary terms act as routing signals that determine which knowledge clusters the model activates. "OWASP Top 10 audit, STRIDE threat model (Shostack)" routes to security engineering expertise; "review the security" routes to blog posts. Neither our context profiles nor our SKILLs carry vocabulary payloads. Every agent we dispatch is operating without the single most impactful quality intervention available.

**Problem 2: No attention-optimized structure.** The U-shaped attention curve (Liu et al., 2024; Wu et al., 2025) shows accuracy drops 30%+ when critical information is in the middle of context. Our SKILL files put Purpose and Audience first (low-value preamble consuming the high-attention opening) and procedures in the middle (where attention is weakest). The research-backed ordering puts vocabulary first, anti-patterns before instructions, and retrieval anchors last.

**Problem 3: Generalist review.** We have one `reviewer` profile. The research shows a panel of brief specialists (<50 token identity each, with domain vocabulary) dramatically outperforms a single generalist (PRISM persona framework; Ranjan et al., 2024). Our code review SKILL has five *dimensions* (conformance, quality, testing, documentation, workflow) but one *reviewer*. Each dimension deserves its own vocabulary payload and anti-pattern set.

**Problem 4: No stage-to-context binding.** Features travel a lifecycle: `proposed → designing → specifying → dev-planning → developing → reviewing → done`. Documents have types (`design`, `specification`, `dev-plan`, `research`, `report`) and stages (`draft → approved → superseded`). But there is no formal mapping from "the feature is in the `specifying` stage" to "use this role and this skill." The binding is implicit — agents read AGENTS.md and figure it out.

**Problem 5: No anti-pattern system.** The research shows named anti-patterns with detect/resolve patterns steer models away from the generic centre of their training distribution (CHI 2023, "Why Johnny Can't Prompt"). Our profiles have conventions ("do X") but not anti-patterns ("never do Y BECAUSE Z"). The BECAUSE clause is what makes rules generalisable to adjacent cases.

**Problem 6: SKILLs and profiles don't compose.** A security-specialist reviewer doing a code review needs both security vocabulary (from the role) and review methodology (from the skill). Currently there's no mechanism for this composition. The reviewer profile provides review conventions; the code review SKILL provides the procedure; but neither carries vocabulary, and there's no way to layer security expertise on top.

---

## 2. Design Principles

These are derived from the research and are constraints on the design, not suggestions.

### DP-1: Vocabulary Is the Primary Routing Mechanism

Every context unit (role or skill) that touches an LLM must carry a vocabulary payload of 15–30 domain-specific terms. These terms must pass the **15-year practitioner test**: would a senior expert with 15+ years of domain experience use this exact term when talking with a peer?

*Source: Ranjan et al. (2024), PRISM framework, Principle 6 and 10.*

### DP-2: Follow the Attention Curve

Content within any context unit must follow the U-shaped attention pattern:

- **Top** (high attention): identity, vocabulary payload, hard constraints
- **Middle** (low attention): procedures, reference material — must be structured as numbered steps with IF/THEN conditions to survive attention degradation
- **Bottom** (high attention): retrieval anchors, evaluation criteria, "questions this answers"

*Source: Liu et al. (2024), Wu et al. (2025), Anthropic context engineering guide (Sep 2025).*

### DP-3: Brief Identities, No Flattery

Role identities must be under 50 tokens. Real job titles only. No superlatives ("world-class expert"), no flattery ("you are the best"), no elaborate backstories. Define competence through vocabulary and anti-patterns, not adjectives.

*Source: PRISM persona framework (2024).*

### DP-4: Named Anti-Patterns with BECAUSE Clauses

Every context unit must include 5–10 named anti-patterns, each with: detection signal, explanation with BECAUSE clause, resolution step. Anti-patterns are both guardrails (preventing mistakes) and steering mechanisms (pushing the model toward project-specific output).

*Source: Zamfirescu-Pereira et al., CHI 2023 ("Why Johnny Can't Prompt"); Vaarta Analytics (2026).*

### DP-5: Composition Over Inheritance

Roles and skills compose by addition: the final context is role vocabulary + skill vocabulary + role anti-patterns + skill anti-patterns + skill procedure. This must be a mechanical operation — the system combines them, not the agent.

The current profile inheritance (`reviewer` inherits `base`) is retained but extended: a role provides the "who" layer, a skill provides the "what" layer, and the system assembles the composite context with correct attention-curve ordering.

### DP-6: Lean by Default — n=5 Beats n=19

At 19 requirements, accuracy drops below what 5 requirements achieve (Vaarta Analytics, 2026). Every context unit must be lean: vocabulary payloads of 15–30 terms (not 50), anti-pattern lists of 5–10 (not 20), procedures with the minimum steps needed. When in doubt, cut. A concise context unit that fits the 15–40% optimal utilisation zone outperforms a comprehensive one that overflows it.

*Source: Vaarta Analytics (2026), Anthropic context engineering guide.*

### DP-7: Separate Generation from Evaluation

The agent that produces output must not be the agent that evaluates it. Evaluation criteria are separated from generation instructions and carried in distinct sections, phrased as gradable questions ("Can the reviewer identify the most critical finding in under 10 seconds?" not "Is the review good?").

*Source: Anthropic harness design research (Mar 2026).*

### DP-8: Stage Bindings Are Explicit

The mapping from workflow stage → role(s) → skill(s) → MCP tool subset is declared, not inferred. When the system assembles context for a task, it knows the feature's lifecycle stage and uses the binding to select the right context units automatically.

---

## 3. Conceptual Architecture

The redesigned system has three layers and a binding registry:

```
┌──────────────────────────────────────────────────────────┐
│                    BINDING REGISTRY                       │
│  Maps workflow stages to roles + skills + tool subsets    │
│  "specifying" → spec-author + write-spec + [doc, entity] │
└───────────────────────┬──────────────────────────────────┘
                        │ selects
          ┌─────────────┴──────────────┐
          ▼                            ▼
┌──────────────────┐        ┌──────────────────────┐
│      ROLES       │        │       SKILLS         │
│  Who you are     │        │  What you're doing   │
│                  │        │                      │
│  • Identity      │        │  • Vocabulary        │
│    (<50 tokens)  │        │    (task-specific)   │
│  • Vocabulary    │        │  • Anti-patterns     │
│    (domain)      │        │    (task-specific)   │
│  • Anti-patterns │        │  • Procedure         │
│    (domain)      │        │  • Output format     │
│  • Tool subset   │        │  • Examples          │
│                  │        │  • Eval criteria     │
└────────┬─────────┘        │  • Retrieval anchors │
         │ inherits         └──────────┬───────────┘
         ▼                             │
┌──────────────────┐                   │
│      BASE        │                   │
│  Project-wide    │                   │
│  identity and    │                   │
│  conventions     │                   │
└──────────────────┘                   │
                                       │
         ┌─────────────────────────────┘
         ▼
┌──────────────────────────────────────────────────────────┐
│                 ASSEMBLED CONTEXT                         │
│  Ordered by attention curve:                             │
│  1. Identity + hard constraints         (high attention) │
│  2. Combined vocabulary payload         (high attention) │
│  3. Combined anti-pattern watchlist     (medium)         │
│  4. Skill procedure                     (structured/mid) │
│  5. Output format + examples            (rising)         │
│  6. Evaluation criteria                 (high attention) │
│  7. Retrieval anchors                   (high attention) │
└──────────────────────────────────────────────────────────┘
```

### 3.1 Roles — "Who You Are"

A role defines the agent's professional identity and domain expertise. It is the routing layer — it determines which region of the model's knowledge space activates before any procedure is loaded.

**Structure:**

```yaml
id: reviewer-security
inherits: reviewer
identity: "Senior application security engineer"

vocabulary:
  - "OWASP Top 10 (2021 edition)"
  - "STRIDE threat model (Shostack)"
  - "CWE weakness classification"
  - "CVSS v3.1 scoring"
  - "input validation boundary"
  - "authentication flow analysis"
  - "authorization bypass pattern"
  - "secrets detection (hardcoded credentials, API keys)"
  - "SQL injection via string concatenation"
  - "insecure direct object reference (IDOR)"
  - "cross-site request forgery (CSRF)"
  - "security header configuration (CSP, HSTS)"
  - "dependency vulnerability scanning"
  - "least privilege principle"
  - "defense-in-depth layering"

anti_patterns:
  - name: "Checkbox Compliance"
    detect: "Evaluating against a checklist without understanding the threat model"
    because: "Real vulnerabilities exist at the intersection of features, not in isolated checklist items"
    resolve: "Map each finding to a specific threat scenario before classifying severity"

  - name: "Scope Creep into Exploitation"
    detect: "Attempting to exploit or demonstrate a vulnerability rather than identify and classify it"
    because: "The reviewer's job is to assess and report, not to prove exploitability"
    resolve: "Describe the attack vector and conditions; leave proof-of-concept to dedicated pentest"

  - name: "Severity Inflation"
    detect: "More than 30% of findings classified as critical"
    because: "Over-classifying dilutes the signal from genuine critical vulnerabilities"
    resolve: "Apply CVSS scoring; only confirmed exploitable paths with high impact are critical"

  - name: "Framework Trust"
    detect: "Assuming a framework's defaults are secure without verification"
    because: "Frameworks provide secure defaults for common cases; custom configurations, middleware ordering, and escape hatches create gaps frameworks don't cover"
    resolve: "Verify each security-relevant configuration explicitly against the deployment context"

  - name: "Boundary Blindness"
    detect: "Reviewing only the changed code without examining the trust boundary it crosses"
    because: "Most vulnerabilities live at boundaries (user→server, service→service, internal→external) not in interior logic"
    resolve: "Trace each input from its trust boundary through to its final use"

tools:
  - entity
  - doc_intel
  - knowledge
  - read_file
  - grep
  - search_graph
```

**Key design decisions:**

- **`vocabulary` is a first-class field.** Not buried in conventions. Not optional. It is the routing mechanism.
- **`anti_patterns` use a `because` clause.** This is what makes them generalisable. "Never do X" covers one case. "Never do X BECAUSE Y" covers adjacent cases the model can reason about.
- **`tools` declares the MCP tool subset.** When context is assembled for this role, only these tool definitions are included. A security reviewer does not need `decompose` or `estimate`. This implements the `jig` pattern (Principle 10) — minimal tool loading per context.
- **`identity` is a real job title under 50 tokens.** Not "You are an expert who excels at finding security vulnerabilities with your deep knowledge and years of experience." Just a job title. The vocabulary does the routing work.
- **`inherits` composes with parent.** `reviewer-security` gets all of `reviewer`'s base review vocabulary and anti-patterns, PLUS its security-specific ones.

### 3.2 Skills — "What You're Doing Right Now"

A skill defines the procedure, output format, and evaluation criteria for a specific type of work. It follows the attention-optimized section ordering from the research.

**File structure:**

```
.kbz/skills/
├── review-code/
│   ├── SKILL.md           (<500 lines — the core skill)
│   └── references/
│       ├── finding-classification.md
│       ├── edge-cases.md
│       └── evaluation-rubric.md
├── write-spec/
│   ├── SKILL.md
│   └── references/
│       ├── acceptance-criteria-patterns.md
│       └── spec-anti-patterns.md
├── write-design/
│   ├── SKILL.md
│   └── references/
│       └── design-document-patterns.md
...
```

**SKILL.md section ordering** (attention curve optimized):

```markdown
---
name: review-code
description:
  expert: "Multi-dimension code review producing classified findings
    with evidence-backed verdicts against acceptance criteria"
  natural: "Review code changes against a spec and produce a structured
    report of what's right and what needs fixing"
triggers:
  - review code changes
  - evaluate implementation against spec
  - check code quality
roles: [reviewer, reviewer-conformance, reviewer-quality,
        reviewer-security, reviewer-testing]
stage: reviewing
---

## Vocabulary

(Task-specific terms — these combine with the role's vocabulary)

- finding classification (blocking, non-blocking)
- evidence-backed verdict
- acceptance criteria traceability
- per-dimension outcome (pass, pass_with_notes, concern, fail)
- review unit decomposition
- structured review output
- remediation recommendation
- spec conformance gap

## Anti-Patterns

(Task-specific anti-patterns — these combine with the role's)

### Rubber-Stamp Review (MAST FM-3.1)
- **Detect:** Verdict is "approved" with zero findings or no evidence citations
- **BECAUSE:** LLM sycophancy makes approval the path of least resistance;
  FM-3.1 is the #1 quality failure in multi-agent systems (MAST, 2024)
- **Resolve:** Require at least one finding OR per-dimension evidence for clearance

### Severity Inflation
- **Detect:** More than 40% of findings classified as blocking
- **BECAUSE:** Over-classifying non-blocking issues as blocking dilutes the
  signal from genuine spec violations and slows remediation
- **Resolve:** Re-check each blocking finding against the classification
  criteria; blocking requires a specific violated requirement

### Dimension Bleed
- **Detect:** A finding in one dimension influences the verdict of another
- **BECAUSE:** Dimensions are independent evaluation axes; a poor test
  score does not make the implementation incorrect
- **Resolve:** Evaluate each dimension in isolation; cross-reference only
  in the aggregate verdict

### Prose Commentary
- **Detect:** Output contains qualitative prose ("well-structured," "clean
  code") instead of structured findings
- **BECAUSE:** Prose is ambiguous and cannot be machine-parsed for
  remediation; structured output has exactly one interpretation
- **Resolve:** Replace every qualitative statement with a finding entry
  containing dimension, location, and evidence

### Missing Spec Anchor
- **Detect:** A finding does not cite a specific spec requirement
- **BECAUSE:** Without a spec anchor, the finding is an opinion, not a
  conformance gap; opinions cannot be objectively verified or remediated
- **Resolve:** Link every blocking finding to a numbered acceptance criterion

## Procedure

### Step 1: Orient from inputs

1. Read the spec section(s) fully. Understand what was required.
2. Read all files in the file list. Understand what was implemented.
3. Note the review profile — this determines required dimensions.
4. IF any input is missing → STOP. Report Missing Context edge case.

### Step 2: Evaluate each dimension independently

For each required dimension, work through its specific evaluation
questions. Record a per-dimension outcome. Do not let a poor result
in one dimension affect your assessment of another.

...

## Output Format

(The structured template for this skill's deliverable)

...

## Examples

### BAD: Rubber-stamp with prose

  Review unit: service-layer
  Overall: approved
  Notes: Code is well-structured and follows Go conventions. Good use
  of error handling. Tests look comprehensive.

WHY BAD: No findings. No evidence citations. No per-dimension verdicts.
Qualitative prose with no structured data. A human or machine cannot
determine what was actually checked.

### GOOD: Evidence-backed structured review

  Review unit: service-layer
  Overall: approved_with_followups
  Dimensions:
    spec_conformance: pass
      Evidence: AC-1 verified (entity creation, L34-52),
      AC-2 verified (validation, L55-71), AC-3 verified (error response, L73-89)
    implementation_quality: pass_with_notes
      Finding (non-blocking): error wrapping in CreateFeature (L48) uses
      fmt.Errorf without %w — loses error chain for callers checking with
      errors.Is
    test_adequacy: pass
      Evidence: 14 test cases covering happy path, validation failures,
      and duplicate detection. Table-driven pattern used throughout.

WHY GOOD: Per-dimension verdicts with specific evidence. Finding has
location and explanation. Spec requirements cited by number. A machine
can parse this; a human can verify each claim.

### GOOD: Evidence-backed clearance (no findings)

  Review unit: storage-layer
  Overall: approved
  Dimensions:
    spec_conformance: pass
      Evidence: AC-4 verified (YAML serialisation, L12-34),
      AC-5 verified (canonical field order, L36-58). Round-trip test
      at L102 confirms deterministic output.
    implementation_quality: pass
      Evidence: Error wrapping with %w throughout. No exported functions
      without doc comments. Interface accepted at consumer (L8), struct
      returned.
    test_adequacy: pass
      Evidence: 22 test cases including round-trip serialisation test.
      Coverage of error paths verified via TestStore_CreateConflict (L145).

WHY GOOD: Zero findings but substantive evidence for each dimension.
The reviewer demonstrably examined the code. Not a rubber stamp.

## Evaluation Criteria

(Separated from the procedure — these are for evaluating the skill's
OUTPUT, not for the agent to self-evaluate during execution)

1. Does every dimension have an explicit outcome (pass/fail/concern)?
   Weight: required.
2. Does every blocking finding cite a specific spec requirement?
   Weight: required.
3. Can a machine extract all findings from the output without ambiguity?
   Weight: high.
4. Are dimensions evaluated independently (no bleed)?
   Weight: high.
5. Does the output distinguish blocking from non-blocking findings?
   Weight: required.
6. Is every "approved" verdict backed by per-dimension evidence?
   Weight: high.

## Questions This Skill Answers

- How do I review code changes against a specification?
- What dimensions should I evaluate during code review?
- How do I classify a finding as blocking vs non-blocking?
- What format should my review output use?
- When should I escalate to a human checkpoint during review?
- What does a well-evidenced "approved" verdict look like?
- How do I handle missing spec sections during review?
- What is the difference between a concern and a fail?
```

**Key design decisions:**

- **Dual-register description** in frontmatter: `expert` (activates deep knowledge on direct invocation) and `natural` (triggers on casual phrasing). Both are needed — expert-only undertriggers; casual-only produces generic output.
- **Vocabulary is FIRST in the body.** This is the highest-attention position. The vocabulary payload activates the right knowledge clusters before the model reads any procedure.
- **Anti-patterns come BEFORE the procedure.** They establish what NOT to do before the model encounters what TO do. Combined positive + negative instruction is the strongest approach (CHI 2023).
- **Procedure is in the MIDDLE.** The middle is the weakest attention zone, but structured numbered steps with IF/THEN conditions survive attention degradation because the structure itself carries signal that prose would lose.
- **Examples are BAD/GOOD pairs with WHY.** The model pattern-matches against examples more reliably than it follows rules (LangChain, 2024). Three examples match nine in effectiveness. Placing the best example last exploits recency bias.
- **Evaluation criteria are SEPARATED from the procedure.** The agent that does the work should not self-evaluate (Anthropic harness design, Mar 2026). Evaluation criteria are phrased as gradable questions with weights so a separate evaluation pass (or human) can assess output quality.
- **"Questions This Skill Answers" is LAST.** This is a retrieval anchor — it sits at the high-attention end of context. It also serves as the trigger matching surface when an orchestrator or the system needs to select the right skill for an ambiguous request.
- **`references/` directory for overflow.** The SKILL.md stays under 500 lines. Detailed anti-pattern documentation, extended examples, and evaluation rubrics go in `references/` and are loaded on-demand (progressive disclosure Layer 3).
- **`roles` field declares compatible roles.** This enables the binding registry to validate that a role-skill combination is sensible.
- **`stage` field declares the workflow stage.** This enables stage-based automatic selection.

### 3.3 The Binding Registry — "When to Use What"

The binding registry maps workflow stages to the role(s), skill(s), and MCP tool subsets that should be loaded. It replaces the current implicit knowledge that agents derive from reading AGENTS.md.

**Location:** `.kbz/config.yaml` (extending the existing project config) or a dedicated `.kbz/stage-bindings.yaml`.

**Structure:**

```yaml
stage_bindings:

  # ── Document authoring stages ─────────────────────────────

  designing:
    description: "Creating or revising a design document"
    roles: [architect]
    skills: [write-design]
    document_type: design
    human_gate: false
    notes: "Single agent. Architect vocabulary routes to system design expertise."

  specifying:
    description: "Writing a formal specification with acceptance criteria"
    roles: [spec-author]
    skills: [write-spec]
    document_type: specification
    human_gate: true  # Spec approval is a required gate
    notes: "Single agent. Spec-author vocabulary routes to requirements engineering."

  dev-planning:
    description: "Breaking a spec into an implementation plan and tasks"
    roles: [architect]
    skills: [write-dev-plan, decompose-feature]
    document_type: dev-plan
    human_gate: true  # Plan review before implementation
    notes: "Single agent. Decomposition uses architect vocabulary for dependency analysis."

  # ── Implementation stage ──────────────────────────────────

  developing:
    description: "Implementing tasks from the dev plan"
    roles: [implementer]
    skills: [implement-task]
    document_type: null
    human_gate: false
    notes: >
      Single agent per task. Implementer role carries language-specific
      vocabulary (e.g., implementer-go for this project). Cascade: start
      Level 0 (single agent + tools). Escalate only if measured output
      is below 45% threshold.

  # ── Review stage ──────────────────────────────────────────

  reviewing:
    description: "Evaluating implementation against the specification"
    roles: [orchestrator]
    skills: [orchestrate-review]
    sub_agents:
      roles: [reviewer-conformance, reviewer-quality, reviewer-security, reviewer-testing]
      skills: [review-code]
      topology: parallel
      max_agents: 4
    document_type: report
    human_gate: true  # Verdict checkpoint
    notes: >
      Multi-agent. Orchestrator dispatches specialist reviewers in parallel.
      Each reviewer gets its own vocabulary-routed role + the shared
      review-code skill. Max 4 concurrent sub-agents (DeepMind saturation
      point). Orchestrator never reads source — metadata only.

  # ── Plan review stage ─────────────────────────────────────

  plan-reviewing:
    description: "Reviewing a completed plan for aggregate delivery"
    roles: [reviewer-conformance]
    skills: [review-plan]
    document_type: report
    human_gate: true
    notes: "Single agent. Plan review is conformance-focused by nature."

  # ── Research and documentation stages ─────────────────────

  researching:
    description: "Producing a research report or analysis"
    roles: [researcher]
    skills: [write-research]
    document_type: research
    human_gate: false

  documenting:
    description: "Updating project documentation for currency"
    roles: [documenter]
    skills: [update-docs]
    document_type: null
    human_gate: false
```

**Key design decisions:**

- **One stage, one binding.** No ambiguity about what to load. The system looks at the feature's current lifecycle state and knows exactly which roles, skills, and tools to provide.
- **`sub_agents` for multi-agent stages.** The `reviewing` stage is the only one that routinely uses multiple agents. The binding declares the sub-agent configuration — roles, skills, topology, and a hard cap derived from the research (max 4 agents — DeepMind saturation point).
- **`human_gate` is declared per stage.** The system knows which stages require human approval before proceeding. This replaces the implicit knowledge that "spec approval is needed before development."
- **`document_type` ties to the document lifecycle.** When a stage produces a document, the binding declares what type. The system can then enforce that the document is registered and approved before the feature advances.
- **`notes` captures the reasoning.** BECAUSE clause at the binding level — why this configuration, not another.

### 3.4 Assembled Context — "The Final Product"

When `handoff(task_id=...)` is called (or when context is assembled for any other reason), the system:

1. Determines the workflow stage from the task's parent feature state.
2. Looks up the stage binding.
3. Loads the role (with inheritance resolution).
4. Loads the skill.
5. Assembles the composite context in attention-curve order.

**Assembly order (attention curve optimized):**

| Position | Content | Source | Attention |
|----------|---------|--------|-----------|
| 1 | Project identity and hard constraints | `base` role | **High** |
| 2 | Role identity | Selected role | **High** |
| 3 | Combined vocabulary payload | Role vocab + Skill vocab | **High** |
| 4 | Combined anti-pattern watchlist | Role anti-patterns + Skill anti-patterns | **Medium-High** |
| 5 | Skill procedure (numbered steps) | Selected skill | **Medium** (structured steps survive) |
| 6 | Output format + examples | Selected skill | **Rising** |
| 7 | Relevant knowledge entries | Knowledge system (auto-surfaced) | **Rising** |
| 8 | Evaluation criteria | Selected skill | **High** |
| 9 | Retrieval anchors ("Questions This Skill Answers") | Selected skill | **High** |

**What is NOT included:**

- Tool definitions for tools not in the role's `tools` list.
- Knowledge entries not relevant to the task's scope.
- Full reference documents (loaded on-demand only).
- Context from other roles or skills not selected by the binding.

**Token budget target:** The assembled context should fit within 15–40% of the context window. The assembly process estimates token count and warns if the assembled context exceeds 40%. If it exceeds 60%, it refuses to assemble and suggests the orchestrator split the work unit.

---

## 4. The Role Taxonomy

Roles are organized by the type of cognitive work they perform, not by seniority or flattery.

### 4.1 Base Roles

These provide project-wide identity and conventions. Every other role inherits from one of these.

#### `base`

The project-wide foundation. Every role inherits this. Contains:
- Project identity ("Kanbanzai — Git-native workflow system for human-AI development")
- Hard constraints ("Spec is law," "No scope creep," "Deterministic YAML serialisation")
- Commit conventions
- Core architectural summary

**Token budget:** ~200–300 tokens. This is always-loaded context (progressive disclosure Layer 1).

### 4.2 Authoring Roles

These are for agents that produce documents and code.

#### `architect`

- **Identity:** "Senior software architect"
- **Vocabulary:** system decomposition, vertical slice, dependency graph, coupling analysis, blast radius assessment, interface boundary, separation of concerns, inversion of control, contract-first design, failure mode enumeration, capacity planning, migration strategy
- **Anti-patterns:** Gold plating, premature abstraction, accidental coupling, distributed monolith, design-by-committee
- **Used in stages:** `designing`, `dev-planning`

#### `spec-author`

- **Identity:** "Senior requirements engineer"
- **Vocabulary:** acceptance criteria (Given/When/Then), requirement traceability, testable assertion, boundary condition, equivalence partition, specification completeness, ambiguity resolution, INVEST criteria (for stories), definition of done
- **Anti-patterns:** Untestable requirement, implicit assumption, scope ambiguity, over-specification (locking in implementation), under-specification (leaving critical behaviour undefined)
- **Used in stages:** `specifying`

#### `implementer` (abstract — project-specific subtypes)

For this project, the concrete role is `implementer-go`:

- **Identity:** "Senior Go engineer"
- **Vocabulary:** goroutine leak, interface segregation, error wrapping (%w), table-driven test, struct embedding, functional option pattern, context propagation, channel direction, sync.Mutex contention, io.Reader/io.Writer composition, zero-value usability, package-level encapsulation
- **Anti-patterns:** God struct, interface pollution (preemptive interfaces), init() coupling, naked goroutine (no context cancellation), error swallowing, stringly-typed API, test-only exports
- **Used in stages:** `developing`

#### `researcher`

- **Identity:** "Senior technical analyst"
- **Vocabulary:** literature review, evidence synthesis, citation traceability, finding classification, confidence assessment, applicability analysis, counter-evidence, research gap identification
- **Anti-patterns:** Cherry-picking (citing only supporting evidence), false equivalence, unsupported generalisation, circular reference
- **Used in stages:** `researching`

#### `documenter`

- **Identity:** "Senior technical writer"
- **Vocabulary:** progressive disclosure, information architecture, cross-reference integrity, terminology consistency, audience-appropriate register, structural parallelism, reading order optimisation
- **Anti-patterns:** Documentation-code divergence, outdated example, assumed knowledge (jargon without definition), documentation duplication (same fact in multiple places)
- **Used in stages:** `documenting`

### 4.3 Review Roles

These are for agents that evaluate work produced by others.

#### `reviewer` (base review role)

- **Identity:** "Senior code reviewer"
- **Vocabulary:** finding classification, evidence-backed verdict, review dimension, blocking vs non-blocking, severity assessment, remediation recommendation
- **Anti-patterns:** Rubber-stamp approval (MAST FM-3.1), dimension bleed, prose commentary, severity inflation

All specialist reviewers inherit from `reviewer` and add domain-specific vocabulary.

#### `reviewer-conformance`

- **Inherits:** `reviewer`
- **Identity:** "Senior requirements verification engineer"
- **Additional vocabulary:** acceptance criteria traceability, spec requirement mapping, gap analysis, criterion-by-criterion verification, deviation classification, conformance matrix
- **Additional anti-patterns:** Assumed conformance (marking pass without tracing to requirement), partial verification (checking some criteria but not all), phantom requirement (finding issues against unstated requirements)

#### `reviewer-quality`

- **Inherits:** `reviewer`
- **Identity:** "Senior software quality engineer"
- **Additional vocabulary:** cyclomatic complexity, error handling chain, defensive copying, invariant assertion, contract violation, resource lifecycle (open/close pairing), naming consistency, package cohesion, dead code detection
- **Additional anti-patterns:** Style-as-defect (flagging style preferences as quality issues), nitpick escalation (trivial issues marked as blocking), improvement suggestion disguised as defect

#### `reviewer-security`

- **Inherits:** `reviewer`
- **Identity:** "Senior application security engineer"
- **Additional vocabulary:** (see the full example in §3.1 above)
- **Additional anti-patterns:** (see the full example in §3.1 above)

#### `reviewer-testing`

- **Inherits:** `reviewer`
- **Identity:** "Senior test engineer"
- **Additional vocabulary:** boundary value analysis, equivalence partitioning, test isolation, fixture management, assertion specificity, coverage metric (statement, branch, path), mutation testing signal, test pyramid (unit, integration, e2e), flaky test detection, test-as-documentation
- **Additional anti-patterns:** Coverage theater (high coverage numbers with weak assertions), mock overuse (testing the mock, not the code), happy-path-only testing, test coupling (tests that break when unrelated code changes), assertion-free tests

### 4.4 Coordination Roles

#### `orchestrator`

- **Identity:** "Senior engineering manager coordinating an agent team"
- **Vocabulary:** task decomposition, work unit boundary, handoff protocol, parallel dispatch, conflict detection, dependency ordering, cascade escalation, review collation, remediation routing, checkpoint placement
- **Anti-patterns:** Over-decomposition (splitting tasks below useful granularity), under-decomposition (monolithic tasks that exceed context budget), context forwarding (dumping full context to sub-agents instead of scoped packets), result-without-evidence (accepting sub-agent output without checking for evidence)
- **Used in stages:** `reviewing` (as the coordinator), complex `dev-planning`
- **Special:** The orchestrator role also carries knowledge of team economics — the 45% threshold, the saturation point at 4 agents, and the cascade pattern. These are hard constraints, not suggestions.

---

## 5. The Skill Catalog

Skills map to specific workflow activities. Each skill follows the attention-optimized structure defined in §3.2.

### 5.1 Document Authoring Skills

| Skill | Stage | Produces | Paired Roles |
|-------|-------|----------|--------------|
| `write-design` | designing | Design document (draft) | architect |
| `write-spec` | specifying | Specification with acceptance criteria (draft) | spec-author |
| `write-dev-plan` | dev-planning | Implementation plan with task breakdown (draft) | architect |
| `write-research` | researching | Research report | researcher |
| `update-docs` | documenting | Updated documentation files | documenter |

Each authoring skill carries:
- **Vocabulary** specific to the document type (e.g., `write-spec` has requirement writing vocabulary)
- **Anti-patterns** specific to common authoring failures for that document type
- **Output format** matching the Kanbanzai document structure for that type
- **Examples** showing BAD vs GOOD document excerpts for the type
- **Evaluation criteria** phrased as gradable questions about the output

#### Document type–skill alignment

The system enforces that documents produced during a stage match the expected type:

| Stage | Expected document type | SKILL that produces it |
|-------|----------------------|------------------------|
| designing | `design` | `write-design` |
| specifying | `specification` | `write-spec` |
| dev-planning | `dev-plan` | `write-dev-plan` |
| reviewing | `report` | `review-code` / `orchestrate-review` |
| researching | `research` | `write-research` |

This replaces the current `document-creation` SKILL, which is generic across all document types. Type-specific skills carry type-specific vocabulary and anti-patterns — a specification author needs requirements vocabulary, not generic document creation procedure.

### 5.2 Implementation Skills

| Skill | Stage | Produces | Paired Roles |
|-------|-------|----------|--------------|
| `implement-task` | developing | Code changes + tests | implementer-go |
| `decompose-feature` | dev-planning | Task breakdown (via `decompose` tool) | architect |

`implement-task` is the skill for individual task execution. It is deliberately lean — the implementer role carries the language-specific expertise, and the task itself carries the spec requirements. The skill's job is to provide the procedure (read spec → implement → test → verify) and anti-patterns (scope creep, untested code paths, spec deviation).

`decompose-feature` guides the use of the `decompose` tool with vocabulary for dependency analysis, vertical slicing, and sizing. It carries anti-patterns for common decomposition failures: over-decomposition, circular dependencies, and missing integration tasks.

### 5.3 Review Skills

| Skill | Stage | Produces | Paired Roles |
|-------|-------|----------|--------------|
| `review-code` | reviewing | Structured review findings | reviewer-* (all specialist reviewers) |
| `orchestrate-review` | reviewing | Collated review report + remediation plan | orchestrator |
| `review-plan` | plan-reviewing | Plan review report | reviewer-conformance |

`review-code` is the sub-agent skill — one sub-agent, one review unit, one structured output. It is used by ALL specialist reviewers (`reviewer-conformance`, `reviewer-quality`, `reviewer-security`, `reviewer-testing`) — the specialisation comes from the role's vocabulary, not from the skill's procedure. This is the composition pattern: same procedure, different expertise.

`orchestrate-review` is the coordinator skill — it decomposes a feature into review units, dispatches specialist sub-agents, collates findings, and routes to remediation or approval. It carries orchestration-specific vocabulary and anti-patterns (result-without-evidence, over-decomposition).

`review-plan` handles plan-level review — checking that all features shipped, specs are approved, documentation is current. This is the existing plan-review SKILL, restructured to follow the attention-optimized format.

### 5.4 Skill Composition During Review

The review stage demonstrates the full composition model:

```
ORCHESTRATOR dispatches:

  Sub-agent 1:
    Role: reviewer-conformance  (spec traceability vocabulary)
    Skill: review-code          (review procedure + methodology vocabulary)
    Scope: files A, B, C + spec §3
    → Assembled context has BOTH vocabulary sets

  Sub-agent 2:
    Role: reviewer-quality      (code quality vocabulary)
    Skill: review-code          (same review procedure)
    Scope: files A, B, C + spec §3
    → Same procedure, different expertise lens

  Sub-agent 3:
    Role: reviewer-security     (security vocabulary)
    Skill: review-code          (same review procedure)
    Scope: files A, B, C + spec §3
    → Same procedure, security-focused lens

  Sub-agent 4:
    Role: reviewer-testing      (testing vocabulary)
    Skill: review-code          (same review procedure)
    Scope: files D (test files) + spec §3
    → Testing specialist reads the tests
```

Four specialists, each seeing the same code through a different vocabulary lens. Each produces structured findings in the same output format. The orchestrator collates, deduplicates, and creates the aggregate verdict.

This directly implements the Specialized Review Principle (P6): "A panel of brief specialists (<50 tokens each) outperforms a single elaborate generalist."

**When fewer reviewers are sufficient:** The binding registry declares max 4 sub-agents, but the orchestrator uses adaptive composition (Captain Agent research — 15–25% better than static teams). Small features (≤10 files) may only need 1–2 reviewers. The orchestrator selects based on the files changed: if no security-relevant code changed, the security reviewer is not dispatched.

---

## 6. How Context Assembly Works

### 6.1 The Assembly Pipeline

```
handoff(task_id="TASK-01KN...")
  │
  ├── 1. Resolve task → parent feature → feature lifecycle stage
  │       (e.g., feature is in "developing" → stage is "developing")
  │
  ├── 2. Look up stage binding
  │       ("developing" → roles: [implementer-go], skills: [implement-task])
  │
  ├── 3. Resolve role with inheritance
  │       implementer-go → implementer → base
  │       Merge: base.conventions + implementer.vocabulary + implementer-go.vocabulary
  │       Merge: base.anti_patterns + implementer.anti_patterns + implementer-go.anti_patterns
  │
  ├── 4. Load skill
  │       implement-task → vocabulary, anti-patterns, procedure, output format,
  │       examples, evaluation criteria, retrieval anchors
  │
  ├── 5. Surface relevant knowledge entries
  │       Query knowledge base for entries matching:
  │       - Task file paths
  │       - Parent feature scope
  │       - Role domain tags
  │       Auto-include entries tagged "always" or matching task scope
  │
  ├── 6. Filter MCP tool definitions
  │       Include only tools listed in the role's `tools` field
  │       (implementer-go needs: entity, read_file, edit_file, grep, terminal, diagnostics, ...)
  │       (does NOT need: decompose, retro, merge, pr, ...)
  │
  ├── 7. Estimate token budget
  │       Sum: role context + skill context + knowledge entries + tool definitions + task specifics
  │       IF > 40% of context window → WARN
  │       IF > 60% of context window → REFUSE and suggest splitting
  │
  └── 8. Assemble in attention-curve order
          Output the assembled context as a structured prompt:
          [identity] [vocabulary] [anti-patterns] [procedure] [format]
          [examples] [knowledge] [eval criteria] [retrieval anchors]
```

### 6.2 Token Budget Management

The research identifies the optimal context utilisation zone as 15–40% of the window.

**Layer 1 — Always loaded (~300–500 tokens):**
- Base role identity and hard constraints
- Specific role identity and vocabulary

**Layer 2 — Task-triggered (~500–2,000 tokens):**
- Skill procedure and anti-patterns
- Output format and examples

**Layer 3 — On-demand (2,000+ tokens):**
- Full spec sections (loaded per review unit, not per feature)
- Reference documents from `references/` directories
- Extended anti-pattern documentation

**Layer 4 — Compressed (variable):**
- Summaries of large documents (when full document exceeds budget)
- Collated findings from prior review passes

The assembly pipeline estimates token cost at each layer and stops loading when the budget is met. Layer 3 and 4 content is loaded only when the procedure explicitly calls for it (e.g., "Read the spec section" in the review-code procedure).

### 6.3 Knowledge Auto-Surfacing

Currently, knowledge entries are available via `knowledge(action: "list")`. In the redesigned system, relevant knowledge entries are automatically included in the assembled context based on:

1. **File path matching:** If the task involves `internal/storage/`, include knowledge entries scoped to `internal/storage/` or to YAML serialisation topics.
2. **Tag matching:** If the role is `reviewer-security`, include knowledge entries tagged `security`.
3. **Explicit "always" entries:** Knowledge entries tagged `always` or with project scope are included in every context assembly.
4. **Recency weighting:** Prefer recently confirmed entries over stale ones.

The auto-surfaced entries appear in the assembled context at position 7 (after examples, before evaluation criteria) — in the rising-attention zone near the end of context. They are formatted as "Always/Never X BECAUSE Y" entries, following the format the research identifies as most effective.

---

## 7. Migration from Current System

### 7.1 What Changes

| Current | New | Migration |
|---------|-----|-----------|
| `.kbz/context/roles/base.yaml` | `.kbz/roles/base.yaml` — gains vocabulary, anti-patterns, tools fields | Restructure, add new fields |
| `.kbz/context/roles/developer.yaml` | `.kbz/roles/implementer-go.yaml` — renamed, gains vocabulary | Rename, restructure |
| `.kbz/context/roles/reviewer.yaml` | `.kbz/roles/reviewer.yaml` + 4 specialist subtypes | Split into hierarchy |
| `.skills/code-review.md` | `.kbz/skills/review-code/SKILL.md` + `orchestrate-review/SKILL.md` | Split, restructure to attention-curve format |
| `.skills/plan-review.md` | `.kbz/skills/review-plan/SKILL.md` | Restructure to attention-curve format |
| `.skills/document-creation.md` | `.kbz/skills/write-design/`, `write-spec/`, etc. | Split by document type |
| No stage bindings | `.kbz/stage-bindings.yaml` | New file |
| No vocabulary payloads | Vocabulary fields in all roles and skills | New content |
| No named anti-patterns | Anti-pattern sections in all roles and skills | New content |

### 7.2 What Stays the Same

- **Profile inheritance mechanism** (`internal/context/resolve.go`) — the inheritance resolution logic is retained and extended to handle vocabulary and anti-pattern merging.
- **Context assembly function** (`internal/context/assemble.go`) — extended to implement the attention-curve ordering and token budget estimation, but the same architectural role.
- **Feature lifecycle state machine** (`internal/validate/lifecycle.go`) — unchanged. The binding registry maps TO the existing states; it does not change them.
- **Document types and stages** — unchanged. The binding registry maps FROM the existing document types.
- **Knowledge system** — unchanged except for the auto-surfacing addition.
- **MCP tool surface** — unchanged. Tool filtering is additive (selecting a subset), not modifying tools.

### 7.3 New Files Location

```
.kbz/
├── roles/                        # was: context/roles/
│   ├── base.yaml                 # project-wide foundation
│   ├── architect.yaml
│   ├── spec-author.yaml
│   ├── implementer.yaml
│   ├── implementer-go.yaml       # was: developer.yaml
│   ├── reviewer.yaml             # base review role
│   ├── reviewer-conformance.yaml
│   ├── reviewer-quality.yaml
│   ├── reviewer-security.yaml
│   ├── reviewer-testing.yaml
│   ├── orchestrator.yaml
│   ├── researcher.yaml
│   └── documenter.yaml
├── skills/                       # was: .skills/ at project root
│   ├── write-design/
│   │   ├── SKILL.md
│   │   └── references/
│   ├── write-spec/
│   │   ├── SKILL.md
│   │   └── references/
│   ├── write-dev-plan/
│   │   ├── SKILL.md
│   │   └── references/
│   ├── write-research/
│   │   ├── SKILL.md
│   │   └── references/
│   ├── implement-task/
│   │   ├── SKILL.md
│   │   └── references/
│   ├── decompose-feature/
│   │   ├── SKILL.md
│   │   └── references/
│   ├── review-code/
│   │   ├── SKILL.md
│   │   └── references/
│   ├── orchestrate-review/
│   │   ├── SKILL.md
│   │   └── references/
│   ├── review-plan/
│   │   ├── SKILL.md
│   │   └── references/
│   └── update-docs/
│       ├── SKILL.md
│       └── references/
└── stage-bindings.yaml           # new: explicit stage→role→skill mapping
```

### 7.4 Compatibility

The `.skills/` directory at the project root is retained during migration for backward compatibility. Agents that read `.skills/code-review.md` directly will continue to find it. Once all references are updated to the new system, the old `.skills/` files are retired.

The `profile(action: "get")` and `handoff` tools are extended (not replaced) to support the new role and skill structures. Existing callers that request `profile(action: "get", id: "reviewer")` continue to work.

---

## 8. Expected Impact

### 8.1 Metrics to Track

| Metric | Current Baseline | Target | Source Principle |
|--------|-----------------|--------|-----------------|
| First-attempt convention compliance | Unknown (not tracked) | >90% | P5 (Institutional Memory) |
| Review finding specificity | Moderate (prose-heavy) | 100% structured, evidence-backed | P6 (Specialized Review) |
| Stale-doc-caused errors | Occasional | Zero | P3 (Living Documentation) |
| Context assembly token utilisation | Unknown | 15–40% of window | P2 (Context Hygiene) |
| Review rubber-stamp rate | Unknown | <15% clean verdicts without evidence | P8 (Strategic Human Gate) |
| Sub-agent dispatch per feature | Variable | 1 (70% of tasks), 2–4 (30%) | P9 (Token Economy) |
| MAST failure mode incidents | Unknown | Tracked and trending down | P7 (Observability) |

### 8.2 Research-Backed Predictions

Based on the cited research:

- **Vocabulary routing** (Ranjan et al., 2024): Expect measurable improvement in domain-specific output quality — the model activates expert knowledge clusters instead of generic training data.
- **Attention-optimized structure** (Liu et al., 2024): Expect fewer missed constraints and anti-patterns, particularly for instructions that were previously in the "dead zone" middle of context.
- **Specialist panel** (PRISM, Ranjan et al.): Expect security, quality, and testing issues caught at rates approaching the research's reported 95% for specialist vs 40% for generalist (security domain).
- **Named anti-patterns** (CHI 2023): Expect repeat errors on codified anti-patterns to drop to near-zero within two weeks of deployment.
- **Token budget management** (DeepMind, 2025): Expect token cost reduction of 40–60% from adaptive tool loading and cascade escalation.
- **BAD/GOOD examples** (LangChain, 2024): Expect output format compliance to improve significantly — the model pattern-matches against demonstrated examples more reliably than it follows written rules.

---

## 9. Design Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| DD-1 | Roles and skills are separate files that compose | Roles carry domain expertise (who); skills carry procedures (what). The same role uses different skills at different stages. The same skill is used by different specialist reviewers. Composition gives both reuse and specificity. |
| DD-2 | Vocabulary is a first-class YAML field in roles, not embedded in prose | Vocabulary is the primary routing mechanism. It must be explicit, auditable, and mechanically mergeable during context assembly. Prose-embedded vocabulary cannot be reliably extracted or composed. |
| DD-3 | Anti-patterns use a BECAUSE clause | The BECAUSE clause makes rules generalisable to adjacent cases (CHI 2023). Without it, rules cover only the literal case stated. With it, the model extends the principle to novel situations. |
| DD-4 | Skills follow a fixed section ordering | The attention curve (Liu et al., 2024; Wu et al., 2025) is architectural, not patchable. The ordering is not a suggestion — it is a design constraint that maps content to attention weight. |
| DD-5 | Evaluation criteria are separated from procedure | The agent doing the work must not self-evaluate (Anthropic, Mar 2026). Evaluation criteria enable a separate pass — human or automated — to assess output quality using gradable questions. |
| DD-6 | Stage bindings are declared, not inferred | Implicit bindings (agents reading AGENTS.md and inferring the right context) are a fuzzy step that should be hardened (P1). Declared bindings are deterministic, auditable, and testable. |
| DD-7 | Maximum 4 concurrent sub-agents per stage | DeepMind (2025) shows team effectiveness saturates at 3–4 agents. Beyond that, coordination overhead exceeds marginal benefit. This is a hard constraint, not a guideline. |
| DD-8 | Tool filtering per role | Every loaded tool definition consumes attention budget. A security reviewer does not need `decompose`. Filtering reduces context size and focuses attention on relevant tools. Implements the jig pattern from P10. |
| DD-9 | Skills and roles move inside `.kbz/` | The `.kbz/` directory is the instance root for all Kanbanzai state. Skills and roles are operational configuration, not project documentation. They belong with the system state, not at the project root. This also makes them manageable by MCP tools. |
| DD-10 | Document-type-specific authoring skills replace generic `document-creation` | A specification needs requirements vocabulary; a design needs architecture vocabulary. A generic skill activates neither. Type-specific skills carry type-specific vocabulary, anti-patterns, and examples. |

---

## 10. Open Questions

1. **Should vocabulary payloads be curated per-project or shipped as defaults?** The current design assumes vocabulary is project-specific (e.g., `implementer-go` for this project). Should Kanbanzai ship default vocabulary payloads for common languages and domains, with project-level overrides?

2. **How should the system handle roles that don't exist yet?** If a stage binding references `reviewer-security` but the project hasn't created that role file, should the system fall back to the parent `reviewer` role, or refuse to proceed?

3. **Should skills carry their own test fixtures?** The `references/` directory could include test inputs and expected outputs for the skill, enabling automated quality assurance of skill output. Is this worth the added complexity?

4. **How does this interact with the `decompose` tool's task creation?** When `decompose` creates tasks, should it also tag each task with the expected stage and role, so `handoff` can resolve the binding automatically?

5. **Should vocabulary payloads have expiry dates?** Domain vocabulary evolves. "OWASP Top 10 (2021)" will eventually be superseded. Should vocabulary terms carry freshness metadata, like knowledge entries do?

6. **What is the right granularity for reviewer specialisation?** Four specialists (conformance, quality, security, testing) maps to the current five review dimensions minus documentation (which is a cross-cutting concern). Should documentation currency be a specialist or remain a dimension within conformance review?

---

## 11. Relationship to Other Documents

| Document | Relationship |
|----------|-------------|
| `work/research/ai-agent-best-practices-research.md` | Research basis — this design applies the 20 recommendations from that report |
| `work/design/machine-context-design.md` | The context assembly model this design extends |
| `work/design/agent-interaction-protocol.md` | Agent behaviour conventions this design must respect |
| `work/design/quality-gates-and-review-policy.md` | Review policy this design operationalises |
| `work/design/document-centric-interface.md` | Document-led workflow model this design integrates with |
| `work/spec/phase-2b-specification.md` | The context profile system this design supersedes |