# Hardening: Kanbanzai 1.0 Robustness Specification

| Document | Hardening: Kanbanzai 1.0 Robustness Specification |
|----------|---------------------------------------------------|
| Status   | Draft                                             |
| Created  | 2026-05-31                                        |
| Updated  | 2026-05-31                                        |
| Related  | `work/design/kanbanzai-1.0.md` §10                |

---

## 1. Purpose

This specification defines the robustness requirements for Kanbanzai 1.0. The goal is to ensure that a first-time user — working on a clean machine, without prior Kanbanzai experience — can install the tool, run `kanbanzai init`, and begin using it without encountering confusing errors, silent failures, or corrupted state.

Hardening is not a single feature but a cross-cutting quality layer applied across the CLI, MCP server, and storage layer. It covers the error message contract, edge case handling at initialisation, CLI discoverability, partial state detection, and MCP protocol annotations.

Two additional items are included in scope beyond the design document: a `doc_record_refresh` MCP tool that lets users update a document record's content hash after editing the file without re-registering the document from scratch, and MCP tool safety annotations that allow MCP clients to make informed auto-approval decisions without prompting the user for every operation.

---

## 2. Goals

1. All user-facing error messages explain what went wrong and what the user should do next. No technical internals (YAML field names, Go type names, stack traces) appear in output visible to users.
2. Every feature works correctly on a machine with no pre-existing Kanbanzai state — no `.kbz/` directory, no cached data, no prior configuration.
3. `kanbanzai init` handles all common failure modes (non-git directory, existing `.kbz/`, no write permission, conflicting `.skills/` files) with clear, actionable messages and no partial state left behind on failure.
4. `kanbanzai --help` and `kanbanzai <command> --help` are sufficient for a new user to understand available commands and their options without consulting external documentation.
5. When an operation is interrupted (disk full, process killed, network failure), the tool detects the resulting partial state on next invocation and reports it clearly rather than silently proceeding on corrupted data.
6. The `doc_record_refresh` MCP tool allows users to update a document record's content hash after editing the file, without re-registering the document from scratch.
7. All Kanbanzai MCP tools carry `readOnlyHint`, `destructiveHint`, and `idempotentHint` annotations, enabling MCP clients to make informed auto-approval decisions.

---

## 3. Scope

### 3.1 In scope

- Error message format contract: structure, language level, and prohibited content
- Clean-machine testing requirements: which surfaces must be tested and what "clean machine" means
- `kanbanzai init` edge case enumeration and expected behaviour for each case
- CLI help text requirements: coverage, format, and content rules for all commands
- Partial state detection: which operations produce detectable partial state and how the tool reports it
- `doc_record_refresh` MCP tool: behaviour, inputs, outputs, and error conditions
- MCP tool safety annotation scheme: annotation semantics, classification rules, and enumeration of all tools with their assigned annotations

### 3.2 Deferred

- Interactive repair of partial state (e.g. a guided wizard that walks the user through recovery steps)
- Automatic recovery from partial state without user intervention
- Structured error codes suitable for machine parsing (e.g. for CI pipeline integration)
- Telemetry or crash reporting

### 3.3 Explicitly excluded

- Migration from other workflow tools
- Backward compatibility with `.kbz/` directories created by pre-1.0 versions (covered separately)
- Repair of corruption caused by direct manual edits to `.kbz/` state files

---

## 4. Design Principles

**Fail loudly and early.** The tool should detect problems at the earliest opportunity — at invocation time, before modifying any state — and surface them clearly. A failed operation that leaves the system in the same state it started in is always preferable to a partially completed operation that leaves it in an unknown state.

**Plain language at the boundary.** The boundary between Kanbanzai and the user is the terminal output and MCP response messages. Internal implementation details — Go struct field names, YAML keys, internal error types, file paths inside `.kbz/cache/` — must not cross this boundary. All messages should be readable by a developer who is unfamiliar with the Kanbanzai codebase.

**One message, one action.** Every error message should tell the user exactly one thing to do next. Avoid presenting multiple options unless the situation genuinely requires a choice from the user.

**No silent partial state.** If an operation cannot complete atomically, it must either roll back completely or leave a detectable marker that causes the next invocation to report the problem. The tool must never silently continue as if a previous operation succeeded when it did not.

