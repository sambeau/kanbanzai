---
name: check-docs
description:
  expert: "Documentation QA producing classified findings for factual accuracy,
    code example verification, hallucination detection, and substance assessment
    against implementation as source of truth"
  natural: "Fact-check a document — verify every claim against the code, test
    every example, flag hallucinations and empty claims"
triggers:
  - fact-check documentation
  - verify document accuracy
  - check docs for hallucinations
  - QA documentation
  - validate technical claims
roles: [doc-checker]
stage: documenting
constraint_level: low
---

## Vocabulary

- **hallucination** — a factual claim in the document not supported by the
  implementation or any verifiable source. The most serious finding this stage
  can produce.
- **source of truth** — the implementation (code, configuration, runtime
  behaviour) as the authority for facts. Design documents provide concepts and
  intentions but must be verified against code.
- **code example verification** — running or tracing every code example in the
  document to confirm it produces the output shown and uses current APIs.
- **substance check** — evaluating whether a claim adds real information or is
  empty filler that could apply to any product.
- **significance inflation** — telling the reader how important something is
  ("plays a crucial role", "marks a significant shift") instead of showing
  evidence. A content problem, not a style problem.
- **vague claim** — a statement so general it could apply to any project: "a
  comprehensive solution", "significantly improves performance", "a wide range
  of use cases".
- **promotional language** — advertising copy masquerading as documentation:
  "boasts", "showcases", "diverse array", "cutting-edge".
- **superficial analysis** — a participle phrase appending shallow commentary to
  a fact: "reflecting the team's commitment to performance" — the fact stands
  on its own.
- **unverifiable claim** — a statement that cannot be checked against any
  available source. Flag but do not delete — the author may have knowledge you
  lack.
- **boilerplate** — formulaic content that adds no information: "Despite
  challenges, the future looks promising."
- **stale reference** — a link, version number, or API reference that points to
  something that no longer exists or has changed.
- **claim classification** — the category assigned to each finding:
  hallucination (provably wrong), unverified (no source found), vague (too
  general), inflated (significance without evidence), promotional (advertising
  language).

## Anti-Patterns

### Design-Doc Trust

- **Detect:** Facts are accepted because they match a design document or
  specification, without checking the implementation.
- **BECAUSE:** Design documents describe intentions; implementations describe
  reality — they diverge as development proceeds. A fact verified only against
  design may be a hallucination against code.
- **Resolve:** Verify every factual claim against the implementation. Use design
  documents for concepts and intentions only.

### Untested Examples

- **Detect:** Code examples are passed without being run or traced through the
  codebase.
- **BECAUSE:** Broken examples are worse than no examples — readers copy-paste
  them directly, and broken examples create debugging sessions that poison the
  developer experience and erode trust in all documentation.
- **Resolve:** Run every code example or trace it through the codebase. Verify
  command output matches what's shown. Check that APIs, flags, and config keys
  exist.

### Vague Claim Pass-Through

- **Detect:** Generic claims like "robust performance", "comprehensive
  solution", or "designed with scalability in mind" are accepted without
  demanding specifics.
- **BECAUSE:** Vague claims add no information and could describe any product.
  They waste the reader's time and signal that the author didn't know the
  specifics — the reader will not trust the rest of the document.
- **Resolve:** Flag every vague claim. Recommend a specific replacement if one
  is available, or flag for the author to provide specifics.

### Significance Inflation Blindness

- **Detect:** Phrases like "plays a crucial role", "is a testament to",
  "underscores the importance of" are not flagged.
- **BECAUSE:** These phrases promise importance without delivering evidence.
  Deleting them almost never loses information — if the importance is not
  obvious from the facts, the fix is evidence, not adjectives.
- **Resolve:** Flag every significance claim. Recommend either deletion or
  replacement with concrete evidence (a number, a comparison, a consequence).

### Scope Creep into Editing

- **Detect:** The fact-checker rewrites sentences, restructures sections, or
  fixes grammar instead of flagging factual issues.
- **BECAUSE:** Rewriting at the QA stage undoes work done by the structural
  editor and creates new text that hasn't been structurally reviewed. The
  checker's job is to verify and flag, not to rewrite.
- **Resolve:** Flag issues with location and recommendation. Do not rewrite
  prose. If a factual correction requires a rewrite, state the correct fact and
  let a later stage handle the prose.

## Checklist

Copy this checklist and track your progress:

Factual accuracy:
- [ ] Every factual claim has been identified and checked against the implementation
- [ ] Every code example has been run or traced
- [ ] Every command shown produces the output described
- [ ] Every API, flag, config key, and file path referenced exists in the current codebase
- [ ] Every version number and link target is current
- [ ] Claims from design documents have been verified against implementation

Substance:
- [ ] No vague claims that could apply to any product
- [ ] No significance inflation (importance claimed without evidence)
- [ ] No promotional language (boasts, showcases, diverse array, cutting-edge)
- [ ] No superficial analysis (participle phrases appending shallow commentary)
- [ ] No "despite challenges" boilerplate

## Procedure

### Step 1: Read and mark claims

Read the document end-to-end, marking every factual claim: behaviour
descriptions, API references, command outputs, configuration options, version
numbers, performance claims. Do not evaluate yet — just identify.

### Step 2: Identify sources of truth

For each factual claim, identify the source of truth. This is always the
implementation — code, configuration files, runtime behaviour. Never a design
document alone.

### Step 3: Verify against implementation

Verify each claim against its source of truth. Check that described behaviour
matches actual behaviour. Check that described APIs exist with the signatures
shown. Record the file and location checked.

### Step 4: Test code examples

For each code example, run it or trace through the codebase. Verify the output
matches what's shown. Check that all imports, functions, and flags exist. Record
the result.

