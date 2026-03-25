package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReviewInput is the input for ReviewService.ReviewTaskOutput.
type ReviewInput struct {
	TaskID        string
	OutputFiles   []string
	OutputSummary string
}

// ReviewResult is the output of ReviewService.ReviewTaskOutput.
type ReviewResult struct {
	TaskID        string          `json:"task_id"`
	TaskSlug      string          `json:"task_slug"`
	Status        string          `json:"status"`
	Findings      []ReviewFinding `json:"findings"`
	TotalFindings int             `json:"total_findings"`
	BlockingCount int             `json:"blocking_count"`
}

// ReviewFinding is a single finding from a worker review.
type ReviewFinding struct {
	Severity string `json:"severity"`
	Type     string `json:"type"`
	Detail   string `json:"detail"`
}

// ReviewService provides worker review for completed tasks.
type ReviewService struct {
	entitySvc *EntityService
	intelSvc  *IntelligenceService
	repoRoot  string
}

// NewReviewService creates a new ReviewService.
func NewReviewService(entitySvc *EntityService, intelSvc *IntelligenceService, repoRoot string) *ReviewService {
	return &ReviewService{entitySvc: entitySvc, intelSvc: intelSvc, repoRoot: repoRoot}
}

// ReviewTaskOutput runs a first-pass review of a task's output against its
// verification criteria and the parent feature's spec. It triggers state
// transitions for tasks in "active" status: fail → needs-rework, pass → needs-review.
func (s *ReviewService) ReviewTaskOutput(input ReviewInput) (ReviewResult, error) {
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return ReviewResult{}, fmt.Errorf("task_id is required")
	}

	// Step 1: Load the task. Reject if status is not active, done, or needs-review.
	taskResult, err := s.entitySvc.Get("task", taskID, "")
	if err != nil {
		return ReviewResult{}, fmt.Errorf("loading task %s: %w", taskID, err)
	}

	status, _ := taskResult.State["status"].(string)
	if status != "active" && status != "done" && status != "needs-review" {
		return ReviewResult{}, fmt.Errorf("task %s has status %q; review requires active, done, or needs-review", taskID, status)
	}

	taskSlug, _ := taskResult.State["slug"].(string)
	taskSummary, _ := taskResult.State["summary"].(string)
	verification, _ := taskResult.State["verification"].(string)

	// Step 2: Load the parent feature and resolve its spec document reference.
	parentFeature, _ := taskResult.State["parent_feature"].(string)
	specDocID := ""
	if parentFeature != "" {
		featureResult, err := s.entitySvc.Get("feature", parentFeature, "")
		if err == nil {
			specDocID, _ = featureResult.State["spec"].(string)
		}
	}

	var findings []ReviewFinding

	// Step 3: Task-level checks.
	findings = append(findings, s.checkOutputFiles(input.OutputFiles)...)
	findings = append(findings, s.checkVerification(input.OutputSummary, verification)...)
	findings = append(findings, s.checkSummaryRelevance(input.OutputSummary, taskSummary)...)

	// Step 4: Spec-level check.
	findings = append(findings, s.checkSpec(taskID, specDocID)...)

	// Step 5: Aggregate result.
	blockingCount := 0
	for _, f := range findings {
		if f.Severity == "error" {
			blockingCount++
		}
	}

	reviewStatus := "pass"
	if blockingCount > 0 {
		reviewStatus = "fail"
	} else if len(findings) > 0 {
		reviewStatus = "pass_with_warnings"
	}

	result := ReviewResult{
		TaskID:        taskResult.ID,
		TaskSlug:      taskSlug,
		Status:        reviewStatus,
		Findings:      findings,
		TotalFindings: len(findings),
		BlockingCount: blockingCount,
	}

	// Step 6: State transitions (only for tasks in "active" status).
	if status == "active" {
		if err := s.applyTransition(taskResult.ID, taskSlug, reviewStatus, findings); err != nil {
			return ReviewResult{}, err
		}
	}

	return result, nil
}

// checkOutputFiles verifies that each output file exists on disk.
func (s *ReviewService) checkOutputFiles(files []string) []ReviewFinding {
	var findings []ReviewFinding
	for _, f := range files {
		fullPath := filepath.Join(s.repoRoot, f)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			findings = append(findings, ReviewFinding{
				Severity: "error",
				Type:     "missing_file",
				Detail:   fmt.Sprintf("output file does not exist: %s", f),
			})
		}
	}
	return findings
}