**Annotations enable trust.** MCP tool annotations are a contract between the tool and its clients. Accurate annotations allow clients to auto-approve safe operations without burdening the user with confirmation prompts. Inaccurate annotations — marking a destructive tool as safe — break client trust and undermine the annotation system entirely.

---

## 5. Error Message Standards

### 5.1 Required structure

Every user-facing error message must answer two questions:

1. **What went wrong?** — A plain-language description of the problem, stated in terms of the user's action and its outcome, not in terms of internal implementation details.
2. **What should the user do next?** — A concrete, actionable instruction. If there is nothing the user can do (e.g. a bug in the tool), the message should say so and direct them to file an issue.

### 5.2 Prohibited content

The following must not appear in any user-facing error message:

- Go type names (e.g. `*yaml.TypeError`, `EntityService`, `StorageBackend`)
- YAML field names as they appear in storage files (e.g. `parent_feature`, `created_at`)
- Stack traces or panic output (these must be caught at the top level and reformatted)
- Internal file paths (e.g. `.kbz/cache/derived.db`, `.kbz/state/features/FEAT-xxx.yaml`)
- Raw OS error strings without context (e.g. `open /path: permission denied` presented without explanation)

### 5.3 Examples

**Acceptable:**
> `Cannot initialise: this directory is not a Git repository. Run 'git init' first, then retry 'kanbanzai init'.`

**Not acceptable:**
> `error: stat .git: no such file or directory`

---

**Acceptable:**
> `Cannot write to '.kbz/': permission denied. Check that the current user has write access to this directory.`

**Not acceptable:**
> `mkdir .kbz: permission denied`

---

**Acceptable:**
> `A '.kbz/' directory already exists in this repository. Run 'kbz status' to inspect the current state, or remove '.kbz/' manually if you want to start over.`

**Not acceptable:**
> `mkdir .kbz: file exists`

### 5.4 MCP error responses

MCP tool error responses follow the same rules. The `message` field in an MCP error response must be a user-readable explanation. Internal error details may be included in a separate `debug` field that is not rendered by default in standard MCP clients, but must never appear in `message`.

---

## 6. `kanbanzai init` Edge Cases

`kanbanzai init` is the first command most users will run. It must handle all common failure modes gracefully, leaving no partial state behind on failure.

### 6.1 Non-Git directory

**Condition:** The current working directory does not contain a `.git/` directory and is not inside a Git repository.

**Expected behaviour:** The command exits with a non-zero status and prints a message explaining that Kanbanzai requires a Git repository. It must suggest running `git init` first. No files or directories are created.

**Rationale:** Kanbanzai's workflow model depends on Git for branch management and commit tracking. Initialising outside a Git repository would produce a broken instance that cannot function.

### 6.2 Existing `.kbz/` directory

**Condition:** A `.kbz/` directory already exists in the repository root.

**Expected behaviour:** The command exits with a non-zero status and prints a message explaining that a Kanbanzai instance already exists. It must tell the user they can run `kbz status` to inspect the existing instance, or explain how to deliberately remove it if they wish to start over. No files are modified.

**Rationale:** Silent reinitialisation would destroy existing workflow state without warning. This must always be an explicit, deliberate user action.

### 6.3 No write permission

**Condition:** The current user does not have write permission on the repository root directory.

**Expected behaviour:** The command exits with a non-zero status and prints a message explaining that it cannot create the `.kbz/` directory due to a permission error, and suggests the user check directory permissions. Write permission must be verified before any files are created, so that no partial state is left behind.

**Rationale:** If the permission check is deferred until after some files have been written, the failure leaves the repository in a partial state. The implementation must check for write permission as the first substantive step.

### 6.4 Conflicting `.skills/` files

**Condition:** A `.skills/` directory already exists and contains one or more files that do not carry the `kanbanzai-managed` marker comment.

**Expected behaviour:** The command exits with a non-zero status and prints a message identifying which files conflict, explaining that Kanbanzai will not overwrite files it did not create. It must provide the user with concrete options: either remove the conflicting files manually before retrying, or pass a flag to skip writing `.skills/` content entirely.

**Rationale:** A user may have pre-existing `.skills/` content from another tool or their own configuration. Silently overwriting this content would cause unrecoverable data loss.

