---
name: write-research
description:
  expert: "Research report authoring producing a structured investigation document
    with defined methodology, graded evidence, synthesised findings, and actionable
    recommendations during the researching stage"
  natural: "Write a research report that investigates a question, evaluates evidence
    from multiple sources, and presents findings with recommendations"
triggers:
  - write a research report
  - research a topic for a feature
  - investigate options and produce a report
  - author a research document
  - conduct research and document findings
roles: [researcher]
stage: researching
constraint_level: medium
---

## Vocabulary

- **research question** — the specific question the investigation aims to answer, scoped narrowly enough to produce actionable findings
- **methodology** — the approach used to gather and evaluate evidence: literature review, code archaeology, benchmarking, prototyping, or expert consultation
- **primary source** — first-hand evidence: source code, official documentation, published benchmarks, API specifications, or direct experimentation
- **secondary source** — evidence derived from primary sources: blog posts, tutorials, conference talks, third-party comparisons
- **evidence grading** — assigning a reliability level to each piece of evidence based on source quality, recency, and reproducibility
- **finding** — a factual claim supported by cited evidence, distinguished from opinion or recommendation
- **synthesis** — combining multiple findings into a coherent narrative that answers the research question
- **recommendation** — an actionable suggestion derived from findings, with stated confidence level
- **limitation** — an explicit boundary on the research's applicability: what was not investigated, what assumptions were made
- **scope of investigation** — the defined boundary of the research — what questions are in scope and what is explicitly excluded
- **reproducibility** — whether another researcher could follow the methodology and reach the same findings
- **evidence weight** — the degree to which a finding supports or contradicts a conclusion, based on source quality and relevance
- **confirmation bias** — the tendency to favour evidence supporting a pre-existing belief while discounting contradictory evidence
- **prior art** — existing solutions, designs, or research that address the same or a closely related problem
- **trade-off matrix** — a structured comparison of alternatives across multiple evaluation dimensions
- **confidence level** — the degree of certainty in a recommendation: high (strong evidence, low risk), medium (adequate evidence, some unknowns), low (limited evidence, significant unknowns)
- **knowledge gap** — an area where available evidence is insufficient to make a recommendation, requiring further investigation
- **source recency** — how current a source is relative to the technology or domain being researched; stale sources may describe superseded behaviour
- **falsifiability** — whether a finding could be disproven by new evidence; unfalsifiable claims are not findings
- **evaluation criterion** — a specific dimension used to compare alternatives in the trade-off matrix

## Anti-Patterns

### Conclusion Without Evidence
- **Detect:** A recommendation or conclusion is stated without citing specific evidence from identified sources
- **BECAUSE:** Unsupported conclusions are indistinguishable from opinions — downstream decisions based on them inherit unknown risk, and reviewers cannot verify the reasoning
- **Resolve:** Every recommendation must trace back to at least one finding, and every finding must cite at least one source

### Cherry-Picked Sources
- **Detect:** All cited sources support the same conclusion; contradictory evidence is absent or dismissed without analysis
- **BECAUSE:** Confirmation bias produces research that validates a predetermined answer rather than investigating the question — the design team makes decisions on an incomplete picture
- **Resolve:** Actively search for contradictory evidence. If none exists, state that explicitly. If it exists, analyse why it conflicts and what that means for the recommendation

### Missing Methodology
- **Detect:** The report presents findings without describing how they were obtained
- **BECAUSE:** Without methodology, the research is not reproducible — a reader cannot assess whether the approach was sound or whether different methods would yield different results
- **Resolve:** State the methodology before presenting findings: what sources were consulted, what criteria were used, what was excluded and why

### Scope Creep
- **Detect:** The report investigates questions not stated in the scope of investigation, or findings address topics outside the research question
- **BECAUSE:** Unscoped research produces diffuse findings that do not clearly answer the question the design team needs answered, wasting effort and delaying decisions
- **Resolve:** Define the scope of investigation upfront. If additional questions emerge, note them as future research topics rather than investigating them in this report

### Ungraded Evidence
- **Detect:** All sources are treated as equally authoritative — official documentation and a three-year-old blog post carry the same weight
- **BECAUSE:** Treating low-quality evidence the same as high-quality evidence undermines the reliability of findings — a recommendation based on a Stack Overflow answer has different confidence than one based on published benchmarks
- **Resolve:** Grade each source by type (primary vs. secondary), recency, and authority. Weight findings accordingly

### Missing Limitations
- **Detect:** The report presents findings and recommendations without acknowledging what was not investigated or what assumptions were made
- **BECAUSE:** Research without stated limitations appears more authoritative than it is — decision-makers cannot calibrate how much to trust the recommendations
- **Resolve:** Include a Limitations section that states what the research did not cover, what assumptions underpin the findings, and what conditions could change the conclusions

