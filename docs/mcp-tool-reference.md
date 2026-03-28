# MCP Tool Reference

Complete reference for every MCP tool exposed by the Kanbanzai server. This document covers transport details, tool organisation, parameter definitions, return values, error conditions, and example calls for all 97 tools.

**Audience:** Agent developers and tool builders.

---

## Table of Contents

1. [Transport and Protocol](#1-transport-and-protocol)
2. [Tool Organisation by Domain](#2-tool-organisation-by-domain)
3. [Entity Management](#3-entity-management)
4. [Plan Management](#4-plan-management)
5. [Configuration](#5-configuration)
6. [Document Records](#6-document-records)
7. [Document Intelligence](#7-document-intelligence)
8. [Knowledge Management](#8-knowledge-management)
9. [Context and Profiles](#9-context-and-profiles)
10. [Work Queue and Dispatch](#10-work-queue-and-dispatch)
11. [Human Checkpoints](#11-human-checkpoints)
12. [Estimation](#12-estimation)
13. [Feature Decomposition and Review](#13-feature-decomposition-and-review)
14. [Conflict Analysis](#14-conflict-analysis)
15. [Incident Management](#15-incident-management)
16. [Git Integration — Worktrees](#16-git-integration--worktrees)
17. [Git Integration — Branches and Cleanup](#17-git-integration--branches-and-cleanup)
18. [Git Integration — Merge](#18-git-integration--merge)
19. [Git Integration — Pull Requests](#19-git-integration--pull-requests)
20. [Agent Capabilities](#20-agent-capabilities)
21. [Batch Operations](#21-batch-operations)
22. [Migration](#22-migration)
23. [Rich Queries](#23-rich-queries)
24. [Retrospective Synthesis](#24-retrospective-synthesis)
25. [Lifecycle Operation Constraints](#25-lifecycle-operation-constraints)
26. [Idempotency Notes](#26-idempotency-notes)

---

## 1. Transport and Protocol

Kanbanzai's MCP server communicates over **stdio** (standard input/output) using the Model Context Protocol. All tool calls use JSON-RPC 2.0 over the MCP protocol.

**Starting the server:**

```
kanbanzai serve
```

The server is **editor-agnostic** — it works with any MCP-compatible client including Zed, Claude Desktop, VS Code, Cursor, and others.

| Property | Value |
|---|---|
| Server name | `kanbanzai` |
| Server version | Matches binary version |
| Transport | stdio |
| Wire format | JSON-RPC 2.0 (MCP) |

Clients send `tools/call` requests with a tool name and a JSON arguments object. The server returns either a text result (JSON-encoded) or a structured error.

---

## 2. Tool Organisation by Domain

Kanbanzai exposes 97 tools across 16 domains:

| Domain | Count | Tools |
|---|---|---|
| Entity Management | 15 | create_epic, create_feature, create_task, create_bug, record_decision, get_entity, list_entities, list_entities_filtered, list_by_tag, list_tags, update_status, update_entity, validate_candidate, health_check, rebuild_cache |
| Plan Management | 5 | create_plan, get_plan, list_plans, update_plan, update_plan_status |
| Configuration | 4 | get_project_config, get_prefix_registry, add_prefix, retire_prefix |
| Document Records | 11 | doc_record_submit, doc_record_get, doc_record_get_content, doc_record_list, doc_record_list_pending, doc_record_approve, doc_record_supersede, doc_record_validate, doc_gaps, doc_trace, doc_supersession_chain |
| Document Intelligence | 9 | doc_outline, doc_section, doc_classify, doc_pending, doc_find_by_concept, doc_find_by_entity, doc_find_by_role, doc_impact, doc_extraction_guide |
| Knowledge Management | 13 | knowledge_contribute, knowledge_get, knowledge_list, knowledge_update, knowledge_confirm, knowledge_flag, knowledge_retire, knowledge_promote, knowledge_compact, knowledge_prune, knowledge_check_staleness, knowledge_resolve_conflict, context_report |
| Context and Profiles | 3 | context_assemble, profile_get, profile_list |
| Work Queue and Dispatch | 4 | work_queue, dependency_status, dispatch_task, complete_task |
| Human Checkpoints | 4 | human_checkpoint, human_checkpoint_get, human_checkpoint_list, human_checkpoint_respond |
| Estimation | 4 | estimate_set, estimate_query, estimate_reference_add, estimate_reference_remove |
| Feature Decomposition | 4 | decompose_feature, decompose_review, slice_analysis, review_task_output |
| Conflict Analysis | 1 | conflict_domain_check |
| Incident Management | 4 | incident_create, incident_update, incident_list, incident_link_bug |
| Git Integration | 12 | worktree_create, worktree_get, worktree_list, worktree_remove, branch_status, cleanup_list, cleanup_execute, merge_readiness_check, merge_execute, pr_create, pr_update, pr_status |
| Agent Capabilities | 3 | suggest_links, check_duplicates, doc_extraction_guide |
| Batch Operations | 1 | batch_import_documents |
| Migration | 1 | migrate_phase2 |
| Rich Queries | 1 | query_plan_tasks |

---

## 3. Entity Management

### create_epic

Create a new epic entity.

| Parameter | Type | Required | Description |
|---|---|---|---|
| slug | string | yes | URL-friendly identifier for the epic |
| title | string | yes | Title of the epic |
| summary | string | yes | Brief summary of the epic |
| created_by | string | no | Who created the epic. Auto-resolved from `.kbz/local.yaml` or git config if not provided |

**Returns:** JSON object with `Type`, `ID`, `DisplayID`, `Slug`, `Path`, and `State`.

**Error conditions:**
- Missing required parameters
- Identity resolution fails (no `.kbz/local.yaml`, no git config)
- Slug collision with existing epic

**Example:**

```json
// Call
{"tool": "create_epic", "arguments": {"slug": "user-auth", "title": "User Authentication", "summary": "Implement core authentication flows"}}
// Response
{"Type": "epic", "ID": "EPIC-01JX...", "DisplayID": "EPIC-01JX...", "Slug": "user-auth", "Path": ".kbz/epics/EPIC-01JX.../user-auth.yaml", "State": {"status": "proposed", "title": "User Authentication", "summary": "Implement core authentication flows"}}
```

---

### create_feature

Create a new feature entity.

| Parameter | Type | Required | Description |
|---|---|---|---|
| slug | string | yes | URL-friendly identifier for the feature |
| parent | string | yes | Parent plan ID (also accepts `epic` for backward compat) |
| summary | string | yes | Brief summary of the feature |
| created_by | string | no | Who created the feature. Auto-resolved from `.kbz/local.yaml` or git config |
| design | string | no | Design document reference |
| tags | array | no | Tags for cross-cutting organisation |

**Returns:** JSON object with `Type`, `ID`, `DisplayID`, `Slug`, `Path`, and `State`.

**Error conditions:**
- Missing required parameters
- Parent plan/epic not found
- Identity resolution fails

**Example:**

```json
// Call
{"tool": "create_feature", "arguments": {"slug": "login-form", "parent": "P1-user-auth", "summary": "Login form with email/password", "tags": ["frontend", "auth"]}}
// Response
{"Type": "feature", "ID": "FEAT-01JX...", "DisplayID": "FEAT-01JX...", "Slug": "login-form", "Path": ".kbz/features/FEAT-01JX.../login-form.yaml", "State": {"status": "proposed", "summary": "Login form with email/password", "parent": "P1-user-auth"}}
```

---

### create_task

Create a new task entity.

| Parameter | Type | Required | Description |
|---|---|---|---|
| parent_feature | string | yes | Parent feature ID |
| slug | string | yes | URL-friendly identifier for the task |
| summary | string | yes | Brief summary of the task |

**Returns:** JSON object with `Type`, `ID`, `DisplayID`, `Slug`, `Path`, and `State`.

**Error conditions:**
- Missing required parameters
- Parent feature not found

**Example:**

```json
// Call
{"tool": "create_task", "arguments": {"parent_feature": "FEAT-01JX...", "slug": "email-validation", "summary": "Implement email format validation on the login form"}}
// Response
{"Type": "task", "ID": "T-01JX...", "DisplayID": "T-01JX...", "Slug": "email-validation", "Path": ".kbz/tasks/T-01JX.../email-validation.yaml", "State": {"status": "queued", "summary": "Implement email format validation on the login form", "parent_feature": "FEAT-01JX..."}}
```

---

### create_bug

Create a new bug entity.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| slug | string | yes | URL-friendly identifier | |
| title | string | yes | Title of the bug | |
| reported_by | string | yes | Who reported the bug | |
| observed | string | yes | Observed behavior | |
| expected | string | yes | Expected behavior | |
| severity | string | yes | Bug severity level | `low`, `medium`, `high`, `critical` |
| priority | string | yes | Bug priority level | `low`, `medium`, `high`, `critical` |
| type | string | yes | Bug type classification | `implementation-defect`, `specification-defect`, `design-problem` |

**Returns:** JSON object with `Type`, `ID`, `DisplayID`, `Slug`, `Path`, and `State`.

**Error conditions:**
- Missing required parameters
- Invalid severity, priority, or type values

**Example:**

```json
// Call
{"tool": "create_bug", "arguments": {"slug": "login-crash", "title": "Login crashes on empty password", "reported_by": "sam", "observed": "App crashes with NPE when password field is empty", "expected": "Should show validation error message", "severity": "high", "priority": "high", "type": "implementation-defect"}}
// Response
{"Type": "bug", "ID": "BUG-01JX...", "DisplayID": "BUG-01JX...", "Slug": "login-crash", "Path": ".kbz/bugs/BUG-01JX.../login-crash.yaml", "State": {"status": "triaged", "title": "Login crashes on empty password", "severity": "high", "priority": "high"}}
```

---

### record_decision

Record a new decision entity.

| Parameter | Type | Required | Description |
|---|---|---|---|
| slug | string | yes | URL-friendly identifier for the decision |
| summary | string | yes | Brief summary of the decision |
| rationale | string | yes | Rationale behind the decision |
| decided_by | string | no | Who made the decision. Auto-resolved from `.kbz/local.yaml` or git config |

**Returns:** JSON object with `Type`, `ID`, `DisplayID`, `Slug`, `Path`, and `State`.

**Error conditions:**
- Missing required parameters
- Identity resolution fails

**Example:**

```json
// Call
{"tool": "record_decision", "arguments": {"slug": "use-jwt", "summary": "Use JWT for API authentication", "rationale": "JWTs are stateless and work well with our microservice architecture"}}
// Response
{"Type": "decision", "ID": "DEC-01JX...", "DisplayID": "DEC-01JX...", "Slug": "use-jwt", "Path": ".kbz/decisions/DEC-01JX.../use-jwt.yaml", "State": {"summary": "Use JWT for API authentication", "rationale": "JWTs are stateless..."}}
```

---

### get_entity

Get a specific entity by type, ID, and slug.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| entity_type | string | yes | Type of entity to retrieve | `epic`, `feature`, `task`, `bug`, `decision` |
| id | string | yes | Entity ID or unambiguous prefix | |
| slug | string | no | Entity slug (resolved from ID prefix if omitted) | |

**Returns:** JSON object with full entity state including all fields.

**Error conditions:**
- Entity not found
- Ambiguous ID prefix (matches multiple entities)
- Invalid entity type

**Example:**

```json
// Call
{"tool": "get_entity", "arguments": {"entity_type": "feature", "id": "FEAT-01JX..."}}
// Response
{"Type": "feature", "ID": "FEAT-01JX...", "DisplayID": "FEAT-01JX...", "Slug": "login-form", "Path": "...", "State": {"status": "proposed", "summary": "Login form with email/password", "parent": "P1-user-auth"}}
```

---

### list_entities

List all entities of a given type.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| entity_type | string | yes | Type of entities to list | `epic`, `feature`, `task`, `bug`, `decision` |

**Returns:** JSON array of entity summary objects, each with `Type`, `ID`, `DisplayID`, `Slug`, `Path`, and `State`.

**Error conditions:**
- Invalid entity type
- Entity directory does not exist (returns empty list, not an error)

**Example:**

```json
// Call
{"tool": "list_entities", "arguments": {"entity_type": "feature"}}
// Response
[{"Type": "feature", "ID": "FEAT-01JX...", "DisplayID": "FEAT-01JX...", "Slug": "login-form", "Path": "...", "State": "proposed"}, ...]
```

---

### list_entities_filtered

List entities of a given type with optional filters for status, tags, parent, and date ranges.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| entity_type | string | yes | Type of entities to list | `epic`, `feature`, `task`, `bug`, `decision`, `plan` |
| status | string | no | Filter by lifecycle status | |
| tags | array | no | Filter by tags (must have at least one) | |
| parent | string | no | Filter by parent entity ID (for features) | |
| created_after | string | no | RFC3339 timestamp filter | e.g. `2024-01-01T00:00:00Z` |
| created_before | string | no | RFC3339 timestamp filter | |
| updated_after | string | no | RFC3339 timestamp filter | |
| updated_before | string | no | RFC3339 timestamp filter | |

**Returns:** JSON object with `success`, `type`, `count`, and `results` array.

**Error conditions:**
- Invalid entity type
- Invalid date format (must be RFC3339)

**Example:**

```json
// Call
{"tool": "list_entities_filtered", "arguments": {"entity_type": "task", "status": "active", "parent": "FEAT-01JX..."}}
// Response
{"success": true, "type": "task", "count": 3, "results": [...]}
```

---

### list_by_tag

List all entities across all types that have the given tag.

| Parameter | Type | Required | Description |
|---|---|---|---|
| tag | string | yes | Tag to search for |

**Returns:** JSON object with `success`, `tag`, `count`, and `entities` array.

**Error conditions:**
- Missing tag parameter

**Example:**

```json
// Call
{"tool": "list_by_tag", "arguments": {"tag": "frontend"}}
// Response
{"success": true, "tag": "frontend", "count": 5, "entities": [...]}
```

---

### list_tags

List all unique tags across all entity types, sorted alphabetically.

| Parameter | Type | Required | Description |
|---|---|---|---|
| *(none)* | | | |

**Returns:** JSON object with `success`, `count`, and `tags` array.

**Error conditions:**
- Filesystem errors reading entity directories

**Example:**

```json
// Call
{"tool": "list_tags", "arguments": {}}
// Response
{"success": true, "count": 8, "tags": ["auth", "backend", "frontend", "phase:1", "priority:high"]}
```

---

### update_status

Update the lifecycle status of an entity. Enforces lifecycle state machine rules.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| entity_type | string | yes | Type of entity to update | `epic`, `feature`, `task`, `bug`, `decision` |
| id | string | yes | Entity ID or unambiguous prefix | |
| slug | string | no | Entity slug (resolved from ID prefix if omitted) | |
| status | string | yes | New lifecycle status | varies by entity type |

**Returns:** JSON object with updated entity state. May include `worktree_created` or `worktree_exists` fields if transitioning a feature to `in-progress` triggers automatic worktree creation.

**Error conditions:**
- Entity not found
- Invalid status transition (enforces lifecycle state machine)
- Entity already in requested status

**Example:**

```json
// Call
{"tool": "update_status", "arguments": {"entity_type": "task", "id": "T-01JX...", "status": "active"}}
// Response
{"Type": "task", "ID": "T-01JX...", "State": {"status": "active", ...}}
```

---

### update_entity

Update fields of an existing entity. Cannot change `id` or `status` (use `update_status` for status changes).

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| entity_type | string | yes | Type of entity to update | `epic`, `feature`, `task`, `bug`, `decision` |
| id | string | yes | Entity ID or unambiguous prefix | |
| slug | string | no | Entity slug | |
| *(additional)* | string | no | Any other string field is passed through as a field update | |

**Returns:** JSON object with updated entity state.

**Error conditions:**
- Entity not found
- Attempting to change `id` or `status` fields

**Example:**

```json
// Call
{"tool": "update_entity", "arguments": {"entity_type": "feature", "id": "FEAT-01JX...", "summary": "Updated summary for the login form feature"}}
// Response
{"Type": "feature", "ID": "FEAT-01JX...", "State": {"summary": "Updated summary for the login form feature", ...}}
```

---

### validate_candidate

Validate candidate entity data without persisting it. Useful for pre-flight checks before creation.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| entity_type | string | yes | Type of entity to validate | `epic`, `feature`, `task`, `bug`, `decision` |
| *(additional)* | any | no | All other arguments are treated as candidate entity fields | |

**Returns:** Validation result — either empty (valid) or a list of validation errors.

**Error conditions:**
- Invalid entity type

**Example:**

```json
// Call
{"tool": "validate_candidate", "arguments": {"entity_type": "bug", "title": "Missing title crash", "severity": "invalid"}}
// Response
[{"field": "severity", "message": "invalid value: must be one of low, medium, high, critical"}]
```

---

### health_check

Run a comprehensive health check across all entities, knowledge entries, worktrees, branches, and context profiles.

| Parameter | Type | Required | Description |
|---|---|---|---|
| *(none)* | | | |

**Returns:** JSON health report with `Summary` (counts by type, error count, warning count), `Errors` array, and `Warnings` array.

**Error conditions:**
- Filesystem errors reading entity directories

**Example:**

```json
// Call
{"tool": "health_check", "arguments": {}}
// Response
{"Summary": {"EntitiesByType": {"feature": 12, "task": 34, "bug": 3}, "ErrorCount": 0, "WarningCount": 2}, "Errors": [], "Warnings": [{"EntityType": "branch", "Message": "branch feat/login-form is 15 commits behind main"}]}
```

---

### rebuild_cache

Rebuild the local derived cache from canonical entity files. The cache accelerates queries but is not required for correctness.

| Parameter | Type | Required | Description |
|---|---|---|---|
| *(none)* | | | |

**Returns:** JSON object with `status` and `entities_cached` count.

**Error conditions:**
- Filesystem errors

**Example:**

```json
// Call
{"tool": "rebuild_cache", "arguments": {}}
// Response
{"status": "ok", "entities_cached": 47}
```

---

## 4. Plan Management

### create_plan

Create a new Plan entity. Plans coordinate bodies of work and organise Features. The prefix must be declared in `.kbz/config.yaml`.

| Parameter | Type | Required | Description |
|---|---|---|---|
| prefix | string | yes | Single-character prefix for the Plan ID (must be in prefix registry) |
| slug | string | yes | URL-friendly identifier (appended after prefix and number) |
| title | string | yes | Human-readable title of the Plan |
| summary | string | yes | Brief description of the Plan's purpose and scope |
| created_by | string | no | Who created the Plan. Auto-resolved from `.kbz/local.yaml` or git config |
| tags | array | no | Freeform tags for organisation (e.g., `phase:2`, `priority:high`) |

**Returns:** JSON object with `Type`, `ID`, `DisplayID`, `Slug`, `Path`, and `State`.

**Error conditions:**
- Prefix not declared in config
- Prefix is retired
- Missing required parameters

**Example:**

```json
// Call
{"tool": "create_plan", "arguments": {"prefix": "P", "slug": "user-auth", "title": "User Authentication", "summary": "Implement all authentication flows"}}
// Response
{"Type": "plan", "ID": "P1-user-auth", "DisplayID": "P1-user-auth", "Slug": "user-auth", "Path": ".kbz/plans/P1-user-auth/plan.yaml", "State": {"status": "proposed", "title": "User Authentication"}}
```

---

### get_plan

Get a Plan by its ID.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Plan ID (e.g., `P1-basic-ui`) |

**Returns:** JSON object with `Type`, `ID`, `DisplayID`, `Slug`, `Path`, and `State`.

**Error conditions:**
- Plan not found

**Example:**

```json
// Call
{"tool": "get_plan", "arguments": {"id": "P1-user-auth"}}
// Response
{"Type": "plan", "ID": "P1-user-auth", "Slug": "user-auth", "State": {"status": "active", "title": "User Authentication"}}
```

---

### list_plans

List all Plans with optional filtering.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| status | string | no | Filter by status | `proposed`, `designing`, `active`, `done`, `superseded`, `cancelled` |
| prefix | string | no | Filter by Plan prefix (single character) | |
| tags | array | no | Filter by tags (Plans must have all specified tags) | |

**Returns:** JSON array of Plan summary objects.

**Error conditions:**
- Filesystem errors

**Example:**

```json
// Call
{"tool": "list_plans", "arguments": {"status": "active"}}
// Response
[{"Type": "plan", "ID": "P1-user-auth", "Slug": "user-auth", "State": "active"}, ...]
```

---

### update_plan

Update mutable fields on a Plan (title, summary, design reference, tags).

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Plan ID |
| slug | string | yes | Plan slug |
| title | string | no | New title |
| summary | string | no | New summary |
| design | string | no | Reference to design document record (empty string to clear) |
| tags | array | no | New tags (replaces existing tags) |

**Returns:** JSON object with updated Plan state.

**Error conditions:**
- Plan not found
- ID/slug mismatch

**Example:**

```json
// Call
{"tool": "update_plan", "arguments": {"id": "P1-user-auth", "slug": "user-auth", "title": "User Authentication v2"}}
// Response
{"Type": "plan", "ID": "P1-user-auth", "State": {"title": "User Authentication v2", ...}}
```

---

### update_plan_status

Transition a Plan to a new lifecycle status.

Valid transitions: `proposed` → `designing`, `designing` → `active`, `active` → `done`. Any non-terminal state can transition to `superseded` or `cancelled`.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| id | string | yes | Plan ID | |
| slug | string | yes | Plan slug | |
| status | string | yes | New status | `proposed`, `designing`, `active`, `done`, `superseded`, `cancelled` |

**Returns:** JSON object with updated Plan state.

**Error conditions:**
- Plan not found
- Invalid status transition

**Example:**

```json
// Call
{"tool": "update_plan_status", "arguments": {"id": "P1-user-auth", "slug": "user-auth", "status": "active"}}
// Response
{"Type": "plan", "ID": "P1-user-auth", "State": {"status": "active", ...}}
```

---

## 5. Configuration

### get_project_config

Get the project configuration including the prefix registry, version, and other settings. Returns the contents of `.kbz/config.yaml`.

| Parameter | Type | Required | Description |
|---|---|---|---|
| *(none)* | | | |

**Returns:** JSON object with `success`, optionally `using_defaults`, and `config` containing `version` and `prefixes`.

**Error conditions:**
- If no config file exists, returns defaults with `using_defaults: true`

**Example:**

```json
// Call
{"tool": "get_project_config", "arguments": {}}
// Response
{"success": true, "config": {"version": "1.0", "prefixes": [{"prefix": "P", "name": "Plan"}]}}
```

---

### get_prefix_registry

Get the Plan ID prefix registry. Lists all declared prefixes with their labels and retired status.

| Parameter | Type | Required | Description |
|---|---|---|---|
| *(none)* | | | |

**Returns:** JSON object with `success`, `active_prefixes`, `retired_prefixes`, `total_count`, and `active_count`.

**Error conditions:**
- Returns defaults if no config file exists

**Example:**

```json
// Call
{"tool": "get_prefix_registry", "arguments": {}}
// Response
{"success": true, "active_prefixes": [{"prefix": "P", "name": "Plan"}], "retired_prefixes": [], "total_count": 1, "active_count": 1}
```

---

### add_prefix

Add a new prefix to the Plan ID prefix registry.

| Parameter | Type | Required | Description |
|---|---|---|---|
| prefix | string | yes | Single non-digit Unicode character |
| name | string | yes | Human-readable name for the prefix |
| description | string | no | Longer description of the prefix purpose |

**Returns:** JSON object with `success`, `message`, and `prefix` details.

**Error conditions:**
- Invalid prefix format (must be single non-digit character)
- Prefix already exists
- Config save fails

**Example:**

```json
// Call
{"tool": "add_prefix", "arguments": {"prefix": "S", "name": "Sprint", "description": "Short-lived sprint plans"}}
// Response
{"success": true, "message": "Prefix added successfully", "prefix": {"prefix": "S", "name": "Sprint", "description": "Short-lived sprint plans", "retired": false}}
```

---

### retire_prefix

Mark a prefix as retired. Retired prefixes cannot be used for new Plans but remain valid for existing Plans. At least one active prefix must remain.

| Parameter | Type | Required | Description |
|---|---|---|---|
| prefix | string | yes | The prefix to retire |

**Returns:** JSON object with `success`, `message`, and `prefix`.

**Error conditions:**
- Prefix not found
- Cannot retire the last active prefix
- Config load/save fails

**Example:**

```json
// Call
{"tool": "retire_prefix", "arguments": {"prefix": "S"}}
// Response
{"success": true, "message": "Prefix retired successfully", "prefix": "S"}
```

---

## 6. Document Records

### doc_record_submit

Register a document with the system, creating a document record in draft status. Computes the content hash and prepares the document for Layer 1-2 analysis. The document file must already exist at the specified path.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| path | string | yes | Relative path to the document file from the repo root | |
| type | string | yes | Document type | `design`, `specification`, `dev-plan`, `research`, `report`, `policy` |
| title | string | yes | Human-readable title | |
| owner | string | no | Parent Plan or Feature ID that owns this document | |
| created_by | string | no | Who is submitting the document. Auto-resolved | |

**Returns:** JSON object with `success`, `message`, and `document` record details including `id`, `path`, `record_path`, `type`, `title`, `status`, `owner`, `content_hash`, `created`, `updated`.

**Error conditions:**
- File does not exist at path
- Invalid document type
- Identity resolution fails

**Example:**

```json
// Call
{"tool": "doc_record_submit", "arguments": {"path": "docs/design/auth-flow.md", "type": "design", "title": "Authentication Flow Design", "owner": "FEAT-01JX..."}}
// Response
{"success": true, "message": "Document submitted successfully", "document": {"id": "DOC-01JX...", "path": "docs/design/auth-flow.md", "type": "design", "title": "Authentication Flow Design", "status": "draft", "owner": "FEAT-01JX...", "content_hash": "sha256:abc123..."}}
```

---

### doc_record_approve

Transition a document from draft to approved status. Approval triggers lifecycle transitions on the owning entity (e.g., approving a design doc can transition a feature from `proposed` to `designing`).

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Document record ID |
| approved_by | string | no | Who is approving. Auto-resolved |

**Returns:** JSON object with updated document record and `message`.

**Error conditions:**
- Document not found
- Document is not in draft status
- Content hash mismatch (file changed since submission)

**Example:**

```json
// Call
{"tool": "doc_record_approve", "arguments": {"id": "DOC-01JX..."}}
// Response
{"success": true, "message": "Document approved successfully", "document": {"id": "DOC-01JX...", "status": "approved", "approved_by": "sam", ...}}
```

---

### doc_record_supersede

Transition a document from approved to superseded status, linking to the newer replacement document. Supersession may trigger backward lifecycle transitions on the owning entity.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Document record ID being superseded |
| superseded_by | string | yes | Document record ID of the replacement |

**Returns:** JSON object with updated document record.

**Error conditions:**
- Document not found
- Document is not in approved status
- Superseding document not found

**Example:**

```json
// Call
{"tool": "doc_record_supersede", "arguments": {"id": "DOC-01JXaaa...", "superseded_by": "DOC-01JXbbb..."}}
// Response
{"success": true, "message": "Document superseded successfully", "document": {"id": "DOC-01JXaaa...", "status": "superseded", "superseded_by": "DOC-01JXbbb..."}}
```

---

### doc_record_get

Get a document record by ID. Returns metadata including status, owner, content hash, and drift detection.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Document record ID |
| check_drift | boolean | no | Whether to check if content has changed since recorded (default: true) |

**Returns:** JSON object with document details. Includes `warning` and `drift: true` fields if content has changed.

**Error conditions:**
- Document not found

**Example:**

```json
// Call
{"tool": "doc_record_get", "arguments": {"id": "DOC-01JX..."}}
// Response
{"success": true, "document": {"id": "DOC-01JX...", "path": "docs/design/auth-flow.md", "status": "approved", "content_hash": "sha256:abc123...", ...}}
```

---

### doc_record_get_content

Get the content of a document file. Includes drift detection warning if content has changed since the record was created.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Document record ID |

**Returns:** JSON object with `document` metadata, `content` (full file text), and optional `warning`/`drift` fields.

**Error conditions:**
- Document not found
- File no longer exists at recorded path

**Example:**

```json
// Call
{"tool": "doc_record_get_content", "arguments": {"id": "DOC-01JX..."}}
// Response
{"success": true, "document": {"id": "DOC-01JX...", "title": "Auth Design"}, "content": "# Authentication Flow\n\n..."}
```

---

### doc_record_list

List all document records with optional filtering.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| type | string | no | Filter by document type | `design`, `specification`, `dev-plan`, `research`, `report`, `policy` |
| status | string | no | Filter by status | `draft`, `approved`, `superseded` |
| owner | string | no | Filter by owner entity ID | |

**Returns:** JSON object with `success`, `count`, and `documents` array.

**Error conditions:**
- Filesystem errors

**Example:**

```json
// Call
{"tool": "doc_record_list", "arguments": {"status": "draft"}}
// Response
{"success": true, "count": 3, "documents": [{"id": "DOC-01JX...", "title": "...", "status": "draft"}, ...]}
```

---

### doc_record_list_pending

List all documents in draft status that are awaiting approval or classification.

| Parameter | Type | Required | Description |
|---|---|---|---|
| *(none)* | | | |

**Returns:** JSON object with `success`, `count`, and `documents` array.

**Error conditions:**
- Filesystem errors

**Example:**

```json
// Call
{"tool": "doc_record_list_pending", "arguments": {}}
// Response
{"success": true, "count": 2, "documents": [{"id": "DOC-01JX...", "status": "draft", "title": "..."}, ...]}
```

---

### doc_record_validate

Validate a document record and check content integrity.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Document record ID |

**Returns:** JSON object with `success`, `document_id`, `valid` (boolean), `issues` array, and `message`.

**Error conditions:**
- Document not found

**Example:**

```json
// Call
{"tool": "doc_record_validate", "arguments": {"id": "DOC-01JX..."}}
// Response
{"success": true, "document_id": "DOC-01JX...", "valid": true, "issues": [], "message": "Document is valid"}
```

---

### doc_gaps

Analyze what document types are missing for a feature. Checks whether design, specification, and dev-plan documents exist.

| Parameter | Type | Required | Description |
|---|---|---|---|
| feature_id | string | yes | Feature ID to analyze |

**Returns:** JSON object with `success`, `feature_id`, `complete` (boolean), `gaps` array, and `message`.

**Error conditions:**
- Feature not found

**Example:**

```json
// Call
{"tool": "doc_gaps", "arguments": {"feature_id": "FEAT-01JX..."}}
// Response
{"success": true, "feature_id": "FEAT-01JX...", "complete": false, "gaps": ["specification", "dev-plan"], "message": "Missing document types: specification, dev-plan"}
```

---

### doc_trace

Trace an entity through the document refinement chain. Returns all document sections that reference the entity, ordered by document type (design → specification → dev-plan).

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID to trace |

**Returns:** JSON object with `success`, `entity_id`, `count`, and `matches` array.

**Error conditions:**
- Entity ID not found in any document

**Example:**

```json
// Call
{"tool": "doc_trace", "arguments": {"entity_id": "FEAT-01JX..."}}
// Response
{"success": true, "entity_id": "FEAT-01JX...", "count": 4, "matches": [{"document_id": "DOC-...", "section_path": "1.2", "document_type": "design"}, ...]}
```

---

### doc_supersession_chain

Follow supersedes/superseded_by links to build the full version chain for a document. Returns documents ordered from oldest to newest.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Document record ID to start from |

**Returns:** JSON object with `success`, `start_id`, `chain_length`, and `chain` array.

**Error conditions:**
- Document not found

**Example:**

```json
// Call
{"tool": "doc_supersession_chain", "arguments": {"id": "DOC-01JX..."}}
// Response
{"success": true, "start_id": "DOC-01JX...", "chain_length": 3, "chain": [{"id": "DOC-01JXaaa...", "status": "superseded", "superseded_by": "DOC-01JXbbb..."}, {"id": "DOC-01JXbbb...", "status": "superseded", "superseded_by": "DOC-01JXccc..."}, {"id": "DOC-01JXccc...", "status": "approved"}]}
```

---

## 7. Document Intelligence

### doc_outline

Get the structural outline (Layer 1) of an indexed document. Returns the section tree with paths, titles, levels, word counts, and content hashes.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Document ID |

**Returns:** JSON object with `success`, `document_id`, and `sections` tree.

**Error conditions:**
- Document not indexed

**Example:**

```json
// Call
{"tool": "doc_outline", "arguments": {"id": "DOC-01JX..."}}
// Response
{"success": true, "document_id": "DOC-01JX...", "sections": [{"path": "1", "level": 1, "title": "Overview", "word_count": 120, "content_hash": "..."}, {"path": "1.1", "level": 2, "title": "Goals", "word_count": 45}]}
```

---

### doc_section

Get a specific section's metadata and raw content from an indexed document.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Document ID |
| section_path | string | yes | Section path (e.g., `1`, `1.2`, `2.3.1`) |

**Returns:** JSON object with `success`, `document_id`, `section` metadata, and `content` text.

**Error conditions:**
- Document not indexed
- Section path not found

**Example:**

```json
// Call
{"tool": "doc_section", "arguments": {"id": "DOC-01JX...", "section_path": "1.2"}}
// Response
{"success": true, "document_id": "DOC-01JX...", "section": {"path": "1.2", "level": 2, "title": "Authentication Requirements", "word_count": 230, "content_hash": "..."}, "content": "## Authentication Requirements\n\nThe system must support..."}
```

---

### doc_classify

Submit agent-provided classifications (Layer 3) for a previously indexed document. Classifications assign semantic roles to document sections.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Document ID to classify |
| content_hash | string | yes | Content hash (must match current index to prevent stale classification) |
| model_name | string | yes | Name of the LLM that produced the classifications |
| model_version | string | yes | Version of the LLM |
| classifications | string | yes | JSON array of classification objects |

Each classification object in the JSON array should contain:
- `section_path` (required): section path from the outline
- `role` (required): semantic role (`requirement`, `decision`, `rationale`, `constraint`, `assumption`, `risk`, `question`, `definition`, `example`, `alternative`, `narrative`)
- `confidence` (required): float 0.0–1.0
- `summary` (optional): brief summary
- `concepts_intro` (optional): concepts introduced in this section
- `concepts_used` (optional): concepts used from elsewhere

**Returns:** JSON object with `success`, `document_id`, `message`, and `count`.

**Error conditions:**
- Document not indexed
- Content hash mismatch (index changed since classification started)
- Invalid JSON in classifications

**Example:**

```json
// Call
{"tool": "doc_classify", "arguments": {"id": "DOC-01JX...", "content_hash": "sha256:abc...", "model_name": "claude", "model_version": "3.5", "classifications": "[{\"section_path\": \"1.2\", \"role\": \"requirement\", \"confidence\": 0.9}]"}}
// Response
{"success": true, "document_id": "DOC-01JX...", "message": "Classifications applied successfully", "count": 1}
```

---

### doc_pending

List document IDs that have been indexed (Layers 1-2) but not yet classified (Layer 3).

| Parameter | Type | Required | Description |
|---|---|---|---|
| *(none)* | | | |

**Returns:** JSON object with `success`, `count`, and `document_ids` array.

**Error conditions:**
- Filesystem errors

**Example:**

```json
// Call
{"tool": "doc_pending", "arguments": {}}
// Response
{"success": true, "count": 2, "document_ids": ["DOC-01JXaaa...", "DOC-01JXbbb..."]}
```

---

### doc_find_by_entity

Find all document sections across the corpus that reference a specific entity.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID to search for (e.g., `FEAT-001`, `TASK-042`, `P1-basic-ui`) |

**Returns:** JSON object with `success`, `entity_id`, `count`, and `matches` array.

**Error conditions:**
- None (returns empty matches if not found)

**Example:**

```json
// Call
{"tool": "doc_find_by_entity", "arguments": {"entity_id": "FEAT-01JX..."}}
// Response
{"success": true, "entity_id": "FEAT-01JX...", "count": 3, "matches": [{"document_id": "DOC-...", "section_path": "2.1", "title": "Login Feature"}, ...]}
```

---

### doc_find_by_concept

Find all document sections that introduce or use a specific concept. Concepts are identified during Layer 3 classification.

| Parameter | Type | Required | Description |
|---|---|---|---|
| concept | string | yes | Concept name to search for (case-insensitive, normalised) |

**Returns:** JSON object with `success`, `concept`, `count`, and `matches` array.

**Error conditions:**
- None (returns empty matches if not found)

**Example:**

```json
// Call
{"tool": "doc_find_by_concept", "arguments": {"concept": "JWT"}}
// Response
{"success": true, "concept": "JWT", "count": 5, "matches": [...]}
```

---

### doc_find_by_role

Find all document fragments with a given semantic role across the corpus.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| role | string | yes | Fragment role to search for | `requirement`, `decision`, `rationale`, `constraint`, `assumption`, `risk`, `question`, `definition`, `example`, `alternative`, `narrative` |
| scope | string | no | Limit search to a specific document ID | |

**Returns:** JSON object with `success`, `role`, `count`, and `matches` array.

**Error conditions:**
- None (returns empty matches if not found)

**Example:**

```json
// Call
{"tool": "doc_find_by_role", "arguments": {"role": "decision"}}
// Response
{"success": true, "role": "decision", "count": 4, "matches": [...]}
```

---

### doc_impact

Find what references or depends on a given section. Returns all graph edges where the target matches the section ID.

| Parameter | Type | Required | Description |
|---|---|---|---|
| section_id | string | yes | Section ID in the format `docID#sectionPath` (e.g., `PROJECT/design-workflow#1.2`) |

**Returns:** JSON object with `success`, `section_id`, `count`, and `edges` array. Each edge has `from`, `from_type`, `to`, `to_type`, and `edge_type`.

**Error conditions:**
- Section not found

**Example:**

```json
// Call
{"tool": "doc_impact", "arguments": {"section_id": "DOC-01JX...#1.2"}}
// Response
{"success": true, "section_id": "DOC-01JX...#1.2", "count": 2, "edges": [{"from": "DOC-01JX...#2.1", "from_type": "section", "to": "DOC-01JX...#1.2", "to_type": "section", "edge_type": "references"}]}
```

---

### doc_extraction_guide

Return an extraction guide for a document: structural outline with section roles, entity references already found, and classification hints. Use this before extracting entities or decisions from a document.

| Parameter | Type | Required | Description |
|---|---|---|---|
| document_id | string | yes | Document record ID |

**Returns:** JSON object with `success`, `document_id`, `document_path`, `content_hash`, `classified` (boolean), `outline` (annotated section tree), `entity_refs` array, and `extraction_hints` array.

**Error conditions:**
- Document not indexed

**Example:**

```json
// Call
{"tool": "doc_extraction_guide", "arguments": {"document_id": "DOC-01JX..."}}
// Response
{"success": true, "document_id": "DOC-01JX...", "classified": true, "outline": [{"path": "1", "title": "Overview", "level": 1, "role": "narrative"}], "entity_refs": [{"entity_id": "FEAT-01JX...", "entity_type": "feature", "section_path": "2.1"}], "extraction_hints": ["Layer 3 classifications are available", "2 requirement section(s) found"]}
```

---

## 8. Knowledge Management

### knowledge_contribute

Contribute a new knowledge entry to the shared knowledge base. Topics are normalised (lowercased, hyphenated). Duplicate detection rejects entries with an identical topic or similar content (Jaccard > 0.65) in the same scope.

| Parameter | Type | Required | Description |
|---|---|---|---|
| topic | string | yes | Topic identifier (will be normalised) |
| content | string | yes | Concise, actionable statement of the knowledge |
| scope | string | yes | Scope: a profile name or `project` |
| tier | number | no | Knowledge tier: 2 (project-level) or 3 (session-level, default) |
| learned_from | string | no | Provenance: Task ID or other reference |
| created_by | string | no | Identity of the contributor |
| tags | array | no | Classification tags |

**Returns:** JSON object with `success`, `message`, and `entry` (the full record fields). On duplicate detection, returns `success: false`, `duplicate: true`, `message`, and `existing` entry.

**Error conditions:**
- Missing required parameters
- Duplicate entry detected (same topic or similar content in scope)

**Example:**

```json
// Call
{"tool": "knowledge_contribute", "arguments": {"topic": "go-error-handling", "content": "Always wrap errors with fmt.Errorf and %w for proper error chains", "scope": "backend", "tier": 2, "tags": ["go", "conventions"]}}
// Response
{"success": true, "message": "Knowledge entry contributed successfully", "entry": {"id": "KE-01JX...", "topic": "go-error-handling", "content": "Always wrap errors with...", "scope": "backend", "tier": 2, "status": "contributed"}}
```

---

### knowledge_get

Get a knowledge entry by ID. Includes staleness information for entries with git_anchors.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Knowledge entry ID (`KE-...`) |

**Returns:** JSON object with `success`, `entry` (full fields), and optional `staleness` object.

**Error conditions:**
- Entry not found

**Example:**

```json
// Call
{"tool": "knowledge_get", "arguments": {"id": "KE-01JX..."}}
// Response
{"success": true, "entry": {"id": "KE-01JX...", "topic": "go-error-handling", "content": "...", "status": "confirmed", "confidence": 0.9}}
```

---

### knowledge_list

List knowledge entries with optional filters. Retired entries are excluded by default.

| Parameter | Type | Required | Description |
|---|---|---|---|
| tier | number | no | Filter by tier: 2 or 3 |
| scope | string | no | Filter by scope |
| status | string | no | Filter by status: `contributed`, `confirmed`, `disputed`, `stale`, `retired` |
| topic | string | no | Filter by exact normalised topic |
| min_confidence | number | no | Minimum confidence score (0.0–1.0) |
| tags | array | no | Entries must have all of these tags |
| include_retired | boolean | no | Include retired entries (default: false) |

**Returns:** JSON object with `success`, `count`, and `entries` array.

**Error conditions:**
- Filesystem errors

**Example:**

```json
// Call
{"tool": "knowledge_list", "arguments": {"scope": "backend", "tier": 2}}
// Response
{"success": true, "count": 8, "entries": [{"id": "KE-01JX...", "topic": "go-error-handling", ...}, ...]}
```

---

### knowledge_update

Update the content of a knowledge entry. Resets `use_count`, `miss_count`, and `confidence` to defaults.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Knowledge entry ID (`KE-...`) |
| content | string | yes | New content for the entry |

**Returns:** JSON object with `success`, `message`, and updated `entry`.

**Error conditions:**
- Entry not found
- Entry is retired

**Example:**

```json
// Call
{"tool": "knowledge_update", "arguments": {"id": "KE-01JX...", "content": "Updated: Always use fmt.Errorf with %w, and define sentinel errors in the errors.go file"}}
// Response
{"success": true, "message": "Knowledge entry updated successfully", "entry": {"id": "KE-01JX...", "content": "Updated: Always use...", "use_count": 0, "miss_count": 0}}
```

---

### knowledge_confirm

Manually confirm a knowledge entry, transitioning it from `contributed` or `disputed` to `confirmed` status.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Knowledge entry ID (`KE-...`) |

**Returns:** JSON object with `success`, `message`, and updated `entry`.

**Error conditions:**
- Entry not found
- Entry already confirmed (no-op)

**Example:**

```json
// Call
{"tool": "knowledge_confirm", "arguments": {"id": "KE-01JX..."}}
// Response
{"success": true, "message": "Knowledge entry confirmed", "entry": {"id": "KE-01JX...", "status": "confirmed"}}
```

---

### knowledge_flag

Flag a knowledge entry as incorrect or disputed. Increments `miss_count` and recomputes confidence. Auto-retires if `miss_count` reaches 2.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Knowledge entry ID (`KE-...`) |
| reason | string | yes | Reason for flagging |

**Returns:** JSON object with `success`, `message`, and updated `entry`.

**Error conditions:**
- Entry not found
- Missing reason

**Example:**

```json
// Call
{"tool": "knowledge_flag", "arguments": {"id": "KE-01JX...", "reason": "This convention was superseded by the new linter rules"}}
// Response
{"success": true, "message": "Knowledge entry flagged", "entry": {"id": "KE-01JX...", "status": "disputed", "miss_count": 1}}
```

---

### knowledge_retire

Manually retire a knowledge entry, marking it as no longer valid. Retired entries are excluded from listing by default.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Knowledge entry ID (`KE-...`) |
| reason | string | yes | Reason for retiring |

**Returns:** JSON object with `success`, `message`, and updated `entry`.

**Error conditions:**
- Entry not found
- Missing reason

**Example:**

```json
// Call
{"tool": "knowledge_retire", "arguments": {"id": "KE-01JX...", "reason": "Project no longer uses this pattern"}}
// Response
{"success": true, "message": "Knowledge entry retired", "entry": {"id": "KE-01JX...", "status": "retired"}}
```

---

### knowledge_promote

Promote a tier-3 knowledge entry to tier 2 in place, extending its TTL from 30 to 90 days.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Knowledge entry ID (`KE-...`) |

**Returns:** JSON object with `success`, `message`, and updated `entry`.

**Error conditions:**
- Entry not found
- Entry is already tier 2

**Example:**

```json
// Call
{"tool": "knowledge_promote", "arguments": {"id": "KE-01JX..."}}
// Response
{"success": true, "message": "Knowledge entry promoted to tier 2", "entry": {"id": "KE-01JX...", "tier": 2}}
```

---

### knowledge_check_staleness

Check staleness of knowledge entries that have `git_anchors`. An entry is stale if any anchored file was modified after the entry was last confirmed.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entry_id | string | no | Specific entry ID to check. If omitted, checks all anchored entries |
| scope | string | no | Filter entries by scope |

**Returns:** JSON object with `success`, `stale_entries` array, and `total_checked` count. Each stale entry includes `entry_id`, `topic`, and `staleness` details.

**Error conditions:**
- Specific entry not found (when entry_id provided)
- Git errors

**Example:**

```json
// Call
{"tool": "knowledge_check_staleness", "arguments": {}}
// Response
{"success": true, "stale_entries": [{"entry_id": "KE-01JX...", "topic": "api-endpoint-list", "staleness": {"is_stale": true, "stale_files": ["internal/api/routes.go"]}}], "total_checked": 15}
```

---

### knowledge_prune

Prune expired knowledge entries based on TTL rules. Tier-3 entries expire after 30 days without use; tier-2 after 90 days.

| Parameter | Type | Required | Description |
|---|---|---|---|
| dry_run | boolean | no | If true, report what would be pruned without acting (default: false) |
| tier | number | no | Only prune entries of this tier (2 or 3) |

**Returns:** JSON object with `success`, `dry_run`, and either `would_prune` (dry run) or `pruned` array. Each item includes `entry_id`, `topic`, `tier`, and `reason`.

**Error conditions:**
- Filesystem errors loading entries

**Example:**

```json
// Call
{"tool": "knowledge_prune", "arguments": {"dry_run": true}}
// Response
{"success": true, "dry_run": true, "would_prune": [{"entry_id": "KE-01JX...", "topic": "old-pattern", "tier": 3, "reason": "tier-3 expired: 42 days since last use (TTL: 30 days)"}]}
```

---

### knowledge_compact

Compact knowledge entries by merging duplicates and near-duplicates, and flagging contradictions. Tier-3 entries are auto-merged; tier-2 entries are flagged for review.

| Parameter | Type | Required | Description |
|---|---|---|---|
| dry_run | boolean | no | If true, report what would be compacted (default: false) |
| scope | string | no | Only compact entries in this scope |

**Returns:** JSON object with `success`, `dry_run`, and `compaction_result` containing `duplicates_merged`, `near_duplicates_merged`, `conflicts_flagged`, and `details` array. Each detail has `action`, `reason`, and `kept`/`discarded`/`entries` identifiers.

**Error conditions:**
- Filesystem errors

**Example:**

```json
// Call
{"tool": "knowledge_compact", "arguments": {"dry_run": true, "scope": "backend"}}
// Response
{"success": true, "dry_run": true, "compaction_result": {"duplicates_merged": 1, "near_duplicates_merged": 0, "conflicts_flagged": 1, "details": [{"action": "merge", "reason": "identical topic", "kept": "KE-01JXaaa...", "discarded": "KE-01JXbbb..."}]}}
```

---

### knowledge_resolve_conflict

Resolve a conflict between two knowledge entries by keeping one and retiring the other. Optionally merge content from the retired entry into the kept entry.

| Parameter | Type | Required | Description |
|---|---|---|---|
| keep | string | yes | ID of the knowledge entry to keep (`KE-...`) |
| retire | string | yes | ID of the knowledge entry to retire (`KE-...`) |
| merge_content | boolean | no | If true, merge usage counts and git_anchors from retired into kept (default: false) |

**Returns:** JSON object with `success` and `resolved` containing `kept`, `retired`, and `merged` fields.

**Error conditions:**
- Either entry not found

**Example:**

```json
// Call
{"tool": "knowledge_resolve_conflict", "arguments": {"keep": "KE-01JXaaa...", "retire": "KE-01JXbbb...", "merge_content": true}}
// Response
{"success": true, "resolved": {"kept": "KE-01JXaaa...", "retired": "KE-01JXbbb...", "merged": true}}
```

---

### context_report

Report knowledge entry usage from a completed task. For each used entry: increments `use_count` and updates `last_used`; auto-confirms if `use_count` >= 3 and `miss_count` == 0. For each flagged entry: increments `miss_count`; auto-retires if `miss_count` >= 2.

| Parameter | Type | Required | Description |
|---|---|---|---|
| task_id | string | yes | ID of the task that consumed the entries |
| used | array | yes | List of knowledge entry IDs that were used and found helpful |
| flagged | string | no | JSON array of flagged entries: `[{"entry_id": "KE-...", "reason": "..."}]` |

**Returns:** JSON object with `success`, `task_id`, `used_count`, `flagged_count`, and `message`.

**Error conditions:**
- Missing task_id or used list
- Invalid JSON in flagged parameter

**Example:**

```json
// Call
{"tool": "context_report", "arguments": {"task_id": "T-01JX...", "used": ["KE-01JXaaa...", "KE-01JXbbb..."], "flagged": "[{\"entry_id\": \"KE-01JXccc...\", \"reason\": \"outdated advice\"}]"}}
// Response
{"success": true, "task_id": "T-01JX...", "used_count": 2, "flagged_count": 1, "message": "Context report processed successfully"}
```

---

## 9. Context and Profiles

### context_assemble

Assemble a context packet for an agent role. Returns the role profile, relevant knowledge entries (Tier 2 and Tier 3 scoped to the role or project), design context from document intelligence (if `task_id` provided), and task instructions — all within the byte budget.

Profile and task instructions are never trimmed. When over budget, lowest-confidence Tier 3 is trimmed first, then Tier 2, then design context.

| Parameter | Type | Required | Description |
|---|---|---|---|
| role | string | yes | Profile ID for the agent role (e.g., `backend`, `frontend`) |
| task_id | string | no | Task entity ID to include task instructions and design context |
| max_bytes | number | no | Maximum byte budget (default: 30720) |
| orchestration_context | string | no | Handoff note from orchestrating agent. Injected as ephemeral Tier 3 entry; not persisted |

**Returns:** JSON object with `success`, `role`, `byte_count`, `trimmed` (list of dropped items), and `items` array. Each item has `source`, `entry_id` (optional), `priority`, `confidence`, `content`, and optional `staleness`.

**Error conditions:**
- Profile not found
- Task not found (when task_id provided)

**Example:**

```json
// Call
{"tool": "context_assemble", "arguments": {"role": "backend", "task_id": "T-01JX...", "orchestration_context": "Focus on error handling patterns"}}
// Response
{"success": true, "role": "backend", "task_id": "T-01JX...", "byte_count": 12480, "trimmed": [], "items": [{"source": "profile", "priority": "required", "content": "..."}, {"source": "knowledge", "entry_id": "KE-01JX...", "priority": "high", "confidence": 0.9, "content": "..."}]}
```

---

### profile_get

Get a context profile by ID. By default returns the fully resolved profile with inheritance applied.

| Parameter | Type | Required | Description |
|---|---|---|---|
| id | string | yes | Profile ID (filename without `.yaml` extension) |
| resolved | boolean | no | Whether to apply inheritance resolution (default: true) |

**Returns:** JSON object with `success` and `profile` containing `id`, `resolved` flag, and optional `description`, `inherits`, `packages`, `conventions`, `architecture`.

**Error conditions:**
- Profile not found

**Example:**

```json
// Call
{"tool": "profile_get", "arguments": {"id": "backend"}}
// Response
{"success": true, "profile": {"id": "backend", "resolved": true, "description": "Go backend development context", "packages": ["net/http", "encoding/json"], "conventions": ["Use structured logging", "Wrap all errors"]}}
```

---

### profile_list

List all context profiles with their ID, parent (inherits), and description.

| Parameter | Type | Required | Description |
|---|---|---|---|
| *(none)* | | | |

**Returns:** JSON object with `success`, `count`, and `profiles` array. Each profile has `id`, `inherits`, and `description`.

**Error conditions:**
- Profile directory not found

**Example:**

```json
// Call
{"tool": "profile_list", "arguments": {}}
// Response
{"success": true, "count": 3, "profiles": [{"id": "base", "inherits": "", "description": "Shared conventions"}, {"id": "backend", "inherits": "base", "description": "Go backend"}, {"id": "frontend", "inherits": "base", "description": "React frontend"}]}
```

---

## 10. Work Queue and Dispatch

### work_queue

Return the current ready task queue, promoting eligible queued tasks first. This is a write-through query: it promotes queued tasks whose dependencies are all in terminal states (`done`, `not-planned`, or `duplicate`) to `ready` status as a side effect.

Returns all ready tasks sorted by estimate (ascending, null last), then age (descending), then task ID.

| Parameter | Type | Required | Description |
|---|---|---|---|
| role | string | no | Filter results to tasks whose parent feature matches this role profile |
| conflict_check | boolean | no | When true, annotate each ready task with conflict risk against currently active tasks |

**Returns:** JSON object with `queue` array, `promoted_count`, and `total_queued`. Each queue item has `task_id`, `slug`, `summary`, `parent_feature`, `feature_slug`, `estimate`, `age_days`, `status`, and optionally `conflict_risk` and `conflict_with`.

**Error conditions:**
- Filesystem errors

**Example:**

```json
// Call
{"tool": "work_queue", "arguments": {"conflict_check": true}}
// Response
{"queue": [{"task_id": "T-01JX...", "slug": "email-validation", "summary": "Implement email format validation", "parent_feature": "FEAT-01JX...", "estimate": 2, "age_days": 3, "status": "ready", "conflict_risk": "low"}], "promoted_count": 1, "total_queued": 5}
```

---

### dependency_status

Show the dependency picture for a given task: each dependency, its current status, and whether it is blocking or resolved.

| Parameter | Type | Required | Description |
|---|---|---|---|
| task_id | string | yes | Task ID to check dependencies for |

**Returns:** JSON object with `task_id`, `slug`, `status`, `depends_on_count`, `blocking_count`, and `dependencies` array. Each dependency has `task_id`, `slug`, `status`, `blocking`, and `terminal_state`.

**Error conditions:**
- Task not found

**Example:**

```json
// Call
{"tool": "dependency_status", "arguments": {"task_id": "T-01JX..."}}
// Response
{"task_id": "T-01JX...", "slug": "email-validation", "status": "queued", "depends_on_count": 2, "blocking_count": 1, "dependencies": [{"task_id": "T-01JXaaa...", "slug": "setup-form", "status": "done", "blocking": false, "terminal_state": "done"}, {"task_id": "T-01JXbbb...", "slug": "api-endpoint", "status": "active", "blocking": true, "terminal_state": null}]}
```

---

### dispatch_task

Atomically claim a ready task and return its context packet. Transitions the task from `ready` to `active`, records dispatch metadata, and assembles the context packet for the executing agent.

| Parameter | Type | Required | Description |
|---|---|---|---|
| task_id | string | yes | Task ID to dispatch (must be in `ready` status) |
| role | string | yes | Role profile ID for the executing agent (e.g., `backend`, `frontend`) |
| dispatched_by | string | yes | Identity string of the orchestrating agent |
| orchestration_context | string | no | Handoff note injected into context packet (ephemeral, not persisted) |
| max_bytes | number | no | Byte budget for context assembly (default: 30720) |

**Returns:** JSON object with `task` (task details) and `context` (assembled context packet with `role`, `byte_usage`, `trimmed`, and `items`).

**Error conditions:**
- Task not found
- Task not in `ready` status
- Profile not found for the specified role
- If context assembly fails after claiming, error includes recovery hint to use `context_assemble` manually

**Example:**

```json
// Call
{"tool": "dispatch_task", "arguments": {"task_id": "T-01JX...", "role": "backend", "dispatched_by": "orchestrator-agent"}}
// Response
{"task": {"id": "T-01JX...", "status": "active", "dispatched_by": "orchestrator-agent"}, "context": {"role": "backend", "byte_usage": 14200, "trimmed": [], "items": [...]}}
```

---

### complete_task

Close the dispatch loop for a completed task. Transitions the task to `done` (or `needs-review`), records completion metadata, and contributes knowledge entries to the knowledge base.

| Parameter | Type | Required | Description |
|---|---|---|---|
| task_id | string | yes | Task ID to complete (must be in `active` status) |
| summary | string | yes | Brief description of what was accomplished |
| to_status | string | no | Target status: `done` (default) or `needs-review` |
| files_modified | array | no | Files created or modified (string array) |
| verification_performed | string | no | Testing or verification carried out |
| blockers_encountered | string | no | Obstacles noted for future work |
| knowledge_entries | array | no | Knowledge entries to contribute. Each object: `{topic, content, scope, tier, tags}` |

**Returns:** JSON object with `task` (updated state), `knowledge_contributions` (with `accepted` and `rejected` arrays, `total_attempted`, `total_accepted`), and `unblocked_tasks` array of tasks that were promoted because this task completed.

**Error conditions:**
- Task not found
- Task not in `active` status
- Invalid `to_status` value

**Example:**

```json
// Call
{"tool": "complete_task", "arguments": {"task_id": "T-01JX...", "summary": "Implemented email validation with RFC 5322 regex", "files_modified": ["internal/auth/validate.go", "internal/auth/validate_test.go"], "verification_performed": "Unit tests pass, covers empty, valid, and malformed inputs", "knowledge_entries": [{"topic": "email-validation-regex", "content": "Use RFC 5322 regex for email validation", "scope": "backend"}]}}
// Response
{"task": {"id": "T-01JX...", "status": "done"}, "knowledge_contributions": {"accepted": [{"entry_id": "KE-01JX...", "topic": "email-validation-regex"}], "rejected": [], "total_attempted": 1, "total_accepted": 1}, "unblocked_tasks": [{"task_id": "T-01JXnext...", "slug": "password-validation", "status": "ready"}]}
```

---

## 11. Human Checkpoints

### human_checkpoint

Record a structured decision point requiring human input. After calling this, stop dispatching new tasks until you poll `human_checkpoint_get` and receive `status: responded`.

| Parameter | Type | Required | Description |
|---|---|---|---|
| question | string | yes | The decision or question requiring human input |
| context | string | yes | Background information to help the human answer |
| orchestration_summary | string | yes | Brief state of the orchestration session at checkpoint time |
| created_by | string | yes | Identity of the orchestrating agent |

**Returns:** JSON object with `checkpoint_id`, `status` (`pending`), `created_at`, and `message` with polling instructions.

**Error conditions:**
- Missing required parameters

**Example:**

```json
// Call
{"tool": "human_checkpoint", "arguments": {"question": "Should we use bcrypt or argon2id for password hashing?", "context": "Both are secure. Bcrypt is more widely supported, argon2id is newer and more resistant to GPU attacks.", "orchestration_summary": "Completing auth feature, 3 of 5 tasks done", "created_by": "orchestrator-agent"}}
// Response
{"checkpoint_id": "CHK-01JX...", "status": "pending", "created_at": "2024-01-15T10:30:00Z", "message": "Checkpoint recorded. Stop dispatching new tasks. Poll human_checkpoint_get with checkpoint_id until status is 'responded'."}
```

---

### human_checkpoint_respond

Record a human response to a pending checkpoint.

| Parameter | Type | Required | Description |
|---|---|---|---|
| checkpoint_id | string | yes | CHK ID of the checkpoint to respond to |
| response | string | yes | The human's answer or decision |

**Returns:** JSON object with `checkpoint_id`, `status` (`responded`), and `responded_at`.

**Error conditions:**
- Checkpoint not found
- Checkpoint already responded

**Example:**

```json
// Call
{"tool": "human_checkpoint_respond", "arguments": {"checkpoint_id": "CHK-01JX...", "response": "Use argon2id. It's worth the extra dependency for the security improvement."}}
// Response
{"checkpoint_id": "CHK-01JX...", "status": "responded", "responded_at": "2024-01-15T11:05:00Z"}
```

---

### human_checkpoint_get

Get the current state of a checkpoint. Poll this after calling `human_checkpoint` until status is `responded`.

| Parameter | Type | Required | Description |
|---|---|---|---|
| checkpoint_id | string | yes | CHK ID of the checkpoint to retrieve |

**Returns:** Full checkpoint record with `id`, `question`, `context`, `orchestration_summary`, `status`, `created_at`, `created_by`, `responded_at`, and `response`.

**Error conditions:**
- Checkpoint not found

**Example:**

```json
// Call
{"tool": "human_checkpoint_get", "arguments": {"checkpoint_id": "CHK-01JX..."}}
// Response
{"id": "CHK-01JX...", "question": "Should we use bcrypt or argon2id?", "status": "responded", "response": "Use argon2id.", "responded_at": "2024-01-15T11:05:00Z"}
```

---

### human_checkpoint_list

List checkpoint records. Optionally filter by status. Returns total count and pending count.

| Parameter | Type | Required | Description |
|---|---|---|---|
| status | string | no | Filter: `pending` or `responded` |

**Returns:** JSON object with `checkpoints` array, `total`, and `pending_count`.

**Error conditions:**
- Filesystem errors

**Example:**

```json
// Call
{"tool": "human_checkpoint_list", "arguments": {"status": "pending"}}
// Response
{"checkpoints": [{"id": "CHK-01JX...", "question": "...", "status": "pending"}], "total": 1, "pending_count": 1}
```

---

## 12. Estimation

### estimate_set

Set a story point estimate on a task, feature, epic, bug, or plan. Entity type is auto-detected from the ID.

Uses the Modified Fibonacci scale: `0`, `0.5`, `1`, `2`, `3`, `5`, `8`, `13`, `20`, `40`, `100`.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID (e.g., `T-01ABC...`, `FEAT-01ABC...`, `BUG-01ABC...`) |
| estimate | number | yes | Story point estimate from the Modified Fibonacci scale |

**Returns:** JSON object with `entity_id`, `entity_type`, `estimate`, `soft_limit_warning` (null or warning string), `references` (calibration examples), and `scale` (reference table).

**Error conditions:**
- Entity not found
- Invalid estimate value (not on the scale)

**Example:**

```json
// Call
{"tool": "estimate_set", "arguments": {"entity_id": "T-01JX...", "estimate": 3}}
// Response
{"entity_id": "T-01JX...", "entity_type": "task", "estimate": 3, "soft_limit_warning": null, "references": [], "scale": [{"points": 0, "meaning": "No effort"}, {"points": 0.5, "meaning": "Trivial"}, ...]}
```

---

### estimate_query

Query the current estimate and rollup statistics for an entity. For features, includes a task-level rollup. For epics/plans, includes a feature-level rollup.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID to query |

**Returns:** JSON object with `entity_id`, `entity_type`, `estimate`, and `rollup`. Rollup contents vary by entity type:
- **Feature rollup:** `task_total`, `progress`, `delta`, `task_count`, `estimated_task_count`, `excluded_task_count`
- **Epic/Plan rollup:** `feature_total`, `progress`, `delta`, `feature_count`, `estimated_feature_count`
- **Task/Bug:** `rollup` is null

**Error conditions:**
- Entity not found

**Example:**

```json
// Call
{"tool": "estimate_query", "arguments": {"entity_id": "FEAT-01JX..."}}
// Response
{"entity_id": "FEAT-01JX...", "entity_type": "feature", "estimate": 8, "rollup": {"task_total": 10, "progress": {"done": 3, "active": 1, "remaining": 6}, "delta": 2, "task_count": 5, "estimated_task_count": 5, "excluded_task_count": 0}}
```

---

### estimate_reference_add

Add a calibration reference example for an entity to help with future estimation. References are stored as project-scoped knowledge entries tagged `estimation-reference` with TTL exempt (`ttl_days=0`).

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID this reference anchors to |
| content | string | yes | Description of the work and its actual complexity |
| created_by | string | no | Identity of the contributor |

**Returns:** JSON object with `entry_id`, `entity_id`, `topic`, and `status`.

**Error conditions:**
- Missing required parameters

**Example:**

```json
// Call
{"tool": "estimate_reference_add", "arguments": {"entity_id": "T-01JX...", "content": "Email validation task: estimated 2, actual 3. Extra time spent on RFC 5322 edge cases and unicode email addresses."}}
// Response
{"entry_id": "KE-01JX...", "entity_id": "T-01JX...", "topic": "estimation-ref-T-01JX...", "status": "added"}
```

---

### estimate_reference_remove

Remove (retire) the estimation calibration reference for an entity.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID whose estimation reference should be removed |

**Returns:** JSON object with `entity_id`, `entry_id`, and `status`.

**Error conditions:**
- No estimation reference found for entity

**Example:**

```json
// Call
{"tool": "estimate_reference_remove", "arguments": {"entity_id": "T-01JX..."}}
// Response
{"entity_id": "T-01JX...", "entry_id": "KE-01JX...", "status": "removed"}
```

---

## 13. Feature Decomposition and Review

### decompose_feature

Propose a task decomposition for a feature based on its linked specification document. Applies embedded decomposition guidance (vertical slices, size limits, explicit dependencies). Returns a proposal preview — does NOT write any tasks.

| Parameter | Type | Required | Description |
|---|---|---|---|
| feature_id | string | yes | FEAT ID of the feature to decompose |
| context | string | no | Additional guidance for the decomposition |

**Returns:** JSON proposal object containing suggested tasks with slugs, summaries, dependencies, and size estimates. This is a preview only — no entities are created.

**Error conditions:**
- Feature not found
- No specification document linked to feature
- Specification not indexed

**Example:**

```json
// Call
{"tool": "decompose_feature", "arguments": {"feature_id": "FEAT-01JX...", "context": "Prefer small tasks under 3 story points"}}
// Response
{"feature_id": "FEAT-01JX...", "tasks": [{"slug": "setup-form-scaffold", "summary": "Create the login form component scaffold", "estimate": 2, "depends_on": []}, {"slug": "email-validation", "summary": "Add email validation", "estimate": 1, "depends_on": ["setup-form-scaffold"]}], "coverage": {"acceptance_criteria_covered": 5, "acceptance_criteria_total": 5}}
```

---

### decompose_review

Review a decomposition proposal against a feature's specification. Checks for uncovered acceptance criteria, oversized tasks, dependency cycles, and ambiguous summaries. Returns structured findings with pass/fail/warn status.

| Parameter | Type | Required | Description |
|---|---|---|---|
| feature_id | string | yes | FEAT ID of the feature |
| proposal | object | yes | The proposal object from `decompose_feature` output |

**Returns:** JSON review result with `status` (`pass`, `fail`, or `warn`), `findings` array (each with `severity`, `category`, and `message`), and `summary`.

**Error conditions:**
- Feature not found
- Invalid proposal format

**Example:**

```json
// Call
{"tool": "decompose_review", "arguments": {"feature_id": "FEAT-01JX...", "proposal": {"tasks": [...]}}}
// Response
{"status": "warn", "findings": [{"severity": "warning", "category": "oversized_task", "message": "Task 'api-integration' estimated at 8 points exceeds recommended maximum of 5"}], "summary": "1 warning, 0 errors"}
```

---

### slice_analysis

Analyse a feature's vertical slice structure without committing to a decomposition. Identifies candidate end-to-end slices from the feature's linked spec document, mapping each to stack layers, acceptance criteria outcomes, and size estimates. Identifies inter-slice dependencies.

Use for planning conversations before `decompose_feature`. Tip: when creating tasks from slices, tag them with `slice:<name>` for traceability.

| Parameter | Type | Required | Description |
|---|---|---|---|
| feature_id | string | yes | FEAT ID of the feature to analyse |

**Returns:** JSON analysis with `slices` array (each with `name`, `layers`, `acceptance_criteria`, `size_estimate`, `dependencies`) and `inter_slice_dependencies`.

**Error conditions:**
- Feature not found
- No specification linked

**Example:**

```json
// Call
{"tool": "slice_analysis", "arguments": {"feature_id": "FEAT-01JX..."}}
// Response
{"feature_id": "FEAT-01JX...", "slices": [{"name": "basic-login", "layers": ["frontend", "api", "auth-service"], "acceptance_criteria": ["User can log in with valid credentials"], "size_estimate": 5}, {"name": "password-reset", "layers": ["frontend", "api", "email-service"], "acceptance_criteria": ["User can reset password via email"], "size_estimate": 3}], "inter_slice_dependencies": [{"from": "password-reset", "to": "basic-login", "reason": "Shares auth service setup"}]}
```

---

### review_task_output

Run a first-pass review of a completed task's output against its verification criteria and parent feature spec. Returns findings with severity (error/warning) and triggers state transitions: fail → `needs-rework`, pass → `needs-review`. Tasks already in `needs-review` or `done` are reviewed without state changes.

| Parameter | Type | Required | Description |
|---|---|---|---|
| task_id | string | yes | TASK ID of the completed or active task |
| output_files | array | no | Paths of files produced or modified (string array) |
| output_summary | string | no | Agent's description of what was done |

**Returns:** JSON review result with `status` (`pass` or `fail`), `findings` array, and `state_transition` information.

**Error conditions:**
- Task not found
- Task not in a reviewable status

**Example:**

```json
// Call
{"tool": "review_task_output", "arguments": {"task_id": "T-01JX...", "output_files": ["internal/auth/validate.go"], "output_summary": "Implemented email validation"}}
// Response
{"status": "pass", "findings": [], "state_transition": {"from": "active", "to": "needs-review"}}
```

---

## 14. Conflict Analysis

### conflict_domain_check

Analyse conflict risk between two or more tasks that might run in parallel. Checks file overlap (planned and git-history), dependency ordering, and architectural boundary crossing. Returns per-pair risk assessment and recommendation.

| Parameter | Type | Required | Description |
|---|---|---|---|
| task_ids | array | yes | Two or more task IDs to check for conflict risk |

**Returns:** JSON object with `task_ids`, `overall_risk`, and `pairs` array. Each pair has `task_a`, `task_b`, `risk`, `dimensions` (with `file_overlap`, `dependency_order`, `boundary_crossing` sub-objects), and `recommendation` (`safe_to_parallelise`, `serialise`, or `checkpoint_required`).

**Error conditions:**
- Fewer than two task IDs provided
- Task not found

**Example:**

```json
// Call
{"tool": "conflict_domain_check", "arguments": {"task_ids": ["T-01JXaaa...", "T-01JXbbb..."]}}
// Response
{"task_ids": ["T-01JXaaa...", "T-01JXbbb..."], "overall_risk": "low", "pairs": [{"task_a": "T-01JXaaa...", "task_b": "T-01JXbbb...", "risk": "low", "dimensions": {"file_overlap": {"risk": "none", "shared_files": [], "git_conflicts": []}, "dependency_order": {"risk": "none", "detail": "no dependency relationship"}, "boundary_crossing": {"risk": "none", "detail": "same package"}}, "recommendation": "safe_to_parallelise"}]}
```

---

## 15. Incident Management

### incident_create

Create a new incident entity in `reported` status.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| slug | string | yes | URL-friendly identifier | |
| title | string | yes | Title of the incident | |
| severity | string | yes | Incident severity | `critical`, `high`, `medium`, `low` |
| summary | string | yes | Brief summary of the incident | |
| reported_by | string | yes | Who reported. Auto-resolved from config | |
| detected_at | string | no | When detected (ISO 8601). Defaults to now | |

**Returns:** JSON object with `Type`, `ID`, `DisplayID`, `Slug`, `Path`, and `State`.

**Error conditions:**
- Missing required parameters
- Invalid severity

**Example:**

```json
// Call
{"tool": "incident_create", "arguments": {"slug": "auth-outage", "title": "Authentication service outage", "severity": "critical", "summary": "Users unable to log in since 14:00 UTC", "reported_by": "sam"}}
// Response
{"Type": "incident", "ID": "INC-01JX...", "Slug": "auth-outage", "State": {"status": "reported", "severity": "critical", "title": "Authentication service outage"}}
```

---

### incident_update

Update an existing incident. Can change status (with lifecycle validation), severity, summary, timestamps, and affected features.

| Parameter | Type | Required | Description |
|---|---|---|---|
| incident_id | string | yes | Incident ID (full or prefix) |
| status | string | no | New lifecycle status |
| severity | string | no | New severity: `critical`, `high`, `medium`, `low` |
| summary | string | no | Updated summary |
| triaged_at | string | no | When triaged (ISO 8601) |
| mitigated_at | string | no | When mitigated (ISO 8601) |
| resolved_at | string | no | When resolved (ISO 8601) |
| affected_features | array | no | List of affected feature IDs (replaces existing) |

**Returns:** JSON object with updated incident state.

**Error conditions:**
- Incident not found
- Invalid status transition
- Invalid severity

**Example:**

```json
// Call
{"tool": "incident_update", "arguments": {"incident_id": "INC-01JX...", "status": "investigating", "affected_features": ["FEAT-01JX..."]}}
// Response
{"id": "INC-01JX...", "status": "investigating", "affected_features": ["FEAT-01JX..."]}
```

---

### incident_list

List incidents with optional status and severity filters.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| status | string | no | Filter by status | `reported`, `triaged`, `investigating`, `resolved`, `closed` |
| severity | string | no | Filter by severity | `critical`, `high`, `medium`, `low` |

**Returns:** JSON array of incident records.

**Error conditions:**
- Filesystem errors

**Example:**

```json
// Call
{"tool": "incident_list", "arguments": {"status": "investigating"}}
// Response
[{"id": "INC-01JX...", "title": "Auth outage", "severity": "critical", "status": "investigating"}]
```

---

### incident_link_bug

Link a bug to an incident. Adds the bug to the incident's `linked_bugs` list. Idempotent — linking the same bug twice has no effect.

| Parameter | Type | Required | Description |
|---|---|---|---|
| incident_id | string | yes | Incident ID (full or prefix) |
| bug_id | string | yes | Bug ID to link |

**Returns:** JSON object with updated incident including linked bugs.

**Error conditions:**
- Incident not found
- Bug not found

**Example:**

```json
// Call
{"tool": "incident_link_bug", "arguments": {"incident_id": "INC-01JX...", "bug_id": "BUG-01JX..."}}
// Response
{"id": "INC-01JX...", "linked_bugs": ["BUG-01JX..."]}
```

---

## 16. Git Integration — Worktrees

### worktree_create

Create a new Git worktree for a feature or bug entity. The worktree provides an isolated workspace for development.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID (`FEAT-...` or `BUG-...`) |
| branch_name | string | no | Custom branch name (auto-generated if omitted) |
| created_by | string | no | Who created the worktree. Auto-resolved |
| slug | string | no | Human-readable slug for branch naming (extracted from entity if omitted) |

**Returns:** JSON object with `success` and `worktree` record containing `id`, `entity_id`, `branch`, `path`, `status`, `created`, `created_by`.

**Error conditions:**
- `INVALID_ENTITY_TYPE`: entity ID must start with `FEAT-` or `BUG-`
- `ENTITY_NOT_FOUND`: entity does not exist
- `WORKTREE_EXISTS`: worktree already exists for this entity
- `GIT_ERROR`: git worktree creation failed

**Example:**

```json
// Call
{"tool": "worktree_create", "arguments": {"entity_id": "FEAT-01JX..."}}
// Response
{"success": true, "worktree": {"id": "WT-01JX...", "entity_id": "FEAT-01JX...", "branch": "feat/FEAT-01JX.../login-form", "path": ".worktrees/feat-FEAT-01JX...-login-form", "status": "active", "created": "2024-01-15T10:00:00Z", "created_by": "sam"}}
```

---

### worktree_get

Get the worktree record for a specific entity.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID (`FEAT-...` or `BUG-...`) |

**Returns:** JSON object with `success` and `worktree` record.

**Error conditions:**
- `NO_WORKTREE`: no worktree found for entity

**Example:**

```json
// Call
{"tool": "worktree_get", "arguments": {"entity_id": "FEAT-01JX..."}}
// Response
{"success": true, "worktree": {"id": "WT-01JX...", "entity_id": "FEAT-01JX...", "branch": "feat/FEAT-01JX.../login-form", "path": "...", "status": "active"}}
```

---

### worktree_list

List all worktrees with optional filtering by status or entity.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| status | string | no | Filter by status (default: `all`) | `active`, `merged`, `abandoned`, `all` |
| entity_id | string | no | Filter by entity ID | |

**Returns:** JSON object with `success`, `count`, and `worktrees` array.

**Error conditions:**
- Filesystem errors

**Example:**

```json
// Call
{"tool": "worktree_list", "arguments": {"status": "active"}}
// Response
{"success": true, "count": 3, "worktrees": [{"id": "WT-01JX...", "entity_id": "FEAT-01JX...", "branch": "feat/...", "status": "active"}, ...]}
```

---

### worktree_remove

Remove a worktree for an entity. By default, fails if there are uncommitted changes.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID (`FEAT-...` or `BUG-...`) |
| force | boolean | no | If true, remove even with uncommitted changes (default: false) |

**Returns:** JSON object with `success` and `removed` containing `id` and `path`.

**Error conditions:**
- `NO_WORKTREE`: no worktree found for entity
- `UNCOMMITTED_CHANGES`: worktree has uncommitted changes (use `force=true`)
- `GIT_ERROR`: git worktree removal failed

**Example:**

```json
// Call
{"tool": "worktree_remove", "arguments": {"entity_id": "FEAT-01JX...", "force": false}}
// Response
{"success": true, "removed": {"id": "WT-01JX...", "path": ".worktrees/feat-FEAT-01JX..."}}
```

---

## 17. Git Integration — Branches and Cleanup

### branch_status

Get branch health metrics for an entity's worktree branch. Reports staleness, drift from main, and merge conflicts.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID (`FEAT-...` or `BUG-...`) |

**Returns:** JSON object with `success`, `branch`, `metrics` (containing `branch_age_days`, `commits_behind_main`, `commits_ahead_of_main`, `last_commit_at`, `last_commit_age_days`, `has_conflicts`), `warnings` array, and `errors` array.

**Error conditions:**
- `NO_WORKTREE`: no worktree found for entity
- `BRANCH_NOT_FOUND`: branch does not exist

**Example:**

```json
// Call
{"tool": "branch_status", "arguments": {"entity_id": "FEAT-01JX..."}}
// Response
{"success": true, "branch": "feat/login-form", "metrics": {"branch_age_days": 5, "commits_behind_main": 3, "commits_ahead_of_main": 7, "last_commit_at": "2024-01-14T16:00:00Z", "last_commit_age_days": 1, "has_conflicts": false}, "warnings": [], "errors": []}
```

---

### cleanup_list

List worktrees pending cleanup. Shows merged and abandoned worktrees that are either ready for cleanup (past grace period) or scheduled (within grace period).

| Parameter | Type | Required | Description |
|---|---|---|---|
| include_pending | boolean | no | Include items past grace period that are ready (default: true) |
| include_scheduled | boolean | no | Include items within grace period, scheduled for future cleanup (default: true) |

**Returns:** JSON object with `success`, `pending_cleanup` array, and `scheduled_cleanup` array. Each item has `worktree_id`, `entity_id`, `branch`, `path`, `status`, and optionally `merged_at`/`cleanup_after`.

**Error conditions:**
- Filesystem errors

**Example:**

```json
// Call
{"tool": "cleanup_list", "arguments": {}}
// Response
{"success": true, "pending_cleanup": [{"worktree_id": "WT-01JX...", "entity_id": "FEAT-01JXold...", "branch": "feat/old-feature", "status": "ready", "merged_at": "2024-01-08T10:00:00Z", "cleanup_after": "2024-01-15T10:00:00Z"}], "scheduled_cleanup": []}
```

---

### cleanup_execute

Execute cleanup on worktrees. Removes worktree directories, deletes local branches, optionally deletes remote branches, and removes tracking records.

| Parameter | Type | Required | Description |
|---|---|---|---|
| worktree_id | string | no | Specific worktree ID to clean up (e.g., `WT-01JX...`). If omitted, cleans all ready items |
| dry_run | boolean | no | If true, simulates cleanup without making changes (default: false) |

**Returns:** JSON object with `success`, `dry_run`, `cleaned` array, and optional `errors` array. Each cleaned item has `worktree_id`, `branch`, `path`, and `remote_branch_deleted`.

**Error conditions:**
- Worktree not found (when specific ID provided)
- Worktree not ready for cleanup (still within grace period)

**Example:**

```json
// Call
{"tool": "cleanup_execute", "arguments": {"dry_run": true}}
// Response
{"success": true, "dry_run": true, "cleaned": [{"worktree_id": "WT-01JX...", "branch": "feat/old-feature", "path": ".worktrees/...", "remote_branch_deleted": false}], "message": "Dry run: no changes made. The listed items would be cleaned."}
```

---

## 18. Git Integration — Merge

### merge_readiness_check

Check if an entity (feature or bug) is ready to merge. Evaluates all merge gates and optionally checks PR status if GitHub is configured.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID (`FEAT-...` or `BUG-...`) |

**Returns:** JSON object with `entity_id`, `branch`, `overall_status` (`passed`, `blocked`, or `warning`), `gates` array (each with `name`, `status`, `severity`, `message`), and optional `pr_status`.

**Error conditions:**
- `NO_WORKTREE`: no worktree found for entity
- Entity not found

**Example:**

```json
// Call
{"tool": "merge_readiness_check", "arguments": {"entity_id": "FEAT-01JX..."}}
// Response
{"entity_id": "FEAT-01JX...", "branch": "feat/login-form", "overall_status": "passed", "gates": [{"name": "all_tasks_done", "status": "passed", "severity": "blocking"}, {"name": "no_conflicts", "status": "passed", "severity": "blocking"}, {"name": "branch_not_stale", "status": "passed", "severity": "warning"}]}
```

---

### merge_execute

Execute a merge for an entity after verifying all gates pass. Use override with reason to bypass blocking gates.

| Parameter | Type | Required | Description | Valid values |
|---|---|---|---|---|
| entity_id | string | yes | Entity ID (`FEAT-...` or `BUG-...`) | |
| override | boolean | no | Override blocking gates (default: false) | |
| override_reason | string | no | Required explanation when override is true | |
| merge_strategy | string | no | Merge strategy (default: `squash`) | `squash`, `merge`, `rebase` |
| delete_branch | boolean | no | Delete branch after merge (default: true) | |

**Returns:** JSON object with `merged` (containing `entity_id`, `branch`, `merge_commit`, `merged_at`) and optional `cleanup_scheduled`.

**Error conditions:**
- `NO_WORKTREE`: no worktree for entity
- `GATES_FAILED`: blocking gates not passed and override not set
- `OVERRIDE_REASON_REQUIRED`: override is true but no reason given
- `MERGE_CONFLICT`: branch has merge conflicts with main
- Invalid merge strategy

**Example:**

```json
// Call
{"tool": "merge_execute", "arguments": {"entity_id": "FEAT-01JX...", "merge_strategy": "squash"}}
// Response
{"merged": {"entity_id": "FEAT-01JX...", "branch": "feat/login-form", "merge_commit": "abc123def456", "merged_at": "2024-01-15T14:00:00Z"}, "cleanup_scheduled": {"cleanup_after": "2024-01-22T14:00:00Z"}}
```

---

## 19. Git Integration — Pull Requests

### pr_create

Create a new pull request for an entity's branch on GitHub.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID (`FEAT-...` or `BUG-...`) |
| draft | boolean | no | Create as draft PR (default: false) |

**Returns:** JSON object with `pr` (containing `url`, `number`, `title`, `state`, `draft`) and optional `warnings`.

**Error conditions:**
- `GITHUB_NOT_CONFIGURED`: no GitHub token in `.kbz/local.yaml`
- `NO_WORKTREE`: no worktree for entity
- `PR_EXISTS`: PR already exists for this branch
- Entity not found

**Example:**

```json
// Call
{"tool": "pr_create", "arguments": {"entity_id": "FEAT-01JX...", "draft": true}}
// Response
{"pr": {"url": "https://github.com/org/repo/pull/42", "number": 42, "title": "FEAT-01JX...: Login form with email/password", "state": "open", "draft": true}}
```

---

### pr_update

Update an existing pull request's description and labels based on current entity state.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID (`FEAT-...` or `BUG-...`) |

**Returns:** JSON object with `pr` containing `url`, `updated` flag, and `changes` array describing what was modified.

**Error conditions:**
- `GITHUB_NOT_CONFIGURED`: no GitHub token
- `NO_WORKTREE`: no worktree for entity
- `NO_PR`: no PR found for the branch

**Example:**

```json
// Call
{"tool": "pr_update", "arguments": {"entity_id": "FEAT-01JX..."}}
// Response
{"pr": {"url": "https://github.com/org/repo/pull/42", "updated": true, "changes": ["Updated description", "Added label: tasks-complete"]}}
```

---

### pr_status

Get the status of a pull request for an entity, including CI status, reviews, and merge conflicts.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_id | string | yes | Entity ID (`FEAT-...` or `BUG-...`) |

**Returns:** JSON object with `pr` containing `url`, `number`, `state`, `draft`, `ci_status`, `review_status`, `reviews` array, `has_conflicts`, and `mergeable`.

**Error conditions:**
- `GITHUB_NOT_CONFIGURED`: no GitHub token
- `NO_WORKTREE`: no worktree for entity
- `NO_PR`: no PR found for the branch

**Example:**

```json
// Call
{"tool": "pr_status", "arguments": {"entity_id": "FEAT-01JX..."}}
// Response
{"pr": {"url": "https://github.com/org/repo/pull/42", "number": 42, "state": "open", "draft": false, "ci_status": "success", "review_status": "approved", "reviews": [{"user": "reviewer", "state": "APPROVED"}], "has_conflicts": false, "mergeable": true}}
```

---

## 20. Agent Capabilities

### suggest_links

Scan free text for entity ID patterns (`FEAT-`, `TASK-`, `BUG-`, `DEC-`, `KE-`, Plan IDs) and look up each found ID in the entity and knowledge stores. Returns a list of confirmed references with their entity type and title.

| Parameter | Type | Required | Description |
|---|---|---|---|
| text | string | yes | Free text to scan for entity references |
| scope | string | no | Entity type filter (e.g., `feature`, `task`, `bug`, `decision`, `plan`, `knowledge_entry`) |

**Returns:** JSON object with `success`, `count`, and `links` array. Each link has `text_span`, `entity_id`, `entity_type`, `entity_title`, and `match_quality`.

**Error conditions:**
- Missing text parameter

**Example:**

```json
// Call
{"tool": "suggest_links", "arguments": {"text": "This relates to FEAT-01JX... and depends on the decision in DEC-01JX..."}}
// Response
{"success": true, "count": 2, "links": [{"text_span": "FEAT-01JX...", "entity_id": "FEAT-01JX...", "entity_type": "feature", "entity_title": "Login form", "match_quality": "exact"}, {"text_span": "DEC-01JX...", "entity_id": "DEC-01JX...", "entity_type": "decision", "entity_title": "Use JWT for API authentication", "match_quality": "exact"}]}
```

---

### check_duplicates

Check whether an entity being created would duplicate an existing one. Computes Jaccard similarity between the candidate's title+summary and existing entities. Returns advisory candidates with similarity >= 0.5. Does NOT block creation.

| Parameter | Type | Required | Description |
|---|---|---|---|
| entity_type | string | yes | Entity type: `feature`, `task`, `bug`, `decision`, `plan`, `knowledge_entry` |
| title | string | yes | Title or topic of the candidate entity |
| summary | string | no | Optional summary of the candidate entity |

**Returns:** JSON object with `success`, `advisory` (always true), `entity_type`, `count`, `candidates` array (each with `entity_id`, `entity_type`, `title`, `similarity`), and `message`.

**Error conditions:**
- Missing required parameters

**Example:**

```json
// Call
{"tool": "check_duplicates", "arguments": {"entity_type": "feature", "title": "Login form", "summary": "Email and password login"}}
// Response
{"success": true, "advisory": true, "entity_type": "feature", "count": 1, "candidates": [{"entity_id": "FEAT-01JX...", "entity_type": "feature", "title": "Login form with email/password", "similarity": 0.72}], "message": "Found 1 candidate duplicate(s). This check is advisory — creation is not blocked."}
```

---

### doc_extraction_guide

*(Also listed under Document Intelligence — included here as it is registered in the Agent Capabilities tool group.)*

See [doc_extraction_guide](#doc_extraction_guide) in Section 7 for full details.

---

## 21. Batch Operations

### batch_import_documents

Batch import document records from a directory. Scans for matching Markdown files and creates document records idempotently. Already-imported files are skipped.

| Parameter | Type | Required | Description |
|---|---|---|---|
| path | string | yes | Directory path to scan for documents |
| default_type | string | no | Fallback document type when no path pattern matches (`design`, `specification`, `dev-plan`, `research`, `report`, `policy`) |
| owner | string | no | Parent Plan or Feature ID to assign as owner of imported documents |
| created_by | string | no | Who is importing. Auto-resolved |
| glob | string | no | Glob pattern to filter files. Path separator in pattern matches relative paths; no separator matches filenames only. `**` recursive matching is not supported. If empty, all `.md` files are imported |

**Returns:** JSON object with `success`, `imported` (count or array of imported records), `skipped` (array of `{path, reason}`), and `errors` (array of `{path, error}`).

**Error conditions:**
- Directory not found
- No `.md` files match

**Example:**

```json
// Call
{"tool": "batch_import_documents", "arguments": {"path": "docs/", "default_type": "design", "owner": "P1-user-auth"}}
// Response
{"success": true, "imported": 5, "skipped": [{"path": "docs/README.md", "reason": "already imported"}], "errors": []}
```

---

## 22. Migration

### migrate_phase2

Migrate Phase 1 epic entities to Phase 2 plan entities. Converts epics to plans, updates feature references, and creates required directories. The migration is idempotent: re-running it skips already-migrated entities. Requires a configured prefix registry in `.kbz/config.yaml`.

| Parameter | Type | Required | Description |
|---|---|---|---|
| *(none)* | | | |

**Returns:** JSON object with `success`, `plans_created`, `features_updated`, `files_moved`, `dirs_created`, optional `errors` array, and `message`.

**Error conditions:**
- Prefix registry not configured
- Filesystem errors during migration

**Example:**

```json
// Call
{"tool": "migrate_phase2", "arguments": {}}
// Response
{"success": true, "plans_created": 3, "features_updated": 12, "files_moved": 6, "dirs_created": 3, "message": "migration completed successfully"}
```

---

## 23. Rich Queries

### query_plan_tasks

Find all tasks belonging to features under a given Plan. Useful for getting a complete task breakdown for a Plan.

| Parameter | Type | Required | Description |
|---|---|---|---|
| plan_id | string | yes | Plan ID (e.g., `P1-basic-ui`) |

**Returns:** JSON object with `success`, `plan_id`, `count`, and `tasks` array.

**Error conditions:**
- Plan not found

**Example:**

```json
// Call
{"tool": "query_plan_tasks", "arguments": {"plan_id": "P1-user-auth"}}
// Response
{"success": true, "plan_id": "P1-user-auth", "count": 15, "tasks": [{"Type": "task", "ID": "T-01JX...", "Slug": "email-validation", "State": "ready"}, ...]}
```

---

## 24. Retrospective Synthesis

### retro

Synthesise accumulated retrospective signals from the knowledge base. Reads signal entries tagged `retrospective`, clusters them into themes by category and Jaccard similarity, and returns a ranked synthesis. Optionally generates and registers a markdown report document.

**Action: synthesise (default)**

| Parameter | Type | Required | Description |
|---|---|---|---|
| action | string | no | `synthesise` (default) or `report` |
| scope | string | no | Plan ID, Feature ID, or `"project"` (default: `"project"`) |
| since | string | no | ISO 8601 timestamp; only include signals created after this time |
| until | string | no | ISO 8601 timestamp; only include signals created before this time |
| min_severity | string | no | Minimum severity to include: `minor` (default), `moderate`, or `significant` |

**Returns:** JSON object with `scope`, `signal_count`, `period` (`{from, to}`), `themes` array, optional `worked_well` array, and optional `experiments` array.

Each theme has: `rank`, `category`, `title`, `signal_count`, `severity_score`, `signals` (KE IDs), `representative_observation`, and optional `top_suggestion`.

Each `worked_well` entry has: `title`, `signal_count`, `representative_observation`.

Each experiment has: `decision_id`, `title`, `positive_signals`, `negative_signals`, `net_assessment`, and `recommendation` (`keep`, `revise`, or `revert`).

**Action: report**

All synthesise parameters plus:

| Parameter | Type | Required | Description |
|---|---|---|---|
| output_path | string | yes | Repository-relative path for the generated markdown report file |
| title | string | no | Document title; defaults to `"Retrospective: {scope} {date}"` |

**Returns:** Same as synthesise, plus a `report` object with `path` (the output_path) and `document_id` (the registered document record ID). The registered document is in `draft` status.

**Error conditions:**
- `output_path` is empty (report action)
- `min_severity` is not one of `minor`, `moderate`, `significant`
- `since` or `until` is not a valid ISO 8601 timestamp
- Report action called twice with the same `output_path` (document already registered)

**Example (synthesise):**

```json
// Call
{"tool": "retro", "arguments": {}}
// Response
{"scope": "project", "signal_count": 12, "period": {"from": "2026-03-01T10:00:00Z", "to": "2026-03-27T18:00:00Z"}, "themes": [{"rank": 1, "category": "spec-ambiguity", "title": "Spec gaps slowed iteration", "signal_count": 5, "severity_score": 17, "signals": ["KE-001", "KE-003"], "representative_observation": "Error format undefined", "top_suggestion": "Add error format examples to spec template"}], "worked_well": [{"title": "Parallel worktrees", "signal_count": 3, "representative_observation": "Parallel worktrees reduced merge conflicts"}]}
```

**Example (report):**

```json
// Call
{"tool": "retro", "arguments": {"action": "report", "scope": "project", "output_path": "docs/retro/sprint-12.md", "title": "Sprint 12 Retrospective"}}
// Response
{"scope": "project", "signal_count": 12, "period": {"from": "2026-03-01T10:00:00Z", "to": "2026-03-27T18:00:00Z"}, "themes": [...], "report": {"path": "docs/retro/sprint-12.md", "document_id": "PROJECT/report-docs-retro-sprint-12"}}
```

---

## 25. Lifecycle Operation Constraints

Several tools enforce lifecycle state machine rules. Understanding these constraints is essential for correct orchestration.

### Status transition enforcement

`update_status` enforces the lifecycle state machine for each entity type. Attempting an invalid transition (e.g., moving a task from `queued` directly to `done`) returns an error. See the Schema Reference for the complete set of valid transitions per entity type.

### Document-driven lifecycle advancement

`doc_record_approve` triggers feature lifecycle advancement. For example:
- Approving a **design** document transitions the owning feature from `proposed` → `designing`
- Approving a **specification** document transitions the owning feature from `designing` → `specifying`

`doc_record_supersede` can trigger **backward** lifecycle transitions on the owning entity when the current approved document is superseded without an immediate replacement in the same type.

### Dispatch constraints

- `dispatch_task` requires the task to be in `ready` status. It atomically transitions the task to `active` and records dispatch metadata. A task in any other status will be rejected.
- `complete_task` requires the task to be in `active` status. It transitions to `done` (default) or `needs-review`.

### Review constraints

- `review_task_output` can transition tasks: `active` → `needs-rework` (on fail) or `active` → `needs-review` (on pass). Tasks already in `needs-review` or `done` are reviewed without state changes.

---

## 26. Idempotency Notes

The following tools are idempotent — calling them multiple times with the same arguments produces the same result without unintended side effects:

| Tool | Idempotency behaviour |
|---|---|
| `batch_import_documents` | Already-imported files are skipped; only new files are imported |
| `incident_link_bug` | Linking the same bug to the same incident twice has no effect |
| `rebuild_cache` | Can be called any number of times safely; always rebuilds from canonical files |
| `health_check` | Read-only, no side effects |
| `work_queue` | Promotes eligible tasks as a side effect, but repeated calls are safe — already-promoted tasks are not re-promoted |
| `knowledge_confirm` | Confirming an already-confirmed entry is a no-op |
| `migrate_phase2` | Re-running skips already-migrated entities |

All other tools may have side effects (creating entities, transitioning statuses, writing files). Callers should track which operations have been performed to avoid duplicate creation.