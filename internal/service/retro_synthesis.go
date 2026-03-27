// Package service retro_synthesis.go — Phase 2 retrospective synthesis service.
//
// RetroService reads accumulated retrospective signals from the knowledge store,
// clusters them by category and textual similarity, ranks themes by severity-
// weighted signal count, and returns a structured synthesis response.
//
// Report mode additionally generates a markdown document and registers it as a
// document record using the DocumentService.
//
// See work/spec/workflow-retrospective.md §7 for the full behaviour specification.
package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"kanbanzai/internal/knowledge"
	"kanbanzai/internal/storage"
)

// severityWeight returns the numeric weight for a severity level per spec §5.3.
// significant=5, moderate=3, minor=1.
func severityWeight(sev string) int {
	switch sev {
	case "significant":
		return 5
	case "moderate":
		return 3
	default: // "minor" and unknown values
		return 1
	}
}

// parsedRetroSignal holds fields extracted from a retrospective knowledge entry.
type parsedRetroSignal struct {
	EntryID         string
	Category        string
	Severity        string
	Content         string // full content string (used for Jaccard clustering)
	Observation     string
	Suggestion      string
	LearnedFrom     string
	RelatedDecision string
	Created         time.Time
}

// parseRetroRecord attempts to parse a knowledge record into a parsedRetroSignal.
// Returns ok=false when the record cannot be parsed as a valid retrospective signal.
func parseRetroRecord(rec storage.KnowledgeRecord) (parsedRetroSignal, bool) {
	tags := knowledgeFieldStrings(rec.Fields, "tags")
	category := ""
	for _, t := range tags {
		if ValidRetroCategories[t] {
			category = t
			break
		}
	}
	if category == "" {
		return parsedRetroSignal{}, false
	}

	content, _ := rec.Fields["content"].(string)
	if content == "" {
		return parsedRetroSignal{}, false
	}

	severity := parseSeverityFromContent(content)
	if severity == "" {
		return parsedRetroSignal{}, false
	}

	observation, suggestion := parseObservationAndSuggestion(content, category)
	learnedFrom, _ := rec.Fields["learned_from"].(string)
	related := parseRelatedDecision(content)
	createdStr, _ := rec.Fields["created"].(string)
	created, _ := time.Parse(time.RFC3339, createdStr)

	return parsedRetroSignal{
		EntryID:         rec.ID,
		Category:        category,
		Severity:        severity,
		Content:         content,
		Observation:     observation,
		Suggestion:      suggestion,
		LearnedFrom:     learnedFrom,
		RelatedDecision: related,
		Created:         created,
	}, true
}

// parseSeverityFromContent extracts the severity from a content string with
// the format "[{severity}] {category}: ...".
func parseSeverityFromContent(content string) string {
	if !strings.HasPrefix(content, "[") {
		return ""
	}
	end := strings.Index(content, "]")
	if end < 0 {
		return ""
	}
	sev := content[1:end]
	if ValidRetroSeverities[sev] {
		return sev
	}
	return ""
}

// parseRelatedDecision extracts the decision ID from a retrospective signal
// content string. The format is: "... Related: {decision_id}" at end of string.
func parseRelatedDecision(content string) string {
	const marker = " Related: "
	i := strings.LastIndex(content, marker)
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(content[i+len(marker):])
}

// parseObservationAndSuggestion extracts the observation and suggestion from
// a retrospective signal content string. The format after the "[severity] category: "
// prefix is: {observation}[ Suggestion: {suggestion}][ Related: {decision_id}]
func parseObservationAndSuggestion(content, category string) (observation, suggestion string) {
	prefix := "] " + category + ": "
	idx := strings.Index(content, prefix)
	if idx < 0 {
		return content, ""
	}
	rest := content[idx+len(prefix):]

	if i := strings.Index(rest, " Suggestion: "); i >= 0 {
		observation = rest[:i]
		tail := rest[i+len(" Suggestion: "):]
		if j := strings.Index(tail, " Related: "); j >= 0 {
			suggestion = tail[:j]
		} else {
			suggestion = tail
		}
	} else if i := strings.Index(rest, " Related: "); i >= 0 {
		observation = rest[:i]
	} else {
		observation = rest
	}
	return observation, suggestion
}

// ─── Response types ───────────────────────────────────────────────────────────

