# .agents/skills — Agent Workflow Skill Index

This directory contains Kanbanzai workflow skills for agents. **Each skill lives
in its own subdirectory** with a `SKILL.md` file. Do not add skill files directly
to this directory.

## Available Skills

| Directory | Purpose |
|-----------|---------|
| `kanbanzai-agents/` | Task dispatch, commits, finish, knowledge contribution |
| `kanbanzai-documents/` | Creating, registering, and approving documents |
| `kanbanzai-getting-started/` | Session orientation and work queue |
| `kanbanzai-plan-review/` | Plan and batch review process |
| `kanbanzai-planning/` | Scoping work, feature vs batch decisions |
| `kanbanzai-workflow/` | Stage gates, lifecycle transitions, when to stop |

See `AGENTS.md` in the repository root for an overview of when to use each skill.
See `.github/copilot-instructions.md` for the stage-to-skill mapping table.
