# Design: `kbz` Binary Rename and Status Command Extension

| Field   | Value                        |
|---------|------------------------------|
| Author  | Sam Phillips                 |
| Created | 2026-07-24                   |
| Status  | draft                        |

Related:

- `work/design/document-centric-interface.md` (document-centric human interface model)
- `work/design/workflow-design-basis.md` (core workflow principles)
- `work/design/fresh-install-experience.md` (init command and editor configuration)
- `work/design/interactive-help-system.md` (help tool design)

---

## 1. Purpose

This document covers two related changes:

1. **Binary rename**: rename the `kanbanzai` binary to `kbz`, clarifying the distinction between Kanbanzai the system and `kbz` the tool you type at a terminal.
2. **Status command extension**: extend `kbz status` to accept a file path or entity ID and produce a useful, human-readable summary — the primary way a human developer inspects the state of a project without invoking an AI agent.

These two changes are designed together because they share a motivation: reducing the friction between humans and the workflow system. The rename makes the tool approachable; the status extension makes it genuinely useful.

---

## 2. Problem Statement

### 2.1 The naming inconsistency

The binary is named `kanbanzai`, but the usage text it prints refers to `kbz`. The `kbz` name was reserved for a future CLI tool and never followed through. This creates a visible inconsistency: `kanbanzai --help` prints `Usage: kbz <command>`, but `kbz` does not exist on the path.

The name `kanbanzai` was chosen for the binary because it is obvious and unambiguous. This is appropriate for documentation and prose, but it is verbose to type at a terminal. The tool that sits alongside `git`, `gh`, and `go` should be `kbz`.

### 2.2 The human status gap

Kanbanzai's `status` MCP tool provides rich, synthesised dashboards for AI agents. The CLI's `kanbanzai status` command exists but is a thin stub: it runs a health check and prints a work queue count. It does not help a human answer questions like:

- "Has this design been turned into a plan?"
- "What features are in progress for this plan?"
- "What's missing before agents can start working on this feature?"
- "Has this code been reviewed?"

Humans think primarily in terms of files they can see in their editor — design documents, specifications, plans. They do not naturally think in terms of entity IDs. The current CLI offers no path from "I am looking at this file" to "what is the state of the work this file represents."

The result is that humans must ask an AI agent to look things up for them, even for simple read-only queries. That is unnecessary overhead and defeats the purpose of having a CLI.

---

## 3. Design Principles

**P-1: The default audience is a human at a terminal.** The default output is prose-like, intended to be read. Machine-readable formats are opt-in.

**P-2: Accept what humans naturally provide.** Humans know the path to a file. They should not need to find an entity ID first.

**P-3: Be consistent with the surrounding tool ecosystem.** Follow conventions established by `git`, `gh`, and `go` for flag naming, TTY detection, and output format switching.

**P-4: The command surface should be guessable.** A developer who has used `gh` or `git` should be able to guess `kbz status work/design/my-design.md` without reading the docs.

**P-5: The MCP server name is stable.** Renaming the binary does not rename the MCP server as seen by MCP clients. Tool permission configs in `.zed/settings.json` (e.g. `mcp:kanbanzai:status`) are unaffected.

**P-6: Agents should prefer MCP tools.** The CLI is not a replacement for the MCP interface for AI agents already running in an MCP session. The `--json` flag exists as an escape hatch for agents in shell-only contexts (CI, GitHub Actions), not as the primary agent interface.

---

## 4. Part A: Binary Rename (`kanbanzai` → `kbz`)

### 4.1 What changes

| Location | Current | Proposed |
|---|---|---|
| Binary name | `kanbanzai` | `kbz` |
| Source directory | `cmd/kanbanzai/` | `cmd/kbz/` |
| Makefile `BINARY` | `kanbanzai` | `kbz` |
| `go install` path | `./cmd/kanbanzai` | `./cmd/kbz` |
| `.mcp.json` template | `"command": "kanbanzai"` | `"command": "kbz"` |
| `.zed/settings.json` template | `"command": "kanbanzai"` | `"command": "kbz"` |
| `AGENTS.md` binary references | `kanbanzai serve`, `kanbanzai init` | `kbz serve`, `kbz init` |
| README and documentation | binary references to `kanbanzai` | `kbz` |
| Usage text in `main.go` | already says `kbz` | no change needed |

