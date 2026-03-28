# agents-md-decompose-rule Specification

| Document | AGENTS.md Decompose Precondition Rule        |
|----------|----------------------------------------------|
| Status   | Approved                                     |
| Created  | 2026-03-28T12:04:00Z                         |
| Updated  | 2026-03-28T12:04:00Z                         |
| Feature  | FEAT-01KMT-58SKYM5C (agents-md-decompose-rule) |
| Plan     | P8-decompose-reliability                     |
| Design   | `work/design/decompose-reliability.md` §2 Fix 3 |

---

## 1. Purpose

This specification defines a documentation-only change to `AGENTS.md`: a
precondition rule added to the decomposition stage gate that instructs agents
to verify two conditions before calling `decompose propose`.

This is the immediate documentation-level safeguard described as Fix 3 in the
design. It ships independently of the code fixes in FEAT-01KMT-58TV8V9C and
takes effect as soon as `AGENTS.md` is updated.

---

## 2. Goals

1. Agents are explicitly instructed to confirm a spec document is in `approved`
   status before calling `decompose propose`.
2. Agents are explicitly instructed to call `index_repository` before calling
   `decompose propose` if the spec was registered in the current session.
3. The rule is placed in `AGENTS.md` at the decomposition stage gate so it is
   encountered in context, not buried elsewhere.
4. No other files are changed.

---

## 3. Scope

### 3.1 In scope

- Adding a precondition block to the Stage 5 (Dev Plan & Tasks) section of
  `AGENTS.md`.

### 3.2 Explicitly excluded

- Changes to any Go source files.
- Changes to any spec, design, or planning documents other than `AGENTS.md`.
- Changes to `work/bootstrap/bootstrap-workflow.md` (that is covered by the
  code-fix feature as part of a broader documentation pass, if needed).

---

## 4. Acceptance Criteria

**AC-01.** `AGENTS.md` contains a precondition block in the Stage 5 (Dev Plan
& Tasks) section, positioned before or immediately after the existing "Agent
role" bullet list for that stage.

**AC-02.** The precondition block states that the spec document must be in
`approved` status before `decompose propose` is called. The wording must make
clear this is a hard requirement, not a suggestion.

**AC-03.** The precondition block states that if the spec was registered in the
current session, `index_repository` must be called before `decompose propose`
to ensure the document intelligence index has processed the file.

**AC-04.** The precondition block gives a concrete corrective action for each
condition:
- Spec not approved → approve it first via `doc approve`.
- Spec registered this session → call `index_repository` then retry.

**AC-05.** The added text does not alter or remove any existing content in the
Stage 5 section.

**AC-06.** No files other than `AGENTS.md` are modified by this task.

---

## 5. Verification

After implementation:

1. `grep -n "index_repository" AGENTS.md` returns at least one match inside
   the Stage 5 section.
2. `grep -n "approved" AGENTS.md` returns at least one match inside the Stage
   5 section relating to spec status.
3. All other content in `AGENTS.md` is byte-for-byte identical to the previous
   version outside the added block.