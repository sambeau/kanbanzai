# Design: Evidence-Based Skills and Roles System for Kanbanzai 3.0

| Field | Value |
|-------|-------|
| Date | 2025-07-30 |
| Status | Draft |
| Author | Design Agent |
| Based on | `work/design/skills-system-redesign.md` (v1, now superseded) |
| Informed by | `work/research/ai-agent-best-practices-research.md`, `work/research/agent-skills-research.md`, `work/research/agent-orchestration-research.md` |
| Supersedes | Current `.skills/` system and `.kbz/context/roles/` profiles |
| Changes from v1 | Cross-document alignment per `work/reviews/3.0-design-cross-document-alignment.md` |

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

### DP-9: Match Constraint Level to Task Risk

Different tasks need different degrees of freedom. The cost of over-constraining creative work is mediocre output; the cost of under-constraining fragile operations is broken state.

- **Low freedom** (exact tool call sequences, deterministic scripts): lifecycle transitions, document registration, stage gate checks. One safe path — take it.
- **Medium freedom** (templates with flexibility): specification writing, plan creation, structured review. A preferred pattern exists; variation is acceptable within bounds.
- **High freedom** (general guidance, trust the agent): design work, implementation, research. Many valid approaches; context determines the best one.

Each skill must declare its constraint level and match its procedure style accordingly. Low-freedom skills provide exact commands. High-freedom skills provide principles and vocabulary. Do not apply uniform medium freedom to everything.

**Constraint levels map to gate enforcement mechanisms.** Masters et al. (2025) distinguish hard constraints (ℋ — violation terminates the workflow) from soft constraints (𝒮 — violation incurs penalties). These align directly with constraint levels:

- **Low freedom → Hard gate enforcement.** The MCP server rejects violations. Lifecycle transitions are blocked if document prerequisites aren't met. Stage gate checks are programmatic, not advisory.
- **Medium freedom → Soft gate enforcement.** The MCP server warns but allows. Document section completeness, cross-reference coverage, and output format compliance are flagged during review, not blocked at transition.
- **High freedom → No gate enforcement.** The system trusts the agent. Design choices, research direction, and implementation approach are guided by vocabulary routing and anti-patterns, not by system-level checks.

Skills that declare `constraint_level: low` should expect the system to *enforce* their procedure, not just advise it. This is the link between the skills layer and the MCP server's gate enforcement.

*Source: Anthropic Skill Authoring Best Practices ("narrow bridge vs. open field" analogy); reinforced by Vaarta Analytics (2026) findings on instruction granularity. Gate enforcement alignment from Masters et al. (2025) hard/soft constraint model and orchestration recommendations §4.3.*

### DP-10: Only Add What the Model Doesn't Know

The context window is a shared resource. Every token in a skill must justify its presence against the question: "Does the model already know this?" The model knows what state machines are, what lifecycle transitions mean, and how to write Go. It does not know Kanbanzai's specific state machine, Kanbanzai's specific lifecycle rules, or this project's specific conventions.

Strip explanations of general concepts. Focus exclusively on project-specific knowledge, constraints, and vocabulary. A skill that is 200 tokens of Kanbanzai-specific content outperforms one that is 800 tokens of general explanation with the same 200 tokens buried inside.

*Source: Anthropic Skill Authoring Best Practices ("Claude is already very smart — only add context Claude doesn't already have"); Anthropic context engineering guide (Sep 2025).*

### DP-11: Descriptions Must Be Assertive

The model tends to under-trigger skills — to not use them when they would be useful. Skill descriptions must actively push for triggering, especially for workflow-critical skills. Both the `expert` and `natural` description registers should include assertive "use even when..." clauses that combat the default tendency to skip.

Example: Instead of "Use when deciding what workflow stage work belongs to," write "Use when deciding what workflow stage work belongs to. Use even when the agent is confident about the next step — workflow errors are expensive to undo."

*Source: Anthropic skill-creator meta-skill ("please make the skill descriptions a little bit 'pushy'"); Anthropic Skill Authoring Best Practices (description quality determines discovery).*

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
│   ├── references/        (one level deep from SKILL.md — see constraint below)
│   │   ├── finding-classification.md
│   │   ├── edge-cases.md
│   │   └── evaluation-rubric.md
│   └── scripts/           (deterministic operations — output enters context, not source)
│       └── check-prerequisites.sh
├── write-spec/
│   ├── SKILL.md
│   ├── references/
│   │   ├── acceptance-criteria-patterns.md
│   │   └── spec-anti-patterns.md
│   └── scripts/
│       └── validate-spec-structure.sh
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
constraint_level: medium  # low | medium | high (see DP-9)
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

## Checklist

(Optional — included for workflow-critical and medium/low-freedom skills.
Agents copy this into their response and check off items as they progress.
This makes compliance visible and prevents step-skipping.)

```
Copy this checklist and track your progress:
- [ ] Read spec section(s) fully
- [ ] Read all files in file list
- [ ] Confirm review profile and required dimensions
- [ ] Evaluate spec_conformance dimension independently
- [ ] Evaluate implementation_quality dimension independently
- [ ] Evaluate test_adequacy dimension independently
- [ ] Evaluate security dimension (if required by profile)
- [ ] Classify all findings (blocking vs non-blocking)
- [ ] Verify every blocking finding has a spec anchor
- [ ] Produce structured output in required format
```

## Procedure

### Step 1: Orient from inputs

1. Read the spec section(s) fully. Understand what was required.
2. Read all files in the file list. Understand what was implemented.
3. Note the review profile — this determines required dimensions.
4. IF any input is missing → STOP. Report Missing Context edge case.
5. IF the spec is ambiguous or incomplete for any dimension → STOP.
   Report the ambiguity. Do not infer intent. (See §8.3: Uncertainty Protocol.)

### Step 2: Evaluate each dimension independently

For each required dimension, work through its specific evaluation
questions. Record a per-dimension outcome. Do not let a poor result
in one dimension affect your assessment of another.

### Step 3: Validate and iterate

