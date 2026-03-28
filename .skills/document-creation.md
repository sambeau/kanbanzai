# SKILL: Document Creation

## Purpose

Register new documents with the kanbanzai system when creating markdown files in the `work/` directory.

## When to Use

- After creating any new `.md` file in `work/design/`, `work/spec/`, `work/plan/`, `work/research/`, `work/reports/`, or `work/reviews/`
- Before committing new documents to Git
- As a safety check before major commits or phase transitions

---

## Procedure: Single Document

### Step 1: Create the markdown file

Place the file in the appropriate subdirectory:

```
work/design/     → design documents, architecture, policies
work/spec/       → specifications with acceptance criteria
work/plan/       → implementation plans, decision logs, progress tracking
work/research/   → research reports, analysis, exploration
work/reports/    → review reports, audit reports, post-implementation reviews
work/reviews/    → feature and bug review reports from the reviewing lifecycle gate
```

### Step 2: Register with the system

Call `doc(action: register)`:

```
doc(action: "register", path: "work/design/my-document.md", type: "design", title: "Human-Readable Title")
```

**Type must match location:**

| Location | Type |
|----------|------|
| `work/design/` | `design` |
| `work/spec/` | `specification` |
| `work/plan/` | `dev-plan` |
| `work/research/` | `research` |
| `work/reports/` | `report` |
| `work/reviews/` | `report` |

The `created_by` parameter is optional and will auto-resolve from `.kbz/local.yaml` or `git config` if omitted.

### Step 3: Verify registration

Check that the document record was created:

```
doc(action: "get", path: "work/design/my-document.md")
```

Or if you have the document record ID from the register response:

```
doc(action: "get", id: "DOC-01JX...")
```

### Step 4: Commit both together

Stage and commit the markdown file and its document record:

```bash
git add work/design/my-document.md .kbz/state/documents/
git commit -m "docs(my-document): create design document for feature X

- Add work/design/my-document.md
- Register document record in .kbz/state/documents/"
```

---

## Procedure: Batch Import

Use when you have created multiple documents or as a safety check.

### Step 1: Run batch import

```
doc(action: "import", path: "work/plan", default_type: "dev-plan")
```

Or import all of `work/`:

```
doc(action: "import", path: "work")
```

The tool will:
- Scan for `.md` files
- Skip already-registered documents (idempotent)
- Infer document type from directory structure
- Return a summary of imported, skipped, and errored files

### Step 2: Review the import summary

Check the output for:
- `imported` — successfully registered documents
- `skipped` — already registered (expected for repeat runs)
- `errors` — files that failed to import (investigate these)

### Step 3: Commit the new document records

```bash
git add .kbz/state/documents/
git commit -m "workflow(PROJECT): register new documents with system

- Batch import work/plan/ documents
- X new documents registered"
```

---

## Refreshing Stale Documents

When a registered document file is edited after registration, its stored content hash becomes stale. Use `doc(action: refresh)` to fix the hash in place:

```
doc(action: "refresh", id: "DOC-01JX...")
```

Or by path:

```
doc(action: "refresh", path: "work/design/my-document.md")
```

If the document was `approved` and the file has changed, the refresh will demote the status back to `draft` and report the transition.

---

## Safety Check Pattern

Before major commits or phase transitions:

```
doc(action: "import", path: "work")
```

This catches any documents you forgot to register individually. It's safe to run repeatedly.

---

## Common Issues

### Issue: Document ID not what you expected

**Symptom:** The document ID is different from what you anticipated.

**Cause:** Document IDs are auto-generated from the file path and type.

**Solution:** Use `doc(action: get)` to find the actual record, or list documents to see all registered IDs:

```
doc(action: "list", type: "design")
```

### Issue: Document already registered

**Symptom:** `doc(action: register)` returns an error that the document already exists.

**Cause:** The document was already registered in a previous session.

**Solution:** This is not an error. The document is already in the system. Use `doc(action: get)` to retrieve its metadata.

### Issue: Wrong document type

**Symptom:** Document registered with incorrect type (e.g., `report` instead of `design`).

**Cause:** Type parameter didn't match the directory or was explicitly set incorrectly.

**Solution:** Currently there's no type update action. The workaround is to delete the record file from `.kbz/state/documents/` and re-register with the correct type.

### Issue: Forgot to commit document record

**Symptom:** Markdown file committed but no corresponding record in `.kbz/state/documents/`.

**Cause:** Forgot step 4 of the single document procedure.

**Solution:** Register the document now and commit the record:

```
doc(action: "register", path: "work/design/my-document.md", type: "design", title: "...")
git add .kbz/state/documents/
git commit -m "workflow(my-document): register document record (missed in previous commit)"
```

### Issue: Content hash is stale after editing

**Symptom:** Health check reports stale content hashes, or `doc(action: validate)` shows hash mismatch.

**Cause:** The document file was edited after registration but the record was not refreshed.

**Solution:** Refresh the content hash:

```
doc(action: "refresh", path: "work/design/my-document.md")
```

---

## Verification

A document is properly registered when:

1. ✅ The markdown file exists in `work/`
2. ✅ A YAML file exists in `.kbz/state/documents/`
3. ✅ `doc(action: "get", path: "work/...")` returns the document metadata
4. ✅ Both files are committed to Git

---

## Related

- `AGENTS.md` — Document Creation Workflow section (rationale and policy)
- `work/bootstrap/bootstrap-workflow.md` — Document registration subsection
- `work/design/document-centric-interface.md` — Document-centric interface model