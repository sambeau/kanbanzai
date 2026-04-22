# Doc-Intel Layer 3 Classification: Pilot Lessons

> **Purpose:** Capture bugs, process insights, and enhancement opportunities discovered
> during the first full-corpus Layer 3 classification run (339 documents, April 2026).
>
> **Created:** 2026-04-22
> **Status:** Draft
> **Audience:** Product / engineering team planning future doc-intel enhancements

---

## 1. Executive Summary

The first full-corpus Layer 3 classification run successfully classified all 339 pending
documents in a single session using parallel sub-agent batches. The run was productive
but revealed two bugs in the existing implementation, generated clear data on batch
sizing limits, and produced four concrete enhancement recommendations that would make
future classification runs — and the ongoing "classify on register" workflow — materially
faster and more reliable.

| Metric | Value |
|--------|-------|
| Documents classified | 339 |
| Sections classified | ~10,000 (estimated from batch reports) |
| Bugs found and fixed | 2 |
| Sub-agent batches dispatched | ~35 across 6 waves |
| Batches that hit token limit | ~7 (all recovered via `pending` recheck) |

---

## 2. Background

Before this session, 339 documents had no Layer 3 classifications. The corpus covered the
full project history — specs, designs, dev-plans, reports, and research documents from
plan P3 through P27.

The loop per document was:

```
doc_intel(action: "guide")  →  read_file  →  doc_intel(action: "classify")
```

The session proceeded in waves, each dispatching 4–8 parallel sub-agents. All findings
below are direct observations from running this loop at scale.

---

## 3. Bugs Discovered and Fixed

### 3.1 Missing JSON tags on `Classification` struct

**File:** `internal/docint/types.go`  
**Severity:** Critical — blocked all classification attempts  
**Status:** Fixed

**Root cause.** The `Classification` struct carried only YAML struct tags:

```go
type Classification struct {
    SectionPath string `yaml:"section_path"`
    Role        string `yaml:"role"`
    Confidence  string `yaml:"confidence"`
    ...
}
```

The `docIntelClassifyAction` handler reads the `classifications` parameter as a raw JSON
string and then calls `json.Unmarshal` to decode it into `[]Classification`. Go's JSON
decoder matches keys against struct field names case-insensitively, but does **not**
normalise underscores — so `"section_path"` (documented, snake_case) never matched
`SectionPath` (CamelCase). Every classification silently decoded with `SectionPath = ""`
and was rejected by the validator with `"unknown section_path \"\""`.

**Why it was hard to spot.** The first attempt used float confidence values (e.g. `0.95`),
which caused a JSON unmarshal error on the `confidence` field before validation ran,
masking the `section_path` failure entirely. The second attempt, with string confidence
values, reached validation and exposed the empty-path errors — but the error message
pointed at section paths, not struct tags.

**Fix.** Added `json:"..."` tags alongside the existing `yaml:"..."` tags on
`Classification` and `ConceptIntroEntry`. The tool description and skill file were also
updated to document the correct field names and valid values explicitly.

**Implication for the wider codebase.** Any struct that is stored as YAML (and therefore
has `yaml` tags) but is *also* decoded from JSON tool parameters via `req.RequireString`
+ `json.Unmarshal` is vulnerable to the same issue. A one-time audit of MCP parameter
structs is warranted (see §5.5).

---

### 3.2 Invalid role examples in `kanbanzai-documents` skill

**File:** `.agents/skills/kanbanzai-documents/SKILL.md`  
**Severity:** Moderate — causes unnecessary failed classify calls  
**Status:** Fixed

**Root cause.** The classify step-by-step procedure cited `context` and `procedure` as
example roles:

> *"Produce classification objects for each section, assigning one or more roles
> (e.g. `requirement`, `decision`, `rationale`, `context`, `procedure`)."*

Neither `context` nor `procedure` is in the `FragmentRole` taxonomy. Any agent following
the skill example verbatim would produce a validation error on the first document it
attempted to classify.

**Fix.** The skill section was rewritten to include a full role-reference table (all 11
valid roles with descriptions), valid confidence values (`high`, `medium`, `low`), and a
concrete JSON example. The stale role names were removed.

**Implication.** Taxonomy changes (adding or removing roles) need a corresponding update
to all skill and role files that reference example roles. A lint step cross-checking skill
content against the Go taxonomy constants would prevent future drift.

---

## 4. Process Insights

### 4.1 Effective batch size is section-count-driven, not document-count-driven

Sub-agent batches of 15 FEAT-level specification documents (averaging 15–25 sections
each, ~250–350 total sections per batch) completed consistently. Batches containing
PROJECT-level documents (design documents with 50–183 sections, phase specifications with
95–181 sections) failed with output token limit errors, even when the document count was
kept to 7 or fewer.

