package context

import (
	"fmt"
	"sort"
	"strings"

	kbzsvc "kanbanzai/internal/service"
)

// AssemblyInput contains the parameters for context assembly.
type AssemblyInput struct {
	Role     string // required: profile ID
	TaskID   string // optional
	MaxBytes int    // default 30720
}

// AssemblySource identifies where a piece of context came from.
type AssemblySource string

const (
	SourceProfile     AssemblySource = "profile"
	SourceDesign      AssemblySource = "design"
	SourceKnowledgeT2 AssemblySource = "knowledge-tier-2"
	SourceKnowledgeT3 AssemblySource = "knowledge-tier-3"
	SourceTask        AssemblySource = "task"
)

// AssemblyItem is a single piece of context in the assembled packet.
type AssemblyItem struct {
	Source     AssemblySource
	EntryID    string // for knowledge entries
	Priority   string // "high", "normal", "low"
	Content    string
	Confidence float64 // for knowledge entries
}

// AssemblyResult is the output of context assembly.
type AssemblyResult struct {
	Role      string
	TaskID    string
	Items     []AssemblyItem
	ByteCount int
	Trimmed   int // number of items trimmed due to budget
}

const defaultMaxBytes = 30720

// Assemble assembles a context packet for the given role and optional task.
// Profile and task instructions are never trimmed. When over budget, Tier 3
// (lowest-confidence) is trimmed first, then Tier 2, then design context.
func Assemble(
	input AssemblyInput,
	profileStore *ProfileStore,
	knowledgeSvc *kbzsvc.KnowledgeService,
	entitySvc *kbzsvc.EntityService,
	intelligenceSvc *kbzsvc.IntelligenceService,
) (*AssemblyResult, error) {
	maxBytes := input.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}

	// 1. Validate and resolve profile.
	profile, err := ResolveProfile(profileStore, input.Role)
	if err != nil {
		return nil, fmt.Errorf("resolve profile %q: %w", input.Role, err)
	}

	// 2. Profile item (never trimmed).
	profileItem := AssemblyItem{
		Source:   SourceProfile,
		Priority: "high",
		Content:  formatProfile(profile),
	}

	// 3. Task and design context (when task_id provided).
	var taskItem *AssemblyItem
	var designItems []AssemblyItem

	if input.TaskID != "" && entitySvc != nil {
		task, terr := entitySvc.Get("task", input.TaskID, "")
		if terr == nil {
			ti := AssemblyItem{
				Source:   SourceTask,
				Priority: "high",
				Content:  formatTask(task.State),
			}
			taskItem = &ti

			// Design context: trace parent feature through document intelligence.
			if intelligenceSvc != nil {
				parentFeature, _ := task.State["parent_feature"].(string)
				if parentFeature != "" {
					matches, merr := intelligenceSvc.TraceEntity(parentFeature)
					if merr == nil {
						for _, match := range matches {
							_, sectionContent, serr := intelligenceSvc.GetSection(match.DocumentID, match.SectionPath)
							if serr != nil || len(sectionContent) == 0 {
								continue
							}
							title := match.SectionTitle
							if title == "" {
								title = match.SectionPath
							}
							designItems = append(designItems, AssemblyItem{
								Source:   SourceDesign,
								Priority: "normal",
								Content:  fmt.Sprintf("=== Design: %s (%s) ===\n%s", title, match.DocumentID, string(sectionContent)),
							})
						}
					}
				}
			}
		}
		// If task not found, skip silently (best-effort).
	}

	// 4–5. Load knowledge entries (Tier 2 and Tier 3).
	var tier2Items []AssemblyItem
	var tier3Items []AssemblyItem

	if knowledgeSvc != nil {
		// Tier 2 knowledge (confidence >= 0.3), scoped to role or "project".
		tier2Records, err := knowledgeSvc.List(kbzsvc.KnowledgeFilters{
			Tier:          2,
			MinConfidence: 0.3,
		})
		if err != nil {
			return nil, fmt.Errorf("list tier-2 knowledge: %w", err)
		}

		for _, rec := range tier2Records {
			scope, _ := rec.Fields["scope"].(string)
			if !matchesScope(scope, input.Role) {
				continue
			}
			entryID, _ := rec.Fields["id"].(string)
			conf := assemblyFieldFloat(rec.Fields, "confidence")
			tier2Items = append(tier2Items, AssemblyItem{
				Source:     SourceKnowledgeT2,
				EntryID:    entryID,
				Priority:   "normal",
				Content:    formatKnowledgeEntry(rec.Fields),
				Confidence: conf,
			})
		}

		// Tier 3 knowledge (confidence >= 0.5), scoped to role or "project".
		tier3Records, err := knowledgeSvc.List(kbzsvc.KnowledgeFilters{
			Tier:          3,
			MinConfidence: 0.5,
		})
		if err != nil {
			return nil, fmt.Errorf("list tier-3 knowledge: %w", err)
		}

		for _, rec := range tier3Records {
			scope, _ := rec.Fields["scope"].(string)
			if !matchesScope(scope, input.Role) {
				continue
			}
			entryID, _ := rec.Fields["id"].(string)
			conf := assemblyFieldFloat(rec.Fields, "confidence")
			tier3Items = append(tier3Items, AssemblyItem{
				Source:     SourceKnowledgeT3,
				EntryID:    entryID,
				Priority:   "low",
				Content:    formatKnowledgeEntry(rec.Fields),
				Confidence: conf,
			})
		}
	}

	// 6. Calculate initial byte count.
	totalBytes := len(profileItem.Content)
	for _, item := range tier2Items {
		totalBytes += len(item.Content)
	}
	for _, item := range tier3Items {
		totalBytes += len(item.Content)
	}
	for _, item := range designItems {
		totalBytes += len(item.Content)
	}
	if taskItem != nil {
		totalBytes += len(taskItem.Content)
	}

	// 7. Trim if over budget.
	trimmed := 0

	// Trim T3 lowest-confidence first.
	if totalBytes > maxBytes {
		sort.SliceStable(tier3Items, func(i, j int) bool {
			return tier3Items[i].Confidence < tier3Items[j].Confidence
		})
		for len(tier3Items) > 0 && totalBytes > maxBytes {
			totalBytes -= len(tier3Items[0].Content)
			tier3Items = tier3Items[1:]
			trimmed++
		}
	}

	// Trim T2 lowest-confidence first, if still over budget.
	if totalBytes > maxBytes {
		sort.SliceStable(tier2Items, func(i, j int) bool {
			return tier2Items[i].Confidence < tier2Items[j].Confidence
		})
		for len(tier2Items) > 0 && totalBytes > maxBytes {
			totalBytes -= len(tier2Items[0].Content)
			tier2Items = tier2Items[1:]
			trimmed++
		}
	}

	// Trim design context from the end, if still over budget.
	for len(designItems) > 0 && totalBytes > maxBytes {
		totalBytes -= len(designItems[len(designItems)-1].Content)
		designItems = designItems[:len(designItems)-1]
		trimmed++
	}

	// 8. Build final ordered list: profile, T2, T3, design, task.
	capacity := 1 + len(tier2Items) + len(tier3Items) + len(designItems)
	if taskItem != nil {
		capacity++
	}
	finalItems := make([]AssemblyItem, 0, capacity)
	finalItems = append(finalItems, profileItem)
	finalItems = append(finalItems, tier2Items...)
	finalItems = append(finalItems, tier3Items...)
	finalItems = append(finalItems, designItems...)
	if taskItem != nil {
		finalItems = append(finalItems, *taskItem)
	}

	// Recompute actual byte count from final items.
	actualBytes := 0
	for _, item := range finalItems {
		actualBytes += len(item.Content)
	}

	return &AssemblyResult{
		Role:      input.Role,
		TaskID:    input.TaskID,
		Items:     finalItems,
		ByteCount: actualBytes,
		Trimmed:   trimmed,
	}, nil
}