### Step 5: Check links and cross-references

Check all links and cross-references. Do they resolve? Do they point to current
content? Flag any stale references with the target that was expected and what
was found.

### Step 6: Flag vague claims

Flag every statement that could apply to any product without modification. Any
claim that lacks specifics needs them. Consult the vague-claims table in
`refs/humanising-ai-prose.md` §5.2.

### Step 7: Flag significance inflation

Flag "plays a crucial role", "marks a significant shift", "is a testament to"
and similar phrases. Recommend deletion or replacement with evidence. Consult
`refs/humanising-ai-prose.md` §5.1.

### Step 8: Flag promotional language and superficial analysis

Flag "boasts a diverse array", "reflecting the team's commitment to" and
similar phrases for the style stage. Consult `refs/humanising-ai-prose.md`
§5.3 and §5.4.

### Step 9: Flag boilerplate

Flag "Despite challenges, the future looks promising" and similar formulaic
endings for deletion. Consult `refs/humanising-ai-prose.md` §5.5.

### Step 10: Compile and classify findings

Compile all findings. Classify each as:

- **hallucination** — provably wrong against the implementation
- **unverified** — no source found to confirm or deny
- **stale** — outdated reference (link, version, API that has changed)
- **vague** — too general, could describe any product
- **inflated** — significance claimed without evidence
- **promotional** — advertising language masquerading as documentation

## Output Format

The Check stage produces a report — it does not edit the document file directly. The orchestrator applies factual corrections (hallucinations, stale references) to the document before passing it to the next stage. Substance issues (vague, inflated, promotional) are flagged for the Style stage, not corrected here. The report must contain enough detail (locations, correct values, source checked) for the orchestrator to make the changes.

```
## QA Report

**Document:** {document path}
**Claims checked:** {count}
**Code examples tested:** {count}

### Findings

#### {Finding title}
- **Location:** {section, line, or quote}
- **Classification:** {hallucination | unverified | stale | vague | inflated | promotional}
- **Claim:** "{the text as written}"
- **Source checked:** {file path, function, or URL checked}
- **Issue:** {what's wrong}
- **Recommendation:** {specific correction or "flag for author"}

### Summary
- Hallucinations: {count}
- Unverified claims: {count}
- Stale references: {count}
- Substance issues (vague/inflated/promotional): {count}
- Verified claims: {count}
```

## Examples

### BAD: Surface-level QA

```
The document looks technically accurate. The code examples appear correct.
No major issues found.
```

**WHY BAD:** No claims enumerated, no examples tested, no sources cited. This
is a rubber stamp — indistinguishable from not checking at all. "Appear
correct" means nothing was actually verified.

### GOOD: Evidence-backed QA with classified findings

```
## QA Report

**Document:** docs/getting-started.md
**Claims checked:** 14
**Code examples tested:** 3

### Findings

#### Hallucination: Non-existent --verbose flag
- **Location:** §3 "Quick Start", line 42
- **Classification:** hallucination
- **Claim:** "Run `kanbanzai serve --verbose` for detailed output"
- **Source checked:** `cmd/serve.go` flag registration
- **Issue:** No `--verbose` flag exists. The flag is `--log-level debug`.
- **Recommendation:** Replace with `kanbanzai serve --log-level debug`

#### Vague claim: Unspecified performance improvement
- **Location:** §1 "Introduction", line 8
- **Classification:** vague
- **Claim:** "significantly improves workflow performance"
- **Source checked:** no benchmark data available
- **Issue:** No specifics given; could describe any tool
- **Recommendation:** Replace with measurable claim or delete

#### Stale reference: Dead link to design doc
- **Location:** §4 "Architecture", line 67
- **Classification:** stale
- **Claim:** "See [Design Overview](docs/design/overview.md)"
- **Source checked:** `docs/design/overview.md`
- **Issue:** File was moved to `work/design/overview.md` in commit a3b8f1c
- **Recommendation:** Update link to `work/design/overview.md`

#### Inflated: Unsupported importance claim
- **Location:** §2 "Core Concepts", line 19
- **Classification:** inflated
- **Claim:** "The entity system plays a crucial role in enabling…"
- **Source checked:** N/A — significance claim, not factual
- **Issue:** Promises importance without evidence; deleting the phrase
  loses no information
- **Recommendation:** Delete "plays a crucial role in enabling" — the
  sentence works without it

### Summary
- Hallucinations: 1
- Unverified claims: 0
- Stale references: 1
- Substance issues (vague/inflated/promotional): 2
- Verified claims: 10
```

**WHY GOOD:** Every finding has a location, classification, the exact text, the
source that was checked, and a specific recommendation. The summary gives a
clear picture. Claims that passed are counted. A human can verify each finding
independently.

## Evaluation Criteria

These criteria are for evaluating the QA output, not for self-evaluation during
the check. They are phrased as gradable questions to support automated
LLM-as-judge evaluation.

1. Is every factual claim in the document identified and checked against the
   implementation? **Weight: 0.25.**
2. Are code examples actually run or traced, with results compared to
   documented output? **Weight: 0.25.**
3. Are findings correctly classified (hallucination vs unverified vs vague vs
   inflated)? **Weight: 0.15.**
4. Does every finding cite the source that was checked? **Weight: 0.15.**
5. Are substance issues (vague claims, significance inflation, promotional
   language) flagged? **Weight: 0.10.**
6. Does the check stay within scope — flagging issues without rewriting prose?
   **Weight: 0.10.**

## Questions This Skill Answers

- Is this document factually accurate?
- Do the code examples actually work?
- Are there hallucinated APIs, flags, or behaviours?
- Are there vague claims that need specifics?
- Which claims could not be verified?
- Are version numbers and links current?