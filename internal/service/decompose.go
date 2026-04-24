package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/model"
)

// DecomposeInput is the input for DecomposeService.DecomposeFeature.
type DecomposeInput struct {
	FeatureID string
	Context   string // optional additional guidance
}

// ProposedTask is a single task in a decomposition proposal.
type ProposedTask struct {
	Slug      string   `json:"slug"`
	Name      string   `json:"name"`
	Summary   string   `json:"summary"`
	Role      string   `json:"role,omitempty"`
	Estimate  *float64 `json:"estimate,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
	Rationale string   `json:"rationale"`
	Covers    []string `json:"covers,omitempty"` // AC texts this task covers
}

// Proposal is the complete decomposition proposal for a feature.
type Proposal struct {
	Tasks          []ProposedTask  `json:"tasks"`
	TotalTasks     int             `json:"total_tasks"`
	EstimatedTotal *float64        `json:"estimated_total,omitempty"`
	Slices         []string        `json:"slices"`
	SliceDetails   []AnalysisSlice `json:"slice_details,omitempty"`
	Warnings       []string        `json:"warnings"`
}

// DecomposeResult is the output of DecomposeService.DecomposeFeature.
type DecomposeResult struct {
	FeatureID       string   `json:"feature_id"`
	FeatureSlug     string   `json:"feature_slug"`
	SpecDocumentID  string   `json:"spec_document_id"`
	Proposal        Proposal `json:"proposal"`
	GuidanceApplied []string `json:"guidance_applied"`
}

// DecomposeReviewInput is the input for DecomposeService.ReviewProposal.
type DecomposeReviewInput struct {
	FeatureID string
	Proposal  Proposal
}

// Finding is a single review finding from decompose_review.
type Finding struct {
	Type     string `json:"type"`                // gap, overlap, oversized, ambiguous, cycle
	TaskSlug string `json:"task_slug,omitempty"` // affected task slug, empty for feature-level findings
	Detail   string `json:"detail"`
	Severity string `json:"severity"` // error (blocking) or warning (non-blocking)
}

// DecomposeReviewResult is the output of DecomposeService.ReviewProposal.
type DecomposeReviewResult struct {
	FeatureID     string    `json:"feature_id"`
	Status        string    `json:"status"` // pass, fail, warn
	Findings      []Finding `json:"findings"`
	TotalFindings int       `json:"total_findings"`
	BlockingCount int       `json:"blocking_count"`
}

// SliceAnalysisInput is the input for DecomposeService.SliceAnalysis.
type SliceAnalysisInput struct {
	FeatureID string
}

// AnalysisSlice is a candidate vertical slice in a feature.
type AnalysisSlice struct {
	Name      string   `json:"name"`
	Outcomes  []string `json:"outcomes"`
	Layers    []string `json:"layers"`
	Estimate  string   `json:"estimate"`
	DependsOn []string `json:"depends_on,omitempty"`
	Rationale string   `json:"rationale"`
}

// SliceAnalysisResult is the output of DecomposeService.SliceAnalysis.
type SliceAnalysisResult struct {
	FeatureID     string          `json:"feature_id"`
	FeatureSlug   string          `json:"feature_slug"`
	Slices        []AnalysisSlice `json:"slices"`
	TotalSlices   int             `json:"total_slices"`
	AnalysisNotes string          `json:"analysis_notes"`
}

// DecomposeService provides feature decomposition and proposal review.
type DecomposeService struct {
	entitySvc *EntityService
	docSvc    *DocumentService
}

// NewDecomposeService creates a new DecomposeService.
func NewDecomposeService(entitySvc *EntityService, docSvc *DocumentService) *DecomposeService {
	return &DecomposeService{entitySvc: entitySvc, docSvc: docSvc}
}

// DecomposeFeature loads a feature's spec document, applies decomposition
// guidance, and returns a proposed task list. It never writes any tasks.
func (s *DecomposeService) DecomposeFeature(input DecomposeInput) (DecomposeResult, error) {
	if strings.TrimSpace(input.FeatureID) == "" {
		return DecomposeResult{}, fmt.Errorf("feature_id is required")
	}

	// 1. Load the feature.
	feat, err := s.entitySvc.Get("feature", input.FeatureID, "")
	if err != nil {
		return DecomposeResult{}, fmt.Errorf("load feature %s: %w", input.FeatureID, err)
	}

	featureSlug, _ := feat.State["slug"].(string)
	specDocID, _ := feat.State["spec"].(string)

	// 2. Verify a spec document is linked.
	if specDocID == "" {
		return DecomposeResult{}, fmt.Errorf("feature %s has no linked specification document", feat.ID)
	}

	// 3. Retrieve spec document content via the document record path.
	content, docResult, err := s.docSvc.GetDocumentContent(specDocID)
	if err != nil {
		return DecomposeResult{}, fmt.Errorf("load spec document %s: %w", specDocID, err)
	}

	// 4. Gate: spec must be approved before decomposition.
	if docResult.Status != string(model.DocumentStatusApproved) {
		return DecomposeResult{}, fmt.Errorf("spec %q is in %q status — approve the spec before decomposing", specDocID, docResult.Status)
	}

	// 5. Parse the spec for structure (always, for slice enrichment on all paths).
	spec := parseSpecStructure(content)

	// Dev plan discovery: check for an approved dev plan and try to use its
	// Task Breakdown section as the source of truth for decomposition.
	var devPlanDocID string
	var devPlanWarnings []string

	// Check direct dev_plan reference on feature state first.
	if ref, _ := feat.State["dev_plan"].(string); ref != "" {
		_, dpResult, dpErr := s.docSvc.GetDocumentContent(ref)
		if dpErr == nil && dpResult.Status == string(model.DocumentStatusApproved) {
			devPlanDocID = dpResult.ID
		}
	}

	// Fall back to listing approved dev plans owned by this feature.
	if devPlanDocID == "" {
		docs, listErr := s.docSvc.ListDocuments(DocumentFilters{
			Owner:  input.FeatureID,
			Type:   "dev-plan",
			Status: string(model.DocumentStatusApproved),
		})
		if listErr == nil && len(docs) > 0 {
			latest := docs[0]
			for _, d := range docs[1:] {
				if d.Updated.After(latest.Updated) {
					latest = d
				}
			}
			devPlanDocID = latest.ID
		}
	}

	if devPlanDocID != "" {
		dpContent, _, dpErr := s.docSvc.GetDocumentContent(devPlanDocID)
		if dpErr == nil {
			tasks, ok := parseDevPlanTasks(featureSlug, []byte(dpContent))
			if ok {
				// Build proposal from dev-plan tasks.
				var sum float64
				allEstimated := len(tasks) > 0
				for _, t := range tasks {
					if t.Estimate != nil {
						sum += *t.Estimate
					} else {
						allEstimated = false
					}
				}
				var estimatedTotal *float64
				if allEstimated && len(tasks) > 0 {
					estimatedTotal = &sum
				}

				warnings := append(devPlanWarnings, fmt.Sprintf("Tasks sourced from dev-plan %s", devPlanDocID))
				if input.Context != "" {
					warnings = append(warnings, fmt.Sprintf("Additional orchestration context provided: %s", input.Context))
				}

				proposal := Proposal{
					Tasks:          tasks,
					TotalTasks:     len(tasks),
					EstimatedTotal: estimatedTotal,
					Slices:         identifySlices(spec),
					Warnings:       warnings,
				}
				guidance := deduplicateStrings([]string{
					"dev-plan-tasks",
					"size-soft-limit-8",
					"explicit-dependencies",
					"role-assignment",
				})

				// Slice enrichment runs on all paths.
				proposal.SliceDetails = analyzeSlices(spec, content)

				// Merge schedule check: warn when plan has >3 features and dev-plan lacks ## Merge Schedule.
				if parentPlan, _ := feat.State["parent"].(string); parentPlan != "" {
					feats, _ := s.entitySvc.List("feature")
					planFeatureCount := 0
					for _, f := range feats {
						if fp, _ := f.State["parent"].(string); fp == parentPlan {
							planFeatureCount++
						}
					}
					if planFeatureCount > 3 && !strings.Contains(dpContent, "## Merge Schedule") {
						proposal.Warnings = append(proposal.Warnings, fmt.Sprintf(
							"plan has %d features but dev-plan has no ## Merge Schedule section; "+
								"consider adding cohort groupings to prevent worktree drift",
							planFeatureCount,
						))
					}
				}

				return DecomposeResult{
					FeatureID:       feat.ID,
					FeatureSlug:     featureSlug,
					SpecDocumentID:  docResult.ID,
					Proposal:        proposal,
					GuidanceApplied: guidance,
				}, nil
			}
			// Parse failed: fall through with a warning.
			devPlanWarnings = append(devPlanWarnings, fmt.Sprintf(
				"dev-plan %s found but Task Breakdown absent or empty — falling back to AC heuristic",
				devPlanDocID,
			))
		}
	}

	// AC heuristic path.
	// Gate: spec must contain parseable acceptance criteria.
	if len(spec.acceptanceCriteria) == 0 {
		return DecomposeResult{}, fmt.Errorf("%s", buildZeroCriteriaDiagnostic(specDocID, content, spec))
	}

	// 7. Generate proposal by applying embedded guidance.
	cfg := config.LoadOrDefault()
	proposal, guidance := generateProposal(spec, featureSlug, input.Context, cfg.Decomposition.MaxTasksPerFeature)
	proposal.Warnings = append(devPlanWarnings, proposal.Warnings...)

	// Slice enrichment runs on all paths.
	proposal.SliceDetails = analyzeSlices(spec, content)

	return DecomposeResult{
		FeatureID:       feat.ID,
		FeatureSlug:     featureSlug,
		SpecDocumentID:  docResult.ID,
		Proposal:        proposal,
		GuidanceApplied: guidance,
	}, nil
}

// SliceAnalysis performs a standalone vertical slice analysis of a feature
// without committing to a decomposition. Returns candidate slices with
// outcomes, stack layers, size estimates, and inter-slice dependencies.
func (s *DecomposeService) SliceAnalysis(input SliceAnalysisInput) (SliceAnalysisResult, error) {
	if strings.TrimSpace(input.FeatureID) == "" {
		return SliceAnalysisResult{}, fmt.Errorf("feature_id is required")
	}

	feat, err := s.entitySvc.Get("feature", input.FeatureID, "")
	if err != nil {
		return SliceAnalysisResult{}, fmt.Errorf("load feature %s: %w", input.FeatureID, err)
	}

	featureSlug, _ := feat.State["slug"].(string)
	specDocID, _ := feat.State["spec"].(string)

	if specDocID == "" {
		return SliceAnalysisResult{}, fmt.Errorf("feature %s has no linked specification document", feat.ID)
	}

	content, _, err := s.docSvc.GetDocumentContent(specDocID)
	if err != nil {
		return SliceAnalysisResult{}, fmt.Errorf("load spec document %s: %w", specDocID, err)
	}

	spec := parseSpecStructure(content)
	slices := analyzeSlices(spec, content)

	var notes string
	if len(slices) == 0 {
		notes = "No candidate slices identified. The spec may lack level-2 section structure."
	} else if len(spec.acceptanceCriteria) == 0 {
		notes = "No acceptance criteria found; slices derived from section structure only."
	}

	return SliceAnalysisResult{
		FeatureID:     feat.ID,
		FeatureSlug:   featureSlug,
		Slices:        slices,
		TotalSlices:   len(slices),
		AnalysisNotes: notes,
	}, nil
}

// ReviewProposal checks a decomposition proposal against the feature's spec
// for gaps, oversized tasks, dependency cycles, and ambiguities.
func (s *DecomposeService) ReviewProposal(input DecomposeReviewInput) (DecomposeReviewResult, error) {
	if strings.TrimSpace(input.FeatureID) == "" {
		return DecomposeReviewResult{}, fmt.Errorf("feature_id is required")
	}

	// 1. Load the feature to get the spec reference.
	feat, err := s.entitySvc.Get("feature", input.FeatureID, "")
	if err != nil {
		return DecomposeReviewResult{}, fmt.Errorf("load feature %s: %w", input.FeatureID, err)
	}

	specDocID, _ := feat.State["spec"].(string)
	if specDocID == "" {
		return DecomposeReviewResult{}, fmt.Errorf("feature %s has no linked specification document", feat.ID)
	}

	// 2. Load spec content.
	content, _, err := s.docSvc.GetDocumentContent(specDocID)
	if err != nil {
		return DecomposeReviewResult{}, fmt.Errorf("load spec document %s: %w", specDocID, err)
	}

	// 3. Parse spec for acceptance criteria.
	spec := parseSpecStructure(content)

	// 4. Run all review checks.
	var findings []Finding
	findings = append(findings, checkGaps(spec, input.Proposal)...)
	findings = append(findings, checkOversized(input.Proposal)...)
	findings = append(findings, checkCycles(input.Proposal)...)
	findings = append(findings, checkAmbiguous(input.Proposal)...)
	findings = append(findings, checkDescriptionPresent(input.Proposal)...)
	findings = append(findings, checkTestingCoverage(input.Proposal)...)
	findings = append(findings, checkDependenciesDeclared(input.Proposal)...)
	findings = append(findings, checkOrphanTasks(input.Proposal)...)
	findings = append(findings, checkSingleAgentSizing(input.Proposal)...)

	// 5. Determine status.
	blockingCount := 0
	for _, f := range findings {
		if isBlockingFinding(f) {
			blockingCount++
		}
	}

	status := "pass"
	if blockingCount > 0 {
		status = "fail"
	} else if len(findings) > 0 {
		status = "warn"
	}

	return DecomposeReviewResult{
		FeatureID:     feat.ID,
		Status:        status,
		Findings:      findings,
		TotalFindings: len(findings),
		BlockingCount: blockingCount,
	}, nil
}

// ---------------------------------------------------------------------------
// Spec structure parsing
// ---------------------------------------------------------------------------

// specStructure holds the parsed structural elements of a spec document.
type specStructure struct {
	sections           []specSection
	acceptanceCriteria []acceptanceCriterion
}

type specSection struct {
	title string
	level int
}

type acceptanceCriterion struct {
	text     string
	section  string // enclosing section title (nearest header)
	parentL2 string // enclosing level-2 section title
}

var (
	// Matches markdown checkbox lines: "- [ ] text" or "- [x] text"
	reCheckbox = regexp.MustCompile(`(?m)^\s*-\s+\[[ xX]\]\s+(.+)$`)
	// Matches markdown headers: "## Title" through "##### Title"
	reHeader = regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	// Matches numbered acceptance criteria: "1. text", "2. text" etc.
	// Only within an "acceptance criteria" section.
	reNumbered = regexp.MustCompile(`(?m)^\s*\d+\.\s+(.+)$`)
	// Matches bold-identifier criteria: "**XX-NN.** text"
	// where XX is one or more uppercase ASCII letters and NN is one or more digits.
	// Group 1: identifier prefix (e.g. "AC"), Group 2: number (e.g. "01"), Group 3: criterion text.
	reBoldIdent = regexp.MustCompile(`^\*\*([A-Z]+)-(\d+)\.\*\*\s+(.+)$`)

	// Dev-plan parsing regexes.
	reDevPlanTaskBreakdown = regexp.MustCompile(`(?im)^## Task Breakdown\s*$`)
	reDevPlanTaskHeading   = regexp.MustCompile(`(?m)^### Task \d+: (.+)$`)
	reDevPlanNextL2        = regexp.MustCompile(`(?m)^## [^#].+$`)
	reDevPlanBoldField     = regexp.MustCompile(`(?m)^\s*[-*]\s+\*\*([^:]+):\*\*\s+(.+)$`)
	reDevPlanTaskRef       = regexp.MustCompile(`Task (\d+)`)
	// Matches bold-identifier prefix in parsed AC text: "AC-01: " etc.
	reBoldIdentPrefix = regexp.MustCompile(`^[A-Z]+-\d+: `)
)

