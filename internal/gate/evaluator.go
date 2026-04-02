package gate

import (
	"fmt"

	"github.com/sambeau/kanbanzai/internal/binding"
	"github.com/sambeau/kanbanzai/internal/model"
)

// GateResult represents the outcome of a single prerequisite check.
type GateResult struct {
	Stage     string
	Satisfied bool
	Reason    string
	Source    string // "registry" or "hardcoded"
}

// DocumentService is the interface needed by gate evaluators.
type DocumentService interface {
	GetDocument(id string, loadContent bool) (*DocumentRecord, error)
	ListDocuments(filters DocumentFilters) ([]*DocumentRecord, error)
}

// EntityService is the interface needed by gate evaluators.
type EntityService interface {
	List(entityType string) ([]EntityResult, error)
}

// DocumentRecord is the minimal document type used by gate evaluators.
type DocumentRecord struct {
	ID     string
	Status string
	Type   string
	Owner  string
}

// DocumentFilters specifies filters for listing documents.
type DocumentFilters struct {
	Owner  string
	Type   string
	Status string
}

// EntityResult is the minimal entity type used by gate evaluators.
type EntityResult struct {
	ID    string
	State map[string]any
}

// PrereqEvalContext provides services needed during prerequisite evaluation.
type PrereqEvalContext struct {
	Feature   *model.Feature
	DocSvc    DocumentService
	EntitySvc EntityService
}

// PrereqEvaluator is a function that evaluates prerequisites of a specific type.
type PrereqEvaluator func(prereqs *binding.Prerequisites, stage string, ctx PrereqEvalContext) []GateResult

var evaluatorRegistry = map[string]PrereqEvaluator{}

// RegisterEvaluator registers an evaluator function for a prerequisite type key.
func RegisterEvaluator(typeKey string, fn PrereqEvaluator) {
	evaluatorRegistry[typeKey] = fn
}

// EvaluatePrerequisites dispatches prerequisite checks to registered evaluators
// based on which fields are populated in the Prerequisites struct.
func EvaluatePrerequisites(prereqs *binding.Prerequisites, stage string, ctx PrereqEvalContext) []GateResult {
	if prereqs == nil {
		return nil
	}

	var results []GateResult

	if len(prereqs.Documents) > 0 {
		results = append(results, dispatch("documents", prereqs, stage, ctx)...)
	}

	if prereqs.Tasks != nil {
		results = append(results, dispatch("tasks", prereqs, stage, ctx)...)
	}

	return results
}

func dispatch(typeKey string, prereqs *binding.Prerequisites, stage string, ctx PrereqEvalContext) []GateResult {
	fn, ok := evaluatorRegistry[typeKey]
	if !ok {
		return []GateResult{{
			Stage:     stage,
			Satisfied: false,
			Reason:    fmt.Sprintf("unknown prerequisite type %q for stage %q", typeKey, stage),
			Source:    "registry",
		}}
	}
	return fn(prereqs, stage, ctx)
}
