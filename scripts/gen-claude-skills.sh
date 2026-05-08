#!/usr/bin/env bash
# scripts/gen-claude-skills.sh — generates Anthropic-format SKILL.md wrappers
# under .claude/skills/<skill>/SKILL.md for the seven Kanbanzai skills
# recognised by Claude. Re-run after canonical skill updates to regenerate.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

write_wrapper() {
  local skill="$1"
  local canonical="$2"
  local description="$3"
  local title="$4"
  local trigger="$5"

  local out_dir="$REPO_ROOT/.claude/skills/$skill"
  mkdir -p "$out_dir"

  cat > "$out_dir/SKILL.md" <<WRAPPER
---
name: $skill
description: "$description"
---

<!-- kanbanzai-generated: true -->
<!-- canonical: $canonical -->

# $title

When to use this skill: $trigger

For the full procedure, vocabulary, and checklist see the canonical skill:
\`$canonical\`
WRAPPER

  echo "wrote .claude/skills/$skill/SKILL.md"
}

write_wrapper \
  "orchestrate-development" \
  ".kbz/skills/orchestrate-development/SKILL.md" \
  "Multi-agent development orchestration — dispatch parallel tasks, monitor progress, handle failures, and close out the feature lifecycle" \
  "Orchestrate Development" \
  "when you are an orchestrator agent coordinating parallel implementation tasks for a feature within a batch."

write_wrapper \
  "implement-task" \
  ".kbz/skills/implement-task/SKILL.md" \
  "Guides you through implementing a single task — read what's required, build it, test it, verify it matches the spec" \
  "Implement Task" \
  "when executing a single implementation task — claim the task, build the code, run tests, and verify each acceptance criterion."

write_wrapper \
  "kanbanzai-getting-started" \
  ".agents/skills/kanbanzai-getting-started/SKILL.md" \
  "Use at the start of every session to orient yourself, find what to work on, and check the current project state" \
  "Kanbanzai Getting Started" \
  "at the start of every session — even if the task seems obvious — to orient yourself, verify entity existence, and check the work queue."

write_wrapper \
  "kanbanzai-workflow" \
  ".agents/skills/kanbanzai-workflow/SKILL.md" \
  "Use when deciding workflow stage transitions, stage gates, entity lifecycle rules, or whether to stop and ask the human" \
  "Kanbanzai Workflow" \
  "when deciding on workflow stage transitions, stage gates, or lifecycle rules — and whenever you are uncertain whether to proceed or stop."

write_wrapper \
  "write-spec" \
  ".kbz/skills/write-spec/SKILL.md" \
  "Author a specification: turn an approved design into traceable requirements, testable acceptance criteria, and a verification plan" \
  "Write Spec" \
  "when authoring a feature specification from an approved design document at the specifying stage."

write_wrapper \
  "write-design" \
  ".kbz/skills/write-design/SKILL.md" \
  "Author a design document: explain the problem, propose a solution, evaluate alternatives, and record architectural decisions" \
  "Write Design" \
  "when creating a design document for a feature at the designing stage."

write_wrapper \
  "review-code" \
  ".kbz/skills/review-code/SKILL.md" \
  "Review code changes against a spec and produce a structured report of what's right and what needs fixing" \
  "Review Code" \
  "when reviewing implementation changes against acceptance criteria at the reviewing stage."

echo "done — seven wrappers generated under .claude/skills/"