// parseSpecStructure extracts sections and acceptance criteria from a
// markdown specification document.
//
// Recognised acceptance criterion formats:
//   - Checkbox: "- [ ] text" or "- [x] text" (anywhere in the document)
//   - Numbered list: "1. text" (only within acceptance-criteria sections)
//   - Bold-identifier: "**XX-NN.** text" (only within acceptance-criteria sections)
func parseSpecStructure(content string) specStructure {
	var spec specStructure
	lines := strings.Split(content, "\n")

	currentSection := ""
	currentL2 := ""
	inACSection := false
	inTableData := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if m := reHeader.FindStringSubmatch(trimmed); m != nil {
			level := len(m[1])
			title := strings.TrimSpace(m[2])
			spec.sections = append(spec.sections, specSection{title: title, level: level})
			currentSection = title
			if level == 2 {
				currentL2 = title
			}
			inACSection = isAcceptanceCriteriaSection(title)
			inTableData = false
			continue
		}

		if m := reCheckbox.FindStringSubmatch(trimmed); m != nil {
			spec.acceptanceCriteria = append(spec.acceptanceCriteria, acceptanceCriterion{
				text:     strings.TrimSpace(m[1]),
				section:  currentSection,
				parentL2: currentL2,
			})
			continue
		}

		if inACSection {
			// Table handling within AC sections.
			if strings.Contains(trimmed, "|") {
				if isTableSeparatorRow(trimmed) {
					inTableData = true
					continue
				}
				if inTableData {
					cells := parseTableRow(trimmed)
					if len(cells) > 0 {
						text := strings.Join(cells, " — ")
						spec.acceptanceCriteria = append(spec.acceptanceCriteria, acceptanceCriterion{
							text:     text,
							section:  currentSection,
							parentL2: currentL2,
						})
					}
					continue
				}
				// Table header row (before separator) — skip it.
				if strings.HasPrefix(trimmed, "|") {
					continue
				}
			} else {
				inTableData = false
			}

			// Bold-identifier pattern: **XX-NN.** text (also handles "- **XX-NN.** text").
			// Strip optional leading list marker before matching.
			bare := trimmed
			for _, pfx := range []string{"- ", "* ", "+ "} {
				if strings.HasPrefix(bare, pfx) {
					bare = bare[len(pfx):]
					break
				}
			}
			if m := reBoldIdent.FindStringSubmatch(bare); m != nil {
				criterion := m[1] + "-" + m[2] + ": " + m[3]
				spec.acceptanceCriteria = append(spec.acceptanceCriteria, acceptanceCriterion{
					text:     criterion,
					section:  currentSection,
					parentL2: currentL2,
				})
				continue
			}

			if m := reNumbered.FindStringSubmatch(trimmed); m != nil {
				spec.acceptanceCriteria = append(spec.acceptanceCriteria, acceptanceCriterion{
					text:     strings.TrimSpace(m[1]),
					section:  currentSection,
					parentL2: currentL2,
				})
			}
		}
	}

	return spec
}