### Recommendation Without Confidence
- **Detect:** Recommendations are presented as definitive ("use X") without indicating the confidence level or conditions under which the recommendation applies
- **BECAUSE:** A high-confidence recommendation backed by benchmarks and a low-confidence recommendation based on a single blog post require different treatment by decision-makers — presenting both as equally certain leads to poorly calibrated decisions
- **Resolve:** Assign a confidence level (high, medium, low) to each recommendation and state the evidence basis

### Report From Memory

- **Detect:** Agent writes a retrospective or research report without first calling `retro(action: "synthesise")` and `knowledge(action: "list")`.
- **BECAUSE:** Retrospective signals and knowledge entries accumulate across sessions. In-session memory only captures the current session. Reports written from memory systematically miss recurring patterns and prior decisions, producing incomplete analysis that cannot support reliable recommendations.
- **Resolve:** Always call `retro(action: "synthesise")` and `knowledge(action: "list")` before writing any report. Treat the synthesised output as the primary input, not a supplement.

## Pre-Writing Checklist

Before beginning to write the research report:

- [ ] Called `retro(action: "synthesise")` to surface retrospective signals from all sessions — do not rely on in-session memory alone
- [ ] Called `knowledge(action: "list")` to retrieve project-level knowledge entries relevant to the report topic

## Procedure

### Step 1: Define the Investigation

1. Read the research question or request. Understand what decision the research is meant to inform.
2. Define the scope of investigation: what is in scope, what is explicitly excluded.
3. IF the research question is vague or too broad → STOP. Ask for clarification. A well-scoped question produces actionable findings; a vague question produces a survey.
4. Select the methodology: literature review, code analysis, prototyping, benchmarking, or a combination.

### Step 2: Gather Evidence

1. Identify primary sources first: official documentation, source code, published specifications, direct experimentation.
2. Supplement with secondary sources where primary sources are insufficient.
3. Grade each source by type, recency, and authority.
4. Actively search for contradictory evidence — do not stop at the first source that answers the question.
5. IF a critical question cannot be answered from available sources → note it as a knowledge gap rather than speculating.

### Step 3: Analyse and Synthesise

1. For each sub-question within the scope, collect the relevant findings and their supporting evidence.
2. Where alternatives are being compared, construct a trade-off matrix with explicit evaluation criteria.
3. Identify patterns across findings — do multiple independent sources converge on the same conclusion?
4. Note where evidence conflicts and analyse why.
5. IF the evidence is insufficient to answer the research question with medium or high confidence → state this explicitly rather than overstating findings.

### Step 4: Draft the Report

1. Write all sections in order: Research Question, Scope and Methodology, Findings, Trade-Off Analysis (if applicable), Recommendations, Limitations.
2. Every finding must cite its source(s).
3. Every recommendation must reference the findings that support it and include a confidence level.
4. The Limitations section must be present and substantive.

### Step 5: Self-Validate

1. Verify every recommendation traces back to at least one finding.
2. Verify every finding cites at least one source.
3. Verify the Limitations section exists and is not empty.
4. Verify no finding addresses a topic outside the stated scope.
5. IF validation fails → fix the gap → re-validate.

## Output Format

The research report uses the following structure. Section headings may be adapted to the specific research topic, but all conceptual sections must be present:

```
## Research Question

State the question this report investigates. What decision does this
research inform? Who requested it and why?

## Scope and Methodology

**In scope:** what this report covers.
**Out of scope:** what this report does not cover.
**Methodology:** how evidence was gathered and evaluated (literature
review, code analysis, benchmarking, prototyping, etc.).

## Findings

Organise by sub-question or theme. Each finding:
- States the factual claim
- Cites the source(s) with evidence grade
- Notes confidence level

### Finding 1: [Topic]

[Factual claim with evidence citation]

Source: [reference] (primary/secondary, [recency])

### Finding 2: [Topic]

...

## Trade-Off Analysis

(Include when comparing alternatives)

| Criterion | Option A | Option B | Option C |
|-----------|----------|----------|----------|
| [dim 1]   | ...      | ...      | ...      |
| [dim 2]   | ...      | ...      | ...      |

## Recommendations

Each recommendation:
- **Recommendation:** what to do
- **Confidence:** high / medium / low
- **Based on:** which findings support this
- **Conditions:** when this recommendation applies

## Limitations

- What was not investigated
- What assumptions were made
- What conditions could change these conclusions
```

## Examples

### BAD: Unsupported Recommendation

> ## Research Question
> Which testing framework should we use?
>
> ## Findings
> After looking at several options, testify seems like the most popular
> choice in the Go ecosystem. It has good assertion helpers and suite support.
>
> ## Recommendations
> Use testify for all tests.

**WHY BAD:** No methodology stated. Single finding cites no sources and uses "seems like" instead of evidence. No alternatives compared. No trade-off analysis. Recommendation has no confidence level and no conditions. No limitations section. A decision-maker cannot evaluate whether this recommendation is well-founded.