### 4.2 What does not change

| Location | Stays the same | Reason |
|---|---|---|
| Go module path | `github.com/sambeau/kanbanzai` | Module identity is independent of binary name |
| Internal package imports | `github.com/sambeau/kanbanzai/internal/...` | Unchanged |
| MCP server name (protocol) | `"kanbanzai"` | Changing this would invalidate all existing tool permission configs in `.zed/settings.json` |
| `.kbz/` directory name | `.kbz/` | The instance root is named after the system, not the binary |
| Prose references to "Kanbanzai" | "Kanbanzai" | The system name is not the binary name |
| `ServerName` constant in `internal/mcp/server.go` | `"kanbanzai"` | Preserves MCP tool names like `mcp:kanbanzai:status` |

The separation between the binary name (`kbz`) and the MCP server name (`kanbanzai`) is intentional and clean. A user types `kbz`. The MCP client launches `kbz serve`. The server identifies itself as `kanbanzai` over the MCP protocol. These are different namespaces and do not need to match.

### 4.3 Migration approach

No deprecation period. The rename is a clean cut. The approach is:

1. Rename `cmd/kanbanzai/` to `cmd/kbz/` and update all build references.
2. Update `kbzinit` to write the new binary name to generated MCP configs. The version bump in `mcpVersion` causes managed configs to be rewritten on next `kbz init --update-skills`.
3. Update all documentation.
4. On existing projects, existing MCP configs will have `"command": "kanbanzai"` and will break when the binary is no longer on the path. The `kbz init` command should detect this condition and print a clear message.

The `kbz init` migration check:

```
$ kbz init
Warning: .mcp.json references "kanbanzai" which is no longer installed.
  Run: kbz init --skip-skills to update editor configuration.
```

This check looks for a managed `.mcp.json` or `.zed/settings.json` that references `"command": "kanbanzai"` and prompts the user to re-run init. The user does not need to manually edit any config files.

### 4.4 Install instructions

After the rename, installation becomes:

```
go install github.com/sambeau/kanbanzai/cmd/kbz@latest
```

This is slightly unusual (the module is `kanbanzai` but the command directory is `kbz`), but it is standard Go practice. For comparison, the GitHub CLI is installed as `go install github.com/cli/cli/v2/cmd/gh@latest` — the module is `cli` but the binary is `gh`.

---

## 5. Part B: `kbz status` Extension

### 5.1 Command interface

```
kbz status [<target>] [--format <fmt>]
```

`<target>` can be:

- **Omitted** — project overview (existing behaviour, extended)
- **A file path** — e.g. `work/design/my-feature.md`; resolves to a registered document then to its owner entity
- **An entity ID** — e.g. `FEAT-042`, `P1-my-plan`, `BUG-007`; full or display-format IDs are both accepted
- **A plan prefix** — e.g. `P1` resolves to the plan with that prefix

Resolution order when the target is ambiguous:

1. If it contains a `/` or ends in a recognised document extension (`.md`, `.txt`) → treat as file path
2. If it matches a known entity ID pattern → treat as entity ID
3. Otherwise → attempt entity ID lookup, then file path lookup, then error

The `--format` flag controls output mode (see §5.4). It accepts `human` (default), `plain`, and `json`.

No other flags are added. The command is intentionally simple.

#### Exit codes

`kbz status` is a query tool. Exit codes follow the Unix query-tool convention (`git status`, `find`), not the assertion-tool convention (`test`, `grep`):

| Situation | Exit code |
|---|---|
| Query succeeded — any result, including "not registered" | 0 |
| File does not exist on disk / entity ID not found | 1 |
| System error (cannot read state store) | 1 |
| Usage error (bad flag, malformed argument) | 2 |

"Not registered" is a valid query result, not a failure. Scripts that need to assert registration should check the output, not the exit code:

```sh
kbz status "$f" --format plain | grep -q '^registered: false' && echo "unregistered: $f"
```

The exit code of `grep -q` does the assertion work. This is the same pattern used with `git status --porcelain` in shell scripts.

### 5.2 File path resolution

When a file path is given:

1. Look up the path in the document record store (exact match, repo-relative).
2. If no record is found, show an "unregistered document" view: the file exists but has not been registered with `kbz doc register`. Include a suggested command.
3. If a record is found, show the document view (§5.5.1), then pivot to the owner entity view (§5.5.2 or §5.5.3) if one exists.

The intent is that the file path is the entry point humans naturally use. The document record is just the bridge to the entity model.

### 5.3 Path resolution for `kbz doc approve` and `kbz doc register`

As a related fix: `kbz doc approve` currently requires an internal document ID (e.g. `DOC-0012`). This is unusable in practice — humans do not know document IDs. Both `kbz doc approve` and `kbz doc list` should accept file paths as an alternative to IDs.

```
kbz doc approve work/design/my-feature.md   # path → ID resolution
kbz doc approve DOC-0012                     # ID still works
```

This is a small change but meaningfully lowers the barrier to the approval workflow.

### 5.4 Output formats

Three modes, selected by `--format` (short: `-f`):

#### `human` (default)

Rich prose output intended to be read by a person at a terminal. When stdout is a TTY, uses Unicode symbols (✓ ✗ ⚠ ●) and ANSI colour. When stdout is not a TTY (piped), suppresses colour and uses ASCII fallbacks (`[ok]`, `[missing]`, `[warn]`, `[*]`) — the same content, plainer presentation. This follows the convention established by `git` and `gh`.

The TTY detection is automatic. Users do not need to pass a flag to get clean output when piping.

#### `plain` (`--format plain`)

Stable `key: value` pairs, one per line. No symbols, no colour, no alignment padding. Consistent and greppable. Follows the model of `git show --format=...` and what `git` calls "porcelain" output. Suitable for shell scripts and pre-commit hooks.

```
scope: feature
id: FEAT-042
slug: my-feature
status: developing
plan: P1-my-plan
doc.design: work/design/my-feature.md
doc.design.status: approved
doc.spec: work/spec/my-feature-spec.md
doc.spec.status: approved
doc.dev-plan: missing
tasks.active: 1
tasks.ready: 3
tasks.done: 7
tasks.total: 11
attention: no dev-plan document registered
```

The key schema is fixed and versioned. New keys may be added but existing keys will not be renamed or removed within a major version.

#### `json` (`--format json`)

Full structured output. Suitable for AI agents in shell-only contexts, CI scripts, and tools that need to process status programmatically. The JSON schema mirrors the data model used internally by the MCP `status` tool, so the output is familiar to anyone who has read the MCP responses.

Entity and document queries are always wrapped in a top-level `results` array, even for a single target. This makes the schema forward-compatible with multi-target queries (see §9, Q-1) without a breaking change — consumers always iterate `results`, regardless of how many targets were given.

```json
{
  "results": [
    {
      "scope": "feature",
      "feature": {
        "id": "FEAT-042",
        "display_id": "F-042",
        "slug": "my-feature",
        "status": "developing",
        "plan_id": "P1-my-plan"
      },
      "documents": {
        "design": { "id": "DOC-0019", "path": "work/design/my-feature.md", "status": "approved" },
        "spec":   { "id": "DOC-0023", "path": "work/spec/my-feature-spec.md", "status": "approved" },
        "dev-plan": null
      },
      "tasks": { "active": 1, "ready": 3, "done": 7, "total": 11 },
      "attention": [
        { "severity": "warning", "message": "No dev-plan document registered — agents cannot begin planning" }
      ]
    }
  ]
}
```

The project overview (no target) is always a single result and uses a distinct top-level shape — `scope: "project"` with summary counts, not a `results` array:

```json
{
  "scope": "project",
  "plans": [
    { "id": "P1-main-plan", "slug": "main-plan", "status": "developing", "features": { "active": 2, "done": 3, "total": 5 } }
  ],
  "health": { "errors": 0, "warnings": 2 },
  "attention": [
    { "severity": "warning", "entity_id": "FEAT-042", "message": "No dev-plan document registered" }
  ]
}
```

Plans include feature counts but not full feature lists. To get full feature detail for a plan, query the plan directly: `kbz status P1 --format json`. This keeps the project overview small and fast regardless of project size.

### 5.5 Output content

What the command shows depends on what `<target>` resolves to.

#### 5.5.1 Unregistered document