// isAcceptanceCriteriaSection returns true if the section title looks like
// it contains acceptance criteria.
func isAcceptanceCriteriaSection(title string) bool {
	lower := strings.ToLower(title)
	return strings.Contains(lower, "acceptance criteria") ||
		strings.Contains(lower, "acceptance") ||
		strings.Contains(lower, "requirements") ||
		strings.Contains(lower, "criteria")
}

// buildZeroCriteriaDiagnostic returns a detailed error message when no
// acceptance criteria could be parsed from the spec. It reports section count
// and titles, and whether bold-identifier lines were found inside or outside
// Acceptance Criteria sections, with a concrete remediation suggestion.
func buildZeroCriteriaDiagnostic(specDocID string, content string, spec specStructure) string {
	var boldInside, boldOutside int
	inAC := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if m := reHeader.FindStringSubmatch(trimmed); m != nil {
			inAC = isAcceptanceCriteriaSection(strings.TrimSpace(m[2]))
			continue
		}
		bare := trimmed
		for _, pfx := range []string{"- ", "* ", "+ "} {
			if strings.HasPrefix(bare, pfx) {
				bare = bare[len(pfx):]
				break
			}
		}
		if reBoldIdent.MatchString(bare) {
			if inAC {
				boldInside++
			} else {
				boldOutside++
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("no acceptance criteria found in spec %q", specDocID))
	sb.WriteString(fmt.Sprintf("\n  sections (%d):", len(spec.sections)))
	for _, sec := range spec.sections {
		sb.WriteString(fmt.Sprintf("\n    - %s (level %d)", sec.title, sec.level))
	}
	if boldOutside > 0 {
		sb.WriteString(fmt.Sprintf("\n  found %d bold-identifier line(s) outside an Acceptance Criteria section", boldOutside))
		sb.WriteString("\n  remediation: move bold-identifier lines into a section with 'Acceptance Criteria' in the heading")
	} else {
		sb.WriteString("\n  no bold-identifier lines found anywhere in the document")
		sb.WriteString("\n  remediation: add an 'Acceptance Criteria' section with checkbox items (- [ ] ...), numbered items, or bold-identifier lines (**AC-NN.** text)")
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// Proposal generation
// ---------------------------------------------------------------------------

// generateProposal builds a task proposal from the parsed spec structure,
// applying the embedded decomposition guidance rules.
func generateProposal(spec specStructure, featureSlug, context string, maxTasksPerFeature int) (Proposal, []string) {
	var tasks []ProposedTask
	var warnings []string
	var appliedGuidance []string

	// Identify vertical slices from top-level sections (level 2 headers).
	slices := identifySlices(spec)
	if len(slices) > 0 {
		appliedGuidance = append(appliedGuidance, "vertical-slice-first")
	}

	if len(spec.acceptanceCriteria) > 0 {
		// Group ACs by parentL2 section for section-based grouping (guidance rule 2).
		type acGroup struct {
			parentL2 string
			acs      []acceptanceCriterion
		}
		seen := make(map[string]int) // parentL2 → index in groups
		var groups []acGroup
		for _, ac := range spec.acceptanceCriteria {
			if idx, ok := seen[ac.parentL2]; ok {
				groups[idx].acs = append(groups[idx].acs, ac)
			} else {
				seen[ac.parentL2] = len(groups)
				groups = append(groups, acGroup{parentL2: ac.parentL2, acs: []acceptanceCriterion{ac}})
			}
		}

		// Determine whether any group produces a merged task (2-4 ACs).
		anyGrouped := false
		for _, g := range groups {
			if len(g.acs) >= 2 && len(g.acs) <= 4 {
				anyGrouped = true
				break
			}
		}
		if anyGrouped {
			appliedGuidance = append(appliedGuidance, "group-by-section")
		} else {
			appliedGuidance = append(appliedGuidance, "one-ac-per-task")
		}

		taskIndex := 0
		for groupIndex, g := range groups {
			n := len(g.acs)
			sectionTitle := g.parentL2

			if n >= 2 && n <= 4 {
				// Grouped task: 2-4 ACs in one section → one task covering all.
				var slug string
				var summary string
				if sectionTitle == "" {
					slug = featureSlug + "-group-" + strconv.Itoa(groupIndex)
					summary = "Implement grouped criteria (" + strconv.Itoa(n) + " criteria)"
				} else {
					slug = featureSlug + "-" + slugify(sectionTitle)
					summary = "Implement " + sectionTitle + " (" + strconv.Itoa(n) + " criteria)"
				}
				var covers []string
				var rationaleLines []string
				for _, ac := range g.acs {
					covers = append(covers, ac.text)
					rationaleLines = append(rationaleLines, "- "+ac.text)
				}
				tasks = append(tasks, ProposedTask{
					Slug:      slug,
					Name:      deriveTaskName("Implement "+sectionTitle, "Implement grouped tasks"),
					Summary:   summary,
					Rationale: "Covers " + strconv.Itoa(n) + " acceptance criteria:\n" + strings.Join(rationaleLines, "\n"),
					Covers:    covers,
				})
			} else {
				// Individual tasks: 1 AC or 5+ ACs per section.
				for i, ac := range g.acs {
					slug := buildTaskSlug(featureSlug, ac.text, taskIndex+i)
					tasks = append(tasks, ProposedTask{
						Slug:    slug,
						Name:    deriveTaskName(ac.text, fmt.Sprintf("Implement AC-%03d", taskIndex+i+1)),
						Summary: ac.text,
						Rationale: fmt.Sprintf(
							"Covers acceptance criterion: %q (section: %s)",
							ac.text, sectionOrDefault(ac.section),
						),
						Covers: []string{ac.text},
					})
				}
			}
			taskIndex += n
		}
	}

	// Check if any tasks need a test companion (guidance rule 6).
	hasTestTask := false
	for _, t := range tasks {
		lower := strings.ToLower(t.Summary)
		if strings.Contains(lower, "test") {
			hasTestTask = true
			break
		}
	}
	if len(tasks) > 0 && !hasTestTask {
		tasks = append(tasks, ProposedTask{
			Slug:      featureSlug + "-tests",
			Name:      "Write tests",
			Summary:   "Write tests for " + featureSlug,
			Rationale: "Guidance rule: test tasks are explicit. No test task was found among proposed tasks.",
		})
		appliedGuidance = append(appliedGuidance, "test-tasks-explicit")
	}

	// Size soft limit is always applied (guidance rule 3) — we note it as
	// applied; actual flagging is done at review time since estimates are
	// optional in the initial proposal.
	appliedGuidance = append(appliedGuidance, "size-soft-limit-8")

	// Explicit dependencies is always applied as guidance (rule 4).
	appliedGuidance = append(appliedGuidance, "explicit-dependencies")

	// Role assignment is always reported as considered (rule 5).
	appliedGuidance = append(appliedGuidance, "role-assignment")

	// Add context-driven guidance note.
	if context != "" {
		warnings = append(warnings, fmt.Sprintf("Additional orchestration context provided: %s", context))
	}

	// Calculate estimated total.
	var estimatedTotal *float64
	allEstimated := len(tasks) > 0
	sum := 0.0
	for _, t := range tasks {
		if t.Estimate != nil {
			sum += *t.Estimate
		} else {
			allEstimated = false
		}
	}
	if allEstimated && len(tasks) > 0 {
		estimatedTotal = &sum
	}

	if maxTasksPerFeature > 0 && len(tasks) > maxTasksPerFeature {
		warnings = append(warnings, fmt.Sprintf(
			"proposal has %d tasks which exceeds decomposition.max_tasks_per_feature limit of %d; consider breaking into smaller features or increasing the limit",
			len(tasks), maxTasksPerFeature,
		))
	}

	return Proposal{
		Tasks:          tasks,
		TotalTasks:     len(tasks),
		EstimatedTotal: estimatedTotal,
		Slices:         slices,
		Warnings:       warnings,
	}, deduplicateStrings(appliedGuidance)
}

// identifySlices extracts vertical slice names from level-2 section headers,
// skipping meta sections like "Introduction" or "References".
func identifySlices(spec specStructure) []string {
	var slices []string
	skip := map[string]bool{
		"introduction": true, "overview": true, "references": true,
		"appendix": true, "glossary": true, "background": true,
		"purpose": true, "scope": true, "definitions": true,
	}
	for _, sec := range spec.sections {
		if sec.level == 2 {
			lower := strings.ToLower(sec.title)
			if !skip[lower] && !isAcceptanceCriteriaSection(sec.title) {
				slices = append(slices, sec.title)
			}
		}
	}
	return slices
}

// analyzeSlices performs rich slice analysis: outcomes, layers, estimates, dependencies.
func analyzeSlices(spec specStructure, content string) []AnalysisSlice {
	sliceNames := identifySlices(spec)
	if len(sliceNames) == 0 {
		return nil
	}

	sectionText := extractSectionContent(content)

	var slices []AnalysisSlice
	for _, name := range sliceNames {
		slice := AnalysisSlice{Name: name}

		for _, ac := range spec.acceptanceCriteria {
			if ac.parentL2 == name || ac.section == name {
				slice.Outcomes = append(slice.Outcomes, ac.text)
			}
		}

		text := sectionText[name]
		for _, o := range slice.Outcomes {
			text += " " + o
		}
		slice.Layers = detectLayers(text)
		slice.Estimate = estimateSliceSize(len(slice.Outcomes), len(slice.Layers))
		slice.Rationale = buildSliceRationale(slice)

		slices = append(slices, slice)
	}

	detectSliceDependencies(slices, sectionText)
	return slices
}

// extractSectionContent splits raw markdown by level-2 headers and returns
// the body text under each section, keyed by section title.
func extractSectionContent(content string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(content, "\n")

	var current string
	var body strings.Builder

	for _, line := range lines {
		if m := reHeader.FindStringSubmatch(line); m != nil && len(m[1]) == 2 {
			if current != "" {
				result[current] = body.String()
			}
			current = strings.TrimSpace(m[2])
			body.Reset()
		} else if current != "" {
			body.WriteString(line)
			body.WriteByte(' ')
		}
	}
	if current != "" {
		result[current] = body.String()
	}
	return result
}

var layerKeywords = map[string][]string{
	"storage": {"database", "db", "store", "persist", "table", "schema", "migration", "sql", "repository", "model", "record", "query"},
	"service": {"service", "logic", "process", "validate", "business", "rule", "domain", "compute", "calculate", "transform"},
	"api":     {"api", "endpoint", "route", "mcp", "tool", "handler", "request", "response", "http", "rest"},
	"cli":     {"cli", "command", "flag", "interface", "display", "output", "terminal", "prompt"},
}

func detectLayers(text string) []string {
	lower := strings.ToLower(text)
	var layers []string
	for layer, keywords := range layerKeywords {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				layers = append(layers, layer)
				break
			}
		}
	}
	order := map[string]int{"storage": 0, "service": 1, "api": 2, "cli": 3}
	for i := 0; i < len(layers)-1; i++ {
		for j := i + 1; j < len(layers); j++ {
			if order[layers[i]] > order[layers[j]] {
				layers[i], layers[j] = layers[j], layers[i]
			}
		}
	}
	return layers
}

func estimateSliceSize(outcomes, layers int) string {
	if outcomes >= 4 || layers >= 3 {
		return "large"
	}
	if outcomes >= 2 || layers >= 2 {
		return "medium"
	}
	return "small"
}

func buildSliceRationale(s AnalysisSlice) string {
	var parts []string
	if len(s.Outcomes) > 0 {
		parts = append(parts, fmt.Sprintf("%d acceptance criteria", len(s.Outcomes)))
	} else {
		parts = append(parts, "no explicit acceptance criteria; derived from section structure")
	}
	if len(s.Layers) > 0 {
		parts = append(parts, fmt.Sprintf("touches %s", strings.Join(s.Layers, ", ")))
	}
	return strings.Join(parts, "; ")
}

func detectSliceDependencies(slices []AnalysisSlice, sectionText map[string]string) {
	for i := range slices {
		text := strings.ToLower(sectionText[slices[i].Name])
		for _, o := range slices[i].Outcomes {
			text += " " + strings.ToLower(o)
		}
		for j := range slices {
			if i == j {
				continue
			}
			nameLower := strings.ToLower(slices[j].Name)
			if strings.Contains(text, nameLower) {
				slices[i].DependsOn = append(slices[i].DependsOn, slices[j].Name)
			}
		}
	}
}

// buildTaskSlug creates a slug for a proposed task from the feature slug
// and a text description.
func buildTaskSlug(featureSlug, text string, index int) string {
	slug := slugify(text)
	if slug == "" {
		slug = fmt.Sprintf("task-%d", index+1)
	}
	// Keep slugs reasonably short.
	if len(slug) > 40 {
		slug = slug[:40]
		// Trim trailing hyphens from truncation.
		slug = strings.TrimRight(slug, "-")
	}
	return featureSlug + "-" + slug
}

// slugify converts text to a URL-friendly slug.
func slugify(text string) string {
	lower := strings.ToLower(text)
	// Replace non-alphanumeric characters with hyphens.
	var b strings.Builder
	prevHyphen := false
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen && b.Len() > 0 {
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	result := b.String()
	return strings.TrimRight(result, "-")
}

// parseDevPlanTasks parses the "## Task Breakdown" section of a dev plan
// document and returns the tasks defined there. Returns (nil, false) if the
// section is absent or contains no task headings (NFR-003: unexported).
func parseDevPlanTasks(featureSlug string, content []byte) ([]ProposedTask, bool) {
	text := string(content)

	// 1. Find ## Task Breakdown heading (case-insensitive).
	loc := reDevPlanTaskBreakdown.FindStringIndex(text)
	if loc == nil {
		return nil, false
	}

	// 2. Extract body from heading to next ##-level heading (or EOF).
	bodyStart := loc[1]
	remaining := text[bodyStart:]
	var body string
	nextL2 := reDevPlanNextL2.FindStringIndex(remaining)
	if nextL2 != nil {
		body = remaining[:nextL2[0]]
	} else {
		body = remaining
	}

	// 3. Find task headings.
	taskMatches := reDevPlanTaskHeading.FindAllStringSubmatchIndex(body, -1)
	if len(taskMatches) == 0 {
		return nil, false
	}

	// Extract task titles and pre-compute slugs (needed for dependency resolution).
	titles := make([]string, len(taskMatches))
	for i, m := range taskMatches {
		titles[i] = strings.TrimSpace(body[m[2]:m[3]])
	}
	slugs := make([]string, len(titles))
	for i, title := range titles {
		slugs[i] = featureSlug + "-" + slugify(title)
	}

	// 4. Extract each task block and parse fields.
	var tasks []ProposedTask
	for i, m := range taskMatches {
		title := titles[i]
		blockStart := m[1] // end of the ### heading line
		var blockEnd int
		if i+1 < len(taskMatches) {
			blockEnd = taskMatches[i+1][0] // start of next ### heading
		} else {
			blockEnd = len(body)
		}
		block := body[blockStart:blockEnd]

		task := ProposedTask{
			Slug:      slugs[i],
			Name:      title,
			Summary:   title,
			Rationale: fmt.Sprintf("Sourced from dev-plan task %d", i+1),
		}

		// 5. Parse bolded fields: Depends on, Effort, Spec requirements.
		fieldMatches := reDevPlanBoldField.FindAllStringSubmatch(block, -1)
		for _, fm := range fieldMatches {
			key := strings.TrimSpace(fm[1])
			value := strings.TrimSpace(fm[2])
			switch key {
			case "Effort":
				var est float64
				switch value {
				case "Small":
					est = 1.0
				case "Medium":
					est = 3.0
				case "Large":
					est = 8.0
				default:
					continue
				}
				task.Estimate = &est
			case "Spec requirements":
				parts := strings.Split(value, ",")
				var covers []string
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						covers = append(covers, p)
					}
				}
				if len(covers) > 0 {
					task.Covers = covers
				}
			case "Depends on":
				// Resolve dependency slugs from "Task N" references.
				// "None" or "None (...)" -> nil.
				lower := strings.ToLower(value)
				if strings.HasPrefix(lower, "none") {
					task.DependsOn = nil
				} else {
					refs := reDevPlanTaskRef.FindAllStringSubmatch(value, -1)
					var deps []string
					for _, ref := range refs {
						idx, err := strconv.Atoi(ref[1])
						if err == nil && idx >= 1 && idx <= len(slugs) {
							deps = append(deps, slugs[idx-1])
						}
					}
					if len(deps) > 0 {
						task.DependsOn = deps
					}
				}
			}
		}

		tasks = append(tasks, task)
	}

	if len(tasks) == 0 {
		return nil, false
	}
	return tasks, true
}