### 6.5 Atomicity requirement

All `init` operations must be atomic with respect to failure: if any step fails after file creation has begun, all created files and directories must be removed before the command exits, leaving the repository in its exact original state. Partial `.kbz/` directories must never be left behind. This is enforced using a write-then-rename pattern for individual files and a deferred cleanup function registered before the first file is written.

---

## 7. CLI Help and Discoverability

### 7.1 Top-level help

Running `kanbanzai --help` (or `kanbanzai -h`) must produce output that:

- Lists every available top-level subcommand with a one-line description
- Groups related commands visually (e.g. "Entity commands", "Server commands", "Utility commands")
- Includes a brief description of what Kanbanzai is and what it does
- Includes usage examples for the most common first-time workflows
- Does not require the user to read `README.md` or any other documentation to understand what the tool does

### 7.2 Per-command help

Running `kanbanzai <command> --help` (or `kanbanzai <command> -h`) must produce output that:

- Names all flags and arguments the command accepts
- Provides a description of what each flag does
- Shows at least one usage example
- Indicates which flags are required versus optional
- Indicates the default value for optional flags where applicable

No command may produce empty or unhelpful output when `--help` is passed. Commands that accept subcommands must list those subcommands in their help output.

### 7.3 Discoverability standard

A developer who has never used Kanbanzai before must be able to:

1. Run `kanbanzai --help` and understand what the tool does in under two minutes.
2. Identify the correct subcommand for a given task without reading documentation.
3. Successfully execute any command using only `--help` output as guidance.

This standard applies to both CLI mode (`kbz`) and to MCP tool descriptions visible in MCP client interfaces such as AI coding assistants.

### 7.4 MCP tool descriptions

Every MCP tool exposed by Kanbanzai must have a `description` field that:

- States what the tool does in one or two sentences
- Names required parameters and their purpose
- Notes any significant side effects (state changes, external API calls, status transitions)

Tool descriptions must be written for an audience that understands software development but may not be familiar with Kanbanzai internals.

---

## 8. Partial State Recovery

### 8.1 Operations that produce detectable partial state

The following operations can leave detectable partial state if interrupted mid-execution:

- `kanbanzai init` — a partial `.kbz/` directory without the `.init-complete` sentinel
- Entity creation — a YAML file written to the state directory but not yet reflected in the cache index
- Document registration — a document record created but with no content hash computed
- Worktree creation — a Git worktree created on disk but no corresponding tracking record in `.kbz/state/worktrees/`
- Merge execution — a pull request merged on GitHub but the entity status not yet updated in `.kbz/`

### 8.2 Detection mechanism

Kanbanzai detects partial state using sentinel files and integrity markers:

- **Init sentinel:** A `.kbz/.init-complete` file is written as the final step of `kanbanzai init`. If `.kbz/` exists but the sentinel is absent, the instance is considered partially initialised. All subsequent commands must check for this condition at startup.
- **Entity integrity:** Entity files are written atomically (written to a temporary file, then renamed into place). A YAML file present in the state directory but absent from the cache index indicates an incomplete write cycle and must be flagged.
- **Worktree integrity:** A worktree tracking record in `.kbz/state/worktrees/` without a corresponding Git worktree directory on disk indicates that the Git operation succeeded but the record write failed, or vice versa.

### 8.3 Reporting behaviour

When partial state is detected on any invocation, the tool must:

1. Print a clear warning naming the type of partial state detected (e.g. "Partial initialisation detected").
2. Identify the affected resource (e.g. the entity ID, the document path, the worktree branch name).
3. Suggest a concrete recovery action (e.g. `kbz init --repair`, `kbz entity delete <id> --force`, or a manual remediation step).
4. Refuse to proceed with the requested operation if doing so would compound the partial state or produce undefined behaviour.

The warning must be printed to stderr so it does not interfere with structured output on stdout.

### 8.4 Scope of detection

Partial state detection runs:

- At startup for every command, checking for the `init` sentinel condition
- At the start of any write operation, for the specific resource being written
- As part of `kbz status` output and the `health_check` MCP tool response

Detection does not run for every read operation, as this would impose unacceptable latency on queries. The `health_check` tool is the appropriate surface for a comprehensive integrity scan.

---

## 9. `doc_record_refresh` MCP Tool

