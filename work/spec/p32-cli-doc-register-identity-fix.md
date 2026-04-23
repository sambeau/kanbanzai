| Field  | Value                                              |
|--------|----------------------------------------------------|
| Date   | 2026-04-23                                         |
| Status | Draft                                              |
| Author | spec-author                                        |
| Plan   | P32-doc-intel-classification-pipeline-hardening    |
| Feature | FEAT-01KPX5CW2PTMD — CLI doc register Identity Resolution Fix |

## Problem Statement

This specification covers the fix for `kbz doc register` failing with `created_by is required` when the `--by` flag is omitted, even when the user's identity is resolvable from `.kbz/local.yaml` or `git config user.name`.

> This specification implements the design described in
> `work/design/p32-independent-fixes.md` — section "Fix 1 — CLI doc register identity resolution".

The root cause is that `runDocRegister` in `cmd/kanbanzai/doc_cmd.go` passes the raw `--by` value (empty string when omitted) directly to `docSvc.SubmitDocument`, bypassing `config.ResolveIdentity`. The MCP `doc(action: "register")` tool already calls `config.ResolveIdentity` and succeeds without an explicit identity argument. This asymmetry causes the CLI to fail in the common case where a user's identity is already configured but `--by` is not supplied.

The fix is a single call to `config.ResolveIdentity(createdBy)` added to `runDocRegister` after flag parsing, matching the pattern used by `runWorktreeCreate`, `runMergeRun`, and `runImport`.

**In scope:**
- Identity resolution behaviour of `kbz doc register` when `--by` is omitted
- Explicit `--by` override behaviour (must be preserved)
- Error reporting when identity cannot be resolved from any source

**Explicitly out of scope:**
- The MCP `doc(action: "register")` tool (already correct — no change)
- `service.SubmitDocument` interface or implementation (no change)
- Any other CLI command's identity resolution
- Fix 2 of the same design document (MCP JSON tag audit — covered by a separate specification)

---

## Requirements

### Functional Requirements

- **REQ-001:** When `--by` is omitted and a `user` field is present in `.kbz/local.yaml`, `kbz doc register` MUST use that value as `created_by` without error.

- **REQ-002:** When `--by` is omitted and `.kbz/local.yaml` does not contain a `user` field (or the file does not exist), `kbz doc register` MUST fall back to the value of `git config user.name` as `created_by`.

- **REQ-003:** When `--by` is provided explicitly, `kbz doc register` MUST use the supplied value as `created_by`, ignoring `.kbz/local.yaml` and `git config user.name`.

- **REQ-004:** When `--by` is omitted and identity cannot be resolved from `.kbz/local.yaml` or `git config user.name`, `kbz doc register` MUST return an error and MUST NOT register the document.

- **REQ-005:** The `--by` flag MUST remain present in the command's help text as an optional flag for explicit identity override.

### Non-Functional Requirements

- **REQ-NF-001:** The identity resolution step MUST add no observable latency beyond the time taken by `git config user.name` lookup (a single subprocess call), which must complete within 500 ms on any supported platform under normal conditions.

- **REQ-NF-002:** The error message returned when identity cannot be resolved MUST be the same message already produced by `config.ResolveIdentity` for this failure mode, with no additional wrapping, so that the failure reason is immediately clear to the user.

---

## Constraints

- `service.SubmitDocument` MUST NOT be modified. It receives a resolved identity string from its caller; it does not perform resolution itself.
- The MCP `doc(action: "register")` tool MUST NOT be modified. Its behaviour is already correct.
- `config.ResolveIdentity` is the only permitted identity resolution mechanism. No new resolution logic may be introduced in `runDocRegister`.
- The `--by` flag signature (name, type, default value of `""`) MUST NOT change. Existing scripts that pass `--by` explicitly must continue to work without modification.
- This specification does NOT cover the MCP JSON tag audit (Fix 2 of `p32-independent-fixes.md`).
- This specification does NOT cover identity resolution for any CLI command other than `doc register`.

---

## Acceptance Criteria

- **AC-001 (REQ-001):** Given `.kbz/local.yaml` contains `user: alice` and `--by` is not supplied, when `kbz doc register` is invoked with a valid path, type, and title, then the document is registered with `created_by: alice` and the command exits with status 0.

- **AC-002 (REQ-002):** Given `.kbz/local.yaml` does not contain a `user` field (or the file is absent) and `git config user.name` returns `bob`, and `--by` is not supplied, when `kbz doc register` is invoked with a valid path, type, and title, then the document is registered with `created_by: bob` and the command exits with status 0.

- **AC-003 (REQ-003):** Given `.kbz/local.yaml` contains `user: alice` and `git config user.name` returns `bob`, when `kbz doc register` is invoked with `--by carol` and a valid path, type, and title, then the document is registered with `created_by: carol` and the command exits with status 0.

- **AC-004 (REQ-004):** Given `.kbz/local.yaml` does not contain a `user` field (or is absent) and `git config user.name` returns an empty value or error, and `--by` is not supplied, when `kbz doc register` is invoked, then the command exits with a non-zero status, prints an error indicating identity could not be resolved, and no document is registered.

- **AC-005 (REQ-005):** Given `kbz doc register --help` is invoked, then the output includes `--by` in the flags list with a description indicating it is the identity override.

- **AC-006 (REQ-NF-001):** Given a system under normal load where `git config user.name` responds within 500 ms, when `kbz doc register` is invoked without `--by`, then the command completes within the same time budget as before the fix plus the `git config` subprocess duration.

- **AC-007 (REQ-NF-002):** Given identity cannot be resolved, when `kbz doc register` returns an error, then the error text is the unmodified message from `config.ResolveIdentity` with no additional prefix or wrapping beyond what `runDocRegister` already applies to other errors.

---

## Verification Plan

| Criterion | Method      | Description |
|-----------|-------------|-------------|
| AC-001    | Test        | Unit/integration test: invoke `runDocRegister` with a temporary `.kbz/local.yaml` containing `user: alice` and no `--by` flag; assert registered `created_by` equals `alice` and no error is returned. |
| AC-002    | Test        | Unit/integration test: invoke `runDocRegister` with no `user` in `.kbz/local.yaml` and a mocked `git config user.name` returning `bob`; assert registered `created_by` equals `bob` and no error is returned. |
| AC-003    | Test        | Unit/integration test: invoke `runDocRegister` with `--by carol` regardless of `.kbz/local.yaml` and git config values; assert registered `created_by` equals `carol`. |
| AC-004    | Test        | Unit/integration test: invoke `runDocRegister` with no resolvable identity (empty `.kbz/local.yaml`, git config returns empty/error) and no `--by` flag; assert non-zero exit and that `docSvc.SubmitDocument` is not called. |
| AC-005    | Inspection  | Code review and manual `--help` output check: confirm `--by` flag is declared on the `doc register` subcommand with appropriate usage text. |
| AC-006    | Inspection  | Code review: confirm the only new operation introduced is a single `config.ResolveIdentity(createdBy)` call; no additional I/O or subprocess invocations are added beyond what `ResolveIdentity` already performs. |
| AC-007    | Inspection  | Code review: confirm the error returned from `config.ResolveIdentity` is propagated directly (e.g. `return err` or `return fmt.Errorf(..., err)`) with no text substitution that would obscure the original message. |