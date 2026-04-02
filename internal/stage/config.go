package stage

// Stage represents a feature lifecycle stage name.
type Stage string

const (
	Designing   Stage = "designing"
	Specifying  Stage = "specifying"
	DevPlanning Stage = "dev-planning"
	Developing  Stage = "developing"
	Reviewing   Stage = "reviewing"
	NeedsRework Stage = "needs-rework"
)

// OrchestrationPattern is the orchestration mode for a stage.
type OrchestrationPattern string

const (
	SingleAgent         OrchestrationPattern = "single-agent"
	OrchestratorWorkers OrchestrationPattern = "orchestrator-workers"
)

// EffortBudget holds effort expectation text for a stage.
type EffortBudget struct {
	Text    string
	Warning string
}

// StageConfig holds all stage-specific configuration for context assembly.
type StageConfig struct {
	Orchestration       OrchestrationPattern
	EffortBudget        EffortBudget
	PrimaryTools        []string
	ExcludedTools       []string
	IncludeFilePaths    bool
	IncludeTestExpect   bool
	IncludeReviewRubric bool
	IncludeImplGuidance bool
	IncludePlanGuidance bool
	OutputConvention    bool
	SpecMode            string // "full" or "relevant-sections"
}

var configs = map[Stage]StageConfig{
	Designing: {
		Orchestration: SingleAgent,
		EffortBudget: EffortBudget{
			Text:    "5\u201315 tool calls. Read related designs, query decisions, draft structured document.",
			Warning: "Do NOT skip to implementation. Complete this stage\u2019s deliverables before advancing.",
		},
		PrimaryTools:        []string{"entity", "doc", "doc_intel", "knowledge", "status"},
		ExcludedTools:       []string{"decompose", "merge", "pr", "worktree", "finish"},
		IncludeFilePaths:    false,
		IncludeTestExpect:   false,
		IncludeReviewRubric: false,
		IncludeImplGuidance: false,
		IncludePlanGuidance: true,
		OutputConvention:    false,
		SpecMode:            "full",
	},
	Specifying: {
		Orchestration: SingleAgent,
		EffortBudget: EffortBudget{
			Text:    "5\u201315 tool calls. Read design document, query knowledge, check related decisions, draft each required section.",
			Warning: "Do NOT skip to implementation. Complete this stage\u2019s deliverables before advancing.",
		},
		PrimaryTools:        []string{"entity", "doc", "doc_intel", "knowledge", "status"},
		ExcludedTools:       []string{"decompose", "merge", "pr", "worktree", "finish"},
		IncludeFilePaths:    false,
		IncludeTestExpect:   false,
		IncludeReviewRubric: false,
		IncludeImplGuidance: false,
		IncludePlanGuidance: true,
		OutputConvention:    false,
		SpecMode:            "full",
	},
	DevPlanning: {
		Orchestration: SingleAgent,
		EffortBudget: EffortBudget{
			Text:    "5\u201310 tool calls. Read spec, decompose into tasks with dependencies, estimate effort, produce plan document.",
			Warning: "Do NOT skip to implementation. Complete this stage\u2019s deliverables before advancing.",
		},
		PrimaryTools:        []string{"entity", "doc", "knowledge", "decompose", "estimate", "status"},
		ExcludedTools:       []string{"merge", "pr", "worktree"},
		IncludeFilePaths:    false,
		IncludeTestExpect:   false,
		IncludeReviewRubric: false,
		IncludeImplGuidance: false,
		IncludePlanGuidance: true,
		OutputConvention:    false,
		SpecMode:            "full",
	},
	Developing: {
		Orchestration: OrchestratorWorkers,
		EffortBudget: EffortBudget{
			Text:    "10\u201350 tool calls per task. Read spec section, implement, test, iterate.",
			Warning: "Do NOT skip testing. Every change must be verified before marking done.",
		},
		PrimaryTools:        []string{"entity", "handoff", "next", "finish", "knowledge", "status", "branch", "worktree"},
		ExcludedTools:       []string{"decompose", "doc_intel"},
		IncludeFilePaths:    true,
		IncludeTestExpect:   true,
		IncludeReviewRubric: false,
		IncludeImplGuidance: true,
		IncludePlanGuidance: false,
		OutputConvention:    true,
		SpecMode:            "relevant-sections",
	},
	Reviewing: {
		Orchestration: OrchestratorWorkers,
		EffortBudget: EffortBudget{
			Text:    "5\u201310 tool calls per review dimension.",
			Warning: "Do NOT skip to implementation. Complete this stage\u2019s deliverables before advancing.",
		},
		PrimaryTools:        []string{"entity", "doc", "doc_intel", "knowledge", "finish", "status"},
		ExcludedTools:       []string{"decompose", "merge", "worktree", "handoff"},
		IncludeFilePaths:    true,
		IncludeTestExpect:   true,
		IncludeReviewRubric: true,
		IncludeImplGuidance: false,
		IncludePlanGuidance: false,
		OutputConvention:    true,
		SpecMode:            "relevant-sections",
	},
	NeedsRework: {
		Orchestration: OrchestratorWorkers,
		EffortBudget: EffortBudget{
			Text:    "10\u201350 tool calls per task. Read review findings, address issues, test fixes, iterate.",
			Warning: "Do NOT skip testing. Every change must be verified before marking done.",
		},
		PrimaryTools:        []string{"entity", "handoff", "next", "finish", "knowledge", "status", "branch", "worktree"},
		ExcludedTools:       []string{"decompose", "doc_intel"},
		IncludeFilePaths:    true,
		IncludeTestExpect:   true,
		IncludeReviewRubric: false,
		IncludeImplGuidance: true,
		IncludePlanGuidance: false,
		OutputConvention:    true,
		SpecMode:            "relevant-sections",
	},
}

// workingStates maps feature status strings to their Stage constant.
var workingStates = map[string]Stage{
	"designing":    Designing,
	"specifying":   Specifying,
	"dev-planning": DevPlanning,
	"developing":   Developing,
	"reviewing":    Reviewing,
	"needs-rework": NeedsRework,
}

// ForStage returns the StageConfig for the given feature lifecycle status.
// Returns the config and true if found, zero value and false if the status
// is not a working stage (e.g. "proposed", "done").
func ForStage(featureStatus string) (StageConfig, bool) {
	s, ok := workingStates[featureStatus]
	if !ok {
		return StageConfig{}, false
	}
	cfg, ok := configs[s]
	return cfg, ok
}

// IsWorkingState returns true if the feature status permits task work.
func IsWorkingState(featureStatus string) bool {
	_, ok := workingStates[featureStatus]
	return ok
}

// AllStages returns all configured stage names.
func AllStages() []Stage {
	return []Stage{
		Designing,
		Specifying,
		DevPlanning,
		Developing,
		Reviewing,
		NeedsRework,
	}
}