### 9.1 Purpose

When a user edits a document file that has already been registered with `doc_record_submit`, the record's stored content hash becomes stale. This stale hash causes drift warnings on every `doc_record_get` call and blocks operations that require the hash to match (such as `doc_classify`). The current workaround — deleting and re-registering the document — loses all document intelligence data associated with the record.

The `doc_record_refresh` tool updates the stored hash to reflect the current file content without re-registering the document from scratch and without discarding any other metadata.

### 9.2 Inputs

| Parameter | Type   | Required | Description                                                      |
|-----------|--------|----------|------------------------------------------------------------------|
| `id`      | string | Yes      | Document record ID (e.g. `DOC-01KMKRQWF0FCH`)                   |

### 9.3 Behaviour

1. Look up the document record by `id`. If the record does not exist, return an error.
2. Locate the document file at the path stored in the record. If the file does not exist at the recorded path, return an error.
3. Recompute the content hash from the current file content.
4. If the computed hash matches the hash already stored in the record, return a success response indicating that no update was needed and the file has not changed.
5. If the record's current status is `approved`, the refresh transitions the record to `draft` status, because the approved content no longer matches the file on disk. The response must clearly communicate this status transition and explain that re-approval is required.
6. Update the record with the new hash and set the `updated` timestamp to the current time.
7. Return a success response including the old hash, the new hash, the new status, and a `changed` boolean.

### 9.4 Outputs

On success:

| Field      | Type    | Description                                                              |
|------------|---------|--------------------------------------------------------------------------|
| `id`       | string  | The document record ID                                                   |
| `old_hash` | string  | The content hash stored in the record before this call                   |
| `new_hash` | string  | The content hash computed from the current file                          |
| `status`   | string  | The record status after the refresh (`draft` or `approved`)              |
| `changed`  | boolean | `true` if the hash was updated; `false` if the file had not changed      |

On error: a plain-language error message indicating what went wrong and what the user should do.

### 9.5 Status transition on refresh of approved documents

Refreshing an approved document resets it to `draft` because the approval was granted for the content as it existed at approval time. Editing the file after approval invalidates that grant. The user must re-approve the document via `doc_record_approve` after reviewing the changes.

This transition is intentional and not a recoverable error. The response message must make this explicit so the user understands why the status changed. The message must not suggest that this is a problem to work around — it is a deliberate lifecycle enforcement.

### 9.6 Error conditions

| Condition                            | Error message                                                                                                                       |
|--------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------|
| Record ID does not exist             | `No document record found with ID '<id>'. Use doc_record_list to see available records.`                                            |
| File not found at recorded path      | `The file '<path>' no longer exists. If the file was moved, delete this record and re-register the document at its new path.`        |
| File not readable (permission error) | `Cannot read '<path>': permission denied. Check that the current user has read access to this file.`                                 |
| Record is in superseded status       | `Document '<id>' has been superseded and cannot be refreshed. Update the superseding document instead.`                             |

---

## 10. MCP Tool Safety Annotations

### 10.1 Purpose

The MCP protocol supports three tool-level hint annotations: `readOnlyHint`, `destructiveHint`, and `idempotentHint`. These allow MCP clients — such as AI coding assistants and automation pipelines — to make informed decisions about whether to prompt users for confirmation before invoking a tool.

Kanbanzai's MCP tools interact almost exclusively with the local `.kbz/` directory. Tools that only read from or write to `.kbz/` do not affect external systems (remote repositories, GitHub, other users' machines) and are safe to auto-approve without user interaction. Tools that interact with external systems or that delete data require more caution and must not be auto-approved.

### 10.2 Annotation semantics

The annotations are hints, not enforcement mechanisms. Their meaning in the Kanbanzai context is as follows:

| Annotation              | Meaning                                                                                                               |
|-------------------------|-----------------------------------------------------------------------------------------------------------------------|
| `readOnlyHint: true`    | The tool reads data and does not modify any state (`.kbz/`, Git, GitHub, or filesystem).                              |
| `readOnlyHint: false`   | The tool may write to `.kbz/`, Git, or external systems.                                                              |
| `destructiveHint: true` | The tool may delete or irreversibly modify data. Auto-approval is not recommended.                                    |
| `destructiveHint: false`| The tool does not delete or irreversibly modify data. Any state changes can be undone by a subsequent write.          |
| `idempotentHint: true`  | Calling the tool multiple times with the same inputs produces the same result without compounding side effects.        |
| `idempotentHint: false` | Calling the tool multiple times may produce different results or accumulate side effects.                              |

