# Design: Entity Names

- Status: proposal
- Date: 2026-06-17
- Author: orchestrator
- Related: `work/design/human-friendly-ids.md`
- Supersedes: `work/design/titles-proposal.txt`

---

## 1. Problem

Entity display names are inconsistent and produce poor results in practice.

**1.1 — The field is called `title` on some entities and missing on others.**

Four entity types have a `title` field (Plan, Epic, Bug, Incident). Three have
no dedicated display name at all (Feature, Task, Decision) — their `summary`
field doubles as a description *and* a display name, which serves neither role
well. Summary fields on Features are frequently full paragraphs that are
unreadable as display labels.

**1.2 — The name `title` produces verbose, formal output.**

`title` carries academic and document associations. In practice, Plan titles in
this repo — the entities where the problem is most visible — read like paper
titles: "P4 Kanbanzai 2.0: MCP Tool Surface Redesign", "P8 — decompose propose
Reliability Fixes". These are the bad examples. They carry phase prefixes,
colons, version numbers, and em-dashes used as separators. The field name
`title` is inviting this. Agents filling a `title` field think "how would I
formally describe this thing?" rather than "what is this thing called?"

**1.3 — The `label` field on Feature and Task was never adopted.**

The `label` field (max 24 chars, optional) was added in P7 to give features and
tasks a short human-navigation string. As of this writing, zero entities in the
`.kbz` store carry a label value. The concept was sound — a short, human-readable
display identifier — but the field never became ergonomic enough to use.

**1.4 — The project itself has no name.**

`.kbz/config.yaml` carries version and prefix configuration but no project name
or title. The viewer and any tool that needs to display "which project am I
looking at?" has no canonical source for this.

---

## 2. Proposed Changes

Three changes, applied together in a single implementation pass.

### Change 1 — Rename `title` → `name` on all entity types

Replace the `title` field with `name` across all entity types that currently
have it: Plan, Epic, Bug, Incident. Apply the same field to the three types that
currently lack a dedicated display name: Feature, Task, Decision.

The result is a single, consistent `name` field on every entity type.

**Why `name` and not `title`:**

`name` carries identity associations. Names are short, distinct, and singular —
they answer "what is this thing called?" not "how would I formally describe it?"
This is a meaningful nudge for AI agents filling the field. `title` is already
producing the verbose, structured strings the system should not have.

**Field position in canonical YAML field order:**

`name` sits immediately after `slug`, before `status`, on all entity types.
This places the human-readable identifier adjacent to the machine-readable
identifier, and before lifecycle state — reflecting the reading priority: what
is it, then what state is it in.

### Change 2 — Retire `label` on Feature and Task

The `label` field is removed from Feature and Task. Its intended role — a short,
human-readable display identifier — is now served by `name`. Since no entities
in the store carry a label value, this is a pure schema simplification with no
data migration cost.

The `label` parameter is removed from the `entity` MCP tool.

### Change 3 — Add `name` to `config.yaml` and `kbz init`

Add a `name` field to `.kbz/config.yaml` to give the project itself a display
name. `kbz init` prompts for this interactively, with a sensible default derived
from the current directory name. A `--name` flag accepts it non-interactively.

```
version: "2"
name: Kanbanzai
prefixes:
  - prefix: P
    name: Plan
```

---

## 3. Name Rules

### 3.1 Hard rules (validated in code, errors returned)

| Rule | Rationale |
|------|-----------|
| Required, non-empty | Every entity must be nameable. |
| Maximum 60 characters | Keeps names short; still allows a few words of context. |
| No leading or trailing whitespace | Normalise silently on input. |
| No colon (`:`) | Colons signal "title: subtitle" structure. Reject at the boundary. |
| No leading phase/version prefix | Pattern `^[A-Z]\d+[\s\-—]` rejects "P4 ...", "P8 — ...". |

### 3.2 Skill guidance (enforced through agent instructions, not code)

- **Target ~4 words.** Not a hard limit, but the right mental model. If a name
  takes more than four or five words, the scope is probably too broad or the
  name is doing the summary's job.
- **No em-dashes or separator punctuation.** Hyphens in compound technical
  terms are fine ("Human-friendly ID display"). Dashes used as separators
  ("P8 — decompose") are not.
- **Name ≠ capitalised slug.** The slug `init-command` becomes "Init command"
  capitalised. A name should add something: "Project init command" or
  "Init and skill install". If the slug is already a perfect summary, the name
  can match it — but make the choice deliberately.
- **Self-contained.** A name should be readable without knowing the parent
  entity. "Update agents" is ambiguous; "Update AGENTS.md layout" is not.
- **No redundant hierarchy context.** The parent is already encoded in the
  entity's `parent` field and displayed by the viewer. Don't include the plan
  name or phase in the entity name.

### 3.3 Examples

**Good ✅**

| Entity | Name |
|--------|------|
| Plan | Kanbanzai 2.0 |
| Feature | Human-friendly ID display |
| Feature | Init and skill install |
| Feature | Specification skill |
| Task | Label model and storage |
| Task | Server info tool |
| Bug | Nil pointer in entity store |
| Decision | Use TSID for entity IDs |

**Bad ❌**

| Entity | Name | Problem |
|--------|------|---------|
| Plan | P4 Kanbanzai 2.0: MCP Tool Surface Redesign | Phase prefix, colon, too long |
| Feature | P8 — decompose propose Reliability Fixes | Phase prefix, separator dash |
| Feature | The kanbanzai init command: creates .kbz/config.yaml, installs Kanbanzai-managed skill files | This is a summary, not a name |
| Task | Update | Too vague; not self-contained |

---

## 4. Migration

### 4.1 Existing entities in this repo

Every existing entity that has a `title` field needs that field renamed to
`name`. Every Feature, Task, and Decision needs a `name` added. This is a
backfill pass, not a schema migration — the YAML files are plain text and the
change is mechanical.

The backfill should be performed as part of the implementation task, after the
model and storage changes are complete. Names should be written to be consistent
with the rules in §3 — this is an opportunity to correct the existing bad
examples. Short, clear names derived from slugs and summaries, written by an
agent following the skill guidance, reviewed by a human.

### 4.2 Backward compatibility

Existing YAML files without a `name` field are technically invalid after this
change. The storage layer should treat a missing `name` as a validation warning
(not a hard error) during a grace period, to avoid breaking reads before the
backfill is complete. Once the backfill is done, the grace period can be removed.

The `title` parameter on the `entity` MCP tool (used for Plan, Epic, Bug,
Incident creation) is renamed to `name`. This is a breaking change to the tool
schema; agent skill files should be updated in the same pass.

---

## 5. Scope

This change is intentionally limited to naming consistency. Out of scope:

- Viewer implementation (the viewer will benefit from consistent `name` fields,
  but viewer changes are a separate feature)
- Semantic search or name-based entity lookup beyond the existing `slug` and
  `label` filter mechanisms
- Name uniqueness enforcement (names are display strings, not identifiers; two
  features can have the same name)