# Default Context Roles Specification

| Document | Default Context Roles Specification                              |
|----------|------------------------------------------------------------------|
| Status   | Draft                                                            |
| Feature  | `FEAT-01KMWJ3ZQZF5R`                                            |
| Related  | `work/design/fresh-install-experience.md` §5.3, FI-D-003        |
|          | `work/spec/init-command.md`                                      |
|          | `work/spec/reviewer-context-profile-and-skill.md`               |

---

## 1. Purpose

This specification defines the acceptance criteria for Feature C of the P11 Fresh Install
Experience plan: the default context role files installed by `kbz init`.

`kbz init` currently installs skill files into `.agents/skills/` but does not install any
context role files. As a result, a freshly initialised project cannot resolve
`role="reviewer"` — the reviewer profile must be created manually by each project. This is
an avoidable setup burden: the reviewer role has no project-specific content and should be
available everywhere kanbanzai is used.

This feature introduces two role files to `kbz init`:

1. **`base.yaml`** — a scaffold profile installed as a starting point for project
   conventions. It is intentionally empty of system content and is owned by the project
   team, not managed by kanbanzai.
2. **`reviewer.yaml`** — a fully populated profile installed as a managed file. It carries
   the `kanbanzai-managed` marker, follows the same version-aware update logic as skill
   files, and is refreshed by `kbz init --update-skills`.

The feature also introduces a `--skip-roles` flag and extends the `--update-skills` flag to
cover managed role files. The broader name `--update-managed` was considered and rejected;
the flag retains its existing name with expanded scope.

---

## 2. Scope

### 2.1 In scope

- `base.yaml` scaffold: file content, inline comments, non-interference on existing project
- `reviewer.yaml` managed role: canonical embedded content, managed marker, version-aware
  install and update logic
- Intentional absence of `developer.yaml`
- `--skip-roles` flag
- `--update-skills` flag extended to update managed role files
- `profile` resolution: a freshly initialised project can resolve `id="reviewer"` to a
  non-empty context packet

### 2.2 Out of scope

- `developer.yaml` — intentionally not installed; see FI-D-003
- Any managed role files beyond `reviewer.yaml`
- MCP server connection files (Feature A)
- Skills consolidation (Feature B)
- Standard document layout (Feature D)
- Changes to the `--skip-skills` flag

---

## 3. Canonical File Content

### 3.1 `base.yaml`

The scaffold installed by `kbz init`. It is owned by the project team and not managed by
kanbanzai. The project owner fills in their conventions directly; the comments explain the
schema.

```kanbanzai/.kbz/context/roles/base.yaml
id: base
description: "Project-wide conventions for all agents"
# Add your project's global conventions here.
# All other roles inherit from base unless they declare their own inherits field.
conventions: []
# architecture:
#   summary: "One paragraph describing the overall project structure"
#   key_interfaces:
#     - "The most important files/packages and what they do"
```

`base.yaml` does **not** carry a `metadata.kanbanzai-managed` field. It is a scaffold for
the user to own.

### 3.2 `reviewer.yaml`

The fully populated managed file installed by `kbz init`. The `metadata.version` field is
set to the binary's version string at build time (e.g. `"1.0.0"` for a release build,
`"dev"` during development). The body content below is canonical; it must match the content
embedded in the binary exactly.

```kanbanzai/.kbz/context/roles/reviewer.yaml
id: reviewer
inherits: base
description: "Context profile for code review agents. Provides review dimensions, structured output format, and quality gate criteria."
metadata:
  kanbanzai-managed: "true"
  version: "1.0.0"
conventions:
  review_approach:
    - "Review is structured, not conversational. Produce findings, not commentary."
    - "Every finding has a dimension, severity, location, and description."
    - "Blocking findings must cite the specific requirement or convention violated."
    - "Non-blocking findings are suggestions, not demands."
    - "When uncertain whether something is a defect, classify as concern, not fail."
  output_format:
    - "Use the structured review output format from the kanbanzai-review skill."
    - "Report per-dimension outcomes: pass, pass_with_notes, concern, fail, not_applicable."
    - "Report overall verdict: approved, approved_with_followups, changes_required, blocked."
    - "List blocking findings separately from non-blocking notes."
  dimensions:
    - "Specification conformance: does the implementation match the approved spec?"
    - "Implementation quality: is the code correct, idiomatic, and maintainable?"
    - "Test adequacy: are tests appropriate, sufficient, and well-structured?"
    - "Documentation currency: is documentation accurate and up to date?"
    - "Workflow integrity: is the workflow state consistent with the work?"
```

---

## 4. Acceptance Criteria

### 4.1 `base.yaml`

**4.1.1** `kbz init` on a new project creates `.kbz/context/roles/base.yaml`.

**4.1.2** The installed `base.yaml` contains `id: base`, a `description` field, and a
`conventions: []` field.

**4.1.3** `base.yaml` contains inline YAML comments directing the project owner to fill in
conventions and architecture. At minimum the file includes: a comment explaining that all
other roles inherit from `base`, and a commented-out `architecture` block showing the
`summary` and `key_interfaces` sub-keys.

**4.1.4** `base.yaml` does not contain a `metadata.kanbanzai-managed` field. It is a user-
owned scaffold, not a managed file.

**4.1.5** If `.kbz/context/roles/base.yaml` already exists at init time, `kbz init` leaves
it untouched — regardless of content. No overwrite, no patch, no warning is emitted.

### 4.2 `reviewer.yaml`