// RetroPeriod is the time range covered by the synthesis.
type RetroPeriod struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// RetroTheme is a ranked cluster of related retrospective signals.
type RetroTheme struct {
	Rank                      int      `json:"rank"`
	Category                  string   `json:"category"`
	Title                     string   `json:"title"`
	SignalCount               int      `json:"signal_count"`
	SeverityScore             int      `json:"severity_score"`
	Signals                   []string `json:"signals"`
	TopSuggestion             string   `json:"top_suggestion,omitempty"`
	RepresentativeObservation string   `json:"representative_observation"`
}

// RetroWorkedWell is an entry in the worked_well section of the synthesis response.
type RetroWorkedWell struct {
	Title                     string `json:"title"`
	SignalCount               int    `json:"signal_count"`
	RepresentativeObservation string `json:"representative_observation"`
}

// RetroExperiment reports how a workflow-experiment decision is performing
// based on signals that reference it. See spec §8.3.
type RetroExperiment struct {
	DecisionID      string `json:"decision_id"`
	Title           string `json:"title"`
	PositiveSignals int    `json:"positive_signals"`
	NegativeSignals int    `json:"negative_signals"`
	NetAssessment   string `json:"net_assessment"`
	Recommendation  string `json:"recommendation"`
}

// RetroSynthesisResult is the structured response from retro synthesis (spec §7.3).
type RetroSynthesisResult struct {
	Scope       string            `json:"scope"`
	SignalCount int               `json:"signal_count"`
	Period      RetroPeriod       `json:"period"`
	Themes      []RetroTheme      `json:"themes"`
	WorkedWell  []RetroWorkedWell `json:"worked_well,omitempty"`
	Experiments []RetroExperiment `json:"experiments,omitempty"`
}

// RetroReportInfo contains metadata about a generated retrospective report document.
type RetroReportInfo struct {
	Path       string `json:"path"`
	DocumentID string `json:"document_id"`
}

// RetroReportResult extends RetroSynthesisResult with generated report metadata (spec §7.4).
type RetroReportResult struct {
	RetroSynthesisResult
	Report RetroReportInfo `json:"report"`
}

// ─── Input types ─────────────────────────────────────────────────────────────

// RetroSynthesisInput holds parameters for the retro synthesise action.
type RetroSynthesisInput struct {
	Scope       string // "project" (default), Plan ID, or Feature ID
	Since       string // ISO 8601 timestamp (optional)
	Until       string // ISO 8601 timestamp (optional)
	MinSeverity string // "minor" (default), "moderate", or "significant"
}

// RetroReportInput extends synthesis input with report-specific parameters.
type RetroReportInput struct {
	RetroSynthesisInput
	OutputPath string // repository-relative path for the generated markdown file
	Title      string // document title; defaults to "Retrospective: {scope} {date}"
	CreatedBy  string // identity of the caller (for document registration)
}

// ─── Service ──────────────────────────────────────────────────────────────────

// RetroService provides retrospective signal synthesis (Phase 2 of P5).
type RetroService struct {
	knowledgeSvc *KnowledgeService
	entitySvc    *EntityService
	docSvc       *DocumentService
	repoRoot     string
	now          func() time.Time
}

