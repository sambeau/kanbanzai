package service

import (
	"fmt"
	"regexp"
	"strings"
)

var testingKeywords = []string{
	"test", "tests", "testing",
	"verify", "verifies", "verification",
	"validate", "validates", "validation",
	"spec", "coverage",
	"assert", "assertion", "assertions",
}

var testingKeywordRe = buildTestingKeywordRe()

func buildTestingKeywordRe() *regexp.Regexp {
	quoted := make([]string, len(testingKeywords))
	for i, kw := range testingKeywords {
		quoted[i] = regexp.QuoteMeta(kw)
	}
	return regexp.MustCompile("\\b(?:" + strings.Join(quoted, "|") + ")\\b")
}

var actionVerbs = []string{
	"implement", "add", "create", "refactor", "update", "fix",
	"remove", "delete", "migrate", "configure", "write", "build",
	"set up", "modify", "change", "extract", "move", "rename",
	"convert", "integrate", "replace", "introduce", "extend",
	"redesign", "rewrite",
}

var coordinatingSeparators = []string{
	" and ", " as well as ", " additionally ", " plus ", "; ",
}

func checkDescriptionPresent(p Proposal) []Finding {
	var findings []Finding
	for _, task := range p.Tasks {
		if strings.TrimSpace(task.Summary) == "" {
			findings = append(findings, Finding{
				Type:     "empty-description",
				Severity: "error",
				TaskSlug: task.Slug,
				Detail:   fmt.Sprintf("task %q has an empty summary", task.Slug),
			})
		}
	}
	return findings
}

func checkTestingCoverage(p Proposal) []Finding {
	for _, task := range p.Tasks {
		text := strings.ToLower(task.Summary + " " + task.Rationale)
		if testingKeywordRe.MatchString(text) {
			return nil
		}
	}
	return []Finding{{
		Type:     "missing-test-coverage",
		Severity: "warning",
		Detail:   "no task addresses testing or verification",
	}}
}

func checkDependenciesDeclared(p Proposal) []Finding {
	depPairs := make(map[string]bool)
	for _, t := range p.Tasks {
		for _, dep := range t.DependsOn {
			depPairs[t.Slug+":"+dep] = true
		}
	}
	var findings []Finding
	seen := make(map[string]bool)
	for _, taskA := range p.Tasks {
		text := strings.ToLower(taskA.Summary + " " + taskA.Rationale)
		for _, taskB := range p.Tasks {
			if taskA.Slug == taskB.Slug {
				continue
			}
			if !slugMatchesAtWordBoundary(text, taskB.Slug) {
				continue
			}
			if depPairs[taskA.Slug+":"+taskB.Slug] || depPairs[taskB.Slug+":"+taskA.Slug] {
				continue
			}
			pairKey := taskA.Slug + ":" + taskB.Slug
			if !seen[pairKey] {
				seen[pairKey] = true
				findings = append(findings, Finding{
					Type:     "undeclared-dependency",
					Severity: "warning",
					TaskSlug: taskA.Slug,
					Detail: fmt.Sprintf("task %q references %q in its description but no dependency is declared between them",
						taskA.Slug, taskB.Slug),
				})
			}
		}
	}
	return findings
}

func checkOrphanTasks(p Proposal) []Finding {
	slugSet := make(map[string]bool, len(p.Tasks))
	for _, t := range p.Tasks {
		slugSet[t.Slug] = true
	}
	edgeCount := 0
	for _, t := range p.Tasks {
		for _, dep := range t.DependsOn {
			if slugSet[dep] {
				edgeCount++
			}
		}
	}
	if edgeCount == 0 {
		return nil
	}
	degree := make(map[string]int, len(p.Tasks))
	for _, t := range p.Tasks {
		degree[t.Slug] = 0
	}
	for _, t := range p.Tasks {
		for _, dep := range t.DependsOn {
			if slugSet[dep] {
				degree[t.Slug]++
				degree[dep]++
			}
		}
	}
	var findings []Finding
	for _, task := range p.Tasks {
		if degree[task.Slug] == 0 {
			findings = append(findings, Finding{
				Type:     "orphan-task",
				Severity: "warning",
				TaskSlug: task.Slug,
				Detail:   fmt.Sprintf("task %q has no dependency edges while other tasks do", task.Slug),
			})
		}
	}
	return findings
}

func checkSingleAgentSizing(p Proposal) []Finding {
	var findings []Finding
	for _, task := range p.Tasks {
		if f := multiAgentFinding(task); f != nil {
			findings = append(findings, *f)
		}
	}
	return findings
}

func multiAgentFinding(task ProposedTask) *Finding {
	for _, sep := range coordinatingSeparators {
		re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(sep))
		parts := re.Split(task.Summary, -1)
		if len(parts) < 2 {
			continue
		}
		var matchedVerbs []string
		for _, part := range parts {
			if ok, verb := clauseStartsWithVerb(part); ok {
				matchedVerbs = append(matchedVerbs, verb)
			}
		}
		if len(matchedVerbs) >= 2 {
			return &Finding{
				Type:     "multi-agent-sizing",
				Severity: "warning",
				TaskSlug: task.Slug,
				Detail: fmt.Sprintf("task %q may span multiple agents: action verbs %q and %q separated by %q",
					task.Slug, matchedVerbs[0], matchedVerbs[1], strings.TrimSpace(sep)),
			}
		}
	}
	return nil
}

func clauseStartsWithVerb(clause string) (bool, string) {
	lower := strings.ToLower(strings.TrimSpace(clause))
	for _, verb := range actionVerbs {
		verbLower := strings.ToLower(verb)
		if !strings.HasPrefix(lower, verbLower) {
			continue
		}
		rest := lower[len(verbLower):]
		if rest == "" || !isWordRune(rune(rest[0])) {
			return true, verb
		}
	}
	return false, ""
}

func slugMatchesAtWordBoundary(text, slug string) bool {
	re := regexp.MustCompile("(?i)\\b" + regexp.QuoteMeta(slug) + "\\b")
	return re.MatchString(text)
}

func isWordRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}
