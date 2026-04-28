# Hardening Dev-Plan

| Document | Hardening Feature Dev-Plan |
|----------|---------------------------|
| Feature  | FEAT-01KMKRQWF0FCH         |
| Spec     | work/spec/hardening.md     |
| Status   | Draft                      |
| Created  | 2026-03-26                 |

---

## Overview

This dev-plan decomposes the hardening feature into tasks. Hardening covers two independent
tracks that can run in parallel, plus two tracks that require the `kanbanzai init` command
(FEAT-01KMKRQRRX3CC) to be substantially complete before they can be verified.

**Independent track (Wave 1 — parallel with init-command):**
- `doc_record_refresh` MCP tool
- MCP tool safety annotations

**Init-dependent track (Wave 3 — after init-command):**
- Error message quality audit
- CLI help text
- Partial initialisation sentinel and detection

---

## Tasks

### TASK-H1: `doc_record_refresh` MCP tool

Implement the `doc_record_refresh` MCP tool as specified in §9 of the hardening spec.

**Work:**
- Add `doc_record_refresh` handler in `internal/mcp/`
- Logic: read current file at the document's recorded path, compute SHA-256 hash
- If hash unchanged: return `{ changed: false }` without modifying the record
- If hash changed and status is `approved`: transition to `draft`, return old hash, new hash, and a message explaining the status transition
- If hash changed and status is `draft`: update hash only, return old and new hash
- Error conditions: document ID not found, file not found at recorded path, file read error
- Register the tool with appropriate safety annotations (idempotent: false, destructive: false)
- Write tests covering AC-11, AC-12, AC-13

**Spec references:** §9, AC-11, AC-12, AC-13

---

### TASK-H2: MCP tool safety annotations

Add `readOnlyHint`, `destructiveHint`, and `idempotentHint` annotations to every MCP tool
registration in the Kanbanzai server, as specified in §10 of the hardening spec.

**Work:**
- Review the full tool classification table in spec §10.4
- For each tool, set all three annotations in the tool registration struct
- Ensure no tool is missing any annotation (AC-14)
- Verify annotation accuracy: tools with `readOnlyHint: true` must not write files or call Git; tools with `destructiveHint: false` must not delete or irreversibly modify data (AC-15)
- Add a test that reflects over all registered tools and asserts each has all three annotations set

**Spec references:** §10, AC-14, AC-15

---

### TASK-H3: Error message quality audit

Audit every error message produced by the CLI and MCP server against the quality standards
in §5 of the hardening spec. Fix any message that fails.

**Work:**
- Enumerate all `fmt.Errorf`, `errors.New`, and MCP error response sites
- Check each against §5.1 (required structure: what went wrong + what to do) and §5.2 (prohibited: Go type names, raw YAML field names, stack traces)
- Rewrite non-conforming messages in plain language with concrete action guidance
- Pay particular attention to storage-layer errors that bubble up to CLI output
- Verify MCP error responses conform to §5.4 (structured `{ code, message, data }` shape)

**Spec references:** §5, AC-01

**Dependency:** Can begin independently but final verification requires a working `init` command.

---

### TASK-H4: CLI help text

Implement top-level and per-command `--help` output meeting the discoverability standard
in §7 of the hardening spec.

**Work:**
- Top-level `kanbanzai --help`: list every subcommand with a one-line description, grouped by category (AC-09)
- Per-command `kanbanzai <command> --help`: all flags with types, descriptions, and defaults; at least one usage example; exits 0 (AC-10)
- Verify every existing command produces non-empty help output
- Ensure `kanbanzai init --help` lists `--docs-path`, `--skip-skills`, `--update-skills`, `--non-interactive` with correct descriptions

**Spec references:** §7, AC-09, AC-10

**Dependency:** Requires `kanbanzai init` command (FEAT-01KMKRQRRX3CC) to be present.

---

### TASK-H5: Partial initialisation sentinel

Implement the `.kbz/.init-complete` sentinel file and partial-state detection as specified
in §8 of the hardening spec.

**Work:**
- In `kanbanzai init`, write `.kbz/.init-complete` as the final step after all files are created (AC-02)
- On any Kanbanzai command invocation: if `.kbz/` exists but `.init-complete` is absent, detect this as partial initialisation, print a warning with a concrete recovery action, and do not proceed silently (AC-08)
- Ensure `kanbanzai init` atomicity: if init fails after beginning to write files, clean up all created files before exit so no partial `.kbz/` is left (AC-07)
- Add tests: interrupted init leaves no partial state; detection triggers on next command; warning message is actionable

**Spec references:** §8, AC-07, AC-08

**Dependency:** Requires `kanbanzai init` command (FEAT-01KMKRQRRX3CC) to be substantially complete.

---

## Delivery Order

```
Wave 1 (parallel with init-command implementation):
  TASK-H1: doc_record_refresh tool
  TASK-H2: MCP safety annotations

Wave 3 (after init-command is complete):
  TASK-H3: Error message audit
  TASK-H4: CLI help text
  TASK-H5: Partial init sentinel
```

Wave 1 tasks are fully independent. Wave 3 tasks require a working `init` command for
end-to-end verification, though TASK-H3 can begin independently (it touches all error
message sites, not just init).