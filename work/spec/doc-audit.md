# Specification: `doc audit` Action

| Field    | Value                                                              |
|----------|--------------------------------------------------------------------|
| Status   | Draft                                                              |
| Created  | 2026-04-01                                                         |
| Updated  | 2026-04-01                                                         |
| Feature  | FEAT-01KN4ZPQFQN1C (doc-audit)                                     |
| Design   | `work/design/kanbanzai-2.5-infrastructure-hardening.md` §7         |

---

## 1. Purpose

This specification defines the requirements for the `audit` action on the `doc`
MCP tool. The action reconciles the document store against the filesystem,
reporting files that are on disk but not registered, and store records whose
files no longer exist on disk.

---

## 2. Goals

1. Agents and humans can discover which markdown files in document directories
   are absent from the document store without manual comparison.
2. Agents and humans can discover which store records have no corresponding file
   on disk.
3. Each unregistered file carries an inferred document type derived from its
   directory path using the same conventions as `doc import`.
4. The scan scope can be narrowed to a specific directory via a `path` parameter.
5. Registered files are counted in the summary but not individually listed by
   default, keeping output concise.

---

## 3. Scope

### 3.1 In Scope

- New `audit` action dispatched by the existing `doc` MCP tool handler.
- Filesystem walk of the default document directories.
- Comparison of walk results against the document store.
- `path` parameter to scope the walk to a single directory.
- `include_registered` parameter to optionally expand the registered file list.
- Structured JSON output: `unregistered`, `missing`, and `summary` fields.
- Type inference for unregistered files using the same directory-to-type mapping
  used by `doc import`.

### 3.2 Out of Scope

- Automatic registration of unregistered files (that is the role of `doc import`).
- Deletion or archiving of missing store records.
- Scanning directories outside the configured document directories.
- Any change to existing `doc` actions (`register`, `approve`, `import`, etc.).
- Configurable document directory lists (hardcoded default set in this release).

---

## 4. Default Document Directories

The audit walk covers the following directories by default:

| Directory        | Inferred Type    |
|------------------|------------------|
| `work/design/`   | design           |
| `work/spec/`     | specification    |
| `work/plan/`     | dev-plan         |
| `work/research/` | research         |
| `work/reports/`  | report           |
| `work/reviews/`  | report           |
| `docs/`          | report           |

Only `.md` files are considered. Subdirectory traversal is recursive within
each configured root.

---

## 5. Requirements

### 5.1 Filesystem Walk

**REQ-01.** When `doc(action: "audit")` is called without a `path` parameter,
the tool walks all default document directories listed in §4.

**REQ-02.** When `doc(action: "audit", path: "<dir>")` is called, the tool walks
only the specified directory. If the directory does not exist, the tool returns
an error.

**REQ-03.** The walk includes only files with a `.md` extension. Non-markdown
files are silently ignored.

**REQ-04.** The walk is recursive — files in subdirectories of a scanned
directory are included.

### 5.2 Comparison Against the Store

**REQ-05.** For each `.md` file found during the walk, the tool checks whether
a document record exists in the store with a matching `path` field. Path
matching is exact and case-sensitive.

**REQ-06.** A file with no matching store record is classified as **unregistered**.

**REQ-07.** A store record whose `path` field points to a file that does not
exist on disk is classified as **missing**.

**REQ-08.** The missing check covers all store records whose paths fall under
the scanned directories (or the full default set when no `path` parameter is
given). Store records in unscanned directories are not reported as missing.

### 5.3 Type Inference

**REQ-09.** Each unregistered file carries an `inferred_type` field. The value
is derived from the file's directory path using the directory-to-type mapping
in §4. Files in a directory not covered by the mapping have `inferred_type` of
`""` (empty string).

**REQ-10.** The type inference for `doc audit` uses the same mapping as
`doc import`. If the mapping changes in `doc import`, the change applies to
`doc audit` without a separate update.

### 5.4 Output Structure

**REQ-11.** The response always contains three top-level fields: `unregistered`,
`missing`, and `summary`.

**REQ-12.** `unregistered` is an array of objects. Each object contains:

| Field           | Type   | Description                              |
|-----------------|--------|------------------------------------------|
| `path`          | string | Repository-relative path of the file    |
| `inferred_type` | string | Inferred document type (may be empty)   |

**REQ-13.** `missing` is an array of objects. Each object contains:

| Field    | Type   | Description                                   |
|----------|--------|-----------------------------------------------|
| `path`   | string | Repository-relative path from the store record |
| `doc_id` | string | Document record ID from the store             |

**REQ-14.** `summary` is an object containing:

| Field            | Type    | Description                                  |
|------------------|---------|----------------------------------------------|
| `total_on_disk`  | integer | Total `.md` files found during the walk     |
| `registered`     | integer | Files with a matching store record           |
| `unregistered`   | integer | Files with no matching store record          |
| `missing`        | integer | Store records with no corresponding file     |

The counts satisfy: `registered + unregistered == total_on_disk`.

**REQ-15.** When `include_registered` is `true`, the response includes an
additional top-level `registered` array. Each entry contains:

| Field    | Type   | Description                              |
|----------|--------|------------------------------------------|
| `path`   | string | Repository-relative path of the file    |
| `doc_id` | string | Document record ID from the store        |

**REQ-16.** When `include_registered` is `false` or omitted, the `registered`
array is absent from the response. The `summary.registered` count is always
present regardless.

### 5.5 Parameters

**REQ-17.** The `audit` action accepts the following optional parameters:

| Parameter            | Type    | Default | Description                                       |
|----------------------|---------|---------|---------------------------------------------------|
| `path`               | string  | —       | Restrict walk to this directory                   |
| `include_registered` | boolean | `false` | Include full registered file list in response     |

**REQ-18.** All parameters are optional. Calling `doc(action: "audit")` with no
additional parameters is valid and scans all default directories.

### 5.6 Empty Results

**REQ-19.** If no unregistered files are found, `unregistered` is an empty
array (`[]`), not absent.

**REQ-20.** If no missing records are found, `missing` is an empty array
(`[]`), not absent.

---

## 6. Acceptance Criteria

**AC-16.** `doc(action: "audit")` returns unregistered files found under default
document directories.

**AC-17.** `doc(action: "audit")` returns missing records whose files no longer
exist on disk.

**AC-18.** Each unregistered file entry includes an `inferred_type` based on its
directory path.

**AC-19.** The `path` parameter scopes the scan to the specified directory;
files outside that directory are not reported.

**AC-20.** Files that are already registered are counted in `summary.registered`
but not individually listed when `include_registered` is absent or `false`.

---

## 7. Dependencies and Assumptions

- The document store is queryable by path (this is an existing capability).
- Directory-to-type inference logic is already implemented in `doc import` and
  is reused without duplication.
- The default document directories (`work/design/`, `work/spec/`, etc.) are
  fixed for this release. Configurability is out of scope.
- The tool operates on the local filesystem. Files not present in the working
  tree (e.g., unstaged deletions) are treated as absent.

---

## 8. Invariants

- The `audit` action is read-only. It does not create, update, or delete any
  store records or files.
- `summary.registered + summary.unregistered == summary.total_on_disk` must
  hold for every response.
- A file cannot appear in both `unregistered` and `registered` in the same
  response.
- The `missing` list is independent of the `unregistered` list — a store record
  can be missing even if no unregistered file has the same path.