// matchesScope returns true if entryScope is "project" or equals roleID.
func matchesScope(entryScope, roleID string) bool {
	return entryScope == "project" || entryScope == roleID
}

// formatProfile formats a resolved profile as a structured text block.
func formatProfile(p *ResolvedProfile) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "=== Role Profile: %s ===\n", p.ID)
	if p.Description != "" {
		fmt.Fprintf(&sb, "%s\n", p.Description)
	}
	if len(p.Packages) > 0 {
		fmt.Fprintf(&sb, "\nPackages: %s\n", strings.Join(p.Packages, ", "))
	}
	if len(p.Conventions) > 0 {
		fmt.Fprintf(&sb, "\nConventions:\n")
		for _, c := range p.Conventions {
			fmt.Fprintf(&sb, "- %s\n", c)
		}
	}
	if p.Architecture != nil {
		if p.Architecture.Summary != "" {
			fmt.Fprintf(&sb, "\nArchitecture: %s\n", p.Architecture.Summary)
		}
		if len(p.Architecture.KeyInterfaces) > 0 {
			fmt.Fprintf(&sb, "Key Interfaces:\n")
			for _, ki := range p.Architecture.KeyInterfaces {
				fmt.Fprintf(&sb, "- %s\n", ki)
			}
		}
	}
	return sb.String()
}

// formatKnowledgeEntry formats a knowledge entry fields map as a text block.
func formatKnowledgeEntry(fields map[string]any) string {
	tier := assemblyFieldInt(fields, "tier")
	topic, _ := fields["topic"].(string)
	scope, _ := fields["scope"].(string)
	conf := assemblyFieldFloat(fields, "confidence")
	content, _ := fields["content"].(string)
	return fmt.Sprintf("[%d] %s (scope: %s, confidence: %.2f)\n%s\n", tier, topic, scope, conf, content)
}

// formatTask formats a task entity's state map as a text block.
func formatTask(state map[string]any) string {
	id, _ := state["id"].(string)
	summary, _ := state["summary"].(string)
	status, _ := state["status"].(string)
	verification, _ := state["verification"].(string)

	var sb strings.Builder
	fmt.Fprintf(&sb, "=== Task: %s ===\n", id)
	fmt.Fprintf(&sb, "Summary: %s\n", summary)
	fmt.Fprintf(&sb, "Status: %s\n", status)
	if verification != "" {
		fmt.Fprintf(&sb, "Verification: %s\n", verification)
	}
	return sb.String()
}

// assemblyFieldInt reads an integer value from a fields map.
func assemblyFieldInt(fields map[string]any, key string) int {
	v, ok := fields[key]
	if !ok {
		return 0
	}
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	}
	return 0
}

// assemblyFieldFloat reads a float64 value from a fields map.
func assemblyFieldFloat(fields map[string]any, key string) float64 {
	v, ok := fields[key]
	if !ok {
		return 0
	}
	switch typed := v.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	}
	return 0
}