Validate findings against classification criteria:
- Does every blocking finding cite a specific spec requirement?
- Is any dimension's verdict influenced by another dimension's result?
- IF validation fails → fix the issue → re-validate.
Repeat until all findings pass validation. Only then produce final output.

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
- **Evaluation criteria are SEPARATED from the procedure.** The agent that does the work should not self-evaluate (Anthropic harness design, Mar 2026). Evaluation criteria are phrased as gradable questions with weights so a separate evaluation pass (or human) can assess output quality. Note: the research template (Appendix B) places evaluation criteria in `references/evaluation-criteria.md` (Layer 3, on-demand). This design deliberately keeps them in the SKILL.md body because they benefit from the high-attention end-of-context position and are needed by every evaluation pass — making them on-demand would risk them not being loaded when the evaluator needs them most. **These criteria are designed to be usable by an LLM-as-judge automated evaluation pass** — Anthropic's multi-agent research found that "a single LLM call with a single prompt outputting scores from 0.0–1.0 and a pass-fail grade was the most consistent and aligned with human judgements." The gradable question format and 0.0–1.0 weight scale support this directly. The automated evaluation mechanism itself is an observability concern (see §10.1 scope note and companion observability design), but the criteria here are its inputs.
- **"Questions This Skill Answers" is LAST.** This is a retrieval anchor — it sits at the high-attention end of context. It also serves as the trigger matching surface when an orchestrator or the system needs to select the right skill for an ambiguous request.
- **`references/` directory for overflow.** The SKILL.md stays under 500 lines. Detailed anti-pattern documentation, extended examples, and evaluation rubrics go in `references/` and are loaded on-demand (progressive disclosure Layer 3). **Critical constraint: all reference files must link directly from SKILL.md, not from each other.** The model may partially read files that are two references deep. Keep references one level deep.
- **`scripts/` directory for deterministic operations.** Operations that are exact and repeatable — checking lifecycle prerequisites, validating document structure, verifying stage gate conditions — should be executable scripts, not prose instructions for the agent to interpret. Only script *output* enters the context window, not the script source. If agents consistently perform the same multi-step operation across tasks, that is a signal the operation should be bundled as a script.
- **`## Checklist` is an optional section** for workflow-critical and medium/low-freedom skills. Agents copy the checklist into their response and tick off items as they progress. This makes compliance visible, creates a trackable record, and directly addresses step-skipping. High-freedom skills (design, research) do not need checklists.
- **Procedures include validate → fix → repeat loops** for fragile operations. Linear step sequences risk the agent proceeding past failures. An explicit iteration point ("validate → if not met → fix → re-validate") ensures prerequisites are met before proceeding.
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
    orchestration: single-agent      # sequential reasoning — do not delegate
    roles: [architect]
    skills: [write-design]
    document_type: design
    human_gate: false
    prerequisites: {}                # no document prerequisites for the first stage
    notes: >
      Single agent. Architect vocabulary routes to system design expertise.
      Design review is human-led via document approval, not automated,
      because design is high-freedom creative work (DP-9). The specifying
      stage's prerequisites enforce that the design document is approved
      before specification begins — the human is the quality gate here.
    effort_budget: "5–15 tool calls. Read related designs, query decisions, draft structured document."

  specifying:
    description: "Writing a formal specification with acceptance criteria"
    orchestration: single-agent      # sequential reasoning — do not delegate
    roles: [spec-author]
    skills: [write-spec]
    document_type: specification
    human_gate: true                 # Spec approval is a required gate
    prerequisites:
      documents:
        - type: design
          status: approved
    notes: "Single agent. Spec-author vocabulary routes to requirements engineering."
    effort_budget: "5–15 tool calls. Read design document, query knowledge, check related decisions, draft each required section."

  dev-planning:
    description: "Breaking a spec into an implementation plan and tasks"
    orchestration: single-agent      # sequential reasoning — decomposition quality is the critical path
    roles: [architect]
    skills: [write-dev-plan, decompose-feature]
    document_type: dev-plan
    human_gate: true                 # Plan review before implementation
    prerequisites:
      documents:
        - type: specification
          status: approved
    notes: "Single agent. Decomposition uses architect vocabulary for dependency analysis."
    effort_budget: "5–10 tool calls. Read spec, decompose into tasks with dependencies, estimate effort, produce plan document."

  # ── Implementation stage ──────────────────────────────────

  developing:
    description: "Implementing tasks from the dev plan"
    orchestration: orchestrator-workers  # parallelisable — dispatch independent tasks
    roles: [orchestrator]                # orchestrator coordinates dispatch
    skills: [orchestrate-development]    # development coordination skill
    sub_agents:
      roles: [implementer]
      skills: [implement-task]
      topology: parallel
      max_agents: null               # limited by task count, not a fixed cap (see notes)
    document_type: null
    human_gate: false
    prerequisites:
      documents:
        - type: dev-plan
          status: approved
      tasks:
        min_count: 1                 # at least one task must exist
    notes: >
      Orchestrator dispatches implementer sub-agents in parallel, one per
      task. Implementer role carries language-specific vocabulary (e.g.,
      implementer-go for this project). No fixed agent cap — implementation
      tasks are independent by construction and don't share the coordination
      overhead that limits specialist panels. Cascade: start Level 0
      (single agent + tools). Escalate only if measured output is below
      45% threshold.
    effort_budget: "10–50 tool calls per task. Read spec section, implement, test, iterate."

  # ── Review stage ──────────────────────────────────────────

  reviewing:
    description: "Evaluating implementation against the specification"
    orchestration: orchestrator-workers  # parallelisable — specialist panel
    roles: [orchestrator]
    skills: [orchestrate-review]
    sub_agents:
      roles: [reviewer-conformance, reviewer-quality, reviewer-security, reviewer-testing]
      skills: [review-code]
      topology: parallel
      max_agents: 4                  # DeepMind saturation point for specialist panels
    document_type: report
    human_gate: true                 # Verdict checkpoint
    max_review_cycles: 3             # escalate to human after 3 fail-rework cycles
    prerequisites:
      tasks:
        all_terminal: true           # all tasks must be done or not-planned
    notes: >
      Multi-agent. Orchestrator dispatches specialist reviewers in parallel.
      Each reviewer gets its own vocabulary-routed role + the shared
      review-code skill. Max 4 concurrent sub-agents (DeepMind saturation
      point for specialist panels sharing coordination overhead).
      Orchestrator never reads source — metadata only. Review-rework loop
      capped at 3 cycles to prevent infinite refinement (Microsoft
      maker-checker pattern).
    effort_budget: "5–10 tool calls per review dimension."

  # ── Plan review stage ─────────────────────────────────────

  plan-reviewing:
    description: "Reviewing a completed plan for aggregate delivery"
    orchestration: single-agent
    roles: [reviewer-conformance]
    skills: [review-plan]
    document_type: report
    human_gate: true
    prerequisites: {}
    notes: "Single agent. Plan review is conformance-focused by nature."
    effort_budget: "5–10 tool calls. Read plan, check feature delivery, verify documentation currency."

  # ── Research and documentation stages ─────────────────────

  researching:
    description: "Producing a research report or analysis"
    orchestration: single-agent
    roles: [researcher]
    skills: [write-research]
    document_type: research
    human_gate: false
    prerequisites: {}
    effort_budget: "10–30 tool calls. Gather sources, synthesise findings, draft structured report."

  documenting:
    description: "Updating project documentation for currency"
    orchestration: single-agent
    roles: [documenter]
    skills: [update-docs]
    document_type: null
    human_gate: false
    prerequisites: {}
    effort_budget: "5–15 tool calls. Identify stale docs, update content, verify cross-references."