// deriveTaskName produces a non-empty task name from text, falling back to
// fallback when the candidate is empty after processing.
//
// Processing steps:
//  1. Strip a bold-ident prefix (e.g. "AC-01: ") if present.
//  2. Trim surrounding whitespace.
//  3. Truncate to 60 characters at a word boundary where possible.
//  4. Return fallback if the result is still empty.
func deriveTaskName(text, fallback string) string {
	candidate := reBoldIdentPrefix.ReplaceAllString(text, "")
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return fallback
	}
	if len(candidate) > 60 {
		truncated := candidate[:60]
		if idx := strings.LastIndex(truncated, " "); idx > 0 {
			truncated = truncated[:idx]
		}
		candidate = strings.TrimSpace(truncated)
	}
	if candidate == "" {
		return fallback
	}
	return candidate
}

func sectionOrDefault(section string) string {
	if section == "" {
		return "top-level"
	}
	return section
}

// ---------------------------------------------------------------------------
// Review checks
// ---------------------------------------------------------------------------

const oversizedThreshold = 8.0

// checkGaps detects acceptance criteria in the spec that are not covered
// by any task in the proposal.
func checkGaps(spec specStructure, proposal Proposal) []Finding {
	var findings []Finding
	for _, ac := range spec.acceptanceCriteria {
		if !isACCovered(ac, proposal.Tasks) {
			findings = append(findings, Finding{
				Type:     "gap",
				Severity: "error",
				Detail:   fmt.Sprintf("Uncovered acceptance criterion: %q (section: %s)", ac.text, sectionOrDefault(ac.section)),
			})
		}
	}
	return findings
}

