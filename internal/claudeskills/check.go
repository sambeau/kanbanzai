package claudeskills

import (
	"os"
	"path/filepath"
	"strings"
)

// WrapperSpec describes the expected metadata for a .claude/skills/ wrapper.
type WrapperSpec struct {
	Skill     string // skill directory name
	Canonical string // canonical skill file path
	Desc      string // single-line description
	Title     string // title heading
	Trigger   string // "when to use" trigger text
}

// ExpectedContent returns the canonical file content for this wrapper,
// matching exactly what scripts/gen-claude-skills.sh produces.
func (s WrapperSpec) ExpectedContent() string {
	lines := []string{
		"---",
		"name: " + s.Skill,
		`description: "` + s.Desc + `"`,
		"---",
		"",
		"<!-- kanbanzai-generated: true -->",
		"<!-- canonical: " + s.Canonical + " -->",
		"",
		"# " + s.Title,
		"",
		"When to use this skill: " + s.Trigger,
		"",
		"For the full procedure, vocabulary, and checklist see the canonical skill:",
		"`" + s.Canonical + "`",
		"", // produces trailing newline
	}
	return strings.Join(lines, "\n")
}

// ExpectedWrappers is the authoritative list of .claude/skills/ wrapper specs,
// kept in sync with scripts/gen-claude-skills.sh.
var ExpectedWrappers = []WrapperSpec{
	{
		Skill:     "orchestrate-development",
		Canonical: ".kbz/skills/orchestrate-development/SKILL.md",
		Desc:      "Multi-agent development orchestration — dispatch parallel tasks, monitor progress, handle failures, and close out the feature lifecycle",
		Title:     "Orchestrate Development",
		Trigger:   "when you are an orchestrator agent coordinating parallel implementation tasks for a feature within a batch.",
	},
	{
		Skill:     "implement-task",
		Canonical: ".kbz/skills/implement-task/SKILL.md",
		Desc:      "Guides you through implementing a single task — read what's required, build it, test it, verify it matches the spec",
		Title:     "Implement Task",
		Trigger:   "when executing a single implementation task — claim the task, build the code, run tests, and verify each acceptance criterion.",
	},
	{
		Skill:     "kanbanzai-getting-started",
		Canonical: ".agents/skills/kanbanzai-getting-started/SKILL.md",
		Desc:      "Use at the start of every session to orient yourself, find what to work on, and check the current project state",
		Title:     "Kanbanzai Getting Started",
		Trigger:   "at the start of every session — even if the task seems obvious — to orient yourself, verify entity existence, and check the work queue.",
	},
	{
		Skill:     "kanbanzai-workflow",
		Canonical: ".agents/skills/kanbanzai-workflow/SKILL.md",
		Desc:      "Use when deciding workflow stage transitions, stage gates, entity lifecycle rules, or whether to stop and ask the human",
		Title:     "Kanbanzai Workflow",
		Trigger:   "when deciding on workflow stage transitions, stage gates, or lifecycle rules — and whenever you are uncertain whether to proceed or stop.",
	},
	{
		Skill:     "write-spec",
		Canonical: ".kbz/skills/write-spec/SKILL.md",
		Desc:      "Author a specification: turn an approved design into traceable requirements, testable acceptance criteria, and a verification plan",
		Title:     "Write Spec",
		Trigger:   "when authoring a feature specification from an approved design document at the specifying stage.",
	},
	{
		Skill:     "write-design",
		Canonical: ".kbz/skills/write-design/SKILL.md",
		Desc:      "Author a design document: explain the problem, propose a solution, evaluate alternatives, and record architectural decisions",
		Title:     "Write Design",
		Trigger:   "when creating a design document for a feature at the designing stage.",
	},
	{
		Skill:     "review-code",
		Canonical: ".kbz/skills/review-code/SKILL.md",
		Desc:      "Review code changes against a spec and produce a structured report of what's right and what needs fixing",
		Title:     "Review Code",
		Trigger:   "when reviewing implementation changes against acceptance criteria at the reviewing stage.",
	},
}

// CheckAll checks all expected wrappers under skillsDir.
// It returns the paths of any stale or missing wrappers.
func CheckAll(skillsDir string) []string {
	var stale []string
	for _, spec := range ExpectedWrappers {
		path := filepath.Join(skillsDir, spec.Skill, "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil {
			stale = append(stale, path)
			continue
		}
		if string(data) != spec.ExpectedContent() {
			stale = append(stale, path)
		}
	}
	return stale
}
