# SKILL: Document Creation

## Purpose

Register new documents with the kanbanzai system when creating markdown files in the `work/` directory.

## When to Use

- After creating any new `.md` file in `work/design/`, `work/spec/`, `work/plan/`, `work/research/`, or `work/reports/`
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
```

### Step 2: Register with the system

Immediately call `doc_record_submit`:

```
doc_record_submit(
  path="work/design/my-document.md",
  type="design",
  title="Human-Readable Title",
  created_by="your-agent-name"
)
```

**Type must match location:**

| Location | Type |
|----------|------|
| `work/design/` | `design` |
| `work/spec/` | `specification` |
| `work/plan/` | `dev-plan` |
| `work/research/` | `research` |
| `work/reports/` | `report` |

The `created_by` parameter is optional and will auto-resolve from `.kbz/local.yaml` or `git config` if omitted.

### Step 3: Verify registration

Check that the document record was created:

```
doc_record_get(id="PROJECT/design-my-document")
```

The document ID follows the pattern: `PROJECT/{type}-{slug}` where the slug is derived from the filename.

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
batch_import_documents(
  path="work/plan",
  default_type="dev-plan",
  created_by="your-agent-name"
)
```

Or import all of `work/`:

```
batch_import_documents(
  path="work",
  created_by="your-agent-name"
)
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

## Safety Check Pattern

Before major commits or phase transitions:

```
batch_import_documents(path="work")
```

This catches any documents you forgot to register individually. It's safe to run repeatedly.

---

## Common Issues

### Issue: Document ID not what you expected

**Symptom:** The document ID is different from what you anticipated.

**Cause:** Document IDs are auto-generated from the file path and type.

**Solution:** Use `doc_record_get` to find the actual ID, or list documents to see all registered IDs:

```
doc_record_list(type="design")
```

### Issue: Document already registered

**Symptom:** `doc_record_submit` returns an error that the document already exists.

**Cause:** The document was already registered in a previous session.

**Solution:** This is not an error. The document is already in the system. Use `doc_record_get` to retrieve its metadata.

### Issue: Wrong document type

**Symptom:** Document registered with incorrect type (e.g., `report` instead of `design`).

**Cause:** Type parameter didn't match the directory or was explicitly set incorrectly.

**Solution:** Currently there's no `doc_record_update_type` tool. The workaround is to delete the record file from `.kbz/state/documents/` and re-register with the correct type. (Future enhancement: add update tool.)

### Issue: Forgot to commit document record

**Symptom:** Markdown file committed but no corresponding record in `.kbz/state/documents/`.

**Cause:** Forgot step 4 of the single document procedure.

**Solution:** Register the document now and commit the record:

```
doc_record_submit(path="work/design/my-document.md", type="design", title="...")
git add .kbz/state/documents/
git commit -m "workflow(my-document): register document record (missed in previous commit)"
```

---

## Verification

A document is properly registered when:

1. ✅ The markdown file exists in `work/`
2. ✅ A YAML file exists in `.kbz/state/documents/PROJECT--{type}-{slug}.yaml`
3. ✅ `doc_record_get(id="PROJECT/{type}-{slug}")` returns the document metadata
4. ✅ Both files are committed to Git

---

## Related

- `AGENTS.md` — Document Creation Workflow section (rationale and policy)
- `work/bootstrap/bootstrap-workflow.md` — Document registration subsection
- `work/design/document-centric-interface.md` — Document-centric interface model