// checkVerification checks whether the output summary addresses the task's
// verification criteria. Uses keyword overlap as a heuristic.
func (s *ReviewService) checkVerification(outputSummary, verification string) []ReviewFinding {
	if verification == "" || outputSummary == "" {
		return nil
	}

	keywords := extractKeywords(verification)
	if len(keywords) == 0 {
		return nil
	}

	summaryLower := strings.ToLower(outputSummary)
	matched := 0
	for _, kw := range keywords {
		if strings.Contains(summaryLower, kw) {
			matched++
		}
	}

	// If less than 30% of verification keywords appear in the output summary,
	// flag as potentially unmet. This is a deliberately lenient heuristic.
	if len(keywords) > 0 && float64(matched)/float64(len(keywords)) < 0.3 {
		return []ReviewFinding{{
			Severity: "warning",
			Type:     "verification_unmet",
			Detail:   fmt.Sprintf("output summary may not address verification criteria: %q", verification),
		}}
	}

	return nil
}

// checkSummaryRelevance checks whether the output summary addresses the task summary.
func (s *ReviewService) checkSummaryRelevance(outputSummary, taskSummary string) []ReviewFinding {
	if outputSummary == "" || taskSummary == "" {
		return nil
	}

	keywords := extractKeywords(taskSummary)
	if len(keywords) == 0 {
		return nil
	}

	summaryLower := strings.ToLower(outputSummary)
	matched := 0
	for _, kw := range keywords {
		if strings.Contains(summaryLower, kw) {
			matched++
		}
	}

	if len(keywords) > 0 && float64(matched)/float64(len(keywords)) < 0.3 {
		return []ReviewFinding{{
			Severity: "warning",
			Type:     "verification_unmet",
			Detail:   fmt.Sprintf("output summary may not address task summary: %q", taskSummary),
		}}
	}

	return nil
}

// checkSpec performs the spec-level check using doc_trace.
func (s *ReviewService) checkSpec(taskID, specDocID string) []ReviewFinding {
	if specDocID == "" {
		return []ReviewFinding{{
			Severity: "warning",
			Type:     "no_spec",
			Detail:   "no spec document registered on the parent feature",
		}}
	}

	if s.intelSvc == nil {
		return []ReviewFinding{{
			Severity: "warning",
			Type:     "spec_gap",
			Detail:   "document intelligence service not available; spec-level check skipped",
		}}
	}

	matches, err := s.intelSvc.TraceEntity(taskID)
	if err != nil || len(matches) == 0 {
		return []ReviewFinding{{
			Severity: "warning",
			Type:     "spec_gap",
			Detail:   "no spec sections found referencing this task; spec coverage may be incomplete",
		}}
	}

	return nil
}

// applyTransition applies the state transition based on the review result.
func (s *ReviewService) applyTransition(taskID, taskSlug, reviewStatus string, findings []ReviewFinding) error {
	if reviewStatus == "fail" {
		// Transition to needs-rework.
		_, err := s.entitySvc.UpdateStatus(UpdateStatusInput{
			Type:   "task",
			ID:     taskID,
			Slug:   taskSlug,
			Status: "needs-rework",
		})
		if err != nil {
			return fmt.Errorf("transitioning task to needs-rework: %w", err)
		}

		// Set rework_reason to a summary of blocking findings.
		reason := summarizeBlockingFindings(findings)
		_, err = s.entitySvc.UpdateEntity(UpdateEntityInput{
			Type:   "task",
			ID:     taskID,
			Slug:   taskSlug,
			Fields: map[string]string{"rework_reason": reason},
		})
		if err != nil {
			return fmt.Errorf("setting rework_reason: %w", err)
		}
	} else {
		// Pass or pass_with_warnings → transition to needs-review.
		_, err := s.entitySvc.UpdateStatus(UpdateStatusInput{
			Type:   "task",
			ID:     taskID,
			Slug:   taskSlug,
			Status: "needs-review",
		})
		if err != nil {
			return fmt.Errorf("transitioning task to needs-review: %w", err)
		}
	}
	return nil
}

// summarizeBlockingFindings produces a rework_reason string from blocking findings.
func summarizeBlockingFindings(findings []ReviewFinding) string {
	var parts []string
	for _, f := range findings {
		if f.Severity == "error" {
			parts = append(parts, f.Detail)
		}
	}
	if len(parts) == 0 {
		return "review failed"
	}
	return strings.Join(parts, "; ")
}

// extractKeywords splits text into lowercase keywords, filtering out short
// and common stop words. This is a simple heuristic for keyword overlap checks.
func extractKeywords(text string) []string {
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"to": true, "of": true, "in": true, "for": true, "on": true,
		"at": true, "by": true, "with": true, "from": true, "as": true,
		"it": true, "its": true, "that": true, "this": true, "not": true,
		"but": true, "if": true, "has": true, "have": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "should": true,
		"can": true, "could": true, "would": true, "may": true, "must": true,
	}

	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_')
	})

	var keywords []string
	for _, w := range words {
		if len(w) < 3 {
			continue
		}
		if stopWords[w] {
			continue
		}
		keywords = append(keywords, w)
	}
	return keywords
}
