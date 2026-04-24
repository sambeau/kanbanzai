package service

import (
	"fmt"
	"strings"
	"time"
)

type BranchLookup interface {
	GetBranchForEntity(entityID string) (branch string, err error)
	GetFilesOnBranch(repoRoot, branch string) ([]string, error)
	GetBranchCreatedAt(entityID string) (time.Time, error)
}

type ConflictCheckInput struct {
	TaskIDs    []string
	FeatureIDs []string
}

type FileOverlapDimension struct {
	Risk         string
	SharedFiles  []string
	GitConflicts []string
}

type DependencyOrderDimension struct {
	Risk   string
	Detail string
}

type BoundaryCrossingDimension struct {
	Risk   string
	Detail string
}

type ConflictDimensions struct {
	FileOverlap      FileOverlapDimension
	DependencyOrder  DependencyOrderDimension
	BoundaryCrossing BoundaryCrossingDimension
}

type ConflictPairResult struct {
	TaskA          string
	TaskB          string
	Risk           string
	Dimensions     ConflictDimensions
	Recommendation string
}

type ConflictCheckResult struct {
	TaskIDs     []string
	OverallRisk string
	Pairs       []ConflictPairResult
}

type FeatureConflictInfo struct {
	FeatureID    string
	FilesPlanned []string
	NoFileData   bool
	DriftDays    *int // nil when no worktree record exists
}

type FeatureConflictPairResult struct {
	FeatureA       string
	FeatureB       string
	Risk           string
	Dimensions     ConflictDimensions
	Recommendation string
}

type FeatureConflictResult struct {
	FeatureIDs  []string
	OverallRisk string
	Pairs       []FeatureConflictPairResult
	Features    []FeatureConflictInfo
}

type ConflictService struct {
	entitySvc    *EntityService
	branchLookup BranchLookup
	repoRoot     string
}

func NewConflictService(entitySvc *EntityService, branchLookup BranchLookup, repoRoot string) *ConflictService {
	return &ConflictService{
		entitySvc:    entitySvc,
		branchLookup: branchLookup,
		repoRoot:     repoRoot,
	}
}

type taskConflictInfo struct {
	id            string
	slug          string
	summary       string
	parentFeature string
	filesPlanned  []string
	dependsOn     []string
	featureSlug   string
	featureSpec   string
}

func (s *ConflictService) Check(input ConflictCheckInput) (ConflictCheckResult, error) {
	if len(input.FeatureIDs) > 0 && len(input.TaskIDs) > 0 {
		return ConflictCheckResult{}, fmt.Errorf("task_ids and feature_ids are mutually exclusive")
	}
	if len(input.TaskIDs) < 2 {
		return ConflictCheckResult{}, fmt.Errorf("conflict_domain_check requires at least two task IDs")
	}

	tasks := make([]taskConflictInfo, 0, len(input.TaskIDs))
	for _, id := range input.TaskIDs {
		result, err := s.entitySvc.Get("task", id, "")
		if err != nil {
			return ConflictCheckResult{}, fmt.Errorf("load task %s: %w", id, err)
		}

		info := taskConflictInfo{
			id:            result.ID,
			slug:          result.Slug,
			summary:       stringFromState(result.State, "summary"),
			parentFeature: stringFromState(result.State, "parent_feature"),
			filesPlanned:  stringSliceFromState(result.State, "files_planned"),
			dependsOn:     stringSliceFromState(result.State, "depends_on"),
		}

		if info.parentFeature != "" {
			featResult, err := s.entitySvc.Get("feature", info.parentFeature, "")
			if err == nil {
				info.featureSlug = featResult.Slug
				info.featureSpec = stringFromState(featResult.State, "spec")
			}
		}

		tasks = append(tasks, info)
	}

	var result ConflictCheckResult
	result.TaskIDs = input.TaskIDs
	result.OverallRisk = "none"

	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			pair := s.analyzePair(tasks[i], tasks[j], tasks)
			result.OverallRisk = maxRisk(result.OverallRisk, pair.Risk)
			result.Pairs = append(result.Pairs, pair)
		}
	}

	return result, nil
}