```

**Key design decisions:**

- **One stage, one binding.** No ambiguity about what to load. The system looks at the feature's current lifecycle state and knows exactly which roles, skills, and tools to provide.
- **`orchestration` is a first-class field.** Each binding declares its orchestration pattern: `single-agent` (sequential reasoning — the agent does the work directly) or `orchestrator-workers` (parallelisable — the agent dispatches sub-agents). This is not a hint; the context assembly pipeline includes this in the assembled context so agents know whether to delegate or work directly. Google Research found multi-agent coordination degrades sequential reasoning tasks by 39–70% but improves parallelisable tasks by 81% — the pattern must match the task structure. *(Source: orchestration recommendations §2.3)*
- **`prerequisites` make stage gates enforceable.** Each binding declares the document and task prerequisites required before a feature can transition *into* that stage. The `entity(action: "transition")` tool checks these prerequisites and rejects the transition if they aren't met. This converts "agents skip steps" from a quality problem into an impossibility — the system literally prevents it. *(Source: MetaGPT SOPs, Masters et al. hard constraints ℋ, orchestration recommendations §2.1)*
- **`sub_agents` for multi-agent stages.** The `reviewing` and `developing` stages use multiple agents. The binding declares the sub-agent configuration — roles, skills, topology, and agent caps. The cap of 4 applies to specialist panels sharing coordination overhead (review). Implementation tasks are independent by construction and don't share that overhead, so `developing` has no fixed cap. *(Source: DeepMind saturation point; Google Research alignment principle)*
- **`max_review_cycles` prevents infinite refinement.** The reviewing binding declares a maximum number of review-rework cycles before escalating to human decision. This is a hard constraint derived from Microsoft's maker-checker pattern — without an iteration cap, review loops can continue indefinitely with diminishing returns. *(Source: orchestration recommendations §2.2)*
- **`effort_budget` embeds effort expectations.** Each binding declares the expected effort range for the stage. This is included in the assembled context so agents know how much work to invest. Anthropic found that "agents struggle to judge appropriate effort for different tasks" and that embedding scaling rules in prompts directly addresses this. *(Source: Anthropic multi-agent system, orchestration recommendations §5.1)*
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
| 3 | Effort budget and orchestration pattern | Stage binding `effort_budget` + `orchestration` | **High** |
| 4 | Combined vocabulary payload | Role vocab + Skill vocab | **High** |
| 5 | Combined anti-pattern watchlist | Role anti-patterns + Skill anti-patterns | **Medium-High** |
| 6 | Skill procedure (numbered steps) | Selected skill | **Medium** (structured steps survive) |
| 7 | Output format + examples | Selected skill | **Rising** |
| 8 | Relevant knowledge entries | Knowledge system (auto-surfaced) | **Rising** |
| 9 | Evaluation criteria | Selected skill | **High** |
| 10 | Retrieval anchors ("Questions This Skill Answers") | Selected skill | **High** |

**Position 3 — Effort budget and orchestration pattern** appears early, in the high-attention zone, because it shapes the agent's entire approach to the task. A specification agent that sees "5–15 tool calls, single-agent task — do not delegate" in position 3 will calibrate its effort before reading the procedure. An implementation orchestrator that sees "orchestrator-workers, dispatch independent tasks in parallel" knows its role immediately. This directly addresses the Anthropic finding that agents cannot judge appropriate effort without explicit guidance, and the Google finding that the wrong orchestration pattern for the task type degrades quality by 39–70%.

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
- Orientation convention (inherited by all roles)
- Project-wide anti-patterns (inherited by all roles)

**Orientation convention** (carried by `base`, inherited by every role):

> On session start, call `status` to see current project state, then call `next` to see the work queue. Orient before acting. This applies to every agent — orchestrators arriving in a fresh context, sub-agents beginning a task, and any agent resuming work after a context switch.

This implements the orchestration research recommendation that every agent session begin with a structured orientation (Anthropic multi-agent system, orchestration recommendations §3.5). The `handoff`/`next` tools assemble context for sub-agents automatically, but the top-level orchestrator needs this convention to ensure it reads the current state before making dispatch decisions.

**Project-wide anti-patterns** (carried by `base`, inherited by every role):

- **Flattery Prompting**
  - **Detect:** Using superlatives or praise in sub-agent dispatch prompts ("expert," "world-class," "the best," "you excel at")
  - **BECAUSE:** PRISM research shows flattery activates motivational and marketing text patterns in the model's training distribution, degrading domain-specific output quality. Competence is defined by vocabulary payloads and anti-patterns, not adjectives.
  - **Resolve:** Remove all superlatives from dispatch prompts. Use the role's identity field (a real job title) and let the vocabulary do the routing work.

- **Silent Scope Expansion**
  - **Detect:** Adding features, refactoring, or making "improvements" not specified in the task or spec
  - **BECAUSE:** Undocumented design decisions made during implementation are expensive to discover during review and may conflict with the human's intent
  - **Resolve:** Implement only what the spec requires. If something seems missing, stop and ask.

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
- **Vocabulary:**
  - *Dispatch mechanics:* task decomposition, work unit boundary, handoff protocol, parallel dispatch, conflict detection, dependency ordering, cascade escalation, review collation, remediation routing, checkpoint placement
  - *Workflow governance:* lifecycle gate, stage prerequisite, transition prerequisite check, document approval cascade, hard constraint (ℋ), soft constraint (𝒮), stage binding lookup, orchestration pattern selection
  - *Quality assessment:* decomposition quality, vertical slice completeness, specification testability, review cycle count, escalation threshold, review verdict (pass / pass-with-notes / fail)
  - *Pattern matching:* sequential reasoning penalty, parallelisable task, single-agent sequential, orchestrator-workers parallel, maker-checker
- **Anti-patterns:**
  - Over-decomposition (splitting tasks below useful granularity)
  - Under-decomposition (monolithic tasks that exceed context budget)
  - Context forwarding (dumping full context to sub-agents instead of scoped packets)
  - Result-without-evidence (accepting sub-agent output without checking for evidence)
  - Reactive communication — **Detect:** excessive `status` calls and messages with few `decompose`, `handoff`, or structural actions. **BECAUSE:** Masters et al. (2025) found this pattern correlates with weaker orchestration outcomes; proactive orchestrators decompose 14.5× more and track dependencies 26× more. **Resolve:** structure work (decompose, add dependencies, refine tasks) before communicating about it.
  - Premature delegation — **Detect:** dispatching implementation sub-agents for a feature that hasn't completed specification. **BECAUSE:** Google Research found multi-agent coordination degrades sequential reasoning (specification, design, planning) by 39–70%. **Resolve:** complete sequential stages as a single agent before parallelising.
  - Infinite refinement loop — **Detect:** review cycle count exceeds `max_review_cycles` for a feature. **BECAUSE:** each review-rework cycle consumes context budget with diminishing returns; after 3 cycles the remaining issues are likely specification ambiguity, not implementation error. **Resolve:** escalate to human checkpoint with a summary of the recurring pattern.
- **Used in stages:** `developing` (as the coordinator dispatching implementers), `reviewing` (as the coordinator dispatching reviewers)
- **Special:** The orchestrator role also carries knowledge of team economics — the 45% threshold, the saturation point at 4 agents for specialist panels, and the cascade pattern. These are hard constraints, not suggestions. The orchestrator treats the binding registry as its decision table — it looks up the orchestration pattern and prerequisites for each stage rather than deciding them.

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

#### Gate-checkable document templates

Each authoring skill's output format must define a **gate-checkable template** — a set of structural requirements that the system can verify programmatically at stage transitions. This implements the MetaGPT finding that structured intermediate artifacts verified before the next stage begins reduce cascading errors.

Each authoring skill template defines:

1. **Required sections** — the section names that must be present (5–8 sections, following DP-6).
2. **Cross-reference requirements** — which other documents or entities must be cited. A specification must reference its parent design document. A dev-plan must reference its parent specification.
3. **Acceptance criteria format** — how acceptance criteria are expressed (e.g., Given/When/Then, or numbered testable assertions).

The required sections per document type are:

**Specification template (5 required sections):**

1. **Problem Statement** — What problem does this solve? Must reference the design document.
2. **Requirements** — Functional and non-functional, each with a unique ID (e.g., REQ-001).
3. **Constraints** — What must not change, what limits apply, what is out of scope.
4. **Acceptance Criteria** — Testable conditions for "done". Each must be verifiable.
5. **Verification Plan** — How will each acceptance criterion be verified? (test, inspection, demo)

**Dev-plan template (5 required sections):**

1. **Scope** — What this plan covers, referencing the specification.
2. **Task Breakdown** — Tasks with descriptions, dependencies, and effort estimates.
3. **Dependency Graph** — Which tasks depend on which, what can be parallelised.
4. **Risk Assessment** — What could go wrong, what mitigations exist.
5. **Verification Approach** — How will the plan verify that the spec is met?

**Design document template (4 required sections):**

1. **Problem and Motivation** — Why this design is needed.
2. **Design** — The proposed approach, with enough detail to write a specification from.
3. **Alternatives Considered** — What else was evaluated and why it was rejected.
4. **Decisions** — Architectural decisions made, with rationale.

These template definitions are the canonical source. Template enforcement (structural checks at stage gates) is designed in the workflow doc (§10.4–10.5).

To keep template content and gate enforcement in sync, the binding registry carries a `document_template` structure per document-producing stage:

```yaml
stage_bindings:
  specifying:
    # ...existing fields...
    document_template:
      required_sections:
        - "Problem Statement"
        - "Requirements"
        - "Constraints"
        - "Acceptance Criteria"
        - "Verification Plan"
      cross_references:
        - parent_design_document
      acceptance_criteria_format: "numbered-testable-assertions"
