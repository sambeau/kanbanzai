package health

import (
	"fmt"
	"time"
)

// CheckDependencyCycles detects circular dependencies in tasks.
// tasks is a slice of maps, each with "id" and "depends_on" ([]string or []any) fields.
func CheckDependencyCycles(tasks []map[string]any) CategoryResult {
	result := NewCategoryResult()

	// Build adjacency map: taskID → []dependencyID
	graph := make(map[string][]string)
	allTaskIDs := make(map[string]struct{})

	for _, t := range tasks {
		id, _ := t["id"].(string)
		if id == "" {
			continue
		}
		allTaskIDs[id] = struct{}{}
		graph[id] = extractStringSlice(t, "depends_on")
	}

	// DFS cycle detection using three-color marking.
	type color int
	const (
		white color = iota // unvisited
		gray               // in current path
		black              // fully visited
	)

	colors := make(map[string]color, len(allTaskIDs))
	parent := make(map[string]string) // for path reconstruction
	reported := make(map[string]struct{})

	var detectCycle func(node string)
	detectCycle = func(node string) {
		colors[node] = gray
		for _, dep := range graph[node] {
			if colors[dep] == gray {
				// Found a cycle — report it
				cycleKey := node + "→" + dep
				if _, seen := reported[cycleKey]; seen {
					continue
				}
				reported[cycleKey] = struct{}{}
				result.AddIssue(Issue{
					Severity: SeverityError,
					EntityID: node,
					Message:  fmt.Sprintf("dependency cycle detected: %s → %s", node, dep),
				})
			} else if colors[dep] == white {
				parent[dep] = node
				detectCycle(dep)
			}
		}
		colors[node] = black
	}

	for id := range allTaskIDs {
		if colors[id] == white {
			detectCycle(id)
		}
	}

	// parent is used in cycle path reconstruction (future enhancement).
	_ = parent

	return result
}

// CheckStalledDispatches detects tasks that have been active too long without git activity.
// tasks is a slice of task field maps.
// worktreeBranches maps entity IDs to their worktree branch names.
// repoPath is the repository root for git log checks.
// stallThresholdDays is 0 to disable, otherwise the number of days before flagging.
func CheckStalledDispatches(tasks []map[string]any, worktreeBranches map[string]string, repoPath string, stallThresholdDays int) CategoryResult {
	result := NewCategoryResult()

	if stallThresholdDays <= 0 {
		return result // disabled
	}

	now := time.Now()
	threshold := time.Duration(stallThresholdDays) * 24 * time.Hour

	for _, t := range tasks {
		status, _ := t["status"].(string)
		if status != "active" {
			continue
		}

		taskID, _ := t["id"].(string)
		dispatchedAtStr, _ := t["dispatched_at"].(string)
		if dispatchedAtStr == "" {
			continue // manually activated, skip
		}

		dispatchedAt, err := time.Parse(time.RFC3339, dispatchedAtStr)
		if err != nil {
			continue
		}

		age := now.Sub(dispatchedAt)
		if age < threshold {
			continue
		}

		// Check git activity on the worktree branch (best-effort).
		parentFeature, _ := t["parent_feature"].(string)
		dispatchedTo, _ := t["dispatched_to"].(string)

		hasGitActivity := false
		if branch, ok := worktreeBranches[parentFeature]; ok && branch != "" {
			hasGitActivity = checkGitActivitySince(repoPath, branch, dispatchedAt)
		}

		if !hasGitActivity {
			dayCount := int(age.Hours() / 24)
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: taskID,
				Message: fmt.Sprintf(
					"%s has been active for %d day(s) with no git activity since dispatch (dispatched to: %s)",
					taskID, dayCount, dispatchedTo,
				),
			})
		}
	}

	return result
}

// CheckEstimationCoverage checks that features in active+ status have estimated tasks.
// features and tasks are field maps. activeStatuses is the set of statuses that trigger the check.
func CheckEstimationCoverage(features []map[string]any, tasks []map[string]any, activeStatuses map[string]struct{}) CategoryResult {
	result := NewCategoryResult()

	// Build map of feature ID → child tasks.
	featureTasks := make(map[string][]map[string]any)
	for _, t := range tasks {
		pf, _ := t["parent_feature"].(string)
		if pf == "" {
			continue
		}
		featureTasks[pf] = append(featureTasks[pf], t)
	}

	for _, f := range features {
		status, _ := f["status"].(string)
		if _, active := activeStatuses[status]; !active {
			continue
		}

		featureID, _ := f["id"].(string)
		childTasks := featureTasks[featureID]
		if len(childTasks) == 0 {
			continue
		}

		// Check if any non-terminal task has an estimate.
		anyEstimated := false
		for _, t := range childTasks {
			taskStatus, _ := t["status"].(string)
			if taskStatus == "not-planned" || taskStatus == "duplicate" {
				continue
			}
			if _, ok := t["estimate"]; ok {
				anyEstimated = true
				break
			}
		}

		if !anyEstimated {
			result.AddIssue(Issue{
				Severity: SeverityWarning,
				EntityID: featureID,
				Message: fmt.Sprintf(
					"%s has %d active task(s) with no estimates — work queue ordering will be incomplete",
					featureID, len(childTasks),
				),
			})
		}
	}

	return result
}

// extractStringSlice extracts a string slice from a map value.
// Handles both []string and []any (from YAML round-trips).
func extractStringSlice(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch typed := v.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// checkGitActivitySince checks if there's any git activity on a branch since the given time.
// Returns false on any error (best-effort). Full git log integration is a future enhancement.
func checkGitActivitySince(repoPath, branch string, since time.Time) bool {
	_ = repoPath
	_ = branch
	_ = since
	return false // always false means time-only check applies
}