// NewRetroService creates a RetroService.
func NewRetroService(
	knowledgeSvc *KnowledgeService,
	entitySvc *EntityService,
	docSvc *DocumentService,
	repoRoot string,
) *RetroService {
	if repoRoot == "" {
		repoRoot = "."
	}
	return &RetroService{
		knowledgeSvc: knowledgeSvc,
		entitySvc:    entitySvc,
		docSvc:       docSvc,
		repoRoot:     repoRoot,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

// Synthesise loads retrospective signals, applies filters, clusters by category
// and similarity, and returns a ranked synthesis response per spec §7.2–§7.3.
func (s *RetroService) Synthesise(input RetroSynthesisInput) (RetroSynthesisResult, error) {
	scope := strings.TrimSpace(input.Scope)
	if scope == "" {
		scope = "project"
	}

	minSeverity := strings.TrimSpace(input.MinSeverity)
	if minSeverity == "" {
		minSeverity = "minor"
	}
	if !ValidRetroSeverities[minSeverity] {
		return RetroSynthesisResult{}, fmt.Errorf(
			"invalid min_severity %q; valid values: minor, moderate, significant", minSeverity,
		)
	}
	minWeight := severityWeight(minSeverity)

	var since, until time.Time
	if raw := strings.TrimSpace(input.Since); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return RetroSynthesisResult{}, fmt.Errorf("invalid since timestamp: %w", err)
		}
		since = t
	}
	if raw := strings.TrimSpace(input.Until); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return RetroSynthesisResult{}, fmt.Errorf("invalid until timestamp: %w", err)
		}
		until = t
	}

	// Build the set of task IDs that satisfy the scope constraint (nil = project scope).
	taskSet, err := s.buildScopeTaskSet(scope)
	if err != nil {
		return RetroSynthesisResult{}, fmt.Errorf("resolve scope %q: %w", scope, err)
	}

	// Load all retrospective-tagged knowledge entries.
	allRecords, err := s.knowledgeSvc.List(KnowledgeFilters{
		Tags: []string{"retrospective"},
	})
	if err != nil {
		return RetroSynthesisResult{}, fmt.Errorf("load retrospective signals: %w", err)
	}

	// Parse and apply all filters.
	var signals []parsedRetroSignal
	for _, rec := range allRecords {
		sig, ok := parseRetroRecord(rec)
		if !ok {
			continue
		}

		// Scope filter: standalone signals (no learned_from) only included for project scope.
		if taskSet != nil {
			if sig.LearnedFrom == "" || !taskSet[sig.LearnedFrom] {
				continue
			}
		}

		// Date filters.
		if !since.IsZero() && sig.Created.Before(since) {
			continue
		}
		if !until.IsZero() && sig.Created.After(until) {
			continue
		}

		// Severity filter.
		if severityWeight(sig.Severity) < minWeight {
			continue
		}

		signals = append(signals, sig)
	}

	now := s.now()

	if len(signals) == 0 {
		return RetroSynthesisResult{
			Scope:       scope,
			SignalCount: 0,
			Period: RetroPeriod{
				From: now.Format(time.RFC3339),
				To:   now.Format(time.RFC3339),
			},
			Themes: []RetroTheme{},
		}, nil
	}

	// Compute time range of the matching signals.
	earliest := signals[0].Created
	latest := signals[0].Created
	for _, sig := range signals[1:] {
		if sig.Created.Before(earliest) {
			earliest = sig.Created
		}
		if sig.Created.After(latest) {
			latest = sig.Created
		}
	}

	// Separate worked-well signals from negative-category signals.
	var negative, workedWellSigs []parsedRetroSignal
	for _, sig := range signals {
		if sig.Category == "worked-well" {
			workedWellSigs = append(workedWellSigs, sig)
		} else {
			negative = append(negative, sig)
		}
	}

	themes := buildRetroThemes(negative)
	workedWell := buildRetroWorkedWell(workedWellSigs)
	experiments := s.buildExperiments(signals)

	return RetroSynthesisResult{
		Scope:       scope,
		SignalCount: len(signals),
		Period: RetroPeriod{
			From: earliest.Format(time.RFC3339),
			To:   latest.Format(time.RFC3339),
		},
		Themes:      themes,
		WorkedWell:  workedWell,
		Experiments: experiments,
	}, nil
}