```

Both the skill (for output format guidance) and the gate check (for structural validation) read this structure. Changes to templates are made here and automatically affect both systems.

The `scripts/` directory for each authoring skill (DD-14) should include a validation script that checks these structural requirements. The script's output is used at stage gates: when a feature attempts to transition past the stage, the binding's `prerequisites` trigger the validation, and the system blocks if required sections are missing. Script output enters the context window; script source does not.

This creates the forcing function the research identifies: agents must engage with each required section, which distributes effort across the document and prevents the rush-to-implementation pattern. The template also provides the "clear acceptance criteria" that Microsoft's maker-checker pattern requires for consistent evaluation.

*Source: MetaGPT (structured intermediate artifacts), Anthropic (output format requirements), Microsoft (acceptance criteria for checker agents), orchestration recommendations §5.2.*

### 5.2 Implementation Skills

| Skill | Stage | Produces | Paired Roles |
|-------|-------|----------|--------------|
| `implement-task` | developing | Code changes + tests | implementer-go |
| `orchestrate-development` | developing | Coordinated parallel task dispatch | orchestrator |
| `decompose-feature` | dev-planning | Task breakdown (via `decompose` tool) | architect |

`implement-task` is the skill for individual task execution. It is deliberately lean — the implementer role carries the language-specific expertise, and the task itself carries the spec requirements. The skill's job is to provide the procedure (read spec → implement → test → verify) and anti-patterns (scope creep, untested code paths, spec deviation).

`orchestrate-development` is the coordinator skill for the developing stage — the counterpart of `orchestrate-review`. It guides the orchestrator through: reading the dev-plan, dispatching implementer sub-agents for independent tasks in parallel (respecting dependency ordering), monitoring progress, handling task failures, and performing context compaction between sequential sub-agent completions. It carries vocabulary for parallel task dispatch, dependency-order sequencing, progress monitoring, and sub-agent output handling (lightweight references, not full output). This skill exists because the developing stage is `orchestrator-workers` topology: the orchestrator needs its own context, distinct from what the implementing sub-agents receive.

**Context compaction guidance** (to be included in the skill's procedure): The orchestration research (Microsoft, Anthropic) identifies context growth across sequential sub-agent dispatches as a quality risk. The `orchestrate-development` skill should include three specific compaction techniques:

1. **Summarise after each sub-agent completes.** Reduce the sub-agent's outcome to 2–3 sentences and a task ID. Do not retain the full sub-agent output in conversation.
2. **Write progress summaries to documents.** If context utilisation exceeds 60% during a multi-task orchestration, write a progress summary to a registered document and start a fresh orchestration session that reads the summary.
3. **Structure multi-feature plans as a sequence of single-feature contexts.** Do not attempt to orchestrate all features in one session. Complete one feature's development tasks, write the summary, then begin the next feature.

These techniques are derived from Anthropic's finding that "direct subagent outputs can bypass the main coordinator for certain types of results, improving both fidelity and performance" and Microsoft's recommendation to "monitor accumulated context size and use compaction techniques between agents."

`decompose-feature` guides the use of the `decompose` tool with vocabulary for dependency analysis, vertical slicing, and sizing. It carries anti-patterns for common decomposition failures: over-decomposition, circular dependencies, and missing integration tasks. The orchestration research identifies decomposition quality as the single strongest predictor of workflow success (Masters et al.: "performance gains correlate almost linearly with the quality of the induced task graph"). This skill should invest disproportionately in decomposition validation: Do tasks have clear descriptions? Are dependencies declared? Are tasks sized for single-agent completion? Are there gaps (e.g., missing test tasks)?

*Tool-level validation checks for decomposition quality (description present, dependencies declared, sizing, testing coverage, orphan detection) are defined in the workflow doc (§11). The `decompose-feature` skill carries vocabulary and anti-patterns for decomposition quality; the `decompose` tool enforces structural validity.*

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
  ├── 0. Validate lifecycle state
  │       Check feature is in the correct state for this task.
  │       Reject with actionable error if not.
  │       (Requirement from workflow doc §7.2.)
  │
  ├── 1. Resolve task → parent feature → feature lifecycle stage
  │       (e.g., feature is in "developing" → stage is "developing")
  │
  ├── 2. Look up stage binding
  │       ("developing" → roles: [orchestrator], skills: [orchestrate-development],
  │        orchestration: orchestrator-workers, effort_budget: "10–50 tool calls per task")
  │
  ├── 3. Apply stage-specific inclusion/exclusion strategy
  │       Vary what context is included based on stage.
  │       (Requirement from workflow doc §7.3. See that section for the
  │        full inclusion/exclusion table per stage.)
  │
  ├── 4. Extract orchestration metadata
  │       From binding: orchestration pattern, effort budget, prerequisites (for validation),
  │       max_review_cycles (if reviewing stage)
  │       These are included at position 3 in the assembled context (high attention).
  │
  ├── 5. Resolve role with inheritance
  │       orchestrator → base
  │       Merge: base.conventions + orchestrator.vocabulary
  │       Merge: base.anti_patterns + orchestrator.anti_patterns
  │
  ├── 6. Load skill
  │       orchestrate-development → vocabulary, anti-patterns, procedure, output format,
  │       examples, evaluation criteria, retrieval anchors
  │
  ├── 7. Surface relevant knowledge entries
  │       Query knowledge base for entries matching:
  │       - Task file paths
  │       - Parent feature scope
  │       - Role domain tags
  │       Auto-include entries tagged "always" or matching task scope
  │
  ├── 8. Apply tool subset from role's `tools` field
  │       For the initial 3.0 release, generate tool subset guidance text
  │       in the assembled context. The design target is hard filtering
  │       (dynamically scoping the MCP tool list per session).
  │       (3.0 mechanism from workflow doc §9.2; design target from DD-8.)
  │
  ├── 9. Estimate token budget
  │       Sum: role context + skill context + orchestration metadata +
  │            knowledge entries + tool definitions + task specifics
  │       IF > 40% of context window → WARN
  │       IF > 60% of context window → REFUSE and suggest splitting
  │
  └── 10. Assemble in attention-curve order
           Output the assembled context as a structured prompt:
           [identity] [effort budget + orchestration pattern] [vocabulary]
           [anti-patterns] [procedure] [format]
           [examples] [knowledge] [eval criteria] [retrieval anchors]

           The effort budget and orchestration pattern appear at position 3
           (high attention) because they shape the agent's entire approach.
           A spec-author seeing "5–15 tool calls, single-agent — do not
           delegate" calibrates before reading the procedure.

           Within each section, order items so the most critical item
           appears LAST, exploiting recency bias (Liu et al., 2024).
           The most dangerous anti-pattern, the most important vocabulary
           term, and the most relevant knowledge entry should each appear
           at the end of their respective sections.
```