When a file path is given but the file is not in the document record store:

```
work/design/my-feature.md
  Not registered with Kanbanzai.

  To register:
    kbz doc register work/design/my-feature.md --type design --title "My Feature"
```

Short and actionable. No attempt to guess ownership or show entity data.

#### 5.5.2 Registered document (with owner entity)

When a file path resolves to a registered document that belongs to a feature:

```
work/design/my-feature.md
  Title:   My Feature Design
  Type:    design
  Status:  approved

Feature  F-042 · my-feature                              developing
  Plan:  P1-my-plan · main-plan

  Documents
    Design:    ✓  work/design/my-feature.md         approved
    Spec:      ✓  work/spec/my-feature-spec.md      approved
    Dev plan:  ✗  missing

  Tasks  ● 1 active · 3 ready · 7 done  (11 total)

  ⚠  No dev-plan document — agents cannot begin planning
     Suggested: kbz doc register work/dev-plan/... --type dev-plan --owner FEAT-042
```

The document block is shown first because it is what the user asked about. The entity block follows as context, using the same layout as a direct entity lookup.

#### 5.5.3 Direct feature lookup

```
$ kbz status FEAT-042

Feature  F-042 · my-feature                              developing
  Plan:  P1-my-plan · main-plan

  Documents
    Design:    ✓  work/design/my-feature.md         approved
    Spec:      ✓  work/spec/my-feature-spec.md      approved
    Dev plan:  ✗  missing

  Tasks  ● 1 active · 3 ready · 7 done  (11 total)

  ⚠  No dev-plan document — agents cannot begin planning
     Suggested: kbz doc register work/dev-plan/... --type dev-plan --owner FEAT-042
```

#### 5.5.4 Plan lookup

```
$ kbz status P1-main-plan

Plan  P1-main-plan · main-plan                           developing

  Features (5)
    F-039  spec-doc-gaps            ✓ done
    F-040  lifecycle-gate-hardening ✓ done
    F-041  binary-rename            ● developing
    F-042  my-feature               ● developing
    F-043  cli-status-command       ○ ready

  Tasks  ● 4 active · 6 ready · 31 done  (51 total)

  ⚠  F-042 my-feature: no dev-plan document
```

#### 5.5.5 Project overview (no target)

The existing project overview is extended to include a document summary section showing any registered documents missing an owner, any draft documents pending approval, and a count of features by lifecycle stage.

```
$ kbz status

Kanbanzai  my-project

  Plans (2)
    P1-main-plan       ● developing   4 features active
    P2-infrastructure  ○ ready        0 features started

  Health  ✓ no errors · 2 warnings

  ⚠  F-042 my-feature: no dev-plan document
  ⚠  DOC-0031 work/design/new-draft.md: draft — not yet approved

  Work queue  6 ready · 2 active
```

### 5.6 Visual conventions

| Symbol | Meaning | ASCII fallback |
|--------|---------|---------------|
| `✓` | present and approved | `[ok]` |
| `✗` | missing | `[missing]` |
| `⚠` | attention item / warning | `[warn]` |
| `●` | active / in progress | `[*]` |
| `○` | ready / queued | `[ ]` |
| `·` | separator in counts | `-` |

Colour (TTY only):
- Green: done, approved, healthy
- Yellow: active, in-progress, warnings
- Red: errors, missing required items
- Default: ready/queued items, neutral labels

Alignment: document rows in the same block are aligned to a common column for readability. No alignment across blocks.

Width: lines are not artificially wrapped. The terminal handles wrapping. Content is written to be readable at 80 columns but does not enforce it.

---

## 6. Non-Human Use Cases

### 6.1 AI agents in MCP sessions

Agents running inside an active MCP session should continue to use the `status` MCP tool, not `kbz status`. The MCP tool returns structured JSON directly, requires no subprocess invocation, and has richer data in some cases (attention item detail, document intelligence links). The CLI `--json` output is not a replacement.

### 6.2 AI agents in shell-only contexts

Some agent environments (GitHub Copilot coding agent, scripts invoked by CI) do not have an MCP session but do have shell access. In these contexts, `kbz status --format json` is the correct interface. The JSON output is designed to be stable and self-describing.

A shell-based agent can run:

```sh
kbz status FEAT-042 --format json | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['feature']['status'])"
```

Or, more practically, read the full JSON blob and reason about it. The JSON schema is documented (§5.4) and versioned.

### 6.3 Shell scripts and pre-commit hooks

`--format plain` is the right format for shell scripts that need to grep or awk the output. The `key: value` structure makes extraction straightforward:

```sh
status=$(kbz status FEAT-042 --format plain | grep '^status:' | cut -d' ' -f2)
```

A pre-commit hook could check that all design documents in `work/design/` are registered. Because `kbz status` exits 0 even for unregistered documents (it is a query tool, not an assertion), the check is done on the output:

```sh
for f in work/design/*.md; do
  kbz status "$f" --format plain 2>/dev/null | grep -q '^registered: false' && {
    echo "Unregistered document: $f"
    exit 1
  }
done
```

The plain output for an unregistered document includes `registered: false`. For a registered document this key is omitted (absence means registered). Scripts may also check `registered: true` explicitly if they prefer symmetry.

### 6.4 CI and reporting

`kbz status --format json` with no target gives a project overview suitable for CI gate-checking and lightweight dashboards. It returns summary counts and attention items — not full feature lists. This keeps the output small and fast regardless of project size.

The `health` block provides the CI gate check:

```sh
errors=$(kbz status --format json | jq '.health.errors')
if [ "$errors" -gt 0 ]; then exit 1; fi
```

For a richer dashboard (feature names, statuses, document gaps per feature), query at the plan level instead:

```sh
kbz status P1 --format json | jq '.results[0].features[]'
```

This two-level design mirrors how a human uses the tool: project overview first, then drill into a specific plan. It also avoids the problem of a project-level JSON response growing without bound as the number of features increases.

---

## 7. Implementation Notes

### 7.1 Service layer reuse

The data for `kbz status` output already exists in the service layer. `internal/mcp/status_tool.go` contains `synthesiseProject`, `synthesisePlan`, `synthesiseFeature`, `synthesiseTask`, and `synthesiseBug` — these functions assemble the structured data currently returned by the MCP `status` tool. Rather than duplicating this logic, the CLI `status` command should call through to the same service-layer functions. The CLI adds a rendering layer (human prose, plain key-value, JSON serialisation) on top of the same underlying data structures.

This means the CLI and MCP tool stay in sync automatically: any new attention item or doc-gap check added to the service layer appears in both outputs.

### 7.2 File path lookup

The document service (`service.DocumentService`) already supports lookup by path via `ListDocuments` with a path filter. The status command needs a `GetDocumentByPath(path string)` convenience method if one does not already exist.

### 7.3 TTY detection

Use `golang.org/x/term` or the equivalent to detect whether stdout is a terminal. This is the same approach used by `gh`. Do not require the user to pass `--no-color` or `--plain` to get clean output when piping — detect it automatically.

### 7.4 Scope of this design

This design covers:

- The binary rename (§4)
- `kbz status [<target>] [--format <fmt>]` (§5)
- `kbz doc approve <path-or-id>` (§5.3)

It does not cover:

- A `kbz doc show` command (separate concern)
- `kbz entity show` or similar (separate concern)
- A TUI (terminal UI) — out of scope
- New MCP tools mirroring the CLI (would be duplication with no benefit)

---

## 8. Decisions

### D-1: One binary, renamed

The binary is renamed from `kanbanzai` to `kbz`. There is no second binary, no symlink, and no alias. The Go module path (`github.com/sambeau/kanbanzai`) is unchanged. The MCP server name (`kanbanzai`) is unchanged.

**Rationale:** Two binaries would add distribution complexity and real maintenance overhead for cosmetic separation. The shared internal library already provides the right factoring. The naming confusion is solved by the rename, not by splitting.

### D-2: No deprecation period

The rename is a clean cut. There is no `kanbanzai` compatibility shim and no grace period. Existing projects get a migration prompt on the next `kbz init` run.

**Rationale:** A shim adds complexity and teaches users the wrong name. The system manages its own development (dogfooding) so the breakage surface is controlled and can be fixed in one pass.

### D-3: MCP server name is stable

`ServerName = "kanbanzai"` in `internal/mcp/server.go` does not change. The binary name and the MCP protocol name are independent.