// Report runs synthesis, generates a markdown document, writes it to OutputPath,
// and registers it as a document record. Returns the synthesis result extended
// with report metadata per spec §7.4.
func (s *RetroService) Report(input RetroReportInput) (RetroReportResult, error) {
	if strings.TrimSpace(input.OutputPath) == "" {
		return RetroReportResult{}, fmt.Errorf("output_path is required for report mode")
	}

	synthesis, err := s.Synthesise(input.RetroSynthesisInput)
	if err != nil {
		return RetroReportResult{}, err
	}

	scope := strings.TrimSpace(input.Scope)
	if scope == "" {
		scope = "project"
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = fmt.Sprintf("Retrospective: %s %s", scope, s.now().Format("2006-01-02"))
	}

	markdown := renderRetroMarkdown(title, synthesis)

	fullPath := filepath.Join(s.repoRoot, input.OutputPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return RetroReportResult{}, fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(fullPath, []byte(markdown), 0644); err != nil {
		return RetroReportResult{}, fmt.Errorf("write report file: %w", err)
	}

	createdBy := strings.TrimSpace(input.CreatedBy)
	if createdBy == "" {
		createdBy = "retro"
	}

	docResult, err := s.docSvc.SubmitDocument(SubmitDocumentInput{
		Path:      input.OutputPath,
		Type:      "report",
		Title:     title,
		CreatedBy: createdBy,
	})
	if err != nil {
		return RetroReportResult{}, fmt.Errorf("register report document: %w", err)
	}

	return RetroReportResult{
		RetroSynthesisResult: synthesis,
		Report: RetroReportInfo{
			Path:       input.OutputPath,
			DocumentID: docResult.ID,
		},
	}, nil
}

// ─── Scope resolution ─────────────────────────────────────────────────────────

// buildScopeTaskSet returns the set of task IDs in scope. Returns nil for
// "project" scope (no task-level filtering). Returns an empty map if the
// Plan/Feature ID exists but has no tasks.
func (s *RetroService) buildScopeTaskSet(scope string) (map[string]bool, error) {
	if scope == "project" {
		return nil, nil
	}

	tasks, err := s.entitySvc.List("task")
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	taskSet := make(map[string]bool)

	if strings.HasPrefix(scope, "FEAT-") {
		// Feature scope: include tasks whose parent_feature matches.
		for _, t := range tasks {
			pf, _ := t.State["parent_feature"].(string)
			if pf == scope {
				taskSet[t.ID] = true
			}
		}
		return taskSet, nil
	}

	// Plan scope: find all features in the plan, then collect their tasks.
	features, err := s.entitySvc.List("feature")
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}
	featureSet := make(map[string]bool)
	for _, f := range features {
		parent, _ := f.State["parent"].(string)
		if parent == scope {
			featureSet[f.ID] = true
		}
	}
	for _, t := range tasks {
		pf, _ := t.State["parent_feature"].(string)
		if featureSet[pf] {
			taskSet[t.ID] = true
		}
	}
	return taskSet, nil
}

// ─── Clustering and ranking ───────────────────────────────────────────────────

// retroCluster holds a group of similar signals within a category.
type retroCluster struct {
	signals       []parsedRetroSignal
	centroidWords map[string]struct{}
}