*This is the single canonical pipeline description. The workflow doc (§7) contributes stage-awareness requirements to this pipeline — specifically lifecycle state validation (§7.2), stage-specific inclusion/exclusion strategies (§7.3), and orchestration pattern signalling (§7.4).*

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

**Lean constraint on auto-surfacing:** The n=5-beats-n=19 finding (DP-6) applies to knowledge entries as well as to requirements. If more entries match than can fit within the knowledge portion of the token budget, the assembly pipeline must cap the number surfaced — top 10 by recency-weighted confidence score — rather than including all matches. When entries are dropped due to the cap, the assembly should log which entries were excluded so the orchestrator can request them explicitly if needed. If the knowledge base for a given scope grows past a threshold where the cap is routinely hit, the `health` tool should recommend compaction.

### 6.4 Freshness Tracking for Skills and Roles

Skills and roles are operational context consumed on every task. Stale skills are poisoned context — they silently misdirect every agent that receives them (P2, P3).

Role files (`.kbz/roles/*.yaml`) and skill files (`.kbz/skills/*/SKILL.md`) must carry a `last_verified` metadata field recording when the content was last confirmed as current. The `health` tool should flag any role or skill that has not been verified within a configurable window (default: 30 days). This extends the same freshness tracking that document records already have (`doc(action: "refresh")`) to all operational context files.

When a skill or role is flagged as stale, it remains usable — the system does not block context assembly — but the staleness warning appears in the health report and in the assembled context's metadata, so orchestrators and humans are aware the context may be outdated.

*Source: P3 (Living Documentation) improvement #1 — "context profiles and SKILLs do not carry `last-verified` metadata. These files are operational context that agents read on every task. They should have the same freshness tracking as registered documents."*

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
│   ├── orchestrate-development/
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

## 8. Skill Content Authoring Guidelines

These guidelines apply to anyone writing skill content — human or agent. They complement the structural constraints (section ordering, file layout) with content-level constraints that determine whether the assembled context actually improves agent output.

### 8.1 The Novelty Test

Before writing any line of skill content, apply the novelty test: "Does the model already know this?"

- **Include:** Kanbanzai-specific lifecycle states, transition rules, field ordering requirements, tool call sequences, project conventions, vocabulary payloads.
- **Exclude:** What a state machine is, how lifecycle transitions work in general, what Go interfaces are, how to write tests, what YAML is.

If a paragraph explains a general concept, delete it. If it explains a general concept *and* a Kanbanzai-specific rule, keep only the Kanbanzai-specific rule. The model has the general knowledge; it lacks the project-specific knowledge.

### 8.2 Tone: Explain Why, Not Rigid Imperatives

Anti-patterns use BECAUSE clauses (DP-4), and this extends to all instructional content. The tone should be explanatory rather than authoritarian. When the model understands *why* a convention exists, it generalises correctly to novel situations. When it only knows *what* the rule is, it follows it rigidly in matching cases and ignores it in novel ones.

- **Avoid:** "ALWAYS do X." / "NEVER do Y."
- **Prefer:** "Do X because Y." / "Avoid X because Y — instead, do Z."

This is not a prohibition on clear conventions — it is a prohibition on *unexplained* imperatives. A BECAUSE clause transforms a brittle rule into a generalisable principle.

*Source: Anthropic skill-creator meta-skill ("Try to explain to the model why things are important in lieu of heavy-handed musty MUSTs").*

