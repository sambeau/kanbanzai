# Specification: Batch Handler False-Positive Fix

| Field    | Value                                                              |
|----------|--------------------------------------------------------------------|
| Status   | Draft                                                              |
| Created  | 2026-04-01                                                         |
| Updated  | 2026-04-01                                                         |
| Feature  | FEAT-01KN4ZPKCJWPF (batch-handler-false-positive)                  |
| Design   | `work/design/kanbanzai-2.5-infrastructure-hardening.md` §5         |

---

## 1. Purpose

This specification defines the requirements for correcting `ExecuteBatch` so that
MCP tool-result errors returned as `(data, nil)` are counted as failures rather
than successes. The current behaviour causes batch operations to report inflated
success counts when individual item handlers encounter validation or other errors,
making batch operations untrustworthy for automation.

---

## 2. Background and Scope

### 2.1 The Problem

The Kanbanzai MCP convention for inline errors is to return a structured error
payload as the data value with a `nil` Go error. `ExecuteBatch` only checks the
Go error return, so these payloads are indistinguishable from genuine successes
at the batch level. The result is a batch summary that silently overcounts
successes and undercounts failures.

### 2.2 In Scope

- Detection of tool-result error payloads in `ExecuteBatch`.
- Correct status and error message propagation to `ItemResult` for such items.
- Regression safety for items returning genuine success data.
- Regression safety for items returning Go errors.

### 2.3 Out of Scope

- Changes to any individual MCP tool handler's return convention.
- Changes to the `BatchItemHandler` function signature.
- New error types or sentinel values introduced outside the existing convention.
- Any change to how batch input parsing or dispatch works.

---

## 3. Dependencies and Assumptions

- **DEP-01.** The Kanbanzai MCP convention for inline tool errors is a
  `map[string]any` value containing an `"error"` key. This specification relies
  on that convention being stable and consistently used by all MCP tool handlers.
- **DEP-02.** No `BatchItemHandler` in the current codebase intentionally returns
  a `map[string]any` with an `"error"` key as a genuine (non-error) success
  payload. If any handler does, it must be updated before this fix ships.
- **ASM-01.** The `ItemResult` struct has a `Status` field that accepts at least
  `"ok"` (success) and `"error"` values, and an **Error field of type
  `*ErrorDetail`** — a struct with `Code` and `Message` string fields. The error
  message is conveyed via `ErrorDetail.Message`; `ErrorDetail.Code` carries a
  short machine-readable code.

---

## 4. Requirements

### 4.1 Tool-Result Error Detection

**REQ-01.** After a `BatchItemHandler` returns `(itemID, data, nil)`, `ExecuteBatch`
MUST inspect the `data` value for the tool-result error pattern before classifying
the item as succeeded.

**REQ-02.** The tool-result error pattern is: `data` is a `map[string]any` that
contains an `"error"` key whose value is a non-empty string.

**REQ-03.** An item whose handler returns `(itemID, data, nil)` where `data`
matches the tool-result error pattern MUST be counted as `failed`, not
`succeeded`, in the batch summary.

**REQ-04.** The `ItemResult` for such an item MUST have `Status: "error"`.

**REQ-05.** The `ItemResult.Error` field for such an item MUST be populated with
the string value of the `"error"` key from the tool-result payload.

### 4.2 Regression Invariants

**REQ-06.** A handler returning `(itemID, data, nil)` where `data` does NOT match
the tool-result error pattern MUST continue to be counted as `succeeded`.

**REQ-07.** A handler returning `(itemID, data, err)` where `err` is a non-nil
Go error MUST continue to be counted as `failed`, regardless of the `data` value.

**REQ-08.** The batch summary `succeeded` count MUST equal the number of items
classified as succeeded. The `failed` count MUST equal the number of items
classified as failed. These counts MUST be mutually exclusive and exhaustive.

### 4.3 No Change to Handler Contract

**REQ-09.** The `BatchItemHandler` function signature MUST NOT change as a result
of this fix.

**REQ-10.** No existing call site that constructs a `BatchItemHandler` MUST
require modification as a result of this fix, except for any handler identified
under DEP-02.

---

## 5. Acceptance Criteria

**AC-12.** When a batch item handler returns `(itemID, data, nil)` where `data`
is a `map[string]any` containing an `"error"` key, `ExecuteBatch` counts the
item as `failed`, not `succeeded`.

**AC-13.** The `ItemResult` for such an item has `Status: "error"` and its
`Error` field contains the message from the tool-result payload.

**AC-14.** A handler returning genuine success data — `(itemID, data, nil)` where
`data` does not contain an `"error"` key — is still counted as `succeeded` with
no change in result shape.

**AC-15.** A handler returning a non-nil Go error — `(itemID, _, err)` — is still
counted as `failed` with the Go error message in `ItemResult.Error`.

---

## 6. Verification

| Criterion | Method                                                              |
|-----------|---------------------------------------------------------------------|
| AC-12     | Unit test: handler returns `(errorMap, nil)` → item counted failed  |
| AC-13     | Unit test: `ItemResult.Status == "error"`, `Error` matches payload  |
| AC-14     | Unit test: handler returns `(successData, nil)` → item counted succeeded |
| AC-15     | Unit test: handler returns `(_, _, err)` → item counted failed      |

All tests MUST pass under `go test ./...` with no regressions to the existing
batch test suite.