// isACCovered returns true if any proposed task appears to cover the given
// acceptance criterion. When a task has a non-empty Covers slice, exact string
// match is used; otherwise keyword overlap is used as a fallback heuristic.
func isACCovered(ac acceptanceCriterion, tasks []ProposedTask) bool {
	acWords := significantWords(ac.text)
	if len(acWords) == 0 {
		return true // vacuous criterion
	}
	for _, task := range tasks {
		// Exact match via Covers field (populated by section-based grouping).
		if len(task.Covers) > 0 {
			for _, covered := range task.Covers {
				if covered == ac.text {
					return true
				}
			}
			// Has Covers but this AC didn't match; don't fall back to heuristic.
			continue
		}
		// Legacy fallback: keyword overlap heuristic.
		combined := strings.ToLower(task.Summary + " " + task.Rationale)
		matched := 0
		for _, w := range acWords {
			if strings.Contains(combined, w) {
				matched++
			}
		}
		// Require at least two-thirds of significant words to match.
		if matched*3 >= len(acWords)*2 {
			return true
		}
	}
	return false
}

// isTableSeparatorRow returns true if the line is a markdown table separator
// row (e.g. "| --- | --- |" or "| :--- | ---: |").
func isTableSeparatorRow(line string) bool {
	if !strings.Contains(line, "|") {
		return false
	}
	cells := strings.Split(line, "|")
	sepCount := 0
	for _, cell := range cells {
		trimmed := strings.TrimSpace(cell)
		if trimmed == "" {
			continue
		}
		// Cell must be only hyphens with optional leading/trailing colons.
		cleaned := strings.Trim(trimmed, ":")
		if cleaned == "" {
			continue
		}
		allHyphens := true
		for _, r := range cleaned {
			if r != '-' {
				allHyphens = false
				break
			}
		}
		if allHyphens && len(cleaned) >= 1 {
			sepCount++
		}
	}
	return sepCount > 0
}

