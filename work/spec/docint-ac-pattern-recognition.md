# Specification: `docint` Acceptance Criteria Pattern Recognition

| Field    | Value                                                              |
|----------|--------------------------------------------------------------------|
| Status   | Draft                                                              |
| Created  | 2026-04-01                                                         |
| Updated  | 2026-04-01                                                         |
| Feature  | `FEAT-01KN4ZPCMJ1FP` (docint-ac-pattern-recognition)              |
| Design   | `work/design/kanbanzai-2.5-infrastructure-hardening.md` §3        |
| Plan     | P15 — Kanbanzai 2.5 Infrastructure Hardening                      |

---

## 1. Purpose

This specification defines requirements for extending the acceptance criteria
extractor (`asmExtractCriteria`) to recognise the bold-identifier format used
in Kanbanzai specification documents. The extractor currently silently returns
zero criteria when parsing specs that use `**AC-NN.**` notation, causing
`decompose propose` to fall back to section headers and produce meaningless
task summaries.

---

## 2. Scope

### 2.1 In Scope

- Extending `asmExtractCriteria` in `internal/mcp/assembly.go` to recognise
  the bold-identifier pattern `**XX-NN.**` (where `XX` is one or more uppercase
  letters and `NN` is one or more digits).
- Context-sensitive extraction rules: all bold-identifier lines extracted in
  acceptance/criteria/requirement sections; only RFC 2119-containing lines
  extracted elsewhere.
- Preserving the identifier prefix in the extracted criterion text.
- Regression coverage ensuring existing list-item and numbered-list extraction
  is unaffected.

### 2.2 Out of Scope

- Changes to the document intelligence taxonomy or classifier
  (`internal/docint/taxonomy.go`).
- Changes to how `decompose propose` constructs task summaries (it consumes
  extracted criteria; this fix makes extraction correct).
- Support for bold-identifier variants that do not follow the `**XX-NN.**`
  format (e.g., inline bold mid-sentence).
- Changes to any stored document records or the document index.

---

## 3. Dependencies and Assumptions

- **DEP-01.** `asmExtractCriteria` is the sole extraction function responsible
  for producing criteria lines that `decompose propose` consumes. No other
  path produces criteria for decomposition.
- **DEP-02.** The document intelligence classifier already identifies sections
  whose heading role is `acceptance-criteria`, `requirements`, or `constraints`.
  This classification is available to `asmExtractCriteria` at call time.
- **ASM-01.** The bold-identifier pattern `**XX-NN.**` is only used
  intentionally as a criterion or requirement identifier in Kanbanzai documents;
  it is not expected to appear as decorative formatting in running prose.
- **ASM-02.** RFC 2119 keywords for the purposes of this specification are:
  `MUST`, `MUST NOT`, `SHALL`, `SHALL NOT`, `SHOULD`, `SHOULD NOT`, `MAY`,
  `REQUIRED`, `RECOMMENDED`, `OPTIONAL` (case-sensitive, per RFC 2119).

---

## 4. Requirements

### 4.1 Bold-Identifier Pattern

**REQ-01.** The extractor MUST recognise lines of the form:

```
**XX-NN.** <criterion text>
```

where `XX` is one or more uppercase ASCII letters, `NN` is one or more ASCII
digits, and `<criterion text>` is any non-empty text on the same line following
the closing `**`.

**REQ-02.** The extractor MUST also recognise the variant with a trailing
period inside the bold markers:

```
**XX-NN.** <criterion text>
```

is the canonical form. No other structural variants (e.g., `*XX-NN.*`,
`XX-NN:`, `__XX-NN.__`) need be supported.

**REQ-03.** Recognition MUST be case-sensitive for the identifier prefix (`XX`
MUST be uppercase letters). Lines where `XX` contains lowercase letters do not
match this pattern.

### 4.2 Context-Sensitive Extraction Rules