### 8.3 Uncertainty Protocol

Every skill that produces work output (authoring skills, implementation skills) must include an explicit uncertainty instruction positioned early in the procedure, where attention is highest:

> If the specification is ambiguous, incomplete, or contradictory for any aspect of this task, STOP and report the ambiguity. Do not infer intent. Do not make undocumented design decisions. The cost of asking is low; the cost of guessing wrong is high.

This directly addresses the observed problem of agents being too eager to proceed and making decisions that should be human-owned. The instruction grants explicit permission to admit uncertainty — without this, the model's default behaviour is to produce *something* rather than nothing.

*Source: Anthropic hallucination reduction guide ("Explicitly give Claude permission to admit uncertainty").*

### 8.4 Spec Citation for Implementation Decisions

Authoring and implementation skills should require agents to cite the specific spec section that justifies each non-trivial decision. This creates an audit trail and forces decisions to be grounded in the specification rather than general knowledge.

For the `implement-task` skill: "When making an implementation choice between alternatives, cite the spec requirement that drives the decision. If no spec requirement covers the decision, note it as an assumption and flag it for human review."

For review skills, this is already captured in the "Missing Spec Anchor" anti-pattern. Implementation skills need the complementary rule.

*Source: Anthropic hallucination reduction guide ("Use direct quotes for factual grounding"; "Explicitly instruct Claude to only use information from provided documents").*

### 8.5 Terminology Consistency Within Skills

Each skill's vocabulary payload defines the canonical terms for its domain. Within the skill's own content — procedure, examples, anti-patterns — use those terms exclusively. Do not alternate between synonyms.

- If the vocabulary says "finding," never write "issue" or "problem" in the procedure.
- If the vocabulary says "acceptance criteria," never write "requirements" or "success conditions."

The vocabulary payload routes the model to the right knowledge clusters. Inconsistent terminology within the skill undermines that routing.

*Source: Anthropic Skill Authoring Best Practices ("Use consistent terminology — choose one term and use it throughout").*

---

## 9. Skill Development Process

Skills are not documentation — they are executable context that directly shapes agent output quality. They must be developed with the same rigour as code: tested against real scenarios, evaluated for effectiveness, and iterated based on observed behaviour.

### 9.1 The Development Loop

Every skill — new or revised — follows this loop:

1. **Identify the gap.** Run agents on representative tasks *without* the skill. Document what they get wrong: missed conventions, incorrect output format, skipped steps, wrong vocabulary.

2. **Write minimal content.** Address only the observed gaps. Do not document imagined problems or explain things the model already handles correctly. Start lean and add only when evaluation shows a need.

3. **Define test scenarios.** Write 3–5 realistic test scenarios per skill. Each scenario should be a concrete task description — the kind of thing an orchestrator or human would actually say. Include both should-trigger and should-not-trigger scenarios for the description.

4. **Run and compare.** Execute agents on the test scenarios with and without the skill. Compare outputs qualitatively. Does the skill improve the specific gaps identified in step 1? Does it introduce new problems (e.g., wasted tokens on content the agent ignores)?

5. **Iterate.** Based on the comparison:
   - If the skill doesn't improve output → cut content, not add more.
   - If agents ignore a section → it may be in the attention dead zone, or it may explain something the model already knows. Try repositioning or deleting.
   - If agents consistently perform the same multi-step operation → bundle it as a script in `scripts/`, not as prose instructions.
   - If a stubborn issue persists → try different metaphors or patterns rather than adding more rigid constraints.
   - **Use agents to diagnose failures.** Anthropic found that "Claude 4 models can diagnose prompt failures and suggest improvements when given a prompt and a failure mode." When a skill consistently fails to produce the desired output, give an agent the skill content plus a concrete failure example and ask it to identify why the skill didn't work. This is often faster than manual analysis and surfaces attention-curve or vocabulary issues that humans miss.

6. **Pass the quality gate** (§9.2) before committing.

This loop need not be heavy. For a simple skill update, steps 1–2 might take 15 minutes. For a new skill, the full loop might take a few hours. The point is that *zero* evaluation is never acceptable.

*Source: Anthropic Skill Authoring Best Practices ("Create evaluations BEFORE writing extensive documentation"); Anthropic skill-creator meta-skill (create → test → review → improve loop).*

### 9.2 Skill Quality Gate

Every skill must pass this checklist before shipping. This is a gate, not a guideline.

**Structure:**
- [ ] SKILL.md body is under 500 lines
- [ ] Section ordering follows the attention curve (Vocabulary → Anti-patterns → Checklist → Procedure → Output Format → Examples → Evaluation Criteria → Questions)
- [ ] All reference files link directly from SKILL.md (one level deep, never reference-to-reference)
- [ ] `constraint_level` is declared in frontmatter and procedure style matches (exact sequences for low, templates for medium, guidance for high)

**Content:**
- [ ] Every paragraph passes the novelty test (§8.1) — no general-knowledge explanations
- [ ] Vocabulary payload has 15–30 terms that pass the 15-year practitioner test
- [ ] Anti-patterns have 5–10 entries, each with detect/because/resolve
- [ ] At least 2 BAD/GOOD example pairs with WHY explanations
- [ ] Terminology is consistent with the vocabulary payload throughout (§8.5)
- [ ] Tone is explanatory, not authoritarian (§8.2)
- [ ] No time-sensitive information (use "current method" / "previous method," not dates)

**Description:**
- [ ] `expert` and `natural` descriptions are both present
- [ ] Descriptions include both *what* the skill does and *when* to use it
- [ ] Workflow-critical skills include assertive "use even when..." clauses (DP-11)
- [ ] Description uses third person consistently

**Testing:**
- [ ] At least 3 test scenarios defined (realistic task descriptions)
- [ ] Tested with representative tasks — outputs compared with and without skill
- [ ] Observed improvement on the gaps the skill was written to address

### 9.3 Description Optimization

For the most critical skills (those bound to stage gates or high-frequency stages), run a lightweight description optimization:

1. Write 10 test queries — a mix of should-trigger and should-not-trigger.
2. Test whether the skill triggers correctly on each query.
3. If under-triggering: make the description more assertive, add trigger terms the agent uses when thinking about the task.
4. If over-triggering: narrow the description's scope, add "do not use when..." exclusions.
5. Iterate until trigger accuracy is acceptable.

This is not needed for every skill — only for the 3–4 skills where incorrect triggering has the highest cost (e.g., `implement-task`, `review-code`, workflow-stage skills).

*Source: Anthropic skill-creator meta-skill (description optimization workflow).*

---

## 10. Expected Impact

### 10.1 Metrics to Track