### 10.3 Auto-approvability classification

A Kanbanzai MCP tool is **auto-approvable** (safe for MCP clients to invoke without user confirmation) if all of the following are true:

1. It does not interact with external systems (GitHub API, Git remote, filesystem outside `.kbz/`).
2. It is not destructive (does not delete or irreversibly modify data).
3. Its effects are limited to reading from or writing to `.kbz/`.

Auto-approvable write tools should carry `readOnlyHint: false`, `destructiveHint: false`. Auto-approvable read tools should carry `readOnlyHint: true`, `destructiveHint: false`, `idempotentHint: true`.

### 10.4 Tool classification table

Tools are grouped into three tiers.

**Tier 1 — Read-only** (`readOnlyHint: true`, `destructiveHint: false`, `idempotentHint: true`):

These tools do not modify any state and may be auto-approved freely.

- `get_entity`, `list_entities`, `list_entities_filtered`, `list_by_tag`
- `get_plan`, `list_plans`, `get_prefix_registry`, `get_project_config`
- `doc_record_get`, `doc_record_list`, `doc_record_list_pending`, `doc_record_validate`
- `doc_outline`, `doc_section`, `doc_find_by_concept`, `doc_find_by_entity`, `doc_find_by_role`
- `doc_trace`, `doc_impact`, `doc_supersession_chain`, `doc_extraction_guide`, `doc_pending`
- `doc_gaps`, `health_check`, `dependency_status`, `branch_status`, `pr_status`
- `estimate_query`, `merge_readiness_check`, `conflict_domain_check`
- `knowledge_get`, `knowledge_list`
- `human_checkpoint_get`, `human_checkpoint_list`
- `worktree_get`, `worktree_list`, `cleanup_list`
- `context_assemble`, `profile_get`, `profile_list`

**Tier 2 — Auto-approvable writes** (`readOnlyHint: false`, `destructiveHint: false`):

These tools write only to `.kbz/` and do not interact with external systems. They may be auto-approved.

- `create_feature`, `create_task`, `create_bug`, `create_epic`, `create_plan`
- `update_entity`, `update_status`, `update_plan`, `update_plan_status`
- `record_decision`, `add_prefix`, `retire_prefix`
- `doc_record_submit`, `doc_record_refresh`, `doc_record_approve`
- `doc_classify`
- `estimate_set`, `estimate_reference_add`, `estimate_reference_remove`
- `knowledge_contribute`, `knowledge_confirm`, `knowledge_update`, `knowledge_promote`
- `knowledge_check_staleness`, `knowledge_prune`, `knowledge_compact`
- `knowledge_resolve_conflict`
- `human_checkpoint`, `human_checkpoint_respond`
- `dispatch_task`, `complete_task`, `context_report`
- `incident_create`, `incident_update`, `incident_link_bug`
- `decompose_feature`, `decompose_review`, `slice_analysis`
- `work_queue` (promotes tasks as a write side effect; `idempotentHint: false`)
- `check_duplicates`, `suggest_links`, `validate_candidate`
- `rebuild_cache`

**Tier 3 — Requires confirmation** (`destructiveHint: true`, or interacts with external systems):

These tools interact with Git, GitHub, or the broader filesystem, or perform irreversible operations. MCP clients must not auto-approve these.

- `worktree_create`, `worktree_remove` — create or remove Git worktrees on disk
- `merge_execute`, `merge_pull_request` — merge branches on Git and GitHub
- `pr_create`, `pr_update`, `pr_update` — create or update pull requests on GitHub
- `cleanup_execute` — deletes worktree directories and local/remote branches
- `knowledge_retire`, `knowledge_flag` — may irreversibly affect knowledge entry lifecycle
- `doc_record_supersede` — marks a document as superseded (not easily reversible)
- `batch_import_documents` — bulk-imports documents; idempotent but operates on the filesystem
- `review_task_output` — triggers state transitions on tasks

### 10.5 Implementation requirement