// parseTableRow extracts non-empty cell values from a markdown table data row.
func parseTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		cell := strings.TrimSpace(p)
		if cell != "" {
			cells = append(cells, cell)
		}
	}
	return cells
}

// significantWords extracts lowercase words of 4+ characters, skipping
// common stop words.
func significantWords(text string) []string {
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "has": true,
		"her": true, "was": true, "one": true, "our": true, "out": true,
		"with": true, "this": true, "that": true, "from": true, "they": true,
		"been": true, "have": true, "many": true, "some": true, "them": true,
		"than": true, "each": true, "make": true, "like": true, "when": true,
		"will": true, "must": true, "should": true, "shall": true, "also": true,
	}

	words := strings.Fields(strings.ToLower(text))
	var result []string
	for _, w := range words {
		// Strip punctuation.
		w = strings.Trim(w, ".,;:!?\"'`()[]{}#*-_/\\")
		if len(w) >= 4 && !stopWords[w] {
			result = append(result, w)
		}
	}
	return result
}

// checkOversized detects tasks with estimates above the soft limit.
func checkOversized(proposal Proposal) []Finding {
	var findings []Finding
	for _, task := range proposal.Tasks {
		if task.Estimate != nil && *task.Estimate > oversizedThreshold {
			findings = append(findings, Finding{
				Type:     "oversized",
				Severity: "warning",
				TaskSlug: task.Slug,
				Detail:   fmt.Sprintf("Task %q estimated at %.0f points (soft limit: %.0f)", task.Slug, *task.Estimate, oversizedThreshold),
			})
		}
	}
	return findings
}

