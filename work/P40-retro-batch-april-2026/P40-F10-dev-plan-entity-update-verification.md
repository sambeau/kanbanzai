# Dev-Plan: Add Verification Parameter to Entity Update (B41-F4)

| Field  | Value                         |
|--------|-------------------------------|
| Date   | 2026-04-30                    |
| Status | approved |
| Author | Architect                     |
| Feature | FEAT-01KQG3AX1AD0K           |
| Spec   | B41-fix-doc-ownership-lifecycle/spec-p40-spec-b41-doc-ownership-lifecycle |

---

## Overview

Add `verification` (string) and `verification_status` ("passed" | "failed")
optional parameters to `entity(action: update)`. When provided, write directly
to the entity record with no lifecycle transition. Reject for entity types that
don't support verification fields.

---

## Task Breakdown

### T1 — Add verification params to entity update handler

**Deliverable:** Updated entity update handler accepting verification fields.

**Scope:**
- Add `verification` and `verification_status` as optional string parameters.
- When provided for features: write directly to entity record.
- When provided for unsupported types (plans, batches): return clear error.
- No lifecycle transition triggered (REQ-007).

**Dependencies:** None.

**Verification:** Code review. Tests below.

**Estimated effort:** 1

### T2 — Write tests for verification parameter

**Deliverable:** Tests covering all verification parameter scenarios.

**Scope:**
- Set verification on feature → fields updated, no transition (AC-008).
- Set verification, then call merge check → verification gates pass (AC-009).
- Set verification on batch → error returned (AC-010).
- Existing entity update calls unchanged (AC-012).

**Dependencies:** T1.

**Verification:** `go test ./...` in entity service package.

**Estimated effort:** 1

---

## Dependency Graph

```
T1 (verification params) ──→ T2 (tests)
```

---

## Interface Contracts

| Boundary | Contract |
|----------|----------|
| T1 → T2 | After T1, entity update accepts verification and verification_status params. T2 tests all scenarios: feature update, merge gate pass-through, batch rejection, backward compat. |

---

## Traceability Matrix

| Requirement | Task(s) |
|-------------|---------|
| REQ-007 | T1 |
| REQ-008 | T1 |
| REQ-NF-003 | T1, T2 |
| AC-008 | T2 |
| AC-009 | T2 |
| AC-010 | T2 |
| AC-012 | T2 |