**REQ-04.** In sections whose classifier role is `acceptance-criteria`,
`requirements`, or `constraints`, ALL lines matching the bold-identifier pattern
MUST be extracted as criteria, regardless of whether they contain RFC 2119
keywords.

**REQ-05.** In sections whose classifier role is anything other than
`acceptance-criteria`, `requirements`, or `constraints`, a bold-identifier line
MUST be extracted only if its criterion text also contains at least one RFC 2119
keyword.

**REQ-06.** The context-sensitive rules in REQ-04 and REQ-05 MUST NOT alter
the behaviour of the existing extraction rules for standard list items (`-`,
`*`, `+`, `•`) and numbered lists (`1. `, `2. `, etc.). Those rules remain
independent.

### 4.3 Extracted Criterion Format

**REQ-07.** The extracted criterion text MUST include the identifier prefix,
formatted as `XX-NN: <criterion text>`. The bold markers (`**`) are stripped;
the trailing period after `NN` inside the bold span is stripped; the colon and
space before the criterion text are added by the extractor.

Example: the line `**AC-01.** The system must do X.` is extracted as
`AC-01: The system must do X.`

**REQ-08.** The criterion text portion (after the identifier) MUST be
reproduced verbatim, including any inline formatting, punctuation, and RFC 2119
keywords.

### 4.4 Identifier Prefix Coverage

**REQ-09.** The extractor MUST recognise at minimum the following identifier
prefixes when they appear in appropriate sections: `AC`, `REQ`, `C`, `INV`,
`DEP`, `ASM`. The recognition mechanism MUST be generic (matching any
`XX-NN.` pattern) rather than enumerating specific prefixes.

### 4.5 Regression Invariants

**REQ-10.** All existing extraction behaviour for standard list items MUST
continue to operate correctly. A document that previously produced N criteria
via list-item extraction MUST still produce those same N criteria after this
change.

**REQ-11.** All existing extraction behaviour for numbered lists MUST continue
to operate correctly under the same condition as REQ-10.

**REQ-12.** A document containing no bold-identifier lines and no list items
must still produce zero criteria (no spurious extraction).

### 4.6 End-to-End Behaviour

**REQ-13.** When `decompose propose` is called on a specification document
whose acceptance criteria section uses exclusively the `**AC-NN.**` format,
the generated task summaries MUST be derived from the extracted criterion text,
not from section headings.

**REQ-14.** REQ-13 is satisfied if at least one extracted criterion produces a
recognisably criterion-derived task summary and no task summary is of the form
"Implement \<section heading\>" when a criteria section was present.

---

## 5. Acceptance Criteria

**AC-01.** `asmExtractCriteria` extracts criteria from lines matching
`**AC-NN.** text` when those lines appear in a section with role
`acceptance-criteria`.

**AC-02.** `asmExtractCriteria` extracts criteria from lines matching
`**REQ-NN.** text` and `**C-NN.** text` when those lines appear in sections
with role `requirements` or `constraints` respectively.

**AC-03.** The extracted criterion text for each matched line preserves the
identifier prefix, e.g. `AC-01: The system must...`.

**AC-04.** Bold-identifier lines appearing outside acceptance/requirement/
constraint sections are only extracted when their text contains an RFC 2119
keyword.

**AC-05.** Existing list-item and numbered-list extraction is unaffected;
a regression test with a document using only list-item criteria produces the
same results before and after this change.

**AC-06.** `decompose propose` on a specification file that uses `**AC-NN.**`
format produces task summaries derived from the criteria text, not from
section headers.

---

## 6. Non-Requirements

The following are explicitly not required by this specification:

- The extractor need not detect malformed bold-identifier lines (e.g.,
  `**AC-01**` without a trailing period, or `**AC-01.`** with mismatched
  markers). Such lines may be silently skipped.
- The extractor need not validate that identifier numbers are sequential or
  non-duplicate.
- No changes to the document record store, index, or any persistence layer are
  required.
- No UI or CLI changes are required.