Annotations must be declared statically at tool registration time, not computed dynamically. Each tool definition in the MCP server must include all three annotation fields with explicit boolean values. Any tool that does not yet have annotations assigned must default to `destructiveHint: true`, `readOnlyHint: false`, `idempotentHint: false` until it is classified, ensuring safe defaults for unclassified tools.

### 10.6 Annotation accuracy requirement

Annotations are a contract. A tool annotated as non-destructive must never delete or irreversibly modify data under any code path. A tool annotated as read-only must never write to `.kbz/`, Git, or any external system. If a tool's behaviour changes in a way that affects its annotation, the annotation must be updated in the same commit as the behaviour change. Annotation correctness is a verification criterion for any change to MCP tool behaviour.

---

## 11. Acceptance Criteria

1. **AC-01 Error message structure.** Every error message produced by the CLI or MCP server includes (a) a plain-language description of what went wrong and (b) a concrete action the user should take. No error message contains a Go type name, a raw YAML field name from internal storage, or a stack trace.

2. **AC-02 Clean-machine init.** On a machine with no pre-existing `.kbz/` directory, running `kanbanzai init` in a valid Git repository completes successfully, creates a valid `.kbz/` structure including the `.init-complete` sentinel file, and produces output confirming what was created.

3. **AC-03 Init in non-Git directory.** Running `kanbanzai init` in a directory that is not a Git repository exits with a non-zero status, prints a message directing the user to run `git init` first, and leaves no files or directories behind.

4. **AC-04 Init with existing `.kbz/`.** Running `kanbanzai init` in a repository where `.kbz/` already exists exits with a non-zero status, prints a message explaining that an instance already exists and how to inspect or remove it, and does not modify any existing files.

5. **AC-05 Init with no write permission.** Running `kanbanzai init` in a directory where the current user lacks write permission exits with a non-zero status, prints a message describing the permission problem, and leaves no partial `.kbz/` directory behind.

6. **AC-06 Init with conflicting `.skills/` files.** Running `kanbanzai init` when `.skills/` contains files without the `kanbanzai-managed` marker exits with a non-zero status, names the conflicting files in its output, and does not overwrite or modify them.

7. **AC-07 Init atomicity.** If `kanbanzai init` fails for any reason after beginning to create files, all created files and directories are cleaned up before the process exits. No partial `.kbz/` directory is left behind in any failure scenario.

8. **AC-08 Partial initialisation detection.** If `.kbz/` exists but `.kbz/.init-complete` is absent (indicating a previously interrupted init), the next invocation of any Kanbanzai command detects this condition, prints a warning identifying it as partial initialisation, and suggests a concrete recovery action rather than proceeding silently.

9. **AC-09 CLI top-level help.** Running `kanbanzai --help` lists every available subcommand with a one-line description, grouped by category. A developer unfamiliar with the tool can identify the correct command for any common task using only this output.

10. **AC-10 CLI per-command help.** Running `kanbanzai <command> --help` for every command produces output listing all flags, their types, descriptions, and defaults, plus at least one usage example. No command produces empty output or exits with a non-zero status when `--help` is passed.

11. **AC-11 `doc_record_refresh` updates hash.** Calling `doc_record_refresh` on a document record after editing the associated file updates the stored content hash to match the current file content and returns both the old hash and the new hash in the response.

12. **AC-12 `doc_record_refresh` on approved document.** Calling `doc_record_refresh` on an approved document whose file content has changed transitions the record status from `approved` to `draft` and clearly states this transition and its reason in the response. The status does not remain `approved`.

13. **AC-13 `doc_record_refresh` with unchanged file.** Calling `doc_record_refresh` on a document whose file has not changed since last registration returns a success response with `changed: false` and does not modify the record's status or timestamp.

14. **AC-14 MCP annotation coverage.** Every Kanbanzai MCP tool has explicit values for `readOnlyHint`, `destructiveHint`, and `idempotentHint` defined at tool registration time. No tool is missing any of the three annotations.

15. **AC-15 MCP annotation accuracy.** No tool annotated with `destructiveHint: false` deletes or irreversibly modifies any data. No tool annotated with `readOnlyHint: true` writes to `.kbz/`, interacts with Git, or makes any network call. This is verified by code review and by test coverage of each annotated tool's side effects.