// CheckFeatures checks conflict risk across a set of features by aggregating
// their tasks' files_planned and comparing branch drift.
func (s *ConflictService) CheckFeatures(featureIDs []string) (FeatureConflictResult, error) {
	return s.checkFeatures(featureIDs)
}

func (s *ConflictService) checkFeatures(featureIDs []string) (FeatureConflictResult, error) {
	allTasks, err := s.entitySvc.List("task")
	if err != nil {
		return FeatureConflictResult{}, fmt.Errorf("list tasks: %w", err)
	}

	featureInfos := make([]FeatureConflictInfo, 0, len(featureIDs))
	for _, fid := range featureIDs {
		var filesPlanned []string
		for _, t := range allTasks {
			if stringFromState(t.State, "parent_feature") == fid {
				fps := stringSliceFromState(t.State, "files_planned")
				filesPlanned = append(filesPlanned, fps...)
			}
		}

		info := FeatureConflictInfo{
			FeatureID:    fid,
			FilesPlanned: filesPlanned,
			NoFileData:   len(filesPlanned) == 0,
		}

		if s.branchLookup != nil {
			if createdAt, err := s.branchLookup.GetBranchCreatedAt(fid); err == nil {
				days := int(time.Since(createdAt).Hours() / 24)
				info.DriftDays = &days
			}
		}

		featureInfos = append(featureInfos, info)
	}

	// Build synthetic taskConflictInfo entries for pair analysis.
	// id and parentFeature are both set to the feature ID so that
	// checkFileOverlap can look up branches via GetBranchForEntity.
	fakeTasks := make([]taskConflictInfo, len(featureInfos))
	for i, info := range featureInfos {
		fakeTasks[i] = taskConflictInfo{
			id:            info.FeatureID,
			parentFeature: info.FeatureID,
			filesPlanned:  info.FilesPlanned,
		}
	}

	var result FeatureConflictResult
	result.FeatureIDs = featureIDs
	result.OverallRisk = "none"
	result.Features = featureInfos

	for i := 0; i < len(fakeTasks)-1; i++ {
		for j := i + 1; j < len(fakeTasks); j++ {
			pair := s.analyzePair(fakeTasks[i], fakeTasks[j], fakeTasks)
			result.OverallRisk = maxRisk(result.OverallRisk, pair.Risk)
			result.Pairs = append(result.Pairs, FeatureConflictPairResult{
				FeatureA:       pair.TaskA,
				FeatureB:       pair.TaskB,
				Risk:           pair.Risk,
				Dimensions:     pair.Dimensions,
				Recommendation: pair.Recommendation,
			})
		}
	}

	return result, nil
}

func (s *ConflictService) analyzePair(a, b taskConflictInfo, allTasks []taskConflictInfo) ConflictPairResult {
	fileOverlap := s.checkFileOverlap(a, b)
	depOrder := checkDependencyOrder(a, b, allTasks)
	boundary := checkBoundaryCrossing(a, b)

	pairRisk := maxRisk(fileOverlap.Risk, maxRisk(depOrder.Risk, boundary.Risk))

	var recommendation string
	if depOrder.Risk != "none" {
		recommendation = "serialise"
	} else if riskLevel(pairRisk) >= riskLevel("medium") {
		recommendation = "checkpoint_required"
	} else {
		recommendation = "safe_to_parallelise"
	}

	return ConflictPairResult{
		TaskA: a.id,
		TaskB: b.id,
		Risk:  pairRisk,
		Dimensions: ConflictDimensions{
			FileOverlap:      fileOverlap,
			DependencyOrder:  depOrder,
			BoundaryCrossing: boundary,
		},
		Recommendation: recommendation,
	}
}