| Metric | Current Baseline | Target | Source Principle |
|--------|-----------------|--------|-----------------|
| First-attempt convention compliance | Unknown (not tracked) | >90% | P5 (Institutional Memory) |
| Review finding specificity | Moderate (prose-heavy) | 100% structured, evidence-backed | P6 (Specialized Review) |
| Stale-doc-caused errors | Occasional | Zero | P3 (Living Documentation) |
| Context assembly token utilisation | Unknown | 15–40% of window | P2 (Context Hygiene) |
| Review rubber-stamp rate | Unknown | <15% clean verdicts without evidence | P8 (Strategic Human Gate) |
| Sub-agent dispatch per feature | Variable | 1 (70% of tasks), 2–4 (30%) | P9 (Token Economy) |
| MAST failure mode incidents | Unknown | Tracked and trending down | P7 (Observability) |

**Scope note on observability:** The metrics above require collection mechanisms that are outside the scope of this design. Specifically, the original research (R8, R9, R14, R19) recommends structured handoff logging for sub-agent dispatches, per-reviewer metrics tracking (approval rate, finding count, review duration), gate rejection rate monitoring, and automated MAST failure mode detection. These are system-level instrumentation concerns — they require changes to the MCP server's logging layer, the `spawn_agent` dispatch path, and the `checkpoint` tool — not to the skills and roles architecture. They should be addressed in a companion observability design. This document provides the *inputs* to observability (structured review output, stage bindings, role declarations) but not the *collection infrastructure*.

### 10.2 Research-Backed Predictions

Based on the cited research:

- **Vocabulary routing** (Ranjan et al., 2024): Expect measurable improvement in domain-specific output quality — the model activates expert knowledge clusters instead of generic training data.
- **Attention-optimized structure** (Liu et al., 2024): Expect fewer missed constraints and anti-patterns, particularly for instructions that were previously in the "dead zone" middle of context.
- **Specialist panel** (PRISM, Ranjan et al.): Expect security, quality, and testing issues caught at rates approaching the research's reported 95% for specialist vs 40% for generalist (security domain).
- **Named anti-patterns** (CHI 2023): Expect repeat errors on codified anti-patterns to drop to near-zero within two weeks of deployment.
- **Token budget management** (DeepMind, 2025): Expect token cost reduction of 40–60% from adaptive tool loading and cascade escalation.
- **BAD/GOOD examples** (LangChain, 2024): Expect output format compliance to improve significantly — the model pattern-matches against demonstrated examples more reliably than it follows written rules.

---