**Rationale:** Changing the MCP server name would invalidate all existing `.zed/settings.json` tool permission configs, silently breaking per-tool approval rules for every existing user. The cost is not justified.

### D-4: Default output is human prose; machines opt in

`--format human` is the default. `--format plain` and `--format json` are opt-in. TTY detection controls colour and symbols within the `human` format.

**Rationale:** The primary user of `kbz status` is a human at a terminal. Defaulting to a machine-readable format would make the default output worse for the common case in order to simplify the uncommon case. This is the wrong trade-off.

### D-5: Agents prefer MCP tools; CLI JSON is for shell-only contexts

The `--format json` output is designed for agents that do not have an MCP session available. Agents with MCP access should continue to use the `status` MCP tool.

**Rationale:** MCP tools return richer data in context-aware formats and require no subprocess overhead. Duplicating the full MCP status response in the CLI JSON would add maintenance burden. The CLI JSON is a useful escape hatch, not the primary agent interface.

### D-6: `kbz status` is a query tool; exit 0 for all successful queries

`kbz status` exits 0 for any result that is a valid answer to the query, including "not registered" and "not found in entity store." It exits non-zero only when the query itself could not be completed (file missing from disk, system error, usage error).

**Rationale:** This follows the Unix query-tool convention (`git status`, `find`) rather than the assertion-tool convention (`test`, `grep`). The command answers a question; it does not make a claim. Scripts that need assertions should check the output, not the exit code.

### D-7: JSON schema is always array-wrapped for entity/document queries

Entity and document queries return `{"results": [...]}` even for a single target. The project overview (no target) uses a distinct `{"scope": "project", ...}` shape and is always singular.

**Rationale:** Multi-target support (Q-1) may be added in a future version. Using a `results` array from the start means multi-target is a non-breaking addition — existing consumers iterate `results[0]` today and `results[n]` tomorrow with no schema change. The minor verbosity cost for single-target queries is worth the forward-compatibility.

### D-8: Project overview JSON contains summary counts, not full feature lists

The project overview JSON (`kbz status --format json`) includes plan summaries with feature counts and a flat attention item list. Full feature detail is available by querying the plan directly (`kbz status P1 --format json`).

**Rationale:** Full feature lists at the project level grow without bound as projects scale. The common uses of project-level JSON (CI gate-checking, high-level dashboards) need counts and attention items, not individual feature data. Consumers that need full feature detail know which plan to query.

---

## 9. Open Questions

### Q-1: Should `kbz status` accept multiple targets?

```
kbz status work/design/a.md work/design/b.md
```

Shell scripts iterating over a glob of files would benefit from this — a single subprocess call rather than a loop. AI agents in shell-only contexts assessing several features' readiness would also benefit. Not required for the initial implementation, but the JSON schema is designed for it from the start (D-7): consumers iterate `results[n]` and the addition of multi-target support is non-breaking. Human (prose) output would need a clear visual delimiter between results. Deferred to a follow-on iteration.

### Q-2: Should there be a `kbz status --watch` mode?

A polling mode that re-runs the status query on a short interval and clears the terminal, useful for monitoring a long-running agent session. This is a UX nicety, not a core need, and is deferred.

### Q-3: `kbz doc approve` path resolution scope

§5.3 proposes that `kbz doc approve` accepts a file path. The resolution logic (path → document record → ID) is straightforward. However, should it also support partial title matching (e.g. `kbz doc approve "My Feature Design"`)? Title matching is fuzzy and could be surprising. The recommendation is path and ID only; title matching is out of scope.

---

## 10. Summary

Two tightly related changes that reduce friction for human developers:

1. **Rename `kanbanzai` → `kbz`**: clean cut, no deprecation, MCP server name unchanged, migration prompt in `kbz init`. Short, typeable, consistent with what the usage text already says.

2. **Extend `kbz status`**: accept a file path or entity ID; resolve paths through the document record to the owner entity; show a human-readable summary with document gaps and attention items; support `--format plain` for shell scripts and `--format json` for agents in shell-only contexts. Build on the existing service-layer synthesis functions rather than duplicating logic.

Together these make `kbz status work/design/my-feature.md` the natural first thing a developer types when they want to know where a piece of work stands — no AI required.