func (s *ConflictService) checkFileOverlap(a, b taskConflictInfo) FileOverlapDimension {
	result := FileOverlapDimension{Risk: "none"}

	aFiles := make(map[string]bool, len(a.filesPlanned))
	for _, f := range a.filesPlanned {
		aFiles[f] = true
	}
	for _, f := range b.filesPlanned {
		if aFiles[f] {
			result.SharedFiles = append(result.SharedFiles, f)
		}
	}

	if len(result.SharedFiles) > 0 {
		result.Risk = "medium"
	}

	if s.branchLookup != nil {
		branchA, errA := s.branchLookup.GetBranchForEntity(a.parentFeature)
		branchB, errB := s.branchLookup.GetBranchForEntity(b.parentFeature)
		if errA == nil && errB == nil && branchA != "" && branchB != "" {
			filesA, errA := s.branchLookup.GetFilesOnBranch(s.repoRoot, branchA)
			filesB, errB := s.branchLookup.GetFilesOnBranch(s.repoRoot, branchB)
			if errA == nil && errB == nil && len(filesA) > 0 && len(filesB) > 0 {
				bFileSet := make(map[string]bool, len(filesB))
				for _, f := range filesB {
					bFileSet[f] = true
				}
				for _, f := range filesA {
					if bFileSet[f] {
						result.GitConflicts = append(result.GitConflicts, f)
					}
				}
			}
		}
	}

	if len(result.GitConflicts) > 0 {
		if len(result.SharedFiles) > 0 {
			result.Risk = "high"
		} else {
			result.Risk = maxRisk(result.Risk, "medium")
		}
	}

	return result
}

func checkDependencyOrder(a, b taskConflictInfo, allTasks []taskConflictInfo) DependencyOrderDimension {
	for _, dep := range a.dependsOn {
		if dep == b.id {
			return DependencyOrderDimension{
				Risk:   "high",
				Detail: fmt.Sprintf("%s depends on %s", a.id, b.id),
			}
		}
	}
	for _, dep := range b.dependsOn {
		if dep == a.id {
			return DependencyOrderDimension{
				Risk:   "high",
				Detail: fmt.Sprintf("%s depends on %s", b.id, a.id),
			}
		}
	}

	taskMap := make(map[string]*taskConflictInfo, len(allTasks))
	for i := range allTasks {
		taskMap[allTasks[i].id] = &allTasks[i]
	}

	if reachable(a.id, b.id, taskMap) {
		return DependencyOrderDimension{
			Risk:   "medium",
			Detail: fmt.Sprintf("transitive dependency path exists between %s and %s", a.id, b.id),
		}
	}
	if reachable(b.id, a.id, taskMap) {
		return DependencyOrderDimension{
			Risk:   "medium",
			Detail: fmt.Sprintf("transitive dependency path exists between %s and %s", b.id, a.id),
		}
	}

	return DependencyOrderDimension{Risk: "none"}
}

func reachable(fromID, toID string, taskMap map[string]*taskConflictInfo) bool {
	visited := make(map[string]bool)
	queue := []string{fromID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		t, ok := taskMap[current]
		if !ok {
			continue
		}
		for _, dep := range t.dependsOn {
			if dep == toID {
				return true
			}
			if !visited[dep] {
				queue = append(queue, dep)
			}
		}
	}
	return false
}

func checkBoundaryCrossing(a, b taskConflictInfo) BoundaryCrossingDimension {
	aKeywords := extractConflictKeywords(a)
	bKeywords := extractConflictKeywords(b)

	var shared []string
	for kw := range aKeywords {
		if bKeywords[kw] {
			shared = append(shared, kw)
		}
	}

	if len(shared) >= 3 {
		return BoundaryCrossingDimension{
			Risk:   "medium",
			Detail: fmt.Sprintf("shared terms: %s", strings.Join(shared, ", ")),
		}
	}
	if len(shared) >= 1 {
		return BoundaryCrossingDimension{
			Risk:   "low",
			Detail: fmt.Sprintf("shared terms: %s", strings.Join(shared, ", ")),
		}
	}
	return BoundaryCrossingDimension{Risk: "none"}
}

func extractConflictKeywords(t taskConflictInfo) map[string]bool {
	raw := strings.Join([]string{t.summary, t.slug, t.featureSlug, t.featureSpec}, " ")
	tokens := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_' || r == '/' || r == '.'
	})
	keywords := make(map[string]bool, len(tokens))
	for _, tok := range tokens {
		tok = strings.ToLower(tok)
		if len(tok) >= 3 {
			keywords[tok] = true
		}
	}
	return keywords
}

func maxRisk(a, b string) string {
	if riskLevel(a) >= riskLevel(b) {
		return a
	}
	return b
}

func riskLevel(r string) int {
	switch r {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	default:
		return 0
	}
}