// buildRetroThemes groups negative-category signals by category, clusters each
// group by Jaccard similarity, ranks by severity-weighted score, and returns
// ordered RetroTheme values.
func buildRetroThemes(signals []parsedRetroSignal) []RetroTheme {
	if len(signals) == 0 {
		return []RetroTheme{}
	}

	byCategory := make(map[string][]parsedRetroSignal)
	for _, sig := range signals {
		byCategory[sig.Category] = append(byCategory[sig.Category], sig)
	}

	// Sort categories for deterministic output.
	categories := make([]string, 0, len(byCategory))
	for cat := range byCategory {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	var allClusters []retroCluster
	for _, cat := range categories {
		allClusters = append(allClusters, clusterRetroSignals(byCategory[cat])...)
	}

	// Rank: descending by cluster_score = signal_count × max_severity_weight,
	// then descending by signal_count as tiebreaker (spec §7.2).
	sort.SliceStable(allClusters, func(i, j int) bool {
		si := retroClusterScore(allClusters[i])
		sj := retroClusterScore(allClusters[j])
		if si != sj {
			return si > sj
		}
		return len(allClusters[i].signals) > len(allClusters[j].signals)
	})

	themes := make([]RetroTheme, len(allClusters))
	for i, c := range allClusters {
		themes[i] = retroClusterToTheme(i+1, c)
	}
	return themes
}

// buildExperiments cross-references signals with decision entities tagged
// workflow-experiment and returns experiment tracking entries. Returns nil
// when no signals reference any decision ID (P5-3.7).
func (s *RetroService) buildExperiments(signals []parsedRetroSignal) []RetroExperiment {
	// Group signals by their RelatedDecision.
	byDecision := make(map[string][]parsedRetroSignal)
	for _, sig := range signals {
		if sig.RelatedDecision != "" {
			byDecision[sig.RelatedDecision] = append(byDecision[sig.RelatedDecision], sig)
		}
	}
	if len(byDecision) == 0 {
		return nil
	}

	if s.entitySvc == nil {
		return nil
	}

	// Build a lookup of decision entities tagged workflow-experiment.
	decisions, err := s.entitySvc.List("decision")
	if err != nil {
		return nil
	}
	decisionTitles := make(map[string]string)
	for _, d := range decisions {
		tags, _ := d.State["tags"].([]any)
		for _, t := range tags {
			if s, ok := t.(string); ok && s == "workflow-experiment" {
				summary, _ := d.State["summary"].(string)
				decisionTitles[d.ID] = summary
				break
			}
		}
	}

	// Sort decision IDs for deterministic output.
	ids := make([]string, 0, len(byDecision))
	for id := range byDecision {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var experiments []RetroExperiment
	for _, id := range ids {
		title, ok := decisionTitles[id]
		if !ok {
			// Not a workflow-experiment decision; skip.
			continue
		}

		sigs := byDecision[id]
		positive, negative := 0, 0
		for _, sig := range sigs {
			if sig.Category == "worked-well" {
				positive++
			} else {
				negative++
			}
		}

		var recommendation string
		if positive > negative {
			recommendation = "keep"
		} else if positive == 0 && negative > 0 {
			recommendation = "revert"
		} else {
			recommendation = "revise"
		}

		assessment := fmt.Sprintf("%d positive, %d negative", positive, negative)

		experiments = append(experiments, RetroExperiment{
			DecisionID:      id,
			Title:           title,
			PositiveSignals: positive,
			NegativeSignals: negative,
			NetAssessment:   assessment,
			Recommendation:  recommendation,
		})
	}

	if len(experiments) == 0 {
		return nil
	}
	return experiments
}

// buildRetroWorkedWell clusters worked-well signals and returns summary entries.
func buildRetroWorkedWell(signals []parsedRetroSignal) []RetroWorkedWell {
	if len(signals) == 0 {
		return nil
	}
	clusters := clusterRetroSignals(signals)
	entries := make([]RetroWorkedWell, len(clusters))
	for i, c := range clusters {
		rep := retroRepresentative(c.signals)
		entries[i] = RetroWorkedWell{
			Title:                     retroThemeTitle("worked-well", rep.Observation),
			SignalCount:               len(c.signals),
			RepresentativeObservation: rep.Observation,
		}
	}
	return entries
}

// clusterRetroSignals groups signals using greedy Jaccard similarity (threshold 0.5).
// Each signal either joins the first existing cluster whose centroid similarity
// is >= 0.5, or starts a new singleton cluster.
func clusterRetroSignals(signals []parsedRetroSignal) []retroCluster {
	if len(signals) == 0 {
		return nil
	}

	// Pre-compute word sets for each signal.
	wordSets := make([]map[string]struct{}, len(signals))
	for i, sig := range signals {
		wordSets[i] = knowledge.ContentWords(sig.Content)
	}

	clusters := make([]retroCluster, 0, len(signals))
	for i, sig := range signals {
		placed := false
		for j := range clusters {
			if knowledge.JaccardSimilarity(wordSets[i], clusters[j].centroidWords) >= 0.5 {
				clusters[j].signals = append(clusters[j].signals, sig)
				// Expand the centroid to include all tokens from the new signal.
				for w := range wordSets[i] {
					clusters[j].centroidWords[w] = struct{}{}
				}
				placed = true
				break
			}
		}
		if !placed {
			c := retroCluster{
				signals:       []parsedRetroSignal{sig},
				centroidWords: make(map[string]struct{}, len(wordSets[i])),
			}
			for w := range wordSets[i] {
				c.centroidWords[w] = struct{}{}
			}
			clusters = append(clusters, c)
		}
	}
	return clusters
}

// retroClusterScore computes signal_count × max_severity_weight per spec §7.2.
func retroClusterScore(c retroCluster) int {
	maxWeight := 0
	for _, sig := range c.signals {
		if w := severityWeight(sig.Severity); w > maxWeight {
			maxWeight = w
		}
	}
	return len(c.signals) * maxWeight
}

// retroRepresentative returns the signal with the highest severity weight
// (first signal wins ties), used as the representative observation for a cluster.
func retroRepresentative(signals []parsedRetroSignal) parsedRetroSignal {
	best := signals[0]
	for _, sig := range signals[1:] {
		if severityWeight(sig.Severity) > severityWeight(best.Severity) {
			best = sig
		}
	}
	return best
}

// retroThemeTitle generates a short informational title for a theme cluster.
func retroThemeTitle(category, observation string) string {
	const maxLen = 60
	obs := observation
	if len(obs) > maxLen {
		obs = obs[:maxLen] + "..."
	}
	return category + ": " + obs
}

// retroClusterToTheme converts a retroCluster into a RetroTheme.
func retroClusterToTheme(rank int, c retroCluster) RetroTheme {
	rep := retroRepresentative(c.signals)

	maxWeight := 0
	for _, sig := range c.signals {
		if w := severityWeight(sig.Severity); w > maxWeight {
			maxWeight = w
		}
	}

	entryIDs := make([]string, len(c.signals))
	for i, sig := range c.signals {
		entryIDs[i] = sig.EntryID
	}

	// top_suggestion: first non-empty suggestion in the cluster.
	topSuggestion := ""
	for _, sig := range c.signals {
		if sig.Suggestion != "" {
			topSuggestion = sig.Suggestion
			break
		}
	}

	return RetroTheme{
		Rank:                      rank,
		Category:                  c.signals[0].Category,
		Title:                     retroThemeTitle(c.signals[0].Category, rep.Observation),
		SignalCount:               len(c.signals),
		SeverityScore:             len(c.signals) * maxWeight,
		Signals:                   entryIDs,
		TopSuggestion:             topSuggestion,
		RepresentativeObservation: rep.Observation,
	}
}

// ─── Report rendering ─────────────────────────────────────────────────────────

// renderRetroMarkdown generates the markdown content for a retrospective report.
func renderRetroMarkdown(title string, result RetroSynthesisResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "| Field | Value |\n")
	fmt.Fprintf(&b, "|-------|-------|\n")
	fmt.Fprintf(&b, "| Scope | %s |\n", result.Scope)
	fmt.Fprintf(&b, "| Total Signals | %d |\n", result.SignalCount)
	if result.Period.From != "" {
		fmt.Fprintf(&b, "| Period | %s to %s |\n", result.Period.From, result.Period.To)
	}
	b.WriteString("\n---\n\n")

	if len(result.Themes) == 0 && len(result.WorkedWell) == 0 && len(result.Experiments) == 0 {
		b.WriteString("No signals found for the given filters.\n")
		return b.String()
	}

	if len(result.Themes) > 0 {
		b.WriteString("## Themes\n\n")
		for _, t := range result.Themes {
			fmt.Fprintf(&b, "### %d. %s\n\n", t.Rank, t.Title)
			fmt.Fprintf(&b, "**Category:** %s | **Signals:** %d | **Severity Score:** %d\n\n",
				t.Category, t.SignalCount, t.SeverityScore)
			fmt.Fprintf(&b, "> %s\n\n", t.RepresentativeObservation)
			if t.TopSuggestion != "" {
				fmt.Fprintf(&b, "**Suggestion:** %s\n\n", t.TopSuggestion)
			}
			if len(t.Signals) > 0 {
				fmt.Fprintf(&b, "Signals: %s\n\n", strings.Join(t.Signals, ", "))
			}
		}
	}

	if len(result.Experiments) > 0 {
		b.WriteString("## Experiments\n\n")
		for _, exp := range result.Experiments {
			fmt.Fprintf(&b, "### %s (%s)\n\n", exp.Title, exp.DecisionID)
			fmt.Fprintf(&b, "**Recommendation:** %s\n\n", exp.Recommendation)
			fmt.Fprintf(&b, "- Positive signals: %d\n", exp.PositiveSignals)
			fmt.Fprintf(&b, "- Negative signals: %d\n", exp.NegativeSignals)
			fmt.Fprintf(&b, "- Assessment: %s\n\n", exp.NetAssessment)
		}
	}

	if len(result.WorkedWell) > 0 {
		b.WriteString("## What Worked Well\n\n")
		for _, ww := range result.WorkedWell {
			fmt.Fprintf(&b, "### %s\n\n", ww.Title)
			fmt.Fprintf(&b, "**Signals:** %d\n\n", ww.SignalCount)
			fmt.Fprintf(&b, "> %s\n\n", ww.RepresentativeObservation)
		}
	}

	return b.String()
}