// checkCycles detects dependency cycles within the proposal using DFS.
func checkCycles(proposal Proposal) []Finding {
	// Build adjacency from slug → depends_on slugs.
	adj := make(map[string][]string)
	slugSet := make(map[string]bool)
	for _, task := range proposal.Tasks {
		slugSet[task.Slug] = true
		if len(task.DependsOn) > 0 {
			adj[task.Slug] = task.DependsOn
		}
	}

	const (
		white = 0 // unvisited
		gray  = 1 // in current path
		black = 2 // fully explored
	)
	color := make(map[string]int)
	var cycleNodes []string

	var dfs func(node string) bool
	dfs = func(node string) bool {
		color[node] = gray
		for _, dep := range adj[node] {
			if !slugSet[dep] {
				continue // skip references to unknown slugs
			}
			switch color[dep] {
			case gray:
				cycleNodes = append(cycleNodes, node, dep)
				return true
			case white:
				if dfs(dep) {
					return true
				}
			}
		}
		color[node] = black
		return false
	}

	var findings []Finding
	for _, task := range proposal.Tasks {
		if color[task.Slug] == white {
			cycleNodes = nil
			if dfs(task.Slug) {
				findings = append(findings, Finding{
					Type:     "cycle",
					Severity: "error",
					Detail:   fmt.Sprintf("Dependency cycle detected involving: %s", strings.Join(cycleNodes, " → ")),
				})
				// Reset to find additional cycles.
				for k := range color {
					if color[k] == gray {
						color[k] = white
					}
				}
			}
		}
	}
	return findings
}

// checkAmbiguous detects tasks with very short or generic summaries.
func checkAmbiguous(proposal Proposal) []Finding {
	var findings []Finding
	for _, task := range proposal.Tasks {
		summary := strings.TrimSpace(task.Summary)
		if len(summary) < 10 {
			findings = append(findings, Finding{
				Type:     "ambiguous",
				Severity: "warning",
				TaskSlug: task.Slug,
				Detail:   fmt.Sprintf("Task %q has a very short summary (%d characters)", task.Slug, len(summary)),
			})
		}
	}
	return findings
}

// isBlockingFinding returns true for findings that should block confirmation
// of a proposal. Blocking is determined by severity, not type.
func isBlockingFinding(f Finding) bool {
	return f.Severity == "error"
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func deduplicateStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	var result []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