### BAD: Scope Creep with Ungraded Evidence

> ## Research Question
> How should we implement caching for the entity API?
>
> ## Scope and Methodology
> Reviewed caching strategies for Go applications.
>
> ## Findings
> Redis is fast. A blog post from 2019 says Redis handles 100k ops/sec.
> Memcached is also fast. Someone on Reddit said they prefer Memcached.
> We should also consider CDN caching for static assets and maybe
> implement a service mesh while we're at it.

**WHY BAD:** Scope creep (CDN caching and service mesh are unrelated to entity API caching). Evidence is ungraded — a 2019 blog post and a Reddit comment are treated as authoritative. No distinction between primary and secondary sources. Findings drift from the stated research question.

### GOOD: Structured Research with Graded Evidence

> ## Research Question
> What approach should the skills system use for validation scripts:
> POSIX shell or Go executables? This informs the implementation plan
> for the skills system redesign.
>
> ## Scope and Methodology
> **In scope:** Runtime characteristics, portability, maintainability, and
> implementation cost of POSIX shell vs. Go for validation scripts.
> **Out of scope:** Alternative languages (Python, Node). Script content
> and logic.
> **Methodology:** Code analysis of existing scripts in the repository,
> official POSIX and Go documentation review, measurement of startup
> times on the development environment.
>
> ## Findings
>
> ### Finding 1: Startup Time
> POSIX shell scripts start in ~5ms. Go binaries compiled with
> `go build` start in ~15-30ms. For validation scripts that run at
> stage gates, both are well within the 5-second budget.
>
> Source: Direct measurement on macOS 14 (primary, current).
>
> ### Finding 2: Portability
> POSIX shell is available on all target environments (macOS, Linux)
> without additional tooling. Go requires the Go toolchain for
> compilation, but the project already depends on it.
>
> Source: POSIX.1-2017 specification §2 (primary, current);
> project dependency analysis of go.mod (primary, current).
>
> ### Finding 3: Maintainability
> The existing codebase is 100% Go. Shell scripts introduce a second
> language that contributors must know. However, validation scripts
> are simple (grep for headings) and unlikely to grow complex.
>
> Source: Repository language analysis via `tokei` (primary, current).
>
> ## Trade-Off Analysis
>
> | Criterion       | POSIX Shell        | Go Executable       |
> |-----------------|--------------------|---------------------|
> | Startup time    | ~5ms               | ~15-30ms            |
> | Portability     | Universal          | Requires Go toolchain |
> | Maintainability | Second language    | Consistent with codebase |
> | Complexity fit  | Good for simple checks | Over-engineered for grep |
> | Test coverage   | Harder to unit test | Standard Go testing |
>
> ## Recommendations
>
> **Recommendation:** Use POSIX shell for validation scripts.
> **Confidence:** High.
> **Based on:** Findings 1-3. Validation scripts perform simple
> structural checks (grep for headings). Shell is the natural fit for
> this complexity level. The maintainability cost is low because the
> scripts are small and unlikely to grow.
> **Conditions:** This recommendation assumes validation scripts remain
> simple structural checks. If validation logic grows to require parsing
> or complex conditionals, revisit in favour of Go.
>
> ## Limitations
>
> - Did not evaluate Python or Node as alternatives (out of scope)
> - Startup measurements taken on a single machine; CI environments may differ
> - Maintainability assessment is subjective and based on current team composition

**WHY GOOD:** Clear research question tied to a specific decision. Scope and methodology stated upfront. Three findings with cited primary sources and recency noted. Trade-off matrix compares alternatives across specific dimensions. Recommendation includes confidence level, evidence basis, and conditions. Limitations acknowledge boundaries honestly. A decision-maker can evaluate each claim independently.

## Evaluation Criteria

1. Does the report state a clear research question tied to a specific decision? Weight: required.
2. Does the Scope and Methodology section describe what was investigated and how? Weight: required.
3. Does every finding cite at least one source with an evidence grade? Weight: required.
4. Does every recommendation include a confidence level and reference to supporting findings? Weight: required.
5. Is a Limitations section present with substantive content? Weight: high.
6. Are contradictory sources acknowledged and analysed rather than ignored? Weight: high.
7. Is a trade-off matrix included when comparing alternatives? Weight: medium.
8. Can a design author use this report to make a well-informed decision without additional research? Weight: high.

## Questions This Skill Answers

- How do I write a research report for a Kanbanzai feature?
- What structure should a research document follow?
- How do I grade evidence from different source types?
- How do I present findings with appropriate confidence levels?
- When should I stop researching and present what I have?
- How do I compare alternatives in a research report?
- What belongs in the Limitations section of a research report?
- How do I handle conflicting evidence from different sources?
- What methodology should I describe in a research report?