**4.2.1** `kbz init` on a new project creates `.kbz/context/roles/reviewer.yaml` with the
canonical embedded content defined in §3.2.

**4.2.2** The installed `reviewer.yaml` declares `id: reviewer` and `inherits: base`.

**4.2.3** `reviewer.yaml` contains `metadata.kanbanzai-managed: "true"` and a
`metadata.version` field set to the binary's version string.

**4.2.4** `reviewer.yaml` contains a `conventions` block with exactly three named sub-keys:
`review_approach`, `output_format`, and `dimensions`.

**4.2.5** The `output_format` conventions reference the `kanbanzai-review` skill as the
procedure to follow for review output format. The reference must be to the skill by name
(`kanbanzai-review`), not to a file path.

**4.2.6** If `.kbz/context/roles/reviewer.yaml` already exists at init time without a
`metadata.kanbanzai-managed` field, `kbz init` skips it and prints a warning to stderr that
names the file and states it was skipped because it is not managed.

**4.2.7** If `.kbz/context/roles/reviewer.yaml` already exists with
`metadata.kanbanzai-managed: "true"` and a version string that is older than the binary's
embedded version, `kbz init` overwrites the file with the current canonical content.

**4.2.8** If `.kbz/context/roles/reviewer.yaml` already exists with
`metadata.kanbanzai-managed: "true"` and a version string equal to the binary's embedded
version, `kbz init` makes no change to that file.

### 4.3 `developer.yaml`

**4.3.1** `kbz init` does not create `.kbz/context/roles/developer.yaml` on a new project.
After a clean `kbz init`, the file must be absent.

### 4.4 Flags

**4.4.1** `kbz init --skip-roles` skips creation of all role files. Neither
`.kbz/context/roles/base.yaml` nor `.kbz/context/roles/reviewer.yaml` is created.

**4.4.2** `kbz init --skip-roles` on a new project exits with status 0. Absence of role
files is not an error condition.

**4.4.3** `kbz init --update-skills` updates `.kbz/context/roles/reviewer.yaml` if the file
carries `metadata.kanbanzai-managed: "true"` and its version is older than the binary's
embedded version. The update follows the same logic as AC 4.2.7.

**4.4.4** `kbz init --update-skills` does not touch `.kbz/context/roles/base.yaml`,
regardless of its content. `base.yaml` carries no managed marker and is never subject to
automatic update.

### 4.5 Profile resolution

**4.5.1** On a freshly initialised project — immediately after `kbz init` with no additional
configuration — calling `profile` with `action: get` and `id: "reviewer"` (or equivalent
`context_assemble(role="reviewer")`) returns a non-empty context packet containing all three
convention keys: `review_approach`, `output_format`, and `dimensions`.

---

## 5. Verification

| # | Check | Method |
|---|-------|--------|
| V1 | `base.yaml` created on new project | `kbz init` in a temp git repo; verify `.kbz/context/roles/base.yaml` exists |
| V2 | `base.yaml` has correct fields | Inspect file: `id: base`, `description`, `conventions: []` |
| V3 | `base.yaml` has required comments | Inspect file: comment referencing `inherits`, commented `architecture` block |
| V4 | `base.yaml` has no managed marker | Inspect file: no `metadata.kanbanzai-managed` field |
| V5 | `base.yaml` not overwritten on existing project | Pre-create `base.yaml` with custom content; re-run `kbz init`; verify content unchanged |
| V6 | `reviewer.yaml` created on new project | `kbz init` in a temp git repo; verify `.kbz/context/roles/reviewer.yaml` exists |
| V7 | `reviewer.yaml` has correct identity fields | Inspect file: `id: reviewer`, `inherits: base` |
| V8 | `reviewer.yaml` has managed marker and version | Inspect file: `metadata.kanbanzai-managed: "true"`, `metadata.version` present |
| V9 | `reviewer.yaml` has all three convention sub-keys | Inspect file: `review_approach`, `output_format`, `dimensions` all present |
| V10 | `reviewer.yaml` references `kanbanzai-review` skill | Inspect `output_format` list: contains the string `kanbanzai-review` |
| V11 | `reviewer.yaml` skipped with warning when unmanaged | Pre-create `reviewer.yaml` without managed marker; run `kbz init`; verify file unchanged; verify stderr contains warning naming the file |
| V12 | `reviewer.yaml` overwritten when managed at older version | Pre-create `reviewer.yaml` with managed marker and a lower version; run `kbz init`; verify content updated to current canonical |
| V13 | `reviewer.yaml` no-op when at current version | Pre-create `reviewer.yaml` at current version; run `kbz init`; verify file unchanged |
| V14 | `developer.yaml` absent after init | `kbz init` in a temp git repo; verify `.kbz/context/roles/developer.yaml` does not exist |
| V15 | `--skip-roles` skips both files | `kbz init --skip-roles`; verify neither `base.yaml` nor `reviewer.yaml` created |
| V16 | `--skip-roles` exits cleanly | Verify exit status 0 with `--skip-roles` on a new project |
| V17 | `--update-skills` updates managed `reviewer.yaml` | Pre-create `reviewer.yaml` with managed marker at older version; run `kbz init --update-skills`; verify file updated |
| V18 | `--update-skills` ignores `base.yaml` | `kbz init --update-skills`; verify `base.yaml` content unchanged |
| V19 | Profile resolution works after init | `profile action:get id:reviewer` on fresh install; verify non-empty packet with `review_approach`, `output_format`, `dimensions` |