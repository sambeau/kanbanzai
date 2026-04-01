# Agent Skills

This directory contains skill documents that provide guidance to AI agents
working on this project. Each subdirectory contains a `SKILL.md` file with
structured instructions for a specific domain.

> **Note:** Kanbanzai workflow skills live in `.agents/skills/kanbanzai-*/SKILL.md`,
> not in this directory. See `.github/copilot-instructions.md` for a complete
> skill index across both locations.

## Skills

| Skill | Provenance | Purpose |
|-------|-----------|---------|
| `planning-excellence` | Project-owned | Planning philosophy and design quality framework |
| `codebase-memory-reference` | Upstream (`codebase-memory-mcp`) | Tool reference for all 14 graph tools |
| `codebase-memory-exploring` | Upstream (`codebase-memory-mcp`) | Codebase exploration and indexing workflows |
| `codebase-memory-tracing` | Upstream (`codebase-memory-mcp`) | Call chain tracing and impact analysis |
| `codebase-memory-quality` | Upstream (`codebase-memory-mcp`) | Code quality analysis patterns |

## Provenance

- **Project-owned** skills are maintained in this repository.
- **Upstream** skills originate from `codebase-memory-mcp install` and are
  also installed locally at `~/.claude/skills/`. If upstream updates occur,
  refresh the repo copies:
  ```
  cp -r ~/.claude/skills/codebase-memory-* .github/skills/
  ```

## For Claude

Claude loads these natively from `~/.claude/skills/`. The copies here exist
so that other agents and tools can discover the same guidance by reading the
repository.