## 11. Design Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| DD-1 | Roles and skills are separate files that compose | Roles carry domain expertise (who); skills carry procedures (what). The same role uses different skills at different stages. The same skill is used by different specialist reviewers. Composition gives both reuse and specificity. |
| DD-2 | Vocabulary is a first-class YAML field in roles, not embedded in prose | Vocabulary is the primary routing mechanism. It must be explicit, auditable, and mechanically mergeable during context assembly. Prose-embedded vocabulary cannot be reliably extracted or composed. |
| DD-3 | Anti-patterns use a BECAUSE clause | The BECAUSE clause makes rules generalisable to adjacent cases (CHI 2023). Without it, rules cover only the literal case stated. With it, the model extends the principle to novel situations. |
| DD-4 | Skills follow a fixed section ordering | The attention curve (Liu et al., 2024; Wu et al., 2025) is architectural, not patchable. The ordering is not a suggestion — it is a design constraint that maps content to attention weight. |
| DD-5 | Evaluation criteria are separated from procedure | The agent doing the work must not self-evaluate (Anthropic, Mar 2026). Evaluation criteria enable a separate pass — human or automated — to assess output quality using gradable questions. |
| DD-6 | Stage bindings are declared, not inferred | Implicit bindings (agents reading AGENTS.md and inferring the right context) are a fuzzy step that should be hardened (P1). Declared bindings are deterministic, auditable, and testable. |
| DD-7 | Agent cap of 4 applies to specialist panels, not independent tasks | DeepMind (2025) shows team effectiveness saturates at 3–4 agents sharing coordination overhead (e.g., specialist reviewers evaluating the same code from different angles). Implementation tasks are independent by construction — different files, different concerns — and don't share that coordination tax. The cap is per-topology: parallel-specialists cap at 4; parallel-independents are limited by task count. *(Refined based on Google Research alignment principle and orchestration recommendations.)* |
| DD-8 | Tool filtering per role | Every loaded tool definition consumes attention budget. A security reviewer does not need `decompose`. Filtering reduces context size and focuses attention on relevant tools. Implements the jig pattern from P10. *(Implementation note: The design target is hard filtering — dynamically scoping the MCP tool list per session based on the role's `tools` field. For the initial 3.0 release, soft filtering (guidance text in assembled context) is accepted as a pragmatic stepping stone. Tool selection compliance will be tracked via the evaluation suite; if soft filtering proves insufficient, hard filtering implementation is prioritised. The `tools` field, §6.1 step 7, and §3.4 retain hard-filtering language as the design intent. See workflow doc §9 for the 3.0 implementation timeline. Per cross-document alignment report §1.1.)* |
| DD-9 | Skills and roles move inside `.kbz/` | The `.kbz/` directory is the instance root for all Kanbanzai state. Skills and roles are operational configuration, not project documentation. They belong with the system state, not at the project root. This also makes them manageable by MCP tools. |
| DD-10 | Document-type-specific authoring skills replace generic `document-creation` | A specification needs requirements vocabulary; a design needs architecture vocabulary. A generic skill activates neither. Type-specific skills carry type-specific vocabulary, anti-patterns, and examples. |
| DD-11 | Constraint level is declared per skill | Different tasks need different degrees of freedom (DP-9). Low-freedom skills provide exact tool call sequences; high-freedom skills provide principles. Declaring the constraint level in frontmatter makes this explicit and auditable, and forces the skill author to match procedure style to risk. |
| DD-12 | Skills include optional copy-paste checklists | Checklists that agents copy into their response make compliance visible and prevent step-skipping (Anthropic Best Practices). They are required for low/medium-freedom skills and optional for high-freedom skills. |
| DD-13 | Skill development follows an evaluation-driven loop | Skills are written to address observed gaps, not imagined problems. The create → test → compare → iterate loop (§9.1) and quality gate (§9.2) ensure skills demonstrably improve agent output before shipping. Writing evaluations before writing extensive skill content is a hard process constraint. |
| DD-14 | Scripts replace prose for deterministic operations | Operations that are exact and repeatable (prerequisite checks, structure validation, stage gate verification) belong in `scripts/`, not in prose instructions. Script output enters the context window; script source does not. This reduces token cost and eliminates interpretation errors. |
| DD-15 | Orchestration pattern is a first-class binding field | Google Research found multi-agent coordination degrades sequential reasoning by 39–70% but improves parallelisable tasks by 81%. The orchestration pattern (`single-agent` vs `orchestrator-workers`) must be declared per stage and included in assembled context, not left to agent discretion. *(Source: orchestration recommendations §2.3.)* |
| DD-16 | Stage gates are enforceable prerequisites, not advisory | Masters et al. distinguish hard constraints (ℋ — violation terminates) from soft constraints (𝒮 — penalties). Stage transitions require programmatic prerequisite checks (document approval status, task completion). This design declares the prerequisite data (in the binding registry) and the requirement that transitions are rejected when prerequisites are unmet. The enforcement mechanism — how the `entity(action: "transition")` tool checks and rejects transitions — is designed in the workflow doc (§3). *(Source: MetaGPT SOPs, Masters et al. hard constraints, orchestration recommendations §2.1.)* |
| DD-17 | Effort budgets are included in assembled context | Anthropic found that "agents struggle to judge appropriate effort for different tasks." Each stage binding declares an effort budget (expected tool call range and activity description). The assembly pipeline includes this at position 3 (high attention) so the agent calibrates before reading the procedure. *(Source: Anthropic multi-agent system, orchestration recommendations §5.1.)* |
| DD-18 | Review-rework loops have an iteration cap | Microsoft's maker-checker pattern requires an iteration cap to prevent infinite refinement. The reviewing binding declares `max_review_cycles: 3`. After 3 fail-rework cycles, the system escalates to human decision. *(Source: Microsoft AI Agent Orchestration Patterns, orchestration recommendations §2.2.)* |
| DD-19 | Document templates are gate-checkable | MetaGPT's core mechanism is structured intermediate artifacts verified before the next stage begins. Each authoring skill defines required sections and cross-references. Validation scripts in `scripts/` check these at stage gates. *(Source: MetaGPT, orchestration recommendations §5.2.)* |
| DD-20 | Design optimises for constraint adherence over workflow runtime | Masters et al. (2025) identify a fundamental three-way trade-off: "Goal achievement, constraint adherence, and workflow runtime cannot all be maximized simultaneously." Kanbanzai's current problems (inconsistency, step-skipping, rushed specifications) indicate constraint adherence is the weakest axis. This design therefore prioritises constraint adherence — enforceable gates, structured templates, prerequisite checks — accepting that these mechanisms add latency to the workflow. This is a conscious trade-off, not an oversight. If constraint adherence improves to a satisfactory level, future iterations may relax selected gates to recover runtime. *(Source: Masters et al. (2025), orchestration recommendations §1.)* |

---

## 12. Open Questions

1. **Should vocabulary payloads be curated per-project or shipped as defaults?** The current design assumes vocabulary is project-specific (e.g., `implementer-go` for this project). Should Kanbanzai ship default vocabulary payloads for common languages and domains, with project-level overrides?

2. **~~How should the system handle roles that don't exist yet?~~** **Resolved.** If a stage binding references a role that doesn't exist (e.g., `reviewer-security` not yet created), the system falls back to the parent role (e.g., `reviewer`) and logs a warning. It does not refuse to proceed. Rationale: hard gates block *incorrect* transitions; missing context is a soft concern to be flagged, not blocked. The `health` tool reports the missing role so it can be created. *(Resolved via orchestration recommendations review — Masters et al. hard/soft constraint alignment.)*

3. **Should skills carry their own test fixtures?** The `references/` directory could include test inputs and expected outputs for the skill, enabling automated quality assurance of skill output. Is this worth the added complexity?

4. **~~How does this interact with the `decompose` tool's task creation?~~** **Resolved.** When `decompose` creates tasks, it should tag each task with the expected stage and role so that `handoff` can resolve the binding automatically. Additionally, `decompose` should validate decomposition quality: tasks have clear descriptions, dependencies are declared, tasks are sized for single-agent completion, and there are no obvious gaps (e.g., missing test tasks). Decomposition quality is the single strongest predictor of workflow success (Masters et al.). *(Resolved via orchestration recommendations §5.5.)*

5. **Should vocabulary payloads have expiry dates?** Domain vocabulary evolves. "OWASP Top 10 (2021)" will eventually be superseded. Should vocabulary terms carry freshness metadata, like knowledge entries do?

6. **~~What is the right granularity for reviewer specialisation?~~** **Resolved.** Four specialists (conformance, quality, security, testing) is the right granularity. Documentation currency remains a cross-cutting concern within conformance review — it doesn't warrant its own vocabulary payload for most changes. The adaptive composition approach (§5.4) allows the orchestrator to dispatch fewer reviewers when the change doesn't warrant all four. *(Resolved via orchestration research — Google's finding that the optimal architecture depends on task properties, not a fixed team structure.)*

7. **How should skills be versioned and tracked?** If skills follow the evaluation-driven development loop (§9), should test scenarios and evaluation results be stored alongside the skill (in `references/` or a `tests/` directory)? Should skill changes be tracked with the same lifecycle discipline as entity changes?

8. **Should we adopt a skill-creator meta-skill?** The Anthropic skill-creator (§9 of the research report) demonstrates a comprehensive meta-skill for creating, testing, and iterating on skills. At our current scale (~10 skills), the full tooling is overkill. But if the skill catalog grows significantly, a Kanbanzai-adapted skill-creator could enforce the quality gate and development loop automatically.

---

## 13. Relationship to Other Documents

| Document | Relationship |
|----------|-------------|
| `work/research/ai-agent-best-practices-research.md` | Research basis — this design applies the 20 recommendations from that report |
| `work/research/agent-orchestration-research.md` | Orchestration research — findings on enforceable gates, ACI tool design, effort budgets, orchestration patterns, and decomposition quality integrated into this design (DD-15 through DD-19, DP-9 extension, binding registry enrichment) |
| `work/design/orchestration-recommendations.md` | Companion document — concrete orchestration-layer recommendations derived from the orchestration research. This design implements the SKILL and role integration recommendations from that document. |
| `work/design/machine-context-design.md` | The context assembly model this design extends |
| `work/design/agent-interaction-protocol.md` | Agent behaviour conventions this design must respect |
| `work/design/quality-gates-and-review-policy.md` | Review policy this design operationalises |
| `work/design/document-centric-interface.md` | Document-led workflow model this design integrates with |
| `work/spec/phase-2b-specification.md` | The context profile system this design supersedes |
| `work/research/agent-skills-research.md` | Anthropic documentation research — pragmatic additions (§8, §9) drawn from this report |
| *(companion design needed)* | Observability infrastructure — structured handoff logging, review metrics, gate rejection monitoring, MAST detection (R8, R9, R14, R19 from research). Out of scope for this design; see §10.1 scope note. |