package service

import (
	"testing"
)

// TestIncrementFeatureReviewCycle_ZeroInit verifies that incrementing a feature
// entity that has no review_cycle field (legacy record) starts from zero and
// produces review_cycle = 1.
func TestIncrementFeatureReviewCycle_ZeroInit(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	svc := NewEntityService(stateRoot)

	// Write a legacy feature record that has no review_cycle field.
	id, slug := "FEAT-01RRKZERO001", "rr-zero-init"
	writeTestEntity(t, stateRoot, "feature", id, slug,
		makeFeatureFields(id, slug, "", "reviewing", nil))

	if err := svc.IncrementFeatureReviewCycle(id, slug); err != nil {
		t.Fatalf("IncrementFeatureReviewCycle: %v", err)
	}

	got, err := svc.Get("feature", id, slug)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	rc, _ := got.State["review_cycle"].(int)
	if rc != 1 {
		t.Errorf("review_cycle = %d, want 1 (zero-value init for legacy record)", rc)
	}
}

// TestIncrementFeatureReviewCycle_NToNPlusOne verifies that incrementing a
// feature whose review_cycle is already N produces N+1.
func TestIncrementFeatureReviewCycle_NToNPlusOne(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	svc := NewEntityService(stateRoot)

	id, slug := "FEAT-01RRKNPLUS1", "rr-n-plus-one"
	fields := makeFeatureFields(id, slug, "", "reviewing", nil)
	fields["review_cycle"] = 2 // start at N=2
	writeTestEntity(t, stateRoot, "feature", id, slug, fields)

	if err := svc.IncrementFeatureReviewCycle(id, slug); err != nil {
		t.Fatalf("IncrementFeatureReviewCycle: %v", err)
	}

	got, err := svc.Get("feature", id, slug)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	rc, _ := got.State["review_cycle"].(int)
	if rc != 3 {
		t.Errorf("review_cycle = %d, want 3 (2→3)", rc)
	}
}

// TestIncrementFeatureReviewCycle_RoundTrip verifies that two successive
// increments persist correctly (round-trip through the store).
func TestIncrementFeatureReviewCycle_RoundTrip(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	svc := NewEntityService(stateRoot)

	id, slug := "FEAT-01RRKROUND1", "rr-round-trip"
	writeTestEntity(t, stateRoot, "feature", id, slug,
		makeFeatureFields(id, slug, "", "reviewing", nil))

	for i := 0; i < 2; i++ {
		if err := svc.IncrementFeatureReviewCycle(id, slug); err != nil {
			t.Fatalf("IncrementFeatureReviewCycle #%d: %v", i+1, err)
		}
	}

	got, err := svc.Get("feature", id, slug)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	rc, _ := got.State["review_cycle"].(int)
	if rc != 2 {
		t.Errorf("review_cycle = %d after two increments, want 2", rc)
	}
}

// TestPersistFeatureBlockedReason_SetReason verifies that a non-empty reason
// is written to and readable from the feature entity.
func TestPersistFeatureBlockedReason_SetReason(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	svc := NewEntityService(stateRoot)

	id, slug := "FEAT-01RRKSETR01", "rr-set-reason"
	writeTestEntity(t, stateRoot, "feature", id, slug,
		makeFeatureFields(id, slug, "", "reviewing", nil))

	const want = "Review iteration cap reached (3/3). Human decision required."
	if err := svc.PersistFeatureBlockedReason(id, slug, want); err != nil {
		t.Fatalf("PersistFeatureBlockedReason: %v", err)
	}

	got, err := svc.Get("feature", id, slug)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	br, _ := got.State["blocked_reason"].(string)
	if br != want {
		t.Errorf("blocked_reason = %q, want %q", br, want)
	}
}

// TestPersistFeatureBlockedReason_ClearReason verifies that passing an empty
// string removes the blocked_reason field from the entity.
func TestPersistFeatureBlockedReason_ClearReason(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	svc := NewEntityService(stateRoot)

	// Write a feature that already has a blocked_reason.
	id, slug := "FEAT-01RRKCLRR01", "rr-clear-reason"
	fields := makeFeatureFields(id, slug, "", "reviewing", nil)
	fields["blocked_reason"] = "previous block"
	writeTestEntity(t, stateRoot, "feature", id, slug, fields)

	if err := svc.PersistFeatureBlockedReason(id, slug, ""); err != nil {
		t.Fatalf("PersistFeatureBlockedReason (clear): %v", err)
	}

	got, err := svc.Get("feature", id, slug)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, present := got.State["blocked_reason"]; present {
		t.Errorf("blocked_reason should be absent after clearing, got %v", got.State["blocked_reason"])
	}
}

// TestPersistFeatureBlockedReason_RoundTrip verifies set-then-clear semantics:
// setting a reason persists it, and clearing it removes it — both confirmed by
// reading back from the store.
func TestPersistFeatureBlockedReason_RoundTrip(t *testing.T) {
	t.Parallel()

	stateRoot := t.TempDir()
	svc := NewEntityService(stateRoot)

	id, slug := "FEAT-01RRKRTTRP1", "rr-round-trip-br"
	writeTestEntity(t, stateRoot, "feature", id, slug,
		makeFeatureFields(id, slug, "", "reviewing", nil))

	const reason = "Cap exceeded"

	// Set reason and verify.
	if err := svc.PersistFeatureBlockedReason(id, slug, reason); err != nil {
		t.Fatalf("set blocked_reason: %v", err)
	}
	got1, err := svc.Get("feature", id, slug)
	if err != nil {
		t.Fatalf("Get after set: %v", err)
	}
	if br, _ := got1.State["blocked_reason"].(string); br != reason {
		t.Errorf("blocked_reason after set = %q, want %q", br, reason)
	}

	// Clear reason and verify it is removed.
	if err := svc.PersistFeatureBlockedReason(id, slug, ""); err != nil {
		t.Fatalf("clear blocked_reason: %v", err)
	}
	got2, err := svc.Get("feature", id, slug)
	if err != nil {
		t.Fatalf("Get after clear: %v", err)
	}
	if _, present := got2.State["blocked_reason"]; present {
		t.Errorf("blocked_reason should be absent after clearing")
	}
}
