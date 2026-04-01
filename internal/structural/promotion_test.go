package structural

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPromotionState_DefaultMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ps, err := LoadPromotionState(dir)
	if err != nil {
		t.Fatalf("LoadPromotionState: %v", err)
	}

	key := CheckKey{CheckType: "required_sections", DocumentType: "design"}
	mode := ps.GetMode(key)
	if mode != "warning" {
		t.Errorf("GetMode = %q, want warning", mode)
	}
}

func TestPromotionState_PromoteAfterThreshold(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ps, err := LoadPromotionState(dir)
	if err != nil {
		t.Fatalf("LoadPromotionState: %v", err)
	}

	key := CheckKey{CheckType: "required_sections", DocumentType: "design"}

	// 4 passes: still warning
	for i := 0; i < 4; i++ {
		ps.RecordPass(key)
	}
	if got := ps.GetMode(key); got != "warning" {
		t.Errorf("after 4 passes GetMode = %q, want warning", got)
	}

	// 5th pass: promote
	ps.RecordPass(key)
	if got := ps.GetMode(key); got != "hard_gate" {
		t.Errorf("after 5 passes GetMode = %q, want hard_gate", got)
	}
}

func TestPromotionState_FalsePositiveDemotes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ps, err := LoadPromotionState(dir)
	if err != nil {
		t.Fatalf("LoadPromotionState: %v", err)
	}

	key := CheckKey{CheckType: "required_sections", DocumentType: "design"}

	// Promote
	for i := 0; i < 5; i++ {
		ps.RecordPass(key)
	}
	if got := ps.GetMode(key); got != "hard_gate" {
		t.Fatalf("GetMode = %q, want hard_gate", got)
	}

	// False positive demotes back to warning
	ps.RecordFalsePositive(key, "false positive reason")
	if got := ps.GetMode(key); got != "warning" {
		t.Errorf("after false positive GetMode = %q, want warning", got)
	}
}

func TestPromotionState_FalsePositiveResetsConsecutiveClean(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ps, err := LoadPromotionState(dir)
	if err != nil {
		t.Fatalf("LoadPromotionState: %v", err)
	}

	key := CheckKey{CheckType: "cross_reference", DocumentType: "specification"}

	// 3 passes
	for i := 0; i < 3; i++ {
		ps.RecordPass(key)
	}

	// False positive resets
	ps.RecordFalsePositive(key, "fp")

	// Need 5 more to promote (counter reset)
	for i := 0; i < 4; i++ {
		ps.RecordPass(key)
	}
	if got := ps.GetMode(key); got != "warning" {
		t.Errorf("after reset+4 passes GetMode = %q, want warning", got)
	}

	ps.RecordPass(key)
	if got := ps.GetMode(key); got != "hard_gate" {
		t.Errorf("after reset+5 passes GetMode = %q, want hard_gate", got)
	}
}

func TestPromotionState_Persistence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Write state
	ps1, err := LoadPromotionState(dir)
	if err != nil {
		t.Fatalf("LoadPromotionState: %v", err)
	}
	key := CheckKey{CheckType: "required_sections", DocumentType: "dev-plan"}
	for i := 0; i < 5; i++ {
		ps1.RecordPass(key)
	}
	if err := ps1.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Re-load and verify
	ps2, err := LoadPromotionState(dir)
	if err != nil {
		t.Fatalf("LoadPromotionState (reload): %v", err)
	}
	if got := ps2.GetMode(key); got != "hard_gate" {
		t.Errorf("after reload GetMode = %q, want hard_gate", got)
	}
}

func TestPromotionState_LoadMissingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// No file created — should return empty state without error
	ps, err := LoadPromotionState(filepath.Join(dir, "nonexistent"))
	if err != nil {
		t.Fatalf("LoadPromotionState missing file: %v", err)
	}
	key := CheckKey{CheckType: "x", DocumentType: "y"}
	if got := ps.GetMode(key); got != "warning" {
		t.Errorf("GetMode = %q, want warning", got)
	}
}

func TestPromotionState_AtomicWrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ps, err := LoadPromotionState(dir)
	if err != nil {
		t.Fatalf("LoadPromotionState: %v", err)
	}

	key := CheckKey{CheckType: "required_sections", DocumentType: "design"}
	ps.RecordPass(key)
	if err := ps.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify the state file exists and no temp files remain
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "structural-check-state.yaml" {
			t.Errorf("unexpected file in state dir: %s", e.Name())
		}
	}
}

func TestPromotionEntry_PromotedAtTimestamp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	ps, err := LoadPromotionState(dir)
	if err != nil {
		t.Fatalf("LoadPromotionState: %v", err)
	}

	key := CheckKey{CheckType: "required_sections", DocumentType: "specification"}
	before := time.Now().UTC()
	for i := 0; i < 5; i++ {
		ps.RecordPass(key)
	}
	after := time.Now().UTC()

	entry := ps.entries[entryKey(key)]
	if entry.PromotedAt == nil {
		t.Fatal("PromotedAt should be set after promotion")
	}
	if entry.PromotedAt.Before(before) || entry.PromotedAt.After(after) {
		t.Errorf("PromotedAt %v not in expected range [%v, %v]", entry.PromotedAt, before, after)
	}
}