The practical threshold observed was approximately **150–200 total sections per batch**.
Beyond that, the combination of reading large documents and generating classification
arrays consumes enough output tokens to trigger the limit before all documents complete.

Since section count is not exposed in the `pending` response (see §5.1), the working
heuristic was:

| Document family | Typical section count | Recommended batch size |
|-----------------|-----------------------|------------------------|
| FEAT-level specs, dev-plans | 6–30 | 8–15 |
| FEAT-level designs | 10–65 | 8–12 |
| Plan-level designs / dev-plans | 20–95 | 5–8 |
| PROJECT-level designs | 10–90 | 5–8 |
| PROJECT-level phase specs | 60–183 | 2–4 |
| Reports, retrospectives, research | 5–130 | 8–13 |

This heuristic was developed through trial and error across Waves 1–4. It would not have
been necessary if section counts were available in the `pending` response.

---

### 4.2 The "concise output" instruction is load-bearing for bulk operations

Wave 3 batches used the same document count as the successful Wave 1 batches but failed.
The only material difference was that Wave 4 onwards added:

> *"Be concise — no commentary between documents, just run the tools. Report only final
> counts."*

This instruction reduced per-document output from ~200–400 tokens of classification
reasoning to ~5–10 tokens of summary, freeing the budget for the actual tool calls and
their responses. All Wave 4+ batches succeeded at the same document counts that had
previously failed.

**Recommendation.** Any sub-agent prompt for a bulk data operation (classification,
registration, index repair) should default to minimal-output mode. Verbose reasoning
should be opt-in, used only when debugging a specific failure.

---

### 4.3 `classify` calls are atomic; batch failures are fully recoverable

When a batch hits the output token limit mid-run, documents that had already received a
successful `classify` call remain committed in the persistent index. The `pending` list
is the authoritative ground truth — checking it after any failure immediately reveals
what was completed and what remains.

In practice, most "failed" batches were partially successful: of the ~7 batches that hit
the token limit, all but one had completed at least half their documents before failing.
No data was lost in any failure.

**Recommendation.** Document this property explicitly in the skill file so future
operators know to check `pending` after a batch failure rather than assuming all work
was lost and re-dispatching blindly (which would attempt to re-classify already-classified
documents, wasting tokens on no-ops or errors).

---

## 5. Enhancement Recommendations

### 5.1 Add section count to the `pending` response

**Current behaviour.** `doc_intel(action: "pending")` returns a flat list of document IDs
with no additional metadata.

**Problem.** Agents planning batch sizes have no way to know which documents are large
(100+ sections) versus small (5–10 sections) without calling `guide` on each one
individually — which defeats the purpose of planning.

**Proposed change.** Include a `section_count` (or at minimum a size bucket:
`small` / `medium` / `large`) alongside each document ID in the `pending` response.
The section count is already stored in the Layer 1 index and requires no additional
computation.

**Benefit.** Agents can right-size batches from a single `pending` call rather than
through trial-and-error. This would have reduced the number of failed batches in this
session from ~7 to approximately zero.

---

### 5.2 Embed the role taxonomy in the `guide` response

**Current behaviour.** The `guide` response includes an outline, entity refs, and
extraction hints, but does not include the list of valid roles or confidence values.
Agents must know these from memory or from reading the skill file.

**Problem.** Every classify attempt that failed on role names (`context`, `procedure`,
float confidence values) did so because the agent lacked the taxonomy at call time.
Even after fixing the skill file, agents working from a cold context window remain
exposed to the same failure.

**Proposed change.** Add a `taxonomy` block to the `guide` response:

```json
"taxonomy": {
  "roles": [
    "requirement", "decision", "rationale", "constraint", "assumption",
    "risk", "question", "definition", "example", "alternative", "narrative"
  ],
  "confidence": ["high", "medium", "low"]
}
```

**Benefit.** The classify loop becomes self-contained. An agent that has never read the
skill file can correctly classify any document using only the information in the `guide`
response. The tool also becomes resilient to future taxonomy additions — the guide
response is always the authoritative source.

---

### 5.3 Support outline-only classification for well-structured documents

**Current behaviour.** The skill requires reading document content before classifying.
The `guide` step already performs lightweight keyword-based role detection on section
titles, but these hints are advisory and incomplete.

**Observation.** For well-structured documents following standard templates, section
roles are almost entirely determinable from headings alone:

| Heading pattern | Role (always) |
|----------------|---------------|
| "Acceptance Criteria", "AC-\d+" | `requirement` |
| "Purpose", "Motivation", "Problem Statement" | `rationale` |
| "Scope", "In Scope", "Deferred", "Excluded" | `constraint` |
| "Glossary", "Definitions", "Reference Table" | `definition` |
| "Example", "Sample" | `example` |
| "Alternatives Considered" | `alternative` |
| Front matter metadata table | `narrative` |

For approximately 60% of sections encountered in this run, the heading alone was
sufficient for high-confidence classification.

**Proposed change.** Expand the `guide` response to include `"suggested_classifications"`
— pre-populated role assignments for sections where the heading match is unambiguous
(confidence `high`). Agents review and override rather than derive from scratch.
Sections without a strong heading match are left for the agent to fill in after reading.

**Benefit.** Eliminates the `read_file` step for the majority of sections in
well-structured documents, reducing the loop from 3 tool calls to 2. Significant context
budget saving for backlog-clearance runs and for the ongoing classify-on-register
workflow.

---

### 5.4 Include `content_hash` and outline in the register classification nudge

**Current behaviour.** `doc(action: "register")` returns a `classification_nudge`
telling the agent to classify the document. The agent must then call
`doc_intel(action: "guide")` to retrieve the `content_hash` and section outline before
classifying.

**Problem.** The `content_hash` and outline are computed at registration time and are
already known to the server when the nudge is generated. Requiring an extra `guide` call
to retrieve them adds a round-trip for every registered document.

**Proposed change.** Include `content_hash` and `outline` directly in the
`classification_nudge` payload returned by `doc register`.

**Benefit.** The classify-on-register workflow shrinks from 3 tool calls
(`register` → `guide` → `classify`) to 2 (`register` → `classify`). At scale, this
saves one tool call per document. For the ongoing single-document case at registration
time, it removes the most common friction point in the workflow.

---

### 5.5 Audit MCP parameter structs for missing JSON tags

**Scope.** Any struct that (a) is stored as YAML state and therefore has `yaml:` tags,
and (b) is decoded from a JSON string tool parameter via `json.Unmarshal`, is vulnerable
to the silent deserialization failure described in §3.1.

**Proposed change.** Add a Go test or `go vet` check asserting that every exported field
on parameter structs used with `json.Unmarshal` has an explicit `json:` tag.
Alternatively, run a one-time audit and add JSON tags proactively to all candidate
structs.

**Priority.** Medium. The `Classification` and `ConceptIntroEntry` structs are fixed.
Other structs may not be affected today but could be silently broken if future handlers
follow the same `RequireString` + `json.Unmarshal` pattern without awareness of this
pitfall.

---

## 6. Priority Summary

| # | Item | Impact | Effort | Priority |
|---|------|--------|--------|----------|
| 3.1 | JSON tags on `Classification` *(done)* | Critical | Trivial | Done |
| 3.2 | Fix stale role examples in skill *(done)* | High | Trivial | Done |
| 5.2 | Taxonomy in `guide` response | High | Low | **P1** |
| 5.1 | Section count in `pending` response | High | Low | **P1** |
| 5.4 | `content_hash` + outline in register nudge | Medium | Low | **P2** |
| 5.3 | Suggested classifications in `guide` | Medium | Medium | **P2** |
| 5.5 | Audit MCP structs for missing JSON tags | Medium | Low | **P2** |

---

## 7. Appendix: Batch Strategy Reference

For future full-corpus or partial backlog runs, the following conventions produced
zero failures in Waves 4–6 of this session.

**Sub-agent prompt requirements for bulk classification:**

1. Concise-output instruction: *"Be concise — no commentary between documents, just run
   the tools. Report only final counts."*
2. Large-file guidance: *"If `read_file` returns a symbol outline, use `start_line` /
   `end_line` to read in chunks — section headings alone are sufficient to assign
   roles confidently."*
3. Recovery instruction: *"After any failed batch, call `doc_intel(action: 'pending')`
   to confirm ground truth before re-dispatching."*

**Classify call reference:**

```
doc_intel(
  action:        "classify",
  id:            "<document_id>",
  content_hash:  "<hash from guide response>",
  model_name:    "<model>",
  model_version: "<version>",
  classifications: "[
    {\"section_path\": \"1\",   \"role\": \"narrative\",   \"confidence\": \"high\"},
    {\"section_path\": \"1.1\", \"role\": \"rationale\",   \"confidence\": \"high\"},
    {\"section_path\": \"1.2\", \"role\": \"requirement\", \"confidence\": \"high\"}
  ]"
)
```

Valid roles: `requirement`, `decision`, `rationale`, `constraint`, `assumption`, `risk`,
`question`, `definition`, `example`, `alternative`, `narrative`

Valid confidence: `high`, `medium`, `low`
