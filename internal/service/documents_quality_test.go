package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sambeau/kanbanzai/internal/config"
	"github.com/sambeau/kanbanzai/internal/model"
)

// requireQualityConfig returns a *config.Config with RequireForApproval=true
// and the default threshold (0 → treated as 0.7 by ApproveDocument).
func requireQualityConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.QualityEvaluation.RequireForApproval = true
	return &cfg
}

// submitQualityTestDoc creates a document file in svc's repo root and submits
// it, returning the resulting document ID.
func submitQualityTestDoc(t *testing.T, svc *DocumentService) string {
	t.Helper()
	docPath := "work/design/quality-test.md"
	fullPath := filepath.Join(svc.RepoRoot(), docPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(fullPath), err)
	}
	if err := os.WriteFile(fullPath, []byte("# Quality Test\n\nContent."), 0o644); err != nil {
		t.Fatalf("write %s: %v", fullPath, err)
	}
	submitted, err := svc.SubmitDocument(SubmitDocumentInput{
		Path:      docPath,
		Type:      "design",
		Title:     "Quality Test Design",
		CreatedBy: "tester",
	})
	if err != nil {
		t.Fatalf("SubmitDocument: %v", err)
	}
	return submitted.ID
}

// TestApproveDocument_QualityEvaluationGate verifies the three FR-018 behaviours
// for the quality evaluation approval gate when RequireForApproval is true:
//
//  1. Missing evaluation → approval blocked with an error.
//  2. Evaluation with pass=false → approval blocked with an error.
//  3. Evaluation with pass=true → approval succeeds.
func TestApproveDocument_QualityEvaluationGate(t *testing.T) {
	t.Parallel()

	t.Run("missing evaluation blocked", func(t *testing.T) {
		t.Parallel()

		stateRoot := t.TempDir()
		repoRoot := t.TempDir()
		svc := NewDocumentService(stateRoot, repoRoot)
		svc.SetConfigProvider(requireQualityConfig)

		docID := submitQualityTestDoc(t, svc)

		_, err := svc.ApproveDocument(ApproveDocumentInput{
			ID:         docID,
			ApprovedBy: "reviewer",
		})
		if err == nil {
			t.Fatal("expected error when quality evaluation is required but no evaluation is attached")
		}
	})

	t.Run("failing evaluation blocked", func(t *testing.T) {
		t.Parallel()

		stateRoot := t.TempDir()
		repoRoot := t.TempDir()
		svc := NewDocumentService(stateRoot, repoRoot)
		svc.SetConfigProvider(requireQualityConfig)

		docID := submitQualityTestDoc(t, svc)

		if _, err := svc.AttachQualityEvaluation(AttachEvaluationInput{
			ID: docID,
			Evaluation: model.QualityEvaluation{
				OverallScore: 0.4,
				Pass:         false,
				EvaluatedAt:  time.Now().UTC(),
				Evaluator:    "test-model",
				Dimensions:   map[string]float64{"clarity": 0.4},
			},
		}); err != nil {
			t.Fatalf("AttachQualityEvaluation: %v", err)
		}

		_, err := svc.ApproveDocument(ApproveDocumentInput{
			ID:         docID,
			ApprovedBy: "reviewer",
		})
		if err == nil {
			t.Fatal("expected error when quality evaluation pass=false")
		}
	})

	t.Run("passing evaluation allowed", func(t *testing.T) {
		t.Parallel()

		stateRoot := t.TempDir()
		repoRoot := t.TempDir()
		svc := NewDocumentService(stateRoot, repoRoot)
		svc.SetConfigProvider(requireQualityConfig)

		docID := submitQualityTestDoc(t, svc)

		if _, err := svc.AttachQualityEvaluation(AttachEvaluationInput{
			ID: docID,
			Evaluation: model.QualityEvaluation{
				OverallScore: 0.85,
				Pass:         true,
				EvaluatedAt:  time.Now().UTC(),
				Evaluator:    "test-model",
				Dimensions:   map[string]float64{"clarity": 0.9, "completeness": 0.8},
			},
		}); err != nil {
			t.Fatalf("AttachQualityEvaluation: %v", err)
		}

		result, err := svc.ApproveDocument(ApproveDocumentInput{
			ID:         docID,
			ApprovedBy: "reviewer",
		})
		if err != nil {
			t.Fatalf("ApproveDocument: %v", err)
		}
		if result.Status != string(model.DocumentStatusApproved) {
			t.Errorf("Status = %q, want %q", result.Status, model.DocumentStatusApproved)
		}
	})
}
